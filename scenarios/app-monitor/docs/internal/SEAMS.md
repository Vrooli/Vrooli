# SEAMS

## Stable Seams
- Preview navigation + bridge boundary:
  - `ui/src/hooks/usePreviewNavigationSession.ts`
  - `ui/src/components/views/usePreviewNavigation.ts`
- Preview URL orchestration:
  - `ui/src/hooks/usePreviewUrlOrchestration.ts`
- Lifecycle command seam:
  - `ui/src/hooks/usePreviewAppLifecycle.ts`
- Preview report/session seam (shared by single + pane preview):
  - `ui/src/hooks/usePreviewReportSession.ts`
- Workspace state seam:
  - `ui/src/features/preview-workspace/state/previewWorkspaceStore.ts`

## Weak Seams To Improve
- `AppPreviewView` still owns route-specific and feature orchestration in a single large file.
- Workspace layout type (`grid|split`) exists in store but UI only exposes interaction mode toggle.
