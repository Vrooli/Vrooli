//go:build windows

package setup

import "syscall"

func detachedProcessAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{}
}
