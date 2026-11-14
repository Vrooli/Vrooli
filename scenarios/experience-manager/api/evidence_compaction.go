package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	emdb "experience-manager/internal/database"

	"github.com/vrooli/api-core/storage"
)

// The evidence database predates incremental auto-vacuum, so pages freed by
// evidence retention stay in the file until one full VACUUM rewrites it into
// auto_vacuum=INCREMENTAL mode. After that, retention returns pages online in
// bounded batches and this rewrite is never needed again. The owner performs
// it itself once the retention backlog has drained, inside a quiet window,
// with enough disk for the rewrite, while health keeps answering.

// evidenceCompactionPolicy gates the one-time conversion.
type evidenceCompactionPolicy struct {
	WindowStartHour  int
	WindowEndHour    int
	MinFreeBytes     int64
	MinFreeFraction  float64
	SpaceMarginBytes int64
	// MaxBacklog is the most superseded candidates the last retention cycle
	// may have reported; rewriting before the backlog drains would copy rows
	// that are about to be deleted.
	MaxBacklog int
}

func defaultEvidenceCompactionPolicy() evidenceCompactionPolicy {
	return evidenceCompactionPolicy{WindowStartHour: 2, WindowEndHour: 5, MinFreeBytes: 1 << 30, MinFreeFraction: 0.20, SpaceMarginBytes: 4 << 30, MaxBacklog: 1000}
}

type dbFootprint struct {
	PageSize      int64 `json:"page_size"`
	PageCount     int64 `json:"page_count"`
	FreelistCount int64 `json:"freelist_count"`
	AutoVacuum    int64 `json:"auto_vacuum"`
	FileBytes     int64 `json:"file_bytes"`
	WALBytes      int64 `json:"wal_bytes"`
}

func (f dbFootprint) freeBytes() int64 { return f.FreelistCount * f.PageSize }
func (f dbFootprint) liveBytes() int64 { return (f.PageCount - f.FreelistCount) * f.PageSize }

type evidenceCompactionReceipt struct {
	DecidedAt      time.Time    `json:"decided_at"`
	Decision       string       `json:"decision"`
	Reason         string       `json:"reason"`
	Before         dbFootprint  `json:"before"`
	After          *dbFootprint `json:"after,omitempty"`
	ReclaimedBytes int64        `json:"reclaimed_bytes,omitempty"`
	FreeDiskBytes  uint64       `json:"free_disk_bytes,omitempty"`
	NeedDiskBytes  int64        `json:"need_disk_bytes,omitempty"`
	DurationMillis int64        `json:"duration_millis,omitempty"`
	Error          string       `json:"error,omitempty"`
}

const (
	compactionNotNeeded = "not_needed"
	compactionDeferred  = "deferred"
	compactionPerformed = "performed"
	compactionFailed    = "failed"
)

// evidenceCompactor owns the conversion. Its mutex is shared with the
// retention cycle so the two never run together.
type evidenceCompactor struct {
	db          func() *sql.DB
	path        string
	policy      evidenceCompactionPolicy
	now         func() time.Time
	freeSpace   func(dir string) (uint64, error)
	receiptPath string

	run    *sync.Mutex
	active atomic.Bool

	mu          sync.Mutex
	lastBacklog int
	backlogSeen bool
}

// recordRetention feeds the drained-backlog gate from each retention cycle.
func (c *evidenceCompactor) recordRetention(candidates int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err != nil {
		c.backlogSeen = false
		return
	}
	c.lastBacklog, c.backlogSeen = candidates, true
}

// Active reports whether the conversion VACUUM is running.
func (c *evidenceCompactor) Active() bool { return c != nil && c.active.Load() }

func (c *evidenceCompactor) footprint(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
},
) (dbFootprint, error) {
	var f dbFootprint
	for _, p := range []struct {
		pragma string
		dest   *int64
	}{{"page_size", &f.PageSize}, {"page_count", &f.PageCount}, {"freelist_count", &f.FreelistCount}, {"auto_vacuum", &f.AutoVacuum}} {
		if err := q.QueryRowContext(ctx, "PRAGMA "+p.pragma).Scan(p.dest); err != nil {
			return f, fmt.Errorf("PRAGMA %s: %w", p.pragma, err)
		}
	}
	if info, err := os.Stat(c.path); err == nil {
		f.FileBytes = info.Size()
	}
	if info, err := os.Stat(c.path + "-wal"); err == nil {
		f.WALBytes = info.Size()
	}
	return f, nil
}

// MaybeCompact evaluates every gate and performs the conversion when all hold.
// Only performed and failed outcomes are persisted.
func (c *evidenceCompactor) MaybeCompact(ctx context.Context) (evidenceCompactionReceipt, error) {
	now := c.now()
	receipt := evidenceCompactionReceipt{DecidedAt: now.UTC()}
	db := c.db()
	if db == nil {
		return receipt, errors.New("database pool is closed")
	}
	before, err := c.footprint(ctx, db)
	if err != nil {
		return receipt, err
	}
	receipt.Before = before

	c.mu.Lock()
	backlog, seen := c.lastBacklog, c.backlogSeen
	c.mu.Unlock()
	free := before.freeBytes()
	switch {
	case before.AutoVacuum == 2:
		receipt.Decision, receipt.Reason = compactionNotNeeded, "already incremental; retention returns free pages online"
		return receipt, nil
	case !seen:
		receipt.Decision, receipt.Reason = compactionDeferred, "no successful retention cycle yet; backlog unknown"
		return receipt, nil
	case backlog > c.policy.MaxBacklog:
		receipt.Decision, receipt.Reason = compactionDeferred, fmt.Sprintf("retention backlog %d candidates exceeds %d; wait for it to drain", backlog, c.policy.MaxBacklog)
		return receipt, nil
	case free < c.policy.MinFreeBytes || float64(free) < c.policy.MinFreeFraction*float64(before.PageCount*before.PageSize):
		receipt.Decision, receipt.Reason = compactionDeferred, fmt.Sprintf("only %d free bytes; below the rewrite threshold", free)
		return receipt, nil
	case !hourInWindow(now.Hour(), c.policy.WindowStartHour, c.policy.WindowEndHour):
		receipt.Decision, receipt.Reason = compactionDeferred, fmt.Sprintf("outside the quiet window %02d:00-%02d:00", c.policy.WindowStartHour, c.policy.WindowEndHour)
		return receipt, nil
	}
	// VACUUM writes a temp copy of the live pages (temp_store=FILE) and
	// passes the rewritten database through the WAL.
	need := 2*before.liveBytes() + before.WALBytes + c.policy.SpaceMarginBytes
	receipt.NeedDiskBytes = need
	diskFree, err := c.freeSpace(filepath.Dir(c.path))
	if err != nil {
		receipt.Decision, receipt.Reason = compactionDeferred, "cannot measure free disk: "+err.Error()
		return receipt, nil
	}
	receipt.FreeDiskBytes = diskFree
	if int64(diskFree) < need {
		receipt.Decision, receipt.Reason = compactionDeferred, fmt.Sprintf("insufficient disk: %d free, need %d", diskFree, need)
		return receipt, nil
	}

	if !c.run.TryLock() {
		receipt.Decision, receipt.Reason = compactionDeferred, "evidence retention is running"
		return receipt, nil
	}
	defer c.run.Unlock()

	start := c.now()
	after, err := c.convert(ctx, db)
	receipt.DurationMillis = c.now().Sub(start).Milliseconds()
	if err != nil {
		receipt.Decision, receipt.Error = compactionFailed, err.Error()
		c.persist(receipt)
		return receipt, err
	}
	receipt.After = &after
	receipt.ReclaimedBytes = (before.PageCount - after.PageCount) * before.PageSize
	receipt.Decision, receipt.Reason = compactionPerformed, "converted to auto_vacuum=INCREMENTAL"
	c.persist(receipt)
	return receipt, nil
}

// convert runs the rewrite on one pinned connection: auto_vacuum, temp_store
// and VACUUM are per-connection state.
func (c *evidenceCompactor) convert(ctx context.Context, db *sql.DB) (dbFootprint, error) {
	conn, err := db.Conn(ctx)
	if err != nil {
		return dbFootprint{}, fmt.Errorf("pin compaction connection: %w", err)
	}
	defer conn.Close()

	c.active.Store(true)
	defer c.active.Store(false)
	for _, stmt := range []string{"PRAGMA temp_store=FILE", "PRAGMA auto_vacuum=INCREMENTAL", "VACUUM", "PRAGMA wal_checkpoint(TRUNCATE)"} {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return dbFootprint{}, fmt.Errorf("%s: %w", stmt, err)
		}
	}
	after, err := c.footprint(ctx, conn)
	if err != nil {
		return after, err
	}
	if after.AutoVacuum != 2 {
		return after, fmt.Errorf("auto_vacuum is %d after compaction, want 2 (INCREMENTAL)", after.AutoVacuum)
	}
	return after, nil
}

func (c *evidenceCompactor) persist(receipt evidenceCompactionReceipt) {
	if c.receiptPath == "" {
		return
	}
	data, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		return
	}
	_ = storage.WriteFileAtomic(c.receiptPath, append(data, '\n'), 0o644)
}

func hourInWindow(hour, start, end int) bool {
	if start == end {
		return true
	}
	if start < end {
		return hour >= start && hour < end
	}
	return hour >= start || hour < end
}

func parseCompactionWindow(raw string, p evidenceCompactionPolicy) evidenceCompactionPolicy {
	var start, end int
	if _, err := fmt.Sscanf(raw, "%d-%d", &start, &end); err != nil || start < 0 || start > 23 || end < 0 || end > 23 {
		return p
	}
	p.WindowStartHour, p.WindowEndHour = start, end
	return p
}

// maintenancePinger keeps the health check answering while the owner's
// conversion VACUUM holds the single database connection. Outside that
// window it pings the database as before.
type maintenancePinger struct {
	inner     emdb.Pinger
	compactor *evidenceCompactor
}

func (p maintenancePinger) PingContext(ctx context.Context) error {
	if p.compactor.Active() {
		return nil
	}
	return p.inner.PingContext(ctx)
}

// newDatabasePageLimit is the size below which converting a database with an
// immediate VACUUM is effectively free (a fresh or nearly empty file).
const newDatabasePageLimit = 256

// ensureIncrementalOnNewDatabase makes a fresh database incremental at once.
// The connection DSN already put the file in WAL mode, which initializes the
// header, so the auto_vacuum pragma alone stays pending until a VACUUM. On a
// tiny file that VACUUM is instant; a large legacy file is left to the gated
// owner conversion instead of rewriting it during startup.
func ensureIncrementalOnNewDatabase(ctx context.Context, db *sql.DB) error {
	conn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()
	var mode, pages int64
	if err := conn.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&mode); err != nil {
		return err
	}
	if mode == 2 {
		return nil
	}
	if _, err := conn.ExecContext(ctx, "PRAGMA auto_vacuum=INCREMENTAL"); err != nil {
		return err
	}
	if err := conn.QueryRowContext(ctx, "PRAGMA page_count").Scan(&pages); err != nil {
		return err
	}
	if pages > newDatabasePageLimit {
		return nil
	}
	_, err = conn.ExecContext(ctx, "VACUUM")
	return err
}
