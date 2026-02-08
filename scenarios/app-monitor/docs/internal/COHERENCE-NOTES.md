# COHERENCE NOTES

## Preview Surface Coherence
- Single preview and workspace pane now share report-session logic via `usePreviewReportSession`.
- Both surfaces continue to share navigation/lifecycle orchestration hooks.

## Remaining Coherence Work
- Consider extracting a shared preview-body component for iframe/logs/device-emulation rendering.
- Continue shrinking `AppPreviewView` by splitting route concerns from preview-surface concerns.
