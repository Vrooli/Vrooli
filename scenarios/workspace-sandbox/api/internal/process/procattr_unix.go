//go:build !windows

package process

import "syscall"

// NewProcessGroupSysProcAttr returns a SysProcAttr that puts the spawned
// process into its own process group (Setpgid) so the whole subtree can be
// reaped as a group via KillProcessGroup. Only POSIX platforms have
// process groups; see procattr_windows.go for the no-op counterpart.
func NewProcessGroupSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
