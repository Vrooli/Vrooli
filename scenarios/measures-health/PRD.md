# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: measures-health is a meta / interface-enabler scenario that enforces and grades scenario adoption of the Vrooli "Measures" capability — named, typed, parameterized analytical queries declared per scenario — and hosts the central measures index plus the single registered search-hub "measure" provider. It enables a natural-language analytical question ("how many backlog items closed this week") to be matched to a declared measure, have its parameters resolved, and (for safe read-only measures at high confidence) be executed and answered directly. It also produces a soft `measures` dimension for the Ecosystem Manager (EM) maturity ladder, mirroring how security-health and cli-health operate today.

Target users: Vrooli itself is the primary consumer — the EM maturity ladder (via test-genie producer findings), agents and operators posing analytical questions through search-hub, and scenario authors adopting the measures contract. Secondary consumers are fleet operators viewing measures coverage and tier status across all scenarios.

Deployment surfaces: Tier 1 local Vrooli stack (full installation). Surfaces include programmatic CLI (`measures-health validate`) and Connect RPC (`MeasuresService.ValidateScenario`), conversational/agentic (measures-health is a federated search-hub provider that answers natural-language analytical questions end-to-end), and a direct fleet-view UI showing scenario × measure coverage and tier.

Value proposition: Makes the Measures capability enforceable (declare-without-implement is caught by the behavioral adoption probe), measurable (per-scenario coverage and tier feed the EM ladder and a fleet dashboard), and directly useful (analytical questions are answered end-to-end without routing to a coding agent). Each adopting scenario's statistics become individually addressable, typed, parameterized measures instead of monolithic stat blobs, raising the quality floor for the entire Vrooli ecosystem.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Static Coverage Validation | `measures-health validate scenario <name> --json` emits an expected/covered/waived report per scenario with a per-measure tier (full/partial/fallback) and a pass/fail verdict, where an uncovered stateful domain is an ERROR and a stale waiver pointing at a non-stateful or nonexistent domain is a WARNING, matching cli-health expected/covered/waived semantics.
- [ ] OT-P0-002 | Central Measures Index and Registered Search-Hub Provider | Harvest every scenario's manifest `measure` blocks, index them via aisearch-go hybrid search (embedding the measures-go MeasureComposer questions key), serve a single provider RPC reusing the shared measures-go Engine, register once via `search-hub providers register` (type `measure`, bucket STATE), answer read-only measures at high confidence automatically, and return resolved-but-unexecuted with a confirmation signal for write/destructive measures.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Behavioral Adoption Probe | Call each declared measure endpoint, assert the response conforms to its declared `result` shape, and round-trip one `questions[]` example end-to-end (match → extract → execute → plausible); a hollow declaration (404 or non-conforming response) is an ERROR, proving real adoption rather than mere declaration.
- [ ] OT-P1-002 | Producer Wiring into EM Maturity Ladder | Expose a Connect RPC `MeasuresService.ValidateScenario`; add a test-genie `measures` phase that shells the validate verb and maps results to a new `FINDING_SOURCE_MEASURES`; introduce a soft `measures` dimension in the EM ladder dimensions SSOT gated at a soft rung (R3/R4) so a scenario remains runnable without measures but cannot reach top maturity without full coverage.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Fleet-View UI | Ship a polished direct UI showing which scenarios expose measures, at what tier and coverage, with loading/error/empty states, a per-scenario EM rung derived automatically from the decision-trace, and a measures-specific drill-down matching the `measures-health validate` report shape.
- [ ] OT-P2-002 | Eval and Observability Hooks | Surface match and param-extraction accuracy signals feeding the Phase 8 eval harness, expose the auto-exec confidence threshold theta as a config lever, and support an optional architecture-cartographer DomainsService cross-check to enrich domain derivation.

## 🧱 Tech Direction Snapshot
Preferred stacks: Go API (proto-first Connect-RPC) and Go CLI (cli-core declarative ArgSchema) for the server and validation tooling; React/Vite UI (vrooli-default design kit) per the react-vite template for the fleet-view UI. Reuse the shared `packages/measures-go` Engine, Gate, MeasureComposer, LLMExtractor, and HTTPExecutor without re-authoring. Reuse `packages/aisearch-go` for the hybrid index. Mirror security-health (substrate-driven expectation + severity grading + producer wiring) and cli-health (expected/covered/waived + behavioral end-to-end boot probe) as structural reference implementations.

Preferred storage: SQLite for the local registry and cache of harvested measures and validation reports (no shared resource required for v1). The central index requires an embedder and a vector store consistent with aisearch-go usage in cli-health and KO: qdrant for vector storage and ollama for embeddings, both existing resources with established wiring patterns.

Integration strategy: Connect-RPC and CLI are the canonical programmatic surface. Register exactly one provider with search-hub (measures-health owns the single registered provider and proxies execution to the owning scenario via search-hub's cross-scenario URL resolver; never client-computed). Resolve the owning scenario at execution time. The search-hub router remains thin; no measure-specific logic belongs in the router.

Non-goals: No operational/Prometheus telemetry. Measures are not prompt-manager actions (separate provider class; no auto-generation from CLI). Never auto-execute write or destructive measures under any confidence level. Do not put measure-specific routing logic in the search-hub router. Do not duplicate param types into the manifest (derive from proto). The `measures` dimension is a soft rung and must never become a hard gate.

## 🤝 Dependencies & Launch Plan
Required resources: ollama (embeddings and constrained extraction completer) and qdrant (vector store) for the central index, reusing existing resource wiring patterns from cli-health and KO. SQLite for local persistence of harvested measures and validation reports.

Scenario dependencies: search-hub (provider registration and cross-scenario URL resolution; the `SearchHit.measure` carrier shipped in Phase 3 is a prerequisite); test-genie (producer phase integration for the `measures` finding source); ecosystem-manager (ladder dimension consumer of `FINDING_SOURCE_MEASURES` results). swarm-manager is the first real adopter scenario (Phase 6) but is not a hard runtime dependency of measures-health itself.

Operational risks: Param extraction correctness — wrong numbers are worse than no answer, mitigated by deterministic time_window resolver, mandatory abstention on low-confidence extractions, and full provenance in every response. Two-hop routing latency between the search-hub provider and the owning scenario, mitigated by per-provider timeouts and caching of hot measures. Domain statefulness misclassification, mitigated by the manifest `measures.domains[]` and `omitted[]` escape hatches available to scenario authors.

Launch sequencing: (1) Stand up static coverage validation CLI and the behavioral adoption probe (P0 OT-P0-001 and P1 OT-P1-001). (2) Build the central measures index and register the single search-hub provider (P0 OT-P0-002); resolve-first before read-only auto-exec is already shipped in measures-go and search-hub Phase 3. (3) Wire the producer phase and EM ladder soft dimension (P1 OT-P1-002). (4) Ship the fleet-view UI (P2 OT-P2-001). (5) Add eval hooks and observability levers (P2 OT-P2-002).

## 🎨 UX & Branding
User experience: Operational console framing — a fleet dashboard presenting a scenario × measures coverage/tier matrix, with a per-scenario drill-down that mirrors the `measures-health validate` report shape (expected, covered, waived, tier, verdict). Loading, error, and empty states are required on all async views. The CLI output is structured JSON (via `--json`) for machine consumers and a human-readable table for interactive use, consistent with cli-health conventions.

Visual design: vrooli-default design kit throughout. Diagnostic, data-dense layout appropriate for an enforcement and measurement tool. Per-scenario rung and coverage tier use the design kit's status token palette (pass/warn/error) for at-a-glance scanning. Keep the seeded `ui/public/site.webmanifest` and PWA manifest icons valid; replace generic icons when final branding assets are available.

Accessibility: WCAG AA contrast enforced via design tokens. All interactive elements carry appropriate ARIA roles and labels. `data-testid` selectors present on all actionable and status-bearing elements for automated test coverage. Full keyboard navigation for the fleet dashboard and drill-down views. Meets the template accessibility floors as the minimum bar.