# Domains — Flow Verifier

This document is the canonical map of product capabilities, bounded
contexts, and ownership for this scenario. Keep it current whenever a
domain is added, renamed, split, merged, or removed.

`notes` is a worked example carried over from the template; it will be
removed during Gate 7 once the four real flow-verifier domains
(`flows`, `verification`, `codegen`, `runs`) and the orchestrator
(`pipeline`) are green.

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
| flows | Discover, parse, and model flow/flow.json contracts on disk. | Discovery / query | None (filesystem-truth). | API, CLI, UI | DISC-001, DISC-002 | `api/internal/flows/`, `cli/domains/flows/`, `ui/src/features/inventory/` |
| verification | Run Quint typecheck/test/verify/run, lint hand-test wrappers, normalize ITF traces. | Pipeline step | None (transient artifacts). | API, CLI | VER-001, VER-002, VER-003 | `api/internal/verification/`, `cli/domains/verify/` |
| codegen | Emit `runtime.{ts,go}`, `replay.helper.{ts,go}`, `model.qnt` from a typed Flow + Artifact. | Transformation | None (pure functions). | (internal only) | VER-002 | `api/internal/codegen/` |
| runs | Persist a row per verification: status, hashes, counterexample, duration. | Repository / store | `verification_runs` SQLite table. | API, CLI | RUNS-001, RUNS-002, RUNS-003 | `api/internal/runs/`, `initialization/sqlite/migrations/`, `cli/domains/runs/` (planned) |
| pipeline | Orchestrate compile → artifact → codegen → lint → persist for one flow. | Orchestrator | None (composes flows + verification + codegen + runs). | (internal only) | VER-001, VER-002, RUNS-002 | `api/internal/pipeline/` |
| health | Report runtime readiness and dependency reachability. | Reporting / query | None. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/` |
| notes (template example) | Worked CRUD reference with attachment upload exception. | CRUD / entity | Notes and attachment metadata. | API, CLI, UI | Template starter only; removed at Gate 7. | `api/internal/notes/`, `cli/domains/notes/`, `ui/src/features/notes/` |

## Domain Details

### flows

- Purpose: own everything about *discovering and modeling* a flow on disk. Walks the repo for `flow/flow.json`, parses + validates against the embedded schema, compiles into a typed `Flow`, and emits the deterministic per-flow output layout (`generated/...`). Also owns the `flow-verifier flows new` scaffolder.
- Primary archetype: discovery / query.
- Owns: `Flow`, `Contract`, `TransitionMatrix`, `Layout`, the embedded JSON-Schema files, the scaffold templates.
- Does not own: Quint invocation, codegen rendering, persistence.
- API: `GET /api/v1/flows`, `GET /api/v1/flows/:id`.
- CLI: `flow-verifier flows list|validate|new|explain`.
- UI: Flow Inventory page consumes `/api/v1/flows`.
- Storage: filesystem only; no DB.
- Tests: package-level unit tests preserved from `tools/temporal-model/internal/{contract,discovery,compile,model,layout,scaffold}/_test.go`; new tests for `embed.FS` schema loading.

### verification

- Purpose: own the Quint invocation surface, the formal-artifact hash chain, and the lint gate that rejects malformed hand-test wrappers.
- Primary archetype: pipeline step.
- Owns: `quint.Runner`, `quint.Render`, ITF trace normalization, `artifact.Build`, AST-level Go + structural-regex TS lint.
- Does not own: discovery, codegen rendering, run history.
- API: `POST /api/v1/verifications`, `GET /api/v1/verifications/:runId`.
- CLI: `flow-verifier verify run|check`.
- Tests: preserved from `tools/temporal-model/internal/{quint,artifact,lint}/_test.go`.

### codegen

- Purpose: render the canonical Go and TypeScript runtime + replay helpers + Quint source for one flow.
- Primary archetype: pure transformation.
- Owns: code-emission templates for both languages; module-path detection so generated imports resolve correctly across scenarios.
- Does not own: discovery, Quint invocation, persistence.
- Surfaces: internal only; consumed by `pipeline`.
- Tests: golden-file assertions migrated from `tools/temporal-model/internal/codegen`.

### runs

- Purpose: durable history of verification outcomes; the trail of what verifying flows produced.
- Primary archetype: repository / store.
- Owns: `verification_runs` table, `Insert`, `LatestByFlow`, `ListByFlow`, `Get`.
- Does not own: flow content (flows are filesystem-truth).
- API: `GET /api/v1/runs`, `GET /api/v1/runs/:id`.
- CLI: `flow-verifier runs list|show`.
- Storage: SQLite via `modernc.org/sqlite` (pure-Go); single migration `0001_verification_runs.sql`.
- Tests: in-memory SQLite (`file::memory:?cache=shared`) repository tests; handler tests.

### pipeline

- Purpose: orchestrate the per-flow execution. Modes `ModeGenerate` (write artifacts) and `ModeCheck` (assert freshness + lint).
- Primary archetype: orchestrator.
- Owns: nothing on its own; sequences calls into `flows`, `verification`, `codegen`, `runs`.
- Surfaces: internal only; the CLI and HTTP handlers invoke `pipeline.Run`.

### health (carried from template)

- Purpose: expose API/database readiness and show the UI can read live backend state.
- API: `api/handlers/health/`. UI: `ui/src/features/health/HealthCard.tsx`.

### notes (template example, slated for removal)

- Purpose: demonstrate the expected vertical slice for a real domain. Removed during Gate 7 once the real flow-verifier domains are green.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Domain | Product capability boundary that should be easy to find, test, and delete. | `DOMAINS.md` defines the map; code owns implementation. |
| Surface | API, UI, CLI, or contract layer exposing the same product capability. | `ARCHITECTURE.md`. |
| Seam | Test-substitutable boundary wired once in production. | `../internal/SEAMS.md`. |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |
| Flow | A `flow/flow.json` (schema v6) + hand-authored wrapper + generated artifacts under `generated/`. | `flows` domain. |
| Verification run | One pass of compile → artifact → codegen → lint for one flow. Persisted as a `verification_runs` row. | `pipeline` orchestrator; `runs` stores the trail. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| state-graph visualizer | P1 UI; planned after MVP inventory. | When OT-P1-001 is scheduled. |
| trace player | P1 UI; needs counterexample ingestion. | When OT-P1-002 is scheduled. |
| counterexample diff | P1 UI. | When OT-P1-003 is scheduled. |
| verification timeline | P1 chart; reads `runs` data only — no new domain. | When OT-P1-004 is scheduled. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/handlers/` — thin presentation glue; one file per domain.
- `api/internal/spec/` — version + tool name constants shared across domains.
- `api/internal/testkit/` — test-only fixtures and fake runners.
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
