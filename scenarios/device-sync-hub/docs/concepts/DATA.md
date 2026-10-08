# Data — Device Sync Hub

This document is the canonical data ownership and storage map for the
scenario. Update it when domains add tables, files, blobs, external
records, retention rules, migrations, imports, or exports. Domain
ownership mirrors [`DOMAINS.md`](DOMAINS.md) exactly.

## Purpose Of This Document

Use this document to answer:

- What data does the scenario persist?
- Which domain owns each data shape?
- Where is the source of truth?
- What is ephemeral versus durable?
- What is the retention/deletion story?
- How are schema changes handled?

## Storage Overview

Device Sync Hub stores two kinds of state, through two `api-core` seams:

- **Relational metadata** (devices, pairing codes, item metadata,
  settings) via `api-core/storage`, the `data` storage class, rooted at
  `~/.vrooli/data/vrooli/device-sync-hub`. The backing engine is
  embedded SQLite through `modernc.org/sqlite`; the lifecycle sets
and the API applies
  schemas on startup through `api-core/database`. The schema is written
  Postgres-compatible for forward multi-tenant readiness, but v1 ships
  SQLite for the single-owner deployment.
- **Binary payloads** (file bytes, image thumbnails) via
  `api-core/blobstore`. Bytes never enter proto payloads or the SQLite
  store; the `items` row holds only a `blob_ref` / `thumbnail_ref`
  pointer. Uploads stream straight to the blobstore; downloads stream
  back preserving the original filename. Optional at-rest AES-256
  encryption (OT-P1-007, default off) is applied at the blobstore seam.

Presence is **not** stored here — it is ephemeral (see below).

## Storage Class And Path Policy

| Concern | Policy |
|---|---|
| Storage class | `data` (runtime, owner-private). |
| Metadata root | `~/.vrooli/data/vrooli/device-sync-hub`, resolved by `api-core/storage` from the scenario id. |
| Blob root | Resolved by `api-core/blobstore` under the same data class. |
| Access | Only through the storage `Resolver` and `BlobStore` seams; no domain reads the filesystem directly. |
| Cross-scenario access | Forbidden — other scenarios reach data only through the versioned CLI/API, never the DB or blobstore. |

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Durability | Notes |
|---|---|---|---|---|---|
| Device records | devices | SQLite (`devices`) | `api/internal/devices/schema.sql` | Durable until revoked | Trust-group membership; revoke drops the row. |
| Pairing codes | devices | SQLite (`pairing_codes`) | `api/internal/devices/schema.sql` | Short-TTL, single-use | Expire/burn on redeem; not long-lived. |
| Item metadata | transfer | SQLite (`items`) | `api/internal/transfer/schema.sql` | Durable per retention policy | Points at blobstore bytes; carries retention/expiry. |
| Item bytes + thumbnails | transfer | `api-core/blobstore` | BlobStore implementation in transfer module | Same lifecycle as the item row | Opaque bytes stay outside proto payloads; optional at-rest encryption. |
| Owner settings | settings | SQLite (`settings`) | `api/internal/settings/schema.sql` | Durable singleton | Retention default, quota limits, encryption toggle. |
| Presence | realtime | In-memory (optionally Redis) | WebSocket presence registry | **Ephemeral** | Online/offline state; lost on restart, rebuilt as devices reconnect. |
| Owner identity / JWT / sessions | auth | **Not stored here** | `scenario-authenticator` | n/a | Delegated; see [`INTEGRATIONS.md`](INTEGRATIONS.md). |

## Schema Map

| Table/File/Object | Owner | Key Columns | Defined In |
|---|---|---|---|
| `devices` | devices | `id`, `owner`, `name`, `type/os`, `capabilities`, `last_seen`, `trust_state` | `api/internal/devices/schema.sql` |
| `pairing_codes` | devices | code (short-TTL, single-use), owner, `expires_at`, redeemed/used flag | `api/internal/devices/schema.sql` |
| `items` | transfer | `id`, `owner`, `origin_device`, `kind`=file\|text, `name`, `mime`, `size`, `thumbnail_ref`, `blob_ref`, `retention`=Live\|Held\|Pinned, `expires_at`, `target`=broadcast\|device_id, `created_at` | `api/internal/transfer/schema.sql` |
| `settings` | settings | owner config singleton: retention default, per-device + global quota limits, at-rest encryption toggle | `api/internal/settings/schema.sql` |
| system schema | infrastructure | reachability/bootstrap tables | `api/internal/database/system.sql` |

## Retention And Deletion

Retention is owned by `transfer`; the global default lives in `settings`
and is stamped onto each `items` row at upload time. See the retention
purge lifecycle in [`FLOWS.md`](FLOWS.md).

| Retention | Delete Trigger | Rule |
|---|---|---|
| Live | Delivered to all connected target devices | Deliver-then-purge: the item and its blob are removed once delivered. |
| Held | Time-based | Auto-purge after the configured default (out-of-the-box 24 h) via `expires_at` and the purge scheduler. |
| Pinned | Manual deletion only | Kept until the owner deletes it. |

Additional deletion triggers:

- **Device revocation** drops the `devices` row and the device's
  authenticator session; the device immediately loses read/write access
  to all items (no per-item delete needed — access is gated, not the
  bytes wiped).
- **Clear-all** (permission-gated, in `settings`) purges all items and
  blobs for the owner.
- A purged item's blob and thumbnail are removed from the blobstore in
  the same operation as the metadata row.

## Quotas

Per-device and global storage quotas are owned by `transfer` and
configured in `settings`. Quotas are checked **before** any bytes are
written to the blobstore (OT-P1-002); an upload that would breach a
quota is rejected with a clear error and writes nothing. Quota
accounting sums the `size` of durable (`Held`/`Pinned`) items; `Live`
items are transient and purge on delivery.

## Migrations And Compatibility

Schema changes go through numbered, idempotent migration files beside
the owning domain's code; the database is **never** dropped or recreated
on schema change (OT-P0-007). CI must reject any code path that would
drop or recreate the store. Domain schema files use additive,
forward-compatible definitions; for column drops, renames, or backfills,
add a scenario-specific migration plan here and record the tradeoff in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

Note: `ADD COLUMN IF NOT EXISTS` is Postgres-only; SQLite migrations
must guard column changes explicitly rather than relying on that syntax.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| None in v1. | n/a | n/a | Items are short-lived transfers; bulk import/export is out of scope until a real requirement lands. |

## Privacy Notes

All stored data is owner-private and lives on the owner-operated Vrooli
server. The security model is TLS in transit (mandatory) plus optional
at-rest blob encryption; there is **no** end-to-end or zero-knowledge
guarantee — the server is owner-trusted and can read item bytes. If a
deployment stores regulated or third-party data, update this document
and [`../internal/SECURITY.md`](../internal/SECURITY.md) before scope
expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — storage seams and request path
- [`FLOWS.md`](FLOWS.md) — retention purge and revocation lifecycles
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
