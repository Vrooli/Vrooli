package runs

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/execution"
	"test-genie/internal/selfhealthsnapshots"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

type fakeSnapshotReader struct {
	latest    selfhealthsnapshots.Snapshot
	hasLatest bool
	series    []selfhealthsnapshots.Snapshot
}

func (f fakeSnapshotReader) Latest(context.Context) (selfhealthsnapshots.Snapshot, bool, error) {
	return f.latest, f.hasLatest, nil
}

func (f fakeSnapshotReader) Series(context.Context, selfhealthsnapshots.SeriesQuery) ([]selfhealthsnapshots.Snapshot, error) {
	return f.series, nil
}

type fakeLedgerSource struct {
	observations []execution.PhaseObservation
	outcomes     []execution.RunOutcomeCount
}

func (f fakeLedgerSource) AggregatePhaseObservations(context.Context, time.Time, int) ([]execution.PhaseObservation, error) {
	return f.observations, nil
}

func (f fakeLedgerSource) CountRunOutcomes(context.Context, time.Time, int) ([]execution.RunOutcomeCount, error) {
	return f.outcomes, nil
}

func TestGetSelfHealthAssemblesPayload(t *testing.T) {
	src := fakeLedgerSource{
		observations: []execution.PhaseObservation{
			{ScenarioName: "demo", TerminalOutcome: "passed", PhaseName: "proto", Status: "passed", DurationSeconds: 9, MetricsPresent: true},
			{ScenarioName: "demo", TerminalOutcome: "failed", PhaseName: "proto", Status: "failed", DurationSeconds: 4},
		},
		outcomes: []execution.RunOutcomeCount{
			{TerminalOutcome: "passed", Count: 4},
			{TerminalOutcome: "errored", Count: 1},
		},
	}
	// scenariosRoot can be any path; conformance is skipped so the real tree is
	// never touched.
	svc := NewService(t.TempDir(), nil, nil, src)

	resp, err := svc.GetSelfHealth(context.Background(), connect.NewRequest(&runspb.GetSelfHealthRequest{
		WindowDays:      14,
		SkipConformance: true,
	}))
	if err != nil {
		t.Fatalf("GetSelfHealth: %v", err)
	}
	sh := resp.Msg.GetSelfHealth()
	if sh == nil {
		t.Fatal("self_health missing")
	}

	if sh.GetConformanceFreshness() != "skipped" {
		t.Fatalf("conformance_freshness = %q, want skipped", sh.GetConformanceFreshness())
	}
	if len(sh.GetConformance()) != 0 {
		t.Fatalf("conformance should be empty when skipped, got %d", len(sh.GetConformance()))
	}

	cat := sh.GetCatalog()
	if cat.GetTotalPhases() == 0 {
		t.Fatal("catalog summary should enumerate phases")
	}
	if cat.GetTotalPhases() != cat.GetDelegatedPhases()+cat.GetNativePhases() {
		t.Fatalf("catalog totals inconsistent: total=%d delegated=%d native=%d",
			cat.GetTotalPhases(), cat.GetDelegatedPhases(), cat.GetNativePhases())
	}
	if cat.GetNativePhases() != 0 {
		t.Fatalf("default catalog should be fully provider-delegated, native=%d", cat.GetNativePhases())
	}
	// proto is a delegated phase; its summary entry must name its provider.
	var foundProto bool
	for _, p := range cat.GetPhases() {
		if p.GetName() == "proto" {
			foundProto = true
			if !p.GetDelegated() || p.GetProvider() == "" {
				t.Fatalf("proto catalog entry should be delegated with a provider: %+v", p)
			}
		}
	}
	if !foundProto {
		t.Fatal("proto phase missing from catalog summary")
	}

	ledger := sh.GetLedger()
	if ledger.GetWindowDays() != 14 {
		t.Fatalf("ledger windowDays = %d, want 14", ledger.GetWindowDays())
	}
	if ledger.GetRunCount() != 5 {
		t.Fatalf("ledger runCount = %d, want 5", ledger.GetRunCount())
	}
	// availability = 4/5 = 0.8 (errored run correctly in the denominator).
	if ledger.GetAvailability() < 0.79 || ledger.GetAvailability() > 0.81 {
		t.Fatalf("ledger availability = %v, want ~0.8", ledger.GetAvailability())
	}
	var protoRel *runspb.PhaseReliability
	for _, p := range ledger.GetPhases() {
		if p.GetPhase() == "proto" {
			protoRel = p
		}
	}
	if protoRel == nil {
		t.Fatal("proto phase reliability missing")
	}
	if protoRel.GetPassed() != 1 || protoRel.GetFailed() != 1 {
		t.Fatalf("proto reliability unexpected: %+v", protoRel)
	}
	if protoRel.GetMetricsAdopted() != 1 {
		t.Fatalf("proto metricsAdopted = %d, want 1", protoRel.GetMetricsAdopted())
	}
	if protoRel.GetProvider() == "" {
		t.Fatal("proto reliability should carry its provider from catalog meta")
	}
}

func TestGetSelfHealthAttachesTrend(t *testing.T) {
	src := fakeLedgerSource{
		outcomes: []execution.RunOutcomeCount{{TerminalOutcome: "passed", Count: 9}, {TerminalOutcome: "failed", Count: 1}},
	}
	prev := time.Date(2026, 6, 19, 10, 0, 0, 0, time.UTC)
	reader := fakeSnapshotReader{
		latest:    selfhealthsnapshots.Snapshot{CapturedAt: prev, Availability: 0.85, RunCount: 6},
		hasLatest: true,
		series: []selfhealthsnapshots.Snapshot{
			{CapturedAt: prev.Add(time.Hour), Availability: 0.90, RunCount: 8, HardViolations: 1, MetricsAdopted: 11},
			{CapturedAt: prev, Availability: 0.85, RunCount: 6},
		},
	}
	svc := NewService(t.TempDir(), nil, nil, src).SetSnapshotReader(reader)

	resp, err := svc.GetSelfHealth(context.Background(), connect.NewRequest(&runspb.GetSelfHealthRequest{
		WindowDays:      30,
		SkipConformance: true,
		IncludeTrend:    true,
	}))
	if err != nil {
		t.Fatalf("GetSelfHealth: %v", err)
	}
	ledger := resp.Msg.GetSelfHealth().GetLedger()
	if ledger.GetCapturedAt() == "" {
		t.Fatal("captured_at must be filled from the latest snapshot")
	}
	trend := ledger.GetTrend()
	if trend == nil {
		t.Fatal("trend delta must be attached when a snapshot exists")
	}
	// availability = (passed+failed)/total = 10/10 = 1.0 (only errored/aborted/
	// timeout reduce it); previous = 0.85 → delta ~0.15.
	if d := trend.GetAvailabilityDelta(); d < 0.14 || d > 0.16 {
		t.Fatalf("availability delta = %v, want ~0.15", d)
	}
	if trend.GetRunCountDelta() != ledger.GetRunCount()-6 {
		t.Fatalf("run_count delta = %d, want %d", trend.GetRunCountDelta(), ledger.GetRunCount()-6)
	}
	if len(resp.Msg.GetSelfHealth().GetTrendSeries()) != 2 {
		t.Fatalf("trend series length = %d, want 2", len(resp.Msg.GetSelfHealth().GetTrendSeries()))
	}
}

func TestGetSelfHealthNoSnapshotReaderHasNoTrend(t *testing.T) {
	src := fakeLedgerSource{outcomes: []execution.RunOutcomeCount{{TerminalOutcome: "passed", Count: 1}}}
	svc := NewService(t.TempDir(), nil, nil, src) // no snapshot reader wired
	resp, err := svc.GetSelfHealth(context.Background(), connect.NewRequest(&runspb.GetSelfHealthRequest{SkipConformance: true}))
	if err != nil {
		t.Fatalf("GetSelfHealth: %v", err)
	}
	ledger := resp.Msg.GetSelfHealth().GetLedger()
	if ledger.GetTrend() != nil || ledger.GetCapturedAt() != "" {
		t.Fatal("compute-on-read path must carry no trend fields without a snapshot store")
	}
}
