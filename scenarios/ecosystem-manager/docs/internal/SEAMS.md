# Seams

## Auto Steer Execution State
- **API seam:** `/auto-steer/execution/{taskId}` for read, `/auto-steer/execution/seek` for manual initialization.
- **UI seam:** `useAutoSteerExecutionState` treats 404 as "missing state" while surfacing real errors.
- **Lifecycle seam:** task create/update handlers call `maybeInitializeAutoSteer` to best-effort initialize state before execution.

## API Error Surface
- **Client seam:** `ApiError` carries HTTP status codes so UI can distinguish missing state vs failures.
- **Hook seam:** Auto Steer state query translates 404 into undefined while preserving other errors.

## Architecture Alignment Notes
- Auto Steer initialization now has explicit entry points in both task lifecycle and UI, reducing hidden coupling with execution-time only initialization.
- Error handling is centralized at the API client seam, preventing UI components from inferring status codes from message strings.
