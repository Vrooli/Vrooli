# Domains — Plan Manager

This document is the canonical map of product capabilities, bounded contexts, and
ownership for this scenario. Keep it current whenever a domain is added, renamed,
split, merged, or removed.

The scaffold ships `health` plus one clearly fenced worked example domain (`notes`,
never product scope). The four real domains below — `plans`, `authoring`,
`execution`, `validation` — map to the three engines from the founding design
(Composer = authoring, Runner = execution, Ledger = validation) over the plan
**store**. `vrooli scenario detemplate plan-manager` removes the fenced example
once the real domains are green.

## Purpose Of This Document

Use this document to answer:

- What product capabilities does this scenario expose?
- Which domain owns each concept, table, proto, endpoint, UI feature, CLI
  command, and test surface?
- Which concepts are shared, deferred, or deliberately not domains?

The structured plan/phase schema every domain operates on lives in
[`PLAN-MODEL.md`](PLAN-MODEL.md). System-level architecture belongs in
[`ARCHITECTURE.md`](ARCHITECTURE.md). Workflow details belong in
[`FLOWS.md`](FLOWS.md). Storage details belong in [`DATA.md`](DATA.md).

## Domain Inventory

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| health | Report runtime readiness and dependency reachability. | Reporting / query | No product data. | API, UI | Starter scaffold health. | `api/handlers/health/`, `ui/src/features/health/`, `packages/proto/schemas/plan-manager/v1/shared/health.proto` |
| plans | Own the structured plan + phase record (SSOT): CRUD, markdown render, lifecycle, supersession graph, templates. | CRUD / entity | Plans, phases, references, supersession edges. | API, CLI, UI | PM-STORE-001/002, PM-GRAPH-001 | `api/internal/plans/`, `api/handlers/plans/`, `cli/domains/plans/`, `ui/src/features/plans/`, `packages/proto/schemas/plan-manager/v1/plans/` |
| authoring | Guided composer wizard: section-by-section flow, structure-validation gate, autofill of mechanical sections. | Workflow / guided process | Authoring-session state; validation findings. | API, CLI, UI | PM-AUTHOR-001/002 | `api/internal/authoring/`, `api/handlers/authoring/`, `cli/domains/authoring/`, `ui/src/features/authoring/`, `packages/proto/schemas/plan-manager/v1/authoring/` |
| execution | Guided runner: phase transitions, just-in-time context injection, in-flow decision/finding capture, thin completion + canonical handoff, velocity. | Workflow / state machine | Execution/run linkage, handoff records, candidate findings, velocity series. | API, CLI, UI | PM-EXEC-001/002, PM-HANDOFF-001/002, PM-VEL-001 | `api/internal/execution/`, `api/handlers/execution/`, `cli/domains/execution/`, `ui/src/features/execution/`, `packages/proto/schemas/plan-manager/v1/execution/` |
| validation | Plan health: code-reference resolution, staleness tiers, baseline-scope derivation, baseline/check orchestration, DoD verification. | Verification / analysis | Reference resolutions, validation results, staleness factors. | API, CLI, UI | PM-REF-001, PM-STALE-001, PM-VALID-001/002 | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validation/`, `ui/src/features/validation/`, `packages/proto/schemas/plan-manager/v1/validation/` |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `vrooli scenario detemplate`)

The template ships `notes` as a worked CRUD vertical slice with a binary upload
exception. Copy its shape for your own domains, then remove it.

| Domain | Purpose | Primary Archetype | Owns Data | Surfaces | Requirements | Source Paths |
|---|---|---|---|---|---|---|
| notes | Worked CRUD reference with attachment upload exception. | CRUD / entity | Notes and attachment metadata. | API, CLI, UI | Template starter only. | `api/internal/notes/`, `api/handlers/notes/`, `cli/domains/notes/`, `ui/src/features/notes/`, `packages/proto/schemas/plan-manager/v1/notes/` |

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
- Requirements: template starter only; replace with PRD-specific requirements.
- Tests: repository, service, handler, CLI, UI, accessibility, and workflow tests.
- Related docs: [`FLOWS.md`](FLOWS.md), [`DATA.md`](DATA.md),
  [`../internal/SEAMS.md`](../internal/SEAMS.md).
<!-- EXAMPLE-DOMAIN:notes END -->

## Domain Details

### health

- Purpose: expose API/database readiness and show the UI can read live backend state.
- Primary archetype: reporting / query.
- Owns: health response construction and dependency status mapping.
- Does not own: product data, business rules, or scenario-specific behavior.
- API: `api/handlers/health/`. CLI: built-in `status` via cli-core. UI: `ui/src/features/health/HealthCard.tsx`.
- Storage: none; probes configured database reachability.
- Requirements: starter scaffold health only.
- Tests: handler, module, UI feature, and accessibility tests.

### plans

- Purpose: own the structured plan + phase record as the single source of truth (see [`PLAN-MODEL.md`](PLAN-MODEL.md)) — CRUD, the rendered markdown view, lifecycle (`draft`/`active`/`complete`/`archived`), the supersession/dependency graph, and per-surface plan templates.
- Primary archetype: CRUD / entity.
- Owns: plan + phase records, references, supersession edges, content hashes.
- Does not own: the authoring flow, execution, or validation logic — those operate *on* plans but live in their own domains. Persistence substrate (the durable home store) is shared infrastructure, not owned here.
- API: `api/handlers/plans/`. CLI: `cli/domains/plans/`. UI: `ui/src/features/plans/`.
- Storage: structured plan records in the scenario-independent `~/.vrooli` home store (see [`DATA.md`](DATA.md)).
- Requirements: PM-STORE-001, PM-STORE-002, PM-GRAPH-001.
- Tests: repository round-trips, render determinism, status-transition legality, supersession resolution.

### authoring

- Purpose: the guided composer wizard (OT-P0-002) — walk the plan's sections in order, validate structure as it goes, and auto-fill the mechanical sections (regression anchor, required-reading, code references) so a small model supplies only genuine prose.
- Primary archetype: workflow / guided process.
- Owns: authoring-session progression, the structure-validation gate, autofill orchestration (behind seams to git-control-tower, prompt-manager, code-facts).
- Does not own: the plan record itself (delegates writes to `plans`); the actual baseline/reference computation (delegates to `validation`).
- API: `api/handlers/authoring/`. CLI: `cli/domains/authoring/`. UI: `ui/src/features/authoring/`.
- Storage: transient authoring-session state; the plan it produces is owned by `plans`.
- Requirements: PM-AUTHOR-001, PM-AUTHOR-002.
- Tests: gate rejects empty mandatory sections; autofill degrades gracefully when a source is down.

### execution

- Purpose: the guided runner (OT-P0-003, OT-P1-001/002) — phase status transitions, just-in-time context injection (`status`/`next`), in-flow capture of decisions/findings, the thin guided completion process, the canonical structured handoff, and per-plan velocity.
- Primary archetype: workflow / state machine.
- Owns: run↔plan linkage, captured decisions/findings, candidate (unvalidated) findings, the canonical handoff record, the velocity series.
- Does not own: the prose final-message handoff (orchestration-layer concern — see [`INTEGRATIONS.md`](INTEGRATIONS.md)); promotion of candidate findings to real bugs (operator triage); the validation it surfaces (delegates to `validation`).
- API: `api/handlers/execution/`. CLI: `cli/domains/execution/`. UI: `ui/src/features/execution/`.
- Storage: execution/run state, handoff records, candidate findings, velocity points.
- Requirements: PM-EXEC-001/002, PM-HANDOFF-001/002, PM-VEL-001.
- Tests: resume-point derivation, completion nudges, attribution-keyed dedup, handoff assembly from captured state.

### validation

- Purpose: plan health (OT-P0-004/005) — resolve code references, compute staleness tiers, derive each phase's baseline scope, orchestrate baseline/check runs on request, and verify Definition of Done against the regression anchor.
- Primary archetype: verification / analysis.
- Owns: reference resolutions, staleness factors, baseline-scope derivation, validation results.
- Does not own: project-level validation of resources/packages/whole-project (consumed from test-genie / scenario-validation, not owned here); the baseline mechanism itself (composes git-control-tower) or the staleness mechanism (composes the freshness engine + code-facts).
- API: `api/handlers/validation/`. CLI: `cli/domains/validation/`. UI: `ui/src/features/validation/`.
- Storage: reference + validation result records keyed to plan/phase.
- Requirements: PM-REF-001, PM-STALE-001, PM-VALID-001/002.
- Tests: tier derivation (fresh/lightly-stale/definitely-stale), command-set derivation, DoD verdict from baseline diff.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Plan / Phase | The structured record + its first-class phases. | [`PLAN-MODEL.md`](PLAN-MODEL.md); `plans` domain owns persistence. |
| Reference | A connected-code locator (`[CODE:]`/`[REQ:]`) on a plan/phase. | `validation` resolves; `plans` stores. |
| Staleness tier | fresh / lightly-stale / definitely-stale. | `validation`. |
| Regression anchor | The "before" baseline/sha for the plan. | `validation`; auto-filled by `authoring`. |
| Handoff (canonical) | Structured handoff assembled from captured state. | `execution`. |
| Candidate finding | An unvalidated possible-bug awaiting operator triage. | `execution`. |
| Domain | Product capability boundary that is easy to find, test, and delete. | This document. |
| Seam | Test-substitutable boundary wired once in production. | [`../internal/SEAMS.md`](../internal/SEAMS.md). |
| Requirement | Implementation-facing measurement tied back to the PRD. | `requirements/`. |

## Deferred Domains

| Candidate Domain | Why Deferred | Revisit Trigger |
|---|---|---|
| consumer-inversion adapters | OT-P2-002 re-points swarm-manager/hygiene/`vrooli plans` to delegate here; sequenced after standalone proof. | After the P0/P1 domains are green and proven standalone. |
| prose-handoff capture | Owned by the orchestration layer (agent-manager/swarm-manager), not plan-manager — it reads transcripts, which this scenario must not. | Never becomes a plan-manager domain; tracked in INTEGRATIONS. |

## Non-Domains

These are important but should not become product domains:

- `api/internal/server/` — HTTP composition substrate.
- `api/internal/module/` — shared module descriptor type.
- `api/internal/modules/` — thin registry for boot/codegen.
- `api/internal/database/` — cross-cutting database/home-store infrastructure.
- `api/internal/testutil/` — cross-domain test harnesses.
- `ui/src/components/` — shared presentation primitives.
- `ui/src/test-utils/` — cross-feature testing support.

If one of these starts using product vocabulary, split the product piece into an
owning domain instead of growing infrastructure.

## Cross-References

- [`PLAN-MODEL.md`](PLAN-MODEL.md) — the structured plan + phase schema
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — system shape and extension rules
- [`FLOWS.md`](FLOWS.md) — workflows and state transitions
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — boundary registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test strategy
