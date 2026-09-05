# Decisions — Scenario Authenticator

This document records durable decisions and tradeoffs future agents
should not accidentally relitigate.

## Purpose Of This Document

Use this document when a choice:

- affects multiple files or future agents,
- rejects a plausible alternative,
- changes architecture, deployment, data, security, monetization, or
  testing direction,
- needs a revisit trigger.

Routine implementation log entries belong in [`PROGRESS.md`](PROGRESS.md).
Known unresolved issues belong in [`PROBLEMS.md`](PROBLEMS.md).

## Decision Log

| Date | Decision | Context | Consequences | Revisit Trigger |
|---|---|---|---|---|
| 2026-06-18 | **Rewrite from the `react-vite` template rather than migrate the old scenario in place.** | The old scenario-authenticator predated current standards (shared Postgres, REST-only, no proto/Connect, sprawling cruft incl. a committed 8.9 MB binary). It could not boot here. | A clean, standards-current foundation; the old scenario is preserved read-only at `/tmp/scenario-authenticator-OLD-reference` as a porting reference. | If migrating real account data ever becomes necessary (it is not — local fleet, regenerable data). |
| 2026-06-18 | **Move persistence to SQLite via the `api-core/storage` seam; remove shared Postgres.** | The old scenario shared a Postgres instance and the `vrooli` role with sibling scenarios; reconciling its DB password took down prompt-manager — a fleet-wide blast radius unacceptable for foundational auth. | Per-scenario SQLite eliminates the shared-DB failure class. The seam keeps the DB swappable to a managed server DB for cloud scale, so SQLite is a default, not a lock-in. | If a single deployment needs write throughput beyond SQLite's single-writer ceiling → swap the seam to a managed DB (OT-P2-006). |
| 2026-06-18 | **Keep Redis as a required resource for hot state.** | Redis backs sessions, token-family revocation, OAuth CSRF state, and distributed rate limiting; these are security controls, not a cache. | Redis is `required: true` in `.vrooli/service.json`. It is also the horizontal-scale enabler (shared session/rate-limit state across replicas behind a load balancer). | If a deployment must run with zero extra infra, evaluate an in-process fallback for sessions/rate-limit (the old rate limiter already had a memory primary). |
| 2026-06-18 | **Model the system as an Identity Provider (IdP) with Relying Parties (RPs) that verify tokens locally.** | Adopting scenarios need auth without coupling. The OIDC/Keycloak split makes "who you are" (IdP) vs "what you may do here" (RP) crisp. | RPs verify RS256 tokens locally against JWKS and never call back per request — this is both the API-to-API contract and the scale lever (the hot path never touches the authenticator). The authenticator owns authn + coarse realm roles/scopes; the RP owns fine-grained domain authorization. | If RPs need to delegate fine-grained authorization → a delegated policy engine (OT-P2-004). |
| 2026-06-18 | **API-to-API only; no cross-origin browser calls, ever.** | A browser must talk only to its own scenario's API; that API forwards same-origin to the authenticator (resolved by slug via `api-core/discovery`). | This is the device-sync-hub pattern. No CORS surface for inter-scenario auth; no port-hunting. | Never — this is a hard architectural rule for the fleet. |
| 2026-06-18 | **The Realm is the multi-tenant primitive; even the single default realm issues `aud`-scoped tokens.** | "Reusable as a dependency + multi-tenant + scalable" all fall out of one tenant boundary. Promotes the old `applications` registry into a real isolation boundary. | One default realm locally (feels single-tenant, generalizes for free); a realm per customer/product for hosted SaaS. A token minted for realm A is rejected by realm B's verifier — cross-realm acceptance is forbidden and must be tested. | If a grouping layer above the user is needed → groups/orgs (OT-P2-003). |
| 2026-06-18 | **Keep it fully-featured; tier the roadmap P0/P1/P2 rather than trimming scope.** | The scenario has big ambitions (all types of auth for local + hosted/SaaS). Right-sizing down was explicitly rejected. | All capability (MFA, OAuth, SAML, API keys, OIDC-provider mode, SCIM, etc.) stays in scope but is sequenced; only what device-sync-hub + local needs now is P0. | Revisit tiering if a downstream consumer needs a P1/P2 capability sooner. |
| 2026-06-18 | **Treat the rewrite as scaffold-regen + verbatim crypto-port, not a clean-room reimplementation.** | Auth is the highest-stakes boundary in the fleet; re-deriving crypto risks a subtle, fleet-wide bug. | RS256/JWKS/claims/issuer and the load-or-generate persisted keypair are ported verbatim so the live consumer device-sync-hub verifies unchanged. | Never re-derive the auth core from scratch; port the proven primitives. |
| 2026-06-18 | **Proto/Connect-RPC is the primary contract; REST is only for non-RPC web standards.** | Typed, drift-proof contracts matching the fleet; the old scenario was REST-only. | The only REST endpoints are `/.well-known/jwks.json`, OAuth/OIDC callbacks, and (P2) SAML ACS. device-sync-hub's forwarder migrates from REST to the typed Connect client in lockstep. | If a standard requires a non-Connect HTTP shape, add it to the REST edge explicitly. |
| 2026-06-18 | **Hash passwords with Argon2id (replacing the old bcrypt).** | Argon2id is the modern default; the rewrite is the moment to adopt it. | New password records use Argon2id at a documented cost. Only hashes + signed material are stored at rest. | If a credible reason to change KDF arises; never store plaintext. |
| 2026-06-18 | Use the generated `react-vite` scenario documentation contract. | Scenario scaffold was generated from the template. | Docs start with stubs and maturity metadata in `docs/manifest.json`. | Revisit when scenario adopts a different template or doc contract. |

## Key Decisions In Depth

**Why SQLite-per-scenario is the keystone operational decision.** The single
most important change in this rewrite is *not* a feature — it is removing the
shared-Postgres blast radius. An auth service is the most foundational thing in
the fleet; it must never be able to take down a sibling because someone
reconciled a database password. Per-scenario SQLite (via the `api-core/storage`
seam) deletes that entire failure class. The seam, not the database choice, is
what preserves cloud-scale options. See [`../concepts/DATA.md`](../concepts/DATA.md).

**Why stateless local verification matters for scale.** The high-frequency path
(verifying a token on every RP request) is stateless and local — RPs fetch JWKS
once and verify in-process. The authenticator is only hit on login/refresh/revoke,
which is low-frequency. This is why "modest scale" is achievable on SQLite: the
write-bound store is never on the hot path. See
[`PERFORMANCE.md`](PERFORMANCE.md).

## Superseded Decisions

| Date | Superseded Decision | Replacement | Details |
|---|---|---|---|
| 2026-06-18 | Old scenario: shared Postgres + Redis, REST-only, bcrypt, no realms. | This rewrite: SQLite-via-seam, Connect-primary, Argon2id, realm-scoped. | The old scenario remains at `/tmp/scenario-authenticator-OLD-reference` for porting reference only. |

## Cross-References

- [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md) — system decisions
- [`../../PRD.md`](../../PRD.md) — operational targets + Appendix (IdP/RP, realms, invariants)
- [`PROBLEMS.md`](PROBLEMS.md) — unresolved drift and debt
- [`PROGRESS.md`](PROGRESS.md) — completed work history
