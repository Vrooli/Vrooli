package persistence

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// explainPlan returns the concatenated EXPLAIN QUERY PLAN output for a query.
// Used to assert that rewritten read queries use the expected indexes; these
// tests are the only thing that catches a future regression where someone
// wraps `created_at` in `datetime(...)` and silently defeats the indexes.
func explainPlan(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("EXPLAIN QUERY PLAN error: %v\nquery: %s", err, query)
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var id, parent, notused int
		var detail string
		if err := rows.Scan(&id, &parent, &notused, &detail); err != nil {
			t.Fatalf("scan EXPLAIN row: %v", err)
		}
		lines = append(lines, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows error: %v", err)
	}
	return strings.Join(lines, "\n")
}

func assertPlanUsesIndex(t *testing.T, plan, expectedIndex string) {
	t.Helper()
	if !strings.Contains(plan, expectedIndex) {
		t.Fatalf("expected plan to reference %q, got:\n%s", expectedIndex, plan)
	}
	if strings.Contains(plan, "SCAN health_results") && !strings.Contains(plan, "USING INDEX") && !strings.Contains(plan, "USING COVERING INDEX") {
		t.Fatalf("plan does a full table scan on health_results:\n%s", plan)
	}
}

func TestPlanner_GetRecentResults_UsesCheckIDIndex(t *testing.T) {
	db := openSQLiteTestDB(t)
	seedHealthResults(t, db, time.Now().UTC())

	plan := explainPlan(t, db, `
		SELECT check_id, status, message, details, duration_ms, created_at
		FROM health_results
		WHERE check_id = ?
		ORDER BY created_at DESC
		LIMIT ?
	`, "infra-network", 20)

	assertPlanUsesIndex(t, plan, "idx_health_results_check_id_created")
}

func TestPlanner_GetTimelineEvents_UsesCreatedAtIndex(t *testing.T) {
	db := openSQLiteTestDB(t)
	seedHealthResults(t, db, time.Now().UTC())

	plan := explainPlan(t, db, `
		SELECT check_id, status, message, details, created_at
		FROM health_results
		ORDER BY created_at DESC
		LIMIT ?
	`, 50)

	assertPlanUsesIndex(t, plan, "idx_health_results_created_at")
}

func TestPlanner_GetUptimeStats_UsesCreatedAtIndex(t *testing.T) {
	db := openSQLiteTestDB(t)
	seedHealthResults(t, db, time.Now().UTC())

	cutoff := rfc3339NanoCutoff(time.Now(), 24)
	plan := explainPlan(t, db, `
		SELECT COUNT(*), 0, 0, 0
		FROM health_results
		WHERE created_at >= ?
	`, cutoff)

	assertPlanUsesIndex(t, plan, "idx_health_results_created_at")
}

func TestPlanner_GetUptimeHistory_UsesCreatedAtIndex(t *testing.T) {
	db := openSQLiteTestDB(t)
	seedHealthResults(t, db, time.Now().UTC())

	now := time.Now().UTC()
	start := now.Add(-24 * time.Hour)
	plan := explainPlan(t, db, `
		SELECT
			CAST((CAST(strftime('%s', created_at) AS INTEGER) - ?) / ? AS INTEGER) AS bucket,
			status,
			COUNT(*)
		FROM health_results
		WHERE created_at >= ? AND created_at <= ?
		GROUP BY bucket, status
	`, start.Unix(), int64(3600), start.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))

	assertPlanUsesIndex(t, plan, "idx_health_results_created_at")
}

func TestPlanner_CleanupOldResults_UsesCreatedAtIndex(t *testing.T) {
	db := openSQLiteTestDB(t)
	seedHealthResults(t, db, time.Now().UTC())

	cutoff := rfc3339NanoCutoff(time.Now(), 24)
	plan := explainPlan(t, db, `
		DELETE FROM health_results WHERE created_at < ?
	`, cutoff)

	if !strings.Contains(plan, "idx_health_results_created_at") &&
		!strings.Contains(plan, "USING INDEX") {
		t.Fatalf("delete plan should use an index on created_at:\n%s", plan)
	}
}

// TestNoDatetimeWrappersInQueries is a defense-in-depth source-text check: a
// future contributor must not reintroduce datetime(created_at) in the SQLite
// store. This catches the regression even if EXPLAIN tests for that specific
// query are missing.
func TestNoDatetimeWrappersInQueries(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	srcBytes, err := os.ReadFile(filepath.Join(wd, "store_sqlite.go"))
	if err != nil {
		t.Fatalf("read store_sqlite.go: %v", err)
	}
	src := string(srcBytes)
	// Scan only inside backtick-delimited SQL strings — comments may legitimately mention
	// datetime() to explain the historical mistake.
	for i := 0; i < len(src); i++ {
		if src[i] != '`' {
			continue
		}
		end := strings.Index(src[i+1:], "`")
		if end < 0 {
			break
		}
		segment := src[i+1 : i+1+end]
		if strings.Contains(segment, "datetime(created_at") {
			t.Fatalf("found datetime(created_at...) in SQL — defeats indexes:\n%s", segment)
		}
		i += end + 1
	}
}

// Equivalence test: the rewritten getUptimeHistorySQLite returns the same
// bucket count and status totals as a hand-computed expectation on a known
// fixture. The semantic shifted from "snapshot at boundary" to "events per
// slice", so we don't compare against the old implementation byte-for-byte;
// we assert the new contract directly.
func TestGetUptimeHistory_BucketAggregationMatchesFixture(t *testing.T) {
	db := openSQLiteTestDB(t)
	store := NewStore(db)

	now := time.Now().UTC()
	seedHealthResults(t, db, now)

	history, err := store.GetUptimeHistory(context.Background(), 1, 4) // 1h window, 4 × 15min buckets
	if err != nil {
		t.Fatalf("GetUptimeHistory error: %v", err)
	}
	if len(history.Buckets) != 4 {
		t.Fatalf("bucket count = %d, want 4", len(history.Buckets))
	}
	// Seed has 6 events, all within the past 25 minutes → all should fall in
	// the last two buckets (and the totals must match).
	totals := 0
	for _, b := range history.Buckets {
		totals += b.Total
	}
	if totals != 6 {
		t.Fatalf("sum of bucket totals = %d, want 6 (matches seed count)", totals)
	}
	if history.Overall.TotalEvents != 6 {
		t.Fatalf("overall TotalEvents = %d, want 6", history.Overall.TotalEvents)
	}
}

// Benchmark establishes the order-of-magnitude shape of the rewritten
// queries. The previous N+1 implementation scaled with bucket × distinct-check
// count and full-table scans; this should be roughly constant on populated
// databases.
func benchSeed(b *testing.B, db *sql.DB, n int) {
	b.Helper()
	now := time.Now().UTC()
	tx, err := db.Begin()
	if err != nil {
		b.Fatalf("begin: %v", err)
	}
	stmt, err := tx.Prepare(`
		INSERT INTO health_results (check_id, status, message, details, duration_ms, created_at)
		VALUES (?, ?, 'bench', '{}', 5, ?)
	`)
	if err != nil {
		b.Fatalf("prepare: %v", err)
	}
	statuses := []checks.Status{checks.StatusOK, checks.StatusWarning, checks.StatusCritical}
	checkIDs := []string{"infra-network", "system-memory", "system-disk", "system-cpu"}
	for i := 0; i < n; i++ {
		ts := now.Add(-time.Duration(i) * time.Minute / 4).Format(time.RFC3339Nano)
		if _, err := stmt.Exec(checkIDs[i%len(checkIDs)], string(statuses[i%len(statuses)]), ts); err != nil {
			b.Fatalf("insert: %v", err)
		}
	}
	if err := stmt.Close(); err != nil {
		b.Fatalf("close stmt: %v", err)
	}
	if err := tx.Commit(); err != nil {
		b.Fatalf("commit: %v", err)
	}
}

func openBenchDB(b *testing.B) *sql.DB {
	b.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		b.Fatalf("open: %v", err)
	}
	b.Cleanup(func() { _ = db.Close() })
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if _, err := db.Exec(Schema()); err != nil {
		b.Fatalf("apply schema: %v", err)
	}
	return db
}

func BenchmarkGetUptimeHistory_100k(b *testing.B) {
	db := openBenchDB(b)
	benchSeed(b, db, 100_000)
	store := NewStore(db)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.GetUptimeHistory(ctx, 24, 24); err != nil {
			b.Fatalf("GetUptimeHistory: %v", err)
		}
	}
}

func BenchmarkGetRecentResults_100k(b *testing.B) {
	db := openBenchDB(b)
	benchSeed(b, db, 100_000)
	store := NewStore(db)
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := store.GetRecentResults(ctx, "infra-network", 20); err != nil {
			b.Fatalf("GetRecentResults: %v", err)
		}
	}
}
