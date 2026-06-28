package authoring

import (
	"fmt"
	"strconv"
	"strings"
)

func stepForSession(sess Session) GuidedStep {
	sec, ok := sectionByKey(sess.Sections, sess.CurrentSectionKey)
	if !ok {
		return stepForReview(sess)
	}
	step := stepForSection(sess, sec)
	step.NextActions = append([]NextAction{
		{
			ID:     "author-next",
			Kind:   NextActionRecommended,
			Label:  "Open next authoring step",
			Reason: "The session has been created and the API has selected the first required section.",
			Argv:   []string{"author", "next", sess.ID},
		},
	}, step.NextActions...)
	return step
}

func stepForSection(sess Session, sec Section) GuidedStep {
	step := sectionBaseStep(sec.Key)
	placeholder := contentPlaceholderForSection(sec.Key)
	step.NextActions = []NextAction{
		{
			ID:                 "submit-" + string(sec.Key),
			Kind:               NextActionRecommended,
			Label:              "Submit " + sec.Label,
			Reason:             "This section is the current authoring input.",
			Argv:               []string{"author", "section-submit", sess.ID, "--section", string(sec.Key), "--content", placeholder},
			ContentPlaceholder: placeholder,
		},
	}
	if sec.Key == SectionReferences {
		step.NextActions = append(step.NextActions, NextAction{
			ID:     "autofill-" + string(sec.Key),
			Kind:   NextActionAlternative,
			Label:  "Autofill " + sec.Label,
			Reason: "This section can be populated by a composed dependency when available.",
			Argv:   []string{"author", "autofill", sess.ID, "--sources", string(sec.Key)},
		})
	}
	if sec.Key == SectionRegressionAnchor {
		// The anchor carries concrete recovery actions (autofill, capture a
		// baseline snapshot, or submit a structured fallback block) so a degraded
		// autofill never leaves the agent without an exact next command.
		step.NextActions = regressionAnchorRecoveryActions(sess)
		step.Examples = append(step.Examples, RegressionAnchorFallbackTemplate(sess))
	}
	if sec.Key == SectionRelevantContext {
		step.NextActions = []NextAction{
			{
				ID:                 "discover-context",
				Kind:               NextActionRecommended,
				Label:              "Discover context candidates",
				Reason:             "Relevant context should come from decomposed discovery concepts before manual acceptance.",
				Argv:               []string{"author", "context-discover", sess.ID, "--concepts", "<concept one>,<concept two>", "--complexity", "architectural"},
				ContentPlaceholder: "<concept one>,<concept two>",
			},
			{
				ID:     "submit-context",
				Kind:   NextActionAlternative,
				Label:  "Submit known context directly",
				Reason: "Use only when the setup item is already known and has a concrete reason.",
				Argv:   []string{"author", "context-submit", sess.ID, "--kind", "command", "--label", "<label>", "--reason", "<reason>", "--instruction", "<instruction>", "--command", "<command>", "--required"},
			},
		}
	}
	if sec.Key == SectionPhases {
		step.NextActions = []NextAction{
			{
				ID:                 "add-phase",
				Kind:               NextActionRecommended,
				Label:              "Add a structured phase",
				Reason:             "Plans need at least one phase before finalization.",
				Argv:               []string{"author", "phase-add", sess.ID, "--title", "<phase title>", "--intent", "<phase intent>"},
				ContentPlaceholder: "<phase title> / <phase intent>",
			},
			{
				ID:     "next-phase",
				Kind:   NextActionAlternative,
				Label:  "Inspect next incomplete phase",
				Reason: "Use this when phases already exist and one needs field-level completion.",
				Argv:   []string{"author", "phase-next", sess.ID},
			},
		}
	}
	return step
}

func sectionBaseStep(key SectionKey) GuidedStep {
	switch key {
	case SectionPurpose:
		return GuidedStep{
			StepKind:       "purpose",
			Title:          "Purpose",
			Summary:        "Explain why this plan exists and what pressure created it.",
			Instructions:   []string{"Write one concise paragraph.", "Name the outcome, not the implementation details.", "Include enough context for a later implementation agent to understand why this work matters."},
			RequiredInputs: []string{"purpose"},
			Examples:       []string{"Make plan-manager authoring reliable enough for local agents by replacing large markdown submissions with guided structured steps."},
			CommonMistakes: []string{"Listing tasks instead of explaining the need.", "Assuming the next agent remembers the originating conversation."},
		}
	case SectionProblemStatement:
		return GuidedStep{
			StepKind:       "problem_statement",
			Title:          "Problem / Need",
			Summary:        "State the concrete problem, gap, or need this plan closes.",
			Instructions:   []string{"Describe what is wrong or missing today, specifically.", "Name the pain a reviewer or user feels.", "Do not describe the solution here — that is the technical approach."},
			RequiredInputs: []string{"problem or need"},
			Examples:       []string{"The current plan model is too thin for a human reviewer or a small implementation agent to trust as the main artifact."},
			CommonMistakes: []string{"Describing the solution instead of the problem.", "Being so vague the need can't be verified as solved."},
		}
	case SectionTargetOutcome:
		return GuidedStep{
			StepKind:       "target_outcome",
			Title:          "Target Outcome",
			Summary:        "Describe the observable end state once this plan is done.",
			Instructions:   []string{"State what is observably true when the work is complete.", "Make it concrete enough to check.", "Keep it outcome-focused, not a task list."},
			RequiredInputs: []string{"target outcome"},
			Examples:       []string{"A human can run `plan-manager plans render <plan>` and judge plan quality without reading DB JSON."},
			CommonMistakes: []string{"Listing tasks instead of the end state.", "Restating the purpose."},
		}
	case SectionAssumptions:
		return GuidedStep{
			StepKind:       "assumptions",
			Title:          "Assumptions",
			Summary:        "Record the preconditions taken as given.",
			Instructions:   []string{"List environment, access, or prior-work assumptions, one per line.", "Only include assumptions that, if false, change the plan."},
			RequiredInputs: []string{"assumptions (optional)"},
			Examples:       []string{"The regression baseline is captured before any code change."},
			CommonMistakes: []string{"Listing requirements instead of assumptions."},
		}
	case SectionWorkPosture:
		return GuidedStep{
			StepKind:       "work_posture",
			Title:          "Work Posture (autofilled)",
			Summary:        "Work posture is derived automatically from scenario maturity (default greenfield). Review only — do not author the Greenfield/Brownfield block.",
			Instructions:   []string{"You do not write this section.", "The renderer injects the Greenfield (or Brownfield) block based on the associated scenario's maturity.", "Do not put compatibility-shim/legacy-wrapper language in constraints for a greenfield plan; validation will flag the conflict."},
			RequiredInputs: []string{"(none — autofilled)"},
			CommonMistakes: []string{"Hand-writing a Greenfield block.", "Authoring constraints that contradict the derived posture."},
		}
	case SectionTechnicalApproach:
		return GuidedStep{
			StepKind:       "technical_approach",
			Title:          "Technical Approach",
			Summary:        "Explain the chosen approach and the design rationale — why this way.",
			Instructions:   []string{"Describe the strategy at a design level, not a phase-by-phase list.", "Justify the key decisions and name the main alternatives ruled out.", "Keep it concise; the phases carry the step detail."},
			RequiredInputs: []string{"technical approach"},
			Examples:       []string{"Model-first contract change: expand the proto/Go plan model, then wire renderer, parser, wizard, CLI, and UI through that single contract."},
			CommonMistakes: []string{"Turning this into the phase list.", "Stating what without why."},
		}
	case SectionProhibitedApproaches:
		return GuidedStep{
			StepKind:       "prohibited_approaches",
			Title:          "Prohibited Approaches",
			Summary:        "Name approaches that are explicitly off-limits, only when genuinely relevant.",
			Instructions:   []string{"List approaches a reasonable agent might try but must not.", "Skip this section if nothing is genuinely off-limits."},
			RequiredInputs: []string{"prohibited approaches (optional)"},
			Examples:       []string{"Do not clone the legacy 13-section markdown format.", "Do not make markdown the source of truth."},
			CommonMistakes: []string{"Repeating non-goals.", "Listing obvious bad practice with no plan-specific value."},
		}
	case SectionValidationStrategy:
		return GuidedStep{
			StepKind:       "validation_strategy",
			Title:          "Validation Strategy",
			Summary:        "Describe how the plan proves it works: baseline approach, what evidence counts, and the final validation commands.",
			Instructions:   []string{"State the baseline/regression approach and the suites/commands that prove success.", "Distinguish per-phase validation from the final end-of-plan validation.", "Reference the exact commands a reviewer runs at the end."},
			RequiredInputs: []string{"validation strategy"},
			Examples:       []string{"Run focused Go/CLI/UI suites per phase; finish with `vrooli scenario test plan-manager` and a clean baseline diff against the captured anchor."},
			CommonMistakes: []string{"Saying 'tests pass' without naming them.", "Confusing the method (validation) with the outcome (definition of done)."},
		}
	case SectionScope:
		return GuidedStep{
			StepKind:       "scope",
			Title:          "Scope",
			Summary:        "Draw the boundary around the work.",
			Instructions:   []string{"State what is in scope and out of scope.", "Name affected scenarios, packages, commands, and surfaces when known.", "Keep future expansion separate from required work."},
			RequiredInputs: []string{"scope"},
			Examples:       []string{"In scope: authoring API/CLI/UI phase wizard. Out of scope: swarm-manager consumer inversion."},
			CommonMistakes: []string{"Using scope as another purpose paragraph.", "Omitting explicit non-goals for tempting adjacent work."},
		}
	case SectionReferences:
		return GuidedStep{
			StepKind:       "references",
			Title:          "References",
			Summary:        "Capture connected locations so staleness and validation can be computed.",
			Instructions:   []string{"Add one locator per line using [CODE:], [DOC:], or [REQ:].", "Include existing and proposed locations; mark future paths in prose when needed.", "If there truly are no connected code references, write NO_CODE_REFS: followed by the reason."},
			RequiredInputs: []string{"references or NO_CODE_REFS reason"},
			Examples:       []string{"[CODE: scenarios/plan-manager/api/internal/authoring/service.go]", "[REQ: PM-AUTHOR-002]", "NO_CODE_REFS: documentation-only operator decision."},
			CommonMistakes: []string{"Leaving references empty.", "Using plain paths without the machine-readable marker."},
		}
	case SectionRegressionAnchor:
		return GuidedStep{
			StepKind: "regression_anchor",
			Title:    "Regression Anchor",
			Summary:  "Record the before-state used to detect regressions.",
			Instructions: []string{
				"Prefer autofill so git-control-tower supplies the anchor.",
				"If autofill degraded (git-control-tower/baseline unavailable), capture a baseline snapshot yourself or submit the structured fallback block — never leave it blank.",
				"Do not claim validation passed here; this is only the before anchor.",
			},
			RequiredInputs: []string{"anchor strategy (autofilled, snapshot, or structured fallback block)"},
			Examples:       []string{"baseline name <slug>-baseline", "HEAD sha abc123 with allowlist scenarios/<scenario>/**"},
			CommonMistakes: []string{"Putting final test results in the anchor.", "Leaving the anchor blank because the dependency was down."},
		}
	case SectionRelevantContext:
		return GuidedStep{
			StepKind:       "relevant_context",
			Title:          "Relevant Context",
			Summary:        "Discover setup context and accept only items with a clear relevance reason.",
			Instructions:   []string{"Decompose the work into 2-5 search concepts.", "Run context-discover to generate candidate setup commands.", "Accept useful candidates globally or for a phase; reject noisy candidates with a reason."},
			RequiredInputs: []string{"context concepts or explicit NO_CONTEXT reason in phase scope"},
			Examples:       []string{"author context-discover <session> --concepts 'plan-manager execution resume,authoring context discovery' --complexity architectural"},
			CommonMistakes: []string{"Accepting raw discovery output without a reason.", "Putting phase-only setup in global context.", "Relying on the legacy required_reading section as the primary model."},
		}
	case SectionDefinitionOfDone:
		return GuidedStep{
			StepKind:       "definition_of_done",
			Title:          "Definition of Done",
			Summary:        "Define objective plan-level success.",
			Instructions:   []string{"Use pass/fail criteria that another agent can verify.", "Include validation expectations and adoption readiness.", "Avoid vague language like 'works' or 'complete'."},
			RequiredInputs: []string{"definition_of_done"},
			Examples:       []string{"Authoring API/CLI tests pass; plan-manager validate run returns PASS or documented UNKNOWN dependency gaps; scenario requirements validate green."},
			CommonMistakes: []string{"Restating phase steps.", "Using subjective acceptance criteria."},
		}
	case SectionPhases:
		return GuidedStep{
			StepKind:       "phase_outline",
			Title:          "Phase Outline",
			Summary:        "Create the phase list, then fill each phase through phase-native commands.",
			Instructions:   []string{"Use author phase-add for each phase instead of submitting one large markdown blob.", "Keep phases sequential and handoff-sized.", "Every phase needs intent, references or a no-code reason, and objective acceptance."},
			RequiredInputs: []string{"phase list"},
			Examples:       []string{"author phase-add <session> --title 'Authoring contract' --intent 'Add phase-native RPC and CLI surface.'"},
			CommonMistakes: []string{"Making one giant implementation phase.", "Skipping per-phase references."},
		}
	default:
		return GuidedStep{
			StepKind:       string(key),
			Title:          "Authoring Step",
			Summary:        "Fill this section with concise, implementation-relevant information.",
			Instructions:   []string{"Keep the content specific to this section.", "Use current command/reference markers when mentioning CLI or code locations."},
			RequiredInputs: []string{string(key)},
			CommonMistakes: []string{"Adding broad reminders that belong in phase guidance."},
		}
	}
}

// regressionAnchorRecoveryActions are the concrete recovery commands surfaced
// when the anchor still needs input (including after a degraded autofill).
func regressionAnchorRecoveryActions(sess Session) []NextAction {
	name := anchorBaselineName(sess)
	reason := firstNonEmpty(sess.Title, "regression baseline before implementation")
	return []NextAction{
		{
			ID:     "autofill-anchor",
			Kind:   NextActionRecommended,
			Label:  "Autofill the regression anchor",
			Reason: "Let git-control-tower supply the baseline anchor mechanically.",
			Argv:   []string{"author", "autofill", sess.ID, "--sources", "regression_anchor"},
		},
		{
			ID:                 "capture-baseline-snapshot",
			Kind:               NextActionRecovery,
			Label:              "Capture a baseline snapshot, then re-run autofill",
			Reason:             "Run when autofill degraded because the baseline snapshot/manifest intent was missing.",
			Argv:               []string{"git-control-tower", "baseline", "snapshot", "--scenario", "<scenario>", "--name", name, "--reason", reason},
			ContentPlaceholder: "<scenario>",
		},
		{
			ID:                 "submit-fallback-anchor",
			Kind:               NextActionRecovery,
			Label:              "Submit the structured fallback anchor block",
			Reason:             "Use when git-control-tower cannot run at all; record an honest SHA/allowlist strategy instead of leaving the anchor blank.",
			Argv:               []string{"author", "section-submit", sess.ID, "--section", string(SectionRegressionAnchor), "--content", RegressionAnchorFallbackTemplate(sess)},
			ContentPlaceholder: RegressionAnchorFallbackTemplate(sess),
		},
	}
}

// RegressionAnchorFallbackTemplate returns the structured fallback anchor block
// an agent can submit verbatim when autofill degrades. Its field keys match the
// plans-domain anchor parser (Strategy / Scenario baseline / Baseline name /
// HEAD sha / Allowlist / Diff command), so a fallback anchor parses into typed
// fields rather than degrading to legacy prose.
func RegressionAnchorFallbackTemplate(sess Session) string {
	name := anchorBaselineName(sess)
	return strings.Join([]string{
		"Strategy: scenario-baseline",
		"Scenario baseline: <scenario>",
		"Baseline name: " + name,
		"HEAD sha: <output of `git rev-parse HEAD`>",
		"Allowlist: scenarios/<scenario>/**",
		"`git-control-tower baseline diff --scenario <scenario> --name " + name + " --branch <branch> --wait`",
	}, "\n")
}

func anchorBaselineName(sess Session) string {
	base := firstNonEmpty(sess.Slug, sess.Title, "plan")
	base = strings.ToLower(strings.TrimSpace(base))
	base = strings.ReplaceAll(base, " ", "-")
	return base + "-baseline"
}

// stepForGlobalContextCheckpoint is the explicit plan-wide relevant-context
// checkpoint the continue loop surfaces before phase work. The agent resolves it
// by discovering+accepting context, submitting a known item, or recording an
// explicit NO_CONTEXT skip reason — the loop never silently bypasses it.
func stepForGlobalContextCheckpoint(sess Session) GuidedStep {
	return GuidedStep{
		StepKind:       "global_relevant_context",
		Title:          "Global Relevant Context",
		Summary:        "Decide the plan-wide setup context a fresh or resumed agent should load before any phase. This checkpoint cannot be skipped silently.",
		Instructions:   []string{"Discover context candidates and accept the relevant ones, or submit a known global setup item.", "If this plan genuinely needs no plan-wide setup context, record an explicit NO_CONTEXT reason.", "Phase-specific setup belongs on the phase, not here."},
		RequiredInputs: []string{"global relevant context item(s) or an explicit NO_CONTEXT reason"},
		Examples:       []string{"author context-discover " + sess.ID + " --concepts 'plan-manager execution resume' --complexity architectural", "author section-submit " + sess.ID + " --section relevant_context --content 'NO_CONTEXT: single-file docs change needs no plan-wide setup.'"},
		CommonMistakes: []string{"Skipping plan-wide context by leaving it empty.", "Putting phase-only setup in global context."},
		NextActions: []NextAction{
			{
				ID:                 "discover-context",
				Kind:               NextActionRecommended,
				Label:              "Discover global context candidates",
				Reason:             "Generate candidate plan-wide setup commands to accept or reject.",
				Argv:               []string{"author", "context-discover", sess.ID, "--concepts", "<concept one>,<concept two>", "--complexity", "architectural"},
				ContentPlaceholder: "<concept one>,<concept two>",
			},
			{
				ID:                 "submit-global-context",
				Kind:               NextActionAlternative,
				Label:              "Submit a known global context item",
				Reason:             "Use when a plan-wide setup item is already known.",
				Argv:               []string{"author", "context-submit", sess.ID, "--kind", "command", "--label", "<label>", "--reason", "<reason>", "--instruction", "<instruction>", "--command", "<command>", "--required"},
				ContentPlaceholder: "<label> / <reason> / <command>",
			},
			{
				ID:                 "skip-global-context",
				Kind:               NextActionAlternative,
				Label:              "Record no global context (with reason)",
				Reason:             "Use only when the plan genuinely needs no plan-wide setup context.",
				Argv:               []string{"author", "section-submit", sess.ID, "--section", string(SectionRelevantContext), "--content", "NO_CONTEXT: <reason>"},
				ContentPlaceholder: "NO_CONTEXT: <reason>",
			},
		},
	}
}

func stepForContextDiscovery(sess Session) GuidedStep {
	return GuidedStep{
		StepKind:       "context_discovery",
		Title:          "Context Discovery",
		Summary:        "Review discovered context candidates and accept or reject each one.",
		Instructions:   []string{"Accept only candidates that materially reduce implementation ambiguity.", "Assign phase-specific setup to the phase where it is needed.", "Reject noisy or duplicate candidates with a short reason."},
		RequiredInputs: []string{"candidate decision"},
		Examples:       []string{"author context-accept " + sess.ID + " <candidate-id>", "author context-reject " + sess.ID + " <candidate-id> --reason 'duplicate of global setup'"},
		CommonMistakes: []string{"Accepting every candidate.", "Leaving degraded candidates untriaged.", "Accepting a command candidate without inspecting whether it is current."},
		NextActions: []NextAction{
			{
				ID:     "list-context",
				Kind:   NextActionRecommended,
				Label:  "List accepted context",
				Reason: "Use accepted context as the reviewable setup list before finalizing.",
				Argv:   []string{"author", "context-list", sess.ID},
			},
		},
	}
}

func stepForPhase(sess Session, phase PhaseDraft) GuidedStep {
	field := nextMissingPhaseField(phase)
	switch field {
	case PhaseFieldTitle:
		return phaseStep(sess, phase, field, "phase_title", "Phase Title", "Give the phase a short action-oriented name.", []string{"title"}, []string{"Authoring contract", "Validation wiring"}, "<phase title>")
	case PhaseFieldIntent:
		return phaseStep(sess, phase, field, "phase_intent", "Phase Intent", "State what this phase accomplishes.", []string{"intent"}, []string{"Add phase-native RPCs and service operations without changing execution semantics."}, "<phase intent>")
	case PhaseFieldReferences:
		step := phaseStep(sess, phase, field, "phase_references", "Phase References", "Attach connected code/docs/requirements for this phase.", []string{"references or no_code_refs_reason"}, []string{"[CODE: scenarios/plan-manager/api/internal/authoring/service.go]", "NO_CODE_REFS: operator-only review phase."}, "[CODE: path/to/file.go]")
		step.NextActions = append(step.NextActions, NextAction{
			ID:                 "phase-no-code-reason",
			Kind:               NextActionAlternative,
			Label:              "Record no-code reference reason",
			Reason:             "Use this only when the phase genuinely has no connected code, docs, or requirements.",
			Argv:               []string{"author", "phase-submit", sess.ID, phase.ID, "--field", string(PhaseFieldNoCodeRefsReason), "--content", "NO_CODE_REFS: <reason>"},
			ContentPlaceholder: "NO_CODE_REFS: <reason>",
		})
		return step
	case PhaseFieldSteps:
		return phaseStep(sess, phase, field, "phase_steps", "Phase Ordered Steps", "List the concrete implementation steps an agent follows, one per line, in order.", []string{"ordered steps"}, []string{"Add proto fields\nRegenerate proto\nWire the converters\nRun go test ./internal/planproto"}, "<step one>\n<step two>")
	case PhaseFieldValidation:
		return phaseStep(sess, phase, field, "phase_validation", "Phase Validation", "State the METHOD of checking this phase — the exact commands/checks you run. This is distinct from acceptance (the outcome gate) and must not be identical to it.", []string{"validation method"}, []string{"go test ./internal/planproto ./internal/plans"}, "<commands/checks that verify this phase>")
	case PhaseFieldAcceptance:
		return phaseStep(sess, phase, field, "phase_acceptance", "Phase Acceptance", "Define the objective pass/fail OUTCOME for this phase (not the commands — that is validation).", []string{"acceptance"}, []string{"Generated proto compiles and the converter round-trips every new field."}, "<phase acceptance criteria>")
	case PhaseFieldRelevantContext:
		step := phaseStep(sess, phase, field, "phase_relevant_context", "Phase Relevant Context", "Attach phase-scoped setup context or record why no setup context exists.", []string{"relevant_context or NO_CONTEXT reason"}, []string{"prompt-manager skill read api-steer", "NO_CONTEXT: docs-only review phase has no extra setup."}, "NO_CONTEXT: <reason>")
		step.NextActions = append(step.NextActions, NextAction{
			ID:                 "phase-context-submit",
			Kind:               NextActionAlternative,
			Label:              "Submit phase context item",
			Reason:             "Use when the phase has a concrete setup item with a relevance reason.",
			Argv:               []string{"author", "context-submit", sess.ID, "--phase", phase.ID, "--kind", "doc", "--label", "<label>", "--reason", "<reason>", "--target", "<target>", "--required"},
			ContentPlaceholder: "<label> / <reason> / <target>",
		})
		return step
	default:
		return GuidedStep{
			StepKind:       "phase_review",
			Title:          "Phase Review",
			Summary:        "This phase has the required fields. Add relevant context or reminders if useful.",
			Instructions:   []string{"Use context-submit or context-accept for phase-specific setup.", "Use reminders for constraints the implementation agent must see just-in-time.", "Move to the next incomplete phase when ready."},
			Examples:       []string{"author phase-next " + sess.ID},
			CommonMistakes: []string{"Repeating whole-plan context in every phase.", "Adding generic reminders with no phase-specific value."},
			NextActions: []NextAction{
				{
					ID:     "next-phase",
					Kind:   NextActionRecommended,
					Label:  "Find next incomplete phase",
					Reason: "This phase is complete enough for review.",
					Argv:   []string{"author", "phase-next", sess.ID},
				},
			},
		}
	}
}

func phaseStep(sess Session, phase PhaseDraft, field PhaseField, kind, title, summary string, required, examples []string, placeholder string) GuidedStep {
	return GuidedStep{
		StepKind:       kind,
		Title:          title,
		Summary:        summary,
		Instructions:   []string{"Submit only this field's content.", "Keep it concrete enough for a later agent to act on without reading the whole conversation."},
		RequiredInputs: required,
		Examples:       examples,
		CommonMistakes: []string{"Bundling multiple phase fields into one response.", "Using vague prose that cannot be validated."},
		NextActions: []NextAction{
			{
				ID:                 "submit-phase-" + string(field),
				Kind:               NextActionRecommended,
				Label:              "Submit phase " + string(field),
				Reason:             "This is the next missing field for phase " + strconv.Itoa(phase.Order) + ".",
				Argv:               []string{"author", "phase-submit", sess.ID, phase.ID, "--field", string(field), "--content", placeholder},
				ContentPlaceholder: placeholder,
			},
		},
	}
}

func nextMissingPhaseField(phase PhaseDraft) PhaseField {
	switch {
	case phase.Title == "":
		return PhaseFieldTitle
	case phase.Intent == "":
		return PhaseFieldIntent
	case len(phase.References) == 0 && phase.NoCodeRefsReason == "":
		return PhaseFieldReferences
	case len(phase.Steps) == 0:
		return PhaseFieldSteps
	case strings.TrimSpace(phase.Validation) == "":
		return PhaseFieldValidation
	case phase.Acceptance == "":
		return PhaseFieldAcceptance
	case !hasPhaseContextOrNoContextReason(phase):
		return PhaseFieldRelevantContext
	default:
		return ""
	}
}

func stepForReview(sess Session) GuidedStep {
	return GuidedStep{
		StepKind:       "final_review",
		Title:          "Final Review",
		Summary:        "All mandatory authoring inputs are present. Validate before finalizing.",
		Instructions:   []string{"Run author validate.", "Preview the rendered markdown to review the plan as a human would, before finalizing.", "Resolve every violation instead of finalizing around it."},
		Examples:       []string{"author validate " + sess.ID, "author preview " + sess.ID, "author finalize " + sess.ID},
		CommonMistakes: []string{"Finalizing before phase steps, validation, and acceptance are objective.", "Skipping the render preview and shipping an unreviewed plan."},
		NextActions: []NextAction{
			{
				ID:     "validate-session",
				Kind:   NextActionRecommended,
				Label:  "Validate structure",
				Reason: "The session appears complete; validation is the gate before finalization.",
				Argv:   []string{"author", "validate", sess.ID},
			},
			{
				ID:     "preview-plan",
				Kind:   NextActionAlternative,
				Label:  "Preview rendered plan",
				Reason: "Review the rendered markdown review artifact before finalizing.",
				Argv:   []string{"author", "preview", sess.ID},
			},
			{
				ID:     "finalize-session",
				Kind:   NextActionAlternative,
				Label:  "Finalize plan",
				Reason: "Use after validation returns valid and the preview looks right.",
				Argv:   []string{"author", "finalize", sess.ID},
			},
		},
	}
}

func stepForValidation(sess Session, valid bool, violations []StructureViolation) GuidedStep {
	if valid {
		return GuidedStep{
			StepKind:     "validation_passed",
			Title:        "Validation Passed",
			Summary:      "The authoring session is structurally valid and can be finalized.",
			Instructions: []string{"Finalize the plan, then inspect the persisted record."},
			NextActions: []NextAction{
				{
					ID:     "finalize-session",
					Kind:   NextActionRecommended,
					Label:  "Finalize plan",
					Reason: "The structure gate passed.",
					Argv:   []string{"author", "finalize", sess.ID},
				},
			},
		}
	}
	key := sess.CurrentSectionKey
	if len(violations) > 0 && violations[0].SectionKey != "" {
		key = violations[0].SectionKey
	}
	sec, ok := sectionByKey(sess.Sections, key)
	if !ok {
		sec = Section{Key: key, Label: string(key), Mandatory: true}
	}
	step := stepForSection(sess, sec)
	step.StepKind = "validation_recovery"
	step.Title = "Validation Recovery"
	step.Summary = "The structure gate found an issue that must be fixed before finalization."
	for i := range step.NextActions {
		step.NextActions[i].Kind = NextActionRecovery
		step.NextActions[i].Reason = "Fix validation violation: " + firstViolationMessage(violations)
	}
	return step
}

func stepForFinalizedPlan(sess Session, planID, slug string) GuidedStep {
	return GuidedStep{
		StepKind:     "finalized",
		Title:        "Plan Finalized",
		Summary:      "The structured plan is persisted and ready for review or execution.",
		Instructions: []string{"Inspect the persisted plan before execution.", "Start execution when the plan is ready to implement."},
		NextActions: []NextAction{
			{
				ID:     "view-plan",
				Kind:   NextActionRecommended,
				Label:  "View plan",
				Reason: "Inspect the persisted structured record.",
				Argv:   []string{"plans", "get", slug},
			},
			{
				ID:     "start-execution",
				Kind:   NextActionAlternative,
				Label:  "Start execution",
				Reason: "Begin guided implementation for this plan.",
				Argv:   []string{"exec", "start", planID},
			},
			{
				ID:     "view-session-plan",
				Kind:   NextActionOptional,
				Label:  "View by session slug",
				Reason: "Use the authoring slug if it differs from the persisted id.",
				Argv:   []string{"plans", "get", sess.Slug},
			},
		},
	}
}

func contentPlaceholderForSection(key SectionKey) string {
	switch key {
	case SectionPurpose:
		return "<one concise purpose paragraph>"
	case SectionProblemStatement:
		return "<the concrete problem or need this plan closes>"
	case SectionTargetOutcome:
		return "<the observable end state once done>"
	case SectionAssumptions:
		return "<preconditions taken as given>"
	case SectionTechnicalApproach:
		return "<chosen approach and why>"
	case SectionProhibitedApproaches:
		return "<approaches that are off-limits>"
	case SectionValidationStrategy:
		return "<how the plan proves it works + final validation commands>"
	case SectionScope:
		return "<in scope / out of scope>"
	case SectionConstraints:
		return "<hard constraints>"
	case SectionNonGoals:
		return "<explicit non-goals>"
	case SectionReferences:
		return "[CODE: path/to/file.go]"
	case SectionRegressionAnchor:
		return "<regression anchor strategy and commands>"
	case SectionRequiredReading:
		return "<legacy required-reading migration input>"
	case SectionDefinitionOfDone:
		return "<objective definition of done>"
	case SectionPhases:
		return "<phase outline>"
	default:
		return fmt.Sprintf("<%s content>", key)
	}
}

func sectionByKey(sections []Section, key SectionKey) (Section, bool) {
	for _, sec := range sections {
		if sec.Key == key {
			return sec, true
		}
	}
	return Section{}, false
}

func firstViolationMessage(violations []StructureViolation) string {
	if len(violations) == 0 {
		return "unknown validation issue"
	}
	return violations[0].Message
}

func onlyRecommendedAction(step GuidedStep) GuidedStep {
	for _, action := range step.NextActions {
		if action.Kind == NextActionRecommended {
			step.NextActions = []NextAction{action}
			return step
		}
	}
	if len(step.NextActions) > 1 {
		step.NextActions = step.NextActions[:1]
	}
	return step
}
