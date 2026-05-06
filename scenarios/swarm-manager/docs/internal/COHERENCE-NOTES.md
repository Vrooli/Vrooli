# React Coherence Audit - 2026-02-05

> Current State (2026-02-14): Active UI pages are Backlog, Scenarios, Execution, and Settings.
>
> Historical note: Earlier sections retain Ideas/Recommendations naming from pre-greenfield iterations. Treat those references as archived context, not current architecture.

## Test Harness Update - 2026-05-01

## Session Details UX Update - 2026-05-06

The agent session details surface now follows the graph-first app model more
closely:

- Graph creation actions show persistent local status/error feedback and route
  to `/sessions/:sessionId` after successful creation.
- Session details uses shared chat primitives instead of page-local transcript
  rendering.
- Desktop uses a collapsible, resizable right-side inspector for proposals,
  artifacts, and metadata.
- Mobile uses full-page tabs for Conversation, Proposals, Artifacts, and
  Details instead of stacking secondary content below the conversation.

The generic chat components live under `components/chat/`; session-only
presentation lives under `components/session/`. Keep future chat-like flows on
the shared chat seam unless they need genuinely different behavior.

**Current Pattern**: UI tests are beginning to converge on `src/test-utils` for provider and environment setup.

**Created**:
- `src/test-utils/query.ts`: React Query test client defaults with retries disabled.
- `src/test-utils/render.tsx`: `renderWithProviders`, `renderHookWithProviders`, and router wrappers with React Router future flags.
- `src/test-utils/browser.ts`: matchMedia, ResizeObserver, and browser storage helpers.
- `src/test-utils/stores.ts`: storage reset helper.
- `src/test-utils/console.ts`: narrow expected-console helper for tests that intentionally assert thrown errors or warnings.

**Migrated First**:
- `hooks/useStats.test.ts`
- `hooks/useCaptureContent.test.ts`
- `hooks/use-url-state.test.tsx`

**Validation**: Focused migrated tests pass, `pnpm exec tsc --noEmit` passes, and full `pnpm test` passes with `1809 passed`.

**Remaining Noise**: Existing hot spots still emit React `act(...)` warnings, React Router future-flag warnings in tests that do not yet use the shared router wrapper, and query warnings from tests whose mocked query functions return `undefined`. Do not add broad React/router/query suppression; migrate hot spots through the shared helpers and silence only expected errors locally.

**2026-05-01 Follow-up**: `components/ui/file-preview.test.tsx` now uses the shared QueryClient test factory and no longer skips the fetch-error state test. Because `useFilePreviewState` spreads production `defaultQueryOptions` into the query, the test file locally mocks `defaultQueryOptions.retry` to `false`; future component tests with query-level production defaults should use the same explicit seam or promote this override into the shared render harness once the high-noise files are migrated.

**2026-05-01 Warning Cleanup**: `surfaces/graph/components/SettingsDrawer.tsx` no longer nests the status-group action button inside the accordion button and now keys toggle buttons created through the helper renderer. `SettingsDrawer.test.tsx` uses the shared render harness and asserts that no React `console.error` warnings are emitted. `contexts/BacklogDetailContext.test.tsx` now uses `withExpectedReactHookError` from `src/test-utils/console.ts` for the intentional provider-invariant failure, keeping expected jsdom/React stacks out of normal test output. `pages/ExecutionPage.test.tsx` now uses the shared render/query harness and has dropped its local router future-flag warning, but the page still needs a polling seam to eliminate its `act(...)` warning. `setupTests.ts` filters the known `[api-base]` diagnostic stdout prefix from `@vrooli/api-base`; this is a narrow test-environment filter for library startup diagnostics, not a general console suppression policy.

## State Management

**Current Pattern**: React Query for server state + Zustand stores for shared state.

**Stores Found**:
- `useBacklogStore`
- `useScenariosStore`
- `useRecommendationsStore`
- `useBacklogFormStore` (form state)

**Observations**:
- Backlog form state consolidated into a focused Zustand store, removing multi-field `useState` scatter in the dialog.
- BacklogDetailsPage still holds 10+ local UI state values; consider a page-level UI store if additional panels or cross-component state is added.
- BacklogDetails file search now uses the shared Input component with a right-side action slot for clearer design-system alignment.

## Duplication

**Result**: No new actionable UI duplication found. Shared UI remains centralized under `components/ui`.

## Styling

**Status**: No changes in this pass. CVA usage stays in shared controls (Button, Input, Card, Select).

## Priority Actions

1. ✅ Moved backlog form state into `src/stores/backlog-form-store.ts`
2. ✅ Promoted `BacklogFormValues` to `src/types/domain.ts` for shared typing
3. ✅ Replaced bespoke file search input with shared `Input` + right-slot affordance for consistent styling
4. ✅ Removed arbitrary `top-[180px]` spacing in ScenarioDetailsPage (now uses `top-44`)

## Follow-ups

1. Consider a BacklogDetails UI store if state density grows further.
2. Review API mutation stores for rate limiting if they expand beyond current usage patterns.

---

# React Coherence Audit - 2026-01-28

This document tracks the coherence audit findings and improvements for the swarm-manager UI.

## State Management

**Current Pattern**: React Query for server state, no Zustand stores yet

**Stores Found**: None - the codebase currently uses React Query for data fetching without shared UI state stores.

**Observations**:
- IdeasPage and ScenariosPage use React Query correctly for server state
- SettingsPage is static UI without state persistence (documented limitation)
- No global toast, modal, or settings state management
- No theme persistence mechanism

**Recommendation**: When implementing settings persistence or global notifications, create stores under `src/stores/` following the pattern:
```
src/stores/
├── settingsStore.ts   # Theme and execution-governance preferences
├── toastStore.ts      # Global notification state
└── index.ts           # Re-exports
```

## Components - Duplication Resolved

### Before (Phase 11)
- IdeasPage and ScenariosPage had identical search input implementations
- 100+ characters of duplicated Tailwind classes for form inputs
- No shared Input component

### After (Phase 11)
- Created `components/ui/input.tsx` - reusable Input component with CVA variants
- Created `components/ui/search-bar.tsx` - extracts common search pattern
- Both pages now import and use `<SearchBar />` instead of inline markup
- SettingsPage uses `<Input />` for custom focus field

## Code Organization

**Current Structure** (Correct):
```
src/
├── components/
│   ├── layout/         # Layout components (MainLayout)
│   └── ui/             # Reusable UI (Button, Input, ErrorState, etc.)
├── config/             # Centralized configuration
├── consts/             # Constants (selectors)
├── lib/                # Utilities (api-client, error-utils, utils)
├── pages/              # Page components
├── services/           # API service layer
└── types/              # Domain types and constants
```

**Notes**:
- The structure follows recommended patterns
- Services layer properly abstracts API calls with DI for testing
- lib/index.ts provides clean re-exports
- No scattered state (components don't define their own hooks for shared state)

## Styling

**Design Token Usage**: Good
- Colors defined in `types/constants.ts` (IDEA_STATUS_COLORS, SCENARIO_STATUS_COLORS)
- Using CVA for component variants (button.tsx, input.tsx)
- `cn()` utility using clsx + tailwind-merge

**CVA Adoption**: Partial → Improving
- Button component uses CVA ✅
- Input component now uses CVA ✅ (added in Phase 11)
- Tabs component uses static classes (from Radix template)

**Inconsistencies Identified**:
- Theme toggle buttons in SettingsPage use inline classes (not extracted)
- Checkbox inputs use inline styling
- These are acceptable for now as they're page-specific, but consider extracting if more settings controls are added

## Priority Actions

### Completed (Phase 11)
1. ✅ Extracted SearchBar component from duplicate patterns
2. ✅ Created Input component with CVA variants
3. ✅ Updated IdeasPage, ScenariosPage, SettingsPage to use shared components

### Future Recommendations
1. **Settings State Store** - Consider a Zustand store if settings complexity grows further
2. **Toast Store** - For global notifications
3. **Theme System** - CSS variables for proper dark/light mode support
4. **Checkbox Component** - Extract styled checkbox with CVA if more form controls needed

## Files Modified (Phase 11)

### Created
- `ui/src/components/ui/input.tsx` - Input component with CVA variants
- `ui/src/components/ui/search-bar.tsx` - Reusable search bar pattern
- `docs/internal/COHERENCE-NOTES.md` - This document

### Modified
- `ui/src/pages/IdeasPage.tsx` - Now uses SearchBar
- `ui/src/pages/ScenariosPage.tsx` - Now uses SearchBar
- `ui/src/pages/SettingsPage.tsx` - Now uses Input component

## Coherence Metrics

| Metric | Before | After |
|--------|--------|-------|
| UI Files | 35 | 37 |
| Duplicate search patterns | 2 | 0 |
| Shared input components | 0 | 2 |
| CVA component coverage | Button only | Button, Input |

---

## Phase 28: Card Component Consolidation

### Problem Identified
The repeated panel styling pattern `rounded-xl border border-white/10 bg-slate-800/30 p-{4|6|8}` appeared 16+ times across 6 page files:
- IdeasPage.tsx (3 occurrences)
- ScenariosPage.tsx (4 occurrences)
- IdeaDetailsPage.tsx (2 occurrences)
- ScenarioDetailsPage.tsx (3 occurrences)
- SettingsPage.tsx (3 occurrences)
- RecommendationsPage.tsx (1 occurrence)

### Solution
Created `components/ui/card.tsx` - a shared Card component with CVA variants:

```tsx
const cardVariants = cva(
  "rounded-xl border border-white/10 bg-slate-800/30",
  {
    variants: {
      padding: { none: "", sm: "p-4", default: "p-6", lg: "p-8" },
      interactive: { true: "transition hover:...", false: "" },
      centered: { true: "text-center", false: "" },
    },
    defaultVariants: { padding: "default", interactive: false, centered: false },
  }
);
```

### Changes Made
1. **Created**: `ui/src/components/ui/card.tsx` - Card component with padding, interactive, and centered variants
2. **Updated Pages** (replaced inline card styling with Card component):
   - IdeasPage.tsx - Loading state, empty state
   - ScenariosPage.tsx - Loading state, empty state, no results state
   - IdeaDetailsPage.tsx - Loading state, header
   - ScenarioDetailsPage.tsx - Loading state, header, metadata section
   - SettingsPage.tsx - Theme, recommendation, and insights sections
   - RecommendationsPage.tsx - Empty state

### Benefits
- Single source of truth for panel styling
- Consistent padding (lg=p-8 for states, default=p-6 for content)
- Easy to update styling globally
- Variants handle common patterns (centered for states, interactive for clickable items)

### Remaining Inline Card Patterns (Intentionally Left)
- Interactive Link cards in list views (require `<Link>` wrapper, Card inside Link would cause hydration issues)
- Danger zone section in ScenarioDetailsPage (uses red border intentionally)

### Files Modified
- Created: `ui/src/components/ui/card.tsx`
- Modified: IdeasPage.tsx, ScenariosPage.tsx, IdeaDetailsPage.tsx, ScenarioDetailsPage.tsx, SettingsPage.tsx, RecommendationsPage.tsx

### Updated Metrics

| Metric | Before (Phase 11) | After (Phase 28) |
|--------|-------------------|------------------|
| UI Files | 37 | 38 |
| Duplicate panel patterns | 16+ | ~4 (intentional) |
| CVA component coverage | Button, Input | Button, Input, Card |
| All tests pass | ✅ | ✅ (311 tests) |

---

## Phase 28 Iteration 2: Coherence Verification Audit

### Coherence Iteration Summary
Comprehensive audit confirming the codebase follows React coherence patterns established in previous iterations.

### State Management Assessment
**Pattern**: React Query for all server state (no Zustand stores)

**Findings**:
- ✅ IdeasPage, ScenariosPage: useQuery with defaultQueryOptions
- ✅ IdeaDetailsPage, ScenarioDetailsPage: useQuery + useMutation for CRUD
- ✅ No useState for shared state - only local UI state (selectedFile, showUpload, etc.)
- ✅ No Context for frequently-changing state

**Recommendation**: Current pattern is appropriate. Add Zustand stores only when implementing:
- Global toast notifications
- Theme persistence
- Settings sync across components

### Component Duplication Check
**Result**: No actionable duplicates found

| Component | Status | Notes |
|-----------|--------|-------|
| Card | ✅ Consolidated | CVA variants for padding, interactive, centered |
| Button | ✅ Consolidated | CVA variants for variant, size |
| Input | ✅ Consolidated | CVA with dark mode styling |
| SearchBar | ✅ Consolidated | Shared search pattern |
| ErrorState | ✅ Consolidated | Auto-detects error category |
| TagList | ✅ Consolidated | Handles truncation logic |

### Service Layer Assessment
**Pattern**: Factory functions with interface-based DI

```typescript
// Pattern used in ideas-service.ts and scenarios-service.ts
export function createIdeasService(apiClient: IApiClient = defaultApiClient): IIdeasService
```

**Findings**:
- ✅ Clean separation of concerns
- ✅ Testable via mock injection
- ✅ No business logic in services (data access only)

### Design Token Assessment
**Location**: `types/constants.ts`

**Findings**:
- ✅ Status colors centralized (IDEA_STATUS_COLORS, SCENARIO_STATUS_COLORS)
- ✅ Status icons centralized (SCENARIO_STATUS_ICONS)
- ✅ Format functions use shared utilities (formatDisplayText)

### Remaining Patterns (Intentionally Unchanged)
1. **SettingsPage option buttons (6 instances)**: Settings are now stateful; still acceptable inline while control count stays small. Revisit if new groups are added.
2. **Link wrapper cards in list views**: Must use inline styling due to React hydration constraints with Card inside Link.
3. **Danger zone border in ScenarioDetailsPage**: Intentionally red border for visual differentiation.

### Tracked Files
Recorded 17 visits via visited-tracker react-coherence campaign:
- 6 pages (ScenariosPage, IdeasPage, ScenarioDetailsPage, IdeaDetailsPage, SettingsPage, RecommendationsPage)
- 5 UI components (Button, Card, Input, SearchBar, ErrorState)
- 3 services (ideas-service, scenarios-service, index)
- 3 type/config files (constants, domain, config/index)

### Conclusion
The codebase demonstrates strong architectural coherence:
- Single source of truth for UI patterns (Card, Button, Input)
- Consistent state management (React Query for server state)
- Clean service seam architecture with DI
- Centralized design tokens

No high-value consolidation opportunities remain. The codebase is well-organized for future maintenance and extension.

---

## Phase 28 Iteration 4: Final Coherence Verification

### Coherence Iteration Summary
Fourth and final coherence verification iteration confirming no regressions and continued architectural alignment.

### State Management Re-Verification
**Pattern Remains Consistent**: React Query for server state, local useState only for ephemeral UI

| Page | useState Count | Purpose | Assessment |
|------|---------------|---------|------------|
| ScenariosPage | 3 | search, statusFilter, showFilters | ✅ Correct - local filter state |
| ScenarioDetailsPage | 4 | localGreenfield, localRecommendations, showDeleteDialog, archiveOnDelete | ✅ Correct - optimistic UI + dialog state |
| IdeaDetailsPage | 2 | selectedFile, showUpload | ✅ Correct - file selection + upload toggle |
| IdeasPage | 0 | N/A | ✅ Correct - pure data display |

**Note**: ScenarioDetailsPage uses `useEffect` to sync server state to local state for optimistic updates - this is the correct pattern for forms that need immediate visual feedback while mutations are in-flight.

### CVA Component Coverage
| Component | Status | Variants |
|-----------|--------|----------|
| Button | ✅ | variant (default/outline/destructive), size (default/sm) |
| Card | ✅ | padding (none/sm/default/lg), interactive, centered |
| Input | ✅ | variant (default/error), size (default/sm/lg) |

### Remaining Inline Styling Patterns (Intentionally Preserved)
1. **Link-wrapped cards in list views** - JSX structure requires different element composition
2. **Danger zone section** - Uses intentional red border for visual differentiation
3. **SettingsPage option buttons** - Placeholder page, documented as future work

### Files Audited This Iteration
Recorded 8 visits via visited-tracker react-coherence campaign:
- 4 pages: ScenariosPage, ScenarioDetailsPage, IdeaDetailsPage, IdeasPage
- 3 components: ConfirmDialog, FileTree, FilePreview
- 1 service: services/index.ts

### Completeness Score
- Before: 54/100
- After: 54/100 (unchanged - coherence-focused phase)

### Validation Results
- ✅ All 20 unit tests pass
- ✅ UI smoke test passes (1259ms)
- ✅ ESLint: 0 errors
- Auditor: 1 HIGH (PATH-001 false positive - path traversal protection) + 4 LOW (PRD template - intentional)

### Conclusion
**Phase 28 React Coherence COMPLETE** (4 iterations). The codebase demonstrates production-ready architectural coherence:
- No duplicate component patterns
- Consistent state management
- Proper use of CVA variants
- Centralized design tokens
- Clean service seam architecture

No code changes were needed in this iteration as the codebase already follows all coherence patterns.

---

*Last updated: 2026-01-28 (Phase 28 iter 4: Final Coherence Verification)*

---

## Phase 28 Iteration 5: Final Verification Sweep

### Coherence Iteration Summary
Fifth and final iteration (max iterations reached) confirming the codebase maintains all established coherence patterns.

### Complete Component Inventory
All shared UI components now use CVA for variants:

| Component | File | CVA Variants |
|-----------|------|--------------|
| Button | `components/ui/button.tsx` | variant (default/outline/destructive), size (default/sm) |
| Card | `components/ui/card.tsx` | padding (none/sm/default/lg), interactive, centered |
| Input | `components/ui/input.tsx` | variant (default/error), size (default/sm/lg) |
| SearchBar | `components/ui/search-bar.tsx` | Composes Input with search icon |
| Tabs | `components/ui/tabs.tsx` | Radix-based, inline cn() composition |
| ErrorState | `components/ui/error-state.tsx` | Auto-categorizes errors, provides retry |
| TagList | `components/ui/tag-list.tsx` | Handles tag truncation |
| ConfirmDialog | `components/ui/confirm-dialog.tsx` | Destructive action confirmation |
| FileTree | `components/ui/file-tree.tsx` | Hierarchical file display |
| FilePreview | `components/ui/file-preview.tsx` | File content display |
| FileUpload | `components/ui/file-upload.tsx` | Drag-and-drop upload |
| ErrorBoundary | `components/ui/error-boundary.tsx` | Catches render errors |
| PageErrorBoundary | `components/ui/page-error-boundary.tsx` | Page-level error isolation |

### State Management Summary
**Pattern**: React Query for server state, useState for local UI state only

| Category | Pattern | Examples |
|----------|---------|----------|
| Server State | React Query useQuery/useMutation | ideas list, scenarios list, file content |
| Optimistic Updates | useState + useMutation onError | greenfield toggle, recommendations toggle |
| Local UI State | useState | selectedFile, showUpload, search filters |
| Global State | None needed yet | Future: toast store, settings store |

### Design Token Centralization
All domain-specific display constants are in `types/constants.ts`:
- `IDEA_STATUS_COLORS` - Maps IdeaStatus to Tailwind bg classes
- `SCENARIO_STATUS_COLORS` - Maps ScenarioStatus to Tailwind text classes
- `SCENARIO_STATUS_ICONS` - Maps ScenarioStatus to Lucide icons
- `formatIdeaStatus()` - Consistent status formatting

### Service Layer Architecture
Clean seam design with dependency injection for testability:
```
services/
├── ideas-service.ts     # createIdeasService(apiClient?) → IIdeasService
├── scenarios-service.ts # createScenariosService(apiClient?) → IScenariosService
└── index.ts             # Re-exports all services
```

### Code Organization (Final Structure)
```
src/
├── components/
│   ├── layout/         # MainLayout with responsive nav
│   └── ui/             # 13 shared UI components
├── config/             # Centralized tunable levers
├── consts/             # Test selectors
├── lib/                # API client, error utils, format utils, query utils
├── pages/              # 6 page components
├── services/           # API service layer (2 services)
└── types/              # Domain types and constants
```

### Final Assessment
**No coherence issues found.** The codebase demonstrates:

1. **Single Source of Truth**: Each UI pattern has one canonical implementation
2. **Consistent State Management**: React Query for server, useState for ephemeral UI
3. **CVA Adoption**: All major components use class-variance-authority
4. **Clean Seams**: Services are testable via DI, components are composable
5. **Centralized Config**: All tunable levers in `config/index.ts`

### Phase 28 React Coherence Complete
5 iterations completed (maximum reached). The codebase is production-ready from an architectural coherence standpoint. Future work should:
- Add Zustand stores when implementing settings persistence
- Add toast store for global notifications
- Consider extracting SettingsPage option buttons when that feature is implemented

---

## Phase 30: Theme Tokenization + Light Mode Coherence (2026-02-04)

### Problem Identified
Light mode toggled at the document level, but components still used hardcoded dark palette utilities. This caused partial theme switching (some areas updated, many remained dark), plus mismatched editor/markdown themes.

### Solution
Introduced a theme-aware slate palette backed by CSS variables and removed manual light/dark branching in layout styling. Added a resolved-theme hook to align Monaco editors and markdown rendering with the active theme.

### Changes Made
1. **Theme Tokens**: Mapped Tailwind slate colors to CSS variables in `ui/tailwind.config.ts`
2. **Theme Palette**: Defined dark + light palette inversion in `ui/src/styles.css`
3. **Light Overrides**: Added scoped overrides for `border-white/*` utilities in light mode
4. **Theme Hook**: `useResolvedTheme` in `ui/src/lib/theme-utils.ts` with a custom theme-change event
5. **Layout Update**: Simplified `MainLayout` to use tokenized colors (no manual theme branches)
6. **Editor + Markdown**: File preview now switches Monaco theme and prose styling based on resolved theme

### Files Modified
- `ui/tailwind.config.ts`
- `ui/src/styles.css`
- `ui/src/lib/theme-utils.ts`
- `ui/src/lib/index.ts`
- `ui/src/components/layout/MainLayout.tsx`
- `ui/src/components/ui/button.tsx`
- `ui/src/components/ui/file-preview.tsx`

### Result
Light mode now updates all slate-based UI components consistently, and editor/markdown views respond to theme changes without manual page-level overrides.

*Last updated: 2026-02-04 (Phase 30: Theme Tokenization + Light Mode Coherence)*

## Output Tab Consolidation (2026-04-02)

### Problem
Execution/activity data was scattered across three disconnected UI surfaces:
- `BacklogActiveRunBanner` — vanished when agent finished (no completed execution indicator)
- `BacklogScenariosPanel` on Info tab — mixed post-run review results with static metadata
- `ActivityTimeline` in a hidden Drawer — not discoverable

### Changes
1. **New Output tab** (4th tab) consolidates all execution output:
   - `LatestExecutionSummary` — persistent card, always visible (empty/active/completed states)
   - `ScenarioReviewResults` — target scenario chips + post-run review status
   - `ActivityTimeline` — inline, promoted from Drawer

2. **Info tab simplified**:
   - `BacklogScenariosPanel` now only shows scenario name chips (navigation), no review results
   - Active run banner removed from Info tab

3. **Deleted components**:
   - `backlog-active-run-banner.tsx` — replaced by `LatestExecutionSummary`
   - ActivityTimeline Drawer in `backlog-dialogs.tsx` — replaced by inline tab content
   - History button in `backlog-desktop-header.tsx` — no longer needed

4. **Store cleanup**:
   - Removed `isTimelineOpen`, `openTimeline`, `closeTimeline` from `backlog-detail-ui-store`
   - Timeline polling now gated on `activeTab === "output"` (URL state)

5. **Tab badge**: Output tab shows pulsing cyan dot when agent is active (visible on all tabs)

### Files Modified
- `ui/src/pages/BacklogDetailsPage.tsx`
- `ui/src/components/backlog/output-tab.tsx` (new)
- `ui/src/components/backlog/latest-execution-summary.tsx` (new)
- `ui/src/components/backlog/scenario-review-results.tsx` (new)
- `ui/src/components/backlog/backlog-scenarios-panel.tsx` (simplified)
- `ui/src/components/backlog/backlog-desktop-header.tsx` (simplified)
- `ui/src/components/backlog/backlog-dialogs.tsx` (simplified)
- `ui/src/stores/backlog-detail-ui-store.ts` (cleaned)
- `ui/src/consts/selectors.ts` (updated)

## UI Test Harness Coherence (2026-05-01)

Hot-spot page and dialog tests should use `src/test-utils` for QueryClient defaults, router future flags, and browser API mocks. `ExecutionPage`, `ScenarioDetailsPage`, `ScenariosPage`, `FeedbackDialog`, and `FeedbackPanel` now follow this pattern and their focused test runs are warning-free.

Important teardown rule: if a component subscribes to a Zustand store, unmount it before resetting that store in test cleanup, or wrap the reset in `act`. Otherwise teardown can create a real unwrapped update warning after assertions have already passed.

Timer rule: when fake timers drive React state, advance them inside `act`. This now covers `useCapturePolling`, `ClarificationPanel` staleness checks, and `FollowUpSheet` mount-effect flushing.

Router rule: tests that need app routing should use `renderWithProviders` or `createRouterWrapper` from `src/test-utils`. Those helpers now support `initialIndex` and set the React Router future flags, so page/detail tests do not need local `MemoryRouter` wrappers. `InitiativeDetailsPage`, `BacklogDetailsPage`, `NotFoundPage`, `DetailPageHeader`, `ExecutionOverviewTab`, `FocusActionsSection`, `DependencyChipList`, `ScenarioResultCards`, `InitiativeDependencyGraph`, and `ScenarioBadge` now follow this pattern.

Async hook rule: mount-time async browser storage loads should be flushed inside `act` before making initial-state assertions. `useIndexedDBAttachments.test.ts` uses a local render helper for this so the expected empty initial attachment state does not race the IndexedDB load microtask.
