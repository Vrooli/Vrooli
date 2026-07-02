# Utils Unification Notes

This document tracks the utility consolidation audit and improvements for the swarm-manager scenario.

## Last Updated
2026-05-01 (Test utility architecture slice)

## Summary

The codebase utility structure is well-organized but had one significant duplication issue resolved in this phase. The overall architecture follows screaming architecture principles with utilities organized by concern.

2026-05-06 update: Agent-session, clarification, and evidence-request chat
surfaces now share `ui/src/components/chat/` primitives for markdown rendering,
message alignment, waiting indicators, auto-scroll, and composer behavior.
Session-specific artifact node mapping was extracted to
`ui/src/components/session/session-artifact-routing.ts` with pure unit coverage,
so session page navigation no longer owns ad hoc artifact parsing.

2026-05-01 update: Swarm Manager now has a dedicated UI test utility layer under `ui/src/test-utils/` for test-only query clients, provider/router rendering, browser API mocks, storage reset helpers, and expected-console handling. Initial hook tests have been migrated to this layer; future UI test work should extend these helpers instead of recreating local QueryClient, MemoryRouter, matchMedia, ResizeObserver, localStorage, or console-silencing setup.

2026-05-01 follow-up: `components/ui/file-preview.test.tsx` now reuses `createTestQueryClient` and the previously skipped fetch-error assertion is active. This exposed one important boundary: production query options spread directly inside components can override QueryClient test defaults, so tests that need immediate error rendering must explicitly disable the component-level retry seam until the shared harness owns that override centrally.

2026-05-01 warning-cleanup follow-up: `src/test-utils/console.ts` now includes `withExpectedReactHookError` for provider-invariant hook tests. Use it only around tests that intentionally render a throwing hook; ordinary React warnings should be fixed at the component or harness seam. `src/setupTests.ts` filters the known `[api-base]` startup diagnostic prefix so individual tests do not need local `@vrooli/api-base` mocks just to keep stdout readable.

## Current Utility Architecture

```
src/
├── lib/                          # Core utilities layer
│   ├── api-client.ts             # HTTP infrastructure (ApiError, ApiClient)
│   ├── api-endpoints.ts          # API route constants
│   ├── error-utils.ts            # Error categorization, logging, ID generation
│   ├── format-utils.ts           # Formatting: formatFileSize, capitalize, formatDisplayText, getFileExtension
│   ├── query-utils.ts            # React Query default options
│   ├── utils.ts                  # General utilities (cn for classnames)
│   └── index.ts                  # Clean re-exports
├── config/                       # Configuration layer
│   └── index.ts                  # All tunable configuration
├── types/                        # Domain types layer
│   ├── domain.ts                 # Business types (Idea, Scenario, etc.)
│   ├── constants.ts              # Display constants (colors, icons)
│   └── index.ts                  # Re-exports
├── services/                     # Data access layer
│   ├── ideas-service.ts          # Ideas API operations
│   ├── scenarios-service.ts      # Scenarios API operations
│   └── index.ts                  # Re-exports
├── consts/                       # UI constants
│   └── selectors.ts              # Test selector registry
├── test-utils/                   # Test-only provider, browser, storage, and console helpers
│   ├── query.ts                  # React Query test client defaults
│   ├── render.tsx                # render/renderHook wrappers with providers/router
│   ├── browser.ts                # matchMedia/ResizeObserver and storage helpers
│   ├── stores.ts                 # storage reset helper
│   └── console.ts                # narrow expected-console and expected hook-error helpers
└── components/ui/                # Shared UI components
    ├── button.tsx                # Button with CVA variants
    ├── input.tsx                 # Input with CVA variants
    ├── search-bar.tsx            # Reusable search pattern
    ├── error-state.tsx           # User-friendly error display
    ├── error-boundary.tsx        # Runtime error catching
    └── page-error-boundary.tsx   # Page-level error isolation
```

## Duplications Resolved

### Error ID Generation (Phase 12)

**Before**: Three separate implementations of essentially the same function:
1. `lib/error-utils.ts` - `generateCorrelationId()` → `err_${timestamp}_${random}`
2. `components/ui/error-boundary.tsx` - `generateErrorId()` → `err_${timestamp}_${random}`
3. `components/ui/page-error-boundary.tsx` - `generateErrorId()` → `page_err_${timestamp}_${random}`

**After**: Single `generateUniqueId(prefix)` function in `lib/error-utils.ts`
- Accepts prefix parameter for flexible ID generation
- `generateCorrelationId()` kept as convenience wrapper
- Both error boundary components import and use the shared utility

**Benefits**:
- Single source of truth for ID generation algorithm
- Easier to test (one location, comprehensive test coverage)
- Consistent format across all error IDs
- Extensible for future use cases (trace IDs, request IDs, etc.)

### React Query Options (Phase 12, Iteration 2)

**Before**: Duplicate React Query configuration in every data-fetching page:
```tsx
// IdeasPage.tsx
const { data } = useQuery({
  queryKey: ["ideas"],
  queryFn: () => ideasService.list(),
  retry: dataFetchingConfig.retryCount,
  retryDelay: (attemptIndex) =>
    dataFetchingConfig.retryDelayMs * Math.pow(2, attemptIndex),
  staleTime: dataFetchingConfig.staleTimeMs,
  gcTime: dataFetchingConfig.cacheTimeMs,
  refetchOnWindowFocus: dataFetchingConfig.refetchOnWindowFocus,
});

// ScenariosPage.tsx (identical configuration)
const { data } = useQuery({
  queryKey: ["scenarios"],
  queryFn: () => scenariosService.list(),
  retry: dataFetchingConfig.retryCount,
  retryDelay: (attemptIndex) =>
    dataFetchingConfig.retryDelayMs * Math.pow(2, attemptIndex),
  staleTime: dataFetchingConfig.staleTimeMs,
  gcTime: dataFetchingConfig.cacheTimeMs,
  refetchOnWindowFocus: dataFetchingConfig.refetchOnWindowFocus,
});
```

**After**: Single `defaultQueryOptions` object in `lib/query-utils.ts`
```tsx
// IdeasPage.tsx
const { data } = useQuery({
  queryKey: ["ideas"],
  queryFn: () => ideasService.list(),
  ...defaultQueryOptions,
});

// ScenariosPage.tsx
const { data } = useQuery({
  queryKey: ["scenarios"],
  queryFn: () => scenariosService.list(),
  ...defaultQueryOptions,
});
```

**Benefits**:
- Eliminates 5 lines of duplicate configuration per page
- Centralizes retry/cache behavior for consistency
- Easier to modify data fetching behavior globally
- Reduces coupling between pages and `dataFetchingConfig`

### File Size Formatting (Phase 22)

**Before**: Identical `formatFileSize` implementations in two components:
1. `components/ui/file-tree.tsx` (lines 32-38)
2. `components/ui/file-upload.tsx` (lines 297-303)

**After**: Single `formatFileSize` function in `lib/format-utils.ts`
- Both components import from `../../lib`
- Pure function, no dependencies
- Comprehensive test coverage in `format-utils.test.ts`

### Status Capitalization (Phase 22)

**Before**: Same capitalization pattern duplicated:
1. `pages/ScenarioDetailsPage.tsx` - `formatStatus()` function
2. `pages/ScenariosPage.tsx` - inline `status.charAt(0).toUpperCase() + status.slice(1)`

**After**: Single `capitalize` function in `lib/format-utils.ts`
- Both pages import from `../lib`
- Also added `formatDisplayText` for snake_case/kebab-case handling
- Pure functions, comprehensive tests

### File Extension Extraction (Phase 22, Iteration 2)

**Before**: Duplicate file extension extraction in `file-preview.tsx`:
1. Line 38: `const ext = fileName.split(".").pop()?.toLowerCase() || "";` (in `getFileType`)
2. Line 108: `const ext = fileName.split(".").pop()?.toLowerCase() || "";` (in `CodeView`)

**After**: Single `getFileExtension(fileName)` function in `lib/format-utils.ts`
- `file-preview.tsx` imports and uses the shared utility in both places
- Pure function, handles edge cases (dotfiles, no extension, multiple dots)
- Comprehensive test coverage added (9 tests)

### Domain Status Formatting (Phase 22, Iteration 2)

**Before**: `formatIdeaStatus` in `types/constants.ts` had its own implementation:
```ts
export function formatIdeaStatus(status: IdeaStatus): string {
  return status.replace(/_/g, " ");
}
```

This duplicated the underscore-to-space logic already present in `formatDisplayText`.

**After**: `formatIdeaStatus` delegates to `formatDisplayText`:
```ts
export function formatIdeaStatus(status: IdeaStatus): string {
  return formatDisplayText(status);
}
```

**Benefits**:
- Single source of truth for text display formatting
- Consistent behavior (now capitalizes first letter)
- Domain function preserved for type safety (accepts `IdeaStatus` only)
- Eliminates logic duplication between domain and core utilities

## Audit Findings

### Search Patterns Used
```bash
rg "function (format|parse|normalize|serialize|deserialize|validate)" --type ts
rg "helpers|utils|misc" --type ts -l
rg "Date\.now\(\)" --type ts
rg "generateErrorId|generateCorrelationId|err_" --type ts
```

### No Action Required
- **Date formatting**: Only one instance in IdeasPage (`new Date(idea.updated).toLocaleDateString()`) - not worth extracting for a single use
- **Template formatting**: `formatTemplate` and `normalizeParams` in `consts/selectors.ts` are internal to the selector registry
- **API error handling**: Already well-consolidated in `lib/api-client.ts` and `lib/error-utils.ts`

### Red Flags Checked
- [x] Same function name implemented in multiple files → **Resolved** (generateErrorId)
- [x] Slightly different formatting logic per feature → **Not found**
- [x] "utils" or "helpers" files with mixed responsibilities → **Clean** (`lib/utils.ts` only has `cn()`)
- [x] Utility functions that import React or app-level modules → **Clean** (utilities are pure)

## Utility Tier Classification

| Tier | Location | Purpose | Examples |
|------|----------|---------|----------|
| Core | `lib/` | Pure, framework-agnostic utilities | `cn()`, `generateUniqueId()`, `formatFileSize()`, `capitalize()`, `getFileExtension()` |
| Framework | `lib/` | React/framework-specific utilities | `defaultQueryOptions` |
| Domain | `types/` | Business logic utilities | `formatIdeaStatus()`, status colors |
| UI | `components/ui/` | React-specific patterns | `ErrorBoundary`, `ErrorState` |
| Data | `services/` | API interaction patterns | `ideasService`, `scenariosService` |

## Testing Seams

The codebase properly implements testing seams for utilities:

1. **API Client**: `IApiClient` interface allows mock injection
2. **Services**: Factory functions (`createIdeasService(apiClient)`) accept mock clients
3. **Error Utils**: Pure functions with no hidden dependencies
4. **ID Generation**: Now testable via `generateUniqueId(prefix)` with deterministic format
5. **Query Options**: `defaultQueryOptions` references config, making it testable and overridable
6. **UI Test Utilities**: `src/test-utils` centralizes React Query, router, browser, storage, and expected-console seams for tests only
7. **Hot-spot Render Harness**: `ExecutionPage`, `ScenarioDetailsPage`, `ScenariosPage`, `FeedbackDialog`, and `FeedbackPanel` tests now use the shared render/query/browser helpers instead of local QueryClient, router, and matchMedia setup
8. **Act-Safe Timer Harnessing**: Polling and staleness tests wrap timer advancement and subscribed store resets in `act` (`useCapturePolling`, `ClarificationPanel`, `FollowUpSheet`) so warning-free focused runs reflect real React update boundaries

## Future Consolidation Candidates

### Low Priority (Single Use)
- Date formatting helper (if more date display needs arise)
- Progress bar rendering (if reused beyond ScenariosPage)

### Resolved (Iteration 2)
- ~~Tag truncation logic~~ → Uses config values (`displayLimitsConfig`), not duplicate logic

### Monitor For Duplication
- Form validation patterns (when forms are implemented)
- Toast/notification handling (when toast system is added)
- Theme utilities (when dark/light mode is implemented)

## Files Modified (Phase 12)

### Iteration 1 - Error ID Consolidation

**Created:**
- `docs/internal/UTILS_UNIFICATION_NOTES.md` - This document

**Modified:**
- `lib/error-utils.ts` - Added `generateUniqueId(prefix)` function
- `lib/index.ts` - Added `generateUniqueId` to exports
- `lib/error-utils.test.ts` - Added tests for `generateUniqueId`
- `components/ui/error-boundary.tsx` - Uses shared `generateUniqueId`
- `components/ui/page-error-boundary.tsx` - Uses shared `generateUniqueId`

### Iteration 2 - Query Options Consolidation

**Created:**
- `lib/query-utils.ts` - `defaultQueryOptions` for React Query
- `lib/query-utils.test.ts` - Tests for query utilities

**Modified:**
- `lib/index.ts` - Added `defaultQueryOptions` export
- `pages/IdeasPage.tsx` - Uses `...defaultQueryOptions` instead of inline config
- `pages/ScenariosPage.tsx` - Uses `...defaultQueryOptions` instead of inline config

## Files Modified (Phase 22)

### Iteration 1 - Formatting Utilities Consolidation

**Created:**
- `lib/format-utils.ts` - `formatFileSize`, `capitalize`, `formatDisplayText`
- `lib/format-utils.test.ts` - Comprehensive tests for formatting utilities

**Modified:**
- `lib/index.ts` - Added formatting utilities to exports
- `components/ui/file-tree.tsx` - Uses shared `formatFileSize` from lib
- `components/ui/file-upload.tsx` - Uses shared `formatFileSize` from lib
- `pages/ScenarioDetailsPage.tsx` - Uses shared `capitalize` from lib
- `pages/ScenariosPage.tsx` - Uses shared `capitalize` from lib

### Iteration 2 - File Extension and Status Formatting

**Modified:**
- `lib/format-utils.ts` - Added `getFileExtension(fileName)` function
- `lib/format-utils.test.ts` - Added 9 tests for `getFileExtension`
- `lib/index.ts` - Added `getFileExtension` to exports
- `components/ui/file-preview.tsx` - Uses shared `getFileExtension` in both `getFileType` and `CodeView`
- `types/constants.ts` - `formatIdeaStatus` now delegates to `formatDisplayText`
- `types/constants.test.ts` - Updated tests to reflect capitalized output
- `pages/IdeasPage.test.tsx` - Updated test expectation for capitalized status

## Metrics

| Metric | Phase 12 Start | After Phase 12 | After Phase 22 Iter 1 | After Phase 22 Iter 2 |
|--------|----------------|----------------|------------------------|------------------------|
| Duplicate ID generators | 3 | 1 | 1 | 1 |
| Duplicate query configs | 2 | 0 | 0 | 0 |
| Duplicate formatFileSize | - | 2 | 0 | 0 |
| Duplicate capitalize | - | 2 | 0 | 0 |
| Duplicate getFileExtension | - | - | 2 | 0 |
| Duplicate status formatting | - | - | 1 | 0 |
| Lines eliminated | 0 | 22 | 42 | 46 |
| New utility functions | 0 | 2 | 5 | 6 |
| Tests added | 0 | 13 | 31 | 40 |

---

*Last updated: 2026-01-28 (Phase 22: Utils Unification, Iteration 2)*

---

## Test Utility Consolidation (2026-05-01)

The UI test utility layer is now the canonical home for app-level test providers and browser shims:

- `src/test-utils/render.tsx` owns QueryClient + router wrappers, including React Router future flags and `initialIndex` support.
- `src/test-utils/query.ts` owns retry-free React Query defaults.
- `src/test-utils/browser.ts` owns `matchMedia`, `ResizeObserver`, and storage helpers.
- `src/test-utils/console.ts` owns narrow expected-console suppression helpers.

Recent migrations removed local router/query/browser setup from `InitiativeDetailsPage`, `BacklogDetailsPage`, `NotFoundPage`, `DetailPageHeader`, `ExecutionOverviewTab`, `FocusActionsSection`, `DependencyChipList`, `ScenarioResultCards`, `InitiativeDependencyGraph`, and `ScenarioBadge`. Keep future page or routed component tests on these helpers unless the test is intentionally validating a lower-level routing primitive.

Remaining consolidation candidates:
- Local `QueryClientProvider` wrappers in smaller component/hook tests.
- Remaining raw `MemoryRouter` wrappers in low-level navigation component tests.
- Local IndexedDB/FileReader mocks in `useIndexedDBAttachments.test.ts`, if another browser-storage test needs the same seam.

## API Test Timing Utilities (2026-05-01)

`api/internal/testutil.Eventually` (defined in `helpers.go`) is the canonical helper for tests that observe asynchronous fire-and-forget work. (An earlier `assertx` subpackage was consolidated back into the flat `testutil` package — see the Test Utility Boundary seam in `SEAMS.md` — so import `testutil.Eventually` directly.)

Use it for positive eventual conditions such as index notifications, graph materialization drains, background reindex completion, and dispatch hook rebuilds. Do not use it to hide absence checks; tests that intentionally validate "nothing happened after a short window" should keep a narrow fixed sleep with a comment naming that real-time contract.

Recent migrations removed local polling loops from AI search integration tests, AI search reindex tests, graph materializer scheduling tests, the root graph materialization integration helper, and the root initiative feedback/review integration helpers. Remaining direct sleeps are limited to the shared `Eventually` polling interval, explicitly commented negative absence checks, and fake upstream latency used to pin singleton semantics.

## Plan-Lens Consolidation Audit (2026-07-02)

Utils audit of the surfaces touched by the Plan-lens consolidation (Operations
Center + Command Post + Topology/Operations lens retirement):

**Consolidated / rehomed**
- `components/cards/BoardCard.tsx` is the single card primitive for board
  surfaces (status dot, title, meta, badge/action slots); all Plan board card
  variants (`PlanCardView`, gate cards, outcome cards) render on it. The
  live-activity row keeps `components/operations/ActivityRow.tsx` — it is a
  self-contained store-connected component reused verbatim by the Now column,
  not a second primitive.
- Board grouping/sorting/labeling lives in pure `surfaces/plan/lib/` modules
  (`plan-presentation.ts`, `plan-url-state.ts`), unit-tested without React.
  Server-side grouping/sorting lives in `internal/planview` (mirroring the
  Command Post ordering via the pre-existing `backlogrank` package — no new
  ranking implementation was written).
- The Operations Center URL-filter vocabulary was extracted into
  `plan-url-state.ts` instead of duplicating the page-local helpers it
  replaced; old `/operations?...` links keep working through the redirect.
- `lib/command-post-utils.ts` (groupActionItems etc.) survives with its three
  remaining consumers (SidebarTabs, badge hook, board menus via
  `useCommandPostItemActions`); the server `internal/gates` read-model is the
  authoritative reimplementation of the same predicates, and the two are kept
  aligned by the parity tests in `gates_test.go`.

**Deleted rather than unified** (single-consumer code that died with its page):
`OpsHeader`, `OpsFilterBar`, `OpsBody`, `ByInitiativeView`, `ByPhaseView`,
`SummaryView`, `ActionGroupCard`, `RecentSection`, `SnoozedSection`,
`EmptyState`, `ExecutionCaptureCard`, `CommandPostButton`, `Breadcrumb`,
`ClusterNode`, `clustering-utils`, the edge-count perf guards, MiniMap wiring,
`getStatusRgb`, the graph-service focus plumbing, and the server
flow/operations projection with its node builders.

**Known intentional duplication**: date/status formatting on board cards uses
the shared `formatRelativeTime` / status label helpers already in `lib/`; no
new formatters were introduced.
