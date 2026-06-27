package execution

import "strings"

func stepForStarted(e Execution) GuidedStep {
	return GuidedStep{
		StepKind:     "execution_started",
		Title:        "Execution Started",
		Summary:      "The run is linked to the plan and ready for just-in-time phase context.",
		Instructions: []string{"Fetch status before editing so the current phase, required reading, reminders, and validation state are fresh."},
		NextActions: []NextAction{
			{
				ID:     "execution-status",
				Kind:   NextActionRecommended,
				Label:  "Fetch execution status",
				Reason: "Status returns the current phase context for this execution.",
				Argv:   []string{"exec", "status", e.ID},
			},
		},
	}
}

func stepForContext(executionID string, ctx PhaseContext, complete bool) GuidedStep {
	if complete || ctx.Completeness == CompletenessFull || (!ctx.HasCurrent && strings.TrimSpace(ctx.ResumePhaseID) == "") {
		return GuidedStep{
			StepKind:     "execution_complete",
			Title:        "Execution Complete",
			Summary:      "No actionable phase remains.",
			Instructions: []string{"Assemble or inspect the canonical handoff."},
			NextActions: []NextAction{
				{
					ID:     "complete-execution",
					Kind:   NextActionRecommended,
					Label:  "Complete execution",
					Reason: "The runner reports no remaining actionable phase.",
					Argv:   []string{"exec", "complete", executionID},
				},
			},
		}
	}
	phaseID := ctx.ResumePhaseID
	if ctx.HasCurrent && ctx.CurrentPhase.ID != "" {
		phaseID = ctx.CurrentPhase.ID
	}
	step := GuidedStep{
		StepKind:     "phase_context",
		Title:        "Phase Context",
		Summary:      "Use the current phase context to implement, capture decisions/findings, and transition status.",
		Instructions: []string{"Read the phase-scoped required reading and reminders before editing.", "Record decisions or findings as they occur.", "Transition the phase when acceptance is satisfied."},
		NextActions: []NextAction{
			{
				ID:     "transition-active",
				Kind:   NextActionRecommended,
				Label:  "Mark current phase active",
				Reason: "This records that work is underway for the current phase.",
				Argv:   []string{"exec", "transition", executionID, phaseID, "--status", "active"},
			},
			{
				ID:                 "record-decision",
				Kind:               NextActionOptional,
				Label:              "Record decision",
				Reason:             "Capture design choices in-flow so the handoff can be assembled from structured state.",
				Argv:               []string{"exec", "decision", executionID, "--phase", phaseID, "--summary", "<decision summary>", "--detail", "<decision detail>"},
				ContentPlaceholder: "<decision summary> / <decision detail>",
			},
			{
				ID:                 "record-finding",
				Kind:               NextActionOptional,
				Label:              "Record candidate finding",
				Reason:             "Capture possible bugs as candidates for later operator triage.",
				Argv:               []string{"exec", "finding", executionID, "--phase", phaseID, "--title", "<finding title>", "--detail", "<finding detail>"},
				ContentPlaceholder: "<finding title> / <finding detail>",
			},
		},
	}
	if phaseID == "" {
		step.NextActions[0].BlockedBy = []string{"no current phase id"}
	}
	return step
}

func stepForTransition(e Execution) GuidedStep {
	if e.Complete {
		return GuidedStep{
			StepKind:     "transition_complete",
			Title:        "Transition Complete",
			Summary:      "All phases are terminal after this transition.",
			Instructions: []string{"Complete the execution and inspect the canonical handoff."},
			NextActions: []NextAction{
				{
					ID:     "complete-execution",
					Kind:   NextActionRecommended,
					Label:  "Complete execution",
					Reason: "The transition left no current phase.",
					Argv:   []string{"exec", "complete", e.ID},
				},
			},
		}
	}
	return GuidedStep{
		StepKind:     "transition_recorded",
		Title:        "Transition Recorded",
		Summary:      "The phase status changed and the runner pointer was recomputed.",
		Instructions: []string{"Fetch the next phase context before continuing."},
		NextActions: []NextAction{
			{
				ID:     "execution-next",
				Kind:   NextActionRecommended,
				Label:  "Fetch next phase",
				Reason: "GetNext returns the next actionable phase and its just-in-time context.",
				Argv:   []string{"exec", "next", e.ID},
			},
		},
	}
}

func stepForRecorded(executionID, phaseID string) GuidedStep {
	return GuidedStep{
		StepKind:     "captured_state_recorded",
		Title:        "Captured State Recorded",
		Summary:      "The execution has new structured state for the handoff.",
		Instructions: []string{"Continue with the current phase or fetch fresh status if the context may have changed."},
		NextActions: []NextAction{
			{
				ID:     "execution-status",
				Kind:   NextActionRecommended,
				Label:  "Fetch execution status",
				Reason: "Refresh the current phase context after recording state.",
				Argv:   []string{"exec", "status", executionID},
			},
			{
				ID:     "transition-done",
				Kind:   NextActionAlternative,
				Label:  "Mark phase done",
				Reason: "Use when acceptance and validation for the phase are satisfied.",
				Argv:   []string{"exec", "transition", executionID, phaseID, "--status", "done"},
			},
		},
	}
}

func stepForComplete(executionID string, nudges []CompletionNudge) GuidedStep {
	step := GuidedStep{
		StepKind:     "completion_review",
		Title:        "Completion Review",
		Summary:      "The canonical structured handoff has been assembled.",
		Instructions: []string{"Address unsatisfied nudges when they represent real missing captured state.", "Fetch the handoff for the final report."},
		NextActions: []NextAction{
			{
				ID:     "execution-handoff",
				Kind:   NextActionRecommended,
				Label:  "Fetch handoff",
				Reason: "The handoff contains decisions, candidate findings, validation, staleness, and resume state.",
				Argv:   []string{"exec", "handoff", executionID},
			},
		},
	}
	for _, nudge := range nudges {
		if nudge.Satisfied {
			continue
		}
		step.NextActions = append(step.NextActions, NextAction{
			ID:        "completion-nudge-" + nudge.Kind,
			Kind:      NextActionRecovery,
			Label:     "Address " + nudge.Kind,
			Reason:    nudge.Message,
			BlockedBy: []string{nudge.Message},
		})
	}
	return step
}

func stepForHandoff(executionID string) GuidedStep {
	return GuidedStep{
		StepKind:     "handoff_ready",
		Title:        "Handoff Ready",
		Summary:      "Use the structured handoff to produce the final user-facing report.",
		Instructions: []string{"Summarize the handoff without inventing uncaptured state."},
		NextActions: []NextAction{
			{
				ID:     "execution-status",
				Kind:   NextActionOptional,
				Label:  "Refresh status",
				Reason: "Use only if execution state may have changed since handoff assembly.",
				Argv:   []string{"exec", "status", executionID},
			},
		},
	}
}
