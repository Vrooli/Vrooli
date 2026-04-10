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
- Block auto-queue when validation fails (forceable `BlockingReason` in queue handler)
- New `plan_validation_json` string field on BacklogItem proto response (raw JSON passthrough)
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
1. `WorkshopSave()` in `workshop_save.go` (line 33) saves round responses
2. Auto-advance logic (`ShouldAutoAdvance` + `resolveNextMode`) decides whether to trigger finalize
3. `spawnWorkshopAsync(item, ResearchModeFinalize)` (line 168/221) spawns the finalize agent
4. Pre-finalization checks in `research.go:272-296`: round exists, decisions answered, readiness >= 3, synthesis pending
5. Finalize agent writes plan.md via file-upload endpoint, then calls WorkshopSave with a mode="finalize" round
6. **Key timing confirmation (round 2):** plan.md is always written BEFORE the finalize round is saved — post-validation in WorkshopSave is safe with no race condition

### Key File-Reading Utilities
- `workshop.LoadPlanContent(itemDir)` — reads plan.md from item directory (workshop/workshop.go:270-272)
- `workshop.LoadPlanContentByName(itemDir, filename)` — reads named deliverable file
- `workshop.HasPlanByName(itemDir, filename)` — checks if deliverable file exists
- `DeliverableForKind(kind)` — returns "conclusion.md" for research, "plan.md" for others (kind_config.go:25-30)

### Prompt Assembly (research.go:310-342)
- `fetchResearchPrompt()` resolves skill by mode+kind, builds variable map
- Additional user context appended as `"\n\nAdditional context from user:\n" + userPrompt`
- Pre-finalization gap report will be injected similarly as additional context

### Queue Handler & Blocking (queue_ops.go + blocking.go)
- `isQueueableItem()` (queue_ops.go:25-33) checks allowed statuses
- `executionService.ProcessPreflight()` (queue_ops.go:248-287) returns blocking reasons
- `BlockingReason` struct (blocking.go:12-15): `{Message string, Forceable bool}`
- Force override: if `force=true` and `AllForceable(reasons)` → proceed (queue_ops.go:186-194)
- Existing forceable reasons: pending workshop decisions, unmet dependencies
- **Decision (round 2, d1):** Validation failure appended as a new forceable `BlockingReason` during preflight — same existing pattern

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
- **Backlog detail page:** `ui/src/pages/BacklogDetailsPage.tsx` — tabbed layout, data via `useBacklogDetailData` hook
- **Backlog card:** `ui/src/components/backlog/backlog-card.tsx` — badge row at lines 145-148 (ScenarioBadge, AgentRunningBadge, CircuitBrokenBadge)
- **Badge pattern:** small indicator components with conditional render (best model: CircuitBrokenBadge — simple conditional + icon + title)

## Target End State

After implementation:
1. Every finalization attempt runs plan content validation and passes gap info to the finalize agent as a structured markdown checklist
2. After the finalize round is saved via WorkshopSave (detected by `round.Mode == "finalize"`), the same validation runs and writes `validation-report.json` as a sibling file
3. Items with failed post-finalization validation cannot auto-queue — validation failure appears as a **forceable** `BlockingReason`, overridable with `force` flag
4. The API exposes validation via `plan_validation_json` raw JSON string field on BacklogItem proto and a dedicated `/validation` endpoint
5. The UI shows validation badges (CircuitBrokenBadge pattern) and detailed reports
6. Research items are excluded from plan validation entirely
7. Stale `validation-report.json` is re-generated on GET when plan.md mtime > report mtime

## Implementation Strategy

### Phase 1: Core Validation Logic (`plan_validate.go`)
Create `api/internal/backlog/plan_validate.go` with:

```go
type PlanValidationResult struct {
    SectionsPresent []string  `json:"sections_present"`
    SectionsMissing []string  `json:"sections_missing"`
    Warnings        []string  `json:"warnings"`
    Passed          bool      `json:"passed"`
    ValidatedAt     string    `json:"validated_at"`
}
```

`ValidatePlanCompleteness(planContent string, kind BacklogKind) PlanValidationResult`:
- Extract all `## ` header lines from planContent (h2 only — h3 does NOT match)
- Normalize each: lowercase, strip leading numbers/dots/punctuation, trim whitespace
- Match against canonical keyword list (e.g., "required reading", "problem statement", "testing plan")
- **Header presence only — no content depth check** (decision round 2, d4). Empty-but-headed sections are acceptable; the finalize agent skill instructions handle filling sections
- Check for `prompt-manager skill read` string presence → warning if missing
- Check for greenfield declaration → warning if missing
- Check for `vrooli scenario restart` → warning if missing
- Idea-specific: check for `scenario-generation` or `vrooli scenario create` → warning if missing
- Idea-specific: check for UI mention (case-insensitive "ui" in context of template/section) → warning if missing
- `Passed` = no missing sections AND no critical warnings (prompt-manager and greenfield are critical)
- Return early with `Passed: true` for research kind (skip validation)

### Phase 2: Pre-Finalization Gate
In `research.go`, after existing pre-finalization checks (~line 296) and before `spawnWorkshopAsync`:
- Load plan.md via `workshop.LoadPlanContent(itemDir)` using `DeliverableForKind(kind)` to get filename
- Skip if kind is research
- Call `ValidatePlanCompleteness(content, kind)`
- If gaps found, format as **structured markdown checklist** (decision round 1, d4):
  ```
  ## Plan Validation Gaps
  The following issues were found in plan.md. Please fix these during finalization:
  - [ ] Missing section: Testing Plan
  - [ ] Missing section: Definition of Done
  - [ ] Warning: No `prompt-manager skill read` command found in Required Reading
  ```
- Append to the prompt context (same pattern as `req.Prompt` injection)
- If plan.md doesn't exist, note "plan.md does not exist yet — create all 13 mandatory sections"
- Does NOT block finalization — advisory only

### Phase 3: Post-Finalization Gate
**Trigger point: in WorkshopSave after saving the finalize round** (decision round 2, d1). After the round file is written (~line 100, after the slog.Info call):
- Check if `round.Mode == "finalize"`
- If so, load plan.md via `workshop.LoadPlanContent(itemDir)`
- Call `ValidatePlanCompleteness(content, kind)`
- Marshal result to JSON and write **`validation-report.json`** as a sibling file (decision round 1, d1) to `itemDir`
- Log validation result summary

### Phase 4: Queue Blocking Integration
In `queue_ops.go`, during preflight evaluation (alongside existing dependency and pending-decisions checks):
- Load `validation-report.json` from item directory
- If file exists and `Passed == false`:
  - Append `BlockingReason{Message: "plan validation failed: missing sections: X, Y, Z", Forceable: true}`
- If file doesn't exist, no blocking (validation hasn't run yet — item may not have been finalized)

### Phase 5: API Surface
- Add **`plan_validation_json` optional string field** to BacklogItem proto message (decision round 2, d2) — raw JSON passthrough avoids proto schema complexity for ephemeral data
- In `backlogToProto()` (types.go:169-206): read `validation-report.json`, **re-run validation if plan.md mtime > report mtime** (decision round 2, d3), then set field as raw JSON string
- Add `GET /api/v1/backlog/{kind}/{name}/validation` endpoint: always re-runs validation and returns fresh result
- Add `validation_status` filter parameter to list endpoint (values: `passed`, `failed`, `none`)

### Phase 6: UI Components
- **ValidationBadge** (`validation-badge.tsx`): Follow CircuitBrokenBadge pattern
  - Props: `validationJson: string | undefined`
  - Parse JSON, render: green check if passed, yellow warning if passed with warnings, red X if failed
  - Returns null if no validation data
- **ValidationReport** component on detail page info tab:
  - Collapsible panel showing sections present (green), missing (red), and warnings (yellow)
  - Rendered in BacklogDetailsPage info tab
- **List filter**: add validation status filter chip alongside existing filters
- Add ValidationBadge to backlog-card.tsx badge row (line ~148, after CircuitBrokenBadge)

## Contract Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Validation result storage | Sibling file (`validation-report.json`) | Clean separation; ephemeral/regenerable data separate from authoritative metadata |
| Auto-queue blocking | Forceable `BlockingReason` in queue handler | Integrates with existing blocking flow; override with `force` flag |
| Section header matching | Normalized keyword matching (lowercase, strip numbers/punctuation) | Robust to formatting variations without regex complexity |
| Pre-finalization gap format | Structured markdown checklist | Agents parse markdown naturally; clear and actionable |
| Post-finalization trigger | In WorkshopSave after saving finalize round | Plan.md is guaranteed to exist (agent writes it before round); synchronous, no race conditions |
| API field type | Raw JSON string (`plan_validation_json`) on proto | Avoids proto schema complexity for ephemeral data |
| Content depth check | Header presence only, no minimum content threshold | Avoids false positives; finalize skill instructions are clear about filling sections |
| Stale report handling | Re-validate on GET when plan.md mtime > report mtime | Ensures freshness; acceptable ~1ms I/O for low-traffic backlog endpoints |

## Testing Plan

### Unit Tests (`plan_validate_test.go`)
- **Complete plan**: all 13 sections present → `Passed: true`, empty `SectionsMissing`
- **Missing sections**: plan missing 3 sections → `SectionsMissing` lists exact names
- **Missing prompt-manager**: no `prompt-manager skill read` → warning in `Warnings`
- **Missing greenfield**: no greenfield declaration → warning
- **Missing vrooli scenario restart**: → warning
- **Idea missing template**: idea kind, no `vrooli scenario create` → warning
- **Idea missing UI**: idea kind, no UI mention → warning
- **Empty plan**: all 13 sections missing, all critical warnings present
- **Research kind**: returns early with `Passed: true` (skipped)
- **Fuzzy matching cases**:
  - `## Required Reading` → matches
  - `## 2. Required Reading` → matches
  - `## required reading` → matches
  - `### Required Reading` (h3) → does NOT match
  - `## Required-Reading` → matches
  - `## Risks + Mitigations` and `## Risks and Mitigations` → both match

### Integration Tests
- **Pre-finalization**: mock plan.md with gaps, verify gap report appears in assembled finalize prompt
- **Post-finalization**: save a finalize round via WorkshopSave, verify `validation-report.json` is created with correct content
- **Queue blocking**: create item with failed validation, attempt queue without force → blocked; with force → proceeds
- **Stale report**: modify plan.md after validation, GET request → report is re-generated
- **Validation endpoint**: GET `/validation` returns fresh `PlanValidationResult`
- **Research skip**: research item finalization → no `validation-report.json` written

### UI Tests
- ValidationBadge renders green/yellow/red for passed/warnings/failed states
- ValidationBadge returns null when no validation data
- ValidationReport shows correct sections in detail view
- List filter correctly filters by validation status

## Rollout / Validation Checklist

1. [ ] `plan_validate.go` + `plan_validate_test.go` — core validation function with unit tests
2. [ ] Pre-finalization gate integrated in `research.go` after line ~296
3. [ ] Post-finalization gate integrated in `workshop_save.go` after line ~100 (on mode=="finalize")
4. [ ] `validation-report.json` written to item directory after finalization
5. [ ] Queue blocking via forceable `BlockingReason` in `queue_ops.go` during preflight
6. [ ] `plan_validation_json` string field added to BacklogItem proto
7. [ ] `backlogToProto()` reads/refreshes validation report (stale mtime check)
8. [ ] Dedicated `GET /api/v1/backlog/{kind}/{name}/validation` endpoint
9. [ ] List endpoint supports `validation_status` filter
10. [ ] UI `ValidationBadge` component in backlog-card.tsx (after CircuitBrokenBadge)
11. [ ] UI `ValidationReport` component on detail page info tab
12. [ ] UI list filter chip for validation status
13. [ ] All tests pass: `go test ./... -timeout 300s`
14. [ ] `vrooli scenario restart swarm-manager`

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Fuzzy header matching false positives/negatives | Medium | Comprehensive test cases covering formatting variations; conservative keyword list |
| Pre-finalization context bloats prompt | Low | Concise checklist format; only list missing sections, not analysis |
| Post-validation runs before plan.md is written | None | Confirmed in round 2: finalize agent writes plan.md before saving round — no race condition |
| `validation-report.json` grows stale | Low | Re-generate on GET when plan.md mtime > report mtime |
| Auto-queue blocking frustrates users | Low | Forceable blocking reason; clear UI indication of what's missing |
| Normalized matching misses unusual section names | Low | Conservative keyword list with test coverage; agents follow standard naming from skill |
| Plan.md doesn't exist at finalization time | Low | Pre-gate handles missing plan.md gracefully (reports "plan.md does not exist yet") |

## Non-goals / Prohibited Patterns

- Do NOT block finalization based on validation results — pre-gate is advisory only
- Do NOT validate conclusion.md for research items
- Do NOT modify the workshop round schema
- Do NOT add validation to normal workshop rounds — only finalization
- Do NOT add backwards-compatibility shims for items finalized before this feature
- Do NOT create a new item status for validation failures — use existing blocking reason system
- Do NOT check content depth under section headers — header presence is sufficient

## Definition of Done

1. `ValidatePlanCompleteness` correctly identifies all 13 mandatory sections and additional required elements via normalized keyword matching (header presence only)
2. Pre-finalization gate passes gap report to finalize agent prompt as structured markdown checklist (advisory, non-blocking)
3. Post-finalization gate in WorkshopSave (triggered on `round.Mode == "finalize"`) writes `validation-report.json` as a sibling file
4. Queue handler adds forceable `BlockingReason` when validation fails
5. API exposes `plan_validation_json` raw JSON string on BacklogItem GET and list, plus dedicated `/validation` endpoint
6. Stale reports are re-generated when plan.md mtime > report mtime
7. UI shows ValidationBadge (CircuitBrokenBadge pattern) and ValidationReport on detail page
8. All unit and integration tests pass
9. Research items excluded from validation
10. Greenfield — no compatibility shims
11. Final step: `vrooli scenario restart swarm-manager`
