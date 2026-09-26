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
  status, created_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason
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
	var out EditorLease
	err = s.withTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `
INSERT INTO runtime_editor_leases (session_id, harness, agent, pid, host_boot_id, working_dir, scope, containment_method, claims_json,
  status, created_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NULL, '')
ON CONFLICT(session_id) DO UPDATE SET
  harness = excluded.harness, agent = excluded.agent, pid = excluded.pid, host_boot_id = excluded.host_boot_id,
  working_dir = excluded.working_dir, scope = excluded.scope, containment_method = excluded.containment_method,
  claims_json = excluded.claims_json, status = excluded.status, last_heartbeat_at = excluded.last_heartbeat_at,
  heartbeat_deadline_at = excluded.heartbeat_deadline_at, stopped_at = NULL, stop_reason = ''`,
			lease.SessionID, lease.Harness, lease.Agent, optionalPID(lease.PID), lease.HostBootID, lease.WorkingDir, lease.Scope,
			lease.ContainmentMethod, string(claims), EditorLeaseActive, formatTime(now), formatTime(now), formatTime(deadline))
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
	var stoppedAt sql.NullString
	if err := row.Scan(&lease.SessionID, &lease.Harness, &lease.Agent, &pid, &lease.HostBootID, &lease.WorkingDir, &lease.Scope, &lease.ContainmentMethod, &claims,
		&lease.Status, &createdAt, &heartbeatAt, &deadlineAt, &stoppedAt, &lease.StopReason); err != nil {
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
	return lease, nil
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
