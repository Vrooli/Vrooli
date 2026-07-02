# API Endpoints — Business Health

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/business-health/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/business-health/v1/shared/errors.proto`):

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
| **CLI** | `business-health status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/business-health/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Validation & contract services (planned)

Business Health's product surfaces are exposed as proto-typed
Connect-RPC services under
`POST /vrooli.business_health.v1.<domain>.<Domain>Service/<Method>`,
with REST exceptions (such as multipart uploads) mounted at explicit
REST paths. These services are being built phase by phase; only the
`GET /health` endpoint above is live today. Document each RPC here as
it lands — one section per method, with its auth, request/response
proto shapes, error codes, and CLI mirror.

Planned services:

- **`ScenarioValidationService`** (shared) — `Validate`, `PreviewFix`,
  and `ApplyFix` over a target scenario's business contract. Other
  scenarios reuse this service for contract checks and remediation.
- **Business Health report service** — aggregates validation, drift,
  and traceability into the human and `--json` report surface.
- **Business Health wizard service** — drives the guided authoring flow
  for `PRD.md` and `requirements/`.

---

## Adding a new endpoint

For a new domain, add its proto service under
`packages/proto/schemas/business-health/v1/<domain>/`, wire the handler
module, then document each endpoint in a new section above.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/business-health/v1/<domain>/`, then run
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
