package sqlite

import (
	"context"
	"testing"
	"time"
)

// insertMetricAt inserts a metric row with an explicit timestamp, bypassing
// SaveMetrics (which always stamps time.Now) so retention windows are testable.
func insertMetricAt(t *testing.T, repo *Repository, collector string, ts time.Time, payload string) {
	t.Helper()
	if _, err := repo.db.ExecContext(
		context.Background(),
		"INSERT INTO metrics (collector_name, metric_data, timestamp) VALUES (?, ?, ?)",
		collector, payload, ts.UTC(),
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
