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

## Work ladder

- Rung: W2
- Evidence: App Monitor unit run `20261105-061756-db789895` failed the two `CheckIframeBridgeRule` tests because they used CLI scenario-port discovery despite supplying fake HTTP responses. The target scenario was stopped. After adding an injectable port resolver, the focused service tests and Test Genie unit run `20261105-071359-cc8125a7` pass without starting scenario-auditor.
- Blocker: closed; the unit tests now control the scenario-auditor port and HTTP responses.
- Measured: 2026-11-05

- Rung: W3
- Evidence: App Monitor unit run `20261105-071359-cc8125a7` reported `api/handlers/inspector.go` at 0.0% coverage. Repository search found no callers outside that file, no `InspectorHandler` registration in `api/server.go`, and no scenario summary/list endpoint in the declared PRD, README, API docs, CLI, or UI.
- Blocker: closed; removed the unregistered handler because it had no product contract or reachable behavior to test.
- Measured: 2026-11-05

- Rung: W3
- Evidence: `/api/v1/apps/:id/fixes/report` is registered in `api/server.go`. `ReportAppFix` returned HTTP 500 for every service error, although the shared handler mapper assigns `ErrSwarmManagerUnavailable` to 503 and malformed request payloads are already 400. The service's report validation failures now have typed sentinels; handler regressions cover validation 400, Swarm Manager 503, unexpected 500, and unchanged success response fields. Focused handler tests passed and App Monitor unit run `20261105-074011-10c99e5f` passed.
- Blocker: closed; the report endpoint maps known client/upstream failures according to the shared service error classes.
- Measured: 2026-11-05

- Rung: W3
- Evidence: `sanitizeIssueCaptures` accepted and validated `data:image/png;base64,...` but retained the data URL prefix. The Swarm evidence builder decodes the capture as bare base64, so a valid accepted capture failed before any Swarm request. The report service regression covers raw-base64, PNG data URL, unsupported JPEG data URL, and empty data URL/raw payloads; it checks exact bytes for supported inputs and confirms unsupported or empty captures are filtered. Focused service test passed; App Monitor unit run `20261105-080254-ca48ff64` passed.
- Blocker: closed; supported PNG data URLs are normalized to raw base64, while unsupported MIME and empty payloads are filtered to match the PNG evidence contract.
- Measured: 2026-11-05
