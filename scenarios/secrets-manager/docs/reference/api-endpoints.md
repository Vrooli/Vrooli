# API Endpoints — Secrets Manager

## System

### `GET /health`

Returns lifecycle and dependency posture. `GET /api/v1/health` is the API-prefixed form for UI consumers.

## Domain Endpoints

| Area | Routes | Purpose |
|---|---|---|
| Credentials | `/api/v1/credentials/secrets/status`, `/validate`, `/provision` | Metadata-only coverage, validation, and stdin-guarded provisioning through the credential authority |
| Security | `/api/v1/security/scan`, `/compliance`, `/vulnerabilities` | Scan and posture reporting |
| Resources | `/api/v1/resources/{resource}` | Resource detail and strategy mutation |
| Deployment | `/api/v1/deployment/secrets`, `/readiness` | Bundle-safe strategy manifests |
| Scenarios | `/api/v1/scenarios`, overrides routes | Inventory and strategy overrides |
| Operations | orientation, campaigns, allowlist, watchlist, receipt signing | Operator workflows and supporting controls |

Secret values and authority management credentials are not response fields.

## Adding A New Endpoint

Add the handler to its capability route group in `api/server.go`, add handler tests, update `.vrooli/endpoints.json` through `make endpoints`, and document the stable contract here.

## Cross-References

- [CLI Commands](cli-commands.md)
- [Error Handling](../internal/ERROR-HANDLING.md)
