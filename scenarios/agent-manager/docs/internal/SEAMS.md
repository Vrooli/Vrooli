# Agent Manager — Architectural Seams

This document maps every testability boundary in the codebase: what's mockable, where the seam lives, and the test that exercises it.

The four big seams added by the Phase 6 refactor:
- **`emit.Gate`** — single Emit choke point for run events.
- **`core.Runner` + `Codec`** — generic runner pipeline + per-model codec.
- **`phases/*` functions** — per-phase logic with explicit input structs.
- **`config.Levers`** — single home for adjustable thresholds.

For the architectural overview and design invariants, see [`../concepts/ARCHITECTURE.md`](../concepts/ARCHITECTURE.md).

## Consumption and charge projection seam

Runner codecs emit `UsageEventData` for consumption on every observed usage
shape, then emit an independent `ChargeEventData` when billing evidence exists.
`invocationreadmodel.ProjectRun` folds those payloads separately; pricing is
never allowed to gate token or turn telemetry. Historical fused JSON is
normalized in `domain.NormalizeEventPayloadJSON` before decoding. The pricing
service reads a lifecycle-refreshed price book and never performs provider I/O
in a request path. The architecture test
`internal/archtest/pricing_wiring_test.go::TestCostTrackingCodecsHaveChargeSource`
guards production wiring. Billing mode and workload identity are stamped into
the run snapshot at creation and are not reconstructed from mutable policy.

## Architectural Principles

The codebase follows three key architectural principles:

1. **Decision Boundary Extraction** — Key decisions are explicit, named, and testable.
2. **Cognitive Load Reduction** — Code is organized to minimize mental overhead.
3. **Control Surface Design** — Tunable levers are organized, validated, and documented.

## Enforced Import Boundaries

`api/internal/archtest` turns the following architectural rules into build
failures rather than conventions:

- `workflowruntime` is a leaf runtime and must not import `orchestration`.
- Runtime orchestration code depends on repository interfaces and must not
  import the concrete `database` package. Test-only helpers are excluded from
  this rule because cross-package integration tests import them directly.
- Handler persistence is limited to documented CQRS read-side exceptions:
  `PricingHandler` reads pricing/statistics projections, `EventsHandler` reads
  the typed event log, and `OperationalStatsHandler` reads the operational
  statistics projection. Mutation handlers go through their service boundary.

## Task Path Decision

Tasks remain a **first-class agent-manager domain**. The task lifecycle is
actively consumed by the scenario CLI, the Tasks UI, and run-detail lookups;
its status transitions also anchor run ownership and operator review. It is
therefore neither a declared-run projection nor a removal candidate. The
decision was verified on 2026-07-23 by searching fleet consumers and the
scenario's CLI/UI surfaces.

## Folder Structure

Agent-manager uses a **screaming architecture** where folder structure clearly expresses domain intent:

```
api/internal/
├── domain/                 # Core domain entities and decisions
│   ├── types.go            # Task, Run, AgentProfile, Policy entities
│   ├── errors.go           # Domain-specific error types
│   ├── decisions.go        # Explicit decision helpers (state machines, classification)
│   └── validation.go       # Entity validation logic
├── orchestration/          # Coordination layer
│   ├── run_executor.go     # Thin coordinator (~560 LOC). Owns shared state + phase ordering.
│   ├── recovery.go         # Restart-resume entry points
│   ├── reconciler.go       # Stale-run detection + orphan reaper
│   ├── service.go          # HTTP-facing service surface
│   ├── phases/             # Per-phase functions with explicit input structs
│   ├── emit/               # The single Emit choke point (Gate)
│   └── integration/        # End-to-end recovery regression gate
├── adapters/
│   ├── runner/
│   │   ├── core/           # Generic Runner pipeline (one impl, four model bindings)
│   │   ├── codecs/         # Per-model codecs: claude, codex, opencode, grok
│   │   │                   #   base.go (embeddable baseCodec + shared helpers),
│   │   │                   #   pricing.go (runner-agnostic cost seam)
│   │   ├── interface.go    # Runner / EventSink / TranscriptParser interfaces
│   │   └── launcher.go     # Process launcher (host / sandbox)
│   ├── sandbox/            # workspace-sandbox HTTP adapter
│   ├── event/              # Event streaming and storage
│   └── artifact/           # Diff and artifact collection
├── policy/                 # Policy evaluation logic
├── repository/             # Persistence interfaces
├── handlers/               # HTTP handlers (thin presentation layer)
└── config/
    └── levers.go           # Tunables struct — single home for adjustable thresholds
```

## Core Seams

### Durable analytics read model

**Purpose:** Make cross-run statistics stable after raw event retention and
keep every analytical question on one compute path.

**Contract:** `runreport` is the only fold from operational `run_events` into
`invocation_read_model_facts`, `invocation_read_model_errors`, and
`invocation_read_model_runs`. Projection replacement advances facts, errors,
terminal run summary, signals, and watermark atomically. Typed Measures RPCs
are the machine contract; legacy statistics routes are compatibility readers
over the same projection and may only join product metadata (tasks/profiles).

**Freshness:** A watermark records last event id/time, classifier version, and
projection time per run. Missing watermarks are explicitly unprojected; replay
never silently fabricates pruned source events.

**Consumers:** Measures RPC/CLI, statistics UI, export, pricing model catalog,
and retained stats compatibility routes. `repository_stats_test.go` seeds one
snapshot and proves status, success, duration, costs, breakdowns, tools,
errors, time series, and model catalog against projected data.

### Agent Run Environment

**Purpose:** Keep runner process environment ownership explicit.

**Contract:**
- `VROOLI_AGENT_IDENTITY_TOKEN` is the only identity variable Agent Manager injects into spawned agent processes.
- Sandboxed runs also receive `VROOLI_SANDBOX_ID`, `VROOLI_SANDBOX_MERGED`, and `VROOLI_SANDBOX_SCOPE` so lifecycle-aware tools can resolve the copy-on-write workspace.
- Agent Manager must not synthesize API-base variables for Swarm Manager, Prompt Manager, Workspace Sandbox, Agent Manager, or any other scenario. Scenario CLIs discover their own APIs through `cli-core` lifecycle discovery, and scenario APIs use `api-core/discovery` for peer calls.

**Testability:**
- `api/internal/orchestration/phases/env_test.go` locks token-only identity env, sandbox env, and merge precedence.

### 0. Realtime Event Delivery (`handlers` + UI store)

**Purpose:** Keep list-level run status live without streaming every run event body to every browser.

**Backend boundary:**
- `api/internal/handlers/websocket.go`
- `WebSocketHub` owns connection registration and fanout.
- `WebSocketClient.shouldReceive` is the broadcast filter:
  - `RUN_STATUS` is global metadata for run lists.
  - `TASK_STATUS` has no run id and remains global.
  - `RUN_EVENT` and `RUN_PROGRESS` require explicit run subscription or all-event subscription.
- Per-client subscription fields are guarded by the client mutex; all mutation must go through `updateSubscription` or `updateAllEventsSubscription`.

**UI boundary:**
- `ui/src/App.tsx` owns WebSocket ingress and dispatches normalized messages into `useRunEventStore`.
- `ui/src/lib/runEventStore.ts` owns ordering, dedupe, subscriptions, and gap-fill intent.
- `ui/src/hooks/useSelectedRunController.ts` owns selected-run event subscription and REST gap-fill for the current detail view.
- `ui/src/lib/webSocketConnection.ts` owns pure reconnect decisions used by `useWebSocket`.

**Testability:**
- Backend filtering/race coverage: `api/internal/handlers/websocket_test.go`
- Store and connection decisions: `ui/tests/lib/runEventStore.test.ts`, `ui/tests/lib/webSocketConnection.test.ts`

---

### 1. Runner Adapter (`adapters/runner`)

**Purpose:** Abstract agent execution across different runners.

**Interface:** `runner.Runner`
```go
type Runner interface {
    Type() domain.RunnerType
    Capabilities() Capabilities
    Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)
    Continue(ctx context.Context, req ContinueRequest) (*ExecuteResult, error)
    Stop(ctx context.Context, runID uuid.UUID) error
    IsAvailable(ctx context.Context) (bool, string)
}
```

**Why it's a seam:**
- Different runners (claude-code, codex, opencode) have different APIs and behaviors
- Enables testing with mock runners
- New runners added by implementing the interface, not modifying orchestration

**Implementations:**
- `MockRunner` - Simulates agent execution for testing (implemented)
- `StubRunner` - Placeholder for unavailable runners (implemented)
- `DefaultRegistry` - Runner registry with registration and lookup (implemented)
- `MockRunner` / `StubRunner` / `DefaultRegistry` (implemented)
- Four codecs wired through `core.Runner` (all implemented ✅): `Claude`
  (`claude --print --output-format stream-json`), `Codex` (`codex exec --json`),
  `OpenCode` (`opencode run --format json`), `Grok`
  (`grok -p … --output-format streaming-json`).

**Dependencies:**
- Receives: `ExecuteRequest` with profile, task, working directory
- Produces: `ExecuteResult` with canonical `RunResult`, its compatibility-only
  summary projection, metrics, and exit code
- Side effects: Streams `RunEvent` to `EventSink`

---

### 1a. StructuredResult resolver + extractor (`internal/structuredresult`)

**Purpose:** Turn the canonical final handoff into an optional typed value
without trusting provider formatting or creating a second classifier model.

**Seam shape:**
- `NormalizeSpec` validates the closed `result-spec/v1` JSON Schema subset,
  canonicalizes schema bytes, and calculates the persisted digest.
- `Resolver.Resolve` owns the deterministic parse → local validate → optional
  constrained extraction → local revalidate ladder.
- `Extractor` accepts only bounded source/schema plus a portable role and
  returns an untrusted candidate with provider/policy provenance.
- Production wires `GatewayExtractor`, which sends `extract.structured`
  through the active role-policy catalog and existing runner registry. It runs
  a one-turn extraction request and returns only a deterministic JSON candidate;
  `Resolver` still owns authoritative local validation.

**Boundary rule:** An extractor error, outage, or abstention is a structured
`abstained` result. An extractor candidate that fails local validation is
`invalid`. Neither path can manufacture success, and diagnostic text never
includes provider errors or source output.

---

### 1b. Codec + baseCodec + Pricing (`adapters/runner/codecs`)

**Purpose:** Isolate the per-runner glue (CLI args, stdout decode, classify,
metrics, cost) from the generic `core.Runner` pipeline. One file per runner
implements `Codec`; `core.Runner` owns launch/scan/transcript/events.

**Seam shape:**
- `Codec` (`codecs/codec.go`) — the per-runner contract.
- `baseCodec` (`codecs/base.go`) — embeddable struct every codec composes for
  the genuinely-identical surface: binary resolution + availability
  (`resolveBinary`), `Type`/`BinaryDescription`/`TagEnvKey`/`Labels`, the
  available-gated `ProbeModel` default, `ContinueTag`, the drain-to-EOF
  `OnEarlyTerminate` and no-op `PostClassify` defaults, and `ParseTranscriptLine`
  (delegating to each codec's `NewTranscriptParser` via a wired `newParser`
  func). Free helpers `standardBuildEnv` / `testBase` cover env construction and
  the `*ForTest` variant. Each `*.go` codec contains ONLY its unique surface
  (BuildArgs, stream decode, Classify, UpdateMetrics, cost).
- Pricing seam (`codecs/pricing.go`) — runner-agnostic `PricingService` /
  `PricingCostRequest` / `PricingCostCalculation` + `buildCostEvent(runID,
  runnerType, pricing, model, tokens)`. Parameterised by `RunnerType` so any
  codec whose CLI reports tokens-but-not-cost (codex today) shares one
  cost-event builder. Claude/OpenCode report cost natively and bypass it.

**Why it's a seam:**
- Adding a runner = one new `codecs/<name>.go` embedding `baseCodec` (+ enum and
  registry plumbing), not a copy of the availability/probe/labels/transcript
  boilerplate. Adding Grok was the forcing function that extracted it.

**Control-flag seam:** `Codec.ControlArgs` owns the runner-native argv for
model, effort, allowed tools, and denied tools. Both `BuildArgs` and the
interactive substrate consume it, so a new runner adds its translation in its
codec rather than adding a runner-type branch to orchestration.

**Replay seam:** `cmd/fake-agent` is a test-only executable built once by
`internal/testutil.BuildFakeAgent`. Process replay uses it with recorded codec
corpora to exercise the real launcher, tag environment, transcript writer,
terminal persistence, and sandbox attribution without invoking a real runner
or opening a network connection.

**Role-policy resolution and capability gates:**
- `config/role-policy-catalog.json` declares portable roles and ordered
  `(runner, resourceRole)` candidates only. `internal/rolepolicy` strictly
  parses it, rejects semantic drift with field-specific diagnostics, and
  identifies an immutable revision as `sha256:<64 lowercase hex characters>`
  over the exact source bytes.
- Raw codecs advertise measured runner mechanics and runtime-discovered model
  namespaces. They do not compose a static model inventory: concrete selection
  is resolved by the owning resource at run creation and retained only in the
  immutable execution snapshot.
- `codecs/model_parity_test.go` gates valid role-catalog runner references
  without importing concrete model data into codec capability output.
- `runner/runnertype_conformance_test.go` requires the RunnerType set to match
  `domain.ValidRunnerTypes()` and the generated proto enum.

**Capability honesty (Grok):** grok's headless stdout surfaces assistant text +
session id but NOT tool-call/result events or token/cost — even when a tool
runs. `Grok.Capabilities()` reflects exactly that (no tool events, no cost);
`codecs/capabilities_conformance_test.go` pins the honest contract.

**Tests:** `codecs/{claude,codex,opencode,grok}_test.go`, `golden_test.go`,
`classify_test.go`, `capabilities_conformance_test.go`, plus the two drift gates.

---

### 1aa. Role-policy Activation State (`internal/rolepolicy.State`)

**Purpose:** Own exactly one validated active catalog revision and make catalog
failure/reload state observable without letting orchestration create
independent caches.

**Contract:**
- `NewState(path, requirement)` always returns a state object, even when the
  initial load fails. The failure is retained as a structured diagnostic with
  the resolved path.
- `Validate()` parses a candidate without activation.
- `Reload()` parses and semantically validates outside the lock, then swaps the
  active immutable `*Revision` under one lock. A failed reload records the
  attempt and retains the prior active revision.
- `Status()` returns path, requirement reason, readiness, active digest,
  activation time, and the latest reload attempt as a deep copy.
- `Active()` is the only runtime revision read. Orchestration receives this
  same state through its role-policy resolver seam.

**Production wiring:** `api/main.go` constructs the state before accepting
runs and registers a critical role-policy readiness check. A role catalog
contains only portable `(runner, resourceRole)` candidates; each resource
resolves concrete model/fallback data when a run is created.

**Tests:** `internal/rolepolicy` state and resolver contract tests.

---

### 1ab. Run Role Resolution Snapshot (rolepolicy → orchestration)

**Purpose:** Convert profile plus inline role intent into a durable,
explainable candidate sequence before the run row is created.

**Contract:**
- Runtime profile, API, scenario-config, and execution surfaces accept no
  legacy preset input; authority is `roleRef` plus the resulting snapshot.
  SQLite startup applies only the current declarative schema and never rewrites
  operator data.
- Creation-time preflight resolves resource-owned candidates in catalog order,
  recording each concrete runner/model result and any unavailable candidate.
- RunConfig.PolicySnapshot persists catalog digest, role ref, full ordered
  candidates, selected index/candidate, source/precedence explanation, and
  preflight evidence inside the existing resolved_config JSON column.
- The generated RunConfig policy_snapshot field is output-only in practice:
  create requests use RunConfigOverrides, which has no snapshot field. Proto
  conversion round-trips the stored snapshot so run-detail clients see the
  exact persisted decision.
- Historical resolved_config documents without policySnapshot stay readable
  with a nil snapshot. They are not reconstructed from the active catalog;
  doing so would falsely attribute a mutable current revision to a past run.

**Production wiring:** Orchestrator.resolveRunConfig receives the one
rolepolicy state/resolver injected at boot and writes the snapshot before
RunRepository.Create. Reload changes new resolutions only; an already-built
snapshot owns copied candidate values and its original digest.

**Tests:** internal/rolepolicy resolution tests,
internal/database/repository_test.go::TestRunWithComplexFields, plus
internal/protoconv/convert_test.go::TestExecutionPolicySnapshotRoundTrip.

**Execution:** policy-backed runs walk only persisted
PolicySnapshot.Candidates starting at the creation-time selected index or the
later candidate already projected into ResolvedConfig before a restart.
Unavailable runners are skipped, model-unavailable results advance the
sequence, cross-runner candidates are acquired from the runner registry, and
runner_default clears the launch model only at the adapter boundary. Each state
emits policy.candidate.attempt with catalog digest and snapshot index; total
exhaustion emits the terminal exhausted outcome and returns a typed
RUNNER_UNAVAILABLE error. The snapshot itself is never mutated; ResolvedConfig
runner/model are persisted runtime projections of the current attempt. Resume
therefore retries the interrupted candidate at least once without replaying
earlier candidates or consulting live policy. Historical rows with no snapshot
remain readable and use their stored single runner/model projection; they do
not receive a snapshot fabricated from current policy.

**Execution tests:** internal/orchestration/phases/execute_test.go covers
cross-runner fallback, unavailable-runner skips, typed terminal exhaustion,
JSON round-trip restart/resume, and an atomic catalog reload between attempts.

---

### 1ac. Role-policy Operator Surface (rolepolicy → handlers/CLI/UI)

Handlers inject the same role-policy activation owner used by readiness and
orchestration. Generated API responses expose status, active catalog
inspection, non-activating validation, atomic reload, and profile/run
explanation. Roles are sorted deterministically for diff-friendly output.

The former `runner models`, `runner models-update`, and
`/api/v1/runner-models` routes are intentionally absent. Catalog inspection is
available only through the generated `role-policy catalog` contract.
Reload accepts no catalog payload, so it cannot become a competing desired-state
writer. Profile explanation uses the same resolver/preflight path as run
creation. Run explanation returns nil for historical rows without snapshots
rather than reconstructing provenance from the active catalog. The CLI group is
`agent-manager role-policy`; the Settings UI is read-only.

**Tests:** role-policy handler and CLI contract tests.

---

### 1ad. Desired-permission Control Plane (`permissionpolicy` → handlers/CLI/UI)

`internal/permissionpolicy.State` atomically owns the declared portable
permission catalog; `Service` owns deterministic resource planning,
authorization-gated reconciliation, and metadata-only audit evidence. Resource
CLIs remain the only native-file writers. The generated API and CLI expose only
whole-document status, catalog, validate, reload, plan, reconcile, and doctor
operations; the Settings **Permissions** tab uses those same endpoints.

Required hard-enforcement rules have a readiness contract beyond catalog
syntax: current reconcile evidence must match the active digest and report a
native or hook-backed enforcing resource. Optional unavailable resources stay
visible in plan/audit evidence without degrading health. The UI requires an
explicit authorization acknowledgement before it enables reconcile, and it
does not offer individual-rule or native-syntax mutation.

The structured-log surface records `permission_policy_reloaded`, plan/doctor
observations, `permission_policy_reconcile_started`, and reconcile completion.
Stable attributes carry only catalog digest, resource/drift/unsupported counts,
hard-enforcement state, missing rule IDs, and partial-failure state—never rule
patterns or copies of native files.

**Tests:** `internal/permissionpolicy/service_test.go`, permission-policy
handler tests, and UI lint/type checks.

---

### 1b. Flag Validator (`adapters/runner`)

**Purpose:** Validate runner-specific CLI flags against runner allowlists without coupling orchestration to runner internals.

**Interface:** `runner.FlagValidator`
```go
type FlagValidator interface {
    ValidateFlags(runnerType domain.RunnerType, flags []string) error
    AllowedFlags(runnerType domain.RunnerType) []string
    SupportedFeatures(runnerType domain.RunnerType) []string
}
```

**Why it's a seam:**
- Decouples flag validation from runner execution
- Orchestration validates without knowing runner internals
- Runners declare capabilities (features + allowed flags) via `Capabilities()`
- `MockFlagValidator` for testing, `RegistryFlagValidator` for production

**Implementations:**
- `RegistryFlagValidator` — Derives allowlists from `Capabilities()` (production)
- `MockFlagValidator` — Configurable validation via func fields (testing)

**Related types:**
- `domain.FeatureFlags` — Typed feature flags (e.g., `EnableBrowser`) mapped to runner-specific CLI args
- `domain.RunnerExtraFlags` — Per-runner validated extra CLI flags (`map[RunnerType][]string`)
- `Capabilities.SupportedFeatures` — Which typed features a runner supports
- `Capabilities.AllowedExtraFlags` — Allowlist of extra CLI flags a runner accepts

---

### 1c. Process Launcher (`adapters/runner` + `adapters/sandbox`)

**Purpose:** Decouple where the agent process actually executes from how the runner builds its argv/env. The runner pipeline (transcript parsing, idle-timer reset, heartbeat) consumes a `LaunchedProcess` whose stdout/stderr/exit semantics are identical regardless of whether the process runs on the host or inside a workspace-sandbox container.

**Interface:** `runner.Launcher`
```go
type Launcher interface {
    Launch(ctx context.Context, req LaunchRequest) (LaunchedProcess, error)
}

type LaunchedProcess interface {
    Stdout() io.Reader
    Stderr() io.Reader
    ResetIdleTimer()
    TimedOut() bool
    Kill()
    Signal(grace time.Duration)
    Wait() error
    PID() int
}
```

**Why it's a seam:**
- Protected mode (`SandboxConfig.Mode == Protected`) routes the agent process tree itself through workspace-sandbox `/processes` so bwrap isolation, network mode, and the git-verb allowlist are enforced at the OS level — not just on the merged-overlay output. Tracking mode runs the agent on the host with overlay tracking only.
- Without the Launcher seam, every runner would carry an inline `if mode == Protected { ... } else { ... }` switch around `exec.CommandContext`. The seam reduces "add a new runner" to picking a `Launcher` rather than copying a switch statement.
- A factory pattern (`SandboxLauncherFactory`) keeps the runner package free of any sandbox import. The sandbox provider implements the factory, so the runner asks for a launcher per-call without taking on a sandbox-package dependency. This avoids a sandbox→runner→sandbox cycle that would otherwise force an interface bloat.

**Implementations:**
- `runner.HostLauncher` — wraps `os/exec.CommandContext` and the existing `managedProcess` machinery (process group, idle timeout, grandchild cleanup). Default for non-protected runs and for protected runs whose factory is missing or returns nil.
- `sandbox.SandboxLauncher` — POSTs to workspace-sandbox `/api/v1/sandboxes/{id}/processes` (with `withStdin: true` when stdin is supplied), streams `req.Stdin` directly to `/processes/{pid}/stdin?close=true`, opens **two SSE connections** to `/processes/{pid}/logs/stream?stream=stdout|stderr` for push-based byte delivery, and consumes the terminal `event: exit` frame to surface structured `ExitInfo` (exit code + signal + OOMKilled) as `*remoteExitError`. DELETEs `/processes/{pid}` on `Kill()`. Surfaces the workspace-sandbox structured 403 (git-allowlist denial) as a typed `*sandbox.LaunchBlocked`. **No client-side polling.**
  - **Path translation (2026-04-28):** SandboxLauncher is the *only* place that translates host paths to in-namespace paths. The `WorkingDir` field of `runner.LaunchRequest` and any value in the env map that exactly matches the sandbox merged dir are rewritten to `SandboxNamespacePath` (`/workspace`) before posting. A workdir that is neither under the merged dir nor under `/workspace` is rejected as `*LaunchBlocked{Code:"workdir_outside_sandbox"}` — sandbox-routed launches must run inside the namespace, full stop. The constant must stay aligned with the `--bind <merged> /workspace` arg in `workspace-sandbox/api/internal/driver/bwrap.go`. See `translateHostPathToNamespace` in `internal/adapters/sandbox/sandbox_launcher.go`.
  - **Missing exit info (2026-04-28):** When both SSE log streams close without `event: exit`, the launcher surfaces `ErrSandboxNoExitInfo` from `Wait` instead of treating the run as a clean success. Pre-fix this race let bwrap launch failures (chdir, missing exec) masquerade as exit-0 successes.
- `sandbox.WorkspaceSandboxProvider` — implements `runner.SandboxLauncherFactory.LauncherFor(sandboxID)` so the provider can be passed to a runner constructor as the protected-mode factory.

**Routing logic (shared in `runner.launcherSelector`, used by every runner / every path):**
```go
cfg := req.GetConfig()
if cfg == nil || cfg.SandboxConfig == nil || cfg.SandboxConfig.Mode.Effective() != domain.SandboxModeProtected {
    return host  // tracking-mode runs stay on the host
}
if factory == nil          { warn-and-fallback }
if sandboxID == nil        { warn-and-fallback }
if launcher := factory.LauncherFor(*sandboxID); launcher != nil {
    return launcher        // protected-mode + factory wired → sandbox path
}
warn-and-fallback           // factory returned nil → host path with explicit log
```

Every fallback path emits a `runner.warn` log event so misconfigured environments are visible rather than silently downgraded.

**Testability:**
- `HostLauncher` contract tests: `runner/launcher_test.go` (7 tests; spawns real `/bin/echo` / `/bin/cat`).
- `launcherSelector` routing tests: `runner/launcher_selector_test.go` (11 tests covering Pick + PickFor for tracking, protected, factory-missing, sandbox-ID-missing, factory-returned-nil, etc.).
- `SandboxLauncher` lifecycle tests: `sandbox/sandbox_launcher_test.go` — an `httptest.Server` simulator implements POST processes, POST stdin, GET SSE logs, DELETE processes, and drives the structured exit frame. 7+ lifecycle tests, no real workspace-sandbox required.
- Per-runner routing tests pin the protected-vs-host fork end-to-end: `runner/codex_runner_routing_test.go` (6), `runner/opencode_runner_routing_test.go` (6); claude_code routing is covered transitively by selector tests + integration tests.
- The workspace-sandbox `/processes` git-allowlist enforcement (the symmetric counterpart of `/exec`) is tested in `workspace-sandbox/api/internal/handlers/process_start_git_allowlist_test.go` (4 tests).

**Status of the runner-fork rollout (2026-04-28):** Every coding-agent runner (claude_code, codex, opencode) routes **every launch path** through the shared `launcherSelector`: streaming Execute, durable-transcript Execute, durable-transcript Continue, streaming Continue. Default `SandboxConfig.Mode` is `Protected`. The four ws-sb-* follow-on items (split stdout/stderr, native stdin pipe, SSE streaming, structured exit codes) shipped together with this rollout, so the launcher carries no compromises today.

**See also:** `scenarios/agent-manager/docs/PROTECTED_MODE_RUNNERS.md` for the per-runner capability matrix; `scenarios/workspace-sandbox/docs/EXECUTION_MODES.md` for the workspace-sandbox `/processes` capability matrix this seam consumes.

---

### 2. Sandbox Provider (`adapters/sandbox`)

**Purpose:** Abstract sandbox creation and lifecycle management.

**Interface:** `sandbox.Provider`
```go
type Provider interface {
    Create(ctx context.Context, req CreateRequest) (*Sandbox, error)
    Get(ctx context.Context, id uuid.UUID) (*Sandbox, error)
    Delete(ctx context.Context, id uuid.UUID) error
    GetWorkspacePath(ctx context.Context, id uuid.UUID) (string, error)
    GetDiff(ctx context.Context, id uuid.UUID) (*DiffResult, error)
    Approve(ctx context.Context, req ApproveRequest) (*ApproveResult, error)
    Reject(ctx context.Context, id uuid.UUID, actor string) error
    PartialApprove(ctx context.Context, req PartialApproveRequest) (*ApproveResult, error)
    ApplyAtRunEnd(ctx context.Context, req ApplyAtRunEndRequest) (*ApplyAtRunEndResult, error)
    TurnCheckpoint(ctx context.Context, req TurnCheckpointRequest) (*TurnCheckpointResult, error)
    Stop(ctx context.Context, id uuid.UUID) error
    Start(ctx context.Context, id uuid.UUID) error
    Resume(ctx context.Context, id uuid.UUID) (*Sandbox, error)
    IsAvailable(ctx context.Context) (bool, string)
}
```

**Why it's a seam:**
- Isolates workspace-sandbox API specifics from orchestration
- Enables testing without actual sandbox creation
- Could support alternative isolation mechanisms (containers, VMs)

**Implementations:**
- `WorkspaceSandboxProvider` - HTTP client for workspace-sandbox API (implemented ✅)
  - Creates sandboxes with overlayfs isolation
  - Retrieves diffs and applies changes
  - Exposes distinct final apply-at-run-end and continuable `/turn-checkpoint` calls
  - Resumes checkpointed sandboxes before continuation process launch
  - Reports post-run apply/checkpoint outcome through `Run.FinalizationStatus` so sandbox infrastructure failures do not masquerade as active runner execution
  - Supports full/partial approval workflows
  - Health checks for availability monitoring

**Related interface:** `sandbox.LockManager`
- Manages scope-based locking for concurrent runs
- Prevents overlapping path scopes from conflicting

### 2a. Workspace Sandbox Ensurer (`orchestration` / `phases`)

**Purpose:** Ensure the `workspace-sandbox` scenario is available at the moment a sandboxed run needs it.

**Interface:** `phases.WorkspaceSandboxEnsurer`
```go
type WorkspaceSandboxEnsurer interface {
    EnsureAvailable(ctx context.Context) error
}
```

**Why it's a seam:**
- Keeps `sandbox.Provider` focused on workspace-sandbox HTTP operations.
- Keeps run-time dependency recovery out of agent-manager process bootstrap; Vrooli lifecycle still owns declared dependency startup when agent-manager itself starts.
- Allows setup/finalization tests to simulate stopped or unhealthy workspace-sandbox without launching real services.
- Coalesces same-process start attempts while Vrooli lifecycle provides cross-process scenario locking.

**Production implementation:**
- `orchestration.CommandWorkspaceSandboxEnsurer` checks provider health, runs `vrooli --no-stale-check scenario start workspace-sandbox` when unavailable, then polls health until bounded by `Sandbox.EnsureStartTimeout`.
- It must never execute workspace-sandbox binaries directly (`./api/scenario-api`, launcher scripts, `nohup`, etc.).

**Call sites:**
- Fresh sandbox creation calls this seam when the provider health check reports unavailable or transient create transport failures occur.
- Post-turn `/turn-checkpoint` and pre-response transport failures from `/apply-at-run-end` call it once before retrying within the sandbox retry bounds.
- Existing sandbox reuse and in-place runs do not call this seam.

---

### 3. Event System (`adapters/event`)

**Purpose:** Abstract event capture, storage, and streaming.

**Interfaces:**

`event.Store` - Persistence:
```go
type Store interface {
    Append(ctx context.Context, runID uuid.UUID, events ...*domain.RunEvent) error
    Get(ctx context.Context, runID uuid.UUID, opts GetOptions) ([]*domain.RunEvent, error)
    Stream(ctx context.Context, runID uuid.UUID, opts StreamOptions) (<-chan *domain.RunEvent, error)
    Count(ctx context.Context, runID uuid.UUID) (int64, error)
    Delete(ctx context.Context, runID uuid.UUID) error
}
```

`event.Collector` - Capture during execution:
```go
type Collector interface {
    Log(level, message string)
    Message(role, content string)
    ToolCall(toolName string, input map[string]interface{})
    ToolResult(toolName string, output string, err error)
    StatusChange(oldStatus, newStatus string)
    Metric(name string, value float64)
    Artifact(artifactType, path string)
    Error(code, message string)
    Flush(ctx context.Context) error
}
```

**Why it's a seam:**
- Decouples event production from consumption
- Enables different storage backends (PostgreSQL, Redis, files)
- Allows real-time streaming vs. batch collection
- Runners produce events without knowing how they're stored

**Implementations:**
- `SQLiteStore` - SQLite-backed event storage with streaming support (implemented, `adapters/event/sqlite.go`). Appends use an immediate SQLite transaction before sequence allocation so concurrent writers for the same run cannot race on `MAX(sequence)`.
- `orchestration.appendAndBroadcastEvents` - shared durable delivery helper. It appends first, then broadcasts the persisted event with assigned ID/sequence. Durable event paths must go through this helper or a sink that delegates to it; broadcasting an event that failed to append is a contract violation.
- `orchestration.runEventSink` / `broadcastingEventSink` - runner-facing event sink creation. The sink chooses append-and-broadcast when both store and broadcaster exist, append-only when only the store exists, and no-op when event storage is absent.

**Realtime UI seam:**
- `ui/src/lib/webSocketProtocol.ts` owns WebSocket wire parsing/building.
- `ui/src/lib/webSocketSubscriptions.ts` owns durable subscription intent and replay after reconnect.
- `ui/src/lib/runEventStore.ts` is the pure reducer for run snapshots, run events, per-run last sequence, dedupe, reconnect intent, and REST gap-fill reconciliation.
- `ui/src/hooks/useRunEventStore.ts` is the React integration seam. `App.tsx` dispatches `run_status`, `run_event`, and `task_status` messages into this store; `RunsPage.tsx` reads selected-run snapshots/events from it instead of maintaining a parallel timeline state machine.

---

### 3a. Typed-Operational Event Log (`internal/eventlog`)

**Purpose:** Carry structured operational signals (runner/model fallback,
persisted policy-candidate execution,
sandbox lifecycle outcomes, heartbeat misses, checkpoint failures,
model/runner health transitions) as typed payloads on the existing
`run_events` table — so consumers can query patterns instead of grepping
log strings.

**Interfaces:**

```go
// eventlog package
type Payload interface { /* closed set; markers in types.go */ }

func BuildEvent(runID uuid.UUID, payload Payload) (*domain.RunEvent, error)
func Register(eventType domain.RunEventType, schemaVersion int, factory PayloadFactory)
func Decode(eventType domain.RunEventType, schemaVersion int, body json.RawMessage) (Payload, error)

type Repository interface {
    SinceForRun(ctx context.Context, runID uuid.UUID, afterSeq int64, limit int) ([]Record, error)
    SinceID(ctx context.Context, afterID int64, limit int) ([]Record, error)
    ByEventType(ctx context.Context, eventType domain.RunEventType, since time.Time, limit int) ([]Record, error)
}
```

**Why it's a seam:**

- Typed payloads round-trip JSON through `domain.TypedEventData` so the
  emit + persist path is the same one legacy `LogEventData` events use.
  No parallel write path. Single emission choke point at `emit.Gate`
  remains the gate.
- `(event_type, schema_version)` dispatch decouples payload evolution
  from consumer code: adding a field is non-breaking; renaming bumps the
  version and registers a new entry; old rows decode through the old
  entry forever.
- Stats engine (Phase 3) and health audit (Phase 2) read through
  `eventlog.Repository`, never through the legacy `event.Store.Get`
  default branch — the Repository's contract is "typed events, decoded".

**Implementations:**

- Phase 1: `SQLiteRepository` over the existing `run_events` table
  (extended with a `schema_version` column).
- Phase 2 added the persisted health audit consumer.
- Phase 3 added the stats engine consumer (next seam below).

**Authoritative references:**

- `internal/eventlog/types.go` — payload struct definitions.
- `internal/eventlog/dispatch.go` — registry init().
- `docs/internal/EVENT_TAXONOMY.md` — full event list, JSON shapes, and
  versioning rules.

---

### 3b. Operational Stats Engine (`internal/stats`)

**Purpose:** Incrementally aggregate typed-operational events into the
metrics surfaced by `/api/v1/stats/operational`, `/api/v1/stats/fallback`,
and the `agent-manager ops` CLI. Watermark + checkpoint pattern so the
engine resumes after a process restart instead of replaying the entire
event log from zero.

**Interfaces:**

```go
// stats package
type Engine struct { /* opaque; see engine.go */ }
func NewEngine(repo eventlog.Repository, checkpoint CheckpointStore, name string) *Engine

func (e *Engine) Rebuild(ctx context.Context) error  // resume from saved checkpoint
func (e *Engine) Refresh(ctx context.Context) error  // advance watermark only
func (e *Engine) GetSummary() Summary                 // bundled view
func (e *Engine) GetFallback() FallbackInsights       // standalone fallback view
// + GetHealth, GetSandbox, GetHeartbeat, GetCheckpoint, GetRetry

type CheckpointStore interface {
    Load(ctx context.Context, name string) (int64, error)
    Save(ctx context.Context, name string, rowid int64) error
}

type Processor func(state *aggregateState, rec eventlog.Record)
func RegisterProcessor(eventType domain.RunEventType, schemaVersion int, p Processor)
```

**Why it's a seam:**

- **Schema-version dispatch table** (not switch). Processors are keyed
  by `(event_type, schema_version)`. Adding a new event without wiring
  a processor is caught by `TestAllEmittedEventsAreProcessed`; bumping
  a schema version requires registering the new pair, the old pair keeps
  decoding through its registered processor.
- **Resumable replay.** Rebuild resumes from `stats_checkpoint.last_rowid`,
  not zero. `TestRebuildResumesFromCheckpoint` pins this contract: a
  fresh engine sharing the same checkpoint store processes only the
  events appended since the saved watermark.
- **Typed Category enum at the HTTP edge.** `OperationalStatsHandler`
  parses `?category=…` against a closed set; unknown values return HTTP
  400 with the known-categories list, never an empty-but-200 response.
  `TestOperationalHandler_BadCategory400` pins this.
- **Honesty contract.** Every response carries `history.{has_history,
  history_days, min_sample_meaningful}` so consumers can refuse to
  draw conclusions from thin samples.
- **Policy execution remains an overlay.** `policy.candidate.attempt` processors
  count outcomes and failure classes for diagnostics. These counters never
  mutate catalog policy, health state, or persisted run snapshots.

**Implementations:**

- `Engine` plus `SQLiteCheckpointStore` over `stats_checkpoint`.
- Processors live in `internal/stats/processors.go`, one per typed
  event. Adding a new event is: register the payload in `eventlog`,
  register a processor here.

**Authoritative references:**

- `internal/stats/engine.go` — engine, watermark, checkpoint plumbing.
- `internal/stats/registry.go` — processor map, RegisteredProcessorKeys.
- `internal/stats/processors.go` — per-event aggregation logic.
- `internal/handlers/operational_stats.go` — Category enum + 400 path.
- `internal/database/schema.sql` — `stats_checkpoint` table.

**Tests pinning the contract:**

- `internal/stats/engine_test.go::TestAllEmittedEventsAreProcessed`
- `internal/stats/engine_test.go::TestRebuildResumesFromCheckpoint`
- `internal/stats/engine_test.go::TestEngine_AggregatesFallbackInsights`
- `internal/handlers/operational_stats_test.go::TestOperationalHandler_BadCategory400`

---

### 3bb. Durable Invocation Analytics Read Model (`internal/invocationreadmodel`)

**Purpose:** Preserve the versioned tool-invocation evidence and terminal run
summaries used for friction and throughput analysis after source `run_events`
retention expires. This is the authoritative substrate for ownership, retry,
help-recovery, repeated work, file rereads, per-tool outcomes, run success,
cycle time, cost, and volume questions across the run corpus.

**Ownership and freshness:** `runreport.DeriveInvocationFacts` remains the
single classifier fold. `orchestration` starts a fire-and-forget projection
when a run reaches a terminal state; projection failures are logged and never
alter the run transition. Facts, watermark, terminal run summary, and reread
signals advance in the same transaction, so no analytical surface can observe
a partially advanced run. A refresh reports `skipped`, `refreshed`,
or `unreplayable`; an unreplayable run has no retained source events and keeps
its prior facts and classifier version unchanged.

**Consumer surfaces:**

- `GET /api/v1/runs/invocation-facts/{metrics,aggregate,cohort}` supplies the
  corpus analytics and bounded run-id selector.
- `agent-manager run invocation-{metrics,aggregate,cohort}` exposes the same
  predicate for operators; replay and refresh remain maintenance commands.
- `run report` reads durable facts when available, retaining its on-demand
  derivation fallback only for historical runs that were never projected.
- The Statistics UI's friction overview consumes the metrics endpoint and its
  shared time-window state.
- Investigation creation accepts the same selector predicate instead of only
  pre-known run ids; a truncated selector is rejected so it cannot be mistaken
  for complete evidence.

**Retained product boundary:** the existing operational statistics UI and
`StatsSummary` machine contract remain while per-question parity is incomplete.
They must not be removed or silently re-derived from raw event JSON until that
parity gate passes. Trend views and full UI migration remain tracked work.

**Tests:** `internal/invocationreadmodel/contract_test.go`,
`internal/adapters/database/repository_investigation_projection_test.go`, and
`internal/orchestration/invocation_read_model_test.go` pin projection,
migration, aggregate/cohort, and replay behaviour.

---

### 3c. Fallback Classifier (`internal/fallback`)

**Purpose:** Single source of truth for "why did the runner or model reject this attempt?" Replaces the regex `runner.ClassifyModelError` and 3-value `ModelErrorKind` (deleted 2026-05-07, Phase 2).

**Interface:** `fallback.Classifier`
```go
type Classifier interface {
    Classify(in ClassifyInput) *ClassifiedError
}
```

Implementations: `fallback.TextClassifier` (residual safety net) plus per-codec `Classify` methods on `claude.Claude`/`codec.Codex`/`codec.OpenCode`. Codecs consult their own structured signals (HTTP status, exit code, JSON event fields) before delegating to the text classifier.

**Wired in:**
- `runner.Runner.Classify(stderr, exitCode)` — exposed by `core.Runner` via codec delegation; consumed by `phases/execute.go::classifyExecutionOutcome` and the health probe.
- `health.NewProbe(..., classifier, ...)` — defaults to `TextClassifier`; injectable for tests.

**Why a seam:**
- Decouples codec native-signal classification from the executor's "what to do next" logic.
- The closed `Reason` enum (14 values) replaces freeform error strings as the wire-form on `runner.fallback.attempted`/`model.fallback.attempted` events.
- `Recovery(reason)` is exhaustively tested against `AllReasons()` so adding a Reason without updating the action map is a CI failure (`TestReasonRecoveryActionExhaustive`).

**Tests:**
- `internal/fallback/fallback_test.go::TestTextClassifier_DispatchTable`, `TestReasonRecoveryActionExhaustive`.
- `internal/adapters/runner/codecs/classify_test.go::Test{Claude,Codex,OpenCode}_Classify` per-codec assertions.

DOC: `internal/fallback/reason.go`, `docs/internal/ERROR_SEMANTICS.md` (Reason taxonomy + recovery actions).

---

### 3d. Health Audit Store (`internal/health`)

**Purpose:** Persisted SQLite-backed audit log of model + runner health observations. Health snapshots survive API restart and remain an availability overlay, never desired state.

**Interface:** `*health.Store`
```go
RecordModel(ctx, runner, modelID, status, reason, message, triggeredBy) error
RecordRunner(ctx, runner, status, reason, message, triggeredBy) error
LatestModelStatus(ctx, runner, modelID) (ModelEntry, error)
Snapshot(ctx) (Snapshot, error)
QueryModelAudit(ctx, AuditQuery) ([]AuditRow, error)
QueryRunnerAudit(ctx, AuditQuery) ([]AuditRow, error)
EvictByRetention(ctx, retention) (int, error)
```

Two append-only tables (`model_health_audit`, `runner_health_audit`) are the source of truth; current-status queries derive from `MAX(id) GROUP BY (runner, model)`.

**Wired in:**
- `orchestration.WithHealthStore(*health.Store)` — orchestration option; the per-run `healthMarkerAdapter` writes audit rows on every `MarkModel{Healthy,Unavailable}` from the executor.
- `health.NewProbe(store, rolePolicyState, resolveProber, classifier, config)` — periodic probe reads the same active catalog revision used by resolution and codec visibility; it writes audit rows on every probe outcome with classified `fallback.Reason`.
- `Orchestrator.GetModelHealthSnapshot` reads from the store via `Snapshot`.

**Why a seam:**
- Snapshots survive process restart (no more in-memory flapping).
- Triggered-by attribution distinguishes runtime classifications (carry the `runID`) from probe sweeps (`"probe"`).
- Eviction is a separate concern (`EvictByRetention`) — the store itself never silently drops rows.

**Tests:**
- `internal/health/health_test.go` — append, snapshot derivation, eviction.

DOC: `internal/health/types.go`, `internal/database/schema.sql` (audit tables + indexes).

---

### 3b. Run Lifecycle Transitions (`orchestration/run_lifecycle.go`)

**Purpose:** Keep status mutation, durable status events, run-status broadcasts, and action hydration consistent across stop, continue, and continuation terminal paths.

**Boundary:**
```go
type RunStatusTransitionInput struct {
    Run       *domain.Run
    NewStatus domain.RunStatus
    Phase     domain.RunPhase
    Reason    string
    EndedAt   *time.Time
    // plus optional error, exit-code, heartbeat, progress, and summary fields
}
```

`applyRunStatusTransition` updates the run, persists it, appends a durable status event when the status changed, broadcasts that event, hydrates `run.actions`, broadcasts `run_status`, and returns the hydrated run. Mutating endpoints should return that hydrated run when the proto response has room for it.

---

### 3e. Durable Park/Wait — Waiter seam + Await-handle registry (`orchestration`)

**Purpose:** Let an agent-manager run *wait* on externally-owned async work (a test-genie suite, a git-control-tower baseline diff) without burning tokens or relying on agent-loop discipline. The run *parks* (process exits, non-terminal `RunStatusParked`, sandbox preserved), agent-manager performs the blocking wait on the agent's behalf, then *wakes* the run by resuming the conversation with the result injected as the next turn.

**Interface (the per-producer seam):**
```go
type Waiter interface {
    Producer() string                                  // matched against AwaitHandle.Producer
    Wait(ctx context.Context, key string) (string, error) // blocks until resolved; honours ctx
}
```

**Why it's a seam:**
- Adding a parkable async source is a one-Waiter change — the registry, transitions, persistence, and restart recovery are producer-agnostic.
- Production Waiters shell the producer's *own* blocking-wait CLI (`test-genie runs wait --json <scenario> <run-id>`, `git-control-tower baseline diff --scenario <s> --name <n> --json`) through the `CommandRunner` seam, mirroring `CommandWorkspaceSandboxEnsurer`'s "delegate, never re-implement" discipline. agent-manager's process is *not* an agent-controlled context, so those commands block normally rather than re-parking.
- Tests inject a fake `CommandRunner` (no real binaries) and `mocks.FakeWaiter` drives the registry's resolve/error/deadline/cancel paths deterministically.

**Production implementations:** `testGenieWaiter` (`ProducerTestGenie`), `gctBaselineWaiter` (`ProducerGCT`). Both parse an await key of the form `"<scenario>/<work-id>"` (`splitProducerKey`). The producer-side park trigger (Phase 5) writes the same key.

**Await-handle registry (`AwaitRegistry`):** owns one background watcher goroutine per parked run. Per watcher:
- resolve → `WakeRun(result)`; producer error → `WakeRun("[wait error] …")`; deadline elapsed → `WakeRun(result, timedOut=true)`; `Cancel`/`Stop` → exit **without** waking (the stop/external-wake path owns that transition).
- `WakeRun` is idempotent (a non-parked run is a no-op), so a double-resolve never double-wakes; `Cancel` is wired into `WakeRun`/`StopRun` for defence in depth.
- **Restart recovery:** `RecoverParkedRuns` re-spawns a watcher for every persisted parked run on boot (`ListParkedRuns` reloads each by ID because the pruned list-columns omit the heavy `await_handle`). Handles are persisted on the run row, so a restart never strands a parked run.

**Construction:** built after the orchestrator (it is the registry's waker) and wired back via `SetAwaitRegistry`, mirroring `SetReconciler`. `ParkRun` registers the watcher; `RecoverParkedRuns` runs in `startServer` alongside reconciler recovery; `Stop` drains all watchers on shutdown.

**Producer-side park trigger (cross-package seam, `packages/cli-core/cliutil`):** a producer command opts in by calling `cliutil.ParkForAwait(ParkRequest{Producer, Key, Deadline})` *before* its blocking wait. `ParkForAwait` gates on identity-token presence (`DetectIdentity().IsIdentityPresent()` — the strict AM-run signal), recovers the run id via `VerifyIdentity()` claims, and POSTs `/api/v1/runs/{id}/park`. Its three-valued contract keeps non-AM callers unchanged:
- `parked=false, err=nil` → not an AM run (human/CI/another agent's shell). The caller blocks normally — *zero behavioural change outside agent-manager*.
- `parked=true, result` → parked successfully; the caller prints the clean tool-result and exits 0 (agent-manager ends the turn and wakes later).
- `parked=true, err` → in an AM run but park failed; the caller degrades gracefully to blocking.

Adopters today: `test-genie runs wait` and `git-control-tower baseline diff`. The producer-key constants (`ParkProducerTestGenie`, `ParkProducerGCT`) must match the AM `Waiter` `Producer()` strings — the same `"<scenario>/<work-id>"` key the `Waiter` parses with `splitProducerKey`. Adding a parkable producer is therefore symmetric: one `Waiter` in agent-manager + one `ParkForAwait` call in the producer.

**Observability:** the `running→parked` transition emits a durable `run_status` event whose reason names the handle and ETA (`Parked waiting on <producer>:<key> (ETA …)`); the handle is also carried on the run DTO (`Run.await_handle`, populated only while parked) so the UI shows *what* a parked run waits on and *when* it resumes — a parked run reads as suspended, not hung. See `TEMPORAL-FLOWS.md` (park/wake flow).

---

### 3f. Run env + identity assembler (`orchestration/phases`)

**Purpose:** Guarantee a run's full environment — caller-supplied custom `VROOLI_*` vars, sandbox routing vars, and a fresh identity token — is assembled the *same way* on every code path that launches or relaunches an agent process. Before this seam the fresh-run path (`RunExecutor`) and the continue/wake path diverged: continue left `ContinueRequest.Environment = nil`, silently dropping custom env, `VROOLI_SANDBOX_*`, and `VROOLI_AGENT_IDENTITY_TOKEN` (the latent bug Phase 0 of the park/resume work fixed).

**Interface (single assembler):**
```go
func AssembleRunEnv(in AssembleRunEnvInput) map[string]string
// in: {Custom, RunMode, SandboxID, WorkDir, ScopePath, IdentityToken}
// merge order: custom → sandbox → identity  (system vars override custom)
```

**Why it's a seam:**
- Both `RunExecutor.MergedEnvVars` (fresh run) and the continue/wake path (`assembleContinuationEnv` → `resumeConversation`) call it, so the two can never diverge again.
- Custom env is persisted on the run row (`Run.CustomEnv`, JSON column) precisely so wake can re-inject it; the identity token is **regenerated per wake** (`GenerateIdentityToken` re-persists the hash — plaintext is never stored), and sandbox vars are re-derived from the still-alive sandbox. A woken turn therefore has a verifiable identity and intact sandbox routing.
- Covered by `phases/env_test` (`TestAssembleRunEnv_*`) and the `orchestration` continue/wake env+identity regression tests.

---

### 3g. Unified declaration reconcile + self-registration (`orchestration/declaration_reconcile.go`)

**Purpose:** Register every scenario-owned agent asset — profiles and workflows — through one entry point that reads the single `config.declarations` block, routes each source file by `schemaVersion`, and preserves each kind's lifecycle (profiles mutable with `update_if_unmodified` drift; workflows digest-pinned atomic batch). This is the seam the declared-run doctrine and the no-fallback cutover land on (see [`INVARIANTS.md`](INVARIANTS.md) I17).

**Shape:**
- `readScenarioDeclarationConfig` is the strict config reader and the single rejection site for the legacy `config.profiles`/`config.workflows` blocks and old-directory sources. `ValidateScenarioDeclarationConfig` wraps it for read-only conformance so the mutating and read-only surfaces cannot accept different manifests.
- `reconcileDeclarationSources` is the shared core behind both the config-driven scenario path (`reconcileScenarioDeclarationsAt`) and the directory-driven **self-registration** path (`reconcileSelfDeclarationsAt`). Self-registration discovers agent-manager's own files from `.vrooli/agent-manager/`, bypasses only the dependency-declaration gate, and runs the identical validators and ownership checks (I22).
- Per-source isolation is deliberate and asymmetric: a bad **profile** fails alone; a bad **workflow** withholds the whole atomic workflow batch, but never rolls back a profile already reconciled in the same call.
- `ReconcileDeclaringScenarios` is the startup sweep with per-scenario `recover()` isolation, so one broken manifest never blocks readiness. The two legacy delegate RPCs (`ReconcileScenarioProfiles`/`ReconcileScenarioWorkflows`) are thin projections over this core reading only the new block.

**Why it's a seam:** the fixtures drive `reconcileScenarioDeclarationsAt` / `reconcileSelfDeclarationsAt` over `t.TempDir` scenario trees with no repo-contract resolution, so the full fan-out (routing, validation, drift, atomic withhold, promptRef resolution) is exercised without a live filesystem layout. The promptRef source is itself a seam (`promptmanager.SourceClient`), faked in tests to exercise resolve/pin/withhold without a live prompt-manager.

**Tests:** `internal/orchestration/declaration_reconcile_test.go` (routing, drift, mixed-kind, atomic withhold, per-source isolation, self-registration), `workflow_promptref_test.go` (resolve/pin/withhold), `legacy_layout_guard_test.go` (no legacy readers), `internal/conformance/service_test.go` (read-only parity).

### 3h. Workflow completion nudge + blocking wait (`orchestration/workflow_nudge.go`, `workflow_wait.go`)

**Purpose:** Kill the fleet's workflow-polling pattern. A run terminal *pushes* the crash-safe pull loop instead of a consumer *pulling* it on a ticker, and adopters block on a server-owned wait instead of writing pollers. See [`TEMPORAL-FLOWS.md`](TEMPORAL-FLOWS.md#workflow-completion-nudge--blocking-wait) for the flow and [`INVARIANTS.md`](INVARIANTS.md) I20/I21.

**Shape:**
- `WorkflowNudger` is an in-process, deduplicating work queue: `nudgeWorkflowForRun` resolves the owning execution via `WorkflowExecutionRepository.ExecutionIDForRun` (the reverse of the one-directional `attempt.RunID` link) and enqueues one drive per execution over the existing CAS-guarded `driveWorkflowExecution`. Child-workflow terminals nudge `ParentExecutionID`. It is a trigger, not a scheduler; `RecoverWorkflowExecutions` stays the durable backstop.
- `WaitWorkflowExecution` is an event-driven long-poll: subscribe to a per-execution notifier, re-read to close the subscribe/settle race, then block on a wake channel — no ticker, no per-wait poller. It only reads execution state, so cancelling the waiter never cancels the execution.

**Why it's a seam:** the nudger takes the drive function (`o.NudgeDrive`) and worker/timeout levers by injection, so tests drive dedupe and concurrency without the real engine; the wait registry's notifier is exercised directly. The integration test wires the real engine + SQLite repo + a fake run launcher to prove zero-consumer-advance progression, including across a simulated restart.

**Tests:** `internal/orchestration/workflow_nudge_test.go`, `workflow_wait_test.go`, `workflow_nudge_integration_test.go`, `internal/database/repository_workflow_test.go::TestWorkflowExecutionRepositoryExecutionIDForRun`.

---

### 4. Policy Evaluator (`policy`)

**Purpose:** Centralize and abstract policy decisions.

**Interface:** `policy.Evaluator`
```go
type Evaluator interface {
    EvaluateRunRequest(ctx context.Context, req EvaluateRequest) (*Decision, error)
    EvaluateApproval(ctx context.Context, runID uuid.UUID, actor string) (*ApprovalDecision, error)
    CheckConcurrency(ctx context.Context, req ConcurrencyRequest) (*ConcurrencyDecision, error)
    GetEffectivePolicies(ctx context.Context, scopePath, projectRoot string) ([]*domain.Policy, error)
}
```

**Why it's a seam:**
- Policy rules can change without modifying orchestration
- Enables testing policy logic in isolation
- Supports multiple policy sources (database, config, defaults)

**Policy decisions include:**
- Whether sandbox is required
- Whether approval is required
- Concurrency limits
- Resource limits
- Runner restrictions

---

### 5. Artifact Collector (`adapters/artifact`)

**Purpose:** Abstract artifact storage and validation.

**Interface:** `artifact.Collector`
```go
type Collector interface {
    Store(ctx context.Context, req StoreRequest) (*Artifact, error)
    Get(ctx context.Context, id uuid.UUID) (*Artifact, error)
    Read(ctx context.Context, id uuid.UUID) (io.ReadCloser, error)
    List(ctx context.Context, runID uuid.UUID, opts ListOptions) ([]*Artifact, error)
    Delete(ctx context.Context, id uuid.UUID) error
    DeleteByRun(ctx context.Context, runID uuid.UUID) error
}
```

**Why it's a seam:**
- Abstracts storage location (local, S3, database)
- Enables testing without file I/O
- Supports validation results, diffs, logs, screenshots

---

### 6. Repository Layer (`repository`)

**Purpose:** Abstract data persistence for domain entities.

**Interfaces:**
- `ProfileRepository` - AgentProfile CRUD
- `TaskRepository` - Task CRUD
- `RunRepository` - Run CRUD with status filtering and recommendation extraction
- `EventRepository` - Append-only event log
- `PolicyRepository` - Policy CRUD with scope matching
- `LockRepository` - Scope lock management
- `CheckpointRepository` - Run checkpoint persistence
- `IdempotencyRepository` - Idempotency key tracking
- `StatsRepository` - Aggregation queries for analytics (status counts, cost, duration, breakdowns)
- `InvestigationSettingsRepository` - Investigation settings singleton

**Why it's a seam:**
- Decouples domain logic from storage technology
- Enables testing with mock/stub repositories
- Migration-safe: schema changes don't affect domain code

**Implementations:**
- SQLite-backed implementations in `database/` package (`repository.go`, `repository_run.go`, `repository_stats.go`, `repository_pricing.go`, `repository_support.go`)
- Single embedded SQLite database file at the scenario `api-core/storage` data path
- Schema auto-initialized on connection via `database/schema.sql`

---

## Responsibility Boundaries

### Entry/Presentation Layer (`handlers/`)
- HTTP request parsing and validation
- Response formatting (JSON)
- Authentication/authorization checks
- Error translation to HTTP status codes
- **Does NOT contain:** Business logic, domain rules

### Coordination Layer (`orchestration/`)
- Wires together domain, adapters, and policies
- Manages run lifecycle (create → execute → review → approve)
- Handles async execution coordination
- **Does NOT contain:** Infrastructure details, HTTP concerns, policy rules

### Domain Layer (`domain/`)
- Entity definitions (Task, Run, AgentProfile, Policy)
- Status transitions and validation
- Domain error types
- **Does NOT contain:** Persistence, HTTP, external integrations

### Adapter Layer (`adapters/`)
- External system integration (runners, sandbox, storage)
- Protocol translation
- Retry and circuit-breaker logic
- **Does NOT contain:** Business rules, domain validation

### Policy Layer (`policy/`)
- Policy rule evaluation
- Decision making based on configuration
- Default policy providers
- **Does NOT contain:** Persistence logic, HTTP concerns

---

## Testing Strategy

Each seam enables specific testing patterns:

| Seam | Test Approach |
|------|---------------|
| Runner | Mock runner returns controlled results; test execution flow |
| Flag Validator | MockFlagValidator with configurable validation; test flag rejection |
| Sandbox | Mock provider skips isolation; test orchestration logic |
| Events | In-memory store; verify event sequences |
| Policy | Test policy rules in isolation; mock for orchestration tests |
| Repository | In-memory implementations; test persistence logic |
| Waiter / Await-registry | `mocks.FakeWaiter` + fake `CommandRunner`; drive resolve/error/deadline/cancel/restart-recovery without real producer CLIs |
| Env + identity assembler | `phases/env_test` asserts the custom→sandbox→identity merge + precedence; `orchestration` continue/wake tests assert a woken turn carries verifiable identity + sandbox + custom env |

**Unit tests:** Test domain logic and policy rules without external dependencies.

**Integration tests:** Use real repositories with test database; mock external services (runners, sandbox).

**End-to-end tests:** Full stack with actual runners and sandbox (requires resources).

---

## Extension Points

### Adding a New Runner

The codec seam (§1a) makes this small. Using Grok as the worked example:

1. **Capture real traces first** — never write a decoder from guessed fields.
   Save run/resume/failure stdout to `codecs/testdata/<name>_trace.jsonl`.
2. **Enum tri-source** (gated by `runnertype_conformance_test.go`): add
   `RUNNER_TYPE_<NAME>` to `packages/proto/.../types.proto` (`make gen-code`),
   `domain.RunnerType<Name>` + `ValidRunnerTypes()`, and a corresponding
   candidate for the runner in `config/role-policy-catalog.json`. Add the
   proto↔domain cases in `protoconv/convert.go`.
3. **Codec**: add `codecs/<name>.go` embedding `baseCodec` (via `resolveBinary`)
   + `<name>_test.go` written against the captured trace. Keep
   `Capabilities()` honest — only the bools the trace proves.
4. **Wire** in `main.go` (real codec or `StubRunner` when unavailable) and add a
   process-detection case in `orchestration/terminator.go` + a live-probe case
   in `orchestration/service.go`.
5. **Surface** in the UI (`lib/utils.ts` label/slug maps, the runner-select
   lists) and refresh `@vrooli/proto-types` (`pnpm install` in `ui/`).

### Adding a New Policy Rule

1. Add field to `domain.PolicyRules` struct
2. Update `policy.Evaluator` to check new rule
3. Add validation in policy repository
4. Update CLI/API for new rule configuration

### Adding a New Event Type

1. Add constant to `domain.RunEventType`
2. Add fields to `domain.RunEventData` if needed
3. Update `event.Collector` interface if capturing from runners
4. Update event handlers/streaming as needed

---

## Design Principles

1. **Sandbox-first:** All runs use sandbox by default; in-place requires explicit policy override
2. **Task-centered:** Primary unit is Task with Runs; not ad-hoc agent invocations
3. **Policy-driven:** Central policy evaluation; no scattered permission checks
4. **Event-native:** All agent activity captured as structured events
5. **Extensible by design:** New agents = new configuration, not new orchestration code

---

## Decision Boundaries (`domain/decisions.go`)

Key decisions are extracted into explicit, testable functions. This makes behavior predictable and easy to locate.

### State Transitions

State machine logic for Tasks and Runs is centralized:

```go
// Check if a transition is valid
ok, reason := TaskStatusQueued.CanTransitionTo(TaskStatusRunning)
ok, reason := RunStatusRunning.CanTransitionTo(RunStatusNeedsReview)
```

### Approval Decisions

```go
// Check if a run can be approved
ok, reason := run.IsApprovable()
ok, reason := run.IsRejectable()
```

### Run Mode Decisions

`SandboxConfig.Mode` is the single source of truth for whether a run is
sandboxed. `DeriveRunMode` is the only function that translates a
resolved `SandboxConfig` to a `RunMode`:

```go
runMode := domain.DeriveRunMode(sandboxCfg)
// Off            → RunModeInPlace  (explicit no-sandbox)
// Tracking       → RunModeSandboxed
// Protected      → RunModeSandboxed
// Unspecified    → RunModeSandboxed (Effective() resolves to Protected)
// nil cfg        → RunModeInPlace (treated as Off; in practice the
//                                  orchestrator always populates a
//                                  non-nil SandboxConfig before calling)
```

The orchestrator's `CreateRun` composes that with two optional overrides:

```go
runMode := domain.DeriveRunMode(sandboxConfig)
if req.RunMode != nil {
    runMode = *req.RunMode      // explicit caller override (highest priority)
} else if req.ForceInPlace {
    runMode = domain.RunModeInPlace
}
```

Policies enforce a minimum sandbox strictness via
`policy.Decision.RequiredSandboxMode` (zero-value = no requirement);
the orchestrator rejects the run with `ErrCodePolicyValidation` when
the resolved `SandboxConfig.Mode` is below the required minimum. Use
`SandboxMode.AtLeast` to compare strictness rank
(`Off < Tracking < Protected`).

For the rationale behind picking `SandboxConfig.Mode` over a parallel
boolean field — and the worked example explaining why a Go bool is the
wrong shape for a "safe by default" decision — see
[`INVARIANTS.md`](INVARIANTS.md) (run-mode invariant).

### Result Classification

```go
outcome := ClassifyRunOutcome(err, exitCode, cancelled, timedOut)
if outcome.RequiresReview() { ... }
if outcome.IsTerminalFailure() { ... }
```

### Observability (Phase 2 of the reliability pass)

The orchestration layer emits structured logs and run-timeline lifecycle
events through a single package: `internal/orchestration/obs`.

`obs.Logger` is the package-level `*slog.Logger`, installed at server
startup from `Levers.Observability.LogFormat` / `LogLevel`. Component
code never constructs its own logger — it calls `obs.Logger()`,
`obs.Component(name)`, or (per-run) reads a context-scoped logger via
`obs.L(ctx)`. The only allowed log keys are the `KeyXxx` constants
declared in `obs/log.go`; new keys are a contract change.

The lifecycle event taxonomy lives in `obs/events.go`. Six transitions
are emitted as `EventTypeLifecycle` (a new typed payload added in
Phase 2):

| Phase                     | Producer site                                                 |
|---------------------------|---------------------------------------------------------------|
| `spawn_enqueued`          | `spawn.Dispatcher.Enqueue` (added Phase 3)                    |
| `spawn_started`           | `spawn.Dispatcher` worker on slot acquisition (added Phase 3) |
| `runner_acquired`         | `core.Runner.Execute` after `launcher.Launch`                 |
| `runner_exited`           | `core.Runner.Execute` after `classifyResult`                  |
| `finalize_started`        | `phases.Finalize` entry                                       |
| `finalize_completed`      | `phases.Finalize` exit                                        |

Helpers in `obs/events.go` are the *only* construction site — direct
construction of `LifecycleEventData` outside `obs/` is a contract
violation. Helpers always log + emit so a missing sink (nil Gate, test
seams) still surfaces the transition in stderr.

The structured-log + lifecycle-event surfaces are tested in
`obs/log_test.go` (key stability, format selection, RunCtx threading)
and `obs/events_test.go` (taxonomy coverage, sink-nil safety).

Catalog activation is observable through the `role_policy_catalog` critical
dependency on `/health`. Initial-load failure reports HTTP 503 with the
resolved path, requirement reason, last attempt time, and root cause. Successful
boot logs the active digest. Failed reload state remains available through
`rolepolicy.State.Status()` while the previously active digest stays ready;
the read-only operator status/reload commands are added by the control-surface
phase rather than creating a second state owner.

### Spawn Dispatcher (Phase 3 of the reliability pass)

`internal/orchestration/spawn.Dispatcher` is the **single startup-
serialization choke point** for runs.

- `Dispatcher.Enqueue` is the only path through which a run begins.
  CreateRun and ResumeRun both call it; direct `go executeRun(...)` is
  forbidden.
- The dispatcher caps `MaxStartingConcurrency` runs in the codex-
  bootstrap window simultaneously. Default 1 (strict serialization);
  raise via `Levers.Spawn.MaxStartingConcurrency` once the burst-test
  proves the runner tolerates parallelism.
- `MinSpacing` enforces a minimum delay between successive slot
  acquisitions for cases where MaxStartingConcurrency > 1 still races.
- `QueueCapacity` bounds the queue depth; full → `*domain.CapacityExceededError`
  with `Resource: "spawn_queue"`, mapped to the existing 429 path.
- The startup slot releases either when the executor calls the injected
  `StartedFn` (signalling the run reached `RunStatusRunning`) or when
  `ExecuteFn` returns — whichever comes first. The defer-release
  semantics protect against panics and early-exit terminal failures.

`CreateRunResponse` proto carries `queue_depth`, `active_count`,
`starting_count` populated from `Dispatcher.Stats()`. UI/CLI callers see
backpressure on every accept response, no separate stats endpoint.

Tests: `spawn/dispatcher_test.go` (unit) +
`integration/spawn_serialization_test.go` (full orchestrator burst gate).

### Codec Terminal-Error Classification (Phase 4 of the reliability pass)

`Codec.ClassifyTerminalError(stderr string, exitCode int) *domain.RunnerError`
is the **single codec-side error classifier**. Codecs return typed
`*RunnerError` values directly — there is no `Detect…(stderr) bool`
proliferation; new failure shapes are added by extending the switch
inside the codec, not by adding more boolean predicates.

When a run exits non-zero, `core.Runner` calls `ClassifyTerminalError`
and stores the typed error on `ExecuteResult.TerminalError`. The
orchestration layer's existing promotion-into-`ExecErr` step
(`phases.ExecuteAgent`) lifts it into `EmitFailureEvent`'s typed-error
branch, where it lands on the timeline as the typed `ErrorCode` rather
than `INTERNAL`.

| Codec    | Pattern                                          | ErrorCode                          |
|----------|--------------------------------------------------|------------------------------------|
| codex    | `record_rollout_items` + `thread … not found`    | `RUNNER_SESSION_STATE_LOST`        |
| codex    | `thread … not found` (no rollout-writer context) | `RUNNER_SESSION_EXPIRED`           |
| claude   | `session` + `not found`                          | `RUNNER_SESSION_EXPIRED`           |
| opencode | `session` + (`not found`\|`expired`\|`invalid`)  | `RUNNER_SESSION_EXPIRED`           |

New stderr fixtures landing as `INTERNAL` are a signal to extend
`ClassifyTerminalError` for the relevant codec; never to add a one-off
`Detect…` helper. The regression gate
`internal/orchestration/phases/emitters_test.go::TestEmitFailureEvent_TypedRunnerError_PreservesCode`
fails loudly if a typed `*RunnerError` ever leaks to the timeline as
`INTERNAL`.

### Scope Conflict Detection

```go
if ScopesOverlap("src/", "src/foo/bar") {
    // Parent-child relationship detected
}
```

---

## Control Surface (`config/levers.go`)

The control surface is the set of tunable parameters operators can adjust without code changes.

### Categories

| Category | What It Controls |
|----------|------------------|
| **Execution** | Timeouts, max turns, event buffering |
| **Safety** | Sandbox requirements, file limits, deny patterns |
| **Concurrency** | Max runs, scope locks, queue timeouts |
| **Approval** | Review requirements, auto-approve patterns |
| **Runners** | Binary paths, health checks |
| **Server** | Port, timeouts, request limits |
| **Storage** | Database settings, retention policies |

### Key Levers

**Safety Levers** (accident prevention):
- `RequireSandboxByDefault` - Sandbox-first philosophy
- `MaxFilesPerRun` - Blast radius control (default: 500)
- `MaxBytesPerRun` - Size limit (default: 50MB)
- `DenyPathPatterns` - Hard guardrails (`.git/**`, `.env*`, etc.)

**Concurrency Levers**:
- `MaxConcurrentRuns` - Global parallelism (default: 10)
- `MaxConcurrentPerScope` - Scope-level exclusivity (default: 1)
- `ScopeLockTTL` - Lock timeout (default: 30m)

**Execution Levers**:
- `DefaultTimeout` - Run timeout (default: 30m, range: 1m-4h)
- `DefaultMaxTurns` - Agent turns (default: 100, range: 1-1000)

### Profiles

Pre-configured lever sets for common scenarios:

```go
levers := LeversForProfile(ProfileDevelopment) // Faster, smaller limits
levers := LeversForProfile(ProfileTesting)     // Fast, deterministic
levers := LeversForProfile(ProfileProduction)  // Conservative defaults
```

### Configuration Loading

Priority (highest to lowest):
1. Environment variables (`AGENT_MANAGER_SAFETY_MAX_FILES_PER_RUN`)
2. Config file (via `AGENT_MANAGER_CONFIG` path)
3. Default values

All values are validated with safe bounds to prevent catastrophic misconfiguration.

---

## Cognitive Load Reduction

### Run Executor (`orchestration/run_executor.go`)

The execution flow is extracted into a dedicated, step-by-step executor:

```
1. UpdateStatusToStarting()
2. SetupWorkspace()     - Creates sandbox if needed
3. AcquireRunner()      - Gets and validates runner
4. Execute()            - Runs the agent
5. HandleResult()       - Processes outcome
```

Each step is independently testable and clearly named.

### Approval Operations (`orchestration/approval.go`)

Approval workflow is grouped into one file:
- `ApproveRun()` - Full approval
- `RejectRun()` - Rejection
- `PartialApprove()` - File-level approval

### Validation (`domain/validation.go`)

Entity validation is centralized:
- `profile.Validate()` - AgentProfile validation
- `task.Validate()` - Task validation
- `run.ValidateForCreation()` - Run creation validation
- `policy.Validate()` - Policy validation

---

## File Structure Reference

```
api/internal/
├── domain/
│   ├── types.go           # Core entities (Task, Run, AgentProfile, etc.)
│   ├── errors.go          # Domain error types
│   ├── decisions.go       # Decision helpers (state machines, classification)
│   ├── decisions_test.go  # Decision helper tests
│   ├── invariants.go      # Invariant checking
│   └── validation.go      # Entity validation logic
├── orchestration/
│   ├── service.go         # Main orchestration service and interface
│   ├── run_executor.go    # Run lifecycle execution
│   └── approval.go        # Approval workflow operations
├── adapters/
│   ├── runner/
│   │   ├── interface.go   # Runner interface and registry
│   │   ├── registry.go    # DefaultRegistry, MockRunner, StubRunner
│   │   └── claude_code.go # ClaudeCodeRunner implementation
│   ├── sandbox/
│   │   ├── interface.go   # Sandbox provider and lock manager
│   │   └── workspace_sandbox.go # WorkspaceSandboxProvider implementation
│   ├── event/
│   │   ├── interface.go   # Event store and collector interfaces
│   │   └── sqlite.go      # SQLiteStore implementation
│   └── artifact/
│       └── interface.go   # Artifact collector and validation
├── database/
│   ├── connection.go      # SQLite connection, DSN resolution, schema init
│   ├── schema.sql         # Full database schema (idempotent)
│   ├── repository.go      # Profile, Task, Event, Policy, Lock, Checkpoint, Idempotency repos
│   ├── repository_run.go  # Run repository (CRUD + recommendation extraction)
│   ├── repository_stats.go    # Stats aggregation queries (analytics)
│   ├── repository_pricing.go  # Model pricing and alias repositories
│   ├── repository_support.go  # Investigation settings repository
│   ├── json_types.go      # Custom SQL scanner/valuer for JSON columns
│   └── errors.go          # Database error wrapping
├── policy/
│   └── interface.go       # Policy evaluator interface
├── repository/
│   └── interface.go       # All repository interfaces (no implementations here)
├── handlers/
│   └── handlers.go        # HTTP handlers (thin presentation layer)
└── config/
    ├── config.go          # Legacy config (deprecated)
    ├── levers.go          # Control surface definition
    └── loader.go          # Configuration loading
```

---

---

## Resilience Patterns

Agent-manager implements three architectural patterns for professional-grade reliability:

### 1. Idempotency & Replay Safety

**Purpose:** Enable safe retries and prevent duplicate work when operations are repeated.

**Key Components:**

- `IdempotencyRecord` - Tracks whether an operation has been performed
- `IdempotencyRepository` - Persists idempotency keys and results
- `CreateRunRequest.IdempotencyKey` - Client-provided key for deduplication

**How It Works:**

```go
// Client provides idempotency key
req := CreateRunRequest{
    TaskID:         taskID,
    AgentProfileID: profileID,
    IdempotencyKey: "run:task-123:2024-01-15T10:30:00Z",
}

// If key exists and succeeded, return cached result
// If key exists and failed, allow retry
// If key is new, process and record result
```

**Idempotent Operations:**

| Operation | Idempotency Key | Behavior on Retry |
|-----------|-----------------|-------------------|
| CreateRun | `run:{taskID}:{timestamp}` | Return existing run |
| CreateSandbox | `sandbox:run:{runID}` | Return existing sandbox |
| ApproveRun | `approve:{runID}` | Return approval result |

**Implementation Files:**

- `domain/types.go` - `IdempotencyRecord`, `IdempotencyStatus`
- `repository/interface.go` - `IdempotencyRepository`
- `orchestration/service.go` - Idempotency checks in `CreateRun`

---

### 2. Temporal Flow & Heartbeat

**Purpose:** Detect stalled runs, enforce timeouts, and enable monitoring.

**Key Components:**

- `HeartbeatConfig` - Configures heartbeat interval and timeout
- `RunCheckpoint.LastHeartbeat` - Tracks liveness
- `Run.IsStale()` - Detects stalled runs
- `ListStaleRuns()` - Finds runs needing recovery

**How It Works:**

```
┌─────────────────────────────────────────────────────────────────┐
│                      Run Execution Timeline                      │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Start    Heartbeat   Heartbeat   Heartbeat       Complete/Fail │
│    │         │           │           │                │         │
│    ●─────────●───────────●───────────●────────────────●         │
│    │         │           │           │                │         │
│    └─────────┴───────────┴───────────┴────────────────┘         │
│        30s        30s        30s           ...                   │
│                                                                  │
│  If no heartbeat for 2 minutes → Considered "stale"              │
│  If stale → Can be resumed or marked failed                      │
└─────────────────────────────────────────────────────────────────┘
```

**Configuration (`ExecutorConfig`):**

```go
type ExecutorConfig struct {
    Timeout            time.Duration // 30m default - max execution time
    HeartbeatInterval  time.Duration // 30s default - heartbeat frequency
    CheckpointInterval time.Duration // 1m default - checkpoint frequency
    MaxRetries         int           // 3 default - retries per phase
    StaleThreshold     time.Duration // 2m default - stale detection
}
```

**Implementation Files:**

- `domain/types.go` - `HeartbeatConfig`, `Run.IsStale()`, `Run.LastHeartbeat`
- `orchestration/run_executor.go` - `heartbeatLoop()`, `sendHeartbeat()`
- `orchestration/service.go` - `ListStaleRuns()`

---

### 3. Progress Continuity & Interruption Resilience

**Purpose:** Enable safe interruption and resumption of runs after failures.

**Key Components:**

- `RunPhase` - Explicit phases of run execution
- `RunCheckpoint` - State needed to resume from any phase
- `CheckpointRepository` - Persists checkpoints
- `ResumeRun()` - Resumes from last checkpoint

**Execution Phases:**

```
┌──────────────────────────────────────────────────────────────────┐
│                      Run Phase State Machine                      │
├──────────────────────────────────────────────────────────────────┤
│                                                                   │
│   queued → initializing → sandbox_creating → runner_acquiring     │
│                                                         ↓         │
│            completed ← cleaning_up ← applying ← awaiting_review   │
│                 ↑                                       ↑         │
│                 └───────────────────────────────────────┘         │
│                          collecting_results ← executing           │
│                                                                   │
│   RESUMABLE PHASES: queued, initializing, sandbox_creating,       │
│                     runner_acquiring, executing                   │
│                                                                   │
│   TERMINAL PHASES: completed                                      │
└──────────────────────────────────────────────────────────────────┘
```

**Checkpoint Structure:**

```go
type RunCheckpoint struct {
    RunID             uuid.UUID
    Phase             RunPhase      // Current execution phase
    StepWithinPhase   int           // Progress within phase
    SandboxID         *uuid.UUID    // Preserved sandbox reference
    WorkDir           string        // Workspace directory
    LockID            *uuid.UUID    // Held scope lock
    LastEventSequence int64         // Last persisted event
    LastHeartbeat     time.Time     // Liveness indicator
    RetryCount        int           // Retries for current phase
    SavedAt           time.Time     // When checkpoint was saved
    Metadata          map[string]string // Phase-specific state
}
```

**Resumption Flow:**

```go
// 1. Check if run is resumable
if !run.IsResumable() {
    return error // Terminal state
}

// 2. Get last checkpoint
checkpoint := checkpoints.Get(runID)

// 3. Create executor with resume state
executor := NewRunExecutor(...).WithResumeFrom(checkpoint)

// 4. Executor skips completed phases
if !e.shouldSkipPhase(domain.RunPhaseSandboxCreating) {
    // Create sandbox
} else {
    // Reuse sandbox from checkpoint
}
```

**Progress Tracking:**

```go
// Get progress for display
progress := GetRunProgress(runID)
// Returns:
// - Phase: executing
// - PhaseDescription: "Agent is executing"
// - PercentComplete: 50
// - CurrentAction: "Agent is working on the task"
// - ElapsedTime: 5m30s
// - LastUpdate: 2024-01-15T10:35:00Z
```

**Implementation Files:**

- `domain/types.go` - `RunPhase`, `RunCheckpoint`, `RunProgress`
- `repository/interface.go` - `CheckpointRepository`
- `orchestration/run_executor.go` - Phase management, checkpoint saving
- `orchestration/service.go` - `ResumeRun()`, `GetRunProgress()`

---

### Resilience Pattern Integration

These patterns work together to provide comprehensive reliability:

```
┌────────────────────────────────────────────────────────────────────┐
│                     Resilience Pattern Flow                         │
├────────────────────────────────────────────────────────────────────┤
│                                                                     │
│   CreateRun Request                                                 │
│         │                                                           │
│         ▼                                                           │
│   ┌─────────────┐                                                   │
│   │ Idempotency │  ← Check if already processed                     │
│   │    Check    │  ← Return cached result if complete               │
│   └──────┬──────┘                                                   │
│          │                                                          │
│          ▼                                                          │
│   ┌─────────────┐                                                   │
│   │   Create    │  ← Reserve idempotency key                        │
│   │     Run     │  ← Initialize phase & progress                    │
│   └──────┬──────┘                                                   │
│          │                                                          │
│          ▼                                                          │
│   ┌─────────────┐    ┌─────────────┐    ┌─────────────┐            │
│   │  Execute    │───▶│  Heartbeat  │───▶│ Checkpoint  │            │
│   │   Phase     │    │   Update    │    │    Save     │            │
│   └──────┬──────┘    └─────────────┘    └─────────────┘            │
│          │                                                          │
│          ▼                                                          │
│   ┌─────────────┐                                                   │
│   │ Phase Done? │──No──▶ Next Phase ──▶ [loop]                     │
│   └──────┬──────┘                                                   │
│          │ Yes                                                      │
│          ▼                                                          │
│   ┌─────────────┐                                                   │
│   │  Complete   │  ← Mark idempotency complete                      │
│   │     Run     │  ← Delete checkpoint                              │
│   └─────────────┘                                                   │
│                                                                     │
│   On Failure/Interruption:                                          │
│   ┌─────────────────────────────────────────────────────────────┐  │
│   │ • Checkpoint preserves state for resumption                  │  │
│   │ • Idempotency allows safe retry                              │  │
│   │ • Heartbeat timeout triggers stale detection                 │  │
│   │ • ResumeRun() picks up from last checkpoint                  │  │
│   └─────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────┘
```

---

### Testing Resilience

**Unit Tests:**

```go
// Test idempotency
func TestCreateRun_Idempotent(t *testing.T) {
    key := "test-key-123"
    run1, _ := service.CreateRun(ctx, CreateRunRequest{IdempotencyKey: key})
    run2, _ := service.CreateRun(ctx, CreateRunRequest{IdempotencyKey: key})
    assert.Equal(t, run1.ID, run2.ID)
}

// Test resumption
func TestResumeRun_SkipsCompletedPhases(t *testing.T) {
    checkpoint := &RunCheckpoint{Phase: RunPhaseExecuting}
    executor := NewRunExecutor(...).WithResumeFrom(checkpoint)
    // Verify sandbox creation is skipped
}

// Test stale detection
func TestRun_IsStale(t *testing.T) {
    run := &Run{LastHeartbeat: time.Now().Add(-3 * time.Minute)}
    assert.True(t, run.IsStale(2 * time.Minute))
}
```

**Integration Tests:**

- Simulate crashes during each phase
- Verify resumption restores correct state
- Test concurrent requests with same idempotency key
- Verify stale run detection and recovery

---

## Namespace contract seam (2026-04-29)

The agent-manager side of the workspace-sandbox home-overlay contract.
`SandboxLauncher` builds a `NamespaceLayout` from the sandbox's
`HomeOverlayState` (returned by workspace-sandbox `GET /sandboxes/{id}`)
and uses it to refuse `$HOME/.local/...` commands when the home overlay
is missing — surfacing the failure on the run timeline before any HTTP
call.

Decision tree (one place):

```
command lives under $HOME?
├── yes → state == Present?
│         ├── yes → unchanged (host path is reachable inside namespace)
│         └── no  → ErrCommandHomeOverlayUnavailable (Code: SANDBOX_HOME_OVERLAY_UNAVAILABLE)
├── /usr/bin, /bin, /usr/local/bin → unchanged
├── any other host-absolute → path.Base(X) (sandbox PATH lookup)
└── relative or empty → unchanged
```

`run_executor.emitGenericFailureEvent` walks the error chain, picks up
the typed `Code()` method on `ErrCommandHomeOverlayUnavailable`, and emits
a structured `ErrorEventData{code: "SANDBOX_HOME_OVERLAY_UNAVAILABLE",
retryable: true}` instead of the generic `INTERNAL`. What previously
surfaced as `env: …/claude: No such file or directory` at exec time
now surfaces on the run timeline as a typed, retryable code with a
useful message.

[CODE: api/internal/adapters/sandbox/sandbox_launcher.go#translateCommandToNamespace] •
[CODE: api/internal/adapters/sandbox/sandbox_launcher.go#NamespaceLayout] •
[CODE: api/internal/adapters/sandbox/sandbox_launcher.go#ErrCommandHomeOverlayUnavailable] •
[CODE: api/internal/orchestration/run_executor.go#emitGenericFailureEvent]

---

## Reliable-result and workflow seams

These seams are pinned for `OT-P2-001`. The result resolver, structured-result
pipeline, workflow catalog, and durable interpreter are implemented contracts.
They extend the existing
runner, codec, repository, reconciliation, and park/wake boundaries.

| Seam | Pure/side-effect boundary | Failure signal |
|---|---|---|
| Final-output resolver | Pure function over normalized provider evidence scoped to the latest durable provider turn; provider policies are data/strategy inputs. | `ambiguous` or `unavailable` with candidate evidence and algorithm version. |
| Structured-result pipeline | Deterministic parse, local schema validation, optional bounded extractor, local revalidation. | `invalid`, `unsupported`, `extractor_unavailable`, or `abstained`; never unvalidated success. |
| Workflow catalog | Scenario-local source loader and immutable digest repository; activation is atomic. | Previous revision stays active and readiness exposes digest/count diagnostics only. |
| Workflow interpreter | Pure transition decision over pinned definition, journal projection, signals, and budgets. | Typed terminal reason; no consumer lifecycle vocabulary. |
| Node dispatcher | Persists dispatch intent/idempotency identity before fresh Run, continuation, or child-workflow side effects. | Recovery reuses the intent and never duplicates the child operation. |
| Journal binding evaluator | Deterministic selectors and bounded JSON/text rendering over declared inputs and typed journal entries, including explicit child-workflow terminal output. | Typed binding diagnostic; no transcript or hidden-state fallback. |
| External signal gate | Validates correlation, idempotency key, wait state, payload schema, and deadline before append. | Duplicate is a no-op; stale/invalid signal is rejected without advancing. |
| Subworkflow boundary | Maps declared typed input into a separately pinned child execution with parent attempt and depth identity, then aggregates its budget ledger. | Missing revision, recursion/child budget exhaustion, or typed child terminal; no catalog inference during recovery. |
| Parallel join boundary | Atomically persists visit-scoped membership and intents, dispatches total fan-out in concurrency-sized batches, and converges through `all`, early `any`, or early positive `quorum`. | Impossible joins fail; satisfied early joins stop and durably short-circuit losers; a loop revisit gets new attempts and keys. |
| Cancellation boundary | Commits `cancelling`, recursively stops active Runs and child workflows, records a retry-generation cleanup disposition, then publishes `cancelled`. Failure and budget terminals use the same cleanup ledger. | Missing/failed cleanup remains recoverable and is retried on boot; no abnormal terminal is considered disposed merely because the parent row is terminal. |

The operator-visible story is intentionally compact: catalog activation emits
revision digest/counts; each execution shows current node, attempt, strategy,
child identity, budget use, wait deadline, and terminal reason; each RunResult
shows selection status/rule and structured-validation status. Prompts,
transcripts, structured values, schemas, and handoffs are not copied into list
signals or logs.

- Agent Manager owns provider interpretation and workflow progress.
- A consumer owns its DTO mapping, semantic validation, approvals, and domain
  mutation.
- The handshake is idempotent command in, typed result out; neither side
  mirrors the other's lifecycle.

## Codec-pipe session-home seam

`orchestration.PrepareCodecSessionHome` gives codec-pipe Codex and Grok runs a
durable per-run home under `runstate.RunDir`. The runtime directory is beneath
the root Agent Manager registers as a Workspace Sandbox auxiliary root. The
runner declares that home as a writable mount on each sandboxed launch; the
Sandbox validates the resolved directory against its persisted roots before
binding it. This preserves continuation without restoring inherited
web-console session variables or letting either scenario infer the other's
private filesystem layout. Codex credential/config seed files are copied into
that home and removed at terminal cleanup; rollout and transcript data remain
for recovery and replay.

The session-home regression proves turn two reads the rollout turn one wrote.
Codex's private goal database is intentionally outside Agent Manager's
analytical contract; canonical consumption comes from the run usage projection.

## Related Documentation

## Friction investigation seam

`handlers.EpisodesService` is the sole transport adapter for episode reads,
deterministic self-report spans, receipt-ledger disclosure, and transcript
import. It translates Connect requests into `runreport` and `orchestration`
calls; `runreport` remains transport-free and receipt projections remain
opaque `Struct` values at the boundary.

## Stats measure and billing seam

The Connect measure service is the single transport boundary for Stats
analytics. It computes from `invocation_read_model` projections, returns a
validity assessment plus time-window/filter provenance, and exposes the
definition registry used by the UI. The UI renders a measure only when its
validity is usable; otherwise it shows an evidence-state card and does not
substitute legacy REST aggregates.

Breakdowns and cohort links are observational read-model queries. Their
`/runs` links carry the selected window and dimensions so an operator can
inspect the underlying durable runs. Tool-command grouping is an optional
read-model capability: unavailable command telemetry is reported as
unavailable rather than inferred from tool names.

Run cost keeps metered, subscription, unknown, and unpriced charge bases
separate. Subscription periods are operator-owned records with micro-USD
amounts, quota metadata, and overlap rejection. Runner policies declare
billing mode/provider/plan metadata; the default is explicitly `unknown`.
Allocation requests require an explicit allocation basis and remain
observational until a pricing allocator is configured.

[CODE: api/internal/measures/measures.go#Handler] •
[CODE: api/internal/invocationreadmodel/contract.go#Filter] •
[CODE: api/internal/adapters/database/repository_invocation_read_model.go#RunBreakdown] •
[CODE: api/internal/adapters/database/repository_pricing.go#CreateSubscriptionPeriod] •
[UI: ui/src/features/stats/components/measure/MeasureFrame.tsx]

- [PRD.md](../../PRD.md) - Product requirements and operational targets
- [README.md](../../README.md) - Overview and quick start
- [requirements/README.md](../../requirements/README.md) - Detailed requirements by module
