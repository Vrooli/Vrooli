// DOC: docs/reference/smoke-test-pipeline.md#storage--persistence
package smoketest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// MaxFinishedPerScenario bounds the persisted history: only the newest
// finished smoke tests per scenario are kept. Unfinished records are never
// trimmed.
const MaxFinishedPerScenario = 50

// interruptedReason is recorded on runs whose owning process exited before
// they reached a terminal status.
const interruptedReason = "Smoke test interrupted by server restart"

// isTerminal reports whether a smoke test status is final.
func isTerminal(status string) bool {
	return status == "passed" || status == "failed"
}

// FileStore provides a persistent smoke test store backed by a JSON file.
// See docs/reference/smoke-test-pipeline.md#storage--persistence for data structure.
type FileStore struct {
	mu        sync.RWMutex
	statusMap map[string]*Status
	path      string
	errWriter io.Writer // For logging persist errors; defaults to os.Stderr
	// interrupted counts records reconciled at load because the process that
	// ran them is gone.
	interrupted int
}

// StoreOption configures a FileStore.
type StoreOption func(*FileStore)

// WithPath sets the storage path.
func WithPath(path string) StoreOption {
	return func(s *FileStore) {
		s.path = path
	}
}

// WithErrorWriter sets the writer for persist error logs.
func WithErrorWriter(w io.Writer) StoreOption {
	return func(s *FileStore) {
		s.errWriter = w
	}
}

// NewStore creates a new smoke test store with the given path.
func NewStore(path string, opts ...StoreOption) (*FileStore, error) {
	store := &FileStore{
		statusMap: make(map[string]*Status),
		path:      path,
		errWriter: os.Stderr,
	}
	for _, opt := range opts {
		opt(store)
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

// NewInMemoryStore creates an in-memory store (no persistence).
func NewInMemoryStore() *FileStore {
	return &FileStore{
		statusMap: make(map[string]*Status),
		errWriter: os.Stderr,
	}
}

// load reads the store from disk.
func (s *FileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return nil // In-memory mode
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create smoke test directory: %w", err)
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read smoke test store: %w", err)
	}

	var statuses []*Status
	if err := json.Unmarshal(data, &statuses); err != nil {
		return fmt.Errorf("parse smoke test store: %w", err)
	}
	for _, status := range statuses {
		if status == nil || status.SmokeTestID == "" {
			continue
		}
		s.statusMap[status.SmokeTestID] = status
	}
	// This process is the store's only writer, so a non-terminal record found
	// at load belongs to a run whose goroutine died with the previous process.
	// Left as "running" it would claim liveness (and protect staging) forever.
	now := time.Now().UTC()
	for _, status := range s.statusMap {
		if isTerminal(status.Status) {
			continue
		}
		status.Status = "failed"
		status.Error = interruptedReason
		status.CurrentState = StateFailed
		status.CompletedAt = &now
		status.Logs = append(status.Logs, "FAILED: "+interruptedReason)
		s.interrupted++
	}
	trimmed := s.trimFinished()
	if s.interrupted > 0 || trimmed > 0 {
		if err := s.persist(); err != nil {
			fmt.Fprintf(s.errWriter, "[smoke-test-store] persist failed: %v\n", err)
		}
	}
	return nil
}

// Interrupted returns how many unfinished runs were marked failed at load
// because the process that owned them no longer exists.
func (s *FileStore) Interrupted() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interrupted
}

// trimFinished drops finished records beyond MaxFinishedPerScenario per
// scenario, oldest first. Callers must hold the write lock.
func (s *FileStore) trimFinished() int {
	finished := map[string][]*Status{}
	for _, status := range s.statusMap {
		if isTerminal(status.Status) {
			finished[status.ScenarioName] = append(finished[status.ScenarioName], status)
		}
	}
	removed := 0
	for _, list := range finished {
		if len(list) <= MaxFinishedPerScenario {
			continue
		}
		sort.Slice(list, func(i, j int) bool { return list[i].StartedAt.After(list[j].StartedAt) })
		for _, status := range list[MaxFinishedPerScenario:] {
			delete(s.statusMap, status.SmokeTestID)
			removed++
		}
	}
	return removed
}

// persist writes the store to disk.
func (s *FileStore) persist() error {
	if s.path == "" {
		return nil // In-memory mode
	}
	var statuses []*Status
	for _, status := range s.statusMap {
		statuses = append(statuses, status)
	}
	data, err := json.MarshalIndent(statuses, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal smoke test store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o600); err != nil {
		return fmt.Errorf("write smoke test store: %w", err)
	}
	return nil
}

// Save inserts or replaces a smoke test status.
func (s *FileStore) Save(status *Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusMap[status.SmokeTestID] = status
	s.trimFinished()
	if err := s.persist(); err != nil {
		// Log but don't fail the operation - in-memory state is authoritative
		fmt.Fprintf(s.errWriter, "[smoke-test-store] persist failed: %v\n", err)
	}
}

// Get returns the status for the given smoke test ID if it exists.
func (s *FileStore) Get(id string) (*Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.statusMap[id]
	return status, ok
}

// GetByScenario returns the most recent smoke test for a given scenario name.
// It returns the status and true if found, nil and false otherwise.
func (s *FileStore) GetByScenario(scenarioName string) (*Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *Status
	for _, st := range s.statusMap {
		if st.ScenarioName != scenarioName {
			continue
		}
		if best == nil || st.StartedAt.After(best.StartedAt) {
			best = st
		}
	}
	if best == nil {
		return nil, false
	}
	return best, true
}

// Update executes fn while holding a write lock on the requested smoke test.
// It returns false when the smoke test ID is unknown.
func (s *FileStore) Update(id string, fn func(status *Status)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.statusMap[id]
	if !ok {
		return false
	}
	fn(status)
	if err := s.persist(); err != nil {
		// Log but don't fail the operation - in-memory state is authoritative
		fmt.Fprintf(s.errWriter, "[smoke-test-store] persist failed: %v\n", err)
	}
	return true
}

// DefaultCancelManager manages smoke test cancellation functions.
type DefaultCancelManager struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// NewCancelManager creates a new cancel manager.
func NewCancelManager() *DefaultCancelManager {
	return &DefaultCancelManager{
		cancels: make(map[string]context.CancelFunc),
	}
}

// SetCancel registers a cancellation function for a smoke test.
func (m *DefaultCancelManager) SetCancel(id string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancels[id] = cancel
}

// TakeCancel retrieves and removes the cancellation function for a smoke test.
func (m *DefaultCancelManager) TakeCancel(id string) context.CancelFunc {
	m.mu.Lock()
	defer m.mu.Unlock()
	cancel, ok := m.cancels[id]
	if !ok {
		return nil
	}
	delete(m.cancels, id)
	return cancel
}

// Clear removes the cancellation function without calling it.
func (m *DefaultCancelManager) Clear(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancels, id)
}

// HasActiveForScenario protects staging used by any running smoke test.
// Unfinished records from earlier processes are reconciled at load, so a
// non-terminal record here is a run owned by this process.
func (s *FileStore) HasActiveForScenario(scenario string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, status := range s.statusMap {
		if status != nil && status.ScenarioName == scenario && !isTerminal(status.Status) {
			return true
		}
	}
	return false
}
