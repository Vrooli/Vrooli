package execution

import (
	"context"
	"strings"
	"testing"
	"time"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/profileplanner"
)

type stubPlanBuilder struct {
	requests []orchestrator.SuiteExecutionRequest
	preview  *orchestrator.ExecutionPlanPreview
	err      error
}

func (s *stubPlanBuilder) PreviewExecution(req orchestrator.SuiteExecutionRequest) (*orchestrator.ExecutionPlanPreview, error) {
	s.requests = append(s.requests, req)
	return s.preview, s.err
}

type stubPhaseSampleReader struct {
	samples     []PhaseDurationSample
	planSamples []PlanDurationSample
	err         error
}

func TestMeasuredOrchestrationOverheadUsesMedianResidual(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	samples := []PlanDurationSample{
		{StartedAt: now, CompletedAt: now.Add(20 * time.Second), PhaseDurationMilliseconds: 10000},
		{StartedAt: now, CompletedAt: now.Add(30 * time.Second), PhaseDurationMilliseconds: 10000},
		{StartedAt: now, CompletedAt: now.Add(40 * time.Second), PhaseDurationMilliseconds: 10000},
	}
	if got := measuredOrchestrationOverheadSeconds(samples); got != 20 {
		t.Fatalf("measured overhead = %d, want median residual 20", got)
	}
}

func (s *stubPhaseSampleReader) ListPhaseSamples(ctx context.Context, scenario string, phaseNames []string, since time.Time, limit int) ([]PhaseDurationSample, error) {
	return s.samples, s.err
}

func (s *stubPhaseSampleReader) ListPlanSamples(ctx context.Context, scenario string, since time.Time, limit int) ([]PlanDurationSample, error) {
	return s.planSamples, s.err
}

func TestExecutionPlanServicePreviewUsesScenarioHistoryFirst(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{
			preview: &orchestrator.ExecutionPlanPreview{
				ScenarioName: "demo",
				PresetUsed:   "quick",
				Phases: []orchestrator.PlannedPhase{
					{Name: "unit", DisplayName: "Unit", Description: "Runs unit tests", TimeoutSeconds: 900},
				},
			},
		},
		&stubPhaseSampleReader{
			samples: []PhaseDurationSample{
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 32},
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 28},
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 30},
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 31},
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 29},
				{ScenarioName: "other", PhaseName: "unit", Status: "passed", DurationSeconds: 400},
			},
		},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if preview.Phases[0].EstimatedDurationSeconds != 32 {
		t.Fatalf("expected scenario P90 phase estimate 32, got %d", preview.Phases[0].EstimatedDurationSeconds)
	}
	if preview.Summary.EstimatedDurationSeconds != 32 {
		t.Fatalf("expected additive phase estimate without unmeasured overhead, got %d", preview.Summary.EstimatedDurationSeconds)
	}
	if got := preview.Phases[0].EstimateSource; got != EstimateSourceScenarioHistory {
		t.Fatalf("expected scenario history source, got %s", got)
	}
	if got := preview.Phases[0].EstimateConfidence; got != EstimateConfidenceMedium {
		t.Fatalf("expected medium confidence, got %s", got)
	}
	if got := preview.Phases[0].DisplayName; got != "Unit" {
		t.Fatalf("displayName = %q, want Unit", got)
	}
}

func TestExecutionPlanServicePreviewBlendsScenarioAndGlobalHistory(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{
			preview: &orchestrator.ExecutionPlanPreview{
				ScenarioName: "demo",
				Phases: []orchestrator.PlannedPhase{
					{Name: "integration", TimeoutSeconds: 900},
				},
			},
		},
		&stubPhaseSampleReader{
			samples: []PhaseDurationSample{
				{ScenarioName: "demo", PhaseName: "integration", Status: "passed", DurationSeconds: 120},
				{ScenarioName: "demo", PhaseName: "integration", Status: "passed", DurationSeconds: 180},
				{ScenarioName: "other-a", PhaseName: "integration", Status: "passed", DurationSeconds: 300},
				{ScenarioName: "other-b", PhaseName: "integration", Status: "passed", DurationSeconds: 360},
				{ScenarioName: "other-c", PhaseName: "integration", Status: "passed", DurationSeconds: 420},
			},
		},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	estimate := preview.Phases[0]
	if estimate.EstimateSource != EstimateSourceBlendedHistory {
		t.Fatalf("expected blended source, got %s", estimate.EstimateSource)
	}
	if estimate.EstimatedDurationSeconds <= 180 || estimate.EstimatedDurationSeconds >= 420 {
		t.Fatalf("expected blended estimate between scenario/global p75s, got %d", estimate.EstimatedDurationSeconds)
	}
}

func TestExecutionPlanServicePreviewFallsBackToGlobalHistory(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{
			preview: &orchestrator.ExecutionPlanPreview{
				ScenarioName: "demo",
				Phases: []orchestrator.PlannedPhase{
					{Name: "business", TimeoutSeconds: 600},
				},
			},
		},
		&stubPhaseSampleReader{
			samples: []PhaseDurationSample{
				{ScenarioName: "other-a", PhaseName: "business", Status: "passed", DurationSeconds: 40},
				{ScenarioName: "other-b", PhaseName: "business", Status: "passed", DurationSeconds: 44},
				{ScenarioName: "other-c", PhaseName: "business", Status: "passed", DurationSeconds: 42},
			},
		},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if got := preview.Phases[0].EstimateSource; got != EstimateSourceGlobalHistory {
		t.Fatalf("expected global history source, got %s", got)
	}
	if got := preview.Phases[0].EstimatedDurationSeconds; got != 44 {
		t.Fatalf("expected global p75 44, got %d", got)
	}
}

func TestExecutionPlanServicePreviewPrefersExactComparableFullRun(t *testing.T) {
	now := time.Now().UTC()
	svc := NewExecutionPlanService(
		&stubPlanBuilder{preview: &orchestrator.ExecutionPlanPreview{
			ScenarioName: "demo", PhaseSetDigest: "phase-set:current", DescriptorSnapshotDigest: "ds:current", ConfigurationFingerprint: "execution-config:current",
			Phases: []orchestrator.PlannedPhase{{Name: "structure", TimeoutSeconds: 60}, {Name: "unit", TimeoutSeconds: 600}},
		}},
		&stubPhaseSampleReader{planSamples: []PlanDurationSample{
			{ScenarioName: "demo", PhaseSetDigest: "phase-set:old", DescriptorSnapshotDigest: "ds:current", ConfigurationFingerprint: "execution-config:current", DurationSeconds: 12, CompletedAt: now},
			{ScenarioName: "demo", PhaseSetDigest: "phase-set:current", DescriptorSnapshotDigest: "ds:old", ConfigurationFingerprint: "execution-config:current", DurationSeconds: 12, CompletedAt: now},
			{ScenarioName: "demo", PhaseSetDigest: "phase-set:current", DescriptorSnapshotDigest: "ds:current", ConfigurationFingerprint: "execution-config:current", TerminalOutcome: "passed", DurationSeconds: 300, CompletedAt: now},
			// A timeout is retained as slow/censored full-run evidence rather than discarded.
			{ScenarioName: "demo", PhaseSetDigest: "phase-set:current", DescriptorSnapshotDigest: "ds:current", ConfigurationFingerprint: "execution-config:current", TerminalOutcome: "timeout", DurationSeconds: 420, CompletedAt: now},
		}},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if preview.Summary.EstimateMode != "comparable_full_run" {
		t.Fatalf("estimate mode = %q, want comparable_full_run", preview.Summary.EstimateMode)
	}
	if preview.Summary.EstimateSampleSize != 2 || preview.Summary.EstimatedDurationSeconds != 420 {
		t.Fatalf("exact P90 full-run estimate = %#v, want 420 from two comparable samples", preview.Summary)
	}
	if preview.Summary.OrchestrationOverheadSeconds != 0 {
		t.Fatalf("exact full-run estimate must not add synthetic overhead: %#v", preview.Summary)
	}
}

func TestExecutionPlanServicePreviewMismatchedComparableHistoryFallsBackAdditive(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{preview: &orchestrator.ExecutionPlanPreview{
			ScenarioName: "demo", PhaseSetDigest: "phase-set:current", DescriptorSnapshotDigest: "ds:current", ConfigurationFingerprint: "execution-config:current",
			Phases: []orchestrator.PlannedPhase{{Name: "unit", TimeoutSeconds: 100}},
		}},
		&stubPhaseSampleReader{planSamples: []PlanDurationSample{{ScenarioName: "demo", PhaseSetDigest: "phase-set:current", DescriptorSnapshotDigest: "ds:changed", ConfigurationFingerprint: "execution-config:current", DurationSeconds: 2, CompletedAt: time.Now().UTC()}}},
	)
	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if preview.Summary.EstimateMode != "additive_phase_history" || preview.Summary.OrchestrationOverheadSeconds != 0 {
		t.Fatalf("descriptor mismatch must fail closed into additive estimate: %#v", preview.Summary)
	}
}

func TestExecutionPlanServicePreviewMarksUnknownHistoryExplicitly(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{
			preview: &orchestrator.ExecutionPlanPreview{
				ScenarioName: "demo",
				Phases: []orchestrator.PlannedPhase{
					{Name: "playbooks", TimeoutSeconds: 900},
				},
			},
		},
		&stubPhaseSampleReader{},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if got := preview.Phases[0].EstimateSource; got != EstimateSourceUnknown {
		t.Fatalf("expected unknown estimate source, got %s", got)
	}
	if got := preview.Phases[0].EstimatedDurationSeconds; got != 900 {
		t.Fatalf("expected conservative timeout budget 900, got %d", got)
	}
	if !preview.Phases[0].EstimateUnknown {
		t.Fatalf("expected unknown estimate flag")
	}
	if preview.Summary.TimeoutSeconds != 900 || preview.Summary.EstimatedDurationSeconds != 900 {
		t.Fatalf("unexpected summary: %#v", preview.Summary)
	}
}

func TestExecutionPlanServicePreviewRetainsFailedDurationAsSlowEvidence(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{
			preview: &orchestrator.ExecutionPlanPreview{
				ScenarioName: "demo",
				Phases: []orchestrator.PlannedPhase{
					{Name: "unit", TimeoutSeconds: 900},
				},
			},
		},
		&stubPhaseSampleReader{
			samples: []PhaseDurationSample{
				{ScenarioName: "demo", PhaseName: "unit", Status: "failed", DurationSeconds: 1},
				{ScenarioName: "demo", PhaseName: "unit", Status: "skipped", DurationSeconds: 2},
				{ScenarioName: "demo", PhaseName: "unit", Status: "provider_unavailable", DurationSeconds: 3},
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 0},
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 20},
			},
		},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	estimate := preview.Phases[0]
	if estimate.EstimateSource != EstimateSourceScenarioHistory {
		t.Fatalf("estimate source = %s, want scenario history", estimate.EstimateSource)
	}
	if estimate.EstimateSampleSize != 2 || estimate.EstimatedDurationSeconds != 20 {
		t.Fatalf("unexpected filtered estimate: %#v", estimate)
	}
}

func TestExecutionPlanServicePreviewPlansQuickFromComprehensiveCandidates(t *testing.T) {
	builder := &stubPlanBuilder{
		preview: &orchestrator.ExecutionPlanPreview{
			ScenarioName: "demo",
			PresetUsed:   "comprehensive",
			Phases: []orchestrator.PlannedPhase{
				{
					Name:            "structure",
					DisplayName:     "Structure",
					TimeoutSeconds:  20,
					SelectionStatus: "selected",
					Policy:          phasepolicy.RequiredProviderPolicy(),
				},
				{
					Name:            "unit",
					DisplayName:     "Unit",
					TimeoutSeconds:  120,
					SelectionStatus: "selected",
					Policy:          phasepolicy.BestEffortProviderPolicy(),
				},
				{
					Name:            "performance",
					DisplayName:     "Performance",
					TimeoutSeconds:  900,
					SelectionStatus: "selected",
					Policy:          phasepolicy.BestEffortProviderPolicy(),
				},
				{
					Name:            "security",
					DisplayName:     "Security",
					TimeoutSeconds:  300,
					SelectionStatus: "selected",
					Policy:          phasepolicy.BestEffortProviderPolicy(),
				},
			},
		},
	}
	svc := NewExecutionPlanService(
		builder,
		&stubPhaseSampleReader{
			samples: []PhaseDurationSample{
				{ScenarioName: "demo", PhaseName: "unit", Status: "passed", DurationSeconds: 50},
				{ScenarioName: "demo", PhaseName: "performance", Status: "passed", DurationSeconds: 200},
			},
		},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{
		ScenarioName: "demo",
		Preset:       "quick",
	})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(builder.requests) != 1 || builder.requests[0].Preset != "comprehensive" {
		t.Fatalf("quick planning must score comprehensive candidates, got requests %#v", builder.requests)
	}
	if preview.PresetUsed != "quick" {
		t.Fatalf("presetUsed = %q, want quick", preview.PresetUsed)
	}
	if preview.Profile == nil || preview.Profile.Name != "quick" || preview.Profile.Strategy != string(profileplanner.StrategyBudgetFastFeedback) {
		t.Fatalf("unexpected profile metadata: %#v", preview.Profile)
	}
	if got := phaseNames(preview.Phases); got != "structure,unit,security" {
		t.Fatalf("selected phases = %s, want structure,unit,security", got)
	}
	if got := phaseNames(preview.OmittedPhases); got != "performance" {
		t.Fatalf("omitted phases = %s, want performance", got)
	}
	if got := selectionReason(preview.Phases, "security"); got != profileplanner.ReasonSelectedUnknownCost {
		t.Fatalf("security selection reason = %q, want unknown cost", got)
	}
	if got := omissionReason(preview.OmittedPhases, "performance"); got != profileplanner.ReasonOmittedBudget {
		t.Fatalf("performance omission reason = %q, want budget", got)
	}
	quickProfile, _ := phases.AdaptiveProfile("quick")
	if preview.Summary.BudgetSeconds != quickProfile.BudgetSeconds {
		t.Fatalf("budget = %d, want %d", preview.Summary.BudgetSeconds, quickProfile.BudgetSeconds)
	}
	if preview.Summary.EstimatedDurationSeconds != 370 {
		t.Fatalf("estimated total = %d, want 370", preview.Summary.EstimatedDurationSeconds)
	}
}

func omissionReason(phases []PlannedPhase, name string) string {
	for _, phase := range phases {
		if phase.Name == name && len(phase.OmissionReasons) > 0 {
			return phase.OmissionReasons[0]
		}
	}
	return ""
}

func selectionReason(phases []PlannedPhase, name string) string {
	for _, phase := range phases {
		if phase.Name == name && len(phase.SelectionReasons) > 0 {
			return phase.SelectionReasons[0]
		}
	}
	return ""
}

func phaseNames(phases []PlannedPhase) string {
	names := make([]string, 0, len(phases))
	for _, phase := range phases {
		names = append(names, phase.Name)
	}
	return strings.Join(names, ",")
}
