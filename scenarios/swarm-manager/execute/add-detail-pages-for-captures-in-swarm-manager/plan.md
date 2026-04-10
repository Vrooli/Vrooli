# Implementation Plan: Add Detail Pages for Captures in Swarm-Manager

## 1. Purpose

Add full detail/view pages for captures in the swarm-manager UI, consistent with existing detail pages for backlog items, scenarios, executions, and initiatives. Users should be able to click a capture to view its full information in a dedicated overlay page.

## Required Reading

```bash
prompt-manager skill read react-coherence ux implementation-plan-authoring
```

**Key files to read before implementing:**
- `scenarios/swarm-manager/ui/src/pages/BacklogDetailsPage.tsx` — reference detail page pattern
- `scenarios/swarm-manager/ui/src/components/detail/DetailPageLayout.tsx` — shared layout wrapper
- `scenarios/swarm-manager/ui/src/components/detail/DetailPageHeader.tsx` — shared header
- `scenarios/swarm-manager/ui/src/stores/detail-selection-store.ts` — detail selection state
- `scenarios/swarm-manager/ui/src/hooks/useDetailUrlSync.ts` — URL ↔ store sync
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphWorkspace.tsx` — overlay rendering (lines ~414-424)
- `scenarios/swarm-manager/ui/src/components/capture/capture-card.tsx` — existing capture display + triage actions
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/CapturesTab.tsx` — capture list in sidebar
- `scenarios/swarm-manager/ui/src/services/capture-service.ts` — capture API service layer
- `scenarios/swarm-manager/ui/src/stores/capture-store.ts` — capture list state + persistence

## 2. Problem Statement

Captures in the swarm-manager UI are displayed only in the sidebar CapturesTab via inline CaptureCard components. Unlike backlog items, scenarios, executions, and initiatives, captures:
- Are not clickable to open a detail view
- Have no dedicated full-page overlay
- Cannot be deep-linked via URL parameters
- Have no entry in the detail-selection-store

This makes it difficult to inspect full capture details, view attachments at full size, review classification results, and share links to specific captures.

## 3. Scope

### In scope
- New `CaptureDetailsPage` component following existing detail page patterns
- Extend `detail-selection-store` to support `entityType: "capture"`
- Extend `useDetailUrlSync` to handle capture URL params (`detail=capture&id=...`)
- Extend `GraphWorkspace` overlay rendering for capture detail pages
- Make captures clickable in CapturesTab sidebar
- Display: raw text, attachments (full-size with lightbox), classification results, timestamps, status
- Triage actions on the detail page via shared CaptureTriage component
- Extract shared triage component from CaptureCard for reuse
- Custom lightbox modal for full-resolution attachment viewing

### Out of scope
- Capture editing (captures are raw input, not meant to be edited after creation)
- Adding captures as graph nodes
- Capture search/filtering on the detail page
- New API endpoints (existing `GET /api/v1/captures/{id}` is sufficient)
- Third-party lightbox libraries

### acceptance_allow
```
scenarios/swarm-manager/ui/**
```

## 4. Current Technical Context

### Existing detail page architecture
All detail pages follow the same pattern:
1. `detail-selection-store` (Zustand) holds `detailSelection: { entityType, ...identifiers, tab }`
2. `useDetailUrlSync` syncs store ↔ URL query params bidirectionally
3. `GraphWorkspace` renders the appropriate page as an absolute overlay (z-40)
4. Pages use `DetailPageLayout` + `DetailPageHeader` shared components
5. Pages are lazy-loaded via `React.lazy`

### Detail selection store structure
```typescript
type DetailEntityType = "backlog" | "scenario" | "execution" | "initiative" | "capture";

interface DetailSelection {
  entityType: DetailEntityType;
  kind?: string;        // Backlog kind
  name?: string;        // Entity name
  identifier?: string;  // Execution ID or Capture ID
  tab?: string;         // Active tab
}
```

### selectionToNodeId behavior
`selectionToNodeId()` returns the graph node ID for highlighting. Since captures are NOT graph nodes, the existing `default: return null` case handles this correctly — no code change needed (settled R2-d2=A).

### Capture data model
```typescript
interface Capture {
  id: string;
  text: string;
  attachments: string[];
  created: string;
  status: "classifying" | "classified" | "failed";
  classification: CaptureClassification | null;
}
```

### Existing API (confirmed working)
- `GET /api/v1/captures/{id}` — returns full capture data
- `DELETE /api/v1/captures/{id}` — delete capture
- `POST /api/v1/captures/{id}/classify` — retry classification
- `POST /api/v1/captures/{id}/create-item` — accept classification → create backlog item

### Capture service layer
`captureService` (in `capture-service.ts`) already provides:
- `list()`, `get(id)`, `create()`, `remove(id)`, `classify(id)`
- `get(id)` returns a fully mapped `Capture` object — ready for detail page use

## 5. Target End State

- Clicking any capture in the sidebar opens `CaptureDetailsPage` as a full overlay
- URL updates to `/graph?detail=capture&id={captureId}` for deep-linking
- Detail page shows: full text, full-size attachment gallery with custom lightbox, classification results with triage actions, metadata (created date, status)
- Back/close returns to graph with sidebar state preserved
- Mobile-responsive layout following existing detail page patterns
- Shared `CaptureTriage` component used by both CaptureCard and CaptureDetailsPage
- Custom lightbox modal (dark backdrop + centered image + close button) for full-resolution attachment viewing, consistent with existing overlay patterns

## 6. Implementation Strategy

### Phase 1: Store & Routing Infrastructure
1. **Extend `DetailEntityType`** in `detail-selection-store.ts`:
   - Add `"capture"` to the union type
   - Add `selectCapture(captureId: string)` action that sets `{ entityType: "capture", identifier: captureId }`
   - Reuse `identifier` field (same as executions) for capture ID
   - `selectionToNodeId` needs no changes — default case returns null (R2-d2=A)
2. **Extend `useDetailUrlSync`**:
   - Add case for `detail=capture` reading `id` param → hydrate as `{ entityType: "capture", identifier: id }`
   - Add serialization case for capture → `detail=capture&id={identifier}`
3. **Add lazy import** for `CaptureDetailsPage` in `GraphWorkspace.tsx`
4. **Add render case** in the overlay block: `{detailSelection.entityType === "capture" && <CaptureDetailsPage />}`

### Phase 2: Shared Triage Component Extraction
1. **Create `CaptureTriage.tsx`** in `components/capture/`:
   - Extract suggestion list rendering + per-item accept/edit/dismiss from CaptureCard (R2-d4=A)
   - Include "Accept all" batch action
   - Props: `capture: Capture`, `onEditItem`, `onCaptureUpdate` callbacks
   - Parent components handle delete/retry/dismiss-capture (context-specific)
2. **Refactor `CaptureCard`** to use `CaptureTriage` internally
3. Verify CaptureCard behavior is unchanged after extraction

### Phase 3: Detail Page Component
1. **Create `CaptureDetailsPage.tsx`** in `pages/`:
   - **Data fetching**: Read capture from store first (instant for sidebar click-through), fall back to `captureService.get(id)` via useQuery for deep-links (R2-d3=A)
   - Use `DetailPageLayout` + `DetailPageHeader` with entity type "Capture"
   - **Layout** — single scrollable page with stacked sections (R1-d1=A):
     - **Capture text** — full raw text display
     - **Attachments** — full-width inline images with click-to-lightbox (R1-d2=A)
     - **Classification** — `CaptureTriage` component (R1-d3=A, R2-d4=A)
     - **Status** — classifying spinner, failed state with retry, no-op state
     - **Metadata** — created timestamp, capture ID
   - Header actions: delete capture, retry classification (if failed)

### Phase 4: Click-Through Navigation
1. **Make CaptureCard clickable** in CapturesTab:
   - Card background click → `selectCapture(capture.id)` (R1-d4=A)
   - Triage buttons use `event.stopPropagation()` to prevent navigation
   - Add `cursor-pointer` and hover state to card
2. Wire up CapturesTab's `onItemClick` or add direct store access

### Phase 5: Lightbox & Polish
1. **Custom lightbox modal** (R2-d1=A):
   - Dark backdrop overlay (consistent with existing z-40+ overlay patterns)
   - Centered full-resolution image
   - Close button + click-outside-to-close + Escape key
   - No third-party dependencies — use existing Dialog/overlay patterns from the codebase
2. Mobile-responsive layout for the detail page
3. Test deep-linking: direct URL navigation to capture detail
4. Ensure virtual keyboard doesn't break layout

## 7. Contract Decisions

### Settled (Round 1)
| ID | Decision | Choice |
|----|----------|--------|
| R1-d1 | Detail page content layout | Single scrollable page with stacked sections |
| R1-d2 | Attachment viewing experience | Inline full-width images with click-to-lightbox |
| R1-d3 | Classification triage actions | Extract shared triage component from CaptureCard |
| R1-d4 | Click target for opening detail | Card background click; triage buttons use stopPropagation |

### Settled (Round 2)
| ID | Decision | Choice |
|----|----------|--------|
| R2-d1 | Lightbox implementation | Custom modal overlay using existing Dialog/overlay patterns (no library) |
| R2-d2 | selectionToNodeId for captures | Return null via existing default case; no code change needed |
| R2-d3 | Data fetching strategy | Store-first with captureService.get(id) fallback for deep-links |
| R2-d4 | Shared triage extraction scope | Single CaptureTriage component (suggestion list + per-item actions + accept all) |

## 8. Testing Plan

### Unit Tests
- `detail-selection-store`: test `selectCapture()`, verify `selectionToNodeId` returns null for captures
- `useDetailUrlSync`: test bidirectional sync with `detail=capture&id=...` params
- `CaptureTriage`: test accept/edit/dismiss per item, "accept all", auto-dismiss behavior
- `CaptureDetailsPage`: test rendering for all 3 capture statuses (classifying, classified, failed)
- Lightbox: test open/close, Escape key, click-outside-to-close

### Integration Tests
- Click CaptureCard in sidebar → detail page opens with correct capture
- Deep-link to `/graph?detail=capture&id=X` → detail page renders
- Triage action on detail page → capture store updates, UI reflects change
- Close detail page → returns to graph, sidebar preserved

### Regression Tests
- Existing detail pages (backlog, scenario, execution, initiative) still work
- CaptureCard inline triage still works after CaptureTriage extraction
- CaptureCard triage buttons don't trigger navigation (stopPropagation)

## 9. Rollout/Validation Checklist

- [ ] CaptureDetailsPage renders for all capture statuses (classifying, classified, failed)
- [ ] URL updates correctly when opening/closing capture detail
- [ ] Deep-link to capture detail works on fresh page load
- [ ] Triage actions (accept, edit, dismiss, delete, retry) work from detail page
- [ ] Attachments display at full size with custom lightbox modal
- [ ] Mobile layout is usable
- [ ] CaptureCard still works identically after triage extraction
- [ ] Existing detail pages (backlog, scenario, execution, initiative) are unaffected
- [ ] Shared CaptureTriage component renders correctly in both CaptureCard and detail page contexts
- [ ] Lightbox opens/closes correctly (click, Escape, click-outside)

## 10. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| CaptureCard click conflicts with inline triage buttons | Medium | Medium | Use event.stopPropagation on triage buttons; click on card background opens detail |
| CaptureTriage extraction breaks CaptureCard behavior | Medium | Medium | Extract carefully; run existing tests after refactor; verify identical behavior |
| Capture polling disrupts detail page | Low | Low | Detail page reads from capture store, which is already updated by polling |
| Deep-link to deleted capture | Low | Low | Show "Capture not found" state, offer return to graph |
| Identifier field overload (used for both execution ID and capture ID) | Low | Low | EntityType discriminates usage; selectionToNodeId already handles unknown types gracefully |
| Store-first fetch returns stale data after external mutation | Low | Low | Store is refreshed by polling; deep-link path fetches fresh from API |

## 11. Non-goals / Prohibited Patterns

- Do NOT add captures as graph nodes (separate concern, not in scope)
- Do NOT create new API endpoints — use existing capture API
- Do NOT duplicate triage logic — use shared CaptureTriage component
- Do NOT add capture editing capability
- Do NOT add new npm dependencies for lightbox — use custom modal with existing patterns
- Do NOT modify selectionToNodeId — the default null return handles captures correctly

## 12. Definition of Done

- CaptureDetailsPage exists and renders for all capture statuses
- Captures are clickable in sidebar, opening the detail page
- URL-based deep-linking works for captures
- Triage actions work from detail page via shared CaptureTriage component
- Attachments shown full-size with custom lightbox modal
- Mobile-responsive
- CaptureCard still works identically (no regression from triage extraction)
- No regressions to existing detail pages

## 13. Phased Delivery Summary

| Phase | What | Key Files |
|-------|------|-----------|
| 1 | Store + routing | detail-selection-store.ts, useDetailUrlSync.ts, GraphWorkspace.tsx |
| 2 | Triage extraction | capture-card.tsx → CaptureTriage.tsx (new) |
| 3 | Detail page | CaptureDetailsPage.tsx (new) |
| 4 | Click-through | CapturesTab.tsx, capture-card.tsx |
| 5 | Lightbox + polish | CaptureDetailsPage.tsx, lightbox component |
