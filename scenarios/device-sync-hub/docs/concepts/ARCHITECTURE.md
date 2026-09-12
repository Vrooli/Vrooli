# Architecture — Device Sync Hub

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, how the device-sync-hub
domains compose on top of it, and where the load-bearing seams sit. It
then points to the specialized documents that own product domains,
workflows, data, integrations, deployment, operations, and strategy.

Keep this file high-signal. Do not turn it into a warehouse for every
domain, endpoint, workflow, or decision. If a concern has a dedicated
document below, update that document and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape,
- the role of each surface,
- how contracts and data flow between surfaces,
- where auth middleware and the storage seams sit,
- the shared infrastructure boundary,
- extension rules for future code,
- architecture maturity and intentional deviations.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage details and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- deployment and operations: [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md),
- commercial strategy: [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Mental Model

Device Sync Hub is a **server-relayed** transfer hub for a single
owner's many trusted devices. Every transfer is `device → server →
device`; there is no peer-to-peer in v1. A device authenticates (auth),
joins the trust group (devices), pushes an item that is stored on the
server (transfer), and the item is fanned out live to the owner's other
online devices (realtime). The owner tunes defaults and manages devices
in one place (settings). `auth` is a thin integration boundary over the
existing `scenario-authenticator` scenario — this scenario does **not**
own user identity, JWTs, or sessions; it owns *devices*, *pairing*,
*trust*, *presence*, and *per-item ACL*.

## Scenario Shape

A scenario is one product expressed through three coordinated surfaces
and one canonical contract layer, plus a persistent WebSocket channel
co-located in the Go API process.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   device-sync-hub/v1/...    │
                       └──────────────┬──────────────┘
                                      │ canonical wire shape
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
        ┌──────────┐  Connect-JSON ┌──────────┐ Connect-JSON ┌──────────┐
        │   ui/    │ ◀───────────▶ │   api/   │ ◀──────────▶ │   cli/   │
        │ React    │   + WebSocket │   Go     │              │   Go     │
        │ +Vite+TW │ ◀───────────▶ │ HTTP+WS  │              │ cli-core │
        └──────────┘               └────┬─────┘              └──────────┘
                                        │
                          ┌─────────────┴─────────────┐
                          ▼                           ▼
                   ┌─────────────┐            ┌───────────────┐
                   │ api-core/   │            │ api-core/     │
                   │ storage     │            │ blobstore     │
                   │ (SQLite     │            │ (binary       │
                   │  metadata)  │            │  payloads)    │
                   └─────────────┘            └───────────────┘
                                        │
                                        ▼  HTTP (fail-closed)
                              ┌──────────────────────┐
                              │ scenario-authenticator│
                              │  (identity/JWT/session)│
                              └──────────────────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge, WebSocket gateway | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | React + Vite + Tailwind on the vrooli-default "Operational Console" kit; split-screen Send/Receive; components, i18n, accessibility | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, JSON output, API invocation (send, receive, devices, revoke) | Business rules, duplicated validation |
| Contracts (`packages/proto/schemas/device-sync-hub/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## How Domains Compose

Domains layer from the trust boundary inward. See [`DOMAINS.md`](DOMAINS.md)
for ownership detail.

```
   request ─▶ auth middleware ─▶ devices (trust check) ─▶ transfer / settings
                  │                    │                        │
                  ▼                    ▼                        ▼
        scenario-authenticator     presence (realtime)     storage + blobstore
        (validate / revoke)        WebSocket fan-out        (metadata + bytes)
```

- **auth** wraps every mutating and reading route in middleware that
  validates the owner's access token against `scenario-authenticator`
  and attaches request-scoped owner identity. In production it **fails
  closed**: if the authenticator is unreachable beyond a brief cache
  window (default ≤ 60 s), the request is rejected.
- **devices** turns request-scoped owner identity into a concrete
  trusted device. No item route proceeds unless the calling device is
  in the owner's trust group; an unpaired device sees nothing.
- **transfer** owns the item lifecycle: streaming/chunked upload to the
  blobstore, metadata in SQLite, retention (Live/Held/Pinned), delivery
  ACL (broadcast vs directed), and quota enforcement.
- **realtime** is the WebSocket gateway: it tracks per-device presence
  and fans out `item-arrived` / `pairing-request` / presence events to
  the owner's online, authenticated devices.
- **settings** holds the owner config singleton and the
  device-management surface; destructive actions are permission-gated.
- **health** reports API/DB readiness and authenticator reachability.

## Request Path

A typical authenticated item request flows:

1. HTTP/Connect request arrives at the Go server (`api/internal/server/`).
2. **Auth middleware** (`api/internal/middleware/`, backed by
   `api/internal/auth/`) validates the access token via the
   `AuthClient` seam against `scenario-authenticator`. On failure it
   rejects (fail-closed); on success it attaches owner identity.
3. The **devices** layer confirms the calling device is trusted; an
   untrusted or revoked device is rejected here.
4. The owning **domain service** runs business rules and persists
   through the storage `Resolver` (SQLite metadata) and `BlobStore`
   (bytes).
5. **realtime** is notified to fan out events to other online devices
   over the WebSocket.
6. A proto-typed response (or stream) returns to the caller.

The WebSocket upgrade path performs the same auth + trust check at
connection time, then registers the device in the presence registry.

## System Boundaries

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas under
  `packages/proto/schemas/device-sync-hub/`.

The scenario does not own:

- shared package implementation under `packages/` (including
  `api-core/storage` and `api-core/blobstore`),
- Vrooli resource implementation,
- `scenario-authenticator` (consumed over HTTP — see
  [`INTEGRATIONS.md`](INTEGRATIONS.md)),
- generated proto outputs under `packages/proto/gen/`.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/device-sync-hub/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/device-sync-hub/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/device-sync-hub/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/device-sync-hub/v1/...   (ui)
       └──▶ packages/proto/gen/python/device_sync_hub/v1/...          (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data` — the transfer upload endpoint, where bytes stream straight to the blobstore. |
| `RESTReasonWebhookReceiver` | Endpoint shape is dictated by a third-party system we do not own. |
| `RESTReasonThirdPartyShape` | Request or response is an externally-defined contract (e.g. OAuth callbacks). |
| `RESTReasonOpsProbe` | Lifecycle systems, load balancers, and `curl` must reach the endpoint without a generated client (plain `GET /health`). |

The WebSocket gateway (`api/internal/realtime/`) is a long-lived
upgrade, not a Connect procedure; its event payloads stay proto-typed
even though the transport is a socket. Streaming downloads preserve the
original filename and stream bytes from the blobstore rather than
buffering whole files.

Mechanical enforcement: `cmd/gen-endpoints` rejects any
`EndpointDescriptor.Path` that is not a generated Connect procedure
constant unless the descriptor carries a `RESTException` with one of
the four reasons. A REST endpoint without that tag fails `make
endpoints`, which fails `make test`, which fails CI. The fix is either
to author a proto service method (preferred) or to tag the exception
explicitly.

Even for REST exceptions, the **payload shape** stays proto-typed
wherever possible: the transfer upload handler accepts multipart bytes
but returns the proto item-metadata message, so API/UI/CLI never drift.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP+WS server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/middleware/` | Auth middleware that attaches request-scoped owner identity. | Cross-cutting transport edge wired once. | All authenticated handlers. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting; pairing TTL and retention need it test-substitutable. | Pairing, retention, presence. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains. | API tests. |
| `ui/src/test-utils/` | Cross-feature render, a11y, and model-test helpers. | Used by unrelated UI features. | UI tests. |

`api-core/storage` (the `data` storage class) and `api-core/blobstore`
are consumed as shared packages, not re-implemented here. See
[`DATA.md`](DATA.md) for the storage class and path policy.

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/device-sync-hub/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

Two seams must be preserved as the scenario grows:

- the **transfer byte-flow seam** (the "where bytes move" boundary) is
  isolated so a P2 WebRTC same-LAN fast path can slot in without
  redesign; the relay path stays the always-available fallback;
- the **`AuthClient` seam** is the single point of contact with
  `scenario-authenticator`; no domain talks to the authenticator
  directly.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md).
For persistence and retention, update [`DATA.md`](DATA.md). For
temporal behavior, update [`FLOWS.md`](FLOWS.md).

## Architecture Maturity

This scenario is in Phase 1 (design/charter). The documents describe the
intended design; no product code exists yet.

| Area | Maturity | Evidence | Remaining Drift |
|---|---|---|---|
| API | Planned | Domain map, request path, and seams designed in these docs. | All domain code (auth, devices, transfer, realtime, settings) is unimplemented. |
| UI | Planned | Split-screen Send/Receive layout specified in the PRD. | No feature folders yet. |
| CLI | Planned | send/receive/devices/revoke command surface specified. | No command groups yet. |
| Docs | Contract-ready | Concept docs and `docs/manifest.json` describe the intended system. | Implementation stubs must be filled as domains land. |

Use `docs/manifest.json` as the documentation contract. The declared
`maturity` values are expected to be maintained by agents and later
grounded by Knowledge Observatory validation.

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-06-17 | API process hosts a persistent WebSocket gateway alongside Connect/REST. | Real-time delivery and presence (OT-P0-005) require a push channel; co-locating avoids separate notification infrastructure. | Multi-instance scale-out (Redis-backed presence) or a dedicated gateway. |
| 2026-06-17 | Owner identity/JWT/session delegated to `scenario-authenticator` over HTTP, fail-closed. | This scenario owns devices/trust, not identity; reusing the authenticator avoids reimplementing auth. | Authenticator unavailability patterns or a need for offline trust. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, and migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, and external services | `docs/concepts/INTEGRATIONS.md` |
| Monetization and packaging | `docs/business/MONETIZATION.md` |
| Go-to-market strategy | `docs/business/GO-TO-MARKET.md` |
| Deployment tiers and readiness | `docs/operations/DEPLOYMENT.md` |
| Operator procedures | `docs/operations/RUNBOOK.md` |
| Telemetry, metrics, and alerts | `docs/operations/OBSERVABILITY.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |
| Known drift and deferred work | `docs/internal/PROBLEMS.md` |
| Change history | `docs/internal/PROGRESS.md` |

Every durable scenario document should be registered in
`docs/manifest.json`. Put deep domain-specific documentation under
`docs/domains/<domain>/` when `DOMAINS.md` would become noisy.

## Cross-References

- [`START-HERE.md`](../START-HERE.md) — first implementation workflow
- [`QUICKSTART.md`](../QUICKSTART.md) — clone-to-running flow
- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts (scenario-authenticator, redis)
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — commercial story
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — error semantics
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues / tech debt
- [`../internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
