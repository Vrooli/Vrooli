package readiness

import (
	"context"
	"errors"
	"testing"

	planmodel "plan-manager/internal/planmodel"
)

type fakeCommandValidator struct {
	result CommandResult
	err    error
}

func (f fakeCommandValidator) ValidateCommandReference(context.Context, CommandRequest) (CommandResult, error) {
	return f.result, f.err
}

type fakeReferenceResolver struct {
	resolution planmodel.ReferenceResolution
	err        error
}

func (f fakeReferenceResolver) Resolve(_ context.Context, ref planmodel.Reference) (planmodel.Reference, error) {
	ref.Resolution = f.resolution
	if f.resolution == planmodel.ResolutionMissing {
		ref.Note = "target missing"
	}
	return ref, f.err
}

func TestEvaluateDeterministicModeReportsPlanQualityFailures(t *testing.T) {
	p := readyPlan()
	p.Phases[0].Steps = nil
	p.Phases[0].Validation = ""

	got := Evaluate(context.Background(), p, Options{Mode: DeterministicMode()})

	if got.Verdict != VerdictFail {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictFail)
	}
	requireFinding(t, got.Findings, SourceQuality, "phase_missing_steps", "phase.ph1.steps")
	requireFinding(t, got.Findings, SourceQuality, "phase_missing_validation", "phase.ph1.validation")
}

func TestEvaluateDeterministicModeReportsRelevantContextStructure(t *testing.T) {
	p := readyPlan()
	p.RelevantContext = append(p.RelevantContext, planmodel.RelevantContextItem{
		Kind:     planmodel.RelevantContextCommand,
		Scope:    planmodel.RelevantContextScopeGlobal,
		Required: true,
	})

	got := Evaluate(context.Background(), p, Options{Mode: DeterministicMode()})

	if got.Verdict != VerdictFail {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictFail)
	}
	requireFinding(t, got.Findings, SourceStructure, "missing_repeat_policy", "plan.relevant_context[2]")
	requireFinding(t, got.Findings, SourceStructure, "missing_context_payload", "plan.relevant_context[2]")
	requireFinding(t, got.Findings, SourceStructure, "missing_context_reason", "plan.relevant_context[2].reason")
	requireFinding(t, got.Findings, SourceStructure, "missing_context_instruction", "plan.relevant_context[2].instruction")
	requireFinding(t, got.Findings, SourceStructure, "missing_context_command", "plan.relevant_context[2].command")
}

func TestEvaluatePreflightModeRunsExternalChecks(t *testing.T) {
	p := readyPlan()
	p.Purpose = "Run `cli:vrooli scenario tost plan-manager` before handoff."
	p.RelevantContext = append(p.RelevantContext, planmodel.RelevantContextItem{
		Kind:         planmodel.RelevantContextDoc,
		Scope:        planmodel.RelevantContextScopeGlobal,
		Target:       "docs/missing.md",
		Label:        "Missing doc",
		Instruction:  "Inspect the missing doc.",
		Reason:       "preflight fixture",
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
	})

	got := Evaluate(context.Background(), p, Options{
		Mode: PreflightMode(),
		CommandValidator: fakeCommandValidator{result: CommandResult{
			Verdict:         "invalid",
			ValidationLevel: "argument_shape_validated",
			Issues:          []CommandIssue{{Code: "unknown_command", Message: "unknown command"}},
		}},
		ReferenceResolver: fakeReferenceResolver{resolution: planmodel.ResolutionMissing},
	})

	if got.Verdict != VerdictFail {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictFail)
	}
	requireFinding(t, got.Findings, SourceCommandReference, "", "plan.purpose")
	requireFinding(t, got.Findings, SourceContextReference, "context_reference_unresolved", "plan.relevant_context[2].target")
}

func TestEvaluatePreflightModeDegradesUnavailableDependencies(t *testing.T) {
	p := readyPlan()
	p.Purpose = "Run `cli:vrooli scenario tost plan-manager` before handoff."
	p.RelevantContext = append(p.RelevantContext, planmodel.RelevantContextItem{
		Kind:         planmodel.RelevantContextDoc,
		Scope:        planmodel.RelevantContextScopeGlobal,
		Target:       "docs/missing.md",
		Label:        "Missing doc",
		Instruction:  "Inspect the missing doc.",
		Reason:       "preflight fixture",
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
	})

	got := Evaluate(context.Background(), p, Options{
		Mode:              PreflightMode(),
		CommandValidator:  fakeCommandValidator{err: errors.New("down")},
		ReferenceResolver: fakeReferenceResolver{err: errors.New("down")},
	})

	if got.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %s, want %s", got.Verdict, VerdictUnknown)
	}
	requireDependencyStatus(t, got.Findings, SourceCommandReference, DependencyUnavailable)
	requireDependencyStatus(t, got.Findings, SourceContextReference, DependencyUnavailable)
}

func readyPlan() planmodel.Plan {
	boundary := planmodel.ChangeBoundary{
		AcceptanceAllow: []string{"scenarios/plan-manager/**"},
	}
	return planmodel.Plan{
		ID:                 "p1",
		Slug:               "p1",
		Title:              "Readiness fixture",
		Purpose:            "Validate one shared readiness contract.",
		ProblemStatement:   "Draft and persisted validation can drift.",
		TargetOutcome:      "Both surfaces report the same deterministic readiness findings.",
		Scope:              "Plan Manager readiness tests.",
		TechnicalApproach:  "Use the shared readiness evaluator.",
		ValidationStrategy: "Run readiness unit tests.",
		DefinitionOfDone:   "Readiness findings are deterministic.",
		Constraints:        "NO_CODE_REFS: readiness unit fixture has no plan-level code references.",
		ChangeBoundary:     boundary,
		BaselineSet:        planmodel.BaselineSetFromBoundary(boundary, "p1-baseline"),
		RegressionAnchor: planmodel.RegressionAnchor{
			Strategy:     planmodel.AnchorStrategyChangeBoundary,
			BaselineName: "p1-baseline",
		},
		RelevantContext: []planmodel.RelevantContextItem{
			noContextItem(planmodel.RelevantContextScopeGlobal, ""),
			noSkillContextItem(),
		},
		Phases: []planmodel.Phase{{
			ID:              "ph1",
			Order:           1,
			Title:           "Implement readiness",
			Intent:          "Exercise readiness behavior.",
			Steps:           []string{"Run the evaluator."},
			Validation:      "go test ./internal/readiness",
			Acceptance:      "The readiness fixture passes.",
			Reminders:       []string{"NO_CODE_REFS: readiness unit fixture has no phase code references."},
			RelevantContext: []planmodel.RelevantContextItem{noContextItem(planmodel.RelevantContextScopePhase, "ph1")},
		}},
	}
}

func noContextItem(scope planmodel.RelevantContextScope, phaseID string) planmodel.RelevantContextItem {
	return planmodel.RelevantContextItem{
		ID:           "ctx-no-context-" + string(scope) + phaseID,
		Kind:         planmodel.RelevantContextNote,
		Scope:        scope,
		PhaseID:      phaseID,
		Label:        "NO_CONTEXT: readiness unit fixture has no setup.",
		Instruction:  "NO_CONTEXT: readiness unit fixture has no setup.",
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
		Source:       planmodel.RelevantContextSourceAuthored,
		Status:       planmodel.RelevantContextStatusReady,
	}
}

func noSkillContextItem() planmodel.RelevantContextItem {
	return planmodel.RelevantContextItem{
		ID:           "ctx-no-skill",
		Kind:         planmodel.RelevantContextNote,
		Scope:        planmodel.RelevantContextScopeGlobal,
		Label:        "NO_SKILL_CONTEXT: readiness unit fixture has no internal skill setup.",
		Instruction:  "NO_SKILL_CONTEXT: readiness unit fixture has no internal skill setup.",
		Required:     true,
		RepeatPolicy: planmodel.RelevantContextOncePerExecution,
		Source:       planmodel.RelevantContextSourceAuthored,
		Status:       planmodel.RelevantContextStatusReady,
	}
}

func requireFinding(t *testing.T, findings []Finding, source, code, location string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Source != source {
			continue
		}
		if code != "" && finding.Code != code && !contains(finding.IssueCodes, code) {
			continue
		}
		if finding.Location == location {
			return
		}
	}
	t.Fatalf("missing finding source=%q code=%q location=%q in %#v", source, code, location, findings)
}

func requireDependencyStatus(t *testing.T, findings []Finding, source, status string) {
	t.Helper()
	for _, finding := range findings {
		if finding.Source == source && finding.DependencyStatus == status {
			return
		}
	}
	t.Fatalf("missing finding source=%q dependency_status=%q in %#v", source, status, findings)
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
