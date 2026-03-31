# Implementation Plan: Build UI for Discovering and Importing Existing Scenario Branding

## 1. Purpose

Add a discovery/import flow to the Brand Manager UI so users can create brands from scenarios that already have branding elements (colors, logos, manifest.json, etc.), rather than starting from scratch. The flow is presented as a modal overlay on BrandListPage.

## 2. Required Reading

```bash
prompt-manager skill read brand-manager react-coherence ux implementation-plan-authoring
```

**Key files:**
- Discovery handler: `scenarios/brand-manager/api/handlers/discovery.go`
- Domain types: `scenarios/brand-manager/api/domain/types.go`
- Brand list page: `scenarios/brand-manager/ui/src/pages/BrandListPage.tsx`
- Brand form page: `scenarios/brand-manager/ui/src/pages/BrandFormPage.tsx`
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
- Modal/overlay discovery flow on BrandListPage showing results (sources, confidence, draft brand preview, suggestions)
- Import action that creates the brand and navigates to BrandDetailPage
- "Edit before importing" action that stores draft in sessionStorage and navigates to BrandFormPage pre-filled
- API client functions for `discoverScenario()` and `importDiscovery()`
- TypeScript types mirroring `DiscoveryResult` and `DiscoverySource`
- Modification to BrandFormPage to read pre-fill data from sessionStorage
- React component tests (vitest + testing-library) for new components
- Go integration tests for API client contract validation

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
- No modal/dialog component exists (only native `confirm()` for delete)
- Hash-based routing via `useRouter()` in `router.ts`
- UI uses Tailwind (dark theme, slate-950 bg), React Query for data fetching, lucide-react for icons
- Existing components: `ColorSwatch`, `BrandCard`, `Section`, `Button`, `Input`, `ErrorAlert`
- BrandFormPage accepts only `brandId` prop — uses `emptyForm` default state, populates from API when editing
- `brandToForm()` helper converts Brand → FormState; can be reused for discovery pre-fill
- Vitest + Testing Library already configured (vitest 2.1.4, @testing-library/react 16.1.0, jsdom)

## 6. Target End State

1. BrandListPage has an "Import from Scenario" button alongside "New Brand"
2. Clicking it opens a modal overlay with a text input for scenario name
3. After entering a scenario name and triggering discovery, the modal shows:
   - Overall confidence indicator (traffic-light badge: High ≥0.7 / Medium ≥0.4 / Low <0.4)
   - Source-by-source breakdown table (file, type, confidence, fields count)
   - Draft brand preview: identity (name, tagline), color swatches, typography, assets
   - Suggestions list for missing branding elements
4. User can: Accept & Import, Edit Before Importing, or Cancel
5. Accept & Import creates the brand and navigates to BrandDetailPage
6. Edit Before Importing stores draft brand in sessionStorage and navigates to BrandFormPage, which reads and clears it on mount

## 7. Implementation Strategy

### Phase 1: API Client & Types
- Add `DiscoveryResult`, `DiscoverySource`, and `ImportResult` TypeScript interfaces to `api.ts`
- Add `discoverScenario(scenario: string)` function
- Add `importDiscovery(scenario: string)` function

### Phase 2: Discovery Modal Component
- Create `DiscoveryModal.tsx` component with:
  - Backdrop overlay (click-outside to close)
  - Scenario name input (text field, reusing existing Input component)
  - "Discover" button to trigger the scan
  - Loading/error/results states
  - Results display: confidence badge, source table, draft brand preview, suggestions
  - Action buttons: "Import Brand", "Edit Before Importing", "Cancel"
- Integrate into BrandListPage with state toggle

### Phase 3: Discovery Results Display (within modal)
- Confidence badge component (traffic-light: green/yellow/red with High/Medium/Low label)
- Source breakdown table (collapsible rows)
- Draft brand preview section (reuse `ColorSwatch` for colors, inline text for identity/typography)
- Suggestions list with info styling

### Phase 4: Import & Pre-fill Flows
- "Import Brand" → calls `importDiscovery()` → closes modal → navigates to `/brands/{id}`
- "Edit Before Importing" → stores `draft_brand` in `sessionStorage` under key `discovery-prefill` → navigates to `/brands/new`
- Modify BrandFormPage to check `sessionStorage.getItem('discovery-prefill')` on mount, parse as Brand, call `brandToForm()`, and clear storage
- "Cancel" → closes modal

### Phase 5: Tests
- React component tests for DiscoveryModal (render states, user interactions, API mocking)
- Test for BrandFormPage sessionStorage pre-fill behavior
- Go integration tests validating API client contract

## 8. Contract Decisions

### API Functions
```typescript
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

discoverScenario(scenario: string): Promise<DiscoveryResult>
importDiscovery(scenario: string): Promise<ImportResult>
```

### Modal Integration
- BrandListPage manages `showDiscovery: boolean` state
- DiscoveryModal receives `isOpen`, `onClose`, `onNavigate` props
- Modal renders conditionally based on `isOpen`

### Pre-fill Strategy (decided: sessionStorage)
- Key: `discovery-prefill`
- Value: JSON-serialized `Brand` object from `discovery_result.draft_brand`
- BrandFormPage reads on mount: `const prefill = sessionStorage.getItem('discovery-prefill')`
- If present: `setForm(brandToForm(JSON.parse(prefill)))`, then `sessionStorage.removeItem('discovery-prefill')`
- Tab-scoped, auto-cleans on close, no new dependencies

## 9. Testing Plan

### React Component Tests (vitest + testing-library)
- **DiscoveryModal.test.tsx:**
  - Renders nothing when `isOpen=false`
  - Shows scenario input and discover button when open
  - Shows loading state during discovery
  - Shows error state on API failure
  - Renders confidence badge, source table, draft preview, suggestions on success
  - "Import Brand" calls importDiscovery and onNavigate
  - "Edit Before Importing" writes sessionStorage and calls onNavigate
  - "Cancel" calls onClose
  - Backdrop click calls onClose
- **BrandFormPage pre-fill test:**
  - When sessionStorage has `discovery-prefill`, form initializes with discovered values
  - sessionStorage is cleared after reading

### Go Integration Tests
- Existing discovery handler tests cover API contract
- Validate response shapes match TypeScript interfaces

## 10. Rollout/Validation Checklist

- [ ] API client functions work against running brand-manager API
- [ ] Modal opens/closes correctly from BrandListPage
- [ ] Discovery handles loading/error/empty/results states
- [ ] Source breakdown displays correctly for scenarios with varying branding state
- [ ] Color swatches render for discovered colors
- [ ] Confidence badge shows correct tier (high/medium/low)
- [ ] Import creates brand and navigates to detail page
- [ ] Edit-before-import stores to sessionStorage and opens form pre-filled
- [ ] sessionStorage is cleared after form reads it
- [ ] Cancel closes modal, returns to brand list
- [ ] No regressions on existing pages
- [ ] All React component tests pass
- [ ] All Go tests pass

## 11. Risks + Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Scenario picker not yet built | Users must type scenario name manually | Use simple text input (same as ScannerPage); scenario picker is a separate backlog item |
| No existing modal component | Must build modal from scratch | Minimal implementation: backdrop + centered panel with Tailwind; matches dark theme |
| Large discovery results for scenarios with many assets | UI clutter in modal | Cap source display, collapse by default, scrollable modal body |
| Pre-fill state lost if user refreshes mid-flow | Acceptable since sessionStorage persists per tab | sessionStorage survives refresh; only lost on tab close (acceptable UX) |
| Backend returns unexpected shapes | UI crash | Type-safe parsing + ErrorAlert fallback |
| Modal accessibility | Screen readers may not detect overlay | Add aria-modal, role="dialog", focus trap, Escape key handler |

## 12. Non-goals / Prohibited Patterns

- Do NOT modify backend discovery endpoints
- Do NOT build a full scenario picker (separate backlog item)
- Do NOT add automatic post-import scenario assignment
- Do NOT introduce new state management beyond React Query + local state + sessionStorage
- Do NOT add a routing library — keep hash-based routing

## 13. Definition of Done

- "Import from Scenario" button on BrandListPage
- Modal overlay with scenario input, discovery results display, and action buttons
- `discoverScenario()` and `importDiscovery()` API client functions
- Import flow creates brand and navigates correctly
- Edit-before-import pre-fills the brand form via sessionStorage
- Traffic-light confidence badge (High/Medium/Low)
- React component tests for DiscoveryModal and pre-fill behavior
- All new code follows existing patterns (Tailwind dark theme, React Query, hash routing)
- All tests pass (React + Go)
