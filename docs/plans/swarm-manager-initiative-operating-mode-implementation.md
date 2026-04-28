# Plan B: Initiative-Level Operating Mode Implementation

## 1. Purpose

Implement initiative-level execution mode in swarm-manager — a second first-class operating mode that complements (does not replace) the existing backlog-item-level mode. The conceptual framing for both modes lives in [`scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`](../../scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md); read it before this plan.

In short: when backlog items are wrong as the unit of execution — most visibly when they are coupled by a shared substrate (the 2026-04-27 sandboxing trap is the canonical example), but also when they are likely to shift mid-flight, mis-scoped, or only validatable as a system — executing them item-by-item leaves the system in inconsistent intermediate states or thrashes the item graph. The structured `investigate → plan → execute → review` loop that successfully completed those initiatives outside the harness is the second mode — async-reviewable just like backlog-item mode, but operating on the initiative as the unit of work. This plan adds it as a first-class capability:

- Mode metadata on initiatives (`item-level` default, `initiative-level` opt-in).
- An investigation phase as a first-class step (no analog at backlog-item scale).
- An initiative-level workshop round model that produces `initiative-plan.md`.
- An execute → review → replan loop at initiative scope, with item-level follow-ups for residual small work.
- Mode-switch operations (item-level ↔ initiative-level), including in-flight cancellation.
- A new initiative-level review skill (acceptance-oriented), distinct from the existing decision-oriented `swarm-manager-initiative-review`.

Plan A (`docs/plans/swarm-manager-initiative-feedback-ux.md`) is a companion that improves rescoping inside item-level mode. Plan B is independent of Plan A; either can land first. They do not conflict.

## 2. Required Reading

```bash
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement
prompt-manager skill read implementation-plan-authoring
prompt-manager skill read swarm-manager-initiative-feedback
prompt-manager skill read swarm-manager-initiative-review
prompt-manager skill read swarm-manager-initiative-context
```

Required file reads:

- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md` — load-bearing framing, including all open questions this plan resolves.
- `scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md` — staging-and-review framing this plan builds on.
- `scenarios/swarm-manager/docs/guides/workshop-workflow.md` — backlog-item-level workshop loop (initiative-level loop is parallel but distinct; read for shape contrast).
- `scenarios/swarm-manager/api/internal/initiatives/model.go` — current Initiative model (no mode field; this plan adds one).
- `scenarios/swarm-manager/api/internal/initiatives/service.go` and `handler.go` — current CRUD + rollup; new endpoints land here.
- `scenarios/swarm-manager/api/internal/initiativereview/service.go` and `doc.go` — existing decision-oriented review service; provides patterns for the new acceptance-oriented review.
- `scenarios/swarm-manager/api/internal/feedback/service.go` and `routes_feedback.go` — feedback service patterns: spawn / lock / poll / cancel / state-builder; the initiative-level mode reuses these primitives.
- `scenarios/swarm-manager/api/internal/workshop/workshop.go` — readiness model and round I/O (item-level); initiative-level mode mirrors the round-file shape.
- `scenarios/swarm-manager/api/internal/initiativelock/lock.go` — single-agent-per-initiative lock; reused for initiative-level mode.
- `scenarios/swarm-manager/api/internal/agentactivity/service.go` — activity tracking surface; new `OwnerInitiative` purposes are added here.
- `scenarios/swarm-manager/ui/src/types/domain.ts` and `types/feedback.ts` — current type surface to extend.

## 3. Problem Statement

### The empirical cue

On 2026-04-27, two initiatives (`agent-sandbox-audit-foundation`, 10/10 items; `protected-agent-sandboxing`, 7/7 items) completed via operator-direct execution outside the swarm-manager harness. The reason was *not* size: it was *coupling*. Every item in both initiatives modified how agent runs route through the workspace sandbox. Completing item N changed routing behavior in a way that broke the next item's execution, because each item is run as itself an agent execution that uses the very routing being changed. Swarm-manager-managed execution stalled mid-initiative; the operator stepped out and used a coding agent directly to:

1. Investigate the actual current state across both initiatives.
2. Generate a consolidated plan covering large chunks of remaining work.
3. Execute against the plan in waves.
4. Re-investigate, revise the plan, repeat.

This pattern succeeded. The ad-hoc nature is the problem: there is no support for it inside swarm-manager, no audit trail consistent with the rest of the system, no provenance for the resulting changes. "Operator stepped outside the harness" is treated as a workaround when it should be a first-class mode.

### What the existing primitives can't express

- **Coupled-item execution.** The workshop loop is per-item; it does not refine cross-item plans. The execution flow operates one item at a time. The review flow ratifies one item at a time. There is no surface for "investigate this initiative as a whole and plan against the system."
- **Investigation as a step.** Workshop's first round is a refinement step over a one-item spec, not a system-state investigation. There is no `findings.md` deliverable, no mechanism for the agent to report "current code state diverges from item descriptions in these ways."
- **Replan as a first-class loop step.** Workshop has rounds; rounds *refine* a single item's plan. There is no "execute partial → discover something → revise the whole plan" loop at the initiative scope.
- **Mode-aware locks and budgets.** The single-agent-per-initiative lock exists (`initiativelock.Lock`) but only for feedback rounds. Initiative-level executions need the same exclusivity but for a longer-running, more expensive run.
- **Acceptance at the initiative scope.** Backlog items have `acceptance_allow`/`acceptance_deny` globs. Initiatives have only a `Status` field. There is no way to say "this initiative is done when the system as a whole satisfies these criteria."

### What "won't" work

- *Just merge the items into one big item.* Loses tracking, partial cancellation, dependency model, and per-item history.
- *Just iterate the workshop loop more times.* Workshop is per-item; no amount of iteration produces an initiative-level plan.
- *Just write a longer skill prompt.* The skill itself is fine; the missing pieces are persisted artifacts, lifecycle, mode metadata, and review semantics.

## 4. Scope

### In scope

- Add `Mode` field to `Initiative` model with values `"item-level"` (default) and `"initiative-level"`.
- Add `AcceptanceCriteria []string` field to `Initiative` model (free-form, no glob model in this plan).
- Add API endpoints for initiative-level mode lifecycle: switch mode, kick off investigation, kick off plan-round, kick off execute-round, kick off review, mark initiative complete.
- Add an `initiative-plan.md` deliverable convention at the initiative folder; add a `findings.md` deliverable updated by each investigation pass.
- Add an `initiative-rounds/` subdirectory at the initiative folder mirroring backlog `workshop/round-N.json` shape, with rounds tagged by phase (`investigate` | `plan` | `execute` | `review` | `replan`).
- Add three new prompt-manager skills:
  - `swarm-manager-initiative-investigate` — produces `findings.md`.
  - `swarm-manager-initiative-plan` — produces/updates `initiative-plan.md`.
  - `swarm-manager-initiative-execution-review` — acceptance-oriented review against `AcceptanceCriteria`.
- Reuse the existing `swarm-manager-initiative-feedback` skill for replanning loops triggered by feedback (Plan A's improvements compose; not required).
- Reuse `proposals.Applier` for any item-graph mutations the initiative-level plan emits (e.g., adding a follow-up item discovered during execution).
- Add a 7-dimension readiness model for initiative-level rounds: the existing 5 (`problem_clarity`, `scope_defined`, `approach_solid`, `testable`, `risk_awareness`) plus two new ones (`coupling_understood`, `system_acceptance_defined`).
- Mode-switch protocol with explicit operator confirmation for cancelling in-flight per-item executions on the way into initiative-level mode.
- UI: an Initiative detail page mode-switch control; an "Initiative Workspace" view that surfaces findings, plan, rounds, execution history, and review state distinct from the per-item Backlog detail view.
- Single-agent-per-initiative lock semantics extended to cover all initiative-level rounds (investigate / plan / execute / review).
- Activity tracking: new `agentactivity.Purpose` values for each initiative-level round phase.
- Tests: Go unit + integration for service / handler / lock; Vitest/RTL for UI; skill simulation for each new skill.
- Docs updates: `EXECUTION-MODES.md` is the framing; `docs/guides/initiative-level-mode.md` is the operator-facing how-to (new).

### Out of scope

- **Plan A's `merge_items` op.** Independent; either can land first.
- **Workshop schema changes for backlog items.** Item-level workshop is unchanged.
- **Backwards compatibility for old initiatives without `Mode`.** Greenfield: missing field defaults to `item-level` at load time; never write the default to disk.
- **Cross-initiative coordination.** This plan handles single-initiative initiative-level runs only. Multi-initiative execution remains a future concern.
- **Cost-budget enforcement.** UI shows a "this is a long-running mode" hint before kickoff (see Phase 5); no hard budget gate is implemented.
- **Auto-mode-detection.** No heuristic decides for the operator whether to switch modes; the operator chooses.
- **Tech-tree integration.** Out of scope; framework doc references a future tech-tree-designer scenario but this plan doesn't touch that.

## 5. Current Technical Context

### Initiative model (`api/internal/initiatives/model.go`)

```go
type Initiative struct {
    Name        string
    Title       string
    Description string
    Status      string   // "active" | "completed" | future statuses
    Items       []string // "kind/name" refs
    // (other metadata — priority, depends_on, note, archived_at)
}
```

No mode, no acceptance criteria. The `Status` field uses `active`/`completed`/`archived` semantics validated by `ValidateStatus` and is user-settable through a small enum.

### Initiative storage (`api/internal/initiatives/store.go`)

Filesystem-first. Each initiative is `<root>/initiatives/<name>/initiative.json` with related files in the same directory (orchestration-summary.md is conventional but not required). Plan B's `initiative-plan.md`, `findings.md`, and `initiative-rounds/round-N.json` files live in the same directory.

### Existing review (`api/internal/initiativereview/service.go`)

Decision-oriented: handles "which member items need decide actions, who's blocked, what's the operator queue" — an aggregator over per-item review state. **It is not an acceptance-of-initiative-as-a-whole flow.** This plan introduces a separate flow for that.

### Existing feedback flow (`api/internal/feedback/service.go`, `routes_feedback.go`)

Single-agent-per-initiative lock via `initiativelock.Lock`. Spawn → poll → render proposal → operator decides. The lock primitive, the spawn adapter, and the poller are all reusable; they do not assume "feedback" semantics specifically.

### Existing workshop (`api/internal/workshop/workshop.go`)

Per-item readiness scoring and round I/O. Round files are `<item-dir>/workshop/round-N.json`. Plan B reuses the round JSON shape (questions / proposals / info items / readiness scores) but extends the readiness dimension list and tags rounds by phase.

### Activity tracking (`api/internal/agentactivity/`)

`OwnerType` includes `OwnerBacklog` and `OwnerInitiative`. `Purpose` includes `PurposeFeedback`, `PurposeFeedbackContinue`, `PurposeInitiativeReview`, plus backlog-item purposes. Plan B adds new initiative purposes.

### Skill catalog (`api/internal/promptcatalog/`)

`ResolveInitiativeSkill(purpose)` is the existing seam for purpose→skill mapping at the initiative scope. Plan B's three new skills are registered through this catalog.

## 6. Target End State

After this plan lands:

- An initiative can be switched to initiative-level mode via a CLI command and a UI control, with an operator-confirmed cancel-in-flight gate.
- Initiative-level mode flow is fully runnable end-to-end: investigate → plan → execute → review, with replan as an explicit loop step.
- Each phase produces durable on-disk artifacts (`findings.md`, `initiative-plan.md`, `initiative-rounds/round-N.json`) under the initiative folder.
- All four phases' agent runs are tracked via `agentactivity` with explicit purposes.
- Single-agent-per-initiative is enforced across both feedback and initiative-level rounds (the lock is unified).
- Acceptance criteria can be set on an initiative (CLI + UI) and are the contract the review phase validates against.
- The 7-dimension readiness model is implemented for initiative-level rounds.
- Member backlog items in initiative-level mode are tracked but not independently executed; their status is updated by the initiative-level execution agent as plan milestones land.
- A new `swarm-manager-initiative-execution-review` skill produces a structured acceptance verdict; the existing `swarm-manager-initiative-review` (decision-oriented) is unchanged.
- A user guide at `scenarios/swarm-manager/docs/guides/initiative-level-mode.md` explains the operator-facing flow with examples.
- All new behavior is covered by automated tests.
- `vrooli scenario restart swarm-manager` succeeds; `make test` passes.

## 7. Implementation Strategy

Phases are ordered for incremental safety: model + storage first, then primitives, then phases, then UI, then docs. Each phase ships a passing test set before the next begins.

### Phase 1 — Model and storage

1. **`api/internal/initiatives/model.go`**
   - Add `Mode string` and `AcceptanceCriteria []string` to `Initiative`.
   - Constants: `ModeItemLevel = "item-level"`, `ModeInitiativeLevel = "initiative-level"`.
   - `ValidateMode(mode string) error` — accepts both modes; empty string maps to `item-level` at load time but is rejected on write.
   - Add `Mode` and `AcceptanceCriteria` to `CreateRequest` and `UpdateRequest` patches.
   - Default-on-load: `(*Initiative) Normalize()` sets `Mode` to `item-level` if empty.
2. **`api/internal/initiatives/validation.go`**
   - Reject `Mode` values outside the constants.
   - Reject mode changes on archived initiatives.
3. **`api/internal/initiatives/store.go`**
   - No structural store change; `initiative.json` gains the new fields via JSON marshal.
   - On read: call `Normalize()` so callers always see a populated `Mode`.
4. **Tests**
   - `model_test.go` (new or extend existing): validate normalization, validation, marshal/unmarshal.
   - `store_test.go`: round-trip an initiative with `Mode` and `AcceptanceCriteria`.

### Phase 2 — Lock unification

1. **`api/internal/initiativelock/lock.go`**
   - The lock already exists for feedback. Add a `Holder` enum to distinguish `feedback` / `investigate` / `plan` / `execute` / `review`. The lock fundamentally remains "one agent per initiative at a time"; the holder field is for diagnostics and override prompts.
   - Add `Acquire(initiativeName, holder Holder) (token, error)` and `Release(initiativeName, token)`.
2. **`api/routes_feedback.go`**
   - Existing feedback acquires the lock with `holder=feedback`. No semantic change; just pass the new holder constant.
3. **Tests**
   - `lock_test.go`: holder field is recorded; concurrent attempts collide regardless of holder.

### Phase 3 — Investigation phase

1. **Skill: `swarm-manager-initiative-investigate`**
   - Inputs (rendered from prompt-manager): initiative metadata, current item graph, current `findings.md` if present, current `initiative-plan.md` if present, prior investigation rounds.
   - Output: structured `findings.md` with sections: `## Current State`, `## Drift Detected`, `## Acceptance-Criterion Status`, `## Open Questions`. Plus a JSON envelope (in a fenced block) summarizing key findings into an `initiative-rounds/round-N.json` entry of phase=`investigate`.
   - Hard rules: read-only on code; never proposes mutations; never edits item files.
2. **API: `POST /api/v1/initiatives/{name}/investigate`**
   - Server-side: acquire lock; spawn the investigate skill via the existing feedback spawner pattern (rename or generalize the spawner adapter as `initiativeAgentSpawner` if needed); track via `agentactivity` with `Purpose = PurposeInitiativeInvestigate`.
   - On agent completion: persist `findings.md` and append to `initiative-rounds/round-N.json` with `phase = "investigate"`.
3. **CLI: `swarm-manager initiative investigate <name>`**
   - Calls the API; tails the agent output; reports the findings file path.
4. **Tests**
   - Unit: handler returns 409 if not in `initiative-level` mode; 423 if lock held; 200 with run-id otherwise.
   - Integration: end-to-end spawn + completion + persistence.

### Phase 4 — Plan phase

1. **Skill: `swarm-manager-initiative-plan`**
   - Inputs: same as investigate, plus the latest `findings.md`.
   - Output: `initiative-plan.md` (overwrites prior version) — implementation-grade plan covering the work the initiative needs to do as a whole. JSON envelope: `initiative-rounds/round-N.json` with `phase = "plan"` carrying the 7-dimension readiness scores.
   - The plan skill follows the structure documented by `implementation-plan-authoring` but at initiative scope.
2. **Readiness model (`api/internal/workshop/initiative_readiness.go` or extend existing)**
   - 7 dimensions: 5 existing + `coupling_understood` (do we know which items are coupled and how) + `system_acceptance_defined` (are the initiative's acceptance criteria stated and testable).
   - Boost formula reuses the item-level shape but with `N=2` always (initiative-level is always cautious).
3. **API: `POST /api/v1/initiatives/{name}/plan`**
   - Spawns the plan skill; persists output; appends round.
4. **CLI: `swarm-manager initiative plan <name>`**
5. **Tests**
   - Round persistence; readiness scoring; lock contention; reject when not in `initiative-level` mode.

### Phase 5 — Execute phase

1. **Skill: `swarm-manager-initiative-execute`**
   - Inputs: the latest `initiative-plan.md`, current member items, current item graph, prior execute rounds.
   - Output: real changes to repo files (the agent edits code per the plan). Also updates member item statuses as plan milestones land — completing an item must call back through the swarm-manager API rather than mutating spec.json directly. JSON envelope round-N.json with `phase = "execute"` summarizing what the run completed and what remains.
   - Hard rule: the agent reports `replan_needed: bool` in the round envelope when execution discovered something material to the plan.
2. **API: `POST /api/v1/initiatives/{name}/execute`**
   - Long-running spawn (mirrors backlog execution paths). Tracked via `agentactivity` with `Purpose = PurposeInitiativeExecute`.
   - On `replan_needed=true`: surface in the rollup status so the UI shows "ready to replan" rather than "ready to review".
3. **API: `POST /api/v1/initiatives/{name}/items/{item-ref}/complete`** (new helper)
   - Used by the execute agent to mark a member item completed when its plan milestone lands. Validates the caller is the active execute agent (matching run-id). Greenfield: no auth bypass for tests beyond a documented test seam.
4. **CLI: `swarm-manager initiative execute <name>`**
5. **UI cost-hint modal**
   - Before kickoff, modal displays: "Initiative-level execute runs can take many minutes and consume significant tokens. Continue?"
   - Acknowledge persists for 24h per browser to avoid friction.
6. **Tests**
   - Item-completion endpoint requires matching run-id; rejects stale.
   - Replan signal surfaces in rollup.

### Phase 6 — Review phase

1. **Skill: `swarm-manager-initiative-execution-review`**
   - Inputs: initiative metadata, `AcceptanceCriteria`, latest `initiative-plan.md`, all execute rounds, current item graph, current code state (read-only).
   - Output: `accept` | `request_replan` | `request_changes` verdict plus a structured rationale per acceptance criterion. JSON envelope `initiative-rounds/round-N.json` with `phase = "review"`.
   - Hard rule: never edits files; only emits a verdict.
2. **API: `POST /api/v1/initiatives/{name}/review`**
   - Spawns the review skill; persists round.
   - On `accept`: requires explicit operator confirmation via `POST /api/v1/initiatives/{name}/complete` before transitioning the initiative `Status` to `completed`. The agent never auto-completes.
3. **CLI: `swarm-manager initiative review <name>`** and `swarm-manager initiative complete <name>`.
4. **Tests**
   - Verdict shape; auto-complete rejection (status remains `active` until explicit complete).

### Phase 7 — Replan loop

The loop is implicit: an `execute` round with `replan_needed=true` directs the operator to run another `plan` (or `investigate` → `plan`) round. No new endpoint is required; the rollup status surfaces "ready to replan" and the UI offers the corresponding action.

### Phase 8 — Mode switch

1. **API: `POST /api/v1/initiatives/{name}/mode`**
   - Body: `{ "mode": "initiative-level", "cancel_in_flight": true }`.
   - On switch to initiative-level: enumerate member items in `in_progress`; if any exist and `cancel_in_flight=true`, cancel each via `executionCancellerAdapter.CancelForBacklog`; if `cancel_in_flight=false`, return 409 with the list of in-flight items.
   - On switch to item-level (drain-back): permitted only when no initiative-level run is active; clears `Mode` back to `item-level`. `findings.md`, `initiative-plan.md`, and rounds are preserved as historical record.
2. **CLI: `swarm-manager initiative mode set <name> <item-level|initiative-level> [--cancel-in-flight]`**
3. **UI**
   - Initiative detail page shows current mode in the header.
   - Mode-switch action in the page menu opens a confirmation modal listing in-flight items if any.
4. **Tests**
   - 409 path when in-flight items exist and flag absent; success path with cancellation.

### Phase 9 — Activity tracking and prompt catalog

1. **`agentactivity.Purpose` constants**: `PurposeInitiativeInvestigate`, `PurposeInitiativePlan`, `PurposeInitiativeExecute`, `PurposeInitiativeExecutionReview`. Existing `PurposeInitiativeReview` (decision-oriented) is unchanged.
2. **`promptcatalog.ResolveInitiativeSkill`**: extend to map each new purpose to its skill ID.
3. **Tests**: catalog round-trip; activity record contains correct purpose for each run.

### Phase 10 — UI

1. **Initiative detail page**: add a Mode header chip, mode-switch menu action, and conditional "Initiative Workspace" tab visible only when `Mode = initiative-level`.
2. **Initiative Workspace**: tabs/sections for Findings (renders `findings.md`), Plan (renders `initiative-plan.md` with edit button that submits to the plan-skill via a feedback-style flow), Rounds (round timeline), Execute (status of in-flight execute run + replan-needed banner), Review (verdict + acceptance criteria checkbox state).
3. **Acceptance criteria editor**: add/edit/remove free-form criteria on the initiative.
4. **Cost-hint modal** (Phase 5).
5. **Tests**
   - Mode chip renders correct value.
   - Workspace tab visibility gated on mode.
   - Acceptance criteria editor wires through to the API.

### Phase 11 — Docs

1. **`scenarios/swarm-manager/docs/guides/initiative-level-mode.md`** — operator-facing how-to: when to switch modes, walkthrough of investigate / plan / execute / review, examples, anti-patterns.
2. **`scenarios/swarm-manager/docs/manifest.json`** — register the new guide.
3. **`scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`** — replace the "Open questions" section's items with cross-references to where each is resolved (this plan, mostly), preserving the historical context but pointing to the answers.
4. **`scenarios/swarm-manager/docs/reference/api-endpoints.md`** — register every new endpoint.
5. **`scenarios/swarm-manager/docs/reference/cli-commands.md`** — register every new CLI command.

### Phase 12 — Restart and validation

`vrooli scenario restart swarm-manager` and full `make test` from `scenarios/swarm-manager/`. Plan must leave the scenario healthy.

## 8. Contract Decisions

This section locks decisions for every open question raised in `EXECUTION-MODES.md`.

1. **Workshop schema at initiative scale**: 7 dimensions = 5 existing + `coupling_understood` + `system_acceptance_defined`. Boost divisor `N=2`. Round file shape mirrors backlog `workshop/round-N.json` with an added `phase` field.
2. **Plan artifact location**: `<initiative-folder>/initiative-plan.md`. Distinct from per-item `plan.md` to avoid collision and to make the deliverable greppable across initiatives.
3. **Investigation deliverable**: `<initiative-folder>/findings.md`, regenerated each investigation pass. Prior versions are preserved via the round-file's embedded summary (we do not maintain a `findings-1.md`, `findings-2.md` chain).
4. **Item status propagation**: the execute agent calls `POST /api/v1/initiatives/{name}/items/{item-ref}/complete` when a plan milestone covers an item. The endpoint validates the caller's run-id matches the active execute run. The operator does *not* mark items completed; the agent does, with audit.
5. **Mode metadata storage**: explicit `Mode` field on `Initiative`. Default `item-level`; written explicitly on every save (no inference from artifact presence).
6. **Mode-switch protocol on entering initiative-level mode**: in-flight per-item executions are cancelled with explicit operator confirmation (`cancel_in_flight=true`). Without the flag, the switch returns 409. Drain-back to item-level is permitted only when no initiative-level run is active.
7. **Review reuse vs. new**: new skill `swarm-manager-initiative-execution-review`, distinct from existing `swarm-manager-initiative-review`. Existing review is decision-oriented (which items need decide actions); new review is acceptance-oriented (does the system meet the initiative's acceptance criteria).
8. **Acceptance-criterion model**: free-form `AcceptanceCriteria []string` on the initiative. No glob model in this plan. Future work may add structure if criteria become hard to evaluate.
9. **Parallel-agent boundary**: single-agent-per-initiative across all initiative-level rounds and feedback rounds. Rationale: the on-disk artifacts (`findings.md`, `initiative-plan.md`, `initiative-rounds/round-N.json`) have a single writer at a time; concurrent agents would race on these files and produce inconsistent state. The lock is a contention guard, not a presence assumption; the holder field is for diagnostics.
10. **Cost / budget surfacing**: UI cost-hint modal before any execute-round kickoff. No hard budget gate. Acknowledgement persists 24h per browser.

Wire shapes (JSON envelopes inside agent replies) for each new round phase mirror the existing workshop round JSON: `{ round, phase, generated_at, readiness, items, plan_updates }` with `phase` ∈ `"investigate" | "plan" | "execute" | "review"`. Each phase has phase-specific item types it may emit; the validator enforces the schema per phase.

## 9. Testing Plan

All verification is automated.

### Go (`scenarios/swarm-manager/api/`)

- `initiatives/model_test.go`: Mode normalization, validation, marshal/unmarshal of new fields.
- `initiatives/handler_test.go`: each new endpoint — investigate / plan / execute / review / mode / items/complete — returns correct status codes for happy and error paths.
- `initiatives/service_test.go`: state transitions, lock interactions.
- `initiativelock/lock_test.go`: holder field; concurrency invariants.
- `agentactivity` purpose round-trip tests.
- `promptcatalog/catalog_test.go`: new purpose → skill resolutions.
- E2E: `e2e_initiative_mode_test.go` — full investigate → plan → execute → review → complete flow against a test initiative on a temp filesystem.

### TypeScript (`scenarios/swarm-manager/ui/`)

- `feedback-dialog.test.tsx`: regression-only (no behavior change in feedback dialog).
- New tests for mode-switch modal, Initiative Workspace tab visibility, acceptance criteria editor, cost-hint modal.
- Type-check pass.

### Skills

- `prompt-manager skill simulate` against each new skill with synthetic inputs verifies the skill produces the expected JSON envelope shape.
- `prompt-manager skill list --filter swarm-manager-initiative-` enumerates the four initiative-level skills (existing review + three new).

### Cross-scenario

- `vrooli scenario restart swarm-manager` succeeds.
- `vrooli scenario test swarm-manager` passes.
- Adjacent scenarios (`prompt-manager`, `agent-manager`) baseline pass (no regressions).

## 10. Rollout/Validation Checklist

- [ ] Initiative model has `Mode` and `AcceptanceCriteria`; round-trips through the store.
- [ ] Lock primitive has `Holder` and is reused by feedback + all four initiative-level phases.
- [ ] Three new skills present in prompt-manager catalog and resolvable via `ResolveInitiativeSkill`.
- [ ] New `agentactivity.Purpose` values present and used.
- [ ] All four phase APIs (investigate / plan / execute / review) return correct status codes.
- [ ] Mode-switch API enforces in-flight cancellation gate.
- [ ] Item-completion endpoint enforces run-id validation.
- [ ] UI exposes Mode chip, mode-switch modal, Initiative Workspace tab (gated on mode), acceptance criteria editor, cost-hint modal.
- [ ] `findings.md`, `initiative-plan.md`, `initiative-rounds/round-N.json` produced and persisted under the initiative folder.
- [ ] Operator-facing user guide written and registered in manifest.
- [ ] `EXECUTION-MODES.md` open-questions section updated to point to resolutions.
- [ ] `make test` passes; `vrooli scenario restart swarm-manager` succeeds; adjacent scenarios baseline pass.

## 11. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Initiative-level execute runs leave member items in a half-completed state if the agent crashes | Medium | High | Item completion is the agent's last act per milestone; round-N.json journals plan milestones the agent intended to land. On resume, the next execute round reads the journal to know what's actually done in code. |
| Operator switches modes mid-flight and loses work-in-progress on per-item executions | Medium | High | Mode-switch requires explicit `cancel_in_flight=true` flag; the UI modal lists every in-flight item and what will be cancelled. No silent loss. |
| 7-dimension readiness model is more theatre than signal | Low | Medium | Review against actual usage at 1 month + 3 months. The two new dimensions have falsifiable definitions (`coupling_understood` requires a written list of coupled items in `findings.md`; `system_acceptance_defined` requires non-empty `AcceptanceCriteria`). |
| `initiative-plan.md` and per-item `plan.md` diverge over time | Medium | Medium | Plan skill is required to read all member items' `plan.md` and explicitly note where it supersedes. Drift is surfaced in `findings.md` during the next investigate round. |
| Cost-hint modal becomes annoying and gets dismissed permanently | Medium | Low | 24h acknowledgement; modal text is short. Not a blocking gate. |
| Two scenarios on the same machine race for the initiative lock | Low | Medium | Lock is filesystem-based; single-machine deployments are the only supported topology today. |
| Replan loop never terminates (each execute round triggers another replan) | Low | High | Acceptance criteria are the termination contract. Review skill must produce `accept` to complete the initiative; if review repeatedly produces `request_replan`, the operator can escalate to manual completion (which still requires the explicit `complete` action — operator override is logged). |
| New skills' prompts grow too long and exceed context budget | Medium | Medium | Each skill has a token-budget assertion in its simulate test. CI fails the test if the rendered prompt exceeds 12k tokens for typical inputs. |
| Mode-switch lands but no operator ever uses initiative-level mode | Low | Low | The framework doc captures the empirical cue; the user guide makes the use case clear. If unused for 60 days post-merge, revisit at a vision walk. Acceptable cost. |

## 12. Non-goals and Prohibited Patterns

- **No mode auto-detection.** Mode is operator-chosen.
- **No silent in-flight cancellation.** Always require explicit confirmation.
- **No backwards-compatibility shim for missing `Mode`.** Greenfield: load-time normalization is one line; no migration tool.
- **No reuse of `swarm-manager-initiative-review` for acceptance review.** Two distinct skills.
- **No per-item `plan.md` writes by the initiative-level execute agent.** It updates only `initiative-plan.md` and item status (via the API endpoint). Per-item plans are historical.
- **No agent-driven status mutation outside the documented API.** Direct spec.json writes from the execute agent are prohibited; the run-id-validated endpoint is the only path.
- **No coupling between Plan A and Plan B.** Either can land first; neither is required for the other.
- **No multi-initiative orchestration.** Single-initiative scope.
- **No tech-tree integration.** Defer until the tech-tree-designer scenario exists.
- **No hidden "preview" mode that runs investigation without persistence.** All rounds persist; investigation is cheap and the audit trail is load-bearing.

## 13. Definition of Done

The plan is done when **all** of the following hold:

1. Initiative model carries `Mode` and `AcceptanceCriteria`; load/save round-trips correctly.
2. All four phase APIs (investigate / plan / execute / review) plus the mode-switch and item-completion endpoints are implemented, return the documented status codes, and have automated tests covering happy and error paths.
3. The single-agent-per-initiative lock is unified across feedback and the four initiative-level phases.
4. The three new skills (`swarm-manager-initiative-investigate`, `swarm-manager-initiative-plan`, `swarm-manager-initiative-execution-review`) are committed, resolvable via `ResolveInitiativeSkill`, and pass their simulate tests.
5. UI exposes mode chip, mode-switch modal with in-flight enumeration, Initiative Workspace tab, acceptance criteria editor, and cost-hint modal; all are covered by automated tests.
6. The operator-facing guide at `docs/guides/initiative-level-mode.md` is written and registered in the docs manifest.
7. `EXECUTION-MODES.md` open-questions section is updated to point to in-plan resolutions.
8. `cd scenarios/swarm-manager && make test` exits 0.
9. `vrooli scenario restart swarm-manager` succeeds; runtime mode-switch flow demonstrates the new affordances end-to-end.
10. Adjacent scenarios pass `vrooli scenario test`.
11. Every check in the Rollout/Validation Checklist is satisfied by an automated test or a CLI exit code — no manual verification.
