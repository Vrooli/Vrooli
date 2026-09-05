# Agent Manager Architecture

This document describes the architectural patterns, invariants, and design decisions that make agent-manager robust, maintainable, and extensible.

## Domain compression — the eight concepts

The system reduces to eight concepts. Anything outside this list is plumbing, not architecture.

| Concept | What it is | Where it lives |
|---|---|---|
| **Run** | One execution of an agent against a task. Carries status, phase, transcript, sandbox, identity. | `internal/domain.Run` |
| **Runner** | The generic stdout-scan-decode-emit-wait pipeline. One implementation, four model bindings (claude-code, codex, opencode, grok). | `internal/adapters/runner/core.Runner` |
| **Codec** | Per-model translation: arg shape, JSON event schema, transcript replay. Each embeds the shared `baseCodec` (`base.go`) and contains only its unique surface. | `internal/adapters/runner/codecs/*.go` |
| **Role policy** | Versioned portable roles with ordered `(runner, resourceRole)` candidates. Resource-owned policy resolves concrete models and fallbacks into immutable run evidence. | `config/role-policy-catalog.json` + `internal/rolepolicy` |
| **Phase** | One step of the run lifecycle (Setup, Acquire, Execute, Validate, Result, Finalize). Pure-ish function with explicit input struct. | `internal/orchestration/phases/*.go` |
| **Sandbox** | Per-run overlayfs workspace for accountability/provenance — captures which files this run changed, not for safety. | `internal/adapters/sandbox` + `scenarios/workspace-sandbox` |
| **Gate** (`emit.Gate`) | The single Emit choke point. Future invariants (dedupe, audit hooks, ordering) attach inside the gate. | `internal/orchestration/emit.Gate` |
| **Tunables** (`config.Levers`) | Single home for every adjustable threshold (timeouts, intervals, buffers). One struct, validated on load. | `internal/config/levers.go` |

Adding a new runner = one codec plus a resource-owned role-policy implementation and an Agent Manager role-catalog candidate. Concrete model and fallback changes belong to the owning resource policy. Adding a new phase = one file in `phases/`. Adding a new tunable = one field on `Levers` with a default + validation.

## Folder structure (post-Phase-6 refactor)

The codebase follows **screaming architecture** where folder structure expresses domain intent:

```
api/internal/
├── domain/                    # Core business logic (pure, no external deps)
│   ├── types.go               # Entity definitions
│   ├── errors.go              # Error taxonomy with recovery guidance
│   ├── decisions.go           # Pure decision functions
│   ├── validation.go          # Input validation
│   └── invariants.go          # Runtime invariant enforcement
├── config/
│   └── levers.go              # Single Tunables struct (timeouts, intervals, buffers)
├── rolepolicy/                # Portable role catalog, resource resolution, immutable revisions
├── permissionpolicy/          # Declared permissions, projection planning, reconciliation audit
├── conformance/               # Read-only scenario dependency/profile validation provider
├── orchestration/
│   ├── run_executor.go        # Thin coordinator (~560 LOC). Owns shared state + phase ordering only.
│   ├── recovery.go            # Restart-resume logic (RecoverInFlightRuns, drainTranscript, tailer)
│   ├── reconciler.go          # Stale-run detection, orphan reaper, periodic reconcile
│   ├── phases/                # Per-phase functions with explicit input structs
│   │   ├── deps.go            # Shared dependency bundle
│   │   ├── emitters.go        # Event emission helpers — route through Gate
│   │   ├── env.go             # Sandbox + identity env-var construction
│   │   ├── setup.go           # Workspace creation
│   │   ├── acquire.go         # Single-runner acquisition for snapshot-less history
│   │   ├── execute.go         # Immutable policy-candidate execution
│   │   ├── validate.go        # Silent-launch failure detection
│   │   ├── result.go          # Outcome classification + handler dispatch
│   │   ├── finalize.go        # Apply-at-run-end + sandbox teardown
│   │   ├── failure.go         # FailWithError + HandleContextError + cleanup helpers
│   │   ├── checkpoint.go      # Phase-ladder advancement + checkpoint save
│   │   ├── heartbeat.go       # Heartbeat goroutine body
│   │   └── identity.go        # Identity token generation
│   ├── emit/
│   │   └── gate.go            # The single Emit choke point
│   └── integration/
│       └── restart_resume_test.go  # End-to-end recovery regression gate
├── adapters/
│   ├── runner/
│   │   ├── interface.go       # Runner / EventSink / TranscriptParser interfaces
│   │   ├── launcher.go        # Process-launch abstraction (host / sandbox)
│   │   ├── core/              # Generic Runner: stdout-scan + decode + emit
│   │   │   └── runner.go
│   │   └── codecs/            # Per-model codecs (each embeds baseCodec)
│   │       ├── codec.go       # Codec interface
│   │       ├── base.go        # Embeddable baseCodec + shared helpers
│   │       ├── pricing.go     # Runner-agnostic cost seam
│   │       ├── claude.go      # Claude Code codec
│   │       ├── codex.go       # Codex codec
│   │       ├── opencode.go    # OpenCode codec
│   │       └── grok.go        # Grok codec
│   └── sandbox/               # workspace-sandbox HTTP adapter
├── repository/                # Persistence abstractions
├── policy/                    # Authorization decisions
└── handlers/                  # HTTP presentation (thin layer)
```

## The four big invariants

These behaviors are normative. Future changes must preserve them or fail loudly.

1. **`emit.Gate` is the single Emit choke point.** Runners and phases never hold a raw `EventSink`. The Gate's API is `Emit(*domain.RunEvent) error` and `Close() error`. Future invariants (dedupe by event ID, ordering) attach inside the gate.

2. **`core.Runner` is the only Runner implementation.** Codecs implement the `Codec` interface, not the `Runner` interface. Adding a runner means one codec plus catalog/proto/domain registration; adding a model never requires editing a codec.

3. **Phase functions take explicit input structs, no shared receiver.** Each phase function in `orchestration/phases/` takes a `<Phase>Input` struct (no `*RunExecutor` receiver) and returns an explicit result struct. Phase ordering is owned by `run_executor.go`'s `Execute()` and nowhere else.

4. **`Levers` is the only home for adjustable thresholds.** A new hard-coded duration, count, or buffer size in agent-manager source is a code-review fail. Add it to `config/levers.go` with a default and documented purpose.

## Role-policy activation flow

`internal/rolepolicy.State` owns the portable role catalog path and active
immutable revision. Startup preserves the path and validation diagnostic even
when activation fails. A candidate reload is fully parsed and validated before
an atomic active-pointer swap; a failed reload retains the previous revision.

Role policy has no model inventory. At run creation, its ordered
`(runner, resourceRole)` candidates are resolved by the owning resources into
concrete model/fallback candidates. The immutable snapshot persists the role,
catalog digest, resource-policy provenance, enforcement posture, availability
preflight, selected candidate, and concrete runner/model evidence. Profile and
run-create surfaces accept only `roleRef`; historical snapshot-less rows remain
readable and are never backfilled from mutable current policy.

The operator surface projects portable role state through generated protobuf
contracts at `/api/v1/role-policy/{status,catalog,validate,reload,explain}`.
Catalog inspection is read-only. Validation never activates, reload validates
before the atomic state swap, profile explanation resolves against the active
revision, and run explanation returns only the snapshot persisted with that
run. The Settings UI is an inspection view; declared state remains Git-managed.

## Agent-conformance provider

Agent Manager owns the `agent-conformance` Test Genie descriptor and implements
the shared `ScenarioValidationService`. Test Genie supplies only generic target
facts; the descriptor applies when a target declares the Agent Manager scenario
dependency or contains a bounded `.vrooli/agent-profiles/*.json` source.

The provider resolves only target-local, repository-contained paths and is
read-only: it validates enabled dependency declarations and, when a scenario
owns profile sources, registered sources, role-only profile inputs, profile-key
ownership, and role catalog membership. It reports an orphan file even when a
scenario has no dependency declaration, so the descriptor's profile-glob
applicability cannot hide configuration drift. Direct role-request consumers
need not invent a profile file. A deliberately narrow static scan treats a
direct coding-agent executable spawn as a blocking L3 capability gap: the
fleet has clean evidence for the rule, so a consumer must route execution
through Agent Manager instead. The provider also reports an unready required
global permission posture as a blocking L4 safety issue. It never reconciles
profiles, projects permissions, starts target services, or writes target
files. Missing, disabled, legacy, invalid, and unresolved-role findings are
returned as structured maturity evidence.

## Restart-resume invariants

The most load-bearing behavior. Tests live in `internal/orchestration/integration/restart_resume_test.go`.

- The agent process is launched detached (`setsid` for host, sandbox supervisor for sandboxed) so killing agent-manager does not kill the agent.
- Stdout is tee'd to `run.TranscriptPath` *before* in-memory event processing.
- Transcript writes are append-only and line-flushed.
- `TranscriptCursor` is persisted **before** events are emitted to the broadcaster (at-least-once delivery; downstream dedupes by event ID).
- `RecoverInFlightRuns` re-fetches each run with `Get` (not `List`) so `ResolvedConfig` is populated for the recovery parser. The 2026-04-29 production bug — recovery silently no-oping because pruned `List` results lacked `ResolvedConfig` — was fixed when the integration test surfaced it.

## Validation Strategy

### System Boundary Validation

Validation is performed at **system boundaries** (API handlers, before persistence):

```go
// Handler validates before passing to service
func (h *Handler) CreateProfile(w http.ResponseWriter, r *http.Request) {
    if err := profile.Validate(); err != nil {
        writeError(w, r, err)
        return
    }
    result, err := h.svc.CreateProfile(r.Context(), &profile)
    // ...
}
```

### Validation Types

| Entity | Method | Purpose |
|--------|--------|---------|
| `AgentProfile` | `Validate()` | Ensures name, runner type, tool/path configs are valid |
| `Task` | `Validate()` | Ensures title, scope path, attachments are valid |
| `Run` | `Validate()` | General validation for any run state |
| `Run` | `ValidateForCreation()` | Stricter validation for new runs |
| `Policy` | `Validate()` | Ensures policy rules are consistent |
| `PolicyRules` | `Validate()` | Ensures limits don't conflict |

### Status/State Validity

Each enum type has an `IsValid()` method:

- `TaskStatus.IsValid()` / `IsTerminal()`
- `RunStatus.IsValid()` / `IsTerminal()` / `IsActive()`
- `ApprovalState.IsValid()`
- `RunPhase.IsValid()`
- `RunEventType.IsValid()`
- `IdempotencyStatus.IsValid()`

## Invariant Enforcement

Invariants are conditions that must **always** be true for system correctness. Unlike validation (which checks input), invariants detect programming errors.

### Key Invariants

| ID | Name | Description |
|----|------|-------------|
| `INV_RUN_SANDBOX` | Sandbox Requirement | Sandboxed runs past creation phase must have SandboxID |
| `INV_APPROVAL_STATE` | Approval State Validity | Approval can only change after NeedsReview status |
| `INV_TERMINAL_IMMUTABLE` | Terminal Immutability | Terminal states cannot transition to non-terminal |
| `INV_PHASE_SEQUENCE` | Phase Progression | Phases must progress forward (no backwards jumps) |

### Invariant Checking Modes

```go
// Development: Panic on violation (fail fast)
SetInvariantMode(InvariantModePanic)

// Production: Log violation but continue
SetInvariantMode(InvariantModeLog)

// Performance-critical: Skip checks
SetInvariantMode(InvariantModeDisabled)
```

### Usage

```go
// Check invariant before state transition
if !run.CheckTerminalTransition(newStatus) {
    // Invariant violation detected
}

// Check all invariants at once
violations := run.CheckAllInvariants()
if len(violations) > 0 {
    // Handle violations
}
```

## Safe Accessors

To prevent nil pointer panics, domain entities provide safe accessor methods:

```go
// Instead of risky direct access:
id := *run.SandboxID  // Panics if nil!

// Use safe accessors:
id := run.SafeSandboxID()  // Returns uuid.Nil if nil
if run.HasSandbox() {
    // Safe to use sandbox
}
```

### Available Safe Accessors

| Entity | Method | Returns if nil |
|--------|--------|----------------|
| `Run` | `SafeSandboxID()` | `uuid.Nil` |
| `Run` | `SafeStartedAt()` | `time.Time{}` |
| `Run` | `SafeEndedAt()` | `time.Time{}` |
| `Run` | `SafeLastHeartbeat()` | `time.Time{}` |
| `Run` | `SafeExitCode()` | `-1` |
| `Run` | `SafeApprovedAt()` | `time.Time{}` |
| `Run` | `Duration()` | `0` if not started |
| `RunCheckpoint` | `SafeSandboxID()` | `uuid.Nil` |
| `RunCheckpoint` | `SafeLockID()` | `uuid.Nil` |

## State Assertions

For explicit precondition checking at service boundaries:

```go
// Before starting a run
if err := run.AssertCanStart(); err != nil {
    return err  // Returns StateError with reason
}

// Before stopping a run
if err := run.AssertCanStop(); err != nil {
    return err
}

// Before approving
if err := run.AssertCanApprove(); err != nil {
    return err
}
```

## Lifecycle State Helpers

Simplified lifecycle view for decision making:

```go
state := run.GetLifecycleState()
// Returns: LifecycleStateNew | LifecycleStateActive |
//          LifecycleStateReviewable | LifecycleStateFinished

if run.CanReceiveEvents() {
    // Accept events
}

if run.CanReceiveHeartbeats() {
    // Accept heartbeats
}
```

## Error Taxonomy

Errors are structured for consistent client handling:

```go
type DomainError interface {
    error
    Code() ErrorCode           // Machine-readable code
    Recovery() RecoveryAction  // What client should do
    Retryable() bool          // Can retry help?
    UserMessage() string      // Human-friendly message
    Details() map[string]interface{}
}
```

### Error Categories

| Category | HTTP Status | Example |
|----------|-------------|---------|
| `NOT_FOUND_*` | 404 | Task not found |
| `VALIDATION_*` | 400 | Invalid input |
| `STATE_*` | 409 | Invalid transition |
| `POLICY_*` | 403 | Policy violation |
| `CAPACITY_*` | 503 | Resource limits |
| `RUNNER_*` | 502/504 | Runner issues |
| `SANDBOX_*` | 502/503 | Sandbox issues |

### Recovery Actions

| Action | Meaning |
|--------|---------|
| `retry_immediate` | Try again now |
| `retry_backoff` | Wait then retry |
| `wait` | External change needed |
| `fix_input` | Correct the request |
| `use_alternative` | Try different approach |
| `escalate` | Human intervention |
| `abort` | Give up |

## Decision Functions

Pure functions for testable business logic:

```go
// State transitions
CanTaskTransitionTo(current, target TaskStatus) (bool, string)
CanRunTransitionTo(current, target RunStatus) (bool, string)

// Mode decisions
DecideRunMode(profile, policy, request) RunModeDecision

// Outcome classification
ClassifyRunOutcome(result) RunOutcome

// Resumption logic
DecideResumption(run, checkpoint) ResumptionDecision
DecideStaleRunAction(run, staleThreshold) StaleRunDecision

// Scope conflicts
ScopesOverlap(scope1, scope2 string) bool
```

## Testing Strategy

### Unit Tests

Domain logic has comprehensive unit tests:

```
internal/domain/
├── decisions_test.go    # State machine, mode decisions
├── validation_test.go   # All Validate() methods
└── invariants_test.go   # Invariant checks, safe accessors
```

### Test Coverage

- All validation methods have positive/negative test cases
- Status validity helpers (`IsValid`, `IsTerminal`, `IsActive`)
- Invariant violations in both pass/fail scenarios
- Safe accessors with nil and non-nil values
- State assertions for each operation type

## Extension Points

### Adding New Runner Types

1. Add constant to `RunnerType` in `types.go`
2. Update `ValidRunnerTypes()` and `IsValid()`
3. Implement `runner.Runner` interface
4. Register in `runner.Registry`

### Adding New Policies

1. Define new `PolicyRules` fields
2. Update `PolicyRules.Validate()`
3. Implement evaluation in `policy.Evaluator`

### Adding New Event Types

1. Add constant to `RunEventType`
2. Create new `*EventData` struct implementing `EventPayload`
3. Add `New*Event()` constructor
4. Update `RunEventType.IsValid()`

## Performance Considerations

- Validation is O(n) for slice checks (tools, paths)
- Invariant checking can be disabled in hot paths
- Safe accessors are zero-cost when values are non-nil
- Decision functions are pure and stateless (easily cached)

## Reliable results and workflow-runtime boundary

This section describes the implemented `OT-P2-001` reliable-results and
workflow-runtime boundary. The contracts, persistence, APIs, CLI, operator UI,
and scenario reconciliation described here are live; consumer adoption remains
intentionally limited to the two Swarm pilots.

Agent Manager is a programmatic meta-scenario. Its reusable boundary is a
command-result handshake: a consumer supplies typed workflow input, Agent
Manager executes agent work and returns a typed terminal result, and the
consumer applies its own business transition. Agent Manager does not fetch or
mutate consumer entities.

### The declared-run doctrine

Every programmatic (non-chat) agent interaction is a scenario-owned, registration-validated
**workflow declaration** — the fleet contract. The test for whether an interaction must be a
declared workflow is: **code composes the prompt and code consumes the output**. When that holds,
the middle — prompt assembly, prompt-manager fetching, execution, extraction, classification,
retries, looping — lives in a definition, not hand-rolled glue. Chat and interactive surfaces
(web-console sessions, agent-manager interactive mode, the operator CLI) stay on plain runs; the
raw run API remains the substrate *under* workflow nodes and internal plumbing. A scenario adopting
the doctrine keeps exactly **two ends in code**: building the typed input snapshot and applying the
typed result to domain state. The value purchased is declaration plus registration-time validation,
not orchestration — a one-run feature is still a workflow file (a single-node workflow spelled with
sugar), and no new registrable entity kind is introduced. The full authoring contract, the minimal
example, `promptRef`, and the CEL journal helpers live in
[`../reference/scenario-declarations.md`](../reference/scenario-declarations.md).

Scenario-owned profiles and workflows are declared through **one** service.json block
(`dependencies.scenarios.agent-manager.config.declarations`) and live in **one** directory
(`.vrooli/agent-manager/`), discriminated per file by `schemaVersion`. One reconcile entry point
fans out by kind while preserving each kind's lifecycle (profiles mutable with drift tracking;
workflows digest-pinned atomic revisions). The cutover is **no-fallback**: the retired
`.vrooli/agent-profiles/` / `.vrooli/agent-workflows/` directories and the legacy
`config.profiles` / `config.workflows` blocks are rejected at reconcile and flagged by conformance —
no reader for them remains. Agent Manager registers its own declarations at startup through a
self-registration seam and its investigation feature runs as the first declared-workflow adopter.

### Aggregate split

- `Run` remains one atomic agent conversation. It owns runner identity,
  transcript recovery, lifecycle, and one or more explicitly continued turns.
- `RunResult` is the durable interpretation of a terminal run turn. It owns
  canonical final-output selection, provenance, optional structured output,
  and honest ambiguity or abstention.
- `WorkflowDefinition` is immutable, versioned authored data.
- `WorkflowExecution` is a durable execution pinned to one definition digest.
  It links node attempts to fresh Runs, explicit continuations, waits, and child
  workflows without pretending those children are one conversation.

There is no workflow-global Run, profile, transcript, or mutable context bag.
Every agent-executing node declares an `AgentExecutionBinding`:

- `fresh_run` resolves that node's profile or portable role and creates a new
  Run identity on every attempt, including loop revisits.
- `continue` selects one earlier Run or conversation from the same execution,
  preserves that conversation ancestry, and permits only continuation-safe
  overrides.

### Responsibility decision table

| Concern | Authoritative owner | Boundary rule |
|---|---|---|
| Prompt and skill content | Prompt Manager or scenario-authored source | Agent Manager renders declared references; it does not become the content owner. A `promptRef` node embeds the resolved skill content at reconcile with pinned provenance. |
| Agent profiles and portable roles | Scenario source plus Agent Manager reconciliation and role policy | A workflow node references a profile or role; a workflow has no implicit global profile. |
| Run and conversation identity | Agent Manager | Fresh nodes create distinct Runs; only explicit continue nodes reuse conversation state. |
| Canonical final output and RunResult | Agent Manager | Provider evidence is primary; ambiguous evidence abstains. |
| Workflow and profile definition source | Owning scenario | `.vrooli/agent-manager/` (unified layer) is desired state, declared through `config.declarations`; the catalog is a digest-pinned projection. |
| Execution journal and binding evaluation | Agent Manager | Append-only entries are authoritative; bindings select declared, size-bounded values only. |
| Workflow execution and external signals | Agent Manager | Signals advance waits idempotently but do not encode consumer business approval. |
| Consumer input/output DTOs | Consumer scenario | The consumer maps domain state to workflow input and validates result semantics. |
| Human approvals and domain mutations | Consumer scenario | Agent Manager returns a terminal outcome; it never invokes consumer transition actions. |

### Versioned contract vocabulary

| Contract | Required invariant |
|---|---|
| `FinalOutputSelection` | Carries candidates, selected candidate when unique, stable rule/version, evidence, competing candidates, and `selected`, `ambiguous`, or `unavailable` status. |
| `ResultSpec` | Carries a canonical JSON Schema boundary and digest; classification is an enum/discriminated schema, not a second framework. |
| `StructuredResult` | Contains only locally validated JSON plus extraction and validation provenance; invalid extraction cannot be success. |
| `NodeAttempt` | Has a stable attempt identity, node id, dispatch intent, child link, input snapshot, counters, and terminal record. |
| `ExecutionJournalEntry` | Is append-only and typed; prompt bindings cannot request undeclared full transcripts or hidden mutable state. |
| `ExternalSignal` | Is idempotent, correlated to one wait, payload-validated, and durably recorded before advancement. |
| `Budget` | Bounds time, turns, tokens, cost, attempts, children, concurrency, recursion, retries, waits, and serialized binding size. |

The closed V1 node vocabulary is `run`, `continue`, `child_workflow`, `wait`,
`branch`, `join`, and `end`. V1 has no arbitrary shell, CLI, HTTP callback,
tool, or consumer-mutation node. Every edge that participates in a graph cycle
must declare a positive traversal cap in addition to execution-wide budgets.

The V1 `ResultSpec` subset supports object, array, string, number, integer,
boolean, and null types; `properties`, `required`, `additionalProperties`,
`items`, `enum`, `const`, `oneOf`, and bounded string/array/number constraints.
Schemas must be rooted, finite, depth/byte bounded, and canonicalizable to a
stable digest. Remote references, dynamic references, unevaluated keywords,
custom code, and recursive schemas are rejected as unsupported rather than
silently ignored. A discriminated union uses `oneOf` plus a required `const`
tag. Consumer semantic truth remains outside JSON Schema and outside Agent
Manager.

The implemented `result-spec/v1` pipeline is deterministic-first. A caller may
omit `ResultSpec`, provide a bounded JSON Schema, or use the classification
convenience form; classification values are compiled into a canonical string
`enum` schema before the run is persisted. Canonical schema bytes and their
SHA-256 digest live in `Run.resolved_config.result_spec`, while the terminal
validation outcome lives in `Run.result.structured`. There is no separate
classifier record.

Only a whole-document JSON value or exactly one `json` fenced block is a
deterministic candidate. Multiple JSON fences are `ambiguous`; prose containing
an unfenced object is not guessed. Candidates are locally validated against the
canonical schema. When `constrained_fallback` is requested, the optional
`extract.structured` portable-role seam receives bounded source and schema
bytes. Its response is untrusted and must pass the same local validation.
Missing/outage extraction therefore becomes `abstained`, never success.
Structured terminal statuses are `success`, `unavailable`, `invalid`,
`ambiguous`, and `abstained`; only `success` carries a value. Diagnostics expose
stable codes and instance paths but never source text, schema fragments, or
provider error strings.

### Workflow terminal and error semantics

| Terminal | Meaning | Consumer guidance |
|---|---|---|
| `succeeded` | The declared end result validated successfully. | Apply the typed result idempotently. |
| `blocked` | Agent work produced a valid typed blocker result. | Decide the consumer-domain attention/approval transition. |
| `abstained` | Final-output or structured-result evidence was unavailable, ambiguous, or invalid after allowed policy. | Inspect provenance; retry only through an explicit bounded policy. |
| `budget_exhausted` | A named edge or execution-wide limit stopped further work. | Raise the limit only after reviewing consumption and graph behavior. |
| `failed` | Infrastructure, dispatch, definition, binding, or child execution failed without a valid business result. | Use the typed diagnostic and retryability signal. |
| `cancelled` | An authorized cancellation won the terminal transition race. | Do not apply a result; a new execution needs a new idempotency intent. |

These are workflow execution outcomes only. They do not mirror a consumer
entity state. Result-selection states (`selected`, `ambiguous`, `unavailable`)
and structured-validation states (`valid`, `invalid`, `unsupported`,
`extractor_unavailable`, `abstained`) remain nested provenance, so operators
can distinguish why a workflow abstained without parsing prose.

The durable trace endpoint and `agent-manager workflow trace` expose execution,
attempt, child Run/conversation, journal sequence/kind, budget, and terminal
metadata. Journal bodies, prompt snapshots, and model result bodies remain in
the protected persistence boundary. WebSocket workflow lifecycle messages use
the same metadata-only projection after each committed journal transition.

The generated operator surface is one contract across REST, CLI, and browser:
`GET /api/v1/workflow-executions` lists bounded execution history;
execution get, trace, signal, cancel, retry, resume, and advance endpoints own
all transitions; and `/workflows` renders those same metadata projections. The
page refreshes from metadata-only `workflow_lifecycle` WebSocket messages and
falls back to REST, so a dropped socket cannot become a second state model.
Mixed node profiles remain attempt-local, Run/conversation IDs remain
independent, continuation and child ancestry are explicit, and budget usage is
shown on the execution that owns it.

Routine execution, list, trace, operation, CLI, UI, and WebSocket responses
omit workflow input, prompt snapshots, journal bodies, and output. Payload
inspection is a deliberately separate `execution-result` path that requires an
explicit authorization switch; the API rejects a payload request without that
second signal. This is disclosure friction, not a replacement for deployment
authentication and authorization.

Composition remains a tree of separate identities. A `child_workflow` creates
a separately pinned child execution with explicit parent execution/attempt and
depth fields; its consumption is rolled into the parent budget ledger and its
typed blocked, abstained, budget-exhausted, failed, or cancelled terminal is
preserved by the parent. Authored `child_workflow_output` bindings may select a
bounded value from the child's terminal output without exposing its journal or
transcript. A parallel branch persists all member intents before dispatch and
converges only through an authored `all`, `any`, or positive `quorum` join.
Every fork visit has a durable visit ordinal and distinct member idempotency
keys, so returning through a cycle creates fresh member attempts instead of
reusing a prior join. Fresh Runs, continuations, and child executions are never
collapsed into a synthetic current conversation.

`run.maxTurns`, `run.timeoutSeconds`, `continue.maxTurns`, and
`continue.timeoutSeconds` are node-attempt limits, not documentation. Fresh
Run creation receives the resolved limits after profile selection; a named
continuation receives a per-turn resolved-config overlay while preserving the
source Run's immutable stored configuration and conversation ancestry.

External waits belong to `WorkflowExecution`, not Run park/wake state. Their
signal contract, correlation key, opaque resume token, and deadline are
durable. A signal may arrive before its wait arms: the pinned definition
validates and buffers it, and arming consumes the newest matching record. A
signal wait with `timeoutSeconds: 0` is indefinite and its paused time does not
consume the workflow wall-time budget. A bounded wait may declare `onTimeout`
as an existing node id; expiry appends a `wait_timeout` journal record and
routes there, preserving `wait_timeout` as the terminal reason when that route
reaches an end node. Signal, cancel, retry, and resume writes require
idempotency keys and support compare-and-swap preconditions; cancellation
disposition reports which active children stopped and which external work could
not be stopped.

Prompt bindings keep selection separate from presentation. `renderAs` supports
raw `text`, structured output `json`, prompt-facing `json_pretty`, `xml`,
`markdown`, and `fenced` forms; `wrapTag` and `lang` make XML and fenced blocks
explicit. Prompt-facing bindings may declare `overflow: "truncate"`, which
uses an in-band byte-loss marker. Repeated journal selections can render as an
XML container with `itemTag`, `itemMaxBytes`, and an `evictionPolicy` of
`keep_last`, `keep_first`, or `keep_ends`; each retained item carries its node,
sequence, and ordinal, while omitted whole items are shown with an explicit
`elided` marker. Rendering is deterministic from the pinned binding and
journal; it does not summarize or otherwise call a model.

When a run node requires a structured result and validation fails, its raw
terminal output and normalized validation context are retained on the failed
node attempt. Before the normal node-attempt budget considers a fresh rerun,
the runtime dispatches one repair continuation on the same Run conversation;
the repair prompt requests corrected JSON and carries the concrete validation
errors. A repaired success journals and routes as the normal result. A failed
repair remains durable evidence and returns control to the declared fresh-attempt
budget rather than looping indefinitely.

### Final-output evidence tiers

1. A provider-native terminal result or terminal turn associated with a main
   assistant message.
2. Provider-native message/turn identity, origin, stop reason, parent relation,
   and completion marker that uniquely identify the handoff.
3. Conservative generic terminal evidence when the provider exposes no richer
   semantics.
4. Schema satisfaction only as secondary disambiguation after terminal
   identity; it never makes an incidental earlier message "final."

Tail position alone is never authoritative. If equally supported candidates
remain, selection is `ambiguous`; if no supported candidate exists, it is
`unavailable`. Both are successful acts of abstention, not fabricated output.

### Consumer impact matrix

The Phase 1 inventory found direct generated-contract or Agent Manager client
consumers in `agent-inbox`, `development-toolchain-validator`,
`react-component-library`, `scenario-auditor`, `scenario-to-cloud`,
`scenario-to-desktop`, `swarm-manager`, `system-monitor`, `test-genie`, and
`web-search`. Within Agent Manager, the primary writable/readable summary path
is:

| Surface | Current summary/event dependency | Migration obligation |
|---|---|---|
| Codec/core runner | `UpdateMetrics` writes a mutable last-assistant string; `classifyResult` creates `RunSummary` from it. | Preserve provider evidence and resolve a canonical RunResult before projecting the legacy summary. |
| Recovery | Transcript replay rebuilds a summary from message events or a terminal summary. | Re-run the same pure resolver and persist identical provenance; never invent evidence for old rows. |
| Repository | `runs.summary` is nullable JSON and list projections intentionally omit it. | Store RunResult independently and keep historical rows readable with an absent result. |
| Proto conversion/API | `Run.summary` is mapped in `protoconv/entities.go` and returned by run detail/create/continue flows. | Add backward-readable result fields; retain one documented legacy projection. |
| CLI/UI/WebSocket | Run detail renders summary; status broadcasts carry run projections. | Expose compact result status/provenance without broadcasting transcript bodies. |
| Recommendation/investigation | Falls back from summary description to event text. | Consume canonical selected text when available and retain explicit fallback provenance. |
| External scenario clients | Mostly inspect Run status, summary, session id, cost, or events through generated contracts/manual DTOs. | Inventory compilation and JSON-shape tests before changing or removing any existing field. |

### Canonical result implementation

`Run.result` is now the sole terminal-output owner for codec-pipe and interactive
execution. The pure `domain.ResolveRunResult` resolver consumes persisted
`MessageEventData` provider evidence and returns every viable candidate plus a
`selected`, `ambiguous`, or `unavailable` decision. It never selects by
transcript tail position. Terminal main-assistant evidence outranks explicit
completion reasons, which outrank the conservative fallback; an equal best tier
abstains, and parent/subagent output is retained for audit but cannot win.
When a provider reports completion after its message (for example Codex
`turn.completed`), an evidence-only message record correlates the terminal
marker to the earlier event id without duplicating assistant content.

The `runs.run_result` JSON column is additive and nullable. A null value means a
historical run predating this contract, whereas a present result with an
`unavailable` selection is an explicit modern outcome. Live completion and
restart recovery both resolve the same domain model before the terminal run
update. `Run.summary` remains backward readable but is generated only by
`SummaryFromRunResult`; ambiguous or unavailable text is never copied into it.
Run detail and CLI JSON expose the full result. WebSocket run-status messages
carry only selection status and rule, not transcript or candidate bodies.

Profile reconciliation remains the pattern for workflow desired-state
reconciliation, not a contract to overload: bounded scenario-local sources are
validated and atomically projected while the authored files stay authoritative.

### Workflow catalog boundary

Scenario manifests may declare JSON files beneath `.vrooli/agent-workflows`.
`workflowcatalog` strictly decodes the closed `agent-workflow/v1` contract,
normalizes every supported JSON Schema and ResultSpec, validates graph
reachability and explicit continuation ancestry, and requires a positive
traversal limit on every cycle edge. Bindings select only typed append-only
journal projections and always carry item and byte limits; transcript and
mutable-variable sources do not exist in the enum.

Validated definitions are canonicalized and addressed by a SHA-256 digest.
SQLite stores immutable revisions, while one partial unique index represents
the active `(owner,key)` pointer. A scenario reload parses and resolves every
source before one transaction inserts revisions and moves active pointers, so
a malformed sibling cannot partially activate a catalog. Source files remain
the sole desired-state writer. The catalog contains no command, callback, or
consumer-domain action node.

## Related Documentation

## Friction investigation transport

`EpisodesService` is the proto-first boundary for bounded friction evidence:
`GetEpisodes`, `GetSelfReportSpans`, `GetCrossScenarioLedger`, and
`ImportTranscript`. The CLI invokes each through its generated Connect client;
the retained REST paths are compatibility routes rather than new command
bindings. Episode cohort selection is owned by `MeasuresService.EpisodeCohort`.

- [SEAMS.md](../internal/SEAMS.md) - Architectural boundaries and interfaces
- [FAILURE_TOPOGRAPHY.md](../FAILURE_TOPOGRAPHY.md) - Failure mode analysis
- [PROBLEMS.md](../internal/PROBLEMS.md) - Known issues and deferred work
- [RESEARCH.md](../RESEARCH.md) - Architecture decisions and research
