# Data — Development Toolchain Validator

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

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Notes | notes | SQLite | `api/internal/notes/schema.sql` | Until deleted by future product behavior | Template reference data; remove with notes domain. |
| Attachment metadata | notes | SQLite | `api/internal/notes/schema.sql` | Until parent note or attachment is deleted by future product behavior | Metadata only; bytes are stored through BlobStore. |
| Attachment bytes | notes | Filesystem BlobStore by default | BlobStore implementation in notes handler module | Same lifecycle as metadata | Opaque bytes stay outside proto payloads. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| notes tables | notes | `api/internal/notes/schema.sql` | notes repository/service/handlers |
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

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Template notes data | Domain removal or future product delete behavior | Local development data only | Real scenarios must define product-specific deletion semantics. |

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
