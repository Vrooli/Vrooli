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

**No blob storage (D-020).** Unlike the template's `notes` example, `journal`
has no BlobStore seam. An entry is bounded prose plus its derived facet texts;
a memory that needs to point at a large artifact stores the reference as text.
This is deliberate rather than unimplemented: compaction scores clusters by node
count, not information, so chunking a long document into entries would distort
frontier accounting as well as flood it. Reopening this is the decision the
source-ledger direction would need to make first — see
[`ARCHITECTURE.md`](ARCHITECTURE.md) § Deliberately Not Built.

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
| Facet definition / policy | facets | SQLite | `api/internal/facets/schema.sql` | Seeded at boot; overwritten on reseed. | The closed facet set and its facet→retention-policy mapping, held as **data rather than Go constants** (D-019). Closed still means closed — validation loads this table and an unrecognised facet is a hard error at write. |
| Facet assignment | facets | SQLite | `api/internal/facets/schema.sql` | Follows its entry. | Exactly one facet per entry from a closed set. Corrections write a new assignment; history is kept. |
| Pin state | facets | SQLite | `api/internal/facets/schema.sql` | Until unpinned or lapsed. | Pinned entries are exempt from compaction candidacy and unconditionally included in `wake`. Carries a review date; lapsing clears the pin flag only — the journal entry is untouched and stays a searchable standing-rule memory. |
| Pin review / merge proposal | facets | SQLite | `api/internal/facets/schema.sql` | Until resolved. | The curation queue behind `VMEM-P1-010`. Proposals are advisory records; resolving one writes pin state, never journal content. |
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
| facet_definitions, facet_policies | facets | `api/internal/facets/schema.sql` | facets validation and policy lookup; seeded at boot (D-019) |
| facet_assignments, pins, pin_reviews, merge_proposals, marks | facets | `api/internal/facets/schema.sql` | facets policy engine; read by forest (candidacy) and recall (wake) |
| summaries, tree_edges | forest | `api/internal/forest/schema.sql` | forest compaction pass; read by recall |
| projections | harness | `api/internal/harness/schema.sql` | harness projection writer |
| `.vrooli/search.json` | federation | scenario root | boot self-registration; search-hub sweep write-back |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

### Rebuild contract

`forest` tables are a cache. A rebuild drops `summaries` and `tree_edges`, then
replays compaction from `entries` + `facet_assignments`. This is the recovery
path for a bad summarization generation and the reason
`VMEM-P2-003` (drift monitoring) is deferrable rather than blocking — a
corrupted tree is always recoverable, so drift costs quality, never data.

The reverse never holds: `journal` cannot be rebuilt from `forest`. Any change
that would let a derived table become authoritative for entry content is a
defect in the layering, not a schema decision.

## Migrations And Compatibility

The generated template uses idempotent schema bootstrap. Domain schema
files should use `CREATE TABLE IF NOT EXISTS` and live beside the code
that interprets them.

For production data migrations that need column drops, renames, or data
backfills, add a scenario-specific migration plan here and update
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) with the tradeoff.

## Import / Export

Import is a **first-class, permanent capability**, not a one-time migration
tool. Even where native writes are captured by a hook, the importer is still
required for two things a hook cannot do: the initial backfill of everything
already sitting in a harness store, and drift recovery when a hook fails or an
agent writes through a path nobody anticipated.

### Inventory

| Path | Format | Owner | Status |
|---|---|---|---|
| Harness memory stores (`resources/<agent>/`) | markdown file-per-fact, markdown section, single-blob markdown, JSONL, SQLite | harness | `VMEM-P0-011` — recurring, idempotent |
| `swarm-manager records` | Connect-RPC read | harness | `VMEM-P1-001` — one-time |
| Existing `MEMORY.md` content | markdown | harness | Covered by the harness adapter for Claude Code |
| Export | — | — | None planned. The journal is the export surface; `recall --json` and the API serve it. |

### Adapters are declarative

A harness adapter is a descriptor, not Go code per vendor — the same idiom as
`.vrooli/search.json` making a new search provider a registry row rather than a
router change. Adding a runtime is a descriptor, not a build.

| Field | Purpose |
|---|---|
| `harness_id` | `claude-code`, `codex`, `gemini`, … — matches `resources/<agent>/` |
| `locations[]` | Glob paths to the memory store(s) |
| `format` | `markdown-per-file` \| `markdown-section` \| `markdown-blob` \| `jsonl` \| `sqlite` |
| `extract` | Format-specific: split level for markdown, section header for `markdown-section`, field map for JSONL, query + column map for SQLite |
| `provenance` | Optional field map lifting source metadata into the journal entry |

Storage shapes and which harness uses which are recorded in
[`INTEGRATIONS.md`](INTEGRATIONS.md) → Harness Capability Matrix.

### Idempotency

Re-import must never duplicate. The mechanism is a **content-addressed import
key**, unique-indexed on the journal:

```
import_key = hash(harness_id, source_path, normalized_content)
```

Re-running an import is a no-op for anything unchanged. This is the diff, and
it requires no watermark, cursor, or state file — which matters because
watermarks desynchronise whenever a source is edited out of order, and every
one of these sources is a file a human or an agent may rewrite at any time.

**Deliberately not positional.** File offsets and line numbers are not stable
keys: markdown memory files get reordered and rewritten constantly, and the
first reflow would re-import an entire file as new.

**Consequence, decided rather than discovered.** A source memory *edited in
place* changes its hash and therefore imports as a **new** journal entry rather
than updating the old one. Given the append-only journal and facet supersession
this is correct — the edit is a new assertion, and supersession resolves it.
The alternative, fuzzy-matching edits back onto existing entries, is a
meaningfully harder system with a false-merge failure mode. Recorded as D-016
in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

### Provenance on import

Imported entries carry their origin so an operator can tell earned memory from
migrated memory:

| Field | Source |
|---|---|
| `source_harness` | Adapter `harness_id` |
| `source_path` | Concrete resolved path |
| `imported_at` | Sweep timestamp |
| `origin_session` | Where the harness records it — Claude Code memory files carry `metadata.originSessionId` in frontmatter |

### Import does not auto-pin

Facets are assigned by classification on import, but **promotion to pinned is
operator-confirmed**. A misclassified standing rule is this scenario's
highest-consequence error, and an import sweep is exactly the circumstance in
which a large batch of unreviewed classifications would otherwise be trusted at
once.

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Journal entry, facet text, embedding | None. | Permanent. | None — permanence is the scenario's central invariant. |
| Facet assignment, supersession / expiry mark | None. Corrections are additive. | Permanent, with history. | None. |
| Pin state | Operator unpin, renewal, or a review date lapsing without reconfirmation. | Until unpinned or lapsed. | The standing-rule resident budget is seeded in `facet_policies`; review renewal updates `review_at` without duplicating the journal entry. |
| Pin review / merge proposal | Operator resolves or dismisses it. | Until resolved. | None. |
| Summary node, tree edge | Forest rebuild. | Rebuildable cache; safe to drop and recompute from the journal. | None. |
| Projection state | Overwritten on each refresh. | Latest only. | None — the projected file is a generated artifact, not data. |

No deletion path in this table reaches a journal entry. Unpinning, lapsing, and
merging all operate on `facets` state; the memory itself remains and stays
retrievable by `recall`.

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
