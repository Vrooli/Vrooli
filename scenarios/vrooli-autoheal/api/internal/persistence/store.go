// Package persistence provides database operations for health check results
// [REQ:PERSIST-STORE-001] [REQ:PERSIST-QUERY-001] [REQ:PERSIST-QUERY-002]
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

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
