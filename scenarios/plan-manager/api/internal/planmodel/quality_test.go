package planmodel

import (
	"strings"
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

func TestAssessPlanQualityRequiresCurrentCollectionBaseline(t *testing.T) {
	plan := executionGradePlan()
	plan.BaselineSet = BaselineSetIntent{}

	report := AssessPlanQuality(plan, "")

	require.False(t, report.ExecutionReady())
	requireQualityCode(t, report, "plan_missing_baseline_set")

	plan.BaselineSet = BaselineSetIntent{Compatibility: BaselineSetCompatibilityLegacy}
	report = AssessPlanQuality(plan, "")
	require.True(t, report.ExecutionReady(), "legacy is an explicit adoption path, not an omitted-baseline bypass")
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
			Strategy:     AnchorStrategyChangeBoundary,
			BaselineName: "plan-1-baseline",
		},
		BaselineSet: BaselineSetFromBoundary(ChangeBoundary{AcceptanceAllow: []string{"scenarios/plan-manager/**"}}, "plan-1-baseline"),
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

// TestAssessPlanQualityFlagsDuplicatedSkillReadPrefix pins the doubled
// `prompt-manager skill read` guard: a corrupted stored command or a skill
// Target still carrying the full command must fail quality, while the healthy
// typed shape (bare slug target + single command) stays clean.
func TestAssessPlanQualityFlagsDuplicatedSkillReadPrefix(t *testing.T) {
	base := executionGradePlan()

	hasFinding := func(p Plan, code string) bool {
		for _, f := range AssessPlanQuality(p, "").Findings {
			if f.Code == code {
				return true
			}
		}
		return false
	}

	doubledCommand := base
	doubledCommand.RelevantContext = append([]RelevantContextItem(nil), base.RelevantContext...)
	doubledCommand.RelevantContext = append(doubledCommand.RelevantContext, RelevantContextItem{
		ID:      "ctx-doubled",
		Kind:    RelevantContextSkill,
		Scope:   RelevantContextScopeGlobal,
		Label:   "scientific-debugging",
		Target:  "scientific-debugging",
		Command: "prompt-manager skill read prompt-manager skill read scientific-debugging",
	})
	if !hasFinding(doubledCommand, "context_duplicated_skill_read") {
		t.Fatal("doubled skill-read command must fail quality")
	}

	targetCarriesCommand := base
	targetCarriesCommand.RelevantContext = append([]RelevantContextItem(nil), base.RelevantContext...)
	targetCarriesCommand.RelevantContext = append(targetCarriesCommand.RelevantContext, RelevantContextItem{
		ID:     "ctx-target-command",
		Kind:   RelevantContextSkill,
		Scope:  RelevantContextScopeGlobal,
		Label:  "Scientific debugging skill",
		Target: "prompt-manager skill read scientific-debugging",
	})
	if !hasFinding(targetCarriesCommand, "context_duplicated_skill_read") {
		t.Fatal("skill target carrying the full command must fail quality")
	}

	clean := base
	clean.RelevantContext = append([]RelevantContextItem(nil), base.RelevantContext...)
	clean.RelevantContext = append(clean.RelevantContext, RelevantContextItem{
		ID:      "ctx-clean",
		Kind:    RelevantContextSkill,
		Scope:   RelevantContextScopeGlobal,
		Label:   "scientific-debugging",
		Target:  "scientific-debugging",
		Command: "prompt-manager skill read scientific-debugging",
	})
	if hasFinding(clean, "context_duplicated_skill_read") {
		t.Fatal("healthy typed skill context must not fail quality")
	}
}

// TestSingleHomeWarningsFireAndStaySoft pins the D9 single-home discipline:
// each warning fires on a restated fact, stays silent on distinct content, and
// never escalates the report to a hard failure.
func TestSingleHomeWarningsFireAndStaySoft(t *testing.T) {
	codes := func(p Plan) map[string]QualitySeverity {
		out := map[string]QualitySeverity{}
		for _, f := range AssessPlanQuality(p, "").Findings {
			out[f.Code] = f.Severity
		}
		return out
	}

	longPurpose := executionGradePlan()
	longPurpose.Purpose = strings.Repeat("word ", 140)
	got := codes(longPurpose)
	if got["purpose_over_length_target"] != QualitySeverityWarning {
		t.Fatalf("long purpose must warn, got %v", got)
	}

	dupValidation := executionGradePlan()
	dupValidation.ValidationStrategy = "Run the focused planmodel and execution suites, then the full scenario test and baseline diff."
	dupValidation.Phases[0].Validation = dupValidation.ValidationStrategy
	got = codes(dupValidation)
	if got["phase_validation_duplicates_strategy"] != QualitySeverityWarning {
		t.Fatalf("duplicated phase validation must warn, got %v", got)
	}

	dupDoD := executionGradePlan()
	dupDoD.Phases[0].Acceptance = "Incomplete plans return an invalid execution error from every entry point."
	dupDoD.DefinitionOfDone = "- All suites green.\n- Incomplete plans return an invalid execution error from every entry point."
	got = codes(dupDoD)
	if got["dod_restates_phase_acceptance"] != QualitySeverityWarning {
		t.Fatalf("DoD restating phase acceptance must warn, got %v", got)
	}

	clean := executionGradePlan()
	got = codes(clean)
	for _, code := range []string{"purpose_over_length_target", "phase_validation_duplicates_strategy", "dod_restates_phase_acceptance"} {
		if _, ok := got[code]; ok {
			t.Fatalf("clean plan must not carry %s", code)
		}
	}
	for _, p := range []Plan{longPurpose, dupValidation, dupDoD} {
		if AssessPlanQuality(p, "").HasFailures() {
			t.Fatal("single-home warnings must never be hard failures")
		}
	}
}
