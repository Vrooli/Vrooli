# Data — Offer Desk

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
| Nodes and edges | catalog | SQLite | `api/internal/catalog/schema.sql` | Retired nodes are kept, never deleted. | One status vocabulary across all node kinds. |
| Status history | catalog | SQLite | `api/internal/catalog/schema.sql` | **Never deleted.** Not regenerable. | Append-only: actor, timestamp, prior status, reason. |
| Trigger declarations | gates | SQLite | `api/internal/gates/schema.sql` | Deleted with the node. | Parsed predicate, stored in its declared form. |
| Facts | gates | SQLite | `api/internal/gates/schema.sql` | Superseded by newer values; history retained. | Each carries a source and an observed-at time. |
| Evaluation runs | gates | SQLite | `api/internal/gates/schema.sql` | Windowed retention. | Records which fact satisfied which clause. |
| Promotion proposals | gates | SQLite | `api/internal/gates/schema.sql` | Kept after disposition. | Carries the proposing actor; survives restarts. |
| Import findings | catalog | SQLite | `api/internal/catalog/schema.sql` | Retained until resolved. | Unresolvable source references, kept visible rather than dropped. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| `nodes`, `edges`, `status_history`, `import_findings` | catalog | `api/internal/catalog/schema.sql` | catalog repository/service; read by gates and board |
| `triggers`, `facts`, `evaluation_runs`, `proposals` | gates | `api/internal/gates/schema.sql` | gates repository/service; read by board |
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

The status history is the only record of how an offer reached its current state, and nothing upstream holds it — the source markdown files that preceded this scenario carried a current value and no history at all.

Two consequences:

1. Status history is declared **non-regenerable** to the platform's backup policy. It is not ordinary scenario state.
2. No API path deletes or edits a history entry. A mistaken transition is corrected by a further transition with a reason, which preserves both.

Import findings are deliberately retained rather than resolved-and-dropped: the source catalog had a measured 19% broken internal-link rate, and discarding those findings at import time would make the drift disappear along with the documents that carried it.

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
