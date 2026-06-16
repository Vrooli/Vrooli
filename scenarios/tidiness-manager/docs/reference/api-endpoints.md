# API Endpoints

The API is served under `/api/v1` and is consumed by the CLI, UI, Test Genie, and other scenarios.

## Health

- `GET /health`
- `GET /api/v1/health`

Returns service readiness and database dependency status.

## Scans

- `POST /api/v1/scan/tidiness`
  - Primary maintainability scan for Test Genie and agents.
  - Body includes `scenario_name` and optional timeout.
- `POST /api/v1/scan/light`
  - Compatibility scan for metrics and parser workflows.
  - Body includes `scenario_path` and optional timeout.
- `POST /api/v1/scan/smart`
  - AI-backed file analysis.
  - Body includes scenario, files, optional campaign ID, and force-rescan flag.
- `POST /api/v1/scan/light/parse-lint`
- `POST /api/v1/scan/light/parse-type`
  - Parser compatibility endpoints for stored light-scan workflows.

## Agent Issues

- `GET /api/v1/agent/issues`
  - Query scenario issues with filters for file, folder, category, severity, status, and limit.
- `POST /api/v1/agent/issues`
  - Store an issue candidate.
- `PATCH /api/v1/agent/issues/{id}`
  - Resolve, ignore, or reopen an issue.
- `POST /api/v1/agent/issues/generate-from-metrics`
  - Generate issues from stored metrics.

## Scenario Summaries

- `GET /api/v1/agent/scenarios`
- `GET /api/v1/agent/scenarios/{name}`
- `GET /api/v1/agent/staleness`
- `GET /api/v1/agent/refactor-recommendations`

These endpoints power the UI and CLI recommendations.

## Tidiness Score

- `GET /api/v1/scenarios/{scenario}/tidiness`
- `GET /api/v1/scan/{scenario}`

The second route is retained for ecosystem-manager compatibility.

## Campaigns

- `POST /api/v1/campaigns`
- `GET /api/v1/campaigns`
- `GET /api/v1/campaigns/{id}`
- `POST /api/v1/campaigns/{id}/action`

Campaign actions include pause, resume, and terminate.

## Error Semantics

Client-facing errors should be sanitized. Server logs retain detailed context. Path, timeout, body-size, and unknown-field validation should happen before expensive scan work.
