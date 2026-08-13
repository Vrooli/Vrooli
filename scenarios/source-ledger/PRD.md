# Product Requirements Document (PRD)

> **Template Version**: 2.0
> **Canonical Reference**: `scenarios/business-health/docs/reference/canonical-prd-template.md`
> **Validation**: Enforced by `business-health` (`validate scenario source-ledger`)
> **Policy**: Generated once and treated as read-only (checkboxes may auto-update)

## 🎯 Overview

- **Purpose**: Source Ledger is a permanent, scoped append-only record for facts, decisions, events, and other durable source material. It provides bounded recall over a rebuildable compaction canopy while keeping the journal as the sole corpus authority.
- **Primary users/verticals**: Coding-agent harnesses, teams, investigators, and future research workflows that need durable history without transcript-sized recall.
- **Deployment surfaces**: Go Connect/API service, reusable CLI, direct operator UI, search-hub providers, and lifecycle-managed maintenance and backup operations.
- **Value promise**: Consumers share one trustworthy corpus and one reusable ledger engine instead of rebuilding journal, scope, recall, and compaction behavior for every scenario.

## 🎯 Operational Targets

> Checkboxes auto-update from requirements sync; do not hand-edit them.

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Append-only journal | The ledger shall append every accepted entry and reject updates or deletes to journal rows.
- [ ] OT-P0-002 | Scope-isolated recall | When a client supplies a scope, the ledger shall return only entries and derived nodes from that scope.
- [ ] OT-P0-003 | Bounded compaction canopy | When compaction runs, the ledger shall preserve journal rows and maintain a scored frontier with rebuildable summaries and edges.
- [ ] OT-P0-004 | Stable programmatic surface | The ledger shall expose scope-aware Connect, API, and CLI operations for append, recall, wake, frontier, and health.
- [x] OT-P0-005 | Durable corpus authority | The ledger shall keep one authoritative SQLite corpus with registered backup and restore evidence.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Per-scope vocabulary | Operators should create and update each scope's facet vocabulary, retention rules, residency budgets, and frontier target.
- [ ] OT-P1-002 | Federated scope discovery | Search clients should discover one provider per scope and receive scope-labelled results without duplicate corpus answers.
- [x] OT-P1-003 | Corpus-first operator surface | The operator UI should provide a ledger list, scope detail, journal timeline, frontier explorer, vocabulary editor, and cross-scope search.
- [ ] OT-P1-004 | Append-only migration seam | The ledger should support a verified migration that preserves entry bodies and dependent derived data without deleting source rows.
- [ ] OT-P1-005 | Measured recall cost | The ledger should measure recall cost and apply a bounded candidate strategy when corpus growth exceeds the recorded latency ceiling.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Retention and review analytics | The ledger may expose operator analytics for facet retention, pin review, supersession, and compaction trade-offs.
- [ ] OT-P2-002 | Additional storage backends | The ledger may support a governed storage backend beyond SQLite when one corpus requires it.
- [ ] OT-P2-003 | Retrieval evaluation corpus | The ledger may provide a repeatable evaluation corpus for comparing recall quality across policies and embedding models.

## 🧱 Tech Direction Snapshot

- **Preferred stacks / frameworks**: Go for the API, Connect-generated contracts for programmatic calls, a Go-native CLI, and React/Vite for the operator UI.
- **Data + storage expectations**: SQLite is the authoritative append-only journal and stores scope policy plus rebuildable derived forest data. Journal rows are non-regenerable; summaries, edges, and embeddings are rebuildable.
- **Integration strategy**: Use lifecycle-managed scenario dependencies, generated contracts, search-hub registration, data-backup-manager targets, and the governed ai-gateway inference seam.
- **Non-goals / guardrails**: Do not make the engine agent-memory-specific, do not edit or delete journal rows during compaction, do not create a local read replica in a consumer, and do not add access-control partitioning between scopes.

## 🤝 Dependencies & Launch Plan

- **Required resources**: ai-gateway for governed classification, embeddings, and compaction summaries.
- **Scenario dependencies**: The service-contract phase will declare search-hub for one provider per scope and data-backup-manager for authoritative corpus registration. This scaffold phase records those integrations without starting them before their capability surfaces exist.
- **Operational risks**: A provider outage can pause derived enrichment but must not block journal append or projection consumers; one database remains the authority; compaction failures must be visible without deleting source rows.
- **Launch sequencing**: Author the contract, define the service boundary, move and validate engine packages, migrate the corpus, cut over vrooli-memory, build the corpus-first UI, federate scopes, and then migrate marketing-crew.

## 🎨 UX & Branding

- **Look & feel**: Calm, evidence-oriented corpus tooling with clear scope identity, dense but readable timelines, explicit frontier scores, and light/dark theme support.
- **Accessibility**: Meet WCAG 2.2 AA for keyboard navigation, focus visibility, semantic regions, contrast, reduced motion, and screen-reader labels.
- **Voice and messaging**: Use precise source, scope, retention, and compaction language; distinguish authoritative journal content from rebuildable derived views and failed enrichment.
- **Branding hooks**: Use the generated design tokens and app-shell seams, then replace template assets with a restrained ledger mark and scope-aware status colors.

## 📎 Appendix

- Extraction boundary: journal, forest, facets, recall, policy, vector, inference, and federation move into this scenario; harness adapters and projection remain in vrooli-memory.
- Governing plan: the production-readiness and fleet-adoption plan recorded in the workspace plan registry.
- Phase 12 acceptance requires a healthy contract scaffold and no engine code movement.
