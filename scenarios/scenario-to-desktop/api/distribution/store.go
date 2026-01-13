package distribution

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore implements Store using in-memory storage.
type InMemoryStore struct {
	mu       sync.RWMutex
	statuses map[string]*DistributionStatus
}

// NewInMemoryStore creates a new in-memory store.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		statuses: make(map[string]*DistributionStatus),
	}
}

// Save creates or updates a distribution status.
func (s *InMemoryStore) Save(status *DistributionStatus) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.statuses[status.DistributionID] = status
}

// Get retrieves a distribution status by ID.
func (s *InMemoryStore) Get(distributionID string) (*DistributionStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.statuses[distributionID]
	return status, ok
}

// Update updates a distribution status.
func (s *InMemoryStore) Update(distributionID string, fn func(status *DistributionStatus)) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	status, ok := s.statuses[distributionID]
	if !ok {
		return false
	}
	fn(status)
	return true
}

// List returns all distribution statuses.
func (s *InMemoryStore) List() []*DistributionStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	list := make([]*DistributionStatus, 0, len(s.statuses))
	for _, status := range s.statuses {
		list = append(list, status)
	}

	return list
}

// Delete removes a distribution status.
func (s *InMemoryStore) Delete(distributionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.statuses[distributionID]
	delete(s.statuses, distributionID)
	return ok
}

// InMemoryCancelManager implements CancelManager using in-memory storage.
type InMemoryCancelManager struct {
	mu      sync.RWMutex
	cancels map[string]context.CancelFunc
}

// NewCancelManager creates a new cancel manager.
func NewCancelManager() *InMemoryCancelManager {
	return &InMemoryCancelManager{
		cancels: make(map[string]context.CancelFunc),
	}
}

// Set stores a cancel function for a distribution ID.
func (m *InMemoryCancelManager) Set(distributionID string, cancel context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancels[distributionID] = cancel
}

// Get retrieves a cancel function for a distribution ID.
func (m *InMemoryCancelManager) Get(distributionID string) (context.CancelFunc, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cancel, ok := m.cancels[distributionID]
	return cancel, ok
}

// Delete removes a cancel function for a distribution ID.
func (m *InMemoryCancelManager) Delete(distributionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.cancels, distributionID)
}

// UUIDGenerator implements IDGenerator using UUID.
type UUIDGenerator struct{}

// Generate generates a new UUID.
func (g *UUIDGenerator) Generate() string {
	return uuid.New().String()
}
