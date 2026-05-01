# Swarm Manager Operating Mode Authoring Architecture Plan

## 0. Current Status

Status as of 2026-05-01: **implemented; closeout validation remains**.

The architecture this plan originally requested has landed in the current codebase:

- concrete production mode definitions live in focused files:
  - `scenarios/swarm-manager/api/internal/operatingmode/mode_item_level.go`
  - `scenarios/swarm-manager/api/internal/operatingmode/mode_holistic_loop.go`
  - `scenarios/swarm-manager/api/internal/operatingmode/mode_phased_plan_drain.go`
- `definition_builder.go` provides the static initiative-mode authoring surface.
- `state.go` derives phase availability from registry transitions and transition rules.
- `artifact_applier.go` handles derived artifacts through phase `ResultBindings`.
- operating-mode prompt catalog entries are generated from registry phase metadata.
- mode metrics semantics are registry-owned through `MetricsPolicy`.
- catalog/workspace responses expose backend-derived mode capabilities.
- `synthetic_mode_test.go` proves a non-production mode can exercise transitions, result bindings, prompt catalog validation, metrics policy, and capability projection without leaking into production mode lists.
- `scenarios/swarm-manager/docs/internal/OPERATING-MODE-AUTHORING.md` documents the add-a-mode contract and is registered in `docs/manifest.json`.

Remaining work for this plan is limited to closeout:

1. Run the targeted backend, CLI, and UI validation commands in section 10.
2. Fix any regressions found by validation.
3. Keep this plan as implementation history and source context for the `swarm-manager-operating-mode-authoring` skill.

## 1. Purpose

Make Swarm Manager operating modes easy and safe to add before building a `mode-authoring` skill.

The current operating-mode implementation is functional and already hardened in important ways: static registry, phase lifecycle ownership, structured `operating_mode_result` parsing, output contracts, profile selection, backlog sync, stats, CLI, UI, and docs exist. The remaining work is architectural consolidation so a future agent can add a fourth operating mode by following an obvious authoring surface instead of editing scattered framework internals.

Target principle:

> Adding a new static operating mode should mostly require one mode definition file plus prompt skills. Special behavior should be expressed through registered policies, not by adding mode-specific branches across the framework.

## 2. Required Reading

Run this before implementation:

```bash
prompt-manager skill read implementation-plan-authoring cli-steer api-steer utils-unification seam-discovery-and-enforcement boundary-of-responsibility-enforcement decision-boundary-extraction screaming-architecture-audit
```

Also read:

- `docs/plans/swarm-manager-initiative-operating-mode-implementation.md`
- `docs/plans/swarm-manager-operating-mode-hardening-plan.md`
- `docs/plans/swarm-manager-operating-mode-architecture-hardening-plan.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/internal/INVARIANTS.md`
- `scenarios/swarm-manager/docs/internal/TEMPORAL-FLOWS.md`
- `scenarios/swarm-manager/api/internal/operatingmode/registry.go`
- `scenarios/swarm-manager/api/internal/operatingmode/state.go`
- `scenarios/swarm-manager/api/internal/operatingmode/artifact_applier.go`
- `scenarios/swarm-manager/api/internal/operatingmode/output.go`
- `scenarios/swarm-manager/api/internal/operatingmode/payload.go`
- `scenarios/swarm-manager/api/internal/operatingmode/phase_runner.go`
- `scenarios/swarm-manager/api/internal/operatingmode/workspace.go`
- `scenarios/swarm-manager/api/internal/promptcatalog/catalog.go`
- `scenarios/swarm-manager/api/internal/agentactivity/types.go`
- `scenarios/swarm-manager/api/internal/initiativelock/lock.go`
- `scenarios/swarm-manager/api/internal/stats/engine.go`
- `scenarios/swarm-manager/ui/src/components/initiative/operating-mode/`
- `scenarios/swarm-manager/ui/src/services/initiative-mode-service.ts`
- `scenarios/swarm-manager/ui/src/types/operating-mode.ts`

## 3. Problem Statement

The framework supports multiple operating modes, but adding a new mode still requires too much knowledge of hidden cross-package conventions.

Current add-a-mode touch points include:

- `api/internal/operatingmode/registry.go` for constants, nested definitions, phases, artifacts, profiles, and contracts.
- `api/internal/operatingmode/state.go` when phase transitions depend on phase output. It currently references `ModeHolisticLoop`, `ModePhasedPlanDrain`, `execute`, `execute_next`, and `classify_progress`.
- `api/internal/operatingmode/artifact_applier.go` when a mode needs derived artifacts. It currently hardcodes phased-plan progress persistence to `modes/phased-plan-drain/progress.json`.
- `api/internal/promptcatalog/catalog.go` for prompt metadata that duplicates registry mode/phase/skill/output-path facts.
- `api/internal/agentactivity/types.go` and `api/internal/initiativelock/lock.go` for new purpose constants, even though purposes are stable strings.
- Stats logic in `api/internal/stats/engine.go` for mode-specific metric semantics such as which phases count toward replan rate.
- UI and CLI surfaces that are mostly generic but still rely on implied capabilities instead of backend-declared mode capabilities.

This is acceptable for two non-default modes, but it is not the right substrate for a future `mode-authoring` skill. A skill should not need to edit framework logic in multiple packages just because a new methodology has a classifier phase, derived artifact, or different metric semantics.

## 4. Scope

### In Scope

- Split concrete operating-mode definitions away from registry core logic.
- Add definition helpers/builders that make mode definitions declarative and difficult to assemble incorrectly.
- Move conditional transition behavior into registry-owned transition policies.
- Move derived artifact writes into registry-owned result bindings.
- Stop requiring new shared `agentactivity` and `initiativelock` constants for every mode phase.
- Reduce prompt catalog duplication for operating-mode phases.
- Move operating-mode metric semantics into registry policy.
- Add backend-declared mode capabilities for UI/CLI rendering.
- Add a synthetic test mode or equivalent test harness proving a new mode can exercise nontrivial behavior without framework edits.
- Add authoring documentation that a future `mode-authoring` skill can use as source material.
- Update tests and internal docs to match the clarified boundaries.

### Out of Scope

- Building the `mode-authoring` skill itself.
- Dynamic runtime plugin loading.
- Runtime creation or editing of modes.
- Data-file-only mode definitions.
- Adding a new production operating mode.
- Changing user-facing behavior of `item-level`, `holistic-loop`, or `phased-plan-drain`.
- Reworking backlog-item execution internals.
- Rewriting the operating-mode UI beyond capability-driven rendering and small view-model adjustments.

## 5. Greenfield Constraint

This is greenfield for the operating-mode authoring architecture. Do not preserve awkward compatibility paths inside the framework.

Rules:

- No new silent fallbacks.
- No duplicate mode facts across packages unless a validation test proves they cannot drift.
- No new mode-specific `if mode == ...` branches in shared operating-mode control flow.
- No UI-local phase or capability inference when the backend can declare the rule.
- No broad rewrites unrelated to making modes easier to author.

Existing persisted rounds and existing production mode names should continue to read correctly, but this plan is not a migration compatibility project.

## 6. Current Technical Context

The core operating-mode package is `scenarios/swarm-manager/api/internal/operatingmode/`.

Important current strengths:

- `registry.go` defines `Definition`, `PhaseDefinition`, `PhaseOutputContract`, profile policy, backlog sync policy, metrics policy, lock policy, and UI policy.
- `ValidateRegistry` and `ValidatePromptCatalog` already catch malformed definitions and prompt drift.
- `phase_runner.go` owns phase start lifecycle, prompt rendering, lock acquisition, AgentManager spawn, run ID persistence, and failure cleanup.
- `round_refresher.go`, `output.go`, and `artifact_applier.go` parse final agent output and enforce phase result contracts before a round can complete.
- `payload.go` centralizes typed payload access for common persisted fields.
- `state.go` computes backend-authoritative phase actions.
- `backlog_reconciler.go` owns run-id-validated backlog completion and proposal application.
- `events.go` emits structured operating-mode events used by stats.
- UI mode workspace components have already been decomposed, and `round-view-model.ts` keeps raw payload parsing out of `round-card.tsx`.

Current friction points:

- Concrete mode definitions are large nested literals inside `registry.go`.
- Run-strategy and transition semantics are partly declarative but still partly hardcoded in `state.go`.
- Derived result persistence is not declarative; `artifact_applier.go` knows about phased-plan progress.
- Activity and lock purpose constants force shared packages to learn every new mode phase.
- Prompt catalog metadata repeats registry facts.
- Stats has phase-name assumptions for replan and acceptance semantics.
- UI catalog/workspace metadata does not expose all capabilities needed by a future mode.

## 7. Target End State

After this plan lands:

- A future mode author can add a static mode by creating one `mode_<name>.go` definition file, prompt skills, tests, and docs.
- Shared framework files do not need mode-specific branches for transitions, derived artifacts, purposes, metrics, or UI capabilities.
- The registry validates every authoring contract at startup and in unit tests.
- Prompt catalog and registry facts are generated from one source or validated so drift fails fast.
- Metrics are driven by mode policy, not hardcoded phase names.
- UI and CLI render mode actions from backend-declared capabilities.
- A synthetic-mode test proves the authoring architecture supports a new nontrivial mode without editing core control flow.
- Documentation explicitly states what adding a mode should and should not touch.

## 8. Implementation Strategy

The phases below describe the implementation that now exists. Treat unchecked items in this section as historical execution notes unless validation proves a listed acceptance criterion is no longer true.

### Phase 1 - Split Concrete Mode Definitions From Registry Core

Files:

- `api/internal/operatingmode/registry.go`
- new `api/internal/operatingmode/mode_item_level.go`
- new `api/internal/operatingmode/mode_holistic_loop.go`
- new `api/internal/operatingmode/mode_phased_plan_drain.go`
- `api/internal/operatingmode/registry_test.go`

Tasks:

1. Keep shared types, lookup, validation, and registry assembly in `registry.go`.
2. Move `ModeItemLevel`, `ModeHolisticLoop`, and `ModePhasedPlanDrain` concrete definitions into focused files.
3. Preserve existing mode IDs, phase IDs, profile keys, artifact paths, prompt IDs, and output contracts.
4. Keep tests passing with no behavior change.

Acceptance criteria:

- `registry.go` no longer contains the full concrete definitions for all modes.
- Each production mode has a focused definition file.
- `cd scenarios/swarm-manager/api && GOWORK=off go test ./internal/operatingmode` passes.

### Phase 2 - Add Definition Builder Helpers

Files:

- `api/internal/operatingmode/registry.go`
- possible new `api/internal/operatingmode/definition_builder.go`
- mode definition files from Phase 1
- `api/internal/operatingmode/registry_test.go`

Tasks:

1. Add small typed helpers for initiative mode construction.
2. Keep helpers boring and explicit; avoid a clever DSL.
3. Ensure helpers reduce duplicated fields such as catalog ID, skill ID, artifact root, round root, profile policy, output contract defaults, and activity/lock purpose derivation.
4. Refactor existing production definitions to use the helpers.

Acceptance criteria:

- A mode definition reads as a declarative list of phases, transitions, artifacts, profiles, and policies.
- Validation remains the final authority.
- No behavior change in catalog, workspace, phase starts, round refresh, or stats.

### Phase 3 - Declarative Transition Policies

Files:

- `api/internal/operatingmode/registry.go`
- `api/internal/operatingmode/state.go`
- mode definition files
- tests in `api/internal/operatingmode/`, especially existing phase-action and `StartPhase` coverage

Tasks:

1. Add a registry-owned transition policy that can express output-dependent transitions.
2. Represent current holistic behavior declaratively:
   - after `execute`, `replan_needed=true` routes to `investigate`
   - otherwise routes to `review`
3. Represent current phased-plan behavior declaratively:
   - after `classify_progress`, `continue` routes to `execute_next`
   - `replan` routes to `prepare_plan`
   - `complete` routes to `review`
   - `blocked` has no startable next phase
4. Refactor `allowedNextPhases`, `loopNextPhases`, and `sequentialNextPhases` so shared logic does not reference concrete modes or concrete phase names.

Acceptance criteria:

- `state.go` has no references to `ModeHolisticLoop`, `ModePhasedPlanDrain`, `execute_next`, `classify_progress`, or `prepare_plan`.
- Existing transition tests still pass.
- New tests prove output-dependent transitions are driven by registry policy.

### Phase 4 - Declarative Result Bindings For Derived Artifacts

Files:

- `api/internal/operatingmode/registry.go`
- `api/internal/operatingmode/artifact_applier.go`
- `api/internal/operatingmode/artifacts.go`
- `api/internal/operatingmode/output.go`
- tests in `api/internal/operatingmode/`

Tasks:

1. Add a result-binding concept to phase definitions.
2. Support at least `progress -> artifact path` binding for phased-plan `progress.json`.
3. Consider support for future bindings such as `verdict`, `readiness`, `backlog_sync`, or `handoff_index`, but only implement what is immediately useful and cleanly testable.
4. Update output contract validation so required artifacts can be satisfied by either agent-provided artifacts or derived bindings.
5. Remove the hardcoded `ModePhasedPlanDrain` path branch from `artifact_applier.go`.

Acceptance criteria:

- `artifact_applier.go` has no concrete mode name or phased-plan artifact path.
- Phased-plan `classify_progress` still writes `modes/phased-plan-drain/progress.json`.
- Required artifact contract tests cover derived artifacts.

### Phase 5 - Consolidate Activity And Lock Purpose Authoring

Files:

- `api/internal/operatingmode/registry.go`
- mode definition files
- `api/internal/agentactivity/types.go`
- `api/internal/initiativelock/lock.go`
- tests that assert activity purposes

Tasks:

1. Stop requiring shared packages to define a new constant for every operating-mode phase.
2. Keep existing purpose string values for current modes unless deliberately changing them is cleaner and all tests/docs are updated.
3. Define phase purpose strings in the operating-mode definition layer or derive them from `mode + phase`.
4. Keep `agentactivity.Purpose` and lock `Purpose` as string-compatible values.

Acceptance criteria:

- Adding a new mode phase does not require editing `agentactivity/types.go`.
- Adding a new mode phase does not require editing `initiativelock/lock.go`.
- Agent activity metadata still records `operating_mode`, `phase`, `run_strategy`, `round_number`, `artifact_set`, and `agent_profile_key`.

### Phase 6 - Reduce Prompt Catalog Duplication

Files:

- `api/internal/promptcatalog/catalog.go`
- `api/internal/promptcatalog/catalog_test.go`
- `api/internal/operatingmode/registry.go`
- possible new `api/internal/operatingmode/prompt_catalog_entries.go`

Tasks:

1. Choose one source of truth for operating-mode prompt metadata.
2. Preferred approach: operating-mode definitions expose generated prompt catalog entries consumed by `promptcatalog`.
3. Acceptable approach: keep prompt catalog entries separate but add strict tests that compare every registry phase with catalog mode, operation, skill ID, catalog ID, and output paths.
4. Preserve Prompt Center visibility for all operating-mode skills.

Acceptance criteria:

- Adding a new phase does not require manually duplicating mode, phase, skill ID, and output paths in unrelated code without validation.
- `ValidatePromptCatalog` remains strict.
- Prompt catalog tests fail on missing or mismatched operating-mode phase entries.

### Phase 7 - Registry-Driven Metrics Semantics

Files:

- `api/internal/operatingmode/registry.go`
- `api/internal/stats/engine.go`
- `api/internal/stats/metrics.go`
- `api/internal/stats/engine_test.go`
- `ui/src/types/stats.ts`
- `ui/src/surfaces/graph/components/StatsPanel.tsx` only if response shape changes

Tasks:

1. Extend `MetricsPolicy` with explicit semantic fields, for example:
   - phases that count toward replan sample size
   - phases that count toward acceptance sample size
   - accepted verdict values
2. Refactor stats aggregation so it does not hardcode operating-mode phase names such as `execute` and `execute_next`.
3. Preserve current stats output shape unless a response change is truly justified.

Acceptance criteria:

- `stats/engine.go` does not hardcode current operating-mode phase names for replan or acceptance semantics.
- Existing mode stats tests pass with unchanged observed values.
- A synthetic or unit test proves a new mode can opt into metrics by registry policy.

### Phase 8 - Backend-Declared Mode Capabilities For UI And CLI

Files:

- `api/internal/operatingmode/service.go`
- `api/internal/operatingmode/workspace.go`
- `ui/src/types/operating-mode.ts`
- `ui/src/services/initiative-mode-service.ts`
- `ui/src/components/initiative/operating-mode/`
- `cli/cmd_initiatives_operating_mode.go`
- related UI/CLI tests

Tasks:

1. Add capability metadata to catalog/workspace responses.
2. Include capabilities such as:
   - supports phases
   - can start phases
   - can complete items
   - can apply backlog sync proposals
   - requires acceptance criteria
   - supports artifacts
   - supports handoffs
3. Refactor UI controls to render from backend capabilities rather than implied mode behavior.
4. Keep CLI thin: display capabilities, but do not implement business rules locally.

Acceptance criteria:

- UI avoids mode-name checks except where displaying the current mode label.
- CLI remains a thin API wrapper.
- Existing mode panel and service tests pass.

### Phase 9 - Synthetic Mode Test Harness

Files:

- `api/internal/operatingmode/registry_test.go`
- `api/internal/operatingmode/service_test.go`
- possible new test-only helpers in `api/internal/operatingmode/`

Tasks:

1. Add a test-only synthetic operating mode that is not production-registered.
2. The synthetic mode should exercise at least:
   - non-default initiative scope
   - two or more phases
   - output-dependent transition
   - derived artifact binding
   - metrics policy
   - prompt catalog validation
3. Use it to prove the framework can support a new methodology without editing shared control-flow branches.

Acceptance criteria:

- Tests can validate and execute synthetic-mode behavior through the same public service helpers used by production modes.
- The synthetic mode does not leak into production catalog responses.
- A failing framework branch or hardcoded mode assumption is caught by tests.

### Phase 10 - Authoring Documentation

Files:

- new `scenarios/swarm-manager/docs/internal/OPERATING-MODE-AUTHORING.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/internal/INVARIANTS.md`
- `scenarios/swarm-manager/docs/internal/TEMPORAL-FLOWS.md`
- `scenarios/swarm-manager/docs/manifest.json` if internal docs are registered there

Tasks:

1. Document the intended add-a-mode workflow.
2. Include the expected files to touch and files that should not need edits.
3. Include examples for phases, transitions, output contracts, result bindings, metrics policy, profile policy, prompt skills, and tests.
4. Include validation commands.
5. Update seams/invariants/temporal docs to reflect the new authoring architecture.

Acceptance criteria:

- A future agent can read `OPERATING-MODE-AUTHORING.md` and identify the exact authoring surface.
- Docs explicitly prohibit adding mode-specific branches to shared framework logic.
- Documentation references current code paths accurately.

## 9. Contract Decisions

- **Mode definitions remain static Go code.** Runtime plugins and data-file-only modes are intentionally out of scope.
- **Registry is the source of framework behavior.** Transitions, result bindings, output contracts, metrics semantics, profile policy, backlog sync policy, and UI capabilities belong in or derive from mode definitions.
- **Prompt skills remain prompt-manager skills.** The framework should validate exact skill IDs and rendered content, not inline prompt bodies in Go.
- **API remains authoritative.** UI and CLI should consume catalog/workspace capabilities and action state; they should not infer business rules locally.
- **Round payload JSON can remain flexible.** Access must go through typed helpers/view models where practical.
- **Backlog audit remains mandatory.** Non-default modes may execute initiative-wide work, but backlog reconciliation must continue through run-id-validated APIs and source-attributed events.
- **Existing mode names and persisted artifact paths remain stable.** This plan improves authoring architecture, not user-facing semantics.

## 10. Testing Plan

Backend:

```bash
cd scenarios/swarm-manager/api && GOWORK=off go test ./internal/operatingmode ./internal/promptcatalog ./internal/stats ./internal/agentactivity ./internal/initiativelock
```

CLI:

```bash
cd scenarios/swarm-manager/cli && GOWORK=off go test ./...
```

UI:

```bash
cd scenarios/swarm-manager/ui && pnpm test -- --run
```

Targeted validation after each phase:

- Phase 1-2: `go test ./internal/operatingmode -run 'Registry|Catalog|Phase'`
- Phase 3: `go test ./internal/operatingmode -run 'Phase|Transition|StartPhase'`
- Phase 4: `go test ./internal/operatingmode -run 'Artifact|Output|RefreshRound'`
- Phase 5: `go test ./internal/operatingmode ./internal/agentactivity ./internal/initiativelock`
- Phase 6: `go test ./internal/promptcatalog ./internal/operatingmode -run 'Prompt|Catalog|Registry'`
- Phase 7: `go test ./internal/stats -run 'Mode|OperatingMode'`
- Phase 8: API tests plus UI/CLI tests for mode workspace/catalog rendering
- Phase 9: synthetic mode tests must fail if shared logic hardcodes production mode names
- Phase 10: documentation references should be manually spot-checked with `rg`

Full scenario validation when phases are complete:

```bash
vrooli scenario start swarm-manager
test-genie execute swarm-manager --preset comprehensive
```

If full validation is unavailable, record the reason and the targeted tests that did run.

## 11. Rollout / Validation Checklist

- [ ] Existing production modes still appear in `/api/v1/operating-modes`.
- [ ] `item-level` remains the default mode.
- [ ] `holistic-loop` start, execute replan routing, review gating, artifacts, and backlog sync still work.
- [ ] `phased-plan-drain` prepare, execute-next, classify-progress routing, `progress.json`, handoffs, review gating, and backlog sync still work.
- [ ] Existing stats fields remain populated.
- [ ] UI operating-mode panel renders from backend state and capabilities.
- [ ] CLI mode commands remain thin wrappers over API endpoints.
- [ ] No shared operating-mode framework file contains concrete production mode branches for behavior that belongs in mode policy.
- [ ] Authoring doc explains how to add a mode and what not to edit.

## 12. Risks + Mitigations

| Risk | Mitigation |
|---|---|
| Refactor accidentally changes current mode behavior | Keep phases small; run targeted tests after each phase; preserve mode IDs, phase IDs, artifact paths, and output shapes |
| Builder helpers become a clever DSL that hides behavior | Keep helpers explicit and backed by `ValidateRegistry`; prefer plain structs when clearer |
| Declarative transition policy underfits future modes | Implement current needs plus a synthetic mode test; document any intentionally unsupported transition shape |
| Prompt catalog integration creates import cycles | If direct generation causes cycles, keep separate entries and enforce drift with tests |
| UI capability fields create response churn | Additive fields only; preserve existing fields unless tests show they are redundant |
| Synthetic test mode leaks into production catalog | Keep synthetic registry construction test-local; never add it to the production registry map |
| Docs drift immediately after refactor | Update internal docs in the same phase that changes the boundary |

## 13. Non-Goals / Prohibited Patterns

- Do not implement the `mode-authoring` skill in this plan.
- Do not add a production fourth mode as part of this refactor.
- Do not add runtime plugin loading.
- Do not make mode definitions editable in the UI.
- Do not hand-roll mode-specific logic in handlers, UI components, or stats after this refactor.
- Do not let agents directly edit backlog item `spec.json` as a substitute for backlog reconciliation APIs.
- Do not weaken output contract validation to make tests easier.
- Do not add compatibility aliases or silent fallbacks for malformed mode actions.

## 14. Definition of Done

This plan is complete when:

- Concrete production mode definitions are separated and authorable.
- Current mode-specific transition and derived-artifact branches are represented as mode policy.
- New operating-mode phases do not require edits to shared agentactivity or initiativelock constants.
- Prompt catalog drift is eliminated or strictly validated.
- Metrics semantics are registry-driven.
- UI and CLI consume backend-declared capabilities.
- Synthetic-mode tests prove a nontrivial new mode can be supported without shared control-flow edits.
- `OPERATING-MODE-AUTHORING.md` documents the authoring contract for future agents and the planned `mode-authoring` skill.
- Targeted backend, CLI, and UI tests pass, or any unavailable validation is explicitly documented with rationale.
