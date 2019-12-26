package worker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/maoxs2/go-cpuaffinity"
	"github.com/maoxs2/gxminer/go-hwloc"
	"github.com/maoxs2/gxminer/go-randomx"
)

var (
	maxNicehashN = binary.LittleEndian.Uint32([]byte{255, 255, 255, 0})
)

type Job struct {
	// common
	ID    string
	JobID string

	// recv
	Blob         []byte
	Target       uint64
	SeedHash     []byte
	NextSeedHash []byte

	// sent
	Nonce  []byte
	Result []byte
}

// a hashing unit
type Worker struct {
	Id       uint32
	conf     *Config
	vm       *randomx.RxVM
	submitCh chan Job

	startTime time.Time
	maxTimes  uint64

	job Job

	newJobCh  chan Job
	closeCh   chan struct{}
	hashCount uint64

	mask     uint64
	affinity []int

	fixedByte byte
}

type Config struct {
	WorkerNum    uint32 `json:"worker-num"`
	Nicehash     bool   `json:"nicehash"`
	InitNum      uint32 `json:"init-num"`
	HugePage     bool   `json:"huge-page"`
	HardAES      bool   `json:"hard-aes"`
	FullMem      bool   `json:"full-mem"`
	JIT          bool   `json:"jit"`
	Argon2SSE3   bool   `json:"argon2-sse3"`
	Argon2AVX2   bool   `json:"argon2-avx2"`
	AffinityMask string `json:"affinity-mask"`

	mask     uint64
	affinity []int
}

func (c *Config) Flags() []randomx.Flag {
	var flags []randomx.Flag
	if c.JIT {
		flags = append(flags, randomx.FlagJIT)
	}

	if c.HardAES {
		flags = append(flags, randomx.FlagHardAES)
	}

	if c.FullMem {
		flags = append(flags, randomx.FlagFullMEM)
	}

	if c.HugePage {
		flags = append(flags, randomx.FlagLargePages)
	}

	if c.Argon2SSE3 {
		flags = append(flags, randomx.FlagArgon2SSSE3)
	}

	if c.Argon2AVX2 {
		flags = append(flags, randomx.FlagArgon2AVX2)
	}

	return flags
}

func NewWorker(id uint32, ds *randomx.RxDataset, conf *Config, submitCh chan Job) *Worker {
	var affinity []int

	vm, _ := randomx.NewRxVM(ds, conf.Flags()...)

	mask, err := strconv.ParseUint(conf.AffinityMask, 16, 64)
	if err != nil || mask > 1<<runtime.NumCPU() {
		fmt.Println("invalid affinity", err)
	} else {
		for i := 0; i < runtime.NumCPU(); i++ {
			if mask&(1<<i) == 1<<i {
				affinity = append(affinity, i)
			}
		}
	}

	if conf.WorkerNum < uint32(len(affinity)) {
		affinity = nil
	}

	w := &Worker{
		Id:        id,
		conf:      conf,
		vm:        vm,
		startTime: time.Now(),

		newJobCh: make(chan Job),
		closeCh:  make(chan struct{}),
		submitCh: submitCh,

		mask:     mask,
		affinity: affinity,
	}

	w.maxTimes = 1 << 8

	return w
}

func (w *Worker) CStart(initJob Job) {
	go func() {
		if w.Id < uint32(len(w.affinity)) {
			cpuaffinity.SetCPUAffinity(w.affinity[w.Id])
			hwloc.BindToNUMANode(int64(w.affinity[w.Id]))
		} else {
			cpuaffinity.SetCPUAffinityMask(w.mask)
		}

		job := initJob
		w.job = initJob

		var lastNonce, n uint32
		if w.conf.Nicehash {
			n = maxNicehashN / w.conf.WorkerNum * w.Id
			w.fixedByte = job.Blob[42]
		} else {
			n = math.MaxUint32 / w.conf.WorkerNum * w.Id
		}

		var blob = make([]byte, 76)
		copy(blob, job.Blob)
		binary.LittleEndian.PutUint32(blob[39:43], n)
		w.vm.CalcHashFirst(job.Blob)
		lastNonce = n

		for {
			select {
			case job = <-w.newJobCh:
				if w.conf.Nicehash {
					n = maxNicehashN / w.conf.WorkerNum * w.Id
					w.fixedByte = job.Blob[42]
				} else {
					n = math.MaxUint32 / w.conf.WorkerNum * w.Id
				}

				copy(blob, job.Blob)

			case <-w.closeCh:
				return

			default:
				lastNonce = n
				n++
				binary.LittleEndian.PutUint32(blob[39:43], n)
				if w.conf.Nicehash {
					blob[42] = w.fixedByte
				}

				result := w.vm.CalcHashNext(blob)
				if binary.LittleEndian.Uint64(result[24:32]) < job.Target {
					_job := job
					_job.Result = result
					binary.LittleEndian.PutUint32(_job.Nonce, lastNonce)
					if w.conf.Nicehash {
						_job.Nonce[3] = w.fixedByte
					}
					w.submitCh <- _job
				}

				w.hashCount++
			}
		}
	}()

	w.startTime = time.Now()
}

func (w *Worker) UpdateVM(rxDataset *randomx.RxDataset) {
	w.vm.UpdateDataset(rxDataset)
}

func (w *Worker) AssignNewJob(job Job) {
	if bytes.Compare(w.job.Blob, job.Blob) != 0 {
		w.job = job
		w.newJobCh <- job
	}
}

func (w *Worker) Hashrate() float64 {
	var hs = float64(w.hashCount) / time.Since(w.startTime).Seconds()
	return hs
}

func (w *Worker) Close() {
	w.closeCh <- struct{}{}
}
