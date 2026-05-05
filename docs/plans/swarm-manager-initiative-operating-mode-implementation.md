# Swarm Manager Operating Mode Framework + Initiative Modes

## 1. Purpose

Implement Swarm Manager's operating mode framework by shipping two non-default initiative-scoped modes:

- `holistic-loop`: a holistic `investigate -> plan -> execute -> review -> replan` workflow for initiatives whose member backlog items are the wrong unit of execution or validation.
- `phased-plan-drain`: a sequential handoff workflow for large multi-phase plans where each agent completes the earliest contiguous phase(s) it can fully finish, persists a final handoff, and lets the next agent continue from the accumulated handoffs.

The architectural feature is an **operating mode framework** where a mode defines entity scope, phases, run strategy, artifact policy, backlog/audit reconciliation policy, metrics policy, prompt routing, locking, and UI workspace surface.

The two shipped modes deliberately exercise different methodologies:

- `holistic-loop` is an operator-gated investigation/planning/execution/review loop.
- `phased-plan-drain` is a stable-plan sequential execution chain with progress classification and handoff accumulation.

Implementing both in the same plan is intentional. It forces the framework to support more than one real methodology from the start, instead of fitting one mode and treating extensibility as a theoretical claim.

The target is:

- `item-level` remains the default mode and existing backlog-item execution behavior remains intact.
- `holistic-loop` and `phased-plan-drain` both ship as registered non-default modes implemented through the framework.
- Modes can vary the unit of work, phase graph, run strategy, artifact model, review semantics, and audit policy while reusing the same core mode infrastructure.
- The plan explicitly protects Swarm Manager's audit role: backlog items and initiatives are not only execution inputs; they are the durable project-management paper trail.

## 2. Required Reading

Run this command before implementing:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health intent-clarification seam-discovery-and-enforcement decision-boundary-extraction boundary-of-responsibility-enforcement api-steer react-coherence
```

Also read:

- `path:scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md` - current conceptual framing; update it to include the operating-mode model and both shipped non-default modes.
- `path:scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md` - Swarm Manager's staging/review mental model and existing seams.
- `path:scenarios/swarm-manager/docs/guides/workshop-workflow.md` - item-level workshop loop, readiness model, and per-item plan handoff.
- `path:scenarios/swarm-manager/docs/internal/SEAMS.md` - current seam inventory; update it with the new Operating Mode Boundary.
- `path:scenarios/swarm-manager/api/internal/initiatives/model.go` - initiative metadata currently has no mode or acceptance criteria.
- `path:scenarios/swarm-manager/api/internal/initiatives/service.go` and `handler.go` - initiative CRUD, rollup, membership, and event dispatch.
- `path:scenarios/swarm-manager/api/internal/initiativelock/lock.go` - existing single-agent-per-initiative lock using `.feedback-lock`.
- `path:scenarios/swarm-manager/api/internal/feedback/service.go` and `routes_feedback.go` - proven initiative-scoped spawn/poll/cancel/state-builder patterns.
- `path:scenarios/swarm-manager/api/internal/initiativereview/` - existing decision-oriented initiative review, intentionally distinct from holistic-loop acceptance review.
- `path:scenarios/swarm-manager/api/internal/execution/` - current backlog-item execution lifecycle, finalization, retry/follow-up, and GCT review integration.
- `path:scenarios/swarm-manager/.vrooli/service.json` - scenario dependency config that declares AgentManager profile source files.
- `path:scenarios/swarm-manager/.vrooli/agent-profiles/default.json` - current single Swarm Manager AgentManager profile.
- `path:scenarios/swarm-manager/api/internal/agentmanager/service.go` and `profile.go` - AgentManager integration seam, scenario profile reconciliation, and default profile ref selection.
- `path:scenarios/swarm-manager/api/internal/agentactivity/types.go` and `service.go` - canonical tracked AgentManager usage.
- `path:scenarios/swarm-manager/api/internal/eventlog/types.go` and `emitter.go` - append-only analytics source; extend for operating mode events.
- `path:scenarios/swarm-manager/api/internal/stats/` - event-log-derived metrics engine and stats response.
- `path:scenarios/swarm-manager/api/internal/promptcatalog/catalog.go` - prompt/skill catalog; current initiative resolver is hardcoded around feedback/review.
- `path:scenarios/agent-manager/api/internal/orchestration/profile_reconcile.go` - source-of-truth behavior for reconciling scenario-owned profile JSON files.
- `path:scenarios/swarm-manager/ui/src/pages/InitiativeDetailsPage.tsx` - current initiative detail route and tab composition.
- `path:scenarios/swarm-manager/ui/src/surfaces/graph/components/StatsPanel.tsx` - existing stats UI.
- `path:scenarios/swarm-manager/ui/src/services/initiative-service.ts` and `path:ui/src/types/initiative.ts` - UI initiative API seam and type surface.
- `path:scenarios/swarm-manager/ui/src/types/stats.ts` and `path:ui/src/services/stats-service.ts` - stats API seam.
- `path:scenarios/swarm-manager/ui/src/consts/selectors.ts` - selector registry for new UI affordances.

## 3. Problem Statement

Swarm Manager currently assumes backlog items are the unit of execution and validation. That assumption is valuable when items are right-sized, independent, stable, and reviewable in isolation. It fails when work is coupled across items, when partial intermediate states break the system, when the right item shape shifts during execution, or when success can only be judged at initiative/system scope.

The 2026-04-27 sandboxing initiatives were the empirical cue. Both initiatives changed the shared substrate used by agent executions. Completing one backlog item changed the runtime path that subsequent Swarm Manager item executions depended on, leaving the harness unstable. The successful workaround was a holistic loop outside Swarm Manager:

1. Investigate current code and initiative state.
2. Produce a consolidated initiative-wide plan.
3. Execute in waves against that plan.
4. Re-investigate and replan as ground truth changed.
5. Review whether the system as a whole satisfied the initiative.

That should be a first-class Swarm Manager operating mode with persistence, auditability, async operator cadence, lifecycle control, and metrics.

There is a second, broader architecture problem. Operating modes should not be treated as phase lists over one initiative runner. That is not enough.

Modes differ on several independent axes:

- **Scope:** backlog item, initiative, scenario, repo/workspace, or a composite scope.
- **Methodology:** workshop-then-execute, investigate-plan-execute-review, phased plan drain, parallel worker fanout, review-gated continuation, etc.
- **Run strategy:** one run per phase, a sequential chain of handoff-fed runs, a bounded loop, a classifier-driven continuation, or controlled parallelism.
- **Agent profile policy:** which AgentManager profile is used for each phase, based on required reasoning depth, write permissions, sandbox policy, timeout, and cost.
- **Audit policy:** how backlog items and initiatives are created, updated, completed, linked to runs, and preserved as a paper trail.
- **Metrics policy:** what counts as use, success, rework, friction, handoff quality, completion, or follow-up.

The implementation must recognize those axes now. Shipping both `holistic-loop` and `phased-plan-drain` is the validation that the abstraction is real.

## 4. Scope

### In Scope

- Add explicit initiative `Mode` metadata with values:
  - `item-level` - default, existing behavior.
  - `holistic-loop` - holistic initiative execution mode.
  - `phased-plan-drain` - sequential handoff execution against a multi-phase initiative plan.
- Add `AcceptanceCriteria []string` to initiatives for holistic-loop and phased-plan-drain acceptance review.
- Introduce a backend **operating mode framework**:
  - static mode registry
  - scope policy
  - phase graph / transition policy
  - run strategy policy
  - artifact policy
  - prompt/context policy
  - AgentManager profile policy
  - backlog/audit reconciliation policy
  - metrics/event policy
  - shared lock and activity tracking integration
  - shared round/run persistence primitives
- Implement holistic-loop mode through that framework:
  - scope: initiative
  - phase graph: `investigate -> plan -> execute -> review`, with `execute` able to signal `replan_needed`
  - run strategy: one initiative-scoped agent run per phase, operator-triggered between phases
  - artifacts under a mode-scoped root: `modes/holistic-loop/findings.md`, `modes/holistic-loop/initiative-plan.md`, `modes/holistic-loop/rounds/round-NNN.json`
  - readiness: existing five dimensions plus `coupling_understood` and `system_acceptance_defined`
  - audit policy: member backlog items remain scope/progress markers and may be marked complete only through run-id-validated APIs
- Implement phased-plan-drain mode through that framework:
  - scope: initiative
  - phase graph: `prepare_plan -> execute_next -> classify_progress -> review`, with `classify_progress` deciding `continue`, `blocked`, `replan`, or `complete`
  - run strategy: sequential handoff chain where each execute run receives the plan plus prior final handoffs
  - artifacts under a mode-scoped root: `modes/phased-plan-drain/phased-plan.md`, `modes/phased-plan-drain/progress.json`, `modes/phased-plan-drain/rounds/round-NNN.json`, and handoffs inside round JSON
  - audit policy: backlog reconciliation runs after each progress classification, so completed phases and follow-up work are reflected in Swarm Manager entities
- Add mode-switch API/CLI/UI behavior, including explicit cancellation of in-flight item-level executions when entering holistic-loop or phased-plan-drain mode.
- Add phase-level AgentManager profile selection:
  - declare multiple Swarm Manager-owned profiles under `path:scenarios/swarm-manager/.vrooli/agent-profiles/`
  - register those profile source files in `path:scenarios/swarm-manager/.vrooli/service.json`
  - let each operating-mode phase select a profile by stable `profileKey`
  - record selected profile keys in agent activity, event payloads, round envelopes, and stats
- Add mode event logging and stats:
  - mode changes
  - phase starts/completions/failures/cancellations
  - replan signals
  - backlog reconciliation events
  - per-mode usage, duration, outcome, replan, follow-up, and acceptance metrics
- Add a mode-aware Initiative Workspace UI tab/section for holistic-loop artifacts, rounds, execution status, acceptance criteria, and review verdicts.
- Add a Stats UI surface for operating mode metrics.
- Add prompt-manager skills:
  - `swarm-manager-holistic-loop-investigate`
  - `swarm-manager-holistic-loop-plan`
  - `swarm-manager-holistic-loop-execute`
  - `swarm-manager-holistic-loop-review`
  - `swarm-manager-phased-plan-prepare`
  - `swarm-manager-phased-plan-execute-next`
  - `swarm-manager-phased-plan-classify-progress`
  - `swarm-manager-phased-plan-review`
- Extend `promptcatalog` and `agentactivity` around explicit mode/phase/run-strategy purposes.
- Extend `agentmanager` so spawn requests can carry a profile reference selected by the operating mode definition while preserving `swarm-manager/default` as the fallback for existing callers.
- Keep existing decision-oriented `swarm-manager-initiative-review` intact and separate.
- Update docs:
  - `EXECUTION-MODES.md`
  - `path:docs/guides/holistic-loop-mode.md`
  - `path:docs/guides/phased-plan-drain-mode.md`
  - `path:docs/reference/api-endpoints.md`
  - `path:docs/reference/cli-commands.md`
  - `path:docs/internal/SEAMS.md`
  - `path:docs/manifest.json`
- Automated tests for backend, UI, skills, stats, docs, and cross-scenario validation.

### Out of Scope

- Dynamic runtime plugin loading for modes.
- Runtime creation of ad hoc AgentManager profiles from operating-mode data.
- Letting operators edit profile selection per initiative in the first pass.
- Auto-detecting which mode an initiative should use.
- Automatic scheduling of phased-plan-drain runs without operator action.
- Parallel worker fanout for phased-plan-drain.
- Sophisticated semantic proof that a phase is complete beyond the classifier/review contract.
- Cross-initiative execution.
- Tech-tree integration.
- Hard cost-budget enforcement.
- Rewriting backlog item workshop schemas.
- Reusing the decision-oriented initiative review as acceptance review.
- Direct writes by holistic-loop execute agents to backlog `spec.json` files.
- Replacing item-level execution internals in the first pass; item-level can be represented as a registered bridge over existing behavior.

## 5. Current Technical Context

### Initiative Metadata

`path:api/internal/initiatives/model.go` currently stores initiative name, title, description, status, priority, dependency refs, item refs, timestamps, notes, and archive state. It does not store mode or acceptance criteria.

Status currently supports `active`, `in_review`, `review_pending`, `completed`, `failed`, and `needs_followup`.

Add:

- `Mode string`
- `AcceptanceCriteria []string`

Mode is orthogonal to status. Status answers lifecycle/result state; mode answers which execution machinery owns the initiative right now.

### Initiative Service and Handler

`path:api/internal/initiatives/service.go` owns CRUD, rollup aggregation, item membership, and event dispatch. It should own metadata updates and thin mode metadata lifecycle calls, but not phase orchestration, run strategy, prompt building, polling, or artifact parsing.

### Existing Lock

`path:api/internal/initiativelock/lock.go` implements a single-agent-per-initiative lock using `.feedback-lock`. The lock holder stores `RunID`, `Purpose`, `RoundNumber`, `AcquiredBy`, and initiative name.

The implementation should not introduce a parallel lock. It should broaden vocabulary from feedback/review to operating-mode purposes while preserving the filename unless a deliberate migration is chosen.

### Existing Feedback Flow

`path:api/internal/feedback/service.go` has a useful initiative-scoped lifecycle:

- reserve round directory
- persist provisional round
- check item-level active runs
- acquire initiative lock
- spawn/continue an initiative-scoped agent
- poll run state
- persist agent output
- parse/validate proposals
- apply accepted mutations through the proposals layer

`path:api/routes_feedback.go` also contains useful adapters:

- prompt/context collection
- agentactivity-backed initiative spawn
- promptcatalog skill resolution
- graph/current-state builder
- active item run detection
- run cancellation

The new operating mode framework should extract or mirror the patterns behind narrow interfaces. Do not copy large chunks into a new package.

### Current Backlog-Item Execution

`path:api/internal/execution/` owns the governed backlog-item run lifecycle. It creates execution records, builds prompts from per-item deliverables, routes AgentManager spawns through agentactivity, runs finalization/GCT review, supports cancellation, retry, and follow-up, and emits execution events.

For this plan:

- Existing item-level execution stays intact.
- The operating mode registry may represent `item-level` as a bridge definition that points to existing execution/workshop/review flows.
- Mode-level stats should distinguish backlog item execution mode from the governed `execution.Mode` values `manual` and `yolo`; those are start policies, not operating modes.

### Current Initiative Review

`path:api/internal/initiativereview/` is decision-oriented. It asks whether completed member items collectively justify a terminal initiative decision and may propose follow-up mutations. It can run fresh GCT checks against affected scenarios.

Holistic-loop acceptance review is different: it evaluates the system against `AcceptanceCriteria`, `modes/holistic-loop/initiative-plan.md`, execute rounds, code state, and optionally fresh GCT/control-tower evidence. Phased-plan-drain review similarly evaluates `AcceptanceCriteria`, `modes/phased-plan-drain/phased-plan.md`, progress state, handoffs, and code state. These flows must remain separate from decision-oriented initiative review even if they share small infrastructure.

### Prompt Catalog

`path:api/internal/promptcatalog/catalog.go` has `initiative-feedback` and `initiative-review`, with `ResolveInitiativeSkill(purpose)` hardcoded for `"feedback"`, `"feedback_continue"`, and `"review"`.

This is a decision boundary. Future modes should not require another switch that only understands a few initiative purposes. The catalog should resolve by registered mode+phase, or by a structured purpose registered by the mode definition.

### Agent Activity

`path:api/internal/agentactivity/` is the canonical tracked AgentManager seam. It already supports `OwnerInitiative`, `PurposeFeedback`, `PurposeFeedbackContinue`, and `PurposeInitiativeReview`, with free-form `Metadata`.

This plan adds mode/phase/run-strategy purposes and requires operating-mode spawns to flow through `agentactivity.Service.SpawnInitiative`. Records should always include metadata keys such as:

- `entrypoint`
- `operating_mode`
- `phase`
- `run_strategy`
- `agent_profile_key`
- `round_number`
- `artifact_set`

### AgentManager Profiles

Swarm Manager currently declares one scenario-owned AgentManager profile source at `path:scenarios/swarm-manager/.vrooli/agent-profiles/default.json`. The scenario dependency config in `path:scenarios/swarm-manager/.vrooli/service.json` lists that file under `dependencies.scenarios.agent-manager.config.profiles.sources`.

`path:api/internal/agentmanager/service.go` reconciles scenario profile sources by calling AgentManager's `/api/v1/profiles/reconcile-scenario` endpoint, then all current spawn paths call `defaultProfileRef()`, which returns `ProfileRef{ProfileKey: "swarm-manager/default"}`. `SpawnResearch`, `SpawnBacklog`, and `SpawnInitiative` therefore all use the same runner/model/tool policy today.

AgentManager already supports multiple scenario-owned profile source files. `path:scenarios/agent-manager/api/internal/orchestration/profile_reconcile.go` reads every source listed in the scenario manifest, requires each `profileKey` to start with the owning scenario prefix, and records owner/source metadata. The operating-mode work should use that existing profile-source contract instead of creating profiles at runtime or sending inline defaults on every run.

The missing Swarm Manager seam is phase-level profile selection:

- `agentmanager` spawn requests need an optional `ProfileKey` or `ProfileRef` field.
- `agentmanager.AgentService` needs a `profileRefFor(key)` helper that falls back to `swarm-manager/default` only when no explicit key is supplied.
- `agentactivity` records and operating-mode round envelopes need to persist the selected profile key.
- `operatingmode` definitions need a profile policy per phase, so cost/quality choices are registered with the methodology instead of scattered through spawn call sites.

### Event Log and Stats

`path:api/internal/eventlog/` is the append-only source for stats. The current event catalog includes backlog, initiative, execution, queue, workshop, clarification, review, view, and system migration events. It has no operating-mode events.

`path:api/internal/stats/` derives analytics from the event log. Current `StatsResponse` contains history, throughput, timing, scope, blocking, agent, dashboard, and review sections. It has no mode-level metrics.

Operating modes must be observable at the event-log layer, not only inferred from UI state or agentactivity metadata.

### UI

`path:ui/src/pages/InitiativeDetailsPage.tsx` currently owns the initiative detail route and tabs: `info`, `feedback`, `review`, and `files`. It is already large. New mode behavior should be extracted into mode workspace components/hooks/services.

`path:ui/src/services/initiative-service.ts` is the right seam for initiative metadata. Add a separate `initiative-mode-service.ts` for phase/workspace operations if the surface grows.

`path:ui/src/surfaces/graph/components/StatsPanel.tsx` renders stats tabs. Add mode metrics there or in extracted stats components, without turning the panel into an unbounded monolith.

## 6. Target End State

After this plan lands:

- Swarm Manager has a small, explicit operating mode framework.
- `item-level` is represented as the default operating mode, not as absence of mode.
- `holistic-loop` is implemented as a registered mode with scope, phase graph, run strategy, artifacts, backlog reconciliation, metrics, prompt, lock, and UI policies.
- `phased-plan-drain` is implemented as a registered mode with a sequential handoff run strategy, progress classifier, artifacts, backlog reconciliation, metrics, prompt, lock, and UI policies.
- Initiatives can be switched between item-level, holistic-loop, and phased-plan-drain modes through API, CLI, and UI.
- Entering either non-default initiative mode requires explicit cancellation confirmation if member items have active item-level executions.
- Holistic-loop phases run end-to-end:
  - `investigate` produces/updates `modes/holistic-loop/findings.md`
  - `plan` produces/updates `modes/holistic-loop/initiative-plan.md`
  - `execute` edits the repo according to `modes/holistic-loop/initiative-plan.md`, journals an execute round, and marks member items complete only through a run-id-validated API
  - `review` produces an acceptance verdict against `AcceptanceCriteria`
- Phased-plan-drain phases run end-to-end:
  - `prepare_plan` creates or updates `modes/phased-plan-drain/phased-plan.md`
  - `execute_next` spawns an agent with the plan plus prior final handoffs and asks it to complete the earliest contiguous phase(s) it can fully finish
  - `classify_progress` records whether the plan should continue, is blocked, needs replanning, or is complete
  - `review` produces an acceptance verdict against the plan, handoffs, code state, and `AcceptanceCriteria`
- Mode rounds are durable under `modes/<mode>/rounds/round-NNN.json`.
- Single-agent-per-initiative is enforced across feedback, decision review, holistic-loop phases, and phased-plan-drain phases.
- Agent activity records show operating mode, phase, run strategy, and round metadata.
- Agent activity records and round envelopes show which AgentManager profile key was used for each phase.
- Event log and stats expose mode usage and outcomes.
- The UI exposes a Mode chip, mode-switch flow, acceptance criteria editor, mode-aware Initiative Workspace, and mode metrics.
- Docs explain all shipped modes and the extension path for additional modes.

## 7. Architecture and Responsibility Boundaries

### Core Owns

Swarm Manager core owns concerns every mode uses:

- initiative metadata and membership
- mode selection and mode switching
- lock coordination
- activity records
- event logging
- stats aggregation primitives
- API/CLI routing
- common agent spawn/poll/cancel plumbing
- AgentManager profile reconciliation and profile-ref resolution plumbing
- artifact persistence primitives
- graph/current-state context loading
- backlog mutation and proposal application APIs
- UI service infrastructure and route composition

### Modes Own

Each mode definition owns:

- unit of execution and validation
- scope policy
- supported phases and transitions
- run strategy
- artifact names and schemas
- prompt skills and prompt variables
- AgentManager profile policy per phase
- readiness dimensions
- review semantics
- completion semantics
- backlog/audit reconciliation rules
- mode-specific metrics interpretation
- how member backlog items are interpreted

For `item-level`, member backlog items are execution units. For `holistic-loop` and `phased-plan-drain`, member backlog items are tracking, scope, dependency, and audit markers; the initiative is the execution/validation unit.

### Package Boundary

Add a new backend package:

```text
api/internal/operatingmode/
```

Prefer `operatingmode` over `initiativemode` because the framework is meant to model methodologies, not only initiative-scoped phases. Both non-default modes in this plan are initiative-scoped, but the abstraction should not encode initiative as the only possible scope forever.

This package should own:

- mode definitions
- scope policy types
- phase graph definitions
- run strategy definitions
- mode transition validation
- artifact policy helpers
- round envelope schemas
- backlog reconciliation policy interfaces
- AgentManager profile policy types
- metrics event payload definitions or helpers
- prompt catalog integration helpers

It should depend on narrow interfaces for initiatives, backlog, agent activity, AgentManager profile resolution, prompt rendering, graph/current state, execution cancellation, event emission, stats-relevant event logging, and locks.

`path:api/internal/initiatives/` remains CRUD + rollup + membership.

`path:api/internal/feedback/` remains feedback rounds/proposals.

`path:api/internal/initiativereview/` remains decision-oriented terminal review.

`path:api/internal/execution/` remains backlog-item execution.

Shared code can live in `operatingmode` or a smaller helper package only when multiple callers truly need the same primitive.

## 8. Operating Mode Framework Contract

Implement a framework with concepts like these. Exact names may vary, but the responsibility split must hold.

```go
type Mode string
type ScopeKind string
type Phase string
type RunStrategyKind string

const (
    ModeItemLevel        Mode = "item-level"
    ModeHolisticLoop     Mode = "holistic-loop"
    ModePhasedPlanDrain  Mode = "phased-plan-drain"
)

const (
    ScopeBacklogItem ScopeKind = "backlog_item"
    ScopeInitiative  ScopeKind = "initiative"
)

const (
    RunStrategySinglePhaseRun     RunStrategyKind = "single_phase_run"
    RunStrategySequentialHandoff  RunStrategyKind = "sequential_handoff"
    RunStrategyOperatorGatedLoop   RunStrategyKind = "operator_gated_loop"
)

type Definition struct {
    Mode             Mode
    Label            string
    Scope            ScopePolicy
    PhaseGraph       PhaseGraph
    RunStrategy      RunStrategyPolicy
    ArtifactPolicy   ArtifactPolicy
    PromptPolicy     PromptPolicy
    ProfilePolicy    ProfilePolicy
    BacklogSync      BacklogSyncPolicy
    MetricsPolicy    MetricsPolicy
    LockPolicy       LockPolicy
    UI               UIPolicy
}

type PhaseDefinition struct {
    Phase            Phase
    ActivityPurpose  agentactivity.Purpose
    LockPurpose      string
    CatalogID        string
    SkillID          string
    ProfileKey       string
    WritesRepo       bool
    OutputArtifacts  []ArtifactDefinition
    RequiresCriteria bool
}
```

Minimum required behavior:

- A static registry validates mode values and returns definitions.
- A definition includes scope, phase graph, run strategy, artifact policy, profile policy, backlog sync policy, metrics policy, prompt policy, lock policy, and UI policy.
- Phase start uses `(mode, phase)` to resolve skill, AgentManager profile key, lock purpose, activity purpose, artifact policy, metrics events, and run strategy behavior.
- Unknown modes/phases fail closed.
- `item-level` can be registered as a bridge over existing backlog behavior.
- `holistic-loop` and `phased-plan-drain` are not hardcoded in generic services except as registered definitions.
- Both non-default modes use the same lock, activity, profile selection, event, artifact, prompt, stats, and route primitives.

Do not build dynamic plugin loading. Static Go registration is enough. The important part is that decisions have one home.

## 9. AgentManager Profile Policy Contract

Operating modes must choose AgentManager profiles through explicit phase policy, not by relying on one global default profile.

Profile source files should be scenario-owned JSON files under:

```text
scenarios/swarm-manager/.vrooli/agent-profiles/
```

Register every source in:

```text
scenarios/swarm-manager/.vrooli/service.json
```

Suggested initial profile keys:

- `swarm-manager/default` - existing compatibility/default profile for item-level and any caller that does not yet provide a specific key.
- `swarm-manager/deep-work` - high-capability coding/reasoning profile for broad investigation, planning, and repo-editing phases.
- `swarm-manager/analysis` - lower-cost read-mostly profile for classification, reconciliation, and acceptance-review phases where full implementation capability is not required.

Exact runner/model choices belong in the profile JSON files, not in `operatingmode` code. The operating-mode registry should reference stable `profileKey` values only.

Add policy types similar to:

```go
type ProfilePolicy struct {
    DefaultProfileKey string
    PhaseProfiles     map[Phase]string
}
```

Minimum required behavior:

- Reconcile all declared Swarm Manager profile sources at startup through AgentManager's existing scenario profile reconciliation endpoint.
- Fail startup if any declared profile source fails validation or if a profile key referenced by an operating-mode definition is not returned by reconciliation.
- Add optional `ProfileKey` or `ProfileRef` fields to `BacklogSpawnRequest`, `ResearchSpawnRequest`, and `InitiativeSpawnRequest`.
- Route run creation through `profileRefFor(explicitKey)` instead of direct `defaultProfileRef()` calls.
- Preserve `swarm-manager/default` fallback for existing item-level call sites that do not provide a key.
- Operating-mode phase starts must provide the profile key selected by the mode definition.
- Agent activity metadata, operating-mode events, round envelopes, and workspace state must include `agent_profile_key`.
- Do not use inline AgentManager profile defaults for mode phases. Profile configuration should remain in scenario-owned profile source files so operators and future agents can inspect and adjust cost/capability policy in one place.

Recommended initial phase mapping:

- `holistic-loop`:
  - `investigate` -> `swarm-manager/deep-work`
  - `plan` -> `swarm-manager/deep-work`
  - `execute` -> `swarm-manager/deep-work`
  - `review` -> `swarm-manager/analysis`
- `phased-plan-drain`:
  - `prepare_plan` -> `swarm-manager/deep-work`
  - `execute_next` -> `swarm-manager/deep-work`
  - `classify_progress` -> `swarm-manager/analysis`
  - `review` -> `swarm-manager/analysis`

This mapping is intentionally conservative: broad repo-editing phases start with the stronger profile, while read-mostly classifier/review phases exercise the cost-efficiency seam from the first implementation. Future tuning should be profile-file changes or registry mapping changes, not ad hoc spawn overrides.

## 10. Metrics and Event Contract

Operating mode usage must be event-log observable.

Add event types similar to:

```go
EventInitiativeModeChanged       EventType = "initiative.mode_changed"
EventOperatingModePhaseStarted   EventType = "operating_mode.phase_started"
EventOperatingModePhaseCompleted EventType = "operating_mode.phase_completed"
EventOperatingModePhaseFailed    EventType = "operating_mode.phase_failed"
EventOperatingModePhaseCanceled  EventType = "operating_mode.phase_canceled"
EventOperatingModeReplanNeeded   EventType = "operating_mode.replan_needed"
EventOperatingModeBacklogSynced  EventType = "operating_mode.backlog_synced"
```

Payloads should include:

- `mode`
- `scope_kind`
- `scope_id`
- `initiative_name` when applicable
- `phase`
- `run_strategy`
- `agent_profile_key`
- `round_number`
- `run_id`
- `duration_seconds`
- `status`
- `verdict`
- `replan_needed`
- `backlog_items_completed`
- `backlog_items_created`
- `backlog_items_updated`
- `artifact_paths`

Stats should expose a new section:

```go
type ModeStats struct {
    UsageByMode              map[string]int
    PhaseRunsByMode          map[string]map[string]int
    CompletedByMode          map[string]int
    FailedByMode             map[string]int
    CanceledByMode           map[string]int
    ReplanRateByMode         map[string]KindRate
    AcceptanceRateByMode     map[string]KindRate
    AvgPhaseDurationSeconds  map[string]map[string]float64
    AvgRunsPerCompletedScope map[string]float64
    BacklogSyncByMode        map[string]BacklogSyncStats
    UsageByProfile           map[string]int
    PhaseRunsByProfile       map[string]map[string]int
}
```

UI stats should add a `Modes` tab or extracted section. Do not bury mode metrics inside the existing Agent tab; they answer different questions.

Counting rules:

- `item-level` usage can be derived from execution events only after those events carry operating-mode metadata, or from a migration/backfill that maps existing backlog executions to `item-level`.
- `holistic-loop` phase starts/completions must emit operating-mode events directly.
- Replan rate denominator is completed execute phases with a parseable `replan_needed` signal.
- Acceptance rate denominator is completed review phases with a verdict.
- Backlog sync counts only API-mediated item mutations linked to a mode run/round.
- Profile usage counts come from the selected `agent_profile_key` in phase lifecycle events, not from live AgentManager profile state.

## 11. Backlog and Audit Reconciliation Contract

Backlog items are the project-management ledger, not merely execution inputs.

For every non-item-level mode, define a `BacklogSyncPolicy`:

```go
type BacklogSyncCapability string

const (
    BacklogSyncReadOnly          BacklogSyncCapability = "read_only"
    BacklogSyncProposeMutations  BacklogSyncCapability = "propose_mutations"
    BacklogSyncMarkComplete      BacklogSyncCapability = "mark_complete"
    BacklogSyncCreateFollowups   BacklogSyncCapability = "create_followups"
    BacklogSyncUpdateScope       BacklogSyncCapability = "update_scope"
)

type BacklogSyncPolicy struct {
    Capabilities       []BacklogSyncCapability
    RequiresRunID      bool
    RequiresMembership bool
    EventSource        string
}
```

For `holistic-loop`:

- Agents may read member item specs, plans, conclusions, validation reports, and histories as context.
- Agents may not directly edit backlog specs.
- Execute agents may mark member items complete only through the run-id-validated endpoint.
- Review/replan may propose follow-up items or item updates through existing proposal/application primitives or a new audited reconciliation endpoint.
- Every item mutation caused by a non-default initiative mode must include mode, phase, round, run ID, and source in event metadata.
- Switching back to item-level preserves initiative artifacts and round history.

For `phased-plan-drain`:

- Prior final handoffs are first-class audit inputs.
- A classifier/reconciler decides whether phases are complete, partial, blocked, or require another run.
- The reconciler creates or updates backlog items as the paper trail, even though execution was driven by plan phases rather than item runs.
- Execute agents may mark member items complete only through the same run-id-validated endpoint used by holistic-loop mode.
- Each progress classification emits a backlog-sync summary event even when no backlog mutations are applied.

## 12. Implementation Strategy

### Phase 0 - Reconnaissance and Seam Notes

Before code edits:

1. Run the required skill-read command from this plan.
2. Inspect the files listed in Required Reading.
3. Update `path:scenarios/swarm-manager/docs/internal/SEAMS.md` with an "Operating Mode Boundary" section before implementing the package.
4. Record current decision points:
   - mode selection
   - phase transition
   - run strategy
   - prompt resolution
   - lock purpose
   - backlog reconciliation
   - metrics emission

Contract:

- Implementation starts from documented boundaries, not ad hoc package growth.

### Phase 1 - Initiative Model and Storage

Files:

- `path:api/internal/initiatives/model.go`
- `path:api/internal/initiatives/service.go`
- `path:api/internal/initiatives/store.go`
- `path:api/internal/initiatives/*_test.go`
- `path:ui/src/types/initiative.ts`
- proto/type mapping files if initiative types are proto-backed in this repo version

Tasks:

1. Add `Mode string` and `AcceptanceCriteria []string` to `Initiative`.
2. Add mode constants in a cycle-safe location. Prefer `operatingmode` constants only if initiatives can import them without dependency issues; otherwise mirror string constants with tests.
3. Add normalization so missing mode loads as `item-level`.
4. Add `ValidateMode`.
5. Extend create/update requests for acceptance criteria.
6. Reject mode changes on archived initiatives.
7. Keep mode changes out of generic PATCH; use a dedicated mode-switch endpoint.
8. Tests for normalization, validation, marshal/unmarshal, and store round-trip.

Contract:

- Missing mode on disk is accepted and normalized to `item-level`.
- Invalid mode is rejected.
- Default mode is visible to callers after load.

### Phase 2 - Operating Mode Registry and Decision Home

Files:

- `path:api/internal/operatingmode/definition.go`
- `path:api/internal/operatingmode/registry.go`
- `path:api/internal/operatingmode/scope.go`
- `path:api/internal/operatingmode/phase_graph.go`
- `path:api/internal/operatingmode/run_strategy.go`
- `path:api/internal/operatingmode/profile_policy.go`
- `path:api/internal/operatingmode/backlog_sync.go`
- `path:api/internal/operatingmode/metrics.go`
- `path:api/internal/operatingmode/*_test.go`
- `path:api/internal/promptcatalog/catalog.go`
- `path:api/internal/promptcatalog/catalog_test.go`
- `path:api/internal/agentactivity/types.go`
- `path:api/internal/agentactivity/service_test.go`

Tasks:

1. Add the `operatingmode` package.
2. Register `item-level`, `holistic-loop`, and `phased-plan-drain`.
3. Define holistic-loop scope, phase graph, run strategy, artifact policy, prompt policy, profile policy, backlog sync policy, metrics policy, lock policy, and UI policy.
4. Define phased-plan-drain scope, phase graph, run strategy, artifact policy, prompt policy, profile policy, backlog sync policy, metrics policy, lock policy, and UI policy.
5. Add activity purposes:
   - `holistic_loop_investigate`
   - `holistic_loop_plan`
   - `holistic_loop_execute`
   - `holistic_loop_review`
   - `phased_plan_prepare`
   - `phased_plan_execute_next`
   - `phased_plan_classify_progress`
   - `phased_plan_review`
6. Replace or extend `promptcatalog.ResolveInitiativeSkill(purpose)` with a resolver that supports mode+phase while preserving existing feedback/review lookups.
7. Add tests proving every registered non-default phase resolves to catalog entries, profile keys, activity purposes, lock purposes, run strategy metadata, and metrics metadata.

Contract:

- Mode/phase/run-strategy decisions live in one package.
- Profile selection decisions for operating-mode phases live in the mode definition, not in handlers or spawn call sites.
- Prompt catalog does not grow an unbounded hardcoded switch.
- Unknown mode/phase combinations return typed validation errors.
- Tests fail if either shipped non-default mode requires bespoke lock, activity, event, artifact, prompt, stats, or route primitives.

### Phase 2A - AgentManager Profile Sources and Spawn Ref Selection

Files:

- `path:scenarios/swarm-manager/.vrooli/service.json`
- `path:scenarios/swarm-manager/.vrooli/agent-profiles/default.json`
- `path:scenarios/swarm-manager/.vrooli/agent-profiles/deep-work.json`
- `path:scenarios/swarm-manager/.vrooli/agent-profiles/analysis.json`
- `path:api/internal/agentmanager/service.go`
- `path:api/internal/agentmanager/profile.go`
- `path:api/internal/agentmanager/*_test.go`
- `path:api/internal/agentactivity/service.go`
- `path:api/internal/agentactivity/types.go`
- `path:api/internal/agentactivity/*_test.go`

Tasks:

1. Add profile source files for `swarm-manager/deep-work` and `swarm-manager/analysis`.
2. Register the new files in `dependencies.scenarios.agent-manager.config.profiles.sources`.
3. Keep `swarm-manager/default` as the fallback profile for existing callers.
4. Extend `AgentService.Initialize` to retain the set of reconciled profile keys, not only the default profile ID.
5. Add `ProfileKey` or `ProfileRef` fields to `ResearchSpawnRequest`, `BacklogSpawnRequest`, and `InitiativeSpawnRequest`.
6. Replace direct `defaultProfileRef()` use in run creation with `profileRefFor(explicitKey)`.
7. Validate that explicit profile keys are scenario-owned (`swarm-manager/...`) and were returned by reconciliation.
8. Add selected profile key to agentactivity metadata for tracked spawns.
9. Add tests for:
   - multiple profile sources declared in service config
   - spawn with no explicit key uses `swarm-manager/default`
   - spawn with explicit key uses that profile ref
   - unknown explicit key fails before run creation
   - activity metadata records `agent_profile_key`

Contract:

- Profile configuration lives in `.vrooli/agent-profiles/*.json`.
- Spawn call sites select by stable profile key only.
- No operating-mode phase sends inline profile defaults.
- Profile selection is auditable in activity records and round/event metadata.

### Phase 3 - Event Log and Stats Foundation

Files:

- `path:api/internal/eventlog/types.go`
- `path:api/internal/eventlog/emitter.go`
- `path:api/internal/eventlog/*_test.go`
- `path:api/internal/stats/types.go`
- `path:api/internal/stats/engine.go`
- `path:api/internal/stats/metrics.go`
- `path:api/internal/stats/*_test.go`
- `path:ui/src/types/stats.ts`
- `path:ui/src/surfaces/graph/components/StatsPanel.tsx`
- stats component tests

Tasks:

1. Add operating-mode event types and typed payloads.
2. Add emitter methods for mode changes, phase lifecycle, replan-needed, and backlog sync.
3. Extend stats aggregate state for mode counters.
4. Add `ModeStats` to `StatsResponse`.
5. Add UI stats types.
6. Add a `Modes` stats tab or extracted mode section, including profile usage by mode/phase.
7. Add tests for event aggregation and UI rendering.

Contract:

- Operating mode usage is observable even when no UI is open.
- Stats derive from durable event log, not from scanning transient files.
- Existing stats remain backwards compatible for clients that ignore the new `modes` field.

### Phase 4 - Lock and Activity Unification

Files:

- `path:api/internal/initiativelock/lock.go`
- `path:api/internal/feedback/service.go`
- `path:api/internal/initiativereview/service.go`
- `path:api/internal/operatingmode/service.go`
- tests for all three callers

Tasks:

1. Keep `.feedback-lock` unless a migration is deliberately chosen.
2. Document that it is the initiative agent lock, not just feedback.
3. Add lock purpose constants:
   - `feedback`
   - `feedback_continue`
   - `initiative_review`
   - `holistic_loop_investigate`
   - `holistic_loop_plan`
   - `holistic_loop_execute`
   - `holistic_loop_review`
4. Ensure feedback, decision review, and operating-mode phases use the same lock API.
5. Add tests proving different holder purposes conflict on the same initiative.
6. Ensure holder metadata is enough for UI override/cancellation prompts.

Contract:

- Only one active initiative-scoped agent can hold the initiative lock.
- Purpose is diagnostic/routing metadata, not a separate lock namespace.

### Phase 5 - Shared Operating Mode Runner

Files:

- `path:api/internal/operatingmode/service.go`
- `path:api/internal/operatingmode/artifacts.go`
- `path:api/internal/operatingmode/rounds.go`
- `path:api/internal/operatingmode/context.go`
- `path:api/internal/operatingmode/poller.go`
- `path:api/internal/operatingmode/backlog_reconcile.go`
- `path:api/internal/operatingmode/handler.go`
- `path:api/routes_operating_mode.go` or equivalent server wiring

Tasks:

1. Implement `StartPhase(ctx, StartPhaseRequest)`.
2. Load scope and validate mode supports phase.
3. Validate transition/state rules.
4. Acquire lock according to lock policy.
5. Build prompt context according to prompt policy:
   - initiative metadata
   - acceptance criteria
   - member item summaries
   - member item deliverables where useful
   - current graph/materialized state
   - prior operating-mode rounds
   - relevant current artifacts
   - prior final handoffs when the run strategy requires sequential handoff context
6. Resolve skill through mode+phase definition.
7. Resolve AgentManager profile key through mode+phase definition.
8. Spawn through `agentactivity.Service.SpawnInitiative` with the selected profile key.
9. Emit phase-started event with `agent_profile_key`.
10. Persist provisional round with agent-running status and `agent_profile_key`.
11. Poll run state and persist terminal output.
12. Extract phase artifacts from agent output and write atomically.
13. Run backlog sync policy for any allowed reconciliation.
14. Emit phase-completed/failed/canceled and backlog-sync events.
15. Release lock on terminal state.

Contract:

- Phase endpoints are thin wrappers over the shared runner.
- The runner does not contain phase-specific prompt prose.
- Phase-specific parsing/validation is delegated by phase definition/policy.
- Run strategy is explicit: holistic-loop uses one agent run per phase, while phased-plan-drain uses sequential handoff.
- AgentManager profile selection is explicit and comes from the registered phase definition.

### Phase 6 - Artifact, Round, and Handoff Model

Files:

- `path:api/internal/operatingmode/artifacts.go`
- `path:api/internal/operatingmode/rounds.go`
- `path:api/internal/operatingmode/readiness.go`
- `path:scenarios/swarm-manager/docs/internal/SEAMS.md`

Tasks:

1. Add mode-scoped artifact paths:
   - `<initiative>/modes/holistic-loop/findings.md`
   - `<initiative>/modes/holistic-loop/initiative-plan.md`
   - `<initiative>/modes/holistic-loop/rounds/round-NNN.json`
   - `<initiative>/modes/phased-plan-drain/phased-plan.md`
   - `<initiative>/modes/phased-plan-drain/progress.json`
   - `<initiative>/modes/phased-plan-drain/rounds/round-NNN.json`
2. Define a common round envelope:
   - `round`
   - `mode`
   - `scope_kind`
   - `scope_id`
   - `phase`
   - `run_strategy`
   - `agent_profile_key`
   - `generated_at`
   - `run_id`
   - `status`
   - `readiness`
   - `items`
   - `artifact_updates`
   - `handoffs`
   - phase-specific payload
3. Add readiness policy for holistic-loop:
   - `problem_clarity`
   - `scope_defined`
   - `approach_solid`
   - `testable`
   - `risk_awareness`
   - `coupling_understood`
   - `system_acceptance_defined`
4. Add progress classification schema for phased-plan-drain:
   - `continue`
   - `blocked`
   - `replan`
   - `complete`
5. Use boost divisor `N=2` for holistic-loop readiness.
6. Add unit tests for mode-scoped path resolution, round numbering, atomic writes, malformed JSON rejection, handoff preservation, profile key preservation, progress classification parsing, and readiness scoring.

Contract:

- Round files are append-only evidence.
- Current-state markdown files are mutable latest-state artifacts.
- Handoffs are preserved in round JSON so sequential modes have a first-class audit primitive.

### Phase 7 - Mode Skills

Files:

- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-holistic-loop-investigate/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-holistic-loop-plan/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-holistic-loop-execute/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-holistic-loop-review/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-phased-plan-prepare/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-phased-plan-execute-next/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-phased-plan-classify-progress/SKILL.md`
- `path:scenarios/prompt-manager/store/skills/packs/core/swarm-manager-phased-plan-review/SKILL.md`
- prompt catalog tests/simulations

Holistic-loop skill contracts:

1. `investigate`
   - read-only on code
   - writes `modes/holistic-loop/findings.md` content
   - emits `phase=investigate` round envelope
   - never edits backlog items or repo files
2. `plan`
   - reads latest findings, graph, item deliverables, and acceptance criteria
   - writes `modes/holistic-loop/initiative-plan.md`
   - emits seven-dimension readiness
3. `execute`
   - edits repo files according to `modes/holistic-loop/initiative-plan.md`
   - emits completed milestones and `replan_needed`
   - marks member items complete only through the documented API
   - never writes per-item `plan.md` or `spec.json`
4. `execution-review`
   - read-only on code unless explicitly gathering external review artifacts through approved tools
   - evaluates `AcceptanceCriteria`
   - emits verdict: `accept`, `request_replan`, or `request_changes`
   - never completes the initiative automatically

Tests:

- `prompt-manager skill read`/simulate for each skill.
- Token budget checks for typical initiative context.
- JSON envelope shape validation.

Phased-plan-drain skill contracts:

1. `prepare`
   - reads initiative context, existing member item deliverables, and optional uploaded/operator plan material
   - writes `modes/phased-plan-drain/phased-plan.md`
   - emits a round envelope with a phase inventory and current completion assumptions
   - never edits repo files
2. `execute-next`
   - reads `phased-plan.md`, `progress.json`, prior handoffs, and current code state
   - edits repo files for the earliest contiguous phase(s) it can complete fully
   - emits a final handoff summarizing completed phases, changed files, tests, blockers, and recommended next step
   - never starts a phase it cannot complete professionally
   - never writes backlog specs directly
3. `classify-progress`
   - read-only on code and artifacts
   - classifies progress as `continue`, `blocked`, `replan`, or `complete`
   - updates `progress.json`
   - proposes backlog reconciliation actions or records no-op reconciliation with rationale
4. `review`
   - read-only on code unless explicitly gathering external review artifacts through approved tools
   - evaluates `AcceptanceCriteria`, `phased-plan.md`, progress state, handoffs, and code state
   - emits verdict: `accept`, `request_replan`, or `request_changes`
   - never completes the initiative automatically

### Phase 8 - Phase APIs

Files:

- `path:api/internal/operatingmode/handler.go`
- `path:api/routes_operating_mode.go`
- `path:ui/src/lib/api-endpoints.ts`
- `path:docs/reference/api-endpoints.md`

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
  - `POST /api/v1/initiatives/{name}/prepare-plan`
  - `POST /api/v1/initiatives/{name}/execute-next`
  - `POST /api/v1/initiatives/{name}/classify-progress`
- `POST /api/v1/initiatives/{name}/items/{item-ref}/complete`
- `POST /api/v1/initiatives/{name}/complete`

Preferred internal routing:

- Stable aliases call the generic phase handler.
- The generic phase handler calls `operatingmode.Service.StartPhase`.

Status contracts:

- `400` invalid mode/phase/body.
- `404` initiative not found.
- `409` mode/transition conflict or in-flight item executions without cancellation confirmation.
- `423` initiative lock held.
- `202` phase run accepted/spawned.

### Phase 9 - Mode Switch Protocol

Tasks:

1. Implement `POST /api/v1/initiatives/{name}/mode`.
2. Body:

   ```json
   {
     "mode": "holistic-loop",
     "cancel_in_flight": true
   }
   ```

3. Entering `holistic-loop` or `phased-plan-drain`:
   - enumerate active member item-level executions/activities
   - if any exist and `cancel_in_flight=false`, return `409` with blockers
   - if `cancel_in_flight=true`, cancel them through execution/activity services
   - set mode
   - emit `initiative.mode_changed`
4. Returning to item-level:
   - reject if any non-default mode phase is active
   - preserve holistic-loop and phased-plan-drain artifacts as history
   - set mode to `item-level`
   - emit `initiative.mode_changed`
5. Reject mode changes on archived initiatives.

Contract:

- Mode switch is not a generic metadata PATCH.
- No silent cancellation.
- No artifact deletion on drain-back.

### Phase 10 - Execute Phase and Item Completion API

Tasks:

1. Add run-id-validated item completion endpoint.
2. The active execute run ID must match the caller-provided run ID or authenticated agent context. This applies to holistic-loop `execute` and phased-plan-drain `execute_next`.
3. Mark member item complete only if it belongs to the initiative.
4. Journal completion in the execute round.
5. Emit backlog-sync event with source mode/phase/round/run.
6. Surface `replan_needed=true` in holistic-loop workspace state.
7. Surface phased-plan-drain progress classification in workspace state.

Contract:

- Holistic-loop execute agent may update repo files.
- It may not mutate backlog specs directly.
- Item completion is audited and mediated by API.

### Phase 11 - CLI

Files:

- `path:scenarios/swarm-manager/cli` sources (`path:cli/app.go` or current command package)
- `path:docs/reference/cli-commands.md`

Commands:

```bash
swarm-manager initiative mode set <name> <item-level|holistic-loop|phased-plan-drain> [--cancel-in-flight]
swarm-manager initiative investigate <name>
swarm-manager initiative plan <name>
swarm-manager initiative execute <name>
swarm-manager initiative review <name>
swarm-manager initiative prepare-plan <name>
swarm-manager initiative execute-next <name>
swarm-manager initiative classify-progress <name>
swarm-manager initiative complete <name>
```

CLI commands should call the API rather than duplicating filesystem writes.

### Phase 12 - UI Service and Types

Files:

- `path:ui/src/types/initiative.ts`
- `path:ui/src/types/initiative-mode.ts` if useful
- `path:ui/src/services/initiative-service.ts`
- `path:ui/src/services/initiative-mode-service.ts`
- `path:ui/src/services/stats-service.ts`
- `path:ui/src/types/stats.ts`
- `path:ui/src/lib/api-endpoints.ts`
- service tests

Tasks:

1. Add `mode` and `acceptanceCriteria` to initiative normalization.
2. Add mode-switch and phase-start methods.
3. Add workspace fetch method.
4. Add type-safe models for phases, run strategies, rounds, artifacts, handoffs, progress classifications, review verdicts, and blockers.
5. Add mode stats and profile usage types.
6. Tests for snake_case to camelCase normalization.

### Phase 13 - Initiative Workspace UI

Files:

- `path:ui/src/pages/InitiativeDetailsPage.tsx`
- `path:ui/src/components/initiative/initiative-mode-chip.tsx`
- `path:ui/src/components/initiative/initiative-mode-switch-dialog.tsx`
- `path:ui/src/components/initiative/initiative-workspace.tsx`
- `path:ui/src/components/initiative/initiative-acceptance-criteria-editor.tsx`
- `path:ui/src/components/initiative/initiative-phase-controls.tsx`
- `path:ui/src/components/initiative/initiative-round-timeline.tsx`
- `path:ui/src/consts/selectors.ts`

Tasks:

1. Add a compact Mode chip in the initiative header.
2. Add a mode-switch action with in-flight blocker/cancellation confirmation.
3. Add an Initiative Workspace tab visible when `mode === "holistic-loop"` or `mode === "phased-plan-drain"`.
4. Workspace sections:
   - Findings (holistic-loop)
   - Initiative Plan (holistic-loop)
   - Phased Plan (phased-plan-drain)
   - Progress classification (phased-plan-drain)
   - Rounds and handoffs
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

### Phase 14 - Stats UI

Files:

- `path:ui/src/surfaces/graph/components/StatsPanel.tsx`
- extracted stats components if needed
- `path:ui/src/types/stats.ts`
- stats tests

Tasks:

1. Add `modes` to stats response types.
2. Add a `Modes` tab.
3. Show:
   - usage by mode
   - phase counts by mode
   - acceptance/replan rates with sample-size handling
   - average phase duration
   - backlog sync counts
   - AgentManager profile usage by mode/phase
4. Reuse existing insufficient-data patterns.
5. Keep copy precise: `item-level`, `holistic-loop`, and `phased-plan-drain` are operating modes, not `manual`/`yolo` run start policies.

### Phase 15 - Docs

Files:

- `path:scenarios/swarm-manager/docs/guides/holistic-loop-mode.md`
- `path:scenarios/swarm-manager/docs/guides/phased-plan-drain-mode.md`
- `path:scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `path:scenarios/swarm-manager/docs/reference/api-endpoints.md`
- `path:scenarios/swarm-manager/docs/reference/cli-commands.md`
- `path:scenarios/swarm-manager/docs/internal/SEAMS.md`
- `path:scenarios/swarm-manager/docs/manifest.json`

Tasks:

1. Update `EXECUTION-MODES.md` into an operating-mode concept doc covering shipped modes and the extension model.
2. Add operator guide for when and how to use holistic-loop mode.
3. Add operator guide for when and how to use phased-plan-drain mode.
4. Document the extension path for additional modes.
5. Document how operating-mode profile policies reference scenario-owned AgentManager profile source files.
6. Register new API and CLI contracts.
7. Register new docs in manifest.
8. Add `[CODE: ...]` references to new framework files once implemented.

### Phase 16 - Validation and Restart

Run targeted tests as each phase lands, then full scenario validation:

```bash
cd scenarios/swarm-manager/api && go test ./internal/initiatives/... ./internal/operatingmode/... ./internal/initiativelock/... ./internal/agentmanager/... ./internal/agentactivity/... ./internal/promptcatalog/... ./internal/eventlog/... ./internal/stats/... -timeout 300s
cd scenarios/swarm-manager/ui && npm test -- InitiativeDetailsPage initiative-service initiative-mode StatsPanel stats-service
cd scenarios/swarm-manager/ui && npm run typecheck
cd scenarios/swarm-manager && make test
vrooli scenario restart swarm-manager
vrooli scenario test swarm-manager
vrooli scenario test prompt-manager
vrooli scenario test agent-manager
```

Use longer timeouts where needed; full scenario tests can take several minutes.

## 13. Contract Decisions

1. **Operating mode is explicit metadata.** `mode` lives on the initiative and defaults to `item-level` on load.
2. **Mode switch is a lifecycle operation.** Use a dedicated endpoint/CLI; do not rely on generic PATCH for mode transitions.
3. **Status and mode are separate.** `status` says lifecycle/result; `mode` says execution machinery/methodology.
4. **Static registry.** No dynamic plugin system in this plan.
5. **Modes are more than phases.** A mode definition includes scope, phase graph, run strategy, artifacts, prompt policy, AgentManager profile policy, backlog sync, metrics, lock, and UI policy.
6. **Run strategy is first-class.** Holistic-loop uses one run per phase; phased-plan-drain uses sequential handoff.
7. **Backlog remains the ledger.** Non-item modes must reconcile work back to backlog/initiative entities through audited APIs or proposals.
8. **Phase runner is shared.** Endpoint aliases are allowed, but internals route through a common operating-mode service.
9. **Non-default mode artifacts are current-state files plus round evidence.** Markdown files hold latest state; round JSON preserves history and handoffs.
10. **Single initiative agent lock.** Feedback, decision review, holistic-loop phases, and phased-plan-drain phases all contend on the same lock.
11. **Member items remain scope/progress/audit markers in non-default initiative modes.** They are not independently executed while holistic-loop or phased-plan-drain mode is active.
12. **Execute agent marks item completion through API only.** No direct spec mutation.
13. **Acceptance review is distinct.** `swarm-manager-holistic-loop-review` and `swarm-manager-phased-plan-review` are not `swarm-manager-initiative-review`.
14. **Operator completes explicitly.** A review verdict of `accept` enables completion but does not auto-complete the initiative.
15. **Replan is a loop state, not a separate endpoint.** Execute can signal `replan_needed`; operator runs plan/investigate again.
16. **Profile selection is registered policy.** Mode phases choose scenario-owned AgentManager profiles by stable `profileKey`; no inline per-run profile defaults.
17. **Operating mode metrics come from events.** Do not derive core mode stats from transient UI state or ad hoc filesystem scans.
18. **Cost hint is UI-only.** No hard budget gate in this plan.

## 14. Testing Plan

### Backend Unit Tests

- Initiative model normalization and validation.
- Mode registry rejects unknown modes/phases/run strategies.
- `holistic-loop` and `phased-plan-drain` both register without bespoke primitives outside the operating-mode framework and thin adapters.
- Prompt catalog resolves every mode+phase.
- AgentManager profile source reconciliation supports every profile referenced by mode definitions.
- AgentManager spawn requests use explicit phase profile keys when provided and fall back to `swarm-manager/default` otherwise.
- Agent activity accepts new purposes and preserves metadata.
- Lock conflicts across different purposes.
- Event emitter writes mode event payloads.
- Stats engine aggregates mode events.
- Stats engine aggregates profile usage from operating-mode events.
- Phase runner validates mode/phase, lock contention, transition rules, and run strategy.
- Artifact writes are atomic and round numbering is stable.
- Handoff persistence works.
- Phased-plan-drain progress classification accepts only `continue`, `blocked`, `replan`, or `complete`.
- Readiness scoring for seven dimensions.
- Mode switch rejects in-flight items without `cancel_in_flight`.
- Mode switch cancels in-flight items with explicit confirmation.
- Item completion endpoint rejects wrong run IDs and non-member refs.
- Backlog sync emits auditable events with mode/phase/round/run metadata.

### Backend Integration / E2E

- Full mode switch -> investigate -> plan -> execute -> review -> complete flow against a temp filesystem and stub AgentManager.
- Execute with `replan_needed=true` surfaces replan state and stats.
- Full mode switch -> prepare_plan -> execute_next -> classify_progress -> execute_next -> classify_progress -> review -> complete flow against a temp filesystem and stub AgentManager.
- Phased-plan-drain `execute_next` receives prior handoffs in the next prompt context.
- Drain-back to item-level is blocked while a phase is active and succeeds after completion.
- Existing feedback and decision review still lock correctly.
- Existing item-level execution still produces current execution stats.
- Operating-mode phase runs use the profile key defined by the registered phase policy.

### UI Tests

- Initiative service normalizes `mode` and `acceptance_criteria`.
- Mode chip renders.
- Mode-switch dialog lists blockers and sends `cancel_in_flight`.
- Workspace tab is visible in holistic-loop and phased-plan-drain modes.
- Acceptance criteria editor updates via service.
- Phase controls call the correct service methods.
- Round timeline renders artifact updates, progress classifications, and handoffs.
- Cost-hint modal appears before execute and respects 24h acknowledgement.
- Stats panel renders mode and profile usage metrics with insufficient-data behavior.
- Existing feedback/review/files tabs still render.

### Skill Tests

- Each new skill renders through prompt-manager.
- Each new skill emits the expected JSON envelope in simulation.
- Prompt token budgets stay within documented limits.

### Docs / Cross-Scenario

- New docs registered in manifest.
- API/CLI reference includes new endpoints/commands.
- `EXECUTION-MODES.md` documents item-level, holistic-loop, phased-plan-drain, profile policy, and the extension model.
- `SEAMS.md` documents the Operating Mode Boundary and decision home.
- `make test` and `vrooli scenario test swarm-manager` pass.
- Adjacent `prompt-manager` and `agent-manager` scenarios pass baseline tests.

## 15. Rollout / Validation Checklist

- [ ] `Initiative.Mode` and `AcceptanceCriteria` exist and round-trip.
- [ ] `operatingmode` registry exists with `item-level`, `holistic-loop`, and `phased-plan-drain`.
- [ ] Mode definitions include scope, phase graph, run strategy, activity purpose, lock purpose, skill/catalog identity, profile policy, artifact policy, backlog sync policy, metrics policy, and UI policy.
- [ ] Swarm Manager declares and reconciles `default`, `deep-work`, and `analysis` AgentManager profile source files.
- [ ] AgentManager spawn requests accept explicit profile keys and fall back to `swarm-manager/default`.
- [ ] Prompt catalog resolves mode+phase skills.
- [ ] Agent activity supports holistic-loop purposes and metadata.
- [ ] Agent activity supports phased-plan-drain purposes and metadata.
- [ ] Shared lock represents feedback, decision review, holistic-loop phases, and phased-plan-drain phases cleanly.
- [ ] Operating-mode events are emitted.
- [ ] Stats response and UI expose mode metrics and AgentManager profile usage.
- [ ] Shared phase runner starts, tracks, polls, persists, emits events, syncs backlog, and releases locks.
- [ ] Holistic-loop artifacts persist under the mode-scoped initiative folder.
- [ ] Phased-plan-drain artifacts, progress, rounds, and handoffs persist under the mode-scoped initiative folder.
- [ ] Mode-switch endpoint enforces cancellation gate.
- [ ] Item completion endpoint enforces active execute run ID.
- [ ] UI has Mode chip, mode-switch dialog, Initiative Workspace, acceptance criteria editor, phase controls, round timeline, and cost hint.
- [ ] New skills are present and simulate successfully.
- [ ] Docs and manifest are updated.
- [ ] `make test` passes in `path:scenarios/swarm-manager`.
- [ ] `vrooli scenario restart swarm-manager` succeeds.
- [ ] `vrooli scenario test swarm-manager`, `prompt-manager`, and `agent-manager` pass.

## 16. Risks and Mitigations

| Risk | Likelihood | Impact | Mitigation |
|---|---:|---:|---|
| Framework becomes over-abstract and delays delivery | Medium | High | Keep registry static. Implement abstractions needed by item-level, holistic-loop, and phased-plan-drain only. |
| Implementation scatters `holistic-loop` checks | Medium | High | Require mode/run-strategy decisions to live in `operatingmode`; add registry and mode-boundary tests. |
| Implementation scatters `phased-plan-drain` checks | Medium | High | Require phased-plan-drain to use the same runner, lock, activity, events, artifacts, stats, and workspace seams as holistic-loop. |
| Run strategy is hand-waved and additional modes require rewrites | Medium | High | Make `RunStrategyPolicy` explicit and validate it through shipped holistic-loop and phased-plan-drain flows. |
| Backlog audit trail gets weaker in flexible modes | Medium | High | Add `BacklogSyncPolicy`, run-id validation, and backlog-sync events. |
| Phase runner duplicates feedback service behavior | Medium | Medium | Extract or adapt reusable patterns; do not copy large blocks without consolidating common concerns. |
| Event volume becomes noisy | Low | Medium | Emit one lifecycle event per phase state transition and one summarized backlog-sync event per reconciliation, not per internal step. |
| Stats mislead by mixing operating mode with manual/yolo execution mode | Medium | Medium | Name fields and UI copy explicitly; test that operating mode stats use operating-mode events. |
| Profile selection becomes scattered through spawn call sites | Medium | High | Require phase profile keys to live in `operatingmode` definitions and test every registered phase resolves a reconciled profile key. |
| Lower-cost profile is assigned to a phase that needs repo-editing strength | Medium | Medium | Start with conservative `deep-work` mapping for implementation phases; tune only classifier/review phases to `analysis` until evidence supports more changes. |
| Lock filename `.feedback-lock` confuses future agents | Medium | Low | Document in `initiativelock` comments and `SEAMS.md`; rename only if migration cost is justified. |
| Holistic-loop execute crashes after partial item completion | Medium | High | Journal execute rounds, persist handoffs, and require next execute/investigate to read code state plus round history. |
| Agent directly edits item specs | Medium | High | Skill hard rule plus API-only completion endpoint; tests assert completion endpoint contract. |
| UI page becomes too large | Medium | Medium | Extract workspace and stats components; keep route/panel components as composition roots. |
| Existing decision review semantics regress | Low | High | Keep acceptance review separate; add regression tests for `initiativereview`. |
| Prompt catalog grows confusing purpose names | Medium | Medium | Centralize names in mode definitions and activity constants; catalog tests cover each purpose. |

## 17. Non-Goals and Prohibited Patterns

- No hidden "only two modes forever" assumptions.
- No broad `if mode == "holistic-loop"` branching across handlers, UI pages, and services.
- No mode abstraction that only models phase names while ignoring run strategy, audit policy, and metrics policy.
- No direct spec writes from holistic-loop agents.
- No direct spec writes from phased-plan-drain agents.
- No silent cancellation of item-level executions.
- No separate initiative lock implementation.
- No reuse of decision-oriented initiative review for acceptance review.
- No dynamic plugin architecture.
- No runtime-created per-mode AgentManager profiles.
- No inline AgentManager profile defaults in operating-mode phase spawns.
- No profile selection hardcoded in HTTP handlers, UI components, or individual spawn call sites.
- No cross-initiative execution.
- No per-item workshop schema changes.
- No deleting non-default mode artifacts when switching back to item-level.
- No mode metrics derived only from UI state.
- No unregistered docs.
- No UI selectors outside `selectors.ts`.

## 18. Definition of Done

This plan is complete when:

1. The repo contains a tested `operatingmode` framework with a static registry for `item-level`, `holistic-loop`, and `phased-plan-drain`.
2. The framework models scope, phase graph, run strategy, artifacts, prompt policy, AgentManager profile policy, backlog sync policy, metrics policy, lock policy, and UI policy.
3. Holistic-loop and phased-plan-drain both run through the same core operating-mode primitives.
4. Initiative metadata includes `Mode` and `AcceptanceCriteria`, with load normalization and validation.
5. Mode switching is implemented as a lifecycle operation with explicit in-flight cancellation behavior.
6. Holistic-loop phases run through a shared operating-mode runner and produce durable artifacts/rounds.
7. Phased-plan-drain phases run through a shared operating-mode runner and produce durable artifacts/rounds/handoffs/progress state.
8. New prompt-manager skills exist and are catalog-resolvable by mode+phase.
9. Agent activity and locks represent holistic-loop and phased-plan-drain phases cleanly.
10. Swarm Manager declares multiple scenario-owned AgentManager profiles and each operating-mode phase records its selected `agent_profile_key`.
11. Event log and stats expose operating-mode usage, outcomes, replan signals, durations, progress classifications, AgentManager profile usage, and backlog reconciliation.
12. Holistic-loop execute and phased-plan-drain execute-next can mark member items complete only through the run-id-validated API.
13. Non-default mode backlog reconciliation is audited and tied to mode/phase/round/run metadata.
14. Holistic-loop and phased-plan-drain acceptance reviews are separate from decision-oriented initiative review.
15. The UI exposes mode-aware controls, workspace components, and mode/profile metrics without bloating `InitiativeDetailsPage.tsx` or `StatsPanel.tsx`.
16. Docs, API reference, CLI reference, manifest, and `SEAMS.md` are updated.
17. All targeted backend, UI, skill, scenario, and adjacent-scenario tests pass.
18. A future agent can read this plan plus `SEAMS.md` and understand how to add another operating mode without reconstructing this conversation.

## 19. Recommended Execution Pattern for This Plan

This plan is large. Execute it with a phased handoff-chain approach:

1. The agent reads the full plan.
2. The agent implements the earliest contiguous phase(s) it can complete fully and professionally.
3. The agent does not start a phase it cannot finish.
4. The final handoff lists:
   - completed phases
   - changed files
   - tests run
   - tests not run and why
   - blockers
   - first incomplete phase
   - recommended next command/request
5. The next agent reads the plan plus all prior final handoffs and continues from the first incomplete phase.

This execution pattern is also the methodology being encoded as the shipped `phased-plan-drain` mode. Use it while implementing this plan to avoid a single over-broad agent run producing partial architecture.
