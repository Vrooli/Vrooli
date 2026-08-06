# Domains — Document Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Document Manager has nine product domains plus the scaffold `health`
domain. Each product domain owns exactly one noun. A file enters at
`intake`, is judged for exposure in `sensitivity`, becomes structure in
`derivation`, becomes citable in `anchors`, gains meaning in
`enrichment`, becomes findable through `retrieval`, is accounted for in
`custody`, is held in `corpus`, and — optionally — is cited from
elsewhere through `handoff`.

Two boundaries in that list are load-bearing and are argued in
[`../internal/DECISIONS.md`](../internal/DECISIONS.md): `sensitivity`
resolves *before* `derivation` picks a tier, and `retrieval` answers over
**this** corpus only. `handoff` is optional — the scenario is fully
useful with it absent.

The scaffold also ships one clearly fenced worked example domain (never
product scope) as a copyable reference; `template-manager detemplate
document-manager` removes every fenced example once the real domains are
green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## The Ordering Rule

One ordering decision is load-bearing and must not be rearranged:
**`sensitivity` resolves before `derivation` selects a tier.** A
document's privacy class determines its routing profile, so a
confidential document is never *eligible* for the remote tier rather
than being blocked from it after the fact. Reversing this turns a
structural residency guarantee into an advisory one, which is the entire
product claim.

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/document-manager/v1/shared/health.proto` |
| intake | Accept bytes from every source and decide what they are. | Give every document one stable identity so nothing is stored or parsed twice. | Sources, blobs, content addresses, type verdicts. | ingestion | crud | Source, Blob, ContentAddress, TypeVerdict | `api/internal/intake/` |
| derivation | Route a document through the cheapest correct parse tier and version the result. | Turn bytes into one normalized structure without ever discarding a prior reading. | Derivations, tier decisions, normalized document models. | pipeline | service | Derivation, Tier, DocumentModel, DerivationVersion | `api/internal/derivation/` |
| anchors | Segment a derivation into citable units bound to source geometry. | Make every claim traceable to an exact region of an exact document. | Units, anchors, anchor-resolution index. | projection | query | Unit, Anchor, SourceRegion | `api/internal/anchors/` |
| enrichment | Add meaning to units through governed inference. | Produce summaries, entities, structured extractions, and embeddings callers can trust. | Enrichments, embeddings, extraction schemas. | service | pipeline | Enrichment, Embedding, ExtractionSchema | `api/internal/enrichment/` |
| retrieval | Answer questions over this corpus and return anchored units. | Make the corpus queryable without becoming a second ledger. | Query plans, result rankings, the retrieval index. | query | service | Query, Result, RetrievalIndex | `api/internal/retrieval/` |
| sensitivity | Detect exposure risk and decide how a document may be processed. | Make residency structural: privacy class selects the routing profile. | Detections, privacy classifications, redactions, redaction manifests. | policy | service | PrivacyClass, Detection, Redaction, RedactionManifest | `api/internal/sensitivity/` |
| custody | Record what happened to a document, where, and under whose authority. | Produce the evidence a compliance reviewer accepts. | Custody records, receipts, access events, legal holds. | ledger | reporting | Receipt, CustodyRecord, Attestation, LegalHold | `api/internal/custody/` |
| corpus | Hold collections, govern the artifact store, and let everything leave. | Keep the corpus durable, bounded, and portable. | Collections, membership, retention state, exports. | crud | service | Collection, ArtifactStore, Export | `api/internal/corpus/` |
| handoff | Publish *findings* — never units — into a declared ledger scope. | Let a consumer cite this corpus from a ledger without either system depending on the other. | Publication state, scope bindings, sync cursors. | integration | service | Publication, ScopeBinding, Finding | `api/internal/handoff/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary
upload exception. Copy its shape for your own domains, then remove it.

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| notes | Provide the worked CRUD reference with attachment upload exception. | Demonstrate the expected vertical slice for a real domain. | Notes and attachment metadata. | crud | service | Note, Attachment | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/document-manager/v1/notes/` |

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live
  backend state.
- Primary archetype: reporting / query.
- Secondary traits: operational health.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific
  domain behavior.
- API: `api/handlers/health/`.
- CLI: built-in `status` command is provided through cli-core.
- UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### intake

- Purpose: accept bytes from every source and give each document one
  stable identity before anything expensive happens.
- Primary archetype: ingestion.
- Secondary traits: CRUD over sources, idempotency.
- Owns: source registration, blob writes to the artifact store, content
  addressing (hash), exact dedupe, near-duplicate detection, MIME
  sniffing, the text-native versus scanned verdict that drives all
  downstream cost, and applying the collection's default privacy class
  to each arriving document.
- Does not own: parsing (that is `derivation`), byte-level file
  utilities (that is `file-tools`), or retention (that is `corpus`).
- **Two kinds of duplicate, and only one is a hash question.** Content
  addressing catches byte-identical files. It does not catch two copies
  of the same paper from different sources, which differ in metadata,
  compression or a watermark. That second case is caught by comparing
  embeddings, and it is one of the two declared readers of the
  `enrichment` vector store.
- API: `api/internal/intake/`, `api/handlers/intake/`.
- CLI: `cli/domains/intake/` — `ingest`, `watch`, `sources`.
- UI: intake drop zone and source configuration within the Corpus
  surface.
- Storage: `sources`, `blobs`, `documents` tables; raw bytes in the
  artifact store under a `regenerable: false` storage kind.
- Requirements: `DOC-P0-001`, `DOC-P0-002`, `DOC-P0-003`,
  `DOC-P0-025`, `DOC-P1-016`, `DOC-P1-017`.
- Tests: hash stability and dedupe under repeat submission, MIME
  sniffing against a fixture corpus, scanned-versus-native verdict
  accuracy, watched-directory idempotency.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md).

### derivation

- Purpose: route each document through the cheapest tier that is
  actually correct, and record the result as a version rather than a
  replacement.
- Primary archetype: pipeline.
- Secondary traits: routing policy, versioning.
- Owns: the tier router (T1 native, T2 structural, T3 vision), the
  normalized document model every tier emits, derivation versions, tier
  and confidence metadata, and re-derivation.
- Does not own: the parsing runtimes themselves (resources own those),
  the decision about whether a document *may* use a remote tier (that is
  `sensitivity`), or unit segmentation (that is `anchors`).
- API: `api/internal/derivation/`, `api/handlers/derivation/`.
- CLI: `cli/domains/derivation/` — `derive`, `rederive`, `versions`,
  `diff`.
- UI: tier and confidence badges in the Reader; re-derivation controls.
- Storage: `derivations`, `derivation_versions`, `tier_decisions`;
  parse outputs in the artifact store under a `regenerable: true`
  storage kind.
- Requirements: `DOC-P0-004`, `DOC-P0-005`, `DOC-P0-006`,
  `DOC-P0-007`, `DOC-P1-001`, `DOC-P1-002`, `DOC-P1-003`.
- Tests: tier-selection table tests, normalized-model equivalence
  across tiers, version monotonicity, re-derivation never mutating a
  prior version, escalation on low confidence.
- Related docs: [`FLOWS.md`](FLOWS.md), [`INTEGRATIONS.md`](INTEGRATIONS.md).

### anchors

- Purpose: make every derived unit point back to an exact region of an
  exact document, and keep doing so after the document is re-derived.
- Primary archetype: projection.
- Secondary traits: query.
- Owns: unit segmentation, anchor minting, anchor resolution, and the
  alignment maps that carry `logical` anchors across derivation
  versions.
- Does not own: what a unit *means* (that is `enrichment`), how it is
  found (that is `retrieval`), or where it ends up (that is `handoff`).
- **Two anchor kinds, and the distinction is load-bearing:**
  - `geometric` — page plus bounding box in the *original document's*
    coordinate space. Durable across every derivation without any
    further work, because it is a property of the original bytes, which
    are `regenerable: false`. Available only where the tier supplies
    geometry: PDF and image sources.
  - `logical` — a structural path plus character offset. Covers every
    format including DOCX, HTML, EPUB and plain text, but is *not*
    inherently stable across versions: a better parser changes
    whitespace, reading order and table handling, so the same offsets
    mean different things. Carrying a `logical` anchor to a new version
    requires an alignment map computed at re-derivation time.
  - `DOC-P0-009`'s guarantee therefore holds unconditionally for
    `geometric` anchors, and for `logical` anchors only through a
    recorded alignment. An unaligned `logical` anchor resolves to its
    minting version or reports `unresolved` — it never silently returns
    the wrong region.
- API: `api/internal/anchors/`, `api/handlers/anchors/`.
- CLI: `cli/domains/anchors/` — `units`, `resolve`, `cite`.
- UI: the Reader's bidirectional highlight — select a unit to highlight
  its source region, select a region to scroll to its unit.
- Storage: `units`, `anchors` (with `kind` from the first migration),
  `anchor_alignments`; an index supporting resolution by
  (document, version, offset) and by (document, page, bbox).
- Requirements: `DOC-P0-008`, `DOC-P0-009`, `DOC-P0-019`.
- Tests: anchor round-trip resolution per kind, cross-version resolution
  (a v1 `geometric` anchor resolves after v2 with no alignment; a v1
  `logical` anchor resolves only with one and reports `unresolved`
  without), boundary cases at page and block edges, and an assertion
  that resolution never depends on a pruned `regenerable: true` parse
  output.
- Related docs: [`DATA.md`](DATA.md),
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md).

### enrichment

- Purpose: add meaning to units through inference that is always
  governed by AI Gateway.
- Primary archetype: service.
- Secondary traits: pipeline, external integration.
- Owns: summaries, classification suggestions, entity and claim
  extraction, schema-first structured extraction, table and equation
  handling, embedding generation, and the embedding metadata contract.
- Does not own: provider selection, credentials, or model identity —
  all of that belongs to AI Gateway. It does not own *query* over the
  vectors it produces, which is `retrieval`. It does not own recall over
  any other scenario's corpus.
- **Embeddings have two declared readers, and naming them is a
  requirement rather than commentary.** An embedding table with no
  stated consumer is how a second corpus gets built by accident. The
  readers are `retrieval` (semantic query over units) and `intake`
  (near-duplicate detection, which catches what content hashing cannot —
  two copies of the same paper from different sources are not
  byte-identical). A finding published to a ledger scope is embedded
  *again* by the ledger in its own space; that duplication is correct
  and must not be optimised away by passing precomputed vectors across
  the boundary.
- API: `api/internal/enrichment/`, `api/handlers/enrichment/`.
- CLI: `cli/domains/enrichment/` — `enrich`, `extract`, `schema`.
- UI: per-unit summaries and extraction results in the Reader.
- Storage: `enrichments`, `embeddings`, `extraction_schemas`; every
  embedding row carries role, model, dimension, content-version and
  retarget strategy.
- Requirements: `DOC-P0-010`, `DOC-P0-011`, `DOC-P1-004`,
  `DOC-P1-005`, `DOC-P1-006`, `DOC-P1-007`, `DOC-P1-008`.
- Tests: gateway-only assertion (no direct provider HTTP anywhere in
  the tree), embedding metadata completeness, schema-extraction
  validation against the caller's schema, equation and table fidelity
  fixtures.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### retrieval

- Purpose: answer a question over *this* corpus and return anchored
  units, so a caller can cite what it found.
- Primary archetype: query.
- Secondary traits: service, ranking.
- Owns: hybrid semantic-plus-lexical query over units, filtering by
  collection and privacy class, ranking and reranking, result
  assembly, and the `search-hub` provider surface — declared in
  `.vrooli/search.json` and self-registered at boot through
  `packages/searchregister-go`, in a background goroutine that must
  never block or fail startup.
- Does not own: embedding *generation* (that is `enrichment`), anchor
  minting or resolution (that is `anchors`), and — emphatically —
  recall over any corpus but this one. **Querying the ledger is not
  this domain's job and never becomes it**; a caller that needs both
  goes through `search-hub`.
- Scope boundary: this domain exists because sources and findings are
  different content classes with different lifecycles. It is not a
  reintroduction of the retired scenario's `POST /api/search` over a
  private Qdrant collection — see the superseded decisions in
  [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
- API: `api/internal/retrieval/`, `api/handlers/retrieval/`.
- CLI: `cli/domains/retrieval/` — `query`, `similar`.
- UI: search within the Corpus surface; "find similar" from a unit in
  the Reader.
- Storage: `retrieval_index` derived state only. It is a cache over
  `units`, `anchors` and `embeddings` — rebuildable from them, and
  therefore a `regenerable: true` artifact-store kind.
- **Mechanism** (decided; see `DECISIONS.md`): FTS5 virtual table for the
  lexical half, in-process cosine over the SQLite-resident `float32`
  vector blobs for the semantic half, fused by reciprocal rank. Privacy
  class and collection membership narrow the candidate set **before**
  scoring, so `DOC-P0-024` is a precondition of the query rather than a
  filter applied to results. `resource-reranker` is an escalation, not a
  default, and being an inference call it routes through the
  `gatewayreq` choke point under the document's class.
- **Degraded mode.** Embedding generation belongs to `enrichment` and
  therefore to AI Gateway. When the gateway is unreachable, newly
  ingested units have no vectors, so semantic recall silently misses
  them. The lexical half must keep answering and the result must say the
  index is partial — which makes FTS5 load-bearing for *availability*,
  not merely for recall quality.
- Requirements: `DOC-P0-023`, `DOC-P0-024`, `DOC-P0-018`.
- Tests: privacy-class filtering (a confidential unit never surfaces in
  a query scoped to a collection the caller cannot read), every result
  carrying a resolvable anchor, ranking determinism for a fixed index,
  and `search-hub` provider contract conformance.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`DATA.md`](DATA.md).

### sensitivity

- Purpose: decide how exposed a document is, and let that decision —
  not a user preference — govern where it may be processed.
- Primary archetype: policy.
- Secondary traits: service, human-in-the-loop workflow.
- Owns: PII and PHI detection, categorization by regulation, privacy
  classification, the mapping from privacy class to AI Gateway routing
  profile, redaction proposals, redaction confirmation, and redaction
  manifests.
- Does not own: the redaction *review* UI (that is the UI layer) or
  provider selection within a profile (that is AI Gateway).
- **It does own enforcement, and this correction matters.** AI Gateway's
  fail-closed behavior is a property of the *profile*, not of the
  privacy class: `PROFILE_LOCAL_FIRST` is documented to fall back to a
  permitted remote provider, so a confidential document sent under the
  wrong profile routes remote and the gateway is behaving correctly. The
  guarantee therefore lives entirely in this domain's class→profile
  mapping. It is enforced at a single choke point — `internal/gatewayreq`,
  the only construction site for a `GatewayRequest` — which refuses to
  emit a profile weaker than the document's class. See the
  `GatewayRequest constructor` seam in
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
- API: `api/internal/sensitivity/`, `api/handlers/sensitivity/`.
- CLI: `cli/domains/sensitivity/` — `scan`, `classify`, `redact`.
- UI: the redaction review surface; privacy-class filters in the Corpus.
- Storage: `detections`, `classifications`, `redactions`,
  `redaction_manifests`.
- Requirements: `DOC-P0-012`, `DOC-P0-013`, `DOC-P1-009`,
  `DOC-P1-010`, `DOC-P1-011`, `DOC-P1-012`.
- Tests: a confidential document must fail closed rather than route
  remote (asserted as a failure, not skipped); an AST check asserting
  `internal/gatewayreq` is the only `GatewayRequest` construction site
  in the tree; a table test over every (class, profile) pair proving
  weaker profiles are rejected; redaction requires explicit
  confirmation; redacted derivatives only expose redacted units.
- Related docs: [`FLOWS.md`](FLOWS.md),
  [`../internal/SECURITY.md`](../internal/SECURITY.md).

### custody

- Purpose: produce the evidence that answers "what happened to this
  document, where did each step run, and who authorized it."
- Primary archetype: ledger.
- Secondary traits: reporting, access control.
- Owns: the append-only custody journal, per-document processing
  receipts assembled from AI Gateway `RouteEvidence`, access events,
  legal holds, and the exportable residency attestation.
- Does not own: routing decisions (it records them) or retention
  enforcement (that is `corpus` plus `storage-manager`).
- API: `api/internal/custody/`, `api/handlers/custody/`.
- CLI: `cli/domains/custody/` — `receipt`, `attest`, `audit`, `hold`.
- UI: the Receipt surface — a per-document timeline with a hard visual
  boundary between local and remote execution.
- Storage: `custody_records` (append-only; no UPDATE or DELETE path),
  `access_events`, `legal_holds`.
- Requirements: `DOC-P0-014`, `DOC-P0-015`, `DOC-P1-013`,
  `DOC-P1-014`, `DOC-P1-015`.
- Tests: append-only enforcement at the repository seam, receipt
  completeness for every inference step, tombstone-on-delete rather
  than trail erasure, attestation export determinism.
- Related docs: [`../internal/SECURITY.md`](../internal/SECURITY.md),
  [`INTEGRATIONS.md`](INTEGRATIONS.md).

### corpus

- Purpose: keep the corpus durable, bounded, and portable.
- Primary archetype: CRUD.
- Secondary traits: service, retention governance.
- Owns: collections and membership, **the default privacy class a
  collection confers on its documents**, the artifact-store layout,
  storage kind declarations and retention budgets, prune coordination,
  and full export.
- Does not own: retention *enforcement*, which is `storage-manager`'s;
  this domain declares kinds and budgets and honors holds.
- **A collection is the privacy boundary a user actually reasons about.**
  A legal matter or a patient file is confidential wholesale, so
  `collections.default_privacy_class` is set at creation and inherited
  on intake. A document may be classified *more* restrictively than its
  collection, never less. Classifying document-by-document alone invites
  the single mistake that breaks the residency claim, and that mistake
  is silent.
- API: `api/internal/corpus/`, `api/handlers/corpus/`.
- CLI: `cli/domains/corpus/` — `collections`, `export`, `prune`.
- UI: the Corpus surface — browse, filter by privacy class and
  confidence, and the low-confidence review queue.
- Storage: `collections` (carrying `default_privacy_class`),
  `collection_members`, `retention_state`; the artifact store reached
  only through the routed file-store seam.
- Requirements: `DOC-P0-016`, `DOC-P0-017`, `DOC-P0-025`, `DOC-P1-018`.
- Tests: export round-trip fidelity, retention declarations present for
  every kind, prune respecting legal hold and `regenerable: false`,
  inheritance of the collection default on intake, and rejection of a
  document classification weaker than its collection's default.
- Related docs: [`DATA.md`](DATA.md),
  [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md).

### handoff

- Purpose: let a caller record a *finding* about this corpus into a
  ledger scope, carrying an anchor as provenance.
- Primary archetype: integration.
- Secondary traits: service.
- **P1, not P0, and optional by construction.** The scenario is fully
  useful with this domain absent. It exists so a consumer — a research
  or investigation scenario — can write "this branch is ruled out,
  see anchor X" without either sibling importing the other.
- Owns: publication state, scope bindings, sync cursors, provenance
  construction from an anchor, and re-derivation propagation to
  already-published entries.
- Does not own: anything about how the ledger stores, compacts, or
  recalls. It also does not decide *what is worth recording* — that
  judgement belongs to the consumer.
- **Never publishes units.** A unit is source text; publishing every
  unit would flood a bounded-attention system with unbounded material,
  which is precisely what the ledger's compaction exists to prevent. A
  publication is one small entry citing an anchor. See the superseded
  decisions in [`../internal/DECISIONS.md`](../internal/DECISIONS.md).
- API: `api/internal/handoff/`, `api/handlers/handoff/`.
- CLI: `cli/domains/handoff/` — `publish`, `scope`, `sync`.
- UI: publication status per collection.
- Storage: `publications`, `scope_bindings`, `sync_cursors`.
- Requirements: `DOC-P1-020`, `DOC-P1-023`.
- Tests: idempotent republish producing no duplicates, provenance
  resolving back through `anchors`, behavior when the ledger scope does
  not yet exist, and an assertion that no publication payload carries
  full unit text.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`FLOWS.md`](FLOWS.md).

## Cross-Cutting Requirements

These are not domains; they are obligations every domain shares.

| Requirement | Obligation |
|---|---|
| `DOC-P0-020` | Every domain verb is a proto service method with a generated CLI wrapper. No hand-written second implementation. |
| `DOC-P0-021` | The Reader spans `anchors`, `derivation` and `enrichment`; it is owned by the UI layer, not by one domain. |
| `DOC-P0-022` | Lifecycle and test compliance across all phases, including `ai-conformance`. |
| `DOC-P0-026` | Every outbound `GatewayRequest` is built at the single `internal/gatewayreq` choke point, asserted by an AST check. Spans `enrichment`, `sensitivity` and `derivation` tier 3. |
| `DOC-P1-019` | Agent tools and widgets declared and discoverable via `cli-health`. |
| `DOC-P1-021`, `DOC-P1-022` | Metering through LPBS on the paid tier, with BYOK never charged. |

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Document | An immutable content-addressed byte sequence plus its identity. | `intake`. |
| Derivation | One versioned reading of a document produced by one tier. | `derivation`. |
| Unit | A citable fragment of a derivation carrying an anchor. | `anchors`. |
| Receipt | The record of where every step for one document executed. | `custody`. |
| Scope | The ledger partition units are published into. | `handoff` (defined by `vrooli-memory`). |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| `graph` (citation network) | Reference extraction (`DOC-P1-007`) produces the edges, but the traversal and visualization surface is P2 scope. | When `DOC-P2-004` is scheduled and reference extraction is proven accurate. |
| `annotation` (collaborative) | A genuinely different product with its own multi-user concerns. Deliberately last. | When `DOC-P2-009` is scheduled and multi-user access control exists beyond `DOC-P1-014`. |
| `transcripts` (audio/video) | A transcript is a document with time anchors instead of page anchors; it may fold into `intake` plus `anchors` rather than becoming its own domain. | When `DOC-P2-003` is scheduled — decide fold-in versus split at that point. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- Parsing runtimes — owned by resources (`unstructured-io`,
  pdf-inspector), never vendored into a domain.
- Provider clients — owned by AI Gateway. A provider HTTP client
  appearing in any domain is an `ai-conformance` ERROR finding.
- `api/internal/gatewayreq/` — the `GatewayRequest` constructor choke
  point. Infrastructure with one job; the *policy* it applies belongs to
  `sensitivity`.
- Recall over any other scenario's corpus — `retrieval` answers over
  this corpus only. Cross-corpus questions go through `search-hub`.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
