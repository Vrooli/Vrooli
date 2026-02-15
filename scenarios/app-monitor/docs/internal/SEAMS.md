# SEAMS

## Stable Seams
- Preview navigation + bridge boundary:
  - `ui/src/hooks/usePreviewNavigationSession.ts`
  - `ui/src/components/views/usePreviewNavigation.ts`
  - `ui/src/components/views/previewNavigationStateMachine.ts`
  - Transition contract is enforced through `previewNavigationActions` so hooks/components dispatch typed actions instead of ad-hoc payloads.
  - Pane reassignment resets call `previewNavigationActions.reset(true)` through `usePreviewNavigationSession.clearNavigationSession`.
  - Bridge `sync-from-bridge` transitions update persisted navigation state (`previewUrlInput/history/initialPreviewUrl`) and mark navigation as custom without replacing live iframe `src`; refresh/reload paths intentionally apply the latest committed route.
- Preview URL orchestration:
  - `ui/src/hooks/usePreviewUrlOrchestration.ts`
- Pane assignment reset seam:
  - `ui/src/features/preview-workspace/components/PreviewPane.tsx`
  - Resets pane-local custom URL/bridge state when `appId` assignment changes, preventing stale proxy targets from persisting across scenario swaps.
- Lifecycle command seam:
  - `ui/src/hooks/usePreviewAppLifecycle.ts`
- Preview report/session seam (shared by single + pane preview):
  - `ui/src/hooks/usePreviewReportSession.ts`
- Workspace state seam:
  - `ui/src/features/preview-workspace/state/previewWorkspaceStore.ts`
- Workspace minimap seam:
  - Pure mapping seam:
    - `ui/src/features/preview-workspace/utils/layout.ts`
    - `buildWorkspaceMinimapPaneMarkers(...)`
    - `scrollTopFromWorkspaceMinimapPointer(...)`
    - `workspaceViewportFromScrollMetrics(...)`
  - Imperative seam:
    - `ui/src/features/preview-workspace/components/PreviewWorkspaceView.tsx`
    - Owns scroll-container binding (`.preview-workspace__panes-scroll` or pinned `.preview-workspace__scroll-column`) and minimap pointer/keyboard interactions.
  - Preference seam:
    - `ui/src/features/preview-workspace/state/previewWorkspaceStore.ts`
    - `isWorkspaceMinimapVisible` is persisted and toggled by workspace controls.
- Pane fullscreen visibility seam:
  - `ui/src/features/preview-workspace/components/PreviewPane.tsx`
  - `ui/src/components/Shell.css`
  - Pane full view sets `body.preview-pane-fullscreen-active`; shell bottom navigation visibility now keys off that body-level contract so pane fullscreen can hide workspace-level bottom actions without coupling shell logic to pane internals.
- Preview toolbar compact/mobile seam:
  - `ui/src/components/AppPreviewToolbar.tsx`
  - `ui/src/components/AppPreviewToolbar.css`
  - Toolbar now owns compact-nav mode switching (`isFullView || max-width: 640px`) and small-screen suppression of lifecycle actions while preserving URL input visibility.

## Weak Seams To Improve
- `AppPreviewView` still owns route-specific and feature orchestration in a single large file.
- Workspace layout type (`grid|split`) exists in store but UI only exposes interaction mode toggle.
