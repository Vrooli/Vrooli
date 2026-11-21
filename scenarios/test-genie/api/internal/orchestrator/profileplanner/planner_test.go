package profileplanner

import (
	"strings"
	"testing"

	"test-genie/internal/orchestrator/phasepolicy"
)

func TestPlanProfileSelectsRequiredAndBudgetFit(t *testing.T) {
	estimator := NewEstimator("demo", []Sample{
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 40},
		{ScenarioName: "demo", PhaseName: "performance", Status: "passed", DurationSeconds: 200},
	})
	plan := PlanProfile(Profile{
		Name:          "quick",
		BudgetSeconds: 100,
		Strategy:      StrategyBudgetFastFeedback,
	}, []Candidate{
		{Name: "performance", TimeoutSeconds: 900, Policy: phasepolicy.BestEffortProviderPolicy(), Order: 2},
		{Name: "structure", TimeoutSeconds: 20, Policy: phasepolicy.RequiredProviderPolicy(), Order: 0},
		{Name: "unit", TimeoutSeconds: 120, Policy: phasepolicy.BestEffortProviderPolicy(), Order: 1},
	}, estimator)

	if got := decisionNames(plan.Selected); got != "structure,unit" {
		t.Fatalf("selected = %s, want structure,unit", got)
	}
	if got := decisionNames(plan.Omitted); got != "performance" {
		t.Fatalf("omitted = %s, want performance", got)
	}
	if plan.Omitted[0].Reasons[0] != ReasonOmittedBudget {
		t.Fatalf("omission reason = %#v, want budget", plan.Omitted[0].Reasons)
	}
}

func TestPlanProfileSelectsUnknownOptionalCandidatesToEarnHistory(t *testing.T) {
	estimator := NewEstimator("demo", nil)
	plan := PlanProfile(Profile{
		Name:          "quick",
		BudgetSeconds: 100,
		Strategy:      StrategyBudgetFastFeedback,
	}, []Candidate{
		{Name: "structure", TimeoutSeconds: 20, Policy: phasepolicy.RequiredProviderPolicy()},
		{Name: "security", TimeoutSeconds: 300, Policy: phasepolicy.BestEffortProviderPolicy(), Order: 1},
		{Name: "manual", TimeoutSeconds: 60, Policy: phasepolicy.Policy{
			Selection:         phasepolicy.SelectionExplicitOnly,
			ProviderReadiness: phasepolicy.ProviderReadinessBestEffort,
			ProviderLifecycle: phasepolicy.ProviderLifecycleCheckOnly,
			Freshness:         phasepolicy.FreshnessRequireReachable,
			ResultGating:      phasepolicy.ResultGatingAdvisory,
			Unavailable:       phasepolicy.UnavailableAdvisory,
		}, Order: 2},
	}, estimator)

	if got := decisionNames(plan.Selected); got != "structure,security" {
		t.Fatalf("selected = %s, want structure,security", got)
	}
	if got := reasonFor(plan.Selected, "security"); got != ReasonSelectedUnknownCost {
		t.Fatalf("security reason = %q, want selected unknown cost", got)
	}
	if got := reasonFor(plan.Omitted, "manual"); got != ReasonOmittedExplicitOnly {
		t.Fatalf("manual reason = %q, want explicit-only", got)
	}
	if plan.UnknownEstimateCount != 3 || plan.SelectedUnknownEstimates != 2 {
		t.Fatalf("unexpected unknown counts: %#v", plan)
	}
}

func TestEstimatorExcludesCensoredSamplesFromPointEstimate(t *testing.T) {
	estimator := NewEstimator("demo", []Sample{
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 8},
		{ScenarioName: "demo", PhaseName: "unit", Status: "failed", DurationSeconds: 10},
		{ScenarioName: "demo", PhaseName: "unit", Status: "timeout", DurationSeconds: 900},
		{ScenarioName: "demo", PhaseName: "unit", Status: "skipped", DurationSeconds: 700},
	})
	estimate := estimator.Estimate("unit", 120)
	if estimate.DurationSeconds != 10 || estimate.CensoredSampleCount != 1 || estimate.ExcludedSampleCount != 1 {
		t.Fatalf("censored sample changed estimate or counts: %#v", estimate)
	}
	if estimate.Confidence != EstimateConfidenceLow {
		t.Fatalf("censored bucket confidence = %q, want low", estimate.Confidence)
	}
}

func TestEstimatorUsesReliableSamplesOnlyWhenBucketIsMixed(t *testing.T) {
	samples := make([]Sample, 0, 5)
	for _, duration := range []int{10, 11, 12, 13} {
		samples = append(samples, Sample{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: duration, CPUReliability: "RELIABILITY_RELIABLE", MemoryReliability: "RELIABILITY_RELIABLE"})
	}
	samples = append(samples, Sample{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 900, CPUReliability: "RELIABILITY_BEST_EFFORT", MemoryReliability: "RELIABILITY_BEST_EFFORT"})
	estimate := NewEstimator("demo", samples).Estimate("unit", 120)
	if estimate.DurationSeconds >= 900 || estimate.ReliabilityComposition != "reliable" {
		t.Fatalf("mixed bucket used best-effort sample: %#v", estimate)
	}
}

func TestEstimatorNamesBestEffortOnlyReason(t *testing.T) {
	samples := make([]Sample, 0, 5)
	for _, duration := range []int{10, 11, 12, 13, 14} {
		samples = append(samples, Sample{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: duration, CPUReliability: "RELIABILITY_BEST_EFFORT", MemoryReliability: "RELIABILITY_BEST_EFFORT"})
	}
	estimate := NewEstimator("demo", samples).Estimate("unit", 120)
	if estimate.ReliabilityComposition != "best_effort" || estimate.LowConfidenceReason == "" || estimate.Confidence != EstimateConfidenceLow {
		t.Fatalf("best-effort estimate is not honest: %#v", estimate)
	}
}

func TestEstimatorMillisecondsAndSecondsProduceSameBudgetSelection(t *testing.T) {
	seconds := NewEstimator("demo", []Sample{
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 40},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 41},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 42},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 43},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 44},
	})
	milliseconds := NewEstimator("demo", []Sample{
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationMilliseconds: 40000},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationMilliseconds: 41000},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationMilliseconds: 42000},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationMilliseconds: 43000},
		{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationMilliseconds: 44000},
	})
	candidates := []Candidate{{Name: "unit", TimeoutSeconds: 120, Policy: phasepolicy.BestEffortProviderPolicy()}}
	left := PlanProfile(Profile{Name: "quick", BudgetSeconds: 100}, candidates, seconds)
	right := PlanProfile(Profile{Name: "quick", BudgetSeconds: 100}, candidates, milliseconds)
	if decisionNames(left.Selected) != decisionNames(right.Selected) || left.EstimatedTotalSeconds != right.EstimatedTotalSeconds {
		t.Fatalf("unit conversion changed selection: seconds=%#v milliseconds=%#v", left, right)
	}
}

func TestPlanProfileReportsRequiredOverflowAndKeepsCheapOptionalCoverage(t *testing.T) {
	estimator := NewEstimator("demo", []Sample{
		{ScenarioName: "demo", PhaseName: "required", Status: "passed", DurationSeconds: 190},
		{ScenarioName: "demo", PhaseName: "cheap", Status: "passed", DurationSeconds: 2},
		{ScenarioName: "demo", PhaseName: "expensive", Status: "passed", DurationSeconds: 200},
	})
	plan := PlanProfile(Profile{Name: "quick", BudgetSeconds: 180}, []Candidate{
		{Name: "required", Policy: phasepolicy.RequiredProviderPolicy()},
		{Name: "expensive", Policy: phasepolicy.BestEffortProviderPolicy(), Order: 1},
		{Name: "cheap", Policy: phasepolicy.BestEffortProviderPolicy(), Order: 2},
	}, estimator)
	if !plan.BudgetExceededByRequired || plan.BudgetOverflowSeconds != 10 {
		t.Fatalf("missing required overflow: %#v", plan)
	}
	if reasonFor(plan.Selected, "cheap") == "" || reasonFor(plan.Omitted, "expensive") != ReasonOmittedBudget {
		t.Fatalf("unexpected overflow selection: %#v", plan)
	}
}

func TestPlanProfileUsesMakespanWhenConcurrencyGranted(t *testing.T) {
	estimator := NewEstimator("demo", []Sample{
		{ScenarioName: "demo", PhaseName: "one", Status: "passed", DurationSeconds: 60},
		{ScenarioName: "demo", PhaseName: "two", Status: "passed", DurationSeconds: 50},
	})
	plan := PlanProfile(Profile{Name: "quick", BudgetSeconds: 70, ConcurrencyGranted: true}, []Candidate{
		{Name: "one", Policy: phasepolicy.BestEffortProviderPolicy(), ConcurrencyMode: "parallel-safe"},
		{Name: "two", Policy: phasepolicy.BestEffortProviderPolicy(), ConcurrencyMode: "parallel-safe"},
	}, estimator)
	if plan.FitMode != FitModeMakespan || plan.EstimatedTotalSeconds != 60 || len(plan.Selected) != 2 {
		t.Fatalf("unexpected makespan plan: %#v", plan)
	}
}

func decisionNames(decisions []Decision) string {
	names := make([]string, 0, len(decisions))
	for _, decision := range decisions {
		names = append(names, decision.Candidate.Name)
	}
	return strings.Join(names, ",")
}

func reasonFor(decisions []Decision, name string) string {
	for _, decision := range decisions {
		if decision.Candidate.Name == name && len(decision.Reasons) > 0 {
			return decision.Reasons[0]
		}
	}
	return ""
}
