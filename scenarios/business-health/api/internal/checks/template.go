package checks

import (
	"context"

	"business-health/internal/extraction"

	intent "intent-go"
)

// prdPresenceCheck emits prd_missing_prd when the scenario has no PRD.md.
type prdPresenceCheck struct{}

func (prdPresenceCheck) Name() string { return "prd-presence" }

func (prdPresenceCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if c.PRDPresent {
		return nil
	}
	return []intent.Finding{{
		Code:       "prd_missing_prd",
		Severity:   "error",
		Message:    "PRD.md is missing — the scenario states no product intent.",
		Suggestion: "Author a PRD with `business-health wizard " + c.Scenario + "` (interview-driven, conformant by construction).",
		Locations:  []string{"PRD.md"},
		Provenance: "business-health",
	}}
}

// templateCheck runs the intent-go template checks (sections, unexpected,
// content, OT id format) against the canonical template.
type templateCheck struct {
	template []intent.TemplateSection
}

func newTemplateCheck() templateCheck {
	return templateCheck{template: intent.DefaultPRDTemplate()}
}

func (templateCheck) Name() string { return "prd-template" }

func (t templateCheck) Run(_ context.Context, c extraction.Contract) []intent.Finding {
	if !c.PRDDoc.Present {
		return nil // prd_missing_prd covers absence
	}
	var out []intent.Finding
	out = append(out, intent.CheckTemplateSections(c.PRDDoc, t.template)...)
	out = append(out, intent.CheckUnexpectedSections(c.PRDDoc, t.template)...)
	out = append(out, intent.CheckTemplateContent(c.PRDDoc, t.template)...)
	out = append(out, intent.CheckOTIDFormat(c.PRDDoc)...)
	return out
}
