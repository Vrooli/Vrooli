package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

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
SET supervisor_id = ?, supervised_at = ?, owner_kind = ?, updated_at = ?
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

func (s *SQLiteStore) HeartbeatSupervisedLeaseBatch(ctx context.Context, claims []SupervisionClaim, ttl time.Duration) ([]Instance, error) {
	if len(claims) == 0 {
		return nil, nil
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	out := make([]Instance, 0, len(claims))
	err := s.withTx(ctx, func(tx *sql.Tx) error {
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
				return staleGenerationOrNotFound(ctx, tx, claim.InstanceID)
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
