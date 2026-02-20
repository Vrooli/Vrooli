# React Coherence Notes

## Last Updated
2026-02-19

## State Management Patterns
- **Global state approach**: None (no Redux/Zustand/Context). All state lives in hooks or page components and is passed via props.
- **Component-local state**: `useState` for UI toggles, form inputs, and transient state.
- **Session state**: Centralized in `useSessionManager` hook ([CODE: ui/src/hooks/useSessionManager.ts]). Returns session list, pane state, and CRUD callbacks. Consumed by `Workspace`, `SessionDrawer`, `TerminalLauncher`, and page components.
- **Server state**: Fetched via `fetch()` calls in [CODE: ui/src/lib/api.ts]. No TanStack Query or SWR — manual fetch + setState pattern. Cache invalidation is "refetch on action" (create → re-list).
- **WebSocket state**: Managed per-pane in `useTerminalSocket` hook ([CODE: ui/src/hooks/useTerminalSocket.ts]). Connection lifecycle is scoped to the pane's mount/unmount.

## Duplication Audit
- **No significant duplication found**: Utils are consolidated in `lib/` and `consts/`. Policy options are shared via [CODE: ui/src/consts/policy-options.ts] across `SessionDrawer` and `SessionsPage`. The `useCountdown` hook ([CODE: ui/src/hooks/useCountdown.ts]) is shared by both surfaces that display expiration countdowns.
- **Previous duplication**: Utils were consolidated in a prior phase (see [DOC: docs/internal/UTILS_UNIFICATION_NOTES.md]).

## Styling Patterns
- **Approach**: Tailwind CSS with `cn()` utility for conditional classes ([CODE: ui/src/lib/classnames.ts])
- **Theme tokens**: Dark terminal aesthetic — `bg-gray-900`, `text-gray-100`, `border-gray-700` as base; accent colors for status indicators
- **Component library**: shadcn/ui button component ([CODE: ui/src/components/ui/button.tsx]) with CVA variants
- **Inconsistencies found**: None — all components follow the same Tailwind pattern

## Component Coherence
- **Prop naming conventions**: Consistent `onXxx` for callbacks, `isXxx` for booleans. Session-related props use `SessionResponse` type from API.
- **Error handling**: Components use `ErrorBanner` ([CODE: ui/src/components/ErrorBanner.tsx]) for API errors and `ErrorBoundary` ([CODE: ui/src/components/ErrorBoundary.tsx]) for render crashes. Pattern is consistent across all pages.
- **Loading state patterns**: Health check gate in `App.tsx` shows loading spinner during initial API check. Individual components show inline loading states during fetch operations.
- **Empty states**: Workspace shows "No active terminals" with launcher CTA. Session list shows "No sessions" message.

## Recommendations
1. Consider adopting TanStack Query for server state management if the scenario grows beyond current complexity (manual fetch/refetch pattern is adequate for current scope).
2. No immediate coherence improvements needed — patterns are consistent and well-documented.
