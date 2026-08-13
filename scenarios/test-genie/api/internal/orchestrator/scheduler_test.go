package orchestrator

import (
	"context"
	"testing"
	"time"

	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"
	"test-genie/internal/orchestrator/phases"
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

func batchAllEligible() phaseBatchPolicy {
	return phaseBatchPolicy{admissionEnabled: true}
}

func TestNextPhaseBatchHonorsExclusiveAndProviderSerial(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), ProviderScenario: "provider-a", Concurrency: phases.Concurrency{Mode: "provider-serial"}},
		{Name: phases.Name("three"), ProviderScenario: "provider-a", Concurrency: phases.Concurrency{Mode: "provider-serial"}},
		{Name: phases.Name("four"), Concurrency: phases.Concurrency{Mode: "exclusive"}},
	}
	policy := batchAllEligible()
	if got := nextPhaseBatch(defs, 0, policy); got != 2 {
		t.Fatalf("first batch end = %d, want 2", got)
	}
	if got := nextPhaseBatch(defs, 2, policy); got != 3 {
		t.Fatalf("second provider batch end = %d, want 3", got)
	}
	if got := nextPhaseBatch(defs, 3, policy); got != 4 {
		t.Fatalf("exclusive batch end = %d, want 4", got)
	}
	serial := batchAllEligible()
	serial.forceSerial = true
	if got := nextPhaseBatch(defs, 0, serial); got != 1 {
		t.Fatalf("forced serial batch end = %d, want 1", got)
	}
}

func TestNextPhaseBatchSerializesDeadlineSensitivePhase(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("short"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("docs"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("after"), Timeout: time.Minute, Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	policy := batchAllEligible()
	policy.timeoutRisk = func(def phases.Definition) bool { return def.Name.Key() == "docs" }

	if got := nextPhaseBatch(defs, 0, policy); got != 2 {
		t.Fatalf("short batch end = %d, want 2 with deadline-sensitive phase deferred", got)
	}
	if got := nextPhaseBatch(defs, 2, policy); got != 3 {
		t.Fatalf("deadline-sensitive batch end = %d, want singleton", got)
	}
	if got := nextPhaseBatch(defs, 1, policy); got != 2 {
		t.Fatalf("trailing batch end = %d, want 3", got)
	}
}

// A deadline-sensitive phase must not cost the phases beside it their
// concurrency. It is deferred into a later batch rather than splitting the
// whole run into a serial walk.
func TestNextPhaseBatchIsolatesDeadlineSensitivePhaseWithoutCollapsingNeighbours(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("unit"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("storage"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("workflow"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("business"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("security"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	policy := phaseBatchPolicy{admissionEnabled: true, timeoutRisk: func(def phases.Definition) bool { return def.Name.Key() == "workflow" }}

	if got := nextPhaseBatch(defs, 0, policy); got != 4 {
		t.Fatalf("batch should skip deadline-sensitive phase and end at %d, want 4", got)
	}
	if got := nextPhaseBatch(defs, 4, policy); got != 5 {
		t.Fatalf("deferred phase batch end = %d, want a singleton", got)
	}
}

// With no broker wired there is nothing to admit against, so the run walks the
// list rather than attempting a batch.
func TestNextPhaseBatchIsSerialWithoutAdmissionBroker(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	if got := nextPhaseBatch(defs, 0, phaseBatchPolicy{}); got != 1 {
		t.Fatalf("batch end = %d, want a serial walk", got)
	}
}

// The deadline guard reads measured history in preference to the planner's
// prediction. The prediction is rounded to whole seconds and biased upward —
// 56 s for a phase measured at 23.9 s on 2026-08-08 — so trusting it here
// serializes phases that have ample headroom.
func TestPhaseTimeoutRiskPrefersMeasuredHistoryOverPrediction(t *testing.T) {
	def := phases.Definition{Name: phases.Name("contracts"), Timeout: 90 * time.Second}
	predicted := map[string]int64{"contracts": 70_000}
	measured := func(phases.Definition) (int64, bool) { return 23_900, true }

	if phaseTimeoutRisk(def, predicted, measured) {
		t.Fatal("serialized a phase whose measured duration leaves headroom")
	}
	if !phaseTimeoutRisk(def, predicted, nil) {
		t.Fatal("expected a prediction without history to trip the guard")
	}
}

// A phase that genuinely runs near its deadline is still kept out of batches.
func TestPhaseTimeoutRiskFiresOnMeasuredDeadlinePressure(t *testing.T) {
	def := phases.Definition{Name: phases.Name("security"), Timeout: 180 * time.Second}
	measured := func(phases.Definition) (int64, bool) { return 154_000, true }

	if !phaseTimeoutRisk(def, nil, measured) {
		t.Fatal("did not serialize a phase measured at 154s against a 180s timeout")
	}
}

func TestPhaseTimeoutRiskAllowsMeasuredHeadroomAtCalibratedAllowance(t *testing.T) {
	def := phases.Definition{Name: phases.Name("security"), Timeout: 180 * time.Second}
	measured := func(phases.Definition) (int64, bool) { return 100_000, true }

	if phaseTimeoutRisk(def, nil, measured) {
		t.Fatal("serialized a measured 100s phase with 80s of timeout headroom")
	}
}

// A phase with neither history nor prediction is not guessed at. Its size is
// still unknown to admission, which is what keeps it out of a batch.
func TestPhaseTimeoutRiskIsQuietWithoutAnyInput(t *testing.T) {
	def := phases.Definition{Name: phases.Name("brand-new"), Timeout: time.Minute}
	if phaseTimeoutRisk(def, nil, func(phases.Definition) (int64, bool) { return 0, false }) {
		t.Fatal("reported deadline risk with no measurement and no prediction")
	}
}

func TestAdmitPhaseBatchDenialFallsBackWithReason(t *testing.T) {
	o := &SuiteOrchestrator{
		capacity:      &schedulerBrokerStub{kind: "deny", reason: "insufficient ram"},
		costEstimator: schedulerCostStub{},
	}
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	leases, reason, admitted, _ := o.admitPhaseBatch(context.Background(), "demo", "run-1", defs)
	if admitted != 1 {
		t.Fatalf("admitted = %d, want a serial fallback of 1", admitted)
	}
	if len(leases) != 0 {
		t.Fatalf("held %d leases after falling back to serial", len(leases))
	}
	if reason != "insufficient ram" {
		t.Fatalf("fallback reason = %q, want broker reason", reason)
	}
}

func TestAdmitPhaseBatchUsesConservativeReservationWhenEstimateMissing(t *testing.T) {
	o := &SuiteOrchestrator{
		capacity:      &schedulerBrokerStub{kind: "grant", grantCount: 2},
		costEstimator: schedulerCostStub{unsizable: map[string]bool{"unknown": true}},
	}
	defs := []phases.Definition{
		{Name: phases.Name("unknown"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("known"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	leases, reason, admitted, _ := o.admitPhaseBatch(context.Background(), "demo", "run-1", defs)
	if admitted != 2 || len(leases) != 2 || reason != "" {
		t.Fatalf("estimated admission = leases:%d reason:%q admitted:%d, want two granted leases without serial fallback", len(leases), reason, admitted)
	}
}

// A host that fills up part-way through a batch should run what fits. Releasing
// the grants already held and dropping to serial throws away admitted capacity
// and makes the run pay for the denial twice.
func TestAdmitPhaseBatchRunsTheGrantedPrefix(t *testing.T) {
	o := &SuiteOrchestrator{
		capacity:      &schedulerBrokerStub{kind: "queue", reason: "host full", grantCount: 3},
		costEstimator: schedulerCostStub{},
	}
	defs := make([]phases.Definition, 0, 5)
	for _, name := range []string{"one", "two", "three", "four", "five"} {
		defs = append(defs, phases.Definition{Name: phases.Name(name), Concurrency: phases.Concurrency{Mode: "parallel-safe"}})
	}

	leases, reason, admitted, _ := o.admitPhaseBatch(context.Background(), "demo", "run-1", defs)
	if admitted != 3 {
		t.Fatalf("admitted = %d, want the three granted phases", admitted)
	}
	if len(leases) != 3 {
		t.Fatalf("held %d leases, want one per admitted phase", len(leases))
	}
	if reason != "host full" {
		t.Fatalf("reason = %q, want the broker's stopping reason recorded", reason)
	}
}

// The whole batch fitting is the ordinary case and must not be reported as a
// partial admission.
func TestAdmitPhaseBatchAdmitsFullBatch(t *testing.T) {
	o := &SuiteOrchestrator{
		capacity:      &schedulerBrokerStub{grantCount: 10},
		costEstimator: schedulerCostStub{},
	}
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
	}
	_, reason, admitted, _ := o.admitPhaseBatch(context.Background(), "demo", "run-1", defs)
	if admitted != len(defs) || reason != "" {
		t.Fatalf("admitted = %d reason = %q, want the full batch with no reason", admitted, reason)
	}
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
		policy.timeoutRisk(def)
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
