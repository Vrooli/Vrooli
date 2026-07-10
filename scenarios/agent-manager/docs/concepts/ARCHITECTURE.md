# Agent Manager Architecture

This document describes the architectural patterns, invariants, and design decisions that make agent-manager robust, maintainable, and extensible.

## Domain compression — the eight concepts

The system reduces to eight concepts. Anything outside this list is plumbing, not architecture.

| Concept | What it is | Where it lives |
|---|---|---|
| **Run** | One execution of an agent against a task. Carries status, phase, transcript, sandbox, identity. | `internal/domain.Run` |
| **Runner** | The generic stdout-scan-decode-emit-wait pipeline. One implementation, four model bindings (claude-code, codex, opencode, grok). | `internal/adapters/runner/core.Runner` |
| **Codec** | Per-model translation: arg shape, JSON event schema, transcript replay. Each embeds the shared `baseCodec` (`base.go`) and contains only its unique surface. | `internal/adapters/runner/codecs/*.go` |
| **Model policy** | Versioned static model inventory plus named, ordered runner/model or runner-default candidate policies. A validated source-byte digest identifies each immutable revision. | `config/model-policy-catalog.json` + `internal/modelpolicy` |
| **Phase** | One step of the run lifecycle (Setup, Acquire, Execute, Validate, Result, Finalize). Pure-ish function with explicit input struct. | `internal/orchestration/phases/*.go` |
| **Sandbox** | Per-run overlayfs workspace for accountability/provenance — captures which files this run changed, not for safety. | `internal/adapters/sandbox` + `scenarios/workspace-sandbox` |
| **Gate** (`emit.Gate`) | The single Emit choke point. Future invariants (dedupe, audit hooks, ordering) attach inside the gate. | `internal/orchestration/emit.Gate` |
| **Tunables** (`config.Levers`) | Single home for every adjustable threshold (timeouts, intervals, buffers). One struct, validated on load. | `internal/config/levers.go` |

Adding a new runner = one ~250 LOC codec file plus one catalog inventory entry. Adding a model or changing fallback policy = a catalog-only change. Adding a new phase = one file in `phases/`. Adding a new tunable = one field on `Levers` with a default + validation. Anything else means the architecture is not being respected — open a discussion.

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
├── modelpolicy/               # Typed catalog, strict validation, immutable revisions
├── orchestration/
│   ├── run_executor.go        # Thin coordinator (~560 LOC). Owns shared state + phase ordering only.
│   ├── recovery.go            # Restart-resume logic (RecoverInFlightRuns, drainTranscript, tailer)
│   ├── reconciler.go          # Stale-run detection, orphan reaper, periodic reconcile
│   ├── phases/                # Per-phase functions with explicit input structs
│   │   ├── deps.go            # Shared dependency bundle
│   │   ├── emitters.go        # Event emission helpers — route through Gate
│   │   ├── env.go             # Sandbox + identity env-var construction
│   │   ├── setup.go           # Workspace creation
│   │   ├── acquire.go         # Runner selection + fallback chain
│   │   ├── execute.go         # runner.Execute wrapper + model fallback
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

## Model-policy activation flow

`internal/modelpolicy.State` is the sole runtime owner of the catalog path and
active immutable revision. Startup creates the state even when loading fails so
the exact path and validation diagnostic remain observable. A candidate reload
is fully parsed and semantically validated before one atomic active-pointer
swap; a failed reload retains the previous revision.

The same state feeds codec model visibility, periodic model-health probing, and
orchestration policy resolution. No consumer keeps its own catalog cache.
`/health` treats the state as a critical dependency because built-in agent
profiles require declared policy: required state with no active revision is
unready, while a failed reload after a successful activation remains ready on
the prior digest.

Run creation now persists the chosen revision, full ordered candidates,
selected candidate, resolution source, and preflight evidence inside
RunConfig.PolicySnapshot. The generated RunConfig API contract exposes that
snapshot on run detail, while create surfaces deliberately accept the separate
RunConfigOverrides type so callers cannot submit orchestration-owned snapshot
state. Legacy FAST/CHEAP/SMART values are migration inputs that resolve once to
named policy references. Historical rows without policy_snapshot remain
readable as legacy records; they are not backfilled from current policy because
that would invent provenance. Creation-time preflight walks
past stale model IDs and can choose an explicit runner_default, preserving the
installed CLI's safe fallback without encoding it as an empty model ID. The
runtime executor consumes only PolicySnapshot.Candidates for policy-backed
runs. It records attempted, skipped, failed, selected, and terminal exhausted
candidate outcomes with the snapshot digest and index, and can cross runner
boundaries without reading the active catalog. Before each launch it persists
the projected runner/model pair; restart/resume retries that interrupted
candidate at least once without replaying earlier candidates. Historical rows
with no snapshot temporarily retain their legacy execution path until the
hard-cutover migration phase.

The operator surface projects the same state through generated protobuf
contracts at `/api/v1/model-policy/{status,catalog,validate,reload,explain}`.
Catalog inspection is read-only. Validation never activates, reload validates
before the atomic state swap, profile explanation resolves against the active
revision, and run explanation returns only the snapshot persisted with that
run. The Settings UI is an inspection view; declared state remains Git-managed.

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

## Related Documentation

- [SEAMS.md](../internal/SEAMS.md) - Architectural boundaries and interfaces
- [FAILURE_TOPOGRAPHY.md](../FAILURE_TOPOGRAPHY.md) - Failure mode analysis
- [PROBLEMS.md](../internal/PROBLEMS.md) - Known issues and deferred work
- [RESEARCH.md](../RESEARCH.md) - Architecture decisions and research
