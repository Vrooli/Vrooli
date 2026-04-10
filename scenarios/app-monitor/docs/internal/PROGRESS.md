# PROGRESS

## 2026-02-13
- Added a toggleable workspace minimap for preview panes in:
  - `ui/src/features/preview-workspace/components/PreviewWorkspaceView.tsx`
  - `ui/src/features/preview-workspace/components/PreviewWorkspaceView.css`
- Added pure minimap mapping helpers and unit coverage in:
  - `ui/src/features/preview-workspace/utils/layout.ts`
  - `ui/src/features/preview-workspace/utils/layout.test.ts`
- Added persisted minimap preference to workspace state and tests in:
  - `ui/src/features/preview-workspace/state/previewWorkspaceStore.ts`
  - `ui/src/features/preview-workspace/state/previewWorkspaceStore.test.ts`
- Added workspace manager toggle coverage in:
  - `ui/src/components/workspace/WorkspaceManagerDialog.tsx`
  - `ui/src/components/workspace/WorkspaceManagerDialog.test.tsx`

## 2026-02-07
- Added shared preview report/session hook used by:
  - `ui/src/components/views/AppPreviewView.tsx`
  - `ui/src/features/preview-workspace/components/PreviewPane.tsx`
- Removed unused legacy hooks.
- Updated stale `ui/README.md` script commands to current `pnpm` workflow.
- Added docs navigation manifest and internal seam/problem/progress notes.
