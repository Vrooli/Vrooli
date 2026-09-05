# API Endpoints — Source Ledger

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/source-ledger/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/source-ledger/v1/shared/errors.proto`):

```json
{ "code": "<canonical_code>", "message": "<human readable>", "details": [...] }
```

Canonical REST codes used today are `invalid_request`, `not_found`, and
`internal`. Add to the proto enum when a new
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
| **Errors** | None — always returns a success response and reports an unhealthy status if a dependency fails |
| **CLI** | `source-ledger status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/source-ledger/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Ledger domain endpoints

Each product domain exposes its endpoints under
`POST /vrooli.source_ledger.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/source-ledger/v1/<domain>/`, then run
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

## Architecture Maturity

The current endpoint surface is the lifecycle health contract. Domain RPCs
are reserved for the service-contract phase and will be generated from the
source-ledger proto package.

## Contracts And Data Flow

The health endpoint is the current lifecycle edge; future domain RPCs will
carry scoped requests through generated Connect contracts.

## Team Corpus Contract

The team-adoption contract is the consumer-facing shape for the service
contract phase. These operations use the vocabulary already proven in
vrooli-memory and always carry an explicit `scope`:

| Operation | Request shape | Response guarantee |
|---|---|---|
| `scopes create` | scope id, facet vocabulary, `frontier_target`, `max_entry_lines`, `wake_budget` | create or reuse one scope; reject an undersized wake budget |
| `scopes list` | optional scope filter | registered scopes and their budgets |
| `journal note` | scope, prose body, optional kind and provenance | append one immutable journal entry |
| `recall wake` | scope and line budget | bounded wake block that identifies its scope |
| `recall recall` | scope and semantic query | results restricted to the named scope |
| `rules` / `facets` | scope policy inspection and rule operations | the configured classification vocabulary and policy |

`scopes create` must fail when `wake_budget` cannot hold
`frontier_target` entries at `max_entry_lines` each. Journal rows remain
append-only; corrections are new entries with an explicit supersession mark.
The missing governed RPC bindings are tracked by Swarm Manager capture
`cap-ec8f3c2ee6b5f2ab` and belong to the active Source Ledger engine plan.

## Cross-references

- [`cli-commands.md`](cli-commands.md) — CLI commands that mirror these endpoints
- [`configuration.md`](configuration.md) — env vars (e.g., `API_PORT`)
- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md#proto-as-the-canonical-contract) — proto bridge details
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — handler/service/repository seams
- [`../internal/TESTING.md`](../internal/TESTING.md) — endpoint test patterns
