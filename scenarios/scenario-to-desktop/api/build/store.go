package build

import "sync"

// InMemoryStore provides a thread-safe in-memory build status store.
type InMemoryStore struct {
	mu        sync.RWMutex
	statusMap map[string]*Status
}

// StoreOption configures an InMemoryStore.
type StoreOption func(*InMemoryStore)

// NewStore creates a new in-memory build store.
func NewStore(opts ...StoreOption) *InMemoryStore {
	s := &InMemoryStore{
		statusMap: make(map[string]*Status),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Save inserts or replaces a build status.
func (s *InMemoryStore) Save(status *Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusMap[status.BuildID] = status
}

// Get returns the status for the given build if it exists.
func (s *InMemoryStore) Get(id string) (*Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.statusMap[id]
	return status, ok
}

// Update executes fn while holding a write lock on the requested build.
// It returns false when the build ID is unknown.
func (s *InMemoryStore) Update(id string, fn func(status *Status)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.statusMap[id]
	if !ok {
		return false
	}
	fn(status)
	return true
}

// UpdatePlatform updates a specific platform's build result within a build.
func (s *InMemoryStore) UpdatePlatform(buildID, platform string, fn func(status *Status, result *PlatformResult)) bool {
	return s.Update(buildID, func(status *Status) {
		result, ok := status.PlatformResults[platform]
		if !ok {
			result = &PlatformResult{Status: "failed", SkipReason: "platform not initialized"}
			status.PlatformResults[platform] = result
		}
		fn(status, result)
	})
}

// Snapshot returns a shallow copy of the current build status map.
func (s *InMemoryStore) Snapshot() map[string]*Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]*Status, len(s.statusMap))
	for id, status := range s.statusMap {
		snapshot[id] = status
	}
	return snapshot
}

// Len reports how many builds are tracked.
func (s *InMemoryStore) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.statusMap)
}
