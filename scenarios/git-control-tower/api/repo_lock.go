package main

import (
	"context"
	"sync"
)

// RepoLock provides per-repository mutual exclusion for git operations.
//
// Git commands that touch the index (status, add, commit, reset, checkout, etc.)
// contend for .git/index.lock. When concurrent HTTP handlers or background
// goroutines issue these commands simultaneously, the loser gets:
//
//	fatal: Unable to create '.git/index.lock': File exists.
//
// RepoLock serializes these operations at the application level, eliminating
// the race window entirely. Each repository path gets its own channel-based
// mutex, so operations on different repositories proceed independently.
type RepoLock struct {
	mu    sync.Mutex
	locks map[string]chan struct{}
}

// NewRepoLock creates a new RepoLock instance.
func NewRepoLock() *RepoLock {
	return &RepoLock{
		locks: make(map[string]chan struct{}),
	}
}

// getOrCreate returns the channel-based mutex for the given repo path,
// creating one if it doesn't exist yet. The channel has capacity 1 and
// starts with a token, acting as a mutex: receiving the token = lock,
// sending the token back = unlock.
func (rl *RepoLock) getOrCreate(repoDir string) chan struct{} {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	ch, ok := rl.locks[repoDir]
	if !ok {
		ch = make(chan struct{}, 1)
		ch <- struct{}{} // Starts unlocked
		rl.locks[repoDir] = ch
	}
	return ch
}

// Acquire blocks until the lock for repoDir is available or ctx is cancelled.
// On success, returns an unlock function (safe to call multiple times).
// On context cancellation, returns nil and the context error.
func (rl *RepoLock) Acquire(ctx context.Context, repoDir string) (unlock func(), err error) {
	ch := rl.getOrCreate(repoDir)

	select {
	case <-ch:
		var once sync.Once
		return func() {
			once.Do(func() { ch <- struct{}{} })
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
