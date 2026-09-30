//go:build !windows

package cliutil

import "syscall"

func isPIDRunning(pid int) bool {
	return pid > 0 && syscall.Kill(pid, 0) == nil
}
