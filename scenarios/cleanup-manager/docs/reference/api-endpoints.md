# API Endpoints — Cleanup Manager

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/cleanup-manager/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/cleanup-manager/v1/shared/errors.proto`):

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
| **CLI** | `cleanup-manager status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/cleanup-manager/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.cleanup_manager.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

## Cleanup

The cleanup domain is defined in
`packages/proto/schemas/cleanup-manager/v1/cleanup/cleanup.proto`.
All cleanup operations are preview-first. `ApplyPlan` requires an
existing plan id, policy version, and idempotency key; providers with
owner/operator approval requirements also require the matching approval
mode and token.

#### `POST /vrooli.cleanup_manager.v1.cleanup.CleanupService/ListProviders`

| | |
|---|---|
| **Auth** | Operator-local API access |
| **Response** | `ListProvidersResponse { providers: Provider[] }` |
| **CLI** | `cleanup-manager cleanup providers` |

#### `POST /vrooli.cleanup_manager.v1.cleanup.CleanupService/GetPolicy`

| | |
|---|---|
| **Auth** | Operator-local API access |
| **Response** | `GetPolicyResponse { policy: Policy }` |
| **CLI** | `cleanup-manager cleanup policy` |

#### `POST /vrooli.cleanup_manager.v1.cleanup.CleanupService/SetPolicyProfile`

| | |
|---|---|
| **Auth** | Operator-local API access |
| **Request** | `SetPolicyProfileRequest { profile: string }` |
| **Response** | `SetPolicyProfileResponse { policy: Policy }` |
| **Errors** | `invalid_argument` — unknown profile |
| **CLI** | `cleanup-manager cleanup set-profile --profile <profile>` |

#### `POST /vrooli.cleanup_manager.v1.cleanup.CleanupService/CreatePlan`

| | |
|---|---|
| **Auth** | Operator-local API access |
| **Response** | `CreatePlanResponse { plan: Plan }` |
| **Errors** | `internal` — provider estimate/preview failure |
| **CLI** | `cleanup-manager cleanup plan` |

The plan id is a stable hash of policy version, provider versions,
provider policies, preview items, and blocked reasons. Creating a plan
does not mutate host state.

#### `POST /vrooli.cleanup_manager.v1.cleanup.CleanupService/ApplyPlan`

| | |
|---|---|
| **Auth** | Operator-local API access plus approval token when required |
| **Request** | `ApplyPlanRequest { plan_id, policy_version, approval_mode, approval_token, idempotency_key }` |
| **Response** | `ApplyPlanResponse { reclaimed_bytes, already_applied, results[] }` |
| **Errors** | `failed_precondition` — missing idempotency key, stale policy version, missing approval, unknown plan, provider version drift, or provider apply failure |
| **CLI** | `cleanup-manager cleanup apply ...` |

Repeating an apply with the same idempotency key returns the stored
result with `already_applied=true` and does not call providers again.

#### `POST /vrooli.cleanup_manager.v1.cleanup.CleanupService/ListAudit`

| | |
|---|---|
| **Auth** | Operator-local API access |
| **Response** | `ListAuditResponse { events: AuditEvent[] }` |
| **CLI** | `cleanup-manager cleanup audit` |

Audit messages are redacted before storage when they may contain host
paths or command output.

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/cleanup-manager/v1/<domain>/`, then run
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
