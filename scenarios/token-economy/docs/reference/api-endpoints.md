# API Endpoints — Token Economy

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/token-economy/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/token-economy/v1/shared/errors.proto`):

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
| **CLI** | `token-economy status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/token-economy/v1/shared/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Authenticated Connect services

All product RPCs are Connect `POST` methods. The access interceptor validates a
scenario-authenticator JWT and requires the scope shown below before a domain
delegate runs.

### `MinterService` — scope `token-economy:minter`

| Domain | Methods | CLI group |
|---|---|---|
| Token types | `CreateTokenType`, `GetTokenType`, `ListTokenTypes`, `RetireTokenType`, `MintSupply` | `mints` |
| Grants | `CreateGrant`, `GetGrant`, `ListGrants`, `RevokeGrant` | `grants` |
| Holders | `CreateHolder`, `GetHolder`, `ListHolders` | `holders` |
| Journal | `ListJournalEvents`, `ShowBalance`, `ExportJournal`, `ReverseEvent` | `journal` |
| Catalog | `CreateCatalogEntry`, `UpdateCatalogEntry`, `GetCatalogEntry`, `ListCatalogEntries`, `RetireCatalogEntry` | `catalog` |
| Redemption | `ListPendingRedemptions`, `ApproveRedemption`, `DenyRedemption` | `redemption` |

`UpdateGrantRule` remains present in the stable descriptor but deliberately
returns unimplemented: issued rule evidence is immutable, so callers revoke and
reissue a grant instead.

### `HolderService` — scope `token-economy:holder`

| Domain | Methods | CLI group |
|---|---|---|
| Holder projection | `ViewEconomy` | `holders` |
| Catalog | `BrowseCatalog` | `catalog` |
| Redemption | `RequestRedemption` | `redemption` |

`SubmitRequest` remains an explicitly unimplemented placeholder until a general
holder-request product contract exists. No minter authority method appears on
this descriptor.

### `EarningService` — scope `token-economy:earning`

| Domain | Methods | CLI group |
|---|---|---|
| Earning adapters | `SubmitEarning`, `ListEarnings` | `earning` |

The authenticated subject supplies adapter/actor identity; the request cannot
assert either identity. `ListEarnings` returns the durable privacy-minimal
receipt (digest summary, grant outcome, adapter identity, and timestamp), not
the discarded raw earning reason or holder payload.

The full generated path is
`/vrooli.token_economy.v1.<package>.<Service>/<Method>`. The authoritative path,
request/response descriptor, and CLI mapping for each method are generated in
[`.vrooli/endpoints.json`](../../.vrooli/endpoints.json).

---

## Adding a new endpoint

For a new domain, copy the worked vertical slice in the fenced example
above first, then replace it once your real domain is green.

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/token-economy/v1/<domain>/`, then run
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
