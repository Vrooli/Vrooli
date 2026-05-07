// Package hostsem implements a host-wide cross-process counting semaphore
// backed by N flock(2)'d slot files. It exists so that resource gateway CLIs
// (e.g. resource-ollama gateway embed) can serialize concurrent invocations
// from many scenarios against a single shared upstream daemon, without any
// of the participating processes needing to know about each other.
//
// Acquire scans the slot files round-robin, attempting LOCK_EX|LOCK_NB on
// each. The first one that succeeds becomes the caller's slot until release
// is called. If all slots are busy, Acquire blocks (with backoff) until a
// slot frees up or the supplied context is cancelled.
//
// Linux-only by design — Vrooli targets Linux per project memory. The
// underlying syscalls are gated by the standard `unix` build constraints.
package hostsem

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Semaphore is a cross-process counting semaphore.
//
// Construct via New; reuse a single instance across goroutines. The instance
// holds no kernel state itself — every Acquire opens a fresh fd and the
// kernel-side lock lives on that fd until release runs.
type Semaphore struct {
	dir   string
	slots int
}

// New prepares a Semaphore rooted at lockDir with the given number of slots.
// lockDir is created (recursively, mode 0o755) if missing. slots must be > 0.
func New(lockDir string, slots int) (*Semaphore, error) {
	if slots <= 0 {
		return nil, fmt.Errorf("hostsem: slots must be > 0, got %d", slots)
	}
	if err := os.MkdirAll(lockDir, 0o755); err != nil {
		return nil, fmt.Errorf("hostsem: prepare lock dir %s: %w", lockDir, err)
	}
	return &Semaphore{dir: lockDir, slots: slots}, nil
}

// Slots returns the configured slot count.
func (s *Semaphore) Slots() int { return s.slots }

// Acquire blocks until a slot is held or ctx is cancelled. The returned
// release function MUST be called exactly once to free the slot — defer it
// at the call site.
func (s *Semaphore) Acquire(ctx context.Context) (func(), error) {
	if ctx == nil {
		ctx = context.Background()
	}
	backoff := 5 * time.Millisecond
	const maxBackoff = 100 * time.Millisecond
	for {
		release, ok, err := s.tryAcquireOnce()
		if err != nil {
			return nil, err
		}
		if ok {
			return release, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < maxBackoff {
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}
}

func (s *Semaphore) tryAcquireOnce() (func(), bool, error) {
	for i := 0; i < s.slots; i++ {
		path := filepath.Join(s.dir, fmt.Sprintf("slot-%d.lock", i))
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return nil, false, fmt.Errorf("hostsem: open slot %d: %w", i, err)
		}
		if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
			_ = f.Close()
			if err == syscall.EWOULDBLOCK {
				continue
			}
			return nil, false, fmt.Errorf("hostsem: flock slot %d: %w", i, err)
		}
		release := func() {
			_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
			_ = f.Close()
		}
		return release, true, nil
	}
	return nil, false, nil
}
