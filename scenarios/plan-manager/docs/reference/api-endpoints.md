# API Endpoints — Plan Manager

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/plan-manager/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/plan-manager/v1/shared/errors.proto`):

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
| **CLI** | `plan-manager status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/plan-manager/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.plan_manager.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `vrooli scenario detemplate
<scenario>` removes it once your real domains are green.

---

## PlansService ReconcilePlans

`POST /vrooli.plan_manager.v1.plans.PlansService/ReconcilePlans`

Repairs rendered markdown mirrors and canonicalizes misplaced markdown sources. The
request supports dry-run mode, mirror repair, source intake, archived-plan
inclusion, conflict policy (`report_only` or `skip_existing`), and source
selection for runtime-home `plans`, repo `docs/plans`, and repo `plans`.

The response returns per-item actions (`mirror_fresh`, `mirror_repair_needed`,
`mirror_repaired`, `import_planned`, `imported`, `skipped_duplicate`,
`parse_failed`, `conflict`, `already_canonical`) plus source path, mirror
metadata, and `source_untouched`. The operation is non-destructive to source
markdown sources. CLI mirror: `plan-manager plans reconcile`.

## PlansService RenderMarkdown

`POST /vrooli.plan_manager.v1.plans.PlansService/RenderMarkdown`

Returns the rendered markdown projection for a plan id or slug. The request
accepts `workspace` scope so slug lookup is deterministic across workspaces. The
response includes the markdown, rendered mirror metadata, a `repaired` flag, and
the resolved plan metadata so root and scenario CLIs can preserve provenance
without issuing a second lookup. CLI mirror: `plan-manager plans render`.

## ValidationService durable operations

Plan validation is a server-owned operation. `StartValidation` persists an
operation, its complete child oracle set, and an optional scoped idempotency key
before bounded concurrent dispatch. The remaining methods inspect or reattach
to that durable identity; transport lifetime never determines the verdict.

| RPC | Purpose | CLI mirror |
| --- | --- | --- |
| `StartValidation` | Persist queued operation/children and return the stable operation ID. Repeating the same `(plan, phase, idempotency_key)` returns the original operation. | `plan-manager validate start` |
| `GetValidationOperation` | Read current checkpoints and terminal result without waiting. | `plan-manager validate show` |
| `WaitValidationOperation` | Block once for the existing operation; timeout/cancel only detaches. | `plan-manager validate wait` |
| `ResumeValidationOperation` | Resume unfinished queued/running children after restart and block once. Terminal children are not replayed. | `plan-manager validate resume` |
| `RunValidation` | Compatibility blocking wrapper over the durable lifecycle. | `plan-manager validate run` |

Queue residence, operation execution, individual child execution, and transport
attachment have independent budgets. Every required oracle must be terminal and
comparable for PASS; missing, unavailable, timed-out, or not-comparable evidence
remains UNKNOWN/degraded. Unexpected EOF permits one non-blocking inspection by
operation ID and never a duplicate start.

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/plan-manager/v1/<domain>/`, then run
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
