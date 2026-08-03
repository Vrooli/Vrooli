//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package runner

import "syscall"

func newProcessAttributes() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
