# App UI Refactor Plan

## Context
- `ui/src/App.tsx` (~900 LOC) owns websocket lifecycle, REST interactions, filter/search logic, URL syncing, dialog/snackbar/theme effects, and manual DOM tweaks.
- Downstream components rely on App's internal state setters and globals, which makes the codebase brittle, hard to test, and difficult to extend safely.

## Goals
- Preserve current behaviour while decomposing concerns into cohesive modules/hooks and a slimmed-down app shell.
- Establish reusable primitives (filters, URL sync, snackbar, modal management) to support future features.
- Improve testability and maintainability without altering the visual design or backend contracts.

## Non-Goals
- Introducing brand-new features or redesigning UI components.
- Modifying API interfaces or websocket protocols.

## Proposed Architecture
1. **Top-Level Shell**
   - Create `IssueAppProviders` wrapper(s) so `App.tsx` focuses on composition and rendering.
2. **Services Layer**
   - New `ui/src/services/issues.ts` encapsulates REST calls and issue transformations (list, create, update status, delete).
3. **Issue Data Context**
   - Replace plain `useIssueTrackerData` with `IssueDataProvider` + `useIssueData` hook exposing intent-specific actions (`refreshAll`, `setProcessorSettings`, `updateIssue`, etc.).
   - Websocket integration lives inside the provider so state updates propagate consistently.
4. **Filters & URL Sync**
   - `useIssueFilters` manages priority/app/search state and derives filtered issues and available app list.
   - `useSearchParamSync` handles reading/writing/popping query params with SSR guards.
5. **UI State Utilities**
   - `useDialogState` (with `useLockBodyScroll`) for modal toggles.
   - `useSnackbarQueue` for enqueue/dismiss with auto-hide timers.
   - `useThemeClass` to manage body/html classes based on display settings.
6. **Shared Constants**
   - Move kanban column metadata to `constants/board.ts` for reuse across components.

## Data & Control Flow Overview
1. `App.tsx` renders providers (IssueData, Snackbar) and consumes hooks to retrieve derived state.
2. `useIssueData` provides base issue list, stats, automation settings, rate-limit info, and mutation helpers.
3. `useIssueFilters` consumes issues from context, maintains filter/query state, and returns filtered result set + handlers.
4. Components (board, dialogs) receive data via props or direct hooks; they no longer call raw setters or perform API work themselves.
5. Snackbar and dialogs rely on dedicated hooks, ensuring consistent UX and easier testing.

## Migration Phases
1. **Phase 1 – Services Layer**
   - Lift REST helpers out of `App.tsx` into `services/issues.ts`.
   - Update `useIssueTrackerData` to use the new services while retaining its public API.
2. **Phase 2 – Issue Data Provider**
   - Introduce provider/context, migrate `useIssueTrackerData` internals into it, update `App.tsx` to use the new hook.
3. **Phase 3 – Filters & URL Sync**
   - Extract filter/query logic into `useIssueFilters` + `useSearchParamSync` and adapt App shell.
4. **Phase 4 – UI State Hooks**
   - Replace snackbar, dialog, theme, and body-scroll effects with dedicated hooks.
5. **Phase 5 – Cleanup**
   - Remove obsolete helpers, slim `App.tsx`, ensure layering boundaries (e.g., shared constants file), and adjust imports.

## Testing Strategy
- Add unit tests for new hooks (`useIssueFilters`, `useSearchParamSync`, `useSnackbarQueue`).
- Smoke-test App shell via existing scenario tests (`make test`) after each phase or major milestone.
- Perform targeted manual QA (websocket updates, filter changes, modal interactions) after the provider migration.

## Risks & Mitigations
- **Websocket regressions**: mitigate with manual verification and provider-level tests/mocks.
- **URL sync edge cases**: cover with hook unit tests simulating push/pop flows.
- **Context rerender churn**: memoize selectors or split contexts if performance degrades.

## Open Questions
- Websocket hookup: keep inside `IssueDataProvider` (preferred for cohesion).
- Snackbar delivery: deliver via context so any component can enqueue notifications.

## Status Tracking
- ✅ Phase 1 – Services extraction complete (`services/issues.ts`).
- ✅ Phase 2 – Issue data provider in place (`IssueTrackerDataProvider`).
- ✅ Phase 3 – Filters and URL sync managed via dedicated hooks.
- ✅ Phase 4 – Snackbar, dialog, theme, and scroll-lock hooks shipped with tests.
- ✅ Phase 5 – App shell slimmed and shared constants extracted.
- 🔭 Follow-up – Continue shrinking `AppContent` into composable view/controller hooks for easier testing.
