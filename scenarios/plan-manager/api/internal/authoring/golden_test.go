package authoring_test

import (
	"context"
	"strings"
	"testing"

	"plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"
)

// TestGoldenAuthoringWizardHardening is the replayable end-to-end guard for the
// authoring-wizard-hardening plan. It walks a realistic plan-manager authoring
// run (modeled on the Plan Manager logging-polish plan) through the Service and
// asserts every observed friction point stays fixed:
//
//  1. a docs path tagged [CODE:] is rejected at submit time;
//  2. filling the last mandatory section does NOT jump to final review while
//     global relevant context is unresolved;
//  3. the references step always offers a concrete fallback;
//  4. phase-scoped context defaults to phase_entry (not once_per_execution);
//  5. the plan finalizes and renders with its structured references and phases.
//
// If any fix regresses, the corresponding assertion fails here in one place.
func TestGoldenAuthoringWizardHardening(t *testing.T) {
	ctx := context.Background()
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer, Renderer: testRenderer{}})

	sess, _, err := svc.StartSession(ctx, "Plan Manager Logging Polish", "plan-manager-logging-polish", "")
	require.NoError(t, err)

	// Prose mandatory sections.
	prose := []struct {
		key authoring.SectionKey
		val string
	}{
		{authoring.SectionPurpose, "Make plan-manager execution logging coherent and queryable."},
		{authoring.SectionProblemStatement, "Decisions, findings, and notes are scattered across ad-hoc surfaces."},
		{authoring.SectionTargetOutcome, "All execution log entries live in one durable, queryable ledger."},
		{authoring.SectionScope, "In: planlog domain ledger. Out: execution semantics redesign."},
		{authoring.SectionTechnicalApproach, "Unify entries behind a single log_entries table and LogService."},
		{authoring.SectionValidationStrategy, "Run the planlog suite, then the full scenario test, then a baseline diff."},
		{authoring.SectionDefinitionOfDone, "planlog tests green; baseline diff exit 0; docs updated."},
	}
	for _, p := range prose {
		_, violations, _, err := svc.SubmitSection(ctx, sess.ID, p.key, p.val)
		require.NoError(t, err)
		require.Empty(t, violations, "prose section %s should pass", p.key)
	}

	// Change boundary (mandatory): declare the allow globs the plan may change.
	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionAcceptanceBoundary,
		"acceptance_allow:\n- scenarios/plan-manager/**\n- packages/proto/**")
	require.NoError(t, err)
	require.Empty(t, violations, "change boundary should pass")

	// Friction 3: the references step surfaces a NO_CODE_REFS fallback.
	_, refStep, err := svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	requireActionID(t, refStep, "submit-no-code-refs")

	// Friction 1: a docs path tagged [CODE:] is rejected; the corrected mix passes.
	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE: docs/concepts/PLAN-MODEL.md]")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "docs path tagged [CODE:] must be rejected")

	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences,
		"[CODE: scenarios/plan-manager/api/internal/planlog/service.go]\n[DOC: docs/concepts/PLAN-MODEL.md]")
	require.NoError(t, err)
	require.Empty(t, violations, "a correct CODE/DOC mix passes")

	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRegressionAnchor,
		"Strategy: change_boundary\nBaseline name: plan-manager-logging-polish-baseline\nHEAD sha: abc123")
	require.NoError(t, err)
	require.Empty(t, violations)

	// One structured phase (this also fills the mandatory phases section).
	// Friction 6 (summary) is asserted via PhaseAddSummary's unit test; here we
	// confirm the phase itself carries both authored fields.
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Unify the ledger", "Introduce a single log_entries table and LogService.")
	require.NoError(t, err)
	require.Equal(t, "Unify the ledger", phase.Title)
	require.Equal(t, "Introduce a single log_entries table and LogService.", phase.Intent)

	// Friction 2: all mandatory sections filled and a phase exists, but no skill
	// decision was made. The wizard steers to the context checkpoint (recommending
	// author skill-pack, with a NO_SKILL_CONTEXT escape) before phase work —
	// steering only, never a finalize blocker.
	_, _, _, ready, _, step, err := svc.ContinueAuthoring(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, ready, "must not be ready while the phase is incomplete")
	require.Equal(t, "global_relevant_context", step.StepKind,
		"without a skill decision the wizard steers to the skill/context checkpoint")
	requireActionID(t, step, "discover-skill-pack")

	// Resolve global context with a real plan-wide setup item (once_per_execution).
	_, globalItem, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextSkill,
		Label:       "Recall planlog prior work",
		Reason:      "Plan-wide recall before any phase.",
		Instruction: "Load before starting implementation.",
		Target:      "scientific-debugging",
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextOncePerExecution, globalItem.RepeatPolicy)

	// Friction 2 (cont.): global context is advisory and the phase is still
	// incomplete → still not final review.
	_, _, _, ready, _, step, err = svc.ContinueAuthoring(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, ready)
	require.NotEqual(t, "final_review", step.StepKind, "an incomplete phase must not allow final review")

	phaseFields := []struct {
		field authoring.PhaseField
		val   string
	}{
		{authoring.PhaseFieldReferences, "[CODE: scenarios/plan-manager/api/internal/planlog/service.go]"},
		{authoring.PhaseFieldSteps, "Add the log_entries schema\nWire the LogService\nMigrate callers"},
		{authoring.PhaseFieldValidation, "go test ./internal/planlog"},
		{authoring.PhaseFieldAcceptance, "Every log entry persists and round-trips through the planlog suite."},
	}
	for _, f := range phaseFields {
		_, _, _, err := svc.SubmitPhaseField(ctx, sess.ID, phase.ID, f.field, f.val)
		require.NoError(t, err)
	}

	// Friction 4: phase-scoped context defaults to phase_entry.
	_, phaseItem, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, phase.ID, internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextSkill,
		Label:       "planlog API steer",
		Reason:      "Phase-specific implementation guidance.",
		Instruction: "Load before editing the planlog service.",
		Target:      "api-steer",
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextPhaseEntry, phaseItem.RepeatPolicy)

	// Now structurally ready: validate, preview, finalize.
	_, _, _, ready, _, _, err = svc.ContinueAuthoring(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, ready, "all required inputs resolved → ready to finalize")

	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid, "structure must validate clean, got %#v", violations)

	preview, _, err := svc.PreviewPlan(ctx, sess.ID)
	require.NoError(t, err)
	require.Contains(t, preview, "Unify the ledger")

	result, _, err := svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	plan := result.Plan
	require.NoError(t, err)
	require.Equal(t, 1, writer.calls)
	require.Len(t, plan.Phases, 1)
	require.NotEmpty(t, plan.References, "finalized plan carries structured references")
	// The CODE/DOC kinds survive into the persisted plan.
	kinds := map[internalplans.ReferenceKind]bool{}
	for _, ref := range plan.References {
		kinds[ref.Kind] = true
	}
	require.True(t, kinds[internalplans.ReferenceCode], "a [CODE:] reference is persisted")
	require.True(t, kinds[internalplans.ReferenceDoc], "a [DOC:] reference is persisted")

	rendered := internalplans.RenderMarkdown(plan)
	require.Contains(t, rendered, "Plan Manager Logging Polish")
	require.Contains(t, strings.ToLower(rendered), "phase 1")
}

func requireActionID(t *testing.T, step authoring.GuidedStep, id string) {
	t.Helper()
	for _, a := range step.NextActions {
		if a.ID == id {
			return
		}
	}
	t.Fatalf("expected guided step to offer action %q, actions=%v", id, actionIDs(step))
}

func actionIDs(step authoring.GuidedStep) []string {
	out := make([]string, 0, len(step.NextActions))
	for _, a := range step.NextActions {
		out = append(out, a.ID)
	}
	return out
}
