package planmodel

import (
	"errors"
	"strings"
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
	// Plan-level references are scoped to the pre-phases body; the phase's
	// [DOC:] reference belongs to the phase, not the plan (see phase.References).
	if got := len(plan.References); got != 2 {
		t.Fatalf("len(References) = %d, want 2", got)
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

func TestParsePlanMarkdownExtractsTypedRegressionAnchor(t *testing.T) {
	t.Parallel()

	plan, err := ParsePlanMarkdown("# Harden planner\n\n" +
		"## Regression Anchor\n\n" +
		"- Strategy: scenario_baseline\n" +
		"- Scenario baseline: `plan-manager` (name `plan-manager-hardening-readiness`)\n" +
		"- HEAD sha: `abc123`\n" +
		"- Capture status: requested; usable only after status validation\n" +
		"- Capture reason: test-genie unavailable\n" +
		"- Fallback: HEAD sha + allowlist diff\n" +
		"- Captured at: `2026-06-27T14:00:17Z`\n")
	if err != nil {
		t.Fatalf("ParsePlanMarkdown() error = %v", err)
	}

	anchor := plan.RegressionAnchor
	if anchor.Strategy != "scenario_baseline" || anchor.Scenario != "plan-manager" || anchor.BaselineName != "plan-manager-hardening-readiness" {
		t.Fatalf("anchor identity = %#v", anchor)
	}
	if anchor.HeadSha != "abc123" || anchor.CapturedAt != "2026-06-27T14:00:17Z" {
		t.Fatalf("anchor metadata = %#v", anchor)
	}
	if anchor.CaptureStatus != "requested; usable only after status validation" ||
		anchor.CaptureReason != "test-genie unavailable" ||
		anchor.Fallback != "HEAD sha + allowlist diff" {
		t.Fatalf("anchor capture health = %#v", anchor)
	}
	want := "git-control-tower baseline diff --scenario plan-manager --name plan-manager-hardening-readiness --wait"
	if !containsString(anchor.Commands, want) {
		t.Fatalf("anchor.Commands = %#v, want %q", anchor.Commands, want)
	}
}

func TestParseRegressionAnchorBlockMarksUnstructuredLegacyProse(t *testing.T) {
	t.Parallel()

	anchor := ParseRegressionAnchorBlock("baseline captured at HEAD abc123")
	if anchor.Strategy != "legacy_prose" || !anchor.Unavailable {
		t.Fatalf("legacy anchor = %#v, want explicit degraded legacy marker", anchor)
	}
	if anchor.BaselineName != "baseline captured at HEAD abc123" {
		t.Fatalf("BaselineName = %q", anchor.BaselineName)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
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

func TestParsePlanMarkdownRecoversRelevantContextSetup(t *testing.T) {
	t.Parallel()

	plan, err := ParsePlanMarkdown(`# Context plan

## Global Execution Setup

### Load Skills

- api-steer _(required, run on resume, discovered)_
  - Reason: API contract changes.
  - Instruction: Load API steering.
  ` + "```bash" + `
  prompt-manager skill read api-steer
  ` + "```" + `

### Run Discovery Searches

- Prior records _(as needed)_
  - Reason: Avoid repeating prior work.
  ` + "```bash" + `
  search-hub query "plan manager relevant context" --type record,doc
  ` + "```" + `

## Phases

### Phase 1 — Parser

- Intent: Preserve rendered setup.
- Status: active

**Phase Context Setup:**

### Read Docs

- docs/concepts/PLAN-MODEL.md _(required)_
  - Reason: Parse semantics live there.
  ` + "```bash" + `
  sed -n '1,220p' docs/concepts/PLAN-MODEL.md
  ` + "```" + `

### Inspect References

- [REQ: PM-CTX-001] _(required, unresolved)_
  - Status: Requirement index unavailable.

### Operator Notes

- Keep Markdown as a projection, not the source of truth.
`)
	if err != nil {
		t.Fatalf("ParsePlanMarkdown() error = %v", err)
	}

	assertGlobalRelevantContext(t, plan.RelevantContext)
	if got := len(plan.Phases); got != 1 {
		t.Fatalf("len(plan.Phases) = %d, want 1", got)
	}
	assertPhaseRelevantContext(t, plan.Phases[0].RelevantContext)
}

func assertGlobalRelevantContext(t *testing.T, context []RelevantContextItem) {
	t.Helper()

	if got := len(context); got != 2 {
		t.Fatalf("len(plan.RelevantContext) = %d, want 2", got)
	}
	skill := context[0]
	if skill.Kind != RelevantContextSkill || skill.Target != "api-steer" || !skill.Required {
		t.Fatalf("skill context = %#v", skill)
	}
	if skill.RepeatPolicy != RelevantContextOnResume || skill.Source != RelevantContextSourceDiscovered {
		t.Fatalf("skill policy/source = %#v", skill)
	}
	if skill.Reason != "API contract changes." || skill.Instruction != "Load API steering." {
		t.Fatalf("skill reason/instruction = %#v", skill)
	}
	search := context[1]
	if search.Kind != RelevantContextSearch || search.Command == "" || search.RepeatPolicy != RelevantContextAsNeeded {
		t.Fatalf("search context = %#v", search)
	}
}

func assertPhaseRelevantContext(t *testing.T, context []RelevantContextItem) {
	t.Helper()

	if got := len(context); got != 3 {
		t.Fatalf("len(phase context) = %d, want 3", got)
	}
	if context[0].Kind != RelevantContextDoc || context[0].Target != "docs/concepts/PLAN-MODEL.md" {
		t.Fatalf("doc context = %#v", context[0])
	}
	if context[0].RepeatPolicy != RelevantContextPhaseEntry || context[0].Scope != RelevantContextScopePhase {
		t.Fatalf("doc scope/policy = %#v", context[0])
	}
	if context[1].Kind != RelevantContextReqRef || context[1].Target != "PM-CTX-001" || context[1].Status != RelevantContextStatusUnresolved {
		t.Fatalf("req context = %#v", context[1])
	}
	if context[2].Kind != RelevantContextNote || context[2].Instruction == "" {
		t.Fatalf("note context = %#v", context[2])
	}
}

func TestParsePlanMarkdownMigratesLegacyRequiredReading(t *testing.T) {
	t.Parallel()

	plan, err := ParsePlanMarkdown(`# Legacy setup

## Required Reading

- prompt-manager skill read api-steer
- search-hub query "plan manager context" --type record,doc
- docs/concepts/PLAN-MODEL.md

## Phases

### Phase 1 — Implement

- Intent: Make the change.
- Status: todo

**Required Reading:**
- [REQ: PM-CTX-001]
- cli: vrooli scenario requirements validate plan-manager
`)
	if err != nil {
		t.Fatalf("ParsePlanMarkdown() error = %v", err)
	}
	if got := len(plan.RelevantContext); got != 3 {
		t.Fatalf("len(plan.RelevantContext) = %d, want 3", got)
	}
	if plan.RelevantContext[0].Kind != RelevantContextSkill ||
		plan.RelevantContext[0].Target != "api-steer" ||
		plan.RelevantContext[0].Source != RelevantContextSourceMigrated ||
		plan.RelevantContext[0].RepeatPolicy != RelevantContextOncePerExecution {
		t.Fatalf("global skill context = %#v", plan.RelevantContext[0])
	}
	if plan.RelevantContext[1].Kind != RelevantContextSearch || plan.RelevantContext[1].Command == "" {
		t.Fatalf("global search context = %#v", plan.RelevantContext[1])
	}
	if plan.RelevantContext[2].Kind != RelevantContextDoc || plan.RelevantContext[2].Target != "docs/concepts/PLAN-MODEL.md" {
		t.Fatalf("global doc context = %#v", plan.RelevantContext[2])
	}
	if got := len(plan.Phases); got != 1 {
		t.Fatalf("len(plan.Phases) = %d, want 1", got)
	}
	phase := plan.Phases[0]
	if got := len(phase.RequiredReading); got != 2 {
		t.Fatalf("len(phase.RequiredReading) = %d, want 2", got)
	}
	if got := len(phase.RelevantContext); got != 2 {
		t.Fatalf("len(phase.RelevantContext) = %d, want 2", got)
	}
	if phase.RelevantContext[0].Kind != RelevantContextReqRef ||
		phase.RelevantContext[0].Target != "PM-CTX-001" ||
		phase.RelevantContext[0].RepeatPolicy != RelevantContextPhaseEntry {
		t.Fatalf("phase req context = %#v", phase.RelevantContext[0])
	}
	if phase.RelevantContext[1].Kind != RelevantContextCommand ||
		phase.RelevantContext[1].Command != "vrooli scenario requirements validate plan-manager" {
		t.Fatalf("phase command context = %#v", phase.RelevantContext[1])
	}
}

func TestParsePlanMarkdownMigratesLegacyRequiredReadingCommandFences(t *testing.T) {
	t.Parallel()

	plan, err := ParsePlanMarkdown(`# Legacy setup fences

## Required Reading

Run these before implementation:

` + "```bash" + `
prompt-manager skill read api-steer test
sed -n '1,260p' scenarios/plan-manager/docs/concepts/PLAN-MODEL.md
` + "```" + `

Useful context:

- search-hub query "plan manager import" --type record,doc
`)
	if err != nil {
		t.Fatalf("ParsePlanMarkdown() error = %v", err)
	}
	if got := len(plan.RelevantContext); got != 3 {
		t.Fatalf("len(plan.RelevantContext) = %d, want 3: %#v", got, plan.RelevantContext)
	}
	if plan.RelevantContext[0].Kind != RelevantContextSkill || plan.RelevantContext[0].Target != "api-steer test" {
		t.Fatalf("skill context = %#v", plan.RelevantContext[0])
	}
	doc := plan.RelevantContext[1]
	if doc.Kind != RelevantContextDoc || doc.Command != "sed -n '1,260p' scenarios/plan-manager/docs/concepts/PLAN-MODEL.md" ||
		doc.Target != "scenarios/plan-manager/docs/concepts/PLAN-MODEL.md" {
		t.Fatalf("doc command context = %#v", doc)
	}
	for _, item := range plan.RelevantContext {
		if item.Label == "```bash" || item.Label == "```" || item.Label == "Run these before implementation:" {
			t.Fatalf("legacy prose/fence leaked into setup context: %#v", plan.RelevantContext)
		}
	}
}

func TestRelevantContextItemFromSetupLineExtractsDocTarget(t *testing.T) {
	item := RelevantContextItemFromSetupLine(
		"sed -n '1,220p' scenarios/plan-manager/docs/concepts/PLAN-MODEL.md",
		RelevantContextScopeGlobal,
		"",
		"test",
	)
	if item.Kind != RelevantContextDoc {
		t.Fatalf("kind = %q, want doc", item.Kind)
	}
	if item.Target != "scenarios/plan-manager/docs/concepts/PLAN-MODEL.md" {
		t.Fatalf("target = %q", item.Target)
	}
	if item.Label != item.Target {
		t.Fatalf("label = %q, want target %q", item.Label, item.Target)
	}
	if strings.Contains(item.Target, "sed -n") {
		t.Fatalf("target should not contain shell command: %q", item.Target)
	}
}

func TestParsePlanMarkdownPromotesLegacyPhaseSections(t *testing.T) {
	t.Parallel()

	plan, err := ParsePlanMarkdown(`# Legacy phase detail

## Phases

### Phase 1 — Import cleanup

Objective:
Make imported setup context executable.

Checklist:
- Parse fenced command blocks.
- Preserve useful setup commands.

Expected outputs:
- Structured relevant_context items.

Validation:
go test ./internal/planmodel

Definition of done:
Imported phases render with executable steps and validation.
`)
	if err != nil {
		t.Fatalf("ParsePlanMarkdown() error = %v", err)
	}
	if got := len(plan.Phases); got != 1 {
		t.Fatalf("len(plan.Phases) = %d, want 1", got)
	}
	phase := plan.Phases[0]
	if phase.Intent != "Make imported setup context executable." {
		t.Fatalf("phase.Intent = %q", phase.Intent)
	}
	if got := phase.Steps; len(got) != 2 || got[0] != "Parse fenced command blocks." || got[1] != "Preserve useful setup commands." {
		t.Fatalf("phase.Steps = %#v", got)
	}
	if got := phase.ExpectedOutputs; len(got) != 1 || got[0] != "Structured relevant_context items." {
		t.Fatalf("phase.ExpectedOutputs = %#v", got)
	}
	if phase.Validation != "go test ./internal/planmodel" {
		t.Fatalf("phase.Validation = %q", phase.Validation)
	}
	if phase.Acceptance != "Imported phases render with executable steps and validation." {
		t.Fatalf("phase.Acceptance = %q", phase.Acceptance)
	}
}

func TestParsePlanMarkdownRejectsMalformedRelevantContextBlock(t *testing.T) {
	t.Parallel()

	_, err := ParsePlanMarkdown(`# Bad context

## Global Execution Setup

### Run Commands

- Setup
  ` + "```bash" + `
  vrooli help
`)
	var invalid ErrInvalidPlan
	if !errors.As(err, &invalid) {
		t.Fatalf("ParsePlanMarkdown() error = %v, want ErrInvalidPlan", err)
	}
}
