//go:build windows

package runtimesupervisor

import "syscall"

func backgroundProcessAttr() *syscall.SysProcAttr {
	return nil
}
