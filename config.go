package main

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"

	"github.com/intel-go/cpuid"

	"github.com/maoxs2/gxminer/manager"
)

func initWithJson(rawJson string) *manager.Manager {
	log.Println("Parsing config from `config.json` file")
	var conf manager.UserConfig
	err := json.Unmarshal([]byte(rawJson), &conf)
	if err != nil {
		log.Fatal("failed to parse the config file", err)
	}

	if strings.Contains(conf.Pools[0].Pool, ":") == false || len(strings.Split(conf.Pools[0].Pool, ":")) != 2 {
		log.Fatal("Pool address is wrong")
	}

	if conf.Workers.WorkerNum == 0 {
		for _, cache := range cpuid.CacheDescriptors {
			if cache.Level == 3 {
				conf.Workers.WorkerNum = conf.Workers.WorkerNum + uint32(cache.CacheSize/2/1024)
				if conf.Workers.WorkerNum == 0 {
					conf.Workers.WorkerNum = uint32(runtime.NumCPU())
				}
				log.Println("Set to", conf.Workers.WorkerNum, "workers by default")
			}
		}
	}

	m := manager.NewManager(BuildVersion, conf)
	m.Init()

	return m
}

func saveToJson(conf manager.UserConfig) {
	log.Println("Saving config to `config.json` file")

	f, err := os.OpenFile("config.json", os.O_WRONLY|os.O_CREATE, 0660)
	if err != nil {
		log.Fatal(err)
	}
	raw, _ := json.MarshalIndent(conf, "", "  ")
	_, _ = f.Write(raw)
	_ = f.Close()
}
