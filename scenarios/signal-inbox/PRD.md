# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `/scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (the test-genie `business` phase)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own the fleet's capture of external material as a permanent capability — one append-only journal of every signal the operator saved from outside Vrooli (bookmarks, articles, videos, AI conversations, screenshots, pasted notes), organized into operator-defined categories, retrievable by both structured filter and semantic search, and consumable by any scenario as a stable contract.
- **Primary users/verticals**: The operator, who captures and triages; agents that consume signals by category — near-term the `vision-walk-prep` director-swarm member draining alpha signals, later any scenario that owns a category (a meal-planning scenario consuming `meals`, a marketing member consuming `marketing`). This is not a multi-tenant product; there is exactly one operator.
- **Deployment surfaces**: UI (capture, triage queue, browse and search, signal detail, adapter control), CLI (capture and query verbs for agents and scripts), API (Connect-RPC, the consumption contract), and a search-hub provider registration.
- **Value promise**: External material the operator finds valuable currently survives only in their head or in a platform's private saved list, and reaches Vrooli only by being retold in conversation. This makes that capture durable, categorized, and queryable — so a signal is captured once, routed once, and never re-discussed.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Immutable signal journal | The system shall persist every captured signal to an append-only journal whose entries are never rewritten and never deleted, and shall reject a duplicate capture of identical source content by content hash.
- [ ] OT-P0-002 | Manual capture | The system shall accept a signal supplied directly by the operator as a URL, as pasted text, or as a pasted or uploaded image, with an optional note recorded at capture time.
- [ ] OT-P0-003 | Content extraction with empty-result detection | When a signal carries a resolvable source, the system shall extract its text content, and shall mark the signal `needs-attention` rather than storing an empty body when extraction yields no content.
- [ ] OT-P0-004 | Operator-defined categories | The system shall let the operator create, rename, and retire categories at runtime, and shall never require a code change to add one.
- [ ] OT-P0-005 | Assisted classification with operator override | When a signal is captured, the system shall propose a category with a confidence score, shall require operator confirmation before the assignment is authoritative, and shall allow reassignment at any later time.
- [ ] OT-P0-006 | Annotation thread | The system shall accept append-only annotations on any signal, including free-text notes and typed outcome links that reference what the signal produced.
- [ ] OT-P0-007 | Disposition lifecycle | The system shall track exactly one disposition per signal across `new`, `triaged`, `routed`, `done`, and `dropped`, and shall support deferring a signal to a future review date.
- [ ] OT-P0-008 | Idempotent archive import | The system shall import operator-supplied platform exports, and re-running an import over unchanged source data shall create no duplicate signals.
- [ ] OT-P0-009 | Structured query contract | The system shall expose filtering by category, disposition, source, capture date, and tag through both API and CLI as a stable contract other scenarios consume.
- [ ] OT-P0-010 | Semantic search over signals | The system shall embed each signal's extracted content and shall answer natural-language queries over the full corpus, including signals whose disposition is `done` or `dropped`.
- [ ] OT-P0-011 | Federated retrieval registration | The system shall register its corpus as a search-hub provider through a declarative descriptor so signals are reachable from federated query without any router change.
- [ ] OT-P0-012 | Budgeted ambient view | When an agent requests the ambient view, the system shall return only signals whose disposition is `new` or `triaged`, bounded by a configured budget, and shall never surface a `done` signal ambiently.
- [ ] OT-P0-013 | Triage surface | The system shall provide a keyboard-operable review queue that presents one unresolved signal at a time with its proposed category and supports accept, override, annotate, and drop without leaving the queue.
- [ ] OT-P0-014 | Risk-tiered source adapters | The system shall require every source adapter to declare a risk tier, shall ship every adapter above tier 0 disabled, and shall disable an adapter automatically when a request returns a rate-limit, forbidden, or challenge response rather than retrying it.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Category taxonomies | The system should let a category declare a typed vocabulary so signals in that category carry a subtype, while categories without a taxonomy carry none.
- [ ] OT-P1-002 | Intake-pipeline routing | The system should route a triaged signal into the agent-system intake pipeline as a team knowledge entry under the topic shape that pipeline expects, and should record the routing as an outcome link.
- [ ] OT-P1-003 | Reddit saved-posts adapter | The system should import saved posts through Reddit's official API under operator-supplied credentials as the first tier-1 adapter.
- [ ] OT-P1-004 | Conversation share-link extraction | The system should extract full conversation content from AI chat share links whose transcripts are embedded in the page rather than served by an API.
- [ ] OT-P1-005 | Video transcript capture | The system should capture a video signal by requesting only its transcript from `video-downloader`, with an optional operator timestamp that marks the passage of interest.
- [ ] OT-P1-006 | Consumer event publication | The system should publish a domain event when a signal enters a category so consuming scenarios can react without polling.
- [ ] OT-P1-007 | Saved views | The system should let the operator save a named filter over the corpus and reopen it from the UI and CLI.
- [ ] OT-P1-008 | Classification quality measurement | The system should measure classification accuracy against the accumulated record of operator overrides and should report it as a trend.
- [ ] OT-P1-009 | Bulk triage | The system should support accepting or dropping a filtered set of signals in one operation, with the affected set shown before the operation commits.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Authenticated session-replay adapter | The system may capture bookmarks from platforms with no usable export or API by replaying an authenticated browser session through `browser-automation-studio` under the tier-2 safety envelope.
- [ ] OT-P2-002 | Browser-extension capture | The system may accept a signal pushed from a browser extension so capture does not require leaving the page.
- [ ] OT-P2-003 | Cross-category signals | The system may allow a signal to carry more than one authoritative category once a disposition-ownership rule exists for the multi-consumer case.
- [ ] OT-P2-004 | Retrieval eval corpus | The system may register a golden retrieval suite with search-hub so signal recall quality is measured over time like any other provider.
- [ ] OT-P2-005 | Category suggestion from clustering | The system may propose new categories by clustering signals that repeatedly land in `uncategorized`.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API with Connect-RPC proto contracts, Go CLI over cli-core primitives, React + TypeScript + Vite UI — the standard react-vite scenario shape. No REST-with-mux and no hand-rolled HTTP routing.
- Data + storage expectations: SQLite in-process for signals, annotations, dispositions, categories, and adapter state; FTS5 for keyword and structured filtering; vector index and embeddings through the shared aisearch-go package used by every existing search provider. No Postgres, and no database-migration code carried in this scenario.
- Integration strategy: ai-gateway for embeddings and classification inference; search-hub for federated retrieval via a declarative provider descriptor; `image-tools` for image-to-text extraction; `video-downloader` for transcript-only video requests; `browser-automation-studio` for tier-2 capture only; vrooli-events receipts arrive automatically through api-core with no integration work. Every enrichment capability another scenario already owns is delegated to, never reimplemented here.
- Non-goals / guardrails: Not a multi-tenant product and not a social-media management tool — there is one operator and no per-user partitioning. Not a content archive of everything the operator reads, only what they deliberately saved. Categorization and disposition are display and routing devices, never storage devices: no signal is deleted, hidden from search, or excluded from the index because of how it was classified. No source adapter may be added without a declared risk tier.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite in-process; ollama via ai-gateway for embedding and classification inference.
- Scenario dependencies: ai-gateway (embeddings, classification), search-hub (federated retrieval registration), image-tools (image-to-text for pasted screenshots), video-downloader (transcript-only capture, P1), browser-automation-studio (tier-2 capture, P2), prompt-manager (intake-pipeline routing target, P1), vrooli-events (run correlation, automatic via api-core).
- Operational risks: A tier-2 adapter that trips a platform's automated defenses can cost the operator the underlying account, which is unrecoverable — this is why every adapter above tier 0 ships disabled and self-disables on the first ambiguous response. Classification quality is unmeasured until an override corpus accumulates, so early category assignments require review rather than trust. Extraction coverage varies by source shape, and a silently empty extraction is indistinguishable from an empty document unless detection is built in from the start. `video-downloader` is assumed to expose a transcript-only request and does not do so today, so `OT-P1-005` is blocked on that capability existing. The platform export formats this scenario imports are owned by the platforms and change without notice.
- Launch sequencing: journal, manual capture, and extraction first, so a signal can be captured and read end to end; then categories, classification, and the operator override that makes assignments trustworthy; then annotations and disposition, which together make a signal's history and its handled-state legible; then archive import, which backfills a real corpus and is what makes classification quality and search relevance measurable rather than asserted; then structured query, semantic search, and search-hub registration; then the ambient view and the triage surface, the two consumption paths that make the corpus useful; then the adapter risk-tier contract, exercised first by the tier-0 import adapter before any networked adapter exists. Import precedes search and classification measurement deliberately — relevance tuning against an empty corpus cannot be validated.

## 🎨 UX & Branding

- Look & feel: Vrooli Operational Console per root `DESIGN.md` and this scenario's `DESIGN.md` — calm, dense, technical, slate neutrals with blue primary and cyan technical emphasis; light, dark, and system modes are first-class. The signature surface is the triage queue, optimized for speed: one signal at a time, keyboard-operable, with accept as the default path and override always one keystroke away. Capture is a single always-reachable input that accepts a URL, text, or an image without asking the operator to choose a type first.
- Accessibility: WCAG AA contrast in both themes, visible focus states, 44px touch targets, no status conveyed by color alone, reduced-motion respected, and full keyboard reachability for capture, triage, browse, and correction flows. The triage queue is keyboard-complete, not merely keyboard-accessible.
- Voice & messaging: Precise and operational. A signal's source, its extracted content, and the operator's annotations are always visually distinct so a reader never mistakes an extraction for something the operator wrote, or a classifier's proposal for a confirmed assignment. Confidence is shown as a number, never as a bare label.
- Branding hooks: Inherits the vrooli-default design kit; replace the generic PWA icons when product branding exists.

## 📎 Appendix

- Design record for the workshop that produced this scenario: [`docs/internal/DECISIONS.md`](docs/internal/DECISIONS.md).
- Predecessor: this scenario replaces `bookmark-intelligence-hub`, whose implementation was discarded. The one salvaged artifact is [`docs/reference/conversation-extraction.md`](docs/reference/conversation-extraction.md).
