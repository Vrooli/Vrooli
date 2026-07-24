package authoring_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

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

// TestMutationSteersToSkillCheckpointThenFinalReview proves the wizard steers to
// the global-context checkpoint until an explicit skill decision exists (skill
// item or NO_SKILL_CONTEXT reason), then proceeds — steering, not a finalize
// blocker (finalize itself never gates on skill context).
func TestMutationSteersToSkillCheckpointThenFinalReview(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "No premature review", "no-premature-review", "")
	require.NoError(t, err)
	fillMandatorySectionsOnly(t, svc, sess.ID)

	// Re-submit a filled section to observe the post-mutation guided step: with
	// no skill decision the wizard must steer to the context checkpoint.
	_, _, step, err := svc.SubmitSection(ctx, sess.ID, authoring.SectionPurpose, "Make widgets better, again.")
	require.NoError(t, err)
	require.Equal(t, "global_relevant_context", step.StepKind,
		"without a skill decision the wizard steers to the skill/context checkpoint")
	requireActionID(t, step, "discover-skill-pack")
	requireActionID(t, step, "skip-skill-context")

	// Recording an explicit skip reason resolves the checkpoint.
	_, _, step, err = svc.SubmitSection(ctx, sess.ID, authoring.SectionRelevantContext, "NO_SKILL_CONTEXT: fixture has no skill setup.")
	require.NoError(t, err)
	require.Equal(t, "final_review", step.StepKind,
		"an explicit NO_SKILL_CONTEXT decision unlocks final review")
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
	require.True(t, ids["search-references"], "references step offers direct search-hub guidance")
	require.True(t, ids["submit-references"], "references step offers manual submission")
	require.True(t, ids["submit-no-code-refs"], "references step offers a NO_CODE_REFS fallback")
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

// TestDiscoverSkillPackExecutesPromptManagerAndIsIdempotent pins the simplified
// discovery contract: prompt-manager skills are added directly as global
// relevant context, without a candidate queue or batch disposition step.
func TestDiscoverSkillPackExecutesPromptManagerAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	skillJSON := `{"results":[
		{"type":"skill","id":"scientific-debugging","name":"Scientific debugging","description":"reproduce first","score":0.9},
		{"type":"skill","id":"test","name":"Testing discipline","description":"test the expected behavior","score":0.5}],
		"readCommand":"prompt-manager skill read scientific-debugging test",
		"recommendedReadCommand":"prompt-manager skill read scientific-debugging test",
		"budgetStatus":"ok"}`
	runner := func(_ context.Context, name string, args ...string) ([]byte, error) {
		require.Equal(t, "prompt-manager", name)
		require.Contains(t, args, "discover")
		require.Contains(t, args, "--type")
		require.Contains(t, args, "skill")
		if !strings.Contains(strings.Join(args, " "), "microphone lifecycle") {
			t.Fatalf("concepts not passed to prompt-manager: %#v", args)
		}
		return []byte(skillJSON), nil
	}
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Skills: authoring.NewCommandSkillPackDiscoverer(runner),
	})
	sess, _, err := svc.StartSession(ctx, "Probe discovery", "probe-discovery", "")
	require.NoError(t, err)

	updated, result, added, kept, violations, _, err := svc.DiscoverSkillPack(ctx, sess.ID, []string{"microphone lifecycle"}, "architectural")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.False(t, result.Degraded)
	require.Equal(t, "prompt-manager skill read scientific-debugging test", result.ReadCommand)
	require.Len(t, added, 2)
	require.Empty(t, kept)
	require.Len(t, updated.RelevantContext, 2)
	require.Equal(t, planmodel.RelevantContextSkill, updated.RelevantContext[0].Kind)
	require.Equal(t, "scientific-debugging", updated.RelevantContext[0].Target)
	require.Equal(t, planmodel.RelevantContextScopeGlobal, updated.RelevantContext[0].Scope)
	require.Equal(t, planmodel.RelevantContextOncePerExecution, updated.RelevantContext[0].RepeatPolicy)

	updated, _, added, kept, violations, _, err = svc.DiscoverSkillPack(ctx, sess.ID, []string{"microphone lifecycle"}, "architectural")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Empty(t, added)
	require.Len(t, kept, 2)
	require.Len(t, updated.RelevantContext, 2)
}

func TestDiscoverSkillPackDegradesWithoutBlocking(t *testing.T) {
	ctx := context.Background()
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}})
	sess, _, err := svc.StartSession(ctx, "All down", "all-down", "")
	require.NoError(t, err)

	updated, result, added, kept, violations, step, err := svc.DiscoverSkillPack(ctx, sess.ID, []string{"anything"}, "")
	require.NoError(t, err, "skill-pack discovery must never block authoring when prompt-manager is unavailable")
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradedReason, "unavailable")
	require.Empty(t, added)
	require.Empty(t, kept)
	require.Empty(t, violations)
	require.Empty(t, updated.RelevantContext)
	require.NotEqual(t, "context_discovery", step.StepKind)
}

// TestDiscoverSkillPackNormalizesAndValidatesComplexity pins the complexity
// boundary: prompt-manager's levels pass through, agent-habit synonyms are
// mapped, empty omits the flag, and unknown values fail fast naming the
// vocabulary instead of degrading on a downstream 400.
func TestDiscoverSkillPackNormalizesAndValidatesComplexity(t *testing.T) {
	ctx := context.Background()
	emptyPack := `{"results":[],"readCommand":"","budgetStatus":"ok"}`

	cases := []struct {
		name           string
		complexity     string
		wantForwarded  string // "" means the --complexity flag must be absent
		wantErrorNames bool
	}{
		{name: "canonical level passes through", complexity: "architectural", wantForwarded: "architectural"},
		{name: "case and whitespace normalized", complexity: "  MAJOR ", wantForwarded: "major"},
		{name: "synonym high maps to major", complexity: "high", wantForwarded: "major"},
		{name: "synonym low maps to minor", complexity: "low", wantForwarded: "minor"},
		{name: "synonym medium maps to moderate", complexity: "medium", wantForwarded: "moderate"},
		{name: "empty omits the flag", complexity: "", wantForwarded: ""},
		{name: "unknown value rejected naming levels", complexity: "huge", wantErrorNames: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var gotArgs []string
			runner := func(_ context.Context, _ string, args ...string) ([]byte, error) {
				gotArgs = args
				return []byte(emptyPack), nil
			}
			svc := newService(t, authoring.Deps{
				Writer: &fakePlanWriter{},
				Skills: authoring.NewCommandSkillPackDiscoverer(runner),
			})
			sess, _, err := svc.StartSession(ctx, "Complexity probe", "complexity-probe-"+strings.ReplaceAll(tc.name, " ", "-"), "")
			require.NoError(t, err)

			_, result, _, _, _, _, err := svc.DiscoverSkillPack(ctx, sess.ID, []string{"anything"}, tc.complexity)
			if tc.wantErrorNames {
				require.Error(t, err)
				require.ErrorAs(t, err, &authoring.ErrInvalidSession{})
				require.Contains(t, err.Error(), "minor, moderate, major, architectural")
				require.Nil(t, gotArgs, "prompt-manager must not be invoked for an invalid complexity")
				return
			}
			require.NoError(t, err)
			require.False(t, result.Degraded)
			joined := strings.Join(gotArgs, " ")
			if tc.wantForwarded == "" {
				require.NotContains(t, joined, "--complexity")
			} else {
				require.Contains(t, joined, "--complexity "+tc.wantForwarded)
			}
		})
	}
}

// TestDiscoverSkillPackDegradedReasonCarriesOutputAndRecovery pins the
// observability contract: a failing prompt-manager call surfaces the command's
// own diagnostic (not just "exit status 1") plus the manual fallback commands.
func TestDiscoverSkillPackDegradedReasonCarriesOutputAndRecovery(t *testing.T) {
	ctx := context.Background()
	runner := func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		out := []byte("Error: discover failed: api error (400): complexity must be one of: minor, moderate, major, architectural\n")
		return out, fmt.Errorf("run prompt-manager discover: exit status 1")
	}
	svc := newService(t, authoring.Deps{
		Writer: &fakePlanWriter{},
		Skills: authoring.NewCommandSkillPackDiscoverer(runner),
	})
	sess, _, err := svc.StartSession(ctx, "Degraded probe", "degraded-probe", "")
	require.NoError(t, err)

	_, result, _, _, _, _, err := svc.DiscoverSkillPack(ctx, sess.ID, []string{"anything"}, "architectural")
	require.NoError(t, err, "discovery failure degrades, never blocks")
	require.True(t, result.Degraded)
	require.Contains(t, result.DegradedReason, "exit status 1")
	require.Contains(t, result.DegradedReason, "api error (400)", "the command's own diagnostic must survive into the degraded reason")
	require.Contains(t, result.DegradedReason, "fallback:", "degraded reason must name the manual recovery path")
	require.Contains(t, result.DegradedReason, "context-submit "+sess.ID)
}

// TestDiscoveryComplexityContract pins plan-manager's complexity vocabulary to
// prompt-manager's enforcement source. Prompt-manager owns the contract; when
// its levels change, this fails here instead of at authoring time. Skips when
// the sibling scenario is not present (out-of-repo builds).
func TestDiscoveryComplexityContract(t *testing.T) {
	source := filepath.Join("..", "..", "..", "..", "prompt-manager", "api", "aisearch", "handlers.go")
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Skipf("prompt-manager source not available at %s: %v", source, err)
	}
	require.Contains(t, string(raw), "complexity must be one of: minor, moderate, major, architectural",
		"prompt-manager's discovery complexity vocabulary changed; update normalizeDiscoveryComplexity (context_workflow.go) and this pin together")
}

// fakeSkillResolver returns canned steered suggestions (the D7 seam's future
// data-driven sources stand in behind this fake).
type fakeSkillResolver struct{ suggestions []authoring.SkillSuggestion }

func (f fakeSkillResolver) SuggestSkills(context.Context, planmodel.ChangeBoundary) []authoring.SkillSuggestion {
	return f.suggestions
}

// TestSkillContextIsAdvisoryAndResolverSuggestionsAutoAdd verifies that skill
// resolver hints are directly added during skill-pack discovery, while missing
// skill context remains advisory instead of a finalization blocker.
func TestSkillContextIsAdvisoryAndResolverSuggestionsAutoAdd(t *testing.T) {
	ctx := context.Background()
	resolver := fakeSkillResolver{suggestions: []authoring.SkillSuggestion{
		{Slug: "ui-health", Reason: "The boundary touches a React UI."},
	}}
	svc := newService(t, authoring.Deps{Writer: &fakePlanWriter{}, SkillResolver: resolver})
	sess, _, err := svc.StartSession(ctx, "Gate v3", "gate-v3", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sess.ID)

	updated, _, added, _, violations, _, err := svc.DiscoverSkillPack(ctx, sess.ID, []string{"gate semantics"}, "minor")
	require.NoError(t, err)
	require.Empty(t, violations)
	require.Len(t, added, 1)
	require.Equal(t, "ui-health", added[0].Target)
	require.Len(t, updated.RelevantContext, 1)

	valid, violations, _, err := svc.ValidateStructure(ctx, sess.ID)
	require.NoError(t, err)
	require.True(t, valid, "auto-added resolver skill should keep structure valid, got %#v", violations)

	sessSkip, _, err := svc.StartSession(ctx, "Gate v3 skip", "gate-v3-skip", "")
	require.NoError(t, err)
	fillMandatory(t, svc, sessSkip.ID)
	valid, violations, _, err = svc.ValidateStructure(ctx, sessSkip.ID)
	require.NoError(t, err)
	require.True(t, valid, "missing skill context is advisory, not a hard blocker; got %#v", violations)
}
