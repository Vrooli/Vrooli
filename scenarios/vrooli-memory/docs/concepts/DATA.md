# Data — Vrooli Memory

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
| Journal entry | journal | SQLite | `api/internal/journal/schema.sql` | **Never deleted.** | Prose body, facet tag, author attribution, run correlation ids, timestamps. Append-only; no UPDATE, no DELETE. |
| Facet text | journal | SQLite | `api/internal/journal/schema.sql` | Follows its entry. | Short derived texts per entry (topic, rule/implication, entities). Human-readable so clustering is debuggable. |
| Embedding | journal | SQLite via `aisearch-go` | `api/internal/journal/schema.sql` | Follows its facet text. | One vector per facet text. Summaries are embedded too — required, or nothing above depth 1 can form. |
| Facet assignment | facets | SQLite | `api/internal/facets/schema.sql` | Follows its entry. | Exactly one facet per entry from a closed set. Corrections write a new assignment; history is kept. |
| Pin state | facets | SQLite | `api/internal/facets/schema.sql` | Until unpinned. | Pinned entries are exempt from compaction candidacy and unconditionally included in `wake`. |
| Supersession / expiry mark | facets | SQLite | `api/internal/facets/schema.sql` | Never deleted. | Marks an entry superseded or a thread resolved. The superseded entry itself is retained — marks are additive, like everything else. |
| Summary node | forest | SQLite | `api/internal/forest/schema.sql` | Rebuildable. | Summary text, span (`lo`–`hi`), depth, generation count. Safe to drop and recompute from the journal. |
| Tree edge | forest | SQLite | `api/internal/forest/schema.sql` | Rebuildable. | Parent/child links. Frontier membership is derived from these (a node with no parent is on the frontier). |
| Provider descriptor | federation | JSON file | `.vrooli/search.json` | Versioned in git. | Scenario-owned SSOT for routing description, result mapping, and tuning. Search-hub may write tuning back through the config endpoint. |
| Projection state | harness | SQLite | `api/internal/harness/schema.sql` | Overwritten. | Last projection hash and target paths. The projected file itself is a generated artifact, not data. |

**Retention summary.** Only two things in this scenario are ever destroyed:
summary nodes and tree edges, both of which are a rebuildable cache. Everything
in `journal` and `facets` is permanent. If a retention requirement ever demands
deleting a journal entry, that is a change to the scenario's central invariant
and belongs in [`../internal/DECISIONS.md`](../internal/DECISIONS.md), not in a
migration.

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| entries, facet_texts, embeddings | journal | `api/internal/journal/schema.sql` | journal repository/service; read by recall and forest |
| facet_assignments, pins, marks | facets | `api/internal/facets/schema.sql` | facets policy engine; read by forest (candidacy) and recall (wake) |
| summaries, tree_edges | forest | `api/internal/forest/schema.sql` | forest compaction pass; read by recall |
| projections | harness | `api/internal/harness/schema.sql` | harness projection writer |
| `.vrooli/search.json` | federation | scenario root | boot self-registration; search-hub sweep write-back |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Rebuild contract

`forest` tables are a cache. A rebuild drops `summaries` and `tree_edges`, then
replays compaction from `entries` + `facet_assignments`. This is the recovery
path for a bad summarization generation and the reason
`VROOLIME-P2-003` (drift monitoring) is deferrable rather than blocking — a
corrupted tree is always recoverable, so drift costs quality, never data.

The reverse never holds: `journal` cannot be rebuilt from `forest`. Any change
that would let a derived table become authoritative for entry content is a
defect in the layering, not a schema decision.

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

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
