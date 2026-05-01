# Architecture Seams & Internal Design

> Current State (2026-03-28): Swarm Manager runtime is graph-first backlog/scenarios/execution/agent-activity/settings/prompts with proto-backed UI↔API seams. Recommendation generation is owned by Prompt Manager teams and should not be implemented in Swarm Manager.
>
> Historical note: Some lower sections preserve pre-greenfield recommendation-era references for audit history only.

## Seam Overview

This document captures the architecture seams (integration points, boundaries) and internal implementation details for Swarm Manager. It serves as a living record of design decisions and drift from the documented mental model.

## Current Architecture State

### Alignment Assessment (2026-02-14)

| Aspect | Documented | Actual | Gap |
|--------|------------|--------|-----|
| API endpoints | /backlog, /scenarios, /execution, /settings | /backlog, /scenarios, /execution, /settings, /queue, /agent-manager/status, /health | Resolved |
| Persistence | Filesystem (ideas/, research/, fix/, execute/, .vrooli/settings.json, .vrooli/queue.json, .vrooli/execution-*.json) | Backlog/scenarios/settings/queue/execution implemented | Resolved |
| Selector registry | All UI selectors defined | ✅ Fully populated | Resolved |
| UI workspace | Graph-first primary surface with sidebar + detail routes | `/graph` is primary operator route; detail pages remain for drill-down/edit flows | Resolved |
| Integration clients | agent-manager, ecosystem-manager | Discovery-based agent-manager + ecosystem-manager clients | ✅ Resolved |
| Domain types | Shared across UI | ✅ Centralized in types/ module | Resolved |

### Logical vs Physical Gaps

1. **API Layer Gap** (RESOLVED)
   - Expected: Domain-organized handlers (backlog/, scenarios/, execution/, settings/)
   - Actual: Backlog, scenarios, settings, queue, execution, and agent-manager status handlers implemented
   - Impact: Backlog UI now fully functional across kinds

2. ~~**Selector Registry Gap**~~ (RESOLVED)
   - ~~Expected: `literalSelectors` and `dynamicSelectorDefinitions` populated~~
   - ~~Actual: Both objects are empty `{}`~~
   - Status: Fully populated in Phase 1 (Architecture Alignment)

3. **Business Logic Gap** (RESOLVED)
   - Expected: Service layer for backlog CRUD, scenario operations, settings/execution
   - Actual: Backlog, scenarios, settings, execution, and execution-policy services implemented
   - Impact: UI now reads/writes all core domains

## Seam Definitions

### Test Utility Boundary

`api/internal/testutil` and `cli/internal/testutil` are test-only utility
seams. They own reusable helpers, fakes, fixtures, HTTP harnesses, and assertion
helpers that make domain tests shorter without leaking test-only dependencies
into production builds.

- **Import rule**: Only `_test.go` files may import
  `swarm-manager/internal/testutil`, `swarm-manager/cli/internal/testutil`, or
  their future subpackages.
- **Guardrail**: `api/internal/testutil/no_prod_import_test.go` walks
  production Go files under `api/internal` and fails if any production file
  imports the API test utility package. `cli/internal/testutil/no_prod_import_test.go`
  provides the same guard for CLI production files.
- **Ownership rule**: Shared fakes belong here only when they model stable
  seams used by multiple packages. Package-specific failure modes should stay
  local to the package test that needs them.
- **Growth path**: Expand this package into focused subpackages such as
  `assertx`, `fixtures`, `fsx`, `httpx`, `mocks`, and `services` as duplicate
  helpers are migrated. CLI tests should use `cli/internal/testutil` for
  `httptest.Server` setup, `SWARM_MANAGER_API_BASE` isolation, request capture,
  and command execution helpers.

### UI-to-API Seam

The UI-to-API seam has been refactored into multiple layers for better testability:

```
ui/src/
├── lib/
│   ├── api-client.ts    # HTTP infrastructure (IApiClient interface)
│   ├── api-endpoints.ts # Endpoint path constants
│   ├── error-utils.ts   # Error categorization and recovery paths
│   ├── proto-contracts.ts # Generated proto schema parsing + validation helpers
│   ├── query-utils.ts   # React Query default options
│   └── index.ts         # Barrel export
└── services/
    ├── backlog-service.ts      # Backlog CRUD and actions
    ├── agent-manager-service.ts # Agent-manager availability
    ├── agent-activity-service.ts # Tracked agent activity queries and stop actions
    ├── graph-service.ts        # Graph projection mapping
    ├── scenarios-service.ts    # Scenarios operations
    ├── settings-service.ts     # Settings persistence
    └── index.ts                # Barrel export
```

**Service Seam Pattern:**
```typescript
// Interface defines the seam
export interface IBacklogService {
  list(kinds?: BacklogKind[]): Promise<BacklogItem[]>;
  get(kind: BacklogKind, name: string): Promise<BacklogItem>;
  create(item: Omit<BacklogItem, "created" | "updated">): Promise<BacklogItem>;
  update(kind: BacklogKind, name: string, item: Partial<BacklogItem>): Promise<BacklogItem>;
  delete(kind: BacklogKind, name: string): Promise<void>;
}

// Factory allows dependency injection for testing
export function createBacklogService(
  apiClient: IApiClient = defaultApiClient
): IBacklogService { ... }

// Default instance for production use
export const backlogService = createBacklogService();
```

**Testing at the Seam:**
```typescript
// Pages mock at the service level (cleaner)
vi.mock("../services", () => ({
  backlogService: { list: vi.fn(), ... }
}));

// Services inject mock API client (explicit dependency)
const mockClient: IApiClient = { get: vi.fn(), ... };
const service = createBacklogService(mockClient);
```

**Status**: ✅ Service layer implemented. Structured UI↔API payloads are proto-backed, including the graph projection.

### Graph Projection Boundary

`api/internal/graph/` and `ui/src/services/graph-service.ts` form the graph projection seam.

- **Ingress/egress contract**: `GET /api/v1/graph?lens=X[&focus_node_id=Y]` returns proto `swarm-manager.v1.api.GraphResponse`
- **Projection params**: The `ProjectionParams` struct holds `Lens` and optional `FocusNodeID`. The handler validates focus format (must start with `backlog-item/`, `initiative/`, or `scenario/`). Cache keys are `{Lens, FocusNodeID}` tuples.
- **Projection payloads**: Graph nodes use typed oneof payloads (`backlog`, `initiative`, `capture`, `scenario`, `execution`, `activity`, `run`). Backlog nodes include cross-lens `active_execution_status` and `active_execution_count` from topology enrichment.
- **Focus-based drill-down**: Flow lens requires focus — dispatches to `buildFlowForBacklogItem`, `buildFlowForInitiative`, or `buildFlowForScenario`. Operations lens accepts optional focus for filtered view.
- **Meta contract**: Response meta includes `focus_node_id`, `focus_node_type`, and `hint` for empty states.
- **UI mapper**: `graph-service.ts` parses proto JSON through `proto-contracts.ts` and maps it into the typed graph node union used by the store/canvas/presentation helpers
- **Library seam**: React Flow still exposes node data as `Record<string, unknown>` in renderer callbacks; the only intentional UI casts are localized in the graph renderers/helpers at that library boundary

**Testing at the seam**: Go handler/projection tests validate proto JSON shape including focus routing. UI service/store/presentation tests validate typed graph mapping, clustering, and canvas rendering against the shared graph contract.

### Agent Activity Boundary

`api/internal/agentactivity/` is the canonical Swarm Manager seam for tracked AgentManager usage.

- **Purpose**: persist one durable `AgentActivity` record for every tracked `SpawnBacklog` and `ContinueRun`
- **Ownership model**: activities belong to a backlog item, scenario, or capture and may optionally link to an execution record
- **Control-plane split**: execution records remain the governed workflow object; agent activity records are the telemetry/audit object
- **Integration rule**: backlog, capture, and execution flows must route AgentManager spawn/continue calls through the tracked activity service instead of calling the raw AgentManager client directly
- **Graph impact**: flow and operations projections read activity records to show workshop/research/classify/follow-up/runtime lineage, not only governed execution runs

**Testing at the seam**: `service_test.go` covers spawn, continue, refresh, and stop transitions against a stub AgentManager seam without filesystem-global state. `handler_test.go` locks the HTTP contract for listing, filtering, and fetching tracked activities.

### Agent Identity and Provenance Boundary

Agent identity is the canonical attribution seam for API mutations performed by
Agent Manager runs.

- **Runner env contract**: Agent Manager injects `VROOLI_AGENT_IDENTITY_TOKEN`
  into spawned agent processes. It also injects
  `VROOLI_AGENT_MANAGER_API_BASE`, resolved from the current Agent Manager API
  environment, so cli-core consumers can verify tokens without caller-managed
  setup.
- **CLI forwarding rule**: Swarm Manager CLI detects
  `VROOLI_AGENT_IDENTITY_TOKEN` through cli-core and forwards it as
  `X-Agent-Identity-Token` on every API request. Domain commands must not add
  separate `--created-by`, `--run-id`, or `--session-id` flags for normal
  identity propagation.
- **API verification rule**: Swarm Manager API middleware verifies
  `X-Agent-Identity-Token` through Agent Manager's
  `/api/v1/identity/verify` endpoint and stores `identity.Provenance` in the
  request context. Verification is fail-open: missing, invalid, or unreachable
  identity verification yields explicit operator provenance, never a rejected
  mutation.
- **Mutation stamping rule**: Mutation handlers and services read provenance
  from request context and persist it as durable `created_by` metadata where the
  domain supports attribution. Single backlog create, batch backlog create, and
  initiative create are locked by regression tests. Agent Sessions extend this
  same provenance shape with session fields instead of replacing the identity
  standard.
- **Session enrichment rule**: After identity verification, Swarm Manager uses
  the server-owned Agent Session resolver to map verified `run_id` values to
  durable session ownership. If a match exists, `identity.Provenance` is
  enriched with `session_id`, `session_kind`, and `source=session/<id>`. The
  lookup is fail-open and server-derived; agents do not pass these fields in
  prompts, JSON, or CLI flags.
- **Session artifact rule**: When enriched provenance includes `session_id`,
  backend mutation chokepoints record durable Agent Session artifact links.
  Single backlog create, batch backlog create, and direct initiative create
  write artifact records through the Agent Sessions service before success is
  returned. The UI must read these records instead of inferring session links
  from display strings or Agent Manager tags.
- **Batch rule**: Batch writes stamp every created backlog item with the same
  verified request provenance. Initiative metadata created through batch
  operations receives provenance through the initiative assigner adapter.

### API-to-Integration Seam

```go
// Pattern: Integration services behind interfaces

type AgentManagerService interface {
    IsEnabled() bool
    IsAvailable(ctx context.Context) bool
    ResolveURL(ctx context.Context) (string, error)
    GetProfileID() string

    SpawnBacklog(ctx context.Context, req BacklogSpawnRequest) (RunResult, error)
    SpawnResearch(ctx context.Context, req ResearchSpawnRequest) (RunResult, error)
    GetRunState(ctx context.Context, runID string) (RunState, error)
    StopRun(ctx context.Context, runID string) error
}

type EcosystemManagerClient interface {
    CreateTask(ctx context.Context, req ecosystem.CreateTaskRequest) (string, error)
}
```

**Status**: Agent-manager service seam implemented; handlers depend on the service interface while HTTP/proto details stay in the integration layer.

### Filesystem Seam

```
ideas/
└── {item-name}/
    ├── spec.json        # Required: metadata (backlog item root)
    ├── notes.md         # Optional: context
    ├── archive/         # Preserved scenario files (only for archived scenarios)
    │   ├── PRD.md       #   Files from the original scenario, namespaced
    │   ├── README.md    #   to avoid collisions with backlog-specific data
    │   └── docs/
    ├── plan.md          # Primary execution artifact (workshop output)
    └── workshop/        # Workshop rounds (iterative refinement)
        ├── round-1.json
        ├── round-2.json
        └── ...
research/
└── {item-name}/
    ├── spec.json
    └── research/
        └── summary.md
fix/
└── {item-name}/
    └── spec.json
execute/
└── {item-name}/
    └── spec.json

.vrooli/
├── settings.json          # User/system settings (persisted)
├── queue.json             # Pending local queue items (persisted)
└── agent-activities.json  # Durable tracked agent usage history
```

**Status**: Backlog, scenario metadata, settings, queue, and agent activity storage implemented.

### Dependency Graph Boundary

`api/internal/depgraph/graph.go` is a pure computation boundary with zero I/O dependencies. It provides:

- **Graph construction**: `New(items)` builds a directed graph from backlog items' `depends_on` fields
- **Cycle detection**: `HasCycle()` returns true if the graph contains any circular dependency
- **Topological sort**: `TopoSort()` returns items in dependency-safe execution order
- **Validation**: `ValidateRefs(items)` ensures all `depends_on` references point to existing items

This package is imported by `internal/backlog/` (batch create validation, batch queue ordering) and `internal/overview/` (graph visualization data).

**Testing at the seam**: Unit tests exercise cycle detection (diamond graphs, self-loops), topological ordering guarantees, and invalid reference rejection without needing HTTP or filesystem mocks.

### Unblocking Value Computation Boundary

`ui/src/lib/dependency-sort.ts` contains two pure-function seams for priority ranking:

- **`computeDepthMap(items)`**: Forward-edge graph walk computing topological depth for dependency-aware sort ordering
- **`computeUnblockingMap(items)`**: Reverse-edge graph walk computing transitive dependent counts for unblocking value scoring
- **`computeEffectivePriority(manualPriority, transitiveDependentCount)`**: Pure formula applying capped boost to manual priority

Both graph computations are O(V+E) with memoization. They share the same `SORT_RESOLVED_STATUSES` set and `archivedAt` check to determine which items count as resolved.

Consumed by `backlog-sort.ts` (sidebar/command post sorting) and `feed.ts` (unified feed). All consumers compute the unblocking map once per sort call and close over it in a compareFn — the map is never recomputed per comparison.

**Testing at the seam**: Pure-function unit tests in `dependency-sort.test.ts` covering chains, diamonds, cycles, completed/archived exclusion, dangling refs, and cross-kind dependencies. Integration tests in `backlog-sort.test.ts` verify that unblocking boost composes correctly with depth ordering.

### Initiatives Boundary

`api/internal/initiatives/` provides initiative CRUD and rollup status computation.

- **BacklogLoader interface**: Seam between initiatives and the backlog store -- initiatives need to list backlog items to compute rollup status, but do not depend on the backlog HTTP handlers
- **Partial update contract**: Initiative updates accept only the fields that are changing (`title`, `description`, `status`, `priority`, `depends_on`, `items`, `acceptance_criteria`, `note`). Public create/update requests reject `mode`; every mode mutation flows through the operating-mode switch boundary.
- **Rollup status**: Derived from member item statuses (pending if all backlog, active if any in_progress, completed if all done, blocked if any has unmet deps)
- **Operating-mode metadata**: CRUD owns persistence of `mode` and public mutation of `acceptance_criteria`, but not public mode mutation, phase orchestration, prompt rendering, artifact parsing, or runner behavior. Blank historical mode normalizes to `item-level`.

**Testing at the seam**: Mock `BacklogLoader` to test rollup computation without touching the filesystem.

### Operating Mode Boundary

`api/internal/operatingmode/` is the static methodology registry and lifecycle
runner for Swarm Manager execution modes. The package is deliberately split by
responsibility so methodology declarations, phase-state decisions, runner
orchestration, audit reconciliation, and workspace read models do not collapse
back into one service file.

- **Registry ownership**: Mode values, scope kind, phase graph, transition rules, run strategy, prompt/catalog IDs, profile keys, artifact roots, result bindings, backlog-sync policy, metrics semantics, lock policy, and UI workspace IDs live in one package.
- **Authoring boundary**: Static mode definitions are focused files such as `mode_holistic_loop.go` and `mode_phased_plan_drain.go`. Initiative-scoped modes use the definition builder for repeated policy. Adding a mode should not require mode-specific branches in shared lifecycle, stats, UI, CLI, prompt catalog, artifact, activity, or lock code. See [DOC: internal/OPERATING-MODE-AUTHORING.md].
- **Current modes**: `item-level`, `holistic-loop`, and `phased-plan-drain` are registered. `item-level` bridges to existing backlog execution; the two initiative-scoped modes define phase/profile/transition/artifact/metrics policy and use backend-enforced phase actions.
- **Catalog rule**: `GET /api/v1/operating-modes` is the read-only registry catalog for operator-facing mode selection. UI and CLI selection surfaces consume that endpoint and its backend-declared capabilities instead of maintaining hard-coded mode option lists.
- **Validation rule**: Unknown modes and invalid phase starts fail closed. Phase start validation is derived from the registry phase graph, transition rules, and completed round state; failed/canceled rounds do not advance the graph and active rounds block all new starts.
- **Profile policy**: Registry references stable scenario-owned AgentManager profile keys such as `swarm-manager/deep-work` and `swarm-manager/analysis`; runner code must not inline model/tool policy. API startup passes the registry's required profile keys into AgentManager reconciliation and fails startup when any referenced key is missing or outside the `swarm-manager/` namespace.
- **Activity purpose policy**: Initiative mode phases declare stable lowercase snake-case activity and lock purpose tokens in the mode definition. Shared `agentactivity` and `initiativelock` packages must not learn a new constant for every mode phase.
- **Prompt rule**: Operating-mode phase prompts are fail-closed. `operatingmode.Service` validates the registry's `(mode, phase)` skill against the prompt catalog resolver and requires prompt-manager to render non-empty content. A prompt catalog miss, skill mismatch, prompt-manager error, or empty render marks the reserved round failed, releases the lock, and returns an error before any AgentManager spawn.
- **Prompt catalog rule**: Operating-mode prompt catalog entries are generated from registry phase metadata, then consumed by `promptcatalog`. Drift in catalog ID, skill ID, mode, phase, or output paths fails validation.
- **Lock seam rule**: `operatingmode.Service` depends on a narrow initiative-lock interface rather than the concrete file lock. The production adapter is still `initiativelock.Lock`, but tests can inject lock failures at specific lifecycle moments such as the provisional-to-run-ID ownership swap.
- **Audit rule**: Mode changes are emitted as `initiative.mode_changed` events. Phase runners emit typed `operating_mode.phase_*`, `operating_mode.replan_needed`, and `operating_mode.backlog_synced` events rather than relying on file scans or agentactivity inference.
- **Switch rule**: External callers switch modes through `POST /api/v1/initiatives/{name}/operating-mode/switch`, not generic initiative metadata update. The switch service is the lifecycle boundary that detects active item-level executions, requires explicit cancellation confirmation before entering a non-default initiative mode, and rejects switching out of non-default modes while a mode round is active.
- **Backlog reconciliation rule**: Non-default modes may mark member items complete through the run-id-validated `complete-items` endpoint. The service passes a typed backlog mutation source (`entrypoint`, initiative, mode, phase, round, run ID, requested-by) to the adapter; the event log records that source on both the `backlog.status_changed` event and the `operating_mode.backlog_synced` summary. Create/update/follow-up reconciliation must flow through `apply-backlog-sync`, which adapts the round's `backlog_sync.proposal` to the existing `proposals.Applier` and carries equivalent mode/phase/run metadata through `proposals.Source`; `operatingmode` depends only on a narrow reconciler interface to avoid importing the proposal/graph/initiative stack directly.
- **Stats rule**: Mode stats derive from the durable event log and registry metrics policy. Profile usage, phase counts, replan/acceptance rates, phase durations, and backlog-sync counts are all sourced from operating-mode event payloads. Replan and acceptance sample phases are opt-in mode policy rather than hardcoded phase names; `operating_mode.replan_needed` stays available for timeline observability without double-counting the numerator.
- **Artifact and round persistence**: `operatingmode.Store` owns mode-scoped paths under each initiative (`modes/<mode>/...`), current-state artifact reads/writes, and append-only round envelopes at `modes/<mode>/rounds/round-NNN.json`. Runner and handler code should use this store rather than constructing paths directly. Derived artifact writes belong in phase `ResultBindings`, not in mode-specific artifact applier branches.
- **Round envelope contract**: Round JSON records mode, phase, scope, run strategy, selected `agent_profile_key`, run ID, status, readiness, artifact updates, handoffs, and phase payload. Sequential modes preserve handoffs in round JSON so future runs have durable context.
- **Classification/readiness helpers**: Holistic readiness and phased-plan progress classification are parsed/validated in `operatingmode`; UI and runner code should not duplicate accepted dimension or decision enums.
- **Capability rule**: The backend declares capabilities on catalog and workspace responses. UI and CLI rendering must use those capabilities and phase actions instead of inferring support from mode names or local phase-shape guesses.
- **UI round view-model rule**: The UI service owns wire normalization and `ui/src/components/initiative/operating-mode/round-view-model.ts` owns operating-mode round payload interpretation for cards. React components must not reach into `round.payload` for backlog-sync plans, applied-sync state, mutation defaults, or action availability; they render the view model and call service callbacks.

**Code boundary map:**

| File | Responsibility |
|------|----------------|
| [CODE: api/internal/operatingmode/registry.go] | Registry core types, validation, lookup, prompt catalog validation, profile key collection |
| [CODE: api/internal/operatingmode/definition_builder.go] | Initiative-mode definition helper for repeated authoring policy |
| [CODE: api/internal/operatingmode/mode_item_level.go] | Default item-level mode definition |
| [CODE: api/internal/operatingmode/mode_holistic_loop.go] | Holistic-loop mode definition |
| [CODE: api/internal/operatingmode/mode_phased_plan_drain.go] | Phased-plan-drain mode definition |
| [CODE: api/internal/operatingmode/prompt_catalog_entries.go] | Generated operating-mode prompt catalog metadata |
| [CODE: api/internal/operatingmode/state.go] | Backend-authoritative phase actions and transition-rule evaluation |
| [CODE: api/internal/operatingmode/service.go] | Public service contracts, request/response models, dependency wiring, initiative-lock seam |
| [CODE: api/internal/operatingmode/switcher.go] | Lifecycle-only mode switch boundary and active-run conflicts |
| [CODE: api/internal/operatingmode/phase_runner.go] | Phase start orchestration, lock acquisition, AgentManager spawn |
| [CODE: api/internal/operatingmode/prompt.go] | Prompt catalog validation and fail-closed prompt rendering |
| [CODE: api/internal/operatingmode/round_refresher.go] | AgentManager state polling, terminal round persistence, cancellation |
| [CODE: api/internal/operatingmode/artifact_applier.go] | Structured result parsing, contract enforcement, and declared artifact/result-binding writes |
| [CODE: api/internal/operatingmode/backlog_reconciler.go] | Run-id-validated item completion and proposal-backed backlog sync |
| [CODE: api/internal/operatingmode/workspace.go] | Workspace read model and phase action projection |
| [CODE: api/internal/operatingmode/events.go] | Typed operating-mode event emission helpers |
| [CODE: api/internal/operatingmode/synthetic_mode_test.go] | Test-only non-production authoring harness |
| [CODE: ui/src/components/initiative/operating-mode/round-view-model.ts] | UI-side operating-mode round payload parsing, action availability, proposal defaults, and mutation summaries |
| [CODE: ui/src/components/initiative/operating-mode/backlog-sync-actions.tsx] | Focused backlog proposal selection/apply control for round cards |
| [CODE: ui/src/components/initiative/operating-mode/round-card.tsx] | Presentation shell for round status, summary, handoffs, timestamps, and delegated action slots |
| [CODE: api/routes_operating_mode.go] | HTTP adapters between route wiring and narrow operating-mode interfaces |
| [CODE: ui/src/components/initiative/operating-mode-panel.tsx] | Composition-only operating-mode panel |
| [CODE: ui/src/components/initiative/operating-mode/use-operating-mode-workspace.ts] | React Query workspace orchestration and mutations |

**Decision boundaries:**

- To change sequencing, update the mode definition's transitions and transition rules. Do not
  put mode-specific phase ordering in handlers, CLI commands, or React
  components.
- To add derived artifacts, update the phase's result bindings. Do not add mode
  branches to the artifact applier.
- To change replan or acceptance metric semantics, update `MetricsPolicy` through
  the phase definition. Do not add phase-name checks to stats aggregation.
- To change an initiative's mode, call the operating-mode switch service. Public
  initiative create/update request types intentionally do not expose `mode`.
- To reconcile backlog from a mode round, use `complete-items` or
  `apply-backlog-sync`; agents must not edit member item `spec.json` files
  directly.
- To change AgentManager model/tool policy, edit the scenario-owned profile JSON
  files. The registry only references required profile keys and startup fails
  when those keys are absent.

**Testing at the seam**: `api/internal/operatingmode/*_test.go` locks required modes, default normalization, unknown-mode rejection, phase profile mappings, stable registry-authored activity/lock purposes, mode-scoped path resolution, round numbering, malformed JSON rejection, handoff/profile preservation, artifact root validation, transition-rule routing, result-binding writes, registry-driven metrics semantics, backend-declared capabilities, prompt catalog generation, readiness scoring, progress classification validation, and lifecycle cleanup when lock acquisition or run-ID lock ownership fails. The synthetic mode harness proves new non-production methodology behavior can run through framework paths without shared control-flow edits.

### Backlog Import Boundary

`api/internal/backlog/batch_handler.go` is the backlog-import seam for meta-orchestration and batch backlog creation.

- **Strict request decoding**: Unknown fields are rejected at the HTTP boundary. Legacy `scope` payloads are invalid.
- **Preview mode**: The same endpoint validates item payloads, dependency refs, and initiative actions without mutating disk when `preview=true`.
- **Initiative planning**: Each item carries its own `initiative` reference, while initiative metadata is supplied separately through the request's `initiatives` array.
- **Rollback behavior**: Real creates are atomic across initiative metadata changes, backlog item creation, and initiative membership assignment. Failures roll back the batch instead of returning warnings for partial success.

**Testing at the seam**: Unit tests cover preview, invalid legacy fields, rollback on initiative-assignment failure, and dependency validation.

### Overview Service Boundary

`api/internal/overview/` aggregates data from multiple domains into a single response.

- **BacklogLister interface**: Lists backlog items (satisfied by backlog store)
- **InitiativeLister interface**: Lists initiatives (satisfied by initiatives package)

The overview endpoint composes these interfaces to produce a summary containing backlog counts by status/kind, initiative rollups, dependency graph edges, and summary statistics.

**Testing at the seam**: Mock both lister interfaces to test aggregation logic in isolation.

### Prompt Catalog Boundary

`api/internal/promptcatalog/catalog.go` is the canonical prompt inventory for swarm-manager.

- **Entries()**: Returns the full catalog exposed by `/api/v1/prompts/catalog`
- **ResolveBacklogSkill(mode, kind)**: Resolves the runtime workshop/initialize/finalize skill for backlog flows
- **ResolveInitiativeModeSkill(mode, phase)**: Resolves initiative operating-mode phase skills from catalog metadata instead of adding phase-specific switches
- **ResolveCaptureSkill() / ResolveSpecSyncSkill()**: Resolve non-backlog runtime skills
- **SkillUsageCount() / SkillImpactSummary()**: Drive prompt center summaries from the same inventory used by runtime code
- Covers both prompt-manager skills and generated runtime prompts, so the catalog reflects actual execution behavior instead of only stored skills

**Testing at the seam**: Pure lookup and metadata functions with unit tests covering runtime resolution and support-skill references.

### Backlog Store Boundary

`api/internal/backlog/store.go` defines the `Store` interface and `FileStore` concrete implementation for backlog item persistence.

- **Store interface**: Abstracts filesystem CRUD (LoadAll, LoadItem, SaveItem, ValidateDependencies, CheckDependencies, KindDir, ItemDir)
- **FileStore struct**: Filesystem-backed implementation; encapsulates base directory, spec.json serialization, directory creation
- **Sentinel errors**: `ErrNotFound`, `ErrAlreadyExists`, `ErrInvalidKind` in `errors.go` enable `errors.Is` in handlers
- Handlers hold `Store` (interface), enabling mock injection for fault testing (e.g., `failingSaveStore` in batch rollback tests)

**Testing at the seam**: Tests exercise CRUD operations against a temp directory. Mock stores can be injected for fault injection (SaveItem failures, simulated disk errors).

### Execution Queuer Boundary

`api/internal/backlog/batch_queue_handler.go` defines the `ExecutionQueuer` interface for decoupling batch queue from the execution service.

- **ExecutionQueuer interface**: `ProcessPreflight(ctx, kind, name)` and `QueueBacklog(ctx, req)` — the two operations batch-queue needs
- **Setter**: `Handler.SetExecutionQueuer(eq)` follows the same DI pattern as `InitiativeAssigner`
- **Default fallback**: If no queuer is injected, `BatchQueue` constructs `execution.NewService(...)` inline (zero behavior change for production)
- `*execution.Service` satisfies the interface implicitly — no adapter needed

**Testing at the seam**: `mockExecutionQueuer` in test files enables testing the confirm:true path (preflight blocking, queue failures, partial success) without a real agent-manager or execution store.

### Workshop Computation Boundary

`api/internal/workshop/workshop.go` is a pure computation boundary with no HTTP or integration dependencies. It provides:

- **Readiness scoring**: `ComputeEffectiveScores(raw, roundsCompleted, kind)` applies the boost formula
- **Ready check**: `IsReady(effective)` returns true when all 5 dimensions reach 3
- **Round I/O**: `LoadRounds(itemDir)` and `SaveRound(itemDir, round)` handle filesystem serialization
- **Boost configuration**: `BoostN` map defines per-kind divisors
- **OtherKey sentinel**: `OtherKey = "__other__"` — `CountPendingDecisions` treats decisions with `Selected == OtherKey` and empty `Freeform` as unanswered, preventing premature auto-advance

This package is imported by both `internal/backlog/` (for research handler) and `internal/execution/` (for preflight readiness checks), avoiding import cycles.

**Testing at the seam**: Unit tests exercise boost edge cases (raw < 2 not boosted, round accumulation, kind-specific divisors), round serialization, and OtherKey freeform validation without needing HTTP or agent-manager mocks.

### Workshop Auto-Advance & Pending Advance

`api/internal/backlog/workshop_save.go` orchestrates auto-advance after saving a workshop round. When `auto_advance_delay_seconds > 0`, instead of spawning immediately, it writes a **pending advance file** (`.workshop-pending-advance.json`) to the item directory. Key components:

- **Pending advance file** (`workshop_pending.go`): JSON file with `advance_at` timestamp, `next_mode`, item metadata. Functions: `writePendingAdvance`, `readPendingAdvance`, `deletePendingAdvance`.
- **Background ticker** (`workshop_ticker.go`): `WorkshopTicker` polls every 2 seconds, fires `spawnWorkshopAsync` when `advance_at` is reached. Maintains an in-memory `sync.Map` registry. On startup, `RecoverPending()` scans all item directories for leftover files.
- **Cancel endpoint**: `DELETE /api/v1/backlog/{kind}/{name}/workshop/pending-advance` — deletes the pending file and unregisters from the ticker.
- **Idempotency**: New saves replace any existing pending advance. The workshop lock file (`.workshop-lock`) prevents concurrent spawns.

**Testing at the seam**: Integration tests verify pending file creation with delay > 0, immediate spawn with delay = 0, cancel endpoint, and replacement of existing pending advances.

### Workshop Parsing Boundary (UI)

`ui/src/lib/workshop-files.ts` is the UI-side parsing boundary for workshop data:

- **Round parsing**: `parseWorkshopRound(content)` with truncation recovery for robustness against agent crashes mid-write
- **Round serialization**: `buildWorkshopRoundContent(round)` for sending updated rounds back
- **Metrics**: `getUnansweredCount(round)` and `getPendingProposalCount(round)` for UI progress indicators
- **File tree**: `findBacklogFileByPath(files, path)` for navigating backlog file hierarchies

The truncation recovery algorithm (ported from the former `idea-agent-files.ts`) scans for the last complete JSON object in a truncated array, enabling graceful degradation when an agent crashes mid-write.

**Testing at the seam**: Tests exercise valid JSON, truncated JSON recovery, empty/null input, and malformed content.

### Workshop Auto-Trigger Boundary

`api/internal/workshop/autotrigger.go` contains pure decision functions for auto-execution:

- **`ShouldAutoAdvance(enabled bool, latestRound, roundCount, kind, maxAutoRounds)`**: Decides whether to auto-trigger the next round after saving responses. Returns false when: disabled via setting, item is ready (all dimensions >= 3), round cap reached, pending decisions exist, or no rounds exist. Returns a reason string for API consumers.
- **`ShouldAutoInitialize(enabled bool)`**: Decides whether a newly-created item should get its first workshop round auto-triggered. Controlled by the global `auto_initialize_workshop` setting.
- **`ShouldCascade(enabled bool)`**: Decides whether dependency resolution should auto-trigger downstream workshops. Controlled by the global `auto_cascade_workshop` setting.

All three auto-execution behaviors are controlled by global settings in `.vrooli/settings.json`, loaded via `settings.NewStore("").Load()`. There are no per-item overrides.

`api/internal/backlog/workshop_save.go` is the integration boundary that wires auto-trigger decisions to agent spawning:

- **`WorkshopSave` handler**: Dedicated endpoint (`POST /api/v1/backlog/{kind}/{name}/workshop/save`) that saves round JSON and auto-advances when appropriate.
- **`spawnWorkshopAsync` helper**: Shared by both auto-advance (WorkshopSave) and auto-initialize (Create). Acquires an idempotency lock, fetches prompt from prompt-manager, and spawns via agent-manager. Fire-and-forget with background context.
- **Idempotency lock**: `.workshop-lock` file in item dir prevents concurrent spawns. 30-minute TTL auto-cleans stale locks.

**Testing at the seam**: Pure decision functions are unit tested in `autotrigger_test.go`. Integration tests in `workshop_save_test.go` use `mockAgentService` and per-test settings fixtures to verify spawn/no-spawn decisions, lock behavior, and error resilience. Tests disable auto-workshop by default via `disableAutoWorkshopSettings()` and selectively re-enable for specific test cases.

### Execution Review Boundary

`api/internal/execution/review_client.go` defines the `ReviewClient` interface for post-execution readiness reviews via git-control-tower.

- **ReviewClient interface**: `TriggerReview(ctx, req)` starts a review job, `PollReview(ctx, jobID)` checks status, `Ping(ctx)` checks GCT availability
- **HTTPReviewClient**: HTTP implementation using service discovery (`discovery.ResolveScenarioURLDefault("git-control-tower")`)
- **Review result mapping**: `mapJobToResult` converts git-control-tower readiness (green/yellow/red) to internal classification (ready/ready_with_notes/needs_work)
- **Dimension statuses**: Per-dimension status (green/yellow/red/skipped) is computed by GCT and included in the response via `dimensionStatuses`. Swarm-manager uses these directly — GCT is the single source of truth for readiness scoring.
- **Configurable thresholds**: `ReviewRequest` includes an optional `Thresholds` field. When set, GCT uses these thresholds instead of its defaults. Thresholds are loaded from swarm-manager settings via `ReviewThresholdsProvider` interface (implemented by `settingsReviewThresholdsAdapter` in main.go).
- **Wired in main.go**: `registerExecutionRoutes` sets `cfg.ReviewClient = NewHTTPReviewClient(nil)` and `cfg.ReviewThresholdsProvider = &settingsReviewThresholdsAdapter{...}`. Auto-triggering is guarded by `shouldTriggerReview()` (requires acceptance_allow with scenario globs)

**Testing at the seam**: `review_client_test.go` uses `httptest.NewServer` with mock handlers and `triggerReviewDirect`/`pollReviewDirect`/`pingDirect` helpers that bypass service discovery. `service_test.go` uses `stubReviewClient` for service-level tests.

### Trigger-Review API Boundary

`POST /api/v1/execution/{execution_id}/trigger-review` allows manual review trigger for terminal executions.

- **Service method**: `Service.TriggerReview(ctx, executionID)` validates terminal status (completed/needs_fixup/failed), calls `ReviewClient.TriggerReview`, transitions to `StatusValidating`
- **Handler**: `Handler.TriggerReview` extracts execution_id from path, delegates to service, maps errors via `mapMutationError`
- **GCT status**: `GET /api/v1/gct/status` returns `{"available": true/false}` by calling `ReviewClient.Ping`
- **Review skip tracking**: `Record.ReviewSkipReason` captures why automatic review was skipped (GCT unavailable, not configured)
- **Polling timeout**: 10-minute timeout in `refreshRunningLocked` transitions stuck validating records to `StatusNeedsFixup`

**Testing at the seam**: `TestTriggerReview_CompletedExecution`, `TestTriggerReview_WrongStatus`, `TestTriggerReview_NoClient` in `service_test.go`.

### Execution Follow-Up Boundary

`api/internal/execution/service.go` exposes the `FollowUp` method for user-initiated follow-ups from completed/failed/needs_fixup executions.

- **runContinuer interface**: `ContinueRun(ctx, runID, message)` — sends follow-up message to existing agent-manager session
- **agentSpawner interface**: `SpawnBacklog(ctx, req)` — spawns fresh agent run (existing seam)
- **Run mode handling**: `continue` mode calls `ContinueRun` on the previous run; `new` mode spawns fresh via `SpawnBacklog`
- **Prompt construction**: `buildFollowUpMessage` generates context-aware prompts based on follow-up type (fixup/followup/custom)

**Testing at the seam**: Stub implementations of `runContinuer` and `agentSpawner` in test files enable testing both run modes, session expiry handling, and prompt construction without agent-manager.

### Execution Retry-as-New-Attempt Boundary

`api/internal/execution/retry.go` exposes `Retry(ctx, RetryRequest)` and `RetryLatestForBacklog(ctx, kind, name, note)` for user-initiated retries. The semantics are *new-attempt only* — the parent execution row is never mutated.

- **Parent state gate**: `completed | failed | canceled | needs_fixup`. Other states return 400.
- **Idempotency**: in-flight detection inside the locked critical section dedups concurrent retries (same `ParentExecutionID + Operation == "retry"` and a non-terminal status returns the existing record).
- **Backlog-level convenience**: `ExecutionQueuer.RetryLatestForBacklog(ctx, kind, name, note)` resolves the latest terminal execution for an item and calls `Retry`. Used by `POST /api/v1/backlog/{kind}/{name}/retry`.
- **Item reopen**: when the backlog item is in a terminal status, `backlog.Handler.reopenForRetry` flips it back to `in_progress`, writes a `review/decisions/{ts}-reopen.json` audit record, and emits `EmitBacklogStatusChanged`. This is the *only* legitimate writer of backward terminal transitions; see INVARIANTS.md "Terminal State Writers" table.

**Testing at the seam**: `retry_test.go` in both `internal/execution` and `internal/backlog`. The `mockExecutionQueuer.RetryLatestForBacklog` allows handler tests to wire success/failure paths without a real execution service.

## Architectural Decisions

### ADR-001: File-Based Backlog

**Decision**: Store backlog items as git-tracked folders rather than database records.

**Rationale**:
- Git provides version history and collaboration
- Human-readable without tooling
- Easy to inspect, backup, and restore
- Aligns with scenario folder structure

**Consequences**:
- Need filesystem operations in API
- Must handle concurrent file access
- Backlog directories should remain git-tracked for collaboration

### ADR-002: Integration-First Architecture

**Decision**: All complex operations delegate to ecosystem-manager and agent-manager.

**Rationale**:
- Avoid duplicating orchestration logic
- Single source of truth for agent spawning
- Consistent with Vrooli's recursive improvement model

**Consequences**:
- Swarm Manager is a thin orchestration layer
- Depends on other scenarios being available
- Simpler business logic, more integration code

### ADR-003: Three-State Recommendation Engine

**Decision**: Recommendations operate in Off, Suggestions, or YOLO mode.

**Rationale**:
- Off: Manual control, no automated suggestions
- Suggestions: System proposes, human approves
- YOLO: Full autonomy, auto-approve recommendations

**Consequences**:
- Need persistent mode setting
- YOLO mode requires careful guardrails
- Mode affects recommendation lifecycle

## Technical Debt Register

| ID | Area | Description | Priority | Effort | Status |
|----|------|-------------|----------|--------|--------|
| TD-001 | selectors.ts | Selector registry is empty but components use selectors | High | Low | ✅ Resolved (Phase 1) |
| TD-002 | API | Recommendations endpoints not implemented | Medium | Medium | ✅ Resolved |
| TD-003 | Integration | No adapter code for agent-manager/ecosystem-manager | High | Medium | ✅ Resolved (agent-manager client + ecosystem seam) |
| TD-004 | Recommendations | Engine and persistence not implemented | Medium | Medium | ✅ Resolved |
| TD-006 | UI types | Domain types duplicated in page components | Medium | Low | ✅ Resolved (Phase 2) |
| TD-007 | UI constants | Status colors/icons defined inline in pages | Low | Low | ✅ Resolved (Phase 2) |
| TD-008 | API client | Singleton at module scope, hard to substitute in tests | Medium | Medium | ✅ Resolved (Phase 3) |

## Module Boundaries

### UI Module Structure (Updated Phase 4)

```
ui/src/
├── components/
│   ├── layout/        # App chrome (MainLayout)
│   └── ui/            # Reusable primitives (button, tabs)
├── config/            # Centralized configuration (NEW in Phase 4)
│   ├── index.ts       # All tunable levers with documented impacts
│   └── index.test.ts  # Validation tests for configuration bounds
├── pages/             # Feature pages (presentation only)
├── services/          # Data access seams (NEW in Phase 3)
├── stores/            # Zustand stores for shared list state
│   ├── backlog-store.ts      # Backlog list state
│   ├── scenarios-store.ts    # Scenarios list state
│   ├── recommendations-store.ts # Recommendations list state
│   └── index.ts              # Barrel export
├── types/             # Domain types and constants
│   ├── domain.ts      # Idea, Scenario, Recommendation, Settings types
│   ├── constants.ts   # Status colors, icons, formatting functions
│   └── index.ts       # Barrel export
├── lib/               # Infrastructure utilities
│   ├── api-client.ts  # HTTP client class and IApiClient interface
│   ├── api-endpoints.ts # Endpoint path constants
│   ├── error-utils.ts # Error categorization, logging, recovery paths
│   ├── query-utils.ts # React Query default configuration
│   ├── workshop-files.ts # Workshop round parsing, truncation recovery, metrics
│   ├── utils.ts       # Generic utilities (cn for classnames)
│   └── index.ts       # Barrel export
└── consts/            # UI-specific constants
    └── selectors.ts   # Test selector registry (source of truth)
```

**Boundary Rules**:
1. **Pages**: Presentation only - render data and handle user interactions
2. **Config**: Control surface - all tunable levers centralized, mockable for testing
3. **Services**: Data access seams - encapsulate API calls, injectable for testing
4. **Types**: Domain types and related constants - shared across pages
5. **Lib**: Infrastructure utilities - HTTP client, generic helpers
6. **Components**: Reusable UI primitives - no domain logic
7. **Consts**: UI-specific constants - selectors, test IDs

**Seam Hierarchy** (from high-level to low-level):
```
Pages → Config → Services → API Client → HTTP/fetch
           ↑         ↑            ↑
       Seam #3   Seam #1      Seam #2
       (mock for (mock for    (inject for
        behavior) page tests)  service tests)
```

### API Module Structure

```
api/
├── main.go              # Entry point, server wiring, health endpoints
├── go.mod / go.sum      # Go module dependencies
└── internal/
    ├── backlog/         # ✅ Refactored (was single handler.go, now decomposed)
    │   ├── types.go              # Domain types and interfaces
    │   ├── errors.go             # Sentinel errors (ErrNotFound, ErrAlreadyExists, ErrInvalidKind)
    │   ├── store.go              # Store interface + FileStore implementation
    │   ├── handler.go            # Route registration and core CRUD handlers
    │   ├── files.go              # File upload/download handlers
    │   ├── research.go           # Research spawn handlers
    │   ├── queue_ops.go          # Queue/dequeue handlers
    │   ├── archive_handlers.go   # Archive operations
    │   ├── kind_config.go        # Per-kind metadata (deliverable, directory)
    │   ├── batch_handler.go      # POST /batch (all-or-nothing create)
    │   ├── batch_queue_handler.go # POST /batch/queue (topological order)
    │   └── *_test.go             # Tests for each module
    ├── depgraph/        # ✅ NEW — dependency graph (pure computation)
    │   └── graph.go              # Cycle detection, topological sort
    ├── initiatives/     # ✅ NEW — initiative CRUD + rollup status
    ├── overview/        # ✅ NEW — aggregation endpoint
    ├── captures/        # ✅ NEW — capture CRUD and classification
    ├── skills/          # ✅ NEW — centralized skill registry
    │   └── registry.go           # Skill name -> prompt-manager path
    ├── scenarios/       # ✅ Implemented
    ├── settings/        # ✅ Implemented
    ├── workshop/        # ✅ Implemented (readiness scoring, round I/O)
    ├── queue/           # ✅ Implemented
    ├── execution/       # ✅ Implemented
    ├── prompts/         # ✅ Implemented
    └── integrations/    # agent-manager, prompt-manager, ecosystem-manager
```

**Current State**: The backlog package has been decomposed from ~2,286 lines in a single handler.go into ~450 lines per focused module. New domain packages (depgraph, initiatives, overview, captures, skills) follow the same `internal/<domain>/` convention.

## Cross-Cutting Concerns

### Error Handling

- UI: React Query error states, user-friendly messages
- API: Standard error response format from api-core
- CLI: Exit codes and stderr messages

### Logging

- API: Request logging middleware (implemented)
- UI: Console logging for development
- CLI: Verbose flag support (via cli-core)

### Health Checks

- API: `/health` with filesystem-only readiness
- UI: `/health` via server.js static serving
- Both: Defined in service.json health config

## Change Axes

This section documents the primary ways this scenario is likely to change, where those changes should land, and how localized each axis currently is.

### Primary Change Axes

| Axis | Description | Frequency | Current Cost |
|------|-------------|-----------|--------------|
| New Domain Entity Status | Adding new status values (e.g., "paused" for backlog) | Low | **1 file** - `types/domain.ts` |
| Status Display Mapping | Adding colors/icons for new statuses | Low | **1 file** - `types/constants.ts` |
| New API Endpoint | Adding CRUD for new entity or operation | Medium | **2-3 files** - `api-endpoints.ts`, new service |
| New UI Page | Adding a detail view or new tab | Medium | **3 files** - page, `App.tsx`, `selectors.ts` |
| Configuration Tuning | Adjusting thresholds, timeouts, limits | High | **1 file** - `config/index.ts` |
| New CLI Command | Adding command for new API capability | Medium | **1 file** - `cli/app.go` |
| New Integration Target | Adding client for ecosystem scenario | Low | **2-3 files** - new client, API handler |
| Error Type/Recovery | Adding new error category | Low | **2 files** - `api-client.ts`, `error-utils.ts` |

### Change Localization Analysis

#### Well-Localized (1-2 files for typical change)

1. **Domain Types & Status Values**
   - Change location: `ui/src/types/domain.ts`
   - The type union pattern makes adding new status values trivial
   - Display mapping in adjacent `constants.ts` keeps changes cohesive

2. **Configuration Values**
   - Change location: `ui/src/config/index.ts`
   - All tunable levers in one file with documented impacts
   - Tests validate bounds, catching invalid configurations

3. **API Endpoint Paths**
   - Change location: `ui/src/lib/api-endpoints.ts`
   - Single source of truth for endpoint strings
   - Services import from here, not hardcode paths

#### Acceptably Localized (3-4 files for typical change)

1. **New Service Operations**
   - Required: `api-endpoints.ts` (if new endpoint), new service file, possibly page
   - Pattern: Factory function + interface, following `backlog-service.ts` template
   - Trade-off: More boilerplate but clean testability seams

2. **New UI Routes**
   - Required: New page component, `App.tsx` route, `selectors.ts` entries
   - Pattern: Page uses services, imports types/config, registers selectors
   - Trade-off: Explicit routing declaration over magic

#### Areas Needing Attention (Shotgun Surgery Risk)

1. **Adding New Domain Entity (End-to-End)**
   - Currently requires: type in `domain.ts`, constants in `constants.ts`, endpoint, service, page, selectors, API handler, CLI command
   - This is inherent complexity, not poor localization
   - Mitigation: Document the checklist in INTENT.md (done)

2. **Integration Clients (Not Yet Implemented)**
   - Future work: Create `api/integrations/` directory with interface pattern
   - Each integration should be its own file behind an interface
   - Follow the service seam pattern from UI side

### Stable vs Volatile Areas

```
STABLE (change rarely, high impact if changed)
├── ui/src/lib/api-client.ts     # HTTP infrastructure, error types
├── ui/src/consts/selectors.ts   # Test selector machinery (types/helpers)
├── cli-core / api-core          # Shared packages (external)
└── service.json                 # Deployment configuration

VOLATILE (expected to change frequently, should be easy to modify)
├── ui/src/types/domain.ts       # Domain types grow with features
├── ui/src/types/constants.ts    # Display mappings for new statuses
├── ui/src/config/index.ts       # Tunable values adjusted per feedback
├── ui/src/services/*            # New services for new domains
├── ui/src/pages/*               # New pages and page updates
└── api/handlers/*               # Business logic (to be created)

SEMI-STABLE (change occasionally, moderate impact)
├── ui/src/App.tsx               # Routes (grows with pages)
├── ui/src/lib/api-endpoints.ts  # Endpoints (grows with API)
├── selectors.ts (data portion)  # Selector IDs (grows with UI)
└── cli/app.go                   # Commands (grows with features)
```

### Extension Points

When adding new functionality, use these established extension points:

| Need | Extension Point | Pattern to Follow |
|------|-----------------|-------------------|
| New domain type | `types/domain.ts` | Add interface and status union |
| New status colors | `types/constants.ts` | Add to Record<Status, string> |
| New API endpoint | `api-endpoints.ts` | Add string constant |
| New service | `services/` | Copy `backlog-service.ts` structure |
| New page | `pages/` | Copy existing page, add route to `App.tsx` |
| New config value | `config/index.ts` | Add to appropriate group with docs |
| New selector | `selectors.ts` | Add to `literalSelectors` or `dynamicSelectorDefinitions` |
| New CLI command | `cli/app.go` | Add to appropriate `CommandGroup` |
| New error type | `api-client.ts` | Extend `ApiErrorType` union |

### Recommendations for Future Changes

1. **When Adding a P1 Integration** (e.g., test-genie, knowledge-observatory):
   - Create `api/integrations/` directory if not exists
   - Add interface for the client (e.g., `TestGenieClient`)
   - Implement client behind interface for testability
   - Follow existing `IScenariosService` pattern from UI

2. **When Implementing Recommendations Engine**:
   - Recommendation mode (off/suggestions/yolo) is in config already
   - Status type is defined in `types/domain.ts`
   - Create `recommendations-service.ts` following existing pattern
   - Engine logic belongs in API `services/` (to be created)

3. **When Adding YOLO Mode Auto-Approval**:
   - Safety delay and allowed priorities already in `config/index.ts`
   - Implementation should respect these config values
   - Add tests that verify config bounds are respected

## Alignment Improvements Made

### 2026-01-29 - Backlog Unification

**Updated**:
- Reframed the Ideas domain as a unified Backlog with kinds (idea, research, fix, execute)
- Renamed UI routes/pages/services/selectors to Backlog terminology
- Updated API/CLI documentation to reference `/backlog` endpoints and backlog kinds
- Clarified filesystem seams for `ideas/`, `research/`, `fix/`, and `execute/` folders

### 2026-01-28 - Phase 1: Architecture Documentation (Screaming Architecture)

**Created**:
- `docs/concepts/ARCHITECTURE.md` - Mental model and architecture overview
- `docs/internal/SEAMS.md` - This file
- `docs/manifest.json` - Documentation navigation structure
- Populated `ui/src/consts/selectors.ts` with all UI selector definitions

**Resolved**:
- TD-001: Selector registry is now fully populated

**Identified Gaps**:
- Recommendations endpoints missing (TD-002)
- Integration adapters missing (TD-003)
- Recommendation engine/persistence missing (TD-004)

### 2026-01-28 - Phase 2: Boundary-of-Responsibility Enforcement

**Created**:
- `ui/src/types/domain.ts` - Centralized domain type definitions (BacklogItem, Scenario, etc.)
- `ui/src/types/constants.ts` - Domain constants (status colors, icons, formatting)
- `ui/src/types/index.ts` - Barrel export for types module
- `ui/src/lib/index.ts` - Barrel export for lib module

**Refactored**:
- `ui/src/pages/BacklogPage.tsx` - Now imports types from `../types` instead of inline definitions
- `ui/src/pages/ScenariosPage.tsx` - Now imports types and constants from `../types`
- `ui/src/lib/api.ts` - Removed duplicate `fetchHealth` function, clarified module responsibilities

**Resolved**:
- TD-006: Domain types now centralized in `types/` module
- TD-007: Status colors/icons now in `types/constants.ts`

**Boundary Clarifications**:
1. **types/** module owns domain concepts - types and their display representations
2. **lib/** module owns infrastructure - HTTP client, generic utilities
3. **pages/** are now presentation-only - no inline type definitions or domain logic
4. Clear separation: pages import types for data, constants for display mapping

**Testing**: All UI tests continue to pass after refactoring

### 2026-01-28 - Phase 3: Seam Discovery & Enforcement

**Created**:
- `ui/src/lib/api-client.ts` - HTTP client with `IApiClient` interface (seam for substitution)
- `ui/src/lib/api-endpoints.ts` - Endpoint path constants separated from client
- `ui/src/services/backlog-service.ts` - Backlog CRUD operations behind `IBacklogService` interface
- `ui/src/services/scenarios-service.ts` - Scenarios operations behind `IScenariosService` interface
- `ui/src/services/index.ts` - Barrel export for services
- `ui/src/services/backlog-service.test.ts` - Tests demonstrating seam-based testing

**Refactored**:
- `ui/src/lib/api.ts` - Now re-exports from api-client.ts and api-endpoints.ts
- `ui/src/lib/index.ts` - Updated to export new modules
- `ui/src/pages/BacklogPage.tsx` - Now uses `backlogService` instead of direct API calls
- `ui/src/pages/ScenariosPage.tsx` - Now uses `scenariosService` instead of direct API calls
- `ui/src/pages/BacklogPage.test.tsx` - Refactored to mock at service level, not API level

**Resolved**:
- TD-008: API client is now behind interfaces with factory functions for injection

**Seam Improvements**:
1. **IApiClient interface** - HTTP client can be substituted without module mocking
2. **IBacklogService/IScenariosService** - Service layer provides clean testing seam
3. **Factory functions** - `createBacklogService(client)` allows explicit dependency injection
4. **Two-level testing** - Pages mock services, services inject mock clients

**Testing**: All 18 UI tests pass (5 new service tests + 13 existing page/layout tests)

**Testability Benefits**:
- Page tests no longer need to know about HTTP details
- Service tests explicitly show their dependencies
- Factory pattern enables testing with different client configurations
- No magic module-path mocking required for service tests

### 2026-01-28 - Phase 4: Control Surface & Tunable Levers Design

**Created**:
- `ui/src/config/index.ts` - Centralized configuration module with 6 coherent groups
- `ui/src/config/index.test.ts` - 22 validation tests for configuration bounds
- `docs/reference/configuration.md` - User-facing configuration reference

**Refactored**:
- `ui/src/pages/BacklogPage.tsx` - Uses `dataFetchingConfig` and `displayLimitsConfig`
- `ui/src/pages/ScenariosPage.tsx` - Uses `dataFetchingConfig` and `displayLimitsConfig`
- `ui/src/pages/BacklogPage.test.tsx` - Mocks config module for predictable test behavior

**Configuration Groups Designed**:
1. **dataFetchingConfig** - Retry behavior, caching, staleness
2. **displayLimitsConfig** - Tag truncation, pagination sizes
3. **recommendationConfig** - YOLO mode safety, thresholds
4. **insightsConfig** - Pattern detection, confidence thresholds
5. **uiBehaviorConfig** - Debounce, toasts, confirmations
6. **apiConfig** - Timeouts, versioning

**Design Decisions**:
- **NOT exposed** (intentionally internal):
  - HTTP cache policies (internal optimization)
  - Component styling (use Tailwind theme)
  - Type definitions (domain model)
  - Selector IDs (breaking would break tests)

**Testing**: All 40 UI tests pass (22 new config validation + 18 existing)

**Control Surface Benefits**:
- All hard-coded values now centralized in one module
- Clear documentation of impact for each lever
- Bounded values with validation tests
- Mock-friendly for testing with custom configurations

## Decision Points

This section documents the major decision points in the codebase - places where the system chooses between alternatives. Each decision is documented with its location, criteria, inputs, and outcomes.

### Decision Point Categories

| Category | Description | Primary Location |
|----------|-------------|------------------|
| Error Classification | Categorizing errors for recovery path selection | `lib/error-utils.ts` |
| Error Retryability | Deciding whether an error can be retried | `lib/api-client.ts` |
| UI State Rendering | Deciding which UI state to show (loading/error/empty/data) | Pages (`BacklogPage.tsx`, etc.) |
| Status Display | Mapping domain status to visual representation | `types/constants.ts` |
| Configuration | Threshold-based behavior decisions | `config/index.ts` |
| API Response | Deciding HTTP response codes and handling | `api/internal/backlog/handler.go` |
| Routing | Navigating to the correct page/component | `App.tsx` |
| Name Sanitization | Transforming user input to safe format | `api/internal/backlog/handler.go` |

### Well-Extracted Decision Points

These decisions are clearly named, documented, and easy to locate:

#### 1. Error Category Classification (`lib/error-utils.ts`)

**What**: Classify any error into one of 8 categories for recovery path selection.

**Location**: `categorizeError()` function at `lib/error-utils.ts:96-108`

**Criteria**:
- `ApiError` with type `network` → `NETWORK`
- `ApiError` with type `timeout` → `TIMEOUT`
- `ApiError` with HTTP 401/403 → `AUTH`
- `ApiError` with HTTP 404 → `NOT_FOUND`
- `ApiError` with HTTP 400/422 → `VALIDATION`
- `ApiError` with 5xx → `SERVER`
- `ApiError` with type `parse` → `PARSE`
- Non-ApiError with "network" or "fetch" in message → `NETWORK`
- Non-ApiError with "timeout" or "abort" in message → `TIMEOUT`
- Default → `RUNTIME`

**Outcomes**: Each category maps to a recovery path in `RECOVERY_PATHS` constant.

**Testability**: ✅ Pure function, easily unit tested with various error types.

#### 2. Error Retryability (`lib/api-client.ts`)

**What**: Determine if a failed API request should be retried.

**Location**: `ApiError` constructor at `lib/api-client.ts:55-68`

**Criteria**:
- `type === "network"` → retryable
- `type === "timeout"` → retryable
- `isServerError` (5xx status) → retryable
- Client errors (4xx) → NOT retryable
- Parse errors → NOT retryable

**Outcomes**: `ApiError.isRetryable` property used by UI to show retry buttons.

**Testability**: ✅ Clear boolean property, tested in `api-client.test.ts`.

#### 3. YOLO Mode Auto-Approval (`config/index.ts`)

**What**: Decide which recommendations auto-execute in YOLO mode.

**Location**: `recommendationConfig` at `config/index.ts:153-195`

**Criteria**:
- `yoloModeDelayMs`: Safety delay before auto-approval (default: 5s)
- `yoloModeAllowedPriorities`: Only priorities [3, 4, 5] auto-execute
- Priority 1-2 (high priority) requires manual approval even in YOLO mode

**Outcomes**: High-risk recommendations (P1/P2) always require human approval.

**Testability**: ✅ Config values tested in `config/index.test.ts`.

#### 4. Backlog Sorting Order (`api/internal/backlog/handler.go`)

**What**: Determine display order of backlog items in the list.

**Location**: `List()` handler at `api/internal/backlog/handler.go:91-97`

**Criteria**:
1. Sort by priority ascending (P1 before P2)
2. Tie-breaker: sort by updated date descending (newest first)

**Outcomes**: Backlog items appear in priority order with recently-updated items first within each priority.

**Testability**: ✅ Deterministic sorting, tested in handler tests.

#### 5. Backlog Name Sanitization (`api/internal/backlog/handler.go`)

**What**: Transform user-provided backlog name into folder-safe format.

**Location**: `sanitizeName()` function at `api/internal/backlog/handler.go:322-334`

**Criteria**:
- Convert to lowercase
- Replace spaces with hyphens
- Remove characters that aren't alphanumeric or hyphens

**Outcomes**: `"My Awesome Idea!"` → `"my-awesome-idea"`

**Testability**: ✅ Pure function, tested in `handler_test.go#TestSanitizeName`.

### UI State Decisions

The UI uses a consistent pattern for rendering based on data state:

#### Pages Follow This Decision Tree:

```
                    ┌─────────────┐
                    │  isLoading  │
                    └──────┬──────┘
                           │
               ┌───────────┴───────────┐
               │                       │
          true │                  false│
               ▼                       ▼
       ┌───────────────┐       ┌─────────────┐
       │ Loading State │       │   error?    │
       └───────────────┘       └──────┬──────┘
                                      │
                          ┌───────────┴───────────┐
                          │                       │
                     true │                  null │
                          ▼                       ▼
                  ┌───────────────┐       ┌─────────────────┐
                  │  ErrorState   │       │  data?.length   │
                  └───────────────┘       └────────┬────────┘
                                                   │
                                       ┌───────────┴───────────┐
                                       │                       │
                                  === 0│                   > 0 │
                                       ▼                       ▼
                               ┌───────────────┐       ┌───────────────┐
                               │  Empty State  │       │   Data Grid   │
                               └───────────────┘       └───────────────┘
```

**Locations**:
- `BacklogPage.tsx:55-130`
- `ScenariosPage.tsx:54-140`

**Key Distinction**: Empty state (data loaded successfully, zero items) is DIFFERENT from error state (failed to load).

### Error Boundary Decisions

#### 1. App-Level Error Boundary (`App.tsx`)

**What**: Catch catastrophic React errors and show recovery UI.

**Location**: `ErrorBoundary` component wrapping all routes at `App.tsx:45`

**Criteria**: Any unhandled exception in React render/lifecycle methods.

**Outcomes**: Full-page error UI with refresh button.

**Recovery Path**: Page refresh (clears all React state).

#### 2. Page-Level Error Boundary (`App.tsx`)

**What**: Isolate errors to individual pages.

**Location**: `PageErrorBoundary` wrapping each route at `App.tsx:53, 59, 65, 71`

**Criteria**: Unhandled exception in specific page component.

**Outcomes**: Page-specific error UI, other tabs remain functional.

**Recovery Path**: Can navigate to other tabs without refresh.

### API Response Decisions

#### HTTP Status Code Decisions (`api/internal/backlog/handler.go`)

| Condition | Status | Handler Method |
|-----------|--------|----------------|
| Idea not found (GET/PUT/DELETE) | 404 | `Get`, `Update`, `Delete` |
| Idea already exists (POST) | 409 Conflict | `Create` |
| Invalid JSON body | 400 | `Create`, `Update` |
| Missing required fields (name/title) | 400 | `Create` |
| Filesystem read error | 500 | All methods |
| Success (create) | 201 Created | `Create` |
| Success (delete) | 204 No Content | `Delete` |
| Success (read/update) | 200 OK | `Get`, `Update`, `List` |

### Display Mapping Decisions

#### Status-to-Color Mapping (`types/constants.ts`)

**What**: Map idea status to visual indicator color.

**Location**: `IDEA_STATUS_COLORS` at `types/constants.ts:18-26`

```typescript
const IDEA_STATUS_COLORS: Record<IdeaStatus, string> = {
  backlog: "bg-slate-600",      // Neutral gray
  researching: "bg-blue-600",   // Active, in progress
  ready: "bg-green-600",        // Positive, actionable
  queued: "bg-yellow-600",      // Waiting, attention
  in_progress: "bg-purple-600", // Active work
  completed: "bg-emerald-600",  // Success
  archived: "bg-gray-600",      // Inactive
};
```

**Design Intent**: Colors convey status meaning at a glance.

#### Status-to-Icon Mapping (`types/constants.ts`)

**What**: Map scenario status to icon for visual representation.

**Location**: `SCENARIO_STATUS_ICONS` at `types/constants.ts:42-47`

```typescript
const SCENARIO_STATUS_ICONS: Record<ScenarioStatus, LucideIcon> = {
  running: CheckCircle,  // Active and healthy
  stopped: Circle,       // Inactive but normal
  error: AlertCircle,    // Needs attention
  unknown: Circle,       // Indeterminate
};
```

### CLI Endpoint Resolution

**What**: Resolve API v1 endpoint path regardless of configured base URL format.

**Location**: `resolveV1Endpoint()` at `cli/app.go:128-141`

**Criteria**:
- If base URL already ends with `/api/v1` → use path as-is
- Otherwise → prepend `/api/v1` to path

**Example**:
- Base: `http://localhost:3000`, path: `/health` → `/api/v1/health`
- Base: `http://localhost:3000/api/v1`, path: `/health` → `/health`

**Testability**: ✅ Tested in `app_test.go#TestResolveV1Endpoint`.

### Decision Points Needing Attention

These decisions exist but could benefit from further extraction or clarification:

#### 0. Backlog Status Update Guard

**Current Location**: `api/internal/backlog/update_patch.go:validateUpdateBacklogItemRequest()`

**Decision**: Users cannot set backlog status to "queued" or "in_progress" via the update API — these are execution-system-only statuses. The "failed" status is set by the execution service when an agent-manager run fails, and can be manually changed by users (e.g., reset to "backlog" to retry).

**Execution → Backlog Sync**: When an execution's run reaches a terminal state:
- `completed` → backlog status set to "completed"
- `failed` → backlog status set to "failed" (not silently reverted)
- `canceled` → backlog status restored to previous status (user intentionally stopped)

**Status**: Implemented. Guard enforced in the sparse patch validator before backlog items are persisted.

#### 1. Tag Truncation (Inlined in Pages)

**Current Location**: `BacklogPage.tsx:110-120`, `ScenariosPage.tsx:104-114`

**Decision**: Show first N tags, then "+X more" for overflow.

**Status**: Uses `displayLimitsConfig.ideaCardMaxTags` from config, but truncation logic is duplicated across pages.

**Recommendation**: Consider extracting a `TagList` component with built-in truncation logic.

#### 2. Default Priority Assignment (Inlined in Handler)

**Current Location**: `api/internal/backlog/handler.go:157-159`

**Decision**: New backlog items get priority 5 if not specified.

**Status**: Hard-coded in handler. Should this be configurable?

**Recommendation**: Document that priority 5 is the default for new backlog items (lowest priority = safest default).

#### 3. Date Formatting (Inlined in Pages)

**Current Location**: `BacklogPage.tsx:124`

**Decision**: Display dates using `toLocaleDateString()`.

**Status**: Inline browser default formatting.

**Recommendation**: If consistent date formatting is needed, extract to `types/constants.ts` as a `formatDate()` helper.

### Decision Point Testing Coverage

| Decision Point | Unit Tests | Integration Tests | Notes |
|----------------|------------|-------------------|-------|
| Error categorization | ✅ `error-utils.test.ts` | N/A | Pure function |
| Error retryability | ✅ `api-client.test.ts` | N/A | Property tests |
| YOLO mode bounds | ✅ `config/index.test.ts` | N/A | Config validation |
| Idea sorting | ✅ `handler_test.go` | N/A | Go unit test |
| Name sanitization | ✅ `handler_test.go` | N/A | Go unit test |
| UI state rendering | ✅ `BacklogPage.test.tsx` | ❌ Missing | Needs e2e |
| Error boundary | ✅ Implicit | N/A | React built-in |
| HTTP status codes | ✅ `handler_test.go` | N/A | Go unit test |
| Status-to-color | ❌ Missing | N/A | Should add type tests |
| CLI endpoint resolution | ✅ `app_test.go` | N/A | Go unit test |

### 2026-01-28 - Phase 14: Decision Boundary Extraction

**Documented**:
- 10+ major decision points with criteria, inputs, and outcomes
- Decision point categorization (error handling, UI state, API response, etc.)
- UI state decision tree for pages
- Decision points needing attention (tag truncation, default priority, date formatting)
- Decision point testing coverage matrix

**Findings**:
- Most critical decisions are already well-extracted (error handling, configuration)
- UI state rendering follows consistent pattern across pages
- Some minor decisions remain inlined (tag truncation, date formatting)
- Test coverage for decision boundaries is good for error/config, needs improvement for display mappings

**No Code Changes**: This phase focused on documentation and analysis. Existing decision points are well-structured; improvements identified for future phases.

### 2026-01-28 - Phase 15: Cognitive Load Reduction

**Created**:
- `ui/src/components/ui/tag-list.tsx` - Reusable TagList component for tag truncation

**Refactored**:
- `BacklogPage.tsx` - Replaced 14-line tag truncation logic with single TagList component call
- `ScenariosPage.tsx` - Replaced 11-line tag truncation logic with single TagList component call
- `error-state.tsx` - Unified error variant detection to use centralized `categorizeError()` from error-utils.ts

**Simplifications Made**:
1. **Tag truncation pattern elimination**: Previously, both BacklogPage and ScenariosPage had nearly identical 10+ line blocks for tag truncation (slice, map, conditional "+N more"). Now a single `<TagList tags={...} maxTags={...} />` component handles all cases.
2. **Error variant classification unification**: The `getVariantFromError()` function in error-state.tsx duplicated logic from `categorizeError()` in error-utils.ts. Now error-state.tsx imports and uses the centralized function, mapping ErrorCategory to ErrorVariant via a simple constant mapping.

**Testing**: All 115 UI tests pass after refactoring.

## Architecture Clarity Notes

This section records findings from cognitive load reduction efforts, helping future agents understand what has been simplified and what areas still need attention.

### Major Simplifications Made

#### 1. Tag Display Consolidation (Phase 15)

**Before**: Tag rendering with truncation was duplicated across pages:
```tsx
// BacklogPage.tsx:108-121 (14 lines)
{idea.tags && idea.tags.length > 0 && (
  <div className="mt-3 flex flex-wrap gap-1">
    {idea.tags.slice(0, displayLimitsConfig.ideaCardMaxTags).map((tag) => (
      <span key={tag} className="rounded-full bg-slate-700/50 px-2 py-0.5 text-xs text-slate-400">
        {tag}
      </span>
    ))}
    {idea.tags.length > displayLimitsConfig.ideaCardMaxTags && (
      <span className="text-xs text-slate-500">+{idea.tags.length - ...}</span>
    )}
  </div>
)}
```

**After**: Single component call:
```tsx
<TagList tags={idea.tags} maxTags={displayLimitsConfig.ideaCardMaxTags} className="mt-3" />
```

**Impact**: Removed ~25 lines of duplicate code, made tag display behavior consistent and testable in one place.

#### 2. Error Classification Unification (Phase 15)

**Before**: Two separate implementations for error classification:
- `error-utils.ts:categorizeError()` - Returns ErrorCategory (NETWORK, TIMEOUT, etc.)
- `error-state.tsx:getVariantFromError()` - Returns ErrorVariant (network, timeout, etc.)

Both had similar switch/case logic, creating drift risk.

**After**: Single source of truth:
- `error-utils.ts:categorizeError()` - The canonical error classifier
- `error-state.tsx` - Maps ErrorCategory → ErrorVariant via constant

**Impact**: Error classification logic is now in one place; changes to error handling rules only need to happen in error-utils.ts.

### Complexity Hot Spots Identified (Not Yet Addressed)

These areas have higher cognitive load but are stable and rarely modified:

#### 1. Selector Manifest Generation (`selectors.ts:241-306`)

**What**: Recursive tree-flattening algorithms for manifest generation.

**Why It's Complex**:
- `flattenLiteralSelectors()` - Recursively walks nested object tree
- `flattenDynamicSelectors()` - Similar but handles DynamicSelectorDefinition types
- Type guards and generic type parameters

**Why It's Acceptable**:
- This code rarely changes (only when selector system architecture changes)
- Output is validated by tests
- Complexity is inherent to the problem domain (tree flattening)
- Well-documented with clear input/output types

**Recommendation**: Don't simplify unless manifest generation becomes a bottleneck. The complexity is localized and doesn't leak into consuming code.

#### 2. API Client Error Handling (`api-client.ts:148-199`)

**What**: The `request()` method's error handling logic.

**Why It's Complex**:
- Multiple error types to handle (timeout via AbortController, network via TypeError, HTTP errors, parse errors)
- Needs to preserve original error as cause
- Must decide correct ApiError type

**Why It's Acceptable**:
- This is the single place where HTTP errors are classified
- Each branch is well-documented
- The complexity is essential - these are genuinely different failure modes
- Heavily tested in api-client.test.ts

**Recommendation**: Keep as-is. The complexity is necessary and well-contained.

### Areas Where Cognitive Load is Still High

#### 1. Page Component Structure

**Observation**: BacklogPage and ScenariosPage have similar patterns:
- useQuery hook setup
- Header with search/filter
- Loading state
- Error state
- Empty state
- Data grid

**Current Status**: The patterns are similar but not identical enough to extract without over-abstraction. Each page has domain-specific rendering (idea cards vs scenario cards).

**Recommendation**: If a third similar page is added, consider extracting a `ListPage` layout component. For now, the duplication is acceptable.

#### 2. Config Module Size (`config/index.ts`)

**Observation**: The config module is 363 lines with 6 configuration groups.

**Current Status**: Well-organized with clear groupings and documentation. Each config value has documented impact and range.

**Recommendation**: Keep as-is. The length is due to thorough documentation, not complexity. A new developer can understand any config value by reading its section.

### File-Level Cognitive Load Ratings

| File | Complexity | Readability | Notes |
|------|------------|-------------|-------|
| `BacklogPage.tsx` | Low | High | Clear data flow, no nested conditions |
| `ScenariosPage.tsx` | Low | High | Same pattern as BacklogPage |
| `error-state.tsx` | Low | High | Simple mapping from error → display |
| `error-utils.ts` | Medium | High | Well-documented, many categories |
| `api-client.ts` | Medium | High | Complex but essential error handling |
| `selectors.ts` | High | Medium | Tree algorithms, but stable |
| `config/index.ts` | Low | High | Long but well-documented |

### Guidelines for Future Simplifications

1. **Extract when duplication is verbatim**: If two places have >10 lines of identical code, extract.
2. **Don't extract near-duplicates**: Similar-but-different code often becomes harder to maintain when forced into a single abstraction.
3. **Favor explicit over clever**: A 10-line explicit function is better than a 3-line clever one.
4. **Document why complex code is necessary**: If code can't be simplified, explain why in comments.
5. **Measure before optimizing**: Don't simplify stable code that nobody touches.

### 2026-01-28 - Phase 17: Architecture Alignment & Refactoring (Screaming Architecture Audit)

**Audit Scope**: Verified alignment between documented mental model and actual implementation.

**Findings (Architecture Alignment Assessment)**:

| Aspect | Documented | Actual | Status |
|--------|------------|--------|--------|
| UI types module | Domain-organized types | ✅ `types/domain.ts`, `types/constants.ts` | Well-Aligned |
| UI services layer | Service seams with interfaces | ✅ `services/backlog-service.ts`, `services/scenarios-service.ts`, `services/settings-service.ts`, `services/recommendations-service.ts` | Well-Aligned |
| UI lib module | Separated concerns | ✅ `api-client.ts`, `error-utils.ts`, `query-utils.ts` | Well-Aligned |
| UI config module | Centralized configuration | ✅ `config/index.ts` with 6 groups | Well-Aligned |
| API structure | Domain-organized internal packages | ✅ `internal/backlog/handler.go` | Well-Aligned |
| CLI structure | Domain-organized commands | ✅ Grouped by Health, Backlog, Config | Well-Aligned |

**Improvements Made**:

1. **Removed deprecated `lib/api.ts`**: This file was a backward-compatibility shim marked `@deprecated` that re-exported from `api-client.ts`. No code used it (all imports were from the proper modules). Updated `lib/index.ts` to import directly from `api-client.ts`.

2. **Removed empty `api/internal/scenarios/` directory**: This was a placeholder directory with no files. Empty directories can be confusing and suggest incomplete work. (Scenarios endpoints were implemented in later phases.)

3. **Updated SEAMS.md API Module Structure section**: The documentation said "Only main.go exists with all code in one file" but the actual structure had evolved to use `internal/backlog/handler.go`. Updated to reflect current state and added target structure reference for future domains.

**Documentation Health Findings**:

| Area | Status | Notes |
|------|--------|-------|
| docs/manifest.json | ✅ Present | Navigation structure defined |
| Mental model documented | ✅ Yes | ARCHITECTURE.md with flows and layers |
| Code↔Doc references | 12 refs | DOC: comments in code, CODE: links in docs |
| Orphaned docs | 0 files | All docs in manifest |
| Broken references | 0 found | All links valid |

**Architecture Screams Its Purpose**:

The codebase structure clearly expresses what swarm-manager does:
- `ui/src/pages/BacklogPage.tsx` - Backlog management
- `ui/src/pages/ScenariosPage.tsx` - Scenario catalog
- `ui/src/pages/RecommendationsPage.tsx` - Recommendation engine
- `ui/src/pages/SettingsPage.tsx` - User preferences
- `api/internal/backlog/` - Backlog CRUD backend
- `cli/app.go` - CLI with Backlog and Health commands

The top-level structure makes it obvious this is a scenario management dashboard with a backlog, scenario catalog, and recommendation capabilities.

**Testing**: All UI tests pass. All Go tests pass (backlog handler: 12 tests, CLI: 11 tests). Build succeeds.

### 2026-01-28 - Phase 17, Iteration 2: Documentation Drift Cleanup

**Findings**: Architecture documentation referenced removed file (`lib/api.ts`).

**Documentation Drift Fixed**:
1. **SEAMS.md UI-to-API seam diagram** - Updated lib/ structure to show current files (removed api.ts, added error-utils.ts and query-utils.ts)
2. **SEAMS.md UI Module Structure** - Updated lib/ listing to show current files instead of deprecated api.ts
3. **UTILS_UNIFICATION_NOTES.md** - Updated utility architecture diagram to reflect current lib/ structure

**Why This Matters**: Documentation drift creates confusion for future agents and developers. Architecture diagrams must match actual file structure so readers can trust the documentation.

**No Code Changes**: This iteration focused solely on fixing documentation that lagged behind Phase 17 Iteration 1's code changes.

**Testing**: All 115 UI tests pass. All Go tests pass. UI build succeeds. UI smoke test passes.

## Observability Surface

This section documents the signals, logs, and feedback mechanisms that make the scenario's behavior observable to users, operators, and agents.

### Key States & Transitions

The scenario has the following observable states:

| Component | States | Transitions | Observable Signals |
|-----------|--------|-------------|-------------------|
| **Idea** | backlog, researching, ready, queued, in_progress, completed, failed, archived | Create → backlog; Update status; Delete; Execution failure → failed | API logs, HTTP status codes, UI status indicators |
| **API Server** | starting, running, degraded | Startup, health checks | Health endpoint, request logs |
| **UI** | loading, error, empty, data | Data fetch lifecycle | Loading indicators, error states, empty states |

### Signal Inventory

#### API Signals (`api/internal/backlog/handler.go`)

| Operation | Success Signal | Failure Signals |
|-----------|---------------|-----------------|
| **Create idea** | `[backlog] created: "name" (priority=N, status=S)` | `[backlog] create: invalid request body`<br>`[backlog] create: missing required fields`<br>`[backlog] create: conflict - idea "name" already exists`<br>`[backlog] create: failed to create directory` |
| **Update idea** | `[backlog] updated: "name"`<br>`[backlog] updated: "name" (status=A→B, priority=X→Y)` | `[backlog] update: not found "name"`<br>`[backlog] update: invalid request body`<br>`[backlog] update: failed to save` |
| **Delete idea** | `[backlog] deleted: "name"`<br>`[backlog] delete: "name" (already gone, no-op)` | `[backlog] delete: failed to remove` |
| **Request lifecycle** | `[METHOD] /path duration` (via middleware) | HTTP error responses |

**Log Format**: All operation logs use the pattern `[backlog] {action}: {context}` for easy grep/filter.

#### UI Signals (`ui/src/lib/error-utils.ts`)

| Category | Console Level | Format | Purpose |
|----------|---------------|--------|---------|
| **Error logging** | `console.error` | `[CATEGORY] {structured JSON}` | Machine-parseable error tracking |
| **Success logging** | `console.info` | `[OUTCOME] {structured JSON}` | Operation completion tracking |

**Error Categories** (8 total):
- `NETWORK` - Connection failures → retry with backoff
- `TIMEOUT` - Request timed out → retry with backoff
- `AUTH` - Session expired/forbidden → re-authenticate
- `NOT_FOUND` - Resource missing → navigate away
- `SERVER` - Server error (5xx) → retry later
- `VALIDATION` - Bad input → fix and resubmit
- `PARSE` - Invalid response → report bug
- `RUNTIME` - Unexpected error → refresh page

**Success Outcomes** (5 types):
- `CREATED` - New resource created
- `UPDATED` - Existing resource modified
- `DELETED` - Resource removed
- `FETCHED` - Data loaded successfully
- `COMPLETED` - Operation finished

**Structured Log Entry Fields**:
- `timestamp` - ISO 8601 timestamp
- `category` or `outcome` - Error/success type
- `message` - Human-readable summary
- `correlationId` - Unique ID for tracing (format: `err_<timestamp>_<random>` or `op_<timestamp>_<random>`)
- `source` - Component or module name
- `status` - HTTP status (errors only)
- `retryable` - Whether operation can be retried (errors only)
- `context` - Additional metadata (no sensitive data)

#### CLI Signals (`cli/app.go`)

| Command | Success Output | Error Output |
|---------|---------------|--------------|
| `status` | `Status: ok`, `Ready: true`, dependencies | Connection error message |
| `backlog list` | `Found N backlog item(s)` or `No backlog items found.` | Parse/connection errors |
| `backlog get` | Formatted backlog item details | `usage:` message, not found |
| `backlog create` | `Created backlog item: name` with details | Validation, conflict errors |
| `backlog update` | `Updated backlog item: name` with details | Validation, not found errors |
| `backlog delete` | `Deleted backlog item: name` | Not found, permission errors |

### UI Feedback Patterns

The UI provides clear visual feedback for all states:

```
┌─────────────────────────────────────────────────────────────────┐
│                     DATA FETCH LIFECYCLE                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│   ┌─────────┐       ┌─────────┐       ┌─────────┐              │
│   │ Loading │──────▶│  Error  │──────▶│  Retry  │──────┐       │
│   └────┬────┘       └────┬────┘       └─────────┘      │       │
│        │                 │                              │       │
│        ▼                 │                              │       │
│   ┌─────────┐            │                              │       │
│   │  Empty  │◀───────────┴──────────────────────────────┘       │
│   └────┬────┘                                                   │
│        │                                                        │
│        ▼                                                        │
│   ┌─────────┐                                                   │
│   │  Data   │                                                   │
│   └─────────┘                                                   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

**Error State Components**:
- `ErrorState` - User-friendly error display with:
  - Variant-specific icons (WifiOff, Clock, ServerCrash, AlertCircle)
  - Clear titles and messages
  - Retry buttons for recoverable errors
  - Recovery guidance from `RECOVERY_PATHS`

**Empty State Pattern**:
- Distinct from error state (successful fetch, zero results)
- Friendly messaging ("No backlog items yet")
- Clear call-to-action ("Create First Idea")

### Observability Gaps (Signal Debt)

The following areas lack sufficient observability:

| Gap | Impact | Priority | Recommended Fix |
|-----|--------|----------|-----------------|
| No request tracing IDs | Cannot correlate frontend errors to backend logs | Medium | Add X-Request-ID header propagation |
| CLI lacks --verbose mode | Hard to debug CLI issues | Low | Add verbose flag to output debug info |
| No health check degraded state | Binary healthy/unhealthy, no partial failure visibility | Low | Add dependency-specific health status |

### Signal Consumption

**For Operators/Agents**:
- Grep API logs with `[backlog]` prefix for operation tracking
- Parse structured JSON logs from UI console output
- Use correlation IDs to trace errors across layers

**For Users**:
- Loading indicators show active operations
- Error states with retry buttons for recoverable issues
- Success confirmation messages after mutations (when implemented)

### Testing Signal Emission

Critical signals are tested to ensure stable observation:

| Signal | Test Location | What's Asserted |
|--------|---------------|-----------------|
| Error categorization | `error-utils.test.ts` | All 8 categories correctly classified |
| Error log structure | `error-utils.test.ts` | JSON format, required fields present |
| Success log structure | `error-utils.test.ts` | JSON format, outcome types, correlation IDs |
| Recovery paths | `error-utils.test.ts` | Each category has appropriate guidance |
| HTTP status codes | `handler_test.go` | Correct codes for each scenario (201, 204, 400, 404, 409, 500) |

### Seam Change Trail

| Date | Author | Change |
|------|--------|--------|
| 2026-01-28 | Claude (Phase 20) | Created Observability Surface documentation; enhanced API logging with operation context; added structured success logging utility |
| 2026-01-28 | Claude (Phase 24) | Seam Discovery & Enforcement - added scenarios-service tests, created ecosystem client seam interface, documented seam patterns |

## Phase 24: Seam Discovery & Enforcement

### Seam Improvements Made

#### 1. Scenarios Service Test Coverage (UI)

**Problem**: The `scenarios-service.ts` had a clean seam interface (`IScenariosService`) but no tests exercising it, unlike `backlog-service.ts` which had comprehensive tests.

**Solution**: Added `scenarios-service.test.ts` with 6 tests covering:
- List scenarios (success and empty cases)
- Get single scenario by name
- Update metadata (isGreenfield, recommendationsEnabled, both fields)

**Location**: `ui/src/services/scenarios-service.test.ts`

**Benefit**: The seam is now verified by tests. Future changes to the service interface will be caught by test failures.

#### 2. Ecosystem-Manager Integration Seam (Go API)

**Problem**: The `backlog/handler.go` had hardcoded HTTP calls to ecosystem-manager in `createEcosystemTask()`, making it impossible to test the Queue handler without a running ecosystem-manager instance.

**Solution**: Use `internal/ecosystem` client interface as the seam:

```go
// Client is the interface for ecosystem-manager operations.
// This is the seam that allows the integration to be substituted for testing.
type Client interface {
    CreateTask(ctx context.Context, req CreateTaskRequest) (string, error)
}
```

**Changes Made**:
1. Centralized integration contract in `internal/ecosystem` package
2. Added `ecosystem.Client` field to `backlog.Handler` for dependency injection
3. Refactored `createEcosystemTask()` to use injected client if available
4. Default client uses `api-core/discovery` for dynamic ports
5. Tests inject mock client for isolation

**Testing Pattern**:
```go
// In tests, inject a mock client
mockClient := &mockEcosystemClient{
    createTaskFunc: func(ctx context.Context, req ecosystem.CreateTaskRequest) (string, error) {
        return "task-123", nil
    },
}
handler := NewHandlerWithClient(tempDir, mockClient)

// Now Queue handler can be tested without network
```

**Benefit**: Queue functionality can now be unit tested in isolation, without needing ecosystem-manager running.

### Seam Architecture Summary

The scenario now has a clean layered seam architecture:

```
┌──────────────────────────────────────────────────────────────────┐
│                          UI Layer                                │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Pages ──────► Services (IBacklogService, IScenariosService)    │
│                     │                                            │
│                     ▼ [Seam #1: Service Interface]               │
│                                                                  │
│             API Client (IApiClient)                              │
│                     │                                            │
│                     ▼ [Seam #2: HTTP Interface]                  │
│                                                                  │
│                  fetch()                                         │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────────┐
│                          API Layer                               │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│   Handlers ─────► httputil (JSON, BadRequest, NotFound, etc.)    │
│       │                                                          │
│       │                                                          │
│       ├─────► Filesystem (direct os.* calls - acceptable)        │
│       │                                                          │
│       └─────► EcosystemClient [Seam #3: Integration Interface]   │
│                     │                                            │
│                     ▼                                            │
│              ecosystem-manager HTTP API                          │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

### Seam Testing Coverage

| Seam | Interface | Test File | Test Count |
|------|-----------|-----------|------------|
| Backlog Service | `IBacklogService` | `backlog-service.test.ts` | 5 tests |
| Scenarios Service | `IScenariosService` | `scenarios-service.test.ts` | 6 tests |
| API Client | `IApiClient` | `api-client.test.ts` | 12 tests |
| HTTP Utilities | (functions) | `response_test.go` | 16 tests |
| Ecosystem Client | `EcosystemClient` | `client_test.go` | 13 tests |
| ID Generator | `randRead` (package var) | `idgen_test.go` | 2 tests |

### Future Seam Opportunities

The following areas could benefit from additional seams but are acceptable as-is:

1. **Filesystem Operations**: Direct `os.*` calls in handlers. Could be abstracted for testing, but current tests use `t.TempDir()` effectively.

2. **Time/Clock**: Handlers use `time.Now()` directly. Could inject a clock interface for deterministic testing, but not yet needed.

3. **Logging**: Direct `log.Printf()` calls. Could inject a logger interface, but current structured logging is sufficient.

These are documented here for future consideration when test complexity demands better isolation.

### Validation Boundaries (added 2026-03-23)

| Boundary | Location | Behavior | Test |
|----------|----------|----------|------|
| Execution mode validation | `queue_ops.go`, `batch_queue_handler.go` | Handler-layer validation returns 400 for invalid modes. Proto also validates via `buf.validate` for single-item queue. Batch queue uses raw JSON so handler validation is essential. | `TestBatchQueue_InvalidMode`, `TestQueue_InvalidMode` |
| Initiative item ref format | `initiatives/service.go:AddItems()` | Service validates `kind/name` format before accepting. Handler translates to 400. | `TestService_AddItems_InvalidFormat`, `TestHandler_AddItems_InvalidFormat` |
| Research context path validation | `backlog/research.go` | Validates context paths exist via `os.Stat()` before adding to prompt. Missing paths skipped with log warning, not hard failure. | Research proceeds without invalid paths |
| Create atomicity | `backlog/handler.go:Create` | Uses `os.Mkdir` (not `MkdirAll`) for atomic conflict detection — eliminates TOCTOU race between stat and create. Falls back to parent creation + retry if parent dir doesn't exist. | Existing create/conflict tests |
| Spec.json merge safety | `backlog/store.go:SaveItem` | Logs warning on malformed existing spec.json instead of silently discarding. Prevents silent metadata loss during batch operations. | Observable via log output |
| Strict backlog import contract | `backlog/batch_handler.go`, `backlog/handler.go`, `initiatives/handler.go` | Request bodies reject unknown fields. Legacy `scope` payloads fail fast at the HTTP boundary instead of being silently translated. | `TestBatchCreate_RejectsUnknownField`, `TestCreate_RejectsUnknownField`, `TestUpdate_RejectsUnknownField`, `TestHandler_Create_RejectsUnknownField`, `TestHandler_Update_RejectsUnknownField` |
| Batch-create preview and rollback | `backlog/batch_handler.go` | Preview validates multi-initiative imports without writes. Real create applies initiative metadata first, then rolls back both items and initiative changes if membership assignment fails. | `TestBatchCreate_PreviewDoesNotMutateDiskOrInitiatives`, `TestBatchCreate_InitiativeAddItemsFails_RollsBackEverything` |
| Cycle detection contract | `backlog/batch_handler.go`, `batch_queue_handler.go` | Both batch-create and batch-queue use `depgraph.DetectCycle()` for informative error messages (cycle path included) before `TopologicalSort()`. | `TestBatchQueue_CycleDetection_ShowsPath` |
| Error truncation boundary | `batch_handler.go`, `batch_queue_handler.go` | All handler-layer system error messages use `httputil.TruncateErrorMessage(err, 240)` to prevent information leakage. Validation errors (kind/name/title) are not truncated. | Consistent with `queue_ops.go` pattern |
| Dependency helper locality | `backlog/queue_ops.go` | `appendDependencyBlockingReasons` consolidated into `queue_ops.go` (its only caller). Batch-queue uses separate `computeUnmetDependencies` for batch-context-aware dependency checking (tracks items queued within the batch). | `TestBatchQueue_PartialSuccess_DependencyChain` |

### Output Tab Composition Seam (updated 2026-04-02)

Replaced the Activity Timeline Drawer with an inline Output tab on the BacklogDetailsPage. The Output tab is the composition root for all execution output data.

| Boundary | Location | Behavior | Test |
|----------|----------|----------|------|
| Output tab composition root | `ui/src/components/backlog/output-tab.tsx` | Composes `LatestExecutionSummary`, `ScenarioReviewResults`, and `ActivityTimeline` into a single tab view. Receives all data via props — no internal hook calls. | `output-tab.test.tsx` |
| Latest execution summary | `ui/src/components/backlog/latest-execution-summary.tsx` | Persistent card showing most recent execution status. Always renders (empty/active/completed states). Replaces the vanishing `BacklogActiveRunBanner`. | `latest-execution-summary.test.tsx` |
| Scenario review results | `ui/src/components/backlog/scenario-review-results.tsx` | Displays target scenario chips and post-run review status (via `PostRunStatusBadge`). Extracted from `BacklogScenariosPanel` which now only shows scenario chips. | `scenario-review-results.test.tsx` |
| Client-side timeline merge | `ui/src/hooks/useActivityTimeline.ts` | Pure `mergeTimeline()` function groups activities by `executionId`, attaches as children to parent executions, places orphan activities as standalone entries, and sorts newest-first. Two parallel `useQuery` calls (executions + activities) provide the data. Polling gated on `activeTab === "output"`. | `useActivityTimeline.test.ts` |
| Timeline content rendering | `ui/src/components/detail/ActivityTimeline.tsx` | Renders unified chronological feed of executions (with nested activities) and standalone activities. Reuses `PostRunStatusBadge` for finalization display. Now rendered inline in Output tab instead of inside a Drawer. | `ActivityTimeline.test.tsx` |
| Header trigger | `ui/src/pages/BacklogDetailsPage.tsx` | History icon button in both mobile and desktop headers opens the drawer. Mobile subtitle is also clickable. Replaces the old `executionHistorySection` collapsible. | Manual + BacklogDetailsPage tests |

### Review Evidence Boundary (added 2026-04-02)

Post-execution evidence gathering system. A review agent produces typed evidence (screenshots, API tests, CLI output) stored in `review/` within the backlog item directory. Triggered after finalization, policy-gated via `review_agent_enabled`, non-blocking and non-fatal to finalization.

| Boundary | Location | Behavior | Test |
|----------|----------|----------|------|
| Review store (file I/O) | `api/internal/review/store.go` | `LoadRounds()`, `SaveRound()`, `LoadCapture()`, `SaveCapture()` — mirrors `workshop/workshop.go` pattern with `review/round-NNN.json` files and `review/captures/` directory. Path traversal protection on capture I/O. | `review/store_test.go` (14 tests) |
| Review service | `api/internal/review/service.go` | `StartReviewForExecution()`, `VerifyEvidence()`, `RequestMoreEvidence()`, `ContinueRequest()`, `DismissRequest()`, `TriggerReviewAgent()` — orchestrates agent spawning, round management, and event emission. | `review/service_test.go` |
| Review handler | `api/internal/review/handler.go` | 7 HTTP endpoints under `/api/v1/backlog/{kind}/{name}/review/` and `/api/v1/execution/{id}/trigger-review-agent`. Thin handlers delegating to service. | `review/handler_test.go` |
| Finalization integration | `api/internal/execution/finalization.go` | `FinalizationPhaseEvidenceGathering` phase runs after GCT reviewing, before `finishFinalization()`. Policy-gated via `isReviewAgentEnabled()`. Failure produces warning, doesn't block finalization. | Finalization tests |
| ReviewServiceIntegration interface | `api/internal/execution/service.go` | Interface with `StartReviewForExecution()` — injected via `SetReviewService()` to avoid import cycles between execution and review packages. | Execution service tests |
| Evidence panel (UI) | `ui/src/components/backlog/evidence-panel.tsx` | Displays review rounds with typed evidence cards, verification checkboxes, and "Request More" button. Renders between ScenarioReviewResults and ActivityTimeline in Output tab. | — |
| Evidence item card (UI) | `ui/src/components/backlog/evidence-item-card.tsx` | Type-specific rendering: screenshots (inline thumbnail), API tests (pass/fail list), CLI output (code block), recordings, config diffs. Verification checkbox toggle. | — |
| Review Zustand store | `ui/src/stores/review-store.ts` | Minimal store for Request More panel open/close state. Round data lives in React Query cache. | — |
| Review API service | `ui/src/services/review-service.ts` | Typed client for all review endpoints. Follows `backlog-service.ts` pattern. | — |
| Event log events | `api/internal/eventlog/types.go` | 7 new events: `review.started`, `review.evidence_added`, `review.evidence_verified`, `review.request_created`, `review.request_fulfilled`, `review.round_completed`, `review.failed` | — |
| Stats engine | `api/internal/stats/engine.go` | Processes review events to compute `ReviewStats`: rounds completed, avg evidence per round, verification rate, request-more rate, avg duration. | `stats/engine_test.go` |
| CLI commands | `cli/cmd_review.go` | `review-list`, `review-verify`, `review-request`, `review-trigger` — thin API wrappers with human-friendly default output. | — |
| Prompt skill | `prompt-manager/store/skills/packs/core/swarm-manager-review/` | `SKILL.md` instructs the review agent on evidence strategy, output schema, classification rules, and Request More mode. | — |

### Operating Mode Runner Boundary (updated 2026-04-30)

Initiative non-default modes now have a backend runner seam instead of routing
phase starts through feedback or item-level execution.

| Boundary | Location | Behavior | Test |
|----------|----------|----------|------|
| Mode service | `api/internal/operatingmode/service.go` plus focused files in `api/internal/operatingmode/` | Switches modes with explicit item-execution cancellation and active-round guards, computes backend-authoritative phase actions, validates registered phase transitions before reserving rounds, acquires the shared initiative lock through an injectable lock seam, renders the phase skill through prompt-manager, spawns AgentManager through agent-activity when wired, persists run IDs, refreshes terminal state, parses structured phase result envelopes, emits phase/backlog-sync/replan events, and releases locks. | `operatingmode/service_test.go` |
| Result parser | `api/internal/operatingmode/output.go` | Parses final `operating_mode_result` JSON envelopes from agent summaries. Supported outputs are artifact writes, handoffs, readiness, phased-plan progress, verdicts, replan flags, and backlog-sync plans. | `operatingmode/output_test.go` |
| Backlog reconciliation | `api/internal/operatingmode/backlog_reconciler.go` + `api/routes_operating_mode.go` | Run-id-validated boundary for marking member backlog items complete from non-default mode rounds. Agents must call this API instead of editing backlog `spec.json`; every call records a structured source payload on backlog and operating-mode events. | `operatingmode/service_test.go`, `routes_operating_mode_test.go` |
| Mode handler | `api/internal/operatingmode/handler.go` | Thin HTTP layer for workspace read, mode switch, phase start, round refresh, round cancel, and complete-items under `/api/v1/initiatives/{name}/operating-mode/...`. | Service-backed route compile coverage |
| Dependency adapters | `api/routes_operating_mode.go` | Keeps `operatingmode` from importing `initiatives` or `backlog`, avoiding cycles with the existing initiative metadata and prompt-catalog tests. | `go test . -run '^$'` |
| Prompt skills | `prompt-manager/store/skills/packs/core/swarm-manager-holistic-loop-*` and `swarm-manager-phased-plan-*` | Prompt-manager runtime skills for the eight registered mode phases. Each skill asks for a structured final result envelope so the runner can persist artifacts and handoffs; phase start fails closed if the exact registered skill cannot be rendered. | `prompt-manager skill read ...` |
| Round view model | `ui/src/components/initiative/operating-mode/round-view-model.ts` | Pure TypeScript parser/decision boundary for round payload fields, pending completion refs, proposal mutations, applied sync state, default selected mutation IDs, and disabled action reasons. | `round-view-model.test.ts` |
| Backlog sync action component | `ui/src/components/initiative/operating-mode/backlog-sync-actions.tsx` | Owns proposal mutation selection state and the apply button. It receives a normalized proposal and does not parse raw payloads. | `operating-mode-panel.test.tsx` |

### Agent Session Boundary (added 2026-05-01)

Durable conversational workflows now have their own backend seam instead of
being modeled as capture, backlog, or initiative subtypes.

| Boundary | Location | Behavior | Test |
|----------|----------|----------|------|
| Session store | `api/internal/agentsessions/store.go` | Persists `agent-sessions/sess_*/session.json` snapshots plus append-only `messages.jsonl` and `artifacts.jsonl`, with proposal JSON files under `proposals/`. Lists are hydrated from those logs and sortable/filterable for the sidebar. | `agentsessions/store_test.go` |
| Session service | `api/internal/agentsessions/service.go` | Owns create/list/get/continue/refresh/cancel/proposal/artifact operations. Spawns and continues runs through Agent Activity with `owner_type=session`, injects session environment variables, maps Agent Manager run status to session lifecycle state, persists run summaries as assistant transcript messages, and resolves `run_id -> session` for provenance enrichment. | `agentsessions/service_test.go` |
| Session HTTP API | `api/internal/agentsessions/handler.go` | Thin proto-JSON HTTP boundary for `/api/v1/agent-sessions` and `/api/v1/artifacts/by-entity`. Proposal apply routes through typed service appliers instead of agents mutating entities directly. | `agentsessions/handler_test.go` |
| Agent Activity session owner | `api/internal/agentactivity/types.go`, `api/internal/agentactivity/service.go` | Adds `OwnerSession` and session purposes so session-owned Agent Manager work is not mislabeled as initiative or backlog work. | `agentactivity` package tests plus `agentsessions/service_test.go` |
| Agent Manager session spawn | `api/internal/agentmanager/service.go` | Adds `SessionSpawnRequest` and `SpawnSession` as a narrow concrete service method. The broad `agentmanager.Service` handler interface does not include it, preventing unrelated test doubles and consumers from depending on session spawning. | `agentmanager` package tests |
| Session lifecycle events | `api/internal/eventlog/types.go`, `api/internal/eventlog/emitter.go` | Emits `agent_session.*` lifecycle, proposal, and artifact events through the existing eventlog pipeline so stats aggregate session adoption and outcomes from events rather than UI state. | `eventlog` package tests |
| Session stats | `api/internal/stats/*`, `ui/src/surfaces/graph/components/StatsPanel.tsx` | Aggregates `SessionStats` from event streams: sessions by kind/status, proposal apply rates, created artifacts, messages per session, time to first proposal, failed-session rate, and session-created backlog/initiative counts. The Stats panel exposes these in a compact Sessions tab. | `stats/handler_test.go`, `StatsPanel.test.tsx` |
| Session provenance resolver | `api/internal/identity/session.go`, `api/main.go` | Enriches verified agent provenance with session metadata by resolving request `run_id` through the Agent Session service. Missing or failed lookups preserve the original provenance and never reject a request. | `identity/middleware_test.go`, `agentsessions/service_test.go` |
