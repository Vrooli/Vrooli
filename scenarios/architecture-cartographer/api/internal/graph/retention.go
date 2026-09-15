package graph

// DOC: docs/reference/storage-retention.md

import (
	"context"
	"fmt"
	"strings"
)

// DefaultSnapshotRetentionKeep is how many snapshots per scenario are kept when
// nothing else is configured.
//
// Three is what the incident cleanup used: it took graph_snapshots from 2,469
// rows and 77.2 GB down to 238 rows and 3.26 GB across 94 scenarios, while
// still leaving enough history to compare a change against its predecessors.
const DefaultSnapshotRetentionKeep = 3

// defaultRetentionBatchSize bounds how many ids are deleted per statement.
//
// Batching matters for a reason discovered the expensive way during the
// incident cleanup: a DELETE that re-runs a ROW_NUMBER window over the whole
// table on every batch is quadratic, and pruning 2,231 rows that way took over
// twenty minutes of full-CPU work. This implementation selects the doomed ids
// once and then deletes by id, so each batch is a cheap indexed delete.
const defaultRetentionBatchSize = 200

// RetentionPolicy describes how much snapshot history to keep.
type RetentionPolicy struct {
	// KeepPerScenario is the number of newest snapshots retained for each
	// scenario. Values below one are treated as the default rather than as
	// "delete everything", because a misconfigured zero must not be a way to
	// wipe the table.
	KeepPerScenario int

	// BatchSize bounds ids per DELETE statement.
	BatchSize int
}

func (p RetentionPolicy) keep() int {
	if p.KeepPerScenario < 1 {
		return DefaultSnapshotRetentionKeep
	}
	return p.KeepPerScenario
}

func (p RetentionPolicy) batch() int {
	if p.BatchSize < 1 {
		return defaultRetentionBatchSize
	}
	return p.BatchSize
}

// RetentionResult reports what one retention pass did.
type RetentionResult struct {
	// ScenariosScanned is how many scenarios had snapshots.
	ScenariosScanned int
	// RowsRemoved is how many snapshot rows were deleted.
	RowsRemoved int
	// BytesReclaimed is the summed payload length of the deleted rows. It is
	// measured before deletion, because afterwards there is nothing to measure.
	BytesReclaimed int64
	// PagesFreed is how many database pages incremental vacuum returned to the
	// filesystem. Deleting rows alone does not shrink the file.
	PagesFreed int64
}

const (
	// selectDoomedSnapshotsSQL picks every row beyond the retention floor, in
	// one pass over the table. The window function runs exactly once per
	// retention pass, not once per delete batch.
	//
	// Ordering matches the read path (extracted_at DESC, id DESC) so the rows
	// kept are exactly the rows a reader would consider newest.
	selectDoomedSnapshotsSQL = `
SELECT id, length(payload) FROM (
  SELECT id,
         payload,
         ROW_NUMBER() OVER (PARTITION BY scenario ORDER BY extracted_at DESC, id DESC) AS rn
  FROM graph_snapshots
)
WHERE rn > ?`

	countSnapshotScenariosSQL = `SELECT COUNT(DISTINCT scenario) FROM graph_snapshots`
)

// doomedSnapshot is one row selected for deletion.
type doomedSnapshot struct {
	id           string
	payloadBytes int64
}

// PruneSnapshots enforces the retention policy and returns what it removed.
//
// This is the fix for the defect that filled the host disk. A unique index on
// (scenario, content_hash) means every distinct code state is retained
// permanently, so an active repository grows graph_snapshots forever by
// construction. Nothing in the codebase deleted anything except a full
// clear-by-scenario.
func (r *sqliteRepository) PruneSnapshots(ctx context.Context, policy RetentionPolicy) (RetentionResult, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return RetentionResult{}, err
	}

	var result RetentionResult
	if err := r.db.QueryRowContext(ctx, countSnapshotScenariosSQL).Scan(&result.ScenariosScanned); err != nil {
		return RetentionResult{}, fmt.Errorf("count snapshot scenarios: %w", err)
	}

	doomed, err := r.selectDoomedSnapshots(ctx, policy.keep())
	if err != nil {
		return RetentionResult{}, err
	}
	if len(doomed) == 0 {
		return result, nil
	}

	for _, row := range doomed {
		result.BytesReclaimed += row.payloadBytes
	}

	removed, err := r.deleteSnapshotsByID(ctx, doomed, policy.batch())
	result.RowsRemoved = removed
	if err != nil {
		return result, err
	}

	pages, err := r.reclaimFreePages(ctx)
	if err != nil {
		// Freed pages that stay in the file are a space problem, not a
		// correctness one: the rows are gone either way. Report it without
		// discarding the prune that already succeeded.
		return result, fmt.Errorf("reclaim free pages: %w", err)
	}
	result.PagesFreed = pages

	return result, nil
}

// selectDoomedSnapshots runs the window query once and materialises the ids to
// delete.
func (r *sqliteRepository) selectDoomedSnapshots(ctx context.Context, keep int) ([]doomedSnapshot, error) {
	rows, err := r.db.QueryContext(ctx, selectDoomedSnapshotsSQL, keep)
	if err != nil {
		return nil, fmt.Errorf("select snapshots beyond retention: %w", err)
	}
	defer rows.Close()

	var out []doomedSnapshot
	for rows.Next() {
		var row doomedSnapshot
		if err := rows.Scan(&row.id, &row.payloadBytes); err != nil {
			return nil, fmt.Errorf("scan doomed snapshot: %w", err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate doomed snapshots: %w", err)
	}
	return out, nil
}

// deleteSnapshotsByID deletes in bounded batches, checkpointing the
// write-ahead log after each one.
//
// The checkpoint is not optional. In WAL mode a large prune moves every
// deleted page into the WAL, so a retention pass that reclaims gigabytes would
// grow the -wal file by roughly that much before committing — the opposite of
// the intended effect, and actively dangerous when the disk is already full.
func (r *sqliteRepository) deleteSnapshotsByID(ctx context.Context, doomed []doomedSnapshot, batchSize int) (int, error) {
	removed := 0
	for start := 0; start < len(doomed); start += batchSize {
		end := start + batchSize
		if end > len(doomed) {
			end = len(doomed)
		}
		batch := doomed[start:end]

		args := make([]any, 0, len(batch))
		for _, row := range batch {
			args = append(args, row.id)
		}

		query := "DELETE FROM graph_snapshots WHERE id IN (" + placeholders(len(batch)) + ")"
		res, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			return removed, fmt.Errorf("delete snapshot batch: %w", err)
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return removed, fmt.Errorf("rows affected: %w", err)
		}
		removed += int(affected)

		if _, err := r.db.ExecContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`); err != nil {
			return removed, fmt.Errorf("checkpoint wal after batch: %w", err)
		}
	}
	return removed, nil
}

// reclaimFreePages returns freed pages to the filesystem.
//
// Deleting rows does not shrink the file: the pages go on the freelist and are
// reused later. After the incident's manual prune the payload total was 3.26
// GB while the file still measured 73 GB, and only a full VACUUM recovered it.
//
// A full VACUUM is the wrong tool for scheduled retention: it rewrites the
// whole database, needs free space to do it, and takes the database offline
// for the duration — exactly what is unavailable during a disk-full event.
// incremental_vacuum releases pages a chunk at a time with the database live.
func (r *sqliteRepository) reclaimFreePages(ctx context.Context) (int64, error) {
	before, err := r.freelistCount(ctx)
	if err != nil {
		return 0, err
	}
	if before == 0 {
		return 0, nil
	}

	if _, err := r.db.ExecContext(ctx, `PRAGMA incremental_vacuum`); err != nil {
		return 0, fmt.Errorf("incremental_vacuum: %w", err)
	}

	after, err := r.freelistCount(ctx)
	if err != nil {
		return 0, err
	}
	if after > before {
		return 0, nil
	}
	return before - after, nil
}

func (r *sqliteRepository) freelistCount(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.QueryRowContext(ctx, `PRAGMA freelist_count`).Scan(&count); err != nil {
		return 0, fmt.Errorf("read freelist_count: %w", err)
	}
	return count, nil
}

// SnapshotCounts reports how many snapshots each scenario holds. It backs the
// retention preview command and the owner cleanup provider's estimate.
func (r *sqliteRepository) SnapshotCounts(ctx context.Context) (map[string]int, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT scenario, COUNT(*) FROM graph_snapshots GROUP BY scenario`)
	if err != nil {
		return nil, fmt.Errorf("count snapshots per scenario: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var (
			scenario string
			count    int
		)
		if err := rows.Scan(&scenario, &count); err != nil {
			return nil, fmt.Errorf("scan snapshot count: %w", err)
		}
		counts[scenario] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate snapshot counts: %w", err)
	}
	return counts, nil
}

// ReclaimableSnapshotBytes reports the payload bytes held by snapshots beyond
// the retention floor, without deleting anything.
//
// It reports payload length rather than database file size on purpose:
// graph_snapshots shares its file with twelve other tables, and reporting the
// whole file would claim reclaimable space that is not.
func (r *sqliteRepository) ReclaimableSnapshotBytes(ctx context.Context, policy RetentionPolicy) (int64, int, error) {
	if err := r.ensureSourceFingerprintColumn(ctx); err != nil {
		return 0, 0, err
	}
	doomed, err := r.selectDoomedSnapshots(ctx, policy.keep())
	if err != nil {
		return 0, 0, err
	}
	var total int64
	for _, row := range doomed {
		total += row.payloadBytes
	}
	return total, len(doomed), nil
}

// SnapshotPayloadBytes reports the total live payload across all snapshots.
//
// This is the measurement a storage budget is judged against, and it is
// deliberately not ReclaimableSnapshotBytes with a zero policy: RetentionPolicy
// treats KeepPerScenario < 1 as the DEFAULT floor rather than as "keep nothing",
// so a zero policy reports what is reclaimable beyond the floor, not the total.
func (r *sqliteRepository) SnapshotPayloadBytes(ctx context.Context) (int64, error) {
	var total int64
	err := r.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(LENGTH(payload)), 0) FROM graph_snapshots`).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("sum snapshot payload bytes: %w", err)
	}
	return total, nil
}

// placeholders builds "?, ?, ?" for an IN clause of n values.
func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", n), ",")
}
