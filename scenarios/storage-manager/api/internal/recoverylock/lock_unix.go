//go:build !windows

package recoverylock

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

// Acquire takes an advisory host lock and returns its release function.
func Acquire(path string) (func(), error) {
	return AcquireFor(path, fmt.Sprintf("pid:%d", os.Getpid()))
}

// AcquireFor takes the lock and records a human-readable holder id for
// diagnostics when another process arrives concurrently.
func AcquireFor(path, holder string) (func(), error) {
	if path == "" {
		return func() {}, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("create recovery lock directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open recovery lock: %w", err)
	}
	if err := syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		contents, _ := os.ReadFile(path)
		_ = file.Close()
		return nil, &LockHeldError{Holder: string(contents), Cause: err}
	}
	if err := file.Truncate(0); err == nil {
		_, _ = file.WriteString(holder)
		_, _ = file.Seek(0, 0)
	}
	return func() {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
		_ = file.Close()
	}, nil
}
