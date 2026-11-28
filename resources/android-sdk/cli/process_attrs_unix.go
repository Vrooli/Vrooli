//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package main

import "syscall"

func androidSDKDetachedProcessAttrs() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setsid: true}
}
