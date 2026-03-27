# Implementation Plan: Build UI for Discovering and Importing Existing Scenario Branding

## 1. Purpose

Add a discovery/import flow to the Brand Manager UI so users can create brands from scenarios that already have branding elements (colors, logos, manifest.json, etc.), rather than starting from scratch.

## 2. Required Reading

```bash
prompt-manager skill read brand-manager react-coherence ux implementation-plan-authoring
```

**Key files:**
- Discovery handler: `scenarios/brand-manager/api/handlers/discovery.go`
- Domain types: `scenarios/brand-manager/api/domain/types.go`
- Brand list page: `scenarios/brand-manager/ui/src/pages/BrandListPage.tsx`
- API client: `scenarios/brand-manager/ui/src/lib/api.ts`
- Router: `scenarios/brand-manager/ui/src/lib/router.ts`
- App shell: `scenarios/brand-manager/ui/src/App.tsx`

## 3. Problem Statement

The backend discovery endpoints (`GET /api/v1/discover/{scenario}` and `POST /api/v1/discover/{scenario}/import`) are fully implemented and tested, but there is no UI to invoke them. Users cannot:
- Trigger branding discovery from the UI
- Review discovered branding elements before importing
- Create a brand pre-filled with discovered data

This means every brand must be created manually from scratch, even when a scenario already has branding signals.

## 4. Scope

### In scope
- New "Import from Scenario" entry point on BrandListPage
- Discovery page showing results (sources, confidence, draft brand preview, suggestions)
- Import action that creates the brand and navigates to BrandDetailPage
- "Edit before importing" action that opens BrandFormPage pre-filled with discovered data
- API client functions for `discoverScenario()` and `importDiscovery()`
- New route `/discover` and `/discover/:scenario` in the router
- TypeScript types mirroring `DiscoveryResult` and `DiscoverySource`

### Out of scope
- Scenario picker component (separate backlog item: `brand-manager-scenario-picker`)
- Changes to API/backend discovery logic
- Automatic assignment post-import (requires scenario picker integration)
- Brand editing/management enhancements

## 5. Current Technical Context

### Backend (ready)
- `GET /api/v1/discover/{scenario}` → returns `DiscoveryResult { scenario, sources[], draft_brand, confidence, suggestions[] }`
- `POST /api/v1/discover/{scenario}/import` → creates brand from discovered state, returns `{ brand, sources, confidence }`
- `DiscoverySource { file, type, confidence, fields }` where type ∈ {service_json, branding_json, theme_css, manifest, asset}
- Import supports `X-Dry-Run` header for preview

### UI (current state)
- BrandListPage has "New Brand" button → navigates to `/brands/new`
- No discovery-related API functions in `api.ts`
- No scenario picker component exists (ScannerPage uses a simple text input for scenario name)
- Hash-based routing via `useRouter()` in `router.ts`
- UI uses Tailwind (dark theme, slate-950 bg), React Query for data fetching, lucide-react for icons
- Existing components: `ColorSwatch`, `BrandCard`, `Section`, `Button`, `Input`, `ErrorAlert`

## 6. Target End State

1. BrandListPage has an "Import from Scenario" button alongside "New Brand"
2. Clicking it navigates to a discovery page with a text input for scenario name
3. After entering a scenario name and triggering discovery, the page shows:
   - Overall confidence indicator (high/medium/low with color coding)
   - Source-by-source breakdown table (file, type, confidence, fields count)
   - Draft brand preview: identity (name, tagline), color swatches, typography, assets
   - Suggestions list for missing branding elements
4. User can: Accept & Import, Edit Before Importing, or Cancel
5. Accept & Import creates the brand and navigates to BrandDetailPage
6. Edit Before Importing navigates to BrandFormPage pre-filled with discovered draft values

## 7. Implementation Strategy

### Phase 1: API Client & Types
- Add `DiscoveryResult` and `DiscoverySource` TypeScript interfaces to `api.ts`
- Add `discoverScenario(scenario: string)` function
- Add `importDiscovery(scenario: string)` function

### Phase 2: Router & Discovery Page
- Add `discover` and `discover-result` routes to `router.ts`
- Create `DiscoveryPage.tsx` with:
  - Scenario name input (text field, reusing existing Input component)
  - "Discover" button to trigger the scan
  - Results display area
- Register in `App.tsx` with lazy loading

### Phase 3: Discovery Results Display
- Confidence indicator component (reuse existing color/styling patterns)
- Source breakdown table
- Draft brand preview section (reuse `ColorSwatch` for colors)
- Suggestions list
- Action buttons: "Import Brand", "Edit Before Importing", "Cancel"

### Phase 4: Import & Navigation Flow
- "Import Brand" → calls `importDiscovery()` → navigates to `/brands/{id}`
- "Edit Before Importing" → encodes draft brand as query params or state → navigates to `/brands/new?prefill=...`
- "Cancel" → navigates back to `/brands`

## 8. Contract Decisions

### API Functions
```typescript
// New types
interface DiscoverySource {
  file: string;
  type: "service_json" | "branding_json" | "theme_css" | "manifest" | "asset";
  confidence: number;
  fields: number;
}

interface DiscoveryResult {
  scenario: string;
  sources: DiscoverySource[];
  draft_brand?: Brand;
  confidence: number;
  suggestions?: string[];
}

interface ImportResult {
  brand: Brand;
  sources: DiscoverySource[];
  confidence: number;
}

// New functions
discoverScenario(scenario: string): Promise<DiscoveryResult>
importDiscovery(scenario: string): Promise<ImportResult>
```

### Routes
- `/discover` → DiscoveryPage (scenario input + results)
- Hash-based, consistent with existing routing pattern

### Pre-fill Strategy for "Edit Before Importing"
<!-- TBD — depends on decision d2 -->

## 9. Testing Plan

<!-- TBD — depends on decision d3 -->

## 10. Rollout/Validation Checklist

- [ ] API client functions work against running brand-manager API
- [ ] Discovery page renders and handles loading/error/empty states
- [ ] Source breakdown displays correctly for scenarios with varying branding state
- [ ] Color swatches render for discovered colors
- [ ] Import creates brand and navigates to detail page
- [ ] Edit-before-import opens form pre-filled with discovered data
- [ ] Cancel returns to brand list
- [ ] No regressions on existing pages

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scenario picker not yet built | Users must type scenario name manually | Use simple text input (same as ScannerPage); scenario picker is a separate backlog item |
| Large discovery results for scenarios with many assets | UI clutter | Cap source display, collapse by default |
| Pre-fill state lost on navigation | User loses discovered data when going to form | Decide on state transfer mechanism (see d2) |
| Backend returns unexpected shapes | UI crash | Type-safe parsing + ErrorAlert fallback |

## 12. Non-goals / Prohibited Patterns

- Do NOT modify backend discovery endpoints
- Do NOT build a full scenario picker (separate backlog item)
- Do NOT add automatic post-import scenario assignment
- Do NOT introduce new state management beyond React Query + local state

## 13. Definition of Done

- "Import from Scenario" button on BrandListPage
- Discovery page with scenario input, results display, and action buttons
- `discoverScenario()` and `importDiscovery()` API client functions
- Import flow creates brand and navigates correctly
- Edit-before-import pre-fills the brand form
- All new code follows existing patterns (Tailwind dark theme, React Query, hash routing)
- Tests pass (per testing plan)
