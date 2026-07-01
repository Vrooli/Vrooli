package plans_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"plan-manager/internal/plans"
)

// TestPlanModelDocNamesRenderSections is a drift guard between the renderer and
// PLAN-MODEL.md: every section heading the renderer emits for a comprehensive
// plan must also be named in the canonical doc. If the renderer adds/renames a
// section without updating the doc (or vice versa), this fails.
func TestPlanModelDocNamesRenderSections(t *testing.T) {
	docPath := filepath.Join("..", "..", "..", "docs", "concepts", "PLAN-MODEL.md")
	raw, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("read PLAN-MODEL.md: %v", err)
	}
	// Compare case-insensitively: the doc names sections in heading case (## Work
	// Posture) and in lowercase prose (the phase render-order line); both count.
	doc := strings.ToLower(string(raw))

	// The professional section names the renderer emits (and the doc must name).
	sections := []string{
		"Purpose", "Problem / Need", "Target Outcome", "Work Posture", "Scope",
		"Non-Goals", "Assumptions", "Technical Approach", "Constraints",
		"Prohibited Approaches", "Global Execution Setup", "Execution Feedback", "References",
		"Regression Anchor", "Validation Strategy", "Definition of Done",
		"Import Provenance", "Preserved Legacy Sections",
		"Affected Areas", "Ordered Steps", "Expected Outputs", "Phase Validation",
		"Acceptance Criteria", "Handoff Notes",
	}
	for _, s := range sections {
		if !strings.Contains(doc, strings.ToLower(s)) {
			t.Errorf("PLAN-MODEL.md does not name render section %q", s)
		}
	}
}

// TestRenderEmitsDocumentedSections renders a comprehensive plan and asserts the
// markdown carries the documented section headings, so the doc-sync test above is
// anchored to real renderer output (not just doc prose).
func TestRenderEmitsDocumentedSections(t *testing.T) {
	md := plans.RenderMarkdown(comprehensivePlan())
	for _, h := range []string{
		"## Purpose", "## Problem / Need", "## Target Outcome", "## Work Posture",
		"## Scope", "## Technical Approach", "## Execution Feedback", "## Validation Strategy",
		"## Definition of Done",
	} {
		if !strings.Contains(md, h) {
			t.Errorf("renderer did not emit documented heading %q", h)
		}
	}
}
