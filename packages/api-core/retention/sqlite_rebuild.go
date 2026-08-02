package retention

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
)

// Batched deletion is the right tool for a steady-state cycle, where each pass
// removes a small slice of a table. It is the wrong tool for a one-off
// reduction that discards almost everything.
//
// Measured on the 455 GiB autoheal.sqlite this package was written for: deleting
// the oldest rows in batches ran at roughly 330 rows per second, because every
// removed row costs random I/O across four indexes on a file far larger than
// page cache. At that rate the 842M rows to remove would have taken about 700
// hours. Rebuilding instead reads only the ~3.7M rows that SURVIVE, in index
// order, and drops the rest as whole pages.
//
// The cost scales with what is KEPT, not with what is removed, which inverts
// exactly the wrong scaling.

// rebuildSuffix names the transient table holding the original rows.
const rebuildSuffix = "__retention_rebuild_old"

// RebuildToBudget reduces the table to its byte ceiling by rebuilding it around
// the rows that survive, rather than deleting the rows that do not.
//
// Use it for a one-off reduction that discards most of a table. For a steady
// cycle removing a small slice, Prune is both cheaper and safer.
//
// The original table's DDL and every index on it are captured first and
// recreated afterwards, so constraints, defaults, and indexes survive. The whole
// operation runs in one transaction: it either completes or leaves the table
// exactly as it was.
func (p *SQLiteTablePruner) RebuildToBudget(ctx context.Context, b Budget) (Result, error) {
	if !b.HasByteBound() {
		return Result{Budget: b.Name}, fmt.Errorf("rebuild %q: a byte ceiling is required to size the surviving set", b.Name)
	}
	db, ok := p.cfg.DB.(*sql.DB)
	if !ok {
		return Result{Budget: b.Name}, fmt.Errorf("rebuild %q: needs a *sql.DB to run in one transaction", b.Name)
	}

	if err := p.ensureTimeColumn(ctx); err != nil {
		return Result{Budget: b.Name}, err
	}

	if err := p.guardRebuildFreeSpace(ctx, b); err != nil {
		return Result{Budget: b.Name}, err
	}

	beforeBytes, err := p.databaseBytes(ctx)
	if err != nil {
		return Result{Budget: b.Name}, err
	}
	result := Result{Budget: b.Name, Before: Usage{Bytes: beforeBytes}, BoundBy: BoundBytes}

	tableDDL, indexDDL, err := p.captureSchema(ctx)
	if err != nil {
		return result, err
	}

	oldName := p.cfg.Table + rebuildSuffix
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return result, fmt.Errorf("rebuild %q: begin: %w", b.Name, err)
	}
	// Rollback is a no-op after a successful commit, and is what makes a failed
	// rebuild leave the table exactly as it was.
	defer func() { _ = tx.Rollback() }()

	// Step order here is load-bearing, and getting it wrong is not a slow
	// rebuild — it is a rebuild that cannot finish.
	//
	// The original's index on the time column is what makes the copy cheap: it
	// turns "the newest N rows" into a bounded walk of an index. Dropping the
	// indexes before the copy — the obvious way to free their names for the
	// replacements — silently removes that, so ORDER BY falls back to scanning
	// every row and sorting them externally. Measured on the 455 GiB database
	// that meant reading the whole file and spilling a sorter of comparable
	// size to temp, on a disk with 208 GiB free.
	//
	// So the original keeps its indexes until the copy is done. The renamed
	// table carries them, and they are dropped only afterwards to free the names.
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`ALTER TABLE %q RENAME TO %q`, p.cfg.Table, oldName)); err != nil {
		return result, fmt.Errorf("rebuild %q: rename original: %w", b.Name, err)
	}
	if _, err := tx.ExecContext(ctx, tableDDL); err != nil {
		return result, fmt.Errorf("rebuild %q: recreate table: %w", b.Name, err)
	}

	kept, err := p.copyNewestWithinCeiling(ctx, tx, oldName, b.MaxBytes)
	if err != nil {
		return result, fmt.Errorf("rebuild %q: %w", b.Name, err)
	}

	// Now the originals can go: their names are needed by the replacements, and
	// nothing reads them again.
	p.record("rebuild_drop_original_indexes")
	for _, name := range indexNames(indexDDL) {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DROP INDEX IF EXISTS %q`, name)); err != nil {
			return result, fmt.Errorf("rebuild %q: drop index %s: %w", b.Name, name, err)
		}
	}
	for _, ddl := range indexDDL {
		if _, err := tx.ExecContext(ctx, ddl); err != nil {
			return result, fmt.Errorf("rebuild %q: recreate index: %w", b.Name, err)
		}
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DROP TABLE %q`, oldName)); err != nil {
		return result, fmt.Errorf("rebuild %q: drop original: %w", b.Name, err)
	}
	if err := tx.Commit(); err != nil {
		return result, fmt.Errorf("rebuild %q: commit: %w", b.Name, err)
	}
	p.record("rebuild")

	if err := p.checkpoint(ctx); err != nil {
		return result, err
	}
	// Dropping the original leaves its pages on the freelist, so the file is
	// still its original size here. Compaction is what returns them, and it runs
	// after — never before, for the same reason it never precedes a prune.
	skipped, reason, compactErr := p.compact(ctx)
	result.CompactSkipped = skipped
	result.CompactSkipReason = reason
	if compactErr != nil {
		p.cfg.Logger.WarnContext(ctx, "rebuild succeeded but compaction failed; rows are gone and pages are still held",
			"budget", b.Name, "error", compactErr)
	}

	afterBytes, err := p.databaseBytes(ctx)
	if err != nil {
		return result, err
	}
	result.After = Usage{Bytes: afterBytes, Items: kept}
	if afterBytes < beforeBytes {
		result.FreedBytes = beforeBytes - afterBytes
	}
	return result, nil
}

// rebuildHeadroom is the multiplier applied to a rebuild's projected peak before
// comparing it against free space.
const rebuildHeadroom = 115 // percent

// guardRebuildFreeSpace refuses a rebuild the filesystem cannot hold.
//
// A rebuild GROWS the database before it shrinks it, and that is inherent rather
// than a defect to remove: the original table has to keep its rows and its index
// on the time column until the copy is done, because that index is the only
// thing making "the newest N rows" a bounded walk instead of a full scan and
// external sort. So the surviving rows exist twice at the moment before the
// swap.
//
// The peak is therefore about two ceilings above the starting size — one for the
// copy the transaction has written into the WAL, one for what the checkpoint
// then folds into the main file — and none of it is returned until the drop
// commits and compaction runs. Running that on a filesystem which cannot hold it
// fails PART WAY THROUGH A WRITE, which is the one outcome a retention mechanism
// must never produce: it is the disk-full condition this package exists to
// prevent, arriving by way of the tool meant to fix it.
//
// Checking first converts an out-of-space abort into a refusal with a number in
// it. Prune is the fallback and needs no headroom, so the operator has a way
// forward either way.
func (p *SQLiteTablePruner) guardRebuildFreeSpace(ctx context.Context, b Budget) error {
	dir := filepath.Dir(p.cfg.Path)
	available, err := p.cfg.FreeSpace(dir)
	if err != nil {
		return fmt.Errorf("rebuild %q: free space on %s could not be measured: %w", b.Name, dir, err)
	}
	required := b.MaxBytes / 100 * rebuildHeadroom * 2
	if available >= required {
		return nil
	}
	p.cfg.Logger.WarnContext(ctx, "refusing retention rebuild: insufficient free space",
		"budget", b.Name, "required_bytes", required, "available_bytes", available, "database", p.cfg.Path)
	return fmt.Errorf(
		"%w: rebuild of %q needs about %s free because the surviving rows exist twice until the swap, but only %s is available on %s; prune instead, which needs no headroom",
		ErrInsufficientSpace, b.Name, FormatBytes(required), FormatBytes(available), dir,
	)
}

// probeBatch is how many rows the first copy pass inserts before the real
// bytes-per-row is measured. Large enough that index overhead is represented,
// small enough to be cheap on any table.
const probeBatch = 2_000

// copyNewestWithinCeiling fills the rebuilt table with the newest rows that fit
// under maxBytes, MEASURING the cost per row rather than estimating it.
//
// The estimate this replaces divided live bytes by the rowid range, and on an
// AUTOINCREMENT table with a long history of deletes the range overstated the
// live row count by roughly 8x. That understated bytes-per-row by the same
// factor, so the rebuild kept eight times too many rows and landed at 15.8 GiB
// against a 2 GiB ceiling. A row's true cost includes its share of every index,
// which no arithmetic over the whole file can recover.
//
// So: insert a probe batch, measure what the database actually grew by, and size
// the rest from that. Each pass re-measures, so the estimate converges instead of
// compounding one bad guess.
func (p *SQLiteTablePruner) copyNewestWithinCeiling(ctx context.Context, tx *sql.Tx, oldName string, maxBytes int64) (int64, error) {
	copyStmt := fmt.Sprintf(
		`INSERT INTO %q SELECT * FROM %q ORDER BY %q DESC LIMIT ? OFFSET ?`,
		p.cfg.Table, oldName, p.cfg.TimeColumn,
	)

	baseline, err := p.txLiveBytes(ctx, tx)
	if err != nil {
		return 0, err
	}

	var kept int64
	batch := int64(probeBatch)
	p.record("rebuild_copy")
	for {
		if err := ctx.Err(); err != nil {
			return kept, err
		}
		res, err := tx.ExecContext(ctx, copyStmt, batch, kept)
		if err != nil {
			return kept, fmt.Errorf("copy surviving rows: %w", err)
		}
		inserted, err := res.RowsAffected()
		if err != nil {
			return kept, fmt.Errorf("count surviving rows: %w", err)
		}
		kept += inserted
		if inserted == 0 {
			// The original ran out of rows before the ceiling did.
			return kept, nil
		}

		grown, err := p.txLiveBytes(ctx, tx)
		if err != nil {
			return kept, err
		}
		used := grown - baseline
		if used >= maxBytes {
			// A pass can overshoot, and the first one always can: nothing is
			// known about per-row cost until something has been copied. Trim
			// back to the ceiling exactly rather than leaving the rebuild over
			// the budget it exists to enforce.
			return p.trimToCeiling(ctx, tx, baseline, maxBytes, kept)
		}

		// Size the next pass from what the copy has actually cost so far.
		perRow := used / kept
		if perRow <= 0 {
			perRow = 1
		}
		batch = (maxBytes - used) / perRow
		if batch <= 0 {
			return kept, nil
		}
		if batch > probeBatch*50 {
			batch = probeBatch * 50
		}
	}
}

// trimToCeiling deletes the oldest of the just-copied rows until the rebuilt
// table fits under maxBytes. It runs on the new table, which is small, so this
// is the one place an exact answer is cheap.
func (p *SQLiteTablePruner) trimToCeiling(ctx context.Context, tx *sql.Tx, baseline, maxBytes, kept int64) (int64, error) {
	trimStmt := fmt.Sprintf(
		`DELETE FROM %q WHERE rowid IN (SELECT rowid FROM %q ORDER BY %q ASC LIMIT ?)`,
		p.cfg.Table, p.cfg.Table, p.cfg.TimeColumn,
	)
	p.record("rebuild_trim")

	for kept > 0 {
		if err := ctx.Err(); err != nil {
			return kept, err
		}
		grown, err := p.txLiveBytes(ctx, tx)
		if err != nil {
			return kept, err
		}
		used := grown - baseline
		if used <= maxBytes {
			return kept, nil
		}

		// Drop the fraction that is over, with a floor so a tiny overshoot
		// still makes progress.
		perRow := used / kept
		if perRow <= 0 {
			perRow = 1
		}
		drop := (used - maxBytes) / perRow
		if drop < 1 {
			drop = 1
		}
		if drop > kept {
			drop = kept
		}
		res, err := tx.ExecContext(ctx, trimStmt, drop)
		if err != nil {
			return kept, fmt.Errorf("trim to ceiling: %w", err)
		}
		removed, err := res.RowsAffected()
		if err != nil {
			return kept, fmt.Errorf("trim to ceiling: %w", err)
		}
		if removed == 0 {
			return kept, nil
		}
		kept -= removed
	}
	return kept, nil
}

// txLiveBytes measures live payload inside the open transaction, so the growth
// attributable to the copy is visible before it commits.
func (p *SQLiteTablePruner) txLiveBytes(ctx context.Context, tx *sql.Tx) (int64, error) {
	var pageCount, pageSize, freelist int64
	for _, probe := range []struct {
		pragma string
		into   *int64
	}{
		{`PRAGMA page_count`, &pageCount},
		{`PRAGMA page_size`, &pageSize},
		{`PRAGMA freelist_count`, &freelist},
	} {
		if err := tx.QueryRowContext(ctx, probe.pragma).Scan(probe.into); err != nil {
			return 0, fmt.Errorf("read %s: %w", probe.pragma, err)
		}
	}
	pages := pageCount - freelist
	if pages < 0 {
		pages = 0
	}
	return pages * pageSize, nil
}

// captureSchema returns the table's own DDL and the DDL of every index on it, so
// both can be recreated after the rebuild. Indexes SQLite created implicitly for
// UNIQUE and PRIMARY KEY constraints have no DDL of their own and are excluded:
// recreating the table from its DDL brings them back.
func (p *SQLiteTablePruner) captureSchema(ctx context.Context) (tableDDL string, indexDDL []string, err error) {
	row := p.cfg.DB.QueryRowContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type='table' AND name=?`, p.cfg.Table)
	if err := row.Scan(&tableDDL); err != nil {
		return "", nil, fmt.Errorf("read schema for %s: %w", p.cfg.Table, err)
	}

	rows, err := p.cfg.DB.QueryContext(ctx,
		`SELECT sql FROM sqlite_master WHERE type='index' AND tbl_name=? AND sql IS NOT NULL`, p.cfg.Table)
	if err != nil {
		return "", nil, fmt.Errorf("read indexes for %s: %w", p.cfg.Table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var ddl string
		if err := rows.Scan(&ddl); err != nil {
			return "", nil, fmt.Errorf("read index ddl for %s: %w", p.cfg.Table, err)
		}
		indexDDL = append(indexDDL, ddl)
	}
	if err := rows.Err(); err != nil {
		return "", nil, fmt.Errorf("read indexes for %s: %w", p.cfg.Table, err)
	}
	return tableDDL, indexDDL, nil
}

// indexNames extracts the index name from each CREATE INDEX statement, so the
// originals can be dropped before their replacements are created.
func indexNames(indexDDL []string) []string {
	names := make([]string, 0, len(indexDDL))
	for _, ddl := range indexDDL {
		fields := strings.Fields(ddl)
		for i, f := range fields {
			if !strings.EqualFold(f, "INDEX") || i+1 >= len(fields) {
				continue
			}
			name := fields[i+1]
			// Skip the optional IF NOT EXISTS that autoheal's schema uses.
			if strings.EqualFold(name, "IF") && i+4 < len(fields) {
				name = fields[i+4]
			}
			name = strings.Trim(name, `"'`+"`[]")
			if idx := strings.IndexAny(name, "("); idx >= 0 {
				name = name[:idx]
			}
			if name != "" {
				names = append(names, name)
			}
			break
		}
	}
	return names
}
