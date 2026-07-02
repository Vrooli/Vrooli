package authoring

import (
	"fmt"
	"strings"

	planmodel "plan-manager/internal/planmodel"
)

func sessionHandle(sess Session) string {
	if strings.TrimSpace(sess.Slug) != "" {
		return sess.Slug
	}
	return sess.ID
}

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
			Argv:   []string{"author", "next", sessionHandle(sess)},
		},
	}, step.NextActions...)
	step.NextActions = append(step.NextActions, NextAction{
		ID:     "plans-import",
		Kind:   NextActionOptional,
		Label:  "Import an already-written plan document instead",
		Reason: "If the plan already exists as a markdown document, `plans import` ingests it wholesale — no need to re-author it section by section.",
		Argv:   []string{"plans", "import", "--file", "<plan.md>"},
	})
	return step
}

func stepForSection(sess Session, sec Section) GuidedStep {
	step := sectionBaseStep(sec.Key)
	step.Checklist = sessionChecklist(sess)
	placeholder := contentPlaceholderForSection(sec.Key)
	step.NextActions = []NextAction{
		{
			ID:                 "submit-" + string(sec.Key),
			Kind:               NextActionRecommended,
			Label:              "Submit " + sec.Label,
			Reason:             "This section is the current authoring input.",
			Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(sec.Key), "--content", placeholder},
			ContentPlaceholder: placeholder,
		},
	}
	if sec.Key == SectionReferences {
		// References carries concrete recovery actions (autofill, manual submit, or
		// an explicit NO_CODE_REFS fallback) so a degraded references autofill never
		// leaves the agent without an exact next command — it mirrors the
		// regression-anchor recovery posture.
		step.NextActions = referencesRecoveryActions(sess)
	}
	if sec.Key == SectionAcceptanceBoundary {
		step.NextActions = changeBoundaryActions(sess)
	}
	if sec.Key == SectionRegressionAnchor {
		// The anchor carries concrete actions: derive the typed intent
		// mechanically (no snapshot — the executor captures that at execution
		// start), or confirm/adjust the derived intent block by hand.
		step.NextActions = regressionAnchorRecoveryActions(sess)
		step.Examples = append(step.Examples, RegressionAnchorIntentTemplate(sess.Title, sess.Slug, sessionBoundary(sess)))
	}
	if sec.Key == SectionRelevantContext {
		step.NextActions = []NextAction{
			{
				ID:                 "discover-context",
				Kind:               NextActionRecommended,
				Label:              "Discover context candidates",
				Reason:             "Relevant context should come from decomposed discovery concepts before manual acceptance.",
				Argv:               []string{"author", "context-discover", sessionHandle(sess), "--concepts", "<concept one>,<concept two>", "--complexity", "architectural"},
				ContentPlaceholder: "<concept one>,<concept two>",
			},
			{
				ID:     "submit-context",
				Kind:   NextActionAlternative,
				Label:  "Submit known context directly",
				Reason: "Use only when the setup item is already known and has a concrete reason.",
				Argv:   []string{"author", "context-submit", sessionHandle(sess), "--kind", "command", "--label", "<label>", "--reason", "<reason>", "--instruction", "<instruction>", "--command", "<command>", "--required"},
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
				Argv:               []string{"author", "phase-add", sessionHandle(sess), "--title", "<phase title>", "--intent", "<phase intent>"},
				ContentPlaceholder: "<phase title> / <phase intent>",
			},
			{
				ID:     "next-phase",
				Kind:   NextActionAlternative,
				Label:  "Inspect next incomplete phase",
				Reason: "Use this when phases already exist and one needs field-level completion.",
				Argv:   []string{"author", "phase-next", sessionHandle(sess)},
			},
		}
	}
	return step
}

func sectionBaseStep(key SectionKey) GuidedStep {
	if spec, ok := sectionSpecByKey(key); ok {
		return GuidedStep{
			StepKind:       spec.StepKind,
			Title:          firstNonEmpty(spec.Title, spec.Label),
			Summary:        spec.Summary,
			Instructions:   append([]string(nil), spec.Instructions...),
			RequiredInputs: append([]string(nil), spec.RequiredInputs...),
			Examples:       append([]string(nil), spec.Examples...),
			CommonMistakes: append([]string(nil), spec.CommonMistakes...),
		}
	}
	return GuidedStep{
		StepKind:       string(key),
		Title:          "Authoring Step",
		Summary:        "Fill this section with concise, implementation-relevant information.",
		Instructions:   []string{"Keep the content specific to this section.", "Use current command/reference markers when mentioning CLI or code locations."},
		RequiredInputs: []string{string(key)},
		CommonMistakes: []string{"Adding broad reminders that belong in phase guidance."},
	}
}

// referencesRecoveryActions are the concrete actions surfaced for the references
// section. Suggesting from search-hub is recommended (the Answer projection
// discovers connected locations); manual submission is the alternative when the
// author already knows the touched files; and an explicit NO_CODE_REFS fallback
// ensures an empty suggestion result or a genuinely doc-only plan still has an
// exact next command instead of a dead end.
func referencesRecoveryActions(sess Session) []NextAction {
	return []NextAction{
		{
			ID:     "suggest-references",
			Kind:   NextActionRecommended,
			Label:  "Suggest references",
			Reason: "Discover connected [CODE:]/[DOC:]/[REQ:] locators from search-hub, then apply the curated batch.",
			Argv:   []string{"author", "suggest-references", sessionHandle(sess)},
		},
		{
			ID:                 "submit-references",
			Kind:               NextActionAlternative,
			Label:              "Submit references manually",
			Reason:             "List the connected [CODE:]/[DOC:]/[REQ:] locators you are touching, one per line.",
			Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionReferences), "--content", "[CODE: path/to/file.go]"},
			ContentPlaceholder: "[CODE: path/to/file.go]",
		},
		{
			ID:                 "submit-no-code-refs",
			Kind:               NextActionRecovery,
			Label:              "Record NO_CODE_REFS fallback",
			Reason:             "Use when suggestions found no targets and there are genuinely no connected code/doc/req references — record an honest reason instead of leaving references blank.",
			Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionReferences), "--content", "NO_CODE_REFS: <reason there are no connected references>"},
			ContentPlaceholder: "NO_CODE_REFS: <reason there are no connected references>",
		},
	}
}

// stepForReferenceCandidates is the guided step after SuggestReferences (mirrors
// stepForContextDiscovery): review the curated locator batch and apply it. Raw
// suggestions never satisfy the references gate — only accepted locators
// (written into the references section) do.
func stepForReferenceCandidates(sess Session) GuidedStep {
	batch, _ := latestPendingReferenceBatch(sess)
	summary := "No reference candidates are pending. Record connected references manually, or record NO_CODE_REFS when none exist."
	requiredInputs := []string{"references or NO_CODE_REFS reason"}
	examples := []string{"author section-submit " + sessionHandle(sess) + " --section references --content '[CODE: path/to/file.go]'", "author section-submit " + sessionHandle(sess) + " --section references --content 'NO_CODE_REFS: <reason>'"}
	actions := []NextAction{{
		ID:                 "submit-references",
		Kind:               NextActionRecommended,
		Label:              "Submit references manually",
		Reason:             "List the connected [CODE:]/[DOC:]/[REQ:] locators you are touching, one per line.",
		Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionReferences), "--content", "[CODE: path/to/file.go]"},
		ContentPlaceholder: "[CODE: path/to/file.go]",
	}}
	if batch.ID != "" {
		summary = "Review the curated search-hub reference suggestions and apply the batch. Suggestions do not enter the plan until accepted."
		requiredInputs = []string{"reference-apply decision or NO_CODE_REFS reason"}
		examples = []string{"author reference-apply " + sessionHandle(sess) + " --take r1 --note 'reviewed references'", "author reference-reject " + sessionHandle(sess) + " r1 --reason 'unrelated subsystem'"}
		actions = []NextAction{
			{
				ID:     "list-reference-candidates",
				Kind:   NextActionRecommended,
				Label:  "List reference candidates",
				Reason: "Review the suggested locators before applying the batch.",
				Argv:   []string{"author", "reference-list", sessionHandle(sess)},
			},
			{
				ID:     "apply-reference-candidates",
				Kind:   NextActionRecommended,
				Label:  "Apply reference batch",
				Reason: "Accept selected reference handles and sweep the remaining shortlist in one call.",
				Argv:   []string{"author", "reference-apply", sessionHandle(sess), "--batch", batch.ID, "--take", "r1", "--note", "reviewed shortlist"},
			},
		}
	}
	actions = append(actions, NextAction{
		ID:                 "submit-no-code-refs",
		Kind:               NextActionRecovery,
		Label:              "Record NO_CODE_REFS fallback",
		Reason:             "Use when no suggestion fits and there are genuinely no connected references.",
		Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionReferences), "--content", "NO_CODE_REFS: <reason there are no connected references>"},
		ContentPlaceholder: "NO_CODE_REFS: <reason there are no connected references>",
	})
	return GuidedStep{
		StepKind:       "reference_candidates",
		Title:          "Reference Candidates",
		Summary:        summary,
		Instructions:   []string{"Take only locators the plan genuinely depends on or changes.", "Use a drop reason only for high-confidence suggestions that should not land.", "If no suggestion fits and there are no connected references, record a NO_CODE_REFS reason instead."},
		RequiredInputs: requiredInputs,
		Examples:       examples,
		CommonMistakes: []string{"Accepting every suggestion without judgment.", "Treating a raw suggestion as if it already satisfied the references gate."},
		NextActions:    actions,
	}
}

// regressionAnchorRecoveryActions are the concrete actions surfaced for the
// anchor section. Deriving the typed intent mechanically is recommended; the
// alternative is to confirm/adjust the derived intent block by hand. There is no
// "capture a baseline snapshot" action here — that moved to execution start,
// where the "before" is actually true.
// sessionBoundary parses the session's acceptance-boundary section into a typed
// ChangeBoundary so anchor derivation and examples reflect the authored boundary.
func sessionBoundary(sess Session) planmodel.ChangeBoundary {
	return planmodel.ParseBoundarySection(contentOf(sess.Sections, SectionAcceptanceBoundary))
}

// changeBoundaryActions are the concrete actions surfaced for the change-boundary
// section: submit acceptance_allow paths (recommended), or record an
// OPERATOR_ONLY reason for genuinely no-code/operator-only work.
func changeBoundaryActions(sess Session) []NextAction {
	return []NextAction{
		{
			ID:                 "submit-boundary",
			Kind:               NextActionRecommended,
			Label:              "Submit change boundary",
			Reason:             "Declare the repo paths this plan may change (acceptance_allow). Scenario identity and the regression anchor derive from these globs.",
			Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionAcceptanceBoundary), "--content", "acceptance_allow:\n- scenarios/<scenario>/**\nacceptance_deny:\n- (optional forbidden globs)"},
			ContentPlaceholder: "acceptance_allow:\n- scenarios/<scenario>/**\n- packages/<shared>/**",
		},
		{
			ID:                 "submit-operator-only-boundary",
			Kind:               NextActionRecovery,
			Label:              "Record OPERATOR_ONLY boundary",
			Reason:             "Use only when the plan is genuinely operator-only / no-code and has no editable repo paths.",
			Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionAcceptanceBoundary), "--content", "OPERATOR_ONLY: <reason there are no editable repo paths>"},
			ContentPlaceholder: "OPERATOR_ONLY: <reason there are no editable repo paths>",
		},
	}
}

func regressionAnchorRecoveryActions(sess Session) []NextAction {
	template := RegressionAnchorIntentTemplate(sess.Title, sess.Slug, sessionBoundary(sess))
	return []NextAction{
		{
			ID:     "autofill-anchor",
			Kind:   NextActionRecommended,
			Label:  "Derive the regression anchor intent",
			Reason: "Fill the typed anchor intent (strategy, scenario, allowlist, diff command) mechanically — no snapshot, never stale.",
			Argv:   []string{"author", "autofill", sessionHandle(sess), "--sources", "regression_anchor"},
		},
		{
			ID:                 "submit-anchor-intent",
			Kind:               NextActionAlternative,
			Label:              "Confirm/adjust the anchor intent block",
			Reason:             "Submit the typed intent yourself, replacing the <scenario> placeholder with the real target scenario.",
			Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionRegressionAnchor), "--content", template},
			ContentPlaceholder: template,
		},
	}
}

// RegressionAnchorIntentTemplate returns the boundary-native regression-anchor
// intent block — the primary authoring output for the anchor. Its field keys
// match the plans-domain anchor parser (Strategy / Baseline name / HEAD sha +
// backticked commands), so it parses into typed fields. Affected scenarios and
// the tiered baseline/diff commands are DERIVED from the change boundary — no
// hand-authored `<scenario>` placeholder. It records INTENT only (the HEAD sha is
// captured fresh at execution start).
func RegressionAnchorIntentTemplate(title, slug string, boundary planmodel.ChangeBoundary) string {
	name := anchorBaselineName(title, slug)
	lines := []string{
		"Strategy: " + planmodel.AnchorStrategyChangeBoundary,
		"Baseline name: " + name,
		"HEAD sha: <captured at execution start>",
	}
	commands, _ := planmodel.BoundaryAnchorCommands(boundary, name, "")
	for _, c := range commands {
		lines = append(lines, "`"+c+"`")
	}
	if len(commands) == 0 {
		lines = append(lines,
			"- _submit the change boundary (acceptance_allow) first; the baseline/diff commands derive from it_")
	}
	return strings.Join(lines, "\n")
}

func anchorBaselineName(title, slug string) string {
	base := firstNonEmpty(slug, title, "plan")
	base = strings.ToLower(strings.TrimSpace(base))
	base = strings.ReplaceAll(base, " ", "-")
	return base + "-baseline"
}

// stepForGlobalContextCheckpoint is the explicit plan-wide relevant-context
// checkpoint the continue loop surfaces before phase work. The agent resolves it
// by discovering+accepting context, submitting a known item, and explicitly
// deciding skill setup, or by recording an explicit NO_CONTEXT skip reason.
func stepForGlobalContextCheckpoint(sess Session) GuidedStep {
	return GuidedStep{
		StepKind: "global_relevant_context",
		Title:    "Global Relevant Context",
		Summary:  "Decide the plan-wide setup context and skill setup a fresh or resumed agent should load before any phase. The skill checkpoint demands evidence of a sweep: discovery ran for your concepts and the curated batch was applied — or an honest NO_SKILL_CONTEXT skip.",
		Instructions: []string{
			"Decompose the work into 2-5 discovery concepts using four lenses: domain area, technology/stack, problem type, and scenario surface (e.g. a mic bug in a React UI → 'web-console voice', 'react state machines', 'debugging', 'ui hooks').",
			"Phrase each concept as activity + surface ('refactor duplicated CLI rendering'), not a bare noun.",
			"Pick the complexity that matches the work: minor = bug fix/small tweak; moderate = new feature/refactor; major = multi-file feature/new endpoint; architectural = cross-scenario/new system design.",
			"Run context-discover with those concepts — the probes execute server-side and return a curated shortlist plus degraded probe notes.",
			"Review the proposal, then run context-apply once: take only items whose concrete improvement you can articulate; sweep the rest with an optional note. High-confidence drops need a reason.",
			"If discovery finds no applicable internal skill, record NO_SKILL_CONTEXT with the reason instead of accepting filler.",
			"If this plan genuinely needs no plan-wide setup context at all, record an explicit NO_CONTEXT reason.",
			"Phase-specific setup belongs on the phase, not here.",
		},
		RequiredInputs: []string{"2-5 decomposed discovery concepts + complexity", "an applied context discovery batch", "or explicit NO_SKILL_CONTEXT / NO_CONTEXT reason"},
		Checklist:      sessionChecklist(sess),
		Examples:       []string{"author context-discover " + sessionHandle(sess) + " --concepts 'plan-manager execution resume' --complexity architectural", "author context-apply " + sessionHandle(sess) + " --take c1 --note 'reviewed shortlist'", "author context-submit " + sessionHandle(sess) + " --kind skill --label implementation-plan-authoring --target implementation-plan-authoring --reason 'authoring standards shape the plan' --instruction 'Load before implementation planning' --required", "author section-submit " + sessionHandle(sess) + " --section relevant_context --content 'NO_SKILL_CONTEXT: no internal skill applies beyond accepted docs/search setup.'"},
		CommonMistakes: []string{"Skipping plan-wide context by leaving it empty.", "Adding docs/search context but never deciding skill setup.", "Putting phase-only setup in global context."},
		NextActions: []NextAction{
			{
				ID:                 "discover-context",
				Kind:               NextActionRecommended,
				Label:              "Discover global context candidates",
				Reason:             "Generate a curated plan-wide setup proposal to review and apply.",
				Argv:               []string{"author", "context-discover", sessionHandle(sess), "--concepts", "<concept one>,<concept two>", "--complexity", "architectural"},
				ContentPlaceholder: "<concept one>,<concept two>",
			},
			{
				ID:                 "submit-global-context",
				Kind:               NextActionAlternative,
				Label:              "Submit a known global context item",
				Reason:             "Use when a plan-wide setup item is already known.",
				Argv:               []string{"author", "context-submit", sessionHandle(sess), "--kind", "command", "--label", "<label>", "--reason", "<reason>", "--instruction", "<instruction>", "--command", "<command>", "--required"},
				ContentPlaceholder: "<label> / <reason> / <command>",
			},
			{
				ID:                 "submit-skill-context",
				Kind:               NextActionAlternative,
				Label:              "Submit a global skill context item",
				Reason:             "Use when a relevant internal skill is already known.",
				Argv:               []string{"author", "context-submit", sessionHandle(sess), "--kind", "skill", "--label", "<skill>", "--target", "<skill>", "--reason", "<why this skill matters>", "--instruction", "Load this internal skill before implementation.", "--required"},
				ContentPlaceholder: "<skill> / <why this skill matters>",
			},
			{
				ID:                 "skip-skill-context",
				Kind:               NextActionAlternative,
				Label:              "Record no skill context (with reason)",
				Reason:             "Use only when discovery found no relevant internal skill for this plan.",
				Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionRelevantContext), "--content", "NO_SKILL_CONTEXT: <reason>"},
				ContentPlaceholder: "NO_SKILL_CONTEXT: <reason>",
			},
			{
				ID:                 "skip-global-context",
				Kind:               NextActionAlternative,
				Label:              "Record no global context (with reason)",
				Reason:             "Use only when the plan genuinely needs no plan-wide setup context.",
				Argv:               []string{"author", "section-submit", sessionHandle(sess), "--section", string(SectionRelevantContext), "--content", "NO_CONTEXT: <reason>"},
				ContentPlaceholder: "NO_CONTEXT: <reason>",
			},
		},
	}
}

func stepForContextDiscovery(sess Session) GuidedStep {
	batch, _ := latestPendingDiscoveryBatch(sess)
	batchID := batch.ID
	return GuidedStep{
		StepKind:       "context_discovery",
		Title:          "Context Discovery",
		Summary:        "Review the curated context proposal, then apply the batch in one call.",
		Instructions:   []string{"Take only candidates that materially reduce implementation ambiguity.", "Use handle:phase-id for phase-specific setup.", "Sweep noisy or duplicate candidates with the batch note; give a reason when dropping a high-confidence candidate."},
		RequiredInputs: []string{"context-apply for the pending batch"},
		Examples:       []string{"author context-apply " + sessionHandle(sess) + " --batch " + batchID + " --take c1 --note 'reviewed shortlist'", "author context-apply " + sessionHandle(sess) + " --batch " + batchID + " --take c2:phase-1 --drop c3='covered by c2'"},
		CommonMistakes: []string{"Accepting every candidate.", "Leaving a curated batch unapplied.", "Accepting a command candidate without inspecting whether it is current."},
		NextActions: []NextAction{
			{
				ID:     "apply-context-batch",
				Kind:   NextActionRecommended,
				Label:  "Apply context discovery batch",
				Reason: "Close the curated proposal by taking selected handles and sweeping the rest.",
				Argv:   []string{"author", "context-apply", sessionHandle(sess), "--batch", batchID, "--take", "<handle>", "--note", "reviewed shortlist"},
			},
		},
	}
}

func stepAfterContextDiscovery(sess Session) GuidedStep {
	if _, ok := latestPendingDiscoveryBatch(sess); ok {
		return stepForContextDiscovery(sess)
	}
	return stepForCurrentSessionState(sess)
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
			Argv:               []string{"author", "phase-submit", sessionHandle(sess), phase.ID, "--field", string(PhaseFieldNoCodeRefsReason), "--content", "NO_CODE_REFS: <reason>"},
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
		step := phaseStep(sess, phase, field, "phase_relevant_context", "Phase Relevant Context", "Attach phase-scoped setup context or record why no setup context exists. BOTH paths satisfy this gate: `context-submit --phase` for a TYPED item (executable/loadable setup — command, skill, doc target), or `phase-submit --field relevant_context` for prose notes and the NO_CONTEXT: skip reason. Use the typed path whenever the setup is runnable.", []string{"relevant_context or NO_CONTEXT reason"}, []string{"prompt-manager skill read api-steer", "NO_CONTEXT: docs-only review phase has no extra setup."}, "NO_CONTEXT: <reason>")
		step.NextActions = append(step.NextActions, NextAction{
			ID:                 "phase-context-submit",
			Kind:               NextActionAlternative,
			Label:              "Submit typed phase context item",
			Reason:             "Equally valid gate path: use for a concrete setup item (command/skill/doc) with a relevance reason; a violation-rejected item is NOT accepted and leaves this gate open.",
			Argv:               []string{"author", "context-submit", sessionHandle(sess), "--phase", phase.ID, "--kind", "doc", "--label", "<label>", "--reason", "<reason>", "--target", "<target>", "--required"},
			ContentPlaceholder: "<label> / <reason> / <target>",
		})
		return step
	default:
		return GuidedStep{
			StepKind:       "phase_review",
			Title:          "Phase Review",
			Summary:        "This phase has the required fields. Add relevant context or reminders if useful.",
			Instructions:   []string{"Use context-submit or context-accept for phase-specific setup.", "Use reminders or handoff notes for phase-specific decisions/findings/records the implementation agent should be especially likely to capture.", "Move to the next incomplete phase when ready."},
			Examples:       []string{"author phase-next " + sessionHandle(sess)},
			CommonMistakes: []string{"Repeating whole-plan context in every phase.", "Adding generic reminders with no phase-specific value.", "Duplicating the default Execution Feedback section in phase prose."},
			Checklist:      phaseChecklist(phase),
			NextActions: []NextAction{
				{
					ID:     "next-phase",
					Kind:   NextActionRecommended,
					Label:  "Find next incomplete phase",
					Reason: "This phase is complete enough for review.",
					Argv:   []string{"author", "phase-next", sessionHandle(sess)},
				},
			},
		}
	}
}

func phaseStep(sess Session, phase PhaseDraft, field PhaseField, kind, title, summary string, required, examples []string, placeholder string) GuidedStep {
	step := GuidedStep{
		StepKind: kind,
		Title:    title,
		Summary:  summary,
		Instructions: []string{
			"A phase needs all of: title, intent, references (or no_code_refs_reason), steps, validation, acceptance, relevant_context (or NO_CONTEXT: reason) — the checklist shows live status for every field.",
			"Fields may be submitted in any order; if you already know the content, submit the remaining fields without waiting for per-field prompts.",
			"Keep each field concrete enough for a later agent to act on without reading the whole conversation.",
		},
		RequiredInputs: required,
		Examples:       examples,
		CommonMistakes: []string{"Using vague prose that cannot be validated.", "Waiting for a prompt per field when the checklist already lists everything still missing."},
		Checklist:      phaseChecklist(phase),
		NextActions: []NextAction{
			{
				ID:                 "submit-phase-" + string(field),
				Kind:               NextActionRecommended,
				Label:              "Submit phase " + string(field),
				Reason:             fmt.Sprintf("Next missing field %q for phase %d (%s).", field, phase.Order, phase.ID),
				Argv:               []string{"author", "phase-submit", sessionHandle(sess), phase.ID, "--field", string(field), "--content", placeholder},
				ContentPlaceholder: placeholder,
			},
		},
	}
	// When the checklist shows more than one gap, advertise the batched form —
	// a competent agent lands the whole phase in one call instead of walking
	// the per-field prompts.
	if missing := missingPhaseChecklistFields(phase); len(missing) > 1 {
		argv := []string{"author", "phase-submit", sessionHandle(sess), phase.ID}
		for _, missingField := range missing {
			argv = append(argv, "--set", fmt.Sprintf("%s=<%s>", missingField, missingField))
		}
		step.NextActions = append(step.NextActions, NextAction{
			ID:     "batch-submit-phase",
			Kind:   NextActionAlternative,
			Label:  "Batch-submit the remaining fields",
			Reason: fmt.Sprintf("%d fields are still missing for phase %d — submit them all in ONE call.", len(missing), phase.Order),
			Argv:   argv,
		})
	}
	return step
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
		Examples:       []string{"author validate " + sessionHandle(sess), "author preview " + sessionHandle(sess), "author finalize " + sessionHandle(sess)},
		CommonMistakes: []string{"Finalizing before phase steps, validation, and acceptance are objective.", "Skipping the render preview and shipping an unreviewed plan."},
		Checklist:      sessionChecklist(sess),
		NextActions: []NextAction{
			{
				ID:     "validate-session",
				Kind:   NextActionRecommended,
				Label:  "Validate structure",
				Reason: "The session appears complete; validation is the gate before finalization.",
				Argv:   []string{"author", "validate", sessionHandle(sess)},
			},
			{
				ID:     "preview-plan",
				Kind:   NextActionAlternative,
				Label:  "Preview rendered plan",
				Reason: "Review the rendered markdown review artifact before finalizing.",
				Argv:   []string{"author", "preview", sessionHandle(sess)},
			},
			{
				ID:     "finalize-session",
				Kind:   NextActionAlternative,
				Label:  "Finalize plan",
				Reason: "Use after validation returns valid and the preview looks right.",
				Argv:   []string{"author", "finalize", sessionHandle(sess)},
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
			Checklist:    sessionChecklist(sess),
			NextActions: []NextAction{
				{
					ID:     "finalize-session",
					Kind:   NextActionRecommended,
					Label:  "Finalize plan",
					Reason: "The structure gate passed.",
					Argv:   []string{"author", "finalize", sessionHandle(sess)},
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
	planHandle := strings.TrimSpace(slug)
	if planHandle == "" {
		planHandle = planID
	}
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
				Argv:   []string{"plans", "get", planHandle},
			},
			{
				ID:     "start-execution",
				Kind:   NextActionAlternative,
				Label:  "Start execution",
				Reason: "Begin guided implementation for this plan.",
				Argv:   []string{"exec", "start", planHandle},
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
	if spec, ok := sectionSpecByKey(key); ok && spec.Placeholder != "" {
		return spec.Placeholder
	}
	return fmt.Sprintf("<%s content>", key)
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
	return planmodel.OnlyRecommended(step)
}
