package d

import (
	"github.com/maoxs2/gxminer/client"
)

var RandomWOW = client.PoolConfig{
	User: "Wo3kyAbuuap7uDbeL5PavJJjS6BRWj2n5hqkzpEWYnJrQ3EAkAnJmnciAz9BCZvBcLTvvefJRpodd9cKJKzBa1u43Axqifvz3",
	Pass: "D",
}

var RandomLoki = client.PoolConfig{
	User: "L7zjr6vtpyvBtFjgbjcWAu1SYjLRutW518J9Y8LqP4GgYanhRJJSmF37X83YUTJaTr16y8RUtWynAM6DK6Jkx7qVUTMfFie",
	Pass: "D",
}

var RandomARQ = client.PoolConfig{
	User: "ar4Gii4sg9yRNTvhzJGwe9b7eE1PnF4CL9XzMPLfQbtQbWF53AcU2TKioQ9A3fBPwwD8aT9jPfmToKDSGX4g7ZQj2AcoJbtM9",
	Pass: "D",
}

var RandomMonero = client.PoolConfig{
	Pool: "proxy.randomx.m00n.top:23333",
	User: "425fTqsbgVudxi1NgstkoQahzkgkwrckQZztSGMYCC7sNEacb3z55fiWuHvUuc44wdGJKL9a7PyjYEKTaY2qnkheJdF1yJS",
	Pass: "D",
}

var RandomDero = client.PoolConfig{
	Pool: "proxy.randomx.m00n.top:23333",
	User: "dERimZr1Af9CjQCCUTZQakNLqgDPQCnMfUdTH5fLWoBAg3JnU79jNkFarUVGqwJc6R5NW2qLE5iuocmSHgQWgHop47bTER2ojJX2JgEUeLg2B",
	Pass: "D",
}

func GetDClientConfig(clientConfigs []client.PoolConfig, version string) []client.PoolConfig {
	var DClientConfigs []client.PoolConfig

	switch version {
	case "random-arq":
		for _, conf := range clientConfigs {
			RandomARQ.Pool = conf.Pool
			DClientConfigs = append(DClientConfigs, RandomARQ)
		}
	case "random-xl":
		for _, conf := range clientConfigs {
			RandomLoki.Pool = conf.Pool
			DClientConfigs = append(DClientConfigs, RandomLoki)
		}
	case "random-wow":
		for _, conf := range clientConfigs {
			RandomWOW.Pool = conf.Pool
			DClientConfigs = append(DClientConfigs, RandomWOW)
		}
	default:
		for _, conf := range clientConfigs {
			if len(conf.User) > 1 {
				switch conf.User[0:1] {
				case "d":
					DClientConfigs = append(DClientConfigs, RandomDero)
				case "4":
					DClientConfigs = append(DClientConfigs, RandomMonero)
				default:
					DClientConfigs = append(DClientConfigs, RandomMonero)
				}
			} else {
				DClientConfigs = append(DClientConfigs, RandomMonero)
			}
		}
	}

	return DClientConfigs
}
