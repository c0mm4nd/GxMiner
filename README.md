# GxMiner

GxMiner is a highly optimized miner for random-x series algorithm

GxMiner,高度优化,专精RandomX算法的新型挖矿软件。

## Intro

GxMiner acts as a subproejct of project go-randomx which based on C and golang. In this framework, we can mine all cryptocurrencies with random-x series algorithm.

This repo is currently not fully open-sourcing, but its core part, the project go-randomx is open-sourcing, if you wanna help boost GxMiner's speed you can directly post PR to the go-randomx

We plan to open-source after monero's fork on 30 Nov.

## Why GxMiner

As everyone know, xmrig & xmr-stak is the leader of monero miners. But soon monero is not cryptonight algorithm cryptocurrency any longer, it would be the centry of random-x

Comparing to the leaders, GxMiner is **younger** and **modern**, **without any historical burden**. And GxMiner is not slower even sometimes slightly **faster** than the xmrig.

And if you are a developer, it would be much **easier to intergrate** your random-x fork into miner.

## Usage

Take RandomXL(Loki) for example:

```bash
NAME:
   GxMiner - Go randomX Miner

USAGE:
   gxminer.exe [global options] command [command options] [arguments...]

VERSION:
   v0.1.1-random-xl-go1.13.1

DESCRIPTION:
   GxMiner is a highly optimized miner for random-x series algorithm. Make sure you have downloaded from the official page[https://github.com/maoxs2/gxminer]. If you have any problem or advice please take the issue here[https://github.com/maoxs2/open-grin-pool/issues/new]

AUTHOR:
   Command M <maoxs2@163.com>

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --pool value, -o value                          pool address with port, e.g. 192.168.1.100:3333 or mining.pool:3333
   --username value, -u value                      wallet address or login username, e.g. L7zjr6vtpyvBtFjgbjcWAu1SYjLRutW518J9Y8LqP4GgYanhRJJSmF37X83YUTJaTr16y8RUtWynAM6DK6Jkx7qVUTMfFie
   --password value, -p value                      password for login username (default: "x")
   --rigid value                                   rigID for pool displaying (default: "GxMiner")
   --workerNum value, --threadNum value, -t value  the number of hash worker (default: 0)
   --hard-aes                                      on default enabled the hardware aes, using soft aes set this to false
   --full-mem                                      on default enabled the full mem, set false to disable
   --jit                                           on default enabled the jit boost, set false to disable
   --huge-page                                     unavailable, dont enable it on Windows
   --affinity-mask value                           cpu affinity mask in hex (default: "F")
   --help, -h                                      show help
   --version, -v                                   print the version

```

Loki mining example:

```bash
gxminer.exe -o 118.24.119.46:30000 -u L7zjr6vtpyvBtFjgbjcWAu1SYjLRutW518J9Y8LqP4GgYanhRJJSmF37X83YUTJaTr16y8RUtWynAM6DK6Jkx7qVUTMfFie
```

## Build

[After open-sourcing]

## FAQ

- start with "failed to alloc mem for dataset" error

1. Check your platform support large/huge page or not. if not, set `--huge-page=false`

2. Check whether you have enough page. If not, clear it.

## Hashrate Comparition

### RandomXL

Dual-E5-2660v2:

![GxMiner-v0.1.1-windows](./comparations/RandomXL/Dual-E5-2660v2/GxMiner-v0.1.1-windows.png)

![xmrig-v2.99.3-windows](./comparations/RandomXL/Dual-E5-2660v2/xmrig-v2.99.3-windows.png)

It's welcomed that [share your hashrate/comparation on github issue](https://github.com/maoxs2/open-grin-pool/issues/new).
