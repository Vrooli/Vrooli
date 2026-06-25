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

	writeSection(&b, "Purpose", p.Purpose)
	writeSection(&b, "Scope", p.Scope)
	writeSection(&b, "Constraints", p.Constraints)
	writeSection(&b, "Non-Goals", p.NonGoals)

	if len(p.References) > 0 {
		b.WriteString("## References\n\n")
		for _, ref := range p.References {
			b.WriteString(renderReference(ref))
		}
		b.WriteString("\n")
	}

	if anchorPresent(p.RegressionAnchor) {
		b.WriteString("## Regression Anchor\n\n")
		b.WriteString(renderAnchor(p.RegressionAnchor))
		b.WriteString("\n")
	}

	writeSection(&b, "Definition of Done", p.DefinitionOfDone)

	if len(p.Phases) > 0 {
		b.WriteString("## Phases\n\n")
		for i, ph := range p.Phases {
			b.WriteString(renderPhase(ph, i+1))
		}
	}

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
	if ph.Acceptance != "" {
		fmt.Fprintf(&b, "- Acceptance: %s\n", ph.Acceptance)
	}
	b.WriteString("\n")
	if len(ph.RequiredReading) > 0 {
		b.WriteString("**Required reading:**\n")
		for _, r := range ph.RequiredReading {
			fmt.Fprintf(&b, "- %s\n", r)
		}
		b.WriteString("\n")
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
	if a.HeadSha != "" {
		fmt.Fprintf(&b, "- HEAD sha: `%s`\n", a.HeadSha)
	}
	if len(a.AllowlistPaths) > 0 {
		fmt.Fprintf(&b, "- Allowlist: %s\n", strings.Join(a.AllowlistPaths, ", "))
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
