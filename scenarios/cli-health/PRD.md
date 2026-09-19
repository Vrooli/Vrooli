# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Cross-scenario CLI manifest validation and AI-powered command discovery. `cli-health` is the single source of truth for (a) verifying that every scenario's `cli/manifest.json` conforms to its proto contracts and the schema at `.vrooli/schemas/cli-manifest.schema.json`, and (b) giving agents a semantic search surface to discover existing CLI commands across all Vrooli scenarios before proposing new ones.

Target users: Vrooli agents running the promotion ladder (skill-optimizer, planning agents, skill-authoring); `test-genie` (which delegates a new Contracts validation phase to this scenario); engineers triaging scenario CLI health.

Deployment surfaces: Tier 1 local stack only for v1. Headless API and CLI consumed by other scenarios; React UI available for human inspection. No cloud or multi-tenant deployment in scope for v1.

Value proposition: Catches proto/CLI drift before it ships and replaces ad-hoc grepping for existing commands with semantic search backed by manifest and proto descriptors. Every scenario that adopts `cli/manifest.json` receives free validation coverage and full command discoverability automatically, with no additional instrumentation required.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Deterministic manifest validation | `cli-health validate <scenario>` returns schema errors, unresolved `binding.service`/`binding.method` references, orphan proto methods, and stale omission entries; exits nonzero on any error-level finding.
- [ ] OT-P0-002 | Semantic and text-mode command search | `cli-health search "<query>"` returns ranked commands across all adopting scenarios in AI mode; `--text` fallback operates without Ollama or Qdrant.
- [ ] OT-P0-003 | Dependency status reporting | `cli-health status` reports Ollama and Qdrant availability plus the count of currently indexed commands.
- [ ] OT-P0-004 | On-demand and scheduled reindexing | `cli-health reindex` rebuilds the search index on demand; a 5-minute background sync loop keeps the index current without manual intervention.
- [ ] OT-P0-005 | Test-genie Contracts phase integration | `test-genie`'s Contracts phase delegates to `cli-health` through the descriptor-backed provider RPC (`scenario-validation/v1.ScenarioValidationService.ValidateScenario`, `include_execution=true`) and surfaces findings into the report; error findings fail the phase and warning findings produce warnings. cli-health declares the [Phase Capability Contract](../test-genie/docs/concepts/phase-capability-contract.md) for the `contracts` phase (North Star + gated ladder + structured remediation doc) and is the reference lighthouse adopter — the `command_architecture` capability standing renders in run output and its doc-search topics resolve to [`cli-architecture-maturity.md`](docs/reference/cli-architecture-maturity.md).

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Manifest scaffolding command | `cli-health generate-manifest <scenario>` bootstraps a valid `cli/manifest.json` for any scenario that currently lacks one, lowering the barrier to adoption.
- [ ] OT-P1-002 | Non-canonical CLI pattern detection | A `packages/cli-core` usage check across scenarios flags any scenario whose CLI does not follow the canonical pattern, surfacing findings as warnings in validation output.

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Scenario-auditor rule integration | `cli-health` validate and search findings are exposed to `scenario-auditor` as an additional rule source, enabling cross-scenario policy enforcement without duplicating validation logic.
- [ ] OT-P2-002 | Shared AI search package extraction | The AI search pipeline currently duplicated verbatim from `prompt-manager/api/aisearch/` is extracted into a shared package consumed by both scenarios, eliminating future drift between the two implementations.

## 🧱 Tech Direction Snapshot
Preferred stacks: Go API with Connect-RPC and proto contracts; Go CLI built on `packages/cli-core`; React + Vite UI from the `vrooli-default` design kit.

Preferred storage: Qdrant collection `cli-health-commands` (role-resolved dense vectors, cosine similarity, payload-hash drift detection) for vector search; Ollama embeddings resolved through the `embedding.default` policy role. No scenario-local relational store is required in v1.

Integration strategy: `test-genie` delegates the Contracts phase to `cli-health` through the descriptor-backed provider RPC (`scenario-validation/v1.ScenarioValidationService.ValidateScenario`), not a CLI subprocess — the descriptor in `.vrooli/test-genie.json` declares the phase, its maturity ladder, and its `docs.path`, and the provider returns shared status plus `assessment.findings` and per-capability `MaturityAssessment` standings. The AI search pipeline is duplicated verbatim from `prompt-manager/api/aisearch/` for v1; extraction to a shared package is deferred per the duplicate-before-extract policy and tracked under OT-P2-002.

Non-goals: No natural-language-to-command UI (`prompt-manager` owns that surface). No runnable command snippet execution. No new fields added to `cli-manifest.schema.json` beyond the architecture-metadata surface (`architecture` / `exceptions[]`) that backs the `command_architecture` capability. No git mutations during validation. No extraction to a shared package on day 1. No backwards-compatibility shims for pre-manifest scenarios beyond the `manifest_missing` warning severity.

## 🤝 Dependencies & Launch Plan
Required resources: Ollama (with the `embedding.default` role available); Qdrant (`cli-health-commands` collection). Both must be available in the Tier 1 local stack before Phase 3 work begins.

Scenario dependencies: `prompt-manager` (source code reference only for AI search pipeline duplication; not a runtime dependency); `test-genie` (downstream consumer of the Contracts phase integration delivered in Phase 5).

Operational risks: The new Contracts phase could flood `test-genie` reports for scenarios that lack a manifest — mitigated by emitting `manifest_missing` at `severity=warning` in v1, not error, so adoption is encouraged without blocking existing pipelines. Initial reindex latency on a cold Qdrant instance could delay search availability — mitigated by the 5-minute async sync loop starting immediately on service boot.

Launch sequencing: Phase 0 (orient + PRD + requirements) → Phase 1 (proto contracts + CLI skeleton) → Phase 2 (validation service) → Phase 3 (AI search pipeline) → Phase 4 (React inspection UI) → Phase 5 (`test-genie` Contracts phase) → Phase 6 (four cross-scenario doc updates referencing `cli-health search`) → GA.

## 🎨 UX & Branding
User experience: Operational-console aesthetic consistent with the `vrooli-default` design kit. CLI output is terse and machine-parseable; findings are grouped by severity (error, warning, info) with scenario and rule identifiers on each line. The UI presents validation findings and search results in scannable lists with no extraneous decoration.

Visual design: `vrooli-default` design kit light and dark themes. Template-seeded favicons and `site.webmanifest` are retained as-is in v1; cli-health-specific imagery is deferred until a monetised-bundle deployment is in scope. Typography and spacing follow the standard Vrooli operational-console conventions.

Accessibility: WCAG AA compliance required. Full keyboard navigation across all interactive UI elements. Search result lists and validation finding panels must be screen-reader-friendly with appropriate ARIA roles and landmark regions. No accessibility regressions permitted relative to the `vrooli-default` design kit baseline.
