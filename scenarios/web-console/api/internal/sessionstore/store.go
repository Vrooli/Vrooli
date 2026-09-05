// Package sessionstore persists web-console session metadata for restart
// recovery. The Store interface is implemented by a SQL-backed store
// (production) and an in-memory store (tests). Persistent-backend sessions
// (tmux) are reattached on boot by reading the store; standard in-memory
// sessions are not persisted.
package sessionstore

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"sync"
	"time"

	"web-console/internal/backend"
	"web-console/internal/dbx"
	"web-console/internal/policy"
)

// Status is the lifecycle status of a session row. Only Recover() and the
// recovery endpoints transition status; everywhere else the row is
// implicitly StatusLive.
type Status string

const (
	StatusLive             Status = "live"
	StatusAwaitingRecovery Status = "awaiting_recovery"
	StatusDismissed        Status = "dismissed"
)

// Agent is the closed set of agent kinds the recovery flow knows how to
// reattach. New runtimes require an explicit code change; the launch_command
// string is never parsed at recovery time to derive type.
type Agent string

const (
	AgentNone     Agent = "none"
	AgentCodex    Agent = "codex"
	AgentClaude   Agent = "claude"
	AgentOpenCode Agent = "opencode"
	AgentGrok     Agent = "grok"
)

// Origin records who opened a session. It is provenance the UI uses to
// separate human-opened tabs from agent- or remote-launched sessions.
type Origin string

const (
	OriginUI           Origin = "ui"
	OriginProgrammatic Origin = "programmatic"
	OriginRemote       Origin = "remote"
)

// Metadata holds persisted session state for restart recovery.
type Metadata struct {
	ID       string
	Backend  backend.ID
	Shell    string
	Cols     uint16
	Rows     uint16
	Policy   policy.Policy
	Created  time.Time
	Detached bool

	Status Status

	AgentType       Agent
	LaunchCommand   string
	AgentSessionID  string
	CWD             string
	LastRolloutPath string

	LastActivityAt time.Time
	OrphanedAt     time.Time

	RecoveredInto string
	ArchivedAt    time.Time

	Origin       Origin
	Owner        string
	DisplayLabel string
}

// AgentInfo is the partial-update payload used when a populator (codex tailer,
// claude hook, launch-command save) learns more about a session's agent
// identity. Empty fields mean "leave unchanged".
type AgentInfo struct {
	AgentType       Agent
	LaunchCommand   string
	AgentSessionID  string
	CWD             string
	LastRolloutPath string
	LastActivityAt  time.Time
}

// Store persists session metadata for restart recovery.
type Store interface {
	Save(ctx context.Context, meta Metadata) error
	Get(ctx context.Context, id string) (Metadata, error)
	List(ctx context.Context) ([]Metadata, error)
	Delete(ctx context.Context, id string) error
	UpdatePolicy(ctx context.Context, id string, pol policy.Policy) error
	ListDetached(ctx context.Context) ([]Metadata, error)

	UpdateAgentInfo(ctx context.Context, id string, info AgentInfo) error
	MarkOrphaned(ctx context.Context, id string, at time.Time) error
	MarkLive(ctx context.Context, id string) error
	MarkDismissed(ctx context.Context, id string, recoveredInto string) error
	MarkArchived(ctx context.Context, id string, at time.Time) error
	MarkUnarchived(ctx context.Context, id string) error
	ListArchived(ctx context.Context) ([]Metadata, error)
	// ListRetentionCandidates is deliberately narrower than ListArchived:
	// legacy dismissed and crash-recovery rows remain visible in the archive,
	// but retention may act only on rows with an explicit archive timestamp.
	ListRetentionCandidates(ctx context.Context) ([]Metadata, error)
	ListRecoverable(ctx context.Context) ([]Metadata, error)

	// SetProvenance records who opened a session. Unlike UpdateAgentInfo it
	// writes all three fields unconditionally: owner and displayLabel are
	// legitimately empty for anonymous UI sessions, so an empty value is the
	// intended value, not "leave unchanged".
	SetProvenance(ctx context.Context, id string, origin Origin, owner, displayLabel string) error
}

// SQLStore implements Store using SQLite.
type SQLStore struct {
	db dbx.Handle
}

// NewSQL creates a new SQLite-backed session metadata store.
func NewSQL(db dbx.Handle) *SQLStore {
	return &SQLStore{db: db}
}

func (s *SQLStore) Save(ctx context.Context, meta Metadata) error {
	if meta.Status == "" {
		meta.Status = StatusLive
	}
	if meta.AgentType == "" {
		meta.AgentType = AgentNone
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR REPLACE INTO sessions (
			id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached,
			status, agent_type, launch_command, agent_session_id, cwd, last_rollout_path,
			last_activity_at, orphaned_at, recovered_into, archived_at, origin, owner, display_label
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
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
		formatTimeOrEmpty(meta.ArchivedAt),
		string(meta.Origin),
		meta.Owner,
		meta.DisplayLabel,
	)
	return err
}

const selectColumns = `
	id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached,
	status, agent_type, launch_command, agent_session_id, cwd, last_rollout_path,
	last_activity_at, orphaned_at, recovered_into, archived_at, origin, owner, display_label`

func (s *SQLStore) Get(ctx context.Context, id string) (Metadata, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+selectColumns+` FROM sessions WHERE id = ?`, id)
	return scanMetadata(row)
}

func (s *SQLStore) List(ctx context.Context) ([]Metadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectColumns+` FROM sessions ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMetadataRows(rows)
}

func (s *SQLStore) ListDetached(ctx context.Context) ([]Metadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectColumns+`
		FROM sessions
		WHERE detached = 1 AND status = 'live'
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMetadataRows(rows)
}

func (s *SQLStore) ListRecoverable(ctx context.Context) ([]Metadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectColumns+`
		FROM sessions
		WHERE detached = 1 AND status = 'awaiting_recovery'
		ORDER BY COALESCE(NULLIF(last_activity_at,''), created_at) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMetadataRows(rows)
}

func (s *SQLStore) ListArchived(ctx context.Context) ([]Metadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectColumns+`
		FROM sessions
		WHERE archived_at <> '' OR status IN ('dismissed', 'awaiting_recovery')
		ORDER BY COALESCE(NULLIF(archived_at, ''), NULLIF(orphaned_at, ''), NULLIF(last_activity_at, ''), created_at) DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMetadataRows(rows)
}

func (s *SQLStore) ListRetentionCandidates(ctx context.Context) ([]Metadata, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectColumns+`
		FROM sessions
		WHERE archived_at <> ''
		ORDER BY archived_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMetadataRows(rows)
}

func (s *SQLStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *SQLStore) UpdatePolicy(ctx context.Context, id string, pol policy.Policy) error {
	_, err := s.db.ExecContext(ctx, `UPDATE sessions SET policy_mode = ?, policy_duration = ? WHERE id = ?`,
		string(pol.Mode), pol.Duration, id)
	return err
}

func (s *SQLStore) UpdateAgentInfo(ctx context.Context, id string, info AgentInfo) error {
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
	_, err := s.db.ExecContext(ctx, q, args...)
	return err
}

func (s *SQLStore) SetProvenance(ctx context.Context, id string, origin Origin, owner, displayLabel string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE sessions SET origin = ?, owner = ?, display_label = ? WHERE id = ?`,
		string(origin), owner, displayLabel, id,
	)
	return err
}

func (s *SQLStore) MarkOrphaned(ctx context.Context, id string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, orphaned_at = ?
		WHERE id = ? AND detached = 1`,
		string(StatusAwaitingRecovery),
		at.UTC().Format(time.RFC3339),
		id,
	)
	return err
}

func (s *SQLStore) MarkLive(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, orphaned_at = ''
		WHERE id = ?`,
		string(StatusLive),
		id,
	)
	return err
}

func (s *SQLStore) MarkDismissed(ctx context.Context, id string, recoveredInto string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET status = ?, recovered_into = ?, archived_at = ''
		WHERE id = ?`,
		string(StatusDismissed),
		recoveredInto,
		id,
	)
	return err
}

func (s *SQLStore) MarkArchived(ctx context.Context, id string, at time.Time) error {
	result, err := s.db.ExecContext(ctx, `UPDATE sessions SET archived_at = ? WHERE id = ?`, formatTimeOrEmpty(at), id)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return fmt.Errorf("session %s not found", id)
	}
	return nil
}

func (s *SQLStore) MarkUnarchived(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `UPDATE sessions SET archived_at = '' WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if affected, err := result.RowsAffected(); err == nil && affected == 0 {
		return fmt.Errorf("session %s not found", id)
	}
	return nil
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

func scanMetadata(row scannable) (Metadata, error) {
	var meta Metadata
	var (
		backendID, policyMode, createdStr    string
		status, agentType                    string
		lastActivity, orphanedAt, archivedAt string
		origin                               string
		policyDurationPtr                    *string
		detached                             int
	)
	err := row.Scan(
		&meta.ID, &backendID, &meta.Shell, &meta.Cols, &meta.Rows,
		&policyMode, &policyDurationPtr, &createdStr, &detached,
		&status, &agentType, &meta.LaunchCommand, &meta.AgentSessionID,
		&meta.CWD, &meta.LastRolloutPath,
		&lastActivity, &orphanedAt, &meta.RecoveredInto, &archivedAt,
		&origin, &meta.Owner, &meta.DisplayLabel,
	)
	if err != nil {
		return meta, fmt.Errorf("scan session metadata: %w", err)
	}
	meta.Origin = Origin(origin)
	meta.Backend = backend.ID(backendID)
	meta.Policy.Mode = policy.Mode(policyMode)
	if policyDurationPtr != nil {
		meta.Policy.Duration = *policyDurationPtr
	}
	meta.Detached = detached == 1
	meta.Created, _ = time.Parse(time.RFC3339, createdStr)
	if status == "" {
		meta.Status = StatusLive
	} else {
		meta.Status = Status(status)
	}
	if agentType == "" {
		meta.AgentType = AgentNone
	} else {
		meta.AgentType = Agent(agentType)
	}
	meta.LastActivityAt = parseTimeOrZero(lastActivity)
	meta.OrphanedAt = parseTimeOrZero(orphanedAt)
	meta.ArchivedAt = parseTimeOrZero(archivedAt)
	return meta, nil
}

func scanMetadataRows(rows *sql.Rows) ([]Metadata, error) {
	var result []Metadata
	for rows.Next() {
		meta, err := scanMetadata(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, meta)
	}
	return result, rows.Err()
}

// InMemoryStore implements Store for testing.
type InMemoryStore struct {
	mu       sync.Mutex
	sessions map[string]Metadata
}

// NewInMemory creates an in-memory session metadata store.
func NewInMemory() *InMemoryStore {
	return &InMemoryStore{
		sessions: make(map[string]Metadata),
	}
}

func (s *InMemoryStore) Save(_ context.Context, meta Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if meta.Status == "" {
		meta.Status = StatusLive
	}
	if meta.AgentType == "" {
		meta.AgentType = AgentNone
	}
	s.sessions[meta.ID] = meta
	return nil
}

func (s *InMemoryStore) Get(_ context.Context, id string) (Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return Metadata{}, fmt.Errorf("session %s not found", id)
	}
	return meta, nil
}

func (s *InMemoryStore) List(_ context.Context) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := make([]Metadata, 0, len(s.sessions))
	for _, meta := range s.sessions {
		result = append(result, meta)
	}
	return result, nil
}

func (s *InMemoryStore) ListDetached(_ context.Context) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Metadata
	for _, meta := range s.sessions {
		if meta.Detached && meta.Status == StatusLive {
			result = append(result, meta)
		}
	}
	return result, nil
}

func (s *InMemoryStore) ListRecoverable(_ context.Context) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Metadata
	for _, meta := range s.sessions {
		if meta.Detached && meta.Status == StatusAwaitingRecovery {
			result = append(result, meta)
		}
	}
	return result, nil
}

func (s *InMemoryStore) ListArchived(_ context.Context) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Metadata
	for _, meta := range s.sessions {
		if !meta.ArchivedAt.IsZero() || meta.Status == StatusDismissed || meta.Status == StatusAwaitingRecovery {
			result = append(result, meta)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ArchivedAt.After(result[j].ArchivedAt) })
	return result, nil
}

func (s *InMemoryStore) ListRetentionCandidates(_ context.Context) ([]Metadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var result []Metadata
	for _, meta := range s.sessions {
		if !meta.ArchivedAt.IsZero() {
			result = append(result, meta)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ArchivedAt.Before(result[j].ArchivedAt) })
	return result, nil
}

func (s *InMemoryStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

func (s *InMemoryStore) UpdatePolicy(_ context.Context, id string, pol policy.Policy) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Policy = pol
	s.sessions[id] = meta
	return nil
}

func (s *InMemoryStore) UpdateAgentInfo(_ context.Context, id string, info AgentInfo) error {
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

func (s *InMemoryStore) SetProvenance(_ context.Context, id string, origin Origin, owner, displayLabel string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Origin = origin
	meta.Owner = owner
	meta.DisplayLabel = displayLabel
	s.sessions[id] = meta
	return nil
}

func (s *InMemoryStore) MarkOrphaned(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	if !meta.Detached {
		return nil
	}
	meta.Status = StatusAwaitingRecovery
	meta.OrphanedAt = at
	s.sessions[id] = meta
	return nil
}

func (s *InMemoryStore) MarkLive(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Status = StatusLive
	meta.OrphanedAt = time.Time{}
	s.sessions[id] = meta
	return nil
}

func (s *InMemoryStore) MarkDismissed(_ context.Context, id string, recoveredInto string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.Status = StatusDismissed
	meta.RecoveredInto = recoveredInto
	meta.ArchivedAt = time.Time{}
	s.sessions[id] = meta
	return nil
}

func (s *InMemoryStore) MarkArchived(_ context.Context, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.ArchivedAt = at
	s.sessions[id] = meta
	return nil
}

func (s *InMemoryStore) MarkUnarchived(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	meta, ok := s.sessions[id]
	if !ok {
		return fmt.Errorf("session %s not found", id)
	}
	meta.ArchivedAt = time.Time{}
	s.sessions[id] = meta
	return nil
}
