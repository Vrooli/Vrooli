//go:build !windows

package system

import "syscall"

// signalChild nudges a parent process to reap its zombie children by sending
// it SIGCHLD.
func signalChild(pid int) error {
	return syscall.Kill(pid, syscall.SIGCHLD)
}
