//go:build !windows

package usagemode

import (
	"fmt"
	"os"
	"syscall"
	"time"
)

// lockTimeout bounds how long a writer waits for another writer.
const lockTimeout = 10 * time.Second

// lockPath takes an exclusive advisory lock on a sidecar file (not the mode
// file, which writers replace by rename) and returns its release.
func lockPath(path string) (func(), error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	deadline := time.Now().Add(lockTimeout)
	for {
		err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			return func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN); _ = f.Close() }, nil
		}
		if err != syscall.EWOULDBLOCK || time.Now().After(deadline) {
			_ = f.Close()
			if err == syscall.EWOULDBLOCK {
				return nil, fmt.Errorf("usage mode file is locked by another writer")
			}
			return nil, err
		}
		time.Sleep(10 * time.Millisecond)
	}
}
