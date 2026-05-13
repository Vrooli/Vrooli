package operatingmode

func phasedPlanDrainDefinition() Definition {
	return buildInitiativeMode(initiativeModeSpec{
		Mode:        ModePhasedPlanDrain,
		Label:       "Phased Plan Drain",
		Description: "Prepare a stable phased plan, then drain it with sequential handoff runs that classify progress and reconcile the backlog. Use when the work decomposes cleanly upfront.",
		BestFor: []string{
			"Long sequential plans that agents drain over many runs",
			"A multi-phase plan can be prepared once and stays stable",
			"Continuity across handoffs matters more than parallelism",
			"Explicit progress classification between slices is valuable",
		},
		NotFor: []string{
			"The plan is exploratory or unstable — use holistic-loop",
			"Items are independent and parallel execution wins — use item-level",
			"Work is small enough to fit in a single backlog item",
		},
		Tradeoffs: []string{
			"Continuity over parallelism — one slice at a time",
			"Less planning churn than holistic loop once the plan is stable",
			"Explicit progress classification (continue / replan / complete / blocked) between slices",
			"Heavier upfront plan preparation than holistic loop",
		},
		WhenInDoubtPickInstead: ModeHolisticLoop,
		RunStrategy:            RunStrategySequentialHandoff,
		ArtifactRoot:           "modes/phased-plan-drain",
		PromptCatalogPrefix:    "swarm-manager-phased-plan",
		DefaultProfileKey:      ProfileDeepWork,
		StartPhase:             "prepare_plan",
		Terminal:               []Phase{"reconcile"},
		Transitions: map[Phase][]Phase{
			"prepare_plan":      {"execute_next"},
			"execute_next":      {"classify_progress"},
			"classify_progress": {"execute_next", "prepare_plan", "review"},
			"review":            {"reconcile"},
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
				Kind:            PhaseKindInvestigate,
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
				Kind:            PhaseKindExecute,
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
				Kind:             PhaseKindReview,
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
				Kind:             PhaseKindReview,
				Purpose:          "phased_plan_review",
				PromptTitle:      "Phased Plan Acceptance Review",
				PromptTrigger:    "Operator starts phased-plan-drain review phase",
				PromptPurpose:    "Evaluate phased-plan-drain results, handoffs, and progress against acceptance criteria.",
				ProfileKey:       ProfileAnalysis,
				RequiresVerdict:  true,
				RequiresCriteria: true,
				Metrics:          PhaseMetricsSpec{CountsAcceptanceSample: true},
			},
			{
				Phase:               "reconcile",
				Kind:                PhaseKindReconcile,
				AutoStartAfter:      []Phase{"review"},
				Purpose:             "phased_plan_reconcile",
				PromptSuffix:        "reconcile",
				PromptTitle:         "Phased Plan Backlog Reconcile",
				PromptTrigger:       "Round refresher auto-starts phased-plan-drain reconcile after review completes",
				PromptPurpose:       "Read prior round artifacts and propose backlog mutations that align the initiative with the drained plan.",
				ProfileKey:          ProfileAnalysis,
				RequiresBacklogSync: true,
			},
		},
	})
}
