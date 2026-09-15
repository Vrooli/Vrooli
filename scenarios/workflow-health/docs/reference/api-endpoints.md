# API Endpoints — Workflow Health

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/workflow-health/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/workflow-health/v1/shared/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today: `invalid_request` (400),
`not_found` (404), `internal` (500). Add to the proto enum when a new
REST-exception failure mode appears.

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
| **Response** | `Response { status: string, readiness: bool, service: string, timestamp: string, version: string, uptime_seconds: int64, dependencies: map<string, DependencyStatus> }` |
| **Errors** | None — always returns 200 with `status: "unhealthy"` if a dependency fails |
| **CLI** | `workflow-health status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/workflow-health/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Validation

### `POST /scenario-validation/v1.ScenarioValidationService/ValidateScenario`

Validate a scenario's BAS workflow catalog without executing Browser Automation
Studio work. Execution uses the durable lifecycle service below.

| | |
|---|---|
| **Auth** | None |
| **Request** | `ValidateScenarioRequest { scenario: string, path: string }` |
| **Response** | `ValidateScenarioResponse` with workflow maturity assessment, findings, native catalog detail, and metrics |
| **Errors** | `invalid_argument` for missing or unresolvable target; `internal` for maturity response construction failures |
| **CLI** | `workflow-health validate scenario <scenario> [--path <dir>]` |

### Durable validation runs

`DurableValidationRunService` owns execution work in Workflow Health's SQLite
ledger. Start returns a persisted handle and preliminary static response; Get
reattaches, Wait blocks once without cancelling work, and Abort is the only
explicit cancellation route. Terminal runs embed the same shared
`ValidateScenarioResponse` used by direct and Test Genie consumers.

| RPC | CLI |
|---|---|
| `StartValidationRun` | `workflow-health validate start <scenario> --idempotency-key <key>` |
| `GetValidationRun` | `workflow-health validate get <run-id>` |
| `WaitValidationRun` | `workflow-health validate wait <run-id> [--timeout 5m]` |
| `AbortValidationRun` | `workflow-health validate abort <run-id> [--reason <reason>]` |

Callers do not poll: a timeout or disconnect leaves provider work running, and
the caller can later use Get or a single Wait to reattach.

If the Workflow Health process restarts while a run is queued or running, its
startup recovery policy records `RECOVERY_FAILED` rather than replaying BAS
work. This preserves evidence and prevents duplicate side effects; start a new
run with a new idempotency key only after inspecting the failed record.

### `POST /scenario-validation/v1.ScenarioValidationService/PreviewFix`

Preview deterministic fixes for workflow catalog findings.

| | |
|---|---|
| **Auth** | None |
| **Request** | `FixRequest { scenario: string, path: string, rule_ids: string[] }` |
| **Response** | `FixResponse` with candidate diffs and `applied: false` |
| **Errors** | `invalid_argument` for unresolvable targets or unsafe fix inputs |
| **CLI** | `workflow-health fix preview <scenario> [--path <dir>] [--rule <ids>]` |

### `POST /scenario-validation/v1.ScenarioValidationService/ApplyFix`

Apply deterministic workflow metadata and registry fixes.

| | |
|---|---|
| **Auth** | None |
| **Request** | `FixRequest { scenario: string, path: string, rule_ids: string[] }` |
| **Response** | `FixResponse` with applied candidate diffs and `applied: true` |
| **Errors** | `invalid_argument` for unresolvable targets or unsafe fix inputs |
| **CLI** | `workflow-health fix apply <scenario> [--path <dir>] [--rule <ids>]` |

## Workflows

### `POST /vrooli.workflow_health.v1.workflows.WorkflowSearchService/SearchWorkflows`

Search typed workflow leaves in a scenario-owned `bas/` catalog.

| | |
|---|---|
| **Auth** | None |
| **Request** | `SearchWorkflowsRequest { scenario: string, path: string, query: string, types: string[], include_fragments: bool, limit: int32 }` |
| **Response** | `SearchWorkflowsResponse` with ranked `workflow.flow`, `workflow.test`, and optionally `workflow.fragment` leaves |
| **Errors** | `invalid_argument` for missing query or unresolvable target |
| **CLI** | `workflow-health workflows search <query> [--scenario <id>] [--path <dir>] [--type <types>] [--include-fragments] [--limit <n>]` |

---

## Adding a new endpoint

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/workflow-health/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, bind it (or list it in `omitted[]`
   with a reason) in `cli/manifest.json` — the single source of truth for
   the CLI surface.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and the API↔CLI mapping
contract (every Connect endpoint is bound or omitted in `cli/manifest.json`).

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
