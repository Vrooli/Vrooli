package authoring_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"plan-manager/internal/authoring"
	"plan-manager/internal/planmodel"
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
	fillMandatorySectionsOnly(t, svc, sess.ID)

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
	fillMandatorySectionsOnly(t, svc, sess.ID)
	_, _, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_CONTEXT: fixture needs no plan-wide setup.\nNO_SKILL_CONTEXT: fixture has no skill setup.")
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

// TestSubmitSkillContextNormalizesFullCommandTarget pins the entry-point repair
// for the doubled skill-read defect: a skill context item submitted with the
// full `prompt-manager skill read <slug>` command as its Target is normalized
// to a bare-slug Target with the runnable command in Command/Argv, so no
// downstream command assembly can double the prefix (contract decision D6).
func TestSubmitSkillContextNormalizesFullCommandTarget(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "Skill target normalization", "skill-target-normalization", "")
	require.NoError(t, err)

	_, item, violations, _, err := svc.SubmitRelevantContextItem(ctx, sess.ID, "", planmodel.RelevantContextItem{
		Kind:        planmodel.RelevantContextSkill,
		Label:       "Scientific debugging skill",
		Reason:      "State-machine bug; reproduce before fixing.",
		Instruction: "Load this internal skill before implementation.",
		Target:      "prompt-manager skill read scientific-debugging",
		Required:    true,
	})
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Equal(t, "scientific-debugging", item.Target, "target must be normalized to the bare slug")
	require.Equal(t, "prompt-manager skill read scientific-debugging", item.Command, "the full command must move to Command")
	require.Equal(t, []string{"prompt-manager", "skill", "read", "scientific-debugging"}, item.Argv)
}

// TestDecisionsAndAssumptionMitigationsFlowIntoPlan pins the optional D3
// authoring path: a decisions section of '<title>: <statement>' lines and
// assumption lines carrying an '-> mitigation' suffix land as structured
// fields on the finalized plan; a malformed decision line is rejected at
// submit time; both sections stay optional (no gate demands them).
func TestDecisionsAndAssumptionMitigationsFlowIntoPlan(t *testing.T) {
	ctx := context.Background()
	writer := &fakePlanWriter{}
	svc := newService(t, authoring.Deps{Writer: writer})
	sess, _, err := svc.StartSession(ctx, "Decisions flow", "decisions-flow", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, violations, _, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionDecisions, "just a statement with no separator")
	require.NoError(t, err)
	require.NotEmpty(t, violations, "a decision line without '<title>: <statement>' must be rejected")

	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionDecisions,
		"Cluster order: nine clusters, wizard asks in render order.\nDependency posture: search-hub is required:false.")
	require.NoError(t, err)
	require.Empty(t, violations)

	_, violations, _, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionAssumptions,
		"The baseline is captured first.\nprompt-manager JSON is stable -> pin parsing behind the probe seam.")
	require.NoError(t, err)
	require.Empty(t, violations)

	_, _, err = svc.Finalize(ctx, sess.ID, authoring.FinalizeOptions{})
	require.NoError(t, err)
	require.Equal(t, []planmodel.PlanDecision{
		{Title: "Cluster order", Statement: "nine clusters, wizard asks in render order."},
		{Title: "Dependency posture", Statement: "search-hub is required:false."},
	}, writer.created.Decisions)
	require.Equal(t, []planmodel.PlanAssumption{
		{Statement: "prompt-manager JSON is stable", Mitigation: "pin parsing behind the probe seam."},
	}, writer.created.AssumptionRisks)
	require.Equal(t, "The baseline is captured first.", writer.created.Assumptions)
}

// TestDiscoverContextExecutesProbesAndDeduplicates pins the server-side
// discovery contract (D5/D6): DiscoverContextCandidates executes the probes
// through the runner seam, deduplicates by target with provenance preserved,
// carries bare-slug skill targets, and converts failed probes into typed
// degraded notes without ever blocking the wizard.
func TestDiscoverContextExecutesProbesAndDeduplicates(t *testing.T) {
	ctx := context.Background()
	skillJSON := `{"results":[
		{"type":"skill","id":"scientific-debugging","description":"reproduce first","score":0.9},
		{"type":"skill","id":"test","description":"testing discipline","score":0.5}]}`
	recallJSON := `{"ranked":[
		{"type":"skill","id":"scientific-debugging","title":"Scientific Debugging","snippet":"reproduce first","path":"scientific-debugging","score":0.3,"rerank_score":0.4}]}`
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		argv := strings.Join(args, " ")
		switch {
		case name == "prompt-manager" && strings.Contains(argv, "--type skill"):
			return []byte(skillJSON), nil
		case name == "prompt-manager" && strings.Contains(argv, "--type all"):
			return nil, errors.New("prompt-manager actions probe is down")
		case name == "search-hub":
			return []byte(recallJSON), nil
		default:
			return nil, errors.New("unexpected probe " + argv)
		}
	}
	svc := newService(t, authoring.Deps{
		Writer:  &fakePlanWriter{},
		Context: authoring.NewCommandContextDiscovererWithTimeout(runner, time.Second),
	})
	sess, _, err := svc.StartSession(ctx, "Probe discovery", "probe-discovery", "")
	require.NoError(t, err)

	updated, candidates, _, err := svc.DiscoverContextCandidates(ctx, sess.ID, []string{"microphone lifecycle"}, "architectural", false)
	require.NoError(t, err)

	var skills, actions []authoring.ContextCandidate
	for _, c := range candidates {
		switch {
		case c.Item.Kind == planmodel.RelevantContextSkill:
			skills = append(skills, c)
		case c.Item.Kind == planmodel.RelevantContextCommand:
			actions = append(actions, c)
		}
	}
	// scientific-debugging appears in both prompt-manager probes: deduplicated
	// to one candidate (the higher-scored), provenance from both preserved.
	require.Len(t, skills, 2, "duplicate skill across probes must deduplicate")
	var sci authoring.ContextCandidate
	for _, c := range skills {
		if c.Item.Target == "scientific-debugging" {
			sci = c
		}
		require.Empty(t, c.Item.Command, "skill candidates must carry bare slugs, not assembled commands (D6)")
	}
	require.Contains(t, sci.Detail, "score 0.900")
	require.Contains(t, sci.Detail, "also:")

	require.Empty(t, actions, "the down actions probe yields no fabricated candidates")

	require.Len(t, updated.DiscoveryBatches, 1)
	require.Len(t, updated.DiscoveryBatches[0].ProbeNotes, 1, "the down actions probe degrades independently")
	require.Equal(t, "prompt-manager-actions", updated.DiscoveryBatches[0].ProbeNotes[0].Probe)
	require.Contains(t, updated.DiscoveryBatches[0].ProbeNotes[0].Detail, "prompt-manager actions probe is down")

	// All-down: the step still returns a usable (empty + noted) result.
	svcDown := newService(t, authoring.Deps{
		Writer:  &fakePlanWriter{},
		Context: authoring.NewCommandContextDiscovererWithTimeout(nil, time.Second),
	})
	sessDown, _, err := svcDown.StartSession(ctx, "All down", "all-down", "")
	require.NoError(t, err)
	updatedDown, downCandidates, stepDown, err := svcDown.DiscoverContextCandidates(ctx, sessDown.ID, []string{"anything"}, "", false)
	require.NoError(t, err, "discovery must never block or fail the wizard when dependencies are down")
	require.Empty(t, downCandidates)
	require.Len(t, updatedDown.DiscoveryBatches, 1)
	require.Len(t, updatedDown.DiscoveryBatches[0].ProbeNotes, 3)
	require.Equal(t, authoring.DiscoveryBatchApplied, updatedDown.DiscoveryBatches[0].Status)
	require.NotEqual(t, "context_discovery", stepDown.StepKind, "empty auto-applied batches should not ask for context-apply")
	require.NotContains(t, stepDown.RequiredInputs, "context-apply for the pending batch")
}

// fakeSkillResolver returns canned steered suggestions (the D7 seam's future
// data-driven sources stand in behind this fake).
type fakeSkillResolver struct{ suggestions []authoring.SkillSuggestion }

func (f fakeSkillResolver) SuggestSkills(context.Context, planmodel.ChangeBoundary) []authoring.SkillSuggestion {
	return f.suggestions
}

// TestSkillCheckpointGateV3 pins the batch-applied gate end to end: an unapplied
// discovery batch blocks finalize with one clear message; applying the batch
// unblocks; an honest NO_SKILL_CONTEXT skip passes without discovery; a
// rejection without a reason is refused in the single-item lane; and resolver
// suggestions flow through disposition like any discovered candidate (D7).
func TestSkillCheckpointGateV3(t *testing.T) {
	ctx := context.Background()
	resolver := fakeSkillResolver{suggestions: []authoring.SkillSuggestion{
		{Slug: "react-coherence", Reason: "The boundary touches a React UI."},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, SkillResolver: resolver})
	sess, _, err := svc.StartSession(ctx, "Gate v2", "gate-v2", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	_, candidates, _, err := svc.DiscoverContextCandidates(ctx, sess.ID, []string{"gate semantics"}, "minor", false)
	require.NoError(t, err)
	require.NotEmpty(t, candidates)

	var steered *authoring.ContextCandidate
	for i := range candidates {
		if candidates[i].Source == "skill-applicability-resolver" {
			steered = &candidates[i]
		}
	}
	require.NotNil(t, steered, "resolver suggestions must appear as ordinary candidates")
	require.Equal(t, "react-coherence", steered.Item.Target)

	// An unapplied batch blocks finalize with one actionable message.
	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.False(t, valid)
	var sawPendingBatch bool
	for _, v := range violations {
		if strings.Contains(v.Message, "context discovery batch") && strings.Contains(v.Message, "context-apply") {
			sawPendingBatch = true
		}
	}
	require.True(t, sawPendingBatch, "pending batch must be called out, got %#v", violations)

	// A rejection without a reason is refused (D4: reject requires judgment).
	_, _, _, err = svc.RejectContextCandidate(ctx, sess.ID, candidates[0].ID, "  ")
	require.Error(t, err)

	// Disposition everything through the single-item lane: accept the steered
	// suggestion, reject the rest. The batch auto-closes when its shortlist is
	// fully dispositioned.
	for _, c := range candidates {
		if c.ID == steered.ID {
			_, _, _, _, _, err := svc.AcceptContextCandidate(ctx, sess.ID, c.ID, "")
			require.NoError(t, err)
			continue
		}
		_, _, _, err := svc.RejectContextCandidate(ctx, sess.ID, c.ID, "degraded probe; concept covered elsewhere")
		require.NoError(t, err)
	}
	valid, violations, _, err = svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid, "all-dispositioned sweep must pass, got %#v", violations)

	// Honest skip path: a fresh session with NO_SKILL_CONTEXT passes without
	// any discovery.
	sessSkip, _, err := svc.StartSession(ctx, "Gate v2 skip", "gate-v2-skip", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sessSkip.ID)
	_, _, _, err = svc.SubmitSection(ctx, sessSkip.ID, authoring.SectionRelevantContext, "NO_SKILL_CONTEXT: no internal skill applies to this fixture.")
	require.NoError(t, err)
	valid, violations, _, err = svc.ValidateStructure(ctx, sessSkip.ID)
	require.NoError(t, err)
	require.True(t, valid, "honest skip must pass, got %#v", violations)
}
