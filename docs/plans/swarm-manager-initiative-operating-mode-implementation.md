# Plan B: Initiative Execution Mode Framework + Initiative-Level Mode

## 1. Purpose

Implement initiative-level execution as Swarm Manager's second first-class execution mode **and** introduce the reusable mode framework needed to support future execution modes without another large architectural rewrite.

The concrete product feature is initiative-level mode: a holistic `investigate -> plan -> execute -> review -> replan` loop for initiatives whose member backlog items are the wrong unit of execution or validation. The architectural feature is a mode-aware initiative execution layer: mode metadata, phase definitions, artifact conventions, prompt routing, locking, agent activity tracking, and UI workspace routing are all expressed through explicit seams rather than scattered `if mode == "initiative-level"` branches.

This plan replaces the older Plan B framing by making the extensibility goal explicit. It remains a companion to Plan A (`docs/plans/swarm-manager-initiative-feedback-ux.md`), which has already been implemented and improves rescoping inside backlog-item-level mode.

The target is not merely "add one more mode." The target is:

- `item-level` remains the default mode and existing backlog-item execution behavior remains intact.
- `initiative-level` becomes the first non-default mode implemented through a reusable framework.
- Adding a future third mode should primarily mean registering a mode definition, phase definitions, skills, artifacts, UI workspace sections, and tests — not rediscovering locks, activity tracking, prompt routing, round persistence, or lifecycle control.

## 2. Required Reading

Run this command before implementing:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health intent-clarification boundary-of-responsibility-enforcement seam-discovery-and-enforcement interoperability-steer api-steer decision-boundary-extraction react-coherence
```

Also read:

- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md` — conceptual distinction between backlog-item-level and initiative-level mode.
- `scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md` — Swarm Manager's staging/review mental model and current seams.
- `scenarios/swarm-manager/docs/guides/workshop-workflow.md` — item-level workshop loop to contrast with initiative-level phases.
- `scenarios/swarm-manager/docs/internal/SEAMS.md` — current internal seam inventory; update it as part of this work.
- `scenarios/swarm-manager/api/internal/initiatives/model.go` — initiative metadata and statuses.
- `scenarios/swarm-manager/api/internal/initiatives/service.go` and `handler.go` — initiative CRUD and route registration.
- `scenarios/swarm-manager/api/internal/initiativelock/lock.go` — existing single-agent-per-initiative lock used by feedback/review.
- `scenarios/swarm-manager/api/internal/feedback/service.go` and `routes_feedback.go` — reusable spawn/poll/cancel/state-builder patterns.
- `scenarios/swarm-manager/api/internal/initiativereview/` — current decision-oriented initiative review, intentionally distinct from initiative-level acceptance review.
- `scenarios/swarm-manager/api/internal/workshop/workshop.go` — readiness scoring and round I/O patterns.
- `scenarios/swarm-manager/api/internal/agentactivity/types.go` and `service.go` — tracked AgentManager usage.
- `scenarios/swarm-manager/api/internal/promptcatalog/catalog.go` — prompt/skill catalog and `ResolveInitiativeSkill`.
- `scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.tsx` — current initiative detail route and tab layout.
- `scenarios/swarm-manager/ui/src/services/initiative-service.ts` — UI service seam for initiative API calls.
- `scenarios/swarm-manager/ui/src/types/initiative.ts` — UI initiative type surface.
- `scenarios/swarm-manager/ui/src/consts/selectors.ts` — selector registry for new UI affordances.

## 3. Problem Statement

Swarm Manager currently assumes backlog items are the unit of execution and validation. That assumption is valuable when items are right-sized, independent, stable, and reviewable in isolation. It fails when the work is coupled across items, when partial intermediate states break the system, when the right item shape shifts during execution, or when success can only be judged at initiative/system scope.

The 2026-04-27 sandboxing initiatives were the empirical cue. Both initiatives changed the shared substrate used by agent executions. Completing one backlog item changed the runtime path that subsequent Swarm Manager item executions depended on, leaving the harness unstable. The successful workaround was a holistic loop outside Swarm Manager:

1. Investigate current code and initiative state.
2. Produce a consolidated initiative-level plan.
3. Execute in waves against that plan.
4. Re-investigate and replan as ground truth changed.
5. Review whether the system as a whole satisfied the initiative.

That should not be an escape hatch. It should be a first-class Swarm Manager execution mode with the same persistence, auditability, async operator cadence, and lifecycle control as item-level mode.

The second problem is architectural. If initiative-level mode is implemented as one-off endpoints and scattered mode checks, the next mode will force another broad refactor. This feature should establish the extensibility seam now.

## 4. Scope

### In Scope

- Add explicit initiative `Mode` metadata with values:
  - `item-level` — default, existing behavior.
  - `initiative-level` — holistic initiative execution mode.
- Add `AcceptanceCriteria []string` to initiatives for initiative-level acceptance review.
- Introduce a backend **initiative mode framework**:
  - mode registry
  - phase definitions
  - phase transition validation
  - artifact conventions
  - shared phase runner
  - shared prompt context builder
  - shared round persistence
  - shared lock and activity tracking integration
- Implement initiative-level mode through that framework:
  - phases: `investigate`, `plan`, `execute`, `review`
  - replan represented as another plan pass after execute signals `replan_needed`
  - artifacts: `findings.md`, `initiative-plan.md`, `initiative-rounds/round-N.json`
  - readiness: existing five dimensions plus `coupling_understood` and `system_acceptance_defined`
- Add mode-switch API/CLI/UI behavior, including explicit cancellation of in-flight item-level executions when entering initiative-level mode.
- Add a mode-aware Initiative Workspace UI tab/section for initiative-level artifacts, rounds, execution status, acceptance criteria, and review verdicts.
- Add prompt-manager skills:
  - `swarm-manager-initiative-investigate`
  - `swarm-manager-initiative-plan`
  - `swarm-manager-initiative-execute`
  - `swarm-manager-initiative-execution-review`
- Extend `promptcatalog` and `agentactivity` around mode/phase purposes.
- Keep existing decision-oriented `swarm-manager-initiative-review` intact and separate.
- Update docs:
  - `EXECUTION-MODES.md`
  - `docs/guides/initiative-level-mode.md`
  - `docs/reference/api-endpoints.md`
  - `docs/reference/cli-commands.md`
  - `docs/internal/SEAMS.md`
  - `docs/manifest.json`
- Automated tests for backend, UI, skills, and cross-scenario validation.

### Out of Scope

- Plan A's feedback UX / `merge_items` work. It is already implemented and should not be reworked here.
- Auto-detecting which mode an initiative should use.
- Cross-initiative execution.
- Tech-tree integration.
- Hard cost-budget enforcement.
- Rewriting backlog item workshop schemas.
- Reusing the decision-oriented initiative review as acceptance review.
- Direct writes by initiative-level execute agents to backlog `spec.json` files.
- New mode types beyond `initiative-level`; this plan creates the framework but only ships one new mode.

## 5. Current Technical Context

### Initiative Metadata

`api/internal/initiatives/model.go` currently stores initiative name, title, description, status, priority, dependency refs, item refs, timestamps, notes, and archive state. Status is already richer than the old plan assumed: `active`, `in_review`, `review_pending`, `completed`, `failed`, and `needs_followup`.

Plan B adds:

- `Mode string`
- `AcceptanceCriteria []string`

Mode is orthogonal to status. Status answers lifecycle/result state; mode answers which execution machinery owns the initiative right now.

### Initiative Service and Handler

`api/internal/initiatives/service.go` owns CRUD, rollup aggregation, item membership, and event dispatch. `handler.go` registers the initiative routes. This package is the correct place for metadata updates and thin HTTP wrappers, but not for all phase orchestration logic.

The new mode/phase orchestration should live in a dedicated package to prevent `initiatives.Service` from becoming a god object.

### Existing Lock

`api/internal/initiativelock/lock.go` already implements a single-agent-per-initiative lock using `.feedback-lock`. Despite the historical filename, the lock now stores generic holder metadata with `RunID`, `Purpose`, `RoundNumber`, and initiative name. The plan should not introduce a parallel lock. It should make the current lock vocabulary explicitly mode/phase-aware while preserving the file name unless there is a strong reason to migrate.

### Existing Feedback Flow

`api/internal/feedback/service.go` has the reusable lifecycle shape:

- reserve round directory
- persist round
- acquire initiative lock
- spawn/continue an initiative-scoped agent
- poll run state
- persist agent output
- apply validated proposals

`api/routes_feedback.go` has useful adapters:

- prompt/context collection
- agentactivity-backed initiative spawn
- promptcatalog skill resolution
- graph/current-state builder
- active item run detection
- run cancellation

These patterns should be extracted or mirrored behind reusable interfaces. Do not copy large chunks into a new initiative-level package.

### Current Initiative Review

`api/internal/initiativereview/` is decision-oriented. It asks whether completed member items collectively justify a terminal initiative decision and may propose follow-up mutations. Initiative-level acceptance review is different: it evaluates the system against `AcceptanceCriteria`, `initiative-plan.md`, execute rounds, and current code state. These flows must remain separate.

### Prompt Catalog

`api/internal/promptcatalog/catalog.go` currently has `initiative-feedback` and `initiative-review`, with `ResolveInitiativeSkill(purpose)` hardcoded for `"feedback"`, `"feedback_continue"`, and `"review"`.

This is a decision boundary that should be extracted: future modes should not require another switch statement that only understands two initiative purposes. The catalog should resolve by explicit mode+phase or by registered purpose values.

### Agent Activity

`api/internal/agentactivity/` is the canonical tracked AgentManager seam. It already supports `OwnerInitiative`, `PurposeFeedback`, `PurposeFeedbackContinue`, and `PurposeInitiativeReview`. This plan adds mode-phase purposes and requires initiative-level spawns to flow through `agentactivity.Service.SpawnInitiative`.

### UI

`ui/src/pages/InitiativeDetailsPage.tsx` currently owns the initiative details route and tabs: `info`, `feedback`, `review`, and `files`. It already has several local concerns. Initiative-level mode should not turn this page into a monolith. Add a mode-aware workspace as extracted components/hooks/services.

`ui/src/services/initiative-service.ts` is the right API seam for mode switching and acceptance criteria. If phase operations grow, add an initiative mode service module rather than overloading the base CRUD service.

## 6. Target End State

After this plan lands:

- Swarm Manager has a small, explicit initiative execution mode framework.
- `item-level` is represented as the default mode, not as an absence of mode.
- `initiative-level` is implemented as a mode definition with registered phases and artifacts.
- Initiatives can be switched between item-level and initiative-level modes through API, CLI, and UI.
- Entering initiative-level mode requires explicit cancellation confirmation if member items have active item-level executions.
- Initiative-level phases run end-to-end:
  - `investigate` produces/updates `findings.md`
  - `plan` produces/updates `initiative-plan.md`
  - `execute` edits the repo according to `initiative-plan.md`, journals an execute round, and marks member items complete through a run-id-validated API
  - `review` produces an acceptance verdict against `AcceptanceCriteria`
- Initiative-level rounds are durable under `initiative-rounds/round-N.json`.
- Single-agent-per-initiative is enforced across feedback, decision review, and initiative-level phases.
- Agent activity records show mode and phase in purpose/metadata.
- The UI exposes a Mode chip, mode-switch flow, acceptance criteria editor, and Initiative Workspace.
- The codebase has clear seams so future modes can register behavior without large-scale duplication.

## 7. Architecture and Responsibility Boundaries

### Core Owns

Swarm Manager core owns these concerns for every mode:

- initiative metadata and membership
- mode selection and mode switching
- locks
- activity records
- API/CLI routing
- common agent spawn/poll/cancel plumbing
- artifact persistence primitives
- graph/item proposal application
- UI service infrastructure and route composition

### Modes Own

Each mode owns:

- unit of execution and validation
- supported phases
- phase transition rules
- artifact names and schemas
- prompt skills and prompt variables
- readiness dimensions
- review semantics
- completion semantics
- how member backlog items are interpreted

For `item-level`, member backlog items are execution units. For `initiative-level`, member backlog items are tracking and scope markers; the initiative is the execution/validation unit.

### Package Boundary

Add a new backend package:

```text
api/internal/initiativemode/
```

This package should own mode definitions, phase definitions, mode-phase service orchestration, artifact storage helpers, and round schemas. It should depend on narrow interfaces for initiatives, backlog, agent activity, prompt rendering, graph/current state, execution cancellation, and locks.

`api/internal/initiatives/` remains CRUD + rollup + membership. It should not own phase runner logic.

`api/internal/feedback/` remains feedback rounds/proposals. Shared code can be extracted into `initiativemode` or a small internal helper if both packages need it.

## 8. Mode Framework Contract

Implement a mode framework with concepts like these. Exact names may vary, but the responsibility split must hold.

```go
type Mode string
type Phase string

const (
    ModeItemLevel       Mode = "item-level"
    ModeInitiativeLevel Mode = "initiative-level"
)

const (
    PhaseInvestigate Phase = "investigate"
    PhasePlan        Phase = "plan"
    PhaseExecute     Phase = "execute"
    PhaseReview      Phase = "review"
)

type Definition struct {
    Mode              Mode
    Label             string
    DefaultPhase      Phase
    Phases            []PhaseDefinition
    ArtifactPolicy    ArtifactPolicy
    ReadinessPolicy   ReadinessPolicy
    TransitionPolicy  TransitionPolicy
}

type PhaseDefinition struct {
    Phase             Phase
    ActivityPurpose   agentactivity.Purpose
    LockPurpose       string
    CatalogID         string
    SkillID           string
    WritesRepo        bool
    OutputArtifacts   []string
    RequiresCriteria  bool
}
```

Minimum required behavior:

- A registry validates mode values and returns definitions.
- Phase start uses `(mode, phase)` to resolve skills, lock purpose, activity purpose, and artifact policy.
- Unknown modes/phases fail closed.
- `item-level` can be registered even if its phase model is mostly a bridge to existing backlog behavior.
- `initiative-level` is not hardcoded in generic services except as a registered definition.

Do not over-engineer plugin loading or dynamic runtime registration. A static Go registry is enough. The important part is that decisions have one home.

## 9. Implementation Strategy

### Phase 0 — Codebase Reconnaissance and Seam Notes

Before edits:

1. Run the required skill-read command from §2.
2. Inspect current implementations named in §2.
3. Update or add a section in `scenarios/swarm-manager/docs/internal/SEAMS.md` describing the intended Initiative Mode Boundary.

This phase prevents the implementation from drifting back into scattered logic.

### Phase 1 — Initiative Model and Storage

Files:

- `api/internal/initiatives/model.go`
- `api/internal/initiatives/service.go`
- `api/internal/initiatives/store.go`
- `api/internal/initiatives/*_test.go`
- `ui/src/types/initiative.ts`
- proto/type mapping files if initiative types are proto-backed in this repo version

Tasks:

1. Add `Mode string` and `AcceptanceCriteria []string` to `Initiative`.
2. Add constants `ModeItemLevel = "item-level"` and `ModeInitiativeLevel = "initiative-level"` or import equivalents from `initiativemode` without creating a cycle.
3. Add `Normalize()` so missing mode loads as `item-level`.
4. Add `ValidateMode`.
5. Extend create/update requests.
6. Reject mode changes on archived initiatives.
7. Keep mode changes out of generic PATCH if the dedicated mode-switch endpoint is present; generic update may set acceptance criteria but mode switching should use the lifecycle endpoint.
8. Tests for normalization, validation, marshal/unmarshal, and store round-trip.

Contract:

- Missing mode on disk is accepted and normalized to `item-level`.
- Invalid mode is rejected.
- Default mode is visible to callers after load.

### Phase 2 — Initiative Mode Registry and Decisions

Files:

- `api/internal/initiativemode/definition.go`
- `api/internal/initiativemode/registry.go`
- `api/internal/initiativemode/transition.go`
- `api/internal/initiativemode/*_test.go`
- `api/internal/promptcatalog/catalog.go`
- `api/internal/promptcatalog/catalog_test.go`
- `api/internal/agentactivity/types.go`
- `api/internal/agentactivity/service_test.go`

Tasks:

1. Add the `initiativemode` package.
2. Register `item-level` and `initiative-level`.
3. Define initiative-level phases and their activity/lock/catalog/skill metadata.
4. Add activity purposes:
   - `initiative_investigate`
   - `initiative_plan`
   - `initiative_execute`
   - `initiative_execution_review`
5. Replace or extend `promptcatalog.ResolveInitiativeSkill(purpose)` with a resolver that supports mode+phase, while preserving existing feedback/review lookups.
6. Add tests that prove all registered initiative-level phases resolve to catalog entries and activity purposes.

Contract:

- Mode/phase decisions live in one package.
- Prompt catalog does not grow an unbounded hardcoded switch.
- Unknown mode/phase combinations return a typed validation error.

### Phase 3 — Lock and Activity Unification

Files:

- `api/internal/initiativelock/lock.go`
- `api/internal/feedback/service.go`
- `api/internal/initiativereview/service.go`
- `api/internal/initiativemode/service.go`
- tests for all three callers

Tasks:

1. Keep `.feedback-lock` unless a migration is deliberately chosen; document that it is the initiative agent lock.
2. Add lock purpose constants:
   - `feedback`
   - `feedback_continue`
   - `initiative_review`
   - `initiative_investigate`
   - `initiative_plan`
   - `initiative_execute`
   - `initiative_execution_review`
3. Ensure feedback and decision-review use the same lock API as initiative mode phases.
4. Add tests proving different holder purposes still conflict on the same initiative.
5. Ensure lock holder metadata is enough for UI override/cancellation prompts.

Contract:

- Only one active initiative-scoped agent can hold the initiative lock.
- Purpose is diagnostic/routing metadata, not a separate lock namespace.

### Phase 4 — Shared Phase Runner

Files:

- `api/internal/initiativemode/service.go`
- `api/internal/initiativemode/artifacts.go`
- `api/internal/initiativemode/rounds.go`
- `api/internal/initiativemode/context.go`
- `api/internal/initiativemode/poller.go`
- `api/routes_initiative_mode.go` or equivalent server wiring

Tasks:

1. Implement `StartPhase(ctx, StartPhaseRequest)`.
2. Load initiative and validate mode supports phase.
3. Validate transition/state rules.
4. Acquire initiative lock with phase lock purpose.
5. Build prompt context:
   - initiative metadata
   - acceptance criteria
   - member item summaries
   - member item deliverables where useful
   - current graph/current materialized state
   - prior initiative-level rounds
   - current `findings.md` / `initiative-plan.md`
6. Resolve skill through mode+phase definition.
7. Spawn through `agentactivity.Service.SpawnInitiative`.
8. Persist provisional round with `agent_thinking` or equivalent status.
9. Poll run state and persist terminal output.
10. Extract phase artifacts from agent output and write them atomically.
11. Release lock on terminal state.

Contract:

- Phase endpoints are thin wrappers over the shared runner.
- The runner does not contain phase-specific prompt prose.
- Phase-specific parsing/validation is delegated by phase definition.

### Phase 5 — Artifact and Round Model

Files:

- `api/internal/initiativemode/artifacts.go`
- `api/internal/initiativemode/rounds.go`
- `api/internal/initiativemode/readiness.go`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`

Tasks:

1. Add initiative artifact paths:
   - `<initiative>/findings.md`
   - `<initiative>/initiative-plan.md`
   - `<initiative>/initiative-rounds/round-N.json`
2. Define a common round envelope:
   - `round`
   - `phase`
   - `mode`
   - `generated_at`
   - `run_id`
   - `status`
   - `readiness`
   - `items`
   - `artifact_updates`
   - phase-specific payload
3. Add readiness policy for initiative-level:
   - `problem_clarity`
   - `scope_defined`
   - `approach_solid`
   - `testable`
   - `risk_awareness`
   - `coupling_understood`
   - `system_acceptance_defined`
4. Use boost divisor `N=2` for initiative-level readiness.
5. Add unit tests for round numbering, atomic writes, malformed JSON rejection, and readiness scoring.

Contract:

- Round files are append-only evidence.
- `findings.md` and `initiative-plan.md` are current-state artifacts; prior versions are preserved by round summaries, not by numbered markdown files.

### Phase 6 — Initiative-Level Skills

Files:

- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-investigate/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-plan/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-execute/SKILL.md`
- `scenarios/prompt-manager/store/skills/packs/core/swarm-manager-initiative-execution-review/SKILL.md`
- prompt catalog tests/simulations

Skill contracts:

1. `investigate`
   - read-only on code
   - writes `findings.md` content
   - emits `phase=investigate` round envelope
   - never edits backlog items or repo files
2. `plan`
   - reads latest findings, graph, item deliverables, and acceptance criteria
   - writes `initiative-plan.md`
   - emits 7-dimension readiness
3. `execute`
   - edits repo files according to `initiative-plan.md`
   - emits completed milestones and `replan_needed`
   - marks member items complete only through the documented API
   - never writes per-item `plan.md`
4. `execution-review`
   - read-only on code
   - evaluates `AcceptanceCriteria`
   - emits verdict: `accept`, `request_replan`, or `request_changes`
   - never completes the initiative automatically

Tests:

- `prompt-manager skill read`/simulate for each skill.
- Token budget checks for typical initiative context.
- JSON envelope shape validation.

### Phase 7 — Phase APIs

Files:

- `api/internal/initiativemode/handler.go`
- `api/routes_initiative_mode.go`
- `ui/src/lib/api-endpoints.ts`
- `docs/reference/api-endpoints.md`

Endpoints:

- `GET /api/v1/initiatives/{name}/mode`
- `POST /api/v1/initiatives/{name}/mode`
- `GET /api/v1/initiatives/{name}/workspace`
- `POST /api/v1/initiatives/{name}/phases/{phase}`
- Optional stable aliases:
  - `POST /api/v1/initiatives/{name}/investigate`
  - `POST /api/v1/initiatives/{name}/plan`
  - `POST /api/v1/initiatives/{name}/execute`
  - `POST /api/v1/initiatives/{name}/review`
- `POST /api/v1/initiatives/{name}/items/{item-ref}/complete`
- `POST /api/v1/initiatives/{name}/complete`

Preferred internal routing:

- Stable aliases call the generic phase handler.
- The generic phase handler calls `initiativemode.Service.StartPhase`.

Status contracts:

- `400` invalid mode/phase/body.
- `404` initiative not found.
- `409` mode/transition conflict or in-flight item executions without cancellation confirmation.
- `423` initiative lock held.
- `202` phase run accepted/spawned.

### Phase 8 — Mode Switch Protocol

Tasks:

1. Implement `POST /api/v1/initiatives/{name}/mode`.
2. Body:

   ```json
   {
     "mode": "initiative-level",
     "cancel_in_flight": true
   }
   ```

3. Entering initiative-level:
   - enumerate active member item-level executions/activities
   - if any exist and `cancel_in_flight=false`, return `409` with blockers
   - if `cancel_in_flight=true`, cancel them through execution/activity services
   - set mode
4. Returning to item-level:
   - reject if an initiative-level phase is active
   - preserve initiative-level artifacts as history
   - set mode to `item-level`
5. Reject mode changes on archived initiatives.

Contract:

- Mode switch is not a generic metadata PATCH.
- No silent cancellation.
- No artifact deletion on drain-back.

### Phase 9 — Execute Phase and Item Completion API

Tasks:

1. Add run-id-validated item completion endpoint.
2. The active execute run ID must match the caller-provided run ID or authenticated agent context.
3. Mark member item complete only if it belongs to the initiative.
4. Journal completion in the execute round.
5. Surface `replan_needed=true` in workspace state.

Contract:

- Initiative-level execute agent may update repo files.
- It may not mutate backlog specs directly.
- Item completion is audited and mediated by API.

### Phase 10 — CLI

Files:

- `scenarios/swarm-manager/cli` sources (`cli/app.go` or current command package)
- `docs/reference/cli-commands.md`

Commands:

```bash
swarm-manager initiative mode set <name> <item-level|initiative-level> [--cancel-in-flight]
swarm-manager initiative investigate <name>
swarm-manager initiative plan <name>
swarm-manager initiative execute <name>
swarm-manager initiative review <name>
swarm-manager initiative complete <name>
```

CLI commands should call the API rather than duplicating filesystem writes.

### Phase 11 — UI Service and Types

Files:

- `ui/src/types/initiative.ts`
- `ui/src/types/initiative-mode.ts` (new if useful)
- `ui/src/services/initiative-service.ts`
- `ui/src/services/initiative-mode-service.ts` (preferred for phase operations)
- `ui/src/lib/api-endpoints.ts`
- service tests

Tasks:

1. Add `mode` and `acceptanceCriteria` to initiative normalization.
2. Add mode-switch and phase-start methods.
3. Add workspace fetch method.
4. Add type-safe models for phases, rounds, artifacts, review verdicts, and blockers.
5. Tests for snake_case to camelCase normalization.

### Phase 12 — Initiative Workspace UI

Files:

- `ui/src/pages/InitiativeDetailsPage.tsx`
- `ui/src/components/initiative/initiative-mode-chip.tsx`
- `ui/src/components/initiative/initiative-mode-switch-dialog.tsx`
- `ui/src/components/initiative/initiative-workspace.tsx`
- `ui/src/components/initiative/initiative-acceptance-criteria-editor.tsx`
- `ui/src/components/initiative/initiative-phase-controls.tsx`
- `ui/src/components/initiative/initiative-round-timeline.tsx`
- `ui/src/consts/selectors.ts`

Tasks:

1. Add a compact Mode chip in the initiative header.
2. Add a mode-switch action with in-flight blocker/cancellation confirmation.
3. Add an Initiative Workspace tab visible when `mode === "initiative-level"`.
4. Workspace sections:
   - Findings
   - Initiative Plan
   - Rounds
   - Execute state / replan-needed banner
   - Acceptance Criteria
   - Acceptance Review verdict
5. Keep `InitiativeDetailsPage.tsx` as route composition; put new behavior in extracted components/hooks.
6. Add stable selectors for new affordances.
7. Add cost-hint modal before execute; acknowledgement persists for 24h.

React coherence constraints:

- Server data uses React Query or existing service/store patterns.
- Local UI state stays local unless reused across surfaces.
- No second modal framework.
- No duplicate file workspace implementation; reuse existing initiative file service.

### Phase 13 — Docs

Files:

- `scenarios/swarm-manager/docs/guides/initiative-level-mode.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/reference/api-endpoints.md`
- `scenarios/swarm-manager/docs/reference/cli-commands.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/manifest.json`

Tasks:

1. Add operator guide for when and how to use initiative-level mode.
2. Update `EXECUTION-MODES.md` open questions to point to resolved decisions.
3. Register new API and CLI contracts.
4. Register new docs in manifest.
5. Add `[CODE: ...]` references to the new mode framework files once implemented.

### Phase 14 — Validation and Restart

Run targeted tests as each phase lands, then full scenario validation:

```bash
cd scenarios/swarm-manager/api && go test ./internal/initiatives/... ./internal/initiativemode/... ./internal/initiativelock/... ./internal/agentactivity/... ./internal/promptcatalog/... -timeout 300s
cd scenarios/swarm-manager/ui && npm test -- InitiativeDetailsPage initiative-service initiative-mode
cd scenarios/swarm-manager/ui && npm run typecheck
cd scenarios/swarm-manager && make test
vrooli scenario restart swarm-manager
vrooli scenario test swarm-manager
vrooli scenario test prompt-manager
vrooli scenario test agent-manager
```

Use longer timeouts where needed; full scenario tests can take several minutes.

## 10. Contract Decisions

1. **Mode is explicit metadata.** `mode` lives on the initiative and defaults to `item-level` on load.
2. **Mode switch is a lifecycle operation.** Use the dedicated endpoint/CLI; do not rely on generic PATCH for mode transitions.
3. **Status and mode are separate.** `status` says lifecycle/result; `mode` says execution machinery.
4. **Static mode registry.** No dynamic plugin system in this plan.
5. **Phase runner is shared.** Endpoint aliases are allowed, but internals route through a common phase service.
6. **Initiative-level artifacts are current-state files plus round evidence.** Markdown files hold latest state; round JSON preserves history.
7. **Single initiative agent lock.** Feedback, decision-review, and initiative-level phases all contend on the same lock.
8. **Member items remain scope markers in initiative-level mode.** They are not independently executed while the initiative-level mode is active.
9. **Execute agent marks item completion through API only.** No direct spec mutation.
10. **Acceptance review is distinct.** `swarm-manager-initiative-execution-review` is not `swarm-manager-initiative-review`.
11. **Operator completes explicitly.** A review verdict of `accept` enables completion but does not auto-complete the initiative.
12. **Replan is a loop state, not a separate endpoint.** Execute can signal `replan_needed`; operator runs plan/investigate again.
13. **Cost hint is UI-only.** No hard budget gate in this plan.

## 11. Testing Plan

### Backend Unit Tests

- Initiative model normalization and validation.
- Mode registry rejects unknown modes/phases.
- Prompt catalog resolves every mode+phase.
- Agent activity accepts new purposes.
- Lock conflicts across different purposes.
- Phase runner validates mode/phase, lock contention, and transition rules.
- Artifact writes are atomic and round numbering is stable.
- Readiness scoring for seven dimensions.
- Mode switch rejects in-flight items without `cancel_in_flight`.
- Mode switch cancels in-flight items with explicit confirmation.
- Item completion endpoint rejects wrong run IDs and non-member refs.

### Backend Integration / E2E

- Full mode switch -> investigate -> plan -> execute -> review -> complete flow against a temp filesystem and stub AgentManager.
- Execute with `replan_needed=true` surfaces replan state.
- Drain-back to item-level is blocked while a phase is active and succeeds after completion.
- Existing feedback and decision-review still lock correctly.

### UI Tests

- Initiative service normalizes `mode` and `acceptance_criteria`.
- Mode chip renders.
- Mode-switch dialog lists blockers and sends `cancel_in_flight`.
- Workspace tab is visible only in initiative-level mode.
- Acceptance criteria editor updates via service.
- Phase controls call the correct service methods.
- Cost-hint modal appears before execute and respects 24h acknowledgement.
- Existing feedback/review/files tabs still render.

### Skill Tests

- Each new skill renders through prompt-manager.
- Each new skill emits the expected JSON envelope in simulation.
- Prompt token budgets stay within documented limits.

### Docs / Cross-Scenario

- New docs registered in manifest.
- API/CLI reference includes new endpoints/commands.
- `make test` and `vrooli scenario test swarm-manager` pass.
- Adjacent `prompt-manager` and `agent-manager` scenarios pass baseline tests.

## 12. Rollout / Validation Checklist

- [ ] `Initiative.Mode` and `AcceptanceCriteria` exist and round-trip.
- [ ] `initiativemode` registry exists with `item-level` and `initiative-level`.
- [ ] Mode/phase definitions include activity purpose, lock purpose, skill/catalog identity, and artifact policy.
- [ ] Prompt catalog resolves mode+phase skills.
- [ ] Agent activity supports initiative-level purposes.
- [ ] Shared phase runner starts, tracks, polls, persists, and releases locks.
- [ ] Initiative-level artifacts persist under the initiative folder.
- [ ] Mode-switch endpoint enforces cancellation gate.
- [ ] Item completion endpoint enforces active execute run ID.
- [ ] UI has Mode chip, mode-switch dialog, Initiative Workspace, acceptance criteria editor, phase controls, and cost hint.
- [ ] New skills are present and simulate successfully.
- [ ] Docs and manifest are updated.
- [ ] `make test` passes in `scenarios/swarm-manager`.
- [ ] `vrooli scenario restart swarm-manager` succeeds.
- [ ] `vrooli scenario test swarm-manager`, `prompt-manager`, and `agent-manager` pass.

## 13. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---:|---:|---|
| Framework work becomes over-abstract and delays delivery | Medium | High | Keep registry static and concrete. Implement only the abstractions needed by item-level + initiative-level. |
| Implementation still scatters `initiative-level` checks | Medium | High | Require mode/phase decisions to live in `initiativemode`; add tests around registry and transition rules. |
| Phase runner duplicates feedback service behavior | Medium | Medium | Extract reusable patterns or use narrow adapters; do not copy large blocks without consolidating common concerns. |
| Lock filename `.feedback-lock` confuses future agents | Medium | Low | Document it in `initiativelock` comments and `SEAMS.md` as the initiative agent lock; rename only if migration cost is justified. |
| Initiative-level execute crashes after partial item completion | Medium | High | Journal execute rounds and require next execute/investigate to read code state plus round history. |
| Agent directly edits item specs | Medium | High | Skill hard rule plus API-only completion endpoint; tests assert completion endpoint contract. |
| UI page becomes too large | Medium | Medium | Extract workspace components and service hooks; keep route component as composition. |
| Existing decision-review semantics regress | Low | High | Keep acceptance review separate; add regression tests for `initiativereview`. |
| Prompt catalog grows confusing purpose names | Medium | Medium | Centralize names in mode definitions and activity constants; catalog tests cover each purpose. |
| Future modes still require large edits | Medium | High | Definition of Done requires an explicit future-mode extension note and `SEAMS.md` update. |

## 14. Non-Goals and Prohibited Patterns

- No hidden "only two modes forever" assumptions.
- No broad `if mode == "initiative-level"` branching across handlers, UI pages, and services.
- No direct spec writes from initiative-level agents.
- No silent cancellation of item-level executions.
- No separate initiative lock implementation.
- No reuse of decision-oriented initiative review for acceptance review.
- No dynamic plugin architecture.
- No cross-initiative execution.
- No per-item workshop schema changes.
- No deleting initiative-level artifacts when switching back to item-level.
- No unregistered docs.
- No UI selectors outside `selectors.ts`.

## 15. Definition of Done

This plan is complete when:

1. The repo contains a tested `initiativemode` framework with a static registry for `item-level` and `initiative-level`.
2. Initiative metadata includes `Mode` and `AcceptanceCriteria`, with load normalization and validation.
3. Mode switching is implemented as a lifecycle operation with explicit in-flight cancellation behavior.
4. Initiative-level phases run through a shared phase runner and produce durable artifacts/rounds.
5. New prompt-manager skills exist and are catalog-resolvable.
6. Agent activity and locks represent initiative-level phases cleanly.
7. Initiative-level execute can mark member items complete only through the run-id-validated API.
8. Initiative-level acceptance review is separate from decision-oriented initiative review.
9. The UI exposes mode-aware controls and workspace components without bloating `InitiativeDetailsPage.tsx`.
10. Docs, API reference, CLI reference, manifest, and `SEAMS.md` are updated.
11. All targeted backend, UI, skill, scenario, and adjacent-scenario tests pass.
12. A future agent can read this plan plus `SEAMS.md` and understand how to add another mode without reconstructing this conversation.
