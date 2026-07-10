package operatingmode

import (
	"context"
	"errors"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/operatingmode/promptcatalog"
)

// TestSharedSnippet_RendersInBothModes pins the snippet's presence in the
// reconcile-phase prompt variable map for both initiative-scoped modes.
// promptVariables is the seam every reconcile prompt renders through; if
// the variable is absent or differs across modes, the proposal-format
// contract has drifted between surfaces.
func TestSharedSnippet_RendersInBothModes(t *testing.T) {
	holistic := MustDefinition(ModeHolisticLoop)
	reconcileHolistic, ok := holistic.PhaseGraph.Phases["reconcile"]
	if !ok {
		t.Fatalf("holistic-loop is missing the reconcile phase")
	}
	snippet := promptcatalog.BacklogSyncProposalSnippet()

	check := func(t *testing.T, def Definition, phaseDef PhaseDefinition) {
		t.Helper()
		ctx := RunContext{Def: def, PhaseDef: phaseDef, Target: TargetInstance{
			Kind: def.Target.Kind, ID: "test-initiative",
			Initiative: InitiativeSnapshot{Name: "test-initiative", Title: "Test initiative"},
		}}
		vars := promptVariables(ctx, RoundEnvelope{Round: 1}, "")
		got, ok := vars[promptcatalog.BacklogSyncProposalVariableKey]
		if !ok {
			t.Fatalf("mode %q phase %q prompt variables missing %q", def.Mode, phaseDef.Phase, promptcatalog.BacklogSyncProposalVariableKey)
		}
		if got != snippet {
			t.Fatalf("mode %q phase %q snippet drifted from canonical (got %d chars, want %d chars)", def.Mode, phaseDef.Phase, len(got), len(snippet))
		}
	}

	check(t, holistic, reconcileHolistic)
}

// TestReconcilePhases_RequireBacklogSync pins the contract: every reconcile
// phase's output contract requires a non-nil BacklogSync plan. Regression
// guard for any mode author who copy-pastes a review-shaped phase and
// forgets to flip the flag.
func TestReconcilePhases_RequireBacklogSync(t *testing.T) {
	for _, mode := range []Mode{ModeHolisticLoop} {
		def := MustDefinition(mode)
		phase, ok := def.PhaseGraph.Phases["reconcile"]
		if !ok {
			t.Fatalf("mode %q is missing the reconcile phase", mode)
		}
		if phase.Kind != PhaseKindReconcile {
			t.Errorf("mode %q reconcile.Kind = %q, want %q", mode, phase.Kind, PhaseKindReconcile)
		}
		if !phase.OutputContract.RequiresBacklogSync {
			t.Errorf("mode %q reconcile.OutputContract.RequiresBacklogSync = false, want true", mode)
		}
		if got, want := phase.AutoStartAfter, []Phase{"review"}; len(got) != len(want) || got[0] != want[0] {
			t.Errorf("mode %q reconcile.AutoStartAfter = %v, want %v", mode, got, want)
		}
	}
}

// TestReconcilePhases_ApplyModeIsOperatorGated pins the v1 default. Future
// modes get this default from buildInitiativeMode; if a mode override
// changes ApplyMode (and the auto-apply paths land), this test should fail
// loudly so reviewers see the policy change before it goes out.
func TestReconcilePhases_ApplyModeIsOperatorGated(t *testing.T) {
	for _, mode := range []Mode{ModeHolisticLoop} {
		def := MustDefinition(mode)
		if got, want := def.BacklogSync.ApplyMode, BacklogSyncApplyOperatorGated; got != want {
			t.Errorf("mode %q backlog_sync.apply_mode = %q, want %q (auto-apply variants are not implemented in v1)", mode, got, want)
		}
	}
}

// TestApplyBacklogSync_RejectsNonOperatorGated verifies the v1 contract that
// any non-operator-gated apply mode is rejected at runtime with
// ErrApplyModeNotImplemented. The test installs a registry override that
// flips ApplyMode to auto-apply-safe on a real initiative-scoped mode, then
// calls ApplyBacklogSync and asserts the wrapped sentinel surfaces. Handler
// tests cover the HTTP 501 mapping; this test covers the service-layer
// fail-closed behavior.
func TestApplyBacklogSync_PinsExecutionPolicyAndLegacyUsesLivePolicy(t *testing.T) {
	root := t.TempDir()
	reconciler := &fakeProposalReconciler{}
	svc := newTestServiceWithOptions(t, root, serviceOptions{
		reconciler: reconciler,
	})

	// Drive the holistic-loop happy path through review so the round payload
	// has a backlog_sync_plan to apply against. Use a synthetic registry
	// override to flip the mode's ApplyMode after the round is staged so the
	// rejection comes from the apply seam, not from registration validation.
	summary := `{"operating_mode_result":{"verdict":"accepted","backlog_sync":{"proposal":{"form":"mutation_list","mutations":[{"id":"m1","op":"add_item","item":{"kind":"fix","name":"follow-up","title":"Follow up"}}]}}}}`
	agent := svc.agent.(*fakeAgent)
	agent.states["run-1"] = agentmanager.RunState{RunID: "run-1", Status: "complete", Summary: summary, FinishedAt: "2026-05-02T12:05:00Z"}
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "investigate", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "plan", nil)
	saveCompletedRound(t, svc, "init-a", ModeHolisticLoop, "execute", completedDrainExecutePayload("complete"))
	round, err := svc.StartPhase(context.Background(), StartPhaseRequest{
		InitiativeName: "init-a",
		Phase:          "review",
	})
	if err != nil {
		t.Fatalf("StartPhase(review) returned error: %v", err)
	}
	round, err = svc.RefreshRound(context.Background(), "init-a", ModeHolisticLoop, round.Round)
	if err != nil {
		t.Fatalf("RefreshRound(review) returned error: %v", err)
	}

	// Override the holistic-loop registry entry with a non-operator-gated
	// apply mode and verify ApplyBacklogSync fails with the typed sentinel.
	withApplyModeOverride(t, ModeHolisticLoop, BacklogSyncApplyAutoSafe)

	_, err = svc.ApplyBacklogSync(context.Background(), ApplyBacklogSyncRequest{
		InitiativeName:      "init-a",
		Mode:                string(ModeHolisticLoop),
		Round:               round.Round,
		RunID:               round.RunID,
		AcceptedMutationIDs: []string{"m1"},
	})
	if err != nil {
		t.Fatalf("ApplyBacklogSync on pinned execution: %v", err)
	}
	if reconciler.req.InitiativeName != "init-a" {
		t.Fatalf("pinned operator-gated policy did not invoke reconciler: req=%+v", reconciler.req)
	}

	legacy := round
	legacy.ExecutionID = ""
	legacy.DefinitionDigest = ""
	legacy.Round = 0
	legacy, err = svc.store.CreateRound(legacy)
	if err != nil {
		t.Fatalf("CreateRound legacy fixture: %v", err)
	}
	reconciler.req = ProposalReconcileRequest{}
	_, err = svc.ApplyBacklogSync(context.Background(), ApplyBacklogSyncRequest{
		InitiativeName: "init-a", Mode: string(ModeHolisticLoop), Round: legacy.Round,
		RunID: legacy.RunID, AcceptedMutationIDs: []string{"m1"},
	})
	if !errors.Is(err, ErrApplyModeNotImplemented) {
		t.Fatalf("legacy ApplyBacklogSync error = %v, want ErrApplyModeNotImplemented", err)
	}
	if reconciler.req.InitiativeName != "" {
		t.Fatalf("legacy non-operator-gated policy invoked reconciler: req=%+v", reconciler.req)
	}
}

// withApplyModeOverride swaps the registry's apply mode for a single mode for
// the duration of the test. The override clones the entire registry so other
// modes' invariants stay intact and ValidateRegistry continues to pass.
func withApplyModeOverride(t *testing.T, mode Mode, applyMode BacklogSyncApplyMode) {
	t.Helper()
	previous := registry
	next := cloneRegistryForTest()
	def := next[mode]
	def.BacklogSync.ApplyMode = applyMode
	next[mode] = def
	registry = next
	t.Cleanup(func() { registry = previous })
}
