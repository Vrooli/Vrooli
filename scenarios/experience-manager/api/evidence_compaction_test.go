package main

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/vrooli/api-core/storage"
)

func legacyDBWithFreePages(t *testing.T) (*sql.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "experience-manager.db")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	db.SetMaxOpenConns(1)
	for _, stmt := range []string{"PRAGMA auto_vacuum=NONE", "PRAGMA journal_mode=WAL", "CREATE TABLE blobs (id INTEGER PRIMARY KEY, v TEXT)"} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	payload := strings.Repeat("x", 64*1024)
	for i := 0; i < 40; i++ {
		if _, err := db.Exec("INSERT INTO blobs (v) VALUES (?)", payload); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("DELETE FROM blobs WHERE id > 4"); err != nil {
		t.Fatal(err)
	}
	return db, path
}

func newTestCompactor(db *sql.DB, path string, at time.Time) *evidenceCompactor {
	return &evidenceCompactor{
		db: func() *sql.DB { return db }, path: path, now: func() time.Time { return at },
		freeSpace: func(string) (uint64, error) { return 1 << 40, nil }, run: &sync.Mutex{},
		policy: evidenceCompactionPolicy{WindowStartHour: 2, WindowEndHour: 5, MinFreeBytes: 1 << 20, MinFreeFraction: 0.2, SpaceMarginBytes: 1 << 20, MaxBacklog: 1000},
	}
}

func TestEvidenceCompactorConvertsOnlyAfterBacklogDrains(t *testing.T) {
	db, path := legacyDBWithFreePages(t)
	c := newTestCompactor(db, path, time.Date(2026, 10, 29, 3, 0, 0, 0, time.Local))
	ctx := context.Background()

	if r, err := c.MaybeCompact(ctx); err != nil || r.Decision != compactionDeferred || !strings.Contains(r.Reason, "backlog unknown") {
		t.Fatalf("before any retention cycle = %+v (%v), want deferred", r, err)
	}
	c.recordRetention(500000, nil)
	if r, err := c.MaybeCompact(ctx); err != nil || r.Decision != compactionDeferred || !strings.Contains(r.Reason, "drain") {
		t.Fatalf("with backlog = %+v (%v), want deferred until drained", r, err)
	}
	c.recordRetention(12, nil)
	r, err := c.MaybeCompact(ctx)
	if err != nil || r.Decision != compactionPerformed {
		t.Fatalf("drained backlog = %+v (%v), want performed", r, err)
	}
	if r.After == nil || r.After.AutoVacuum != 2 || r.ReclaimedBytes <= 0 {
		t.Fatalf("after = %+v reclaimed=%d, want incremental and a smaller file", r.After, r.ReclaimedBytes)
	}
	if again, _ := c.MaybeCompact(ctx); again.Decision != compactionNotNeeded {
		t.Errorf("second pass = %+v, want not_needed", again)
	}
}

func TestEvidenceCompactorDefersOutsideWindowAndWhileRetentionRuns(t *testing.T) {
	db, path := legacyDBWithFreePages(t)
	ctx := context.Background()
	outside := newTestCompactor(db, path, time.Date(2026, 10, 29, 15, 0, 0, 0, time.Local))
	outside.recordRetention(0, nil)
	if r, _ := outside.MaybeCompact(ctx); r.Decision != compactionDeferred || !strings.Contains(r.Reason, "window") {
		t.Errorf("outside window = %+v, want deferred", r)
	}
	busy := newTestCompactor(db, path, time.Date(2026, 10, 29, 3, 0, 0, 0, time.Local))
	busy.recordRetention(0, nil)
	busy.run.Lock()
	defer busy.run.Unlock()
	if r, _ := busy.MaybeCompact(ctx); r.Decision != compactionDeferred || !strings.Contains(r.Reason, "retention is running") {
		t.Errorf("during retention = %+v, want deferred", r)
	}
}

type failingPinger struct{}

func (failingPinger) PingContext(context.Context) error { return errors.New("database busy") }

func TestMaintenancePingerAnswersDuringConversionOnly(t *testing.T) {
	c := &evidenceCompactor{}
	p := maintenancePinger{inner: failingPinger{}, compactor: c}
	if err := p.PingContext(context.Background()); err == nil {
		t.Error("outside compaction the real ping result must be reported")
	}
	c.active.Store(true)
	if err := p.PingContext(context.Background()); err != nil {
		t.Errorf("during compaction ping = %v, want nil", err)
	}
	if err := (maintenancePinger{inner: failingPinger{}}).PingContext(context.Background()); err == nil {
		t.Error("a nil compactor must fall through to the real ping")
	}
}

func TestEnsureIncrementalOnNewDatabaseConvertsFreshFileOnly(t *testing.T) {
	ctx := context.Background()
	fresh, _ := legacyDBWithFreePages(t) // 4 live rows; small enough to convert at once
	if _, err := fresh.Exec("DELETE FROM blobs"); err != nil {
		t.Fatal(err)
	}
	if _, err := fresh.Exec("VACUUM"); err != nil {
		t.Fatal(err)
	}
	if err := ensureIncrementalOnNewDatabase(ctx, fresh); err != nil {
		t.Fatal(err)
	}
	var mode int
	if err := fresh.QueryRow("PRAGMA auto_vacuum").Scan(&mode); err != nil || mode != 2 {
		t.Errorf("fresh database auto_vacuum = %d (%v), want 2", mode, err)
	}
}
