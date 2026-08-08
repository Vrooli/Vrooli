package orchestrator

import (
	"context"
	"testing"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"test-genie/internal/orchestrator/phases"
)

type schedulerCostStub struct{}

func (schedulerCostStub) PhaseCostEstimate(context.Context, string, string) (int64, int64, bool) {
	return 64, 100, true
}

type schedulerBrokerStub struct {
	kind   string
	reason string
}

func (s schedulerBrokerStub) Acquire(context.Context, string, int64, int64) (sharedcapacity.Lease, sharedcapacity.Verdict, error) {
	return nil, sharedcapacity.Verdict{Kind: s.kind, Reason: s.reason}, nil
}

func TestNextPhaseBatchHonorsExclusiveAndProviderSerial(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), ProviderScenario: "provider-a", Concurrency: phases.Concurrency{Mode: "provider-serial"}},
		{Name: phases.Name("three"), ProviderScenario: "provider-a", Concurrency: phases.Concurrency{Mode: "provider-serial"}},
		{Name: phases.Name("four"), Concurrency: phases.Concurrency{Mode: "exclusive"}},
	}
	if got := nextPhaseBatch(defs, 0, false); got != 2 {
		t.Fatalf("first batch end = %d, want 2", got)
	}
	if got := nextPhaseBatch(defs, 2, false); got != 3 {
		t.Fatalf("second provider batch end = %d, want 3", got)
	}
	if got := nextPhaseBatch(defs, 3, false); got != 4 {
		t.Fatalf("exclusive batch end = %d, want 4", got)
	}
	if got := nextPhaseBatch(defs, 0, true); got != 1 {
		t.Fatalf("forced serial batch end = %d, want 1", got)
	}
}

func TestNextPhaseBatchSerializesDeadlineSensitivePhase(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("short"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("docs"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("after"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	predicted := map[string]int64{"short": 10_000, "docs": 31_000, "after": 10_000}
	if got := nextPhaseBatchWithPredictions(defs, 0, false, predicted); got != 1 {
		t.Fatalf("short batch end = %d, want 1 before deadline-sensitive phase", got)
	}
	if got := nextPhaseBatchWithPredictions(defs, 1, false, predicted); got != 2 {
		t.Fatalf("deadline-sensitive batch end = %d, want singleton", got)
	}
}

func TestAdmitPhaseBatchDenialFallsBackWithReason(t *testing.T) {
	o := &SuiteOrchestrator{
		capacity:      schedulerBrokerStub{kind: "deny", reason: "insufficient ram"},
		costEstimator: schedulerCostStub{},
	}
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	_, reason, admitted := o.admitPhaseBatch(context.Background(), "demo", "run-1", defs)
	if admitted {
		t.Fatal("denied batch was admitted")
	}
	if reason != "insufficient ram" {
		t.Fatalf("fallback reason = %q, want broker reason", reason)
	}
}
