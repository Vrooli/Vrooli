# Integrations — Device Sync Hub

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario. The
load-bearing dependency is `scenario-authenticator`; everything else is
local infrastructure or optional.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| scenario-authenticator | scenario | yes | auth, devices | HTTP `/api/v1/auth/*` + `/api/v1/sessions/*` | **Fail closed** in production — requests rejected if unreachable beyond the cache window. |
| SQLite (api-core/storage) | embedded storage | yes | devices, transfer, settings | `SQLITE_PATH` lifecycle env var, `data` storage class | API reports unhealthy if unreachable. |
| api-core/blobstore | embedded storage | yes | transfer | BlobStore seam under the `data` class | Upload/download fail; API health degrades. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| Redis | resource | optional | realtime | presence registry + pairing-code TTL cache | **Graceful degradation** to in-memory; single-instance only. |

## Vrooli Resources

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Redis | optional | Backs cross-instance WebSocket presence and a short-TTL pairing-code validation cache. Without it, presence and the cache run in-memory (single-instance). | Multi-instance scale-out, where in-memory presence cannot span processes. |

When Redis is absent, `realtime` keeps presence in a process-local
registry and `devices` validates pairing codes directly against SQLite.
The scenario is fully functional single-instance without Redis; adding
it is a scale-out lever, not a correctness requirement.

## Scenario Dependencies

### scenario-authenticator

`device-sync-hub` delegates **all** owner identity, JWT issuance,
token validation, and session/device revocation to
`scenario-authenticator` over HTTP. This scenario does **not**
reimplement or extend authenticator internals; it owns devices, pairing,
trust, presence, and per-item ACL only. All contact goes through the
single `AuthClient` seam (`api/internal/auth/`); no other domain calls
the authenticator directly.

Consumed surface (migrated to the authenticator's typed **Connect-RPC**
contract in lockstep with its P0 rewrite — the old REST edge was retired):

| Surface | Purpose | Consumed By |
|---|---|---|
| `AccountsService.Login` (Connect) | Owner login → access + refresh tokens. | identity forwarder (CLI/UI login) |
| `AccountsService.Register` (Connect) | Create the owner account → tokens. | identity forwarder |
| `GET /.well-known/jwks.json` (REST) | Fetch the RS256 public key once; verify every owner token **locally**. | auth (`Client`) |
| `SessionsService.RevokeSession` (Connect) | Invalidate one device's session on revocation. | devices (revoke) |

Verification is **local**, not a per-request callback: the hub fetches
the JWKS once, caches it, and verifies each owner JWT's signature, `iss`,
`exp`/`nbf`, and `aud` in-process. The authenticator is contacted only on
login/register (forwarder) and on session revoke — never per request — so
a momentarily-unavailable authenticator never breaks an issued session.

**Frozen wire invariants** (the shared cross-scenario contract — see the
authenticator's [INTEGRATIONS](../../../scenario-authenticator/docs/concepts/INTEGRATIONS.md)):
`iss == "scenario-authenticator"`, the identity claim is `user_id`, the
JWKS path is `/.well-known/jwks.json`, and the default-realm audience is
the single shared constant **`scenario-authenticator:default`**, pinned
in code as `auth.AuthExpectedAudience`. A token whose `aud` is any other
realm is rejected even if its signature and issuer are valid (tenant
isolation).

Revocation contract: revoking a device calls
`SessionsService.RevokeSession` (idempotent — a missing session succeeds)
to kill that device's authenticator session **and** drops the device from
the trust group. A partial failure must leave the device locked out, never
half-trusted (see the revocation flow in [`FLOWS.md`](FLOWS.md)).

Fail-closed contract: an invalid/expired token, a non-RS256 `alg`, a wrong
`aud`/`iss`, or an unobtainable signing key all yield no identity, so
owner-gated RPCs reject. There is no shipped test-mode bypass. See
[`DOMAINS.md`](DOMAINS.md) (`auth`) and the request path in
[`ARCHITECTURE.md`](ARCHITECTURE.md).

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| TLS termination | required (deployment) | TLS in transit is mandatory; provided by a Cloudflare tunnel or reverse proxy in front of the Go server. | Operator-provided; see [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md). |
| Cloud storage (S3/GCS/etc.) | not-applicable | All bytes stay on the owner-operated server via `api-core/blobstore`. | Out of scope by design. |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests |
|---|---|---|---|
| scenario-authenticator | validate/login HTTP error or timeout | Gated requests rejected (fail closed); `/health` reports authenticator unreachable. | auth middleware + health tests |
| SQLite | `PingContext` error | `/health` returns unhealthy dependency status. | health handler tests |
| api-core/blobstore | read/write error | Upload/download fail with typed errors; health degrades. | transfer handler tests |
| Redis (optional) | connection error | Fall back to in-memory presence and cache; log + continue. | realtime degradation tests |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries and the AuthClient seam
- [`DOMAINS.md`](DOMAINS.md) — the `auth` and `realtime` domains
- [`DATA.md`](DATA.md) — storage ownership and blobstore
- [`FLOWS.md`](FLOWS.md) — revocation and presence flows
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — TLS and deployment readiness
