package main

import (
	"github.com/maoxs2/gxminer/logger"
	"github.com/maoxs2/gxminer/manager"
	"github.com/maoxs2/gxminer/worker"
)

func getDefaultConfig() manager.UserConfig {
	return manager.UserConfig{
		Pools:   nil,
		Workers: worker.Config{},
		Log:     logger.LogConfig{},
		Http:    manager.HttpConfig{},
		//Master:  manager.MasterConfig{},
		//Slave:   manager.SlaveConfig{},
	}
}
