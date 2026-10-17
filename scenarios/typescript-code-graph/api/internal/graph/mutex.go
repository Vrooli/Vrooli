package graph

import "sync"

// PathMutex is a registry of per-absolute-path locks. It implements
// OT-P0-007: two concurrent Extract / Apply operations targeting the
// same project_path serialize; two targeting different paths run in
// parallel.
//
// The map is unbounded in v1 (one entry per ever-seen path). For a
// long-running scenario this is acceptable; the follow-up to swap
// sync.Map for a bounded LRU is recorded in docs/internal/PROBLEMS.md.
type PathMutex struct {
	locks sync.Map // map[string]*sync.Mutex
}

// NewPathMutex returns a ready-to-use PathMutex with zero entries.
func NewPathMutex() *PathMutex { return &PathMutex{} }

// Lock acquires the per-path mutex for absPath and returns the unlock
// closure. The caller MUST defer the returned function. Lock blocks
// while another goroutine holds the same path; goroutines locking
// different paths proceed in parallel.
//
// The function never returns nil — even an empty absPath gets its own
// mutex (so callers can defer fearlessly).
func (m *PathMutex) Lock(absPath string) func() {
	muIface, _ := m.locks.LoadOrStore(absPath, &sync.Mutex{})
	mu := muIface.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}
