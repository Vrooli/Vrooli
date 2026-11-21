// Package dbmaintenance returns storage-manager's own freed SQLite pages to the
// filesystem.
//
// Retention deletes rows (audit trail, census reports, recovery ledger), but
// with auto_vacuum=NONE a deleted row's pages stay inside the file forever:
// on 2026-10-28 storage-manager.db was 2.7 GB with 0.6 GB on its freelist and
// would never shrink. This follows agent-manager's storagehealth precedent:
// one guarded conversion to incremental auto-vacuum, then bounded
// incremental_vacuum batches on every maintenance pass. No row is deleted
// here; deciding what to keep belongs to the retention owners.
package dbmaintenance

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/retention"
)

// Handle is the database surface a pass needs. The scenario passes its
// RoutedDB, so maintenance follows the same routing seam as every other query;
// conversion pins one connection because auto_vacuum and VACUUM are
// per-connection state.
type Handle interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	Conn(ctx context.Context) (*sql.Conn, error)
}

// Policy bounds one maintenance pass.
type Policy struct {
	// Free pages above FreeTriggerBytes start a drain; the drain stops at
	// FreeFloorBytes, which stays on the freelist as reuse slack.
	FreeTriggerBytes int64
	FreeFloorBytes   int64
	// BatchPages is one incremental_vacuum transaction; TickBudget bounds the
	// whole pass so the single shared connection is never held for long.
	BatchPages int64
	TickBudget time.Duration
	// SpaceMarginBytes is kept free beyond what the one-time conversion's
	// transient copy is estimated to use.
	SpaceMarginBytes int64
}

func DefaultPolicy() Policy {
	return Policy{FreeTriggerBytes: 64 << 20, FreeFloorBytes: 16 << 20, BatchPages: 1024, TickBudget: 5 * time.Second, SpaceMarginBytes: 1 << 30}
}

// Options are the seams a pass needs. FreeSpace and TempDir default to the
// host; tests substitute them.
type Options struct {
	Policy Policy
	// AllowConversion gates the one-time rewrite. The caller withholds it
	// while retention still has bulk work queued, so the rewrite copies the
	// already-pruned database rather than the bloated one.
	AllowConversion bool
	FreeSpace       func(dir string) (int64, error)
	TempDir         func() string
	Logger          *slog.Logger
	Clock           func() time.Time
}

// Stats is one cheap measurement: pragmas and file sizes, no page scan.
type Stats struct {
	Path       string `json:"path"`
	AutoVacuum string `json:"auto_vacuum"`
	PageSize   int64  `json:"page_size"`
	PageCount  int64  `json:"page_count"`
	FreePages  int64  `json:"free_pages"`
	FreeBytes  int64  `json:"free_bytes"`
	LiveBytes  int64  `json:"live_bytes"`
	WALBytes   int64  `json:"wal_bytes"`
}

// Result says what one pass did.
type Result struct {
	Before        Stats  `json:"before"`
	After         Stats  `json:"after"`
	Converted     bool   `json:"converted"`
	ReturnedPages int64  `json:"returned_pages"`
	Deferred      string `json:"deferred,omitempty"`
}

// ErrInsufficientSpace refuses the conversion rather than filling the disk
// that storage-manager exists to protect.
var ErrInsufficientSpace = errors.New("insufficient free space for incremental auto-vacuum conversion")

// Measure reads the file's page accounting.
func Measure(ctx context.Context, db Handle) (Stats, error) {
	var st Stats
	var mode int64
	for _, pragma := range []struct {
		name   string
		target *int64
	}{{"page_size", &st.PageSize}, {"page_count", &st.PageCount}, {"freelist_count", &st.FreePages}, {"auto_vacuum", &mode}} {
		if err := db.QueryRowContext(ctx, "PRAGMA "+pragma.name).Scan(pragma.target); err != nil {
			return Stats{}, fmt.Errorf("read %s: %w", pragma.name, err)
		}
	}
	st.AutoVacuum = autoVacuumName(mode)
	st.FreeBytes = st.FreePages * st.PageSize
	st.LiveBytes = (st.PageCount - st.FreePages) * st.PageSize
	path, err := mainDatabasePath(ctx, db)
	if err != nil {
		return Stats{}, err
	}
	st.Path = path
	if path != "" {
		if info, statErr := os.Stat(path + "-wal"); statErr == nil {
			st.WALBytes = info.Size()
		}
	}
	return st, nil
}

// Maintain runs one bounded pass: it converts a none-mode file once when
// allowed and space permits, and otherwise returns free pages above the floor
// in incremental_vacuum batches within the tick budget.
func Maintain(ctx context.Context, db Handle, opts Options) (Result, error) {
	opts = withDefaults(opts)
	before, err := Measure(ctx, db)
	if err != nil {
		return Result{}, err
	}
	result := Result{Before: before, After: before}
	if before.FreeBytes <= opts.Policy.FreeTriggerBytes {
		return result, nil
	}
	if before.AutoVacuum != "incremental" {
		if !opts.AllowConversion {
			result.Deferred = "conversion deferred until retention finishes its queued pruning"
			return result, nil
		}
		if err := checkSpace(before, opts); err != nil {
			result.Deferred = err.Error()
			return result, err
		}
		if err := convert(ctx, db, opts.Logger); err != nil {
			return result, err
		}
		result.Converted = true
	} else {
		returned, err := vacuumTo(ctx, db, opts)
		result.ReturnedPages = returned
		if err != nil {
			return result, err
		}
	}
	after, err := Measure(ctx, db)
	if err != nil {
		return result, err
	}
	result.After = after
	if result.Converted {
		result.ReturnedPages = before.PageCount - after.PageCount
	}
	return result, nil
}

// convert rewrites the file into incremental mode on one pinned connection
// (auto_vacuum and VACUUM are per-connection state), then truncates the WAL
// the rewrite passed through.
func convert(ctx context.Context, db Handle, logger *slog.Logger) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("pin conversion connection: %w", err)
	}
	defer conn.Close()
	if err := retention.EnsureIncrementalAutoVacuum(ctx, conn, logger); err != nil {
		return err
	}
	var busy, logFrames, checkpointed int64
	if err := conn.QueryRowContext(ctx, "PRAGMA wal_checkpoint(TRUNCATE)").Scan(&busy, &logFrames, &checkpointed); err != nil {
		return fmt.Errorf("checkpoint after conversion: %w", err)
	}
	return nil
}

// checkSpace requires room for VACUUM's transient copy in SQLite's temp
// directory plus the rewritten pages passing through the WAL, each with a
// margin; both land on one filesystem when the directories share a device.
func checkSpace(st Stats, opts Options) error {
	margin := opts.Policy.SpaceMarginBytes
	dbDir := filepath.Dir(st.Path)
	tempDir := opts.TempDir()
	needDB := st.LiveBytes + st.WALBytes + margin
	needTemp := st.LiveBytes + margin
	if sameFilesystem(dbDir, tempDir) {
		needDB += st.LiveBytes
		needTemp = 0
	}
	for _, check := range []struct {
		dir  string
		need int64
	}{{dbDir, needDB}, {tempDir, needTemp}} {
		if check.need <= 0 {
			continue
		}
		free, err := opts.FreeSpace(check.dir)
		if err != nil {
			return fmt.Errorf("%w: measure %s: %v", ErrInsufficientSpace, check.dir, err)
		}
		if free < check.need {
			return fmt.Errorf("%w: %s has %d bytes free, conversion needs %d", ErrInsufficientSpace, check.dir, free, check.need)
		}
	}
	return nil
}

// vacuumTo returns free pages in BatchPages transactions until at most the
// floor remains free or the tick budget ends.
func vacuumTo(ctx context.Context, db Handle, opts Options) (int64, error) {
	deadline := opts.Clock().Add(opts.Policy.TickBudget)
	var returned int64
	for opts.Clock().Before(deadline) {
		var pageSize, free int64
		if err := db.QueryRowContext(ctx, "PRAGMA page_size").Scan(&pageSize); err != nil {
			return returned, err
		}
		if err := db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&free); err != nil {
			return returned, err
		}
		excess := free - opts.Policy.FreeFloorBytes/max(pageSize, 1)
		if excess <= 0 {
			return returned, nil
		}
		if err := incrementalVacuum(ctx, db, min(excess, opts.Policy.BatchPages)); err != nil {
			return returned, fmt.Errorf("incremental vacuum: %w", err)
		}
		var after int64
		if err := db.QueryRowContext(ctx, "PRAGMA freelist_count").Scan(&after); err != nil {
			return returned, err
		}
		if after >= free {
			return returned, nil
		}
		returned += free - after
	}
	return returned, nil
}

// incrementalVacuum steps every result row: the pragma moves one page per row
// stepped, so a caller that does not drain the cursor returns a single page.
func incrementalVacuum(ctx context.Context, db Handle, pages int64) error {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA incremental_vacuum(%d)", pages))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
	}
	return rows.Err()
}

func mainDatabasePath(ctx context.Context, db Handle) (string, error) {
	rows, err := db.QueryContext(ctx, "PRAGMA database_list")
	if err != nil {
		return "", fmt.Errorf("read database_list: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var seq int64
		var name, file string
		if err := rows.Scan(&seq, &name, &file); err != nil {
			return "", fmt.Errorf("scan database_list: %w", err)
		}
		if name == "main" {
			return file, nil
		}
	}
	return "", rows.Err()
}

func withDefaults(opts Options) Options {
	if opts.Policy == (Policy{}) {
		opts.Policy = DefaultPolicy()
	}
	if opts.FreeSpace == nil {
		opts.FreeSpace = retention.FreeSpace
	}
	if opts.TempDir == nil {
		opts.TempDir = sqliteTempDir
	}
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	return opts
}

// sqliteTempDir mirrors SQLite's unix temporary-directory search, which is
// where VACUUM writes its transient copy.
func sqliteTempDir() string {
	for _, key := range []string{"SQLITE_TMPDIR", "TMPDIR"} {
		if dir := strings.TrimSpace(os.Getenv(key)); dir != "" && isDir(dir) {
			return dir
		}
	}
	for _, dir := range []string{"/var/tmp", "/usr/tmp", "/tmp"} {
		if isDir(dir) {
			return dir
		}
	}
	return "."
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func autoVacuumName(mode int64) string {
	switch mode {
	case 1:
		return "full"
	case 2:
		return "incremental"
	default:
		return "none"
	}
}
