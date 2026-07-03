package plans

import (
	"fmt"
	"strings"

	"plan-manager/internal/planmodel"
)

const defaultValidationStrategy = "Baseline diff shows no unexplained regressions; expected related surfaces improve."

type RenderOptions struct {
	Compact            bool
	AuthoringSessionID string
}

// RenderMarkdown renders a structured Plan to its human-readable markdown
// projection. The output is DETERMINISTIC — the same record always renders the
// same bytes — because the markdown is a *view*, never parsed back into truth
// (see docs/concepts/PLAN-MODEL.md). Field order is fixed; empty optional
// sections are omitted so the view stays readable, but the section ordering for
// present fields never varies.
func RenderMarkdown(p Plan) string {
	return RenderMarkdownWithOptions(p, RenderOptions{})
}

// RenderMarkdownWithOptions renders the nine reader-question clusters in fixed
// order (contract decision D1): Purpose / Problem / Outcome / Approach &
// Decisions / Boundaries / Assumptions & Risks / Verification / Execution
// Setup / Phases. Field identity is preserved (D2) — clusters are a render
// grouping over the same structured fields, rendered as `###` subsections.
func RenderMarkdownWithOptions(p Plan, opts RenderOptions) string {
	var b strings.Builder
	renderHeader(&b, p, opts)
	writeSection(&b, "Purpose", p.Purpose)
	writeSection(&b, "Problem", p.ProblemStatement)
	writeSection(&b, "Outcome", p.TargetOutcome)
	renderApproachCluster(&b, p)
	renderBoundariesCluster(&b, p)
	renderAssumptionsRisksCluster(&b, p)
	renderVerificationCluster(&b, p)
	renderExecutionSetupCluster(&b, p, opts)
	renderPhaseSections(&b, p)
	renderGovernanceSections(&b, p, opts)
	return b.String()
}

// renderApproachCluster renders the Approach & Decisions cluster: the technical
// approach prose (design rationale) directly under the cluster heading, then
// the pinned plan-time contract decisions as an ordered D1..Dn list (D3).
func renderApproachCluster(b *strings.Builder, p Plan) {
	hasApproach := strings.TrimSpace(p.TechnicalApproach) != ""
	if !hasApproach && len(p.Decisions) == 0 {
		return
	}
	b.WriteString("## Approach & Decisions\n\n")
	if hasApproach {
		b.WriteString(strings.TrimRight(p.TechnicalApproach, "\n"))
		b.WriteString("\n\n")
	}
	if len(p.Decisions) > 0 {
		b.WriteString("### Decisions\n\n")
		b.WriteString("_Pinned at plan time; do not relitigate during execution._\n\n")
		for i, d := range p.Decisions {
			fmt.Fprintf(b, "- **D%d — %s:** %s\n", i+1, strings.TrimSpace(d.Title), strings.TrimSpace(d.Statement))
		}
		b.WriteString("\n")
	}
}

// renderBoundariesCluster answers "what may I touch, what must I not do?" in
// one place: scope, non-goals, constraints, prohibited approaches, the derived
// work posture, and the acceptance change boundary.
func renderBoundariesCluster(b *strings.Builder, p Plan) {
	b.WriteString("## Boundaries\n\n")
	writeSubSection(b, "Scope", p.Scope)
	writeSubSection(b, "Non-Goals", p.NonGoals)
	writeSubSection(b, "Constraints", p.Constraints)
	writeSubSection(b, "Prohibited Approaches", p.ProhibitedApproaches)
	// Posture is always rendered (autofilled; default greenfield).
	b.WriteString("### Work Posture\n\n")
	b.WriteString(renderWorkPosture(p))
	b.WriteString("\n")
	if !p.ChangeBoundary.IsZero() {
		b.WriteString("### Change Boundary\n\n")
		b.WriteString(renderChangeBoundary(p.ChangeBoundary))
		b.WriteString("\n")
	}
}

func renderAssumptionsRisksCluster(b *strings.Builder, p Plan) {
	if strings.TrimSpace(p.Assumptions) == "" && strings.TrimSpace(p.RisksHazards) == "" && len(p.AssumptionRisks) == 0 {
		return
	}
	b.WriteString("## Assumptions & Risks\n\n")
	if len(p.AssumptionRisks) > 0 {
		b.WriteString("| Assumption | If wrong → mitigation |\n|---|---|\n")
		for _, a := range p.AssumptionRisks {
			fmt.Fprintf(b, "| %s | %s |\n", escapeTableCell(a.Statement), escapeTableCell(a.Mitigation))
		}
		b.WriteString("\n")
	}
	writeSubSection(b, "Assumptions", p.Assumptions)
	writeSubSection(b, "Risks / Hazards", p.RisksHazards)
}

// escapeTableCell keeps a markdown table row well-formed: pipes are escaped and
// newlines collapse to spaces (a cell is one line by construction).
func escapeTableCell(v string) string {
	v = strings.Join(strings.Fields(strings.TrimSpace(v)), " ")
	return strings.ReplaceAll(v, "|", "\\|")
}

// renderVerificationCluster answers "how do we prove it works?" in one place:
// the regression anchor, the validation strategy, and the definition of done.
func renderVerificationCluster(b *strings.Builder, p Plan) {
	hasStrategy := strings.TrimSpace(p.ValidationStrategy) != "" || len(p.FinalValidationCommands) > 0
	if !anchorPresent(p.RegressionAnchor) && !hasStrategy && strings.TrimSpace(p.DefinitionOfDone) == "" {
		return
	}
	b.WriteString("## Verification\n\n")
	if anchorPresent(p.RegressionAnchor) {
		b.WriteString("### Regression Anchor\n\n")
		b.WriteString(renderAnchor(p.RegressionAnchor, p.ChangeBoundary))
		b.WriteString("\n")
	}
	if hasStrategy {
		b.WriteString("### Validation Strategy\n\n")
		strategy := strings.TrimSpace(p.ValidationStrategy)
		if strategy == "" && len(p.FinalValidationCommands) > 0 {
			strategy = defaultValidationStrategy
		}
		if strategy != "" {
			b.WriteString(strings.TrimRight(strategy, "\n"))
			b.WriteString("\n")
		}
		if len(p.FinalValidationCommands) > 0 {
			b.WriteString("\n**Final validation commands:**\n")
			for _, c := range p.FinalValidationCommands {
				fmt.Fprintf(b, "- `%s`\n", c)
			}
		}
		b.WriteString("\n")
	}
	writeSubSection(b, "Definition of Done", p.DefinitionOfDone)
}

// renderExecutionSetupCluster answers "what do I load before starting?" in one
// place: the global setup context (skills/docs/searches/commands), the
// connected references, and the one-line execution-feedback pointer.
func renderExecutionSetupCluster(b *strings.Builder, p Plan, opts RenderOptions) {
	hasContext := len(p.RelevantContext) > 0
	hasRefs := len(p.References) > 0
	if !hasContext && !hasRefs && opts.Compact {
		return
	}
	b.WriteString("## Execution Setup\n\n")
	if hasContext {
		b.WriteString(renderRelevantContext(p.RelevantContext, RelevantContextScopeGlobal))
		b.WriteString("\n")
	}
	if hasRefs {
		b.WriteString("### References\n\n")
		for _, ref := range p.References {
			b.WriteString(renderReference(ref))
		}
		b.WriteString("\n")
	}
	if !opts.Compact {
		b.WriteString("### Execution Feedback\n\n")
		b.WriteString(renderExecutionFeedback(p))
		b.WriteString("\n")
	}
}

// writeSubSection renders one `###` subsection inside a cluster, omitted when
// empty.
func writeSubSection(b *strings.Builder, heading, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	fmt.Fprintf(b, "### %s\n\n%s\n\n", heading, strings.TrimRight(body, "\n"))
}

func renderHeader(b *strings.Builder, p Plan, opts RenderOptions) {
	title := p.Title
	if title == "" {
		title = p.Slug
	}
	fmt.Fprintf(b, "# %s\n\n", title)
	fmt.Fprintf(b, "> Status: **%s**", statusLabel(p.Status))
	if p.ContentHash != "" {
		fmt.Fprintf(b, " · content-hash `%s`", shortHash(p.ContentHash))
	}
	b.WriteString("\n\n")
	b.WriteString(renderQualityNotice(p, opts))
}

func renderPhaseSections(b *strings.Builder, p Plan) {
	if len(p.Phases) > 0 {
		b.WriteString("## Phases\n\n")
		for i, ph := range p.Phases {
			b.WriteString(renderPhase(ph, i+1))
		}
	}
}

func renderGovernanceSections(b *strings.Builder, p Plan, opts RenderOptions) {
	if !opts.Compact {
		// Governance: import provenance + preserved legacy sections (only when present).
		b.WriteString(renderImportGovernance(p))
	}

	// Plan-graph edges as a trailing footnote when present.
	if !opts.Compact && (len(p.Supersedes) > 0 || len(p.SupersededBy) > 0) {
		b.WriteString("## Plan Graph\n\n")
		if len(p.Supersedes) > 0 {
			fmt.Fprintf(b, "- Supersedes: %s\n", strings.Join(p.Supersedes, ", "))
		}
		if len(p.SupersededBy) > 0 {
			fmt.Fprintf(b, "- Superseded by: %s\n", strings.Join(p.SupersededBy, ", "))
		}
		b.WriteString("\n")
	}
}

func renderQualityNotice(p Plan, opts RenderOptions) string {
	report := planmodel.AssessPlanQuality(p, "")
	if !report.HasFindings() {
		return ""
	}
	var b strings.Builder
	b.WriteString("> Plan quality: **")
	b.WriteString(report.Status)
	b.WriteString("**")
	if opts.AuthoringSessionID != "" {
		fmt.Fprintf(&b, " · validate with `plan-manager author validate %s`", opts.AuthoringSessionID)
	} else if p.Slug != "" {
		fmt.Fprintf(&b, " · validate with `plan-manager validate run %s`", p.Slug)
	}
	b.WriteString("\n")
	limit := len(report.Findings)
	if limit > 6 {
		limit = 6
	}
	for i := 0; i < limit; i++ {
		finding := report.Findings[i]
		fmt.Fprintf(&b, "> - %s `%s` at `%s`: %s\n", finding.Severity, finding.Code, finding.Location, finding.Message)
	}
	if len(report.Findings) > limit {
		fmt.Fprintf(&b, "> - ...and %d more quality finding(s)\n", len(report.Findings)-limit)
	}
	b.WriteString("\n")
	return b.String()
}

// renderExecutionFeedback renders the default capture policy every plan carries
// while executing: a one-line pointer at the typed log commands, plus the
// pre-filled completion-record command. The command is stamped concretely —
// scenario and title filled in — because transcript analysis showed agents
// reconstructing it from memory at end-of-plan was a dominant failure source.
func renderExecutionFeedback(p Plan) string {
	var b strings.Builder
	b.WriteString("Log typed work products as they happen. Example:\n\n")
	b.WriteString("```bash\nplan-manager log decision-add <execution-id> --phase <phase-id> --title \"...\" --detail \"...\"\n```\n\n")
	b.WriteString("Other variants: `finding-add`, `bug-add`, `record-add`, `note-add`. When the handle is an execution id, omitting `--phase` uses that execution's current phase; `--phase` also accepts a phase id or 1-based ordinal. If the computed scope is wrong, run `plan-manager log reassign <entry-id> --phase <phase-id-or-ordinal>`.\n\n")
	b.WriteString("On completion, write the learning-loop record — copy, fill the `<...>` placeholders, run:\n\n")

	scenario := "<scenario>"
	if affected := p.ChangeBoundary.AffectedScenarios(); len(affected) > 0 {
		scenario = affected[0]
	}
	trigger := "<one-line goal>"
	if title := strings.TrimSpace(strings.ReplaceAll(p.Title, "'", "")); title != "" {
		trigger = title + ": <one-line goal>"
	}
	fmt.Fprintf(&b, "```bash\nswarm-manager records create --kind execute --scenario %s \\\n"+
		"  --trigger '%s' \\\n"+
		"  --approach '<what was built + key decisions>' \\\n"+
		"  --evidence '<suites/baselines/live checks that prove it>' \\\n"+
		"  --outcome shipped\n```\n", scenario, trigger)
	return b.String()
}

// renderWorkPosture renders the always-present Work Posture section: a source/
// detail line plus the exact guidance block for the posture. An unset posture
// falls back to greenfield so the block is never contradictory or missing.
func renderWorkPosture(p Plan) string {
	posture := p.WorkPosture
	if posture == WorkPostureUnspecified {
		posture = WorkPostureGreenfield
	}
	var b strings.Builder
	source := p.WorkPostureSource
	if source == WorkPostureSourceUnspecified {
		source = WorkPostureSourceDefault
	}
	fmt.Fprintf(&b, "- Posture: **%s**\n", posture)
	fmt.Fprintf(&b, "- Source: %s\n", source)
	if strings.TrimSpace(p.WorkPostureDetail) != "" {
		fmt.Fprintf(&b, "- Detail: %s\n", strings.TrimSpace(p.WorkPostureDetail))
	}
	b.WriteString("\n")
	b.WriteString(PostureBlock(posture))
	b.WriteString("\n")
	return b.String()
}

// renderImportGovernance renders the Import Provenance and Preserved Legacy
// Sections blocks, only when the plan was imported.
func renderImportGovernance(p Plan) string {
	if p.ImportProvenance == nil && len(p.PreservedLegacySections) == 0 {
		return ""
	}
	var b strings.Builder
	if p.ImportProvenance != nil {
		b.WriteString("## Import Provenance\n\n")
		if p.ImportProvenance.SourcePath != "" {
			fmt.Fprintf(&b, "- Source: `%s`\n", p.ImportProvenance.SourcePath)
		}
		if p.ImportProvenance.OriginalFormat != "" {
			fmt.Fprintf(&b, "- Original format: %s\n", p.ImportProvenance.OriginalFormat)
		}
		if p.ImportProvenance.ImportedAt != "" {
			fmt.Fprintf(&b, "- Imported at: `%s`\n", p.ImportProvenance.ImportedAt)
		}
		if p.ImportProvenance.WorkspaceID != "" {
			fmt.Fprintf(&b, "- Workspace ID: `%s`\n", p.ImportProvenance.WorkspaceID)
		}
		if p.ImportProvenance.WorkspaceRoot != "" {
			fmt.Fprintf(&b, "- Workspace root: `%s`\n", p.ImportProvenance.WorkspaceRoot)
		}
		if p.ImportProvenance.Note != "" {
			fmt.Fprintf(&b, "- Note: %s\n", p.ImportProvenance.Note)
		}
		b.WriteString("\n")
	}
	if len(p.PreservedLegacySections) > 0 {
		b.WriteString("## Preserved Legacy Sections\n\n")
		b.WriteString("_Imported sections that did not map to a canonical field, kept verbatim so nothing is lost._\n\n")
		for _, sec := range p.PreservedLegacySections {
			fmt.Fprintf(&b, "### %s\n\n", sec.Heading)
			if sec.MappedTo != "" {
				fmt.Fprintf(&b, "- Mapped to: %s\n", sec.MappedTo)
			}
			reason := sec.PreservationReason
			if reason == "" {
				reason = PreservationReasonUnmapped
			}
			fmt.Fprintf(&b, "- Preservation reason: %s\n\n", reason)
			if strings.TrimSpace(sec.Content) != "" {
				b.WriteString(strings.TrimRight(sec.Content, "\n"))
				b.WriteString("\n\n")
			}
		}
	}
	return b.String()
}

// renderStringList renders a bold-header block of bullet (or numbered) items.
func renderStringList(b *strings.Builder, header string, items []string, numbered bool) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(b, "**%s:**\n", header)
	for i, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		if numbered {
			fmt.Fprintf(b, "%d. %s\n", i+1, item)
		} else {
			fmt.Fprintf(b, "- %s\n", item)
		}
	}
	b.WriteString("\n")
}

func renderPhase(ph Phase, fallbackOrder int) string {
	var b strings.Builder
	order := ph.Order
	if order <= 0 {
		order = fallbackOrder
	}
	fmt.Fprintf(&b, "### Phase %d — %s\n\n", order, ph.Title)
	fmt.Fprintf(&b, "- Status: **%s**\n", phaseStatusLabel(ph.Status))
	if ph.Intent != "" {
		fmt.Fprintf(&b, "- Intent: %s\n", ph.Intent)
	}
	b.WriteString("\n")
	renderStringList(&b, "Affected Areas", ph.AffectedAreas, false)
	if !ph.ChangeBoundary.IsZero() {
		b.WriteString(renderPhaseChangeBoundary(ph.ChangeBoundary))
	}
	context := phaseRelevantContext(ph)
	if reason, ok := noContextOnlyReason(context); ok {
		// A phase whose only setup context is an explicit NO_CONTEXT skip renders
		// one honest line, not a full context-setup block around a note.
		fmt.Fprintf(&b, "- Context: none needed — %s\n\n", reason)
	} else if len(context) > 0 {
		b.WriteString("**Phase Context Setup:**\n\n")
		b.WriteString(renderRelevantContext(context, RelevantContextScopePhase))
		b.WriteString("\n")
	}
	renderStringList(&b, "Ordered Steps", ph.Steps, true)
	renderStringList(&b, "Expected Outputs", ph.ExpectedOutputs, false)
	if strings.TrimSpace(ph.Validation) != "" {
		b.WriteString("**Phase Validation:**\n\n")
		b.WriteString(strings.TrimRight(ph.Validation, "\n"))
		b.WriteString("\n\n")
	}
	if ph.Acceptance != "" {
		fmt.Fprintf(&b, "- Acceptance: %s\n\n", ph.Acceptance)
	}
	renderStringList(&b, "Risks / Hazards", ph.RisksHazards, false)
	if strings.TrimSpace(ph.HandoffNotes) != "" {
		b.WriteString("**Handoff Notes:**\n\n")
		b.WriteString(strings.TrimRight(ph.HandoffNotes, "\n"))
		b.WriteString("\n\n")
	}
	if len(ph.Reminders) > 0 {
		b.WriteString("**Reminders:**\n")
		for _, r := range ph.Reminders {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
	}
	if len(ph.BaselineScope) > 0 {
		b.WriteString("**Baseline scope:**\n")
		for _, c := range ph.BaselineScope {
			fmt.Fprintf(&b, "- `%s`\n", c)
		}
		b.WriteString("\n")
	}
	if len(ph.References) > 0 {
		b.WriteString("**References:**\n")
		for _, ref := range ph.References {
			b.WriteString(renderReference(ref))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// phaseRelevantContext migrates a legacy phase's raw RequiredReading lines into
// typed context items via the model-owned parser — the SSOT for setup-line
// classification — so a skill line's Target carries only the skill slug and the
// full runnable command lands in Command/Argv. A renderer-local classifier here
// once stored the whole command in Target, which the command renderer then
// re-prefixed into `prompt-manager skill read prompt-manager skill read <x>`.
// noContextOnlyReason reports whether every context item of a phase is a typed
// NO_CONTEXT skip note, returning the (first) skip reason for the compact
// single-line render. Any concrete setup item keeps the full block.
func noContextOnlyReason(items []RelevantContextItem) (string, bool) {
	if len(items) == 0 {
		return "", false
	}
	reason := ""
	for _, item := range items {
		if !planmodel.IsNoContextItem(item) {
			return "", false
		}
		if reason == "" {
			for _, value := range []string{item.Label, item.Instruction, item.Reason} {
				value = strings.TrimSpace(value)
				if strings.HasPrefix(strings.ToUpper(value), "NO_CONTEXT:") {
					reason = strings.TrimSpace(value[len("NO_CONTEXT:"):])
					break
				}
			}
		}
	}
	if reason == "" {
		reason = "no phase-specific setup context."
	}
	return reason, true
}

func phaseRelevantContext(ph Phase) []RelevantContextItem {
	if len(ph.RelevantContext) > 0 {
		return ph.RelevantContext
	}
	if len(ph.RequiredReading) == 0 {
		return nil
	}
	items := make([]RelevantContextItem, 0, len(ph.RequiredReading))
	for i, raw := range ph.RequiredReading {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		item := planmodel.RelevantContextItemFromSetupLine(raw, RelevantContextScopePhase, ph.ID, "")
		item.ID = fmt.Sprintf("%s-required-reading-%d", ph.ID, i+1)
		items = append(items, item)
	}
	return items
}

func renderRelevantContext(items []RelevantContextItem, defaultScope RelevantContextScope) string {
	var b strings.Builder
	groups := []struct {
		heading string
		kinds   map[RelevantContextKind]bool
	}{
		{heading: "Load Skills", kinds: map[RelevantContextKind]bool{RelevantContextSkill: true}},
		{heading: "Read Docs", kinds: map[RelevantContextKind]bool{RelevantContextDoc: true}},
		{heading: "Run Discovery Searches", kinds: map[RelevantContextKind]bool{RelevantContextSearch: true}},
		{heading: "Run Commands", kinds: map[RelevantContextKind]bool{RelevantContextCommand: true}},
		{heading: "Inspect References", kinds: map[RelevantContextKind]bool{RelevantContextCodeRef: true, RelevantContextReqRef: true}},
		{heading: "Operator Notes", kinds: map[RelevantContextKind]bool{RelevantContextNote: true, "": true}},
	}
	for _, group := range groups {
		filtered := relevantContextByKind(items, group.kinds, defaultScope)
		if len(filtered) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s\n\n", group.heading)
		for _, item := range filtered {
			b.WriteString(renderRelevantContextItem(item))
		}
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n") + "\n"
}

func relevantContextByKind(items []RelevantContextItem, kinds map[RelevantContextKind]bool, defaultScope RelevantContextScope) []RelevantContextItem {
	out := make([]RelevantContextItem, 0, len(items))
	for _, item := range items {
		if item.Scope == "" {
			item.Scope = defaultScope
		}
		if kinds[item.Kind] {
			out = append(out, item)
		}
	}
	return out
}

func renderRelevantContextItem(item RelevantContextItem) string {
	var b strings.Builder
	label := firstNonEmpty(item.Label, item.Target, item.Command, item.Instruction, "context")
	label, labelNote := displayRelevantContextLabel(item, label)
	fmt.Fprintf(&b, "- %s", label)
	command := relevantContextCommand(item)
	if command != "" && shouldInlineRelevantContextCommand(item) {
		fmt.Fprintf(&b, " — `%s`", command)
	}
	annotations := relevantContextAnnotations(item)
	if len(annotations) > 0 {
		fmt.Fprintf(&b, " _(%s)_", strings.Join(annotations, ", "))
	}
	b.WriteString("\n")
	// The canned authoring placeholder reason and any value that merely repeats
	// the label/instruction add no information — omit them instead of stamping
	// a boilerplate "Reason:" line under every note.
	reason := firstNonEmpty(item.Reason, labelNote)
	if reason != "" && reason != planmodel.AuthoredPhaseNoteReason && reason != label && reason != item.Instruction {
		fmt.Fprintf(&b, "  - Reason: %s\n", compactContextReason(reason))
	}
	if item.Instruction != "" && item.Instruction != label && !defaultRelevantContextInstruction(item) {
		fmt.Fprintf(&b, "  - Instruction: %s\n", item.Instruction)
	}
	if command != "" && !shouldInlineRelevantContextCommand(item) {
		b.WriteString("  ```bash\n")
		fmt.Fprintf(&b, "  %s\n", command)
		b.WriteString("  ```\n")
	}
	if item.StatusDetail != "" {
		fmt.Fprintf(&b, "  - Status: %s\n", item.StatusDetail)
	}
	return b.String()
}

func relevantContextAnnotations(item RelevantContextItem) []string {
	var out []string
	if item.Required {
		out = append(out, "required")
	}
	if item.RepeatPolicy != "" && item.RepeatPolicy != RelevantContextOncePerExecution && item.RepeatPolicy != RelevantContextPhaseEntry {
		out = append(out, repeatPolicyLabel(item.RepeatPolicy))
	}
	if item.Source != "" && item.Source != RelevantContextSourceAuthored {
		out = append(out, string(item.Source))
	}
	if item.Status != "" && item.Status != RelevantContextStatusReady {
		out = append(out, string(item.Status))
	}
	return out
}

// relevantContextCommand delegates to the model-owned idempotent assembler.
func relevantContextCommand(item RelevantContextItem) string {
	return planmodel.RelevantContextCommandLine(item)
}

func displayRelevantContextLabel(item RelevantContextItem, label string) (string, string) {
	switch item.Kind {
	case RelevantContextSkill, RelevantContextDoc:
		return planmodel.SplitSetupLineNoteForRender(label)
	default:
		return label, ""
	}
}

func shouldInlineRelevantContextCommand(item RelevantContextItem) bool {
	switch item.Kind {
	case RelevantContextSkill, RelevantContextDoc, RelevantContextSearch, RelevantContextCommand:
		return true
	default:
		return false
	}
}

func defaultRelevantContextInstruction(item RelevantContextItem) bool {
	instruction := strings.TrimSpace(item.Instruction)
	switch item.Kind {
	case RelevantContextSkill:
		return instruction == "Load this internal skill before implementation."
	case RelevantContextDoc:
		return instruction == "Read this document before implementation."
	case RelevantContextSearch:
		return instruction == "Run this discovery search before implementation."
	case RelevantContextCommand:
		return instruction == "Run this command before implementation." ||
			instruction == "Recall this prior-work record before implementation." ||
			instruction == "Inspect this executable action before hand-rolling the step."
	default:
		return false
	}
}

func compactContextReason(reason string) string {
	const max = 180
	reason = strings.Join(strings.Fields(strings.TrimSpace(reason)), " ")
	if len(reason) <= max {
		return reason
	}
	cut := max
	if i := strings.LastIndex(reason[:max], " "); i > 80 {
		cut = i
	}
	return strings.TrimSpace(reason[:cut]) + "..."
}

func repeatPolicyLabel(policy RelevantContextRepeatPolicy) string {
	switch policy {
	case RelevantContextOnResume:
		return "run on resume"
	case RelevantContextEveryPhase:
		return "run every phase"
	case RelevantContextAsNeeded:
		return "as needed"
	default:
		return string(policy)
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func renderReference(ref Reference) string {
	marker := referenceMarker(ref.Kind)
	line := fmt.Sprintf("- [%s: %s]", marker, ref.Target)
	annotations := make([]string, 0, 3)
	if ref.Future {
		annotations = append(annotations, "future")
	}
	if ref.Resolution != "" && ref.Resolution != ResolutionResolved {
		annotations = append(annotations, string(ref.Resolution))
	}
	if ref.Staleness != "" && ref.Staleness != StalenessFresh {
		annotations = append(annotations, string(ref.Staleness))
	}
	if len(annotations) > 0 {
		line += " _(" + strings.Join(annotations, ", ") + ")_"
	}
	return line + "\n"
}

// renderChangeBoundary renders the acceptance_allow / acceptance_deny lists (and
// an operator-only reason when no allow list exists). Globs are backticked so the
// parser recovers them verbatim and the markdown reads cleanly. Affected
// scenarios are NOT rendered here — they are derived and surfaced by posture and
// the anchor, keeping this section purely the authored source of truth.
func renderChangeBoundary(b ChangeBoundary) string {
	b = b.Normalized()
	var sb strings.Builder
	if len(b.AcceptanceAllow) > 0 {
		sb.WriteString("**Acceptance allow:**\n")
		for _, g := range b.AcceptanceAllow {
			fmt.Fprintf(&sb, "- `%s`\n", g)
		}
		sb.WriteString("\n")
	}
	if len(b.AcceptanceDeny) > 0 {
		sb.WriteString("**Acceptance deny:**\n")
		for _, g := range b.AcceptanceDeny {
			fmt.Fprintf(&sb, "- `%s`\n", g)
		}
		sb.WriteString("\n")
	}
	if b.OperatorOnlyReason != "" {
		fmt.Fprintf(&sb, "- Operator-only: %s\n", b.OperatorOnlyReason)
	}
	return sb.String()
}

// renderPhaseChangeBoundary renders a phase's optional boundary refinement as a
// compact bold-header block (Allow/Deny comma lists) terminated like other phase
// blocks. It is recovered by parsePhaseChangeBoundary.
func renderPhaseChangeBoundary(b ChangeBoundary) string {
	b = b.Normalized()
	var sb strings.Builder
	sb.WriteString("**Change Boundary:**\n")
	if len(b.AcceptanceAllow) > 0 {
		fmt.Fprintf(&sb, "- Allow: %s\n", backtickJoin(b.AcceptanceAllow))
	}
	if len(b.AcceptanceDeny) > 0 {
		fmt.Fprintf(&sb, "- Deny: %s\n", backtickJoin(b.AcceptanceDeny))
	}
	if b.OperatorOnlyReason != "" {
		fmt.Fprintf(&sb, "- Operator-only: %s\n", b.OperatorOnlyReason)
	}
	sb.WriteString("\n")
	return sb.String()
}

func backtickJoin(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, "`"+v+"`")
	}
	return strings.Join(quoted, ", ")
}

func renderAnchor(a RegressionAnchor, boundary ChangeBoundary) string {
	var b strings.Builder
	if a.Strategy == AnchorStrategyLegacyProse {
		b.WriteString("- Legacy anchor prose, not executable.\n")
		if prose := strings.TrimSpace(a.BaselineName); prose != "" {
			fmt.Fprintf(&b, "- Prose: %s\n", prose)
		}
		if a.Unavailable {
			b.WriteString("- Repair required: replace with a change-boundary anchor before execution.\n")
		}
		return b.String()
	}
	if a.Unavailable {
		b.WriteString("- _anchor autofill was unavailable; capture before changes_\n")
	}
	if a.Strategy != "" {
		fmt.Fprintf(&b, "- Strategy: %s\n", a.Strategy)
	}
	if a.Scenario != "" {
		fmt.Fprintf(&b, "- Scenario baseline: `%s`", a.Scenario)
		if a.BaselineName != "" {
			fmt.Fprintf(&b, " (name `%s`)", a.BaselineName)
		}
		b.WriteString("\n")
		status := strings.TrimSpace(a.CaptureStatus)
		if status == "" {
			status = "requested; usable only after `git-control-tower baseline snapshot status --wait --json` reports one or more captured surfaces"
		}
		fmt.Fprintf(&b, "- Capture status: %s\n", status)
		if reason := strings.TrimSpace(a.CaptureReason); reason != "" {
			fmt.Fprintf(&b, "- Capture reason: %s\n", reason)
		}
	}
	if a.Scenario == "" && a.BaselineName != "" {
		fmt.Fprintf(&b, "- Baseline name: `%s`\n", a.BaselineName)
	}
	if a.HeadSha != "" {
		if isExecutionStartHeadSentinel(a.HeadSha) {
			b.WriteString("- HEAD sha: captured at execution start\n")
		} else {
			fmt.Fprintf(&b, "- HEAD sha: `%s`\n", a.HeadSha)
		}
	}
	if len(a.AllowlistPaths) > 0 {
		fmt.Fprintf(&b, "- Allowlist: %s\n", strings.Join(a.AllowlistPaths, ", "))
	}
	if a.HeadSha != "" || len(a.AllowlistPaths) > 0 {
		fallback := strings.TrimSpace(a.Fallback)
		if fallback == "" {
			fallback = "HEAD sha + allowlist diff is informational when the scenario baseline is unavailable or captured zero surfaces"
		}
		fmt.Fprintf(&b, "- Fallback: %s\n", fallback)
	}
	if a.CapturedAt != "" {
		fmt.Fprintf(&b, "- Captured at: `%s`\n", a.CapturedAt)
	}
	renderAnchorCommands(&b, renderAnchorCommandSet(a, boundary))
	return b.String()
}

func renderAnchorCommandSet(a RegressionAnchor, boundary ChangeBoundary) []string {
	if len(a.Commands) > 0 {
		return a.Commands
	}
	if a.Strategy == AnchorStrategyChangeBoundary && !boundary.IsZero() {
		commands, _ := planmodel.BoundaryAnchorCommands(boundary, a.BaselineName, a.HeadSha)
		return commands
	}
	return planmodel.RegressionAnchorCommands(a)
}

func isExecutionStartHeadSentinel(v string) bool {
	v = strings.Trim(strings.TrimSpace(v), "`<>")
	return strings.EqualFold(v, "captured at execution start")
}

// renderAnchorCommands groups the anchor's derived commands into the scenario
// baseline ORACLE tier and the repo/path diff INFORMATIONAL tier, with explicit
// labels so a reviewer never mistakes an informational diff for a pass/fail
// oracle. Commands are still rendered as backticked bullets so the parser
// recovers them verbatim; the bold labels are presentation-only.
func renderAnchorCommands(b *strings.Builder, commands []string) {
	if len(commands) == 0 {
		return
	}
	var oracle, informational []string
	for _, c := range commands {
		if isInformationalDiffCommand(c) {
			informational = append(informational, c)
		} else {
			oracle = append(oracle, c)
		}
	}
	if len(oracle) > 0 {
		b.WriteString("**Scenario baseline oracle:**\n")
		for _, c := range oracle {
			fmt.Fprintf(b, "- `%s`\n", c)
		}
	}
	if len(informational) > 0 {
		if len(oracle) > 0 {
			b.WriteString("\n")
		}
		b.WriteString("**Repo/path diff (informational, not a pass/fail oracle):**\n")
		for _, c := range informational {
			fmt.Fprintf(b, "- `%s`\n", c)
		}
	}
}

// isInformationalDiffCommand reports whether a derived anchor/validation command
// is a repo/path diff (informational) rather than a scenario baseline oracle. It
// mirrors the validation domain's isOracleCommand split.
func isInformationalDiffCommand(cmd string) bool {
	return strings.HasPrefix(strings.TrimSpace(cmd), "git diff")
}

func writeSection(b *strings.Builder, heading, body string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	fmt.Fprintf(b, "## %s\n\n%s\n\n", heading, strings.TrimRight(body, "\n"))
}

func anchorPresent(a RegressionAnchor) bool {
	return a.Strategy != "" || a.Scenario != "" || a.HeadSha != "" ||
		len(a.AllowlistPaths) > 0 || len(a.Commands) > 0 || a.Unavailable
}

func referenceMarker(k ReferenceKind) string {
	switch k {
	case ReferenceCode:
		return "CODE"
	case ReferenceReq:
		return "REQ"
	case ReferenceDoc:
		return "DOC"
	default:
		return "CODE"
	}
}

func statusLabel(s PlanStatus) string {
	if s == "" {
		return string(PlanStatusDraft)
	}
	return string(s)
}

func phaseStatusLabel(s PhaseStatus) string {
	if s == "" {
		return string(PhaseStatusTodo)
	}
	return string(s)
}

func shortHash(h string) string {
	if len(h) > 12 {
		return h[:12]
	}
	return h
}
