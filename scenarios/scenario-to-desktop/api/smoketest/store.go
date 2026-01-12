package smoketest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// FileStore provides a persistent smoke test store backed by a JSON file.
type FileStore struct {
	mu        sync.RWMutex
	statusMap map[string]*Status
	path      string
}

// StoreOption configures a FileStore.
type StoreOption func(*FileStore)

// WithPath sets the storage path.
func WithPath(path string) StoreOption {
	return func(s *FileStore) {
		s.path = path
	}
}

// NewStore creates a new smoke test store with the given path.
func NewStore(path string, opts ...StoreOption) (*FileStore, error) {
	store := &FileStore{
		statusMap: make(map[string]*Status),
		path:      path,
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
	return nil
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
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write smoke test store: %w", err)
	}
	return nil
}

// Save inserts or replaces a smoke test status.
func (s *FileStore) Save(status *Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusMap[status.SmokeTestID] = status
	_ = s.persist()
}

// Get returns the status for the given smoke test ID if it exists.
func (s *FileStore) Get(id string) (*Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.statusMap[id]
	return status, ok
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
	_ = s.persist()
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
