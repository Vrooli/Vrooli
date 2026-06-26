package planmodel

import (
	"errors"
	"testing"
)

func TestParsePlanMarkdownExtractsStructuredPlan(t *testing.T) {
	t.Parallel()

	plan, err := ParsePlanMarkdown(`# Migrate auth

## Purpose
Move auth to Connect.

## Scope
Plan-manager only.

## References
- [CODE: api/main.go]
- [REQ: PM-PLAN-001]
- [CODE: api/main.go]

## Definition of Done
All tests pass.

### Phase 1 - Contracts
- Intent: Define proto
- Acceptance: Generated clients compile
- Status: active
- [DOC: docs/concepts/PLAN-MODEL.md]
`)
	if err != nil {
		t.Fatalf("ParsePlanMarkdown() error = %v", err)
	}

	if plan.Title != "Migrate auth" {
		t.Fatalf("Title = %q, want Migrate auth", plan.Title)
	}
	if plan.Purpose != "Move auth to Connect." {
		t.Fatalf("Purpose = %q", plan.Purpose)
	}
	if plan.Scope != "Plan-manager only." {
		t.Fatalf("Scope = %q", plan.Scope)
	}
	if plan.DefinitionOfDone != "All tests pass." {
		t.Fatalf("DefinitionOfDone = %q", plan.DefinitionOfDone)
	}
	if got := len(plan.References); got != 3 {
		t.Fatalf("len(References) = %d, want 3", got)
	}
	if plan.References[0].Kind != ReferenceCode || plan.References[0].Target != "api/main.go" {
		t.Fatalf("first reference = %#v", plan.References[0])
	}
	if plan.References[1].Kind != ReferenceReq || plan.References[1].Target != "PM-PLAN-001" {
		t.Fatalf("second reference = %#v", plan.References[1])
	}
	if got := len(plan.Phases); got != 1 {
		t.Fatalf("len(Phases) = %d, want 1", got)
	}
	phase := plan.Phases[0]
	if phase.Order != 1 || phase.Title != "Contracts" || phase.Status != PhaseStatusActive {
		t.Fatalf("phase identity/status = %#v", phase)
	}
	if phase.Intent != "Define proto" || phase.Acceptance != "Generated clients compile" {
		t.Fatalf("phase authored fields = %#v", phase)
	}
	if got := len(phase.References); got != 1 {
		t.Fatalf("len(phase.References) = %d, want 1", got)
	}
	if phase.References[0].Kind != ReferenceDoc || phase.References[0].Target != "docs/concepts/PLAN-MODEL.md" {
		t.Fatalf("phase reference = %#v", phase.References[0])
	}
}

func TestParsePlanMarkdownRejectsMalformedMachineReadableMarkup(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		markdown string
	}{
		{
			name: "reference",
			markdown: `# Bad

## References
- [CODE:]
`,
		},
		{
			name: "phase",
			markdown: `# Bad

### Phase one
`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParsePlanMarkdown(tc.markdown)
			var invalid ErrInvalidPlan
			if !errors.As(err, &invalid) {
				t.Fatalf("ParsePlanMarkdown() error = %v, want ErrInvalidPlan", err)
			}
		})
	}
}
