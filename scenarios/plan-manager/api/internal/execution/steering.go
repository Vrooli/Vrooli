package execution

import (
	"fmt"
	"strconv"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

// boundaryReminders renders just-in-time change-boundary reminders for the phase
// context step: the planned paths, denied paths, which affected scenarios have a
// baseline oracle, and where validation coverage is only informational. Returns
// nil when the plan carries no boundary.
//
// The allow and deny halves are deliberately NOT symmetric in force.
// acceptance_allow is the plan's authored blast-radius ESTIMATE — it drives
// posture, anchor, and validation scope, but an estimate made before the work
// started cannot be a permission ceiling. An agent that reads it as one writes a
// workaround rather than the clean change, which is a worse outcome than a
// recorded, validated scope extension. acceptance_deny stays hard: it is the
// authored guardrail and the only half that expresses a real prohibition.
func boundaryReminders(b planmodel.ChangeBoundary) []string {
	if b.IsZero() {
		return nil
	}
	var out []string
	if len(b.AcceptanceAllow) > 0 {
		out = append(out,
			"Planned change boundary (expected edits): "+strings.Join(b.AcceptanceAllow, ", "),
			"That boundary is the plan's blast-radius estimate, not a permission ceiling. If the phase's intent needs an edit outside it — a shared type, a proto, the API shape a handler depends on — make the edit properly. Do NOT write a workaround to stay inside the boundary.",
			"Before editing outside the boundary, run `exec boundary-extend <execution> --paths <glob> --reason <why>` so validation scope follows the edit, then record the choice with `log decision-add`. An unextended outside edit is real work that no oracle covers.")
	}
	if len(b.AcceptanceDeny) > 0 {
		out = append(out, "Forbidden paths (do NOT edit, and boundary-extend will refuse them): "+strings.Join(b.AcceptanceDeny, ", "))
	}
	if scenarios := b.AffectedScenarios(); len(scenarios) > 0 {
		out = append(out, "Scenario baseline oracle covers: "+strings.Join(scenarios, ", "))
	}
	if repo := b.RepoPaths(); len(repo) > 0 {
		out = append(out, "No scenario baseline oracle for "+strings.Join(repo, ", ")+" — repo/path diffs there are informational only.")
	}
	if b.OperatorOnlyReason != "" {
		out = append(out, "Operator-only plan (no editable repo paths): "+b.OperatorOnlyReason)
	}
	return out
}

// waitDiscipline is the shared disposition line for every producer-owned step
// that routinely runs for tens of minutes (baseline capture, suite validation).
// Without it agents read a slow run as a stalled one and abandon the phase, or
// they burn tokens re-checking every few seconds. Both failure modes are common
// enough to be worth restating at each long-wait seam rather than relying on the
// agent having read docs/TESTING.md.
const waitDiscipline = "Baseline capture and suite validation routinely run for tens of minutes. Block ONCE on the producer's own wait verb and let it return. Do not poll in a loop, and do not re-run the command while a run is in flight. A run still in progress is NOT a blocker: it is not a reason to stop, re-scope the phase, or report partial completion."

func stepForStarted(e Execution) GuidedStep {
	return GuidedStep{
		StepKind:     "execution_started",
		Title:        "Execution Started",
		Summary:      "The run is linked to the plan and ready for just-in-time setup context.",
		Instructions: []string{"Fetch context before editing so the current phase, structured setup items, reminders, and validation state are fresh."},
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

func stepForAbandoned(e Execution) GuidedStep {
	return GuidedStep{
		StepKind:     "execution_abandoned",
		Title:        "Execution Abandoned",
		Summary:      "The execution is terminal and retained as audit history.",
		Instructions: []string{"Start a new execution for the plan if work should continue."},
		NextActions: []NextAction{{
			ID: "start-replacement", Kind: NextActionRecommended, Label: "Start replacement execution",
			Reason: "Abandoned executions cannot resume.", Argv: []string{"exec", "start", e.PlanID},
		}},
	}
}

func stepForContext(executionID, planID string, ctx PhaseContext, complete bool) GuidedStep {
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
	if ctx.BaselineSet.LegacyAdoptionRequired {
		return legacyBaselineAdoptionStep(executionID)
	}
	if ctx.BaselineSet.Name != "" && !ctx.BaselineSet.Complete() {
		return baselineRequiredStep(ctx.BaselineSet)
	}
	phaseID := ctx.ResumePhaseID
	if ctx.HasCurrent && ctx.CurrentPhase.ID != "" {
		phaseID = ctx.CurrentPhase.ID
	}
	instructions := []string{"Run or read the structured setup items before editing.", "Capture feedback in the log ledger as it happens: decisions, candidate findings, confirmed bugs, reusable records, and notes.", "Run validation before marking the phase done."}
	if reminders := boundaryReminders(ctx.ChangeBoundary); len(reminders) > 0 {
		instructions = append(reminders, instructions...)
	}
	step := GuidedStep{
		StepKind:     "phase_context",
		Title:        "Phase Context",
		Summary:      "Use the current phase setup context to implement, capture decisions/findings, and transition status.",
		Instructions: instructions,
		NextActions: []NextAction{
			phasePrimaryAction(executionID, planID, phaseID, ctx),
			{
				ID:                 "log-decision",
				Kind:               NextActionOptional,
				Label:              "Record decision",
				Reason:             "Capture design choices in-flow so the handoff can be assembled from the log ledger.",
				Argv:               []string{"log", "decision-add", executionID, "--phase", phaseID, "--title", "<decision summary>", "--detail", "<decision detail>"},
				ContentPlaceholder: "<decision summary> / <decision detail>",
			},
			{
				ID:                 "log-finding",
				Kind:               NextActionOptional,
				Label:              "Record candidate finding",
				Reason:             "Capture possible bugs as candidate findings for later triage/promotion.",
				Argv:               []string{"log", "finding-add", executionID, "--phase", phaseID, "--title", "<finding title>", "--detail", "<finding detail>"},
				ContentPlaceholder: "<finding title> / <finding detail>",
			},
			{
				ID:                 "log-bug",
				Kind:               NextActionOptional,
				Label:              "File confirmed bug",
				Reason:             "File confirmed defects in-flow; Plan Manager keeps the entry durable and forwards it internally when configured.",
				Argv:               []string{"log", "bug-add", executionID, "--phase", phaseID, "--title", "<bug title>", "--detail", "<bug detail>"},
				ContentPlaceholder: "<bug title> / <bug detail>",
			},
			{
				ID:                 "log-record",
				Kind:               NextActionOptional,
				Label:              "Capture reusable record",
				Reason:             "Capture reusable learning or completed work before the final handoff.",
				Argv:               []string{"log", "record-add", executionID, "--phase", phaseID, "--title", "<record title>", "--detail", "<record detail>"},
				ContentPlaceholder: "<record title> / <record detail>",
			},
			{
				ID:                 "log-note",
				Kind:               NextActionOptional,
				Label:              "Record progress note",
				Reason:             "Capture lightweight progress or context that should survive resume.",
				Argv:               []string{"log", "note-add", executionID, "--phase", phaseID, "--title", "<note title>", "--detail", "<note detail>"},
				ContentPlaceholder: "<note title> / <note detail>",
			},
			{
				ID:     "log-no-feedback",
				Kind:   NextActionOptional,
				Label:  "Confirm no feedback",
				Reason: "Record an explicit durable checkpoint when the phase has no decisions, findings, bugs, records, or notes to capture.",
				Argv:   []string{"log", "note-add", executionID, "--phase", phaseID, "--title", NoFeedbackCheckpointTitle, "--detail", noFeedbackCheckpointDetail},
			},
		},
	}
	if phaseID == "" {
		step.NextActions[0].BlockedBy = []string{"no current phase id"}
	}
	return step
}

func legacyBaselineAdoptionStep(executionID string) GuidedStep {
	return GuidedStep{
		StepKind: "baseline_adoption_required", Title: "Historical Baseline Adoption Required",
		Summary:      "This historical plan has no trustworthy collection baseline yet.",
		Instructions: []string{"Choose recapture only when the current state still represents the missing before behavior. Ordinary workspace dirtiness is not a reason to degrade: scoped baseline capture remains valid. Otherwise record a degraded path and use partial handoff; normal completion remains disabled."},
		NextActions: []NextAction{
			{ID: "baseline-adopt-recapture", Kind: NextActionRecommended, Label: "Adopt a recaptured collection baseline", Reason: "Creates a producer ticket only; Git Control Tower still owns capture and native wait.", Argv: []string{"exec", "baseline-adopt", executionID, "--mode", "recapture", "--name", "<collection-name>", "--members", "<scenario,...>", "--reason", "<why this is a trustworthy before-state>"}},
			{ID: "baseline-adopt-degraded", Kind: NextActionRecovery, Label: "Record degraded legacy execution", Reason: "Use when a valid before-state cannot be recreated; normal completion is intentionally unavailable.", Argv: []string{"exec", "baseline-adopt", executionID, "--mode", "degraded", "--reason", "<why no trustworthy before-state exists>"}},
		},
	}
}

func baselineRequiredStep(state BaselineSetState) GuidedStep {
	if state.Status == BaselineSetStatusScopeRepairRequired {
		instructions := []string{
			"Git Control Tower's immediate source-evidence estimate requires a narrower selection. Behavioral collection members remain required; source evidence is informational and was not silently omitted.",
			fmt.Sprintf("Measured estimate: %d eligible files, %d bytes; %d ignored files excluded; projected retained content %d bytes.", state.SourcePreflight.EligibleFiles, state.SourcePreflight.EligibleBytes, state.SourcePreflight.ExcludedIgnoredFiles, state.SourcePreflight.RetainedContentBytes),
		}
		for _, contributor := range state.SourcePreflight.TopContributors {
			instructions = append(instructions, fmt.Sprintf("Contributor: %s (%d files, %d bytes)", contributor.Path, contributor.Files, contributor.Bytes))
		}
		for _, recommendation := range state.SourcePreflight.Recommendations {
			instructions = append(instructions, "Repair selection: "+recommendation.Selection+" ("+recommendation.Reason+")")
		}
		recommendation := "<boundary-contained-path>"
		if len(state.SourcePreflight.Recommendations) > 0 {
			recommendation = state.SourcePreflight.Recommendations[0].Selection
		}
		return GuidedStep{
			StepKind: "baseline_scope_repair_required", Title: "Baseline Source Scope Repair Required", Summary: state.Detail, Instructions: instructions,
			NextActions: []NextAction{{ID: "baseline-scope-repair", Kind: NextActionRecommended, Label: "Repair and re-estimate source scope", Reason: "Plan Manager boundary-checks the replacement and asks GCT again; it will issue capture only if that estimate is safe.", Argv: []string{"exec", "baseline-scope-repair", "<execution-id>", "--paths", recommendation, "--reason", "<why this narrow selection is relevant>"}}},
		}
	}
	if state.PreflightUnavailable {
		return GuidedStep{StepKind: "baseline_preflight_unavailable", Title: "Baseline Source Preflight Unavailable", Summary: state.Detail, Instructions: []string{"Do not assume source evidence is safe while Git Control Tower is unavailable. Restore the provider and continue so Plan Manager can issue a fresh authoritative preflight."}}
	}
	if state.Status == BaselineSetStatusPartial && state.Failed > 0 {
		reanchor := append([]string(nil), state.CaptureArgv...)
		reanchor = append(reanchor, "--acknowledge-reanchor")
		return GuidedStep{
			StepKind: "baseline_reanchor_required", Title: "Baseline Re-anchor Required",
			Summary:      "A prior immutable capture failed after source changed. Continue forward by re-anchoring this same collection at the current source state; do not revert source or create a differently named ticket.",
			Instructions: []string{"The failed attempt remains in Git Control Tower history. This acknowledged recapture creates a new immutable generation under the same collection name. Future diffs are valid from that generation forward, but it does not represent the original pre-work state."},
			NextActions: []NextAction{
				{ID: "baseline-reanchor", Kind: NextActionRecommended, Label: "Re-anchor baseline at current source", Reason: "Explicit acknowledgement is required because the original before-state was invalidated; this never resets or stashes source.", Argv: reanchor},
				{ID: "baseline-wait", Kind: NextActionRecovery, Label: "Wait for re-anchored baseline", Reason: "Git Control Tower owns the durable wait.", Argv: state.WaitArgv},
				{ID: "baseline-sync", Kind: NextActionRecovery, Label: "Synchronize re-anchored baseline", Reason: "Record the terminal collection generation in this execution.", Argv: state.SyncArgv},
			},
		}
	}
	return GuidedStep{
		StepKind: "baseline_required", Title: "Baseline Required",
		Summary:      "Git Control Tower must capture the immutable before-state before normal phase work can begin.",
		Instructions: []string{"Run the producer-owned capture command. Use Git Control Tower's own printed one-shot wait/recovery command; do not wait through Plan Manager. Then synchronize the durable collection result here.", waitDiscipline},
		NextActions: []NextAction{
			{ID: "baseline-capture", Kind: NextActionRecommended, Label: "Start baseline capture", Reason: state.Detail, Argv: state.CaptureArgv},
			{ID: "baseline-wait", Kind: NextActionRecovery, Label: "Wait for baseline in Git Control Tower", Reason: "Only Git Control Tower owns wait, timeout, recovery, and parking semantics.", Argv: state.WaitArgv},
			{ID: "baseline-sync", Kind: NextActionRecovery, Label: "Synchronize baseline evidence", Reason: "Read durable GCT state after the producer reaches a terminal result.", Argv: state.SyncArgv},
		},
	}
}

func pathsToArgv(paths []string) []string {
	out := make([]string, 0, len(paths)*2)
	for _, path := range paths {
		out = append(out, "--path", path)
	}
	return out
}

func phasePrimaryAction(executionID, planID, phaseID string, ctx PhaseContext) NextAction {
	if ctx.CurrentPhase.Status == "todo" || ctx.CurrentPhase.Status == "" {
		return NextAction{
			ID:     "transition-active",
			Kind:   NextActionRecommended,
			Label:  "Mark current phase active",
			Reason: "This records that work is underway for the current phase.",
			Argv:   []string{"exec", "transition", executionID, phaseID, "--status", "active"},
		}
	}
	if validationIsRecentPass(ctx.LastValidation, ctx.HasValidation, ctx.Staleness) && ctx.FeedbackCheckpoint.Satisfied {
		return NextAction{
			ID:     "transition-done",
			Kind:   NextActionRecommended,
			Label:  "Mark phase done",
			Reason: "The last stored validation result passed and is fresh.",
			Argv:   []string{"exec", "transition", executionID, phaseID, "--status", "done"},
		}
	}
	if validationIsRecentPass(ctx.LastValidation, ctx.HasValidation, ctx.Staleness) {
		return NextAction{
			ID:     "review-phase-feedback",
			Kind:   NextActionRecommended,
			Label:  "Review phase feedback",
			Reason: ctx.FeedbackCheckpoint.Summary,
			Argv:   []string{"log", "note-add", executionID, "--phase", phaseID, "--title", NoFeedbackCheckpointTitle, "--detail", noFeedbackCheckpointDetail},
			BlockedBy: []string{
				"Capture any decisions/findings/bugs/records first, or run this no-feedback note command when there is nothing to capture.",
			},
		}
	}
	return NextAction{
		ID:     "start-validation-ticket",
		Kind:   NextActionRecommended,
		Label:  "Create phase validation ticket",
		Reason: "Create the producer-owned validation ticket, run its rendered Git Control Tower action and native wait, then synchronize terminal evidence before marking the phase done. " + waitDiscipline,
		Argv:   validationTicketArgv(executionID, planID, phaseID, ctx.ScopeGeneration, ctx.ValidationMembers),
		BlockedBy: []string{
			validationBlockerReason(ctx.LastValidation, ctx.HasValidation, ctx.Staleness),
		},
	}
}

func validationTicketArgv(executionID, planID, phaseID string, generation int, members []string) []string {
	argv := []string{"validate", "start", planID, "--phase", phaseID, "--execution", executionID}
	if generation > 0 {
		argv = append(argv, "--scope-generation", strconv.Itoa(generation))
	}
	if len(members) > 0 {
		argv = append(argv, "--members", strings.Join(members, ","))
	}
	return argv
}

func stepForTransition(e Execution) GuidedStep {
	if e.Complete {
		return GuidedStep{
			StepKind:     "final_dod_required",
			Title:        "Final Definition of Done Required",
			Summary:      "All phases are terminal, but normal completion requires a fresh full-inventory producer validation.",
			Instructions: []string{"Create the final validation ticket without member selectors, run its producer-owned action and native wait, synchronize it, then complete the execution.", waitDiscipline},
			NextActions: []NextAction{
				{
					ID:     "start-final-dod-ticket",
					Kind:   NextActionRecommended,
					Label:  "Create final full-inventory validation ticket",
					Reason: "A phase subset cannot certify plan completion.",
					Argv:   []string{"validate", "start", e.PlanID, "--execution", e.ID},
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

func onlyRecommendedExecutionAction(step GuidedStep) GuidedStep {
	return planmodel.OnlyRecommended(step)
}
