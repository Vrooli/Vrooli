# API Endpoints — Unit Health

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/unit-health/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
The only REST exception, `GET /health`, uses the scenario error
envelope (`packages/proto/schemas/unit-health/v1/errors/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": {...} }
```

---

## System

### `GET /health`

Service health check. Returns API readiness plus dependency status.
Also mounted at `/api/v1/health` for client callers.
This is an operational REST exception by design: lifecycle systems,
load balancers, and curl probes must be able to read it without a Connect
client.

| | |
|---|---|
| **Auth** | None |
| **Response** | `Response { status: string, service: string, timestamp: string, readiness: bool, version: string, uptime_seconds: double, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `unit-health status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/unit-health/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Validation

The `validation` domain exposes one scenario-local service and
implements the platform-wide `ScenarioValidationService` so Test Genie
can drive it through the shared provider contract. Both are mounted by
`api/handlers/validation/module.go`.

### `POST /vrooli.unit_health.v1.validation.ValidationService/ValidateScenario`

Discovers test surfaces through Code Facts, plans and optionally runs
the canonical test commands, analyzes coverage, architecture, and test
quality, and returns normalized findings plus a shared maturity
assessment.

| | |
|---|---|
| **Auth** | None |
| **Request** | `ValidateScenarioRequest { scenario: string, path: string, workspaces: string[], include_execution: bool, use_cache: bool, fast_test_only: bool }` — `scenario` or `path` is required |
| **Response** | `ValidateScenarioResponse { run_id, status, summary, scenario, target_kind, target_path, degraded_reason, surfaces, workspaces, plan, command_results, coverage, findings, diagnostics, maturity, counts, next_steps, assessment, artifacts, projection_checks, suppressed_findings, cache_*, test_quality, traceability, evidence_stages }` |
| **Status values** | `passed`, `failed`, `degraded`, `error` |
| **Errors** | `invalid_argument` — scenario/path missing or unresolvable |
| **CLI** | `unit-health validate scenario <name> [--path <dir>] [--workspace <id>]... [--execution] [--fast-test-only] [--json]` |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.unit_health.v1.validation.ValidationService/ValidateScenario" \
  -H 'Content-Type: application/json' \
  -d '{"scenario":"unit-health","include_execution":false}'
```

With `include_execution: false` (the default) the response describes
the plan and static analyzers without running anything. UI and CLI
code should use the generated client instead of calling this path by
hand.

Field-level documentation lives in
`packages/proto/schemas/unit-health/v1/validation/validation.proto`
(run shape, findings, plan, coverage) and `test_quality.proto`
(test-quality and requirement-traceability reports).

### Shared provider contract — `/vrooli.scenario_validation.v1.ScenarioValidationService/*`

Defined in `packages/proto/schemas/scenario-validation/v1` (platform
owned, not this scenario). Unit Health mounts these procedures:

| Procedure | Purpose | CLI |
|---|---|---|
| `ValidateScenario` | Same engine as above; the native `ValidateScenarioResponse` is packed into `native_detail`. | `unit-health validate scenario` (covers both endpoints) |
| `ValidateTarget` | Validate a first-class repository target. | none |
| `DescribeProvider` | Provider identity, backed phase, maturity spec version, contract, build provenance, and capabilities. Inspects no target. | none |
| `PreviewFix` | Preview deterministic low-risk config/projection fixes without writing files. | none (consumed by Test Genie's fixer) |
| `ApplyFix` | Apply those fixes with before-write drift checks; `failed_precondition` (412) if a target file changed first. | none (consumed by Test Genie's fixer) |

```bash
curl -X POST "http://localhost:${API_PORT}/vrooli.scenario_validation.v1.ScenarioValidationService/DescribeProvider" \
  -H 'Content-Type: application/json' -d '{}'
```

The CLI manifest (`cli/manifest.json`) lists the shared procedures
without a command in its `omitted` array with a reason, and the
manifest coverage test fails if a new RPC has neither a binding nor an
omission entry.

---

## Adding a new endpoint

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/unit-health/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `api/handlers/<domain>/handler.go`; keep it thin and put behavior in
   `api/internal/<domain>/`.
3. Update the `Endpoints` descriptors in `api/handlers/<domain>/module.go`
   (paths must be generated `*Procedure` constants unless the entry
   carries a `RESTException`).
4. If the endpoint has a CLI mirror, add the command and its
   `binding` to `cli/manifest.json`; otherwise add an `omitted` entry.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and CLI manifest
coverage.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
