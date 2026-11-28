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

Native detail includes target resolution, lifecycle evidence, health probe
evidence when execution is requested, HTTP semantics evidence, runtime hygiene
evidence, and summary counts. Migration accounting remains planned.

### `ScenarioValidationService.PreviewFix`

Shared dry-run fix RPC. Returns deterministic local file edits API Health can
apply for mechanical findings without writing.

| | |
|---|---|
| **Auth** | Local scenario transport |
| **Request** | `scenario-validation/v1.FixRequest` |
| **Response** | `scenario-validation/v1.FixResponse` |
| **CLI** | `api-health validate fix-preview <target> [rule_id ...]` |

### `ScenarioValidationService.ApplyFix`

Shared explicit-write fix RPC. Applies the same deterministic candidates exposed
by `PreviewFix`.

| | |
|---|---|
| **Auth** | Local scenario transport |
| **Request** | `scenario-validation/v1.FixRequest` |
| **Response** | `scenario-validation/v1.FixResponse` |
| **CLI** | `api-health validate fix-apply <target> [rule_id ...]` |

## Planned Native Endpoints

| Domain | Endpoint | Purpose | Status |
|---|---|---|---|
| validation | shared `ScenarioValidationService.ValidateScenario` | UI/CLI detail view over the same engine as the shared provider RPC, including provider-native target/probe evidence. | implemented |
| probe | `ProbeService.ProbeHealth` | Run one bounded live health probe and return evidence. | planned |
| remediation | native fix workbench endpoint | Provider-specific fix grouping and UI confirmation metadata over the shared Fix RPC. | planned |

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
