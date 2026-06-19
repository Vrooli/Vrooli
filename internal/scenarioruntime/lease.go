package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const staleLeaseStopReason = "heartbeat deadline elapsed"

func NormalizeHeartbeatTTL(ttl time.Duration) time.Duration {
	if ttl <= 0 {
		return DefaultHeartbeatTTL
	}
	return ttl
}

func (s *SQLiteStore) CreateLease(ctx context.Context, in Instance, ttl time.Duration) (Instance, error) {
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	in.LastHeartbeatAt = &now
	in.HeartbeatDeadlineAt = &deadline
	return s.CreateInstance(ctx, in)
}

func (s *SQLiteStore) HeartbeatLease(ctx context.Context, instanceID string, generation int64, ttl time.Duration) (Instance, error) {
	if strings.TrimSpace(instanceID) == "" {
		return Instance{}, fmt.Errorf("heartbeat lease: instance_id is required")
	}
	now := s.now()
	deadline := now.Add(NormalizeHeartbeatTTL(ttl))
	var out Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET last_heartbeat_at = ?, heartbeat_deadline_at = ?, updated_at = ?
WHERE instance_id = ? AND generation = ? AND status IN (?, ?)`,
			formatTime(now), formatTime(deadline), formatTime(now), instanceID, generation, StatusStarting, StatusRunning)
		if err != nil {
			return fmt.Errorf("heartbeat runtime lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime lease heartbeat: %w", err)
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

func (s *SQLiteStore) ExpireStaleLeases(ctx context.Context, at time.Time) ([]Instance, error) {
	now := s.now()
	var expired []Instance
	err := s.withTx(ctx, func(tx *sql.Tx) error {
		rows, err := tx.QueryContext(ctx, instanceSelectSQL+`
WHERE status IN (?, ?) AND heartbeat_deadline_at IS NOT NULL AND heartbeat_deadline_at <= ?
ORDER BY scenario ASC, generation DESC`,
			StatusStarting, StatusRunning, formatTime(at.UTC()))
		if err != nil {
			return fmt.Errorf("list stale runtime leases: %w", err)
		}
		expired, err = scanInstances(rows)
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		if err != nil {
			return err
		}
		for _, in := range expired {
			result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, updated_at = ?, stop_reason = ?
WHERE instance_id = ? AND generation = ? AND status IN (?, ?)`,
				StatusExpired, formatTime(now), staleLeaseStopReason,
				in.InstanceID, in.Generation, StatusStarting, StatusRunning)
			if err != nil {
				return fmt.Errorf("expire runtime lease %s: %w", in.InstanceID, err)
			}
			affected, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("inspect runtime lease expiry %s: %w", in.InstanceID, err)
			}
			if affected == 0 {
				return ErrStaleGeneration
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for i := range expired {
		expired[i].Status = StatusExpired
		expired[i].UpdatedAt = now
		expired[i].StopReason = staleLeaseStopReason
	}
	return expired, nil
}

func (s *SQLiteStore) StopLease(ctx context.Context, instanceID string, generation int64, reason string) (Instance, error) {
	if strings.TrimSpace(instanceID) == "" {
		return Instance{}, fmt.Errorf("stop lease: instance_id is required")
	}
	now := s.now()
	var out Instance
	err := s.withRetryableTx(ctx, func(tx *sql.Tx) error {
		result, err := tx.ExecContext(ctx, `
UPDATE runtime_instances
SET status = ?, updated_at = ?, stopped_at = ?, stop_reason = ?
WHERE instance_id = ? AND generation = ?`,
			StatusStopped, formatTime(now), formatTime(now), reason, instanceID, generation)
		if err != nil {
			return fmt.Errorf("stop runtime lease: %w", err)
		}
		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("inspect runtime lease stop: %w", err)
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

func staleGenerationOrNotFound(ctx context.Context, tx *sql.Tx, instanceID string) error {
	if _, err := getInstanceTx(ctx, tx, instanceID); err != nil {
		return err
	}
	return ErrStaleGeneration
}
