package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SmokeTestStore tracks smoke test statuses with safe concurrent access.
type SmokeTestStore struct {
	mu        sync.RWMutex
	statusMap map[string]*SmokeTestStatus
	path      string
}

func NewSmokeTestStore(path string) (*SmokeTestStore, error) {
	store := &SmokeTestStore{
		statusMap: make(map[string]*SmokeTestStatus),
		path:      path,
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *SmokeTestStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.path == "" {
		return errors.New("smoke test store path is empty")
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

	var statuses []*SmokeTestStatus
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

func (s *SmokeTestStore) persist() error {
	if s.path == "" {
		return errors.New("smoke test store path is empty")
	}
	var statuses []*SmokeTestStatus
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
func (s *SmokeTestStore) Save(status *SmokeTestStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statusMap[status.SmokeTestID] = status
	_ = s.persist()
}

// Get returns the status for the given smoke test ID if it exists.
func (s *SmokeTestStore) Get(id string) (*SmokeTestStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.statusMap[id]
	return status, ok
}

// Update executes fn while holding a write lock on the requested smoke test.
// It returns false when the smoke test ID is unknown.
func (s *SmokeTestStore) Update(id string, fn func(status *SmokeTestStatus)) bool {
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

func (s *Server) setSmokeTestCancel(id string, cancel context.CancelFunc) {
	s.smokeTestMux.Lock()
	defer s.smokeTestMux.Unlock()
	s.smokeTestCancels[id] = cancel
}

func (s *Server) takeSmokeTestCancel(id string) context.CancelFunc {
	s.smokeTestMux.Lock()
	defer s.smokeTestMux.Unlock()
	cancel, ok := s.smokeTestCancels[id]
	if !ok {
		return nil
	}
	delete(s.smokeTestCancels, id)
	return cancel
}

func (s *Server) clearSmokeTestCancel(id string) {
	s.smokeTestMux.Lock()
	defer s.smokeTestMux.Unlock()
	delete(s.smokeTestCancels, id)
}
