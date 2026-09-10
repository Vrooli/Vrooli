package sqlite

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/scenarios/system-monitor/api/internal/repository"
)

// seedMetricCycles writes n cycles one second apart, each with a cpu and a
// memory observation, and returns the observation time of the last cycle.
func seedMetricCycles(t *testing.T, repo *Repository, n int) time.Time {
	t.Helper()
	base := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	var last time.Time
	for i := 0; i < n; i++ {
		last = base.Add(time.Duration(i) * time.Second)
		err := repo.SaveMetricCycle(context.Background(), fmt.Sprintf("cycle-%05d", i), last, []repository.MetricObservation{
			{CollectorName: "cpu", Values: map[string]interface{}{"usage_percent": float64(i)}},
			{CollectorName: "memory", Values: map[string]interface{}{"usage_percent": float64(i)}},
		})
		if err != nil {
			t.Fatalf("seed cycle %d: %v", i, err)
		}
	}
	return last
}

// queryPlan returns the EXPLAIN QUERY PLAN detail lines for a statement.
func queryPlan(t *testing.T, repo *Repository, query string, args ...interface{}) []string {
	t.Helper()
	rows, err := repo.db.QueryContext(context.Background(), "EXPLAIN QUERY PLAN "+query, args...)
	if err != nil {
		t.Fatalf("explain %q: %v", query, err)
	}
	defer rows.Close()
	var details []string
	for rows.Next() {
		var id, parent, notUsed int
		var detail string
		if err := rows.Scan(&id, &parent, &notUsed, &detail); err != nil {
			t.Fatalf("scan plan row: %v", err)
		}
		details = append(details, detail)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("plan rows: %v", err)
	}
	return details
}

// assertIndexed fails when the plan walks the metrics table without an index
// or has to sort the result in a temporary B-tree; either one is a read whose
// cost grows with the table, which is exactly what a timer-driven query must
// not do against a multi-gigabyte metrics table.
func assertIndexed(t *testing.T, label string, plan []string) {
	t.Helper()
	if len(plan) == 0 {
		t.Fatalf("%s: empty query plan", label)
	}
	for _, line := range plan {
		if strings.Contains(line, "metrics") && !strings.Contains(line, "USING") {
			t.Errorf("%s: plan touches metrics without an index: %q", label, line)
		}
		if strings.HasPrefix(line, "SCAN metrics") && !strings.Contains(line, "COVERING INDEX") {
			t.Errorf("%s: plan performs a table scan: %q", label, line)
		}
		if strings.Contains(line, "TEMP B-TREE") {
			t.Errorf("%s: plan sorts in a temp B-tree: %q", label, line)
		}
	}
	t.Logf("%s: %s", label, strings.Join(plan, " | "))
}

func TestMetricsHotQueriesUseIndexes(t *testing.T) {
	repo := newTestRepo(t)
	last := seedMetricCycles(t, repo, 200)
	// The planner needs statistics to prefer the composite index on a small
	// fixture the way it does on a production-sized table.
	if _, err := repo.db.ExecContext(context.Background(), "ANALYZE"); err != nil {
		t.Fatalf("analyze: %v", err)
	}

	for _, sql := range []string{latestMetricForCollectorSQL, anyMetricExistsSQL, earliestMetricSQL} {
		if strings.Contains(strings.ToUpper(sql), "COUNT(") {
			t.Errorf("hot-path statement counts rows: %q", sql)
		}
	}

	assertIndexed(t, "latest per collector", queryPlan(t, repo, latestMetricForCollectorSQL, "cpu"))
	assertIndexed(t, "earliest", queryPlan(t, repo, earliestMetricSQL))

	unfiltered, _ := metricsWhere(repository.MetricsFilter{})
	assertIndexed(t, "cutoff unfiltered", queryPlan(t, repo, metricsCutoffSQL(unfiltered), 99))

	byCollector, args := metricsWhere(repository.MetricsFilter{CollectorName: "cpu"})
	assertIndexed(t, "cutoff by collector", queryPlan(t, repo, metricsCutoffSQL(byCollector), append(args, 99)...))

	window := repository.TimeRange{StartTime: last.Add(-time.Minute), EndTime: last}
	byRange, args := metricsWhere(repository.MetricsFilter{TimeRange: window})
	assertIndexed(t, "cutoff by range", queryPlan(t, repo, metricsCutoffSQL(byRange), append(args, 99)...))
	assertIndexed(t, "select by range", queryPlan(t, repo, metricsSelectSQL(byRange), args...))

	both, args := metricsWhere(repository.MetricsFilter{CollectorName: "cpu", TimeRange: window})
	assertIndexed(t, "select by collector and range", queryPlan(t, repo, metricsSelectSQL(both), args...))
}

func TestMetricsSchemaReplacesCollectorIndex(t *testing.T) {
	repo := newTestRepo(t)
	rows, err := repo.db.QueryContext(context.Background(), "SELECT name FROM sqlite_master WHERE type = 'index' AND tbl_name = 'metrics'")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	indexes := map[string]bool{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		indexes[name] = true
	}
	if !indexes["idx_metrics_collector_observed_at"] {
		t.Errorf("composite collector index missing; have %v", indexes)
	}
	if indexes["idx_metrics_collector"] {
		t.Errorf("redundant single-column collector index still present; have %v", indexes)
	}
}

func TestGetLatestMetricsReturnsNewestCycleFromLargeTable(t *testing.T) {
	repo := newTestRepo(t)
	last := seedMetricCycles(t, repo, 500)

	latest, err := repo.GetLatestMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetLatestMetrics: %v", err)
	}
	if !latest.Timestamp.Equal(last) {
		t.Fatalf("latest timestamp = %s, want %s", latest.Timestamp, last)
	}
	if latest.CPUUsage != 499 || latest.MemoryUsage != 499 {
		t.Fatalf("latest values = cpu %v memory %v, want 499/499", latest.CPUUsage, latest.MemoryUsage)
	}
}

// A table that holds only collectors outside the latest-metrics set is not
// "not found": the response is empty but present, as before, and the check
// that distinguishes it from an empty table is an EXISTS, not a count.
func TestGetLatestMetricsWithOnlyOtherCollectors(t *testing.T) {
	repo := newTestRepo(t)
	saveTestMetric(t, repo, "pressure", map[string]interface{}{"some": 1.0})

	latest, err := repo.GetLatestMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetLatestMetrics: %v", err)
	}
	if latest.Timestamp.IsZero() {
		t.Fatal("expected a non-zero timestamp for a non-empty table")
	}
}

func TestGetEarliestMetricTimeFromLargeTable(t *testing.T) {
	repo := newTestRepo(t)
	seedMetricCycles(t, repo, 300)

	earliest, err := repo.GetEarliestMetricTime(context.Background())
	if err != nil {
		t.Fatalf("GetEarliestMetricTime: %v", err)
	}
	want := time.Date(2026, 9, 9, 0, 0, 0, 0, time.UTC)
	if !earliest.Equal(want) {
		t.Fatalf("earliest = %s, want %s", earliest, want)
	}
}

func TestGetMetricsWithoutLimitReturnsDefaultCapOfNewestCycles(t *testing.T) {
	repo := newTestRepo(t)
	total := repository.DefaultMetricsLimit + 25
	last := seedMetricCycles(t, repo, total)

	results, err := repo.GetMetrics(context.Background(), repository.MetricsFilter{})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(results) != repository.DefaultMetricsLimit {
		t.Fatalf("unbounded GetMetrics returned %d cycles, want the default cap %d", len(results), repository.DefaultMetricsLimit)
	}
	if got := results[len(results)-1].Timestamp; !got.Equal(last) {
		t.Fatalf("newest returned cycle = %s, want %s", got, last)
	}
	wantOldest := last.Add(-time.Duration(repository.DefaultMetricsLimit-1) * time.Second)
	if got := results[0].Timestamp; !got.Equal(wantOldest) {
		t.Fatalf("oldest returned cycle = %s, want %s", got, wantOldest)
	}
	// Every returned cycle must be whole: both collectors of the cycle hydrate.
	for _, cycle := range results {
		if cycle.CPUUsage != cycle.MemoryUsage {
			t.Fatalf("cycle %s split across the cutoff: cpu %v memory %v", cycle.CycleID, cycle.CPUUsage, cycle.MemoryUsage)
		}
	}
}

func TestGetMetricsLimitKeepsNewestCyclesAndClampsToMax(t *testing.T) {
	repo := newTestRepo(t)
	last := seedMetricCycles(t, repo, 30)
	ctx := context.Background()

	results, err := repo.GetMetrics(ctx, repository.MetricsFilter{Limit: 3})
	if err != nil {
		t.Fatalf("GetMetrics: %v", err)
	}
	if len(results) != 3 {
		t.Fatalf("Limit 3 returned %d cycles", len(results))
	}
	if !results[2].Timestamp.Equal(last) || !results[0].Timestamp.Equal(last.Add(-2*time.Second)) {
		t.Fatalf("Limit 3 did not keep the newest cycles: %s .. %s", results[0].Timestamp, results[2].Timestamp)
	}

	byCollector, err := repo.GetMetrics(ctx, repository.MetricsFilter{CollectorName: "memory", Limit: 2})
	if err != nil {
		t.Fatalf("GetMetrics by collector: %v", err)
	}
	if len(byCollector) != 2 || !byCollector[1].Timestamp.Equal(last) {
		t.Fatalf("collector-filtered Limit 2 = %d cycles ending %s", len(byCollector), byCollector[len(byCollector)-1].Timestamp)
	}

	oversized, err := repo.GetMetrics(ctx, repository.MetricsFilter{Limit: repository.MaxMetricsLimit * 10})
	if err != nil {
		t.Fatalf("GetMetrics oversized limit: %v", err)
	}
	if len(oversized) != 30 {
		t.Fatalf("oversized limit returned %d of 30 cycles", len(oversized))
	}
	if got := (repository.MetricsFilter{Limit: repository.MaxMetricsLimit * 10}).EffectiveLimit(); got != repository.MaxMetricsLimit {
		t.Fatalf("EffectiveLimit above max = %d, want %d", got, repository.MaxMetricsLimit)
	}
}
