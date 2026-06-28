package authoring_test

import (
	"context"
	"strings"
	"testing"

	"plan-manager/internal/authoring"
	internalplans "plan-manager/internal/plans"

	"github.com/stretchr/testify/require"
)

// --- Phase 1: reference kind/path semantic validation ---

// TestReferencesSectionRejectsDocPathAsCode is the headline Phase-1 invariant: a
// documentation path tagged [CODE:] fails at submit time with an actionable
// message, while the same path tagged [DOC:] (and a real source file tagged
// [CODE:]) pass.
func TestReferencesSectionRejectsDocPathAsCode(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Kind gate", "kind-gate", "")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE: docs/concepts/PLAN-MODEL.md]")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "a docs path tagged [CODE:] must be rejected at submit time")
	require.Contains(t, violations[0].Message, "documentation path")

	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[DOC: docs/concepts/PLAN-MODEL.md]")
	require.NoError(t, err)
	require.Empty(t, violations, "the same docs path tagged [DOC:] is valid")

	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[CODE: scenarios/plan-manager/api/internal/authoring/service.go]")
	require.NoError(t, err)
	require.Empty(t, violations, "a real source file tagged [CODE:] is valid")
}

func TestReferencesSectionRejectsCodePathAsDoc(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Kind gate", "kind-gate", "")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionReferences, "[DOC: scenarios/plan-manager/api/internal/authoring/service.go]")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "a source file tagged [DOC:] must be rejected")
	require.Contains(t, violations[0].Message, "source file")
}

func TestPhaseReferenceKindMismatchRejected(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Phase kind gate", "phase-kind-gate", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Wire", "Wire the converters")
	require.NoError(t, err)

	_, violations, _, err := svc.SubmitPhaseField(ctx, sess.ID, phase.ID, authoring.PhaseFieldReferences, "[CODE: docs/internal/SEAMS.md]")
	require.NoError(t, err)
	found := false
	for _, v := range violations {
		if strings.Contains(v.Message, "documentation path") {
			found = true
		}
	}
	require.True(t, found, "a phase [CODE:] reference pointing at a doc must be flagged, got %#v", violations)
}

func TestContextItemReferenceKindMismatchRejected(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Context kind gate", "context-kind-gate", "")
	require.NoError(t, err)

	// code_ref pointing at a markdown doc is rejected.
	_, _, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextCodeRef,
		Label:       "Bad code ref",
		Reason:      "wrong kind",
		Instruction: "read",
		Target:      "docs/concepts/PLAN-MODEL.md",
	})
	require.NoError(t, err)
	require.NotEmpty(t, violations, "a code_ref context item pointing at a doc must be rejected")

	// doc pointing at a source file is rejected.
	_, _, violations, _, err = svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextDoc,
		Label:       "Bad doc ref",
		Reason:      "wrong kind",
		Instruction: "read",
		Target:      "scenarios/plan-manager/api/internal/authoring/service.go",
	})
	require.NoError(t, err)
	require.NotEmpty(t, violations, "a doc context item pointing at a source file must be rejected")
}

// TestCommandContextRequiresInstructionBeforeAcceptance locks the invariant that
// a command/search context item without an instruction is rejected at submit
// time and not stored.
func TestCommandContextRequiresInstructionBeforeAcceptance(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Cmd ctx gate", "cmd-ctx-gate", "")
	require.NoError(t, err)

	_, _, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:    internalplans.RelevantContextCommand,
		Label:   "Run setup",
		Reason:  "needed",
		Command: "vrooli scenario start plan-manager",
	})
	require.NoError(t, err)
	require.NotEmpty(t, violations, "a command context without instruction must be rejected")

	items, _, err := svc.ListRelevantContext(ctx, sess.ID, "")
	require.NoError(t, err)
	require.Empty(t, items, "a rejected command context item must not be stored")
}

// --- Phase 2: no premature final review ---

// TestMutationDoesNotJumpToFinalReviewBeforeGlobalContext proves a mutation that
// fills the last mandatory section reports the global-context checkpoint, not
// final_review, while plan-wide context is unresolved.
func TestMutationDoesNotJumpToFinalReviewBeforeGlobalContext(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "No premature review", "no-premature-review", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	// Re-submit a filled section to observe the post-mutation guided step.
	_, _, step, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Make widgets better, again.")
	require.NoError(t, err)
	require.Equal(t, "global_relevant_context", step.StepKind,
		"all mandatory sections filled but global context unresolved must surface the checkpoint, not final review")
}

// TestMutationDoesNotJumpToFinalReviewWithIncompletePhase proves that once global
// context is resolved, an incomplete structured phase keeps the guided step on
// phase work rather than final review.
func TestMutationDoesNotJumpToFinalReviewWithIncompletePhase(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Incomplete phase", "incomplete-phase", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_CONTEXT: fixture needs no plan-wide setup.")
	require.NoError(t, err)
	// Add an incomplete structured phase (title + intent only).
	_, _, _, _, err = svc.AddPhase(ctx, sess.ID, "Build", "Do the work")
	require.NoError(t, err)

	_, _, step, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Make widgets better, again.")
	require.NoError(t, err)
	require.NotEqual(t, "final_review", step.StepKind, "an incomplete phase must not allow final review")
	require.True(t, strings.HasPrefix(step.StepKind, "phase"), "the guided step must point at the incomplete phase, got %q", step.StepKind)
}

// --- Phase 3: references discovery recovery ---

func TestReferencesStepOffersSuggestAndNoCodeRefsRecovery(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Refs recovery", "refs-recovery", "")
	require.NoError(t, err)

	_, step, err := svc.GetSection(ctx, sess.ID, authoring.SectionReferences)
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, a := range step.NextActions {
		ids[a.ID] = true
	}
	require.True(t, ids["suggest-references"], "references step offers search-hub suggestion")
	require.True(t, ids["submit-references"], "references step offers manual submission")
	require.True(t, ids["submit-no-code-refs"], "references step offers a NO_CODE_REFS fallback")
}

func TestSuggestReferencesStepOffersNoCodeRefsFallback(t *testing.T) {
	ctx := context.Background()
	// No Suggester seam wired -> suggestion degrades to no candidates, but the
	// guided step still names the NO_CODE_REFS fallback so the agent is never stuck.
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Refs degrade", "refs-degrade", "")
	require.NoError(t, err)

	_, candidates, step, err := svc.SuggestReferences(ctx, sess.ID)
	require.NoError(t, err)
	require.Empty(t, candidates)
	ids := map[string]bool{}
	for _, a := range step.NextActions {
		ids[a.ID] = true
	}
	require.True(t, ids["submit-no-code-refs"], "a degraded suggestion still names the exact NO_CODE_REFS fallback")
}

// --- Phase 4: phase context repeat default + summaries ---

func TestPhaseContextDefaultsToPhaseEntry(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Repeat default", "repeat-default", "")
	require.NoError(t, err)
	_, phase, _, _, err := svc.AddPhase(ctx, sess.ID, "Build", "Do the work")
	require.NoError(t, err)

	// Unset repeat policy on a phase item must resolve to phase_entry.
	_, item, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, phase.ID, internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextSkill,
		Label:       "Steer",
		Reason:      "phase setup",
		Instruction: "load before the phase",
		Target:      "api-steer",
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextPhaseEntry, item.RepeatPolicy)

	// An explicit once_per_execution on a phase item is corrected to phase_entry.
	_, item, _, _, err = svc.SubmitRelevantContextItem(ctx, sess.ID, phase.ID, internalplans.RelevantContextItem{
		Kind:         internalplans.RelevantContextSkill,
		Label:        "Steer2",
		Reason:       "phase setup",
		Instruction:  "load before the phase",
		Target:       "api-steer",
		RepeatPolicy: internalplans.RelevantContextOncePerExecution,
	})
	require.NoError(t, err)
	require.Equal(t, internalplans.RelevantContextPhaseEntry, item.RepeatPolicy,
		"once_per_execution is contradictory for a phase-scoped item")
}

func TestGlobalContextDefaultsToOncePerExecution(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Global repeat", "global-repeat", "")
	require.NoError(t, err)

	_, item, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", internalplans.RelevantContextItem{
		Kind:        internalplans.RelevantContextSkill,
		Label:       "Steer",
		Reason:      "plan-wide setup",
		Instruction: "load before any phase",
		Target:      "api-steer",
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, internalplans.RelevantContextOncePerExecution, item.RepeatPolicy)
}

func TestPhaseAddSummaryNamesTitleAndIntent(t *testing.T) {
	summary := authoring.PhaseAddSummary(authoring.PhaseDraft{Order: 2, Title: "Wire converters", Intent: "Map every new field"})
	require.Contains(t, summary, "Wire converters")
	require.Contains(t, summary, "Map every new field")
}
