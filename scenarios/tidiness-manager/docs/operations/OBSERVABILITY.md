# Observability

## Health

The API exposes health at `/health` and `/api/v1/health`. Lifecycle health checks should use these endpoints.

## Logs

Use scenario lifecycle logs:

```bash
cd scenarios/tidiness-manager
make logs
```

Server-side logs should retain detailed scan and database context while HTTP responses stay sanitized.

## Test Genie Artifacts

Scenario validation writes phase results under `coverage/runs/<run-id>/phase-results/`. Use the latest run artifacts to inspect docs, quality, security, and tidiness findings.

## Operational Signals

Important signals include scan duration, scan status, optional-tool degradation, issue count by severity/category, campaign status, campaign error reason, and database readiness.
