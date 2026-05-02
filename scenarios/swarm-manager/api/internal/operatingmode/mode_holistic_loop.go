package operatingmode

func holisticLoopDefinition() Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:                ModeHolisticLoop,
		Label:               "Holistic Loop",
		Description:         "Investigate → plan → execute → review cycles across the whole initiative. Use when scope is exploratory and the plan must be revised as you learn.",
		RunStrategy:         RunStrategyOperatorGatedLoop,
		ArtifactRoot:        "modes/holistic-loop",
		PromptCatalogPrefix: "swarm-manager-holistic-loop",
		DefaultProfileKey:   ProfileDeepWork,
		StartPhase:          "investigate",
		Terminal:            []Phase{"review"},
		Transitions: map[Phase][]Phase{
			"investigate": {"plan"},
			"plan":        {"execute"},
			"execute":     {"review", "investigate"},
			"review":      {"investigate"},
		},
		TransitionRules: map[Phase][]TransitionRule{
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
		},
		Phases: []initiativePhaseSpec{
			{
				Phase:           "investigate",
				Purpose:         "holistic_loop_investigate",
				PromptTitle:     "Holistic Loop Investigate",
				PromptTrigger:   "Operator starts holistic-loop investigate phase",
				PromptPurpose:   "Investigate initiative-wide code, backlog, and system state and produce holistic findings.",
				ProfileKey:      ProfileDeepWork,
				OutputArtifacts: []ArtifactDefinition{requiredOutputArtifact("modes/holistic-loop/findings.md", "text/markdown")},
			},
			{
				Phase:           "plan",
				Purpose:         "holistic_loop_plan",
				PromptTitle:     "Holistic Loop Plan",
				PromptTrigger:   "Operator starts holistic-loop plan phase",
				PromptPurpose:   "Produce or update the initiative-wide implementation plan and readiness assessment.",
				ProfileKey:      ProfileDeepWork,
				OutputArtifacts: []ArtifactDefinition{requiredOutputArtifact("modes/holistic-loop/initiative-plan.md", "text/markdown")},
			},
			{
				Phase:         "execute",
				Purpose:       "holistic_loop_execute",
				PromptTitle:   "Holistic Loop Execute",
				PromptTrigger: "Operator starts holistic-loop execute phase",
				PromptPurpose: "Execute against the initiative-wide plan and report whether replanning is needed.",
				ProfileKey:    ProfileDeepWork,
				WritesRepo:    true,
				Metrics:       PhaseMetricsSpec{CountsReplanSample: true},
			},
			{
				Phase:            "review",
				Purpose:          "holistic_loop_review",
				PromptTitle:      "Holistic Loop Acceptance Review",
				PromptTrigger:    "Operator starts holistic-loop review phase",
				PromptPurpose:    "Evaluate holistic-loop results against acceptance criteria and produce an acceptance verdict.",
				ProfileKey:       ProfileAnalysis,
				RequiresVerdict:  true,
				RequiresCriteria: true,
				Metrics:          PhaseMetricsSpec{CountsAcceptanceSample: true},
			},
		},
	})
}
