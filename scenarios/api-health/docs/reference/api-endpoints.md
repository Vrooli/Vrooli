# API Endpoints — API Health

Human-readable reference for the API. The machine-readable source of truth is
[`.vrooli/endpoints.json`](../../.vrooli/endpoints.json).

Wire shapes for provider endpoints should live in
`packages/proto/schemas/api-health/v1/<domain>/`. Proto-typed calls use
generated Connect-RPC handlers and clients.

## System

### `GET /health`

Service health check for API Health itself. Returns provider readiness plus
dependency status.

| | |
|---|---|
| **Auth** | None |
| **Response** | `api-core/health.Response`-compatible payload |
| **Errors** | None; unhealthy dependencies are represented in the payload |
| **CLI** | `api-health status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

## Validation Provider

### `ScenarioValidationService.ValidateScenario`

Shared Test Genie provider RPC. Validates a target scenario's API readiness and
returns canonical status, maturity assessment, metrics, and API Health native
detail.

| | |
|---|---|
| **Auth** | Local scenario transport |
| **Request** | `scenario-validation/v1.ValidateScenarioRequest` |
| **Response** | `scenario-validation/v1.ValidateScenarioResponse` |
| **CLI** | `api-health validate scenario <target>` |

Native detail is planned to include target resolution, capability summaries,
static findings, live probe evidence, migration accounting, and optional fix
preview metadata.

## Planned Native Endpoints

| Domain | Endpoint | Purpose | Status |
|---|---|---|---|
| validation | `ValidationService.ValidateScenarioDetail` | UI/CLI detail view over the same engine as the shared provider RPC. | planned |
| probe | `ProbeService.ProbeHealth` | Run one bounded live health probe and return evidence. | planned |
| remediation | shared `PreviewFix` / `ApplyFix` | Preview/apply deterministic fixes. | planned |

## Adding a new endpoint

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/api-health/v1/<domain>/`, then run `make generate`.
2. Implement the generated handler method in `api/handlers/<domain>/`.
3. Update endpoint metadata in the handler module.
4. Bind or explicitly omit the CLI mirror in `cli/manifest.json`.
5. Run `make endpoints`; do not edit `.vrooli/endpoints.json` by hand.
6. Update this document and add tests for the touched layers.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — provider architecture
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
