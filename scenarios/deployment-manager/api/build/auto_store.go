package build

import "sync"

// AutoBuildStore keeps track of async auto-builds in memory.
type AutoBuildStore struct {
	mu   sync.RWMutex
	jobs map[string]*AutoBuildStatus
}

// NewAutoBuildStore creates a new build store.
func NewAutoBuildStore() *AutoBuildStore {
	return &AutoBuildStore{
		jobs: make(map[string]*AutoBuildStatus),
	}
}

// Save stores a new build status.
func (s *AutoBuildStore) Save(status *AutoBuildStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[status.BuildID] = status
}

// Get retrieves a build status by ID.
func (s *AutoBuildStore) Get(id string) (*AutoBuildStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.jobs[id]
	if !ok {
		return nil, false
	}
	return cloneAutoBuildStatus(status), true
}

// Update mutates a build status in place.
func (s *AutoBuildStore) Update(id string, fn func(status *AutoBuildStatus)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.jobs[id]
	if !ok {
		return false
	}
	fn(status)
	return true
}

func cloneAutoBuildStatus(status *AutoBuildStatus) *AutoBuildStatus {
	if status == nil {
		return nil
	}
	cloned := *status
	if len(status.Targets) > 0 {
		cloned.Targets = make([]AutoBuildTargetStatus, len(status.Targets))
		for i, target := range status.Targets {
			cloned.Targets[i] = AutoBuildTargetStatus{
				ID:     target.ID,
				Folder: target.Folder,
			}
			if len(target.Platforms) > 0 {
				cloned.Targets[i].Platforms = make([]AutoBuildPlatformStatus, len(target.Platforms))
				copy(cloned.Targets[i].Platforms, target.Platforms)
			}
		}
	}
	if len(status.BuildLog) > 0 {
		cloned.BuildLog = append([]string{}, status.BuildLog...)
	}
	if len(status.ErrorLog) > 0 {
		cloned.ErrorLog = append([]string{}, status.ErrorLog...)
	}
	return &cloned
}
