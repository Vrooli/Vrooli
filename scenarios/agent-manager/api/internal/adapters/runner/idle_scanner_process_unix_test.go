//go:build !windows

package runner

import "syscall"

func managedProcessSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
