//go:build windows

package main

import "syscall"

func detachedProcessAttr() *syscall.SysProcAttr {
	return nil
}
