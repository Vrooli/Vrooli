# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario vrooli-memory`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Own the fleet's agent memory as a permanent capability — one append-only journal of everything an agent deliberately remembers, semantically searchable at full fidelity forever, with pressure-driven hierarchical compaction that keeps ambient recall inside a fixed context budget as the corpus grows without bound.
- **Primary users/verticals**: Coding agents in every supported harness (Claude Code, Codex, Gemini CLI, grok, opencode, antigravity — the six with a Vrooli resource — plus agent-manager runs); the operator reviewing and correcting what the fleet believes; scenarios that need durable cross-session knowledge. Cursor is out of scope until `resources/cursor` exists (`OT-P2-002`).
- **Deployment surfaces**: CLI (the agent-facing write and read verbs), API (Connect-RPC), UI (memory browser, frontier explorer, facet review), and a generated harness memory-file projection.
- **Value promise**: Replaces per-harness private memory stores and the hand-curated MEMORY.md index with one shared, searchable substrate — so knowledge earned by any agent in any runtime is available to every other agent, and retrieval stops depending on a human maintaining pointers.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Append-only journal | The system shall persist every memory write to an append-only journal whose entries are never rewritten and never deleted.
- [ ] OT-P0-002 | Deliberate write verb | When an agent invokes the note verb, the system shall classify the entry's facet, derive its facet texts, embed them, and append the entry to the journal.
- [ ] OT-P0-003 | Full-fidelity semantic recall | When a recall query is issued, the system shall search every journal leaf and every summary and shall never exclude a leaf on the grounds that it has been compacted.
- [ ] OT-P0-004 | Best-node retrieval with descendant collapse | When results span tree depths, the system shall return the best-scoring node at any depth and shall collapse descendants of nodes already returned.
- [ ] OT-P0-005 | Facet-routed retention policy | The system shall assign each memory exactly one facet and shall apply only that facet's declared retention policy, admitting only episode-facet entries to compaction.
- [ ] OT-P0-006 | Guaranteed pinned recall | When wake output is produced, the system shall include every pinned standing-rule memory regardless of its similarity to any query or its position in the tree.
- [ ] OT-P0-007 | Pressure-driven compaction | When the compaction-eligible frontier exceeds its target size, the system shall collapse the highest-scoring candidate cluster and shall repeat until that eligible frontier is under target.
- [ ] OT-P0-008 | Budgeted ambient recall | When wake is invoked, the system shall emit a view bounded by a configured line budget whose granularity is finest for the most recent material.
- [ ] OT-P0-009 | Federated retrieval registration | The system shall register its corpus as a search-hub provider so memory is reachable from federated query without any router change.
- [ ] OT-P0-010 | Harness memory projection | The system shall write wake output to the harness memory file as a one-directional generated projection that it never reads back as input.
- [ ] OT-P0-011 | Harness memory import | The system shall import existing harness memory stores idempotently across markdown, JSONL, and SQLite formats, so re-running an import over unchanged sources creates no duplicates.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Work-record memory kind | The system should accept a work-record memory kind carrying trigger, approach, evidence, and outcome, superseding the separate swarm-manager records write path.
- [ ] OT-P1-002 | Run correlation and sibling events | When a memory is written inside an agent run, the system should store the receipt correlation and should expose a command that lists the other events from that run.
- [ ] OT-P1-003 | Tree descent | When an operator or agent zooms a summary, the system should render that node's immediate constituents and should allow repeated descent to the leaves.
- [ ] OT-P1-004 | Supersession-aware summarization | When a candidate cluster contains contradicting entries, the summarization prompt should resolve them by recency rather than conjoining both claims.
- [ ] OT-P1-005 | Multi-space facet embedding | The system should embed several derived facet texts per memory so that clustering can group entries by more than one notion of relatedness.
- [x] OT-P1-006 | Operator review surface | The system should provide a UI for browsing the journal, inspecting the frontier, and correcting a memory's facet or pinned state.
- [ ] OT-P1-007 | Curated instruction topology repair | On scenario startup, the system should verify that the project-level `AGENTS.md` exists as the canonical curated instruction file and repair declared aliases as symlinks to it without injecting or rewriting programmatic prompt blocks. If the canonical file is missing, it should recover it from a surviving non-generated peer before repairing the links.
- [ ] OT-P1-008 | Harness write capture | The system should capture native memory writes made through each harness's own tooling, via a pre-write hook where the runtime exposes one and store diff everywhere else.
- [ ] OT-P1-010 | Pinned-set bounding | The system should bound the pinned set by operator curation — proposing merges of redundant pins, lapsing pins that are not reconfirmed, and prompting a trade-off when a new pin would exceed the configured budget.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Receipt distillation | The system may propose candidate memories distilled from run receipts for operator confirmation rather than requiring every memory to be written deliberately.
- [ ] OT-P2-002 | Cursor harness support | The system may support Cursor once a `resources/cursor` resource exists; Cursor stores rules in a SQLite BLOB and shares no capture surface with the six CLI harnesses.
- [ ] OT-P2-003 | Re-summarization drift monitoring | The system may re-read a sample of descendant leaves when re-summarizing a node above a configured depth to detect fact mutation.
- [ ] OT-P2-004 | Memory retrieval eval corpus | The system may register a golden retrieval suite with search-hub so memory recall quality is measured over time like any other provider.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Go API with Connect-RPC proto contracts, Go CLI over cli-core primitives, React + TypeScript + Vite UI — the standard react-vite scenario shape.
- Data + storage expectations: SQLite in-process for the journal, tree, and facet metadata; vector index and embedding through the shared aisearch-go package used by every existing search provider. The journal is the sole authority; the tree is a rebuildable cache.
- Integration strategy: ai-gateway for embeddings and summarization; search-hub for federated retrieval via a declarative provider descriptor; vrooli-events receipts arrive automatically through api-core with no integration work.
- Non-goals / guardrails: Not a chat history store and not a telemetry sink — receipts stay in vrooli-events and are referenced, never copied. Compaction is a context-budget device and never a storage device; no leaf is ever deleted to reclaim space. No access-control partitioning of memory: unified read across all scenarios is the product, not a limitation.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite in-process; ollama via ai-gateway for embedding and summarization inference.
- Scenario dependencies: ai-gateway (embeddings, summarization), search-hub (federated retrieval registration), vrooli-events (run correlation, automatic via api-core), swarm-manager (records absorption and migration of existing work records).
- Operational risks: Summarization quality bounds compaction quality, and repeated re-encoding of the same node can mutate facts rather than merely dropping them; cohesion scoring constants are unvalidated and expected to change once real clustering output exists; capture coverage varies by harness storage shape — Codex's native memory directory has no verified pre-write hook, while OpenCode has no documented native memory store — but neither runtime's curated `AGENTS.md` is touched; a harness that silently truncates an oversized projection could drop a pinned standing rule with no signal; and the pinned set grows monotonically unless curated, where the binding constraint is operator attention rather than the token budget, so it degrades ambient recall well before overflow is reported.
- Launch sequencing: journal plus deliberate write path and full-fidelity recall first; then harness memory import, which backfills a real corpus and is what makes everything after it measurable rather than merely asserted; then facet routing and pinning; then pressure-driven compaction and the frontier; then search-hub registration and the harness projection; then work-record absorption and the operator UI. Import precedes compaction deliberately — a compaction loop with an empty journal cannot be validated, and the facet taxonomy has never met a real corpus.

## 🎨 UX & Branding

- Look & feel: Vrooli Operational Console per root DESIGN.md — calm, dense, technical, slate neutrals with blue primary and cyan technical emphasis; light, dark, and system modes are first-class. The signature surfaces are a journal timeline, a frontier explorer that makes the compaction staircase legible, and a facet review queue.
- Accessibility: WCAG AA contrast in both themes, visible focus states, 44px touch targets, no status conveyed by color alone, reduced-motion respected, and full keyboard reachability for browse, zoom, and correction flows.
- Voice & messaging: Precise and operational. Memories are shown verbatim; summaries are always labeled as summaries with their span and depth so a reader never mistakes a compaction for an original.
- Branding hooks: Inherits the vrooli-default design kit; replace the generic PWA icons when product branding exists.
