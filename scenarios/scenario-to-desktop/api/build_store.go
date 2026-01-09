package main

import "sync"

// BuildStore centralizes access to build status tracking so callers do not need
// to manage locking or map mutations themselves.
type BuildStore struct {
	mu        sync.RWMutex
	statusMap map[string]*BuildStatus
}

func NewBuildStore() *BuildStore {
	return &BuildStore{
		statusMap: make(map[string]*BuildStatus),
	}
}

// Save inserts or replaces a build status.
func (b *BuildStore) Save(status *BuildStatus) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.statusMap[status.BuildID] = status
}

// Get returns the status for the given build if it exists.
func (b *BuildStore) Get(id string) (*BuildStatus, bool) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	status, ok := b.statusMap[id]
	return status, ok
}

// Update executes fn while holding a write lock on the requested build.
// It returns false when the build ID is unknown.
func (b *BuildStore) Update(id string, fn func(status *BuildStatus)) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	status, ok := b.statusMap[id]
	if !ok {
		return false
	}
	fn(status)
	return true
}

// Snapshot returns a shallow copy of the current build status map for safe iteration.
func (b *BuildStore) Snapshot() map[string]*BuildStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	snapshot := make(map[string]*BuildStatus, len(b.statusMap))
	for id, status := range b.statusMap {
		snapshot[id] = status
	}
	return snapshot
}

// Len reports how many builds are tracked.
func (b *BuildStore) Len() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.statusMap)
}
