# Product Requirements Document (PRD)

## 🎯 Overview

- **Purpose**: Code Facts provides target-aware, reusable code evidence for Vrooli scenarios and bounded generic code roots. It resolves targets, brokers language graph providers, enriches scenario-aware surface facts when available, and returns selective fact families that downstream validators can trust without parsing source code themselves.
- **Primary users/verticals**: Vrooli validation scenarios (`proto-health`, future `test-genie`, `cli-health`, `ui-health`, `swarm-manager`), migration agents, scenario maintainers, and operators investigating evidence behind health findings.
- **Deployment surfaces**: Connect-RPC API and Go CLI are the primary programmatic surfaces. The React UI is an operator workbench for target inspection, fact-family filtering, warnings, evidence, and cache diagnostics.
- **Value promise**: Replace duplicated grep/source-scanning logic across health and validation scenarios with one deterministic evidence substrate. Language parsing stays in `go-code-graph` and `typescript-code-graph`; Code Facts owns target resolution, brokering, scenario enrichment, proof synthesis, and cache semantics.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [x] OT-P0-001 | Target Resolution | Target resolution supports `path`, `scenario`, `module`, and `project` inputs, including repo-root overrides and language filters.
- [x] OT-P0-002 | Surface Inventory | Surface inventory identifies Vrooli scenario surfaces and generic parse units with source-backed status.
- [x] OT-P0-003 | Analyzer Brokering | Analyzer brokering calls Go and TypeScript graph providers without parsing supported source directly.
- [x] OT-P0-004 | Selective Describe | Selective describe returns only requested fact families and clearly reports unsupported families.
- [x] OT-P0-005 | Evidence Status Model | Evidence status model supports `proven`, `missing`, `contradicted`, `unsupported`, and `unknown`.
- [x] OT-P0-006 | Proto Adoption Facts | Proto adoption facts work for Go API/CLI and TypeScript UI surfaces using generic import/reference/call evidence.
- [x] OT-P0-007 | Endpoint Proof Facts | Endpoint proof facts work through Code Facts framework adapters using generic graph usage facts, not proto-health heuristics.
- [x] OT-P0-008 | Deterministic Cache | Cache keys and invalidation are deterministic, inspectable, and tied to analyzer version, target/options, source hashes, graph hashes, and requested fact families.
- [x] OT-P0-009 | API/CLI Parity | CLI and Connect API expose equivalent core operations for describe, surfaces, proto adoption, endpoint proof, and cache diagnostics.
- [x] OT-P0-010 | Operator Workbench | Operator UI can inspect targets, surfaces, parse units, facts, warnings, evidence, and cache status.
- [ ] OT-P0-011 | Governed Source Catalog | When a governed project root changes, Code Facts MUST maintain one authoritative catalog generation that classifies every eligible source as implementation, contract, generated alias, test, fixture, documentation, or transient and excludes transient artifacts from retrieval.
- [x] OT-P0-012 | Indexed Exact Retrieval | When an exact identifier, symbol, or path query arrives, Code Facts MUST answer from the persistent catalog and FTS index without opening repository source files, with p95 wall time at most 100 ms on the current corpus and 200 ms on the three-times corpus.
- [ ] OT-P0-013 | Selective Hybrid Retrieval | When a natural-language code query arrives, Code Facts MUST fuse evaluated lexical and semantic code-card candidates, reach recall@5 of at least 95% and MRR@3 of at least 85%, and keep p95 wall time at most 500 ms on the current corpus and 750 ms on the three-times corpus.
- [x] OT-P0-014 | Generation-Safe Freshness | Code Facts MUST atomically reconcile ordinary file changes into the active catalog and FTS index, suppress stale candidates, make those changes searchable within 15 seconds p95, repair missed events within five minutes, and route extraction-policy, schema, or model changes through an isolated shadow generation.
- [ ] OT-P0-015 | Stable Evidence Provenance | When Code Facts returns a result or relationship, it MUST provide a stable content or symbol identity, current source range and hash, active generation, retrieval regime, relevance explanation, analyzer provenance, and an evidence status that is independent of relevance score.
- [x] OT-P0-016 | Truthful Degradation | If Qdrant, Ollama, the reranker, or a graph projection is unavailable, then Code Facts MUST keep correct lexical retrieval available and name each unavailable or bypassed stage without presenting lexical similarity as relationship proof.
- [x] OT-P0-017 | Governed Index Controls | When an authorized operator reconciles, reindexes, cancels, promotes, rolls back, or cleans an index generation, Code Facts MUST expose the same durable and truthful job state through typed API, CLI, UI, and Search Hub control surfaces while preserving the last complete generation on failure.
- [x] OT-P0-018 | Bounded Resource Governance | While search, indexing, embedding, graph extraction, fleet analysis, and cache maintenance run concurrently, Code Facts MUST enforce one process-wide admission budget, bounded queues and caches, cancellation, steady RSS at most 150 MiB, query memory delta at most 50 MiB, index RSS at most 500 MiB, and current-corpus derived storage below 1.5 GiB.
- [x] OT-P0-019 | Descriptor-Backed Contract Authority | When Code Facts indexes protobuf contracts, it MUST read resolved structure and the digest from `descriptorimage.Source`, join canonical identities to authoritative `.proto` provenance, and retain the previous valid contract generation when the descriptor image is missing, invalid, or stale.
- [x] OT-P0-020 | Search Hub Indexed Provider | While the active generation is healthy, Code Facts MUST self-register truthful scoped `local_index` code and contract leaves whose status exposes generation, freshness, drift, document counts, degraded stages, and request-budget behavior.
- [x] OT-P0-021 | Evidence Workspace | When an operator searches or manages the corpus, the responsive UI MUST make ranked evidence, provenance, relationships, contracts, freshness, degradation, generation controls, and evaluation comparisons understandable through WCAG 2.2 AA keyboard-accessible journeys at desktop, tablet, and mobile widths.

### 🟠 P1 – Should have post-launch

- [ ] OT-P1-001 | Provider Cost Visibility | Reports distinguish provider-level extraction cost from Code Facts include filtering when graph providers cannot extract selectively.
- [ ] OT-P1-002 | Additional Proof Families | CLI proof and UI widget proof families are exposed behind the same status/evidence model once first consumers are ready.
- [ ] OT-P1-003 | Snapshot Diff Diagnostics | Operators can compare two Code Facts reports to explain changed evidence after a refactor.
- [ ] OT-P1-004 | Adoption Readiness Exports | Downstream validators can persist compact fact summaries for baseline diffs without storing full graphs.

### 🟢 P2 – Future / expansion

- [ ] OT-P2-001 | Additional Language Providers | Add Python, Rust, or other graph providers through analyzer seams without changing target resolution or evidence contracts.
- [ ] OT-P2-002 | Architecture Cartographer Adoption | Architecture Cartographer may consume Code Facts once the lower-level language graph contract is proven stable.
- [ ] OT-P2-003 | Cross-Scenario Fact Cache | Share cache metadata across consumers when lifecycle/runtime policy can guarantee safe reuse.

## 🧱 Tech Direction Snapshot

- Preferred stacks / frameworks: Generated React-Vite scenario with Go API, Go CLI, Connect-RPC, protobuf contracts in `packages/proto/schemas/code-facts/v1`, and the `vrooli-default` design kit.
- Data + storage expectations: SQLite is the authority for the source catalog, FTS index, generation state, normalized projections, jobs, and bounded derived cache data. Qdrant stores selective semantic code cards as optional derived data. The active SQLite generation remains queryable without AI resources.
- Integration strategy: Code Facts calls `go-code-graph` for Go parse units and `typescript-code-graph` for TypeScript projects. It reads Vrooli repo/scenario metadata shallowly (`.vrooli/service.json`, `.vrooli/endpoints.json`, CLI manifests, testing metadata) but does not parse supported source languages itself.
- Non-goals / guardrails: No language parser logic in Code Facts for Go or TypeScript. No proto-health policy inside graph providers. No compatibility aliases for greenfield contracts. No unbounded in-memory repository index, per-query global source scan, line-by-line embedding corpus, or mandatory AI dependency for exact search.

## 🤝 Dependencies & Launch Plan

- Required resources: SQLite and local filesystem access are required. Qdrant, Ollama, and the reranker are optional acceleration and relevance resources; their loss must preserve lexical service with explicit degradation.
- Scenario dependencies: `go-code-graph` and `typescript-code-graph` are production provider dependencies once analyzer brokering is implemented. `proto-health` is the first planned consumer after Code Facts exposes proto adoption and endpoint proof families.
- Operational risks: Provider outages must degrade to typed `unsupported` or `unknown` evidence instead of silent success. Catalog, vector, and graph freshness must be fenced by source hash and generation. Long shadow builds must replay bounded later changes before promotion. Scenario metadata may be incomplete, so surface inventory must distinguish `missing`, `ambiguous`, and `unknown`.
- Launch sequencing: Establish the measured corpus and evaluation contract; introduce capability boundaries; build the shared reconciliation substrate; ship the authoritative catalog and FTS index; add selective semantic and graph retrieval; expose generation controls; adopt Search Hub `local_index`; harden resource governance; complete the evidence workspace; then remove the streaming scan and transitional implementations after scale proof.

## 🎨 UX & Branding

- Look & feel: Dense operational workbench using `vrooli-default`; optimized for scanning evidence tables, filters, warnings, and cache diagnostics rather than marketing content.
- Accessibility: WCAG AA target. All graph/fact visualizations require table or list alternatives, keyboard navigation, visible focus, non-color-only status labels, and stable responsive dimensions.
- Voice & messaging: Engineer-direct. State what was analyzed, what was skipped, what evidence supports each status, and which provider produced it.
- Branding hooks: Inherits Vrooli design tokens and PWA assets from the template. Code Facts is infrastructure, not a standalone consumer brand.

## 📎 Appendix

- Plan source: operator runtime plan `code-facts-scenario-and-graph-provider-generalization.md`
- Provider PRDs: `scenarios/go-code-graph/PRD.md`, `scenarios/typescript-code-graph/PRD.md`
- First consumer context: `scenarios/proto-health/PRD.md`
