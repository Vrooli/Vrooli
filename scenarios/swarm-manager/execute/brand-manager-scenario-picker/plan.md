# Plan: Add Scenario Picker to Brand Creation and Detail Flows

## 1. Purpose

Add a scenario picker component so brands can be associated with a scenario at creation time, and manage that association from the brand detail page. This eliminates the separate assignment step and makes scenario-brand binding a first-class part of the brand lifecycle.

## 2. Required Reading

```bash
prompt-manager skill read api-steer react-coherence test
```

## 3. Greenfield Constraint

**This is greenfield work.** Do not include compatibility shims, legacy wrappers, dead code, unused re-exports, `// removed` comments, or renamed `_unused` variables. The existing `ApplyPreview` free-text scenario input is *replaced* by the new `ScenarioPicker` — it is not preserved as a fallback (see Phase 5).

## 4. Problem Statement

Currently, brand creation and scenario assignment are disconnected workflows:
1. User creates a brand via `POST /api/v1/brands` with no scenario reference.
2. User must separately assign via `POST /api/v1/assignments` with `{brand_id, scenario_name, elements[]}`.
3. The BrandFormPage has no scenario selection UI.
4. The BrandDetailPage doesn't show which scenario is assigned.

This creates friction and makes it easy to create "orphan" brands with no scenario association.

## 5. Scope

**In scope:**
- New `GET /api/v1/scenarios` endpoint listing available scenarios with metadata sourced from each scenario directory's `service.json` (round-1 d1=A).
- New `GET /api/v1/brands/{id}/assignments` endpoint exposing the existing `AssignmentRepository.ListByBrandID` method so the detail page can fetch the brand's current scenario assignment without a payload-shape change to `GET /api/v1/brands/{id}` (round-2 d3, see §10).
- `ScenarioPicker` reusable UI component, rendered as a modal dialog everywhere it appears (round-1 d2=B), backed by a new `components/ui/dialog.tsx` wrapper around `@radix-ui/react-dialog` (round-2 d1=A).
- Integration into BrandFormPage (optional scenario selection at creation time) with a two-call flow on submit: create brand, then create assignment (round-1 d3=A).
- Integration into BrandDetailPage (show / change / unassign scenario).
- Refactor of `ui/src/components/apply-preview.tsx` to consume `ScenarioPicker` instead of the existing free-text `Input` (Phase 5; greenfield removal of the free-text path).
- Tests for new endpoints and UI integration.

**Out of scope:**
- Multi-scenario assignment (UNIQUE constraint on assignments table enforces 1 brand per scenario).
- Changes to existing assignment API contract.
- Scenario management (CRUD on scenarios themselves).
- Changes to the CLI.
- New database tables or migrations.

## 6. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `scenarios/brand-manager/api/handlers/brands.go` | Route registration + all handlers (CRUD, assignments) |
| `scenarios/brand-manager/api/domain/types.go` | Domain structs: Brand, Assignment, ScenarioStatus |
| `scenarios/brand-manager/api/repository/interfaces.go` | Repository interfaces |
| `scenarios/brand-manager/api/repository/sqlite_assignments.go` | Assignment SQLite impl, including `ListByBrandID` (not yet HTTP-exposed) |
| `scenarios/brand-manager/api/config/config.go` | Config with `ScenariosDir` |
| `scenarios/brand-manager/ui/src/pages/BrandFormPage.tsx` | Brand create/edit form |
| `scenarios/brand-manager/ui/src/pages/BrandDetailPage.tsx` | Brand detail view |
| `scenarios/brand-manager/ui/src/components/apply-preview.tsx` | Existing component using free-text scenario name input — to be refactored onto `ScenarioPicker` |
| `scenarios/brand-manager/ui/src/components/ui/` | Currently contains `button`, `input`, `section`, `error-alert` — **no dialog primitive yet** |
| `scenarios/brand-manager/ui/src/lib/api.ts` | TypeScript API client |
| `scenarios/brand-manager/ui/src/lib/router.ts` | Hash-based routing |
| `scenarios/brand-manager/api/database/schema.sql` | SQLite schema |

### Existing Infrastructure
- `config.ScenariosDir` already resolves the scenarios directory path.
- `ScenarioStatus` type exists with `HasBrand`, `BrandID`, `Elements` fields.
- `GET /api/v1/scenarios/{name}/status` endpoint already checks if a scenario has a brand.
- `createAssignment()` and `fetchScenarioStatus()` already exist in the UI API client.
- Assignment table has `UNIQUE(scenario_name)` constraint.
- `package.json` already depends on `@radix-ui/react-slot`; adding `@radix-ui/react-dialog` is in-family.

## 7. Target End State

1. **API**: `GET /api/v1/scenarios` returns scenario list (name, description, path, has_brand, brand_id, brand_name) sourced from `service.json` per scenario directory, enriched with assignment status.
2. **API**: `GET /api/v1/brands/{id}/assignments` returns the brand's assignments.
3. **UI primitive**: `components/ui/dialog.tsx` exists, wrapping `@radix-ui/react-dialog`.
4. **UI - ScenarioPicker**: Modal-only component showing scenario cards with search; assigned scenarios are shown with a badge naming the owning brand (round-1 d4=A).
5. **UI - BrandFormPage**: Optional scenario picker; selecting a scenario auto-creates an assignment after brand save via two sequential calls.
6. **UI - BrandDetailPage**: Shows assigned scenario with badge/link; supports change and unassign actions.
7. **UI - ApplyPreview**: Uses `ScenarioPicker` instead of free-text input.
8. **Tests**: Go handler tests for both new endpoints; Vitest tests for `ScenarioPicker` and the form/detail integrations.

## 8. Implementation Strategy

### Phase 1: API — Scenario List Endpoint
1. Add `ScenarioInfo` struct to `domain/types.go`: `name, description, path, has_brand, brand_id?, brand_name?`.
2. Add `ListScenarios(ctx, filter)` handler in `handlers/brands.go`:
   - Reads directories from `config.ScenariosDir`.
   - Reads `service.json` per directory for display name/description; falls back to directory name on missing/invalid file.
   - Enriches with assignment status from the DB.
   - Supports `?search=` query parameter for name filtering.
3. Register `GET /api/v1/scenarios`.
4. Write handler tests (see §10).

### Phase 2: API — Brand Assignments Endpoint
1. Add handler that calls `AssignmentRepository.ListByBrandID(brandID)` and returns `[]Assignment`.
2. Register `GET /api/v1/brands/{id}/assignments`.
3. Handler tests cover empty result, single assignment, and 404 on unknown brand.

### Phase 3: UI Dialog Primitive + ScenarioPicker
1. Add `@radix-ui/react-dialog` to `scenarios/brand-manager/ui/package.json`.
2. Create `components/ui/dialog.tsx` wrapping Radix's `Root`, `Portal`, `Overlay`, `Content`, `Title`, `Description`, `Close`. Match existing `components/ui/` styling conventions.
3. Add `ScenarioInfo` type and `fetchScenarios(search?)` to `lib/api.ts`.
4. Add `fetchBrandAssignments(brandId)` to `lib/api.ts`.
5. Create `ScenarioPicker` component:
   - Props: `value: string | null`, `onChange: (name: string | null) => void`, `open: boolean`, `onOpenChange: (open: boolean) => void`.
   - Renders inside the new dialog primitive.
   - Debounced search input.
   - Scenario cards: name, description, status; assigned scenarios show a badge with the owning brand name (round-1 d4=A).

### Phase 4: BrandFormPage Integration
1. Add a "Scenario (optional)" affordance on the form that opens the `ScenarioPicker` dialog.
2. Form state holds the selected scenario name (nullable).
3. On submit:
   - `POST /api/v1/brands` to create the brand.
   - If a scenario is selected, follow with `POST /api/v1/assignments`.
   - On assignment failure, the brand persists; surface the error and direct user to retry from the brand's detail page.
4. No scenario selected → existing behavior unchanged.

### Phase 5: BrandDetailPage Integration + ApplyPreview Refactor
1. On load, `GET /api/v1/brands/{id}/assignments`; render assigned scenario as badge/link, or an "Assign to Scenario" prompt when empty.
2. "Change Scenario" opens `ScenarioPicker` dialog.
3. "Unassign" calls `DELETE /api/v1/assignments/{id}`.
4. Refactor `ui/src/components/apply-preview.tsx` to drive `scenarioName` via `ScenarioPicker` instead of the existing free-text `Input`. Remove the free-text path entirely (no fallback) per §3.

### Phase 6: Cleanup & Verification
- Run `go build ./...` and `go test ./... -timeout 300s` from `scenarios/brand-manager/api/`; fix all errors and failing tests, **including pre-existing ones**.
- Run UI lint and type checks (`npm run lint`, `npx tsc --noEmit`) and unit tests (`npm test` / Vitest); fix all issues in modified files, **including pre-existing ones**.
- `vrooli scenario restart brand-manager`.
- Verify health: brand-manager API health endpoint responds, UI loads, scenario picker fetches.

## 9. Contract Decisions

### `GET /api/v1/scenarios`
- Query: `search` (optional, name substring filter).
- Response: `{ scenarios: ScenarioInfo[] }`.
- `ScenarioInfo`: `{ name, description, path, has_brand, brand_id?, brand_name? }`.
- Auth: parity with other brand endpoints.
- Metadata source: `service.json` per scenario directory, with directory-name fallback (round-1 d1=A).

### `GET /api/v1/brands/{id}/assignments`
- Response: `{ assignments: Assignment[] }`.
- Auth: parity with `GET /api/v1/brands/{id}`.
- Does **not** alter `GET /api/v1/brands/{id}` payload (round-2 d3=A as default; see §15).

### BrandFormPage Behavior
- Scenario selection is optional — standalone brand creation still works.
- Two sequential client-side calls: create brand, then create assignment (round-1 d3=A).
- If assignment creation fails post-brand, the brand persists and the user is notified.

### ScenarioPicker UX
- Always rendered as a modal dialog (round-1 d2=B), backed by `@radix-ui/react-dialog`.
- Already-assigned scenarios shown with an owning-brand badge but remain selectable; selecting one triggers a reassignment confirmation step (round-1 d4=A; mechanism deferred — see §15 d4).

## 10. Testing Plan

### Go API tests (`scenarios/brand-manager/api/handlers/`)
- `TestListScenarios_Empty` — empty `ScenariosDir`.
- `TestListScenarios_WithResults` — multiple scenarios, mixed with/without `service.json`.
- `TestListScenarios_Search` — `?search=` substring filter.
- `TestListScenarios_WithAssignments` — assignment status enrichment is correct.
- `TestListBrandAssignments_None` — brand with no assignments returns empty slice.
- `TestListBrandAssignments_OK` — returns assignments owned by the brand.
- `TestListBrandAssignments_UnknownBrand` — returns 404.

### UI tests (Vitest, `scenarios/brand-manager/ui/`)
- `ScenarioPicker.test.tsx` — renders inside dialog; search filters list; selecting a scenario calls `onChange`; assigned scenarios show badge.
- `BrandFormPage.test.tsx` — submit with scenario triggers brand create + assignment create; submit without scenario triggers only brand create; assignment failure surfaces error.
- `BrandDetailPage.test.tsx` — renders assigned scenario badge; "Unassign" calls delete; "Change Scenario" opens picker.
- `ApplyPreview.test.tsx` — uses `ScenarioPicker` (no free-text input present).

## 11. Rollout/Validation Checklist

- [ ] `go build ./...` passes in `scenarios/brand-manager/api/`.
- [ ] `go test ./... -timeout 300s` passes.
- [ ] `npm run lint` and `npx tsc --noEmit` pass for the UI; all warnings in modified files fixed (even pre-existing).
- [ ] `npm test` (Vitest) passes.
- [ ] `vrooli scenario restart brand-manager` completes cleanly.
- [ ] Brand-manager API health check responds.
- [ ] Brand-manager UI loads and the scenario picker fetches scenarios from the live API.

## 12. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| `ScenariosDir` unset or empty | Picker shows nothing | Graceful empty state with explanatory copy. |
| `service.json` missing or malformed in some scenarios | Inconsistent display | Fall back to directory name; log a warning server-side. |
| Assignment creation fails after brand creation | Orphan brand | Surface error; brand detail page allows retry via "Assign to Scenario". |
| UNIQUE(scenario_name) race when reassigning | 409 conflict | Handle conflict in UI with clear message; reassignment flow is delete-then-create (see §15 d4). |
| Sibling overlap with `execute/brand-manager-discovery-import-ui` | Duplicated scenario-listing UI | Both items can share the new `GET /api/v1/scenarios` endpoint and the `ScenarioPicker` primitive — see §16. |
| Adding `@radix-ui/react-dialog` increases bundle | Negligible (~10 KB gz) | Accepted; matches existing Radix dependency. |

## 13. Non-goals / Prohibited Patterns

- No new database tables or migrations (use existing schema).
- No changes to the existing assignment API contract.
- No scenario CRUD operations.
- No combined "create brand + assign" atomic endpoint — keep as two calls.
- No CLI changes.
- No backwards-compatibility wrapper for the old free-text `ApplyPreview` input (greenfield, §3).

## 14. Definition of Done

- `GET /api/v1/scenarios` returns scenario list with assignment enrichment.
- `GET /api/v1/brands/{id}/assignments` returns the brand's assignments.
- `components/ui/dialog.tsx` exists and is consumed by `ScenarioPicker`.
- `ScenarioPicker` is reusable and integrated in `BrandFormPage`, `BrandDetailPage`, and `ApplyPreview`.
- Brand creation flow optionally associates a scenario via the two-call flow.
- Brand detail page shows, changes, and unassigns scenarios.
- All new API handlers and UI components have tests; existing tests continue to pass.
- Greenfield constraint (§3) honored — no `_unused` renames, no `// removed` comments, no free-text fallback.
- Final cleanup & verification (§8 Phase 6) completed.

## 15. Open Decisions Carried Forward

These workshop-2 decisions remain unanswered. The plan assumes the recommended option for each so implementation can proceed; if the user picks differently, update the corresponding section listed below.

- **d2 — Default elements for auto-created assignment.** Assumed: apply all elements (colors, typography, identity, voice). Touchpoint: Phase 4 step 3 in §8, and the `POST /api/v1/assignments` body shape.
- **d3 — How BrandDetailPage fetches the brand's assignment.** Assumed: new `GET /api/v1/brands/{id}/assignments` endpoint (already wired into §5, §7 Phase 2, §9, §10). Switching to option B (embed in `GET /api/v1/brands/{id}`) would change the brand response shape and remove Phase 2.
- **d4 — Reassignment confirmation UX.** Assumed: in-dialog confirmation, then DELETE old + POST new (no new contract). Touchpoint: §9 ScenarioPicker UX. Option B would add a `PATCH /api/v1/assignments/{id}` contract.

## 16. Initiative Context

This item belongs to the `brand-manager-readiness` initiative. The new `GET /api/v1/scenarios` endpoint and `ScenarioPicker` component are likely reusable by sibling `execute/brand-manager-discovery-import-ui`, which also needs a scenario list affordance. The orchestrator should sequence these two so that this item lands first and the discovery-import UI consumes the same primitives.
