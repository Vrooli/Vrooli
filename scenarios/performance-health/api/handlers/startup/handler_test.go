package startup

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"

	internalstartup "performance-health/internal/startup"

	startupv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/startup"

	_ "modernc.org/sqlite"
)

// fakeRunner drives the startup Service's lowest seam (no real restarts).
type fakeRunner struct {
	m   internalstartup.Measurement
	err error
}

func (f fakeRunner) Measure(_ context.Context, scenario string, _ time.Duration) (internalstartup.Measurement, error) {
	f.m.Scenario = scenario
	return f.m, f.err
}

// openStore returns a fresh, schema-applied startup Store backed by a per-test
// SQLite file (mirrors internal/startup's own test setup).
func openStore(t *testing.T) *internalstartup.Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/startup.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), internalstartup.Schema()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return internalstartup.NewStore(db)
}

// TestBenchmarkStartupMapsMeasurementToProto builds the REAL startup service
// over a fake runner + a real SQLite store, calls BenchmarkStartup, and asserts
// the measurement (including per-surface timings) maps correctly to the proto
// response.
func TestBenchmarkStartupMapsMeasurementToProto(t *testing.T) {
	now := time.Now().UTC()
	runner := fakeRunner{m: internalstartup.Measurement{
		CapturedAt:      now,
		TimeToHealthyMs: 2500,
		Healthy:         true,
		Note:            "benchmark",
		SurfaceTimings: []internalstartup.SurfaceTiming{
			{Surface: "api", TimeToHealthyMs: 1200, Healthy: true},
			{Surface: "ui", TimeToHealthyMs: 2500, Healthy: true},
		},
	}}
	svc := internalstartup.NewService(runner, openStore(t), "performance-health")
	h := NewHandler(svc, nil)

	resp, err := h.BenchmarkStartup(context.Background(), connect.NewRequest(&startupv1.BenchmarkStartupRequest{Scenario: "demo", TimeoutSeconds: 90}))
	if err != nil {
		t.Fatalf("BenchmarkStartup: %v", err)
	}
	m := resp.Msg.GetMeasurement()
	if m.GetScenario() != "demo" || m.GetTimeToHealthyMs() != 2500 || !m.GetHealthy() || m.GetNote() != "benchmark" {
		t.Errorf("measurement mapped wrong: %+v", m)
	}
	if len(m.GetSurfaceTimings()) != 2 {
		t.Fatalf("surface timings len = %d, want 2", len(m.GetSurfaceTimings()))
	}
	ui := m.GetSurfaceTimings()[1]
	if ui.GetSurface() != "ui" || ui.GetTimeToHealthyMs() != 2500 || !ui.GetHealthy() {
		t.Errorf("surface timing mapped wrong: %+v", ui)
	}
}

// TestBenchmarkStartupRejectsSelf proves the self-benchmark guard (benchmarking
// performance-health itself would deadlock) surfaces as an Internal error code.
func TestBenchmarkStartupRejectsSelf(t *testing.T) {
	svc := internalstartup.NewService(fakeRunner{}, openStore(t), "performance-health")
	h := NewHandler(svc, nil)

	_, err := h.BenchmarkStartup(context.Background(), connect.NewRequest(&startupv1.BenchmarkStartupRequest{Scenario: "performance-health"}))
	if err == nil {
		t.Fatal("benchmarking performance-health itself must be rejected")
	}
	if connect.CodeOf(err) != connect.CodeInternal {
		t.Fatalf("want Internal, got %v (err=%v)", connect.CodeOf(err), err)
	}
}

// TestGetStartupTrendReadsNewestFirst persists two measurements via the service,
// then reads them back through the GetStartupTrend RPC and asserts the proto
// response carries them newest-first.
func TestGetStartupTrendReadsNewestFirst(t *testing.T) {
	store := openStore(t)
	svc := internalstartup.NewService(fakeRunner{}, store, "performance-health")
	h := NewHandler(svc, nil)
	ctx := context.Background()

	for i, ms := range []int64{1000, 2000} {
		if err := store.Insert(ctx, internalstartup.Measurement{
			Scenario:        "demo",
			CapturedAt:      time.Now().Add(time.Duration(i) * time.Second),
			TimeToHealthyMs: ms,
			Healthy:         true,
		}); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	resp, err := h.GetStartupTrend(ctx, connect.NewRequest(&startupv1.GetStartupTrendRequest{Scenario: "demo", Limit: 10}))
	if err != nil {
		t.Fatalf("GetStartupTrend: %v", err)
	}
	if resp.Msg.GetScenario() != "demo" {
		t.Errorf("scenario = %q", resp.Msg.GetScenario())
	}
	ms := resp.Msg.GetMeasurements()
	if len(ms) != 2 {
		t.Fatalf("measurements len = %d, want 2", len(ms))
	}
	if ms[0].GetTimeToHealthyMs() != 2000 || ms[1].GetTimeToHealthyMs() != 1000 {
		t.Errorf("expected newest-first (2000 then 1000), got %d then %d", ms[0].GetTimeToHealthyMs(), ms[1].GetTimeToHealthyMs())
	}
}

// TestBenchmarkStartupRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestBenchmarkStartupRequiresScenario(t *testing.T) {
	svc := internalstartup.NewService(fakeRunner{}, openStore(t), "performance-health")
	h := NewHandler(svc, nil)
	_, err := h.BenchmarkStartup(context.Background(), connect.NewRequest(&startupv1.BenchmarkStartupRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
