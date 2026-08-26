package main

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	"web-console/internal/dbx"
)

// AgentTranscriptCheckpoint is a per-source ingestion cursor for all agent
// transcript adapters. Cursor is an opaque, source-defined string: a byte
// offset for append-only JSONL, or a JSON high-water mark for reconciliation.
type AgentTranscriptCheckpoint struct {
	Source    string
	SourceKey string
	SessionID string // owning web-console session id
	Cursor    string
	UpdatedAt time.Time
}

// AgentTranscriptCheckpointStore persists ingestion cursors keyed by
// (source, source_key). Both a SQLite-backed (production) and in-memory (test)
// implementation are provided.
type AgentTranscriptCheckpointStore interface {
	Get(ctx context.Context, source, sourceKey string) (AgentTranscriptCheckpoint, bool, error)
	Save(ctx context.Context, checkpoint AgentTranscriptCheckpoint) error
	DeleteSession(ctx context.Context, sessionID string) error
}

type SQLAgentTranscriptCheckpointStore struct {
	db dbx.Handle
}

func NewSQLAgentTranscriptCheckpointStore(db dbx.Handle) *SQLAgentTranscriptCheckpointStore {
	return &SQLAgentTranscriptCheckpointStore{db: db}
}

func (s *SQLAgentTranscriptCheckpointStore) Get(ctx context.Context, source, sourceKey string) (AgentTranscriptCheckpoint, bool, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT source, source_key, web_console_session_id, cursor, updated_at
		FROM agent_transcript_checkpoints
		WHERE source = ? AND source_key = ?`,
		source, sourceKey,
	)
	var cp AgentTranscriptCheckpoint
	var updatedAt string
	if err := row.Scan(&cp.Source, &cp.SourceKey, &cp.SessionID, &cp.Cursor, &updatedAt); err != nil {
		if err == sql.ErrNoRows {
			return AgentTranscriptCheckpoint{}, false, nil
		}
		return AgentTranscriptCheckpoint{}, false, fmt.Errorf("get transcript checkpoint: %w", err)
	}
	if parsed, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		cp.UpdatedAt = parsed
	} else if parsed, err := time.Parse(time.RFC3339, updatedAt); err == nil {
		cp.UpdatedAt = parsed
	}
	return cp, true, nil
}

func (s *SQLAgentTranscriptCheckpointStore) Save(ctx context.Context, checkpoint AgentTranscriptCheckpoint) error {
	updatedAt := checkpoint.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO agent_transcript_checkpoints (source, source_key, web_console_session_id, cursor, updated_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(source, source_key) DO UPDATE SET
			web_console_session_id = excluded.web_console_session_id,
			cursor = excluded.cursor,
			updated_at = excluded.updated_at`,
		checkpoint.Source,
		checkpoint.SourceKey,
		checkpoint.SessionID,
		checkpoint.Cursor,
		formatTime(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("save transcript checkpoint: %w", err)
	}
	return nil
}

func (s *SQLAgentTranscriptCheckpointStore) DeleteSession(ctx context.Context, sessionID string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM agent_transcript_checkpoints WHERE web_console_session_id = ?`, sessionID); err != nil {
		return fmt.Errorf("delete session transcript checkpoints: %w", err)
	}
	return nil
}

type InMemoryAgentTranscriptCheckpointStore struct {
	mu          sync.Mutex
	checkpoints map[string]AgentTranscriptCheckpoint
}

func NewInMemoryAgentTranscriptCheckpointStore() *InMemoryAgentTranscriptCheckpointStore {
	return &InMemoryAgentTranscriptCheckpointStore{
		checkpoints: make(map[string]AgentTranscriptCheckpoint),
	}
}

func transcriptCheckpointKey(source, sourceKey string) string {
	return source + "\x00" + sourceKey
}

func (s *InMemoryAgentTranscriptCheckpointStore) Get(_ context.Context, source, sourceKey string) (AgentTranscriptCheckpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.checkpoints[transcriptCheckpointKey(source, sourceKey)]
	return cp, ok, nil
}

func (s *InMemoryAgentTranscriptCheckpointStore) Save(_ context.Context, checkpoint AgentTranscriptCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if checkpoint.UpdatedAt.IsZero() {
		checkpoint.UpdatedAt = time.Now().UTC()
	}
	s.checkpoints[transcriptCheckpointKey(checkpoint.Source, checkpoint.SourceKey)] = checkpoint
	return nil
}

func (s *InMemoryAgentTranscriptCheckpointStore) DeleteSession(_ context.Context, sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, cp := range s.checkpoints {
		if cp.SessionID == sessionID {
			delete(s.checkpoints, key)
		}
	}
	return nil
}
