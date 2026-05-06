package main

import (
	"database/sql"
	"fmt"
	"sync"
	"time"
)

// SessionStatus is the lifecycle status of a session row. Only Recover() and
// the recovery endpoints transition status; everywhere else the row is
// implicitly 'live'.
type SessionStatus string

const (
	SessionStatusLive             SessionStatus = "live"
	SessionStatusAwaitingRecovery SessionStatus = "awaiting_recovery"
	SessionStatusDismissed        SessionStatus = "dismissed"
)

// AgentType is the closed set of agent kinds the recovery flow knows how to
// reattach. New runtimes require an explicit code change; the launch_command
// string is never parsed at recovery time to derive type.
type AgentType string

const (
	AgentTypeNone   AgentType = "none"
	AgentTypeCodex  AgentType = "codex"
	AgentTypeClaude AgentType = "claude"
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

	// Lifecycle/status (Phase 1 of persistent-session-recovery-hardening-plan).
	Status SessionStatus

	// Agent identity, populated automatically as the agent runs. None of these
	// are required for normal operation; they exist so a recovery flow can
	// reattach the agent without grovelling rollout files.
	AgentType       AgentType
	LaunchCommand   string
	AgentSessionID  string
	CWD             string
	LastRolloutPath string

	// Activity timestamps. Zero time = unknown. Stored as RFC3339 strings in
	// the DB; empty string is the zero-value sentinel.
	LastActivityAt time.Time
	OrphanedAt     time.Time

	// RecoveredInto is set on the orphan row when the recovery endpoint
	// successfully spawns a fresh pane from it. Used for audit and idempotency.
	RecoveredInto string
}

// AgentInfo is the partial-update payload used when a populator (codex tailer,
// claude hook, launch-command save) learns more about a session's agent
// identity. Empty fields mean "leave unchanged".
type AgentInfo struct {
	AgentType       AgentType
	LaunchCommand   string
	AgentSessionID  string
	CWD             string
	LastRolloutPath string
	LastActivityAt  time.Time
}

// SessionMetadataStore persists session metadata for restart recovery.
type SessionMetadataStore interface {
	Save(meta SessionMetadata) error
	Get(id string) (SessionMetadata, error)
	List() ([]SessionMetadata, error)
	Delete(id string) error
	UpdatePolicy(id string, policy ExpirationPolicy) error
	ListDetached() ([]SessionMetadata, error)

	// Phase 1 additions:
	UpdateAgentInfo(id string, info AgentInfo) error
	MarkOrphaned(id string, at time.Time) error
	MarkLive(id string) error
	MarkDismissed(id string, recoveredInto string) error
	ListRecoverable() ([]SessionMetadata, error)
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
	if meta.Status == "" {
		meta.Status = SessionStatusLive
	}
	if meta.AgentType == "" {
		meta.AgentType = AgentTypeNone
	}
	_, err := s.db.Exec(`
		INSERT OR REPLACE INTO sessions (
			id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached,
			status, agent_type, launch_command, agent_session_id, cwd, last_rollout_path,
			last_activity_at, orphaned_at, recovered_into
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		meta.ID,
		string(meta.Backend),
		meta.Shell,
		meta.Cols,
		meta.Rows,
		string(meta.Policy.Mode),
		meta.Policy.Duration,
		meta.Created.UTC().Format(time.RFC3339),
		boolToInt(meta.Detached),
		string(meta.Status),
		string(meta.AgentType),
		meta.LaunchCommand,
		meta.AgentSessionID,
		meta.CWD,
		meta.LastRolloutPath,
		formatTimeOrEmpty(meta.LastActivityAt),
		formatTimeOrEmpty(meta.OrphanedAt),
		meta.RecoveredInto,
	)
	return err
}

const sessionSelectColumns = `
	id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached,
	status, agent_type, launch_command, agent_session_id, cwd, last_rollout_path,
	last_activity_at, orphaned_at, recovered_into`

func (s *SQLSessionStore) Get(id string) (SessionMetadata, error) {
	row := s.db.QueryRow(`SELECT `+sessionSelectColumns+` FROM sessions WHERE id = ?`, id)
	return scanSessionMetadata(row)
}

func (s *SQLSessionStore) List() ([]SessionMetadata, error) {
	rows, err := s.db.Query(`SELECT ` + sessionSelectColumns + ` FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionMetadataRows(rows)
}

func (s *SQLSessionStore) ListDetached() ([]SessionMetadata, error) {
	// Only sessions whose row is currently 'live' are eligible for tmux
	// re-attach during Recover(). Awaiting-recovery rows are surfaced via
	// ListRecoverable() and require explicit operator action.
	rows, err := s.db.Query(`SELECT ` + sessionSelectColumns + `
		FROM sessions
		WHERE detached = 1 AND status = 'live'
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessionMetadataRows(rows)
}

func (s *SQLSessionStore) ListRecoverable() ([]SessionMetadata, error) {
	rows, err := s.db.Query(`SELECT ` + sessionSelectColumns + `
		FROM sessions
		WHERE detached = 1 AND status = 'awaiting_recovery'
		ORDER BY COALESCE(NULLIF(last_activity_at,''), created_at) DESC`)
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

func (s *SQLSessionStore) UpdateAgentInfo(id string, info AgentInfo) error {
	// Build a partial UPDATE so callers can target individual fields; empty
	// strings on AgentInfo mean "leave alone" for AgentType, AgentSessionID,
	// CWD, LastRolloutPath, LaunchCommand. Zero time on LastActivityAt means
	// the same. AgentType="" is treated as no-update; pass AgentTypeNone to
	// explicitly clear it.
	sets := []string{}
	args := []interface{}{}
	if info.AgentType != "" {
		sets = append(sets, "agent_type = ?")
		args = append(args, string(info.AgentType))
	}
	if info.LaunchCommand != "" {
		sets = append(sets, "launch_command = ?")
		args = append(args, info.LaunchCommand)
	}
	if info.AgentSessionID != "" {
		sets = append(sets, "agent_session_id = ?")
		args = append(args, info.AgentSessionID)
	}
	if info.CWD != "" {
		sets = append(sets, "cwd = ?")
		args = append(args, info.CWD)
	}
	if info.LastRolloutPath != "" {
		sets = append(sets, "last_rollout_path = ?")
		args = append(args, info.LastRolloutPath)
	}
	if !info.LastActivityAt.IsZero() {
		sets = append(sets, "last_activity_at = ?")
		args = append(args, info.LastActivityAt.UTC().Format(time.RFC3339))
	}
	if len(sets) == 0 {
		return nil
	}
	q := `UPDATE sessions SET ` + joinComma(sets) + ` WHERE id = ?`
	args = append(args, id)
	_, err := s.db.Exec(q, args...)
	return err
}

func (s *SQLSessionStore) MarkOrphaned(id string, at time.Time) error {
	_, err := s.db.Exec(`
		UPDATE sessions
		SET status = ?, orphaned_at = ?
		WHERE id = ? AND detached = 1`,
		string(SessionStatusAwaitingRecovery),
		at.UTC().Format(time.RFC3339),
		id,
	)
	return err
}

func (s *SQLSessionStore) MarkLive(id string) error {
	_, err := s.db.Exec(`
		UPDATE sessions
		SET status = ?, orphaned_at = ''
		WHERE id = ?`,
		string(SessionStatusLive),
		id,
	)
	return err
}

func (s *SQLSessionStore) MarkDismissed(id string, recoveredInto string) error {
	_, err := s.db.Exec(`
		UPDATE sessions
		SET status = ?, recovered_into = ?
		WHERE id = ?`,
		string(SessionStatusDismissed),
		recoveredInto,
		id,
	)
	return err
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func formatTimeOrEmpty(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTimeOrZero(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}

type scannable interface {
	Scan(dest ...any) error
}

func scanSessionMetadata(row scannable) (SessionMetadata, error) {
	var meta SessionMetadata
	var (
		backend, policyMode, createdStr string
		status, agentType               string
		lastActivity, orphanedAt        string
		policyDurationPtr               *string
		detached                        int
	)
	err := row.Scan(
		&meta.ID, &backend, &meta.Shell, &meta.Cols, &meta.Rows,
		&policyMode, &policyDurationPtr, &createdStr, &detached,
		&status, &agentType, &meta.LaunchCommand, &meta.AgentSessionID,
		&meta.CWD, &meta.LastRolloutPath,
		&lastActivity, &orphanedAt, &meta.RecoveredInto,
	)
	if err != nil {
		return meta, fmt.Errorf("scan session metadata: %w", err)
	}
	meta.Backend = BackendID(backend)
	meta.Policy.Mode = PolicyMode(policyMode)
	if policyDurationPtr != nil {
		meta.Policy.Duration = *policyDurationPtr
	}
	meta.Detached = detached == 1
	meta.Created, _ = time.Parse(time.RFC3339, createdStr)
	if status == "" {
		meta.Status = SessionStatusLive
	} else {
		meta.Status = SessionStatus(status)
	}
	if agentType == "" {
		meta.AgentType = AgentTypeNone
	} else {
		meta.AgentType = AgentType(agentType)
	}
	meta.LastActivityAt = parseTimeOrZero(lastActivity)
	meta.OrphanedAt = parseTimeOrZero(orphanedAt)
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
	if meta.Status == "" {
		meta.Status = SessionStatusLive
	}
	if meta.AgentType == "" {
		meta.AgentType = AgentTypeNone
	}
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
		if meta.Detached && meta.Status == SessionStatusLive {
			result = append(result, meta)
		}
	}
	return result, nil
}

func (s *InMemorySessionStore) ListRecoverable() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []SessionMetadata
	for _, meta := range s.sessions {
		if meta.Detached && meta.Status == SessionStatusAwaitingRecovery {
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

func (s *InMemorySessionStore) UpdateAgentInfo(id string, info AgentInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	if info.AgentType != "" {
		meta.AgentType = info.AgentType
	}
	if info.LaunchCommand != "" {
		meta.LaunchCommand = info.LaunchCommand
	}
	if info.AgentSessionID != "" {
		meta.AgentSessionID = info.AgentSessionID
	}
	if info.CWD != "" {
		meta.CWD = info.CWD
	}
	if info.LastRolloutPath != "" {
		meta.LastRolloutPath = info.LastRolloutPath
	}
	if !info.LastActivityAt.IsZero() {
		meta.LastActivityAt = info.LastActivityAt
	}
	s.sessions[id] = meta
	return nil
}

func (s *InMemorySessionStore) MarkOrphaned(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	if !meta.Detached {
		return nil
	}
	meta.Status = SessionStatusAwaitingRecovery
	meta.OrphanedAt = at
	s.sessions[id] = meta
	return nil
}

func (s *InMemorySessionStore) MarkLive(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Status = SessionStatusLive
	meta.OrphanedAt = time.Time{}
	s.sessions[id] = meta
	return nil
}

func (s *InMemorySessionStore) MarkDismissed(id string, recoveredInto string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Status = SessionStatusDismissed
	meta.RecoveredInto = recoveredInto
	s.sessions[id] = meta
	return nil
}
