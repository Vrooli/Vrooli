# Seams — Ecosystem Manager

A **seam** is a deliberate boundary where production code calls an
interface, not a concrete dependency. The fake substitutes through that
interface in tests; production wires the real implementation once at the
composition edge (`api/cmd/.../main.go` → `internal/server`, or a domain
constructor).

This document is the authoritative list of seams in ecosystem-manager.
Add to it whenever you introduce a new interface that production wires
once and tests substitute. Remove from it only when the seam is
genuinely gone — not when "we don't fake it yet."

> **Transport note.** Ecosystem-manager is mid-migration from a Go REST/JSON
> API over `gorilla/mux` ([CODE: api/pkg/server/server.go]) to proto-first
> Connect-RPC. The **discovery** domain is the first migrated vertical (the
> reference — [MIGRATION-GUIDE.md](MIGRATION-GUIDE.md)): its wire contract is
> the generated `DiscoveryService` and its domain seam is the
> `internal/discovery.Service` boundary below. The remaining domains still
> serve REST routes; their seams are plain Go interfaces with compile-time
> assertions (`var _ Iface = (*impl)(nil)`). Both transports share one mux —
> see [COHERENCE-NOTES.md](COHERENCE-NOTES.md) and
> [../internal/DECISIONS.md](../internal/DECISIONS.md).

## How to read this file

| Column | Meaning |
|---|---|
| **Seam** | Short name used to refer to the boundary in conversation. |
| **Interface** | Go file & symbol that defines the contract. |
| **Production wiring** | Where the real implementation is constructed. |
| **Test fake** | The in-tree fake that substitutes through the interface in tests. |
| **Why it exists** | The class of bug it prevents or the test ergonomic it enables. |

The core domain is **auto-steer** — the autonomous improvement loop,
modeled as a closed-loop controller in
[../concepts/CONTROL-MODEL.md](../concepts/CONTROL-MODEL.md). Almost all
seams below exist to make that loop deterministically testable without
running real agents. The single highest-value seam is **MetricsProvider**:
it is the controller's *sensor*, and faking it is what turns
non-deterministic loop behavior into table-driven unit tests.

## Current seams

### ProfileRepository (steering-profile persistence)

| | |
|---|---|
| **Seam** | Auto-steer profile persistence (CRUD over profiles) |
| **Interface** | [CODE: api/pkg/autosteer/repositories.go]::`ProfileRepository` (`GetProfile`, `CreateProfile`, `UpdateProfile`, `DeleteProfile`, `ListProfiles`) |
| **Production wiring** | `FileProfileRepository` ([CODE: api/pkg/autosteer/profile_repository_fs.go]::`NewFileProfileRepository(rootDir)`) — filesystem-backed YAML under `profiles/`. Asserted with `var _ ProfileRepository = (*FileProfileRepository)(nil)`. |
| **Test fake** | In-memory mock in [CODE: api/pkg/autosteer/repositories_mock.go]. The filesystem impl has its own round-trip test ([CODE: api/pkg/autosteer/profile_repository_fs_test.go]) that pins YAML read/write semantics. |
| **Why it exists** | Profile storage is a YAML-on-disk detail today; the orchestrator depends on the interface, so a hypothetical store swap (e.g. remote service) wouldn't ripple into the control loop. Tests that exercise phase/profile lookup don't touch the filesystem. |

### ExecutionStateRepository (per-task loop state)

| | |
|---|---|
| **Seam** | Persistence + state manipulation for `ProfileExecutionState` (iteration counters, current phase, metrics snapshot) |
| **Interface** | [CODE: api/pkg/autosteer/repositories.go]::`ExecutionStateRepository` (`Get`, `Save`, `Delete`, plus state helpers `InitializeState`, `IncrementIteration`, `AdvanceToNextPhase`, `RecordPhaseCompletion`, `FinalizeExecution`) |
| **Production wiring** | `ExecutionStateManager` over SQLite ([CODE: api/pkg/autosteer/execution_state_manager.go], table `profile_execution_state` from [CODE: api/pkg/autosteer/schema.sql]). Asserted with `var _ ExecutionStateRepository = (*ExecutionStateManager)(nil)`. |
| **Test fake** | `MockExecutionStateRepository` (in-memory) in [CODE: api/pkg/autosteer/repositories_mock.go]. |
| **Why it exists** | The loop's controller state (where are we in the phase, how many iterations) is the thing the orchestrator reads and mutates every tick. Faking it in-memory lets `execution_orchestrator_unit_test.go` drive phase advancement, requeue-vs-stop, and finalization without a database. |

### ExecutionHistoryRepository (completed-run analytics)

| | |
|---|---|
| **Seam** | Persistence of completed profile executions + analytics |
| **Interface** | [CODE: api/pkg/autosteer/repositories.go]::`ExecutionHistoryRepository` (`GetHistory`, `GetExecution`, `GetProfileAnalytics`) |
| **Production wiring** | `HistoryService` over SQLite ([CODE: api/pkg/autosteer/history_service.go], table `profile_executions`). Asserted with `var _ ExecutionHistoryRepository = (*HistoryService)(nil)`. |
| **Test fake** | Substituted via the in-memory mock family in [CODE: api/pkg/autosteer/repositories_mock.go]; history handler tests use it. |
| **Why it exists** | Separates the write-then-analyze surface (history, analytics) from live loop state. Replaying history in a test doesn't perturb the running-execution table. |

### Score (the controller's sensor) — highest-value seam

| | |
|---|---|
| **Seam** | Completeness measurement — the sensor the closed loop reads each iteration |
| **Interface** | [CODE: api/pkg/completeness/score.go]::`Provider` (`Score(ctx, scenario) (Score, error)`) |
| **Production wiring** | `Client` ([CODE: api/pkg/completeness/client.go]) — a Connect-RPC client over scenario-completeness-scoring's `GetScore`, resolved per call via api-core discovery. Does NOT fail open (measurement is load-bearing for termination — plan D2). |
| **Test fake** | A caller-supplied `Provider` returning a synthetic `Score`; the contract test points the client at an in-process handler. |
| **Why it exists** | Real measurement is a cross-scenario RPC — slow, non-deterministic, side-effecting. Injecting a synthetic `Score` drives the loop's termination (objective-met / diminishing-returns / budget) deterministically in unit tests. |

### ConditionEvaluatorAPI / PhaseCoordinatorAPI / IterationEvaluatorAPI / PromptEnhancerAPI (pure-logic components)

| | |
|---|---|
| **Seam** | The control-loop's pure-logic stages — decompose the controller so each is testable in isolation |
| **Interface** | [CODE: api/pkg/autosteer/interfaces.go]::`ConditionEvaluatorAPI`, `PhaseCoordinatorAPI`, `IterationEvaluatorAPI`, `PromptEnhancerAPI` |
| **Production wiring** | Concrete `ConditionEvaluator` ([CODE: api/pkg/autosteer/evaluator.go (19-78)]), `PhaseCoordinator` ([CODE: api/pkg/autosteer/phase_coordinator.go (32-71)]), `IterationEvaluator`, `PromptEnhancer`. Each carries a `var _ <API> = (*impl)(nil)` assertion. |
| **Test fake** | `MockConditionEvaluator` ([CODE: api/pkg/autosteer/phase_coordinator_unit_test.go]) substitutes the evaluator so phase-coordination tests don't depend on real metric comparison; the other three are exercised directly (stateless, no I/O) with synthetic snapshots. |
| **Why it exists** | These stages are stateless and I/O-free by contract, so the orchestrator can be tested by stubbing one stage at a time (e.g., force `ShouldStop` via `MockConditionEvaluator` while pinning the coordinator's max-iterations branch). Keeping the boundaries as interfaces is what lets the orchestrator depend on behavior, not concrete decision code. |

### Agent-manager client (the execution boundary)

| | |
|---|---|
| **Seam** | Outbound run start/stop against the agent-manager scenario — the loop's *actuator* |
| **Interface** | [CODE: api/pkg/agentmanager/api.go]::`AgentServiceAPI` (`IsAvailable`, `ExecuteTask`, `ExecuteTaskAsync`, `GetRunStatus`, `StopRun`, `GetRunEvents`, `WaitForRun`, …) |
| **Production wiring** | `AgentService` ([CODE: api/pkg/agentmanager/service.go]) wrapping the HTTP `Client` ([CODE: api/pkg/agentmanager/client.go]). |
| **Test fake** | `MockAgentService` ([CODE: api/pkg/agentmanager/mock.go]) — per-method error knobs (e.g. `StopRunError`), call counters, and `LastStopRunID`-style recorders. |
| **Why it exists** | Starting/stopping real agent runs is the one truly external, expensive, non-idempotent action in the loop. Faking it lets queue/orchestration tests prove "we asked agent-manager to stop run X" without spawning an agent, and lets failure paths (agent-manager unreachable, StopRun error) be exercised deterministically. |

### Task StorageAPI + ExecutionRegistry (queue persistence & run tracking)

| | |
|---|---|
| **Seam** | Queue task persistence and in-flight run tracking |
| **Interface** | `tasks.StorageAPI` (consumed across [CODE: api/pkg/queue/processor.go], `execution_manager.go`, `insight_manager.go`); run tracking via `ExecutionRegistry` ([CODE: api/pkg/queue/execution_registry.go]) |
| **Production wiring** | Filesystem YAML store under `queue/<status>/` plus the SQLite execution state; `NewExecutionRegistry()` holds the live `taskID → runID` map. |
| **Test fake** | In-memory storage substituted through `tasks.StorageAPI`; `ExecutionRegistry` is constructed real (it is pure in-memory state) in [CODE: api/pkg/queue/autosteer_integration_test.go]. |
| **Why it exists** | The requeue-vs-stop decision ([CODE: api/pkg/queue/autosteer_integration.go (180-227)]) is queue policy that reads task flags (`ProcessorAutoRequeue`, `AutoSteerProfileID`). Faking storage lets the integration test seed exact task shapes and assert the recycle decision without a real queue on disk. |

### Clock / time (timeouts & deadlines)

| | |
|---|---|
| **Seam** | Wall-clock time used for run timeouts and elapsed-duration accounting |
| **Interface** | `time` access funneled through the agent-manager client timeout config and orchestration deadlines (no global `time.Now()` in the decision path). |
| **Production wiring** | Real `time` / `context.WithTimeout` at the orchestration and HTTP-client edges ([CODE: api/pkg/agentmanager/client.go]). |
| **Test fake** | Tests inject short/zero timeouts and pre-set timestamps via the mock agent service and synthetic state rather than racing real clocks. |
| **Why it exists** | Timeout-driven branches (run took too long → stop) must be reachable without `time.Sleep`. Keeping time at the edge means the loop's decision logic never reads the wall clock directly. |

## Adding a new seam

The right time to add a seam is the moment you find yourself reaching
past `*sql.DB`, an HTTP call to another scenario, `os.OpenFile`, or
`time.Now()` from inside the control loop or a handler. The process is
mechanical:

1. **Define the interface beside its consumer.** Auto-steer interfaces
   live in [CODE: api/pkg/autosteer/repositories.go] (persistence/sensor
   seams) or [CODE: api/pkg/autosteer/interfaces.go] (pure-logic
   component seams). Methods are exactly what callers need.
2. **Implement it in production** with the concrete dependency
   (`*sql.DB`, filesystem, HTTP client) wrapped in a struct that
   translates domain calls to I/O.
3. **Add a `var _ Iface = (*impl)(nil)` assertion** in the same file —
   this moves "this type satisfies the interface" from a runtime
   surprise into a compile error. Follow the existing assertions in
   `repositories.go` / `interfaces.go`.
4. **Add an in-tree fake.** For auto-steer, add it to
   [CODE: api/pkg/autosteer/repositories_mock.go] (cross-component
   mocks) or beside the test (component-local mocks like
   `MockConditionEvaluator`). For agent-manager, extend
   [CODE: api/pkg/agentmanager/mock.go]. Use per-method error knobs and
   call counters.
5. **Update this document** with a row using the same five columns. A
   seam that isn't listed will be reinvented in parallel.

## What is NOT a seam

- **Pure-function helpers** — `compareValues` / `GetMetricValue` on the
  evaluator ([CODE: api/pkg/autosteer/evaluator.go (19-78)]),
  `DetermineStopReason` on the coordinator. They have no injected
  dependencies; tests call them directly. The *interface* around the
  evaluator exists so callers can stub the whole component, not because
  the helpers need a seam.
- **Domain value types** — `Score` (`pkg/completeness`), `StopCondition`,
  `ProfileExecutionState`, `PhaseAdvanceDecision`. These are data
  contracts passed through seams, not boundaries themselves.
- **`gorilla/mux` routing** — route registration in
  [CODE: api/pkg/server/server.go] is composition, not a substitutable
  dependency. Handler tests mount the real router and issue HTTP
  requests.
- **Configuration structs** read once at startup. The seam is the
  *consumer* of the config, not the loader.

## Architecture alignment notes

| Area | Drift | Decision | Follow-up |
|---|---|---|---|
| Transport | Newer scenarios use proto + Connect-RPC; ecosystem-manager predates that and is REST/JSON over mux. | Documented deviation, not drift to fix opportunistically. New endpoints stay REST/JSON for internal consistency until a deliberate migration. | See [../internal/DECISIONS.md](../internal/DECISIONS.md) and [ERROR-HANDLING.md](ERROR-HANDLING.md). |
| Loop = controller | The auto-steer loop was historically described procedurally (start → iterate → advance). | Reframed as a closed-loop controller: `MetricsProvider` is the sensor, agent-manager is the actuator, `PhaseCoordinator`/`ConditionEvaluator` are the control law, `ExecutionStateRepository` holds controller state. | Mental model lives in [../concepts/CONTROL-MODEL.md](../concepts/CONTROL-MODEL.md); keep seam responsibilities mapped to controller roles. |
| Mixed persistence | Profiles on filesystem YAML, execution state/history in SQLite. | Intentional: profiles are human-editable templates; loop state is transactional. Both sit behind their own repository seam so the split is invisible to the orchestrator. | If a store moves, change only the production impl + its round-trip test; the interface and fakes stay. |

### DiscoveryService (discovery domain — Connect-RPC reference)

| | |
|---|---|
| **Seam** | Discovery domain service (resources/scenarios/operations/categories), the reference Connect-RPC vertical |
| **Interface** | The transport edge is the generated `DiscoveryServiceHandler` ([CODE: packages/proto/schemas/ecosystem-manager/v1/discovery/discovery.proto]); the domain seam is [CODE: api/internal/discovery/service.go]::`Service` (`Resources`, `Scenarios`, `Resource`, `Scenario`, `Operations`, `ResourceCategories`, `ScenarioCategories`) plus `ToConnectError`. |
| **Production wiring** | `handlers/discovery.Module(assembler)` ([CODE: api/handlers/discovery/module.go]) builds the Connect handler over `internal/discovery.NewService` and mounts it via `connectx.RegisterServices`; registered in `server.connectModules()` ([CODE: api/pkg/server/server.go]) and `internal/modules.AllEndpoints()`/`MigratedDomains()`. |
| **Test fake** | The discovery sweep is faked at the `commandRunner` seam in [CODE: api/internal/discovery/runner.go] (`execRunner`), exercised by [CODE: api/internal/discovery/discovery_test.go]; the Connect handler maps sentinel errors (`ErrResourceNotFound`→`CodeNotFound`, `ErrEmptyName`→`CodeInvalidArgument`). |
| **Why it exists** | The copy-this-shape reference for migrating the remaining REST domains (tasks, queue, autosteer, prompts, executions) to Connect-RPC. The thin handler → `internal/<domain>` service split is also the god-object decomposition reference. |

## Cross-references

- [../concepts/CONTROL-MODEL.md](../concepts/CONTROL-MODEL.md) — auto-steer as a closed-loop controller
- [MIGRATION-GUIDE.md](MIGRATION-GUIDE.md) — how to migrate a domain to Connect-RPC (discovery worked example)
- [COHERENCE-NOTES.md](COHERENCE-NOTES.md) — REST↔Connect transport coexistence during migration
- [../concepts/ARCHITECTURE.md](../concepts/ARCHITECTURE.md) — overall scenario architecture
- [../internal/DECISIONS.md](../internal/DECISIONS.md) — REST/JSON-over-Connect deviation rationale
- [TESTING.md](TESTING.md) — control-loop testing pattern using `MockMetricsProvider`
- [ERROR-HANDLING.md](ERROR-HANDLING.md) — REST error envelope and upstream failure mapping
