package scenarioruntime

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrSupervisorAlreadyRunning = errors.New("runtime supervisor already running")

// ClaimSupervisorSession atomically establishes the one running supervisor
// for the database.
func (s *SQLiteStore) ClaimSupervisorSession(ctx context.Context, session SupervisorSession, ttl time.Duration) (SupervisorSession, error) {
	if strings.TrimSpace(session.SupervisorID) == "" {
		session.SupervisorID = newID("sup")
	}
	if strings.TrimSpace(session.HostBootID) == "" || strings.TrimSpace(session.HostSessionID) == "" {
		return SupervisorSession{}, fmt.Errorf("claim supervisor session: host identity is required")
	}
	now := s.now()
	if session.Status == "" {
		session.Status = SupervisorStatusRunning
	}
	if session.StartedAt.IsZero() {
		session.StartedAt = now
	}
	session.LastHeartbeatAt = now
	session.HeartbeatDeadlineAt = now.Add(NormalizeHeartbeatTTL(ttl))
	var claimed SupervisorSession
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		var existingID string
		err := tx.QueryRowContext(ctx, `SELECT supervisor_id FROM runtime_supervisor_sessions WHERE status = ? LIMIT 1`, SupervisorStatusRunning).Scan(&existingID)
		if err == nil {
			return fmt.Errorf("%w: supervisor_id=%s", ErrSupervisorAlreadyRunning, existingID)
		}
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("inspect running supervisor session: %w", err)
		}
		_, err = tx.ExecContext(ctx, `
INSERT INTO runtime_supervisor_sessions (
  supervisor_id, host_boot_id, host_session_id, pid, status, started_at,
  last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason, version, metadata_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			session.SupervisorID, session.HostBootID, session.HostSessionID, optionalIntValue(session.PID),
			session.Status, formatTime(session.StartedAt), formatTime(session.LastHeartbeatAt),
			formatTime(session.HeartbeatDeadlineAt), formatOptionalTime(session.StoppedAt), session.StopReason,
			session.Version, session.MetadataJSON)
		if err != nil {
			return fmt.Errorf("insert claimed supervisor session: %w", err)
		}
		claimed = session
		return nil
	})
	if err != nil {
		return SupervisorSession{}, err
	}
	return claimed, nil
}

func (s *SQLiteStore) CreateSupervisorSession(ctx context.Context, session SupervisorSession, ttl time.Duration) (SupervisorSession, error) {
	if strings.TrimSpace(session.SupervisorID) == "" {
		session.SupervisorID = newID("sup")
	}
	if strings.TrimSpace(session.HostBootID) == "" {
		return SupervisorSession{}, fmt.Errorf("create supervisor session: host_boot_id is required")
	}
	if strings.TrimSpace(session.HostSessionID) == "" {
		return SupervisorSession{}, fmt.Errorf("create supervisor session: host_session_id is required")
	}
	now := s.now()
	if session.Status == "" {
		session.Status = SupervisorStatusRunning
	}
	if session.StartedAt.IsZero() {
		session.StartedAt = now
	}
	session.LastHeartbeatAt = now
	session.HeartbeatDeadlineAt = now.Add(NormalizeHeartbeatTTL(ttl))
	_, err := s.db.ExecContext(ctx, `
INSERT INTO runtime_supervisor_sessions (
  supervisor_id, host_boot_id, host_session_id, pid, status, started_at,
  last_heartbeat_at, heartbeat_deadline_at, stopped_at, stop_reason, version, metadata_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		session.SupervisorID, session.HostBootID, session.HostSessionID, optionalIntValue(session.PID),
		session.Status, formatTime(session.StartedAt), formatTime(session.LastHeartbeatAt),
		formatTime(session.HeartbeatDeadlineAt), formatOptionalTime(session.StoppedAt),
		session.StopReason, session.Version, session.MetadataJSON)
	if err != nil {
		return SupervisorSession{}, fmt.Errorf("insert supervisor session: %w", err)
	}
	return session, nil
}

func (s *SQLiteStore) HeartbeatSupervisorSession(ctx context.Context, supervisorID string, ttl time.Duration) (SupervisorSession, error) {
	if strings.TrimSpace(supervisorID) == "" {
		return SupervisorSession{}, fmt.Errorf("heartbeat supervisor session: supervisor_id is required")
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	var out SupervisorSession
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_supervisor_sessions
SET last_heartbeat_at = ?, heartbeat_deadline_at = ?, status = ?
WHERE supervisor_id = ? AND status IN (?, ?)`,
			formatTime(now), formatTime(deadline), SupervisorStatusRunning, supervisorID,
			SupervisorStatusRunning, SupervisorStatusStopping)
		if err != nil {
			return fmt.Errorf("heartbeat supervisor session: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect supervisor session heartbeat: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getSupervisorSessionTx(ctx, tx, supervisorID)
		return err
	})
	if err != nil {
		return SupervisorSession{}, err
	}
	return out, nil
}

func (s *SQLiteStore) StopSupervisorSession(ctx context.Context, supervisorID string, status string, reason string) (SupervisorSession, error) {
	if strings.TrimSpace(supervisorID) == "" {
		return SupervisorSession{}, fmt.Errorf("stop supervisor session: supervisor_id is required")
	}
	if status == "" {
		status = SupervisorStatusStopped
	}
	now := s.now()
	var out SupervisorSession
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_supervisor_sessions
SET status = ?, stopped_at = ?, stop_reason = ?
WHERE supervisor_id = ?`,
			status, formatTime(now), reason, supervisorID)
		if err != nil {
			return fmt.Errorf("stop supervisor session: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect supervisor session stop: %w", err)
		}
		if affected == 0 {
			return ErrNotFound
		}
		out, err = getSupervisorSessionTx(ctx, tx, supervisorID)
		return err
	})
	if err != nil {
		return SupervisorSession{}, err
	}
	return out, nil
}

func (s *SQLiteStore) ListSupervisorSessions(ctx context.Context, filter SupervisorSessionFilter) ([]SupervisorSession, error) {
	query := supervisorSessionSelectSQL
	args := []any{}
	if len(filter.Statuses) > 0 {
		query += " WHERE status IN (" + placeholders(len(filter.Statuses)) + ")"
		for _, status := range filter.Statuses {
			args = append(args, status)
		}
	}
	query += " ORDER BY started_at DESC, supervisor_id ASC"
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list supervisor sessions: %w", err)
	}
	defer rows.Close()
	return scanSupervisorSessions(rows)
}

const staleSupervisorStopReasonPrefix = "stale supervisor session"

// StaleSupervisorTrigger reports why a session claiming to run is provably
// dead, or ("", false) when it still looks alive.
//
// Sessions only ever reached a terminal status through graceful shutdown, so
// any supervisor that was SIGKILLed — which, before its unit was repaired, was
// every supervisor — left a row permanently claiming status='running'. Those
// rows are not harmless bookkeeping: Status() answers from the newest running
// session, and an attach looking for "the live supervisor" would happily hand
// an instance to a corpse.
//
// Positive evidence only, mirroring StaleStartingTrigger: an elapsed deadline
// alone never condemns a session whose process is still alive on this boot,
// because a slow or briefly-wedged supervisor is still the fleet's owner.
func StaleSupervisorTrigger(session SupervisorSession, guard StartingLeaseGuard, now time.Time) (string, bool) {
	if session.Status != SupervisorStatusRunning {
		return "", false
	}
	// A session written by a previous boot cannot be alive whatever its
	// deadline says: PIDs do not survive a reboot, and a recycled PID would
	// make the liveness probe lie.
	if guard.CurrentBootID != "" && session.HostBootID != "" && session.HostBootID != guard.CurrentBootID {
		return "boot_id_mismatch", true
	}
	if !session.HeartbeatDeadlineAt.IsZero() && session.HeartbeatDeadlineAt.After(now) {
		return "", false
	}
	if session.PID == nil || *session.PID <= 0 {
		return "owner_pid_missing", true
	}
	if guard.PIDRunning != nil && !guard.PIDRunning(*session.PID) {
		return "owner_pid_dead", true
	}
	return "", false
}

// ExpireStaleSupervisorSessions marks provably dead sessions failed and returns
// them. Safe to run from any process: it only transitions rows the guard can
// prove are dead, so a live supervisor never reaps a live peer.
func (s *SQLiteStore) ExpireStaleSupervisorSessions(ctx context.Context, at time.Time, guard StartingLeaseGuard) ([]SupervisorSession, error) {
	sessions, err := s.ListSupervisorSessions(ctx, SupervisorSessionFilter{Statuses: []string{SupervisorStatusRunning}})
	if err != nil {
		return nil, err
	}
	var expired []SupervisorSession
	for _, session := range sessions {
		trigger, stale := StaleSupervisorTrigger(session, guard, at)
		if !stale {
			continue
		}
		reason := fmt.Sprintf("%s (%s)", staleSupervisorStopReasonPrefix, trigger)
		stopped, err := s.StopSupervisorSession(ctx, session.SupervisorID, SupervisorStatusFailed, reason)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, fmt.Errorf("expire stale supervisor session %s: %w", session.SupervisorID, err)
		}
		expired = append(expired, stopped)
	}
	return expired, nil
}

// AttachLiveSupervision hands a running instance to whichever supervisor
// session is currently live, in one transaction, and refreshes its lease so the
// handover starts with a full window.
//
// This is what keeps lifecycle ownership from outliving the command that
// created it. A start used to finish by spawning a supervisor and returning
// immediately, leaving the instance owned by a CLI process that exits
// milliseconds later: the lease then had a dead owner and no renewer, and the
// scenario read as stopped until some later tick adopted it (measured mean: 24
// minutes). Attaching here closes that window entirely rather than shrinking
// it — there is no adoption latency because there is no adoption to wait for.
//
// Reports false when no live session exists; the caller keeps lifecycle
// ownership and says so rather than handing the instance to a corpse.
func (s *SQLiteStore) AttachLiveSupervision(ctx context.Context, instanceID string, generation int64, ttl time.Duration) (Instance, bool, error) {
	if strings.TrimSpace(instanceID) == "" {
		return Instance{}, false, fmt.Errorf("attach supervision: instance_id is required")
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	var out Instance
	attached := false
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		attached = false
		// Selecting the session inside the transaction is what makes this
		// safe: a separate read-then-claim could attach to a session that
		// died in between.
		var supervisorID string
		err := tx.QueryRowContext(ctx, `
SELECT supervisor_id FROM runtime_supervisor_sessions
WHERE status = ? AND heartbeat_deadline_at > ?
ORDER BY last_heartbeat_at DESC, started_at DESC
LIMIT 1`, SupervisorStatusRunning, formatTime(now)).Scan(&supervisorID)
		if errors.Is(err, sql.ErrNoRows) {
			out, err = getInstanceTx(ctx, tx, instanceID)
			return err
		}
		if err != nil {
			return fmt.Errorf("select live supervisor session: %w", err)
		}
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET supervisor_id = ?, supervised_at = ?, owner_kind = ?, owner_pid = NULL,
    last_heartbeat_at = ?, heartbeat_deadline_at = ?, updated_at = ?
WHERE instance_id = ? AND generation = ? AND status = ?`,
			supervisorID, formatTime(now), OwnerKindSupervisor,
			formatTime(now), formatTime(deadline), formatTime(now),
			instanceID, generation, StatusRunning)
		if err != nil {
			return fmt.Errorf("attach runtime supervision: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime supervision attach: %w", err)
		}
		if affected == 0 {
			return staleGenerationOrNotFound(ctx, tx, instanceID)
		}
		attached = true
		out, err = getInstanceTx(ctx, tx, instanceID)
		return err
	})
	if err != nil {
		return Instance{}, false, err
	}
	return out, attached, nil
}

func (s *SQLiteStore) ClaimSupervision(ctx context.Context, claim SupervisionClaim) (Instance, error) {
	if strings.TrimSpace(claim.InstanceID) == "" {
		return Instance{}, fmt.Errorf("claim supervision: instance_id is required")
	}
	if strings.TrimSpace(claim.SupervisorID) == "" {
		return Instance{}, fmt.Errorf("claim supervision: supervisor_id is required")
	}
	now := s.now()
	var out Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET supervisor_id = ?, supervised_at = ?, owner_kind = ?, owner_pid = NULL, updated_at = ?
WHERE instance_id = ? AND generation = ? AND status = ?`,
			claim.SupervisorID, formatTime(now), OwnerKindSupervisor, formatTime(now),
			claim.InstanceID, claim.Generation, StatusRunning)
		if err != nil {
			return fmt.Errorf("claim runtime supervision: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime supervision claim: %w", err)
		}
		if affected == 0 {
			return staleGenerationOrNotFound(ctx, tx, claim.InstanceID)
		}
		out, err = getInstanceTx(ctx, tx, claim.InstanceID)
		return err
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

func (s *SQLiteStore) ReleaseSupervision(ctx context.Context, instanceID string, generation int64, supervisorID string) (Instance, error) {
	if strings.TrimSpace(instanceID) == "" {
		return Instance{}, fmt.Errorf("release supervision: instance_id is required")
	}
	now := s.now()
	var out Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET supervisor_id = '', supervised_at = NULL, updated_at = ?
WHERE instance_id = ? AND generation = ? AND supervisor_id = ?`,
			formatTime(now), instanceID, generation, supervisorID)
		if err != nil {
			return fmt.Errorf("release runtime supervision: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime supervision release: %w", err)
		}
		if affected == 0 {
			return staleGenerationOrNotFound(ctx, tx, instanceID)
		}
		out, err = getInstanceTx(ctx, tx, instanceID)
		return err
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

func (s *SQLiteStore) UpdateInstanceReconciliation(ctx context.Context, instanceID string, generation int64, status string, reason string) (Instance, error) {
	if strings.TrimSpace(instanceID) == "" {
		return Instance{}, fmt.Errorf("update reconciliation: instance_id is required")
	}
	now := s.now()
	var out Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET last_reconciled_at = ?, reconciliation_status = ?, reconciliation_reason = ?, updated_at = ?
WHERE instance_id = ? AND generation = ?`,
			formatTime(now), status, reason, formatTime(now), instanceID, generation)
		if err != nil {
			return fmt.Errorf("update runtime reconciliation: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime reconciliation update: %w", err)
		}
		if affected == 0 {
			return staleGenerationOrNotFound(ctx, tx, instanceID)
		}
		out, err = getInstanceTx(ctx, tx, instanceID)
		return err
	})
	if err != nil {
		return Instance{}, err
	}
	return out, nil
}

// HeartbeatSupervisedLeaseBatch renews every lease in the batch that this
// supervisor still owns, and returns those it renewed.
//
// A claim that matches no row is SKIPPED, not an error. The batch is assembled
// from a snapshot taken earlier in the tick, so by the time it executes an
// instance may legitimately have been stopped, restarted into a new generation,
// or claimed by another supervisor — all normal concurrent lifecycle activity.
// Failing the batch on the first such row meant one `vrooli scenario stop`
// racing a tick aborted renewal for the ENTIRE fleet and, because the tick
// error propagated all the way out of Run, killed the supervisor process with
// it. One scenario's ordinary state change must not be able to expire every
// other scenario's lease.
func (s *SQLiteStore) HeartbeatSupervisedLeaseBatch(ctx context.Context, claims []SupervisionClaim, ttl time.Duration) ([]Instance, error) {
	if len(claims) == 0 {
		return nil, nil
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	out := make([]Instance, 0, len(claims))
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		out = out[:0]
		for _, claim := range claims {
			if strings.TrimSpace(claim.InstanceID) == "" || strings.TrimSpace(claim.SupervisorID) == "" {
				return fmt.Errorf("heartbeat supervised lease batch: instance_id and supervisor_id are required")
			}
			result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET last_heartbeat_at = ?, heartbeat_deadline_at = ?, updated_at = ?
WHERE instance_id = ? AND generation = ? AND supervisor_id = ? AND status = ?`,
				formatTime(now), formatTime(deadline), formatTime(now),
				claim.InstanceID, claim.Generation, claim.SupervisorID, StatusRunning)
			if err != nil {
				return fmt.Errorf("heartbeat supervised lease %s: %w", claim.InstanceID, err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("inspect supervised lease heartbeat %s: %w", claim.InstanceID, err)
			}
			if affected == 0 {
				// No longer ours to renew. The caller derives the skip count
				// from len(claims)-len(renewed) and reports it.
				continue
			}
			in, err := getInstanceTx(ctx, tx, claim.InstanceID)
			if err != nil {
				return err
			}
			out = append(out, in)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
