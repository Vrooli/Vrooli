package authoring

import (
	"fmt"
	"regexp"
	"strings"

	planmodel "plan-manager/internal/planmodel"

	"github.com/google/uuid"
)

func sessionToPlan(sess Session) (planmodel.Plan, error) {
	parsed, err := parseReferencesAndPhases(sess)
	if err != nil {
		return planmodel.Plan{}, err
	}
	p := planmodel.Plan{
		Title:                sess.Title,
		Slug:                 sess.Slug,
		Purpose:              contentOf(sess.Sections, SectionPurpose),
		ProblemStatement:     contentOf(sess.Sections, SectionProblemStatement),
		TargetOutcome:        contentOf(sess.Sections, SectionTargetOutcome),
		Scope:                contentOf(sess.Sections, SectionScope),
		NonGoals:             contentOf(sess.Sections, SectionNonGoals),
		TechnicalApproach:    contentOf(sess.Sections, SectionTechnicalApproach),
		Constraints:          contentOf(sess.Sections, SectionConstraints),
		ProhibitedApproaches: contentOf(sess.Sections, SectionProhibitedApproaches),
		ValidationStrategy:   contentOf(sess.Sections, SectionValidationStrategy),
		DefinitionOfDone:     contentOf(sess.Sections, SectionDefinitionOfDone),
		References:           parsed.References,
		Phases:               parsed.Phases,
		RelevantContext:      append([]planmodel.RelevantContextItem(nil), sess.RelevantContext...),
	}
	p.Decisions = decisionsFromContent(contentOf(sess.Sections, SectionDecisions))
	p.Definitions = definitionsFromContent(contentOf(sess.Sections, SectionDefinitions))
	p.Assumptions, p.AssumptionRisks = splitAssumptionsContent(contentOf(sess.Sections, SectionAssumptions))
	p.RelevantContext = append(p.RelevantContext, globalContextReasonNotes(contentOf(sess.Sections, SectionRelevantContext))...)
	p.RelevantContext = append(p.RelevantContext, contextItemsFromLines(contentOf(sess.Sections, SectionRequiredReading), planmodel.RelevantContextScopeGlobal, "")...)
	if len(sess.PhaseDrafts) > 0 {
		p.Phases = phaseDraftsToPlanPhases(sess.PhaseDrafts)
	}
	p.ChangeBoundary = planmodel.ParseBoundarySection(contentOf(sess.Sections, SectionAcceptanceBoundary))
	if reason := noCodeRefsReason(contentOf(sess.Sections, SectionReferences)); reason != "" {
		p.Constraints = appendNoCodeRefsReason(p.Constraints, reason)
	}
	if anchor := strings.TrimSpace(contentOf(sess.Sections, SectionRegressionAnchor)); anchor != "" {
		p.RegressionAnchor = planmodel.ParseRegressionAnchorBlock(anchor)
	}
	if p.RegressionAnchor.Strategy == planmodel.AnchorStrategyChangeBoundary {
		p.BaselineSet = planmodel.BaselineSetFromBoundary(p.ChangeBoundary, p.RegressionAnchor.BaselineName)
	}
	return p, nil
}

func globalContextReasonNotes(content string) []planmodel.RelevantContextItem {
	var out []planmodel.RelevantContextItem
	for _, line := range splitLines(content) {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if !strings.HasPrefix(upper, "NO_CONTEXT:") && !strings.HasPrefix(upper, "NO_SKILL_CONTEXT:") {
			continue
		}
		out = append(out, planmodel.RelevantContextItem{
			ID:           uuid.NewString(),
			Kind:         planmodel.RelevantContextNote,
			Scope:        planmodel.RelevantContextScopeGlobal,
			Label:        line,
			Reason:       line,
			Instruction:  line,
			Required:     true,
			RepeatPolicy: planmodel.RelevantContextOncePerExecution,
			Source:       planmodel.RelevantContextSourceAuthored,
			Status:       planmodel.RelevantContextStatusReady,
		})
	}
	return out
}

func phaseDraftsToPlanPhases(drafts []PhaseDraft) []planmodel.Phase {
	out := make([]planmodel.Phase, 0, len(drafts))
	for i, draft := range drafts {
		order := draft.Order
		if order <= 0 {
			order = i + 1
		}
		phaseID := strings.TrimSpace(draft.ID)
		reminders := append([]string(nil), draft.Reminders...)
		if draft.NoCodeRefsReason != "" {
			reminders = append(reminders, "No connected code references: "+draft.NoCodeRefsReason)
		}
		relevantContext := append([]planmodel.RelevantContextItem(nil), draft.RelevantContext...)
		relevantContext = append(relevantContext, contextItemsFromRequiredReading(draft.RequiredReading, phaseID)...)
		for i := range relevantContext {
			relevantContext[i].Scope = planmodel.RelevantContextScopePhase
			relevantContext[i].PhaseID = phaseID
		}
		out = append(out, planmodel.Phase{
			ID:              phaseID,
			Order:           order,
			Title:           draft.Title,
			Intent:          draft.Intent,
			AffectedAreas:   append([]string(nil), draft.AffectedAreas...),
			Steps:           append([]string(nil), draft.Steps...),
			ExpectedOutputs: append([]string(nil), draft.ExpectedOutputs...),
			Validation:      draft.Validation,
			RisksHazards:    append([]string(nil), draft.RisksHazards...),
			HandoffNotes:    draft.HandoffNotes,
			RequiredReading: append([]string(nil), draft.RequiredReading...),
			Reminders:       reminders,
			Acceptance:      draft.Acceptance,
			References:      append([]planmodel.Reference(nil), draft.References...),
			RelevantContext: relevantContext,
			ValidationScope: draft.ValidationScope,
			Status:          planmodel.PhaseStatusTodo,
		})
	}
	return out
}

func normalizeContextItem(item planmodel.RelevantContextItem, phaseID string) planmodel.RelevantContextItem {
	item.ID = strings.TrimSpace(item.ID)
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.PhaseID = strings.TrimSpace(item.PhaseID)
	if phaseID != "" {
		item.PhaseID = strings.TrimSpace(phaseID)
	}
	item.Label = strings.TrimSpace(item.Label)
	item.Reason = strings.TrimSpace(item.Reason)
	item.Instruction = strings.TrimSpace(item.Instruction)
	item.Command = strings.TrimSpace(item.Command)
	item.Target = strings.TrimSpace(item.Target)
	if item.Scope == "" {
		if phaseID != "" || item.PhaseID != "" {
			item.Scope = planmodel.RelevantContextScopePhase
		} else {
			item.Scope = planmodel.RelevantContextScopeGlobal
		}
	}
	if item.RepeatPolicy == "" {
		if item.Scope == planmodel.RelevantContextScopePhase {
			item.RepeatPolicy = planmodel.RelevantContextPhaseEntry
		} else {
			item.RepeatPolicy = planmodel.RelevantContextOncePerExecution
		}
	}
	if item.Source == "" {
		item.Source = planmodel.RelevantContextSourceAuthored
	}
	if item.Status == "" {
		item.Status = planmodel.RelevantContextStatusReady
	}
	if item.Label == "" {
		item.Label = firstNonEmpty(item.Target, item.Command, item.Instruction, string(item.Kind))
	}
	item = normalizeSkillContextTarget(item)
	return item
}

// normalizeSkillContextTarget repairs a skill context item whose Target was
// submitted as the full `prompt-manager skill read <slug>` command instead of
// the bare slug. The full command moves to Command (when empty) and Target
// keeps only the slug, so no downstream command assembly can double the prefix
// (contract decision D6).
func normalizeSkillContextTarget(item planmodel.RelevantContextItem) planmodel.RelevantContextItem {
	const prefix = "prompt-manager skill read "
	if item.Kind != planmodel.RelevantContextSkill || !strings.HasPrefix(item.Target, prefix) {
		return item
	}
	full := item.Target
	item.Target = strings.TrimSpace(strings.TrimPrefix(full, prefix))
	if item.Command == "" && len(item.Argv) == 0 {
		item.Command = full
		item.Argv = strings.Fields(full)
	}
	if item.Label == full {
		item.Label = item.Target
	}
	return item
}

func defaultRepeatForScope(scope planmodel.RelevantContextScope, current planmodel.RelevantContextRepeatPolicy) planmodel.RelevantContextRepeatPolicy {
	if current != "" && !(scope == planmodel.RelevantContextScopePhase && current == planmodel.RelevantContextOncePerExecution) {
		return current
	}
	if scope == planmodel.RelevantContextScopePhase {
		return planmodel.RelevantContextPhaseEntry
	}
	return planmodel.RelevantContextOncePerExecution
}

func indexOfContextItem(items []planmodel.RelevantContextItem, id string) int {
	id = strings.TrimSpace(id)
	for i := range items {
		if strings.TrimSpace(items[i].ID) == id {
			return i
		}
	}
	return -1
}

func indexOfContextItemByKey(items []planmodel.RelevantContextItem, item planmodel.RelevantContextItem) int {
	key := contextItemKey(item)
	if key == "" {
		return -1
	}
	for i := range items {
		if contextItemKey(items[i]) == key {
			return i
		}
	}
	return -1
}

func contextItemKey(item planmodel.RelevantContextItem) string {
	target := strings.ToLower(strings.TrimSpace(item.Target))
	if target == "" {
		target = strings.ToLower(strings.TrimSpace(item.Command))
	}
	if target == "" {
		return ""
	}
	return string(item.Kind) + "\x00" + target
}

func removeContextItemAt(items []planmodel.RelevantContextItem, pos int) []planmodel.RelevantContextItem {
	out := make([]planmodel.RelevantContextItem, 0, len(items)-1)
	out = append(out, items[:pos]...)
	out = append(out, items[pos+1:]...)
	return out
}

// noteContextItemsFromLines classifies every free-form line of a phase
// relevant_context submission as a NOTE (never an executable skill/command), so
// prose can no longer silently become a bad `prompt-manager skill read ...` argv.
// Executable setup context must flow through typed context-submit, which carries
// an explicit kind/command. A NO_CONTEXT: line is preserved verbatim so the
// no-context checkpoint still recognizes it.
func noteContextItemsFromLines(content, phaseID string) []planmodel.RelevantContextItem {
	var out []planmodel.RelevantContextItem
	for _, line := range splitLines(content) {
		item := planmodel.RelevantContextItem{
			ID:      uuid.NewString(),
			Kind:    planmodel.RelevantContextNote,
			Scope:   planmodel.RelevantContextScopePhase,
			PhaseID: phaseID,
			Label:   line,
			// No canned Reason: a free-form note's line IS its content; a
			// placeholder reason only adds rendered boilerplate.
			Instruction:  line,
			Required:     false,
			RepeatPolicy: planmodel.RelevantContextPhaseEntry,
			Source:       planmodel.RelevantContextSourceAuthored,
			Status:       planmodel.RelevantContextStatusReady,
		}
		out = append(out, item)
	}
	return out
}

func contextItemViolations(item planmodel.RelevantContextItem) []StructureViolation {
	var out []StructureViolation
	add := func(msg string) {
		out = append(out, StructureViolation{SectionKey: SectionRelevantContext, Message: msg})
	}
	if item.Kind == "" {
		add("relevant context kind is required")
	}
	if item.Scope == planmodel.RelevantContextScopePhase && strings.TrimSpace(item.PhaseID) == "" {
		add("phase-scoped relevant context requires a phase id")
	}
	if item.Required && item.RepeatPolicy == "" {
		add("required relevant context requires a repeat policy")
	}
	switch item.Kind {
	case planmodel.RelevantContextCommand, planmodel.RelevantContextSearch:
		if strings.TrimSpace(item.Command) == "" && len(item.Argv) == 0 {
			add("command/search context requires command or argv")
		}
		if strings.TrimSpace(item.Instruction) == "" {
			add("command/search context requires an instruction")
		}
		if strings.TrimSpace(item.Reason) == "" {
			add("command/search context requires a reason")
		}
	case planmodel.RelevantContextSkill, planmodel.RelevantContextDoc, planmodel.RelevantContextCodeRef, planmodel.RelevantContextReqRef:
		if strings.TrimSpace(item.Target) == "" && strings.TrimSpace(item.Command) == "" && len(item.Argv) == 0 {
			add("reference context requires a target, command, or argv")
		}
		// A code_ref/doc context item whose target obviously belongs to the other
		// kind is the same docs-as-CODE mistake at context scope; reject it so the
		// rendered plan never mislabels a setup reference.
		if msg := contextItemKindMismatch(item); msg != "" {
			add(msg)
		}
	case planmodel.RelevantContextNote:
		if strings.TrimSpace(item.Instruction) == "" && strings.TrimSpace(item.Reason) == "" {
			add("note context requires an instruction or reason")
		}
	}
	return out
}

func contextItemsFromLines(content string, scope planmodel.RelevantContextScope, phaseID string) []planmodel.RelevantContextItem {
	var out []planmodel.RelevantContextItem
	for _, line := range splitLines(content) {
		item := migratedContextItem(line, scope, phaseID)
		if item.Label != "" || item.Target != "" || item.Command != "" || item.Instruction != "" {
			out = append(out, item)
		}
	}
	return out
}

func contextItemsFromRequiredReading(lines []string, phaseID string) []planmodel.RelevantContextItem {
	out := make([]planmodel.RelevantContextItem, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, migratedContextItem(line, planmodel.RelevantContextScopePhase, phaseID))
	}
	return out
}

func migratedContextItem(line string, scope planmodel.RelevantContextScope, phaseID string) planmodel.RelevantContextItem {
	item := planmodel.RelevantContextItemFromSetupLine(line, scope, phaseID, "Migrated from required-reading authoring input.")
	item.ID = uuid.NewString()
	return item
}

func renderContextItems(items []planmodel.RelevantContextItem) string {
	var b strings.Builder
	for _, item := range items {
		fmt.Fprintf(&b, "- %s\n", renderContextItemSummary(item))
	}
	return strings.TrimSpace(b.String())
}

func renderContextItemSummary(item planmodel.RelevantContextItem) string {
	label := firstNonEmpty(item.Label, item.Target, item.Command, item.Instruction, string(item.Kind))
	return fmt.Sprintf("%s: %s", item.Kind, label)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func appendNoCodeRefsReason(constraints, reason string) string {
	line := "NO_CODE_REFS: " + strings.TrimSpace(reason)
	if strings.TrimSpace(constraints) == "" {
		return line
	}
	return strings.TrimRight(constraints, "\n") + "\n" + line
}

// parseReferencesAndPhases recovers the structured references[] and phases[] from
// the authored references/phases section markup via the plans-domain parser (the
// SSOT for the [CODE:]/[REQ:]/[DOC:] + `### Phase N — Title` grammar). It feeds a
// minimal markdown view — a synthetic title (so the parser accepts it) plus the
// references and phases sections — and returns only the recovered structured
// lists; the prose fields are taken verbatim by the caller. Because these
// sections are machine-readable, non-empty markup that cannot be parsed is a
// typed authoring error, never an empty-list degradation.
func parseReferencesAndPhases(sess Session) (planmodel.Plan, error) {
	var b strings.Builder
	b.WriteString("# ")
	b.WriteString(sess.Title)
	b.WriteString("\n\n")
	refsContent := strings.TrimSpace(contentOf(sess.Sections, SectionReferences))
	phasesContent := strings.TrimSpace(contentOf(sess.Sections, SectionPhases))
	refsOnlyExplainsNoCode := refsContent != "" && noCodeRefsReason(refsContent) != "" && !hasReferenceMarker(refsContent)
	if refsContent != "" && !refsOnlyExplainsNoCode {
		b.WriteString("## References\n\n")
		b.WriteString(refsContent)
		b.WriteString("\n\n")
	}
	if phasesContent != "" {
		b.WriteString("## Phases\n\n")
		b.WriteString(phasesContent)
		b.WriteString("\n")
	}
	parsed, err := planmodel.ParsePlanMarkdown(b.String())
	if err != nil {
		return planmodel.Plan{}, ErrAuthoredMarkup{SectionKey: markupSectionForError(refsContent, phasesContent, err.Error()), Reason: err.Error()}
	}
	if refsContent != "" && !refsOnlyExplainsNoCode && len(parsed.References) == 0 {
		return planmodel.Plan{}, ErrAuthoredMarkup{SectionKey: SectionReferences, Reason: "expected at least one [CODE:], [REQ:], or [DOC:] reference"}
	}
	if phasesContent != "" && len(parsed.Phases) == 0 {
		return planmodel.Plan{}, ErrAuthoredMarkup{SectionKey: SectionPhases, Reason: "expected at least one '### Phase N - Title' heading"}
	}
	return parsed, nil
}

func markupSectionForError(refsContent, phasesContent, reason string) SectionKey {
	if phasesContent != "" && strings.Contains(reason, "phase") {
		return SectionPhases
	}
	if refsContent != "" && strings.Contains(reason, "reference") {
		return SectionReferences
	}
	if phasesContent != "" {
		return SectionPhases
	}
	return SectionPhases
}

// decisionsFromContent parses the authored decisions section: one pinned
// plan-time decision per line as `<title>: <statement>` (a leading `- ` or
// `D<n> —` label is tolerated). Lines without a separator are kept as a
// statement-only decision so nothing is silently dropped; the decisions gate
// rejects them at submit time with an actionable message.
func decisionsFromContent(content string) []planmodel.PlanDecision {
	var out []planmodel.PlanDecision
	for _, line := range splitLines(content) {
		line = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "-"))
		line = decisionLabelPrefixRe.ReplaceAllString(line, "")
		if line == "" {
			continue
		}
		title, statement, found := strings.Cut(line, ":")
		if !found {
			out = append(out, planmodel.PlanDecision{Statement: strings.TrimSpace(line)})
			continue
		}
		out = append(out, planmodel.PlanDecision{Title: strings.TrimSpace(title), Statement: strings.TrimSpace(statement)})
	}
	return out
}

func definitionsFromContent(content string) []planmodel.PlanDefinition {
	var out []planmodel.PlanDefinition
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "-"))
		if line == "" {
			continue
		}
		term, meaning, found := cutDefinitionSeparator(line)
		if found && term != "" && meaning != "" {
			out = append(out, planmodel.PlanDefinition{Term: term, Meaning: meaning})
		}
	}
	return out
}

func cutDefinitionSeparator(line string) (string, string, bool) {
	for _, separator := range []string{"—", ":"} {
		if term, meaning, found := strings.Cut(line, separator); found {
			return strings.TrimSpace(term), strings.TrimSpace(meaning), true
		}
	}
	return line, "", false
}

// decisionLabelPrefixRe strips an explicit `D<n> —`/`D<n>:` label so authored
// numbering never fights the render-time D1..Dn ordering.
var decisionLabelPrefixRe = regexp.MustCompile(`^\*{0,2}D\d+\*{0,2}\s*[—:-]\s*`)

// splitAssumptionsContent separates the assumptions section into prose
// assumptions and structured assumption/mitigation pairs: a line containing an
// `->` (or `→`) separator becomes a table row ("if wrong -> then"), everything
// else stays prose.
func splitAssumptionsContent(content string) (string, []planmodel.PlanAssumption) {
	var prose []string
	var structured []planmodel.PlanAssumption
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "-"))
		statement, mitigation, found := cutAssumptionSeparator(line)
		if !found || statement == "" {
			if strings.TrimSpace(raw) != "" {
				prose = append(prose, raw)
			}
			continue
		}
		structured = append(structured, planmodel.PlanAssumption{Statement: statement, Mitigation: mitigation})
	}
	return strings.Join(prose, "\n"), structured
}

func cutAssumptionSeparator(line string) (string, string, bool) {
	for _, sep := range []string{"->", "→"} {
		if before, after, found := strings.Cut(line, sep); found {
			return strings.TrimSpace(before), strings.TrimSpace(after), true
		}
	}
	return line, "", false
}
