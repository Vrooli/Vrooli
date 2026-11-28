# API Endpoints — Treasury

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/treasury/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/treasury/v1/shared/errors.proto`):

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
| **CLI** | `treasury status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/treasury/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.treasury.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Document your
domain's endpoints here as you build them — one section per RPC, with
its auth, request/response proto shapes, error codes, and CLI mirror.

The scaffold ships one fully worked CRUD vertical slice as a copyable
reference (see the fenced example below); `template-manager detemplate
<scenario>` removes it once your real domains are green.

---

## Budget policy

Budget caps, gating, and budget freeze controls belong to the budget-owned
`BudgetAdminService` in
`packages/proto/schemas/treasury/v1/budget/budget.proto`. The four operator
methods are `SetBudgetCaps`, `SetGating`, `FreezeBudget`, and
`UnfreezeBudget`. They are mounted by `handlers/budget/` and mirrored by the
Treasury CLI manifest. Agent-facing spend APIs expose authorization-owned
headroom projections; they do not grant agents policy mutation authority.

---

## Book administration

Book creation and lookup belong to the book-owned `BookAdminService` in
`packages/proto/schemas/treasury/v1/book/book.proto`. `CreateBook` binds the
beneficiary identity to the authenticated operator; caller-supplied identity
is ignored. Both `CreateBook` and `GetBook` require operator authentication.
Their CLI mirrors are listed under `treasury book`.

---

## Wallet administration and receipt observation

Receive-only operator wallets and Base native-USDC receipt observations belong to
`WalletAdminService` in `packages/proto/schemas/treasury/v1/wallet/wallet.proto`.
All wallet creation, lookup, control proof, recovery qualification/recovery,
portable encrypted recovery bundle export/restore, and receipt verify/recheck/list
methods require an authenticated operator. Receipt observation retains exact
provider-reported transfer evidence only; it neither posts Money Ledger entries
nor assigns valuation or budget credit. The CLI mirrors these operations under
`treasury wallet`; recovery secrets are accepted only through the explicit stdin
flow, and encrypted bundles are loaded from regular local files.

## Budget wallet session rotation

Operator budget-wallet provisioning remains available through TreasuryAdmin's
funding operation. For an active mandate, `rotate_session=true` requires a
`rotation_idempotency_key`; the same key must be reused for an exact retry. The
owner atomically retires the old generation and reserves the next one, then
retains the public result for replay after a process restart. Reusing the key
with changed run binding is rejected; a different key requests a distinct
rotation. The CLI exposes these fields under `treasury wallet provision-budget`
with confirmation required. Rotation changes the run-owned session generation
while preserving the derived budget address; it does not grant an agent policy
or budget mutation authority.

## Instrument administration

Instrument registration belongs to `InstrumentAdminService` in
`packages/proto/schemas/treasury/v1/instrument/instrument.proto`. The
operator-only `RegisterInstrument` binds the submitted credential reference
to mandate and counterparty scope. The reference is consumed internally and
is deliberately omitted from the returned `Instrument` message. The CLI
mirror is listed under `treasury instrument`.

---

## Mandate administration

Operator creation, listing, cancellation, and revocation live under the
mandate-owned `MandateAdminService` in
`packages/proto/schemas/treasury/v1/mandate/mandate.proto`. It also owns the
separate operator `RevokeMandateSession` lifecycle action. Revoking a mandate
also revokes its active session key. The separate `AgentSpend.ListMandates`
response is an authorization-owned projection that omits operator and signing
fields. CLI mirrors are listed under `treasury mandate`.

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/treasury/v1/<domain>/`, then run
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
