# Swarm Manager Operating Mode Hardening Plan

## 1. Purpose

Turn the operating-mode implementation from a broad functional scaffold into a clean, enforced Swarm Manager architecture.

The first implementation shipped the main surfaces: registered modes, operating-mode rounds, artifacts, phase spawns, AgentManager profile keys, API/CLI/UI entry points, event-log stats, and prompt skills. That is enough to begin testing, but the implementation still leaves several core decisions under-enforced:

- Any registered phase can be started when no round is active, regardless of phase graph or run strategy.
- Generic initiative metadata update can still change `mode`, bypassing the operating-mode lifecycle boundary.
- Operating-mode item completion emits a generic backlog status event and drops the mode/phase/round/run audit source.
- Prompt rendering failure falls back to a generic prompt, which is unsafe for broad initiative-level execution.
- Startup validates the default AgentManager profile, but not every profile referenced by registered operating-mode phases.

This is a greenfield hardening pass. Do not preserve compatibility shims, alternate legacy paths, or dead code. If an old path conflicts with the operating-mode architecture, remove it and update all callers to the single intended path.

## 1.1 Implementation Progress

Last updated: 2026-04-30

Completed in the first implementation pass:

- Phase 0 baseline contract tests for the phase-state and mode-switch boundaries.
- Phase 1 backend phase state machine extraction in `api/internal/operatingmode/state.go`.
- Backend phase start validation before round reservation, lock acquisition, prompt rendering, or AgentManager spawn.
- Workspace phase action state (`startable`, `reason`, `next`) exposed by the API and consumed by the UI.
- Phase 2 mode-switch boundary hardening for generic initiative create/update:
  - new initiatives always start in `item-level`
  - public initiative create/update request types no longer accept `mode`
  - lifecycle-only `SetModeLifecycle` is the mode mutation path used by operating-mode switching
  - archived initiatives reject lifecycle mode changes
  - switching out of non-default modes rejects while any mode round is active
- Phase 3 audit-safe backlog reconciliation:
  - `CompleteItems` now passes a typed `BacklogMutationSource` instead of a string source
  - operating-mode item completion emits `backlog.status_changed` metadata with entrypoint, initiative, mode, phase, round, run ID, requested-by, and item refs
  - `operating_mode.backlog_synced` metadata includes the same structured source and affected item refs for explicit mode-stat/audit consumption
  - proposal-based backlog sync carries mode, phase, and run ID through `proposals.Source` and `backlog.proposal_applied` payloads
  - route-level regression tests load event metadata and assert operating-mode source fields are durable
- Phase 4 fail-closed prompt resolution:
  - production fallback prompt prose was removed from operating-mode phase start
  - phase start validates the registry skill against a prompt catalog resolver before accepting prompt output
  - prompt-manager errors and empty rendered prompts mark the reserved round failed, release the lock, and prevent AgentManager spawn
  - all eight registered operating-mode prompt skills were read successfully with `prompt-manager skill read`
- Phase 5 startup profile policy validation:
  - the operating-mode registry now exposes all referenced AgentManager profile keys through `RequiredProfileKeys`
  - registry profile keys fail closed unless they are scenario-owned under the `swarm-manager/` namespace
  - API startup passes the registry-required keys into AgentManager reconciliation
  - AgentManager initialization fails when any required reconciled profile is missing, including `swarm-manager/deep-work` and `swarm-manager/analysis`
  - startup now treats AgentManager initialization failure as fatal when the integration is enabled
- Phase 6 mode stats correction:
  - replan rate is now counted from completed execute-phase payloads only
  - `operating_mode.replan_needed` remains a timeline event but no longer increments the replan numerator
  - operating-mode acceptance verdict matching is centralized and normalized
  - UI Modes-tab coverage asserts replan and acceptance rates render with sample sizes
- API, CLI, concept, and seam docs updated for those completed boundaries.

Validated commands:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode/... ./internal/initiatives/... ./internal/agentmanager/... ./internal/eventlog/... ./internal/stats/... -timeout 300s
go test ./internal/operatingmode/... ./internal/initiatives/... ./internal/agentmanager/... ./internal/eventlog/... ./internal/stats/... ./internal/proposals/... ./internal/promptcatalog/... -timeout 300s
go test ./... -run '^$' -timeout 300s
```

```bash
cd scenarios/swarm-manager/ui
npm test -- InitiativeDetailsPage initiative-mode operating-mode-panel StatsPanel stats-service
npm test -- StatsPanel
```

```bash
cd scenarios/swarm-manager/cli
go test ./... -timeout 300s
```

Known validation note:

- `cd scenarios/swarm-manager/api && go test ./... -timeout 300s` still fails in existing initiative-review E2E tests because the test app attempts to spawn AgentManager without `dependencies.scenarios.agent-manager` configured. Compile-only `go test ./... -run '^$'` passes.

Recommended next phase:

- Resume at Phase 7 - Backend Responsibility Refactor.

## 2. Required Reading

Run before implementation:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health intent-clarification boundary-of-responsibility-enforcement decision-boundary-extraction seam-discovery-and-enforcement utils-unification react-coherence
```

Also read:

- `docs/plans/swarm-manager-initiative-operating-mode-implementation.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/concepts/ARCHITECTURE.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/api/internal/operatingmode/registry.go`
- `scenarios/swarm-manager/api/internal/operatingmode/service.go`
- `scenarios/swarm-manager/api/internal/operatingmode/handler.go`
- `scenarios/swarm-manager/api/routes_operating_mode.go`
- `scenarios/swarm-manager/api/internal/initiatives/service.go`
- `scenarios/swarm-manager/api/internal/agentmanager/service.go`
- `scenarios/swarm-manager/api/internal/agentmanager/profile.go`
- `scenarios/swarm-manager/api/internal/eventlog/types.go`
- `scenarios/swarm-manager/api/internal/stats/engine.go`
- `scenarios/swarm-manager/api/internal/stats/metrics.go`
- `scenarios/swarm-manager/ui/src/components/initiative/operating-mode-panel.tsx`
- `scenarios/swarm-manager/ui/src/services/initiative-mode-service.ts`
- `scenarios/swarm-manager/ui/src/surfaces/graph/components/StatsPanel.tsx`

## 3. Problem Statement

The operating-mode framework currently exists, but the most important architectural constraints are still partly encoded as convention.

The registry describes methodology, but `StartPhase` does not yet enforce phase graph state, transition validity, run-strategy-specific prerequisites, or terminal review semantics. The initiative service persists `mode`, but generic update still exposes mode changes. Backlog reconciliation has run-id checks inside the operating-mode service, but the final backlog status event loses the source context that makes the mutation auditable. Agent profile policy is visible in the registry, but startup does not prove all referenced profiles are reconciled before the app serves traffic. Prompt skills are registered, but prompt-manager failure silently downgrades to a generic fallback prompt.

This is dangerous because Swarm Manager is the primary project-management app for the repository. Operating modes define who owns the work unit, when repo-writing agents may run, how handoffs accumulate, how backlog paper-trail mutations happen, and what metrics mean. These decisions need to be enforced in code at the same boundary where they are declared.

## 4. Scope

### In Scope

- Enforce operating-mode phase transitions and run-strategy rules in backend and UI.
- Make mode switching the only way to change initiative mode.
- Remove generic PATCH mode mutation from public initiative update flow.
- Add a dedicated internal mode-update method used only by the operating-mode switch service.
- Reject switching out of non-default modes while a mode round is active.
- Preserve artifacts and rounds when switching modes, but do not keep alternate legacy switch routes.
- Emit audited backlog mutation metadata for operating-mode item completion and proposal application.
- Fail closed when a registered operating-mode prompt skill cannot be rendered.
- Validate every registry-referenced AgentManager profile key at startup.
- Fix replan stats double counting.
- Refactor `api/internal/operatingmode/service.go` into smaller responsibility-owned units.
- Refactor `ui/src/components/initiative/operating-mode-panel.tsx` into workspace components and hooks.
- Update API, CLI, UI, stats, and internal seam docs to match the hardened architecture.
- Add backend, UI, and CLI regression tests for the new invariants.

### Out of Scope

- Adding new operating modes.
- Dynamic mode plugins.
- Runtime operator profile editing.
- Automatic mode recommendation.
- Parallel worker fanout.
- Cross-initiative execution.
- Cost-budget enforcement.
- Compatibility aliases for old mode mutation shapes.
- Supporting generic PATCH mode changes as a fallback.

## 5. Current Technical Findings

### Phase Graph Is Descriptive, Not Enforced

`api/internal/operatingmode/registry.go` declares phase graphs for `holistic-loop` and `phased-plan-drain`, but `api/internal/operatingmode/service.go:338` only validates that a requested phase exists. The UI renders every phase button as startable whenever no round is active.

Risk:

- Operators can start `review` before `investigate` or `plan`.
- `phased-plan-drain` can skip `classify_progress`.
- Sequential handoff can become a loose collection of unrelated runs.

### Mode Mutation Has Two Public Paths

`api/internal/initiatives/service.go` still accepts `UpdateRequest.Mode`. The operating-mode switch endpoint also updates mode.

Risk:

- Generic PATCH bypasses active item execution cancellation.
- Generic PATCH bypasses active operating-mode round checks.
- Mode-change behavior becomes split across two services.

### Backlog Completion Loses Audit Source

`routes_operating_mode.go` receives a completion source such as `holistic-loop/round-001`, but discards it before emitting `backlog.status_changed`.

Risk:

- Event-log readers cannot reconstruct which mode, phase, round, and run completed a backlog item.
- The durable project-management paper trail is weaker than the plan requires.

### Prompt Fallback Is Too Permissive

`StartPhase` logs prompt-manager failures and falls back to a generic prompt.

Risk:

- A missing or malformed prompt skill can still spawn a repo-writing initiative agent.
- Prompt catalog tests can pass while runtime skill availability is broken.

### Profile Policy Is Not Fully Startup-Validated

AgentManager reconciliation records all returned profile keys, but startup only requires the configured default key. Registry references to `swarm-manager/deep-work` and `swarm-manager/analysis` are validated later at spawn time.

Risk:

- The API can start successfully in a configuration that cannot execute registered operating-mode phases.
- Operators discover profile drift only after starting a phase.

### Stats Double-Count Replan Signals

The stats engine increments replan numerator from completed execute phase payloads and again from `operating_mode.replan_needed` events.

Risk:

- Replan rates can exceed 100% or otherwise misrepresent mode health.

### Monolith Pressure

`api/internal/operatingmode/service.go` is over 1,100 lines and owns switching, phase start, prompt rendering, spawning, polling, artifact application, event emission, backlog sync, and workspace shaping. `ui/src/components/initiative/operating-mode-panel.tsx` is over 500 lines and owns every workspace concern.

Risk:

- New modes will accrete behavior into catch-all files.
- Tests will cover happy paths but miss boundary violations.
- Decision points will drift from the registry into service/UI conditionals.

## 6. Target End State

After this hardening pass:

1. Operating-mode phase start is governed by a state machine derived from the registered mode definition.
2. `holistic-loop` can only follow valid loop transitions.
3. `phased-plan-drain` can only follow valid prepare/execute/classify/review transitions.
4. Run strategies own their own prerequisites, such as prior handoff context for sequential handoff.
5. Mode changes only occur through the operating-mode switch service.
6. Switching into non-default modes requires explicit cancellation of active item-level executions.
7. Switching back to `item-level` is rejected while a non-default mode round is active.
8. Generic initiative PATCH cannot change mode.
9. Every operating-mode backlog mutation event includes mode, phase, round, run ID, and source.
10. Registered operating-mode prompt skills fail closed when unavailable or malformed.
11. Startup fails if any registry-referenced AgentManager profile key is missing from scenario profile reconciliation.
12. Replan metrics are counted from one canonical event path.
13. The operating-mode backend is split into responsibility-focused units with narrow interfaces.
14. The Initiative Mode UI is split into service-backed hooks and mode workspace components.
15. Tests exercise invariants, not only happy-path plumbing.

## 7. Architecture Decisions

### Decision A - Registry Owns Methodology

The registry remains the source of truth for:

- mode values
- scope kind
- phases
- legal transitions
- run strategy
- artifact policy
- prompt skill IDs
- AgentManager profile keys
- backlog sync capability
- metrics policy
- UI workspace identity

No handler or UI component should encode mode-specific sequencing except by consuming workspace state produced by the backend.

### Decision B - Mode Switch Is a Hard Boundary

Initiative CRUD owns title, description, priority, notes, dependency refs, member item refs, acceptance criteria, and status guards.

Operating-mode switch owns mode changes. Generic initiative update must reject or ignore mode fields at decode/validation time. Since this is greenfield, prefer removing `Mode` from public update request types over keeping a deprecated field.

### Decision C - Phase State Is Backend-Authoritative

The backend computes which phases are startable. The UI renders the backend-provided phase action state. The UI may present affordances, but it must not be the source of truth for phase validity.

### Decision D - Prompt Failure Is Execution Failure

Prompt rendering failure for a registered operating-mode phase is a 500/failed-start error. Do not spawn with fallback prose. Fallback prompts may remain only in tests or explicit local debug hooks that are not used by production route wiring.

### Decision E - Audit Events Carry Causality

Operating-mode backlog mutations must emit a source payload that can answer:

- Which initiative?
- Which operating mode?
- Which phase?
- Which round?
- Which AgentManager run?
- Which mutation or item ref?
- Which operator or agent requested it?

### Decision F - Greenfield Cleanup Is Preferred

Remove dead compatibility fields, old aliases, duplicate switch paths, and unused helper code as part of this work. Do not keep hidden secondary paths for mode mutation.

## 8. Implementation Strategy

### Phase 0 - Baseline Contract Tests

Add failing tests before refactoring.

Backend tests:

- `operatingmode.Service.StartPhase` rejects `review` as the first holistic-loop phase.
- `operatingmode.Service.StartPhase` accepts `investigate` as the first holistic-loop phase.
- After completed `investigate`, only `plan` is startable for holistic-loop.
- After completed `execute` with `replan_needed=false`, `review` is startable for holistic-loop.
- After completed `execute` with `replan_needed=true`, `investigate` or `plan` is startable according to the chosen loop policy; document the exact policy in the registry.
- `phased-plan-drain` rejects `execute_next` before `prepare_plan`.
- `phased-plan-drain` rejects `review` before `classify_progress` returns `complete`.
- `phased-plan-drain` starts `execute_next` after `classify_progress` returns `continue`.
- `phased-plan-drain` starts `prepare_plan` after `classify_progress` returns `replan`.
- Mode switch rejects switching to `item-level` while an operating-mode round is active.
- Generic initiative update cannot change `mode`.
- Prompt render failure prevents phase spawn.
- Startup profile validation fails when a registry profile is absent.
- Replan stats count one numerator per completed execute phase with `replan_needed=true`.

UI tests:

- Mode workspace renders only backend-startable phase actions as enabled.
- Invalid phase actions remain disabled with a clear reason.
- Mode switch conflict surfaces active round or active item execution blockers.

### Phase 1 - Extract Phase State Machine

Files:

- `api/internal/operatingmode/registry.go`
- `api/internal/operatingmode/service.go`
- new `api/internal/operatingmode/state.go`
- new `api/internal/operatingmode/run_strategy.go`
- tests in `api/internal/operatingmode/*_test.go`

Tasks:

1. Add a `PhaseState` or `PhaseAction` model:

   ```go
   type PhaseAction struct {
       Phase       Phase
       Startable   bool
       Reason      string
       Next        bool
   }
   ```

2. Add a backend function that computes phase actions from:
   - registered definition
   - completed/canceled/failed rounds
   - latest progress classification
   - latest replan signal
   - active round state
   - acceptance criteria availability
3. Add a transition validator used by `StartPhase`.
4. Move hardcoded phase sort/order out of UI assumptions and into backend workspace shape.
5. Treat failed/canceled rounds deliberately:
   - Active rounds block all new starts.
   - Failed/canceled rounds do not advance the phase graph.
   - The next allowed phase remains based on the last completed round.
6. Add run-strategy-specific validation:
   - `operator_gated_loop`: permits registered loop transitions only.
   - `sequential_handoff`: requires durable handoff/progress context before continuing beyond prepare.
   - `existing_item_flow`: rejects phase starts because item-level is owned by existing backlog execution.

Contract:

- Unknown phases fail closed.
- Registered but currently invalid phases return actionable reasons.
- Handlers remain thin.

### Phase 2 - Harden Mode Switch Boundary

Files:

- `api/internal/initiatives/model.go`
- `api/internal/initiatives/service.go`
- `api/internal/initiatives/handler.go`
- `api/internal/operatingmode/service.go`
- `api/internal/operatingmode/handler.go`
- `api/routes_operating_mode.go`
- `ui/src/services/initiative-service.ts`
- `ui/src/services/initiative-mode-service.ts`
- `cli/cmd_initiatives_operating_mode.go`
- docs and tests

Tasks:

1. Remove `Mode` from public initiative create/update request handling unless create-time mode is still intentionally supported. Preferred greenfield target:
   - Create always defaults to `item-level`.
   - Switch endpoint changes mode.
2. Add an internal initiative method such as `SetModeLifecycle(name, mode string)` or a narrow `ModeStore` adapter for operating-mode switch.
3. Ensure only `operatingmode.Service.SwitchMode` can call that method.
4. Reject mode switches on archived initiatives.
5. Reject switching to `item-level` if any non-default mode round is active.
6. Reject switching between non-default modes if any mode round is active.
7. Preserve artifacts and round files when switching, but do not migrate or delete them.
8. Update UI and CLI to call only operating-mode switch APIs.
9. Update docs to remove generic PATCH mode mutation.

Contract:

- A repository-wide search for public update mode mutation should find no supported path outside operating-mode switch.
- Mode changes always produce an `initiative.mode_changed` event.

### Phase 3 - Audit-Safe Backlog Reconciliation

Files:

- `api/routes_operating_mode.go`
- `api/internal/operatingmode/service.go`
- `api/internal/eventlog/types.go`
- `api/internal/eventlog/emitter.go`
- `api/internal/stats/engine.go`
- `api/internal/backlog` event surfaces if needed
- tests

Tasks:

1. Replace the string-only completion source with a structured source:

   ```go
   type BacklogMutationSource struct {
       Entrypoint     string
       InitiativeName string
       Mode           string
       Phase          string
       Round          int
       RunID          string
       RequestedBy    string
   }
   ```

2. Use this source for `CompleteItems`.
3. Emit event metadata that includes mode, phase, round, run ID, item refs, and source.
4. Ensure proposal-based backlog sync already carries equivalent source through `proposals.Source`; add missing mode/phase/run fields if needed.
5. Update stats to count operating-mode backlog sync from explicit operating-mode events, not from generic backlog status inference.
6. Add tests that load event metadata and assert mode/phase/round/run are present.

Contract:

- No operating-mode item mutation is anonymous.
- Backlog item specs are not directly edited by operating-mode agents.

### Phase 4 - Fail Closed on Prompt Skill Resolution

Files:

- `api/internal/operatingmode/service.go`
- `api/internal/promptcatalog/catalog.go`
- `api/internal/promptcatalog/catalog_test.go`
- prompt-manager skill files
- tests

Tasks:

1. Remove production fallback prompt behavior from operating-mode phase start.
2. Validate the phase definition's catalog entry before rendering.
3. Render the prompt skill through prompt-manager.
4. If catalog lookup fails, skill read fails, or required variables are missing, mark the round failed or avoid creating the round entirely. Choose one consistent behavior:
   - Preferred: validate prompt before reserving a round where possible.
   - If the round must be reserved for audit, save it as `failed` with a prompt error and release lock.
5. Add tests proving no AgentManager spawn occurs on prompt failure.
6. Validate all eight prompt-manager skills with `prompt-manager skill read`.

Contract:

- Registered phases cannot run with generic prompt prose.
- Prompt failures are visible and actionable.

### Phase 5 - Startup Profile Policy Validation

Files:

- `api/internal/agentmanager/service.go`
- `api/internal/agentmanager/profile.go`
- `api/internal/operatingmode/registry.go`
- server startup wiring
- `.vrooli/agent-profiles/*.json`
- tests

Tasks:

1. Add a registry helper that returns every referenced profile key.
2. After AgentManager scenario profile reconciliation, validate all operating-mode profile keys are present.
3. Fail startup if any referenced key is missing.
4. Validate profile keys are scenario-owned and start with `swarm-manager/`.
5. Add unit tests for:
   - all registry profiles present
   - missing `swarm-manager/deep-work`
   - missing `swarm-manager/analysis`
   - non-owned profile key in registry
6. Keep profile contents in `.vrooli/agent-profiles/*.json`; do not inline profile defaults in operating-mode code.

Contract:

- If the API is up, every registered operating-mode phase has a valid reconciled AgentManager profile.

### Phase 6 - Correct Mode Stats

Files:

- `api/internal/stats/engine.go`
- `api/internal/stats/metrics.go`
- `api/internal/stats/*_test.go`
- `ui/src/surfaces/graph/components/StatsPanel.tsx`

Tasks:

1. Choose a single source for replan numerator.
   - Preferred: count replan from completed execute phase payload only.
   - Keep `operating_mode.replan_needed` as an event for timeline observability, but do not increment the numerator from both paths.
2. Add tests for:
   - one completed execute with `replan_needed=true` produces numerator 1 denominator 1
   - completed execute plus separate replan event still produces numerator 1 denominator 1
   - completed execute with `replan_needed=false` produces numerator 0 denominator 1
3. Review acceptance verdict normalization and make verdict constants explicit.
4. Add UI tests for rates and sample sizes.

Contract:

- Mode health metrics are numerically trustworthy.

### Phase 7 - Backend Responsibility Refactor

Files:

- `api/internal/operatingmode/service.go`
- new files under `api/internal/operatingmode/`
- tests

Refactor target:

- `service.go`: public facade and dependency wiring only.
- `switcher.go`: mode switch lifecycle.
- `phase_runner.go`: start phase orchestration.
- `round_refresher.go`: polling and terminal-state refresh.
- `prompt.go`: prompt variable construction and skill rendering.
- `artifact_applier.go`: structured result parsing and artifact writes.
- `backlog_reconciler.go`: completion and proposal application orchestration.
- `workspace.go`: workspace read model and phase actions.
- `events.go`: operating-mode event emission helpers.
- `state.go`: phase graph/run-strategy state machine.

Rules:

- Do not change behavior without a test in the earlier phases.
- Keep adapters in `routes_operating_mode.go` thin; move logic into typed package boundaries where it belongs.
- Avoid circular imports by keeping interfaces narrow.
- Remove helpers made obsolete by the refactor.

Contract:

- Each file has one reason to change.
- Mode-specific decisions remain registry-driven.

### Phase 8 - UI Workspace Refactor

Files:

- `ui/src/components/initiative/operating-mode-panel.tsx`
- new components/hooks under `ui/src/components/initiative/operating-mode/`
- `ui/src/services/initiative-mode-service.ts`
- `ui/src/types/operating-mode.ts`
- tests

Refactor target:

- `OperatingModePanel`: composition only.
- `useOperatingModeWorkspace`: query/mutations/invalidation.
- `ModeSwitchControl`: mode selection and cancellation confirmation.
- `AcceptanceCriteriaEditor`: criteria editing.
- `PhaseControls`: startable phase actions from backend.
- `ArtifactList`: declared/current artifacts.
- `RoundTimeline`: round list.
- `RoundCard`: one round.
- `BacklogSyncActions`: complete-items and proposal application.

Tasks:

1. Render backend-provided startability, reason, and next-phase status.
2. Disable invalid phase buttons with precise reasons.
3. Remove UI-only phase sequencing assumptions.
4. Keep React Query as the server-state owner.
5. Keep local state local to controls.
6. Avoid nested card structures and excessive rounded styling while touching the workspace.
7. Add tests around:
   - phase actions
   - active round blocking
   - mode switch conflict
   - backlog sync actions
   - acceptance criteria save

Contract:

- UI is an operating-mode workspace client, not the mode state machine.

### Phase 9 - Documentation and Seam Health

Files:

- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/reference/api-endpoints.md`
- `scenarios/swarm-manager/docs/reference/cli-commands.md`
- `scenarios/swarm-manager/docs/guides/holistic-loop-mode.md`
- `scenarios/swarm-manager/docs/guides/phased-plan-drain-mode.md`
- `scenarios/swarm-manager/docs/manifest.json`

Tasks:

1. Document phase state machine ownership.
2. Document mode switch as the only mode mutation path.
3. Document backlog mutation audit metadata.
4. Document prompt fail-closed behavior.
5. Document startup profile policy validation.
6. Update API/CLI examples to remove old mode mutation paths.
7. Add code references to the refactored operating-mode boundary files.

Contract:

- Documentation describes the code that exists after the hardening pass, not aspirational behavior.

## 9. Validation Plan

Run targeted tests after each phase:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode/... ./internal/initiatives/... ./internal/agentmanager/... ./internal/eventlog/... ./internal/stats/... -timeout 300s
```

```bash
cd scenarios/swarm-manager/ui
npm test -- InitiativeDetailsPage initiative-mode OperatingModePanel StatsPanel stats-service
```

Run CLI tests after CLI/API changes:

```bash
cd scenarios/swarm-manager/cli
go test ./... -timeout 300s
```

Run scenario validation before completion:

```bash
cd scenarios/swarm-manager
make test
```

Then:

```bash
vrooli scenario test swarm-manager
vrooli scenario test prompt-manager
vrooli scenario test agent-manager
```

Manual validation checklist:

1. Create an initiative.
2. Confirm its mode is `item-level`.
3. Attempt generic PATCH mode mutation and verify rejection.
4. Switch to `holistic-loop` through the operating-mode endpoint.
5. Verify `review` cannot start first.
6. Run `investigate`, refresh to completion, then verify only valid next phases are enabled.
7. Simulate prompt skill failure and verify no AgentManager run is spawned.
8. Simulate missing `swarm-manager/deep-work` profile and verify startup fails.
9. Complete a member item through operating-mode API and inspect event metadata for mode/phase/round/run.
10. Generate a replan-needed execute result and verify stats replan numerator is not double-counted.

## 10. Completion Criteria

This plan is complete only when:

- The first five high-value findings are fixed with regression tests.
- Public generic initiative update cannot change mode.
- Backend phase start rejects invalid phase ordering.
- UI renders backend phase action state.
- Operating-mode backlog mutation events carry full source metadata.
- Prompt-manager failures do not spawn operating-mode agents.
- Startup validates every registry-referenced profile key.
- Replan stats are correct.
- `operatingmode/service.go` is decomposed into responsibility-owned files.
- `operating-mode-panel.tsx` is decomposed into workspace components/hooks.
- Docs and tests reflect the new hard-cutover architecture.

## 11. Implementation Notes

- Prefer deleting bypasses to deprecating them.
- Keep `item-level` as the default mode, but do not treat missing mode as an excuse for hidden legacy code.
- Use typed constants for verdicts, progress decisions, phase names, and event names where they cross package boundaries.
- Keep run-strategy logic testable without AgentManager.
- Keep artifact and round persistence deterministic and filesystem-testable.
- Do not add package-level global state beyond the static registry.
- Do not introduce compatibility aliases for older endpoint shapes unless another active scenario demonstrably requires them and the user explicitly accepts that tradeoff.
