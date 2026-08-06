# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario document-manager`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Adds the permanent capability to turn any document into citable, re-derivable units of meaning — parsed locally by default, anchored back to exact source geometry, sensitivity-classified before routing, and versioned so parser and model upgrades never invalidate an existing citation. It is the ingestion half of Vrooli's infinite ledger: it owns bytes, derivations, versions, anchors, sensitivity and custody, while vrooli-memory owns meaning and recall and search-hub federates. This scenario supersedes and replaces the retired `document-manager` (repo documentation quality), `secure-document-processing` (encryption/compliance shell), and `data-structurer` (schema extraction) scenarios.
- **Primary users/verticals**: Vrooli agents needing citable source material for research, proof search and truth-seeking work; legal and eDiscovery teams under chain-of-custody obligations; healthcare handling PHI under HIPAA minimum-necessary; finance and insurance needing auditability and versioning; researchers and academics working from papers; and privacy-motivated self-hosters currently served only by OCR-and-tag tools.
- **Deployment surfaces**: Connect-RPC API as the primary contract; generated CLI with full verb parity; React/Vite operator UI built around three surfaces (Corpus, Reader, Receipt); agent tools and widgets declared for conversational use; registered as a search-hub provider.
- **Value promise**: Full AI document understanding without the upload. Every hosted parser requires sending the document to a third party, which compliance teams increasingly refuse; every private alternative stops at OCR and tags. This is the third position — understanding, run locally by default, with a per-document custody receipt showing exactly where each step executed. The receipt is the differentiator, and AI Gateway's existing RouteEvidence already emits most of it.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Content-addressed intake | Files keyed by content hash; identical bytes are never stored or derived twice
- [ ] OT-P0-002 | Type and scan verdict | Real MIME sniffing plus a text-native vs scanned decision, recorded before any parse runs
- [ ] OT-P0-003 | Two intake paths | API/CLI upload and a watched directory, both idempotent under repeat submission
- [ ] OT-P0-004 | Tier-1 native parse | Text-native PDFs parsed locally with no AI call and no network egress
- [ ] OT-P0-005 | Tier-2 structural parse | DOCX, HTML, EPUB, Markdown and plain text parsed through the structural resource
- [ ] OT-P0-006 | One normalized document model | Every tier emits the same canonical model; consumers cannot tell which tier ran except by reading metadata
- [ ] OT-P0-007 | Derivation versioning | Each derivation records tier, resource version and model role under a monotonic version; re-derivation never overwrites a prior version
- [ ] OT-P0-008 | Stable unit anchors | Every unit carries document hash, page and character range, plus bounding box where the tier supplies one
- [ ] OT-P0-009 | Anchor resolution survives re-derivation | An anchor minted against v1 still resolves to the correct source region after v2 is produced
- [ ] OT-P0-010 | Gateway-only inference | Zero direct provider calls; ai-conformance reports L2 on all four capabilities
- [ ] OT-P0-011 | Embedding metadata recorded | Role, model, dimension, content-version and retarget strategy stored alongside every vector from the first migration
- [ ] OT-P0-012 | Privacy class per document | Every document carries a PrivacyClass, and that class selects the gateway routing profile rather than a user setting
- [ ] OT-P0-013 | Fail-closed on sensitive documents | Confidential and secret documents never route remote; enforced in code, covered by a test asserting the failure, and visible in the UI
- [ ] OT-P0-014 | Per-document processing receipt | Record of every derivation and inference step: tier, provider, locality, profile, privacy class and timestamps
- [ ] OT-P0-015 | Immutable audit trail | Append-only custody records; no verb rewrites or deletes one, and document deletion tombstones rather than erasing the trail
- [ ] OT-P0-016 | Artifact store under declared budgets | Bytes and derivations registered as storage-manager kinds with explicit retention and a regenerable flag per kind
- [ ] OT-P0-017 | Full corpus export | Whole corpus exports to an open documented format with no proprietary container and no lock-in
- [ ] OT-P0-018 | Ledger handoff | Units append into a declared vrooli-memory scope with provenance resolving back to anchors
- [ ] OT-P0-019 | Idempotent ledger handoff | Re-running publish produces no duplicate entries, matching the ledger composite import key
- [ ] OT-P0-020 | Connect-RPC with CLI parity | Every domain verb defined in proto; the CLI is a thin generated wrapper rather than a second implementation
- [ ] OT-P0-021 | The Reader surface | Side-by-side source page and derived units with bidirectional anchor highlighting and per-unit tier and confidence display
- [ ] OT-P0-022 | Lifecycle and test compliance | setup/start/test/stop through the control plane with test-genie green including the ai-conformance phase

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Tier-3 vision parse | Scanned pages parsed through the gateway vision role, metered; blocked on the vision.default gateway role landing
- [ ] OT-P1-002 | Confidence-driven tier escalation | A low-confidence tier-1 or tier-2 result auto-escalates to the next tier with the decision recorded in the receipt
- [ ] OT-P1-003 | Collection-wide re-derivation | Bulk re-derive on parser or model upgrade, producing a diff report of what changed
- [ ] OT-P1-004 | Schema-first extraction | Caller supplies a schema and receives validated structured output; absorbs the retired data-structurer charter
- [ ] OT-P1-005 | Table fidelity | Tables survive as addressable cell structure rather than flattened prose
- [ ] OT-P1-006 | Equation preservation | LaTeX or MathML retained through the pipeline, the requirement that makes math papers usable
- [ ] OT-P1-007 | Reference extraction | Bibliographies parsed into resolvable links, enabling a corpus citation graph
- [ ] OT-P1-008 | Per-unit summary and classification | Every unit carries a summary and a facet suggestion the ledger can accept or override
- [ ] OT-P1-009 | PII and PHI detection | Detected entities categorized by sensitivity and by regulation such as GDPR and HIPAA, not merely flagged
- [ ] OT-P1-010 | Human-in-the-loop redaction | Proposed redactions require explicit confirmation before finalize; automated redaction without review is not defensible
- [ ] OT-P1-011 | Redaction manifest | Original version, redacted version, and per-redaction actor, timestamp and regulatory basis
- [ ] OT-P1-012 | Redacted-derivative handoff | When a document is redacted, only redacted units may reach a shared ledger scope
- [ ] OT-P1-013 | Residency attestation export | Human-readable signed report per document or collection covering every step, where it ran and under which profile
- [ ] OT-P1-014 | Per-collection access control | Role-based access with every access event written to the audit trail
- [ ] OT-P1-015 | Legal hold | Suspends retention and prune on a collection; release is itself an audited event
- [ ] OT-P1-016 | Source connectors | arXiv and DOI identifiers, arbitrary URLs, and email attachments as intake sources
- [ ] OT-P1-017 | Bulk import with routing preview | Before a bulk run, show where each document will route and what it will cost, then commit explicitly
- [ ] OT-P1-018 | Corpus browser and review queue | Browse and filter by privacy class and confidence; low-confidence parses surface as a work queue
- [ ] OT-P1-019 | Agent tools discoverable | Declared tools and widgets that cli-health can actually discover, not merely present in code
- [ ] OT-P1-020 | search-hub provider registration | Registered and active so corpus content is reachable from federated search
- [ ] OT-P1-021 | Metered tier through LPBS | Reserve, execute and finalize credits on tier-3 and enrichment with a visible usage ledger and graceful insufficient-credit handling
- [ ] OT-P1-022 | BYOK charges nothing | A user supplying their own provider key is never charged credits, per the paid-features canon

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Handwriting and marginalia | Archival and clinical-notes use cases beyond printed text
- [ ] OT-P2-002 | Multilingual and translation | Translated derivations stored as versions rather than replacements
- [ ] OT-P2-003 | Audio and video transcript intake | Bridges audio-tools; a transcript is a document with time anchors instead of page anchors
- [ ] OT-P2-004 | Citation graph UI | The corpus rendered as a network, the surface that makes research work feel different
- [ ] OT-P2-005 | Air-gapped deployment profile | A configuration making remote routing structurally impossible for buyers who need that on paper
- [ ] OT-P2-006 | Compliance presets | HIPAA, GDPR and eDiscovery starting configurations as packaging over existing mechanism
- [ ] OT-P2-007 | Document diff | Compare two derivation versions or two documents at the unit level
- [ ] OT-P2-008 | Corrections as training signal | Reader corrections become labelled data for tier routing and classification accuracy
- [ ] OT-P2-009 | Collaborative annotation | Multi-user annotation and discussion; deliberately last as it is a distinct product

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go API with Connect-RPC over generated proto contracts; Go CLI as a thin generated wrapper; React + TypeScript + Vite UI on the vrooli-default design kit. Domain-first API shape with per-domain service, repository, schema, handler module, mocks and tests.
- **Data + storage expectations**: SQLite for metadata, derivations, anchors, custody records and collections — no shared resource needed for the core. The artifact store (raw bytes plus derivation outputs) registers as storage-manager kinds with explicit retention and per-kind regenerable flags; raw bytes are `regenerable: false`, derivation outputs are `regenerable: true`. All file access flows through the routed file-store seam so test isolation is honored per request. The custody journal is append-only.
- **Integration strategy**: All inference through AI Gateway profiles by role — never direct provider HTTP, never a concrete model slug, never a hard-coded embedding dimension. Parsing runtimes are owned by resources (`unstructured-io` for tier 2, a new pdf-inspector resource for tier 1); this scenario owns only the routing between tiers. Semantic indexing and recall belong to vrooli-memory, federated query to search-hub, byte-level file operations to file-tools, retention enforcement to storage-manager, and credits to landing-page-business-suite.
- **Non-goals / guardrails**: No search or index endpoint of its own — building one would create a second semantic corpus competing with the ledger and break unified cross-scope recall. No encryption or compliance-framework badges as headline features; encryption at rest is table stakes, not a wedge. No dynamic per-schema table creation. No direct provider credentials, URLs or secrets. No collaborative editing. The sensitivity gate runs before tier selection, never after, so residency is structural rather than advisory.

## 🤝 Dependencies & Launch Plan

- **Required resources**: `unstructured-io` (tier-2 structural parse; currently mid-migration to the docker-service structure and must be verified before it is relied on). A new pdf-inspector resource for tier-1 native extraction and classification — Rust library with Node, Python and CLI bindings, so a CLI-shaped resource is the light path.
- **Scenario dependencies**: `ai-gateway` for all inference and for the RouteEvidence records the custody receipt is built from; `vrooli-memory` as the ledger this scenario publishes units into; `storage-manager` for artifact-store retention governance; `search-hub` for federated discoverability; `landing-page-business-suite` for metered credits on the paid tier.
- **Operational risks**: (1) AI Gateway has no vision or multimodal role today — baseline roles cover chat, summarize, classify, extract, embedding, rerank, code and agent, but nothing maps page image to text. Tier 3 is blocked until `vision.default` is declared in resource policy, and building it without that would trip the ai-conformance `ai.direct_ollama_http` ERROR finding. (2) RouteEvidence carries no caller correlation key and ListRouteEvidence filters only by limit and scenario, so a receipt built today is self-attested by this scenario with no independent gateway-side record to corroborate it; adding a correlation field upgrades the attestation from self-report to evidence. (3) OpenRouter `embedding.default` is still unverified, so the bring-your-own-key path for this scenario's most frequent operation is currently local-only. (4) Local vision inference is heavy; declared `required_vram_bytes` footprints must be supplied so the gateway capacity broker can gate local eligibility. (5) The ledger's per-scope CRUD is not yet built, so the handoff seam may need to follow vrooli-memory's scope work.
- **Launch sequencing**: (1) Verify or package the two parse resources. (2) Land the AI Gateway `vision.default` role and route-evidence correlation key as separate small plans outside this scenario. (3) Build intake, derivation tiers 1-2, anchors and custody — the P0 spine. (4) Add sensitivity classification and the fail-closed routing guarantee. (5) Wire the ledger handoff once vrooli-memory scope CRUD exists. (6) Ship the Reader. (7) Layer the commercial surface: tier 3, redaction, attestation export and metering.

## 🎨 UX & Branding

- **Look & feel**: The vrooli-default operational-console design kit, light and dark. Information density suited to reviewing long documents rather than marketing polish. The organizing principle is that locality is a first-class visual property, not a settings page: every document, unit and operation shows where it was processed, always. Three surfaces carry the product — the Corpus (collections and documents, filterable by privacy class and confidence, with a review queue for low-confidence parses), the Reader (source page beside derived units with bidirectional anchor highlighting, per-unit tier and confidence, and corrections that create a new derivation version rather than overwriting), and the Receipt (a per-document timeline of every step showing tier, provider, locality, profile, privacy class, duration and credits, with a hard visual boundary between local and remote execution). Two cross-cutting moments carry disproportionate weight: routing preview before any bulk spend, and redaction review where proposed redactions are never auto-applied.
- **Accessibility**: WCAG 2.1 AA. Semantic HTML with correct heading hierarchy; full keyboard navigation including the Reader's anchor traversal; visible focus states throughout; ARIA labelling on the page-and-units split view so screen-reader users can move between a unit and its source region; status conveyed by text and shape as well as color, since the local-versus-remote distinction must never depend on color alone; `prefers-reduced-motion` respected; safe-area tokens and fixed bottom navigation preserved on mobile.
- **Voice & messaging**: Precise and evidentiary rather than reassuring. The product's claim is falsifiable, so the copy states facts a user can check — "0 bytes left this machine" beside a receipt link, not "your data is safe." Errors name what went wrong and what to do next. Confidence is always shown honestly, including when the machine is unsure.
- **Branding hooks**: Generic template icons and the seeded PWA manifest, service worker and safe-area tokens stay valid until product branding exists. Status-color semantics come from the design kit and are not overridden.

## 📎 Appendix

**Supersedes.** This scenario replaces three retired scenarios whose charters overlapped and none of which served the need: `document-manager` (repo documentation quality management), `secure-document-processing` (an upload form with encryption-themed styling and unimplemented compliance frameworks), and `data-structurer` (schema-first extraction over a direct-to-Ollama pipeline). What carried forward: the managed-corpus idea and the improvement queue shape (now the review queue) from the first; the security-as-monetization thesis, reframed from encryption to provable processing locality, from the second; and schema-first extraction as a first-class enrichment verb from the third.

**Market position.** Hosted document parsers offer understanding but require uploading the document. Self-hosted tools keep files local but stop at OCR and tagging with no understanding, no citable anchors and no audit trail. Legal and financial buyers face chain-of-custody requirements that increasingly bar routing privileged documents through third-party APIs, and healthcare's HIPAA minimum-necessary standard requires explicit justification for PHI leaving the environment. Regulators have stopped accepting vendor attestations about residency and want technical verification plus a record of what was processed where — which AI Gateway's RouteEvidence, combined with the fail-closed `privacy-sensitive` profile, already very nearly produces.

**Monetization posture.** Free: tier-1 and tier-2 parse, anchoring, versioning, corpus management, custody receipts and export — deterministic, local, no marginal cost, and permanently free because it competes with genuinely good free self-hosted tools. Metered: tier-3 vision parse, enrichment and embeddings, routed through the gateway's existing credit path, falling back to a user's own key at zero charge. Gated: nothing, deliberately — the value is real compute or nothing, and a gated feature appearing later is a signal the framing drifted. Bundle placement is operator-curated canon and is not decided here.

**Reference material.** `scenarios/ai-gateway/docs/reference/roles-profiles-policies.md`; `scenarios/test-genie/docs/phases/ai-conformance/README.md`; `packages/proto/schemas/ai-gateway/v1/routing/routing.proto` (RouteEvidence) and `.../shared/gateway.proto` (PrivacyClass, Profile); `docs/concepts/PAID_FEATURES.md`; `docs/concepts/ECOSYSTEM.md`.
