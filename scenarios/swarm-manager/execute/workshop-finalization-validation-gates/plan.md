# Plan: Add Pre and Post Finalization Validation Gates for Plan Quality

## Purpose

Ensure that plan.md files produced during workshop finalization contain all 13 mandatory sections and required structural elements before and after the finalize agent runs. Pre-finalization validation informs the finalize agent of gaps; post-finalization validation gates auto-queue if the plan is still incomplete.

## Required Reading

```bash
prompt-manager skill read implementation-plan-authoring seam-discovery-and-enforcement swarm-manager-workshop-finalize
```

## Greenfield Declaration

This is greenfield work. No compatibility shims or backwards-compatibility hacks. New validation logic is added as a new file; existing files are modified only at well-defined insertion points.

## Problem Statement

When backlog items reach finalization, the finalize agent sometimes produces plan.md files missing critical sections (Required Reading, Testing Plan, etc.), lacking `prompt-manager skill read` commands, or (for idea items) missing scenario template usage instructions. This was discovered when a vrooli-events idea item was finalized and executed — the agent built a scenario from scratch without using templates, omitted the UI, and produced misaligned PRDs. The root cause: no programmatic checks on plan.md content quality.

### Evidence

- Pre-finalization validation in `api/internal/backlog/research.go:272-296` only checks: round exists, decisions answered, readiness >= 3, synthesis pending
- No structural validation of plan.md content exists anywhere in the codebase
- The 13 mandatory sections are defined in the `implementation-plan-authoring` skill but never enforced programmatically

## Scope

### In Scope
- New `plan_validate.go` file in `api/internal/backlog/` with plan content validation logic
- Pre-finalization gate: call validation before spawning finalize agent, pass gaps as context
- Post-finalization gate: re-validate after finalize round is written, store results as `validation-report.json`
- Block auto-queue when validation fails (check in queue handler via `BlockingReason`, forceable)
- New `plan_validation` field on backlog item GET response
- New `GET /api/v1/backlog/{kind}/{name}/validation` endpoint
- Validation status on backlog list endpoint for filtering
- UI: validation badge on item cards, detailed report on item detail page, filter by validation status
- Skip validation for research items (they use conclusion.md)

### Out of Scope
- Changing the finalize agent's prompt or skill content
- Modifying the workshop round schema beyond adding a `plan_validation` field
- Blocking finalization — pre-gate is advisory only
- Validating conclusion.md structure for research items

### Acceptance Allow
```
scenarios/swarm-manager/api/internal/backlog/**
scenarios/swarm-manager/api/internal/workshop/**
scenarios/swarm-manager/ui/src/**
```

### Acceptance Deny
```
scenarios/swarm-manager/api/internal/execution/**
```

## Current Technical Context

### Finalization Flow
1. `WorkshopSave()` in `workshop_save.go` saves round responses
2. Auto-advance logic (`ShouldAutoAdvance`) decides whether to trigger finalize
3. `spawnWorkshopAsync(item, ResearchModeFinalize)` spawns the finalize agent
4. Pre-finalization checks in `research.go:272-296`: round exists, decisions answered, readiness >= 3, synthesis pending
5. Finalize agent runs, writes a finalize round with `mode: "finalize"`, zero decisions, info items only

### Key File-Reading Utilities
- `workshop.LoadPlanContent(itemDir)` — reads plan.md from item directory, returns empty string if missing
- `workshop.LoadPlanContentByName(itemDir, filename)` — reads named deliverable file
- `workshop.HasPlanByName(itemDir, filename)` — checks if deliverable file exists
- `DeliverableForKind(kind)` — returns "conclusion.md" for research, "plan.md" for others

### Prompt Assembly (research.go:310-342)
- `fetchResearchPrompt()` resolves skill by mode+kind, builds variable map
- Additional user context appended as `"\n\nAdditional context from user:\n" + userPrompt`
- Attached files and archive targets appended after
- **Pre-finalization gap report will be injected similarly** as additional context

### Queue Handler Validation (queue_ops.go:36-245)
- `isQueueableItem()` checks allowed statuses
- `executionService.ProcessPreflight()` returns blocking reasons
- `CountPendingDecisions()` adds forceable blocking reason
- `EvaluateDependencyBlocking()` adds dependency reasons
- `DedupeReasons()` deduplicates
- If `force` flag set and all reasons are forceable → proceed anyway

### 13 Mandatory Sections
1. Purpose
2. Required Reading
3. Problem Statement
4. Scope
5. Current Technical Context
6. Target End State
7. Implementation Strategy
8. Contract Decisions
9. Testing Plan
10. Rollout/Validation Checklist
11. Risks + Mitigations
12. Non-goals/Prohibited Patterns
13. Definition of Done

### Additional Required Elements
- `prompt-manager skill read` command in Required Reading
- Greenfield declaration
- Final cleanup step with `vrooli scenario restart`
- For idea items: `scenario-generation` skill reference and `vrooli scenario create` template usage
- For idea items: UI section or UI template mention

### UI Architecture
- **Backlog detail page:** `ui/src/pages/BacklogDetailsPage.tsx` — tabbed layout (info, prompt, files, output, activity), data via `useBacklogDetailData` hook
- **Backlog card:** `ui/src/components/backlog/backlog-card.tsx` — already has badge system (ScenarioBadge, AgentRunningBadge, CircuitBrokenBadge, PendingDecisionBadge, etc.)
- **Existing badge pattern:** badges are small indicator components rendered in the card header area

## Target End State

After implementation:
1. Every finalization attempt runs plan content validation and passes gap info to the finalize agent
2. After finalization completes, the same validation runs again and stores results in `validation-report.json`
3. Items with failed post-finalization validation cannot auto-queue — validation failure appears as a **forceable** `BlockingReason` in the queue handler, so users can override with `force` flag
4. The API exposes validation status on GET endpoints (list + detail + dedicated validation endpoint)
5. The UI shows validation badges (using existing badge pattern) and detailed reports
6. Research items are excluded from plan validation entirely

## Implementation Strategy

### Phase 1: Core Validation Logic (`plan_validate.go`)
Create `api/internal/backlog/plan_validate.go` with:
- `PlanValidationResult` struct: `SectionsPresent []string`, `SectionsMissing []string`, `Warnings []string`, `Passed bool`
- `ValidatePlanCompleteness(planContent string, kind BacklogKind) PlanValidationResult` — uses normalized keyword matching (lowercase header text, strip numbering/punctuation, check for key phrases like "required reading", "problem statement", etc.)
- Section header matching: extract all `## ` lines, normalize to lowercase, strip leading numbers/dots, then check against canonical keyword list
- Additional checks: `prompt-manager skill read` presence, greenfield declaration, `vrooli scenario restart`
- Idea-specific checks: `scenario-generation` or `vrooli scenario create` reference, UI mention
- Return structured result with all findings

### Phase 2: Pre-Finalization Gate
In `research.go`, after the existing pre-finalization checks (line ~296) and before `spawnWorkshopAsync`:
- Load plan.md via `workshop.LoadPlanContent(itemDir)`
- Call `ValidatePlanCompleteness(content, kind)`
- If gaps found, format as structured markdown checklist and append to the prompt as additional context (same pattern as `req.Prompt` injection)
- If plan.md doesn't exist, include that as a gap ("plan.md does not exist yet — create from scratch")
- This does NOT block finalization — it informs the agent

### Phase 3: Post-Finalization Gate
In `workshop_save.go`, detect when a finalize round is saved (check `round.Mode == "finalize"`):
- Re-load and re-validate plan.md
- Write `validation-report.json` to the item directory via the file service
- Structure: `{"sections_present": [...], "sections_missing": [...], "warnings": [...], "passed": bool, "validated_at": "ISO-8601"}`

### Phase 4: Queue Blocking Integration
In `queue_ops.go`, after existing blocking reason evaluation:
- Load `validation-report.json` if it exists
- If `Passed == false`, append a **forceable** `BlockingReason` with code like `"plan_validation_failed"` and a message listing missing sections
- This allows manual override with `force` flag, matching existing queue behavior

### Phase 5: API Surface
- Add `plan_validation` object to `backlogToProto()` in handler_query.go — read from `validation-report.json` if it exists
- Add `GET /api/v1/backlog/{kind}/{name}/validation` endpoint returning detailed `PlanValidationResult`
- Add `validation_status` filter parameter to the list endpoint (passed/failed/none)

### Phase 6: UI Components
- **ValidationBadge:** small badge component following existing pattern (green check / yellow warning / red X), rendered alongside other badges in `backlog-card.tsx`
- **ValidationReport:** detailed panel on item detail page (info tab), showing sections present/missing and warnings
- **List filter:** add validation status filter chip to backlog list page

## Contract Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Validation result storage | Sibling file (`validation-report.json`) | Clean separation from spec.json; ephemeral/regenerable data separate from authoritative metadata |
| Auto-queue blocking | Check `validation-report.json` in queue handler as forceable `BlockingReason` | Integrates with existing blocking flow; forceable allows manual override |
| Section header matching | Normalized keyword matching | Lowercase + strip numbering, robust to formatting variations without regex complexity |
| Pre-finalization gap format | Structured markdown checklist | Agents parse markdown naturally; clear and actionable |

## Testing Plan

### Unit Tests (`plan_validate_test.go`)
- Test `ValidatePlanCompleteness` with:
  - Complete plan (all 13 sections) → `Passed: true`, no missing sections
  - Plan missing 3 sections → lists exact missing sections
  - Plan missing `prompt-manager skill read` → warning
  - Plan missing greenfield declaration → warning
  - Plan missing `vrooli scenario restart` → warning
  - Idea plan missing scenario template reference → section-specific warning
  - Idea plan missing UI mention → warning
  - Empty plan content → all sections missing
  - Research kind → should return early with `Passed: true` (skipped)
- Test fuzzy header matching:
  - `## Required Reading` matches
  - `## 2. Required Reading` matches
  - `## required reading` matches
  - `### Required Reading` (h3) does NOT match (only h2)
  - `## Required-Reading` matches (hyphenated)

### Integration Tests
- Pre-finalization: mock plan.md with gaps, verify gap report appears in assembled prompt
- Post-finalization: save a finalize round, verify `validation-report.json` is created
- Queue blocking: create item with failed validation, attempt queue without force → blocked; with force → proceeds
- Validation endpoint: GET returns correct `PlanValidationResult`

### UI Tests
- ValidationBadge renders green/yellow/red for passed/warnings/failed states
- ValidationReport shows correct sections in detail view
- List filter correctly filters by validation status

## Rollout / Validation Checklist

1. [ ] `plan_validate.go` + `plan_validate_test.go` — core validation function with unit tests
2. [ ] Pre-finalization gate integrated in `research.go` after line ~296
3. [ ] Post-finalization gate integrated in `workshop_save.go` on finalize round save
4. [ ] `validation-report.json` written to item directory after finalization
5. [ ] Queue blocking via forceable `BlockingReason` in `queue_ops.go`
6. [ ] GET endpoint includes `plan_validation` field (handler_query.go)
7. [ ] Dedicated `GET /api/v1/backlog/{kind}/{name}/validation` endpoint
8. [ ] List endpoint supports `validation_status` filter
9. [ ] UI `ValidationBadge` component in backlog-card.tsx
10. [ ] UI `ValidationReport` component on detail page info tab
11. [ ] UI list filter chip for validation status
12. [ ] All tests pass: `go test ./... -timeout 300s`
13. [ ] `vrooli scenario restart swarm-manager`

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Fuzzy header matching produces false positives/negatives | Medium | Comprehensive test cases covering formatting variations; keyword list is conservative |
| Pre-finalization context makes prompt too long | Low | Concise checklist format; only list missing sections, not analysis |
| Post-finalization validation runs before plan.md is fully written | Medium | Ensure validation runs AFTER both round file AND plan.md writes complete; use same write-then-validate ordering |
| `validation-report.json` grows stale if plan is manually edited | Low | Re-generate on GET request if plan.md mtime > validation-report.json mtime |
| Auto-queue blocking frustrates users | Low | Forceable blocking reason allows manual override; clear UI indication of what's missing |
| Normalized matching misses unusual section names | Low | Conservative keyword list with test coverage; agents mostly follow standard naming from skill |

## Non-goals / Prohibited Patterns

- Do NOT block finalization based on validation results — pre-gate is advisory only
- Do NOT validate conclusion.md for research items
- Do NOT modify the workshop round schema (use sibling `validation-report.json` for validation results)
- Do NOT add validation to normal workshop rounds — only finalization
- Do NOT add backwards-compatibility shims for items finalized before this feature
- Do NOT create a new item status for validation failures — use existing blocking reason system

## Definition of Done

1. `ValidatePlanCompleteness` correctly identifies all 13 mandatory sections and additional required elements
2. Pre-finalization gate passes gap report to finalize agent prompt as markdown checklist
3. Post-finalization gate writes `validation-report.json` and adds forceable blocking reason in queue handler
4. API exposes validation status on GET, list, and dedicated endpoint
5. UI shows validation badges (existing badge pattern), detailed reports, and list filtering
6. All unit and integration tests pass
7. Research items are excluded from validation
8. Greenfield declaration honored — no compatibility shims
9. Final step: `vrooli scenario restart swarm-manager`
