# Plan: Add Scenario Picker to Brand Creation and Detail Flows

## 1. Purpose

Add a scenario picker component so brands can be associated with a scenario at creation time, and manage that association from the brand detail page. This eliminates the separate assignment step and makes scenario-brand binding a first-class part of the brand lifecycle.

## 2. Required Reading

```bash
prompt-manager skill read api-steer react-coherence test
```

## 3. Problem Statement

Currently, brand creation and scenario assignment are disconnected workflows:
1. User creates a brand via `POST /api/v1/brands` with no scenario reference
2. User must separately assign via `POST /api/v1/assignments` with `{brand_id, scenario_name, elements[]}`
3. The BrandFormPage has no scenario selection UI
4. The BrandDetailPage doesn't show which scenario is assigned

This creates friction and makes it easy to create "orphan" brands with no scenario association.

## 4. Scope

**In scope:**
- New `GET /api/v1/scenarios` endpoint listing available scenarios with metadata
- ScenarioPicker reusable UI component (search, filter, single-select)
- Integration into BrandFormPage (optional scenario selection at creation time)
- Integration into BrandDetailPage (show/change/unassign scenario)
- Auto-create assignment after brand creation when scenario is selected
- Tests for new endpoint and UI integration

**Out of scope:**
- Multi-scenario assignment (UNIQUE constraint on assignments table enforces 1 brand per scenario)
- Changes to existing assignment API contract
- Scenario management (CRUD on scenarios themselves)
- Changes to the CLI

## 5. Current Technical Context

### Key Files
| File | Role |
|------|------|
| `scenarios/brand-manager/api/handlers/brands.go` | Route registration + all handlers (CRUD, assignments) |
| `scenarios/brand-manager/api/domain/types.go` | Domain structs: Brand, Assignment, ScenarioStatus |
| `scenarios/brand-manager/api/repository/interfaces.go` | Repository interfaces |
| `scenarios/brand-manager/api/repository/sqlite_assignments.go` | Assignment SQLite impl |
| `scenarios/brand-manager/api/config/config.go` | Config with ScenariosDir |
| `scenarios/brand-manager/ui/src/pages/BrandFormPage.tsx` | Brand create/edit form |
| `scenarios/brand-manager/ui/src/pages/BrandDetailPage.tsx` | Brand detail view |
| `scenarios/brand-manager/ui/src/lib/api.ts` | TypeScript API client |
| `scenarios/brand-manager/ui/src/lib/router.ts` | Hash-based routing |
| `scenarios/brand-manager/api/database/schema.sql` | SQLite schema |

### Existing Infrastructure
- `config.ScenariosDir` already resolves the scenarios directory path
- `ScenarioStatus` type exists with `HasBrand`, `BrandID`, `Elements` fields
- `GET /api/v1/scenarios/{name}/status` endpoint already checks if a scenario has a brand
- `createAssignment()` and `fetchScenarioStatus()` already exist in the UI API client
- Assignment table has `UNIQUE(scenario_name)` constraint
- `ApplyPreview` component exists but uses manual text input for scenario name

## 6. Target End State

1. **API**: `GET /api/v1/scenarios` returns a list of available scenarios with name, description, and assignment status
2. **UI - ScenarioPicker**: Reusable component showing scenario cards with search, indicating which already have brands assigned
3. **UI - BrandFormPage**: Optional scenario picker integrated; selecting a scenario auto-creates assignment after brand save
4. **UI - BrandDetailPage**: Shows assigned scenario with badge/link; supports change and unassign actions
5. **Tests**: API handler tests for scenario listing; UI component tests for picker behavior

## 7. Implementation Strategy

### Phase 1: API — Scenario List Endpoint
1. Add `ScenarioInfo` struct to `domain/types.go` (name, description, path, has_brand, brand_id)
2. Add `ListScenarios(ctx, filter)` handler in `handlers/brands.go` that:
   - Reads directories from `config.ScenariosDir`
   - Optionally reads scenario metadata (service.json or similar) for display name/description
   - Enriches with assignment status from the DB
   - Supports `?search=` query parameter for name filtering
3. Register `GET /api/v1/scenarios` route
4. Write handler tests

### Phase 2: UI — Scenario Picker Component
1. Add `ScenarioInfo` type to `api.ts` and `fetchScenarios(search?)` client function
2. Create `ScenarioPicker` component:
   - Search input with debounced filtering
   - List of scenario cards showing name, description, assignment status
   - Already-assigned scenarios are visually distinguished (not disabled — user may want to reassign)
   - Single-select behavior with clear selection option
3. Props: `value: string | null`, `onChange: (scenarioName: string | null) => void`

### Phase 3: BrandFormPage Integration
1. Add ScenarioPicker to the form (optional, placed prominently before brand fields)
2. When scenario selected: store scenario name in form state
3. On form submit:
   - Create brand as usual (`POST /api/v1/brands`)
   - If scenario selected, auto-create assignment (`POST /api/v1/assignments`)
   - Handle assignment creation failure gracefully (brand still created, show error for assignment)
4. When no scenario selected: existing behavior unchanged

### Phase 4: BrandDetailPage Integration
1. Fetch assignment status for the brand (use existing `ListByBrandID` or add convenience)
2. Show assigned scenario as badge/link (if any)
3. "Change Scenario" action opens ScenarioPicker dialog
4. "Unassign" action calls `DELETE /api/v1/assignments/{id}`
5. "Assign to Scenario" prompt when no scenario assigned

## 8. Contract Decisions

### New Endpoint: `GET /api/v1/scenarios`
- **Query params**: `search` (optional, name filter)
- **Response**: `{ scenarios: ScenarioInfo[] }`
- **ScenarioInfo shape**: `{ name, description, path, has_brand, brand_id?, brand_name? }`
- **Auth**: Same auth level as other brand endpoints

### BrandFormPage Behavior
- Scenario selection is optional — standalone brand creation still works
- Assignment creation happens as a separate API call after brand creation (not a new combined endpoint)
- If assignment fails after brand creation, the brand persists and user is notified

## 9. Testing Plan

### API Tests
- `TestListScenarios_Empty` — no scenarios directory or empty
- `TestListScenarios_WithResults` — returns scenario list with correct metadata
- `TestListScenarios_Search` — search filtering works
- `TestListScenarios_WithAssignments` — correctly marks scenarios that have brands assigned

### UI Tests (if test infrastructure supports)
- ScenarioPicker renders and searches
- BrandFormPage submits with scenario and creates assignment
- BrandDetailPage shows and manages assignment

## 10. Rollout/Validation Checklist

- [ ] `go build ./...` passes in brand-manager API
- [ ] `go test ./... -timeout 300s` passes
- [ ] UI builds without errors
- [ ] Manual test: create brand with scenario → assignment auto-created
- [ ] Manual test: create brand without scenario → works as before
- [ ] Manual test: view brand detail → shows assigned scenario
- [ ] Manual test: change/unassign scenario from detail page

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scenarios dir not set or empty | Picker shows nothing | Graceful empty state, clear message |
| Scenario metadata format varies | Inconsistent display | Fall back to directory name if no metadata |
| Assignment creation fails after brand creation | Orphan brand | Show error, allow retry from detail page |
| Race condition on UNIQUE constraint | 409 conflict | Handle conflict error gracefully in UI |

## 12. Non-goals / Prohibited Patterns

- No new database tables or migrations (use existing schema)
- No changes to the existing assignment API contract
- No scenario CRUD operations
- No combined "create brand + assign" atomic endpoint — keep as two calls
- No changes to CLI tooling

## 13. Definition of Done

- `GET /api/v1/scenarios` endpoint returns scenario list with assignment enrichment
- ScenarioPicker component is reusable and integrated in both BrandFormPage and BrandDetailPage
- Brand creation flow optionally associates a scenario
- Brand detail page shows, changes, and unassigns scenarios
- All new API handlers have tests
- Existing tests continue to pass
