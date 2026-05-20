# Product Requirements Document (PRD)

## 🎯 Overview
Purpose: Framework-agnostic UI manifest validation and UI-surface AI search for Vrooli scenarios. `ui-health` is the single source of truth for (a) verifying that every scenario's `ui/manifest.json` conforms to the `scenario-ui-manifest/v1` schema, its on-disk slot directories, and the template-vs-overlay rules; (b) giving agents a semantic search surface to discover existing UI components, pages, features, and hooks across all Vrooli scenarios before duplicating them; and (c) owning the two cross-cutting contracts — `ComponentProvenance` and `WidgetDeclaration` — that every UI-producing component-library scenario must conform to.

Target users: Vrooli agents (test-genie phase orchestrator, agent-inbox unified retrieval, skill-authoring); Vrooli humans authoring scenarios; `scenario-auditor` as a downstream consumer of validation findings.

Deployment surfaces: Tier 1 local stack only for v1. Headless API and CLI consumed by other scenarios; React UI available for human inspection. No cloud or multi-tenant deployment in scope for v1.

Value proposition: Catches UI manifest drift, missing slot directories, and orphan files before they ship; replaces ad-hoc grepping for existing UI surfaces with semantic search backed by manifest, provenance, and proto descriptors; defines the canonical provenance and widget contracts so future framework-library scenarios (vue, svelte, python-tui) plug in cleanly via a single Connect-RPC interface.

## 🎯 Operational Targets

Operational targets are measurable outcomes; checkboxes may auto-update based on validation.

### 🔴 P0 – Must ship for viability
- [ ] OT-P0-001 | Deterministic manifest validation | `ui-health validate scenario <name> --json` validates `ui/manifest.json` + overlay against the `scenario-ui-manifest/v1` JSON schema, the on-disk slot directories, and the template-vs-overlay rules; emits a structured `Report` with `Finding`s for schema violations, missing slot directories, unknown slot names in overlays, missing template references, orphan files, and overlapping slot pathPatterns; exits 0 on a clean scenario.
- [ ] OT-P0-002 | Semantic UI-surface search | `ui-health search query "<text>"` returns ranked UI-surface results (components, pages, features, hooks) across all adopting scenarios in AI mode, with `ComponentProvenance` fields (CUSTOM / ADOPTED_UNMODIFIED / ADOPTED_MODIFIED / UNKNOWN), library, library_version, source_sha256, and drift_hash where applicable; `--mode text` falls back to substring matching when Ollama or Qdrant is unavailable.
- [ ] OT-P0-003 | On-demand reindexing with framework dispatch | `ui-health reindex run` walks every scenario, dispatches `InventoryService.ScanScenario` to the appropriate per-framework component-library scenario (today: `react-component-library`), aggregates `ScanResponse`, embeds each `SurfaceRecord` via Ollama `nomic-embed-text`, and upserts to Qdrant collection `ui-health-surface`.
- [ ] OT-P0-004 | ComponentProvenance contract is single source of truth | `ComponentProvenance` proto in `packages/proto/schemas/ui-health/v1/contracts/provenance.proto` defines the canonical adoption-metadata shape. The SQLite store in `react-component-library` is authoritative; the JSDoc block on adopted source files is the heal-from signal. Both round-trip losslessly with the proto message.
- [ ] OT-P0-005 | test-genie phase_ui_health integration | `test-genie` registers a new `phase_ui_health` (after `phase_contracts`) that shells out to `ui-health validate scenario <name> --json` and surfaces findings as Observations.

### 🟠 P1 – Should have post-launch
- [ ] OT-P1-001 | Drift detection between SQLite and JSDoc | `ui-health validate scenario <name>` reports a finding whenever a file's JSDoc provenance block disagrees with the authoritative SQLite adoption row (drift_hash mismatch or missing-on-either-side).
- [ ] OT-P1-002 | Reindex job lifecycle verbs | `ui-health reindex status <job_id>` and `ui-health reindex cancel <job_id>` expose structured progress and termination for long-running reindex jobs.
- [ ] OT-P1-003 | WidgetDeclaration contract defined | `WidgetDeclaration` proto in `packages/proto/schemas/ui-health/v1/contracts/widget.proto` is fully defined with field semantics documented; a per-framework JSDoc tag (`@vrooliWidget` block) lets components opt in to being embedded as a chat widget. No consumer ships in v1; agent-inbox unified retrieval is the eventual consumer. Documented in this PRD as a deliverable so it survives in the requirements registry.
- [ ] OT-P1-004 | UI inspection surface | A React UI built on the `react-vite` template + `react-component-library` adoptions presents a validation-results view (per-scenario findings table) and a search view (free-text + provenance filter).

### 🟢 P2 – Future / expansion
- [ ] OT-P2-001 | Framework dispatch resolver is data-driven | The mapping from `service.json` template id → component-library scenario lives in ui-health's `service.json` (not hardcoded in code), so new frameworks plug in via configuration.
- [ ] OT-P2-002 | Widget standard consumer integration | Agent-inbox unified retrieval (or successor) consumes `WidgetDeclaration` to render component widgets inline. ui-health remains contract + index only; runtime ownership lives downstream.

## 🧱 Tech Direction Snapshot
Preferred stacks: Go API with Connect-RPC and proto contracts; Go CLI built on `packages/cli-core`; React + Vite + Tailwind UI from the `vrooli-default` design kit, with adoptions from `react-component-library`.

Preferred storage: Qdrant collection `ui-health-surface` (768-dimensional vectors, cosine similarity, payload-hash drift detection) for vector search; Ollama serving `nomic-embed-text` for embedding generation. No scenario-local relational store inside ui-health itself; the authoritative adoption SQLite store remains in `react-component-library`.

Integration strategy: All cross-scenario invocation goes over Connect-RPC. `ui-health` imports `react-component-library`'s generated Connect client. `react-component-library` imports ui-health's proto contract messages (`ComponentProvenance`, `WidgetDeclaration`). The `InventoryService` interface is defined in ui-health's proto and implemented by per-framework component-library scenarios — one-way compile-time dependency, runtime call over RPC.

Non-goals: No widget renderer or agent-inbox-side widget code (contracts only). No framework-specific parsing inside `scenarios/ui-health/` (lives behind `InventoryService.ScanScenario` in component-library scenarios). No shared `uimanifest` Go package (two consumers does not justify extraction). No compatibility shims for the `react-component-library` JSDoc → SQLite shape change (greenfield). No CLI verbs beyond those listed in operational targets. No `make breaking` invocations during feature work.

## 🤝 Dependencies & Launch Plan
Required resources: Ollama (serving `nomic-embed-text`); Qdrant (collection `ui-health-surface`). Both must be available in the Tier 1 local stack before Phase 3 work begins.

Scenario dependencies: `react-component-library` (implements `InventoryService.ScanScenario` for React scenarios; receives the JSDoc-emitter and SQLite-store updates that align it with the `ComponentProvenance` proto). `test-genie` (downstream consumer of the new `phase_ui_health`).

Operational risks: Qdrant or Ollama unavailable during reindex — mitigated by structured error responses and a clear "index empty" finding rather than 500s. JSDoc/SQLite drift between authoritative store and on-disk heal-from signal — surfaced as a first-class finding rather than an error. Circular proto-import temptation — avoided by keeping ui-health's proto messages flow one-way into rcl, with the service interface consumed only over the network.

Launch sequencing: Phase 1 (scaffolding + PRD + requirements) → Phase 2 (proto contracts) → Phase 3 (ui-health API) → Phase 4 (`react-component-library` contract conformance) → Phase 5 (`test-genie` phase_ui_health) → Phase 6 (CLI + UI) → cleanup + verification.

## 🎨 UX & Branding
User experience: Operational-console aesthetic consistent with the `vrooli-default` design kit. CLI output is terse and machine-parseable; findings are grouped by severity (error, warning, info) with scenario and rule identifiers on each line. The UI presents validation findings and search results in scannable lists with provenance badges (Custom / Adopted / Adopted-Modified) color-coded.

Visual design: `vrooli-default` design kit light and dark themes. Template-seeded favicons and `site.webmanifest` are retained as-is in v1; ui-health-specific imagery is deferred until a monetised-bundle deployment is in scope. Typography and spacing follow the standard Vrooli operational-console conventions.

Accessibility: WCAG 2.1 AA compliance required. Full keyboard navigation across all interactive UI elements. Search result lists and validation finding panels must be screen-reader-friendly with appropriate ARIA roles and landmark regions. No accessibility regressions permitted relative to the `vrooli-default` design kit baseline.
