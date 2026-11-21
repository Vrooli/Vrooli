package trend

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"connectrpc.com/connect"

	internaltrend "performance-health/internal/trend"

	trendv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/trend"

	_ "modernc.org/sqlite"
)

// openStore returns a fresh, schema-applied trend Store backed by a per-test
// SQLite file (mirrors internal/trend's own test setup).
func openStore(t *testing.T) *internaltrend.Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file:"+t.TempDir()+"/trend.db?_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.ExecContext(context.Background(), internaltrend.Schema()); err != nil {
		t.Fatalf("ensure schema: %v", err)
	}
	return internaltrend.NewStore(db)
}

// TestGetTrendMapsSamplesToProto persists samples across all axes via the real
// store, then reads them through the GetTrend RPC and asserts the proto response
// carries them newest-first with every field mapped correctly.
func TestGetTrendMapsSamplesToProto(t *testing.T) {
	store := openStore(t)
	svc := internaltrend.NewService(store)
	h := NewHandler(svc, nil)
	ctx := context.Background()

	older := internaltrend.Sample{
		Scenario:   "demo",
		CapturedAt: time.Now().Add(-time.Minute),
		GoBuildMs:  10000,
		Note:       "older",
	}
	newer := internaltrend.Sample{
		Scenario:              "demo",
		CapturedAt:            time.Now(),
		GoBuildMs:             12000,
		UIBuildMs:             40000,
		BundleBytes:           524288,
		LCPMs:                 2400,
		StartupMs:             2500,
		SlowestComponent:      "Dashboard",
		SlowestComponentAvgMs: 18.5,
		SlowestComponentMaxMs: 42.0,
		Note:                  "newer",
	}
	for _, s := range []internaltrend.Sample{older, newer} {
		if err := store.Insert(ctx, s); err != nil {
			t.Fatalf("Insert: %v", err)
		}
	}

	resp, err := h.GetTrend(ctx, connect.NewRequest(&trendv1.GetTrendRequest{Scenario: "demo", Limit: 10}))
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if resp.Msg.GetScenario() != "demo" {
		t.Errorf("scenario = %q", resp.Msg.GetScenario())
	}
	samples := resp.Msg.GetSamples()
	if len(samples) != 2 {
		t.Fatalf("samples len = %d, want 2", len(samples))
	}
	// Newest-first: the "newer" sample leads.
	got := samples[0]
	if got.GetNote() != "newer" {
		t.Fatalf("expected newest-first, leading note = %q", got.GetNote())
	}
	switch {
	case got.GetGoBuildMs() != 12000:
		t.Errorf("go_build_ms = %d", got.GetGoBuildMs())
	case got.GetUiBuildMs() != 40000:
		t.Errorf("ui_build_ms = %d", got.GetUiBuildMs())
	case got.GetBundleBytes() != 524288:
		t.Errorf("bundle_bytes = %d", got.GetBundleBytes())
	case got.GetLcpMs() != 2400:
		t.Errorf("lcp_ms = %d", got.GetLcpMs())
	case got.GetStartupMs() != 2500:
		t.Errorf("startup_ms = %d", got.GetStartupMs())
	case got.GetSlowestComponent() != "Dashboard":
		t.Errorf("slowest_component = %q", got.GetSlowestComponent())
	case got.GetSlowestComponentAvgMs() != 18.5:
		t.Errorf("slowest_component_avg_ms = %v", got.GetSlowestComponentAvgMs())
	case got.GetSlowestComponentMaxMs() != 42.0:
		t.Errorf("slowest_component_max_ms = %v", got.GetSlowestComponentMaxMs())
	}
}

// TestGetTrendEmptyScenarioReturnsNoSamples proves reading a scenario with no
// persisted samples maps to an empty (non-error) response.
func TestGetTrendEmptyScenarioReturnsNoSamples(t *testing.T) {
	h := NewHandler(internaltrend.NewService(openStore(t)), nil)
	resp, err := h.GetTrend(context.Background(), connect.NewRequest(&trendv1.GetTrendRequest{Scenario: "demo", Limit: 10}))
	if err != nil {
		t.Fatalf("GetTrend: %v", err)
	}
	if len(resp.Msg.GetSamples()) != 0 {
		t.Errorf("expected no samples, got %d", len(resp.Msg.GetSamples()))
	}
}

// TestGetTrendRequiresScenario asserts the empty-scenario guard maps to
// InvalidArgument.
func TestGetTrendRequiresScenario(t *testing.T) {
	h := NewHandler(internaltrend.NewService(openStore(t)), nil)
	_, err := h.GetTrend(context.Background(), connect.NewRequest(&trendv1.GetTrendRequest{}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("want InvalidArgument, got %v (err=%v)", connect.CodeOf(err), err)
	}
}
