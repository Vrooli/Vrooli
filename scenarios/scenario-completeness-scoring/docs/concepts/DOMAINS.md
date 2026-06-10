# Domains — Scenario Completeness Scoring

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`notes` is a worked example from the template, not product scope.
Replace it after the first real domain is green.

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

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| signals | Collect raw completeness signals from a target scenario's cached on-disk artifacts (requirements registry, sync metadata, phase results, service manifest, UI sources) behind circuit-breaker-guarded collectors. | Collection / read model | No persisted data; in-memory signal snapshot per request. | internal (consumed by scoring) | OT-P0-006 (resilient collection), feeds OT-P0-002/003. | `api/internal/signals/` |
| freshness | Compute the target scenario's current tree digest and per-phase fresh/stale/unknown verdicts from `coverage/runs.index.json` via `packages/freshness-go`. | Reporting / query | No persisted data. | internal (consumed by scoring) | OT-P0-005 (staleness honesty). | `api/internal/freshness/` |
| scoring | Assemble signals into the maturity rung (maturity-go ladder), 0–100 composite, classification, recommendations with point impact, and the action plan; owns the ScoreService wire contract. | Reporting / query | No persisted data; scores computed on demand. | API, CLI, UI | OT-P0-001..006, OT-P1-002/003, OT-P2-001/002. | `api/internal/scoring/`, `api/handlers/scoring/`, `cli/domains/scores/`, `ui/src/features/scoring/`, `packages/proto/schemas/scenario-completeness-scoring/v1/scoring/` |
| importance | Best-effort importance enrichment from scenario-dependency-analyzer centrality and swarm-manager recency under a hard 1s combined budget; silently omitted on miss. | Integration / enrichment | No persisted data. | internal (consumed by scoring output) | OT-P1-001. | `api/internal/importance/` (deferred to the importance pass) |
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/scenario-completeness-scoring/v1/health/` |
| notes | Worked CRUD reference with attachment upload exception. **Scheduled for removal at Gate 7** once the scoring slice is green. | CRUD / entity | Notes and attachment metadata. | API, CLI, UI | Template starter only. | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/scenario-completeness-scoring/v1/notes/` |

## Domain Details

### signals

- Purpose: produce a `Signals` snapshot for a target scenario from cached
  filesystem artifacts only — never by running tests or calling services.
- Primary archetype: collection / read model.
- Owns: the collector registry and its circuit-breaker policy; decoders for
  `requirements/index.json` (+ imported modules), requirements-sync metadata
  (`coverage/requirements-sync/latest.json` and fallbacks),
  `coverage/phase-results/*.json` findings (proto `ArchitectureFinding`
  decode with the legacy summary-string fallback), `.vrooli/service.json`,
  and the UI source heuristics (template detection, component/page counts,
  routing, API usage, LOC).
- Does not own: score math, freshness verdicts, the wire contract.
- Dimension mapping: phase findings map onto the shared improvement-dimension
  vocabulary from `packages/maturity-go/dimensions`.
- Degradation: a collector that fails repeatedly is disabled by its circuit
  breaker; its weight is redistributed and the degradation is reported in the
  score payload (OT-P0-006).
- Storage: none.
- Tests: per-collector table tests over fixture artifact directories
  (well-formed / malformed / missing).

### freshness

- Purpose: label every number the scenario reports with the tree digest it
  was computed against and per-phase fresh/stale/unknown verdicts plus a
  copy-pastable refresh command.
- Primary archetype: reporting / query.
- Owns: digest computation + verdict assembly calls into
  `packages/freshness-go` (treedigest, runindex read, verdict core) and the
  conversion to the wire shape. The freshness semantics themselves (required
  phase set, verdict logic, suggested-command format) are owned by the shared
  package and reused verbatim — no local policy.
- Does not own: writing `coverage/runs.index.json` (test-genie's write side),
  per-scenario freshness configuration (rejected by design).
- Storage: none.
- Tests: integration test against a temp scenario dir with a real digest and
  synthetic run index; never-tested scenarios yield "unknown", not fresh.

### scoring

- Purpose: the product surface — assemble signals + freshness into the rung
  headline (R0–R4 via `packages/maturity-go/ladder`), composite 0–100 with
  classification, per-dimension breakdown, prioritized recommendations with
  point impact, and the phased action plan.
- Primary archetype: reporting / query.
- Owns: `ScoreService` proto contract, score math (ported from the legacy
  implementation and re-based on the wider signal set), classification bands,
  recommendation/action-plan generation, and the assembled response.
- Does not own: raw artifact decoding (signals), digest/verdict logic
  (freshness), live maturity state (ecosystem-manager computes its own rungs
  in-process on live findings; this domain labels its rung "as of digest
  td:…").
- API: `api/handlers/scoring/`; CLI: `cli/domains/scores/`; UI:
  `ui/src/features/scoring/`.
- Storage: none — no score history in v1 (explicit non-goal).
- Tests: scoring table tests pinning rung derivation and classification
  boundaries, CLI golden-output test, perf assertion (<1s warm).

### importance (deferred to the importance pass)

- Purpose: optional one-line enrichment (reverse-dependency centrality,
  core-set proximity, recent activity) appended to score output.
- Constraint: the ONLY network touch in the scenario; hard 1s combined
  budget; silently omitted on any miss. Core score path stays zero-network.

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

### notes

- Purpose: demonstrate the expected vertical slice for a real domain.
- Primary archetype: CRUD / entity.
- Secondary traits: binary/blob attachment upload, upload workflow.
- Owns: note records, attachment metadata, note validation, note
  service/repository seams, UI note interactions, CLI notes commands.
- Does not own: product scope for a generated scenario.
- API: `api/internal/notes/`, `api/handlers/notes/`.
- CLI: `cli/domains/notes/`.
- UI: `ui/src/features/notes/`, `ui/src/api/notes.ts`.
- Storage: domain-owned SQLite schema in `api/internal/notes/schema.sql`.
- Requirements: template starter only; replace with PRD-specific
  requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and
  workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).

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
| importance | P1 enrichment; depends on the dep-analyzer centrality endpoint shipping (separate plan phase). | dep-analyzer `graph centrality` lands. |
| history / trends | Explicit v1 non-goal (no score persistence). | A consumer demonstrates need for trend data. |
| what-if analysis | P2; port from legacy only if cheap on the new signal set. | Demand from an agent workflow. |

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
