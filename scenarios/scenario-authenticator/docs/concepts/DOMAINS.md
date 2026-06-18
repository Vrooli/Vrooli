# Domains — Scenario Authenticator

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

scenario-authenticator is an **Identity Provider (IdP)**. Its product is
*who you are*: accounts, credentials, token issuance and signing,
sessions, MFA, federation, and API keys — consumed API-to-API by other
scenarios (Relying Parties) that verify its tokens locally against its
JWKS. The bounded contexts below are the data-and-capability boundaries
that realize the PRD operational targets. The admin console, end-user
self-service, and hosted login/consent screens are **UI surfaces** that
compose these domains; they are not themselves domains.

> The scaffold still ships the fenced `notes` worked example and the real
> `health` domain. The auth domains below are the **target** map authored
> during orientation (documentation-first); they are not yet implemented.
> `vrooli scenario detemplate scenario-authenticator` removes the fenced
> example once the first real domain is green (Gate 7 — not yet reached).

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md). The IdP↔RP responsibility split and the
realm tenant model are in [`../../PRD.md`](../../PRD.md) Appendix A/B.

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Tier |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Scaffold health. | shipped |
| realms | The tenant boundary. Isolated identity namespaces with per-realm policy, branding, token TTLs, redirect URIs, and `aud` scope. | Entity / policy | Realm records, per-realm config. | API, CLI, UI | OT-P0-008, OT-P1-001 | P0 (default realm) → P1 (multi-realm) |
| identity | Accounts and credentials. Realm-scoped principals, Argon2id password hashes, email-verification state. | CRUD / entity | Users, credentials, verification state. | API, CLI, UI | OT-P0-001, OT-P0-004, OT-P1-009 | P0 |
| tokens | Token issuance + RS256 signing + JWKS publication + refresh-token families with rotation and reuse detection. Owns the signing keypair. | Crypto / lifecycle | Signing keypair, refresh-token families, JWKS. | API (Connect + JWKS REST), CLI | OT-P0-002, OT-P0-003 | P0 |
| sessions | Server-tracked sessions and revocation ("log out everywhere"). | Lifecycle / cache | Session records (Redis hot state). | API, CLI, UI | OT-P0-005 | P0 |
| authorization | Role and scope *definitions* per realm and their assignment; emitted as token claims. Enforcement is delegated to RPs. | Policy / reference | Roles, scopes, assignments. | API, CLI, UI | OT-P0-009, OT-P1-005 | P0 (admin/user) → P1 (scopes) |
| audit | Append-only log of security-relevant auth events. | Reporting / ledger | Audit events. | API, CLI, UI | OT-P0-007 | P0 |
| mfa | Second factors: TOTP enrollment/challenge/recovery codes; WebAuthn passkeys. | Credential / lifecycle | TOTP secrets, recovery codes, passkey credentials. | API, CLI, UI | OT-P1-002, OT-P1-006 | P1 |
| federation | Inbound external identity: OAuth2/OIDC social providers (P1) and SAML (P2), with account linking and CSRF state. | Integration / adapter | Linked external identities, provider config, CSRF state. | API (+ REST callbacks), UI | OT-P1-003, OT-P2-001, OT-P2-002 | P1–P2 |
| apikeys | Non-human principals: hashed API keys and the client-credentials grant. | Credential / lifecycle | Hashed API keys, client records. | API, CLI, UI | OT-P1-004 | P1 |

## Domain Details

### health (shipped)

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Owns: health response construction and dependency status mapping.
- Source: `api/handlers/health/`, `ui/src/features/health/`.

### realms (P0 default realm → P1 multi-realm)

- Purpose: the tenant boundary. A realm is an isolated identity namespace
  with its own user pool (the same email is distinct principals across
  realms), branding, password policy, token TTLs, enabled methods,
  allowed redirect URIs, and an `aud`-scoped token audience.
- Why it exists from day one: even a single-realm local deployment issues
  `aud`-scoped tokens and rejects cross-realm tokens, so multi-tenancy is
  a configuration change later, not a re-architecture (OT-P0-008).
- Owns: realm records and per-realm policy/branding/config.
- Does not own: the users inside a realm (that is `identity`, realm-scoped).
- Surfaces: Connect API (realm CRUD), CLI, admin-console UI.
- Storage: SQLite via the `api-core/storage` seam.
- Related: PRD Appendix B (the Realm primitive).

### identity (P0)

- Purpose: accounts and credentials. Register/sign-in, Argon2id password
  hashing, email verification, password reset/recovery (OT-P1-009).
- Owns: user records (realm-scoped), credential hashes, verification state.
- Does not own: token minting (that is `tokens`), sessions, MFA secrets.
- Boundary: stores only hashes and signed material — never plaintext.
- Surfaces: Connect API (Register, Login, password/recovery), CLI,
  self-service + hosted login UI.

### tokens (P0)

- Purpose: issue RS256-signed access tokens, publish JWKS so RPs verify
  locally, and manage rotating refresh-token families with reuse
  detection (presenting a rotated refresh token revokes the family).
- Owns: the load-or-generate signing keypair (persisted to the storage
  root), refresh-token family records, and the JWKS document.
- Claims contract (carried over, do not break): `user_id`/`sub`, `email`,
  `roles`, `iss: scenario-authenticator`, `aud` (realm). Algorithm locked
  to exactly RS256 (reject `none`/HS confusion).
- Surfaces: Connect API (Refresh, Validate), REST `/.well-known/jwks.json`,
  CLI.
- Related: PRD Appendix C (carried-over invariants), `../internal/SECURITY.md`.

### sessions (P0)

- Purpose: server-tracked sessions, list + per-session revoke, and "log
  out everywhere"; the live `/api/v1/sessions/{id}` revoke contract
  device-sync-hub calls is preserved (or delivered as the Connect
  equivalent in lockstep).
- Owns: session records, kept hot in Redis.
- Surfaces: Connect API + the carried-over REST revoke, CLI, self-service UI.

### authorization (P0 baseline → P1 scopes)

- Purpose: define realm-level roles and scopes and assign them; emit them
  as token claims. Enforcement of fine-grained "can-they" stays with the
  Relying Party (a delegated policy engine is P2, OT-P2-004).
- Owns: role/scope definitions and assignments.
- Surfaces: Connect API, CLI, admin-console UI.

### audit (P0)

- Purpose: append-only record of security-relevant events (sign-in,
  sign-out, token-family revoke, MFA changes, admin actions) for review.
- Owns: audit events; queryable per realm.
- Surfaces: Connect API (query), CLI, admin-console UI.

### mfa (P1)

- Purpose: second factors — TOTP enrollment/challenge/recovery codes and
  WebAuthn passkeys (first-class credential + second factor).
- Owns: TOTP secrets, recovery codes, passkey credentials.
- Surfaces: Connect API, CLI, self-service UI (enrollment + challenge).

### federation (P1 social → P2 SAML/OIDC-provider)

- Purpose: inbound external identity. OAuth2/OIDC social providers
  (Google, GitHub, Microsoft) with account linking at P1; SAML and
  OIDC-provider mode ("Login with Vrooli") at P2.
- Owns: linked external identities, provider configuration, OAuth CSRF
  state (Redis).
- Surfaces: Connect API, REST OAuth/OIDC callbacks (a non-RPC web
  standard), UI consent.

### apikeys (P1)

- Purpose: non-human principals — hashed API keys and a client-credentials
  grant so machine/service callers authenticate without a human session.
- Owns: hashed API keys and client records.
- Surfaces: Connect API, CLI, admin/self-service UI.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Realm | The tenant boundary; isolated identity namespace with `aud` scope. | `realms` domain; PRD Appendix B. |
| Relying Party (RP) | An adopting scenario that verifies tokens locally and owns its own domain authorization. | PRD Appendix A; `INTEGRATIONS.md`. |
| Principal | A user or machine identity inside a realm. | `identity` / `apikeys`. |
| Claim | A statement in a verified token (`sub`, `aud`, `roles`, `scopes`). | `tokens`; consumed by RPs. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| groups / organizations | A grouping layer above the user with inherited roles. | OT-P2-003 when org hierarchy is needed. |
| policy-engine | Delegated fine-grained authorization RPs can call. | OT-P2-004 when an RP needs to delegate "can-they". |
| provisioning (SCIM) | Enterprise directory sync. | OT-P2-007 when an enterprise realm requires SCIM. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database/storage-seam infra.
- `api/internal/crypto/` — RS256/JWKS/Argon2id primitives shared by
  `tokens` and `identity` (a shared library, not a product capability).
- `ui/src/components/` — shared presentation primitives.

If one of these starts using product vocabulary, split the product piece
into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency + Relying-Party contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — security posture
- [`../../PRD.md`](../../PRD.md) — operational targets + Appendix (IdP/RP, realms, invariants)
