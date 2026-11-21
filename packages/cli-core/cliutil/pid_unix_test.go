//go:build !windows

package cliutil

import (
	"syscall"
	"testing"
)

func TestPIDCheckTreatsEPERMAsAlive(t *testing.T) {
	if !pidCheckAlive(syscall.EPERM) {
		t.Fatal("EPERM means the process exists but cannot be signaled")
	}
	if pidCheckAlive(syscall.ESRCH) {
		t.Fatal("ESRCH must mean the process is not alive")
	}
}
