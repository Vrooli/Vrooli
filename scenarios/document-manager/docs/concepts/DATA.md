# Data — Document Manager

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

Two stores, with different guarantees:

1. **SQLite** (via `modernc.org/sqlite`) holds metadata, derivations,
   anchors, enrichment records, sensitivity decisions, custody records
   and collections. The lifecycle sets `SQLITE_PATH` through
   `.vrooli/service.json`, and the API applies schemas on startup
   through `api-core/database`.
2. **The artifact store** holds opaque bytes: original documents and
   derivation outputs too large for a row. It is filesystem-backed and
   reached **only** through the routed file-store seam
   (`filerouting.RoutedRoots`, picked per request), so
   `X-Vrooli-Test-Mode` isolation is honored without any domain
   retaining a startup root string. Registered with `storage-manager` as
   declared kinds with explicit budgets.

No external storage resource is declared. Embeddings live in SQLite
beside the units they describe, and the `retrieval` domain queries them.
That is deliberate and is an established pattern here — **nine**
scenarios declare embedding tables in their own schema
(`agent-metareasoning-manager`, `api-library`, `audio-tools`,
`calendar`, `prompt-injection-arena`, `task-planner`, `text-tools`,
`vrooli-assistant`, `vrooli-memory`), with `calendar.event_embeddings`
and `task-planner.embedding_metadata` as the worked examples. A further
~38 scenarios touch embeddings in Go without owning a table; do not
conflate the two counts.

Vectors are stored as little-endian `float32` blobs with a leading
dimension header, following `vrooli-memory/api/internal/vector/codec.go`.
Blob storage rather than JSON is not an optimisation detail — it is what
kept that scenario's 8,181 vectors at 32 MB instead of 125 MB.

**What this store must never hold is ledger content.** Sources and
findings are different content classes: sources are permanent and never
compacted, findings are a stream that folds into summaries so ambient
attention stays bounded. Each needs a store that fits it, and each has
exclusive authority over its own class. Duplication would mean *the same
content indexed twice with no defined authority* — which is what a
`retrieval` index over ledger entries would create. Cross-corpus queries
go through `search-hub`. See
[`../internal/DECISIONS.md`](../internal/DECISIONS.md).

## Two Append-Only Surfaces

Most tables are ordinary mutable rows. Two are not, and the distinction
is a product guarantee rather than an implementation preference:

- **`custody_records`** — append-only. No repository method issues
  `UPDATE` or `DELETE`. Deleting a document writes a tombstone; it never
  erases the trail. This is `DOC-P0-015`.
- **`derivation_versions`** — append-only. Re-deriving a document mints
  a new version; it never rewrites an old one, so anchors minted against
  an earlier version keep resolving. This is `DOC-P0-007` and
  `DOC-P0-009`.

## Data Ownership

| Data | Owning Domain | Storage | Source Of Truth | Retention | Notes |
|---|---|---|---|---|---|
| Sources | intake | SQLite | `api/internal/intake/schema.sql` | Until removed by the operator | Watched directories, connector configs. No credentials stored inline. |
| Documents | intake | SQLite | `api/internal/intake/schema.sql` | Until deleted; deletion tombstones custody | Content address is the identity. Dedupe is by hash. |
| Original bytes | intake | Artifact store | Blob written through the routed seam | **`regenerable: false`** — never pruned by budget pressure | Losing these loses the document. Budget pressure must fail loudly rather than prune. |
| Derivations | derivation | SQLite | `api/internal/derivation/schema.sql` | Follows the parent document | One row per (document, version). |
| Derivation versions | derivation | SQLite (append-only) | `api/internal/derivation/schema.sql` | Follows the parent document | Monotonic. Never rewritten. |
| Parse outputs | derivation | Artifact store | Blob written through the routed seam | **`regenerable: true`** — prunable; re-derivable from original bytes | The safe thing to evict under budget pressure. |
| Units | anchors | SQLite | `api/internal/anchors/schema.sql` | Follows the parent derivation version | The citable fragment. |
| Anchors | anchors | SQLite | `api/internal/anchors/schema.sql` | Follows the parent unit | Carries `kind` from the first migration. **`geometric`** = page + bbox in the original document's coordinate space; durable across every derivation because the original bytes are `regenerable: false`. **`logical`** = structural path + char offset; covers every format but needs an alignment to cross a version. |
| Anchor alignments | anchors | SQLite | `api/internal/anchors/schema.sql` | Follows the newer derivation version | Maps `logical` anchors from version N to N+1, computed at re-derivation. Absent alignment, an old `logical` anchor resolves to its minting version or reports `unresolved` — never silently to the wrong region. |
| Retrieval index | retrieval | SQLite | `api/internal/retrieval/schema.sql` | Rebuildable at any time | Derived state only — a cache over units, anchors and embeddings. Registered as a `regenerable: true` kind; losing it costs a rebuild, never data. Two halves: an **FTS5 virtual table** over unit text for the lexical side, and the `enrichment` vector blobs scanned in-process for the semantic side. See the retrieval-mechanism decision in [`../internal/DECISIONS.md`](../internal/DECISIONS.md). |
| Enrichments | enrichment | SQLite | `api/internal/enrichment/schema.sql` | Follows the parent unit | Summaries, entities, claims, extraction results. |
| Embeddings | enrichment | SQLite | `api/internal/enrichment/schema.sql` | Follows the parent unit; invalidated by intentional retarget | **Every row carries role, model, dimension, content-version and retarget strategy.** Missing metadata is an `ai-conformance` ERROR. |
| Extraction schemas | enrichment | SQLite | `api/internal/enrichment/schema.sql` | Until removed by the operator | Caller-supplied shapes for schema-first extraction. |
| Detections | sensitivity | SQLite | `api/internal/sensitivity/schema.sql` | Follows the parent document | PII/PHI findings, categorized by regulation. |
| Classifications | sensitivity | SQLite | `api/internal/sensitivity/schema.sql` | Follows the parent document | The privacy class that selects the routing profile. |
| Redactions + manifests | sensitivity | SQLite | `api/internal/sensitivity/schema.sql` | Retained for the audit lifetime, not the document lifetime | Actor, timestamp and regulatory basis per redaction. Outlives the document deliberately. |
| Custody records | custody | SQLite (append-only) | `api/internal/custody/schema.sql` | Retained beyond document deletion | Assembled from AI Gateway `RouteEvidence` plus local step records. |
| Access events | custody | SQLite (append-only) | `api/internal/custody/schema.sql` | Retained beyond document deletion | Who read what, when. |
| Legal holds | custody | SQLite | `api/internal/custody/schema.sql` | Until explicitly released; release is itself audited | Suppresses all prune paths. |
| Collections | corpus | SQLite | `api/internal/corpus/schema.sql` | Until deleted | Membership is many-to-many with documents. Carries `default_privacy_class`, inherited by documents on intake; a document may be classified more restrictively, never less. |
| Retention state | corpus | SQLite | `api/internal/corpus/schema.sql` | Operational | Mirrors `storage-manager` kind declarations. |
| Publications | handoff | SQLite | `api/internal/handoff/schema.sql` | Follows the cited anchor, not the unit | One row per published *finding*. Carries the anchor URI it cites plus an idempotency key matching the ledger's composite import key. Never holds unit text. |
| Scope bindings | handoff | SQLite | `api/internal/handoff/schema.sql` | Until unbound | Which collection publishes into which ledger scope. |

## Schema Map

| Table/File/Object | Owner | Defined In | Used By |
|---|---|---|---|
| intake tables | intake | `api/internal/intake/schema.sql` | intake repository/service/handlers |
| derivation tables | derivation | `api/internal/derivation/schema.sql` | derivation repository/service/handlers |
| anchors tables | anchors | `api/internal/anchors/schema.sql` | anchors repository/service/handlers |
| enrichment tables | enrichment | `api/internal/enrichment/schema.sql` | enrichment repository/service/handlers |
| sensitivity tables | sensitivity | `api/internal/sensitivity/schema.sql` | sensitivity repository/service/handlers |
| custody tables | custody | `api/internal/custody/schema.sql` | custody repository/service/handlers |
| corpus tables | corpus | `api/internal/corpus/schema.sql` | corpus repository/service/handlers |
| retrieval tables | retrieval | `api/internal/retrieval/schema.sql` | retrieval repository/service/handlers |
| handoff tables | handoff | `api/internal/handoff/schema.sql` | handoff repository/service/handlers |
| artifact store layout | corpus | `api/internal/corpus/artifactstore.go` | intake, derivation, corpus (through the routed seam only) |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

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

Idempotent schema bootstrap: domain schema files use
`CREATE TABLE IF NOT EXISTS` and live beside the code that interprets
them. Declaring a new column is sufficient — `api-core` adds declared
columns to existing databases before boot, so a column addition needs no
hand-written migration.

Two changes are *not* ordinary migrations and need a recorded decision in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md) first:

- **Changing the anchor format.** Existing anchors must keep resolving,
  so a format change is an additive new anchor kind with a resolver that
  understands both — never an in-place rewrite.
- **Retargeting embeddings.** Changing embedding role, model or
  dimension invalidates stored vectors. This is a deliberate, planned
  migration recorded per the retarget strategy on each row, never a
  silent consequence of a model upgrade.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Full corpus export | Open, documented archive: original bytes, derivations, units, anchors, enrichments and custody records | corpus | `DOC-P0-017` — required for viability. No proprietary container, no lock-in. |
| Residency attestation | Human-readable signed report per document or collection | custody | `DOC-P1-013`. The artifact a compliance reviewer accepts. |
| Redaction manifest | Structured record: original version, redacted version, per-redaction actor/time/basis | sensitivity | `DOC-P1-011`. |
| Ledger publication | Scoped, idempotent append of a *finding* — a small entry citing an anchor URI as provenance. **Never a unit, and never unit text.** | handoff | `DOC-P1-020` (publish) and `DOC-P1-023` (idempotency). Not an export format — a live, optional seam. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Original bytes | Explicit document delete, never budget pressure | `regenerable: false`; budget pressure must fail loudly | Depends on `storage-manager` honoring the flag. A known upstream issue: its enforcer has pruned `regenerable: false` entries elsewhere — verify before relying on it. |
| Parse outputs | Budget pressure or explicit prune | `regenerable: true`; re-derivable from original bytes | The intended eviction target — but only safe because anchor resolution must not depend on it. `geometric` anchors resolve against the original bytes, so they are unaffected. `logical` anchors resolve through stored alignments, which live in SQLite rather than the artifact store. A resolution path that reads a pruned parse output is a bug, and there is a test for it. |
| Retrieval index | Budget pressure or explicit rebuild | `regenerable: true`; rebuildable from units, anchors and embeddings | None. Losing it costs a rebuild, never data. |
| Derivation versions | Parent document delete only | Append-only while the document lives | None. |
| Custody records | Never by document delete | Outlive the document; deletion writes a tombstone | Two gaps. (1) Long-horizon retention for the journal itself is undecided. (2) **A known upstream trap:** `storage-manager`'s enforcer prunes any `kind=dir` entry that carries a budget, regardless of append-only intent — `vrooli-memory`'s append-only journal declared `max_age 365d` and became prune-eligible exactly this way. Declare the custody kind without a prunable budget, and verify enforcement behavior before trusting it. |
| Redaction manifests | Never by document delete | Audit lifetime, not document lifetime | Same. |
| Anything under legal hold | Nothing | Hold suppresses every prune path until explicitly released | Release must itself be audited. |

## Privacy Notes

This scenario is designed to hold regulated data — PHI, privileged legal
material, personal information — and its handling is a product feature
rather than a caveat:

- Every document carries a `PrivacyClass` (public / internal /
  confidential / secret) that **selects the AI Gateway routing profile**.
  Confidential and secret documents route `privacy-sensitive`, which
  fails closed rather than falling back to a remote provider.
- The sensitivity gate resolves **before** tier selection, so a
  confidential document is never eligible for the remote tier in the
  first place.
- Detected PII/PHI is categorized by regulation, and redaction requires
  explicit human confirmation before finalize — automated redaction
  without review is not defensible.
- Only redacted units may reach a shared ledger scope.

See [`../internal/SECURITY.md`](../internal/SECURITY.md) and
[`DOMAINS.md`](DOMAINS.md#the-ordering-rule).

## Cross-References

- [`DOMAINS.md`](DOMAINS.md) — data ownership by domain
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — external resources and scenarios
- [`../reference/configuration.md`](../reference/configuration.md) — runtime configuration
- [`../internal/SECURITY.md`](../internal/SECURITY.md) — privacy/security posture
