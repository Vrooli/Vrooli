package recovery

import (
	"context"
	"sync"
	"time"
)

// Store is the interface for persisting session checkpoints.
type Store interface {
	// Save saves or updates a checkpoint.
	Save(ctx context.Context, checkpoint *SessionCheckpoint) error

	// Get retrieves a checkpoint by session ID.
	// Returns nil, nil if not found.
	Get(ctx context.Context, sessionID string) (*SessionCheckpoint, error)

	// Delete removes a checkpoint by session ID.
	Delete(ctx context.Context, sessionID string) error

	// ListActive returns all checkpoints (for recovery on startup).
	ListActive(ctx context.Context) ([]*SessionCheckpoint, error)

	// Cleanup removes checkpoints older than the given duration.
	// Returns the number of deleted checkpoints.
	Cleanup(ctx context.Context, olderThan time.Duration) (int, error)
}

// ActionSource is a function that retrieves recorded actions from the sidecar.
type ActionSource func(ctx context.Context, sessionID string) ([]RecordedAction, string, error)

// Manager manages checkpoint lifecycle for recording sessions.
type Manager interface {
	// Start initializes the checkpoint manager and starts cleanup.
	Start(ctx context.Context) error

	// Stop stops the checkpoint manager.
	Stop() error

	// StartCheckpointing begins periodic checkpointing for a session.
	StartCheckpointing(sessionID string, workflowID string, browserConfig BrowserConfig)

	// StopCheckpointing stops checkpointing and deletes the checkpoint.
	StopCheckpointing(sessionID string)

	// GetRecoveryData retrieves a checkpoint for potential recovery.
	// Returns nil if no checkpoint exists.
	GetRecoveryData(ctx context.Context, sessionID string) (*SessionCheckpoint, error)

	// ListRecoverableSessions returns sessions with available checkpoints.
	ListRecoverableSessions(ctx context.Context) ([]*SessionCheckpoint, error)

	// DeleteCheckpoint explicitly deletes a checkpoint (e.g., user chose "Start Fresh").
	DeleteCheckpoint(ctx context.Context, sessionID string) error
}

// MockStore is a Store implementation for testing.
type MockStore struct {
	Checkpoints map[string]*SessionCheckpoint
	SaveErr     error
	GetErr      error
	DeleteErr   error
	ListErr     error
	CleanupErr  error
	mu          sync.RWMutex
}

// NewMockStore creates a new MockStore.
func NewMockStore() *MockStore {
	return &MockStore{
		Checkpoints: make(map[string]*SessionCheckpoint),
	}
}

// Save implements Store.
func (m *MockStore) Save(ctx context.Context, checkpoint *SessionCheckpoint) error {
	if m.SaveErr != nil {
		return m.SaveErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Checkpoints[checkpoint.SessionID] = checkpoint
	return nil
}

// Get implements Store.
func (m *MockStore) Get(ctx context.Context, sessionID string) (*SessionCheckpoint, error) {
	if m.GetErr != nil {
		return nil, m.GetErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	cp, ok := m.Checkpoints[sessionID]
	if !ok {
		return nil, nil
	}
	return cp, nil
}

// Delete implements Store.
func (m *MockStore) Delete(ctx context.Context, sessionID string) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.Checkpoints, sessionID)
	return nil
}

// ListActive implements Store.
func (m *MockStore) ListActive(ctx context.Context) ([]*SessionCheckpoint, error) {
	if m.ListErr != nil {
		return nil, m.ListErr
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*SessionCheckpoint, 0, len(m.Checkpoints))
	for _, cp := range m.Checkpoints {
		result = append(result, cp)
	}
	return result, nil
}

// Cleanup implements Store.
func (m *MockStore) Cleanup(ctx context.Context, olderThan time.Duration) (int, error) {
	if m.CleanupErr != nil {
		return 0, m.CleanupErr
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-olderThan)
	count := 0
	for id, cp := range m.Checkpoints {
		if cp.UpdatedAt.Before(cutoff) {
			delete(m.Checkpoints, id)
			count++
		}
	}
	return count, nil
}

// compile-time check
var _ Store = (*MockStore)(nil)
