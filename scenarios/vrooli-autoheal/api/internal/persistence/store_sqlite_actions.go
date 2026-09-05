package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func supervisedAvailability(result checks.Result) (available, observed bool) {
	intent, _ := result.Details["supervisionIntent"].(string)
	if intent != "must_start" && intent != "try_start" {
		return false, false
	}
	if explicit, ok := result.Details["available"].(bool); ok {
		return explicit, true
	}
	if serving, ok := result.Details["serving"].(bool); ok {
		return serving, true
	}
	if running, ok := result.Details["running"].(bool); ok && !running {
		return false, true
	}
	if scenarioStatus, ok := result.Details["scenarioStatus"].(string); ok && scenarioStatus != "" {
		if !strings.EqualFold(scenarioStatus, "running") {
			return false, true
		}
		if health, ok := result.Details["healthStatus"].(string); ok && strings.EqualFold(health, "unhealthy") {
			return false, true
		}
		return true, true
	}
	return false, false
}

func (s *Store) observeSupervisedAvailabilitySQLite(ctx context.Context, result checks.Result) error {
	available, observed := supervisedAvailability(result)
	if !observed {
		return nil
	}
	memberID := strings.TrimSpace(result.CheckID)
	if memberID == "" {
		return fmt.Errorf("supervised availability observation requires check id")
	}
	observedAt := result.Timestamp.UTC()
	if observedAt.IsZero() {
		observedAt = time.Now().UTC()
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if !available {
		_, err = tx.ExecContext(ctx, `
			INSERT OR IGNORE INTO outage_records (member_id, cause, opened_at)
			VALUES (?, ?, ?)
		`, memberID, truncatePersistedText(result.Message, maxActionLogTextBytes), observedAt.Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("open outage for %s: %w", memberID, err)
		}
		return tx.Commit()
	}

	var id int64
	var openedRaw any
	err = tx.QueryRowContext(ctx, `
		SELECT id, opened_at FROM outage_records
		WHERE member_id = ? AND closed_at IS NULL
	`, memberID).Scan(&id, &openedRaw)
	if err == sql.ErrNoRows {
		return tx.Commit()
	}
	if err != nil {
		return fmt.Errorf("find open outage for %s: %w", memberID, err)
	}
	openedAt, err := parseDBTime(openedRaw)
	if err != nil {
		return fmt.Errorf("parse outage opening for %s: %w", memberID, err)
	}
	closedAt := observedAt
	if !closedAt.After(openedAt) {
		closedAt = openedAt.Add(time.Nanosecond)
	}
	if _, err = tx.ExecContext(ctx, `UPDATE outage_records SET closed_at = ? WHERE id = ? AND closed_at IS NULL`, closedAt.Format(time.RFC3339Nano), id); err != nil {
		return fmt.Errorf("close outage for %s: %w", memberID, err)
	}
	return tx.Commit()
}

func (s *Store) getOutageSummarySQLite(ctx context.Context, memberID string, from, to time.Time) (*OutageSummary, error) {
	memberID = strings.TrimSpace(memberID)
	from, to = from.UTC(), to.UTC()
	if memberID == "" || from.IsZero() || to.IsZero() || !to.After(from) {
		return nil, fmt.Errorf("outage summary requires member id and an increasing time window")
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT opened_at, closed_at
		FROM outage_records
		WHERE member_id = ? AND opened_at < ? AND (closed_at IS NULL OR closed_at > ?)
		ORDER BY opened_at
	`, memberID, to.Format(time.RFC3339Nano), from.Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("query outage summary: %w", err)
	}
	defer rows.Close()

	summary := &OutageSummary{MemberID: memberID, WindowStart: from, WindowEnd: to}
	for rows.Next() {
		var openedRaw, closedRaw any
		if err := rows.Scan(&openedRaw, &closedRaw); err != nil {
			return nil, fmt.Errorf("scan outage summary: %w", err)
		}
		openedAt, err := parseDBTime(openedRaw)
		if err != nil {
			return nil, fmt.Errorf("parse outage opening: %w", err)
		}
		closedAt, closed := parseNullableDBTime(closedRaw)
		if !closed {
			closedAt = to
			summary.OpenOutageCount++
		}
		start := openedAt
		if start.Before(from) {
			start = from
		}
		end := closedAt
		if end.After(to) {
			end = to
		}
		if end.After(start) {
			summary.TotalUnavailableSeconds += end.Sub(start).Seconds()
			summary.DistinctOutageCount++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate outage summary: %w", err)
	}
	return summary, nil
}

func (s *Store) listOutageRecordsSQLite(ctx context.Context, memberID string, limit int) ([]OutageRecord, error) {
	memberID = strings.TrimSpace(memberID)
	if memberID == "" {
		return nil, fmt.Errorf("outage records require member id")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, member_id, cause, opened_at, closed_at
		FROM outage_records WHERE member_id = ? ORDER BY opened_at DESC LIMIT ?
	`, memberID, limit)
	if err != nil {
		return nil, fmt.Errorf("query outage records: %w", err)
	}
	defer rows.Close()
	var records []OutageRecord
	for rows.Next() {
		var record OutageRecord
		var openedRaw, closedRaw any
		if err := rows.Scan(&record.ID, &record.MemberID, &record.Cause, &openedRaw, &closedRaw); err != nil {
			return nil, fmt.Errorf("scan outage record: %w", err)
		}
		record.OpenedAt, err = parseDBTime(openedRaw)
		if err != nil {
			return nil, err
		}
		if closedAt, ok := parseNullableDBTime(closedRaw); ok {
			record.ClosedAt = &closedAt
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

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

const saveHealTrackerQuery = `
		INSERT INTO heal_trackers (check_id, last_attempt, last_success, consecutive_failures, total_attempts, total_successes, cooldown_until, suspended_at, suspension_reason, disposition, disposition_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(check_id) DO UPDATE SET
			last_attempt = excluded.last_attempt,
			last_success = excluded.last_success,
			consecutive_failures = excluded.consecutive_failures,
			total_attempts = excluded.total_attempts,
			total_successes = excluded.total_successes,
			cooldown_until = excluded.cooldown_until,
			suspended_at = excluded.suspended_at,
			suspension_reason = excluded.suspension_reason,
			disposition = excluded.disposition,
			disposition_at = excluded.disposition_at,
			updated_at = excluded.updated_at
	`

func healTrackerArgs(checkID string, tracker *checks.HealTracker) []any {
	return []any{
		checkID,
		nullableTimeToDBText(tracker.LastAttempt),
		nullableTimeToDBText(tracker.LastSuccess),
		tracker.ConsecutiveFailures,
		tracker.TotalAttempts,
		tracker.TotalSuccesses,
		nullableTimeToDBText(tracker.CooldownUntil),
		nullableTimeToDBText(tracker.SuspendedAt),
		tracker.SuspensionReason,
		tracker.Disposition,
		nullableTimeToDBText(tracker.DispositionAt),
		time.Now().UTC().Format(time.RFC3339Nano),
	}
}

func (s *Store) saveHealTrackerSQLite(ctx context.Context, checkID string, tracker *checks.HealTracker) error {
	_, err := s.db.ExecContext(ctx, saveHealTrackerQuery, healTrackerArgs(checkID, tracker)...)
	return err
}

func (s *Store) saveHealTrackersSQLite(ctx context.Context, trackers map[string]checks.HealTracker) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for checkID, tracker := range trackers {
		copy := tracker
		if _, err := tx.ExecContext(ctx, saveHealTrackerQuery, healTrackerArgs(checkID, &copy)...); err != nil {
			return fmt.Errorf("save heal tracker %s: %w", checkID, err)
		}
	}
	return tx.Commit()
}

func (s *Store) getAllHealTrackersSQLite(ctx context.Context) (map[string]*checks.HealTracker, error) {
	query := `
		SELECT check_id, last_attempt, last_success, consecutive_failures, total_attempts, total_successes, cooldown_until, suspended_at, suspension_reason, disposition, disposition_at
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
		var suspendedAtRaw any
		var dispositionAtRaw any
		var tracker checks.HealTracker

		if err := rows.Scan(
			&checkID,
			&lastAttemptRaw,
			&lastSuccessRaw,
			&tracker.ConsecutiveFailures,
			&tracker.TotalAttempts,
			&tracker.TotalSuccesses,
			&cooldownUntilRaw,
			&suspendedAtRaw,
			&tracker.SuspensionReason,
			&tracker.Disposition,
			&dispositionAtRaw,
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
		if suspendedAt, ok := parseNullableDBTime(suspendedAtRaw); ok {
			tracker.SuspendedAt = suspendedAt
		}
		if dispositionAt, ok := parseNullableDBTime(dispositionAtRaw); ok {
			tracker.DispositionAt = dispositionAt
		}

		trackers[checkID] = &tracker
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return trackers, nil
}

func (s *Store) ensureHealTrackerColumns(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(heal_trackers)`)
	if err != nil {
		return err
	}
	defer rows.Close()
	existing := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return err
		}
		existing[name] = true
	}
	for _, column := range []struct{ name, declaration string }{
		{"suspended_at", "TEXT"},
		{"suspension_reason", "TEXT NOT NULL DEFAULT ''"},
		{"disposition", "TEXT NOT NULL DEFAULT ''"},
		{"disposition_at", "TEXT"},
	} {
		if existing[column.name] {
			continue
		}
		if _, err := s.db.ExecContext(ctx, `ALTER TABLE heal_trackers ADD COLUMN `+column.name+` `+column.declaration); err != nil {
			return fmt.Errorf("add heal_trackers.%s: %w", column.name, err)
		}
	}
	return nil
}

func (s *Store) deleteHealTrackerSQLite(ctx context.Context, checkID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM heal_trackers WHERE check_id = ?`, checkID)
	return err
}
