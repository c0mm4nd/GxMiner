package worker

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/jaypipes/ghw"
	"math"
	"runtime"
	"strconv"
	"time"

	"github.com/maoxs2/go-cpuaffinity"
	"github.com/maoxs2/gxminer/go-hwloc"
	"github.com/maoxs2/gxminer/go-randomx"
)

const (
	maxNicehashN = uint32(255) | uint32(255)<<8 | uint32(255)<<16
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
	topology *hwloc.Topology
	vm       *randomx.RxVM
	submitCh chan Job

	startTime time.Time
	maxTimes  uint64

	job Job

	newJobCh  chan Job
	closeCh   chan struct{}
	hashCount uint64

	mask        uint64
	cpuAffinity []int
	numa        int

	nicehash bool
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

func NewWorker(id uint32, ds *randomx.RxDataset, conf *Config, submitCh chan Job, nicehash bool, topology *hwloc.Topology) *Worker {
	var cpuAffinity = make([]int, 0, runtime.NumCPU()) // may be less than core num so cannot allocate
	var numa = 0
	vm, _ := randomx.NewRxVM(ds, conf.Flags()...)

	mask, err := strconv.ParseUint(conf.AffinityMask, 16, 64)
	if err != nil || mask > 1<<runtime.NumCPU() {
		fmt.Println("invalid mask", err)
	} else {
		for i := 0; i < runtime.NumCPU(); i++ {
			if mask&(1<<i) == 1<<i {
				cpuAffinity = append(cpuAffinity, i)
			}
		}
	}

	if topo, err := ghw.Topology(); err == nil && topo.Architecture == ghw.ARCHITECTURE_NUMA {
		numa = len(topo.Nodes)
	}

	if conf.WorkerNum < uint32(len(cpuAffinity)) {
		cpuAffinity = nil // threads(works) are not corresponding to the cpu affinity, miner should cancel the affinity and make golang optimize automatically
	}

	w := &Worker{
		Id:        id,
		conf:      conf,
		topology:  topology,
		vm:        vm,
		startTime: time.Now(),

		newJobCh: make(chan Job),
		closeCh:  make(chan struct{}),
		submitCh: submitCh,

		numa:        numa,
		mask:        mask,
		cpuAffinity: cpuAffinity,

		nicehash: nicehash,
	}

	w.maxTimes = 1 << 8

	return w
}

func (w *Worker) CStart(initJob Job) {
	go func() {
		if w.Id < uint32(len(w.cpuAffinity)) {
			cpuaffinity.SetCPUAffinity(w.cpuAffinity[w.Id]) // to one
		} else {
			cpuaffinity.SetCPUAffinityMask(w.mask) // to all cores in mask
		}

		if w.numa > 0 {
			nodeSet := w.topology.HwlocGetNUMANodeObjByOSIndex(w.Id % uint32(w.numa))
			w.topology.HwlocSetMemBind(nodeSet, hwloc.HwlocMemBindBind, hwloc.HwlocMemBindThread|hwloc.HwlocMemBindByNodeSet)
		}

		job := initJob
		w.job = initJob

		var lastNonce, n uint32
		if w.nicehash {
			n = maxNicehashN / w.conf.WorkerNum * w.Id
		} else {
			n = math.MaxUint32 / w.conf.WorkerNum * w.Id
		}

		var blob = make([]byte, 76)
		copy(blob, job.Blob)
		blob[39] = byte(n)
		blob[40] = byte(n >> 8)
		blob[41] = byte(n >> 16)
		if !w.nicehash {
			blob[42] = byte(n >> 24)
		}
		w.vm.CalcHashFirst(blob)
		lastNonce = n

		for {
			select {
			case job = <-w.newJobCh:
				if w.nicehash {
					n = maxNicehashN / w.conf.WorkerNum * w.Id
				} else {
					n = math.MaxUint32 / w.conf.WorkerNum * w.Id
				}

				copy(blob, job.Blob)

			case <-w.closeCh:
				return

			default:
				lastNonce = n
				n++
				//binary.LittleEndian.PutUint32(blob[39:43], n)
				blob[39] = byte(n)
				blob[40] = byte(n >> 8)
				blob[41] = byte(n >> 16)
				if !w.nicehash {
					blob[42] = byte(n >> 24)
				}

				result := w.vm.CalcHashNext(blob)
				if binary.LittleEndian.Uint64(result[24:32]) < job.Target {
					_job := job
					_job.Result = result
					//binary.LittleEndian.PutUint32(_job.Nonce, lastNonce)
					_job.Nonce[0] = byte(lastNonce)
					_job.Nonce[1] = byte(lastNonce >> 8)
					_job.Nonce[2] = byte(lastNonce >> 16)
					if !w.nicehash {
						_job.Nonce[3] = byte(lastNonce >> 24)
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
