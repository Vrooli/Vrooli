# Data — Money Ledger

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

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Books and accounts | books | SQLite | `api/internal/books/schema.sql` | Never auto-deleted. | An account cannot exist without a book. |
| Events and postings | journal | SQLite | `api/internal/journal/schema.sql` | **Never deleted.** Not regenerable. | Append-only. A correction is a reversing entry, not an edit. |
| Audit entries | journal | SQLite | `api/internal/journal/schema.sql` | Never deleted. | Actor, timestamp, prior value, reason. |
| Adapter registrations and cursors | ingest | SQLite | `api/internal/ingest/schema.sql` | Cursor may be reset by an operator. | Credentials are **not** stored here — they live in the platform secret store. |
| Ingestion receipts | ingest | SQLite | `api/internal/ingest/schema.sql` | Retained per adapter window. | Keyed on (adapter, external id) — this is what makes ingestion idempotent. |
| Goal declarations | position | SQLite | `api/internal/position/schema.sql` | Deleted with the goal. | A declaration, never a computed result. |
| Position snapshots | position | SQLite | `api/internal/position/schema.sql` | Short-TTL cache; safe to drop. | A cache, never a source of truth. Balances are recomputed. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `books`, `accounts` | books | `api/internal/books/schema.sql` | books repository/service/handlers |
| `events`, `postings`, `audit_entries` | journal | `api/internal/journal/schema.sql` | journal repository/service; read by position |
| `adapters`, `cursors`, `receipts` | ingest | `api/internal/ingest/schema.sql` | ingest repository/service |
| `goals`, `position_snapshots` | position | `api/internal/position/schema.sql` | position repository/service |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

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

## The Not-Regenerable Rule

Most scenario state can be rebuilt from something else. This one cannot. The journal is the only record that a money event occurred, and no upstream is guaranteed to still hold it — a payment processor prunes history, a bank exports only a rolling window, and a manually entered cash sale exists nowhere else at all.

Two consequences that must not be softened:

1. The journal is declared **non-regenerable** to the platform's backup policy. It is not ordinary scenario state and must not be treated as a cache that can be dropped and refilled.
2. Deletion is not offered for events, postings, or audit entries through any API path. A mistake is corrected with a reversing entry that references what it reverses, which preserves both the error and the correction.

Position snapshots are the exception and are safe to drop at any time: they are a cache over the journal, and dropping them costs a recomputation.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| _(your data)_ | What removes it. | How long it is kept. | Real scenarios must define product-specific deletion semantics. |

## Privacy Notes

Generated template data is local development data. If a scenario stores
personal, regulated, customer, financial, or sensitive business data,
update this document and [`../internal/SECURITY.md`](../internal/SECURITY.md)
before implementation expands.

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
