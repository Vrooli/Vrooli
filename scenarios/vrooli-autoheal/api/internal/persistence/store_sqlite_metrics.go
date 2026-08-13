package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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
		WHERE created_at >= ? AND status <> 'not-applicable'
	`
	cutoff := rfc3339NanoCutoff(time.Now(), windowHours)

	var stats UptimeStats
	err := s.db.QueryRowContext(ctx, query, cutoff).Scan(
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

	startUnix := start.Unix()
	bucketSeconds := int64(bucketDuration / time.Second)
	if bucketSeconds <= 0 {
		bucketSeconds = 1
	}

	// Single aggregation: count health-result events per (bucket, status). The
	// previous implementation issued bucketCount × distinctCheckIDs prepared-
	// statement round-trips to read "status as of bucket boundary"; that
	// scaled linearly with bucket × check count and was the dominant source
	// of dashboard latency on populated databases. Counting events per slice
	// is the natural shape for the stacked-area trend chart, which normalizes
	// per render anyway.
	query := `
		SELECT
			CAST((CAST(strftime('%s', created_at) AS INTEGER) - ?) / ? AS INTEGER) AS bucket,
			status,
			COUNT(*) AS n
		FROM health_results
		WHERE created_at >= ? AND created_at <= ? AND status <> 'not-applicable'
		GROUP BY bucket, status
	`
	startStr := start.Format(time.RFC3339Nano)
	endStr := now.Format(time.RFC3339Nano)

	rows, err := s.db.QueryContext(ctx, query, startUnix, bucketSeconds, startStr, endStr)
	if err != nil {
		return nil, fmt.Errorf("query bucket aggregation failed: %w", err)
	}
	defer rows.Close()

	buckets := make([]UptimeHistoryBucket, bucketCount)
	for i := 0; i < bucketCount; i++ {
		bucketTime := start.Add(time.Duration(i) * bucketDuration)
		if bucketTime.After(now) {
			bucketTime = now
		}
		buckets[i] = UptimeHistoryBucket{Timestamp: bucketTime}
	}

	var totalSnapshots, totalOk, totalWarning, totalCritical int
	for rows.Next() {
		var bucket int
		var status string
		var n int
		if err := rows.Scan(&bucket, &status, &n); err != nil {
			return nil, fmt.Errorf("scan bucket row failed: %w", err)
		}
		if bucket < 0 {
			bucket = 0
		}
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		buckets[bucket].Total += n
		switch status {
		case "ok":
			buckets[bucket].Ok += n
			totalOk += n
		case "warning":
			buckets[bucket].Warning += n
			totalWarning += n
		case "critical":
			buckets[bucket].Critical += n
			totalCritical += n
		}
		totalSnapshots += n
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
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
		WHERE created_at >= ? AND status <> 'not-applicable'
		GROUP BY check_id
		ORDER BY
			CASE WHEN COUNT(*) = 0 THEN 100 ELSE COUNT(CASE WHEN status = 'ok' THEN 1 END) * 100.0 / COUNT(*) END ASC,
			check_id ASC
	`
	cutoff := rfc3339NanoCutoff(time.Now(), windowHours)
	rows, err := s.db.QueryContext(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

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
		WHERE check_id = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 1
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare current status query failed: %w", err)
	}
	defer currentStmt.Close()

	recentStmt, err := s.db.PrepareContext(ctx, `
		SELECT status
		FROM health_results
		WHERE check_id = ? AND created_at >= ?
		ORDER BY created_at DESC
		LIMIT 12
	`)
	if err != nil {
		return nil, fmt.Errorf("prepare recent statuses query failed: %w", err)
	}
	defer recentStmt.Close()

	for i := range trends {
		if err := currentStmt.QueryRowContext(ctx, trends[i].CheckID, cutoff).Scan(&trends[i].CurrentStatus); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				trends[i].CurrentStatus = "ok"
			} else {
				return nil, fmt.Errorf("current status query failed: %w", err)
			}
		}

		recentRows, err := recentStmt.QueryContext(ctx, trends[i].CheckID, cutoff)
		if err != nil {
			return nil, fmt.Errorf("recent statuses query failed: %w", err)
		}
		defer recentRows.Close()
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

func (s *Store) getTransitionsSQLite(ctx context.Context, windowHours, limit int) (*TransitionsResponse, error) {
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
				LAG(status) OVER (PARTITION BY check_id ORDER BY created_at) as prev_status
			FROM health_results
			WHERE created_at >= ?
		)
		SELECT check_id, created_at, prev_status, status, message
		FROM ordered_results
		WHERE prev_status IS NOT NULL AND prev_status != status
		ORDER BY created_at DESC
		LIMIT ?
	`
	cutoff := rfc3339NanoCutoff(time.Now(), windowHours)
	rows, err := s.db.QueryContext(ctx, query, cutoff, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var transitions []Transition
	for rows.Next() {
		var i Transition
		var createdRaw any
		if err := rows.Scan(&i.CheckID, &createdRaw, &i.FromStatus, &i.ToStatus, &i.Message); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		ts, err := parseDBTime(createdRaw)
		if err != nil {
			return nil, fmt.Errorf("parse timestamp failed: %w", err)
		}
		i.Timestamp = ts.UTC().Format(time.RFC3339)
		transitions = append(transitions, i)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &TransitionsResponse{
		Transitions: transitions,
		WindowHours: windowHours,
		Total:       len(transitions),
	}, nil
}
