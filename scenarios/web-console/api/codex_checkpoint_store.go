package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"web-console/internal/dbx"
)

type CodexRolloutCheckpoint struct {
	Path      string
	SessionID string
	Offset    int64
	UpdatedAt time.Time
}

type CodexCheckpointStore interface {
	Get(ctx context.Context, path string) (CodexRolloutCheckpoint, bool, error)
	Save(ctx context.Context, checkpoint CodexRolloutCheckpoint) error
	DeleteSession(ctx context.Context, sessionID string) error
}

type SQLCodexCheckpointStore struct {
	db dbx.Handle
}

func NewSQLCodexCheckpointStore(db dbx.Handle) *SQLCodexCheckpointStore {
	return &SQLCodexCheckpointStore{db: db}
}

func (s *SQLCodexCheckpointStore) Get(ctx context.Context, path string) (CodexRolloutCheckpoint, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT path, session_id, offset_bytes, updated_at
		FROM codex_rollout_checkpoints
		WHERE path = ?`,
		path,
	)

	var checkpoint CodexRolloutCheckpoint
	var updatedAt string
	if err := row.Scan(&checkpoint.Path, &checkpoint.SessionID, &checkpoint.Offset, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return CodexRolloutCheckpoint{}, false, nil
		}
		return CodexRolloutCheckpoint{}, false, fmt.Errorf("get checkpoint: %w", err)
	}
	if parsed, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		checkpoint.UpdatedAt = parsed
	} else if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		checkpoint.UpdatedAt = parsed
	}
	return checkpoint, true, nil
}

func (s *SQLCodexCheckpointStore) Save(ctx context.Context, checkpoint CodexRolloutCheckpoint) error {
	updatedAt := checkpoint.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO codex_rollout_checkpoints (path, session_id, offset_bytes, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(path) DO UPDATE SET
			session_id = excluded.session_id,
			offset_bytes = excluded.offset_bytes,
			updated_at = excluded.updated_at`,
		checkpoint.Path,
		checkpoint.SessionID,
		checkpoint.Offset,
		formatTime(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("save checkpoint: %w", err)
	}
	return nil
}

func (s *SQLCodexCheckpointStore) DeleteSession(ctx context.Context, sessionID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM codex_rollout_checkpoints WHERE session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete session checkpoints: %w", err)
	}
	return nil
}

type InMemoryCodexCheckpointStore struct {
	mu          sync.Mutex
	checkpoints map[string]CodexRolloutCheckpoint
}

func NewInMemoryCodexCheckpointStore() *InMemoryCodexCheckpointStore {
	return &InMemoryCodexCheckpointStore{
		checkpoints: make(map[string]CodexRolloutCheckpoint),
	}
}

func (s *InMemoryCodexCheckpointStore) Get(_ context.Context, path string) (CodexRolloutCheckpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	checkpoint, ok := s.checkpoints[path]
	return checkpoint, ok, nil
}

func (s *InMemoryCodexCheckpointStore) Save(_ context.Context, checkpoint CodexRolloutCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if checkpoint.UpdatedAt.IsZero() {
		checkpoint.UpdatedAt = time.Now().UTC()
	}
	s.checkpoints[checkpoint.Path] = checkpoint
	return nil
}

func (s *InMemoryCodexCheckpointStore) DeleteSession(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for path, checkpoint := range s.checkpoints {
		if checkpoint.SessionID == sessionID {
			delete(s.checkpoints, path)
		}
	}
	return nil
}
