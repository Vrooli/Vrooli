// Package persistence provides database operations for health check results
// [REQ:PERSIST-STORE-001] [REQ:PERSIST-QUERY-001] [REQ:PERSIST-QUERY-002]
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/lib/pq"

	"vrooli-autoheal/internal/checks"
)

// Store handles database operations for health check data
type Store struct {
	db *sql.DB
}

// NewStore creates a new persistence store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Ping checks database connectivity
func (s *Store) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// SaveResult persists a health check result to the database
func (s *Store) SaveResult(ctx context.Context, result checks.Result) error {
	detailsJSON, err := json.Marshal(result.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}

	query := `
		INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err = s.db.ExecContext(ctx, query,
		result.CheckID,
		result.Status,
		result.Message,
		detailsJSON,
		result.Duration.Milliseconds(),
		result.Timestamp,
	)
	return err
}

// GetRecentResults retrieves recent health check results
func (s *Store) GetRecentResults(ctx context.Context, checkID string, limit int) ([]checks.Result, error) {
	query := `
		SELECT check_id, status, message, details, duration_ms, created_at
		FROM health_results
		WHERE check_id = $1
		ORDER BY created_at DESC
		LIMIT $2
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

		if err := rows.Scan(&r.CheckID, &r.Status, &r.Message, &detailsJSON, &durationMs, &r.Timestamp); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		r.Duration = checks.Result{}.Duration // Zero value, we store ms separately
		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &r.Details)
		}

		results = append(results, r)
	}

	return results, rows.Err()
}

// CleanupOldResults removes health check results older than the retention period
func (s *Store) CleanupOldResults(ctx context.Context, retentionHours int) (int64, error) {
	query := `
		DELETE FROM health_results
		WHERE created_at < NOW() - INTERVAL '1 hour' * $1
	`
	result, err := s.db.ExecContext(ctx, query, retentionHours)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// TimelineEvent represents a single event in the timeline
type TimelineEvent struct {
	CheckID   string                 `json:"checkId"`
	Status    string                 `json:"status"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp string                 `json:"timestamp"`
}

// GetTimelineEvents retrieves recent events across all checks, ordered by time
// [REQ:UI-EVENTS-001]
func (s *Store) GetTimelineEvents(ctx context.Context, limit int) ([]TimelineEvent, error) {
	query := `
		SELECT check_id, status, message, details, created_at
		FROM health_results
		ORDER BY created_at DESC
		LIMIT $1
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
		var timestamp time.Time

		if err := rows.Scan(&e.CheckID, &e.Status, &e.Message, &detailsJSON, &timestamp); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		e.Timestamp = timestamp.UTC().Format(time.RFC3339)

		if len(detailsJSON) > 0 {
			json.Unmarshal(detailsJSON, &e.Details)
		}

		events = append(events, e)
	}

	return events, rows.Err()
}

// UptimeStats represents uptime statistics over a time window
type UptimeStats struct {
	TotalEvents      int     `json:"totalEvents"`
	OkEvents         int     `json:"okEvents"`
	WarningEvents    int     `json:"warningEvents"`
	CriticalEvents   int     `json:"criticalEvents"`
	UptimePercentage float64 `json:"uptimePercentage"`
	WindowHours      int     `json:"windowHours"`
}

// GetUptimeStats calculates uptime statistics over the given time window
// [REQ:PERSIST-HISTORY-001]
func (s *Store) GetUptimeStats(ctx context.Context, windowHours int) (*UptimeStats, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END) as ok_count,
			SUM(CASE WHEN status = 'warning' THEN 1 ELSE 0 END) as warning_count,
			SUM(CASE WHEN status = 'critical' THEN 1 ELSE 0 END) as critical_count
		FROM health_results
		WHERE created_at >= NOW() - INTERVAL '1 hour' * $1
	`
	var stats UptimeStats
	err := s.db.QueryRowContext(ctx, query, windowHours).Scan(
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
		stats.UptimePercentage = 100 // No data = assume healthy
	}

	return &stats, nil
}

// UptimeHistoryBucket represents a time bucket with aggregated health status counts
type UptimeHistoryBucket struct {
	Timestamp time.Time `json:"timestamp"`
	Total     int       `json:"total"`
	Ok        int       `json:"ok"`
	Warning   int       `json:"warning"`
	Critical  int       `json:"critical"`
}

// UptimeHistory represents the full history response
type UptimeHistory struct {
	Buckets     []UptimeHistoryBucket `json:"buckets"`
	Overall     UptimeStats           `json:"overall"`
	WindowHours int                   `json:"windowHours"`
	BucketCount int                   `json:"bucketCount"`
}

// GetUptimeHistory returns time-bucketed uptime data for charting
// [REQ:PERSIST-HISTORY-001] [REQ:UI-EVENTS-001]
func (s *Store) GetUptimeHistory(ctx context.Context, windowHours, bucketCount int) (*UptimeHistory, error) {
	if bucketCount <= 0 {
		bucketCount = 24
	}
	if windowHours <= 0 {
		windowHours = 24
	}

	// Calculate bucket duration in minutes
	bucketMinutes := (windowHours * 60) / bucketCount

	// Query to bucket health results by time
	query := `
		WITH time_series AS (
			SELECT generate_series(
				date_trunc('hour', NOW()) - INTERVAL '1 hour' * $1 + INTERVAL '1 minute' * $2,
				NOW(),
				INTERVAL '1 minute' * $2
			) AS bucket_start
		),
		bucketed_results AS (
			SELECT
				ts.bucket_start,
				hr.status
			FROM time_series ts
			LEFT JOIN health_results hr ON
				hr.created_at >= ts.bucket_start AND
				hr.created_at < ts.bucket_start + INTERVAL '1 minute' * $2 AND
				hr.created_at >= NOW() - INTERVAL '1 hour' * $1
		)
		SELECT
			bucket_start,
			COUNT(status) as total,
			COUNT(CASE WHEN status = 'ok' THEN 1 END) as ok_count,
			COUNT(CASE WHEN status = 'warning' THEN 1 END) as warning_count,
			COUNT(CASE WHEN status = 'critical' THEN 1 END) as critical_count
		FROM bucketed_results
		GROUP BY bucket_start
		ORDER BY bucket_start ASC
	`

	rows, err := s.db.QueryContext(ctx, query, windowHours, bucketMinutes)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var buckets []UptimeHistoryBucket
	var totalEvents, totalOk, totalWarning, totalCritical int

	for rows.Next() {
		var b UptimeHistoryBucket
		if err := rows.Scan(&b.Timestamp, &b.Total, &b.Ok, &b.Warning, &b.Critical); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}
		buckets = append(buckets, b)

		totalEvents += b.Total
		totalOk += b.Ok
		totalWarning += b.Warning
		totalCritical += b.Critical
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	// Calculate overall uptime
	uptimePercent := 100.0
	if totalEvents > 0 {
		uptimePercent = float64(totalOk) / float64(totalEvents) * 100
	}

	return &UptimeHistory{
		Buckets: buckets,
		Overall: UptimeStats{
			TotalEvents:      totalEvents,
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

// CheckTrend represents per-check trend data
type CheckTrend struct {
	CheckID        string   `json:"checkId"`
	Total          int      `json:"total"`
	Ok             int      `json:"ok"`
	Warning        int      `json:"warning"`
	Critical       int      `json:"critical"`
	UptimePercent  float64  `json:"uptimePercent"`
	CurrentStatus  string   `json:"currentStatus"`
	RecentStatuses []string `json:"recentStatuses"`
	LastChecked    string   `json:"lastChecked"`
}

// CheckTrendsResponse contains all check trends
type CheckTrendsResponse struct {
	Trends      []CheckTrend `json:"trends"`
	WindowHours int          `json:"windowHours"`
	TotalChecks int          `json:"totalChecks"`
}

// GetCheckTrends returns per-check trend data aggregated over the time window
// [REQ:PERSIST-HISTORY-001]
func (s *Store) GetCheckTrends(ctx context.Context, windowHours int) (*CheckTrendsResponse, error) {
	if windowHours <= 0 {
		windowHours = 24
	}

	// Query to get per-check aggregated stats
	query := `
		WITH check_stats AS (
			SELECT
				check_id,
				COUNT(*) as total,
				COUNT(CASE WHEN status = 'ok' THEN 1 END) as ok_count,
				COUNT(CASE WHEN status = 'warning' THEN 1 END) as warning_count,
				COUNT(CASE WHEN status = 'critical' THEN 1 END) as critical_count,
				MAX(created_at) as last_checked
			FROM health_results
			WHERE created_at >= NOW() - INTERVAL '1 hour' * $1
			GROUP BY check_id
		),
		recent_statuses AS (
			SELECT
				check_id,
				status,
				ROW_NUMBER() OVER (PARTITION BY check_id ORDER BY created_at DESC) as rn
			FROM health_results
			WHERE created_at >= NOW() - INTERVAL '1 hour' * $1
		)
		SELECT
			cs.check_id,
			cs.total,
			cs.ok_count,
			cs.warning_count,
			cs.critical_count,
			cs.last_checked,
			(SELECT status FROM recent_statuses WHERE check_id = cs.check_id AND rn = 1) as current_status,
			ARRAY(
				SELECT status FROM recent_statuses
				WHERE check_id = cs.check_id AND rn <= 12
				ORDER BY rn
			) as recent_statuses
		FROM check_stats cs
		ORDER BY
			CASE WHEN cs.ok_count * 100.0 / NULLIF(cs.total, 0) IS NULL THEN 100
			     ELSE cs.ok_count * 100.0 / cs.total
			END ASC,
			cs.check_id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, windowHours)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var trends []CheckTrend
	for rows.Next() {
		var t CheckTrend
		var lastChecked time.Time
		var currentStatus sql.NullString
		var recentStatuses []string

		if err := rows.Scan(
			&t.CheckID, &t.Total, &t.Ok, &t.Warning, &t.Critical,
			&lastChecked, &currentStatus, pq.Array(&recentStatuses),
		); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		t.LastChecked = lastChecked.UTC().Format(time.RFC3339)
		if currentStatus.Valid {
			t.CurrentStatus = currentStatus.String
		} else {
			t.CurrentStatus = "ok"
		}
		t.RecentStatuses = recentStatuses

		if t.Total > 0 {
			t.UptimePercent = float64(t.Ok) / float64(t.Total) * 100
		} else {
			t.UptimePercent = 100
		}

		trends = append(trends, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return &CheckTrendsResponse{
		Trends:      trends,
		WindowHours: windowHours,
		TotalChecks: len(trends),
	}, nil
}

// Incident represents a status transition event
type Incident struct {
	Timestamp  string `json:"timestamp"`
	CheckID    string `json:"checkId"`
	FromStatus string `json:"fromStatus"`
	ToStatus   string `json:"toStatus"`
	Message    string `json:"message"`
}

// IncidentsResponse contains all incidents
type IncidentsResponse struct {
	Incidents   []Incident `json:"incidents"`
	WindowHours int        `json:"windowHours"`
	Total       int        `json:"total"`
}

// GetIncidents returns status transition events over the time window
// [REQ:PERSIST-HISTORY-001]
func (s *Store) GetIncidents(ctx context.Context, windowHours, limit int) (*IncidentsResponse, error) {
	if windowHours <= 0 {
		windowHours = 24
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}

	// Query to detect status transitions using LAG window function
	query := `
		WITH ordered_results AS (
			SELECT
				check_id,
				status,
				message,
				created_at,
				LAG(status) OVER (PARTITION BY check_id ORDER BY created_at) as prev_status
			FROM health_results
			WHERE created_at >= NOW() - INTERVAL '1 hour' * $1
		)
		SELECT
			check_id,
			created_at,
			prev_status,
			status,
			message
		FROM ordered_results
		WHERE prev_status IS NOT NULL AND prev_status != status
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.db.QueryContext(ctx, query, windowHours, limit)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}
	defer rows.Close()

	var incidents []Incident
	for rows.Next() {
		var i Incident
		var timestamp time.Time

		if err := rows.Scan(&i.CheckID, &timestamp, &i.FromStatus, &i.ToStatus, &i.Message); err != nil {
			return nil, fmt.Errorf("scan failed: %w", err)
		}

		i.Timestamp = timestamp.UTC().Format(time.RFC3339)
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
