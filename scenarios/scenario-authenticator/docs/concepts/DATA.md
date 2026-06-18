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

> **Status: documentation-first orientation.** Nothing below the
> `health` domain and the fenced `notes` example is implemented yet. The
> entities and schema files named here are the **target** data model
> carried over from [`../../PRD.md`](../../PRD.md) and
> [`DOMAINS.md`](DOMAINS.md). No tables exist beyond the scaffold.

## Storage Overview

Persistence is **SQLite through the `api-core/storage` seam** — not
shared Postgres. Moving off the shared database is the reason for the
rewrite: a fleet-wide shared DB was a fleet-wide blast radius. The seam
keeps the store swappable to a managed DB for cloud scale, so SQLite is
the default, not a lock-in. The lifecycle sets `SQLITE_PATH` through
`.vrooli/service.json`, and the API applies domain schemas on startup
through the seam.

Two substrates back the data model, by access pattern:

- **SQLite (durable)** holds everything that must survive a restart:
  realms, users, credential hashes, refresh-token families, role/scope
  definitions, audit events, MFA secrets, federation links, and hashed
  API keys. **Only hashes and signed material are stored at rest —
  never plaintext secrets.**
- **Redis (hot/ephemeral state)** is a **required** resource (see
  [`INTEGRATIONS.md`](INTEGRATIONS.md)). It backs sessions, token/family
  revocation lookups, OAuth CSRF state, and cross-replica rate-limit
  coordination — state that is read on the hot path or must be shared
  across replicas.
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
dependency reachability. The table below is the **target** ownership map;
no rows are implemented yet.

| Data | Owning Domain | Store | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Realms + per-realm policy/branding/config | realms | SQLite | `api/internal/realms/schema.sql` | Until the realm is deleted (cascade-deletes its users). | The tenant boundary; carries `aud`, password policy, token TTLs, redirect URIs. |
| Users (realm-scoped principals) | identity | SQLite | `api/internal/identity/schema.sql` | Until account deletion. | Same email is a distinct principal per realm; unique within a realm. |
| Credentials (Argon2id hashes) + verification state | identity | SQLite | `api/internal/identity/schema.sql` | Lifecycle of the owning user. | **Argon2id hash only**, documented cost; no plaintext ever. |
| Signing keypair (`private.pem` / `public.pem`) | tokens | Filesystem (storage root) | `api-core/storage` root, load-or-generate | Durable; rotation is deliberate (P2 OT-P2-005). | Private key never leaves the host; public key is published as JWKS. |
| JWKS document | tokens | Derived (from public key) | Built from the keypair at boot | Re-derived on start. | Served at `/.well-known/jwks.json` for RP local verification. |
| Refresh-token families | tokens | SQLite (+ Redis revocation) | `api/internal/tokens/schema.sql` | Until family expiry or reuse-revocation. | Rotation + **reuse detection**: a reused token revokes the whole family. |
| Sessions | sessions | **Redis** (hot state) | Redis keyspace, `api/internal/sessions/` | TTL-bound; cleared on revoke / "log out everywhere". | Listable + per-session revoke; the carried-over `/api/v1/sessions/{id}` contract. |
| Roles / scopes + assignments | authorization | SQLite | `api/internal/authorization/schema.sql` | Until removed. | *Definitions* are owned here and emitted as claims; enforcement is the RP's. |
| Audit events | audit | SQLite | `api/internal/audit/schema.sql` | Append-only; product-defined retention window. | Security-relevant events, queryable per realm. |
| MFA secrets + recovery codes (P1) | mfa | SQLite | `api/internal/mfa/schema.sql` | Lifecycle of the enrolled user. | TOTP secrets + hashed recovery codes; secrets stored only as needed for challenge. |
| Passkey / WebAuthn credentials (P1) | mfa | SQLite | `api/internal/mfa/schema.sql` | Until the user removes the passkey. | Public-key credential material only. |
| Linked external identities + provider config (P1) | federation | SQLite | `api/internal/federation/schema.sql` | Until the account link is removed. | OAuth/OIDC (P1) and SAML (P2) account linking. |
| OAuth CSRF state (P1) | federation | **Redis** (ephemeral) | Redis keyspace | Short TTL, single-use. | Protects the social-login callback round trip. |
| Hashed API keys + client records (P1) | apikeys | SQLite | `api/internal/apikeys/schema.sql` | Until the key is revoked. | **Hash only**; the secret is shown once at creation, never stored. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| realms tables | realms | `api/internal/realms/schema.sql` | realms repository/service/handlers |
| users + credentials tables | identity | `api/internal/identity/schema.sql` | identity repository/service/handlers; crypto lib for Argon2id |
| refresh-token-family tables | tokens | `api/internal/tokens/schema.sql` | tokens repository/service; crypto lib for RS256/JWKS |
| `private.pem` / `public.pem` | tokens | storage root (load-or-generate) | crypto lib (sign + JWKS publication) |
| session keys | sessions | Redis keyspace (`api/internal/sessions/`) | sessions service/handlers; revocation lookups |
| roles / scopes / assignments tables | authorization | `api/internal/authorization/schema.sql` | authorization repository/service; token claim emission |
| audit-events table | audit | `api/internal/audit/schema.sql` | audit repository/service; every domain that records an event |
| mfa tables (TOTP, recovery, passkeys) | mfa | `api/internal/mfa/schema.sql` | mfa repository/service/handlers |
| federation tables (links, providers) | federation | `api/internal/federation/schema.sql` | federation repository/service; OAuth/SAML callbacks |
| oauth CSRF keys | federation | Redis keyspace | federation callback handlers |
| apikeys tables | apikeys | `api/internal/apikeys/schema.sql` | apikeys repository/service; client-credentials grant |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `vrooli scenario detemplate`)

The template ships the `notes` domain as a worked CRUD slice with a
binary attachment-upload exception, showing how a real domain owns its
tables, metadata, and opaque blob bytes. Copy its shape, then remove it.

Its Data Ownership rows:

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Notes | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted by future product behavior | Template reference data; remove with notes domain. |
| Attachment metadata | notes | SQLite | `api/internal/notes/schema.sql` | Until parent note or attachment is deleted by future product behavior | Metadata only; bytes are stored through BlobStore. |
| Attachment bytes | notes | Filesystem BlobStore by default | BlobStore implementation in notes handler module | Same lifecycle as metadata | Opaque bytes stay outside proto payloads. |

Its Schema Map row:

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| notes tables | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |

Its Retention And Deletion row:

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Template notes data | Domain removal or future product delete behavior | Local development data only | Real scenarios must define product-specific deletion semantics. |
<!-- EXAMPLE-DOMAIN:notes END -->

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
| Session | Revoke, "log out everywhere", or TTL expiry | TTL-bound in Redis. | — |
| Audit events | Append-only; never user-deletable | Product-defined retention window (security record). | Retention window + archival policy to define. |
| MFA secret / recovery codes | User disenrolls or deletes the factor | Lifecycle of the enrolled factor. | — |
| OAuth CSRF state | Single-use on callback, or TTL | Short Redis TTL. | — |
| Hashed API key | Key revocation | Until revoked. | — |

## Privacy Notes

This scenario stores **personal and security-sensitive data** by design
(identities, credential hashes, sessions, audit trails, federation
links). The governing rules: only hashes and signed material at rest,
never plaintext secrets; the signing private key never leaves the host;
audit retention must be reconciled against any erasure obligation. Keep
this document and [`../internal/SECURITY.md`](../internal/SECURITY.md) in
agreement as domains are implemented.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
