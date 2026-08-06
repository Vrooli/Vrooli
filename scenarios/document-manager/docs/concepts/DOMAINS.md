# Domains — Document Manager

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Document Manager has eight product domains plus the scaffold `health`
domain. Each product domain owns exactly one noun, and the pipeline runs
through them in order: a file enters at `intake`, becomes structure in
`derivation`, becomes citable in `anchors`, gains meaning in
`enrichment`, is judged for exposure in `sensitivity`, is accounted for
in `custody`, is held in `corpus`, and leaves through `handoff`.

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
| sensitivity | Detect exposure risk and decide how a document may be processed. | Make residency structural: privacy class selects the routing profile. | Detections, privacy classifications, redactions, redaction manifests. | policy | service | PrivacyClass, Detection, Redaction, RedactionManifest | `api/internal/sensitivity/` |
| custody | Record what happened to a document, where, and under whose authority. | Produce the evidence a compliance reviewer accepts. | Custody records, receipts, access events, legal holds. | ledger | reporting | Receipt, CustodyRecord, Attestation, LegalHold | `api/internal/custody/` |
| corpus | Hold collections, govern the artifact store, and let everything leave. | Keep the corpus durable, bounded, and portable. | Collections, membership, retention state, exports. | crud | service | Collection, ArtifactStore, Export | `api/internal/corpus/` |
| handoff | Publish units into a declared ledger scope. | Make the corpus feed the ledger without becoming a second search system. | Publication state, scope bindings, sync cursors. | integration | service | Publication, ScopeBinding | `api/internal/handoff/` |

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
  addressing (hash), dedupe, MIME sniffing, and the text-native versus
  scanned verdict that drives all downstream cost.
- Does not own: parsing (that is `derivation`), byte-level file
  utilities (that is `file-tools`), or retention (that is `corpus`).
- API: `api/internal/intake/`, `api/handlers/intake/`.
- CLI: `cli/domains/intake/` — `ingest`, `watch`, `sources`.
- UI: intake drop zone and source configuration within the Corpus
  surface.
- Storage: `sources`, `blobs`, `documents` tables; raw bytes in the
  artifact store under a `regenerable: false` storage kind.
- Requirements: `DOC-P0-001`, `DOC-P0-002`, `DOC-P0-003`,
  `DOC-P1-016`, `DOC-P1-017`.
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
- Owns: unit segmentation, anchor minting (document hash, page,
  character range, bounding box where available), and anchor resolution.
- Does not own: what a unit *means* (that is `enrichment`) or where it
  ends up (that is `handoff`).
- API: `api/internal/anchors/`, `api/handlers/anchors/`.
- CLI: `cli/domains/anchors/` — `units`, `resolve`, `cite`.
- UI: the Reader's bidirectional highlight — select a unit to highlight
  its source region, select a region to scroll to its unit.
- Storage: `units`, `anchors` tables with an index supporting
  resolution by (document, version, offset).
- Requirements: `DOC-P0-008`, `DOC-P0-009`.
- Tests: anchor round-trip resolution, cross-version resolution (a v1
  anchor still resolves after v2 exists), boundary cases at page and
  block edges.
- Related docs: [`DATA.md`](DATA.md).

### enrichment

- Purpose: add meaning to units through inference that is always
  governed by AI Gateway.
- Primary archetype: service.
- Secondary traits: pipeline, external integration.
- Owns: summaries, classification suggestions, entity and claim
  extraction, schema-first structured extraction, table and equation
  handling, embedding generation, and the embedding metadata contract.
- Does not own: provider selection, credentials, or model identity —
  all of that belongs to AI Gateway. It also does not own the vector
  index or recall, which belong to `vrooli-memory`.
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

### sensitivity

- Purpose: decide how exposed a document is, and let that decision —
  not a user preference — govern where it may be processed.
- Primary archetype: policy.
- Secondary traits: service, human-in-the-loop workflow.
- Owns: PII and PHI detection, categorization by regulation, privacy
  classification, the mapping from privacy class to AI Gateway routing
  profile, redaction proposals, redaction confirmation, and redaction
  manifests.
- Does not own: enforcement of the route itself — AI Gateway enforces
  fail-closed behavior; this domain supplies the class and profile.
- API: `api/internal/sensitivity/`, `api/handlers/sensitivity/`.
- CLI: `cli/domains/sensitivity/` — `scan`, `classify`, `redact`.
- UI: the redaction review surface; privacy-class filters in the Corpus.
- Storage: `detections`, `classifications`, `redactions`,
  `redaction_manifests`.
- Requirements: `DOC-P0-012`, `DOC-P0-013`, `DOC-P1-009`,
  `DOC-P1-010`, `DOC-P1-011`, `DOC-P1-012`.
- Tests: a confidential document must fail closed rather than route
  remote (asserted as a failure, not skipped), redaction requires
  explicit confirmation, redacted derivatives only expose redacted
  units.
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
- Owns: collections and membership, the artifact-store layout, storage
  kind declarations and retention budgets, prune coordination, and full
  export.
- Does not own: retention *enforcement*, which is `storage-manager`'s;
  this domain declares kinds and budgets and honors holds.
- API: `api/internal/corpus/`, `api/handlers/corpus/`.
- CLI: `cli/domains/corpus/` — `collections`, `export`, `prune`.
- UI: the Corpus surface — browse, filter by privacy class and
  confidence, and the low-confidence review queue.
- Storage: `collections`, `collection_members`, `retention_state`; the
  artifact store reached only through the routed file-store seam.
- Requirements: `DOC-P0-016`, `DOC-P0-017`, `DOC-P1-018`.
- Tests: export round-trip fidelity, retention declarations present for
  every kind, prune respecting legal hold and `regenerable: false`.
- Related docs: [`DATA.md`](DATA.md),
  [`../operations/RUNBOOK.md`](../operations/RUNBOOK.md).

### handoff

- Purpose: publish units into a declared ledger scope, idempotently,
  without becoming a second search system.
- Primary archetype: integration.
- Secondary traits: service.
- Owns: publication state, scope bindings, sync cursors, provenance
  construction, and re-derivation propagation to already-published
  entries.
- Does not own: anything about how the ledger stores, compacts, or
  recalls. **This scenario exposes no search or index endpoint** — that
  boundary is the reason unified cross-scope recall works.
- API: `api/internal/handoff/`, `api/handlers/handoff/`.
- CLI: `cli/domains/handoff/` — `publish`, `scope`, `sync`.
- UI: publication status per collection.
- Storage: `publications`, `scope_bindings`, `sync_cursors`.
- Requirements: `DOC-P0-018`, `DOC-P0-019`, `DOC-P1-020`.
- Tests: idempotent republish producing no duplicates, provenance
  resolving back through `anchors`, behavior when the ledger scope does
  not yet exist.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`FLOWS.md`](FLOWS.md).

## Cross-Cutting Requirements

These are not domains; they are obligations every domain shares.

| Requirement | Obligation |
|---|---|
| `DOC-P0-020` | Every domain verb is a proto service method with a generated CLI wrapper. No hand-written second implementation. |
| `DOC-P0-021` | The Reader spans `anchors`, `derivation` and `enrichment`; it is owned by the UI layer, not by one domain. |
| `DOC-P0-022` | Lifecycle and test compliance across all phases, including `ai-conformance`. |
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

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
