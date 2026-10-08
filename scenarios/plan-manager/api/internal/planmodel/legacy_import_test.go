package planmodel

import (
	"strings"
	"testing"
)

const legacyPlanMarkdown = `# Legacy Implementation Plan

## Purpose
Make the thing work.

## Problem Statement
The legacy thing is broken and nobody can review it.

## Scope
In scope: the fix. Out of scope: rewrites.

## Target End State
The thing works and is reviewable.

## Implementation Strategy
Build it as a vertical slice, then validate.

### Phase 1 — Build
- Intent: Build the slice.
- Acceptance: It compiles.

## Testing Plan
Run the unit suite and the scenario test.

## Risks and Mitigations
It might regress; compare against the baseline.

## Contract Decisions
The API stays REST for now.

## Rollout / Validation Checklist
- [ ] ship it
`

// TestLegacyImportMapsAndPreserves asserts a legacy 13-section plan maps its
// recognized sections into canonical fields and preserves every unmapped section
// so nothing is silently dropped.
func TestLegacyImportMapsAndPreserves(t *testing.T) {
	p, err := ParsePlanMarkdown(legacyPlanMarkdown)
	if err != nil {
		t.Fatalf("parse legacy plan: %v", err)
	}

	// Mapped sections become first-class fields.
	checks := map[string]string{
		"problem_statement":   p.ProblemStatement,
		"target_outcome":      p.TargetOutcome,
		"technical_approach":  p.TechnicalApproach,
		"validation_strategy": p.ValidationStrategy,
		"risks_hazards":       p.RisksHazards,
	}
	wants := map[string]string{
		"problem_statement":   "The legacy thing is broken and nobody can review it.",
		"target_outcome":      "The thing works and is reviewable.",
		"technical_approach":  "Build it as a vertical slice, then validate.",
		"validation_strategy": "Run the unit suite and the scenario test.",
		"risks_hazards":       "It might regress; compare against the baseline.",
	}
	for field, got := range checks {
		if got != wants[field] {
			t.Errorf("%s = %q, want %q", field, got, wants[field])
		}
	}

	if len(p.Phases) != 1 || p.Phases[0].Title != "Build" {
		t.Fatalf("phases = %+v", p.Phases)
	}

	// Unmapped sections are preserved verbatim (never dropped).
	preserved := map[string]string{}
	for _, sec := range p.PreservedLegacySections {
		preserved[sec.Heading] = sec.Content
		if sec.PreservationReason != PreservationReasonUnmapped {
			t.Errorf("preserved %q reason = %q, want %q", sec.Heading, sec.PreservationReason, PreservationReasonUnmapped)
		}
	}
	for _, heading := range []string{"Contract Decisions", "Rollout / Validation Checklist"} {
		if _, ok := preserved[heading]; !ok {
			t.Errorf("legacy section %q was not preserved; have %v", heading, keys(preserved))
		}
	}
	if !strings.Contains(preserved["Contract Decisions"], "stays REST") {
		t.Errorf("preserved Contract Decisions content lost: %q", preserved["Contract Decisions"])
	}
}

func TestLegacyImportMapsNumberedImplementationPlanHeadings(t *testing.T) {
	p, err := ParsePlanMarkdown(`# Numbered Legacy Plan

## 1. Purpose
Make the thing work.

## 3. Problem Statement
The legacy thing is broken and nobody can review it.

## 6. Target End State
The thing works and is reviewable.

## 7. Implementation Strategy
Build it as a vertical slice, then validate.

### Phase 1 — Build
- Intent: Build the slice.
- Acceptance: It compiles.

## 9. Testing Plan
Run the unit suite and the scenario test.

## 11. Risks + Mitigations
It might regress; compare against the baseline.

## 13. Definition of Done
All validation passes.
`)
	if err != nil {
		t.Fatalf("parse numbered legacy plan: %v", err)
	}

	checks := map[string]string{
		"purpose":             p.Purpose,
		"problem_statement":   p.ProblemStatement,
		"target_outcome":      p.TargetOutcome,
		"technical_approach":  p.TechnicalApproach,
		"validation_strategy": p.ValidationStrategy,
		"risks_hazards":       p.RisksHazards,
		"definition_of_done":  p.DefinitionOfDone,
	}
	wants := map[string]string{
		"purpose":             "Make the thing work.",
		"problem_statement":   "The legacy thing is broken and nobody can review it.",
		"target_outcome":      "The thing works and is reviewable.",
		"technical_approach":  "Build it as a vertical slice, then validate.",
		"validation_strategy": "Run the unit suite and the scenario test.",
		"risks_hazards":       "It might regress; compare against the baseline.",
		"definition_of_done":  "All validation passes.",
	}
	for field, got := range checks {
		if got != wants[field] {
			t.Errorf("%s = %q, want %q", field, got, wants[field])
		}
	}
	if len(p.Phases) != 1 || p.Phases[0].Title != "Build" {
		t.Fatalf("phases = %+v", p.Phases)
	}
	if len(p.PreservedLegacySections) != 0 {
		t.Fatalf("numbered canonical legacy sections should not be preserved: %+v", p.PreservedLegacySections)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
