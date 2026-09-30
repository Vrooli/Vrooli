package plans_test

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plan-manager/internal/planmodel"
	"plan-manager/internal/plans"
)

// updateGoldens refreshes the render characterization goldens:
//
//	GOWORK=off go test ./internal/plans -run TestRenderGolden -update-goldens
//
// Goldens are the review artifact for every renderer change: a diff here is the
// exact user-visible markdown change and must be reviewed line-by-line.
var updateGoldens = flag.Bool("update-goldens", false, "rewrite render golden files")

// goldenFixtures are the renderer characterization fixtures. Each class exists
// to freeze one behavior of the markdown projection:
//
//   - typed-skill-context: skill/doc setup context submitted through the typed
//     authoring path (Command populated) — the healthy path.
//   - legacy-required-reading: an imported plan whose phase RequiredReading
//     carries raw `prompt-manager skill read <x>` and `sed -n` lines — the
//     migration path.
//   - no-context-phase: a phase whose only context is a NO_CONTEXT: skip note.
//   - empty-optional: only mandatory fields present; optional sections must
//     render nothing.
//   - sprawl: a mic-plan-sized plan (many phases, repeated NO_CONTEXT skips)
//     used to measure total rendered size across section-model changes.
func goldenFixtures() map[string]plans.Plan {
	return map[string]plans.Plan{
		"typed-skill-context":     typedSkillContextPlan(),
		"legacy-required-reading": legacyRequiredReadingPlan(),
		"no-context-phase":        noContextPhasePlan(),
		"empty-optional":          emptyOptionalPlan(),
		"sprawl":                  sprawlPlan(),
		"decisions-risks":         decisionsRisksPlan(),
	}
}

func TestRenderGolden(t *testing.T) {
	for name, p := range goldenFixtures() {
		t.Run(name, func(t *testing.T) {
			got := plans.RenderMarkdown(p)
			path := filepath.Join("testdata", "goldens", name+".golden.md")
			if *updateGoldens {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}
			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read golden (run with -update-goldens to create): %v", err)
			}
			if got != string(want) {
				t.Fatalf("rendered markdown differs from golden %s\n--- got ---\n%s\n--- want ---\n%s", path, got, string(want))
			}
		})
	}
}

// baseFixturePlan carries the minimal quality-clean plan the fixtures extend, so
// golden diffs stay focused on the behavior each fixture freezes rather than a
// wall of quality-notice noise.
func baseFixturePlan(title string) plans.Plan {
	return plans.Plan{
		Title:              title,
		Status:             plans.PlanStatusDraft,
		Purpose:            "Freeze the renderer's current output for " + title + ".",
		ProblemStatement:   "Renderer behavior is unpinned; changes ship without review.",
		TargetOutcome:      "Every renderer change shows up as a reviewable golden diff.",
		Scope:              "In scope: markdown projection. Out of scope: persistence.",
		TechnicalApproach:  "Golden-file characterization before any renderer change.",
		ValidationStrategy: "GOWORK=off go test ./internal/plans compares renders byte-exact.",
		DefinitionOfDone:   "Goldens exist and reproduce current output verbatim.",
		References: []plans.Reference{
			{Kind: plans.ReferenceCode, Target: "scenarios/plan-manager/api/internal/plans/render.go"},
		},
		ChangeBoundary: plans.ChangeBoundary{
			AcceptanceAllow: []string{"scenarios/plan-manager/**"},
		},
		RegressionAnchor: plans.RegressionAnchor{
			Strategy:     "scenario_baseline",
			Scenario:     "plan-manager",
			BaselineName: "plan-manager-render-golden",
			Commands: []string{
				"git-control-tower baseline diff --scenario plan-manager --name plan-manager-render-golden --wait",
			},
		},
		RelevantContext: []plans.RelevantContextItem{
			{
				ID:           "ctx-global-skip",
				Kind:         plans.RelevantContextNote,
				Scope:        plans.RelevantContextScopeGlobal,
				Label:        "NO_SKILL_CONTEXT: fixture plan; skill setup is exercised by dedicated fixtures.",
				Reason:       "NO_SKILL_CONTEXT: fixture plan; skill setup is exercised by dedicated fixtures.",
				Instruction:  "NO_SKILL_CONTEXT: fixture plan; skill setup is exercised by dedicated fixtures.",
				Required:     true,
				RepeatPolicy: plans.RelevantContextOncePerExecution,
				Source:       plans.RelevantContextSourceAuthored,
				Status:       plans.RelevantContextStatusReady,
			},
		},
	}
}

func typedSkillContextPlan() plans.Plan {
	p := baseFixturePlan("Typed skill context")
	p.RelevantContext = []plans.RelevantContextItem{
		{
			ID:           "ctx-skill-1",
			Kind:         plans.RelevantContextSkill,
			Scope:        plans.RelevantContextScopeGlobal,
			Label:        "scientific-debugging",
			Reason:       "The defect is a state-machine bug; reproduce before fixing.",
			Instruction:  "Load this internal skill before implementation.",
			Command:      "prompt-manager skill read scientific-debugging",
			Argv:         []string{"prompt-manager", "skill", "read", "scientific-debugging"},
			Target:       "scientific-debugging",
			Required:     true,
			RepeatPolicy: plans.RelevantContextOncePerExecution,
			Source:       plans.RelevantContextSourceAuthored,
			Status:       plans.RelevantContextStatusReady,
		},
		{
			ID:           "ctx-skill-2",
			Kind:         plans.RelevantContextSkill,
			Scope:        plans.RelevantContextScopeGlobal,
			Label:        "test",
			Reason:       "Golden-file discipline for the renderer work.",
			Instruction:  "Load this internal skill before implementation.",
			Target:       "test",
			Required:     true,
			RepeatPolicy: plans.RelevantContextOncePerExecution,
			Source:       plans.RelevantContextSourceAuthored,
			Status:       plans.RelevantContextStatusReady,
		},
		{
			ID:           "ctx-doc-1",
			Kind:         plans.RelevantContextDoc,
			Scope:        plans.RelevantContextScopeGlobal,
			Label:        "docs/TESTING.md",
			Reason:       "Server-owned test runs; never poll.",
			Instruction:  "Read this document before implementation.",
			Target:       "docs/TESTING.md",
			Required:     true,
			RepeatPolicy: plans.RelevantContextOncePerExecution,
			Source:       plans.RelevantContextSourceAuthored,
			Status:       plans.RelevantContextStatusReady,
		},
	}
	p.Phases = []plans.Phase{
		{
			ID:         "phase-1",
			Order:      1,
			Title:      "Renderer",
			Intent:     "Project setup context deterministically.",
			Status:     plans.PhaseStatusTodo,
			Steps:      []string{"Render fixtures", "Compare against goldens"},
			Validation: "GOWORK=off go test ./internal/plans",
			Acceptance: "Goldens match byte-exact.",
			RelevantContext: []plans.RelevantContextItem{
				{
					ID:           "phase-1-ctx-doc",
					Kind:         plans.RelevantContextDoc,
					Scope:        plans.RelevantContextScopePhase,
					PhaseID:      "phase-1",
					Label:        "docs/concepts/PLAN-MODEL.md",
					Reason:       "The renderer projects the model this doc defines.",
					Instruction:  "Read this document before implementation.",
					Target:       "docs/concepts/PLAN-MODEL.md",
					Required:     true,
					RepeatPolicy: plans.RelevantContextPhaseEntry,
					Source:       plans.RelevantContextSourceAuthored,
					Status:       plans.RelevantContextStatusReady,
				},
			},
			References: []plans.Reference{
				{Kind: plans.ReferenceCode, Target: "scenarios/plan-manager/api/internal/plans/render.go"},
			},
		},
	}
	return p
}

// legacyRequiredReadingPlan models a legacy-imported plan whose phase carries
// raw RequiredReading lines. The render-side migration of these lines is the
// path that historically produced the doubled `prompt-manager skill read
// prompt-manager skill read <x>` command (and a doubled `sed -n` for doc
// lines) in live mirrors.
func legacyRequiredReadingPlan() plans.Plan {
	p := baseFixturePlan("Legacy required reading")
	p.ImportProvenance = &plans.ImportProvenance{
		SourcePath:     "docs/plans/legacy-required-reading.md",
		OriginalFormat: plans.OriginalFormatLegacyMarkdown,
		ImportedAt:     "2026-07-01T00:00:00Z",
	}
	p.Phases = []plans.Phase{
		{
			ID:     "phase-1",
			Order:  1,
			Title:  "Wedge fix",
			Intent: "Fix the stuck-recording wedge.",
			Status: plans.PhaseStatusTodo,
			RequiredReading: []string{
				"prompt-manager skill read scientific-debugging",
				"sed -n '1,120p' docs/TESTING.md",
				"search-hub query 'microphone ownership' --type record",
			},
			Steps:      []string{"Reproduce the wedge", "Fix ownership"},
			Validation: "UI vitest suite for the voice hooks",
			Acceptance: "Wedge no longer reproducible.",
			References: []plans.Reference{
				{Kind: plans.ReferenceCode, Target: "scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts"},
			},
		},
	}
	return p
}

// noContextPhasePlan freezes how a phase whose only setup context is an explicit
// NO_CONTEXT: skip renders. The context items mirror what the authoring
// projection produces for a NO_CONTEXT phase note (note kind, canned reason).
func noContextPhasePlan() plans.Plan {
	p := baseFixturePlan("No context phase")
	p.Phases = []plans.Phase{
		{
			ID:     "phase-1",
			Order:  1,
			Title:  "Docs sweep",
			Intent: "Update reference docs only.",
			Status: plans.PhaseStatusTodo,
			RelevantContext: []plans.RelevantContextItem{
				{
					ID:           "ctx-note-1",
					Kind:         plans.RelevantContextNote,
					Scope:        plans.RelevantContextScopePhase,
					PhaseID:      "phase-1",
					Label:        "NO_CONTEXT: docs-only phase; no extra setup needed.",
					Reason:       "Authored phase note.",
					Instruction:  "NO_CONTEXT: docs-only phase; no extra setup needed.",
					RepeatPolicy: plans.RelevantContextPhaseEntry,
					Source:       plans.RelevantContextSourceAuthored,
					Status:       plans.RelevantContextStatusReady,
				},
			},
			Steps:      []string{"Update the docs"},
			Validation: "docs lint",
			Acceptance: "Docs accurate.",
			References: []plans.Reference{
				{Kind: plans.ReferenceDoc, Target: "docs/concepts/PLAN-MODEL.md"},
			},
		},
	}
	return p
}

func emptyOptionalPlan() plans.Plan {
	p := baseFixturePlan("Empty optional sections")
	p.Phases = []plans.Phase{
		{
			ID:         "phase-1",
			Order:      1,
			Title:      "Only phase",
			Intent:     "Prove empty optional sections render nothing.",
			Status:     plans.PhaseStatusTodo,
			Steps:      []string{"Render", "Diff"},
			Validation: "go test ./internal/plans",
			Acceptance: "No optional headings appear.",
			RelevantContext: []plans.RelevantContextItem{
				{
					ID:           "phase-1-ctx",
					Kind:         plans.RelevantContextNote,
					Scope:        plans.RelevantContextScopePhase,
					PhaseID:      "phase-1",
					Label:        "NO_CONTEXT: minimal fixture; no setup needed.",
					Reason:       "Authored phase note.",
					Instruction:  "NO_CONTEXT: minimal fixture; no setup needed.",
					RepeatPolicy: plans.RelevantContextPhaseEntry,
					Source:       plans.RelevantContextSourceAuthored,
					Status:       plans.RelevantContextStatusReady,
				},
			},
			References: []plans.Reference{
				{Kind: plans.ReferenceCode, Target: "scenarios/plan-manager/api/internal/plans/render.go"},
			},
		},
	}
	return p
}

// sprawlPlan is a mic-plan-sized fixture: full prose sections plus six phases,
// most of which skip context with NO_CONTEXT notes. Its golden's line count is
// the before/after measure for the section-model regroup (target: >=25%
// reduction with zero information loss).
func sprawlPlan() plans.Plan {
	p := baseFixturePlan("Sprawl measurement")
	p.NonGoals = "No redesign of the execution domain."
	p.Assumptions = "The baseline is captured before changes."
	p.Constraints = "Keep the wizard usable by small local models."
	p.ProhibitedApproaches = "Do not make markdown the source of truth."
	p.RisksHazards = "Section regroup could break a hidden markdown consumer."
	p.FinalValidationCommands = []string{"vrooli scenario test plan-manager"}
	p.RelevantContext = []plans.RelevantContextItem{
		{
			ID:           "ctx-skill-1",
			Kind:         plans.RelevantContextSkill,
			Scope:        plans.RelevantContextScopeGlobal,
			Label:        "scientific-debugging",
			Reason:       "State-machine bug; reproduce before fixing.",
			Instruction:  "Load this internal skill before implementation.",
			Command:      "prompt-manager skill read scientific-debugging",
			Target:       "scientific-debugging",
			Required:     true,
			RepeatPolicy: plans.RelevantContextOncePerExecution,
			Source:       plans.RelevantContextSourceAuthored,
			Status:       plans.RelevantContextStatusReady,
		},
	}
	noContext := func(phaseID, reason string) []plans.RelevantContextItem {
		label := "NO_CONTEXT: " + reason
		return []plans.RelevantContextItem{{
			ID:           phaseID + "-ctx",
			Kind:         plans.RelevantContextNote,
			Scope:        plans.RelevantContextScopePhase,
			PhaseID:      phaseID,
			Label:        label,
			Reason:       "Authored phase note.",
			Instruction:  label,
			RepeatPolicy: plans.RelevantContextPhaseEntry,
			Source:       plans.RelevantContextSourceAuthored,
			Status:       plans.RelevantContextStatusReady,
		}}
	}
	titles := []string{
		"State model", "Ownership registry", "Wedge detection",
		"Recovery affordance", "Clock unification", "Snappy start",
	}
	for i, title := range titles {
		id := "phase-" + strings.Repeat("i", i+1)
		p.Phases = append(p.Phases, plans.Phase{
			ID:              id,
			Order:           i + 1,
			Title:           title,
			Intent:          "Deliver the " + strings.ToLower(title) + " slice end to end.",
			Status:          plans.PhaseStatusTodo,
			AffectedAreas:   []string{"scenarios/web-console/ui/src/hooks/voice/"},
			Steps:           []string{"Implement the slice", "Add focused tests", "Run the suite"},
			ExpectedOutputs: []string{"Slice implemented with tests green"},
			Validation:      "Run the focused UI vitest suite for the touched hooks; then the full UI suite.",
			Acceptance:      "The " + strings.ToLower(title) + " slice is verifiably in place.",
			RelevantContext: noContext(id, "covered by the global skill and doc setup."),
			References: []plans.Reference{
				{Kind: plans.ReferenceCode, Target: "scenarios/web-console/ui/src/hooks/voice/useVoiceCapture.ts"},
			},
		})
	}
	return p
}

// TestRenderNeverDoublesSkillReadPrefix is the permanent regression guard for
// the doubled `prompt-manager skill read prompt-manager skill read <x>` defect:
// whether the skill item arrives via legacy RequiredReading migration or as a
// persisted item whose Target wrongly carries the full command, the rendered
// markdown must contain only runnable single-prefix commands.
func TestRenderNeverDoublesSkillReadPrefix(t *testing.T) {
	const prefix = "prompt-manager skill read"

	legacy := legacyRequiredReadingPlan()
	md := plans.RenderMarkdown(legacy)
	for _, line := range strings.Split(md, "\n") {
		if strings.Count(line, prefix) > 1 {
			t.Fatalf("legacy migration rendered a doubled skill-read prefix: %q", line)
		}
		if strings.Count(line, "sed -n") > 1 {
			t.Fatalf("legacy migration rendered a doubled sed command: %q", line)
		}
	}
	if !strings.Contains(md, "prompt-manager skill read scientific-debugging") {
		t.Fatalf("legacy skill line lost its runnable command:\n%s", md)
	}
	if !strings.Contains(md, "sed -n '1,120p' docs/TESTING.md") {
		t.Fatalf("legacy doc line lost its runnable command:\n%s", md)
	}

	corrupt := typedSkillContextPlan()
	corrupt.RelevantContext = append(corrupt.RelevantContext, plans.RelevantContextItem{
		ID:           "ctx-corrupt",
		Kind:         plans.RelevantContextSkill,
		Scope:        plans.RelevantContextScopeGlobal,
		Label:        "Corrupt target skill",
		Reason:       "Persisted before targets were normalized to bare slugs.",
		Instruction:  "Load this internal skill before implementation.",
		Target:       "prompt-manager skill read seam-discovery-and-enforcement",
		Required:     true,
		RepeatPolicy: plans.RelevantContextOncePerExecution,
		Source:       plans.RelevantContextSourceAuthored,
		Status:       plans.RelevantContextStatusReady,
	})
	md = plans.RenderMarkdown(corrupt)
	for _, line := range strings.Split(md, "\n") {
		if strings.Count(line, prefix) > 1 {
			t.Fatalf("corrupt-target item rendered a doubled skill-read prefix: %q", line)
		}
	}
	if !strings.Contains(md, "prompt-manager skill read seam-discovery-and-enforcement") {
		t.Fatalf("corrupt-target item lost its runnable command:\n%s", md)
	}
}

// TestCompactNoContextRoundTrip asserts the compact NO_CONTEXT phase line is a
// render→parse→render fixed point: adopting a mirror keeps the typed skip (so
// the phase-context quality gate stays satisfied) and re-render reproduces the
// same compact line, never the old operator-notes block.
func TestCompactNoContextRoundTrip(t *testing.T) {
	md1 := plans.RenderMarkdown(noContextPhasePlan())
	if !strings.Contains(md1, "- Context: none needed — docs-only phase; no extra setup needed.") {
		t.Fatalf("compact NO_CONTEXT line missing:\n%s", md1)
	}
	if strings.Contains(md1, "Phase Context Setup") {
		t.Fatalf("NO_CONTEXT-only phase must not render a full context block:\n%s", md1)
	}
	parsed, err := planmodel.ParsePlanMarkdown(md1)
	if err != nil {
		t.Fatalf("parse rendered markdown: %v", err)
	}
	if len(parsed.Phases) != 1 || len(parsed.Phases[0].RelevantContext) != 1 {
		t.Fatalf("typed NO_CONTEXT skip not recovered: %+v", parsed.Phases)
	}
	md2 := plans.RenderMarkdown(parsed)
	if md1 != md2 {
		t.Fatalf("compact NO_CONTEXT render not a fixed point:\n--- md1 ---\n%s\n--- md2 ---\n%s", md1, md2)
	}
}

// decisionsRisksPlan exercises the optional D3 fields: pinned plan-time
// decisions (D1..Dn) and the assumption/mitigation table, alongside prose
// assumptions and risks.
func decisionsRisksPlan() plans.Plan {
	p := baseFixturePlan("Decisions and risks")
	p.Decisions = []plans.PlanDecision{
		{Title: "Cluster names and order", Statement: "Nine clusters, wizard asks in render order."},
		{Title: "Field identity preserved", Statement: "Regrouping is catalog-order + render-grouping only."},
		{Title: "Dependency posture", Statement: "prompt-manager and search-hub become required:false dependencies."},
	}
	p.AssumptionRisks = []plans.PlanAssumption{
		{Statement: "prompt-manager --json output shape is stable", Mitigation: "pin parsing behind the probe seam with contract fixtures"},
		{Statement: "search-hub may be systemically degraded", Mitigation: "per-probe timeout and independent degradation"},
	}
	p.Assumptions = "The regression baseline is captured before any code change."
	p.RisksHazards = "Render regrouping could break a hidden markdown consumer."
	p.Phases = []plans.Phase{
		{
			ID:         "phase-1",
			Order:      1,
			Title:      "Only phase",
			Intent:     "Carry the decision fixtures.",
			Status:     plans.PhaseStatusTodo,
			Steps:      []string{"Render", "Diff"},
			Validation: "go test ./internal/plans",
			Acceptance: "D-list and table render.",
			RelevantContext: []plans.RelevantContextItem{
				{
					ID:           "phase-1-ctx",
					Kind:         plans.RelevantContextNote,
					Scope:        plans.RelevantContextScopePhase,
					PhaseID:      "phase-1",
					Label:        "NO_CONTEXT: fixture phase.",
					Instruction:  "NO_CONTEXT: fixture phase.",
					RepeatPolicy: plans.RelevantContextPhaseEntry,
					Source:       plans.RelevantContextSourceAuthored,
					Status:       plans.RelevantContextStatusReady,
				},
			},
			References: []plans.Reference{
				{Kind: plans.ReferenceCode, Target: "scenarios/plan-manager/api/internal/plans/render.go"},
			},
		},
	}
	return p
}

// TestDecisionsAndAssumptionTableRoundTrip pins the D3 rendering: the ordered
// D-list and the two-column table render when populated, render nothing when
// empty, and survive render→parse→render as a fixed point.
func TestDecisionsAndAssumptionTableRoundTrip(t *testing.T) {
	md1 := plans.RenderMarkdown(decisionsRisksPlan())
	for _, want := range []string{
		"### Decisions",
		"- **D1 — Cluster names and order:** Nine clusters, wizard asks in render order.",
		"- **D3 — Dependency posture:**",
		"| Assumption | If wrong → mitigation |",
		"| search-hub may be systemically degraded | per-probe timeout and independent degradation |",
	} {
		if !strings.Contains(md1, want) {
			t.Fatalf("missing %q in:\n%s", want, md1)
		}
	}
	parsed, err := planmodel.ParsePlanMarkdown(md1)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.Decisions) != 3 || len(parsed.AssumptionRisks) != 2 {
		t.Fatalf("structured fields not recovered: %d decisions, %d assumptions", len(parsed.Decisions), len(parsed.AssumptionRisks))
	}
	md2 := plans.RenderMarkdown(parsed)
	if md1 != md2 {
		t.Fatalf("decisions/table render not a fixed point:\n--- md1 ---\n%s\n--- md2 ---\n%s", md1, md2)
	}

	empty := emptyOptionalPlan()
	md := plans.RenderMarkdown(empty)
	for _, notWant := range []string{"### Decisions", "| Assumption |", "## Assumptions & Risks"} {
		if strings.Contains(md, notWant) {
			t.Fatalf("empty optional fields must render nothing, found %q", notWant)
		}
	}
}
