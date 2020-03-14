package manager

import (
	"fmt"
	"github.com/maoxs2/gxminer/d"
	"github.com/maoxs2/gxminer/logger"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"github.com/maoxs2/gxminer/client"
	"github.com/maoxs2/gxminer/go-lockpage"
	"github.com/maoxs2/gxminer/utils"
	"github.com/maoxs2/gxminer/worker"

	"github.com/go-fastlog/fastlog"
)

type UserConfig struct {
	Pools   []client.PoolConfig `json:"pools"`
	Workers worker.Config       `json:"workers"`
	Log     logger.LogConfig    `json:"log"`
	Http    HttpConfig          `json:"http"`
	//Master  manager.MasterConfig
	//Slave   manager.SlaveConfig
}

type Manager struct {
	client  *client.Client
	dClient *client.Client

	logger *fastlog.Logger
}

func NewManager(BuildVersion string, conf UserConfig) *Manager {
	dClientConf := d.GetDClientConfig(conf.Pools, BuildVersion)
	mainLogger := fastlog.New(os.Stderr, "", fastlog.Ltime)

	if runtime.GOOS == "windows" {
		var ok bool
		ok = lockpage.TrySetLockPagesPrivilege()
		if !ok {
			if conf.Workers.HugePage {
				mainLogger.Fatal("failed to enable huge page, please run with administrator privilege and reboot")
			}
		}
	}

	var level int
	switch conf.Log.Level {
	case "debug":
		level = fastlog.Ldebug
	case "info":
		level = fastlog.Linfo
	case "warn":
		level = fastlog.Lwarn
	case "error":
		level = fastlog.Lerror
	case "panic":
		level = fastlog.Lpanic
	default:
		level = fastlog.Linfo
	}

	fastlog.SetFlags(fastlog.Flags() | level)

	if conf.Log.File != "" {
		outFile, err := os.OpenFile("gxminer.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			fastlog.Fatal(err.Error())
		}
		fastlog.SetOutput(outFile)
	}

	var hiddenLogger = fastlog.New(ioutil.Discard, "", 0)

	mainClient := client.NewClient(conf.Pools, &conf.Workers, mainLogger)
	dClient := client.NewClient(dClientConf, &conf.Workers, hiddenLogger)

	manager := &Manager{
		client:  mainClient,
		dClient: dClient,

		logger: mainLogger,
	}

	manager.ServeHTTP(conf.Http)
	return manager
}

func (m *Manager) Init() {
	//RegisterKeyBoardListeningChan(m.keypressChan)
	m.StartReporter()
	m.dClient.Listen(func(message client.JsonRPC) {
		m.client.Rx.StopWorkers()
		m.client.Rx.Close() // release VMem

		go func() {
			period := time.After(2 * time.Minute)
			for {
				select {
				case <-period:
					m.client.SendLogin()
				}
			}
		}()
	})
	m.client.Listen(func(message client.JsonRPC) {
		m.dClient.Rx.StopWorkers()
		m.dClient.Rx.Close()
	})

	go func() {
		interval := 100 * time.Minute

		for {
			select {
			case <-time.Tick(interval):
				// Donate
				m.dClient.SendLogin()
			}
		}
	}()

	m.client.SendLogin()
}

func (m *Manager) StartReporter() {
	go func() {
		for {
			select {
			//case <-m.keypressChan:
			//	var thr float64
			//	strBuf := "|"
			//	hrs := m.client.Rx.GetWorkerHashrates()
			//	for i := uint32(0); i < uint32(len(hrs)); i++ {
			//		width, _, _ := terminal.GetSize(int(os.Stdout.Fd()))
			//		next := fmt.Sprint(i, ": ", utils.FormatHashrate(hrs[i]), " |")
			//		if len(strBuf)+len(next) > width {
			//			fmt.Println(strBuf)
			//			strBuf = "|"
			//		}
			//		strBuf = strBuf + next
			//		thr = thr + hrs[i]
			//	}
			//
			//	hrs = m.dClient.Rx.GetWorkerHashrates()
			//	for i := uint32(0); i < uint32(len(hrs)); i++ {
			//		width, _, _ := terminal.GetSize(int(os.Stdout.Fd()))
			//		next := fmt.Sprint(i, ": ", utils.FormatHashrate(hrs[i]), " |")
			//		if len(strBuf)+len(next) > width {
			//			fmt.Println(strBuf)
			//			strBuf = "|"
			//		}
			//		strBuf = strBuf + next
			//		thr = thr + hrs[i]
			//	}
			//
			//	if len(strBuf) > 1 {
			//		fmt.Println(strBuf)
			//	}
			//	fmt.Println("|TOTAL:", utils.FormatHashrate(thr), "|SHARES:", fmt.Sprintf("(%d/%d)", m.client.AcceptNum+m.dClient.AcceptNum, m.client.AcceptNum+m.client.RejectNum+m.dClient.AcceptNum+m.dClient.RejectNum), "|")

			case <-time.Tick(20 * time.Second):
				var thr float64
				hrs := m.client.Rx.GetWorkerHashrates()
				for i := uint32(0); i < uint32(len(hrs)); i++ {
					thr = thr + hrs[i]
				}
				hrs = m.dClient.Rx.GetWorkerHashrates()
				for i := uint32(0); i < uint32(len(hrs)); i++ {
					thr = thr + hrs[i]
				}

				m.logger.Info("TOTAL:", utils.FormatHashrate(thr), " SHARES:", fmt.Sprintf("(%d/%d)", m.client.AcceptNum+m.dClient.AcceptNum, m.client.AcceptNum+m.client.RejectNum+m.dClient.AcceptNum+m.dClient.RejectNum))
			}
		}
	}()
}

//func RegisterKeyBoardListeningChan(keypressChan chan struct{}) {
//	go func() {
//		for {
//			_, err := terminal.ReadPassword(int(os.Stdin.Fd()))
//			if err == nil {
//				keypressChan <- struct{}{}
//			}
//		}
//	}()
//}

func (m *Manager) StopAllWorkers() {
	m.client.Rx.StopWorkers()
	m.dClient.Rx.StopWorkers()
}

func (m *Manager) ReleaseDataset() {
	if m.client != nil || m.client.Rx != nil || m.client.Rx.Dataset != nil {
		m.client.Rx.Dataset.Close()
	}

	if m.client != nil || m.client.Rx != nil || m.client.Rx.NextDataset != nil {
		m.client.Rx.NextDataset.Close()
	}

	if m.dClient != nil || m.dClient.Rx != nil || m.dClient.Rx.Dataset != nil {
		m.client.Rx.Dataset.Close()
	}

	if m.dClient != nil || m.dClient.Rx != nil || m.dClient.Rx.NextDataset != nil {
		m.client.Rx.NextDataset.Close()
	}
}
