// Package persistence provides data storage for investigations.
package persistence

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"scenario-to-desktop-api/domain"
)

// InvestigationStore stores investigation records in memory with optional file persistence.
type InvestigationStore struct {
	mu       sync.RWMutex
	items    map[string]*domain.Investigation
	dataDir  string // If set, persists to this directory
	cancels  map[string]context.CancelFunc
	cancelMu sync.Mutex
}

// NewInvestigationStore creates a new investigation store.
// If dataDir is non-empty, investigations will be persisted to disk.
func NewInvestigationStore(dataDir string) *InvestigationStore {
	s := &InvestigationStore{
		items:   make(map[string]*domain.Investigation),
		dataDir: dataDir,
		cancels: make(map[string]context.CancelFunc),
	}
	if dataDir != "" {
		_ = s.loadFromDisk()
	}
	return s
}

// GenerateID creates a new unique investigation ID.
func GenerateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("inv-%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// Create inserts a new investigation record.
func (s *InvestigationStore) Create(inv *domain.Investigation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if inv.ID == "" {
		inv.ID = GenerateID()
	}
	if inv.CreatedAt.IsZero() {
		inv.CreatedAt = time.Now()
	}
	inv.UpdatedAt = time.Now()

	s.items[inv.ID] = inv
	s.persistOne(inv)
	return nil
}

// Get retrieves an investigation by ID.
func (s *InvestigationStore) Get(id string) (*domain.Investigation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inv, ok := s.items[id]
	if !ok {
		return nil, nil
	}
	return inv, nil
}

// GetForPipeline retrieves an investigation by ID scoped to a pipeline.
func (s *InvestigationStore) GetForPipeline(pipelineID, id string) (*domain.Investigation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	inv, ok := s.items[id]
	if !ok || inv.PipelineID != pipelineID {
		return nil, nil
	}
	return inv, nil
}

// List retrieves investigations for a pipeline, ordered by creation time (newest first).
func (s *InvestigationStore) List(pipelineID string, limit int) ([]*domain.Investigation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*domain.Investigation
	for _, inv := range s.items {
		if inv.PipelineID == pipelineID {
			result = append(result, inv)
		}
	}

	// Sort by created_at descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// ListAll returns all investigations, ordered by creation time (newest first).
func (s *InvestigationStore) ListAll(limit int) ([]*domain.Investigation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*domain.Investigation, 0, len(s.items))
	for _, inv := range s.items {
		result = append(result, inv)
	}

	// Sort by created_at descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}

	return result, nil
}

// GetActive returns the currently running investigation for a pipeline, if any.
func (s *InvestigationStore) GetActive(pipelineID string) (*domain.Investigation, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var latest *domain.Investigation
	for _, inv := range s.items {
		if inv.PipelineID != pipelineID {
			continue
		}
		if inv.Status != domain.InvestigationStatusPending && inv.Status != domain.InvestigationStatusRunning {
			continue
		}
		if latest == nil || inv.CreatedAt.After(latest.CreatedAt) {
			latest = inv
		}
	}
	return latest, nil
}

// Update modifies an investigation using a modifier function.
func (s *InvestigationStore) Update(id string, fn func(inv *domain.Investigation)) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	inv, ok := s.items[id]
	if !ok {
		return fmt.Errorf("investigation not found: %s", id)
	}

	fn(inv)
	inv.UpdatedAt = time.Now()
	s.persistOne(inv)
	return nil
}

// UpdateStatus updates the status of an investigation.
func (s *InvestigationStore) UpdateStatus(id string, status domain.InvestigationStatus) error {
	return s.Update(id, func(inv *domain.Investigation) {
		inv.Status = status
		if status == domain.InvestigationStatusCompleted ||
			status == domain.InvestigationStatusFailed ||
			status == domain.InvestigationStatusCancelled {
			now := time.Now()
			inv.CompletedAt = &now
		}
	})
}

// UpdateRunID sets the agent run ID for tracking.
func (s *InvestigationStore) UpdateRunID(id, runID string) error {
	return s.Update(id, func(inv *domain.Investigation) {
		inv.AgentRunID = &runID
	})
}

// UpdateProgress updates the progress percentage.
func (s *InvestigationStore) UpdateProgress(id string, progress int) error {
	return s.Update(id, func(inv *domain.Investigation) {
		inv.Progress = progress
	})
}

// UpdateFindings stores the investigation findings and details.
func (s *InvestigationStore) UpdateFindings(id, findings string, details json.RawMessage) error {
	return s.Update(id, func(inv *domain.Investigation) {
		inv.Findings = &findings
		inv.Details = details
		inv.Status = domain.InvestigationStatusCompleted
		inv.Progress = 100
		now := time.Now()
		inv.CompletedAt = &now
	})
}

// UpdateError sets an error message and marks the investigation as failed.
func (s *InvestigationStore) UpdateError(id, errorMsg string) error {
	return s.Update(id, func(inv *domain.Investigation) {
		inv.Status = domain.InvestigationStatusFailed
		inv.ErrorMessage = &errorMsg
		now := time.Now()
		inv.CompletedAt = &now
	})
}

// UpdateErrorWithDetails sets an error message and details, then marks as failed.
func (s *InvestigationStore) UpdateErrorWithDetails(id, errorMsg string, details json.RawMessage) error {
	return s.Update(id, func(inv *domain.Investigation) {
		inv.Status = domain.InvestigationStatusFailed
		inv.ErrorMessage = &errorMsg
		inv.Details = details
		now := time.Now()
		inv.CompletedAt = &now
	})
}

// Delete removes an investigation record.
func (s *InvestigationStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[id]; !ok {
		return fmt.Errorf("investigation not found: %s", id)
	}
	delete(s.items, id)
	s.deleteFromDisk(id)
	return nil
}

// Cleanup removes completed investigations older than the given duration.
func (s *InvestigationStore) Cleanup(olderThan time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for id, inv := range s.items {
		if inv.CompletedAt != nil && inv.CompletedAt.Before(cutoff) {
			delete(s.items, id)
			s.deleteFromDisk(id)
		}
	}
}

// SetCancel registers a cancellation function for an investigation.
func (s *InvestigationStore) SetCancel(id string, cancel context.CancelFunc) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	s.cancels[id] = cancel
}

// TakeCancel retrieves and removes the cancellation function.
func (s *InvestigationStore) TakeCancel(id string) context.CancelFunc {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	cancel, ok := s.cancels[id]
	if !ok {
		return nil
	}
	delete(s.cancels, id)
	return cancel
}

// ClearCancel removes a cancellation function without calling it.
func (s *InvestigationStore) ClearCancel(id string) {
	s.cancelMu.Lock()
	defer s.cancelMu.Unlock()
	delete(s.cancels, id)
}

// persistOne writes a single investigation to disk if dataDir is set.
func (s *InvestigationStore) persistOne(inv *domain.Investigation) {
	if s.dataDir == "" {
		return
	}

	dir := filepath.Join(s.dataDir, "investigations")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}

	path := filepath.Join(dir, inv.ID+".json")
	data, err := json.MarshalIndent(inv, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(path, data, 0644)
}

// deleteFromDisk removes a persisted investigation file.
func (s *InvestigationStore) deleteFromDisk(id string) {
	if s.dataDir == "" {
		return
	}
	path := filepath.Join(s.dataDir, "investigations", id+".json")
	_ = os.Remove(path)
}

// loadFromDisk loads all persisted investigations from disk.
func (s *InvestigationStore) loadFromDisk() error {
	if s.dataDir == "" {
		return nil
	}

	dir := filepath.Join(s.dataDir, "investigations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var inv domain.Investigation
		if err := json.Unmarshal(data, &inv); err != nil {
			continue
		}
		s.items[inv.ID] = &inv
	}
	return nil
}
