package plans_test

import (
	"strings"
	"testing"

	"plan-manager/internal/planmodel"
	"plan-manager/internal/plans"
)

// comprehensivePlan builds a plan exercising the full professional structure for
// render/parse golden coverage. Status is draft and content-hash is empty so the
// rendered title line round-trips (those are computed, not parsed back).
func comprehensivePlan() plans.Plan {
	return plans.Plan{
		Title:                "Structured rendered plans",
		Status:               plans.PlanStatusDraft,
		Purpose:              "Make plan-manager plans good enough to replace the legacy skill.",
		ProblemStatement:     "The current model is too thin for a human reviewer to trust.",
		TargetOutcome:        "A rendered plan a human can review without reading JSON.",
		Scope:                "In scope: model + render + wizard. Out of scope: consumer inversion.",
		NonGoals:             "No multi-user editor.",
		Assumptions:          "Baseline captured before changes.",
		TechnicalApproach:    "Model-first contract change, then wire every consumer through it.",
		Constraints:          "Keep fields concise for small agents.",
		ProhibitedApproaches: "Do not clone the legacy 13-section format.",
		WorkPosture:          plans.WorkPostureGreenfield,
		WorkPostureSource:    plans.WorkPostureSourceServiceMaturity,
		WorkPostureDetail:    `Scenario "plan-manager" maturity is greenfield.`,
		References: []plans.Reference{
			{Kind: plans.ReferenceCode, Target: "scenarios/plan-manager/api/internal/plans/render.go"},
		},
		RegressionAnchor: plans.RegressionAnchor{
			Strategy:     "scenario_baseline",
			Scenario:     "plan-manager",
			BaselineName: "plan-manager-structured-rendered-plans",
			Commands: []string{
				"git-control-tower baseline diff --scenario plan-manager --name plan-manager-structured-rendered-plans",
			},
		},
		ValidationStrategy:      "Run focused Go tests plus the scenario test; compare against the baseline.",
		FinalValidationCommands: []string{"vrooli scenario test plan-manager"},
		DefinitionOfDone:        "Golden tests pass; rendered plan is reviewable.",
		RisksHazards:            "Too many fields makes the wizard heavy.",
		Phases: []plans.Phase{
			{
				Title:           "Contract",
				Intent:          "Lock the proto + model.",
				Status:          plans.PhaseStatusTodo,
				AffectedAreas:   []string{"model.proto", "planmodel/types.go"},
				Steps:           []string{"Add proto fields", "Add Go fields", "Regenerate"},
				ExpectedOutputs: []string{"Generated proto compiles"},
				Validation:      "go build ./... and go test ./internal/planproto",
				Acceptance:      "Generated code compiles and tests pass.",
				RisksHazards:    []string{"field-number churn"},
				HandoffNotes:    "Phase 2 must finish before the renderer phase.",
				References: []plans.Reference{
					{Kind: plans.ReferenceCode, Target: "packages/proto/schemas/plan-manager/v1/shared/model.proto"},
				},
			},
			{
				Title:           "Renderer",
				Intent:          "Project the new fields deterministically.",
				Status:          plans.PhaseStatusTodo,
				AffectedAreas:   []string{"plans/render.go"},
				Steps:           []string{"Add Work Posture section", "Add new field sections"},
				ExpectedOutputs: []string{"Rendered markdown shows Work Posture"},
				Validation:      "go test ./internal/plans",
				Acceptance:      "Rendered markdown is comprehensive.",
			},
		},
	}
}

// TestRenderParseRenderIdempotent asserts render -> parse -> render is a fixed
// point: nothing the renderer emits is lost or reshaped by the parser. This is
// the core drift guard between renderer and parser.
func TestRenderParseRenderIdempotent(t *testing.T) {
	p := comprehensivePlan()
	md1 := plans.RenderMarkdown(p)

	parsed, err := planmodel.ParsePlanMarkdown(md1)
	if err != nil {
		t.Fatalf("parse rendered markdown: %v\n---\n%s", err, md1)
	}
	md2 := plans.RenderMarkdown(parsed)
	if md1 != md2 {
		t.Fatalf("render -> parse -> render not idempotent:\n--- md1 ---\n%s\n--- md2 ---\n%s", md1, md2)
	}
}

// TestRenderShowsWorkPostureAndGreenfieldBlock asserts the Work Posture section
// and the exact Greenfield block are always present in a greenfield render.
func TestRenderShowsWorkPostureAndGreenfieldBlock(t *testing.T) {
	md := plans.RenderMarkdown(comprehensivePlan())
	for _, want := range []string{
		"## Work Posture",
		"- Posture: **greenfield**",
		"**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables.",
		"## Problem / Need",
		"## Target Outcome",
		"## Technical Approach",
		"## Validation Strategy",
		"**Ordered Steps:**",
		"**Affected Areas:**",
		"**Phase Validation:**",
	} {
		if !strings.Contains(md, want) {
			t.Errorf("rendered markdown missing %q\n---\n%s", want, md)
		}
	}
}

// TestRenderParseRecoversNewFields asserts the parser recovers every new plan and
// phase field the renderer emits.
func TestRenderParseRecoversNewFields(t *testing.T) {
	p := comprehensivePlan()
	parsed, err := planmodel.ParsePlanMarkdown(plans.RenderMarkdown(p))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.ProblemStatement != p.ProblemStatement {
		t.Errorf("problem_statement = %q", parsed.ProblemStatement)
	}
	if parsed.TargetOutcome != p.TargetOutcome {
		t.Errorf("target_outcome = %q", parsed.TargetOutcome)
	}
	if parsed.TechnicalApproach != p.TechnicalApproach {
		t.Errorf("technical_approach = %q", parsed.TechnicalApproach)
	}
	if parsed.ValidationStrategy != p.ValidationStrategy {
		t.Errorf("validation_strategy = %q", parsed.ValidationStrategy)
	}
	if len(parsed.FinalValidationCommands) != 1 || parsed.FinalValidationCommands[0] != "vrooli scenario test plan-manager" {
		t.Errorf("final_validation_commands = %v", parsed.FinalValidationCommands)
	}
	if parsed.WorkPosture != plans.WorkPostureGreenfield {
		t.Errorf("work_posture = %q", parsed.WorkPosture)
	}
	if len(parsed.Phases) != 2 {
		t.Fatalf("phases = %d", len(parsed.Phases))
	}
	ph := parsed.Phases[0]
	if len(ph.Steps) != 3 || ph.Steps[0] != "Add proto fields" {
		t.Errorf("phase steps = %v", ph.Steps)
	}
	if len(ph.AffectedAreas) != 2 {
		t.Errorf("phase affected_areas = %v", ph.AffectedAreas)
	}
	if ph.Validation != "go build ./... and go test ./internal/planproto" {
		t.Errorf("phase validation = %q", ph.Validation)
	}
	if ph.HandoffNotes != "Phase 2 must finish before the renderer phase." {
		t.Errorf("phase handoff = %q", ph.HandoffNotes)
	}
	if len(ph.RisksHazards) != 1 || ph.RisksHazards[0] != "field-number churn" {
		t.Errorf("phase risks = %v", ph.RisksHazards)
	}
	if ph.Acceptance != "Generated code compiles and tests pass." {
		t.Errorf("phase acceptance = %q", ph.Acceptance)
	}
}
