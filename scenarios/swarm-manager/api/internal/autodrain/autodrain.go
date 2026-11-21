// Package autodrain persists the continuous goal-directed auto-enqueue toggle
// (plan D4, default OFF). It is a swarm-manager-local flag — deliberately NOT a
// governance/proto setting — so goal-directed execution stays inside the
// scenario change boundary. When enabled, the execution poller continuously
// enqueues ready goal items through the governed QueueBacklog path.
package autodrain

import (
	"path/filepath"
	"sync"

	"swarm-manager/internal/storage"
)

// stateFileName is the flag file under the data root.
const stateFileName = "auto-drain.json"

// State is the persisted toggle.
type State struct {
	Enabled bool `json:"enabled"`
}

// Store persists the auto-drain toggle at {dataRoot}/auto-drain.json.
type Store struct {
	path string
	mu   sync.RWMutex
}

// NewStore creates a Store rooted at the given data root.
func NewStore(dataRoot string) *Store {
	return &Store{path: filepath.Join(dataRoot, stateFileName)}
}

// Load reads the toggle, defaulting to false (OFF) when the file is absent.
func (s *Store) Load() (State, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var st State
	if _, err := storage.ReadJSON(s.path, &st); err != nil {
		return State{}, err
	}
	return st, nil
}

// Save persists the toggle.
func (s *Store) Save(st State) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return storage.WriteJSONAtomic(s.path, st)
}

// AutoDrainEnabled reports whether continuous auto-enqueue is on, tolerating a
// read error by reporting off. Satisfies execution.AutoDrainProvider.
func (s *Store) AutoDrainEnabled() bool {
	st, err := s.Load()
	if err != nil {
		return false
	}
	return st.Enabled
}
