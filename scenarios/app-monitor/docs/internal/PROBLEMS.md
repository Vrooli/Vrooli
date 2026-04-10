# PROBLEMS

## Active
- `AppPreviewView` remains high-complexity and high-churn.
- Workspace layout state has partial implementation (no layout mode controls in view yet).
- Route-level coverage for redirects (`/logs/:appId` -> preview with `paneLogs=1`) is still light.

## Recently Addressed
- Removed dead legacy hooks:
  - `ui/src/hooks/useAppLifecycle.ts`
  - `ui/src/hooks/useFullscreenMode.ts`
- Consolidated duplicated preview report logic into:
  - `ui/src/hooks/usePreviewReportSession.ts`
