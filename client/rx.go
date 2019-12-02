package client

import (
	"bytes"
	"runtime"
	"sync"

	"github.com/go-fastlog/fastlog"

	"github.com/maoxs2/gxminer/go-randomx"
	"github.com/maoxs2/gxminer/worker"
)

type Rx struct {
	initMu sync.Mutex
	conf   *worker.Config
	logger *fastlog.Logger

	SeedHash     []byte
	NextSeedHash []byte
	Dataset      *randomx.RxDataset
	NextDataset  *randomx.RxDataset

	Workers []*worker.Worker

	submitCh chan worker.Job
}

func NewRxClient(conf *worker.Config, submitCh chan worker.Job, logger *fastlog.Logger) *Rx {
	return &Rx{
		conf:     conf,
		submitCh: submitCh,
		logger:   logger,
	}
}

func (rx *Rx) ReadyInit(seedHash []byte) {
	rx.initMu.Lock()
	defer rx.initMu.Unlock()

	if rx.Dataset != nil {
		return
	}

	rx.logger.Infoln("Dataset initializing")
	rxDataset, err := randomx.NewRxDataset(rx.conf.Flags()...)
	if err != nil {
		rx.logger.Errorln(err)
		if rx.conf.HugePage == true {
			rx.logger.Fatal("Clean or create more VirtualMem/PageFile (require > 2.5G per NUMA Node)")
			if runtime.GOOS == "windows" {
				rx.logger.Fatal("Under windows, you have to reboot the system to clean the VirtualMem/PageFile.")
			}else{
				rx.logger.Fatal("Under linux, you can use `sysctl -w vm.nr_hugepages=1250` to enable the VirtualMem/PageFile.")
			}
		}
	}
	rxDataset.CInit(seedHash, rx.conf.WorkerNum*2)

	rx.logger.Infoln("Dataset initialization finished, start mining")
	rx.SeedHash = seedHash
	rx.Dataset = rxDataset
}

func (rx *Rx) ReadyNext(seedHash []byte) {
	rx.initMu.Lock()
	defer rx.initMu.Unlock()

	if rx.NextDataset != nil {
		return
	}

	rx.logger.Infoln("Getting ready for next seed hash, the hashrate might be lower")
	rxDataset, err := randomx.NewRxDataset(rx.conf.Flags()...)
	if err != nil {
		rx.logger.Errorln(err)
		if rx.conf.HugePage == true {
			rx.logger.Fatal("Clean or create more VirtualMem/PageFile (require > 2.5G per NUMA Node)")
			if runtime.GOOS == "windows" {
				rx.logger.Fatal("Under windows, you have to reboot the system to clean the VirtualMem/PageFile.")
			}else{
				rx.logger.Fatal("Under linux, you can use `sysctl -w vm.nr_hugepages=1250` to enable the VirtualMem/PageFile.")
			}
		}
	}
	rxDataset.CInit(seedHash, rx.conf.WorkerNum*2)

	rx.NextSeedHash = seedHash
	rx.NextDataset = rxDataset
}

func (rx *Rx) UpdateRxDataset(seedHash []byte) {
	rx.initMu.Lock()
	defer rx.initMu.Unlock()

	if bytes.Compare(seedHash, rx.SeedHash) == 0 {
		return
	}

	rx.logger.Infoln("Updating dataset")

	if rx.NextDataset == nil { // means vm is not enough
		rx.Dataset.Close() // release VMem
		rxDataset, err := randomx.NewRxDataset(rx.conf.Flags()...)
		if err != nil {
			rx.logger.Errorln(err)
			if rx.conf.HugePage == true {
				rx.logger.Fatal("Clean or create more VirtualMem/PageFile (require > 2.5G per NUMA Node)")
				if runtime.GOOS == "windows" {
					rx.logger.Fatal("Under windows, you have to reboot the system to clean the VirtualMem/PageFile.")
				}else{
					rx.logger.Fatal("Under linux, you can use `sysctl -w vm.nr_hugepages=1250` to enable the VirtualMem/PageFile.")
				}
			}
		}
		rxDataset.CInit(seedHash, rx.conf.WorkerNum*2)

		rx.NextSeedHash = seedHash
		rx.NextDataset = rxDataset
	} else {
		rx.Dataset.Close() // release VMem
		rx.Dataset = nil
		rx.SeedHash = nil
	}

	for _, w := range rx.Workers {
		w.UpdateVM(rx.NextDataset)
	}

	rx.Dataset = rx.NextDataset
	rx.SeedHash = rx.NextSeedHash
	rx.NextDataset = nil
	rx.NextSeedHash = nil
}

func (rx *Rx) SpawnWorkers(job worker.Job) {
	for i := uint32(0); i < rx.conf.WorkerNum; i++ {
		vm, err := randomx.NewRxVM(rx.Dataset, rx.conf.Flags()...)
		if err != nil {
			rx.logger.Fatal(err)
		}
		w := worker.NewWorker(i, rx.conf, vm, rx.submitCh)
		rx.Workers = append(rx.Workers, w)
		w.CStart(job)
	}
}

func (rx *Rx) AssignNewJob(job worker.Job) {
	for _, w := range rx.Workers {
		w.AssignNewJob(job)
	}
}

func (rx *Rx) GetWorkerHashrates() map[uint32]float64 {
	hrs := make(map[uint32]float64)
	for _, w := range rx.Workers {
		hrs[w.Id] = w.Hashrate()
	}
	return hrs
}

func (rx *Rx) StopWorkers() {
	for _, w := range rx.Workers {
		w.Close()
	}
	rx.Workers = nil
}

func (rx *Rx) Close() {
	if rx.Dataset != nil {
		rx.Dataset.Close()
	}
	if rx.NextDataset != nil {
		rx.NextDataset.Close()
	}

	rx.Dataset = nil
	rx.NextDataset = nil
}
