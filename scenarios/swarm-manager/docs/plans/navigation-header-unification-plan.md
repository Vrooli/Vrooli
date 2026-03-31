# Navigation & Header Unification — Implementation Plan

## 1. Purpose

Unify the navigation, header/toolbar, and sidebar interaction model across all Swarm Manager entity detail pages into a single, production-ready system inspired by Prompt Manager's proven patterns. This is a **greenfield rewrite** of the detail page header/toolbar layer — no legacy compatibility, no dead code, no migration bridges.

## 2. Required Reading

```bash
prompt-manager skill read implementation-plan-authoring documentation-health seam-discovery-and-enforcement utils-unification test
```

Key files to read before implementing:

```bash
# Current shared detail components
cat scenarios/swarm-manager/ui/src/components/detail/DetailPageHeader.tsx
cat scenarios/swarm-manager/ui/src/components/detail/DetailPageLayout.tsx
cat scenarios/swarm-manager/ui/src/components/detail/LensBar.tsx
cat scenarios/swarm-manager/ui/src/components/detail/DetailActionButtons.tsx

# Current stores
cat scenarios/swarm-manager/ui/src/stores/detail-selection-store.ts
cat scenarios/swarm-manager/ui/src/surfaces/graph/stores/graph-ui-store.ts

# Current URL sync
cat scenarios/swarm-manager/ui/src/hooks/useDetailUrlSync.ts
cat scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/useSidebarUrlSync.ts
cat scenarios/swarm-manager/ui/src/hooks/useDrillToLens.ts

# Sidebar and workspace
cat scenarios/swarm-manager/ui/src/surfaces/graph/components/sidebar/Sidebar.tsx
cat scenarios/swarm-manager/ui/src/surfaces/graph/components/GraphWorkspace.tsx

# All detail pages
cat scenarios/swarm-manager/ui/src/pages/ExecutionDetailsPage.tsx
cat scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.tsx
cat scenarios/swarm-manager/ui/src/pages/ScenarioDetailsPage.tsx
# BacklogDetailsPage is ~2753 lines — read in sections

# Prompt Manager reference (navigation pattern we're adopting)
cat scenarios/prompt-manager/ui/src/components/layout/SkillManagerLayout.tsx
```

## 3. Problem Statement

### Root Cause
The Swarm Manager detail page navigation grew organically: BacklogDetailsPage was built first with its own bespoke header (~2753 lines in one file), then a shared `DetailPageHeader` + `DetailPageLayout` was introduced for Scenario/Execution/Initiative pages. Neither toolbar works correctly, and the interaction between detail pages and the sidebar is broken.

### Observed Issues

1. **Close button always dumps sidebar state.** Opening a detail page from the sidebar force-closes the sidebar on mobile (`GraphWorkspace.tsx` lines 277-280). Closing the detail page does NOT reopen the sidebar. The user loses their sidebar context.

2. **X button is the wrong metaphor.** The X implies "dismiss and go back to where I was," but it always navigates to the bare graph regardless of origin. If you came from the sidebar, you expect to return to the sidebar.

3. **BacklogDetailsPage doesn't use the shared header.** It has its own inline header (tabs, lens bar, actions) built directly in the 2753-line file. Improvements to the shared components don't apply to it.

4. **No sidebar access from detail pages.** Once a detail page is open, there's no way to reopen the sidebar without first closing the detail page. You're locked into a "detail XOR sidebar" binary.

5. **Two broken toolbar systems** with divergent UX patterns, styling, and behavior — neither production-ready.

6. **URL parameter systems don't coordinate.** `useDetailUrlSync` and `useSidebarUrlSync` manage separate parameter namespaces with no awareness of each other. `useDrillToLens` explicitly deletes sidebar tab params when navigating to a lens.

## 4. Scope

### In Scope
- Greenfield rewrite of `DetailPageHeader` → unified header for ALL entity types
- Greenfield rewrite of `DetailPageLayout` → shared layout for ALL entity types
- Sidebar state preservation across detail open/close
- Hamburger button on mobile (opens sidebar) / back arrow on desktop (closes detail)
- Home button in sidebar header to return to graph
- Refactor BacklogDetailsPage to use the shared header/layout system
- Break BacklogDetailsPage into composable modules (it's 2753 lines)
- URL coordination between detail and sidebar sync hooks
- Comprehensive test coverage for all new/changed components and hooks
- Internal documentation (SEAMS.md updates, DOC: references)

### Out of Scope
- Graph visualization changes
- Sidebar content/filtering/sorting changes (only sidebar open/close behavior)
- Entity-specific business logic changes
- New entity types or detail page content
- API/backend changes

## 5. Current Technical Context

### Key Files & Their Roles

| File | Lines | Role | Disposition |
|------|-------|------|-------------|
| `components/detail/DetailPageHeader.tsx` | 73 | Shared header (X button, title, actions) | **Rewrite** |
| `components/detail/DetailPageLayout.tsx` | 79 | Layout wrapper + mobile FAB | **Rewrite** |
| `components/detail/LensBar.tsx` | ~50 | Cross-lens navigation buttons | Keep, integrate into new header |
| `components/detail/DetailActionButtons.tsx` | ~80 | Action registry integration | Keep, wire into new header action slot |
| `components/detail/lens-options.ts` | ~30 | Lens config per entity type | Keep as-is |
| `components/detail/StatusBadge.tsx` | ~30 | Status pill rendering | Keep as-is |
| `pages/BacklogDetailsPage.tsx` | 2753 | Backlog detail (monolith) | **Decompose + adopt shared header** |
| `pages/ExecutionDetailsPage.tsx` | ~180 | Execution detail | Adopt new header |
| `pages/InitiativeDetailsPage.tsx` | ~250 | Initiative detail | Adopt new header |
| `pages/ScenarioDetailsPage.tsx` | ~400 | Scenario detail | Adopt new header |
| `stores/detail-selection-store.ts` | 90 | Detail overlay state | Minor additions |
| `surfaces/graph/stores/graph-ui-store.ts` | 301 | Graph UI state incl. sidebar | Minor additions |
| `hooks/useDetailUrlSync.ts` | 173 | Detail ↔ URL sync | Update for sidebar coordination |
| `hooks/useDrillToLens.ts` | ~50 | Lens navigation + sidebar cleanup | Update |
| `surfaces/graph/components/GraphWorkspace.tsx` | ~430 | Main workspace, sidebar interaction | Update sidebar handling |
| `surfaces/graph/components/sidebar/Sidebar.tsx` | ~80 | Sidebar container | Add home button |
| `surfaces/graph/components/sidebar/useSidebarUrlSync.ts` | ~140 | Sidebar ↔ URL sync | Coordinate with detail sync |

### State Architecture

```
useGraphUIStore (Zustand)
├── sidebarCollapsed: boolean  (persisted in localStorage)
├── toggleSidebar()
└── setSidebarCollapsed()

useDetailSelectionStore (Zustand)
├── selection: DetailSelection | null
├── selectBacklog/Scenario/Execution/Initiative()
└── clearSelection()

URL params (react-router-dom useSearchParams)
├── Detail: detail, kind, name, execId, tab
├── Sidebar: sidebar.tab, sidebar.sortField, sidebar.sortDirection, sidebar.statuses, sidebar.kinds, sidebar.modes
└── Graph: lens, select, focus, returnLens
```

## 6. Target End State

### Navigation Model (Prompt Manager Pattern)

```
┌──────────────────────────────────────────────────────────┐
│  Detail Page Header (shared across ALL entity types)     │
│  ┌────────┐  ┌──────────┐  ┌───────────────┐  ┌──────┐  │
│  │ ☰ / ←  │  │ Type     │  │ Title         │  │ Acts │  │
│  │ mobile/ │  │ Badge    │  │ + subtitle    │  │      │  │
│  │ desktop │  │          │  │               │  │      │  │
│  └────────┘  └──────────┘  └───────────────┘  └──────┘  │
│  ┌───────────────────────────────────────────────────┐   │
│  │ LensBar (drill-to-graph buttons)                  │   │
│  └───────────────────────────────────────────────────┘   │
│  ┌───────────────────────────────────────────────────┐   │
│  │ Entity-specific tab bar (optional, e.g. backlog)  │   │
│  └───────────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────────────┤
│  Entity-specific content                                 │
└──────────────────────────────────────────────────────────┘
```

### Navigation Button Behavior

| Context | Button | Icon | Action |
|---------|--------|------|--------|
| Mobile, detail open | Primary nav button | Menu (☰) | Opens sidebar drawer over detail page |
| Desktop, detail open | Primary nav button | ArrowLeft (←) | Closes detail, restores sidebar state |
| Sidebar header | Home button | Home | Clears detail selection + closes sidebar → graph |
| Detail header | LensBar button | Per-lens icon | Clears detail + drills to lens (closes sidebar) |

### Sidebar State Preservation

```
Open detail from sidebar (mobile):
  1. Store sidebarWasOpen = true
  2. Sidebar visually hidden (detail overlay on top)
  3. Hamburger in detail header reopens sidebar as drawer over detail
  4. Close detail → sidebar reopens automatically (sidebarWasOpen was true)

Open detail from graph node (mobile):
  1. Store sidebarWasOpen = false (sidebar wasn't open)
  2. Close detail → sidebar stays closed

Drill to lens from detail:
  1. Always close sidebar (you're explicitly asking to see the graph)
  2. Clear sidebarWasOpen

Refresh with detail open:
  - Sidebar state restored from localStorage (collapsed or expanded)
  - Detail restored from URL params
  - Both coexist correctly
```

### File Organization (Target)

```
src/components/detail/
├── DetailPageHeader.tsx          # Shared header (hamburger/back, title, actions, lens bar)
├── DetailPageHeader.test.tsx     # Header tests
├── DetailPageLayout.tsx          # Shared layout wrapper
├── DetailPageLayout.test.tsx     # Layout tests
├── DetailActionButtons.tsx       # Action registry (keep)
├── LensBar.tsx                   # Cross-lens buttons (keep)
├── LensBar.test.tsx              # (existing)
├── lens-options.ts               # Lens configs (keep)
├── StatusBadge.tsx               # Status pill (keep)
├── index.ts                      # Public barrel export
└── types.ts                      # Shared detail types (NEW)

src/components/backlog/
├── BacklogHeader.tsx             # Backlog-specific header content (tabs, file actions)
├── BacklogHeader.test.tsx        # Tests
├── BacklogInfoTab.tsx            # Info tab content extracted from BacklogDetailsPage
├── BacklogPromptTab.tsx          # Prompt tab content extracted
├── BacklogFilesTab.tsx           # Files tab content extracted
└── ... (existing backlog components)

src/hooks/
├── useDetailNavigation.ts        # NEW: Shared navigation logic (open/close with sidebar awareness)
├── useDetailNavigation.test.ts   # Tests
├── useDetailUrlSync.ts           # Updated: coordinates with sidebar
├── useDetailUrlSync.test.ts      # Tests (NEW)
├── useDrillToLens.ts             # Updated: simplified
├── useDrillToLens.test.ts        # Tests (NEW)
└── ...

src/surfaces/graph/components/sidebar/
├── Sidebar.tsx                   # Updated: home button in header
├── Sidebar.test.tsx              # Tests (NEW)
├── SidebarHeader.tsx             # NEW: Home button + sidebar controls
├── SidebarHeader.test.tsx        # Tests
└── ...
```

## 7. Implementation Strategy

### Phase 1: Foundation — Shared Navigation Hook & Sidebar State Preservation

**Goal:** Fix the core navigation behavior without touching any visual components yet.

**Steps:**

1.1. **Create `useDetailNavigation` hook** (`src/hooks/useDetailNavigation.ts`)
   - Encapsulates all detail open/close logic with sidebar awareness
   - Tracks `sidebarWasOpen` state when detail opens
   - `openDetail(selection, { fromSidebar })` — stores sidebar state, sets selection
   - `closeDetail()` — clears selection, restores sidebar state based on `sidebarWasOpen`
   - `openSidebar()` — opens sidebar drawer (usable from detail pages)
   - Seam: sidebar state accessor/mutator injected via store references (testable without DOM)

1.2. **Add `sidebarWasOpen` to `graph-ui-store.ts`**
   - New field: `sidebarWasOpenBeforeDetail: boolean`
   - New actions: `saveSidebarStateBeforeDetail()`, `restoreSidebarStateAfterDetail()`
   - Persisted in localStorage alongside `sidebarCollapsed` for refresh resilience

1.3. **Update `GraphWorkspace.tsx`** sidebar interaction
   - Remove the inline `if (window.innerWidth < 768 && !sidebarCollapsed) toggleSidebar()` code
   - Replace with `useDetailNavigation().openDetail(selection, { fromSidebar: true })`
   - On detail close, delegate to `useDetailNavigation().closeDetail()` which handles restoration

1.4. **Update `useDrillToLens.ts`**
   - When drilling to a lens, explicitly set `sidebarWasOpenBeforeDetail = false` and close sidebar
   - Stop deleting the sidebar tab URL param (preserve user's sidebar tab preference)

1.5. **Write tests for `useDetailNavigation`**
   - Test: open from sidebar → close → sidebar reopens
   - Test: open from graph → close → sidebar stays closed
   - Test: drill to lens → sidebar closes regardless
   - Test: refresh preserves state

**Acceptance criteria:**
- Opening detail from sidebar, then closing, returns to sidebar-open state
- Opening detail from graph node, then closing, returns to sidebar-closed state
- Drilling to a lens always closes sidebar
- Page refresh with detail open preserves sidebar state correctly

---

### Phase 2: Unified Header Component

**Goal:** Rewrite `DetailPageHeader` and `DetailPageLayout` as the single header system for all entity types.

**Steps:**

2.1. **Define shared types** (`src/components/detail/types.ts`)
   ```typescript
   interface DetailHeaderConfig {
     entityType: string;
     title: string;
     subtitle?: string;
     status?: string;
     lenses: LensOption[];
     nodeId: string | null;
     actions?: ReactNode;
     tabBar?: ReactNode;        // Entity-specific tab bar (e.g., backlog info/prompt/files)
     mobileActions?: ReactNode; // Content for mobile action BottomSheet
   }
   ```

2.2. **Rewrite `DetailPageHeader.tsx`**
   - Mobile: hamburger (Menu icon) → calls `useDetailNavigation().openSidebar()`
   - Desktop: back arrow (ArrowLeft icon) → calls `useDetailNavigation().closeDetail()`
   - Entity type badge, title, subtitle, status badge (keep existing)
   - Inline action slot (keep existing)
   - LensBar integrated below the primary row
   - Optional tab bar slot below LensBar
   - Sticky on mobile, static on desktop (keep existing behavior)

2.3. **Rewrite `DetailPageLayout.tsx`**
   - Accept new `DetailHeaderConfig` or composed header
   - Mobile FAB for actions (keep existing pattern)
   - BottomSheet for mobile action overflow (keep existing)
   - Sidebar drawer overlay rendering when sidebar opened from detail page

2.4. **Write comprehensive tests**
   - Test: hamburger shows on mobile, back arrow on desktop
   - Test: hamburger click opens sidebar
   - Test: back arrow click closes detail
   - Test: LensBar renders correct lenses per entity type
   - Test: tab bar slot renders when provided
   - Test: action slot renders entity-specific actions
   - Test: mobile FAB appears when mobileActions provided
   - Test: sticky header behavior on mobile

**Acceptance criteria:**
- Single header component used by all 4 entity types
- Mobile shows hamburger, desktop shows back arrow
- LensBar and optional tab bar render correctly
- All existing action buttons continue to work

---

### Phase 3: Sidebar Header + Home Button

**Goal:** Add a home button to the sidebar header so users have an explicit "go to graph" action.

**Steps:**

3.1. **Create `SidebarHeader.tsx`** (`src/surfaces/graph/components/sidebar/SidebarHeader.tsx`)
   - Home button (Home icon) → clears detail selection + closes sidebar
   - Sidebar title/label
   - Close button (X) on mobile to dismiss sidebar drawer

3.2. **Update `Sidebar.tsx`**
   - Replace any existing header with `SidebarHeader`
   - When rendered as drawer over detail page (mobile), include backdrop for dismissal

3.3. **Write tests**
   - Test: home button clears detail selection and closes sidebar
   - Test: close button (mobile) dismisses sidebar without clearing detail
   - Test: backdrop click dismisses sidebar without clearing detail

**Acceptance criteria:**
- Home button in sidebar navigates to clean graph view
- Closing sidebar from mobile drawer returns to the detail page (not graph)

---

### Phase 4: BacklogDetailsPage Decomposition

**Goal:** Break the 2753-line monolith into focused modules and adopt the shared header.

**Steps:**

4.1. **Extract backlog-specific header content** → `BacklogHeader.tsx`
   - Tab bar (info/prompt/files)
   - File search/actions (when on files tab)
   - This becomes the `tabBar` slot in the shared `DetailPageHeader`

4.2. **Extract tab content into modules:**
   - `BacklogInfoTab.tsx` — metadata, status, priority, tags, dependencies, readiness, operational targets, workshop panel, plan/conclusion
   - `BacklogPromptTab.tsx` — prompt editing content
   - `BacklogFilesTab.tsx` — file tree, preview, upload, drag-and-drop

4.3. **Slim down `BacklogDetailsPage.tsx`**
   - Should become a thin orchestrator: queries data, selects tab, delegates to tab components
   - Uses shared `DetailPageLayout` + `DetailPageHeader` with `BacklogHeader` as tabBar
   - Target: under 300 lines

4.4. **Preserve all existing behavior** — this is purely structural decomposition
   - All existing functionality must work identically
   - Activity timeline drawer, workshop panel, etc. all preserved

4.5. **Write/update tests**
   - Existing `BacklogDetailsPage.test.tsx` (265 lines) serves as regression baseline
   - Add unit tests for each extracted module
   - Add integration test verifying the assembled page matches current behavior

**Acceptance criteria:**
- BacklogDetailsPage under 300 lines
- Uses shared DetailPageHeader + DetailPageLayout
- All existing features work (tabs, file tree, upload, workshop, activity timeline)
- Existing tests pass, new module-level tests added

---

### Phase 5: Migrate Remaining Detail Pages

**Goal:** Update Execution, Initiative, and Scenario detail pages to use the new header system.

**Steps:**

5.1. **Update `ExecutionDetailsPage.tsx`**
   - Replace `DetailPageHeader` + `DetailPageLayout` usage with new API
   - Pass entity-specific lenses, actions, no tab bar
   - Use `useDetailNavigation()` instead of raw `clearSelection()`

5.2. **Update `InitiativeDetailsPage.tsx`** — same pattern

5.3. **Update `ScenarioDetailsPage.tsx`** — same pattern

5.4. **Update/add tests for each**

**Acceptance criteria:**
- All 4 entity types use the same DetailPageHeader and DetailPageLayout
- No direct calls to `clearSelection()` from detail pages — all go through `useDetailNavigation()`
- Tests pass for all detail pages

---

### Phase 6: URL Sync Coordination & Cleanup

**Goal:** Ensure detail and sidebar URL params coexist correctly.

**Steps:**

6.1. **Update `useDetailUrlSync.ts`**
   - When writing detail params to URL, preserve existing sidebar params (already works, but verify)
   - When clearing detail params, preserve sidebar params

6.2. **Update `useSidebarUrlSync.ts`**
   - When writing sidebar params, preserve existing detail params
   - Remove any code that clears the other namespace's params

6.3. **Update `useDrillToLens.ts`**
   - Stop deleting sidebar tab param
   - Only clear detail params and set graph lens/focus params

6.4. **Write integration tests**
   - Test: open sidebar → open detail → URL has both sets of params
   - Test: close detail → sidebar params preserved
   - Test: drill to lens → sidebar params preserved, detail params cleared
   - Test: browser back/forward navigates correctly through detail+sidebar state changes

**Acceptance criteria:**
- Sidebar and detail URL params never clobber each other
- Browser back/forward works correctly through all navigation flows
- Deep-linking with both sidebar and detail params works

---

### Phase 7: Dead Code Removal & Final Polish

**Goal:** Remove all legacy code, verify no dead imports, ensure clean exports.

**Steps:**

7.1. Remove any old header/toolbar code not used by the new system
7.2. Clean up barrel exports in `components/detail/index.ts`
7.3. Verify no unused imports across all changed files
7.4. Run full test suite, fix any failures
7.5. Manual smoke test of all navigation flows on mobile and desktop viewports

**Acceptance criteria:**
- No dead code related to the old header system remains
- All tests pass
- All navigation flows work correctly on mobile and desktop

## 8. Contract Decisions

### Component API

```typescript
// DetailPageHeader accepts a config object, not individual props
interface DetailPageHeaderProps {
  config: DetailHeaderConfig;
}

// DetailPageLayout wraps the header + body
interface DetailPageLayoutProps {
  header: ReactNode;          // DetailPageHeader instance
  children: ReactNode;        // Page body
  mobileActions?: ReactNode;  // BottomSheet content
  mobileActionsTitle?: string;
}
```

### Hook API

```typescript
// useDetailNavigation — the single entry point for detail navigation
interface DetailNavigation {
  openDetail: (selection: DetailSelection, opts?: { fromSidebar?: boolean }) => void;
  closeDetail: () => void;
  openSidebar: () => void;
  drillToLens: (nodeId: string, targetLens: GraphLens) => void;
}
```

### Store Additions

```typescript
// graph-ui-store additions
interface GraphUIState {
  // ... existing ...
  sidebarWasOpenBeforeDetail: boolean;
  saveSidebarStateBeforeDetail: () => void;
  restoreSidebarStateAfterDetail: () => void;
}
```

### URL Parameter Ownership

| Namespace | Managed By | Params |
|-----------|-----------|--------|
| Detail | `useDetailUrlSync` | `detail`, `kind`, `name`, `execId`, `tab` |
| Sidebar | `useSidebarUrlSync` | `sidebar.tab`, `sidebar.sortField`, `sidebar.sortDirection`, `sidebar.statuses`, `sidebar.kinds`, `sidebar.modes` |
| Graph | `GraphWorkspace` / `useDrillToLens` | `lens`, `select`, `focus`, `returnLens` |

**Rule:** Each sync hook only writes/deletes params in its own namespace.

## 9. Testing Plan

### Unit Tests

| Component/Hook | Test File | Key Scenarios |
|---|---|---|
| `useDetailNavigation` | `useDetailNavigation.test.ts` | Open/close with sidebar awareness, drill-to-lens, refresh |
| `DetailPageHeader` | `DetailPageHeader.test.tsx` | Hamburger vs back arrow, click handlers, slots |
| `DetailPageLayout` | `DetailPageLayout.test.tsx` | Header rendering, FAB, BottomSheet |
| `SidebarHeader` | `SidebarHeader.test.tsx` | Home button, close button, navigation |
| `BacklogHeader` | `BacklogHeader.test.tsx` | Tab switching, file actions |
| `BacklogInfoTab` | `BacklogInfoTab.test.tsx` | Metadata display, actions |
| `BacklogFilesTab` | `BacklogFilesTab.test.tsx` | File tree, upload, preview |
| `useDetailUrlSync` | `useDetailUrlSync.test.ts` | Param preservation, deep-linking |
| `useDrillToLens` | `useDrillToLens.test.ts` | Lens navigation, param management |

### Integration Tests

| Flow | Verified By |
|---|---|
| Sidebar → detail → close → sidebar restored | `useDetailNavigation.test.ts` |
| Graph → detail → close → sidebar stays closed | `useDetailNavigation.test.ts` |
| Detail → drill to lens → graph with sidebar closed | `useDrillToLens.test.ts` |
| URL deep-link with detail + sidebar params | `useDetailUrlSync.test.ts` |
| Browser back/forward through navigation stack | `useDetailUrlSync.test.ts` |
| BacklogDetailsPage assembled correctly | `BacklogDetailsPage.test.tsx` |

### Regression Tests

- All existing tests in `BacklogDetailsPage.test.tsx` (265 lines) must pass
- All existing tests in `LensBar.test.tsx` must pass
- All existing tests in `InitiativeDetailsPage.test.tsx`, `ScenarioDetailsPage.test.tsx`, `ExecutionPage.test.tsx` must pass

## 10. Rollout/Validation Checklist

- [ ] Phase 1: `useDetailNavigation` hook + store updates + tests pass
- [ ] Phase 2: New `DetailPageHeader` + `DetailPageLayout` + tests pass
- [ ] Phase 3: `SidebarHeader` + home button + tests pass
- [ ] Phase 4: BacklogDetailsPage decomposed + all existing tests pass + new tests added
- [ ] Phase 5: All detail pages migrated + tests pass
- [ ] Phase 6: URL sync coordination verified + integration tests pass
- [ ] Phase 7: Dead code removed, full test suite green
- [ ] Smoke test: all navigation flows on mobile (< 768px) and desktop viewports

## 11. Risks + Mitigations

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| BacklogDetailsPage decomposition breaks subtle behavior | Medium | High | Existing test suite as regression baseline; extract without modifying logic |
| URL param sync loops between detail and sidebar hooks | Medium | Medium | Each hook only manages its own namespace; explicit echo-loop prevention with refs |
| Mobile sidebar drawer z-index conflicts with detail overlay | Low | Medium | Detail overlay is z-40; sidebar drawer will be z-50 when opened from detail |
| `sidebarWasOpen` stale after multiple rapid navigations | Low | Low | Store it in Zustand (synchronous), not React state; persist to localStorage |
| Backlog-specific features don't fit cleanly into shared header | Low | Medium | Tab bar and action slots are flexible ReactNode — any content fits |

## 12. Non-goals / Prohibited Patterns

- **No legacy compatibility layers.** Old `DetailPageHeader` props interface is replaced, not wrapped.
- **No dead code.** If code isn't used by the new system, delete it.
- **No `_deprecated` or `_old` suffixed files.** Clean cut.
- **No feature flags or gradual rollout.** All pages switch at once.
- **No changes to sidebar content** (filters, sorting, item rendering). Only sidebar open/close behavior.
- **No changes to graph visualization** (nodes, edges, layout, canvas).
- **No changes to entity-specific business logic** (API calls, data transforms, mutations).
- **No `helpers.ts` or `utils.ts` catch-all files.** Every utility goes in a focused module.

## 13. Definition of Done

1. **Single header system:** All 4 entity types (backlog, scenario, execution, initiative) use `DetailPageHeader` + `DetailPageLayout`.
2. **Correct navigation behavior:**
   - Mobile: hamburger opens sidebar; back from sidebar item returns to sidebar
   - Desktop: back arrow closes detail; sidebar state preserved
   - Home button in sidebar → clean graph view
   - Drill-to-lens → close detail + navigate to lens
3. **BacklogDetailsPage decomposed:** Under 300 lines, with extracted tab modules.
4. **No dead code:** No old header components, no unused imports, no compatibility shims.
5. **Full test coverage:** Unit tests for every new component/hook; integration tests for all navigation flows; all existing tests pass.
6. **URL params correct:** Detail and sidebar params coexist; deep-linking works; browser back/forward works.
7. **Production-ready quality:** Well-documented, well-organized, maintainable code matching or exceeding Prompt Manager's quality bar.
