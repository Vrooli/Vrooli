//go:build windows

package main

import "syscall"

func androidSDKDetachedProcessAttrs() *syscall.SysProcAttr {
	return nil
}
