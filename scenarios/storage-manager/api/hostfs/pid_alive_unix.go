//go:build !windows

package hostfs

import (
	"os"
	"syscall"
)

// pidAlive reports whether pid names a live process.
//
// Signal 0 performs the permission and existence checks without delivering
// anything. EPERM means the process exists and belongs to another user, which
// is still very much alive.
func pidAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}
	return err == os.ErrPermission || err == syscall.EPERM
}
