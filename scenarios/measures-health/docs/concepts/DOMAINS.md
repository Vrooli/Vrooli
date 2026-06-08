# Domains — Measures Health

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

> **Migration note.** The template `notes` worked-example domain has been
> removed (START-HERE Gate 7) from code, proto, and the `.vrooli` surface now
> that the real domains (`validation`, `search`) are green. `vrooli scenario
> orient` passes `example-domain-removed`. Some deeper internal docs
> (`SEAMS.md`, `ARCHITECTURE.md`, reference pages) still carry the template's
> notes-based worked examples — tracked as drift in
> [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md).

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature,
  CLI command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

System-level architecture belongs in [`ARCHITECTURE.md`](ARCHITECTURE.md).
Workflow details belong in [`FLOWS.md`](FLOWS.md). Storage details
belong in [`DATA.md`](DATA.md).

## What this scenario is

`measures-health` is a **meta / interface-enabler** scenario in the
Vrooli ecosystem. It does not own a product entity of its own; it
operates *over* every other scenario's declared **measures** (named,
typed, parameterized analytical queries — see the shared
[`packages/measures-go`](../../../../packages/measures-go/README.md)
contract library and `docs/concepts/MEASURES.md`). Its two jobs:

1. **Enforce + grade** measure adoption (the `validation` domain) —
   the producer that feeds the Ecosystem Manager maturity ladder a
   soft `measures` dimension, mirroring `security-health` /
   `cli-health`.
2. **Federate + answer** analytical questions (the `index` domain) —
   host the central measures index and the single registered
   `search-hub` "measure" provider, so a natural-language analytical
   question is matched to a declared measure, its params resolved, and
   (for safe read-only measures) executed and answered.

The reusable measure *brain* (matching, param resolution, the
auto-execution gate, execution proxy) lives in `packages/measures-go`
and is **reused, not re-authored**. This scenario supplies the seams:
the central aisearch-go index (`Matcher`), the ollama completer
(`Completer`/`ParamExtractor`), and the cross-scenario execution proxy
(`Executor` via search-hub's URL resolver).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/measures-health/v1/health/` |
| validation | Statically validate + behaviorally probe one scenario's measure adoption; emit the expected/covered/waived coverage report with per-measure tier and a pass/fail verdict; roll the fleet up for the UI. | Reporting / query (+ producer) | Harvest cache + last-report cache (SQLite); no canonical product entity. | API, CLI, UI | REQ-P0-001, REQ-P1-001, REQ-P1-002, REQ-P2-001 | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validate/`, `ui/src/features/fleet/`, `packages/proto/schemas/measures-health/v1/validation/` |
| index | Harvest every scenario's measure blocks, build the aisearch-go hybrid index over their `questions[]`, and serve the single registered search-hub "measure" provider query RPC (match → resolve → gate → execute via measures-go Engine). | RPC service (search provider) | Indexed measure declarations (qdrant vectors + SQLite catalog cache). | API, CLI | REQ-P0-002, REQ-P0-003 | `api/internal/index/`, `api/handlers/measuresearch/`, `cli/domains/index/`, `.vrooli/search.json`, `packages/proto/schemas/measures-health/v1/search/` |

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

### validation

- Purpose: answer "is this scenario *really* adopting measures, and how
  well?" Produce the expected/covered/waived coverage report with a
  per-measure tier (full / partial / fallback) and a pass/fail verdict.
  This is the producer test-genie shells into the EM ladder.
- Primary archetype: reporting / query; secondary trait: external
  producer (severity-graded findings, like `security-health`).
- Expectation model (copied from `cli-health` + `security-health`):
  - **EXPECTED** = the target scenario's *stateful domains*, derived
    primarily from `packages/proto/schemas/<scenario>/v1/domain/*.proto`
    (each domain proto == a persisted entity type == stateful by
    construction), then **substrate-filtered** to drop pure
    config/utility domains with no countable rows (→ INFO, never a
    gap). The manifest `measures.domains[]` override is the escape
    hatch for misclassification.
  - **COVERED** = a stateful domain has ≥1 manifest `measure` block
    bound to it.
  - **WAIVED** = a domain listed in `measures.omitted[]` with a reason.
  - Uncovered stateful domain → **ERROR**. Waiver pointing at a
    non-stateful / nonexistent domain → **WARNING** (stale).
- Behavioral adoption probe: for each declared measure, call its
  endpoint, assert the response conforms to the declared `result`
  shape, and round-trip one `questions[]` example end-to-end
  (match → extract → execute → plausible). A hollow declaration
  (404 / non-conforming) → **ERROR** — this is the "declared but not
  implemented" guard.
- Owns: the coverage report shape, the domain-derivation + substrate
  filter, tier grading, the behavioral probe, and the fleet rollup
  read model. Reuses `measures-go` for declaration assembly + the
  engine round-trip.
- Does not own: the measure declaration contract (that is
  `measures-go`), nor the search federation (that is `index`).
- API: `api/handlers/validation/` (`ValidationService.ValidateScenario`,
  `ListFleetCoverage`).
- CLI: `cli/domains/validate/` (`measures-health validate scenario
  <name>`, `measures-health coverage`).
- UI: `ui/src/features/fleet/` (the fleet coverage dashboard +
  per-scenario drill-down).
- Storage: SQLite caches for the harvest + last report; no canonical
  product entity.
- Requirements: REQ-P0-001 (validate CLI), REQ-P1-001 (behavioral
  probe), REQ-P1-002 (ValidateScenario RPC), REQ-P2-001 (fleet UI).
- Tests: classification table tests, tier-grading tests, substrate
  filter tests, behavioral-probe (hollow → error) test, e2e boot probe,
  UI feature + a11y tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

### index

- Purpose: be the **central measures index** and the **single
  registered search-hub "measure" provider**. Harvest every scenario's
  manifest `measure` blocks, embed their `questions[]` via the
  `measures-go` `MeasureComposer`, index them with an `aisearch-go`
  hybrid engine (qdrant + ollama), and serve the provider query RPC
  that search-hub calls — returning hits whose `measure` carrier
  (`SearchHit.measure`, shipped Phase 3) holds the resolved answer or
  the `needs[]`/confirmation signal.
- Primary archetype: RPC service (search provider). Mirrors how
  `cli-health` serves its own `SearchService.Search` and registers via
  `.vrooli/search.json` + `searchregister-go`.
- Auto-execution contract (enforced by `measures-go` `Gate`, never
  re-implemented here): execute iff `confidence ≥ θ` AND `effect ==
  read` AND `run_eligible`; `write`/`destructive` always return
  resolved-but-unexecuted with a confirmation signal. θ is a config
  lever (`MEASURES_HEALTH_CONFIDENCE_THRESHOLD`).
- Owns: the harvest-all-scenarios source, the index lifecycle, the
  provider query RPC, and the search-hub registration descriptor.
  Reuses `measures-go` `Engine`/`Gate`/`MeasureComposer`/`LLMExtractor`
  /`HTTPExecutor` and `aisearch-go` for the index.
- Does not own: the router/classification (that stays thin in
  search-hub), nor cross-scenario URL computation (uses
  `api-core/discovery`, never client-computed).
- API: `api/handlers/measuresearch/` (the provider query
  `SearchService.Search`), plus the shared token-gated
  `search-hub.v1.control.SearchControlService` for reindex/config.
- CLI: `cli/domains/index/` (`measures-health index reindex`,
  `measures-health index list`).
- Storage: qdrant collection `measures-health-measures` + a SQLite
  catalog cache of harvested declarations.
- Requirements: REQ-P0-002 (harvest + index), REQ-P0-003 (provider
  registration + query RPC).
- Tests: harvest test (fixture scenarios), composer/index test, an
  end-to-end query test through the engine, registration descriptor
  test.
- Related docs: [`INTEGRATIONS.md`](INTEGRATIONS.md),
  [`../reference/ai-search-routing.md`](../../../../docs/reference/ai-search-routing.md).

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

Add future or intentionally deferred capabilities here only when they
are real enough to affect architecture or requirements.

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| None yet. | Generated scaffold. | Add after PRD-specific requirements identify future capability boundaries. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product
piece into an owning domain instead of growing infrastructure.

## Cross-References

- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
