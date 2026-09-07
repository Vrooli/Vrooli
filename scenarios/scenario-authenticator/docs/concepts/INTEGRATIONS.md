# Integrations — Scenario Authenticator

This document is the canonical dependency contract for resources,
other scenarios, and third-party services used by the scenario.

## Purpose Of This Document

Use this document to answer:

- What does the scenario depend on?
- Which dependencies are required versus optional?
- Which domain uses each dependency?
- What is the failure or degradation behavior?
- Where is the dependency declared or configured?

> **Status: shipped foundation.** SQLite, lifecycle wiring, account
> authentication, RS256/JWKS, sessions, refresh-token rotation, rate limiting,
> and audit integration are implemented. The remaining rows marked planned or
> deferred are future product capabilities, not current runtime dependencies.

scenario-authenticator is a **foundational, bottom-of-the-stack
capability with no upstream scenario dependencies of its own.** Its
relationships are: required *resources* (SQLite via the storage seam,
Redis), and *downstream consumers* — adopting scenarios acting as
Relying Parties (RPs). Everything is **API-to-API; there are no
cross-origin browser calls anywhere in the model.**

## Dependency Inventory

| Dependency | Type | Required? | Used By | Contract | Failure Behavior |
|---|---|---|---|---|---|
| SQLite (via `api-core/storage` seam) | embedded storage | yes | API, all persistence-backed domains | resolved by `api-core/storage` from the scenario id; seam keeps it swappable to a managed DB at scale | API reports unhealthy if unreachable. |
| Redis | hot-state store | tier-dependent | sessions, tokens (revocation), federation (OAuth CSRF), rate limiting | resource declared in `.vrooli/service.json`; durable local substitute is valid for one replica | A single local replica may use durable local hot state. Multi-replica operation requires shared Redis so revocation and rate limits do not diverge. |
| Signing keypair (`private.pem`/`public.pem`) | persisted secret | yes | tokens (RS256 sign + JWKS) | load-or-generate in the storage root (carried over verbatim) | Regenerating it invalidates every live token — persistence is deliberate. |
| Vrooli lifecycle | local platform | yes | API, UI, CLI | `.vrooli/service.json`, Makefile targets | Scenario must be started through lifecycle commands. |
| `api-core/discovery` | platform service-resolution | yes (for consumers) | RPs resolving this scenario | RPs resolve **by slug** `scenario-authenticator` — no hardcoded URL/port | A consumer that cannot resolve the slug fails closed. |

## Vrooli Resources

The scenario declares Redis as an enabled hot-state option. A single local or
desktop replica may use durable local state behind the same store seam. Redis
is required when more than one replica must share revocation, refresh-family,
or rate-limit state. It does **not** use shared Postgres — moving off the
shared database is the reason for the rewrite; persistence is SQLite via the
`api-core/storage` seam.

| Resource | Status | Reason | Revisit Trigger |
|---|---|---|---|
| Redis | optional for one replica; required above one replica | Sessions, token/family revocation, OAuth CSRF state, and distributed rate-limit coordination. | Only when a shared hot-state implementation provides the same correctness guarantees. |
| Shared Postgres | **deliberately removed** | The shared DB created a fleet-wide blast radius; replaced by per-scenario SQLite via the storage seam. | Managed-DB backing for HA is a P2 ambition through the same seam (OT-P2-006), not a return to a shared DB. |

## Scenario Dependencies

**Upstream: none.** scenario-authenticator is foundational and calls no
other scenario. The relationships below are **downstream consumers**
(Relying Parties) that depend on this scenario.

| Scenario | Direction | Status | Reason / Contract |
|---|---|---|---|
| device-sync-hub | downstream consumer (live) | first migrated RP | The reference integration: resolves this scenario by slug via `api-core/discovery`, forwards sign-in same-origin (`internal/identity.Forwarder`), and verifies tokens locally against JWKS (RS256-locked, cached). Its forwarder migrates from REST to the typed Connect client in lockstep with P0 (OT-P0-012). |
| landing pages | downstream consumer (future) | planned | Reuse this auth instead of rolling their own. |
| landing-page-business-suite (LPBS) | downstream consumer (migration target) | compatibility/current local auth, shared identity planned | LPBS owns customer accounts, subscriptions, downloads, and entitlements. Its identity migration must use explicit account linking; it must not copy website sessions into local apps. |
| hosted SaaS products | downstream consumer (future) | planned | Provision a realm per customer (B2B) or product (B2C). |

## How A Relying Party Integrates

The contract an adopting scenario implements (PRD Appendix A):

1. **Resolve by slug.** Resolve scenario-authenticator's API base URL at
   runtime via `api-core/discovery` using the slug
   `scenario-authenticator` — never a hardcoded URL, port, or env var.
2. **Forward same-origin, never cross-origin.** The RP's browser talks
   only to the RP's own API. For interactive sign-in/register, the RP
   API forwards same-origin to the authenticator and relays the token
   back (the device-sync-hub `internal/identity.Forwarder` pattern).
   **No browser ever calls the authenticator directly.**
3. **Verify locally against JWKS.** Fetch the public key once from
   `/.well-known/jwks.json`, cache it, and verify the RS256 signature
   **offline** on every request — no per-request callback. Lock the
   algorithm to RS256 (reject `none` and HS-family confusion).
4. **Check `aud` and trust the claims.** Reject a token whose `aud` does
   not match the verifying realm. Trust `sub`/`user_id`, `roles`, and
   `scopes`; map the verified identity to the RP's own domain objects.
5. **Own your authorization.** The authenticator answers "valid
   principal + coarse realm roles/scopes"; the RP answers fine-grained
   "can this principal do this action on this resource." The RP never
   sees a password and never calls back to authorize.

### Product and desktop integrations

LPBS website sign-in is authentication for the LPBS deployment and
authorization for LPBS administration, downloads, accounts, and entitlements.
It is not automatically a local Vrooli identity. A future migration may make
LPBS an RP of this provider while leaving commerce and entitlement ownership in
LPBS.

Tier 2 bundles declare one of the project authentication modes:
`personal_local`, `local_multi_user`, `remote_vrooli`, or `shared_provider`.
Only the explicit multi-user or remote modes require a human authenticator
session. A bundled personal app may operate without a sign-in while still
using the desktop supervisor's private loopback token.

### Frozen wire invariants (shared cross-scenario contract)

These three values are a hard contract every relying party depends on;
changing any one silently breaks every RP. The default realm at P0:

| Claim / value | Frozen value |
|---|---|
| Issuer (`iss`) | `scenario-authenticator` |
| Primary identity claim | `user_id` (NOT `sub`; `sub` is mirrored additively) |
| Default-realm audience (`aud`) | `scenario-authenticator:default` during compatibility migration; resource-specific audiences are the target |
| JWKS path | `/.well-known/jwks.json` |
| Token header | `{alg:"RS256", kid:<fingerprint>, typ:"JWT"}` |

The current audience constant **`scenario-authenticator:default`** is the
realm-qualified compatibility audience both the authenticator and current RPs
agree on. It is not the final resource audience. New integrations must record
their target resource audience and support the migration policy before removing
the compatibility verifier.

## Third-Party Services

| Service | Status | Reason | Contract |
|---|---|---|---|
| OAuth2/OIDC providers (Google, GitHub, Microsoft) | planned (P1) | Inbound social sign-in with account linking (OT-P1-003). | REST OAuth/OIDC redirect callbacks (a non-RPC web standard), CSRF-protected. |
| SAML 2.0 IdPs | planned (P2) | Enterprise SSO for realms that require it (OT-P2-001). | REST SAML ACS (externally-defined POST binding). |

## Failure Modes

| Dependency | Failure Signal | Expected Behavior | Tests (target) |
|---|---|---|---|
| SQLite (storage seam) | `PingContext` error | `/health` reports the dependency unhealthy. | health handler tests |
| Redis | connection error | Session revoke + distributed rate-limit degrade; surfaced as unhealthy; do not silently accept stale sessions. | session/rate-limit integration tests |
| Signing keypair | missing/unreadable PEM | Load-or-generate; a regenerated key invalidates live tokens (deliberate, must be deployed knowingly). | tokens/crypto tests |
| `api-core/discovery` (consumer side) | slug unresolvable | RP fails closed (treats request as unauthenticated). | RP-side tests (e.g. device-sync-hub `auth`) |

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system boundaries, IdP/RP role, REST edge
- [`DATA.md`](DATA.md) — storage ownership (SQLite seam + Redis hot state)
- [`FLOWS.md`](FLOWS.md) — the JWKS local-verify / same-origin forward flow
- [`../../PRD.md`](../../PRD.md) — Appendix A (IdP↔RP), D (ecosystem-fit)
- [`../reference/configuration.md`](../reference/configuration.md) — environment and service manifest
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
