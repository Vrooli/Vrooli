package phasebatch

import (
	"testing"
	"time"

	"test-genie/internal/orchestrator/phases"
)

// batchAllEligible is a policy under which every phase is batchable, so a test
// isolates the batching RULES from the admission decision.
func batchAllEligible() Policy {
	return Policy{AdmissionEnabled: true}
}

func TestNextPhaseBatchHonorsExclusiveAndProviderSerial(t *testing.T) {
	defs := []phases.Definition{
		{Name: phases.Name("one"), Concurrency: phases.Concurrency{Mode: "parallel-safe"}},
		{Name: phases.Name("two"), ProviderScenario: "provider-a", Concurrency: phases.Concurrency{Mode: "provider-serial"}},
		{Name: phases.Name("three"), ProviderScenario: "provider-a", Concurrency: phases.Concurrency{Mode: "provider-serial"}},
		{Name: phases.Name("four"), Concurrency: phases.Concurrency{Mode: "exclusive"}},
	}
	policy := batchAllEligible()
	if got := Next(defs, 0, policy); got != 2 {
		t.Fatalf("first batch end = %d, want 2", got)
	}
	if got := Next(defs, 2, policy); got != 3 {
		t.Fatalf("second provider batch end = %d, want 3", got)
	}
	if got := Next(defs, 3, policy); got != 4 {
		t.Fatalf("exclusive batch end = %d, want 4", got)
	}
	serial := batchAllEligible()
	serial.ForceSerial = true
	if got := Next(defs, 0, serial); got != 1 {
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
	policy.TimeoutRisk = func(def phases.Definition) bool { return def.Name.Key() == "docs" }

	if got := Next(defs, 0, policy); got != 2 {
		t.Fatalf("short batch end = %d, want 2 with deadline-sensitive phase deferred", got)
	}
	if got := Next(defs, 2, policy); got != 3 {
		t.Fatalf("deadline-sensitive batch end = %d, want singleton", got)
	}
	if got := Next(defs, 1, policy); got != 2 {
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
	policy := Policy{AdmissionEnabled: true, TimeoutRisk: func(def phases.Definition) bool { return def.Name.Key() == "workflow" }}

	if got := Next(defs, 0, policy); got != 4 {
		t.Fatalf("batch should skip deadline-sensitive phase and end at %d, want 4", got)
	}
	if got := Next(defs, 4, policy); got != 5 {
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
	if got := Next(defs, 0, Policy{}); got != 1 {
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

	if TimeoutRisk(def, predicted, measured) {
		t.Fatal("serialized a phase whose measured duration leaves headroom")
	}
	if !TimeoutRisk(def, predicted, nil) {
		t.Fatal("expected a prediction without history to trip the guard")
	}
}

// A phase that genuinely runs near its deadline is still kept out of batches.
func TestPhaseTimeoutRiskFiresOnMeasuredDeadlinePressure(t *testing.T) {
	def := phases.Definition{Name: phases.Name("security"), Timeout: 180 * time.Second}
	measured := func(phases.Definition) (int64, bool) { return 154_000, true }

	if !TimeoutRisk(def, nil, measured) {
		t.Fatal("did not serialize a phase measured at 154s against a 180s timeout")
	}
}

func TestPhaseTimeoutRiskAllowsMeasuredHeadroomAtCalibratedAllowance(t *testing.T) {
	def := phases.Definition{Name: phases.Name("security"), Timeout: 180 * time.Second}
	measured := func(phases.Definition) (int64, bool) { return 100_000, true }

	if TimeoutRisk(def, nil, measured) {
		t.Fatal("serialized a measured 100s phase with 80s of timeout headroom")
	}
}

// A phase with neither history nor prediction is not guessed at. Its size is
// still unknown to admission, which is what keeps it out of a batch.
func TestPhaseTimeoutRiskIsQuietWithoutAnyInput(t *testing.T) {
	def := phases.Definition{Name: phases.Name("brand-new"), Timeout: time.Minute}
	if TimeoutRisk(def, nil, func(phases.Definition) (int64, bool) { return 0, false }) {
		t.Fatal("reported deadline risk with no measurement and no prediction")
	}
}

func TestPhaseTimeoutRiskUsesPredictionWhenMeasuredHistoryIsWithheld(t *testing.T) {
	def := phases.Definition{Name: phases.Name("unit"), Timeout: time.Minute}
	predicted := map[string]int64{"unit": 50_000}
	measured := func(phases.Definition) (int64, bool) { return 0, false }
	if !TimeoutRisk(def, predicted, measured) {
		t.Fatal("did not use planner prediction after measured history was withheld")
	}
}
