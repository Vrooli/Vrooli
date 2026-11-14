package orchestrator

import (
	"context"
	"testing"
	"time"

	"test-genie/internal/orchestrator/phasebatch"
	"test-genie/internal/orchestrator/phases"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
)

type schedulerCostStub struct {
	// unsizable names phases the estimator has no reliable size for. Empty
	// means every phase is sizable.
	unsizable map[string]bool
	// durations supplies measured history per phase key. A missing key means
	// the phase has none and the caller falls back to the prediction.
	durations map[string]int64
}

func (s schedulerCostStub) PhaseCostEstimate(_ context.Context, _, phase string) (int64, int64, bool) {
	if s.unsizable[phase] {
		return 0, 0, false
	}
	return 64, 100, true
}

func (s schedulerCostStub) PhaseDurationEstimate(_ context.Context, _, phase string) (int64, bool) {
	ms, ok := s.durations[phase]
	return ms, ok
}

// schedulerBrokerStub grants the first grantCount acquisitions and then answers
// with kind/reason, so a case can describe a host that fills up mid-batch.
type schedulerBrokerStub struct {
	kind       string
	reason     string
	grantCount int
	granted    int
	acquires   int
}

func (s *schedulerBrokerStub) Acquire(context.Context, string, int64, int64) (sharedcapacity.Lease, sharedcapacity.Verdict, error) {
	s.acquires++
	if s.granted < s.grantCount {
		s.granted++
		return stubLease{}, sharedcapacity.Verdict{Kind: "grant"}, nil
	}
	return nil, sharedcapacity.Verdict{Kind: s.kind, Reason: s.reason}, nil
}

type stubLease struct{}

func (stubLease) Release(context.Context) error { return nil }

func batchAllEligible() phasebatch.Policy {
	return phasebatch.Policy{AdmissionEnabled: true}
}

// The policy resolves each phase's history once per run. The batcher re-proposes
// the tail of the phase list on every iteration, so a lookup per proposal made
// admission cost grow with the square of the phase count.
func TestPhaseBatchPolicyMemoizesLookupsPerPhase(t *testing.T) {
	counting := &countingCostStub{durations: map[string]int64{"one": 1_000}}
	o := &SuiteOrchestrator{capacity: &schedulerBrokerStub{grantCount: 10}, costEstimator: counting}
	def := phases.Definition{Name: phases.Name("one"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}}

	policy := o.phaseBatchPolicy(context.Background(), "demo", false, map[string]int64{"one": 1_000})
	for i := 0; i < 5; i++ {
		policy.TimeoutRisk(def)
	}

	if counting.durationCalls != 1 {
		t.Fatalf("duration lookups = %d, want 1 per phase per run", counting.durationCalls)
	}
}

type countingCostStub struct {
	durations     map[string]int64
	durationCalls int
}

func (c *countingCostStub) PhaseCostEstimate(context.Context, string, string) (int64, int64, bool) {
	return 64, 100, true
}

func (c *countingCostStub) PhaseDurationEstimate(_ context.Context, _, phase string) (int64, bool) {
	c.durationCalls++
	ms, ok := c.durations[phase]
	return ms, ok
}
