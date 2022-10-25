package main

import (
	"crypto/sha1"
	"encoding/binary"
	"strings"
	"time"
)

func stringToUint64(s string) uint64 {
	hashed := sha1.Sum([]byte(s))
	return binary.BigEndian.Uint64(hashed[:])
}

func stringToColorCode(s string, codes []uint) uint {
	i := stringToUint64(s)
	idx := i % uint64(len(codes))
	return codes[idx]
}

func pad(s string, length int) string {
	if len(s) >= length {
		return s
	}
	output := strings.Repeat(" ", length-len(s)) + s
	return output
}

func roundDuration(d time.Duration) time.Duration {
	div := time.Duration(100)
	switch {
	case d > time.Second:
		d = d.Round(time.Second / div)
	case d > time.Millisecond:
		d = d.Round(time.Millisecond / div)
	case d > time.Microsecond:
		d = d.Round(time.Microsecond / div)
	}
	return d
}
