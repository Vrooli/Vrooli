package execution

import (
	"strings"
	"testing"
)

// readyScenario builds a scenario that passed restart, health, and review so
// that, absent a baseline gate, summarizeFinalization classifies it ready.
func readyScenario(name string, diff *BaselineDiffResult) ScenarioFinalization {
	return ScenarioFinalization{
		ScenarioName: name,
		Restart:      RestartResult{Status: FinalizationStatusCompleted},
		Health:       HealthCheckResult{Status: FinalizationStatusCompleted, SchemaValid: true},
		Review: ScenarioReviewStep{
			Status: FinalizationStatusCompleted,
			Result: &ReviewResult{Classification: FinalizationAggregateReady, Summary: "looks good"},
		},
		BaselineDiff: diff,
	}
}

func regressionDiff(name string) *BaselineDiffResult {
	return &BaselineDiffResult{
		ScenarioName:      name,
		Verdict:           baselineVerdictRegression,
		ExitCode:          1,
		Comparable:        true,
		RegressedSurfaces: []string{"tests"},
		Regressions:       []SurfaceFinding{{Surface: "tests", Detail: "TestNewlyBroken"}},
	}
}

// A genuine regression must gate the outcome to needs_work / actionable even
// when the absolute review came back ready — the core of plan P6 §200-201.
func TestSummarizeFinalization_RegressionGatesWhenEnabled(t *testing.T) {
	fin := Finalization{
		Status:    FinalizationStatusCompleted,
		Scenarios: []ScenarioFinalization{readyScenario("alpha", regressionDiff("alpha"))},
	}

	classification, summary, actionable := summarizeFinalization(fin, true)

	if !actionable {
		t.Fatalf("regression with gate enabled must be actionable, got actionable=false (summary=%q)", summary)
	}
	if classification != FinalizationAggregateNeedsWork {
		t.Errorf("classification = %q, want %q", classification, FinalizationAggregateNeedsWork)
	}
	if !strings.Contains(summary, "introduced 1 regression") {
		t.Errorf("summary should attribute the regression, got %q", summary)
	}
}

// With the gate disabled the regression is observed (still recorded/warned
// elsewhere) but does not change the verdict — the observe-only rollout lever.
func TestSummarizeFinalization_RegressionObservedNotGatedWhenDisabled(t *testing.T) {
	fin := Finalization{
		Status:    FinalizationStatusCompleted,
		Scenarios: []ScenarioFinalization{readyScenario("alpha", regressionDiff("alpha"))},
	}

	classification, _, actionable := summarizeFinalization(fin, false)

	if actionable {
		t.Fatalf("regression with gate disabled must not be actionable")
	}
	if classification != FinalizationAggregateReady {
		t.Errorf("classification = %q, want %q", classification, FinalizationAggregateReady)
	}
}

// not-comparable is not attributable to this change, so it must never gate.
func TestSummarizeFinalization_NotComparableDoesNotGate(t *testing.T) {
	fin := Finalization{
		Status:    FinalizationStatusCompleted,
		Scenarios: []ScenarioFinalization{readyScenario("alpha", notComparableDiff("alpha"))},
	}

	classification, _, actionable := summarizeFinalization(fin, true)

	if actionable {
		t.Fatalf("not_comparable verdict must not gate the outcome")
	}
	if classification != FinalizationAggregateReady {
		t.Errorf("classification = %q, want %q", classification, FinalizationAggregateReady)
	}
}

// A clean before/after diff leaves a ready scenario ready.
func TestSummarizeFinalization_CleanDiffStaysReady(t *testing.T) {
	clean := &BaselineDiffResult{ScenarioName: "alpha", Verdict: baselineVerdictClean, Comparable: true}
	fin := Finalization{
		Status:    FinalizationStatusCompleted,
		Scenarios: []ScenarioFinalization{readyScenario("alpha", clean)},
	}

	classification, _, actionable := summarizeFinalization(fin, true)

	if actionable {
		t.Fatalf("clean diff must not gate")
	}
	if classification != FinalizationAggregateReady {
		t.Errorf("classification = %q, want %q", classification, FinalizationAggregateReady)
	}
}

// A nil BaselineDiff (feature off / no pre-exec baseline) is inert: the gate
// only ever fires on a recorded regression verdict.
func TestSummarizeFinalization_NilDiffIsInert(t *testing.T) {
	fin := Finalization{
		Status:    FinalizationStatusCompleted,
		Scenarios: []ScenarioFinalization{readyScenario("alpha", nil)},
	}

	classification, _, actionable := summarizeFinalization(fin, true)

	if actionable || classification != FinalizationAggregateReady {
		t.Fatalf("nil diff must be inert, got actionable=%v classification=%q", actionable, classification)
	}
}
