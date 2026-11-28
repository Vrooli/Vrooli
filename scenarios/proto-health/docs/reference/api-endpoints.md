# API Endpoints — Proto Health

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/proto-health/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/proto-health/v1/shared/errors.proto`):

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
| **CLI** | `proto-health status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/proto-health/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Proto-health domains

The generated `notes` API and upload route were removed. Current product
RPCs are generated Connect procedures:

| Procedure | Purpose | CLI/UI owner |
|---|---|---|
| `/vrooli.scenario_validation.v1.ScenarioValidationService/ValidateScenario` | Run descriptor-backed policy checks for one scenario. | `proto-health validate scenario <name>` / validation panel |
| `/vrooli.proto_health.v1.validation.ProtoHealthService/DescribeScenarioProtos` | Return one scenario's descriptor-backed proto surface. | `proto-health describe scenario <name>` / proto-health panel |
| `/vrooli.proto_health.v1.validation.ProtoHealthService/DescribeScenariosProtos` | Return per-scenario surface results for a batch. | Programmatic consumers; intentionally omitted from human CLI. |
| `/vrooli.proto_health.v1.impact.ImpactService/GetImpact` | Compare a scenario surface with a git baseline. | `proto-health impact scenario <name>` |

The current wire contracts are in `packages/proto/schemas/scenario-validation/v1/validation.proto`, `packages/proto/schemas/proto-health/v1/validation/validation.proto`, and `packages/proto/schemas/proto-health/v1/impact/impact.proto`. `validate owner` is a local CLI command that checks source schemas before descriptor generation; it is not an API endpoint.

---

## Adding a new endpoint

For a new domain, follow the current API/CLI/UI vertical slices and
their owning tests; there is no active notes slice to copy.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/proto-health/v1/<domain>/`, then run
   `make generate`.
2. Implement the generated handler method in
   `handlers/<domain>/connect_handler.go`; keep it thin.
3. Update endpoint metadata in `handlers/<domain>/module.go`.
4. If the endpoint has a CLI mirror, update
   `api/cmd/gen-endpoints/cli_commands_seed.json`.
5. Run `make endpoints`; do not edit
   [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) by hand.
6. Update this document and add tests for the touched layers.
7. Add a row to [`internal/SEAMS.md`](../internal/SEAMS.md) if you
   introduced a new interface that production wires once and tests
   substitute.

The CI gate enforces endpoint-manifest freshness and command-seed
consistency.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
