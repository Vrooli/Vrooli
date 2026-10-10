package scenarioruntime

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Editor lease statuses.
const (
	EditorLeaseActive  = "active"
	EditorLeaseStopped = "stopped"
	EditorLeaseExpired = "expired"
)

const editorLeaseSelectSQL = `
SELECT session_id, harness, agent, pid, host_boot_id, working_dir, scope, containment_method, claims_json,
  status, created_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason,
  owner_kind, owner_ref, owner_evidence_json, owner_source, owner_recorded_at
FROM runtime_editor_leases`

// CreateEditorLease records a live agent session. A session id that already
// holds an active lease is refreshed in place: the launcher may attach twice
// (spawn after a failed exec), and the row is the session, not the attempt.
func (s *SQLiteStore) CreateEditorLease(ctx context.Context, lease EditorLease, ttl time.Duration) (EditorLease, error) {
	lease.SessionID = strings.TrimSpace(lease.SessionID)
	if lease.SessionID == "" {
		return EditorLease{}, fmt.Errorf("create editor lease: session_id is required")
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	claims, err := json.Marshal(nonNil(lease.Claims))
	if err != nil {
		return EditorLease{}, fmt.Errorf("create editor lease: %w", err)
	}
	owner := normalizeEditorLeaseOwner(lease.Owner, EditorLeaseOwnerLaunch)
	var ownerRecordedAt any
	if owner.Kind != "" {
		ownerRecordedAt = formatTime(now)
	}
	var out EditorLease
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO runtime_editor_leases (session_id, harness, agent, pid, host_boot_id, working_dir, scope, containment_method, claims_json,
  status, created_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason,
  owner_kind, owner_ref, owner_evidence_json, owner_source, owner_recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, '', ?, ?, ?, ?, ?)
ON CONFLICT(session_id) DO UPDATE SET
  harness = excluded.harness, agent = excluded.agent, pid = excluded.pid, host_boot_id = excluded.host_boot_id,
  working_dir = excluded.working_dir, scope = excluded.scope, containment_method = excluded.containment_method,
  claims_json = excluded.claims_json, status = excluded.status, last_heartbeat_at = excluded.last_heartbeat_at,
  heartbeat_deadline_at = excluded.heartbeat_deadline_at, stopped_at = NULL, stop_reason = '',
  owner_kind = CASE WHEN excluded.owner_kind <> '' THEN excluded.owner_kind ELSE runtime_editor_leases.owner_kind END,
  owner_ref = CASE WHEN excluded.owner_kind <> '' THEN excluded.owner_ref ELSE runtime_editor_leases.owner_ref END,
  owner_evidence_json = CASE WHEN excluded.owner_kind <> '' THEN excluded.owner_evidence_json ELSE runtime_editor_leases.owner_evidence_json END,
  owner_source = CASE WHEN excluded.owner_kind <> '' THEN excluded.owner_source ELSE runtime_editor_leases.owner_source END,
  owner_recorded_at = CASE WHEN excluded.owner_kind <> '' THEN excluded.owner_recorded_at ELSE runtime_editor_leases.owner_recorded_at END`,
			lease.SessionID, lease.Harness, lease.Agent, optionalPID(lease.PID), lease.HostBootID, lease.WorkingDir, lease.Scope,
			lease.ContainmentMethod, string(claims), EditorLeaseActive, formatTime(now), formatTime(now), formatTime(deadline),
			owner.Kind, owner.Ref, owner.EvidenceJSON, owner.Source, ownerRecordedAt)
		if err != nil {
			return fmt.Errorf("create editor lease: %w", err)
		}
		out, err = getEditorLeaseTx(ctx, tx, lease.SessionID)
		return err
	})
	if err != nil {
		return EditorLease{}, err
	}
	return out, nil
}

// HeartbeatEditorLease renews an active lease's deadline.
func (s *SQLiteStore) HeartbeatEditorLease(ctx context.Context, sessionID string, ttl time.Duration) (EditorLease, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return EditorLease{}, fmt.Errorf("heartbeat editor lease: session_id is required")
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	var out EditorLease
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE runtime_editor_leases SET last_heartbeat_at = ?, heartbeat_deadline_at = ? WHERE session_id = ? AND status = ?`,
			formatTime(now), formatTime(deadline), sessionID, EditorLeaseActive)
		if err != nil {
			return fmt.Errorf("heartbeat editor lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getEditorLeaseTx(ctx, tx, sessionID)
		return err
	})
	if err != nil {
		return EditorLease{}, err
	}
	return out, nil
}

// StopEditorLease ends a session's lease with a reason.
func (s *SQLiteStore) StopEditorLease(ctx context.Context, sessionID, reason string) (EditorLease, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return EditorLease{}, fmt.Errorf("stop editor lease: session_id is required")
	}
	now := s.now()
	var out EditorLease
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `UPDATE runtime_editor_leases SET status = ?, stopped_at = ?, stop_reason = ? WHERE session_id = ? AND status = ?`,
			EditorLeaseStopped, formatTime(now), strings.TrimSpace(reason), sessionID, EditorLeaseActive)
		if err != nil {
			return fmt.Errorf("stop editor lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getEditorLeaseTx(ctx, tx, sessionID)
		return err
	})
	if err != nil {
		return EditorLease{}, err
	}
	return out, nil
}

// ListEditorLeases returns active leases (and, when asked, stopped and
// expired ones) newest first.
func (s *SQLiteStore) ListEditorLeases(ctx context.Context, includeStopped bool) ([]EditorLease, error) {
	query := editorLeaseSelectSQL
	args := []any{}
	if !includeStopped {
		query += ` WHERE status = ?`
		args = append(args, EditorLeaseActive)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list editor leases: %w", err)
	}
	defer rows.Close()
	var out []EditorLease
	for rows.Next() {
		lease, err := scanEditorLease(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, lease)
	}
	return out, rows.Err()
}

// ExpireStaleEditorLeases retires active leases whose deadline has passed AND
// whose owner is provably gone: a boot id from another boot, no pid, or a pid
// this host can prove dead. A past deadline alone never expires a lease; a
// session that is busy building is slow, not dead.
func (s *SQLiteStore) ExpireStaleEditorLeases(ctx context.Context, at time.Time, guard StartingLeaseGuard) ([]EditorLease, error) {
	now := s.now()
	var expired []EditorLease
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, editorLeaseSelectSQL+` WHERE status = ? AND heartbeat_deadline_at <= ? ORDER BY created_at`, EditorLeaseActive, formatTime(at.UTC()))
		if err != nil {
			return fmt.Errorf("list overdue editor leases: %w", err)
		}
		var overdue []EditorLease
		for rows.Next() {
			lease, err := scanEditorLease(rows)
			if err != nil {
				rows.Close()
				return err
			}
			overdue = append(overdue, lease)
		}
		if err := rows.Close(); err != nil {
			return err
		}
		for _, lease := range overdue {
			trigger, dead := staleEditorLeaseTrigger(lease, guard)
			if !dead {
				continue
			}
			if _, err := tx.ExecContext(ctx, `UPDATE runtime_editor_leases SET status = ?, stopped_at = ?, stop_reason = ? WHERE session_id = ? AND status = ?`,
				EditorLeaseExpired, formatTime(now), "stale editor lease: "+trigger, lease.SessionID, EditorLeaseActive); err != nil {
				return fmt.Errorf("expire editor lease %s: %w", lease.SessionID, err)
			}
			lease.Status = EditorLeaseExpired
			lease.StopReason = "stale editor lease: " + trigger
			expired = append(expired, lease)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return expired, nil
}

// staleEditorLeaseTrigger applies the same proof-of-death rule as the
// starting-lease reaper.
func staleEditorLeaseTrigger(lease EditorLease, guard StartingLeaseGuard) (string, bool) {
	if guard.CurrentBootID != "" && lease.HostBootID != "" && lease.HostBootID != guard.CurrentBootID {
		return "boot_id_mismatch", true
	}
	if lease.PID <= 0 {
		return "owner_pid_missing", true
	}
	if guard.PIDRunning != nil && !guard.PIDRunning(lease.PID) {
		return "owner_pid_dead", true
	}
	return "", false
}

type editorLeaseScanner interface {
	Scan(dest ...any) error
}

func getEditorLeaseTx(ctx context.Context, tx *sql.Tx, sessionID string) (EditorLease, error) {
	lease, err := scanEditorLease(tx.QueryRowContext(ctx, editorLeaseSelectSQL+` WHERE session_id = ?`, sessionID))
	if err == sql.ErrNoRows {
		return EditorLease{}, ErrNotFound
	}
	return lease, err
}

func scanEditorLease(row editorLeaseScanner) (EditorLease, error) {
	var lease EditorLease
	var pid sql.NullInt64
	var claims, createdAt, heartbeatAt, deadlineAt string
	var stoppedAt, ownerRecordedAt sql.NullString
	if err := row.Scan(&lease.SessionID, &lease.Harness, &lease.Agent, &pid, &lease.HostBootID, &lease.WorkingDir, &lease.Scope, &lease.ContainmentMethod, &claims,
		&lease.Status, &createdAt, &heartbeatAt, &deadlineAt, &stoppedAt, &lease.StopReason,
		&lease.Owner.Kind, &lease.Owner.Ref, &lease.Owner.EvidenceJSON, &lease.Owner.Source, &ownerRecordedAt); err != nil {
		return EditorLease{}, err
	}
	if pid.Valid {
		lease.PID = int(pid.Int64)
	}
	_ = json.Unmarshal([]byte(claims), &lease.Claims)
	var err error
	if lease.CreatedAt, err = parseRequiredTime(createdAt); err != nil {
		return EditorLease{}, err
	}
	if lease.LastHeartbeatAt, err = parseRequiredTime(heartbeatAt); err != nil {
		return EditorLease{}, err
	}
	if lease.HeartbeatDeadlineAt, err = parseRequiredTime(deadlineAt); err != nil {
		return EditorLease{}, err
	}
	if lease.StoppedAt, err = parseOptionalTime(stoppedAt); err != nil {
		return EditorLease{}, err
	}
	if lease.Owner.RecordedAt, err = parseOptionalTime(ownerRecordedAt); err != nil {
		return EditorLease{}, err
	}
	return lease, nil
}

// normalizeEditorLeaseOwner trims the owner and stamps a default source. An
// owner without a kind is "not recorded" and carries nothing else.
func normalizeEditorLeaseOwner(owner EditorLeaseOwner, defaultSource string) EditorLeaseOwner {
	owner.Kind = strings.TrimSpace(owner.Kind)
	if owner.Kind == "" {
		return EditorLeaseOwner{}
	}
	owner.Ref = strings.TrimSpace(owner.Ref)
	owner.Source = strings.TrimSpace(owner.Source)
	if owner.Source == "" {
		owner.Source = defaultSource
	}
	return owner
}

// BackfillEditorLeaseOwner records an owner inferred after launch for an
// active lease that has none. It never overwrites an owner already recorded:
// the launch-time record is the stronger evidence, and a second inference is
// not allowed to replace the first. It reports whether a row changed.
func (s *SQLiteStore) BackfillEditorLeaseOwner(ctx context.Context, sessionID string, owner EditorLeaseOwner) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return false, fmt.Errorf("backfill editor lease owner: session_id is required")
	}
	owner = normalizeEditorLeaseOwner(owner, EditorLeaseOwnerBackfill)
	if owner.Kind == "" {
		return false, fmt.Errorf("backfill editor lease owner: owner kind is required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE runtime_editor_leases
SET owner_kind = ?, owner_ref = ?, owner_evidence_json = ?, owner_source = ?, owner_recorded_at = ?
WHERE session_id = ? AND status = ? AND owner_kind = ''`,
		owner.Kind, owner.Ref, owner.EvidenceJSON, owner.Source, formatTime(s.now()), sessionID, EditorLeaseActive)
	if err != nil {
		return false, fmt.Errorf("backfill editor lease owner: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// RefreshEditorLeaseOwnerEvidence replaces the evidence of an owner already
// recorded, only when the kind and ref still match: what an owner's authority
// later reports (web-console's origin for a live session) is added to the
// record, and a different owner can never be written over it this way.
func (s *SQLiteStore) RefreshEditorLeaseOwnerEvidence(ctx context.Context, sessionID string, owner EditorLeaseOwner) (bool, error) {
	sessionID = strings.TrimSpace(sessionID)
	owner.Kind = strings.TrimSpace(owner.Kind)
	if sessionID == "" || owner.Kind == "" {
		return false, fmt.Errorf("refresh editor lease owner evidence: session_id and owner kind are required")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE runtime_editor_leases SET owner_evidence_json = ?
WHERE session_id = ? AND status = ? AND owner_kind = ? AND owner_ref = ?`,
		owner.EvidenceJSON, sessionID, EditorLeaseActive, owner.Kind, strings.TrimSpace(owner.Ref))
	if err != nil {
		return false, fmt.Errorf("refresh editor lease owner evidence: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

// RecordAgentSessionReap appends one reap audit row.
func (s *SQLiteStore) RecordAgentSessionReap(ctx context.Context, reap AgentSessionReap) (AgentSessionReap, error) {
	reap.Outcome = strings.TrimSpace(reap.Outcome)
	if reap.Outcome == "" {
		return AgentSessionReap{}, fmt.Errorf("record agent session reap: outcome is required")
	}
	if strings.TrimSpace(reap.SessionID) == "" && strings.TrimSpace(reap.Scope) == "" {
		return AgentSessionReap{}, fmt.Errorf("record agent session reap: a session id or scope is required")
	}
	if strings.TrimSpace(reap.ReapID) == "" {
		reap.ReapID = newID("reap")
	}
	if reap.CreatedAt.IsZero() {
		reap.CreatedAt = s.now()
	}
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_agent_session_reaps (reap_id, session_id, scope, owner_kind, owner_ref, owner_evidence_json, liveness_detail,
  memory_bytes, swap_bytes, tasks, outcome, error, initiator, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		reap.ReapID, reap.SessionID, reap.Scope, reap.OwnerKind, reap.OwnerRef, reap.OwnerEvidenceJSON, reap.LivenessDetail,
		reap.MemoryBytes, reap.SwapBytes, reap.Tasks, reap.Outcome, reap.Error, reap.Initiator, formatTime(reap.CreatedAt))
	if err != nil {
		return AgentSessionReap{}, fmt.Errorf("record agent session reap: %w", err)
	}
	return reap, nil
}

// ListAgentSessionReaps returns the newest reap audits first.
func (s *SQLiteStore) ListAgentSessionReaps(ctx context.Context, limit int) ([]AgentSessionReap, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT reap_id, session_id, scope, owner_kind, owner_ref, owner_evidence_json, liveness_detail,
  memory_bytes, swap_bytes, tasks, outcome, error, initiator, created_at
FROM runtime_agent_session_reaps ORDER BY created_at DESC, reap_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("list agent session reaps: %w", err)
	}
	defer rows.Close()
	var out []AgentSessionReap
	for rows.Next() {
		var reap AgentSessionReap
		var createdAt string
		if err := rows.Scan(&reap.ReapID, &reap.SessionID, &reap.Scope, &reap.OwnerKind, &reap.OwnerRef, &reap.OwnerEvidenceJSON, &reap.LivenessDetail,
			&reap.MemoryBytes, &reap.SwapBytes, &reap.Tasks, &reap.Outcome, &reap.Error, &reap.Initiator, &createdAt); err != nil {
			return nil, err
		}
		if reap.CreatedAt, err = parseRequiredTime(createdAt); err != nil {
			return nil, err
		}
		out = append(out, reap)
	}
	return out, rows.Err()
}

func optionalPID(pid int) any {
	if pid <= 0 {
		return nil
	}
	return pid
}

func nonNil(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

// LiveEditorLeases keeps the leases whose owner is not provably dead. It is
// the read-side twin of ExpireStaleEditorLeases: a list must not show a
// session whose pid is gone just because the sweep has not run yet, and it
// must not evict anything either.
func LiveEditorLeases(leases []EditorLease, guard StartingLeaseGuard) []EditorLease {
	out := make([]EditorLease, 0, len(leases))
	for _, lease := range leases {
		if _, dead := staleEditorLeaseTrigger(lease, guard); dead {
			continue
		}
		out = append(out, lease)
	}
	return out
}
