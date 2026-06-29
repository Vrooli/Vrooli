package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// AgentTranscriptCheckpoint is a per-source ingestion cursor for the newer
// agent transcript adapters (Grok tailer, OpenCode reconciler). Cursor is an
// opaque, source-defined string: a byte offset for Grok's append-only JSONL, a
// JSON high-water mark for OpenCode's full-history reconciliation. Codex keeps
// its own dedicated byte-offset table (codex_rollout_checkpoints).
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
	Get(source, sourceKey string) (AgentTranscriptCheckpoint, bool, error)
	Save(checkpoint AgentTranscriptCheckpoint) error
	DeleteSession(sessionID string) error
}

type SQLAgentTranscriptCheckpointStore struct {
	db *sql.DB
}

func NewSQLAgentTranscriptCheckpointStore(db *sql.DB) *SQLAgentTranscriptCheckpointStore {
	return &SQLAgentTranscriptCheckpointStore{db: db}
}

func (s *SQLAgentTranscriptCheckpointStore) Get(source, sourceKey string) (AgentTranscriptCheckpoint, bool, error) {
	row := s.db.QueryRow(`
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

func (s *SQLAgentTranscriptCheckpointStore) Save(checkpoint AgentTranscriptCheckpoint) error {
	updatedAt := checkpoint.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}
	_, err := s.db.Exec(`
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

func (s *SQLAgentTranscriptCheckpointStore) DeleteSession(sessionID string) error {
	if _, err := s.db.Exec(`DELETE FROM agent_transcript_checkpoints WHERE web_console_session_id = ?`, sessionID); err != nil {
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

func (s *InMemoryAgentTranscriptCheckpointStore) Get(source, sourceKey string) (AgentTranscriptCheckpoint, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp, ok := s.checkpoints[transcriptCheckpointKey(source, sourceKey)]
	return cp, ok, nil
}

func (s *InMemoryAgentTranscriptCheckpointStore) Save(checkpoint AgentTranscriptCheckpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if checkpoint.UpdatedAt.IsZero() {
		checkpoint.UpdatedAt = time.Now().UTC()
	}
	s.checkpoints[transcriptCheckpointKey(checkpoint.Source, checkpoint.SourceKey)] = checkpoint
	return nil
}

func (s *InMemoryAgentTranscriptCheckpointStore) DeleteSession(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, cp := range s.checkpoints {
		if cp.SessionID == sessionID {
			delete(s.checkpoints, key)
		}
	}
	return nil
}
