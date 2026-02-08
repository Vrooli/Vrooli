# App Monitor Architecture

App Monitor UI has two preview surfaces sharing a common preview pipeline:

- Single preview route: `ui/src/components/views/AppPreviewView.tsx`
- Workspace panes: `ui/src/features/preview-workspace/components/PreviewPane.tsx`

Shared preview seams:
- `ui/src/hooks/usePreviewNavigationSession.ts`
- `ui/src/hooks/usePreviewUrlOrchestration.ts`
- `ui/src/hooks/usePreviewToolbarSession.ts`
- `ui/src/hooks/usePreviewAppLifecycle.ts`
- `ui/src/hooks/usePreviewReportSession.ts`

Workspace orchestration:
- `ui/src/features/preview-workspace/components/PreviewWorkspaceView.tsx`
- `ui/src/features/preview-workspace/state/previewWorkspaceStore.ts`

See also root architecture detail in `../ARCHITECTURE.md`.
