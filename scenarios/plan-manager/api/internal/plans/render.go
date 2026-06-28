package plans

import (
	"fmt"
	"strings"
)

// RenderMarkdown renders a structured Plan to its human-readable markdown
// projection. The output is DETERMINISTIC — the same record always renders the
// same bytes — because the markdown is a *view*, never parsed back into truth
// (see docs/concepts/PLAN-MODEL.md). Field order is fixed; empty optional
// sections are omitted so the view stays readable, but the section ordering for
// present fields never varies.
func RenderMarkdown(p Plan) string {
	var b strings.Builder

	title := p.Title
	if title == "" {
		title = p.Slug
	}
	fmt.Fprintf(&b, "# %s\n\n", title)
	fmt.Fprintf(&b, "> Status: **%s**", statusLabel(p.Status))
	if p.ContentHash != "" {
		fmt.Fprintf(&b, " · content-hash `%s`", shortHash(p.ContentHash))
	}
	b.WriteString("\n\n")

	// Overview.
	writeSection(&b, "Purpose", p.Purpose)
	writeSection(&b, "Problem / Need", p.ProblemStatement)
	writeSection(&b, "Target Outcome", p.TargetOutcome)

	// Work Posture — ALWAYS rendered (autofilled; default greenfield).
	b.WriteString("## Work Posture\n\n")
	b.WriteString(renderWorkPosture(p))
	b.WriteString("\n")

	writeSection(&b, "Scope", p.Scope)
	writeSection(&b, "Non-Goals", p.NonGoals)
	writeSection(&b, "Assumptions", p.Assumptions)

	// Execution Model.
	writeSection(&b, "Technical Approach", p.TechnicalApproach)
	writeSection(&b, "Constraints", p.Constraints)
	writeSection(&b, "Prohibited Approaches", p.ProhibitedApproaches)

	if len(p.RelevantContext) > 0 {
		b.WriteString("## Global Execution Setup\n\n")
		b.WriteString(renderRelevantContext(p.RelevantContext, RelevantContextScopeGlobal))
		b.WriteString("\n")
	}

	if len(p.References) > 0 {
		b.WriteString("## References\n\n")
		for _, ref := range p.References {
			b.WriteString(renderReference(ref))
		}
		b.WriteString("\n")
	}

	// Validation Model.
	if anchorPresent(p.RegressionAnchor) {
		b.WriteString("## Regression Anchor\n\n")
		b.WriteString(renderAnchor(p.RegressionAnchor))
		b.WriteString("\n")
	}

	if strings.TrimSpace(p.ValidationStrategy) != "" || len(p.FinalValidationCommands) > 0 {
		b.WriteString("## Validation Strategy\n\n")
		if strings.TrimSpace(p.ValidationStrategy) != "" {
			b.WriteString(strings.TrimRight(p.ValidationStrategy, "\n"))
			b.WriteString("\n")
		}
		if len(p.FinalValidationCommands) > 0 {
			b.WriteString("\n**Final validation commands:**\n")
			for _, c := range p.FinalValidationCommands {
				fmt.Fprintf(&b, "- `%s`\n", c)
			}
		}
		b.WriteString("\n")
	}

	writeSection(&b, "Definition of Done", p.DefinitionOfDone)

	writeSection(&b, "Risks / Hazards", p.RisksHazards)

	if len(p.Phases) > 0 {
		b.WriteString("## Phases\n\n")
		for i, ph := range p.Phases {
			b.WriteString(renderPhase(ph, i+1))
		}
	}

	// Governance: import provenance + preserved legacy sections (only when present).
	b.WriteString(renderImportGovernance(p))

	// Plan-graph edges as a trailing footnote when present.
	if len(p.Supersedes) > 0 || len(p.SupersededBy) > 0 {
		b.WriteString("## Plan Graph\n\n")
		if len(p.Supersedes) > 0 {
			fmt.Fprintf(&b, "- Supersedes: %s\n", strings.Join(p.Supersedes, ", "))
		}
		if len(p.SupersededBy) > 0 {
			fmt.Fprintf(&b, "- Superseded by: %s\n", strings.Join(p.SupersededBy, ", "))
		}
		b.WriteString("\n")
	}

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
	context := phaseRelevantContext(ph)
	if len(context) > 0 {
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
		items = append(items, RelevantContextItem{
			ID:           fmt.Sprintf("%s-required-reading-%d", ph.ID, i+1),
			Kind:         migratedReadingKind(raw),
			Scope:        RelevantContextScopePhase,
			PhaseID:      ph.ID,
			Label:        raw,
			Instruction:  raw,
			Target:       raw,
			Required:     true,
			RepeatPolicy: RelevantContextPhaseEntry,
			Source:       RelevantContextSourceMigrated,
			Status:       RelevantContextStatusReady,
		})
	}
	return items
}

func migratedReadingKind(raw string) RelevantContextKind {
	switch {
	case strings.HasPrefix(raw, "prompt-manager skill read"):
		return RelevantContextSkill
	case strings.HasPrefix(raw, "search-hub query"):
		return RelevantContextSearch
	case strings.HasPrefix(raw, "cli:"):
		return RelevantContextCommand
	case strings.Contains(raw, ".md") || strings.HasPrefix(raw, "docs/"):
		return RelevantContextDoc
	default:
		return RelevantContextNote
	}
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
	fmt.Fprintf(&b, "- %s", label)
	annotations := relevantContextAnnotations(item)
	if len(annotations) > 0 {
		fmt.Fprintf(&b, " _(%s)_", strings.Join(annotations, ", "))
	}
	b.WriteString("\n")
	if item.Reason != "" {
		fmt.Fprintf(&b, "  - Reason: %s\n", item.Reason)
	}
	if item.Instruction != "" && item.Instruction != label {
		fmt.Fprintf(&b, "  - Instruction: %s\n", item.Instruction)
	}
	command := relevantContextCommand(item)
	if command != "" {
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

func relevantContextCommand(item RelevantContextItem) string {
	if item.Command != "" {
		return item.Command
	}
	if len(item.Argv) > 0 {
		return strings.Join(item.Argv, " ")
	}
	switch item.Kind {
	case RelevantContextSkill:
		if item.Target != "" {
			return "prompt-manager skill read " + item.Target
		}
	case RelevantContextDoc, RelevantContextCodeRef, RelevantContextReqRef:
		if item.Target != "" {
			return "sed -n '1,220p' " + item.Target
		}
	case RelevantContextSearch:
		if item.Target != "" {
			return item.Target
		}
	}
	return ""
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

func renderAnchor(a RegressionAnchor) string {
	var b strings.Builder
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
	}
	if a.Scenario == "" && a.BaselineName != "" {
		fmt.Fprintf(&b, "- Baseline name: `%s`\n", a.BaselineName)
	}
	if a.HeadSha != "" {
		fmt.Fprintf(&b, "- HEAD sha: `%s`\n", a.HeadSha)
	}
	if len(a.AllowlistPaths) > 0 {
		fmt.Fprintf(&b, "- Allowlist: %s\n", strings.Join(a.AllowlistPaths, ", "))
	}
	if a.CapturedAt != "" {
		fmt.Fprintf(&b, "- Captured at: `%s`\n", a.CapturedAt)
	}
	for _, c := range a.Commands {
		fmt.Fprintf(&b, "- `%s`\n", c)
	}
	return b.String()
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
