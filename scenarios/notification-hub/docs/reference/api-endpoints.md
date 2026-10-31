# API Endpoints — Notification Hub

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/notification-hub/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/notification-hub/v1/shared/errors.proto`):

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
| **Errors** | None — always returns 200 and reports an unhealthy status if a dependency fails |
| **CLI** | `notification-hub status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/notification-hub/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Notification Hub read boundary

Ordinary reads do not require an interactive owner login. Recipient-scoped
procedures (`GetAsk`, `ListAsks`, notification `Get`/`List`, delivery
timeline/analytics, routing channel status, recipient metadata, devices, quiet
windows, escalation chain, and preference reads) use the recipient configured
for this Hub instance and return only matching recipient data; an unset
recipient yields no recipient data. The client cannot choose a recipient.
Recipient metadata lookup never creates a missing recipient row. The existing
routing-status call also refreshes its bounded status snapshot, using configured target IDs or `local` for cache keys. Notification detail receipts omit push provider IDs for every caller on this public configured-recipient projection. Catalog reads
expose source/category metadata only. Ask creation and answers,
notification send/relay, subscription/credential operations, and
recipient/device/preferences writes retain their existing identity checks.
Other endpoints are unchanged by this read rule. In particular, reading an ask
never authorizes its answer.

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.notification_hub.v1.<domain>.<Domain>Service/<Method>`
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
   `packages/proto/schemas/notification-hub/v1/<domain>/`, then run
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


## Request search and continuation

`ConversationsService/ListAsks` reads the current recipient's retained requests without
changing their state. `query` is a literal substring search over question/title, source,
answer/options, answer note and requester reference. `state_filter` accepts `all` (default),
`awaiting` (pending and escalated), `pending`, `escalated`, `answered`, `defaulted`, `expired`
and `withdrawn`. Search and state filtering apply on the server before pagination.

`limit` is a page size (default 50, maximum 200), not a total-result cap. Continue with
`next_cursor` as the next request's `cursor`; an empty value means the current query reached
the end. Cursors bind to recipient/query/state; changing those requires a fresh first page.
The compatibility `open_only` shortcut selects awaiting requests; conflicting filters fail
with InvalidArgument. Invalid/foreign-query cursors fail rather than silently restarting.

Pages use a live newest-first keyset with ID as the equal-time tiebreaker. Newly created
requests appear on refresh. State transitions can change a live result set between reads;
this is not a frozen snapshot. History exposes the full lifecycle and Needs you fixes the
state to awaiting. Both share search, accessible Load more/retry and detail-return context.
The product target is `../concepts/FLOWS.md`, amendment
`NH-DESIGN-20261025-HISTORY-COMPLETE`; older latest-200 screenshots are historical.
