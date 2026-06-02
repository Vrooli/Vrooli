package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"system-monitor-api/internal/repository"
)

// EstimateMetricRetention reports the rows a prune at cutoff would remove,
// without modifying any data.
func (r *Repository) EstimateMetricRetention(_ context.Context, cutoff time.Time) (repository.RetentionEstimate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	est := repository.RetentionEstimate{Cutoff: cutoff}

	var (
		rowCount     int64
		payloadBytes sql.NullInt64
		minTS, maxTS sql.NullString
	)
	err := r.db.QueryRow(
		`SELECT COUNT(*), COALESCE(SUM(length(metric_data)), 0), MIN(timestamp), MAX(timestamp)
		 FROM metrics WHERE timestamp < ?`,
		cutoff.UTC(),
	).Scan(&rowCount, &payloadBytes, &minTS, &maxTS)
	if err != nil {
		return repository.RetentionEstimate{}, fmt.Errorf("estimate retention: %w", err)
	}

	est.RowCount = rowCount
	est.PayloadBytes = payloadBytes.Int64
	if minTS.Valid && minTS.String != "" {
		if ts, perr := parseTime(minTS.String); perr == nil {
			est.OldestAffected = ts
		}
	}
	if maxTS.Valid && maxTS.String != "" {
		if ts, perr := parseTime(maxTS.String); perr == nil {
			est.NewestAffected = ts
		}
	}
	return est, nil
}

// PruneMetricsBefore deletes metric rows older than cutoff inside a
// transaction, serialized against other writers via the write mutex.
func (r *Repository) PruneMetricsBefore(ctx context.Context, cutoff time.Time) (repository.RetentionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return repository.RetentionResult{}, fmt.Errorf("begin prune tx: %w", err)
	}

	res, err := tx.ExecContext(ctx, "DELETE FROM metrics WHERE timestamp < ?", cutoff.UTC())
	if err != nil {
		_ = tx.Rollback()
		return repository.RetentionResult{}, fmt.Errorf("prune metrics: %w", err)
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		_ = tx.Rollback()
		return repository.RetentionResult{}, fmt.Errorf("prune rows affected: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return repository.RetentionResult{}, fmt.Errorf("commit prune tx: %w", err)
	}

	return repository.RetentionResult{DeletedRows: deleted, Cutoff: cutoff}, nil
}

// SQLiteStats reports the current database storage footprint.
func (r *Repository) SQLiteStats(_ context.Context) (repository.DatabaseStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sqliteStatsLocked()
}

// sqliteStatsLocked reads database stats; callers must hold r.mu.
func (r *Repository) sqliteStatsLocked() (repository.DatabaseStats, error) {
	var stats repository.DatabaseStats

	for pragma, dest := range map[string]*int64{
		"PRAGMA page_size":      &stats.PageSize,
		"PRAGMA page_count":     &stats.PageCount,
		"PRAGMA freelist_count": &stats.FreelistCount,
	} {
		if err := r.db.QueryRow(pragma).Scan(dest); err != nil {
			return repository.DatabaseStats{}, fmt.Errorf("%s: %w", pragma, err)
		}
	}

	if err := r.db.QueryRow("SELECT COUNT(*) FROM metrics").Scan(&stats.MetricRows); err != nil {
		return repository.DatabaseStats{}, fmt.Errorf("count metrics: %w", err)
	}

	stats.SizeBytes = stats.PageSize * stats.PageCount
	return stats, nil
}

// Compact reclaims free space via VACUUM, serialized against writers.
func (r *Repository) Compact(_ context.Context) (repository.CompactionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	before, err := r.sqliteStatsLocked()
	if err != nil {
		return repository.CompactionResult{}, err
	}

	// VACUUM cannot run inside a transaction; the single open connection plus
	// the held write mutex guarantee no concurrent writers.
	if _, err := r.db.Exec("VACUUM"); err != nil {
		return repository.CompactionResult{}, fmt.Errorf("vacuum: %w", err)
	}

	after, err := r.sqliteStatsLocked()
	if err != nil {
		return repository.CompactionResult{}, err
	}

	reclaimed := before.SizeBytes - after.SizeBytes
	if reclaimed < 0 {
		reclaimed = 0
	}
	return repository.CompactionResult{
		StatsBefore:    before,
		StatsAfter:     after,
		ReclaimedBytes: reclaimed,
	}, nil
}
