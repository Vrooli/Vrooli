# Domains — Plan Manager

This document is the canonical map of product capabilities, bounded contexts, and
ownership for this scenario. Keep it current whenever a domain is added, renamed,
split, merged, or removed.

The scaffold ships `health` plus one clearly fenced worked example domain (`notes`,
never product scope). The five real domains below — `plans`, `authoring`,
`execution`, `validation`, `log` — map to the three engines from the founding design
(Composer = authoring, Runner = execution, Ledger = validation) over the plan
**store**, plus the `log` execution-log ledger that owns the typed work products an
agent produces while executing a plan. `vrooli scenario detemplate plan-manager`
removes the fenced example once the real domains are green.

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
| authoring | Guided composer wizard: section-by-section flow, structure-validation gate, autofill of mechanical sections. | orchestration / workflow | Authoring-session state; validation findings. | API, CLI, UI | PM-AUTHOR-001/002 | `api/internal/authoring/`, `api/handlers/authoring/`, `cli/domains/authoring/`, `ui/src/features/authoring/`, `packages/proto/schemas/plan-manager/v1/authoring/` |
| execution | Guided runner: phase transitions, just-in-time context injection (reads the log ledger), thin completion + canonical handoff, velocity. | orchestration / state machine | Execution/run linkage, handoff records, velocity series. | API, CLI, UI | PM-EXEC-001/002, PM-HANDOFF-001/002, PM-VEL-001, PM-UI-001 | `api/internal/execution/`, `api/handlers/execution/`, `cli/domains/execution/`, `ui/src/features/execution/`, `ui/src/features/triage/`, `ui/src/features/velocity/`, `packages/proto/schemas/plan-manager/v1/execution/` |
| validation | Plan health: code-reference resolution, staleness tiers, baseline-scope derivation, baseline/check orchestration, DoD verification. | provider / verification | Reference resolutions, validation results, staleness factors. | API, CLI, UI | PM-REF-001, PM-STALE-001, PM-VALID-001/002 | `api/internal/validation/`, `api/handlers/validation/`, `cli/domains/validation/`, `ui/src/features/validation/`, `packages/proto/schemas/plan-manager/v1/validation/` |
| log | Execution-log ledger: the single durable home for typed work products an agent produces while executing a plan — decisions, candidate findings, filed bug reports, reusable records, notes; list/get/update/promote/sync; internal downstream forwarding. | ledger / capture | Log entries (decisions, findings, bug reports, records, notes) + downstream sync state. | API, CLI, UI (client + candidate-finding triage view under execution's `triage` feature) | PM-LOG-001/002/003/004 | `api/internal/planlog/`, `api/internal/planmodel/log.go`, `api/handlers/planlog/`, `cli/domains/log/`, `ui/src/api/log.ts`, `packages/proto/schemas/plan-manager/v1/log/` |

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

- Purpose: the guided composer wizard (OT-P0-002) — walk the plan's sections in order, then author each phase as a first-class draft object with API-owned guided steps at every transition. The wizard validates structure as it goes, captures mechanical context where possible (regression anchor, relevant-context candidates, code references), and requires explicit accept/reject decisions for discovered context so a small model supplies only genuine prose and final judgment.
- Primary archetype: orchestration / workflow.
- Owns: authoring-session progression, phase-draft progression, context candidate discovery/accept/reject state, API-owned guided-step payloads, the structure-validation gate, authoring-time `cli:` command-reference feedback, autofill orchestration (behind seams to git-control-tower, prompt-manager, search-hub, cli-health, code-facts).
- Does not own: the plan record itself (delegates writes to `plans`); command truth (delegates to CLI Health); the actual baseline/reference computation (delegates to `validation`).
- API: `api/handlers/authoring/`. CLI: `cli/domains/authoring/`. UI: `ui/src/features/authoring/`.
- Storage: transient authoring-session state; the plan it produces is owned by `plans`.
- Requirements: PM-AUTHOR-001, PM-AUTHOR-002.
- Tests: gate rejects empty mandatory sections, missing plan/phase references without an explicit `NO_CODE_REFS` reason, and invalid current `cli:` references; autofill degrades gracefully when a source is down.

### execution

- Purpose: the guided runner (OT-P0-003, OT-P1-001/002) — phase status transitions, just-in-time setup context injection (`start`/`status`/`context`/`resume`/`continue`/`next`), validation-gated `done` transitions, the thin guided completion process, the canonical structured handoff, and per-plan velocity. Decisions/findings/bugs/records are no longer captured here — they are recorded in the `log` domain, and execution reads compact log summaries through the read-only `LogLedger` seam for its just-in-time context and handoff.
- Primary archetype: orchestration / state machine.
- Owns: run↔plan linkage, the canonical handoff record (which now carries a `LogSummary` + the captured `log_entries`), the velocity series.
- Does not own: the typed work-product ledger itself (delegates to `log` — decisions, findings, bug reports, records, notes); the prose final-message handoff (orchestration-layer concern — see [`INTEGRATIONS.md`](INTEGRATIONS.md)); promotion of candidate findings to real bugs (`log promote` / operator triage); the validation it surfaces (delegates to `validation`).
- API: `api/handlers/execution/`. CLI: `cli/domains/execution/`. UI: `ui/src/features/execution/`, `ui/src/features/triage/`, `ui/src/features/velocity/`.
- Storage: execution/run state, handoff records, velocity points. (The `decisions`/`findings` execution tables were removed; the ledger lives in `log`.)
- Requirements: PM-EXEC-001/002, PM-HANDOFF-001/002, PM-VEL-001, PM-UI-001.
- Tests: resume-point derivation, continue-loop guidance, validation-before-done enforcement, typed completion nudges (`record_finding`/`file_bug`/`capture_record`/`confirm_phase_status`), handoff assembly from the log summary read through `LogLedger`, once-per-execution first-start-via-continue context emission.

### validation

- Purpose: plan health (OT-P0-004/005) — resolve code references, compute staleness tiers, derive each phase's baseline scope, orchestrate baseline/check runs on request, and verify Definition of Done against the regression anchor.
- Primary archetype: provider / verification.
- Owns: reference resolutions, staleness factors, baseline-scope derivation, validation results.
- Does not own: project-level validation of resources/packages/whole-project (consumed from test-genie / scenario-validation, not owned here); the baseline mechanism itself (composes git-control-tower). Per-reference staleness is implemented as filesystem existence plus git-sourced drift from the regression anchor because the live freshness engine is scenario-artifact scoped today.
- API: `api/handlers/validation/`. CLI: `cli/domains/validation/`. UI: `ui/src/features/validation/`.
- Storage: reference + validation result records keyed to plan/phase.
- Requirements: PM-REF-001, PM-STALE-001, PM-VALID-001/002.
- Tests: tier derivation (fresh/lightly-stale/definitely-stale), command-set derivation, DoD verdict from baseline diff.

### log

- Purpose: the execution-log ledger — Plan Manager's single durable home for the typed work products an agent produces while executing a plan: design **decisions**, candidate **findings**, filed **bug reports**, reusable **records**, and lightweight **notes**. Typed `Add*` RPCs create entries; `ListEntries`/`GetEntry` read them; `UpdateEntry` edits mutable fields (including finding triage); `PromoteEntry` turns a finding into a bug report or record (preserving the original, linked via `promoted_from_id`); `SyncEntry` retries downstream forwarding. It replaces the decision/finding capture that used to live on the execution runner so the distinct concepts stay distinct and owned by one domain.
- Primary archetype: ledger / capture.
- Owns: the `log_entries` table (one typed table for all five entry types), per-entry downstream `sync_status`, idempotency/attribution dedup, finding→bug/record promotion, and the compact `LogSummary` roll-up the execution domain reads.
- Does not own: confirmed downstream artifacts (the scenario-qa bug inbox and swarm-manager records are the source of truth once synced — see [`INTEGRATIONS.md`](INTEGRATIONS.md)); plan/phase records (delegates plan/execution resolution to `plans`/`execution` through the `Resolver` seam); operator judgment on promotion.
- Distinct concepts (never conflated): a **finding** is an unvalidated candidate observation; a **bug_report** is a defect deliberately filed to the issue tracker; a **record** is reusable learning for the learning loop. Findings file as `candidate` and are never auto-promoted.
- Seams: `Resolver` (plan-id/slug or execution-id handle → canonical plan+execution scope), `BugReporter` (forward bug reports → scenario-qa), `RecordWriter` (forward records → `swarm-manager records create`). Bug/record forwarding is owned INTERNALLY; the v1 default is a documented pending stub and a failed forward is never fatal — the local entry persists `pending`/`sync_failed` and is retried via `plan-manager log sync`.
- API: `api/handlers/planlog/` (the Go internal package is `api/internal/planlog/`, named to avoid the stdlib `log` clash). CLI: `cli/domains/log/` (flat verbs: `decision-add`/`finding-add`/`bug-add`/`record-add`/`note-add`/`list`/`get`/`update`/`promote`/`sync`). Shared model: `api/internal/planmodel/log.go`.
- Storage: `log_entries` in the `~/.vrooli` home store with partial UNIQUE indexes for idempotency-key dedup and (execution, attribution_run_id, type, normalized title) dedup.
- Requirements: PM-LOG-001 (single ledger / distinct concepts), PM-LOG-002 (internal downstream ownership), PM-LOG-003 (durable + retryable sync), PM-LOG-004 (idempotent retries).
- Tests: typed Add* + dedup, promotion preserves the finding, downstream-unavailable → pending (non-fatal), sync retry, summary roll-up read by execution.

## Shared Concepts

| Concept | Meaning | Owner |
|---|---|---|
| Plan / Phase | The structured record + its first-class phases. | [`PLAN-MODEL.md`](PLAN-MODEL.md); `plans` domain owns persistence. |
| Reference | A connected-code locator (`[CODE:]`/`[REQ:]`) on a plan/phase. | `validation` resolves; `plans` stores. |
| Staleness tier | fresh / lightly-stale / definitely-stale. | `validation`. |
| Regression anchor | The "before" baseline/sha for the plan. | `validation`; auto-filled by `authoring`. |
| Handoff (canonical) | Structured handoff assembled from captured state (carries a `LogSummary` + the run's `log_entries`). | `execution`; reads the ledger via `LogLedger`. |
| Log entry | One typed execution-log work product: decision / finding / bug_report / record / note. | `log`. |
| Candidate finding | An unvalidated possible-bug log entry (triage `candidate`) awaiting operator triage/promotion. | `log`. |
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
