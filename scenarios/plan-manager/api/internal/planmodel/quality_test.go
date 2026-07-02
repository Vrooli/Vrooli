package planmodel

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAssessPlanQualityFailsZeroPhaseImportedSkeleton(t *testing.T) {
	plan := executionGradePlan()
	plan.Phases = nil
	plan.ImportProvenance = &ImportProvenance{SourcePath: "docs/plans/legacy.md"}

	report := AssessPlanQuality(plan, "")

	require.Equal(t, QualityStatusFail, report.Status)
	require.False(t, report.ExecutionReady())
	requireQualityCode(t, report, "plan_missing_phases")
	requireQualityCode(t, report, "legacy_import_requires_review")
}

func TestAssessPlanQualityRequiresExecutionGradePlanFields(t *testing.T) {
	report := AssessPlanQuality(Plan{Title: "Thin plan"}, "")

	require.Equal(t, QualityStatusFail, report.Status)
	for _, code := range []string{
		"plan_missing_purpose",
		"plan_missing_problem",
		"plan_missing_target_outcome",
		"plan_missing_scope",
		"plan_missing_technical_approach",
		"plan_missing_validation_strategy",
		"plan_missing_definition_of_done",
		"plan_missing_change_boundary",
		"plan_missing_regression_anchor",
		"plan_missing_references",
		"plan_missing_global_context",
		"plan_missing_skill_context",
		"plan_missing_phases",
	} {
		requireQualityCode(t, report, code)
	}
}

func TestAssessPlanQualityAcceptsExplicitNoContextAndNoCodeReasons(t *testing.T) {
	report := AssessPlanQuality(executionGradePlan(), "")

	require.Equal(t, QualityStatusPass, report.Status)
	require.True(t, report.ExecutionReady())
	require.Empty(t, report.Findings)
}

func TestAssessPlanQualityFailsIncompletePhase(t *testing.T) {
	plan := executionGradePlan()
	plan.Phases[0] = Phase{ID: "phase-1", Order: 1, Title: "Thin"}

	report := AssessPlanQuality(plan, "")

	require.Equal(t, QualityStatusFail, report.Status)
	for _, code := range []string{
		"phase_missing_intent",
		"phase_missing_steps",
		"phase_missing_validation",
		"phase_missing_acceptance",
		"phase_missing_references",
		"phase_missing_context",
	} {
		requireQualityCode(t, report, code)
	}
}

func requireQualityCode(t *testing.T, report QualityReport, code string) {
	t.Helper()
	for _, finding := range report.Findings {
		if finding.Code == code {
			return
		}
	}
	t.Fatalf("quality code %q not found in %#v", code, report.Findings)
}

func executionGradePlan() Plan {
	return Plan{
		ID:                 "plan-1",
		Slug:               "plan-1",
		Title:              "Execution grade",
		Purpose:            "Improve Plan Manager quality gates.",
		ProblemStatement:   "Incomplete plans can reach execution.",
		TargetOutcome:      "Execution receives only repairable structured plans.",
		Scope:              "Plan Manager plan quality and execution gate.",
		TechnicalApproach:  "Use the shared planmodel quality kernel.",
		ValidationStrategy: "Run focused unit tests.",
		DefinitionOfDone:   "Quality failures block execution.",
		Constraints:        "NO_CODE_REFS: unit fixture does not need connected plan refs.",
		ChangeBoundary: ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/plan-manager/**"},
		},
		RegressionAnchor: RegressionAnchor{
			Strategy: AnchorStrategyChangeBoundary,
		},
		RelevantContext: []RelevantContextItem{{
			ID:           "ctx-global",
			Kind:         RelevantContextNote,
			Scope:        RelevantContextScopeGlobal,
			Label:        "NO_CONTEXT: unit fixture has no extra plan-wide setup.",
			Instruction:  "NO_CONTEXT: unit fixture has no extra plan-wide setup.",
			Required:     true,
			RepeatPolicy: RelevantContextOncePerExecution,
			Source:       RelevantContextSourceAuthored,
			Status:       RelevantContextStatusReady,
		}},
		Phases: []Phase{{
			ID:         "phase-1",
			Order:      1,
			Title:      "Gate execution",
			Intent:     "Reject incomplete plans before execution.",
			Steps:      []string{"Add quality checks", "Gate execution start"},
			Validation: "go test ./internal/planmodel ./internal/execution",
			Acceptance: "Incomplete plans return an invalid execution error.",
			Reminders:  []string{"NO_CODE_REFS: unit fixture has no phase refs."},
			RelevantContext: []RelevantContextItem{{
				ID:           "ctx-phase",
				Kind:         RelevantContextNote,
				Scope:        RelevantContextScopePhase,
				PhaseID:      "phase-1",
				Label:        "NO_CONTEXT: phase uses local test fixture only.",
				Instruction:  "NO_CONTEXT: phase uses local test fixture only.",
				Required:     true,
				RepeatPolicy: RelevantContextPhaseEntry,
				Source:       RelevantContextSourceAuthored,
				Status:       RelevantContextStatusReady,
			}},
			Status: PhaseStatusTodo,
		}},
	}
}
