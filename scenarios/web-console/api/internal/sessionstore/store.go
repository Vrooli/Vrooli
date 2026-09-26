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

// Status is the lifecycle status of a session row. Runtime exit records
// awaiting recovery; archive and recovery endpoints own deliberate policy
// transitions.
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

// LaunchMode is persisted because process names and rollout files do not
// prove who owns a live provider session.
type LaunchMode string

const (
	LaunchModeTerminalPTY    LaunchMode = "terminal_pty"
	LaunchModeCodexAppServer LaunchMode = "codex_app_server"
	LaunchModeNativeAPI      LaunchMode = "native_api"
	LaunchModeUnknown        LaunchMode = "unknown"
)

type ControlMode string

const (
	ControlModeTranscriptOnly ControlMode = "transcript_only"
	ControlModeNativeCapable  ControlMode = "native_capable"
	ControlModeRecoveryOnly   ControlMode = "recovery_only"
	ControlModeUnknown        ControlMode = "unknown"
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

	LaunchMode              LaunchMode
	ControlMode             ControlMode
	NativeOwner             string
	NativeTransport         string
	ProviderVersion         string
	NativeThreadID          string
	LastVerifiedTurnID      string
	ForkedFromNativeSession string
	LaunchDescriptorJSON    string
}

// AgentInfo is the partial-update payload used when a populator (codex tailer,
// claude hook, launch-command save) learns more about a session's agent
// identity. Empty fields mean "leave unchanged".
type AgentInfo struct {
	AgentType               Agent
	LaunchCommand           string
	AgentSessionID          string
	CWD                     string
	LastRolloutPath         string
	LastActivityAt          time.Time
	LaunchMode              LaunchMode
	ControlMode             ControlMode
	NativeOwner             string
	NativeTransport         string
	ProviderVersion         string
	NativeThreadID          string
	LastVerifiedTurnID      string
	ForkedFromNativeSession string
	LaunchDescriptorJSON    string
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
	// Non-expiring sessions may omit policy metadata. Normalize that omission
	// before the upsert so SQLite does not receive an invalid empty enum value.
	if meta.Policy.Mode == "" {
		meta.Policy = policy.Default()
	}
	if meta.Status == "" {
		meta.Status = StatusLive
	}
	if meta.AgentType == "" {
		meta.AgentType = AgentNone
	}
	if meta.LaunchMode == "" || meta.LaunchMode == LaunchModeUnknown {
		if meta.AgentType == AgentCodex {
			meta.LaunchMode = LaunchModeTerminalPTY
			meta.ControlMode = ControlModeTranscriptOnly
		} else {
			meta.LaunchMode = LaunchModeUnknown
		}
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (
			id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached,
			status, agent_type, launch_command, agent_session_id, cwd, last_rollout_path,
			last_activity_at, orphaned_at, recovered_into, archived_at, origin, owner, display_label,
			launch_mode, control_mode, native_owner, native_transport, provider_version, native_thread_id,
			last_verified_turn_id, forked_from_native_session, launch_descriptor_json
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
			?)
		ON CONFLICT(id) DO UPDATE SET
			backend=excluded.backend, shell=excluded.shell, cols=excluded.cols, rows=excluded.rows,
			policy_mode=excluded.policy_mode, policy_duration=excluded.policy_duration,
			detached=excluded.detached, agent_type=excluded.agent_type,
			launch_command=excluded.launch_command, agent_session_id=excluded.agent_session_id,
			cwd=excluded.cwd, last_rollout_path=excluded.last_rollout_path,
			last_activity_at=excluded.last_activity_at, origin=excluded.origin,
			owner=excluded.owner, display_label=excluded.display_label,
			launch_mode=excluded.launch_mode, control_mode=excluded.control_mode,
			native_owner=excluded.native_owner, native_transport=excluded.native_transport,
			provider_version=excluded.provider_version, native_thread_id=excluded.native_thread_id,
			last_verified_turn_id=excluded.last_verified_turn_id,
			forked_from_native_session=excluded.forked_from_native_session,
			launch_descriptor_json=excluded.launch_descriptor_json`,
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
		string(meta.LaunchMode), string(meta.ControlMode), meta.NativeOwner,
		meta.NativeTransport, meta.ProviderVersion, meta.NativeThreadID,
		meta.LastVerifiedTurnID, meta.ForkedFromNativeSession, meta.LaunchDescriptorJSON,
	)
	return err
}

const selectColumns = `
	id, backend, shell, cols, rows, policy_mode, policy_duration, created_at, detached,
	status, agent_type, launch_command, agent_session_id, cwd, last_rollout_path,
	last_activity_at, orphaned_at, recovered_into, archived_at, origin, owner, display_label,
	launch_mode, control_mode, native_owner, native_transport, provider_version, native_thread_id,
	last_verified_turn_id, forked_from_native_session, launch_descriptor_json`

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
		WHERE detached = 1 AND status = 'live' AND archived_at = ''
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
	if info.LaunchMode != "" {
		sets = append(sets, "launch_mode = ?")
		args = append(args, string(info.LaunchMode))
	}
	if info.ControlMode != "" {
		sets = append(sets, "control_mode = ?")
		args = append(args, string(info.ControlMode))
	}
	if info.NativeOwner != "" {
		sets = append(sets, "native_owner = ?")
		args = append(args, info.NativeOwner)
	}
	if info.NativeTransport != "" {
		sets = append(sets, "native_transport = ?")
		args = append(args, info.NativeTransport)
	}
	if info.ProviderVersion != "" {
		sets = append(sets, "provider_version = ?")
		args = append(args, info.ProviderVersion)
	}
	if info.NativeThreadID != "" {
		sets = append(sets, "native_thread_id = ?")
		args = append(args, info.NativeThreadID)
	}
	if info.LastVerifiedTurnID != "" {
		sets = append(sets, "last_verified_turn_id = ?")
		args = append(args, info.LastVerifiedTurnID)
	}
	if info.ForkedFromNativeSession != "" {
		sets = append(sets, "forked_from_native_session = ?")
		args = append(args, info.ForkedFromNativeSession)
	}
	if info.LaunchDescriptorJSON != "" {
		sets = append(sets, "launch_descriptor_json = ?")
		args = append(args, info.LaunchDescriptorJSON)
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
		WHERE id = ? AND archived_at = ''`,
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
		launchMode, controlMode              string
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
		&launchMode, &controlMode, &meta.NativeOwner, &meta.NativeTransport,
		&meta.ProviderVersion, &meta.NativeThreadID, &meta.LastVerifiedTurnID,
		&meta.ForkedFromNativeSession, &meta.LaunchDescriptorJSON,
	)
	if err != nil {
		return meta, fmt.Errorf("scan session metadata: %w", err)
	}
	meta.Origin = Origin(origin)
	meta.LaunchMode = LaunchMode(launchMode)
	meta.ControlMode = ControlMode(controlMode)
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
	if meta.LaunchMode == "" || meta.LaunchMode == LaunchModeUnknown {
		if meta.AgentType == AgentCodex {
			meta.LaunchMode = LaunchModeTerminalPTY
			meta.ControlMode = ControlModeTranscriptOnly
		}
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
	if previous, exists := s.sessions[meta.ID]; exists {
		// Lifecycle columns are owned by explicit transition methods. A late
		// runtime save must not resurrect an archived or recovered row.
		meta.Status = previous.Status
		meta.OrphanedAt = previous.OrphanedAt
		meta.RecoveredInto = previous.RecoveredInto
		meta.ArchivedAt = previous.ArchivedAt
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
		if meta.Detached && meta.Status == StatusLive && meta.ArchivedAt.IsZero() {
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
	if info.LaunchMode != "" {
		meta.LaunchMode = info.LaunchMode
	}
	if info.ControlMode != "" {
		meta.ControlMode = info.ControlMode
	}
	if info.NativeOwner != "" {
		meta.NativeOwner = info.NativeOwner
	}
	if info.NativeTransport != "" {
		meta.NativeTransport = info.NativeTransport
	}
	if info.ProviderVersion != "" {
		meta.ProviderVersion = info.ProviderVersion
	}
	if info.NativeThreadID != "" {
		meta.NativeThreadID = info.NativeThreadID
	}
	if info.LastVerifiedTurnID != "" {
		meta.LastVerifiedTurnID = info.LastVerifiedTurnID
	}
	if info.ForkedFromNativeSession != "" {
		meta.ForkedFromNativeSession = info.ForkedFromNativeSession
	}
	if info.LaunchDescriptorJSON != "" {
		meta.LaunchDescriptorJSON = info.LaunchDescriptorJSON
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
	if !meta.ArchivedAt.IsZero() {
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
