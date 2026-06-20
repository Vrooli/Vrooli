package perf_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"structure-health/internal/perf"
	testutildb "structure-health/internal/testutil/db"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

func newStore(t *testing.T) *perf.Store {
	t.Helper()
	db := testutildb.NewSQLite(t)
	if _, err := db.Exec(perf.Schema()); err != nil {
		t.Fatalf("apply perf schema: %v", err)
	}
	return perf.NewStore(db)
}

// [REQ:SH-PERF-001]
func TestStoreInsertAndSeriesRoundTrip(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()

	m := perf.Measurement{
		Scenario:        "demo",
		CapturedAt:      time.Now(),
		TimeToHealthyMs: 4200,
		Healthy:         true,
		SurfaceTimings:  []perf.SurfaceTiming{{Surface: "api", TimeToHealthyMs: 3000, Healthy: true}},
		Metrics:         &commonv1.ExecutionMetrics{WallClockMs: 4200},
		Note:            "",
	}
	if err := store.Insert(ctx, m); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := store.Series(ctx, "demo", 10)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 measurement, got %d", len(got))
	}
	row := got[0]
	if row.TimeToHealthyMs != 4200 || !row.Healthy {
		t.Fatalf("unexpected row: %+v", row)
	}
	if len(row.SurfaceTimings) != 1 || row.SurfaceTimings[0].Surface != "api" {
		t.Fatalf("surface timings lost: %+v", row.SurfaceTimings)
	}
	if row.Metrics == nil || row.Metrics.GetWallClockMs() != 4200 {
		t.Fatalf("metrics lost: %+v", row.Metrics)
	}
}

// [REQ:SH-PERF-001]
func TestStoreSeriesNewestFirstAndScoped(t *testing.T) {
	store := newStore(t)
	ctx := context.Background()
	base := time.Now()
	mustInsert(t, store, perf.Measurement{Scenario: "demo", CapturedAt: base.Add(-2 * time.Minute), TimeToHealthyMs: 1})
	mustInsert(t, store, perf.Measurement{Scenario: "demo", CapturedAt: base, TimeToHealthyMs: 2})
	mustInsert(t, store, perf.Measurement{Scenario: "other", CapturedAt: base, TimeToHealthyMs: 99})

	got, err := store.Series(ctx, "demo", 10)
	if err != nil {
		t.Fatalf("series: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 demo measurements, got %d", len(got))
	}
	if got[0].TimeToHealthyMs != 2 || got[1].TimeToHealthyMs != 1 {
		t.Fatalf("not newest-first: %+v", got)
	}
}

type fakeRunner struct {
	m   perf.Measurement
	err error
}

func (f fakeRunner) Measure(_ context.Context, scenario string, _ time.Duration) (perf.Measurement, error) {
	out := f.m
	out.Scenario = scenario
	return out, f.err
}

// [REQ:SH-PERF-001]
func TestServiceBenchmarkPersists(t *testing.T) {
	store := newStore(t)
	runner := fakeRunner{m: perf.Measurement{TimeToHealthyMs: 1500, Healthy: true, CapturedAt: time.Now()}}
	svc := perf.NewService(runner, store, "structure-health")

	m, err := svc.Benchmark(context.Background(), "demo", 0)
	if err != nil {
		t.Fatalf("benchmark: %v", err)
	}
	if m.TimeToHealthyMs != 1500 || m.Scenario != "demo" {
		t.Fatalf("unexpected measurement: %+v", m)
	}
	trend, err := svc.Trend(context.Background(), "demo", 0)
	if err != nil {
		t.Fatalf("trend: %v", err)
	}
	if len(trend) != 1 {
		t.Fatalf("benchmark should persist exactly one row, got %d", len(trend))
	}
}

// [REQ:SH-PERF-002]
func TestServiceRejectsSelfBenchmark(t *testing.T) {
	svc := perf.NewService(fakeRunner{}, newStore(t), "structure-health")
	if _, err := svc.Benchmark(context.Background(), "structure-health", 0); err == nil {
		t.Fatal("expected self-benchmark to be rejected (self-deadlock guard)")
	}
}

// [REQ:SH-PERF-002]
func TestServiceBenchmarkRunnerError(t *testing.T) {
	store := newStore(t)
	svc := perf.NewService(fakeRunner{err: errors.New("restart failed")}, store, "structure-health")
	if _, err := svc.Benchmark(context.Background(), "demo", 0); err == nil {
		t.Fatal("expected runner error to surface")
	}
	// A failed measurement is not persisted.
	trend, _ := svc.Trend(context.Background(), "demo", 0)
	if len(trend) != 0 {
		t.Fatalf("failed benchmark must not persist, got %d rows", len(trend))
	}
}

func mustInsert(t *testing.T, store *perf.Store, m perf.Measurement) {
	t.Helper()
	if err := store.Insert(context.Background(), m); err != nil {
		t.Fatalf("insert: %v", err)
	}
}
