# React Coherence Notes

## Last Updated
2026-03-17

## State Management Patterns
- **Global state approach**: Lightweight Zustand store in [CODE: ui/src/stores/useWorkspaceStore.ts] for cross-surface workspace preferences and modal state.
- **Component-local state**: `useState` remains the default for transient section state inside the unified settings surface.
- **Session state**: Centralized in `useSessionManager` hook ([CODE: ui/src/hooks/useSessionManager.ts]). Returns session list, pane state, and CRUD callbacks. Consumed primarily by [CODE: ui/src/components/Workspace.tsx] and the unified settings surface.
- **Server state**: Fetched via `fetch()` calls in [CODE: ui/src/lib/api.ts]. No TanStack Query or SWR — manual fetch + setState pattern. Cache invalidation is "refetch on action" (create → re-list).
- **WebSocket state**: Managed per-pane in `useTerminalSocket` hook ([CODE: ui/src/hooks/useTerminalSocket.ts]). Connection lifecycle is scoped to the pane's mount/unmount.

## Duplication Audit
- **Settings/session duplication removed**: session management now lives inside [CODE: ui/src/components/settings/SessionManagementSection.tsx] and is presented through the single [CODE: ui/src/components/SettingsModal.tsx] shell. The old split between standalone settings and session surfaces is gone.
- **Tab ownership is explicit**: settings information architecture is centralized in [CODE: ui/src/components/settings/tabs.tsx], while tab content is split into one section module per concern.
- **Previous duplication**: Utils were consolidated in a prior phase (see [DOC: docs/internal/UTILS_UNIFICATION_NOTES.md]).

## Styling Patterns
- **Approach**: Tailwind CSS with `cn()` utility for conditional classes ([CODE: ui/src/lib/classnames.ts])
- **Theme tokens**: Dark terminal aesthetic — `bg-gray-900`, `text-gray-100`, `border-gray-700` as base; accent colors for status indicators
- **Component library**: shadcn/ui button component ([CODE: ui/src/components/ui/button.tsx]) with CVA variants
- **Inconsistencies found**: None — all components follow the same Tailwind pattern

## Component Coherence
- **Prop naming conventions**: Consistent `onXxx` for callbacks, `isXxx` for booleans. Section props now follow a single shell contract for close/delete/open concerns.
- **Error handling**: Components use `ErrorBanner` ([CODE: ui/src/components/ErrorBanner.tsx]) for API errors and `ErrorBoundary` ([CODE: ui/src/components/ErrorBoundary.tsx]) for render crashes. Pattern is consistent across all pages.
- **Loading state patterns**: Health check gate in `App.tsx` shows loading spinner during initial API check. Individual components show inline loading states during fetch operations.
- **Empty states**: Workspace shows "No active terminals" with launcher CTA. The unified settings surface provides section-specific empty states instead of a separate sessions page/modal.

## Recommendations
1. Consider moving repeated async loading patterns in settings sections behind small feature hooks if the settings surface grows significantly.
2. Keep future settings additions inside `ui/src/components/settings/` and register them in `tabs.tsx` instead of extending the shell directly.
