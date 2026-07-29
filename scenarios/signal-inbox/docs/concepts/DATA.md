# Data — Signal Inbox

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
| Signal | signals | SQLite | `api/internal/signals/schema.sql` | **Permanent. Never deleted.** | Append-only `signal` table. It records capture facts: source identity, captured_at, raw payload reference, direct-text content (when known at capture), and content hash. Written once; no UPDATE, no DELETE (D-006). |
| Signal media | signals | Filesystem BlobStore | BlobStore seam in the signals module | Same lifecycle as its signal — permanent | Pasted or uploaded images. Opaque bytes stay outside proto payloads. |
| Signal enrichment | enrichment | SQLite | `api/internal/enrichment/schema.sql` | Permanent, append-only | A post-capture extraction attempt: content units, readable content when found, and an attention reason when not. It is a read-model sidecar over `signal`, never an update to it (D-027). |
| Adapter state | sources | SQLite | `api/internal/sources/schema.sql` | Until the adapter is removed | Current per-adapter enabled/disabled flag, declared risk tier, last run, last error, and auto-disable reason. The planned source-stream configuration extends this state per stream rather than treating an entire source account as one toggle. |
| Source stream configuration | sources | SQLite | planned `api/internal/sources/` schema | Until the operator removes the stream | One row per source/activity pair. Stores the selected intake method, enablement, schedule mode, risk tier, priority, local/hosted inference-profile reference, credential **reference** (never its value), and successful-import checkpoint evidence. A stream can declare alternate methods, but only one is active at once. |
| Import run | sources | SQLite | `api/internal/sources/schema.sql` | Rolling window, operator-configurable | Per-run counts of created, deduplicated, and failed entries. Diagnostic, not authoritative. |
| Category | categories | SQLite | `api/internal/categories/schema.sql` | Until retired by the operator | Operator-defined at runtime. Retiring reassigns signals to `uncategorized`; it never deletes them (D-007). |
| Taxonomy | categories | SQLite | `api/internal/categories/schema.sql` | Lifetime of its category | Optional per category. Declares the subtype vocabulary for that category only (D-011). |
| Classification | categories | SQLite | `api/internal/categories/schema.sql` | Permanent | Proposal (category + confidence + model) and confirmation. The proposal is retained after confirmation — it is the input to accuracy measurement (D-009). |
| Disposition | triage | SQLite | `api/internal/triage/schema.sql` | Permanent, mutable | Exactly one row per signal. The only mutable product state in the scenario. Carries `revisit_at`. |
| Annotation | triage | SQLite | `api/internal/triage/schema.sql` | Permanent | Append-only, many per signal. Operator notes, agent notes, override records, and typed outcome links (D-014). A correction is a new annotation, never an edit. |
| Embedding index | retrieval | SQLite + vector index via aisearch-go | Derived from `signal` | Rebuildable cache | Covers **every** signal regardless of category, confidence, or disposition (D-022). Safe to drop and rebuild; the journal is the authority. |

## Schema Map

Each domain's schema file lives beside the code that interprets it. The
`system schema` is the only cross-cutting, non-domain table set.

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| signal, signal_media | signals | `api/internal/signals/schema.sql` | signals repository/service/handlers; read by every other domain |
| signal_enrichment | enrichment | `api/internal/enrichment/schema.sql` | enrichment repository/service; joined into the signal read model by signals/retrieval |
| adapter_state, import_run | sources | `api/internal/sources/schema.sql` | sources repository/service/handlers |
| category, taxonomy, classification | categories | `api/internal/categories/schema.sql` | categories repository/service/handlers |
| disposition, annotation | triage | `api/internal/triage/schema.sql` | triage repository/service/handlers |
| signal_fts (FTS5), embedding index | retrieval | `api/internal/retrieval/schema.sql` | retrieval repository/service/handlers |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Reference direction

Every domain reads `signal`; `signals` reads nothing. That keeps the dependency
chain acyclic and fixes the build order: `signals` first, then `enrichment`,
`categories`, and `triage` beside it, then `retrieval` last because it indexes
what the others produce. `sources` writes into `signals` and reads nothing else.

No table carries a foreign key into another scenario's domain objects. Outcome
links store an identifier and a target kind, never a copied payload, so there is
exactly one truth about whatever a signal produced.

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

Import is a product capability here, not an administrative convenience — it is
the tier-0 ingestion path (D-017) and the source of the first real corpus.

| Path | Format | Owner | Status |
|---|---|---|---|
| X archive import | Platform archive (zip containing JS/JSON payloads) | sources | Measured; parser planned. The archive contains authored posts and likes but no bookmarks. |
| X bookmarks sync | Official authenticated X API | sources | Planned tier-1 stream; disabled by default. Preserves folder context as tags when supplied. |
| Reddit export import | Platform GDPR ZIP containing CSV datasets | sources | Measured adapter exists; broader `SIG-P0-008` evidence remains planned |
| Browser bookmarks import | Netscape bookmark HTML | sources | Measured adapter exists; broader `SIG-P0-008` evidence remains planned |
| Corpus export | JSONL of signals with annotations and dispositions | signals | Planned — operator escape hatch; a capture substrate the operator cannot get data out of is a trap |

All imports are **idempotent by content hash**: re-running an import over
unchanged source data creates no duplicates, which is what makes repeated import
a safe routine operation rather than a destructive one. Import formats are owned
by the platforms and change without notice, so each adapter validates shape
before writing and fails the run rather than importing partial garbage. Archive
adapters also bound compressed input, relevant-file expansion, and captured-entry
counts before accumulating signals; an operator-selected ZIP is still untrusted
input and must not be able to exhaust the scenario process.

The measured Reddit GDPR export is a ZIP containing many CSV datasets. The
current tier-0 adapter deliberately reads only `saved_posts.csv` and
`saved_comments.csv`, whose `permalink` column yields an explicit saved URL.
It does not ingest authored activity, chats, votes, account/profile data, ad
preferences, IP logs, payment data, subreddit subscriptions, or any other
dataset merely because it is present in the export. A saved item is an
immutable capture candidate, not an assertion that it is relevant, valuable,
or classifiable; categorization and disposition remain ambient-only views over
the full stored corpus.

Future operator-authorized automation may request a date-bounded Reddit export
instead of repeatedly requesting all account history. It must persist the
requested window and advance its successful-import checkpoint only after the
corresponding archive has imported without failure. The next request must
overlap the prior window and rely on content-hash idempotency, rather than
treating a timestamp as a lossless cursor. That work is not a P0 adapter and
remains subject to a current terms/permission review before any network request
is enabled.

The planned source-stream model makes priority and sensitive-content handling
explicit. Stream priority controls review and ambient ordering only: `primary`
for authored X activity and bookmarks, `candidate` for likes/upvotes, and no
priority value may delete, suppress from search, or silently recategorize a
signal. Archive-derived text and sensitive-content assessment must run through
an operator-selected local ai-gateway profile before ambient surfacing. The
assessment is a fallible tag, not a deletion filter; suspected adult material
stays in the permanent journal and is excluded from ambient views by default.

## Retention And Deletion

The retention story is unusually simple because the answer is almost always
"forever". Deletion is not a product feature; `dropped` is a disposition, not a
delete (D-006, D-013).

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Signal | **None.** No product path deletes a signal. | Permanent. | Operator-initiated hard-delete for material captured in error is deliberately absent; if it is ever added it must be an explicit, logged, single-signal operation, never a filter-driven bulk path. |
| Signal media | Same as its signal | Permanent | Blob-store growth is unmeasured; large image corpora may need a size budget. |
| Annotation | **None.** Corrections append. | Permanent. | — |
| Classification proposal | **None.** Retained after confirmation. | Permanent. | Retained deliberately: discarding the proposal on confirmation would destroy the accuracy corpus (D-009). |
| Disposition | Never deleted; mutated in place | Permanent, current value only | No disposition history is kept, so "when was this marked done" is unanswerable. Accepted for now; revisit if triage auditing is ever needed. |
| Import run | Rolling window | Operator-configurable | Diagnostic only; safe to prune. |
| Embedding index | Dropped and rebuilt at will | Rebuildable cache | Rebuild cost at large corpus size is unmeasured. |

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
