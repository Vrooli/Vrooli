package persistence

import (
	"context"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func (s *Store) saveActionLogSQLite(ctx context.Context, checkID, actionID string, success, timedOut bool, message, output, errMsg string, durationMs int64) error {
	query := `
		INSERT INTO action_logs (check_id, action_id, success, timed_out, message, output, error, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(
		ctx, query,
		checkID,
		actionID,
		success,
		timedOut,
		truncatePersistedText(message, maxActionLogTextBytes),
		truncatePersistedText(output, maxActionLogTextBytes),
		truncatePersistedText(errMsg, maxActionLogTextBytes),
		durationMs,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func truncatePersistedText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	// The marker makes truncation explicit in incident evidence and reserves
	// space inside the hard persistence budget.
	marker := fmt.Sprintf("\n[truncated: original_bytes=%d]", len(value))
	if len(marker) >= limit {
		return marker[:limit]
	}
	return value[:limit-len(marker)] + marker
}

func (s *Store) getActionLogsSQLite(ctx context.Context, limit int) (*ActionLogsResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT id, check_id, action_id, success, COALESCE(timed_out, 0), message, COALESCE(output, ''), COALESCE(error, ''), duration_ms, created_at
		FROM action_logs
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []ActionLog
	for rows.Next() {
		var l ActionLog
		var createdRaw any
		if err := rows.Scan(&l.ID, &l.CheckID, &l.ActionID, &l.Success, &l.TimedOut, &l.Message, &l.Output, &l.Error, &l.DurationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		l.Timestamp = ts.UTC().Format(time.RFC3339)
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &ActionLogsResponse{Logs: logs, Total: len(logs)}, nil
}

func (s *Store) getActionLogsForCheckSQLite(ctx context.Context, checkID string, limit int) (*ActionLogsResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 20
	}

	query := `
		SELECT id, check_id, action_id, success, COALESCE(timed_out, 0), message, COALESCE(output, ''), COALESCE(error, ''), duration_ms, created_at
		FROM action_logs
		WHERE check_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, checkID, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var logs []ActionLog
	for rows.Next() {
		var l ActionLog
		var createdRaw any
		if err := rows.Scan(&l.ID, &l.CheckID, &l.ActionID, &l.Success, &l.TimedOut, &l.Message, &l.Output, &l.Error, &l.DurationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		l.Timestamp = ts.UTC().Format(time.RFC3339)
		logs = append(logs, l)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &ActionLogsResponse{Logs: logs, Total: len(logs)}, nil
}

func (s *Store) saveHealTrackerSQLite(ctx context.Context, checkID string, tracker *checks.HealTracker) error {
	query := `
		INSERT INTO heal_trackers (check_id, last_attempt, last_success, consecutive_failures, total_attempts, total_successes, cooldown_until, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(check_id) DO UPDATE SET
			last_attempt = excluded.last_attempt,
			last_success = excluded.last_success,
			consecutive_failures = excluded.consecutive_failures,
			total_attempts = excluded.total_attempts,
			total_successes = excluded.total_successes,
			cooldown_until = excluded.cooldown_until,
			updated_at = excluded.updated_at
	`

	_, err := s.db.ExecContext(
		ctx, query,
		checkID,
		nullableTimeToDBText(tracker.LastAttempt),
		nullableTimeToDBText(tracker.LastSuccess),
		tracker.ConsecutiveFailures,
		tracker.TotalAttempts,
		tracker.TotalSuccesses,
		nullableTimeToDBText(tracker.CooldownUntil),
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) getAllHealTrackersSQLite(ctx context.Context) (map[string]*checks.HealTracker, error) {
	query := `
		SELECT check_id, last_attempt, last_success, consecutive_failures, total_attempts, total_successes, cooldown_until
		FROM heal_trackers
	`
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	trackers := make(map[string]*checks.HealTracker)
	for rows.Next() {
		var checkID string
		var lastAttemptRaw any
		var lastSuccessRaw any
		var cooldownUntilRaw any
		var tracker checks.HealTracker

		if err := rows.Scan(
			&checkID,
			&lastAttemptRaw,
			&lastSuccessRaw,
			&tracker.ConsecutiveFailures,
			&tracker.TotalAttempts,
			&tracker.TotalSuccesses,
			&cooldownUntilRaw,
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		lastAttempt, ok := parseNullableDBTime(lastAttemptRaw)
		if ok {
			tracker.LastAttempt = lastAttempt
		}
		lastSuccess, ok := parseNullableDBTime(lastSuccessRaw)
		if ok {
			tracker.LastSuccess = lastSuccess
		}
		cooldownUntil, ok := parseNullableDBTime(cooldownUntilRaw)
		if ok {
			tracker.CooldownUntil = cooldownUntil
		}

		trackers[checkID] = &tracker
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return trackers, nil
}

func (s *Store) deleteHealTrackerSQLite(ctx context.Context, checkID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM heal_trackers WHERE check_id = ?`, checkID)
	return err
}
