package retention

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"time"
)

// DefaultBatchSize is how many rows one delete statement removes. It is
// deliberately conservative: deleting 846M rows in one statement builds a
// journal larger than the data, which turns a disk problem into a bigger disk
// problem.
const DefaultBatchSize = 10_000

// compactionHeadroom is the multiplier applied to the projected copy size before
// comparing it against free space. A VACUUM writes a complete new copy and then
// swaps, so the peak requirement is the copy plus journal overhead, not the copy
// alone.
const compactionHeadroom = 115 // percent

// DefaultReclaimPercent is how far BELOW its ceiling a byte-bound prune reduces
// a target, as a percentage of the ceiling.
//
// Pruning to exactly the ceiling is what makes a busy table prune on every
// single cycle forever. The table lands one row under its limit, the producer
// writes for one interval, and the next cycle is over again — so the expensive
// half of retention (measure, batch-delete, checkpoint, reclaim) becomes
// permanent background load rather than an occasional correction. Measured on
// autoheal: health_results sat at 1.96 GiB of a 2 GiB ceiling, so every 15-minute
// cycle re-entered the delete loop and held the database busy for minutes.
//
// Reclaiming to a low-water mark converts that into real duty-cycling: a cycle
// does visible work, then several cycles find nothing to do. The headroom is the
// producer's runway, and one cycle's worth of writes is what it has to cover.
const DefaultReclaimPercent = 90

// reclaimFloorPercent is the lowest low-water mark a caller may declare. A
// ceiling is a promise about disk, but so is the data the target holds; letting
// a config reclaim to 10% of the ceiling would quietly discard nine tenths of
// the retained history that the budget's author asked to keep.
const reclaimFloorPercent = 50

// SQLiteTableConfig configures the builtin pruner for a sqlite_table target.
type SQLiteTableConfig struct {
	// DB is the open handle to the target database. Required.
	DB Execer
	// Path is the absolute path of the database file. Required: the free-space
	// guard measures the filesystem holding it.
	Path string
	// Table is the bounded table. Required.
	Table string
	// TimeColumn carries row age; it orders deletes oldest-first and is what
	// MaxAge is evaluated against. Required.
	TimeColumn string
	// BatchSize is rows per delete statement. Defaults to DefaultBatchSize.
	BatchSize int
	// Now supplies the current time. Defaults to time.Now.
	Now func() time.Time
	// FreeSpace probes available bytes. Defaults to FreeSpace.
	FreeSpace FreeSpaceFunc
	// AllowFullVacuum permits the one-time full VACUUM that converts a database
	// to incremental auto-vacuum. It is off by default and belongs to an
	// explicit operator command: a full VACUUM writes a complete second copy of
	// the database, so it must never happen as a side effect of startup.
	//
	// Incremental reclamation is NOT gated by this. Once a database is in
	// incremental mode, returning freed pages costs no second copy and always
	// runs, because without it the file never shrinks and the byte bound can
	// never be satisfied.
	AllowFullVacuum bool
	// ReclaimPercent is how far below the byte ceiling a byte-bound prune
	// reduces the target, as a percentage of that ceiling. Defaults to
	// DefaultReclaimPercent. Values above 100 or below reclaimFloorPercent are
	// rejected: the first is not a reduction and the second discards more
	// history than the budget's author declared.
	//
	// It has no effect on the age bound, which is an absolute statement about
	// what may be retained rather than a threshold to back away from.
	ReclaimPercent int
	// BatchPause is how long the pruner waits between delete batches, yielding
	// the database to the serving path. Zero means no pause, which is right for
	// an operator command running against an idle database and wrong for a
	// scheduled cycle sharing a connection with live traffic.
	BatchPause time.Duration
	// MaxDuration bounds one Prune call's wall clock. Zero means unbounded,
	// which is right for an operator command and wrong for a scheduled cycle: on
	// a table with hundreds of millions of rows to remove, an unbounded cycle
	// would occupy the write lock for hours. Exceeding it stops cleanly and
	// reports Incomplete rather than erroring.
	MaxDuration time.Duration
	// Logger receives cycle detail. Defaults to slog.Default.
	Logger *slog.Logger
}

// SQLiteTablePruner enforces a budget over one table of one SQLite database.
type SQLiteTablePruner struct {
	cfg SQLiteTableConfig

	// stages records the operations performed, in order, so a test can assert
	// that pruning precedes compaction rather than inferring it from timing.
	stages []string
	// deadline is the wall-clock limit for the Prune call in flight.
	deadline time.Time
	// outOfTime records that the deadline stopped the cycle, which is reported
	// as Incomplete rather than as an error.
	outOfTime bool
}

// NewSQLiteTablePruner validates cfg and returns the pruner.
//
// Table and column names are validated as bare identifiers because they are
// interpolated into SQL: SQLite cannot bind an identifier as a parameter, so the
// only safe posture is to refuse anything that is not a plain identifier.
func NewSQLiteTablePruner(cfg SQLiteTableConfig) (*SQLiteTablePruner, error) {
	if cfg.DB == nil {
		return nil, fmt.Errorf("sqlite pruner: DB is required")
	}
	if strings.TrimSpace(cfg.Path) == "" {
		return nil, fmt.Errorf("sqlite pruner: Path is required for the free-space guard")
	}
	if err := validateIdentifier("table", cfg.Table); err != nil {
		return nil, err
	}
	if err := validateIdentifier("time column", cfg.TimeColumn); err != nil {
		return nil, err
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = DefaultBatchSize
	}
	if cfg.ReclaimPercent == 0 {
		cfg.ReclaimPercent = DefaultReclaimPercent
	}
	if cfg.ReclaimPercent > 100 || cfg.ReclaimPercent < reclaimFloorPercent {
		return nil, fmt.Errorf("sqlite pruner: reclaim percent %d is outside %d..100", cfg.ReclaimPercent, reclaimFloorPercent)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.FreeSpace == nil {
		cfg.FreeSpace = FreeSpace
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &SQLiteTablePruner{cfg: cfg}, nil
}

// validateIdentifier rejects anything that is not a bare SQL identifier.
func validateIdentifier(what, name string) error {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return fmt.Errorf("sqlite pruner: %s is required", what)
	}
	for i, r := range trimmed {
		alpha := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'
		digit := r >= '0' && r <= '9'
		if alpha || (digit && i > 0) {
			continue
		}
		return fmt.Errorf("sqlite pruner: %s %q is not a bare identifier", what, name)
	}
	return nil
}

// ensureTimeColumn verifies the declared time column actually exists.
//
// This guard exists because of a SQLite legacy quirk: a double-quoted identifier
// that matches no column is silently reinterpreted as a STRING LITERAL. So a
// typo'd time_column does not error — `ORDER BY "occured_at"` sorts every row by
// the same constant, which means "oldest first" becomes an arbitrary order and
// retention deletes the wrong rows without a word of complaint. Checking once
// per operation converts that into an immediate, named failure.
func (p *SQLiteTablePruner) ensureTimeColumn(ctx context.Context) error {
	rows, err := p.cfg.DB.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%q)`, p.cfg.Table))
	if err != nil {
		return fmt.Errorf("read columns of %s: %w", p.cfg.Table, err)
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var (
			cid          int
			name, ctype  string
			notNull, pk  int
			defaultValue sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("read columns of %s: %w", p.cfg.Table, err)
		}
		if name == p.cfg.TimeColumn {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read columns of %s: %w", p.cfg.Table, err)
	}
	if !found {
		return fmt.Errorf("%w: table %s has no column %q to order deletes by", ErrInvalidTarget, p.cfg.Table, p.cfg.TimeColumn)
	}
	return nil
}

// Measure reports what the TABLE occupies — its own pages plus those of every
// index on it — and its row count.
//
// Per-table, not per-file, and the distinction is load-bearing. An author who
// writes `"table": "system_events", "max_bytes": "2GiB"` means that table. An
// earlier version of this package measured the whole database file instead,
// reasoning that page_count is O(1) while dbstat walks pages. That was measured
// against the wrong risk: on the live database system_events was 0.86 GiB of a
// 15.1 GiB file (health_results held 12.2 GiB), so a 2 GiB FILE ceiling was
// unreachable no matter how much system_events was pruned — and the engine
// dutifully deleted the table toward empty chasing it.
//
// A ceiling the budgeted table cannot satisfy by shrinking is not a budget on
// that table. dbstat costs seconds on a database of any sane size, and a
// database that is not of sane size is the condition this package exists to
// prevent.
func (p *SQLiteTablePruner) Measure(ctx context.Context) (Usage, error) {
	bytes, err := p.tableBytes(ctx)
	if err != nil {
		return Usage{}, err
	}
	rows, err := p.rowCount(ctx)
	if err != nil {
		return Usage{}, err
	}
	return Usage{Bytes: bytes, Items: rows}, nil
}

// tableBytes reports the pages held by the table and every index on it.
//
// Indexes are included because they are storage the table causes: system_events
// carried 0.5 GiB of rows and 0.37 GiB of indexes, and a ceiling that ignored
// the second number would be understating the table's cost by 40%.
func (p *SQLiteTablePruner) tableBytes(ctx context.Context) (int64, error) {
	const query = `SELECT COALESCE(SUM(pgsize), 0) FROM dbstat WHERE name = ?
	    OR name IN (SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = ?)`
	var bytes int64
	if err := p.cfg.DB.QueryRowContext(ctx, query, p.cfg.Table, p.cfg.Table).Scan(&bytes); err != nil {
		return 0, fmt.Errorf("measure %s via dbstat: %w", p.cfg.Table, err)
	}
	return bytes, nil
}

// pageStats is one reading of how the database file is laid out. Every size
// question this pruner asks is answered from these three numbers.
type pageStats struct {
	// count is the total pages allocated to the file.
	count int64
	// size is the byte size of one page, and the granularity of every ceiling.
	size int64
	// free is the pages sitting in the file waiting to be reclaimed.
	free int64
}

// allocated is what the filesystem sees.
func (s pageStats) allocated() int64 { return s.count * s.size }

// live is the payload actually held, which is what a full VACUUM copies.
func (s pageStats) live() int64 {
	pages := s.count - s.free
	if pages < 0 {
		pages = 0
	}
	return pages * s.size
}

// readPageStats reads the three pragmas in one place. All three are O(1) header
// reads, unlike a row count, which is a full index scan.
func (p *SQLiteTablePruner) readPageStats(ctx context.Context) (pageStats, error) {
	var stats pageStats
	for _, probe := range []struct {
		pragma string
		into   *int64
	}{
		{`PRAGMA page_count`, &stats.count},
		{`PRAGMA page_size`, &stats.size},
		{`PRAGMA freelist_count`, &stats.free},
	} {
		if err := p.cfg.DB.QueryRowContext(ctx, probe.pragma).Scan(probe.into); err != nil {
			return pageStats{}, fmt.Errorf("read %s: %w", probe.pragma, err)
		}
	}
	return stats, nil
}

func (p *SQLiteTablePruner) databaseBytes(ctx context.Context) (int64, error) {
	stats, err := p.readPageStats(ctx)
	if err != nil {
		return 0, err
	}
	return stats.allocated(), nil
}

// liveBytes reports the payload the database actually holds, excluding freed
// pages it has not returned yet. This is what a full VACUUM copies, and the
// difference from the allocated size is the whole reason pruning runs first.
func (p *SQLiteTablePruner) liveBytes(ctx context.Context) (int64, error) {
	stats, err := p.readPageStats(ctx)
	if err != nil {
		return 0, err
	}
	return stats.live(), nil
}

// pageSize reports the granularity every size measurement and every ceiling is
// really expressed in.
func (p *SQLiteTablePruner) pageSize(ctx context.Context) (int64, error) {
	stats, err := p.readPageStats(ctx)
	if err != nil {
		return 0, err
	}
	return stats.size, nil
}

func (p *SQLiteTablePruner) rowCount(ctx context.Context) (int64, error) {
	var rows int64
	query := fmt.Sprintf(`SELECT COUNT(*) FROM %q`, p.cfg.Table)
	if err := p.cfg.DB.QueryRowContext(ctx, query).Scan(&rows); err != nil {
		return 0, fmt.Errorf("count rows in %s: %w", p.cfg.Table, err)
	}
	return rows, nil
}

// Prune reduces the table to b, then returns freed pages to the filesystem when
// compaction is enabled and the free-space guard permits it.
//
// The order is load-bearing and is asserted by test. VACUUM writes a complete
// new copy of the RESULT, so its cost is the size after pruning. On the host
// this package was written for, compacting first would have needed roughly
// 453 GB of free space against 226 GB available and would have failed part-way
// through a write; pruning first makes the copy roughly the budget size.
func (p *SQLiteTablePruner) Prune(ctx context.Context, b Budget) (Result, error) {
	p.stages = nil
	p.outOfTime = false
	p.deadline = time.Time{}
	if p.cfg.MaxDuration > 0 {
		p.deadline = p.cfg.Now().Add(p.cfg.MaxDuration)
	}

	if err := p.ensureTimeColumn(ctx); err != nil {
		return Result{Budget: b.Name}, err
	}

	// Byte-only measurement, deliberately not Measure. Measure counts rows, which
	// is a full index scan; on the table this package exists for that is slower
	// than the entire delete. The engine measures either side of this call and
	// overwrites Before/After with row counts included, so a scheduled cycle
	// still reports them without this hot path paying for them twice more.
	beforeBytes, err := p.databaseBytes(ctx)
	if err != nil {
		return Result{Budget: b.Name}, err
	}
	result := Result{Budget: b.Name, Before: Usage{Bytes: beforeBytes}, BoundBy: BoundNone}

	deletedByAge, err := p.pruneByAge(ctx, b)
	result.Deleted += deletedByAge
	if deletedByAge > 0 {
		result.BoundBy = BoundAge
	}
	if err != nil {
		result.Incomplete = isCancellation(err)
		return result, err
	}

	deletedByBytes, err := p.pruneByBytes(ctx, b)
	result.Deleted += deletedByBytes
	if deletedByBytes > 0 {
		result.BoundBy = BoundBytes
	}
	if err != nil {
		result.Incomplete = isCancellation(err)
		return result, err
	}

	skipped, reason, compactErr := p.compact(ctx)
	result.CompactSkipped = skipped
	result.CompactSkipReason = reason
	if compactErr != nil {
		// Compaction is an optimization. A table that is within budget but still
		// holding freed pages is a worse-shaped success, not a failure, and must
		// not mask the pruning that did work.
		p.cfg.Logger.WarnContext(ctx, "retention compaction failed; pruned rows are still gone but pages were not returned",
			"budget", b.Name, "error", compactErr)
	}

	afterBytes, err := p.databaseBytes(ctx)
	if err != nil {
		return result, err
	}
	result.After = Usage{Bytes: afterBytes}
	if afterBytes < beforeBytes {
		result.FreedBytes = beforeBytes - afterBytes
	}
	result.Incomplete = result.Incomplete || p.outOfTime
	return result, nil
}

// DatabaseBytes reports the database's allocated size: what the filesystem sees.
func (p *SQLiteTablePruner) DatabaseBytes(ctx context.Context) (int64, error) {
	return p.databaseBytes(ctx)
}

// LiveBytes reports the payload the database actually holds, excluding freed
// pages it has not returned yet. This is what a full VACUUM copies, and unlike a
// row count it costs three pragmas rather than a full index scan.
func (p *SQLiteTablePruner) LiveBytes(ctx context.Context) (int64, error) {
	return p.liveBytes(ctx)
}

// yield pauses between delete batches so the serving path can reach the
// database.
//
// Batched deletes already keep each individual write short, but a loop that
// issues them back to back still leaves no gap for anything else: on a database
// whose connection pool is one deep, retention holds the only connection for
// almost every instant of a multi-minute cycle, and every request-path query
// queues behind it. Autoheal's health probe has a 150ms budget, so a cycle with
// no gaps in it reads to the outside world as a dead database — which is exactly
// how a working prune got its own API killed and restarted.
//
// The pause is what makes a cycle preemptible in practice rather than only in
// principle. It returns ctx.Err() so a cancelled cycle stops in the pause rather
// than after one more batch.
func (p *SQLiteTablePruner) yield(ctx context.Context) error {
	if p.cfg.BatchPause <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(p.cfg.BatchPause)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// expired reports whether this Prune call has used its wall-clock allowance.
func (p *SQLiteTablePruner) expired() bool {
	if p.deadline.IsZero() {
		return false
	}
	if p.cfg.Now().Before(p.deadline) {
		return false
	}
	p.outOfTime = true
	return true
}

// Stages returns the operations this pruner performed on its last Prune, in
// order.
func (p *SQLiteTablePruner) Stages() []string {
	return append([]string(nil), p.stages...)
}

func (p *SQLiteTablePruner) record(stage string) { p.stages = append(p.stages, stage) }

// pruneByAge removes every row older than the declared horizon, oldest first.
func (p *SQLiteTablePruner) pruneByAge(ctx context.Context, b Budget) (int64, error) {
	if !b.HasAgeBound() {
		return 0, nil
	}
	cutoff, err := p.cutoffValue(ctx, b.MaxAge)
	if err != nil {
		return 0, err
	}

	query := fmt.Sprintf(
		`DELETE FROM %q WHERE rowid IN (SELECT rowid FROM %q WHERE %q < ? ORDER BY %q ASC LIMIT %d)`,
		p.cfg.Table, p.cfg.Table, p.cfg.TimeColumn, p.cfg.TimeColumn, p.cfg.BatchSize,
	)

	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		if p.expired() {
			return total, nil
		}
		n, err := p.execDelete(ctx, query, cutoff)
		if err != nil {
			return total, err
		}
		total += n
		if n > 0 {
			p.record("prune_age_batch")
			if err := p.checkpoint(ctx); err != nil {
				return total, err
			}
		}
		if n < int64(p.cfg.BatchSize) {
			return total, nil
		}
		if err := p.yield(ctx); err != nil {
			return total, err
		}
	}
}

// pruneByBytes removes the oldest rows until the database fits its byte ceiling.
//
// The loop cannot simply re-measure and repeat, because deleting rows does not
// shrink page_count on its own: freed pages stay in the file until they are
// reclaimed. So each pass converts the overage into a row count using the
// measured average bytes per row, deletes that many oldest rows, reclaims, and
// re-measures.
func (p *SQLiteTablePruner) pruneByBytes(ctx context.Context, b Budget) (int64, error) {
	if !b.HasByteBound() {
		return 0, nil
	}

	// A SQLite file's size is always a whole number of pages, so a ceiling that
	// is not page-aligned is not reachable. Round it DOWN to the page below:
	// rounding up would leave the file permanently a fraction of a page over its
	// declared ceiling and report BoundBytes forever, and down is the safe
	// direction for a bound on disk.
	ceiling := b.MaxBytes
	pageSize, err := p.pageSize(ctx)
	if err != nil {
		return 0, err
	}
	if pageSize > 0 {
		ceiling = ceiling / pageSize * pageSize
	}

	// The ceiling is what TRIGGERS a reduction; the low-water mark is what the
	// reduction aims at. Two thresholds rather than one is what stops a table
	// that lives near its limit from re-entering this loop on every cycle.
	lowWater := ceiling / 100 * int64(p.cfg.ReclaimPercent)
	if pageSize > 0 {
		lowWater = lowWater / pageSize * pageSize
	}
	if lowWater <= 0 || lowWater > ceiling {
		lowWater = ceiling
	}

	// The loop signal is what the TABLE occupies, and it responds immediately:
	// deleting rows moves their pages onto the freelist, so the table's measured
	// size drops on every batch even though the file has not shrunk yet.
	//
	// Watching the FILE here is what made the engine delete system_events toward
	// empty against a ceiling only health_results could have satisfied. Watching
	// allocated file size would also never move until a reclaim, so such a loop
	// either spins or empties the table.
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		if p.expired() {
			return total, nil
		}
		used, err := p.tableBytes(ctx)
		if err != nil {
			return total, err
		}
		// Trigger on the ceiling, but only on the first pass: once the reduction
		// has started, it runs until the low-water mark so the next cycle has
		// nothing to do. Stopping at the ceiling instead is what turned this loop
		// into permanent background load.
		target := ceiling
		if total > 0 {
			target = lowWater
		}
		if used <= target {
			return total, nil
		}

		deleted, err := p.deleteOldestBatch(ctx)
		if err != nil {
			return total, err
		}
		if deleted == 0 {
			// The table is empty and the payload is still over the ceiling, so
			// the overage is not this table's rows. Nothing further to delete;
			// the engine reports the overage as a finding.
			return total, nil
		}
		total += deleted
		if err := p.yield(ctx); err != nil {
			return total, err
		}
	}
}

// deleteOldestBatch removes one batch of the oldest rows and checkpoints the
// WAL, so a long prune does not simply relocate the space it is reclaiming.
func (p *SQLiteTablePruner) deleteOldestBatch(ctx context.Context) (int64, error) {
	query := fmt.Sprintf(
		`DELETE FROM %q WHERE rowid IN (SELECT rowid FROM %q ORDER BY %q ASC LIMIT %d)`,
		p.cfg.Table, p.cfg.Table, p.cfg.TimeColumn, p.cfg.BatchSize,
	)
	deleted, err := p.cfg.DB.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("delete from %s: %w", p.cfg.Table, err)
	}
	n, err := deleted.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected deleting from %s: %w", p.cfg.Table, err)
	}
	if n > 0 {
		p.record("prune_bytes_batch")
		if err := p.checkpoint(ctx); err != nil {
			return n, err
		}
	}
	return n, nil
}

func (p *SQLiteTablePruner) execDelete(ctx context.Context, query string, arg any) (int64, error) {
	res, err := p.cfg.DB.ExecContext(ctx, query, arg)
	if err != nil {
		return 0, fmt.Errorf("delete from %s: %w", p.cfg.Table, err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("rows affected deleting from %s: %w", p.cfg.Table, err)
	}
	return n, nil
}

// checkpoint truncates the WAL so a long prune does not simply relocate the
// space it is reclaiming.
func (p *SQLiteTablePruner) checkpoint(ctx context.Context) error {
	rows, err := p.cfg.DB.QueryContext(ctx, `PRAGMA wal_checkpoint(TRUNCATE)`)
	if err != nil {
		// A database not in WAL mode has nothing to checkpoint, which is not a
		// pruning failure.
		p.cfg.Logger.DebugContext(ctx, "wal checkpoint unavailable", "error", err)
		return nil
	}
	defer rows.Close()
	p.record("wal_checkpoint")
	return rows.Err()
}

// compact returns freed pages to the filesystem, refusing when the projected
// copy would not fit.
func (p *SQLiteTablePruner) compact(ctx context.Context) (skipped bool, reason string, err error) {
	mode, err := currentAutoVacuum(ctx, p.cfg.DB)
	if err != nil {
		return false, "", err
	}

	if mode == autoVacuumIncremental {
		// Incremental reclamation moves pages out of the file without writing a
		// second copy, so there is nothing for the free-space guard to protect
		// and no reason to gate it behind an operator flag. It is also the only
		// thing that lets the file shrink at all, so a byte bound is
		// unsatisfiable without it.
		if _, err := p.cfg.DB.ExecContext(ctx, `PRAGMA incremental_vacuum`); err != nil {
			return false, "", fmt.Errorf("incremental vacuum: %w", err)
		}
		p.record("compact_incremental")
		return false, "", nil
	}

	// Reaching incremental mode from mode 0 requires a full VACUUM, which
	// rewrites the whole database. That belongs to an explicit operator command,
	// never to a scheduled cycle.
	if !p.cfg.AllowFullVacuum {
		p.record("compact_skipped")
		return true, fmt.Sprintf(
			"database %s is at auto_vacuum=%d, so returning pages needs a one-time full VACUUM; run the operator compaction command",
			p.cfg.Path, mode,
		), nil
	}

	// A full VACUUM writes a complete new copy of the LIVE data before swapping
	// it in, so the requirement is the live payload, not the allocated file size.
	//
	// The difference is the whole reason pruning runs first. On the database this
	// package was written for, the allocated size stays at 455 GB immediately
	// after a large delete because freed pages remain in the file — but the live
	// payload is by then only the budget size, and that is what gets copied.
	// Projecting from the allocated size would refuse every compaction that
	// pruning had just made possible, which is the precise failure Decision 5 of
	// the plan exists to prevent.
	projected, err := p.liveBytes(ctx)
	if err != nil {
		return false, "", err
	}
	required := projected / 100 * compactionHeadroom
	available, err := p.cfg.FreeSpace(filepath.Dir(p.cfg.Path))
	if err != nil {
		return true, fmt.Sprintf("free space on %s could not be measured: %v", filepath.Dir(p.cfg.Path), err), nil
	}
	if available < required {
		reason := fmt.Sprintf(
			"compaction needs about %s free to copy the %s of live payload, but only %s is available",
			FormatBytes(required), FormatBytes(projected), FormatBytes(available),
		)
		p.record("compact_skipped")
		p.cfg.Logger.WarnContext(ctx, "skipping retention compaction: insufficient free space",
			"required_bytes", required, "available_bytes", available, "database", p.cfg.Path)
		return true, reason, nil
	}

	if err := EnsureIncrementalAutoVacuum(ctx, p.cfg.DB, p.cfg.Logger); err != nil {
		return false, "", err
	}
	p.record("compact_full")
	return false, "", nil
}

// cutoffValue renders the age horizon in whatever representation the time column
// actually stores, so the comparison is meaningful for both an RFC3339 text
// column and an epoch integer column.
func (p *SQLiteTablePruner) cutoffValue(ctx context.Context, maxAge time.Duration) (any, error) {
	cutoff := p.cfg.Now().UTC().Add(-maxAge)

	var storedType string
	query := fmt.Sprintf(`SELECT typeof(%q) FROM %q LIMIT 1`, p.cfg.TimeColumn, p.cfg.Table)
	err := p.cfg.DB.QueryRowContext(ctx, query).Scan(&storedType)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		// An empty table has nothing to compare; any representation works.
		return cutoff.Format(time.RFC3339Nano), nil
	case err != nil:
		return nil, fmt.Errorf("probe type of %s.%s: %w", p.cfg.Table, p.cfg.TimeColumn, err)
	}

	switch storedType {
	case "integer", "real":
		return cutoff.Unix(), nil
	default:
		return cutoff.Format(time.RFC3339Nano), nil
	}
}

func isCancellation(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
