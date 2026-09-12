# Data — Scenario Authenticator

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is the retention/deletion story?
- How are schema changes handled?

> **Status: implemented foundation.** The account, refresh-token, session,
> realm, rate-limit, MFA, and audit entities described here are persisted by the
> current API. Future entities are marked planned or deferred in the
> requirements registry; this document does not claim that the `notes`
> template `notes` example exists.

## Storage Overview

Persistence is **SQLite through the `api-core/storage` seam** — not
shared Postgres. Moving off the shared database is the reason for the
rewrite: a fleet-wide shared DB was a fleet-wide blast radius. The seam
keeps the store swappable to a managed DB for cloud scale, so SQLite is
the default, not a lock-in. The path is resolved from the scenario id by
`api-core/storage`, and the API applies domain schemas on startup through
the seam.

Two substrates back the data model, by access pattern:

- **SQLite (durable)** holds everything that must survive a restart in the
  current implementation: realms, users, credential hashes, role/scope
  definitions, audit events, and MFA secrets. Refresh-token families and
  sessions use the configured hot-state store. Federation links and API keys
  remain future domains, not current storage obligations. **Only hashes and
  signed material are stored at rest — never plaintext secrets.**
- **Configured hot-state store** backs sessions, token/family revocation
  lookups, OAuth CSRF state, and rate-limit coordination. A durable local
  implementation is valid for one replica. Shared Redis or an equivalent
  store is required when state must be coordinated across replicas (see
  [`INTEGRATIONS.md`](INTEGRATIONS.md)).
- The **signing keypair** (`private.pem` / `public.pem`) is persisted to
  the **storage root** under a load-or-generate pattern (carried over
  verbatim, PRD Appendix C), not in the database. Losing or regenerating
  it invalidates every live token, so it is treated as durable state.

Note the hot path — RS256 token verification by RPs against the JWKS —
**never touches SQLite**, so the single-writer store is not the fleet's
throughput ceiling (see [`ARCHITECTURE.md`](ARCHITECTURE.md)).

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
dependency reachability. The table below is the current ownership map; rows
for deferred federation, API keys, and future tenancy remain explicitly
marked planned or deferred.

| Data | Owning Domain | Store | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Realms + accounts + machine bindings | accounts | SQLite | `api/internal/accounts/schema.sql` | Until deletion of the owning record. | The default realm is seeded at boot; accounts and bindings are realm-scoped. |
| Credentials (Argon2id hashes) + verification state | accounts | SQLite | `api/internal/accounts/schema.sql` | Lifecycle of the owning account. | **Argon2id hash only**, documented cost; no plaintext ever. |
| Signing keypair (`private.pem` / `public.pem`) | tokens | Filesystem (storage root) | `api-core/storage` root, load-or-generate | Durable; rotation is deliberate (P2 OT-P2-005). | Private key never leaves the host; public key is published as JWKS. |
| JWKS document | tokens | Derived (from public key) | Built from the keypair at boot | Re-derived on start. | Served at `/.well-known/jwks.json` for RP local verification. |
| Refresh-token families and sessions | sessions | Configured hot-state store | `api/internal/sessions/manager.go` | TTL-bound; cleared on revoke or logout everywhere. | Rotation, reuse detection, blacklist, and session indexes share one store seam. |
| Scopes and assignments | authorization | SQLite | `api/internal/authorization/schema.sql` | Until removed. | The authenticator emits opaque scopes; the RP enforces them. |
| Audit events | audit | SQLite | `api/internal/audit/schema.sql` | Append-only; product-defined retention window. | Security-relevant events, queryable per realm. |
| MFA enrollments, challenges, and recovery codes | mfa | SQLite plus credential custodian | `api/internal/mfa/store.go` | Until enrollment removal or account deletion. | TOTP is shipped; passkeys remain deferred. |
| Federation links and API keys | future domains | Not implemented | No schema yet | Deferred | Add storage only when the owning domain is scheduled. |

## Schema Map

Each domain's schema file or schema provider lives beside the code that
interprets it. The system schema and durable hot-state schema are the
cross-cutting infrastructure schemas.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| realms + accounts + machine bindings | accounts | `api/internal/accounts/schema.sql` | accounts repository/service/handlers |
| refresh-token families, sessions, and blacklist | sessions | `api/internal/sessions/manager.go` + hot-state store | sessions service/handlers; token hashing helpers |
| `private.pem` / `public.pem` | tokens | storage root (load-or-generate) | crypto lib (sign + JWKS publication) |
| durable hot-state tables / Redis keys | infrastructure | `api/internal/redisstate/sqlite.go` or Redis keyspace | sessions, refresh, revocation, and rate-limit services |
| roles / scopes / assignments tables | authorization | `api/internal/authorization/schema.sql` | authorization repository/service; token claim emission |
| audit-events table | audit | `api/internal/audit/schema.sql` | audit repository/service; every domain that records an event |
| MFA tables (TOTP, recovery, challenges) | mfa | `api/internal/mfa/store.go` | mfa service/handlers |
| federation and API-key tables | future domains | not present | Add only with the scheduled domain |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

Schema changes are **additive migrations only — never database
recreation**. This is a hard rule for an auth store: recreating the DB
would destroy live credentials, sessions, and audit history. Domain
schema files use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them; new columns are added with additive migrations.
Note `ADD COLUMN IF NOT EXISTS` is Postgres-only syntax — SQLite needs a
guarded one-shot migration, not that idiom.

For any change that needs a column drop, rename, or data backfill, add a
scenario-specific migration plan here and record the tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md). The signing
keypair is never migrated in place — rotation introduces a new key with
an overlapping `kid` during rollover (deferred to P2, OT-P2-005).

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Realm | Realm deletion (admin, gated) | Until deleted; cascade-deletes its user pool. | Cascade + audit-of-deletion semantics to define at implementation. |
| User + credentials | Account deletion or realm cascade | Lifecycle of the account. | Right-to-erasure vs. audit-retention reconciliation to define. |
| Refresh-token family | Expiry, logout, or reuse-revocation | Until terminal; reuse revokes the whole family immediately. | — |
| Session | Revoke, "log out everywhere", or TTL expiry | TTL-bound in the configured hot-state store. | — |
| Audit events | Append-only; never user-deletable | Product-defined retention window (security record). | Retention window + archival policy to define. |
| MFA secret / recovery codes | User disenrolls or deletes the factor | Lifecycle of the enrolled factor. | — |
| OAuth CSRF state | Single-use on callback, or TTL | Short TTL in the configured hot-state store. | — |
| Hashed API key | Key revocation | Until revoked. | — |

## Privacy Notes

This scenario stores **personal and security-sensitive data** by design
(identities, credential hashes, sessions, and audit trails; future federation
links will add another sensitive class). The governing rules: only hashes and
signed material at rest, never plaintext secrets; the signing private key
never leaves the host; audit retention must be reconciled against any
erasure obligation. Keep
this document and [`../internal/SECURITY.md`](../internal/SECURITY.md) in
agreement as domains are implemented.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
