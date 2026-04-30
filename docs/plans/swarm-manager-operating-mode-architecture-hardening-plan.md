# Swarm Manager Operating Mode Architecture Hardening Plan

## 1. Purpose

Make Swarm Manager's operating-mode framework production-grade before broad manual testing.

The first implementation and follow-up hardening pass shipped the major surfaces: mode registry, phase runners, mode switching, round artifacts, AgentManager profile policy, prompt skills, backlog reconciliation, event-log stats, CLI, UI workspace, and docs. The next pass should remove the remaining architectural debt around temporal lifecycle safety, phase-output contracts, registry validation, API misuse resistance, typed payload boundaries, and UI responsibility boundaries.

This is not a compatibility pass. Swarm Manager is greenfield for this feature. Delete bypasses, dead paths, legacy aliases, and soft fallbacks instead of preserving them. The target is one clean architecture that future modes can extend without copying control-flow decisions into handlers, UI components, or ad hoc payload parsing.

## 2. Required Reading

Run before implementation:

```bash
prompt-manager skill read implementation-plan-authoring documentation-health decision-boundary-extraction temporal-flow-audit assumption-mapping-and-hardening utils-unification seam-discovery-and-enforcement test
```

Also read:

- `docs/plans/swarm-manager-initiative-operating-mode-implementation.md`
- `docs/plans/swarm-manager-operating-mode-hardening-plan.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/api/internal/operatingmode/registry.go`
- `scenarios/swarm-manager/api/internal/operatingmode/state.go`
- `scenarios/swarm-manager/api/internal/operatingmode/phase_runner.go`
- `scenarios/swarm-manager/api/internal/operatingmode/round_refresher.go`
- `scenarios/swarm-manager/api/internal/operatingmode/artifact_applier.go`
- `scenarios/swarm-manager/api/internal/operatingmode/backlog_reconciler.go`
- `scenarios/swarm-manager/api/internal/operatingmode/handler.go`
- `scenarios/swarm-manager/api/routes_operating_mode.go`
- `scenarios/swarm-manager/ui/src/components/initiative/operating-mode-panel.tsx`
- `scenarios/swarm-manager/ui/src/components/initiative/operating-mode/round-card.tsx`
- `scenarios/swarm-manager/ui/src/services/initiative-mode-service.ts`

## 3. Greenfield Constraint

Hard rule: no compatibility shims, no legacy endpoint aliases, no silent fallback behavior, no dead code preservation.

If two paths can mutate operating-mode state, choose one and delete the other. If an API shape is easy to misuse, tighten it now. If a prompt, phase result, profile, artifact, or backlog reconciliation contract is malformed, fail closed and make the failure visible.

This constraint applies to:

- public API request shapes
- CLI command behavior
- UI mode workspace behavior
- backend phase lifecycle
- prompt rendering
- AgentManager profile policy
- round payload schemas
- event payload schemas
- tests and docs

## 4. Problem Statement

The current operating-mode implementation is functionally broad but still has several debt points that can cause misleading state or make future modes harder to add:

1. `StartPhase` reserves a round before acquiring the initiative lock. A lock-acquire failure can leave a stale `reserved` round that blocks future starts.
2. A phase can be recorded as `completed` even when the final agent output has no parseable `operating_mode_result`, missing required artifacts, missing progress, or missing review verdict.
3. The registry declares methodology, but there is no full self-validation proving phase graphs, artifacts, profile keys, prompt catalog IDs, and policy fields are internally coherent.
4. Round action endpoints can default to `item-level` when `mode` is omitted, making them easy to misuse even though current UI and CLI callers pass mode.
5. `RoundCard` still owns payload parsing, proposal selection, action rendering, and display concerns in one component.
6. Backend round payloads are still mostly `map[string]any`, which spreads schema knowledge across state transitions, stats, UI normalization, and tests.
7. Route-level operating-mode wiring logs and degrades when some dependencies are unavailable, which can hide a broken feature behind partially registered routes.

The implementation must turn these into explicit boundaries and invariants, not just patch isolated bugs.

## 5. Scope

### In Scope

- Temporal lifecycle refactor for phase start and round terminal handling.
- Required output contract enforcement for every registered phase.
- Registry self-validation and startup validation.
- API hardening for mode-scoped round actions.
- Typed round payload/result helpers to reduce `map[string]any` drift.
- UI round workspace decomposition and typed backlog-sync view model.
- Route wiring fail-closed behavior for operating-mode dependencies.
- Backend, UI, CLI, stats, docs, and manual validation updates.

### Out of Scope

- Adding new operating modes.
- Dynamic mode plugins.
- Runtime operator profile editing.
- Automatic mode recommendation.
- Parallel worker fanout.
- Cross-initiative execution.
- Cost-budget enforcement.
- Compatibility with older operating-mode request shapes.
- Preserving generic fallback prompts or generic mode mutation paths.

## 6. Target End State

After this plan lands:

- Phase start is transactional from the operator's perspective: validation, locking, round reservation, prompt rendering, spawn, lock run-ID swap, and failure cleanup have one owner and one tested lifecycle.
- No stale `reserved` round can be created by a lock conflict, prompt error, spawn error, or run-ID lock swap failure.
- Every completed round satisfies the phase's registered output contract.
- Missing structured result envelopes, missing required artifacts, missing progress decisions, and missing review verdicts fail the round instead of producing misleading completion events.
- The registry validates itself at startup and in unit tests.
- Round actions are scoped by current initiative mode or by an explicit non-default route mode, never by silent `item-level` fallback.
- UI proposal/backlog-sync parsing lives in typed utilities or view-model builders, not inside `RoundCard`.
- Route wiring either fully registers operating-mode behavior or fails loudly during startup.
- Docs describe implemented invariants and link to code owners.

## 7. Architecture Decisions

### Decision A - Phase Lifecycle Has One Owner

Create a clear lifecycle owner inside `api/internal/operatingmode/`, likely `phase_lifecycle.go` or a refactored `phase_runner.go`.

It owns:

- load initiative and definition
- compute phase actions
- acquire/release initiative lock
- reserve/save rounds
- render prompt
- spawn AgentManager
- swap provisional lock holder to real run ID
- fail/cancel cleanup
- phase-start event emission

Handlers and adapters must not assemble this lifecycle.

### Decision B - Completion Requires a Phase Contract

The registry already declares phase artifacts and review criteria. Extend it so each phase has an explicit output contract.

Example shape:

```go
type PhaseOutputContract struct {
    RequiresStructuredResult bool
    RequiredArtifacts        []ArtifactDefinition
    RequiresProgress         bool
    RequiresVerdict          bool
    AllowsBacklogSync        bool
    AllowsReplanSignal       bool
    RequiresHandoff          bool
}
```

This contract should be derived from or stored beside `PhaseDefinition`, then enforced in one validator before `RoundStatusCompleted` is persisted or `operating_mode.phase_completed` is emitted.

### Decision C - Registry Validates Methodology

Add a registry validation boundary. Do not rely on scattered tests that happen to cover current modes.

Validate:

- every mode has label, scope, run strategy, metrics policy, UI policy
- non-item modes have start phase, phase map, transitions, terminal phases, artifact root, round root, lock policy
- transitions reference registered phases
- terminal phases are registered
- `PhaseDefinition.Phase` matches the map key
- `ProfilePolicy.PhaseProfiles` matches phase definitions
- every profile key is scenario-owned under `swarm-manager/`
- every artifact path stays under the mode root
- every non-item phase has catalog ID and skill ID
- prompt catalog entries match registry mode/phase definitions
- output contracts are satisfiable

### Decision D - Round Payloads Need Typed Access

Keep JSON persistence flexible, but stop spreading raw payload field names throughout the code.

Add typed helpers such as:

- `RoundPayloadView`
- `SetAgentSummary`
- `SetFinishedAt`
- `SetPromptFailure`
- `SetSpawnFailure`
- `SetPhaseResult`
- `ProgressFromRound`
- `BacklogSyncPlanFromRound`
- `VerdictFromRound`
- `ReplanSignalFromRound`

The stored JSON can remain compatible with the current format because this is not a migration problem; the goal is to make new code stop using raw maps.

### Decision E - UI Uses View Models

The UI service normalizes wire shape. Components render view models.

Move backlog-sync extraction from `round-card.tsx` into a utility with tests:

- pending completed items
- pending proposal mutations
- applied sync result
- available round actions
- selected mutation defaults

`RoundCard` should become mostly presentation and callbacks.

## 8. Implementation Strategy

### Phase 0 - Baseline Tests for Current Failure Modes

Add failing tests first.

Backend tests:

- Lock conflict during `StartPhase` leaves no active reserved round.
- Prompt render failure saves a failed round and releases lock.
- Spawn failure saves a failed round and releases lock.
- Run-ID lock swap failure fails or cancels the round instead of returning a running round with an unowned lock.
- Completed run with no structured result fails required-output phases.
- `holistic-loop/investigate` requires `findings.md`.
- `holistic-loop/plan` requires `initiative-plan.md`.
- `holistic-loop/review` requires a verdict and acceptance criteria.
- `phased-plan-drain/classify_progress` requires a valid progress decision and writes `progress.json`.
- `phased-plan-drain/execute_next` requires a handoff or explicit progress-relevant result.
- `complete-items` without explicit/current non-default mode does not silently resolve to `item-level`.
- registry validation fails on invalid transition, artifact outside root, non-owned profile key, missing prompt skill, and profile mismatch.

UI tests:

- RoundCard does not show complete/apply actions after sync is applied.
- Proposal mutation parsing handles snake_case and camelCase from persisted payloads.
- Phase buttons stay disabled based only on backend `startable=false`.
- Missing run ID disables sync actions and surfaces a reason.

### Phase 1 - Transactional Phase Start Lifecycle

Files:

- `api/internal/operatingmode/phase_runner.go`
- `api/internal/operatingmode/round_refresher.go`
- `api/internal/operatingmode/rounds.go`
- new `api/internal/operatingmode/phase_lifecycle.go` if useful
- tests in `api/internal/operatingmode/`

Tasks:

1. Reorder lifecycle so the lock is acquired before round reservation, or guarantee cleanup if reservation must happen first.
2. If lock acquisition fails, do not leave a new active round.
3. If prompt rendering fails, persist a failed audit round only after lock state is clean and future phases are not blocked.
4. If spawn fails, persist failed round and release the provisional lock.
5. If the lock holder cannot be swapped from provisional run ID to real run ID, treat the start as failed or immediately cancel/stop the run. Do not return `agent_running` with a lock that no longer represents the run.
6. Make failure cleanup idempotent and covered by tests.
7. Use one helper for terminal failure persistence so prompt, spawn, lock, and refresh failures behave consistently.

Acceptance criteria:

- No failed start path leaves `reserved` or `agent_running` unless an AgentManager run truly exists and the lock points at that run.
- All lock conflict and spawn failure tests pass.

### Phase 2 - Phase Result Contract Enforcement

Files:

- `api/internal/operatingmode/registry.go`
- `api/internal/operatingmode/output.go`
- `api/internal/operatingmode/artifact_applier.go`
- `api/internal/operatingmode/round_refresher.go`
- new `api/internal/operatingmode/output_contract.go`
- tests

Tasks:

1. Add `PhaseOutputContract` to `PhaseDefinition` or derive it in a registry helper.
2. Make `ParsePhaseResult` distinguish:
   - no structured result
   - malformed structured result
   - valid structured result with no meaningful content
   - valid structured result with content
3. Apply artifacts to a staged in-memory representation first where possible.
4. Validate required artifacts, progress, verdict, replan signal permissions, and handoff requirements.
5. Only mark `RoundStatusCompleted` after validation succeeds.
6. If validation fails, mark the round failed, persist agent summary and parse error, release lock, and emit `operating_mode.phase_failed`.
7. Keep `operating_mode.phase_completed` reserved for contract-satisfying completions.

Acceptance criteria:

- A completed round always satisfies its registered phase contract.
- Stats cannot count malformed phases as successful completions.

### Phase 3 - Registry Self-Validation

Files:

- `api/internal/operatingmode/registry.go`
- `api/internal/promptcatalog/catalog.go`
- server startup in `api/main.go`
- tests

Tasks:

1. Add `ValidateRegistry()` in `operatingmode`.
2. Add `ValidatePromptCatalog(resolve func(mode, phase string) (PromptCatalogEntry, bool))` or keep prompt-catalog validation in a test helper to avoid a package cycle.
3. Call registry validation at startup before constructing services.
4. Keep profile-key validation as part of registry validation and AgentManager reconciliation.
5. Add table-driven tests that mutate a copied registry/definition rather than global state where possible.
6. Remove `MustDefinition` usage in runtime code if it can panic on corrupted state; return errors instead.

Acceptance criteria:

- Invalid registry state fails unit tests and startup.
- New modes cannot be added without satisfying registry contracts.

### Phase 4 - API Route Hardening

Files:

- `api/internal/operatingmode/handler.go`
- `api/routes_operating_mode.go`
- `api/internal/operatingmode/service.go`
- CLI and UI services
- docs/reference

Tasks:

1. Stop defaulting round actions to `item-level`.
2. Choose one route contract:
   - preferred: load current initiative mode for round actions when mode is omitted, then require it to be non-default
   - stricter: require `mode` query/body field for every round action and reject blank
3. Use the same rule for `refresh`, `cancel`, `complete-items`, and `apply-backlog-sync`.
4. Return structured 400s for missing mode and structured 409s for lifecycle conflicts.
5. Make `registerOperatingModeRoutes` fail closed when required dependencies are absent. For greenfield, prefer fatal startup failure over silently missing routes when the scenario config declares required dependencies.
6. Fix `operatingModeUpdater.UpdateInitiativeMode` so nil service is a hard error, not a nil snapshot success.

Acceptance criteria:

- API callers cannot accidentally operate on `item-level` rounds through non-default round endpoints.
- Broken route wiring is visible at startup.

### Phase 5 - Typed Payload and Event Boundary

Files:

- `api/internal/operatingmode/rounds.go`
- `api/internal/operatingmode/events.go`
- `api/internal/operatingmode/backlog_reconciler.go`
- `api/internal/stats/engine.go`
- tests

Tasks:

1. Introduce typed payload accessors and use them in operatingmode code.
2. Replace direct `round.Payload["..."]` reads/writes in runtime code with helpers.
3. Keep tests allowed to construct payload maps only through builders.
4. Add event payload builders so phase/backlog events cannot omit required fields silently.
5. Add tests for event payload completeness:
   - mode
   - scope kind
   - scope ID
   - initiative name
   - phase
   - run strategy
   - profile key
   - round
   - run ID where required

Acceptance criteria:

- Payload field names are centralized.
- Event source metadata is complete for every operating-mode event.

### Phase 6 - UI Workspace Decomposition

Files:

- `ui/src/components/initiative/operating-mode/round-card.tsx`
- new `ui/src/components/initiative/operating-mode/backlog-sync-actions.tsx`
- new `ui/src/components/initiative/operating-mode/round-view-model.ts`
- `ui/src/services/initiative-mode-service.ts`
- `ui/src/types/operating-mode.ts`
- tests

Tasks:

1. Extract round payload parsing into `round-view-model.ts`.
2. Extract proposal mutation selection and apply UI into `BacklogSyncActions`.
3. Keep `RoundCard` to layout, status, summary, handoffs, timestamps, and action slots.
4. Add utilities for:
   - active status
   - has applied sync
   - pending completion refs
   - pending proposal
   - selected mutation defaults
5. Add tests for the view model independent of React rendering.
6. Keep the UI controlled by backend phase `startable/reason/next`; do not infer phase order client-side.

Acceptance criteria:

- `round-card.tsx` is presentation-oriented and materially smaller.
- Payload parsing is tested without rendering.

### Phase 7 - Documentation and Seam Updates

Files:

- `scenarios/swarm-manager/docs/internal/SEAMS.md`
- `scenarios/swarm-manager/docs/internal/INVARIANTS.md`
- `scenarios/swarm-manager/docs/internal/TEMPORAL-FLOWS.md`
- `scenarios/swarm-manager/docs/concepts/EXECUTION-MODES.md`
- `scenarios/swarm-manager/docs/reference/api-endpoints.md`
- `scenarios/swarm-manager/docs/reference/cli-commands.md`
- `scenarios/swarm-manager/docs/manifest.json`

Tasks:

1. Document phase lifecycle invariants.
2. Document phase output contracts.
3. Document registry validation.
4. Document route mode requirements.
5. Document typed payload helpers and event source metadata.
6. Add or update `[CODE: ...]` references for the new boundaries.
7. Register any new internal docs in the manifest if the manifest includes internal docs.

Acceptance criteria:

- Docs describe actual code, not intended future state.
- Future agents can find the lifecycle and output-contract boundaries quickly.

## 9. Testing Plan

Run targeted backend tests after each backend phase:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode ./internal/initiatives ./internal/promptcatalog ./internal/agentmanager ./internal/eventlog ./internal/stats -count=1 -timeout 300s
```

Run compile-only API coverage after route wiring changes:

```bash
cd scenarios/swarm-manager/api
go test ./... -run '^$' -timeout 300s
```

Run UI tests after each UI phase:

```bash
cd scenarios/swarm-manager/ui
npm test -- initiative-mode operating-mode-panel StatsPanel stats-service
npm run type-check
```

Run CLI tests after API contract changes:

```bash
cd scenarios/swarm-manager/cli
go test ./... -timeout 300s
```

Run full scenario validation after dependencies are healthy:

```bash
cd scenarios/swarm-manager
make test
```

Then cross-scenario validation:

```bash
vrooli scenario test swarm-manager
vrooli scenario test prompt-manager
vrooli scenario test agent-manager
```

Known current blocker from the previous hardening pass:

- `make test` did not reach Swarm Manager tests because prompt-manager setup failed during UI build with exit code 143. Resolve or rerun after dependency setup is stable.

## 10. Manual Validation Checklist

1. Create a new initiative and verify mode is `item-level`.
2. Confirm generic initiative PATCH cannot change mode.
3. Switch to `holistic-loop`.
4. Force a lock conflict while starting `investigate`; verify no stale reserved round remains.
5. Start `investigate` with prompt-manager unavailable; verify failed round or clean failure and no AgentManager run.
6. Complete `investigate` with no structured result; verify round fails.
7. Complete `investigate` with structured result but no `findings.md`; verify round fails.
8. Complete valid `investigate`; verify only `plan` is next.
9. Complete valid `plan`; verify `execute` is next.
10. Complete `execute` with `replan_needed=true`; verify next phase follows registry policy.
11. Complete `review` with no verdict; verify round fails.
12. Switch to `phased-plan-drain`.
13. Verify `execute_next` cannot start before `prepare_plan`.
14. Complete `classify_progress` with `blocked`; verify no next phase is startable.
15. Complete `classify_progress` with `complete`; verify `review` is startable.
16. Mark member items complete through `complete-items`; inspect event metadata for mode, phase, round, run ID, requested-by, and item refs.
17. Apply a backlog proposal through `apply-backlog-sync`; inspect proposal source metadata.
18. Confirm stats replan and acceptance rates match event counts.
19. Confirm UI round actions match backend state and do not show stale actions after sync.

## 11. Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Lifecycle refactor accidentally changes successful phase start behavior | Add baseline happy-path tests before changing lifecycle order |
| Strict output contracts reject useful human-readable agent summaries | Prompt skills already require structured envelopes; fail closed and improve prompts rather than accepting ambiguous summaries |
| Registry validation creates package cycles with promptcatalog | Keep prompt catalog validation in promptcatalog tests or pass a resolver function into validation |
| Route hardening breaks UI/CLI callers | Update callers in the same phase and run service/CLI/UI tests |
| Typed payload helpers become a second schema layer | Keep helpers thin and directly aligned with persisted JSON names |
| Scenario-level tests remain blocked by dependency setup | Validate targeted API/UI/CLI first, then rerun lifecycle once prompt-manager setup is stable |

## 12. Non-Goals and Prohibited Patterns

Do not:

- keep fallback prompt prose for operating-mode phases
- keep generic initiative mode mutation
- add legacy route aliases
- add compatibility fields for older clients
- let UI infer phase sequencing
- let completed rounds violate phase contracts
- silently register partial operating-mode routes
- spread new raw `map[string]any` payload access across runtime code
- add a generic catch-all `utils` package for domain logic
- directly edit backlog specs from operating-mode agents

## 13. Definition of Done

This plan is complete when:

- All stale-round lifecycle paths are fixed and tested.
- Completed rounds always satisfy phase output contracts.
- Registry validation runs in tests and startup.
- Round action APIs cannot silently default to `item-level`.
- Operating-mode route wiring fails closed when required dependencies are absent.
- Runtime operatingmode code uses typed payload helpers for known fields.
- UI round parsing is extracted and tested separately from rendering.
- Docs and internal seam/invariant notes match the implemented architecture.
- Targeted backend, UI, and CLI tests pass.
- Full `make test` or an explicitly documented dependency-blocked rerun attempt is recorded.

## 14. Implementation Log

### 2026-04-30 Backend Phase Start

Completed in this pass:

- Phase 1 partial: `StartPhase` now persists failed terminal audit rounds for lock-acquire, prompt-render, spawn, and lock run-ID swap failures instead of leaving active `reserved`/`agent_running` rounds behind. Lock run-ID swap failure now stops the spawned run when an agent client is available and returns an error instead of reporting a running round with an unowned lock.
- Phase 2 partial: `PhaseOutputContract` is now part of `PhaseDefinition`. All initiative-level phases require a structured `operating_mode_result`; required phase artifacts are enforced; review phases require a verdict; `phased-plan-drain/classify_progress` requires a valid progress decision; `phased-plan-drain/execute_next` requires a durable handoff.
- Phase 2 partial: `RefreshRound` now marks a completed AgentManager run as `failed` when result parsing/application/contract validation fails. `operating_mode.phase_completed` is emitted only for contract-satisfying completions.
- Phase 3 partial: `ValidateRegistry()` validates core registry coherence and runs during API startup. Tests mutate cloned registry definitions to prove invalid transitions, artifact paths outside the mode root, non-owned profile keys, missing prompt skill IDs, and profile mismatches fail closed.

Validation run:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode -count=1 -timeout 300s
go test ./internal/operatingmode ./internal/initiatives ./internal/promptcatalog ./internal/agentmanager ./internal/eventlog ./internal/stats -count=1 -timeout 300s
go test ./... -run '^$' -timeout 300s
```

Recommended next pass:

- Finish Phase 1 coverage for run-ID lock swap failure with an injectable lock seam or lock adapter test double.
- Finish Phase 2 by making result parsing expose explicit parse states and by staging artifact writes before contract validation.
- Finish Phase 3 prompt catalog validation without introducing package cycles.
- Proceed to Phase 4 API route hardening once backend lifecycle and registry validation are complete.

### 2026-04-30 API Route Hardening

Completed in this pass:

- Phase 4 partial: round action mode resolution now has one service boundary. HTTP handlers may infer mode from the current initiative only when that initiative is already in a non-default operating mode. Blank explicit mode and `item-level` both fail closed instead of silently resolving to the default item flow.
- Phase 4 partial: `RefreshRound`, `CancelRound`, `CompleteItems`, and `ApplyBacklogSync` now reject blank or `item-level` mode even when called directly from Go code instead of through HTTP handlers.
- Phase 4 partial: operating-mode route registration now fails startup when required initiative/backlog services or the operating-mode service constructor are unavailable, instead of silently leaving the feature partially unregistered.
- Phase 4 partial: `operatingModeUpdater.UpdateInitiativeMode` now returns a hard error when the initiative service is missing, rather than returning an empty success snapshot.
- Phase 3 partial: `ValidatePromptCatalog()` now validates every non-item phase's catalog ID and skill ID through an injected resolver, and `NewService` fails construction if the resolver is missing or mismatched. This keeps the validation boundary in `operatingmode` without importing `promptcatalog`.
- Added HTTP/service regression tests proving omitted mode does not operate on `item-level`, omitted mode can resolve from a current non-default initiative, and direct service round actions require non-default mode.
- Added prompt-catalog validation tests for nil resolver, missing entry, catalog ID mismatch, and skill mismatch.

Validation run:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode -count=1 -timeout 300s
go test ./internal/operatingmode ./internal/initiatives ./internal/promptcatalog ./internal/agentmanager ./internal/eventlog ./internal/stats -count=1 -timeout 300s
go test ./... -run '^$' -timeout 300s

cd ../cli
go test ./... -timeout 300s
```

Recommended next pass:

- Finish Phase 2 parse-state/staged-artifact work so malformed structured output and invalid artifact writes cannot partially mutate repository state before contract validation completes.
- Start Phase 5 typed payload helpers for known round fields, then replace runtime raw-map reads/writes incrementally.
- Add structured error codes for round-action 400 responses if API clients need machine-readable mode-contract failures.

### 2026-04-30 Phase Result and Payload Boundary

Completed in this pass:

- Phase 2 partial: `ParsePhaseResultDetailed()` now exposes explicit parse states for no output, no structured result, malformed structured result, empty structured result, and valid structured result. The legacy boolean parser remains as a thin compatibility wrapper inside the package while lifecycle code consumes the explicit state.
- Phase 2 partial: `applyPhaseResult()` now stages payload changes, handoffs, readiness, progress metadata, and artifact updates before mutating the real round. Artifact paths are validated and required-output contracts are checked before any artifact file is written.
- Phase 2 partial: malformed or empty structured phase output now fails with a specific contract error instead of being flattened into a generic missing-result path.
- Phase 5 partial: added `RoundPayloadView` in `payload.go` and moved known runtime payload reads/writes for agent summary, finish/cancel timestamps, verdict, replan, progress, backlog sync, backlog sync plan, and applied-at audit fields behind typed accessors.
- Phase 5 partial: operating-mode runtime code now uses the payload boundary in refresh/cancel, phase-result application, phase sequencing, phase/backlog event emission, and backlog reconciliation.
- Phase 5 partial: backlog-sync event payload construction is now a named builder, matching the existing phase payload builder seam.
- Added focused tests for parse-state classification, artifact staging, phase event metadata completeness, and backlog-sync event metadata/source completeness. A phase result that contains one valid required artifact and one invalid artifact path now fails without writing the valid artifact first.

Validation run:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode -count=1 -timeout 300s
go test ./internal/operatingmode ./internal/initiatives ./internal/promptcatalog ./internal/agentmanager ./internal/eventlog ./internal/stats -count=1 -timeout 300s
go test ./... -run '^$' -timeout 300s
```

Recommended next pass:

- Finish Phase 5 by moving remaining test setup to payload builders where it improves readability and by checking stats/event consumers for any duplicated payload assumptions outside `operatingmode`.
- Move into Phase 6 UI decomposition: extract `round-view-model.ts`, move backlog-sync parsing/action derivation out of `round-card.tsx`, and add utility tests independent of React rendering.
- Update internal docs in Phase 7 after the UI decomposition lands so the documented seams match both backend and frontend boundaries.

### 2026-04-30 UI Round View Model and Docs

Completed in this pass:

- Phase 6 partial: extracted operating-mode round payload parsing, pending/applied backlog-sync decisions, default mutation selection, mutation summaries, and action availability into `ui/src/components/initiative/operating-mode/round-view-model.ts`.
- Phase 6 partial: extracted proposal mutation selection and apply controls into `ui/src/components/initiative/operating-mode/backlog-sync-actions.tsx`, leaving `round-card.tsx` focused on round presentation, status controls, handoffs, timestamps, and delegated action slots.
- Phase 6 partial: added non-React view-model tests covering snake_case and camelCase completed-item plans, applied-sync hiding, malformed proposal mutation filtering, missing run ID action blocking, default selection, and mutation summaries.
- Phase 7 partial: updated `scenarios/swarm-manager/docs/internal/SEAMS.md`, `INVARIANTS.md`, and `TEMPORAL-FLOWS.md` to document the backend lifecycle invariants and the new UI view-model/action seams.

Validation run:

```bash
cd scenarios/swarm-manager/ui
npm test -- operating-mode-panel round-view-model -- --runInBand
npm run type-check
npm test -- initiative-mode operating-mode-panel StatsPanel stats-service round-view-model -- --runInBand

cd ../api
go test ./internal/operatingmode ./internal/initiatives ./internal/promptcatalog ./internal/agentmanager ./internal/eventlog ./internal/stats -count=1 -timeout 300s
```

Recommended next pass:

- Finish Phase 7 reference docs: update `concepts/EXECUTION-MODES.md`, `reference/api-endpoints.md`, and `reference/cli-commands.md` so operator-facing docs include the strict round-action mode requirement and phase-output contract behavior.
- Run the backend targeted suite again after any docs-adjacent code cleanup to make sure the earlier lifecycle and contract work still passes in the combined worktree.
- Consider the remaining Phase 1 run-ID lock-swap test seam if it has not been implemented by the prior backend agents.

### 2026-04-30 Lock Seam and Reference Docs

Completed in this pass:

- Phase 1 completion: `operatingmode.Service` now depends on a narrow initiative-lock interface instead of the concrete file-lock type, allowing lifecycle tests to inject lock failures at exact control points without weakening production behavior.
- Phase 1 completion: added a run-ID lock swap regression test proving a spawned run is stopped, the round is persisted as failed with audit context, and the initiative lock is released when the provisional lock cannot be swapped to the real AgentManager run ID.
- Phase 7 partial: updated `reference/api-endpoints.md`, `reference/cli-commands.md`, `concepts/EXECUTION-MODES.md`, and `internal/SEAMS.md` to document strict non-default round-control mode requirements, phase-output contract behavior, the fail-closed phase start lifecycle, and the injectable lock seam.

Validation run:

```bash
cd scenarios/swarm-manager/api
go test ./internal/operatingmode -count=1 -timeout 300s
go test ./internal/operatingmode ./internal/initiatives ./internal/promptcatalog ./internal/agentmanager ./internal/eventlog ./internal/stats -count=1 -timeout 300s
go test ./... -run '^$' -timeout 300s

cd ../cli
go test ./... -timeout 300s

cd ../ui
npm test -- initiative-mode operating-mode-panel StatsPanel stats-service round-view-model -- --runInBand
npm run type-check
```

Recommended next pass:

- Run the broader backend, CLI, and UI targeted suites in this combined worktree.
- Finish any remaining Phase 7 reference cleanup by checking `docs/reference/api-endpoints.md`, `docs/reference/cli-commands.md`, `docs/concepts/EXECUTION-MODES.md`, and `docs/manifest.json` with documentation-health tooling if available.
- Attempt full `cd scenarios/swarm-manager && make test` once cross-scenario dependency setup is stable.

### 2026-04-30 Full Scenario Validation

Completed in this pass:

- Fixed the scenario startup blocker found by full lifecycle validation: the manifest-owned default AgentManager profile now uses the non-colliding name `swarm-manager-default` while keeping the stable key `swarm-manager/default`.
- Fixed isolated API tests so `newTestServer` disables external AgentManager calls, preventing temp-root tests from depending on a live scenario manifest.
- Hardened initiative-review degraded mode so the shared feedback/review initiative lock is still respected before no-spawner review rounds are created, then released because no live run owns it.
- Fixed Phase 7 docs links in `concepts/EXECUTION-MODES.md` so scenario docs correctly point to repo-level plan files.
- Cleared full-suite lint failures by removing unused backend code, fixing operating-mode UI lint errors, and extracting feedback-dialog envelope helpers into `feedback-dialog-envelope.ts` so the component file exports only React component/type surface.

Validation run:

```bash
cd scenarios/swarm-manager/api
golangci-lint run ./...
go test ./... -timeout 300s

cd ../ui
pnpm exec eslint .
npm run type-check
npm test -- feedback-dialog -- --runInBand

cd scenarios/swarm-manager
make test

vrooli scenario test swarm-manager
```

Result:

- `make test` passed.
- `vrooli scenario test swarm-manager` passed.

Residual warnings observed in full scenario validation:

- `phase-structure`: registry may be stale, seed directory informational warning, playbook informational warnings.
- `phase-standards`: dangerous TypeScript pattern warning, incomplete Makefile quality target warning, missing test file warning for `api/internal/dispatch`.
- `phase-business`: `REQ-P0-009-DESKTOP` references missing `ui/src/components/layout/MainLayout.test.tsx`.
- `phase-performance`: Lighthouse home performance was 78% versus 85% threshold, but performance phase still passed.
