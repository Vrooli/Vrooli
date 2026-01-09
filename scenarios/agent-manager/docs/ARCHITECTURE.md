# Agent Manager Architecture

This document describes the architectural patterns, invariants, and design decisions that make agent-manager robust, maintainable, and extensible.

## Domain-Driven Design

The codebase follows **screaming architecture** where folder structure expresses domain intent:

```
api/internal/
├── domain/          # Core business logic (pure, no external deps)
│   ├── types.go     # Entity definitions
│   ├── errors.go    # Error taxonomy with recovery guidance
│   ├── decisions.go # Pure decision functions
│   ├── validation.go # Input validation
│   └── invariants.go # Runtime invariant enforcement
├── orchestration/   # Coordination layer
├── adapters/        # External integration interfaces
├── repository/      # Persistence abstractions
├── policy/          # Authorization decisions
└── handlers/        # HTTP presentation (thin layer)
```

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

- [SEAMS.md](SEAMS.md) - Architectural boundaries and interfaces
- [FAILURE_TOPOGRAPHY.md](FAILURE_TOPOGRAPHY.md) - Failure mode analysis
- [PROBLEMS.md](PROBLEMS.md) - Known issues and deferred work
- [RESEARCH.md](RESEARCH.md) - Architecture decisions and research
