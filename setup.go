package main

import (
	"bufio"
	"fmt"
	"github.com/maoxs2/gxminer/logger"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/intel-go/cpuid"

	"github.com/maoxs2/gxminer/client"
	"github.com/maoxs2/gxminer/manager"
	"github.com/maoxs2/gxminer/worker"
)

func initWithSetup() *manager.Manager {
	log.Println("Parsing config from setup steps")
	r := bufio.NewScanner(os.Stdin)

	fmt.Print("Pool (e.g. mining.pool.com:3333): ")
	r.Scan()
	pool := r.Text()
	if strings.Contains(pool, ":") == false || len(strings.Split(pool, ":")) != 2 {
		log.Fatal("Pool address is wrong")
	}

	fmt.Print("User (wallet address or login name): ")
	r.Scan()
	user := r.Text()

	fmt.Print("Pass (acts as rig name in some pools, default: x): ")
	r.Scan()
	pass := r.Text()
	if pass == "" {
		pass = "x"
	}

	fmt.Print("Enable TLS? (y/[n]): ")
	r.Scan()
	yn := r.Text()

	var t = 0
	for _, cache := range cpuid.CacheDescriptors {
		if cache.Level == 3 {
			t = t + cache.CacheSize/2/1024
			if t == 0 {
				t = runtime.NumCPU()
			}
		}
	}

	fmt.Printf("WorkerNum/ThreadNum (suggestion: %d): ", t)
	r.Scan()
	str := r.Text()

	n, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		n = uint64(t)
	}

	fmt.Printf("CPU Affinity Mask (default all cores used: %X): ", (1<<(runtime.NumCPU()))-1)
	r.Scan()
	am := r.Text()

	if am == "" {
		am = strconv.FormatUint((1<<(runtime.NumCPU()))-1, 16)
	}

	clientConf := client.PoolConfig{
		Pool:  pool,
		User:  user,
		Pass:  pass,
		RigID: "",
		TLS:   yn == "y",
	}

	workerConf := worker.Config{
		WorkerNum:    uint32(n),
		InitNum:      2 * uint32(n),
		HardAES:      true,
		FullMem:      true,
		JIT:          true,
		HugePage:     true,
		Argon2SSE3:   cpuid.HasFeature(cpuid.SSE3),
		Argon2AVX2:   cpuid.HasExtendedFeature(cpuid.AVX2),
		AffinityMask: am,
	}

	conf := manager.UserConfig{
		Pools:   []client.PoolConfig{clientConf},
		Workers: workerConf,
		Log: logger.LogConfig{
			Level: "info",
			File:  "",
		},
		Http: manager.HttpConfig{
			Port:     2333,
			External: false,
		},
	}

	saveToJson(conf)
	m := manager.NewManager(BuildVersion, conf)
	m.Init()

	return m
}
