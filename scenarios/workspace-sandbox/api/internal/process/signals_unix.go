//go:build !windows

package process

import (
	"errors"
	"syscall"
)

// isProcessGone reports whether a kill error means the target already
// exited. On POSIX that is ESRCH ("no such process").
func isProcessGone(err error) bool {
	return errors.Is(err, syscall.ESRCH)
}

// sysGetpgid returns the process group ID for pid. Wraps syscall.Getpgid,
// which only POSIX platforms provide.
func sysGetpgid(pid int) (int, error) {
	return syscall.Getpgid(pid)
}

// sysKill sends sig to pid. A negative pid targets the process group, per
// POSIX kill(2) semantics. Wraps syscall.Kill.
func sysKill(pid int, sig syscall.Signal) error {
	return syscall.Kill(pid, sig)
}
