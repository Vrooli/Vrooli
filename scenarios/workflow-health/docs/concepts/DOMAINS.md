# Domains — Workflow Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

Workflow Health owns workflow policy and intelligence, not browser automation
runtime internals. Browser Automation Studio remains the execution engine;
Test Genie remains the orchestrator; Search Hub remains the query surface.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Responsibility | Purpose | Owns Data | Primary Archetype | Secondary Traits | Glossary | Source Paths |
|---|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Expose API/database readiness and show the UI can read live backend state. | No product data. | reporting | query | HealthHandler | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/workflow-health/v1/shared/health.proto` |
| catalog | Scan and normalize BAS workflow assets. | Give validation, search, execution, fixes, and UI one deterministic asset model. | Catalog snapshots, asset facts, dependency edges, stale registry facts. | aggregation | validation, query | WorkflowAsset, WorkflowCase, WorkflowFlow, WorkflowAction, SeedContract, DependencyEdge | `api/internal/workflows/`, `cli/domains/workflows/`, `ui/src/features/inventory/` |
| validation | Convert catalog facts into findings, metrics, maturity, and remediation. | Make workflow quality consumable through `ScenarioValidationService`. | Validation runs, finding summaries, maturity assessments, fix previews. | validation | provider, scoring | Finding, MaturityAssessment, FixPreview | `api/internal/validation/`, `api/handlers/validation/`, `.vrooli/maturity.json` |
| execution | Orchestrate safe BAS-backed workflow execution. | Preserve end-to-end workflow evidence while preventing unsafe mutation. | Run records, artifact pointers, timeline summaries, safety verdicts. | orchestration | service, validation | WorkflowRun, SafetyProfile, ArtifactRef | `api/internal/execution/`, `api/internal/artifacts/`, `ui/src/features/runs/` |
| search | Publish workflow leaves for AI/action discovery. | Let agents find safe runnable flows without confusing cases and fragments. | Search leaf metadata, ranking hints, last-run signals. | provider | classification, query | WorkflowFlowLeaf, WorkflowTestLeaf, WorkflowFragmentLeaf | `api/internal/search/`, `cli/domains/workflows/`, `ui/src/features/search/` |
| remediation | Preview and apply deterministic workflow repairs. | Fix mechanical drift without silently changing browser behavior. | Fix plans, applied fix records, diff summaries. | mutation | validation | WorkflowFix, FixPlan | `api/internal/fixes/`, `ui/src/features/fixes/` |
| operator-ui | Present workflow-health posture to operators and future BAS checks. | Make catalog, validation, search, run, finding, and fix state inspectable from the first screen. | Client route/filter state and view models; durable data remains in owning domains. | interface | query, review | Overview, Inventory, WorkflowSearch, RunTimeline, FindingTable, FixPreview | `ui/src/pages/`, `ui/src/layout/`, `ui/src/consts/` |

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
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.
- Related docs: [`../reference/api-endpoints.md`](../reference/api-endpoints.md).

### catalog

- Purpose: scan scenario-owned BAS assets into stable, typed workflow records.
- Primary archetype: aggregation.
- Secondary traits: validation, query, dependency graph extraction.
- Owns: asset IDs, canonical path normalization, role classification, metadata extraction, selector/route/requirement references, seed declarations, and registry freshness checks.
- Does not own: BAS execution semantics or Test Genie phase orchestration.
- API: `api/internal/workflows/`.
- CLI: `cli/domains/workflows/`.
- UI: inventory tables and filters.
- Storage: catalog snapshots and stale registry facts when persistence is introduced.
- Requirements: `REQ-P0-001`, `REQ-P0-002`.
- Tests: scanner fixture/golden tests across cases, flows, actions, seeds, legacy/current JSON shapes, and stale registry states.

### validation

- Purpose: make workflow health a delegated validation-provider contract.
- Primary archetype: validation.
- Secondary traits: provider, scoring.
- Owns: finding catalog, severity gates, maturity ladder, validation response construction, native detail packing, and fix routing.
- Does not own: browser execution implementation or Test Genie report formatting.
- API: `api/handlers/validation/`, `api/internal/validation/`.
- CLI: `workflow-health validate scenario`, `workflow-health fix preview`, and `workflow-health fix apply`.
- UI: findings, maturity, and fix preview surfaces.
- Requirements: `REQ-P0-003`, `REQ-P0-004`, `REQ-P2-001`.
- Current implementation: `api/internal/validation` consumes the catalog scanner and emits stable rule IDs for absent surfaces, missing/stale registries, parse errors, incomplete metadata, missing requirement links, direct selectors, unresolved subflows, invalid execution modes, legacy reset values, mutating safety gaps, and missing seed/fixture setup. `api/handlers/validation` mounts the shared `scenario-validation/v1` provider RPCs and packs native catalog/execution summaries in `native_detail`.
- Maturity: `.vrooli/maturity.json` declares the local ladder `L0` no surface, `L1` discoverable, `L2` traceable, `L3` safe, `L4` executable, and `L5` operationally rich. The current provider covers static `L0` through `L3` gates and can force failed status when BAS-backed execution refuses or fails selected workflows.
- Deterministic fixes: registry rebuild, metadata stub fill, invalid `execution_mode` normalization to `observer`, and legacy `reset=database` normalization to `reset=full`. Requirement links, selector registry choices, unresolved subflows, mutating safety, and seed design remain manual because they affect product behavior or safety.

### execution

- Purpose: execute selected validation cases through BAS with fail-closed safety.
- Primary archetype: orchestration.
- Secondary traits: service, validation.
- Owns: execution options, BAS client adapter, run records, artifact ingestion, safety verdicts, and routed isolation proofs.
- Does not own: browser engine internals, Playwright-level behavior, or Search Hub ranking.
- Requirements: `REQ-P0-005`, `REQ-P0-006`.

### search

- Purpose: expose workflows to agents and operators by intent.
- Primary archetype: provider.
- Secondary traits: classification, query.
- Owns: `workflow.flow`, `workflow.test`, and `workflow.fragment` leaf construction, safety labels, ranking hints, and search CLI/API shape.
- Does not own: global Search Hub ranking infrastructure.
- API: `WorkflowSearchService.SearchWorkflows` in `packages/proto/schemas/workflow-health/v1/workflows/workflows.proto`, mounted by `api/handlers/workflows`.
- CLI: `workflow-health workflows search "<query>" --scenario <id> --type workflow.flow`.
- Current implementation: deterministic catalog-backed search in `api/internal/search`; default results include flow and test leaves, while fragments stay hidden unless explicitly requested. Run/do intents prefer runnable flows, validate/prove intents prefer validation cases, and mutating results are returned with explicit guardrails instead of being auto-runnable.
- Requirements: `REQ-P1-001`.

### remediation

- Purpose: repair deterministic workflow asset drift.
- Primary archetype: mutation.
- Secondary traits: validation.
- Owns: preview/apply plans for registry rebuilds, metadata normalization, execution-mode normalization, and reset metadata.
- Does not own: semantic browser workflow rewrites.
- Requirements: `REQ-P1-002`.

### operator-ui

- Purpose: expose workflow-health as a production operational surface instead of a generated template shell.
- Primary archetype: interface.
- Secondary traits: query, review.
- Owns: route layout, navigation, selector coverage, localized page copy, static view models for the current UI slice, responsive states, and accessibility of the operator surface.
- Does not own: workflow catalog truth, validation logic, search ranking, execution safety, or fix semantics.
- UI: overview, inventory, search, runs, findings, fixes, and settings routes under `ui/src/pages/`.
- Current implementation: replaces the starter dashboard/notes route with dense workflow-health pages for scenario selection, maturity cards, asset catalog tables, typed search results, run timelines, finding remediation, and fix preview/apply affordances. API-backed hooks and BAS screenshot validation remain rollout hardening work.
- Requirements: `REQ-P1-003`, `REQ-P5-001`.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Workflow asset | Any file or registry entry under `bas/` that workflow-health can classify. | catalog domain. |
| Validation case | A `bas/cases` asset used as evidence for requirements and Test Genie phases. | catalog and validation domains. |
| Agent flow | A `bas/flows` asset intended for discovery and guarded operator/agent execution. | catalog and search domains. |
| Fragment | A `bas/actions` asset used as a reusable dependency, hidden from default top-level search results. | catalog and search domains. |
| Safety profile | Execution metadata that proves observer/mutating behavior, confirmations, seeds, resets, and routed isolation. | execution domain. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| Fleet analytics | Roll up workflow maturity across many scenarios. | After provider contract, search, execution, and UI are stable for representative scenarios. |
| Policy packs | Versioned reusable rule profiles for workflow standards. | After the first maturity ladder proves which rules belong in workflow-health versus BAS. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.
- Browser Automation Studio runtime services — workflow-health calls BAS; it does not absorb its engine.
- Test Genie phase orchestration — workflow-health provides validation; Test Genie owns suite execution.
- Search Hub indexing infrastructure — workflow-health provides leaves; Search Hub owns query federation.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
