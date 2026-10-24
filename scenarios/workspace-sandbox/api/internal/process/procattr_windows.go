//go:build windows

package process

import "syscall"

// NewProcessGroupSysProcAttr returns nil on Windows: there are no POSIX
// process groups, so the process starts with default attributes and
// teardown falls back to single-process termination (see sysKill).
func NewProcessGroupSysProcAttr() *syscall.SysProcAttr {
	return nil
}
