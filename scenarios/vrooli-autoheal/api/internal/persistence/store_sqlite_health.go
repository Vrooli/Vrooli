package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

func (s *Store) operationalRetentionStatusSQLite(ctx context.Context) (RetentionStatus, error) {
	var pageCount, pageSize int64
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pageCount); err != nil {
		return RetentionStatus{}, fmt.Errorf("read page count: %w", err)
	}
	if err := s.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return RetentionStatus{}, fmt.Errorf("read page size: %w", err)
	}
	status := RetentionStatus{DatabaseBytes: pageCount * pageSize}
	var oldest, newest sql.NullString
	if err := s.db.QueryRowContext(ctx, "SELECT MIN(created_at), MAX(created_at) FROM health_results").Scan(&oldest, &newest); err != nil {
		return RetentionStatus{}, fmt.Errorf("read retained range: %w", err)
	}
	if oldest.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, oldest.String)
		if err == nil {
			status.OldestAt = &parsed
		}
	}
	if newest.Valid {
		parsed, err := time.Parse(time.RFC3339Nano, newest.String)
		if err == nil {
			status.NewestAt = &parsed
		}
	}
	return status, nil
}

// rfc3339NanoCutoff returns now-windowHours as an RFC3339Nano UTC string,
// the canonical wire format for created_at columns. Computing the cutoff in
// Go (rather than in SQL via datetime('now', ?)) lets the SQLite planner use
// the indexes on created_at for direct string comparison.
func rfc3339NanoCutoff(now time.Time, windowHours int) string {
	return now.UTC().Add(-time.Duration(windowHours) * time.Hour).Format(time.RFC3339Nano)
}

func (s *Store) saveResultSQLite(ctx context.Context, result checks.Result) error {
	detailsJSON, err := json.Marshal(result.Details)
	if err != nil {
		detailsJSON = []byte("{}")
	}
	if len(detailsJSON) > maxHealthResultDetailsBytes {
		detailsJSON, _ = json.Marshal(map[string]interface{}{
			"truncated":      true,
			"original_bytes": len(detailsJSON),
		})
	}

	query := `
		INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err = s.db.ExecContext(
		ctx, query,
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
				ROW_NUMBER() OVER (PARTITION BY check_id ORDER BY created_at DESC) AS rn
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
		ORDER BY created_at DESC
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
		WHERE created_at < ?
	`
	cutoff := rfc3339NanoCutoff(time.Now(), retentionHours)
	result, err := s.db.ExecContext(ctx, query, cutoff)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (s *Store) pruneOperationalHistorySQLite(ctx context.Context, before time.Time, batchSize int) (RetentionResult, error) {
	if batchSize <= 0 {
		batchSize = defaultRetentionBatchSize
	}
	cutoff := before.UTC().Format(time.RFC3339Nano)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RetentionResult{}, err
	}
	defer func() { _ = tx.Rollback() }()
	prune := func(table, column string) (int64, error) {
		query := fmt.Sprintf("DELETE FROM %s WHERE rowid IN (SELECT rowid FROM %s WHERE %s < ? ORDER BY %s LIMIT ?)", table, table, column, column)
		result, err := tx.ExecContext(ctx, query, cutoff, batchSize)
		if err != nil {
			return 0, err
		}
		return result.RowsAffected()
	}
	var out RetentionResult
	if out.HealthResults, err = prune("health_results", "created_at"); err != nil {
		return RetentionResult{}, fmt.Errorf("prune health results: %w", err)
	}
	if out.ActionLogs, err = prune("action_logs", "created_at"); err != nil {
		return RetentionResult{}, fmt.Errorf("prune action logs: %w", err)
	}
	if out.Actions, err = prune("autoheal_actions", "created_at"); err != nil {
		return RetentionResult{}, fmt.Errorf("prune autoheal actions: %w", err)
	}
	if out.SystemEvents, err = prune("system_events", "occurred_at"); err != nil {
		return RetentionResult{}, fmt.Errorf("prune system events: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return RetentionResult{}, err
	}
	return out, nil
}
