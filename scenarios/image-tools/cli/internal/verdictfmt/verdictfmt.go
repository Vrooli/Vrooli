// Package verdictfmt renders model verdicts and state overrides the same way in
// every CLI command (`models list|get|select|explain`, `ai list`), so what works
// and what is not recommended reads identically wherever it appears.
package verdictfmt

import (
	"fmt"
	"strings"

	modelsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/models"
)

// Label is the plain-words grade ("not recommended").
func Label(g modelsv1.VerdictGrade) string {
	switch g {
	case modelsv1.VerdictGrade_VERDICT_GRADE_RECOMMENDED:
		return "recommended"
	case modelsv1.VerdictGrade_VERDICT_GRADE_WORKS:
		return "works"
	case modelsv1.VerdictGrade_VERDICT_GRADE_EXPERIMENTAL:
		return "experimental"
	case modelsv1.VerdictGrade_VERDICT_GRADE_NOT_RECOMMENDED:
		return "not recommended"
	case modelsv1.VerdictGrade_VERDICT_GRADE_BROKEN:
		return "broken"
	default:
		return "untested"
	}
}

// badgeWidth fits the longest label so a list of badges lines up.
const badgeWidth = len("not recommended")

// Badge is the grade padded to a fixed width, for the head of a list line.
func Badge(v *modelsv1.Verdict) string {
	return fmt.Sprintf("%-*s", badgeWidth, Label(v.GetGrade()))
}

// Line is "grade: reason" for a one-line summary.
func Line(v *modelsv1.Verdict) string {
	if strings.TrimSpace(v.GetReason()) == "" {
		return Label(v.GetGrade())
	}
	return Label(v.GetGrade()) + ": " + v.GetReason()
}

// Evidence lists the evidence pointers behind a verdict, one indented line each.
func Evidence(v *modelsv1.Verdict, indent string) []string {
	out := make([]string, 0, len(v.GetEvidence()))
	for _, e := range v.GetEvidence() {
		line := fmt.Sprintf("%s%s: %s", indent, e.GetKind(), e.GetRef())
		if e.GetDate() != "" {
			line += " (" + e.GetDate() + ")"
		}
		out = append(out, line)
	}
	return out
}

// Override renders a stored override with its provenance and its relation to the
// seed, or "" when the model has none.
func Override(o *modelsv1.StateOverride) string {
	if o == nil {
		return ""
	}
	state := "disabled"
	if o.GetEnabled() {
		state = "enabled"
	}
	line := fmt.Sprintf("override: %s by %s", state, o.GetSetBy())
	if at := o.GetSetAt(); at != "" {
		line += " at " + at
	}
	if r := o.GetReason(); r != "" {
		line += " — " + r
	}
	if o.GetPinned() {
		line += " [pinned]"
	}
	switch o.GetRelation() {
	case "contradicts_seed":
		line += fmt.Sprintf(" [contradicts the seed, which says enabled=%t]", o.GetSeedEnabled())
	case "superseded":
		line += " [superseded, not applied]"
	}
	if n := o.GetNote(); n != "" {
		line += "\n    " + n
	}
	return line
}
