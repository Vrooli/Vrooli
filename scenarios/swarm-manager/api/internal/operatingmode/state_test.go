package operatingmode

import "testing"

func TestComputePhaseActionsUsesBooleanGuardTransitions(t *testing.T) {
	def := buildTransitionTestMode(map[Phase][]GuardedTransition{
		"execute": {
			{
				When: Guard{Op: GuardOpEq, Field: payloadReplanNeeded, Value: true},
				To:   []Phase{"investigate"},
			},
			{
				When: Guard{Op: GuardOpAlways},
				To:   []Phase{"review"},
			},
		},
	})

	replanActions := phaseActionsByPhase(ComputePhaseActions(PhaseStateInput{
		Definition: def,
		Rounds: []RoundEnvelope{
			completedRound("execute", map[string]any{payloadReplanNeeded: true}),
		},
	}))
	if !replanActions["investigate"].Startable {
		t.Fatalf("investigate action = %+v, want startable after replan signal", replanActions["investigate"])
	}
	if replanActions["review"].Startable {
		t.Fatalf("review action = %+v, want blocked after replan signal", replanActions["review"])
	}

	reviewActions := phaseActionsByPhase(ComputePhaseActions(PhaseStateInput{
		Definition: def,
		Rounds: []RoundEnvelope{
			completedRound("execute", nil),
		},
		AcceptanceCriteria: []string{"review the result"},
	}))
	if !reviewActions["review"].Startable {
		t.Fatalf("review action = %+v, want startable without replan signal", reviewActions["review"])
	}
	if reviewActions["investigate"].Startable {
		t.Fatalf("investigate action = %+v, want blocked without replan signal", reviewActions["investigate"])
	}
}

func TestComputePhaseActionsUsesProgressDecisionGuardTransitions(t *testing.T) {
	def := buildTransitionTestMode(map[Phase][]GuardedTransition{
		"classify": {
			{
				When: Guard{Op: GuardOpEq, Field: "progress.decision", Value: string(ProgressContinue)},
				To:   []Phase{"execute"},
			},
			{
				When: Guard{Op: GuardOpEq, Field: "progress.decision", Value: string(ProgressReplan)},
				To:   []Phase{"investigate"},
			},
			{
				When: Guard{Op: GuardOpEq, Field: "progress.decision", Value: string(ProgressComplete)},
				To:   []Phase{"review"},
			},
			{
				When: Guard{Op: GuardOpEq, Field: "progress.decision", Value: string(ProgressBlocked)},
				To:   nil,
			},
		},
	})

	continueActions := phaseActionsByPhase(ComputePhaseActions(PhaseStateInput{
		Definition: def,
		Rounds: []RoundEnvelope{
			completedRound("classify", map[string]any{
				payloadProgress: ProgressState{Decision: ProgressContinue},
			}),
		},
	}))
	if !continueActions["execute"].Startable {
		t.Fatalf("execute action = %+v, want startable after continue", continueActions["execute"])
	}

	replanActions := phaseActionsByPhase(ComputePhaseActions(PhaseStateInput{
		Definition: def,
		Rounds: []RoundEnvelope{
			completedRound("classify", map[string]any{
				payloadProgress: map[string]any{"decision": string(ProgressReplan)},
			}),
		},
	}))
	if !replanActions["investigate"].Startable {
		t.Fatalf("investigate action = %+v, want startable after replan", replanActions["investigate"])
	}

	blockedActions := phaseActionsByPhase(ComputePhaseActions(PhaseStateInput{
		Definition: def,
		Rounds: []RoundEnvelope{
			completedRound("classify", map[string]any{
				payloadProgress: ProgressState{Decision: ProgressBlocked},
			}),
		},
		AcceptanceCriteria: []string{"review the result"},
	}))
	for phase, action := range blockedActions {
		if action.Startable {
			t.Fatalf("%s action = %+v, want no startable phase after blocked decision", phase, action)
		}
	}
}

// buildTransitionTestMode builds a minimal in-memory initiative Definition whose
// branching is the given generic guard graph, exercising the runtime routing
// (ComputePhaseActions → nextPhasesForCompletedRound → the guard evaluator)
// without touching the on-disk mode data.
func buildTransitionTestMode(guards map[Phase][]GuardedTransition) Definition {
	return Definition{
		Mode:        "transition-test",
		Label:       "Transition Test",
		Scope:       ScopePolicy{Kind: ScopeInitiative},
		RunStrategy: RunStrategyPolicy{Kind: RunStrategyOperatorGatedLoop},
		PhaseGraph: PhaseGraph{
			StartPhase: "investigate",
			Terminal:   []Phase{"review"},
			Transitions: map[Phase][]Phase{
				"investigate": {"execute"},
				"execute":     {"classify", "investigate", "review"},
				"classify":    {"execute", "investigate", "review"},
				"review":      {"investigate"},
			},
			Guards: guards,
			Phases: map[Phase]PhaseDefinition{
				"investigate": {Phase: "investigate", Kind: PhaseKindInvestigate, ActivityPurpose: "transition_test_investigate", ProfileKey: ProfileDeepWork},
				"execute":     {Phase: "execute", Kind: PhaseKindExecute, ActivityPurpose: "transition_test_execute", ProfileKey: ProfileDeepWork},
				"classify":    {Phase: "classify", Kind: PhaseKindReview, ActivityPurpose: "transition_test_classify", ProfileKey: ProfileAnalysis},
				"review":      {Phase: "review", Kind: PhaseKindReview, ActivityPurpose: "transition_test_review", ProfileKey: ProfileAnalysis, RequiresCriteria: true},
			},
		},
	}
}

func completedRound(phase Phase, payload map[string]any) RoundEnvelope {
	return RoundEnvelope{
		Round:   1,
		Phase:   string(phase),
		Status:  RoundStatusCompleted,
		Payload: payload,
	}
}

func phaseActionsByPhase(actions []PhaseAction) map[Phase]PhaseAction {
	out := make(map[Phase]PhaseAction, len(actions))
	for _, action := range actions {
		out[action.Phase] = action
	}
	return out
}
