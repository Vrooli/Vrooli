package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// errLoopAlreadyRunning is intentionally separate from an IO failure. A
// second invocation is a harmless operator mistake and must not make systemd
// escalate the healthy supervisor as if it had failed.
var errLoopAlreadyRunning = errors.New("vrooli autoheal loop is already running")

var errNativeLockUnavailable = errors.New("native loop lock unavailable")

// loopLock keeps one heartbeat writer and one lifecycle supervisor per host.
// The lock file remains on disk for diagnostics; native advisory locking
// releases it automatically if the owning process crashes.
type loopLock struct {
	release func()
	once    sync.Once
}

func acquireLoopLock(statusPath string) (*loopLock, error) {
	path := filepath.Join(filepath.Dir(statusPath), "loop.lock")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create loop lock directory: %w", err)
	}
	release, err := acquireNativeLoopLock(path)
	if err != nil {
		if errors.Is(err, errNativeLockUnavailable) {
			return nil, fmt.Errorf("%w: %s", errLoopAlreadyRunning, path)
		}
		return nil, fmt.Errorf("lock loop state: %w", err)
	}
	return &loopLock{release: release}, nil
}

func (lock *loopLock) close() {
	if lock == nil {
		return
	}
	lock.once.Do(func() {
		lock.release()
	})
}
