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

Two more join them when the write spine lands (`spec_versions` and
`render_versions`), for the mirrored reason described next.

## The Authority Mirror (write spine, P2)

Read-side authority runs **bytes → derivations**. Write-side authority
runs **spec → renders**, and the two are deliberately the same shape:

| | Read spine | Write spine |
|---|---|---|
| Authority (`regenerable: false`) | Original bytes | The **spec** |
| Derived, versioned (`regenerable: true`) | Derivations / parse outputs | **Render versions** / rendered bytes |
| Append-only version table | `derivation_versions` | `spec_versions`, `render_versions` |
| What triggers a new version | A better parser | A different template, or a source refresh |
| Bulk sweep | Re-derive on parser upgrade (`DOC-P1-003`) | Re-render on template change (`DOC-P2-014`) |

Because the shapes match, the write spine needs no new mechanism for
versioning, diffing or eviction. It also means the safe eviction target
is the same on both sides: **rendered bytes are prunable and
re-producible from the spec**, exactly as parse outputs are prunable and
re-derivable from the original bytes.

One consequence is easy to miss and load-bearing: a rendered artifact is
also **ingested back** through `intake`, so it acquires original bytes of
its own that *are* `regenerable: false`. A generated document therefore
has two authorities at different layers — the spec that explains it, and
the bytes someone was actually shown. Both are kept, deliberately: the
spec is what you edit, the bytes are what you cited.

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
| Anchors | anchors | SQLite | `api/internal/anchors/schema.sql` | Follows the parent unit | Carries `kind` from the first migration — a **three-value** enum. **`geometric`** = page + bbox in the original document's coordinate space; durable across every derivation because the original bytes are `regenerable: false`. **`tabular`** = sheet/table identity + cell range (`sheet:2!B4:D9`); equally durable, because cell coordinates are intrinsic to the source rather than produced by the parser. **`logical`** = structural path + char offset; covers every flowing-text format but needs an alignment to cross a version. The kind a unit carries is what its **handler chain could actually prove**, not what its format could support at best — a PDF parsed without its geometry handler records `logical`. The best available kind per format is in [`../reference/format-matrix.md`](../reference/format-matrix.md). |
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
| Collections | corpus | SQLite | `api/internal/corpus/schema.sql` | Until deleted | Membership is many-to-many with documents. Carries `default_privacy_class`, inherited by documents on intake; a document may be classified more restrictively, never less. Also carries `federated` (default **false**) — the per-collection opt-in that governs whether its units answer search-hub queries, since the federation contract carries no caller identity to filter on. The flag is not sufficient on its own: confidential and secret units never federate regardless of it. |
| Retention state | corpus | SQLite | `api/internal/corpus/schema.sql` | Operational | Mirrors `storage-manager` kind declarations. |
| Publications | handoff | SQLite | `api/internal/handoff/schema.sql` | Follows the cited anchor, not the unit | One row per published *finding*. Carries the canonical anchor URI it cites ([`../reference/anchor-uri.md`](../reference/anchor-uri.md)) plus the finding-body hash; together with `runtime` these form the ledger's `(scope, import_key)` composite. Never holds unit text. **A published URI is never rewritten** — re-derivation propagation appends a new entry rather than updating an old one. |
| Scope bindings | handoff | SQLite | `api/internal/handoff/schema.sql` | Until unbound | Which collection publishes into which ledger scope. |
| Templates | templates | SQLite + corpus document | `api/internal/templates/schema.sql` | Until deleted; a template in use cannot be deleted, only superseded | **P2.** The row is metadata; the template body is stored as a corpus document under a distinguished kind, so it inherits versioning, custody, export and diff. Excluded from `retrieval` by kind, not by a parallel store. |
| Template versions | templates | SQLite (append-only) | `api/internal/templates/schema.sql` | Follows the parent template | **P2.** A confirmed template edit mints a version; documents pin the version they rendered under, so an edit never silently restyles history. |
| Fidelity declarations | templates | SQLite | `api/internal/templates/schema.sql` | Follows the parent template version | **P2.** One row per (template version, render target). A target with no declaration is `no_renderer_for_target` rather than a best-effort attempt. |
| Template proposals | templates | SQLite | `api/internal/templates/schema.sql` | Until confirmed or rejected; the decision is audited | **P2.** A template edit is a proposal because its blast radius is every document using it. A per-document override is not — it applies directly. |
| Specs | composition | SQLite | `api/internal/composition/schema.sql` | Until deleted | **P2. `regenerable: false` — this is the write spine's authority.** Losing a spec loses the document's editability, not merely its bytes. |
| Spec versions | composition | SQLite (append-only) | `api/internal/composition/schema.sql` | Follows the parent spec | **P2.** Minted by an edit or a `refresh`, never by a `render`. Append-only is what makes revert-to-version *be* undo, so no separate undo stack exists. |
| Blocks | composition | SQLite | `api/internal/composition/schema.sql` | Follows the parent spec version | **P2.** Carries `provenance: sourced \| authored`. Unsourced content is allowed and always visible as such; "every claim cited" is a per-template policy, never a global rule. |
| Source bindings | composition | SQLite | `api/internal/composition/schema.sql` | Follows the parent block | **P2.** A *re-runnable* descriptor — a command-center query, a corpus anchor, a chart-generator render id — not a captured value. This is what makes `refresh` possible at all. |
| Source resolutions | composition | SQLite (append-only) | `api/internal/composition/schema.sql` | Follows the binding | **P2.** Each resolution snapshotted with its timestamp, so a rendered document stays reproducible even though its binding is live. Bindings without snapshots would make every past render unexplainable. |
| Overrides | composition | SQLite | `api/internal/composition/schema.sql` | Follows the parent spec | **P2.** Explicit and **enumerable** — "this deck deviates in 4 ways" — and surfaced at switch time with which ones the target template cannot honor. Every override is a bet against future template changes. |
| Render versions | render | SQLite (append-only) | `api/internal/render/schema.sql` | Follows the parent spec | **P2.** One row per (spec version, template version, target). Records the renderer chain and its versions, so re-rendering under a better renderer is a query rather than a guess — the same property that makes `DOC-P1-003` mechanical. |
| Rendered bytes | render | Artifact store | Blob written through the routed seam | **`regenerable: true`** — prunable; re-producible from the spec | **P2.** The write-side twin of parse outputs. Note the rendered artifact is also ingested back through `intake`, where it acquires `regenerable: false` original bytes of its own. |
| Block alignments | render | SQLite | `api/internal/render/schema.sql` | Follows the render version | **P2.** The renderer's spec-block → output-region mapping, emitted as a byproduct because the renderer *placed* each element. This is what makes a generated document's `logical` anchors durable **without** a computed alignment map — the read side's hardest case, solved by construction on the write side. |

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
| templates tables | templates | `api/internal/templates/schema.sql` | templates repository/service/handlers (P2) |
| composition tables | composition | `api/internal/composition/schema.sql` | composition repository/service/handlers (P2) |
| render tables | render | `api/internal/render/schema.sql` | render repository/service/handlers (P2) |
| renderer registry | render | `api/internal/render/registry.json` | render router; projected into `docs/reference/render-matrix.md` with a test asserting agreement (P2) |
| artifact store layout | corpus | `api/internal/corpus/artifactstore.go` | intake, derivation, render, corpus (through the routed seam only) |
| system schema | infrastructure | `api/internal/database/system.sql` | API boot and cross-cutting DB setup |

## Migrations And Compatibility

Idempotent schema bootstrap: domain schema files use
`CREATE TABLE IF NOT EXISTS` and live beside the code that interprets
them. Declaring a new column is sufficient — `api-core` adds declared
columns to existing databases before boot, so a column addition needs no
hand-written migration.

Three changes are *not* ordinary migrations and need a recorded decision
in [`../internal/DECISIONS.md`](../internal/DECISIONS.md) first:

- **Changing the anchor format.** Existing anchors must keep resolving,
  so a format change is an additive new anchor kind with a resolver that
  understands both — never an in-place rewrite.
- **Retargeting embeddings.** Changing embedding role, model or
  dimension invalidates stored vectors. This is a deliberate, planned
  migration recorded per the retarget strategy on each row, never a
  silent consequence of a model upgrade.
- **Changing the spec schema (P2).** A spec is the write spine's
  authority, so an existing spec must keep rendering. A schema change is
  additive with a version discriminator on the spec row — never an
  in-place rewrite of stored specs, for the same reason anchors are never
  rewritten in place. A spec that can no longer be rendered is a
  destroyed document, not a migration cost.

## Import / Export

| Path | Format | Owner | Status |
|---|---|---|---|
| Full corpus export | Open, documented archive: original bytes, derivations, units, anchors, enrichments and custody records | corpus | `DOC-P0-017` — required for viability. No proprietary container, no lock-in. |
| Residency attestation | Human-readable signed report per document or collection | custody | `DOC-P1-013`. The artifact a compliance reviewer accepts. |
| Redaction manifest | Structured record: original version, redacted version, per-redaction actor/time/basis | sensitivity | `DOC-P1-011`. |
| Ledger publication | Scoped, idempotent append of a *finding* — a small entry whose `ImportProvenance.source_locator` is a canonical anchor URI. **Never a unit, and never unit text.** | handoff | `DOC-P1-020` (publish) and `DOC-P1-023` (idempotency), over the URI specified by `DOC-P0-028`. Not an export format — a live, optional seam. |
| Anchor citation | A `vrooli-anchor:` URI: content address, derivation version, anchor kind, coordinates | anchors | `DOC-P0-028`. The only identifier this scenario emits that is designed to outlive its own database, so it carries no database identity, no corpus identity and no content-derived token. Grammar and canonical form: [`../reference/anchor-uri.md`](../reference/anchor-uri.md). |
| Rendered document | Bytes in a declared render target (`.pptx`, `.docx`, `.xlsx`, PDF, …) | render | **P2**, `DOC-P2-012`. Not an escape hatch from `DOC-P0-017`: the corpus export still carries specs, templates and render versions, so a corpus round-trips without a renderer present. |
| Spec + template bundle | The three authoring layers plus the template version a document rendered under | composition + templates | **P2**, `DOC-P2-010`, `DOC-P2-011`. Exporting rendered bytes alone would be lock-in of the worst kind — the artifact without the thing that can regenerate or restyle it. |

## Retention And Deletion

| Data | Delete Trigger | Retention Rule | Current Gap |
|---|---|---|---|
| Original bytes | Explicit document delete, never budget pressure | `regenerable: false`; budget pressure must fail loudly | Depends on `storage-manager` honoring the flag. A known upstream issue: its enforcer has pruned `regenerable: false` entries elsewhere — verify before relying on it. |
| Parse outputs | Budget pressure or explicit prune | `regenerable: true`; re-derivable from original bytes | The intended eviction target — but only safe because anchor resolution must not depend on it. `geometric` anchors resolve against the original bytes, so they are unaffected. `logical` anchors resolve through stored alignments, which live in SQLite rather than the artifact store. A resolution path that reads a pruned parse output is a bug, and there is a test for it. |
| Retrieval index | Budget pressure or explicit rebuild | `regenerable: true`; rebuildable from units, anchors and embeddings | None. Losing it costs a rebuild, never data. |
| Derivation versions | Parent document delete only | Append-only while the document lives | None. |
| Custody records | Never by document delete | Outlive the document; deletion writes a tombstone | Two gaps. (1) Long-horizon retention for the journal itself is undecided. (2) **A known upstream trap:** `storage-manager`'s enforcer prunes any `kind=dir` entry that carries a budget, regardless of append-only intent — `vrooli-memory`'s append-only journal declared `max_age 365d` and became prune-eligible exactly this way. Declare the custody kind without a prunable budget, and verify enforcement behavior before trusting it. |
| Redaction manifests | Never by document delete | Audit lifetime, not document lifetime | Same. |
| Specs (P2) | Explicit document delete, never budget pressure | `regenerable: false`; budget pressure must fail loudly | Same upstream risk as original bytes — `storage-manager`'s enforcer has pruned `regenerable: false` entries elsewhere. A lost spec is a document that can no longer be restyled or refreshed, only re-authored. |
| Rendered bytes (P2) | Budget pressure or explicit prune | `regenerable: true`; re-producible from the spec plus its pinned template version | Safe **only** because the template version is pinned per render version. Pruning bytes whose template version was deleted makes them unreproducible, so a template in use must not be deletable — only supersedable. |
| Spec / render versions (P2) | Parent spec delete only | Append-only while the spec lives | Same growth-versus-history trade the read spine already accepted. Prune rendered bytes before considering version collapse. |
| Block alignments (P2) | Follows the render version | Kept while any anchor references that render version | Deleting an alignment breaks the generated document's durable-anchor guarantee, which is the write spine's equivalent of resolving against a pruned parse output. |
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
