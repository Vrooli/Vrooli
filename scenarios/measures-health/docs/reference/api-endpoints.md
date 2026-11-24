# API Endpoints — Measures Health

The machine-readable source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json). Wire contracts live in the proto schemas under `packages/proto/schemas/measures-health/v1/`; generated Connect handlers and clients use those shared types.

## Current endpoint surface

The registered API includes the following owner domains and RPCs. Exact endpoint metadata remains in [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) and the `Endpoints` declarations beside each handler module.

| Domain | Operations | Contract / implementation |
|---|---|---|
| Health | `GET /health` (also `/api/v1/health`) | Operational readiness probe; `api/handlers/health/`. |
| Scenario validation | `ValidateTarget`, `ValidateScenario`, `DescribeProvider`, `PreviewFix`, `ApplyFix` | Shared `scenario-validation` contract; `api/handlers/validation/`. The scenario request can include `include_execution` to probe declared measures. |
| Measure validation | `ListFleetCoverage` | `measures-health/v1/validation`; `api/handlers/validation/`. |
| Search | `Search`, `Status` | `measures-health/v1/search`; `api/handlers/search/`. |
| Measures | `CountFailedValidations`, `CountValidationCoverage` | `measures-health/v1/measures`; counts over the `validation_runs` history. |

Connect-RPC errors use Connect's canonical error codes. `/health` is the intentional operational REST probe so lifecycle tools and health checks do not need a generated client. Endpoint paths and registered handlers are verified by the endpoint-generation gate.

## Adding an endpoint

1. Add or extend the proto under `packages/proto/schemas/` and generate its bindings.
2. Implement the owner handler and register its module/endpoint metadata.
3. Add a CLI binding only when the command is part of the supported interface; update the command seed as required by endpoint generation.
4. Update this reference and add focused handler/contract tests.
5. Run the scenario's endpoint-generation and test gates; do not hand-edit `.vrooli/endpoints.json`.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI surface
- [`configuration.md`](configuration.md) — runtime configuration
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — API and proto boundaries
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — substitutable dependencies
- [`../internal/TESTING.md`](../internal/TESTING.md) — test entry points
