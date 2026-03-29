package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// SessionMetadata holds persisted session state for restart recovery.
type SessionMetadata struct {
	ID       string
	Backend  BackendID
	Shell    string
	Cols     uint16
	Rows     uint16
	Policy   ExpirationPolicy
	Created  time.Time
	Detached bool // true if session uses a backend that survives restart
}

// SessionMetadataStore persists session metadata for restart recovery.
type SessionMetadataStore interface {
	Save(meta SessionMetadata) error
	Get(id string) (SessionMetadata, error)
	List() ([]SessionMetadata, error)
	Delete(id string) error
	UpdatePolicy(id string, policy ExpirationPolicy) error
	ListDetached() ([]SessionMetadata, error)
}

// SQLSessionStore implements SessionMetadataStore using SQLite.
type SQLSessionStore struct {
	db *sql.DB
}

// NewSQLSessionStore creates a new SQLite-backed session metadata store.
func NewSQLSessionStore(db *sql.DB) *SQLSessionStore {
	return &SQLSessionStore{db: db}
}

func (s *SQLSessionStore) Save(meta SessionMetadata) error {
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sessions (id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		meta.ID,
		string(meta.Backend),
		meta.Shell,
		meta.Cols,
		meta.Rows,
		string(meta.Policy.Mode),
		meta.Policy.Duration,
		meta.Created.UTC().Format(time.RFC3339),
		boolToInt(meta.Detached),
	)
	return err
}

func (s *SQLSessionStore) Get(id string) (SessionMetadata, error) {
	row := s.db.QueryRow(`
		SELECT id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached
		FROM sessions WHERE id = ?`, id)
	return scanSessionMetadata(row)
}

func (s *SQLSessionStore) List() ([]SessionMetadata, error) {
	rows, err := s.db.Query(`
		SELECT id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached
		FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionMetadataRows(rows)
}

func (s *SQLSessionStore) ListDetached() ([]SessionMetadata, error) {
	rows, err := s.db.Query(`
		SELECT id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached
		FROM sessions WHERE detached = 1 ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionMetadataRows(rows)
}

func (s *SQLSessionStore) Delete(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *SQLSessionStore) UpdatePolicy(id string, policy ExpirationPolicy) error {
	_, err := s.db.Exec(`UPDATE sessions SET policy_mode = ?, policy_duration = ? WHERE id = ?`,
		string(policy.Mode), policy.Duration, id)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSessionMetadata(row scannable) (SessionMetadata, error) {
	var meta SessionMetadata
	var backend, policyMode, policyDuration, createdStr string
	var detached int
	var durPtr *string
	err := row.Scan(&meta.ID, &backend, &meta.Shell, &meta.Cols, &meta.Rows,
		&policyMode, &durPtr, &createdStr, &detached)
	if err != nil {
		return meta, fmt.Errorf("scan session metadata: %w", err)
	}
	meta.Backend = BackendID(backend)
	meta.Policy.Mode = PolicyMode(policyMode)
	if durPtr != nil {
		policyDuration = *durPtr
	}
	meta.Policy.Duration = policyDuration
	meta.Detached = detached == 1
	meta.Created, _ = time.Parse(time.RFC3339, createdStr)
	return meta, nil
}

func scanSessionMetadataRows(rows *sql.Rows) ([]SessionMetadata, error) {
	var result []SessionMetadata
	for rows.Next() {
		meta, err := scanSessionMetadata(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, meta)
	}
	return result, rows.Err()
}

// InMemorySessionStore implements SessionMetadataStore for testing.
type InMemorySessionStore struct {
	mu       sync.Mutex
	sessions map[string]SessionMetadata
}

// NewInMemorySessionStore creates an in-memory session metadata store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		sessions: make(map[string]SessionMetadata),
	}
}

func (s *InMemorySessionStore) Save(meta SessionMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[meta.ID] = meta
	return nil
}

func (s *InMemorySessionStore) Get(id string) (SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return SessionMetadata{}, fmt.Errorf("session %s not found", id)
	}
	return meta, nil
}

func (s *InMemorySessionStore) List() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]SessionMetadata, 0, len(s.sessions))
	for _, meta := range s.sessions {
		result = append(result, meta)
	}
	return result, nil
}

func (s *InMemorySessionStore) ListDetached() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []SessionMetadata
	for _, meta := range s.sessions {
		if meta.Detached {
			result = append(result, meta)
		}
	}
	return result, nil
}

func (s *InMemorySessionStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *InMemorySessionStore) UpdatePolicy(id string, policy ExpirationPolicy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Policy = policy
	s.sessions[id] = meta
	return nil
}
