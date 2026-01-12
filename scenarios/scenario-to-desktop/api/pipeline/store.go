package pipeline

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// InMemoryStore implements Store with in-memory storage.
type InMemoryStore struct {
	mu       sync.RWMutex
	statuses map[string]*Status
}

// NewInMemoryStore creates a new in-memory pipeline store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		statuses: make(map[string]*Status),
	}
}

// Save creates or updates a pipeline status.
func (s *InMemoryStore) Save(status *Status) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[status.PipelineID] = status
}

// Get retrieves a pipeline status by ID.
func (s *InMemoryStore) Get(pipelineID string) (*Status, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.statuses[pipelineID]
	return status, ok
}

// Update updates a pipeline status using a modifier function.
func (s *InMemoryStore) Update(pipelineID string, fn func(status *Status)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.statuses[pipelineID]
	if !ok {
		return false
	}
	fn(status)
	return true
}

// UpdateStage updates a specific stage's result within a pipeline.
func (s *InMemoryStore) UpdateStage(pipelineID, stageName string, result *StageResult) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.statuses[pipelineID]
	if !ok {
		return false
	}
	if status.Stages == nil {
		status.Stages = make(map[string]*StageResult)
	}
	status.Stages[stageName] = result
	return true
}

// Delete removes a pipeline status.
func (s *InMemoryStore) Delete(pipelineID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.statuses[pipelineID]; !ok {
		return false
	}
	delete(s.statuses, pipelineID)
	return true
}

// List returns all pipeline statuses.
func (s *InMemoryStore) List() []*Status {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Status, 0, len(s.statuses))
	for _, status := range s.statuses {
		result = append(result, status)
	}
	return result
}

// Cleanup removes completed pipelines older than the given duration.
func (s *InMemoryStore) Cleanup(olderThanUnix int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, status := range s.statuses {
		if status.IsComplete() && status.CompletedAt > 0 && status.CompletedAt < olderThanUnix {
			delete(s.statuses, id)
		}
	}
}

// InMemoryCancelManager implements CancelManager with in-memory storage.
type InMemoryCancelManager struct {
	mu      sync.Mutex
	cancels map[string]context.CancelFunc
}

// NewInMemoryCancelManager creates a new in-memory cancel manager.
func NewInMemoryCancelManager() *InMemoryCancelManager {
	return &InMemoryCancelManager{
		cancels: make(map[string]context.CancelFunc),
	}
}

// Set registers a cancellation function for a pipeline.
func (m *InMemoryCancelManager) Set(pipelineID string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancels[pipelineID] = cancel
}

// Take retrieves and removes the cancellation function.
func (m *InMemoryCancelManager) Take(pipelineID string) context.CancelFunc {
	m.mu.Lock()
	defer m.mu.Unlock()
	cancel, ok := m.cancels[pipelineID]
	if !ok {
		return nil
	}
	delete(m.cancels, pipelineID)
	return cancel
}

// Clear removes a cancellation function without calling it.
func (m *InMemoryCancelManager) Clear(pipelineID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancels, pipelineID)
}

// UUIDGenerator implements IDGenerator using UUIDs.
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUID generator.
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// Generate returns a new unique pipeline ID.
func (g *UUIDGenerator) Generate() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	// Format as UUID (8-4-4-4-12)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// RealTimeProvider implements TimeProvider using actual time.
type RealTimeProvider struct{}

// NewRealTimeProvider creates a new real time provider.
func NewRealTimeProvider() *RealTimeProvider {
	return &RealTimeProvider{}
}

// Now returns the current Unix timestamp in seconds.
func (p *RealTimeProvider) Now() int64 {
	return time.Now().Unix()
}
