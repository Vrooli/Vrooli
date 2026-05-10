//go:build !windows

package runtimesupervisor

import "syscall"

func backgroundProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
