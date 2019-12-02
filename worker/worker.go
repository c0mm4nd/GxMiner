package worker

import (
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

	newJobCh  chan Job
	closeCh   chan struct{}
	hashCount uint64

	mask     uint64
	affinity []int
}

type Config struct {
	WorkerNum    uint32 `json:"worker-num"`
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

func NewWorker(id uint32, conf *Config, rxVM *randomx.RxVM, submitCh chan Job) *Worker {
	var affinity []int

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
		vm:        rxVM,
		startTime: time.Now(),

		newJobCh: make(chan Job),
		closeCh:  make(chan struct{}),
		submitCh: submitCh,

		mask:     mask,
		affinity: affinity,
	}

	w.maxTimes = 1 << 9

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

		var job = initJob
		var blob = job.Blob
		var n = math.MaxUint32/w.conf.WorkerNum*w.Id
		//binary.LittleEndian.PutUint32(nonce, n)
		//copy(blob[39:43], nonce)
		//top := math.MaxUint32 / w.conf.WorkerNum * (w.Id + 1)
		//w.vm.CalcHashFirst(job.Blob)

		w.vm.CalcHashFirst(blob)

		for {
			select {
			case job = <-w.newJobCh:
				blob = job.Blob
				n=math.MaxUint32/w.conf.WorkerNum*w.Id
				w.vm.CalcHashFirst(blob)

			case <-w.closeCh:
				return

			default:
				nonce := make([]byte, 4)
				binary.LittleEndian.PutUint32(nonce, n)
				copy(blob[39:43], nonce)
				
				//w.vm.CalcHashFirst(blob)
				result := w.vm.CalcHashNext(blob)
				if binary.LittleEndian.Uint64(result[24:32]) < job.Target {
					job.Result = result
					job.Nonce = nonce
					w.submitCh <- job
				}

				//result, count := w.vm.Search(nonce, w.maxTimes, w.conf.WorkerNum, job.Target, job.Blob)
				//if count < w.maxTimes {
				//	job.Result = result
				//	job.Nonce = nonce
				//	w.submitCh <- job
				//}

				n++
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
	w.newJobCh <- job
}

func (w *Worker) Hashrate() float64 {
	var hs = float64(w.hashCount) / time.Since(w.startTime).Seconds()
	return hs
}

func (w *Worker) Close() {
	w.closeCh <- struct{}{}
}
