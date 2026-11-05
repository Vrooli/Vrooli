package main

import (
	"os"
	"syscall"
)

func processRSSKB(state *os.ProcessState) int64 {
	if usage, ok := state.SysUsage().(*syscall.Rusage); ok {
		return usage.Maxrss / 1024
	}
	return 0
}
