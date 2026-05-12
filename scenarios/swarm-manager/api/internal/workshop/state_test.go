package operatingmode

import "testing"

func TestComputePhaseActionsUsesPayloadBoolTransitionRules(t *testing.T) {
	def := buildTransitionTestMode(map[Phase][]TransitionRule{
		"execute": {
			{
				When: TransitionCondition{
					Kind:       TransitionConditionPayloadBool,
					PayloadKey: payloadReplanNeeded,
					BoolValue:  true,
				},
				Next: []Phase{"investigate"},
			},
			{
				When: TransitionCondition{Kind: TransitionConditionAlways},
				Next: []Phase{"review"},
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

func TestComputePhaseActionsUsesProgressDecisionTransitionRules(t *testing.T) {
	def := buildTransitionTestMode(map[Phase][]TransitionRule{
		"classify": {
			{
				When: TransitionCondition{
					Kind:             TransitionConditionProgressDecision,
					ProgressDecision: ProgressContinue,
				},
				Next: []Phase{"execute"},
			},
			{
				When: TransitionCondition{
					Kind:             TransitionConditionProgressDecision,
					ProgressDecision: ProgressReplan,
				},
				Next: []Phase{"investigate"},
			},
			{
				When: TransitionCondition{
					Kind:             TransitionConditionProgressDecision,
					ProgressDecision: ProgressComplete,
				},
				Next: []Phase{"review"},
			},
			{
				When: TransitionCondition{
					Kind:             TransitionConditionProgressDecision,
					ProgressDecision: ProgressBlocked,
				},
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

func buildTransitionTestMode(rules map[Phase][]TransitionRule) Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:                "transition-test",
		Label:               "Transition Test",
		RunStrategy:         RunStrategyOperatorGatedLoop,
		ArtifactRoot:        "modes/transition-test",
		PromptCatalogPrefix: "swarm-manager-transition-test",
		DefaultProfileKey:   ProfileDeepWork,
		StartPhase:          "investigate",
		Terminal:            []Phase{"review"},
		Transitions: map[Phase][]Phase{
			"investigate": {"execute"},
			"execute":     {"classify", "investigate", "review"},
			"classify":    {"execute", "investigate", "review"},
			"review":      {"investigate"},
		},
		TransitionRules: rules,
		Phases: []initiativePhaseSpec{
			{
				Phase:      "investigate",
				Purpose:    "transition_test_investigate",
				ProfileKey: ProfileDeepWork,
			},
			{
				Phase:      "execute",
				Purpose:    "transition_test_execute",
				ProfileKey: ProfileDeepWork,
			},
			{
				Phase:      "classify",
				Purpose:    "transition_test_classify",
				ProfileKey: ProfileAnalysis,
			},
			{
				Phase:            "review",
				Purpose:          "transition_test_review",
				ProfileKey:       ProfileAnalysis,
				RequiresCriteria: true,
			},
		},
	})
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
