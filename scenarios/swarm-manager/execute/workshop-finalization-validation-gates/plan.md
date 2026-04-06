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

- Pre-finalization validation in `api/internal/backlog/research.go:209-233` only checks: round exists, decisions answered, readiness >= 3, synthesis pending
- No structural validation of plan.md content exists anywhere in the codebase
- The 13 mandatory sections are defined in the `implementation-plan-authoring` skill but never enforced programmatically

## Scope

### In Scope
- New `plan_validate.go` file in `api/internal/backlog/` with plan content validation logic
- Pre-finalization gate: call validation before spawning finalize agent, pass gaps as context
- Post-finalization gate: re-validate after finalize round is written, store results
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
4. Pre-finalization checks in `research.go:209-233`: round exists, decisions answered, readiness >= 3, synthesis pending
5. Finalize agent runs, writes a finalize round with `mode: "finalize"`, zero decisions, info items only

### Key Types
- `workshop.Round` — round struct with `Readiness`, `Items`, `Mode`, `PendingSynthesis`
- `BacklogItem` — item with `Kind`, `Status`, `AcceptanceAllow`, `AcceptanceDeny`
- `KindConfig` — maps kinds to deliverable filenames (`plan.md` vs `conclusion.md`)

### Existing Validation
- `validate_globs.go` — validates acceptance glob patterns (reusable pattern for validation endpoints)
- `research.go` pre-finalization checks — structural checks on round state, not plan content

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

## Target End State

After implementation:
1. Every finalization attempt runs plan content validation and passes gap info to the finalize agent
2. After finalization completes, the same validation runs again and stores results
3. Items with failed post-finalization validation cannot auto-queue (stay in `ready` with a warning)
4. The API exposes validation status on GET endpoints (list + detail + dedicated validation endpoint)
5. The UI shows validation badges and detailed reports
6. Research items are excluded from plan validation entirely

## Implementation Strategy

### Phase 1: Core Validation Logic (`plan_validate.go`)
Create `api/internal/backlog/plan_validate.go` with:
- `PlanValidationResult` struct: `SectionsPresent []string`, `SectionsMissing []string`, `Warnings []string`, `Passed bool`
- `ValidatePlanCompleteness(planContent string, kind BacklogKind) PlanValidationResult` — scans for 13 section headers (fuzzy match on header text, case-insensitive), checks for `prompt-manager skill read`, greenfield declaration, `vrooli scenario restart`. For idea items: checks `scenario-generation` or `vrooli scenario create`, UI mention.
- Section header matching: use normalized lowercase comparison with flexible patterns (e.g., "## Required Reading" matches "## required reading" or "## 2. Required Reading")
- Return structured result with all findings

### Phase 2: Pre-Finalization Gate
In `research.go`, after the existing pre-finalization checks (line ~233) and before `spawnWorkshopAsync`:
- Load plan.md content via `LoadPlanContentByName`
- Call `ValidatePlanCompleteness`
- If gaps found, append a structured gap report to the prompt passed to the finalize agent
- This does NOT block finalization — it informs the agent

### Phase 3: Post-Finalization Gate
In `workshop_save.go`, after a finalize round is successfully saved:
- Re-run `ValidatePlanCompleteness` on the updated plan.md
- Write results to a `validation-report.json` sibling file in the item directory
- If validation fails, set a flag on the item (e.g., `validation_warning: true` in spec.json or a separate marker file) that prevents auto-queue

### Phase 4: API Surface
- Add `plan_validation` object to the backlog item GET response (read from `validation-report.json` if it exists)
- Add `GET /api/v1/backlog/{kind}/{name}/validation` endpoint returning detailed `PlanValidationResult`
- Add `validation_status` filter parameter to the list endpoint

### Phase 5: UI Components
- Validation badge component: green check (passed), yellow warning (warnings only), red X (missing sections)
- Detailed validation report panel on item detail page
- Filter chip on backlog list page for validation status

## Contract Decisions

<!-- TBD — pending workshop decisions -->

## Testing Plan

### Unit Tests (`plan_validate_test.go`)
- Test `ValidatePlanCompleteness` with: complete plan, plan missing sections, plan missing skill read, idea plan missing template usage, research item (should skip), empty plan
- Test fuzzy header matching edge cases

### Integration Tests
- Test pre-finalization gap report generation
- Test post-finalization validation-report.json creation
- Test auto-queue blocking when validation fails
- Test validation endpoint returns correct data

### UI Tests
- Validation badge renders correct state for each status
- Detail page shows validation report
- List filter works for validation status

## Rollout / Validation Checklist

1. [ ] `plan_validate.go` — core validation function with tests
2. [ ] Pre-finalization gate integrated in `research.go`
3. [ ] Post-finalization gate integrated in `workshop_save.go`
4. [ ] `validation-report.json` written after finalization
5. [ ] Auto-queue blocking when validation fails
6. [ ] GET endpoint includes `plan_validation` field
7. [ ] Dedicated `/validation` endpoint
8. [ ] List endpoint supports `validation_status` filter
9. [ ] UI validation badge component
10. [ ] UI detail page validation report
11. [ ] UI list filter
12. [ ] All tests pass: `go test ./... -timeout 300s`
13. [ ] `vrooli scenario restart swarm-manager`

## Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Fuzzy header matching produces false positives/negatives | Medium | Use normalized lowercase + flexible regex; comprehensive test cases |
| Pre-finalization context makes prompt too long | Low | Summarize gaps concisely; only include missing sections, not full analysis |
| Post-finalization validation runs before plan.md is fully written | Medium | Ensure validation runs after the round file AND plan.md are both written |
| Validation-report.json grows stale if plan is manually edited | Low | Re-generate on GET request if plan.md is newer than validation-report.json |
| Auto-queue blocking frustrates users | Low | Clear UI indication of what's missing + manual override option |

## Non-goals / Prohibited Patterns

- Do NOT block finalization based on validation results — pre-gate is advisory only
- Do NOT validate conclusion.md for research items
- Do NOT modify the workshop round schema (use sibling file for validation results)
- Do NOT add validation to normal workshop rounds — only finalization
- Do NOT add backwards-compatibility shims for items finalized before this feature

## Definition of Done

1. `ValidatePlanCompleteness` correctly identifies all 13 mandatory sections and additional required elements
2. Pre-finalization gate passes gap report to finalize agent prompt
3. Post-finalization gate writes `validation-report.json` and blocks auto-queue on failure
4. API exposes validation status on GET, list, and dedicated endpoint
5. UI shows validation badges, detailed reports, and list filtering
6. All unit and integration tests pass
7. Research items are excluded from validation
8. Greenfield declaration honored — no compatibility shims
9. Final step: `vrooli scenario restart swarm-manager`
