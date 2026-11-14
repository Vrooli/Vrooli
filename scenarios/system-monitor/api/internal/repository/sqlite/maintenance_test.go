package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// insertMetricAt inserts a metric row with an explicit observation time so
// retention windows are testable.
func insertMetricAt(t *testing.T, repo *Repository, collector string, ts time.Time, payload string) {
	t.Helper()
	if _, err := repo.db.ExecContext(
		context.Background(),
		"INSERT INTO metrics (cycle_id, collector_name, metric_data, observed_at) VALUES (?, ?, ?, ?)",
		collector+ts.UTC().Format(time.RFC3339Nano), collector, payload, ts.UTC(),
	); err != nil {
		t.Fatalf("insert metric: %v", err)
	}
}

func TestEstimateMetricRetention(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	insertMetricAt(t, repo, "cpu", now.Add(-40*24*time.Hour), `{"usage_percent":1}`)
	insertMetricAt(t, repo, "cpu", now.Add(-35*24*time.Hour), `{"usage_percent":2}`)
	insertMetricAt(t, repo, "cpu", now.Add(-1*24*time.Hour), `{"usage_percent":3}`)

	cutoff := now.Add(-30 * 24 * time.Hour)
	est, err := repo.EstimateMetricRetention(ctx, cutoff)
	if err != nil {
		t.Fatalf("EstimateMetricRetention: %v", err)
	}
	if est.RowCount != 2 {
		t.Errorf("RowCount = %d, want 2", est.RowCount)
	}
	if est.PayloadBytes <= 0 {
		t.Errorf("PayloadBytes = %d, want > 0", est.PayloadBytes)
	}
	if est.OldestAffected.IsZero() || !est.OldestAffected.Equal(now.Add(-40*24*time.Hour)) {
		t.Errorf("OldestAffected = %v, want %v", est.OldestAffected, now.Add(-40*24*time.Hour))
	}
}

func TestPruneMetricsBefore(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	insertMetricAt(t, repo, "cpu", now.Add(-40*24*time.Hour), `{"usage_percent":1}`)
	insertMetricAt(t, repo, "cpu", now.Add(-1*24*time.Hour), `{"usage_percent":2}`)

	cutoff := now.Add(-30 * 24 * time.Hour)
	res, err := repo.PruneMetricsBefore(ctx, cutoff)
	if err != nil {
		t.Fatalf("PruneMetricsBefore: %v", err)
	}
	if res.DeletedRows != 1 {
		t.Errorf("DeletedRows = %d, want 1", res.DeletedRows)
	}

	// The recent row must survive.
	stats, err := repo.SQLiteStats(ctx)
	if err != nil {
		t.Fatalf("SQLiteStats: %v", err)
	}
	if stats.MetricRows != 1 {
		t.Errorf("MetricRows after prune = %d, want 1", stats.MetricRows)
	}
}

func TestSQLiteStats(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	stats, err := repo.SQLiteStats(ctx)
	if err != nil {
		t.Fatalf("SQLiteStats: %v", err)
	}
	if stats.PageSize <= 0 {
		t.Errorf("PageSize = %d, want > 0", stats.PageSize)
	}
	if stats.SizeBytes != stats.PageSize*stats.PageCount {
		t.Errorf("SizeBytes = %d, want PageSize*PageCount = %d", stats.SizeBytes, stats.PageSize*stats.PageCount)
	}
}

func TestCompactReportsStats(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	// Insert and delete many rows to build up freelist pages, then compact.
	for i := 0; i < 200; i++ {
		insertMetricAt(t, repo, "cpu", now.Add(-time.Duration(i)*time.Hour), `{"usage_percent":1,"padding":"xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"}`)
	}
	if _, err := repo.PruneMetricsBefore(ctx, now); err != nil {
		t.Fatalf("PruneMetricsBefore: %v", err)
	}

	res, err := repo.Compact(ctx)
	if err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if res.StatsAfter.FreelistCount > res.StatsBefore.FreelistCount {
		t.Errorf("freelist grew after compaction: before=%d after=%d", res.StatsBefore.FreelistCount, res.StatsAfter.FreelistCount)
	}
}

func TestPruneEmptyTableNoError(t *testing.T) {
	repo := newTestRepo(t)
	res, err := repo.PruneMetricsBefore(context.Background(), time.Now())
	if err != nil {
		t.Fatalf("PruneMetricsBefore on empty table: %v", err)
	}
	if res.DeletedRows != 0 {
		t.Errorf("DeletedRows = %d, want 0", res.DeletedRows)
	}
}

func TestPruneCollectorMetricsBeforeOnlyTouchesThatCollector(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	insertMetricAt(t, repo, "device_graph", now.Add(-3*24*time.Hour), `{"nodes":[]}`)
	insertMetricAt(t, repo, "device_graph", now.Add(-1*time.Hour), `{"nodes":[]}`)
	insertMetricAt(t, repo, "cpu", now.Add(-3*24*time.Hour), `{"usage_percent":1}`)

	res, err := repo.PruneCollectorMetricsBefore(ctx, "device_graph", now.Add(-48*time.Hour))
	if err != nil {
		t.Fatalf("PruneCollectorMetricsBefore: %v", err)
	}
	if res.DeletedRows != 1 {
		t.Errorf("DeletedRows = %d, want 1 (only the old device_graph row)", res.DeletedRows)
	}
	var cpu, graph int
	if err := repo.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM metrics WHERE collector_name = 'cpu'").Scan(&cpu); err != nil {
		t.Fatal(err)
	}
	if err := repo.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM metrics WHERE collector_name = 'device_graph'").Scan(&graph); err != nil {
		t.Fatal(err)
	}
	if cpu != 1 || graph != 1 {
		t.Errorf("remaining cpu=%d device_graph=%d, want 1 and 1", cpu, graph)
	}
}

func TestNewDatabaseReleasesPrunedPagesToFilesystem(t *testing.T) {
	repo, err := NewRepository(filepath.Join(t.TempDir(), "sm.db"))
	if err != nil {
		t.Fatalf("NewRepository: %v", err)
	}
	t.Cleanup(func() { repo.Close() })
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	var mode int
	if err := repo.db.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != 2 {
		t.Fatalf("auto_vacuum = %d, want 2 (INCREMENTAL) on a new database", mode)
	}

	payload := strings.Repeat("x", 64*1024)
	for i := 0; i < 64; i++ {
		insertMetricAt(t, repo, "device_graph", now.Add(-time.Duration(72+i)*time.Hour), payload)
	}
	full, err := repo.SQLiteStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.PruneCollectorMetricsBefore(ctx, "device_graph", now.Add(-48*time.Hour)); err != nil {
		t.Fatal(err)
	}
	released, err := repo.ReleaseFreePages(ctx, 16)
	if err != nil {
		t.Fatalf("ReleaseFreePages: %v", err)
	}
	after, err := repo.SQLiteStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if released < int64(len(payload))*32 || after.SizeBytes >= full.SizeBytes/2 {
		t.Errorf("released=%d size %d -> %d; want the pruned payload returned to the filesystem", released, full.SizeBytes, after.SizeBytes)
	}
	if after.FreelistCount != 0 {
		t.Errorf("FreelistCount = %d after release, want 0", after.FreelistCount)
	}
}

func TestPruneCollectorMetricsBeforeDrainsBacklogAcrossBatches(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	backlog := collectorPruneBatch*2 + 7
	for i := 0; i < backlog; i++ {
		insertMetricAt(t, repo, "device_graph", now.Add(-72*time.Hour-time.Duration(i)*time.Second), `{}`)
	}
	res, err := repo.PruneCollectorMetricsBefore(ctx, "device_graph", now.Add(-48*time.Hour))
	if err != nil {
		t.Fatalf("PruneCollectorMetricsBefore: %v", err)
	}
	if res.DeletedRows != int64(backlog) {
		t.Errorf("DeletedRows = %d, want %d", res.DeletedRows, backlog)
	}
}

func TestCompactConvertsLegacyDatabaseToIncrementalAutoVacuum(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	repo, err := NewRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// Simulate a database created before incremental mode.
	if _, err := repo.db.ExecContext(ctx, "PRAGMA auto_vacuum=NONE"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.db.ExecContext(ctx, "VACUUM"); err != nil {
		t.Fatal(err)
	}
	var mode int
	if err := repo.db.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&mode); err != nil || mode != 0 {
		t.Fatalf("setup: auto_vacuum = %d (%v), want 0", mode, err)
	}
	repo.Close()

	reopened, err := NewRepository(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reopened.Close() })
	if _, err := reopened.Compact(ctx); err != nil {
		t.Fatalf("Compact: %v", err)
	}
	if err := reopened.db.QueryRowContext(ctx, "PRAGMA auto_vacuum").Scan(&mode); err != nil {
		t.Fatal(err)
	}
	if mode != 2 {
		t.Errorf("auto_vacuum after Compact = %d, want 2 (INCREMENTAL)", mode)
	}
}

func TestReadsFailFastWhileCompacting(t *testing.T) {
	repo := newTestRepo(t)
	repo.compacting.Store(true)
	defer repo.compacting.Store(false)
	// Hold the write lock as Compact does; a read that queued behind it would
	// block this test forever.
	repo.mu.Lock()
	defer repo.mu.Unlock()
	if _, err := repo.GetLatestMetrics(context.Background()); !errors.Is(err, repository.ErrMaintenance) {
		t.Errorf("GetLatestMetrics during compaction = %v, want ErrMaintenance", err)
	}
	if _, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{}); !errors.Is(err, repository.ErrMaintenance) {
		t.Errorf("GetMetrics during compaction = %v, want ErrMaintenance", err)
	}
}
