package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// EstimateMetricRetention reports the rows a prune at cutoff would remove,
// without modifying any data.
func (r *Repository) EstimateMetricRetention(ctx context.Context, cutoff time.Time) (repository.RetentionEstimate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	est := repository.RetentionEstimate{Cutoff: cutoff}

	var (
		rowCount     int64
		payloadBytes sql.NullInt64
		minTS, maxTS sql.NullString
	)
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(length(metric_data)), 0), MIN(observed_at), MAX(observed_at)
		 FROM metrics WHERE observed_at < ?`,
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

	res, err := tx.ExecContext(ctx, "DELETE FROM metrics WHERE observed_at < ?", cutoff.UTC())
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

// collectorPruneBatch bounds one collector-prune transaction. A collector's
// first prune after a window is shortened can cover gigabytes (device-graph
// snapshots are ~125 KB each); deleting it in one statement would hold the
// write mutex for minutes and starve collection writes and health reads.
const collectorPruneBatch = 200

// PruneCollectorMetricsBefore deletes one collector's metric rows older than
// cutoff in bounded batches, releasing the write mutex between batches so
// concurrent writers and readers keep making progress during a large backlog.
func (r *Repository) PruneCollectorMetricsBefore(ctx context.Context, collector string, cutoff time.Time) (repository.RetentionResult, error) {
	result := repository.RetentionResult{Cutoff: cutoff}
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		deleted, err := r.pruneCollectorBatch(ctx, collector, cutoff)
		if err != nil {
			return result, err
		}
		result.DeletedRows += deleted
		if deleted < collectorPruneBatch {
			return result, nil
		}
	}
}

func (r *Repository) pruneCollectorBatch(ctx context.Context, collector string, cutoff time.Time) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	res, err := r.db.ExecContext(ctx,
		`DELETE FROM metrics WHERE id IN (
			SELECT id FROM metrics WHERE collector_name = ? AND observed_at < ? LIMIT ?)`,
		collector, cutoff.UTC(), collectorPruneBatch)
	if err != nil {
		return 0, fmt.Errorf("prune %s metrics: %w", collector, err)
	}
	deleted, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("prune %s rows affected: %w", collector, err)
	}
	return deleted, nil
}

// ReleaseFreePages returns free pages in bounded incremental_vacuum batches,
// taking the write lock per batch so collection writes and reads interleave.
// It only shrinks the file once the database is in auto_vacuum=INCREMENTAL
// mode; in NONE mode it releases nothing and returns at once.
func (r *Repository) ReleaseFreePages(ctx context.Context, batchPages int64) (int64, error) {
	if batchPages <= 0 {
		batchPages = 1024
	}
	var released int64
	for {
		if err := ctx.Err(); err != nil {
			return released, err
		}
		freed, more, err := r.releaseFreePagesBatch(ctx, batchPages)
		released += freed
		if err != nil || !more {
			return released, err
		}
	}
}

func (r *Repository) releaseFreePagesBatch(ctx context.Context, batchPages int64) (int64, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var mode, pageSize, free int64
	if err := r.db.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&mode); err != nil {
		return 0, false, fmt.Errorf("read auto_vacuum: %w", err)
	}
	if mode != repository.AutoVacuumIncremental {
		return 0, false, nil
	}
	if err := r.db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
		return 0, false, err
	}
	if err := r.db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&free); err != nil {
		return 0, false, err
	}
	if free == 0 {
		return 0, false, nil
	}
	// incremental_vacuum(N) returns one row per freed page; drain the result
	// set so the pragma runs to completion.
	rows, err := r.db.QueryContext(ctx, fmt.Sprintf("PRAGMA incremental_vacuum(%d)", batchPages))
	if err != nil {
		return 0, false, fmt.Errorf("incremental vacuum: %w", err)
	}
	for rows.Next() { //nolint:revive // draining is the work
		// Each row is one released page; nothing to read.
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return 0, false, fmt.Errorf("incremental vacuum rows: %w", err)
	}
	_ = rows.Close()

	var after int64
	if err := r.db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&after); err != nil {
		return 0, false, err
	}
	freed := (free - after) * pageSize
	return freed, after > 0 && after < free, nil
}

// SQLiteStats reports the current database storage footprint.
func (r *Repository) SQLiteStats(ctx context.Context) (repository.DatabaseStats, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.sqliteStatsLocked(ctx)
}

// sqliteStatsLocked reads database stats; callers must hold r.mu.
func (r *Repository) sqliteStatsLocked(ctx context.Context) (repository.DatabaseStats, error) {
	var stats repository.DatabaseStats

	for pragma, dest := range map[string]*int64{
		"PRAGMA page_size":      &stats.PageSize,
		"PRAGMA page_count":     &stats.PageCount,
		"PRAGMA freelist_count": &stats.FreelistCount,
	} {
		if err := r.db.QueryRowContext(ctx, pragma).Scan(dest); err != nil {
			return repository.DatabaseStats{}, fmt.Errorf("%s: %w", pragma, err)
		}
	}

	if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM metrics").Scan(&stats.MetricRows); err != nil {
		return repository.DatabaseStats{}, fmt.Errorf("count metrics: %w", err)
	}

	if err := r.db.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&stats.AutoVacuum); err != nil {
		return repository.DatabaseStats{}, fmt.Errorf("PRAGMA auto_vacuum: %w", err)
	}
	stats.SizeBytes = stats.PageSize * stats.PageCount
	stats.Path = r.path
	if r.path != "" {
		if info, err := os.Stat(r.path + "-wal"); err == nil {
			stats.WALBytes = info.Size()
		}
	}
	return stats, nil
}

// Compact reclaims free space via VACUUM, serialized against writers.
func (r *Repository) Compact(ctx context.Context) (repository.CompactionResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	before, err := r.sqliteStatsLocked(ctx)
	if err != nil {
		return repository.CompactionResult{}, err
	}

	// VACUUM cannot run inside a transaction; the single open connection plus
	// the held write mutex guarantee no concurrent writers.
	r.compacting.Store(true)
	defer r.compacting.Store(false)

	// VACUUM's transient copy goes to a temp file rather than RAM.
	if _, err := r.db.ExecContext(ctx, "PRAGMA temp_store=FILE"); err != nil {
		return repository.CompactionResult{}, fmt.Errorf("set temp_store: %w", err)
	}
	// auto_vacuum is a pending per-connection setting until a VACUUM applies
	// it; set it on this connection so the rewrite converts a legacy
	// auto_vacuum=NONE file to INCREMENTAL.
	if _, err := r.db.ExecContext(ctx, "PRAGMA auto_vacuum=INCREMENTAL"); err != nil {
		return repository.CompactionResult{}, fmt.Errorf("set auto_vacuum: %w", err)
	}
	if _, err := r.db.ExecContext(ctx, "VACUUM"); err != nil {
		return repository.CompactionResult{}, fmt.Errorf("vacuum: %w", err)
	}

	// In WAL mode VACUUM writes the rewritten database through the WAL;
	// truncate it so the sidecar does not keep the reclaimed bytes.
	if _, err := r.db.ExecContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)"); err != nil {
		return repository.CompactionResult{}, fmt.Errorf("checkpoint after vacuum: %w", err)
	}

	after, err := r.sqliteStatsLocked(ctx)
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
