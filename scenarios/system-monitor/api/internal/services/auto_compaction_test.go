package services

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository/sqlite"
)

// legacyRepoWithFreePages builds a file database in auto_vacuum=NONE mode
// (as databases created before incremental mode are) holding free pages.
func legacyRepoWithFreePages(t *testing.T) (*sqlite.Repository, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "system-monitor.db")
	repo, err := sqlite.NewRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { repo.Close() })
	ctx := context.Background()
	db := repo.RoutedDB()
	if _, err := db.ExecContext(ctx, "PRAGMA auto_vacuum=NONE"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, "VACUUM"); err != nil {
		t.Fatal(err)
	}
	payload := strings.Repeat("x", 64*1024)
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 48; i++ {
		if _, err := db.ExecContext(ctx, "INSERT INTO metrics (cycle_id, collector_name, metric_data, observed_at) VALUES (?, 'device_graph', ?, ?)",
			fmt.Sprintf("c%d", i), payload, old.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repo.PruneCollectorMetricsBefore(ctx, "device_graph", old.Add(40*time.Minute)); err != nil {
		t.Fatal(err)
	}
	return repo, path
}

func testCompactor(repo repository.MaintenanceRepository, at time.Time, free uint64) *AutoCompactor {
	c := NewAutoCompactor(repo, AutoCompactionPolicy{WindowStartHour: 2, WindowEndHour: 5, MinFreeBytes: 1 << 20, MinFreeFraction: 0.2, SpaceMarginBytes: 1 << 20}, "")
	c.clock = NewStubClock(at)
	c.freeSpace = func(string) (uint64, error) { return free, nil }
	return c
}

func TestAutoCompactorConvertsLegacyDatabaseInsideQuietWindow(t *testing.T) {
	repo, _ := legacyRepoWithFreePages(t)
	c := testCompactor(repo, time.Date(2026, 10, 29, 3, 0, 0, 0, time.Local), 1<<40)

	receipt, err := c.MaybeCompact(context.Background())
	if err != nil {
		t.Fatalf("MaybeCompact: %v", err)
	}
	if receipt.Decision != CompactionPerformed || receipt.ReclaimedBytes <= 0 {
		t.Fatalf("receipt = %+v, want performed with reclaimed bytes", receipt)
	}
	if receipt.After == nil || receipt.After.AutoVacuum != repository.AutoVacuumIncremental {
		t.Fatalf("after = %+v, want incremental auto_vacuum", receipt.After)
	}
	// Once converted, later passes never rewrite the file again.
	again, err := c.MaybeCompact(context.Background())
	if err != nil || again.Decision != CompactionNotNeeded {
		t.Errorf("second pass = %+v (%v), want not_needed", again, err)
	}
}

func TestAutoCompactorDefersOutsideWindowAndWithoutDisk(t *testing.T) {
	repo, _ := legacyRepoWithFreePages(t)

	outside := testCompactor(repo, time.Date(2026, 10, 29, 14, 0, 0, 0, time.Local), 1<<40)
	if r, err := outside.MaybeCompact(context.Background()); err != nil || r.Decision != CompactionDeferred || !strings.Contains(r.Reason, "window") {
		t.Errorf("outside window = %+v (%v), want deferred for window", r, err)
	}
	noDisk := testCompactor(repo, time.Date(2026, 10, 29, 3, 0, 0, 0, time.Local), 1024)
	if r, err := noDisk.MaybeCompact(context.Background()); err != nil || r.Decision != CompactionDeferred || !strings.Contains(r.Reason, "insufficient disk") {
		t.Errorf("no disk = %+v (%v), want deferred for disk", r, err)
	}
	stats, err := repo.SQLiteStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.AutoVacuum == repository.AutoVacuumIncremental {
		t.Error("a deferred decision must not rewrite the database")
	}
}

func TestParseCompactionWindow(t *testing.T) {
	def := DefaultAutoCompactionPolicy()
	if p := ParseCompactionWindow("23-4", def); p.WindowStartHour != 23 || p.WindowEndHour != 4 || !inWindow(1, 23, 4) || inWindow(12, 23, 4) {
		t.Errorf("wrapping window parsed as %+v", p)
	}
	if p := ParseCompactionWindow("bogus", def); p != def {
		t.Errorf("invalid window must fall back to defaults, got %+v", p)
	}
}
