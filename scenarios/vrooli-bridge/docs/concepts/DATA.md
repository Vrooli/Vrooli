# Data — Vrooli Bridge

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

## Storage Overview

The template default is embedded SQLite through `modernc.org/sqlite`.
The lifecycle sets `SQLITE_PATH` through `.vrooli/service.json`, and
the API applies schemas on startup through `api-core/database`.

External storage resources should be introduced only when a real
domain needs them. Document those decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md) before editing
`.vrooli/service.json`.

## Data Ownership

Each domain owns its own tables and is the source of truth for its data.
The `health` domain owns no product data — it only probes configured
database reachability. As you build real domains, add a row per data
shape they persist: name it, name the owning domain, the storage backend,
the schema file that is the source of truth, the retention rule, and any
remarks. Keep blob/opaque bytes outside proto payloads, behind a seam
such as BlobStore.

All control-plane data lives in SQLite via `api-core/storage`. Bridge deliberately stores **no large binaries** — build artifacts move through device-sync-hub, and job logs/artifacts stream back as references. The audit trail is routed to workspace-sandbox, not a bespoke local table.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Node records | registry | SQLite | `api/internal/registry/schema.sql` | Until revoked | OS/arch/revision/endpoint/capabilities/permission scopes; durable infrastructure identity. |
| Pairing codes | pairing | SQLite | `api/internal/pairing/schema.sql` | Single-use, short TTL | Out-of-band bootstrap; expire/burn on redeem. |
| Node credentials | pairing | SQLite (hashed at rest) | `api/internal/pairing/schema.sql` | Until node revoked | Mutual-auth secret material; never stored in plaintext. |
| Presence + health | presence | In-memory (optionally Redis) | live channel state | Ephemeral | Not persisted; rebuilt on reconnect. |
| Run records | runs | SQLite | `api/internal/runs/schema.sql` | Configurable policy | Durable server-owned run state; survives disconnect. |
| Run logs / artifact refs | runs | SQLite (refs); bytes via device-sync-hub | `api/internal/runs/schema.sql` | Configurable policy | Bytes stay out of the control-plane store. |
| Provisioning ops + version history | provisioning | SQLite | `api/internal/provision/schema.sql` | Retained for drift/rollback | Per-node revision history. |
| Gate runs + per-OS verdicts | gate | SQLite | `api/internal/gate/schema.sql` | Configurable policy | Aggregated cross-OS results. |
| Audit records | audit | workspace-sandbox (append-only) | workspace-sandbox | Immutable | Every dispatch + provisioning op; not a local mutable table. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `nodes` | registry | `api/internal/registry/schema.sql` | registry repository/service/handlers |
| `pairing_codes`, `node_credentials` | pairing | `api/internal/pairing/schema.sql` | pairing + auth |
| `runs`, `run_logs` | runs | `api/internal/runs/schema.sql` | runs repository/service/handlers |
| `provisioning_ops`, `node_versions` | provisioning | `api/internal/provision/schema.sql` | provisioning service |
| `gate_runs`, `gate_verdicts` | gate | `api/internal/gate/schema.sql` | gate service |
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

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None yet. | n/a | n/a | Add when product requirements include import/export. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Node record + credentials | Node revocation | Until revoked; credentials destroyed atomically on revoke | Revocation semantics defined at implementation. |
| Pairing code | Redeem or TTL expiry | Single-use, short TTL | — |
| Run records + logs | Configurable retention policy | Kept long enough to read verdicts/logs, then pruned | Default policy to be set at implementation. |
| Audit records | Never (append-only) | Immutable; retained for traceability | Lives in workspace-sandbox. |

## Privacy Notes

Bridge stores **operational metadata about the owner's own machines** — node identities, OS/arch/revision, reachable endpoints, permission scopes, command/job history, and an immutable audit trail. It stores no third-party personal data. Two areas warrant care and are detailed in [`../internal/SECURITY.md`](../internal/SECURITY.md): (1) **node credentials** are mutual-auth secret material and are hashed at rest; (2) **job logs/artifacts** streamed back from a node can contain whatever the executed command emitted, so they inherit the sensitivity of the scenario under test and follow the same retention/access controls as the run records that own them.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
