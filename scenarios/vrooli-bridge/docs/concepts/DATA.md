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

Control-plane metadata lives in SQLite via `api-core/storage`. The inbound
distribution path remains metadata-only: build artifacts move through
device-sync-hub and bridge stores only their delivery reference. Typed outputs
produced by an authenticated run are the deliberate exception: bridge stores
their bounded bytes in the artifacts domain, keyed by run and name, so an
owner can retrieve plan evidence without reading a node-local path. The audit
trail is routed to workspace-sandbox, not a bespoke local table.

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Node records | registry | SQLite | `api/internal/registry/schema.sql` | Until revoked | OS/arch/revision/endpoint/capabilities/permission scopes; durable infrastructure identity. |
| Pairing codes | pairing | SQLite | `api/internal/pairing/schema.sql` | Single-use, short TTL | Out-of-band bootstrap; expire/burn on redeem. |
| Node credentials | pairing | SQLite (public key only) | `api/internal/pairing/schema.sql` | Until node revoked | Ed25519 public key for verification; the private key never leaves the node. |
| Presence + health | presence | In-memory (optionally Redis) | live channel state | Ephemeral | Not persisted; rebuilt on reconnect. |
| Run records | runs | SQLite | `api/internal/runs/schema.sql` | Configurable policy | Durable server-owned run state; survives disconnect. |
| Run logs / artifact refs | runs | SQLite (refs); bytes via device-sync-hub | `api/internal/runs/schema.sql` | Configurable policy | Bytes stay out of the control-plane store. |
| Provisioning ops + version history | provisioning | SQLite | `api/internal/provision/schema.sql` | Retained for drift/rollback | Per-node revision history. |
| Gate runs + per-OS verdicts | gate | SQLite | `api/internal/gate/schema.sql` | Configurable policy | Aggregated cross-OS results. |
| Audit records | audit | workspace-sandbox (append-only) | workspace-sandbox | Immutable | Every dispatch + provisioning op; not a local mutable table. |
| Machine intent + locators | machines | SQLite | `api/internal/machines/schema.sql` | Until archive/removal policy completes | Stable UUID and ordered normalized locators; no copied Node, Presence, or capability facts. |
| Machine Node lineage | machines | SQLite | `api/internal/machines/schema.sql` | Immutable until exceptional purge | At most one current Node per Machine; re-pair appends lineage and supersedes/revokes the prior Node. |
| Enrollment attempts + checkpoints | enrollment | SQLite | `api/internal/onboard/schema.sql` | Immutable history until exceptional purge | Input snapshots, correlation IDs, typed checkpoints, diagnostics, retry lineage, and reconciliation state. No passwords, pairing codes, or private keys. |
| SSH private material / host trust | trust store | Filesystem secret store | machine trust-store adapter | Rotated/revoked by lifecycle policy | Per-Machine private key material stays outside SQLite/API/logs. SQLite holds only opaque references and public fingerprints. |

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
| `machines`, `machine_locators`, `machine_node_lineage`, migration review | machines | `api/internal/machines/schema.sql` | machines service and composed projections |
| `enrollment_attempts`, checkpoints, reconciliations | enrollment | `api/internal/enrollment/schema.sql` | enrollment service and pairing recovery |

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

Machine migration is forward-only, idempotent, and preserves historic IDs and
audit rows. It can automatically link only records with durable pairing
correlation. All other candidate relationships remain in an explicit
`needs_review` state with provenance and confidence; hostname, IP, username,
and display-name matching are forbidden adoption rules.

## Machine Ownership Matrix

| Fact | Sole owner | Allowed projection | Forbidden duplicate |
|---|---|---|---|
| Operator intent, locators, lifecycle, desired policy | machines | enrollment/API/CLI/UI reads | Registry Node fields or Presence snapshot on Machine rows |
| Attempt state, checkpoints, retry/reconciliation history | enrollment | Machine detail/history | mutable Machine lifecycle or Registry Node state |
| Paired identity and approved scopes | registry | Machine lineage/detail | Machine approval or attempt-scoped authorization |
| Live reachability and health | presence | composed readiness | persisted Machine/attempt health copy |
| SSH private key and known-host material | trust store | opaque ref + public fingerprint | SQLite/API/log/private-key field |
| Workload identity/lifecycle | owning external scenario | typed `WorkloadReference` only | Bridge-owned workload truth |

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

Bridge stores **operational metadata about the owner's own machines** — node identities, OS/arch/revision, reachable endpoints, permission scopes, command/job history, and an immutable audit trail. It stores no third-party personal data. Three areas warrant care and are detailed in [`../internal/SECURITY.md`](../internal/SECURITY.md): (1) **node credentials** are mutual-auth secret material and are hashed at rest; (2) **distributed artifacts** remain references because device-sync-hub owns their bytes; (3) **produced run artifacts** can contain whatever the executed command emitted, so their bounded bytes inherit the sensitivity of the scenario under test and are owner-gated by run identity.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
