# Product Requirements Document (PRD)

## 🎯 Overview

- **Purpose**: Code Facts provides target-aware, reusable code evidence for Vrooli scenarios and bounded generic code roots. It resolves targets, brokers language graph providers, enriches scenario-aware surface facts when available, and returns selective fact families that downstream validators can trust without parsing source code themselves.
- **Primary users/verticals**: Vrooli validation scenarios (`proto-health`, future `test-genie`, `cli-health`, `ui-health`, `ecosystem-manager`), migration agents, scenario maintainers, and operators investigating evidence behind health findings.
- **Deployment surfaces**: Connect-RPC API and Go CLI are the primary programmatic surfaces. The React UI is an operator workbench for target inspection, fact-family filtering, warnings, evidence, and cache diagnostics.
- **Value promise**: Replace duplicated grep/source-scanning logic across health and validation scenarios with one deterministic evidence substrate. Language parsing stays in `go-code-graph` and `typescript-code-graph`; Code Facts owns target resolution, brokering, scenario enrichment, proof synthesis, and cache semantics.

## 🎯 Operational Targets

### 🔴 P0 – Must ship for viability

- [ ] OT-P0-001 | Target Resolution | Target resolution supports `path`, `scenario`, `module`, and `project` inputs, including repo-root overrides and language filters.
- [ ] OT-P0-002 | Surface Inventory | Surface inventory identifies Vrooli scenario surfaces and generic parse units with source-backed status.
- [ ] OT-P0-003 | Analyzer Brokering | Analyzer brokering calls Go and TypeScript graph providers without parsing supported source directly.
- [ ] OT-P0-004 | Selective Describe | Selective describe returns only requested fact families and clearly reports unsupported families.
- [ ] OT-P0-005 | Evidence Status Model | Evidence status model supports `proven`, `missing`, `contradicted`, `unsupported`, and `unknown`.
- [ ] OT-P0-006 | Proto Adoption Facts | Proto adoption facts work for Go API/CLI and TypeScript UI surfaces using generic import/reference/call evidence.
- [ ] OT-P0-007 | Endpoint Proof Facts | Endpoint proof facts work for Go REST handlers using generic graph usage facts, not proto-health heuristics.
- [ ] OT-P0-008 | Deterministic Cache | Cache keys and invalidation are deterministic, inspectable, and tied to analyzer version, target/options, source hashes, graph hashes, and requested fact families.
- [ ] OT-P0-009 | API/CLI Parity | CLI and Connect API expose equivalent core operations for describe, surfaces, proto adoption, endpoint proof, and cache diagnostics.
- [ ] OT-P0-010 | Operator Workbench | Operator UI can inspect targets, surfaces, parse units, facts, warnings, evidence, and cache status.

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
- Data + storage expectations: v1 can start with local deterministic cache metadata and filesystem-backed test fakes. SQLite is acceptable for cache index/audit history if Phase 9 needs persistence. No shared resource is required for Phase 5.
- Integration strategy: Code Facts calls `go-code-graph` for Go parse units and `typescript-code-graph` for TypeScript projects. It reads Vrooli repo/scenario metadata shallowly (`.vrooli/service.json`, `.vrooli/endpoints.json`, CLI manifests, testing metadata) but does not parse supported source languages itself.
- Non-goals / guardrails: No language parser logic in Code Facts for Go or TypeScript. No proto-health policy inside graph providers. No compatibility aliases for greenfield contracts. No unbounded monorepo analysis in v1; callers must provide explicit bounded targets.

## 🤝 Dependencies & Launch Plan

- Required resources: None for v1. Local filesystem access is sufficient for target resolution, metadata reads, and cache files.
- Scenario dependencies: `go-code-graph` and `typescript-code-graph` are production provider dependencies once analyzer brokering is implemented. `proto-health` is the first planned consumer after Code Facts exposes proto adoption and endpoint proof families.
- Operational risks: Provider outages must degrade to typed `unsupported` or `unknown` evidence instead of silent success. Cache staleness must be visible through key material and invalidation reasons. Scenario metadata may be incomplete, so surface inventory must distinguish `missing`, `ambiguous`, and `unknown`.
- Launch sequencing: Phase 5 creates the scenario and product contract. Phase 6 defines proto/API/CLI core. Phase 7 implements target and surface discovery. Phase 8 brokers graph providers. Phase 9 adds cache/performance. Phase 10 adds proof synthesis. Phase 11 builds the operator UI. Phase 12 migrates `proto-health`.

## 🎨 UX & Branding

- Look & feel: Dense operational workbench using `vrooli-default`; optimized for scanning evidence tables, filters, warnings, and cache diagnostics rather than marketing content.
- Accessibility: WCAG AA target. All graph/fact visualizations require table or list alternatives, keyboard navigation, visible focus, non-color-only status labels, and stable responsive dimensions.
- Voice & messaging: Engineer-direct. State what was analyzed, what was skipped, what evidence supports each status, and which provider produced it.
- Branding hooks: Inherits Vrooli design tokens and PWA assets from the template. Code Facts is infrastructure, not a standalone consumer brand.

## 📎 Appendix

- Plan source: operator runtime plan `code-facts-scenario-and-graph-provider-generalization.md`
- Provider PRDs: `scenarios/go-code-graph/PRD.md`, `scenarios/typescript-code-graph/PRD.md`
- First consumer context: `scenarios/proto-health/PRD.md`
