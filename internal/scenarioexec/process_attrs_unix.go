//go:build !windows

package scenarioexec

import "syscall"

func detachedProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
