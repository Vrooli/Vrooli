# API Endpoints — Tunnel Manager

Human-readable reference for the API. The machine-readable
source of truth is [`.vrooli/endpoints.json`](../../.vrooli/endpoints.json) —
doc generators, Postman collection builders, and SDK stubs read it
directly. The CI gate fails if the JSON drifts from the registered
handlers or from the CLI commands it claims to mirror.

Wire shapes for every endpoint live in
`packages/proto/schemas/tunnel-manager/v1/<domain>/<file>.proto`.
Proto-typed calls use generated Connect-RPC handlers and clients.
Tests, handlers, UI clients, and CLI handlers all consume generated
types — no hand-written struct mirror exists to drift.

Connect-RPC errors use Connect's canonical error envelope and code set.
REST exceptions, such as multipart uploads, use the template error
envelope (`packages/proto/schemas/tunnel-manager/v1/errors/errors.proto`):

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
| **CLI** | `tunnel-manager status` |

```bash
curl "http://localhost:${API_PORT}/health"
```

The proto type lives at `packages/proto/schemas/tunnel-manager/v1/health/health.proto`
and mirrors `api-core/health.Response` field-for-field.

---

## Domain endpoints — `<domain>`

Each product domain exposes its endpoints under
`POST /vrooli.tunnel_manager.v1.<domain>.<Domain>Service/<Method>`
for proto-typed Connect-RPC calls, with REST exceptions (such as
multipart uploads) mounted at explicit REST paths. Endpoint metadata is
generated from the handler modules and mirrored by CLI manifest coverage.

Read RPCs are local/operator-readable by default. Privileged mutation
RPCs can be fail-closed with `TUNNEL_MANAGER_AUTHZ_ENFORCED=1`; when
enabled, they require `Authorization: Bearer <token>` or
`X-Vrooli-Operator-Token: <token>` matching
`TUNNEL_MANAGER_OPERATOR_TOKEN`, falling back to `API_TOKEN`. Missing
tokens return Connect `Unauthenticated`; wrong tokens return
`PermissionDenied`.

## Implemented Connect-RPC services

Each RPC is reached at
`POST /vrooli.tunnel_manager.v1.<domain>.<Service>/<Method>`. See
[`../concepts/DOMAINS.md`](../concepts/DOMAINS.md) for domain ownership
and [`../../PRD.md`](../../PRD.md) for the operational targets each
method satisfies.

### `routes` — `RoutesService` (exposure manifest SSOT)

| RPC | Purpose |
|---|---|
| `ListRoutes` | List the exposure manifest (subdomain, scenario, domain, local port, tier, lease, enabled). |
| `GetRoute` | Fetch a single route by subdomain or scenario. |
| `CreateRoute` | Create a manifest entry. Privileged mutation. |
| `UpdateRoute` | Update a manifest entry, including enabled state. Privileged mutation. |
| `DeleteRoute` | Remove a manifest entry. Privileged mutation. |

### `exposure` — `ExposureService` (tiered broker)

| RPC | Purpose |
|---|---|
| `Expose` | Request leased exposure of a scenario (default TTL ≈ 1 week); ensures a route + running scenario + ingress. Privileged mutation. |
| `ExtendLease` | Extend an active lease's TTL. Privileged mutation. |
| `RevokeLease` | Revoke a lease early and tear down its ingress. Privileged mutation. |
| `ListLeases` | List active/expired leases. |
| `ListExposures` | List current exposure state with route and probe classification context. |
| `IsExposed` | Query whether a scenario is currently exposed (backs app-monitor). |
| `Reconcile` | Reconcile every `api-core/coreset` member as always-on CORE exposure and reap expired leases. Privileged mutation. |

### `config` — `ConfigService` (Cloudflare ingress & mode)

| RPC | Purpose |
|---|---|
| `GetConfig` | Read current mode (`remote`/`local`), tunnel/account id, Prometheus endpoint, and browser-safe setup readiness. |
| `GetCredentialStatus` | Read Cloudflare credential presence/source metadata without secret values. |
| `SetCloudflareCredentials` | Store write-only Cloudflare credential values in the operator secret store. Privileged mutation. |
| `ClearCloudflareCredentials` | Clear file-backed Cloudflare credential values from the operator secret store. Privileged mutation. |
| `Sync` | Reconcile Cloudflare ingress (remote) or `~/.cloudflared/config.yml` (local) with the manifest. Privileged mutation. |
| `SwitchMode` | Switch and migrate between remote and local mode. Privileged mutation. |

### `audit` — `AuditService` (port-compliance)

| RPC | Purpose |
|---|---|
| `RunAudit` | Compute port-compliance findings from scenario `service.json` vs the manifest. |
| `ListFindings` | List the latest port-compliance findings (mismatch / missing / ranged). |

### `tunnel` — `TunnelService` (health & metrics)

| RPC | Purpose |
|---|---|
| `GetStatus` | Read cloudflared systemd status + `/ready` + degraded-mode signal. |
| `ListMetrics` | Time-series metrics from the `metrics` table. |
| `Scrape` | Scrape current cloudflared metrics and persist the sample. |

### `probes` — `ProbesService` (liveness & classification)

| RPC | Purpose |
|---|---|
| `RunProbes` | Probe each exposed route internally (local port) and externally (public URL). |
| `ListProbes` | Probe history (route, kind, status, latency, error). |
| `Classify` | Classify latest probe pairs (healthy / tunnel-down / scenario-down / config-drift). Cloudflare-outage and DNS-failure are planned richer signals, not currently produced. |

### `recovery` — `RecoveryService` (manual + opt-in scheduled recovery)

| RPC | Purpose |
|---|---|
| `GetState` | Current backoff / circuit-breaker state. |
| `Recover` | Manually trigger a recovery attempt (restart cloudflared). Privileged mutation. |
| `ListEvents` | Recovery event log (trigger, action, outcome, timestamps). |

### `health` — `HealthService`

| RPC | Purpose |
|---|---|
| `Check` | Runtime readiness + dependency reachability (also mounted at `GET /health`). |

---

## Adding a new endpoint

For an endpoint inside an existing domain:

1. Add or extend the `.proto` messages and service in
   `packages/proto/schemas/tunnel-manager/v1/<domain>/`, then run
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
