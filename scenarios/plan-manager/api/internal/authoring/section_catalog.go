package authoring

// sectionSpec is the single catalog entry for an authoring section: skeleton
// order, persistence label, requiredness, base guidance, and the default content
// placeholder used by the guided action.
type sectionSpec struct {
	Key            SectionKey
	Label          string
	Title          string
	Mandatory      bool
	StepKind       string
	Summary        string
	Instructions   []string
	RequiredInputs []string
	Examples       []string
	CommonMistakes []string
	Placeholder    string
}

// defaultSkeleton is the ordered section catalog StartSession seeds and the
// guided wizard renders. The order follows the nine reader-question clusters
// (contract decision D1) so the wizard asks in the same order the artifact
// renders: Purpose / Problem / Outcome / Approach & Decisions / Boundaries
// (scope, non-goals, constraints, prohibited, posture, change boundary) /
// Assumptions & Risks / Verification (anchor, validation strategy, DoD) /
// Execution Setup (relevant context, references) / Phases. The mandatory
// sections gate Finalize; optional sections may be left empty. References and
// the change boundary are mandatory but have typed escape hatches
// (NO_CODE_REFS / OPERATOR_ONLY) enforced by validation.
var defaultSkeleton = []sectionSpec{
	{
		Key:            SectionPurpose,
		Label:          "Purpose",
		Mandatory:      true,
		StepKind:       "purpose",
		Summary:        "Explain why this plan exists and what pressure created it — an abstract, not a second problem statement.",
		Instructions:   []string{"Write one concise paragraph (target: 120 words or fewer).", "Name the outcome, not the implementation details.", "Include enough context for a later implementation agent to understand why this work matters.", "If you find yourself restating the Problem or the Outcome, delete it here — each fact lives in exactly one section."},
		RequiredInputs: []string{"purpose"},
		Examples:       []string{"Make plan-manager authoring reliable enough for local agents by replacing large markdown submissions with guided structured steps."},
		CommonMistakes: []string{"Listing tasks instead of explaining the need.", "Assuming the next agent remembers the originating conversation."},
		Placeholder:    "<one concise purpose paragraph>",
	},
	{
		Key:            SectionDefinitions,
		Label:          "Definitions",
		Mandatory:      false,
		StepKind:       "definitions",
		Summary:        "Define terms this plan coins or narrows so a cold executor has one meaning for each term.",
		Instructions:   []string{"One definition per line as 'Term — meaning' or 'Term: meaning'.", "Define only plan-local coined or narrowed terms.", "Reference docs/concepts/GLOSSARY.md for shared ecosystem terms instead of restating them."},
		RequiredInputs: []string{"definitions (optional)"},
		Examples:       []string{"Trust gate — the validation checkpoint required before a phase can advance.", "Staleness verdict: the computed freshness state for a referenced surface."},
		CommonMistakes: []string{"Restating a shared glossary term.", "Using an empty term or meaning."},
		Placeholder:    "<term> — <meaning>",
	},
	{
		Key:            SectionProblemStatement,
		Label:          "Problem / Need",
		Mandatory:      true,
		StepKind:       "problem_statement",
		Summary:        "State the concrete problem, gap, or need this plan closes.",
		Instructions:   []string{"Describe what is wrong or missing today, specifically.", "Name the pain a reviewer or user feels.", "Do not describe the solution here — that is the technical approach."},
		RequiredInputs: []string{"problem or need"},
		Examples:       []string{"The current plan model is too thin for a human reviewer or a small implementation agent to trust as the main artifact."},
		CommonMistakes: []string{"Describing the solution instead of the problem.", "Being so vague the need can't be verified as solved."},
		Placeholder:    "<the concrete problem or missing capability>",
	},
	{
		Key:            SectionTargetOutcome,
		Label:          "Target Outcome",
		Mandatory:      true,
		StepKind:       "target_outcome",
		Summary:        "Describe the observable end state once this plan is done.",
		Instructions:   []string{"State what is observably true when the work is complete.", "Make it concrete enough to check.", "Keep it outcome-focused, not a task list."},
		RequiredInputs: []string{"target outcome"},
		Examples:       []string{"A human can run `plan-manager plans render <plan>` and judge plan quality without reading DB JSON."},
		CommonMistakes: []string{"Listing tasks instead of the end state.", "Restating the purpose."},
		Placeholder:    "<observable end state>",
	},
	{
		Key:            SectionTechnicalApproach,
		Label:          "Technical Approach",
		Mandatory:      true,
		StepKind:       "technical_approach",
		Summary:        "Explain the chosen approach and the design rationale — why this way.",
		Instructions:   []string{"Describe the strategy at a design level, not a phase-by-phase list.", "Justify the key decisions and name the main alternatives ruled out.", "Keep it concise; the phases carry the step detail."},
		RequiredInputs: []string{"technical approach"},
		Examples:       []string{"Model-first contract change: expand the proto/Go plan model, then wire renderer, parser, wizard, CLI, and UI through that single contract."},
		CommonMistakes: []string{"Turning this into the phase list.", "Stating what without why."},
		Placeholder:    "<approach and rationale>",
	},
	{
		Key:            SectionDecisions,
		Label:          "Decisions",
		Mandatory:      false,
		StepKind:       "decisions",
		Summary:        "Pin plan-time contract decisions (rendered D1..Dn) so execution never relitigates them.",
		Instructions:   []string{"One decision per line as '<title>: <statement>'.", "Pin only decisions an executor might otherwise reopen: names, orders, contracts, dependency postures.", "Skip this section when the approach prose carries no contested choices.", "Execution-time decisions are different: capture those with plan-manager log decision-add as they happen."},
		RequiredInputs: []string{"decisions (optional)"},
		Examples:       []string{"Cluster names and order: Purpose, Problem, Outcome, Approach & Decisions, Boundaries, Assumptions & Risks, Verification, Execution Setup, Phases.", "Dependency posture: prompt-manager and search-hub are declared required:false dependencies."},
		CommonMistakes: []string{"Restating the technical approach as decisions.", "Pinning trivia no executor would relitigate."},
		Placeholder:    "<title>: <statement>",
	},
	{
		Key:          SectionScope,
		Label:        "Scope",
		Mandatory:    true,
		StepKind:     "scope",
		Summary:      "Draw the boundary around the work.",
		Instructions: []string{"State what is in scope and out of scope.", "Name affected scenarios, packages, commands, and surfaces when known.", "Keep future expansion separate from required work."},
		RequiredInputs: []string{
			"scope",
		},
		Examples:       []string{"In scope: authoring API/CLI/UI phase wizard. Out of scope: swarm-manager consumer inversion."},
		CommonMistakes: []string{"Using scope as another purpose paragraph.", "Omitting explicit non-goals for tempting adjacent work."},
		Placeholder:    "In scope: <items>. Out of scope: <items>.",
	},
	{
		Key:            SectionNonGoals,
		Label:          "Non-goals",
		Mandatory:      false,
		StepKind:       "non_goals",
		Summary:        "Name tempting adjacent work this plan intentionally excludes.",
		Instructions:   []string{"List only exclusions that prevent scope drift.", "Keep future follow-up ideas separate from current acceptance."},
		RequiredInputs: []string{"non-goals (optional)"},
		Placeholder:    "<explicit non-goals>",
	},
	{
		Key:            SectionConstraints,
		Label:          "Constraints",
		Mandatory:      false,
		StepKind:       "constraints",
		Summary:        "Record real constraints that affect implementation choices.",
		Instructions:   []string{"Name technical, operational, sequencing, or policy constraints only when they change the work.", "Do not add greenfield-incompatible compatibility-shim language unless this is explicitly brownfield."},
		RequiredInputs: []string{"constraints (optional)"},
		Placeholder:    "<constraints>",
	},
	{
		Key:            SectionProhibitedApproaches,
		Label:          "Prohibited Approaches",
		Mandatory:      false,
		StepKind:       "prohibited_approaches",
		Summary:        "Name approaches that are explicitly off-limits, only when genuinely relevant.",
		Instructions:   []string{"List approaches a reasonable agent might try but must not.", "Skip this section if nothing is genuinely off-limits."},
		RequiredInputs: []string{"prohibited approaches (optional)"},
		Examples:       []string{"Do not clone the legacy 13-section markdown format.", "Do not make markdown the source of truth."},
		CommonMistakes: []string{"Repeating non-goals.", "Listing obvious bad practice with no plan-specific value."},
		Placeholder:    "<prohibited approaches>",
	},
	{
		Key:            SectionWorkPosture,
		Label:          "Work Posture",
		Title:          "Work Posture (autofilled)",
		Mandatory:      false,
		StepKind:       "work_posture",
		Summary:        "Work posture is derived automatically from scenario maturity (default greenfield). Review only — do not author the Greenfield/Brownfield block.",
		Instructions:   []string{"You do not write this section.", "The renderer injects the Greenfield (or Brownfield) block based on the associated scenario's maturity.", "Do not put compatibility-shim/legacy-wrapper language in constraints for a greenfield plan; validation will flag the conflict."},
		RequiredInputs: []string{"(none — autofilled)"},
		CommonMistakes: []string{"Hand-writing a Greenfield block.", "Authoring constraints that contradict the derived posture."},
		Placeholder:    "<autofilled>",
	},
	{
		Key:            SectionAcceptanceBoundary,
		Label:          "Change Boundary",
		Mandatory:      true,
		StepKind:       "acceptance_boundary",
		Summary:        "Declare the repo paths this plan may change as acceptance_allow globs (and optional acceptance_deny guardrails). This is the source of truth for posture, the regression anchor, and validation scope.",
		Instructions:   []string{"List one path glob per line under acceptance_allow: — e.g. scenarios/<name>/**, packages/<pkg>/**, docs/**.", "Use acceptance_deny: for paths that must NOT change (guardrails only — they never widen scope).", "Do NOT name a primary scenario — scenario identity is derived from the allow globs.", "If the plan is genuinely operator-only/no-code, record OPERATOR_ONLY: <reason> instead.", "Replace every <placeholder> with a real path before finalizing."},
		RequiredInputs: []string{"acceptance_allow globs or an OPERATOR_ONLY reason"},
		Examples:       []string{"acceptance_allow:\n- scenarios/plan-manager/**\n- packages/proto/**\nacceptance_deny:\n- scenarios/swarm-manager/**", "OPERATOR_ONLY: documentation-only operator decision with no editable repo paths."},
		CommonMistakes: []string{"Naming a single primary scenario instead of listing path globs.", "Leaving a <scenario>/<path> placeholder unresolved.", "Putting forbidden paths in acceptance_allow."},
		Placeholder:    "acceptance_allow:\n- scenarios/<scenario>/**",
	},
	{
		Key:            SectionAssumptions,
		Label:          "Assumptions",
		Mandatory:      false,
		StepKind:       "assumptions",
		Summary:        "Record the preconditions taken as given.",
		Instructions:   []string{"List environment, access, or prior-work assumptions, one per line.", "Only include assumptions that, if false, change the plan."},
		RequiredInputs: []string{"assumptions (optional)"},
		Examples:       []string{"The regression baseline is captured before any code change."},
		CommonMistakes: []string{"Listing requirements instead of assumptions."},
		Placeholder:    "<assumptions, one per line>",
	},
	{
		Key:       SectionRegressionAnchor,
		Label:     "Regression anchor",
		Title:     "Regression Anchor",
		Mandatory: true,
		StepKind:  "regression_anchor",
		Summary:   "Record the typed INTENT of the before-state (which scenario, allowlist, and diff command). The actual baseline snapshot is captured fresh at execution start — not here.",
		Instructions: []string{
			"Derive the typed anchor intent (strategy, scenario, baseline name, allowlist, diff command), then confirm or adjust it — the <scenario> placeholder must name the real target scenario.",
			"Do not capture a baseline snapshot at authoring time; intent never goes stale and the executor snapshots the real 'before' when execution starts.",
			"Do not claim validation passed here; this is only the before anchor.",
		},
		RequiredInputs: []string{"typed anchor intent (strategy / scenario / baseline name / allowlist / diff command)"},
		Examples:       []string{"Strategy: scenario-baseline", "Allowlist: scenarios/<scenario>/**"},
		CommonMistakes: []string{"Putting final test results in the anchor.", "Trying to capture a snapshot at authoring time instead of recording intent.", "Leaving the <scenario> placeholder unconfirmed."},
		Placeholder:    "Strategy: scenario-baseline\nScenario baseline: <scenario>\nBaseline name: <name>\nHEAD sha: <sha>\nAllowlist: scenarios/<scenario>/**",
	},
	{
		Key:            SectionValidationStrategy,
		Label:          "Validation Strategy",
		Mandatory:      true,
		StepKind:       "validation_strategy",
		Summary:        "Describe how the plan proves it works: baseline approach, what evidence counts, and the final validation commands.",
		Instructions:   []string{"State the baseline/regression approach and the suites/commands that prove success.", "Distinguish per-phase validation from the final end-of-plan validation.", "Reference the exact commands a reviewer runs at the end."},
		RequiredInputs: []string{"validation strategy"},
		Examples:       []string{"Run focused Go/CLI/UI suites per phase; finish with `vrooli scenario test plan-manager` and a clean baseline diff against the captured anchor."},
		CommonMistakes: []string{"Saying 'tests pass' without naming them.", "Confusing the method (validation) with the outcome (definition of done)."},
		Placeholder:    "<validation strategy>",
	},
	{
		Key:            SectionDefinitionOfDone,
		Label:          "Definition of Done",
		Mandatory:      true,
		StepKind:       "definition_of_done",
		Summary:        "Define objective plan-level success gates. Phase acceptances are NOT restated here.",
		Instructions:   []string{"Use pass/fail criteria that another agent can verify.", "List plan-level gates only (full suites, baseline diff, live verification, docs); phase-level acceptance lives on each phase.", "Include validation expectations and adoption readiness.", "Avoid vague language like 'works' or 'complete'."},
		RequiredInputs: []string{"definition_of_done"},
		Examples:       []string{"Authoring API/CLI tests pass; plan-manager validate run returns PASS or documented UNKNOWN dependency gaps; scenario requirements validate green."},
		CommonMistakes: []string{"Restating phase steps.", "Using subjective acceptance criteria."},
		Placeholder:    "<objective done criteria>",
	},
	{
		Key:            SectionRelevantContext,
		Label:          "Relevant context",
		Title:          "Relevant Context",
		Mandatory:      false,
		StepKind:       "relevant_context",
		Summary:        "Discover a broad prompt-manager skill pack for the plan, then add only durable extra context that helps execution.",
		Instructions:   []string{"Decompose the work into 2-5 concepts via four lenses: domain, technology, problem type, scenario surface.", "Run author skill-pack so prompt-manager skills are added directly as global relevant context.", "Keep most returned skills unless they are clearly irrelevant; skills are allowed to support professional code quality, not only the literal task noun.", "Run search-hub directly when docs, records, or code context would help, then submit only durable references or commands that remain useful.", "Do not add generic feedback reminders here; the rendered plan includes the default plan-manager log capture workflow."},
		RequiredInputs: []string{"skill-pack concepts, optional durable context, or explicit NO_CONTEXT reason"},
		Examples:       []string{"author skill-pack <session> --concepts 'plan-manager execution resume,authoring context discovery' --complexity architectural", "prompt-manager skill read implementation-plan-authoring ecosystem-fit", "NO_CONTEXT: documentation-only operator decision."},
		CommonMistakes: []string{"Treating skills as only literal task matches.", "Putting phase-only setup in global context.", "Mirroring raw search-hub output instead of adding durable context.", "Relying on the legacy required_reading section as the primary model."},
		Placeholder:    "NO_CONTEXT: <reason>",
	},
	{
		Key:            SectionReferences,
		Label:          "References",
		Mandatory:      true,
		StepKind:       "references",
		Summary:        "Capture connected locations so staleness and validation can be computed.",
		Instructions:   []string{"Add one locator per line using [CODE:], [DOC:], or [REQ:].", "Give each reference a one-line why when it is not obvious from the path — a reader should know what to look for before opening it.", "Include existing and proposed locations; mark future paths in prose when needed.", "If there truly are no connected code references, write NO_CODE_REFS: followed by the reason."},
		RequiredInputs: []string{"references or NO_CODE_REFS reason"},
		Examples:       []string{"[CODE: scenarios/plan-manager/api/internal/authoring/service.go]", "[REQ: PM-AUTHOR-002]", "NO_CODE_REFS: documentation-only operator decision."},
		CommonMistakes: []string{"Leaving references empty.", "Using plain paths without the machine-readable marker."},
		Placeholder:    "[CODE: path/to/file.go]",
	},
	{
		Key:            SectionPhases,
		Label:          "Phases",
		Title:          "Phase Outline",
		Mandatory:      true,
		StepKind:       "phase_outline",
		Summary:        "Create the phase list, then fill each phase through phase-native commands.",
		Instructions:   []string{"Use author phase-add for each phase instead of submitting one large markdown blob.", "Keep phases sequential and handoff-sized.", "Every phase needs intent, references or a no-code reason, and objective acceptance.", "Use phase handoff notes or reminders only for phase-specific capture triggers; default feedback capture is rendered automatically."},
		RequiredInputs: []string{"phase list"},
		Examples:       []string{"author phase-add <session> --title 'Authoring contract' --intent 'Add phase-native RPC and CLI surface.'"},
		CommonMistakes: []string{"Making one giant implementation phase.", "Skipping per-phase references."},
		Placeholder:    "### Phase 1 — <title>\n- Intent: <intent>\n- Acceptance: <criteria>",
	},
}

func sectionSpecByKey(key SectionKey) (sectionSpec, bool) {
	for _, spec := range defaultSkeleton {
		if spec.Key == key {
			return spec, true
		}
	}
	return sectionSpec{}, false
}

// newSkeleton returns a fresh ordered section list from the default section
// catalog.
func newSkeleton() []Section {
	out := make([]Section, 0, len(defaultSkeleton))
	for _, spec := range defaultSkeleton {
		out = append(out, Section{Key: spec.Key, Label: spec.Label, Mandatory: spec.Mandatory})
	}
	return out
}
