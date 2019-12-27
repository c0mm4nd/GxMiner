package main

import (
	"github.com/go-fastlog/fastlog"
	"os"
	"runtime"
	"strconv"

	"github.com/urfave/cli"
)

var log = fastlog.New(os.Stderr, "", fastlog.Ltime)

var BuildVersion = ""

var poolFlag = cli.StringFlag{
	Name:  "pool, o",
	Value: "",
	Usage: "Pool address with port, e.g. 192.168.1.100:3333 or mining.pool.com:3333",
}

var userFlag = cli.StringFlag{
	Name:  "user, u",
	Value: "",
	Usage: "Wallet address or login username",
}

var passFlag = cli.StringFlag{
	Name:  "pass, password, p",
	Value: "x",
	Usage: "Password for login username",
}

var rigIDFlag = cli.StringFlag{
	Name:  "rig-id",
	Value: "GxMiner",
	Usage: "RigID for Pool displaying",
}

var nicehashFlag = cli.BoolFlag{
	Name:  "nicehash",
	Usage: "enable rig's nicehash mining on pool/proxy",
}

var workerNumFlag = cli.UintFlag{
	Name:  "workerNum, threadNum, t",
	Usage: "the number of hash worker",
	Value: 0,
}

var HugePageFlag = cli.BoolTFlag{
	Name:  "huge-page",
	Usage: "on default enabled the huge/large page, set false to disable",
}

var HardAESFlag = cli.BoolTFlag{
	Name:  "hard-aes",
	Usage: "on default enabled the hardware aes, using soft aes set this to false",
}

var FullMemFlag = cli.BoolTFlag{
	Name:  "full-mem",
	Usage: "on default enabled the full mem, set false to disable",
}

var JitFlag = cli.BoolTFlag{
	Name:  "jit",
	Usage: "on default enabled the jit boost, set false to disable",
}

var Argon2SSE3Flag = cli.BoolTFlag{
	Name:  "argon2-sse3",
	Usage: "enable argon2-sse3",
}

var argon2AVX2Flag = cli.BoolTFlag{
	Name:  "argon2-avx2",
	Usage: "enable argon2-avx2",
}

var AffinityFlag = cli.StringFlag{
	Name:  "affinity-mask",
	Usage: "cpu affinity mask in hex",
	Value: strconv.FormatUint((1<<(runtime.NumCPU()))-1, 16),
}

var tlsFlag = cli.BoolFlag{
	Name:  "tls",
	Usage: "enable tls encryption in tcp transfer",
}

var configFileFlag = cli.StringFlag{
	Name:     "conf",
	Usage:    "Load configuration from `FILE`",
	FilePath: "config.json",
}

var logFileFlag = cli.StringFlag{
	Name:  "log-file",
	Usage: "save log messages to `FILE`",
}

var logLevelFlag = cli.StringFlag{
	Name:  "log-level",
	Usage: "log level (debug, info, warn, error, panic)",
	Value: "info",
}

var httpPortFlag = cli.UintFlag{
	Name:  "http-port",
	Usage: "serve port on `PORT`",
	Value: 2333,
}

var httpExternalFlag = cli.BoolFlag{
	Name:  "http-external",
	Usage: "expose port on the external env",
}

func main() {
	app := cli.NewApp()
	app.Name = "GxMiner"
	app.Usage = "Go randomX Miner"
	app.Version = "v0.2.2-" + BuildVersion + "-" + runtime.Version()
	app.Description = "GxMiner is a highly optimized miner for random-x series algorithm. Make sure you have downloaded from the official page[https://github.com/maoxs2/gxminer]. If you have any problem or advice please take the issue here[https://github.com/maoxs2/gxminer/issues/new] "
	app.Author = "Command M"
	app.Email = "maoxs2@163.com"

	var flags = []cli.Flag{
		configFileFlag,
		logFileFlag, logLevelFlag,
		poolFlag, userFlag, passFlag, rigIDFlag, nicehashFlag,
		workerNumFlag,

		HardAESFlag, FullMemFlag, JitFlag, HugePageFlag, Argon2SSE3Flag, argon2AVX2Flag,
		AffinityFlag, tlsFlag,

		httpPortFlag, httpExternalFlag,
	}
	app.Flags = flags

	app.Action = entrance

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}

func entrance(c *cli.Context) error {
	runtime.GOMAXPROCS(runtime.NumCPU() * 2)

	if c.String("conf") == "" {
		if c.String("pool") != "" {
			initWithFlags(c)
		} else {
			initWithSetup()
		}
	} else {
		initWithJson(c.String("conf"))
	}

	select {}
}
