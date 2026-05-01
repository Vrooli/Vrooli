package operatingmode

func phasedPlanDrainDefinition() Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:                ModePhasedPlanDrain,
		Label:               "Phased Plan Drain",
		RunStrategy:         RunStrategySequentialHandoff,
		ArtifactRoot:        "modes/phased-plan-drain",
		PromptCatalogPrefix: "swarm-manager-phased-plan",
		DefaultProfileKey:   ProfileDeepWork,
		StartPhase:          "prepare_plan",
		Terminal:            []Phase{"review"},
		Transitions: map[Phase][]Phase{
			"prepare_plan":      {"execute_next"},
			"execute_next":      {"classify_progress"},
			"classify_progress": {"execute_next", "prepare_plan", "review"},
			"review":            {"prepare_plan"},
		},
		TransitionRules: map[Phase][]TransitionRule{
			"classify_progress": {
				{
					When: TransitionCondition{
						Kind:             TransitionConditionProgressDecision,
						ProgressDecision: ProgressContinue,
					},
					Next: []Phase{"execute_next"},
				},
				{
					When: TransitionCondition{
						Kind:             TransitionConditionProgressDecision,
						ProgressDecision: ProgressReplan,
					},
					Next: []Phase{"prepare_plan"},
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
		},
		Phases: []initiativePhaseSpec{
			{
				Phase:           "prepare_plan",
				Purpose:         "phased_plan_prepare",
				PromptSuffix:    "prepare",
				PromptTitle:     "Phased Plan Prepare",
				PromptTrigger:   "Operator starts phased-plan-drain prepare phase",
				PromptPurpose:   "Create or update the stable phased plan used by sequential handoff execution.",
				ProfileKey:      ProfileDeepWork,
				OutputArtifacts: []ArtifactDefinition{requiredOutputArtifact("modes/phased-plan-drain/phased-plan.md", "text/markdown")},
			},
			{
				Phase:           "execute_next",
				Purpose:         "phased_plan_execute_next",
				PromptSuffix:    "execute-next",
				PromptTitle:     "Phased Plan Execute Next",
				PromptTrigger:   "Operator starts phased-plan-drain execute-next phase",
				PromptPurpose:   "Execute the earliest contiguous phase block that can be completed professionally and emit a final handoff.",
				ProfileKey:      ProfileDeepWork,
				WritesRepo:      true,
				RequiresHandoff: true,
				Metrics:         PhaseMetricsSpec{CountsReplanSample: true},
			},
			{
				Phase:            "classify_progress",
				Purpose:          "phased_plan_classify_progress",
				PromptSuffix:     "classify-progress",
				PromptTitle:      "Phased Plan Classify Progress",
				PromptTrigger:    "Operator starts phased-plan-drain classify-progress phase",
				PromptPurpose:    "Classify phased-plan progress and record backlog reconciliation intent.",
				ProfileKey:       ProfileAnalysis,
				ResultBindings:   []ResultBinding{progressResultArtifact("modes/phased-plan-drain/progress.json")},
				RequiresProgress: true,
			},
			{
				Phase:            "review",
				Purpose:          "phased_plan_review",
				PromptTitle:      "Phased Plan Acceptance Review",
				PromptTrigger:    "Operator starts phased-plan-drain review phase",
				PromptPurpose:    "Evaluate phased-plan-drain results, handoffs, and progress against acceptance criteria.",
				ProfileKey:       ProfileAnalysis,
				RequiresVerdict:  true,
				RequiresCriteria: true,
				Metrics:          PhaseMetricsSpec{CountsAcceptanceSample: true},
			},
		},
	})
}
