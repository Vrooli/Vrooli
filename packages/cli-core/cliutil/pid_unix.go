//go:build !windows

package cliutil

import "syscall"

func pidCheckAlive(err error) bool {
	return err == nil || err == syscall.EPERM
}

func isPIDRunning(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return pidCheckAlive(err)
}
