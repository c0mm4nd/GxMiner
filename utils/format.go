package utils

import "strconv"

func FormatHashrate(hs float64) string {
	if hs > 1000000 {
		return strconv.FormatFloat(hs/1000000, 'f', 2, 64) + "mh/s"
	}
	if hs > 1000 {
		return strconv.FormatFloat(hs/1000, 'f', 2, 64) + "kh/s"
	}
	return strconv.FormatFloat(hs, 'f', 2, 64) + "h/s"
}
