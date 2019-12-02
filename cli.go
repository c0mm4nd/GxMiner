package main

import (
	"github.com/maoxs2/gxminer/logger"
	"runtime"
	"strconv"
	"strings"

	"github.com/intel-go/cpuid"
	"github.com/urfave/cli"

	"github.com/maoxs2/gxminer/client"
	"github.com/maoxs2/gxminer/manager"
	"github.com/maoxs2/gxminer/worker"
)

func initWithFlags(c *cli.Context) *manager.Manager {
	log.Println("Parsing config from CLI")
	if strings.Contains(c.String("pool"), ":") == false || len(strings.Split(c.String("pool"), ":")) != 2 {
		log.Fatal("Pool address is wrong")
	}

	clientConf := client.PoolConfig{
		Pool:  c.String("pool"),
		User:  c.String("user"),
		Pass:  c.String("pass"),
		RigID: c.String("rig-id"),
		TLS:   c.Bool("tls"),
	}

	if c.Uint("t") == 0 {
		for _, cache := range cpuid.CacheDescriptors {
			if cache.Level == 3 {
				_ = c.Set("t", strconv.FormatUint(uint64(c.Uint("t")+uint(cache.CacheSize/2/1024)), 10))
				if c.Uint("t") == 0 {
					_ = c.Set("t", strconv.FormatUint(uint64(runtime.NumCPU()), 10))
				}
				log.Println("Set to", c.Uint("t"), "workers by default")
			}
		}
	}

	workerConf := worker.Config{
		WorkerNum:    uint32(c.Uint("t")),
		InitNum:      2 * uint32(c.Uint("t")),
		HardAES:      c.BoolT("hard-aes"),
		FullMem:      c.BoolT("full-mem"),
		JIT:          c.BoolT("jit"),
		HugePage:     c.BoolT("huge-page"),
		Argon2SSE3:   c.BoolT("argon2-sse3") && cpuid.HasFeature(cpuid.SSE3),
		Argon2AVX2:   c.BoolT("argon2-avx2") && cpuid.HasExtendedFeature(cpuid.AVX2),
		AffinityMask: c.String("affinity-mask"),
	}

	conf := manager.UserConfig{
		Pools:   []client.PoolConfig{clientConf},
		Workers: workerConf,
		Log: logger.LogConfig{
			Level: c.String("log-level"),
			File:  c.String("log-file"),
		},
		Http: manager.HttpConfig{
			Port:     c.Uint("http-port"),
			External: c.Bool("http-external"),
		},
	}

	saveToJson(conf)
	m := manager.NewManager(BuildVersion, conf)
	m.Init()

	return m
}
