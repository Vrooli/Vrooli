package runs

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/execution"

	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

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
