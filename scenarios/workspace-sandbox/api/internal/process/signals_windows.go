//go:build windows

package process

import (
	"errors"
	"os"
	"syscall"
)

// isProcessGone reports whether a kill error means the target already
// exited. On Windows os.Process.Kill returns os.ErrProcessDone for a
// process that has already finished.
func isProcessGone(err error) bool {
	return errors.Is(err, os.ErrProcessDone)
}

// sysGetpgid always fails on Windows: there are no POSIX process groups.
// Callers treat the error as "no group" and fall back to single-process
// termination.
func sysGetpgid(int) (int, error) {
	return 0, errors.ErrUnsupported
}

// sysKill terminates a single process on Windows, which has neither POSIX
// process groups nor signals. A negative pid (a group target on POSIX) is
// reduced to its absolute pid, and the signal is ignored since only a
// forced kill is available.
func sysKill(pid int, _ syscall.Signal) error {
	if pid < 0 {
		pid = -pid
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return err
	}
	return p.Kill()
}
