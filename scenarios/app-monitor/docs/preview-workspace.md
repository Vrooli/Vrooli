# Preview Workspace Architecture

## Goal
Enable concurrent scenario previews with pane-local state while preserving existing app-monitor routes.

## Module Boundaries
- `ui/src/features/preview-workspace/state/previewWorkspaceStore.ts`
  - Owns workspace-level concerns: pane collection, focused pane, and layout mode.
  - No iframe, routing, or API logic.
- `ui/src/features/preview-workspace/components/PreviewWorkspaceView.tsx`
  - Composes panes and binds workspace store actions to UI controls.
  - Contains no app-fetch or bridge protocol logic.
- `ui/src/features/preview-workspace/components/PreviewPane.tsx`
  - Owns pane-local preview lifecycle: selected app, URL input/history, bridge wiring, iframe rendering, and pane-local logs mode.

## Testing Seams
- `resolveWorkspaceLayout` in `ui/src/features/preview-workspace/utils/layout.ts` is pure and unit tested.
- Workspace store actions are unit tested without rendering components.
- Pane component depends on existing seams (`useIframeBridge`, `usePreviewNavigation`, `appService`) so protocol behavior remains centralized.

## Overlay Scope Decision
- Global overlay query (`overlay`) remains reserved for shell-level overlays (`tabs`, `actions`).
- Preview logs mode is pane-local (`isLogsVisible`) and optional deep-link open uses `paneLogs=1`.

## Tab Switcher Integration
- Tab Switcher now exposes scenario open modes:
  - `Single Preview` -> `/apps/:appId/preview`
  - `Focused Pane` -> `/apps/workspace?workspaceAppId=<id>&workspaceMode=replace-focused`
  - `New Pane` -> `/apps/workspace?workspaceAppId=<id>&workspaceMode=add-pane`
- `PreviewWorkspaceView` consumes these query intents once, applies pane mutations, and clears intent keys from the URL.
- Keyboard shortcuts in Tab Switcher:
  - `Alt+O` cycles open mode
  - `Alt+1` selects `Single Preview`
  - `Alt+2` selects `Focused Pane`
  - `Alt+3` selects `New Pane`
