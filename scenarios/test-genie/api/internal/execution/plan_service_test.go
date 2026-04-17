package execution

import (
	"context"
	"test-genie/internal/orchestrator"
	"testing"
	"time"
)

type stubPlanBuilder struct {
	preview *orchestrator.ExecutionPlanPreview
	err     error
}

func (s *stubPlanBuilder) PreviewExecution(req orchestrator.SuiteExecutionRequest) (*orchestrator.ExecutionPlanPreview, error) {
	return s.preview, s.err
}

type stubPhaseSampleReader struct {
	samples []PhaseDurationSample
	err     error
}

func (s *stubPhaseSampleReader) ListPhaseSamples(ctx context.Context, phaseNames []string, since time.Time, limit int) ([]PhaseDurationSample, error) {
	return s.samples, s.err
}

func TestExecutionPlanServicePreviewUsesScenarioHistoryFirst(t *testing.T) {
	svc := NewExecutionPlanService(
		&stubPlanBuilder{
			preview: &orchestrator.ExecutionPlanPreview{
				ScenarioName: "demo",
				PresetUsed:   "quick",
				Phases: []orchestrator.PlannedPhase{
					{Name: "unit", Description: "Runs unit tests", TimeoutSeconds: 900},
				},
			},
		},
		&stubPhaseSampleReader{
			samples: []PhaseDurationSample{
				{ScenarioName: "demo", PhaseName: "unit", DurationSeconds: 32},
				{ScenarioName: "demo", PhaseName: "unit", DurationSeconds: 28},
				{ScenarioName: "demo", PhaseName: "unit", DurationSeconds: 30},
				{ScenarioName: "demo", PhaseName: "unit", DurationSeconds: 31},
				{ScenarioName: "demo", PhaseName: "unit", DurationSeconds: 29},
				{ScenarioName: "other", PhaseName: "unit", DurationSeconds: 400},
			},
		},
	)

	preview, err := svc.Preview(context.Background(), orchestrator.SuiteExecutionRequest{ScenarioName: "demo"})
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if preview.Summary.EstimatedDurationSeconds != 30 {
		t.Fatalf("expected scenario median estimate 30, got %d", preview.Summary.EstimatedDurationSeconds)
	}
	if got := preview.Phases[0].EstimateSource; got != EstimateSourceScenarioHistory {
		t.Fatalf("expected scenario history source, got %s", got)
	}
	if got := preview.Phases[0].EstimateConfidence; got != EstimateConfidenceMedium {
		t.Fatalf("expected medium confidence, got %s", got)
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
				{ScenarioName: "demo", PhaseName: "integration", DurationSeconds: 120},
				{ScenarioName: "demo", PhaseName: "integration", DurationSeconds: 180},
				{ScenarioName: "other-a", PhaseName: "integration", DurationSeconds: 300},
				{ScenarioName: "other-b", PhaseName: "integration", DurationSeconds: 360},
				{ScenarioName: "other-c", PhaseName: "integration", DurationSeconds: 420},
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
	if estimate.EstimatedDurationSeconds <= 180 || estimate.EstimatedDurationSeconds >= 360 {
		t.Fatalf("expected blended estimate between scenario/global medians, got %d", estimate.EstimatedDurationSeconds)
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
				{ScenarioName: "other-a", PhaseName: "business", DurationSeconds: 40},
				{ScenarioName: "other-b", PhaseName: "business", DurationSeconds: 44},
				{ScenarioName: "other-c", PhaseName: "business", DurationSeconds: 42},
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
	if got := preview.Phases[0].EstimatedDurationSeconds; got != 42 {
		t.Fatalf("expected global median 42, got %d", got)
	}
}

func TestExecutionPlanServicePreviewFallsBackToTimeoutBudget(t *testing.T) {
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
	if got := preview.Phases[0].EstimateSource; got != EstimateSourceTimeoutFallback {
		t.Fatalf("expected timeout fallback, got %s", got)
	}
	if got := preview.Phases[0].EstimatedDurationSeconds; got != 900 {
		t.Fatalf("expected timeout fallback estimate 900, got %d", got)
	}
	if preview.Summary.TimeoutSeconds != 900 || preview.Summary.EstimatedDurationSeconds != 900 {
		t.Fatalf("unexpected summary: %#v", preview.Summary)
	}
}
