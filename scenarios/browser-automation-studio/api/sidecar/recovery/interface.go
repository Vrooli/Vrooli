package recovery

import (
	"context"
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
