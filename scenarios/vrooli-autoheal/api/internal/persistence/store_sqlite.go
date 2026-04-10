package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"vrooli-autoheal/internal/checks"
)

func (s *Store) saveResultSQLite(ctx context.Context, result checks.Result) error {
	detailsJSON, err := json.Marshal(result.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(ctx, query,
		result.CheckID,
		result.Status,
		result.Message,
		detailsJSON,
		result.Duration.Milliseconds(),
		result.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) getLatestResultPerCheckSQLite(ctx context.Context) ([]checks.Result, error) {
	query := `
		SELECT check_id, status, message, details, duration_ms, created_at
		FROM (
			SELECT
				check_id,
				status,
				message,
				details,
				duration_ms,
				created_at,
				ROW_NUMBER() OVER (PARTITION BY check_id ORDER BY datetime(created_at) DESC) AS rn
			FROM health_results
		)
		WHERE rn = 1
		ORDER BY check_id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []checks.Result
	for rows.Next() {
		var r checks.Result
		var detailsJSON []byte
		var durationMs int64
		var createdRaw any

		if err := rows.Scan(&r.CheckID, &r.Status, &r.Message, &detailsJSON, &durationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}

		r.Timestamp = ts
		r.Duration = time.Duration(durationMs) * time.Millisecond
		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &r.Details); err != nil {
				return nil, fmt.Errorf("unmarshal details failed: %w", err)
			}
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) getRecentResultsSQLite(ctx context.Context, checkID string, limit int) ([]checks.Result, error) {
	query := `
		SELECT check_id, status, message, details, duration_ms, created_at
		FROM health_results
		WHERE check_id = ?
		ORDER BY datetime(created_at) DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, checkID, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var results []checks.Result
	for rows.Next() {
		var r checks.Result
		var detailsJSON []byte
		var durationMs int64
		var createdRaw any

		if err := rows.Scan(&r.CheckID, &r.Status, &r.Message, &detailsJSON, &durationMs, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		r.Timestamp = ts
		_ = durationMs
		r.Duration = checks.Result{}.Duration

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &r.Details); err != nil {
				return nil, fmt.Errorf("unmarshal details failed: %w", err)
			}
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

func (s *Store) cleanupOldResultsSQLite(ctx context.Context, retentionHours int) (int64, error) {
	query := `
		DELETE FROM health_results
		WHERE datetime(created_at) < datetime('now', ?)
	`
	modifier := fmt.Sprintf("-%d hours", retentionHours)
	result, err := s.db.ExecContext(ctx, query, modifier)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) getTimelineEventsSQLite(ctx context.Context, limit int) ([]TimelineEvent, error) {
	query := `
		SELECT check_id, status, message, details, created_at
		FROM health_results
		ORDER BY datetime(created_at) DESC
		LIMIT ?
	`
	rows, err := s.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var events []TimelineEvent
	for rows.Next() {
		var e TimelineEvent
		var detailsJSON []byte
		var createdRaw any

		if err := rows.Scan(&e.CheckID, &e.Status, &e.Message, &detailsJSON, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse created_at: %w", err)
		}
		e.Timestamp = ts.UTC().Format(time.RFC3339)

		if len(detailsJSON) > 0 {
			if err := json.Unmarshal(detailsJSON, &e.Details); err != nil {
				return nil, fmt.Errorf("unmarshal timeline details failed: %w", err)
			}
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

func (s *Store) getUptimeStatsSQLite(ctx context.Context, windowHours int) (*UptimeStats, error) {
	if windowHours <= 0 {
		windowHours = 24
	}

	query := `
		SELECT
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END), 0) as ok_count,
			COALESCE(SUM(CASE WHEN status = 'warning' THEN 1 ELSE 0 END), 0) as warning_count,
			COALESCE(SUM(CASE WHEN status = 'critical' THEN 1 ELSE 0 END), 0) as critical_count
		FROM health_results
		WHERE datetime(created_at) >= datetime('now', ?)
	`
	modifier := fmt.Sprintf("-%d hours", windowHours)

	var stats UptimeStats
	err := s.db.QueryRowContext(ctx, query, modifier).Scan(
		&stats.TotalEvents,
		&stats.OkEvents,
		&stats.WarningEvents,
		&stats.CriticalEvents,
	)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	stats.WindowHours = windowHours
	if stats.TotalEvents > 0 {
		stats.UptimePercentage = float64(stats.OkEvents) / float64(stats.TotalEvents) * 100
	} else {
		stats.UptimePercentage = 100
	}

	return &stats, nil
}

func (s *Store) getUptimeHistorySQLite(ctx context.Context, windowHours, bucketCount int) (*UptimeHistory, error) {
	if bucketCount <= 0 {
		bucketCount = 24
	}
	if windowHours <= 0 {
		windowHours = 24
	}

	bucketDuration := time.Duration(windowHours) * time.Hour / time.Duration(bucketCount)
	now := time.Now().UTC()
	start := now.Add(-time.Duration(windowHours) * time.Hour)

	checksQuery := `
		SELECT DISTINCT check_id
		FROM health_results
		WHERE datetime(created_at) >= datetime('now', ?)
	`
	checksModifier := fmt.Sprintf("-%d hours", windowHours+24)
	checkRows, err := s.db.QueryContext(ctx, checksQuery, checksModifier)
	if err != nil {
		return nil, fmt.Errorf("query checks failed: %w", err)
	}

	var checkIDs []string
	for checkRows.Next() {
		var checkID string
		if err := checkRows.Scan(&checkID); err != nil {
			checkRows.Close()
			return nil, fmt.Errorf("scan check_id failed: %w", err)
		}
		checkIDs = append(checkIDs, checkID)
	}
	if err := checkRows.Err(); err != nil {
		checkRows.Close()
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if err := checkRows.Close(); err != nil {
		return nil, fmt.Errorf("close check rows failed: %w", err)
	}

	statusStmt, err := s.db.PrepareContext(ctx, `
		SELECT status
		FROM health_results
		WHERE check_id = ? AND datetime(created_at) <= datetime(?)
		ORDER BY datetime(created_at) DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare status query failed: %w", err)
	}
	defer statusStmt.Close()

	var buckets []UptimeHistoryBucket
	var totalSnapshots, totalOk, totalWarning, totalCritical int
	for i := 0; i < bucketCount; i++ {
		bucketTime := start.Add(time.Duration(i) * bucketDuration)
		if bucketTime.After(now) {
			bucketTime = now
		}

		bucket := UptimeHistoryBucket{Timestamp: bucketTime}
		for _, checkID := range checkIDs {
			var status string
			err := statusStmt.QueryRowContext(ctx, checkID, bucketTime.Format(time.RFC3339Nano)).Scan(&status)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("query status failed: %w", err)
			}

			bucket.Total++
			switch status {
			case "ok":
				bucket.Ok++
			case "warning":
				bucket.Warning++
			case "critical":
				bucket.Critical++
			}
		}

		buckets = append(buckets, bucket)
		totalSnapshots += bucket.Total
		totalOk += bucket.Ok
		totalWarning += bucket.Warning
		totalCritical += bucket.Critical
	}

	uptimePercent := 100.0
	if totalSnapshots > 0 {
		uptimePercent = float64(totalOk) / float64(totalSnapshots) * 100
	}

	return &UptimeHistory{
		Buckets: buckets,
		Overall: UptimeStats{
			TotalEvents:      totalSnapshots,
			OkEvents:         totalOk,
			WarningEvents:    totalWarning,
			CriticalEvents:   totalCritical,
			UptimePercentage: uptimePercent,
			WindowHours:      windowHours,
		},
		WindowHours: windowHours,
		BucketCount: bucketCount,
	}, nil
}

func (s *Store) getCheckTrendsSQLite(ctx context.Context, windowHours int) (*CheckTrendsResponse, error) {
	if windowHours <= 0 {
		windowHours = 24
	}

	query := `
		SELECT
			check_id,
			COUNT(*) as total,
			COUNT(CASE WHEN status = 'ok' THEN 1 END) as ok_count,
			COUNT(CASE WHEN status = 'warning' THEN 1 END) as warning_count,
			COUNT(CASE WHEN status = 'critical' THEN 1 END) as critical_count,
			MAX(created_at) as last_checked
		FROM health_results
		WHERE datetime(created_at) >= datetime('now', ?)
		GROUP BY check_id
		ORDER BY
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE COUNT(CASE WHEN status = 'ok' THEN 1 END) * 100.0 / COUNT(*) END ASC,
			check_id ASC
	`
	modifier := fmt.Sprintf("-%d hours", windowHours)
	rows, err := s.db.QueryContext(ctx, query, modifier)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	var trends []CheckTrend
	for rows.Next() {
		var t CheckTrend
		var lastCheckedRaw any

		if err := rows.Scan(&t.CheckID, &t.Total, &t.Ok, &t.Warning, &t.Critical, &lastCheckedRaw); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		lastChecked, err := parseDBTime(lastCheckedRaw)
		if err != nil {
			return nil, fmt.Errorf("parse last_checked failed: %w", err)
		}
		t.LastChecked = lastChecked.UTC().Format(time.RFC3339)

		if t.Total > 0 {
			t.UptimePercent = float64(t.Ok) / float64(t.Total) * 100
		} else {
			t.UptimePercent = 100
		}

		trends = append(trends, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("rows error: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close trends rows failed: %w", err)
	}

	currentStmt, err := s.db.PrepareContext(ctx, `
		SELECT status
		FROM health_results
		WHERE check_id = ? AND datetime(created_at) >= datetime('now', ?)
		ORDER BY datetime(created_at) DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare current status query failed: %w", err)
	}
	defer currentStmt.Close()

	recentStmt, err := s.db.PrepareContext(ctx, `
		SELECT status
		FROM health_results
		WHERE check_id = ? AND datetime(created_at) >= datetime('now', ?)
		ORDER BY datetime(created_at) DESC
		LIMIT 12
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare recent statuses query failed: %w", err)
	}
	defer recentStmt.Close()

	for i := range trends {
		if err := currentStmt.QueryRowContext(ctx, trends[i].CheckID, modifier).Scan(&trends[i].CurrentStatus); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				trends[i].CurrentStatus = "ok"
			} else {
				return nil, fmt.Errorf("current status query failed: %w", err)
			}
		}

		recentRows, err := recentStmt.QueryContext(ctx, trends[i].CheckID, modifier)
		if err != nil {
			return nil, fmt.Errorf("recent statuses query failed: %w", err)
		}
		for recentRows.Next() {
			var status string
			if err := recentRows.Scan(&status); err != nil {
				recentRows.Close()
				return nil, fmt.Errorf("scan recent status failed: %w", err)
			}
			trends[i].RecentStatuses = append(trends[i].RecentStatuses, status)
		}
		if err := recentRows.Err(); err != nil {
			recentRows.Close()
			return nil, fmt.Errorf("recent rows error: %w", err)
		}
		if err := recentRows.Close(); err != nil {
			return nil, fmt.Errorf("close recent rows failed: %w", err)
		}
	}

	return &CheckTrendsResponse{
		Trends:      trends,
		WindowHours: windowHours,
		TotalChecks: len(trends),
	}, nil
}

func (s *Store) getIncidentsSQLite(ctx context.Context, windowHours, limit int) (*IncidentsResponse, error) {
	if windowHours <= 0 {
		windowHours = 24
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		WITH ordered_results AS (
			SELECT
				check_id,
				status,
				message,
				created_at,
				LAG(status) OVER (PARTITION BY check_id ORDER BY datetime(created_at)) as prev_status
			FROM health_results
			WHERE datetime(created_at) >= datetime('now', ?)
		)
		SELECT check_id, created_at, prev_status, status, message
		FROM ordered_results
		WHERE prev_status IS NOT NULL AND prev_status != status
		ORDER BY datetime(created_at) DESC
		LIMIT ?
	`
	modifier := fmt.Sprintf("-%d hours", windowHours)
	rows, err := s.db.QueryContext(ctx, query, modifier, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var i Incident
		var createdRaw any
		if err := rows.Scan(&i.CheckID, &createdRaw, &i.FromStatus, &i.ToStatus, &i.Message); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		i.Timestamp = ts.UTC().Format(time.RFC3339)
		incidents = append(incidents, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &IncidentsResponse{
		Incidents:   incidents,
		WindowHours: windowHours,
		Total:       len(incidents),
	}, nil
}

func (s *Store) saveActionLogSQLite(ctx context.Context, checkID, actionID string, success bool, message, output, errMsg string, durationMs int64) error {
	query := `
		INSERT INTO action_logs (check_id, action_id, success, message, output, error, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	_, err := s.db.ExecContext(ctx, query,
		checkID,
		actionID,
		success,
		message,
		output,
		errMsg,
		durationMs,
		time.Now().UTC().Format(time.RFC3339Nano),
	)
	return err
}

func (s *Store) getActionLogsSQLite(ctx context.Context, limit int) (*ActionLogsResponse, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	query := `
		SELECT id, check_id, action_id, success, message, COALESCE(output, ''), COALESCE(error, ''), duration_ms, created_at
		FROM action_logs
		ORDER BY datetime(created_at) DESC
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
		if err := rows.Scan(&l.ID, &l.CheckID, &l.ActionID, &l.Success, &l.Message, &l.Output, &l.Error, &l.DurationMs, &createdRaw); err != nil {
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
		SELECT id, check_id, action_id, success, message, COALESCE(output, ''), COALESCE(error, ''), duration_ms, created_at
		FROM action_logs
		WHERE check_id = ?
		ORDER BY datetime(created_at) DESC
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
		if err := rows.Scan(&l.ID, &l.CheckID, &l.ActionID, &l.Success, &l.Message, &l.Output, &l.Error, &l.DurationMs, &createdRaw); err != nil {
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

	_, err := s.db.ExecContext(ctx, query,
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

func parseDBTime(raw any) (time.Time, error) {
	switch v := raw.(type) {
	case time.Time:
		return v, nil
	case string:
		return parseTimeString(v)
	case []byte:
		return parseTimeString(string(v))
	default:
		return time.Time{}, fmt.Errorf("unsupported timestamp type %T", raw)
	}
}

func parseTimeString(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, layout := range layouts {
		if ts, err := time.Parse(layout, s); err == nil {
			return ts, nil
		}
	}
	return time.Time{}, fmt.Errorf("failed to parse time %q", s)
}

func nullableTimeToDBText(ts time.Time) interface{} {
	if ts.IsZero() {
		return nil
	}
	return ts.UTC().Format(time.RFC3339Nano)
}

func parseNullableDBTime(raw any) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	switch v := raw.(type) {
	case time.Time:
		if v.IsZero() {
			return time.Time{}, false
		}
		return v, true
	case string:
		if v == "" {
			return time.Time{}, false
		}
		ts, err := parseTimeString(v)
		if err != nil {
			return time.Time{}, false
		}
		return ts, true
	case []byte:
		if len(v) == 0 {
			return time.Time{}, false
		}
		ts, err := parseTimeString(string(v))
		if err != nil {
			return time.Time{}, false
		}
		return ts, true
	case sql.NullTime:
		if !v.Valid || v.Time.IsZero() {
			return time.Time{}, false
		}
		return v.Time, true
	default:
		return time.Time{}, false
	}
}
