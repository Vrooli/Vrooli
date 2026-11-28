//go:build unix

package cliutil

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// acquireRefreshLock opens path and takes a non-blocking exclusive flock on it.
// It reports acquired=false, with no error, when another refresh holds the lock.
// The lock belongs to the open file description, so a child that inherits the
// file keeps it after the parent closes its copy.
func acquireRefreshLock(path string) (*os.File, bool, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, false, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return f, true, nil
}

// detachProcess puts the refresh in its own session so the hook runner killing
// the hook's process group on timeout does not kill the rebuild.
func detachProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}
