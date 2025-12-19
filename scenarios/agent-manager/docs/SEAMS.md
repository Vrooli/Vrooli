# Agent Manager - Architectural Seams

This document describes the architectural seams (deliberate boundaries) in agent-manager that enable testing, extensibility, and safe evolution.

## Architectural Principles

The codebase follows three key architectural principles:

1. **Decision Boundary Extraction** - Key decisions are explicit, named, and testable
2. **Cognitive Load Reduction** - Code is organized to minimize mental overhead
3. **Control Surface Design** - Tunable levers are organized, validated, and documented

## Overview

Agent-manager uses a **screaming architecture** where the folder structure and naming clearly express the domain:

```
api/internal/
├── domain/              # Core domain entities and decisions
│   ├── types.go         # Task, Run, AgentProfile, Policy entities
│   ├── errors.go        # Domain-specific error types
│   ├── decisions.go     # Explicit decision helpers (state machines, classification)
│   └── validation.go    # Entity validation logic
├── orchestration/       # Coordination layer (wires components together)
│   ├── service.go       # Main orchestration service and interface
│   ├── run_executor.go  # Run lifecycle execution (extracted for clarity)
│   └── approval.go      # Approval workflow operations
├── adapters/            # External integration seams
│   ├── runner/          # Agent runner implementations
│   ├── sandbox/         # workspace-sandbox integration
│   ├── event/           # Event streaming and storage
│   └── artifact/        # Diff and artifact collection
├── policy/              # Policy evaluation logic
├── repository/          # Persistence interfaces
├── handlers/            # HTTP handlers (thin presentation layer)
└── config/              # Configuration management
    ├── config.go        # Legacy config (being deprecated)
    ├── levers.go        # Control surface definition
    └── loader.go        # Configuration loading
```

## Core Seams

### 1. Runner Adapter (`adapters/runner`)

**Purpose:** Abstract agent execution across different runners.

**Interface:** `runner.Runner`
```go
type Runner interface {
    Type() domain.RunnerType
    Capabilities() Capabilities
    Execute(ctx context.Context, req ExecuteRequest) (*ExecuteResult, error)
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
- `ClaudeCodeRunner` - Claude Code CLI integration (implemented ✅)
  - Executes `claude` CLI with `--print --output-format stream-json`
  - Parses streaming events (messages, tool calls, errors)
  - Supports cancellation and timeout
  - Reports full capabilities: messages, tool events, cost tracking, streaming
- `CodexRunner` - OpenAI Codex integration (planned)
- `OpenCodeRunner` - Open-source model integration (planned)

**Dependencies:**
- Receives: `ExecuteRequest` with profile, task, working directory
- Produces: `ExecuteResult` with summary, metrics, exit code
- Side effects: Streams `RunEvent` to `EventSink`

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
    Stop(ctx context.Context, id uuid.UUID) error
    Start(ctx context.Context, id uuid.UUID) error
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
  - Supports full/partial approval workflows
  - Health checks for availability monitoring

**Related interface:** `sandbox.LockManager`
- Manages scope-based locking for concurrent runs
- Prevents overlapping path scopes from conflicting

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
- `MemoryStore` - In-memory event storage with streaming support (implemented)
- PostgreSQL-backed store (planned)
- Redis-backed store for high-throughput (planned)

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
- `RunRepository` - Run CRUD with status filtering
- `EventRepository` - Append-only event log
- `PolicyRepository` - Policy CRUD with scope matching
- `LockRepository` - Scope lock management
- `CheckpointRepository` - Run checkpoint persistence
- `IdempotencyRepository` - Idempotency key tracking

**Why it's a seam:**
- Decouples domain logic from storage technology
- Enables testing with in-memory repositories
- Supports different databases (PostgreSQL, SQLite)
- Migration-safe: schema changes don't affect domain code

**Implementations:**
- `MemoryProfileRepository` - In-memory AgentProfile storage (development/testing)
- `MemoryTaskRepository` - In-memory Task storage (development/testing)
- `MemoryRunRepository` - In-memory Run storage (development/testing)
- `MemoryEventRepository` - In-memory event log (development/testing)
- `MemoryCheckpointRepository` - In-memory checkpoints (development/testing)
- `MemoryIdempotencyRepository` - In-memory idempotency keys (development/testing)
- PostgreSQL implementations (planned)

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
| Sandbox | Mock provider skips isolation; test orchestration logic |
| Events | In-memory store; verify event sequences |
| Policy | Test policy rules in isolation; mock for orchestration tests |
| Repository | In-memory implementations; test persistence logic |

**Unit tests:** Test domain logic and policy rules without external dependencies.

**Integration tests:** Use real repositories with test database; mock external services (runners, sandbox).

**End-to-end tests:** Full stack with actual runners and sandbox (requires resources).

---

## Extension Points

### Adding a New Runner

1. Implement `runner.Runner` interface in `adapters/runner/`
2. Register in `runner.Registry` during startup
3. Add runner type to `domain.RunnerType` constants
4. Document capabilities in runner implementation

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

The decision of whether to use sandbox or in-place execution is explicit:

```go
decision := DecideRunMode(
    requestedMode,         // Explicit request (highest priority)
    forceInPlace,          // Override flag
    policyAllowsInPlace,   // Policy permission
    profileRequiresSandbox, // Profile requirement
)
// Returns: Mode, Reason, PolicyOverride, ExplicitChoice
```

### Result Classification

```go
outcome := ClassifyRunOutcome(err, exitCode, cancelled, timedOut)
if outcome.RequiresReview() { ... }
if outcome.IsTerminalFailure() { ... }
```

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
│   │   └── memory.go      # MemoryStore implementation
│   └── artifact/
│       └── interface.go   # Artifact collector and validation
├── policy/
│   └── interface.go       # Policy evaluator interface
├── repository/
│   ├── interface.go       # All repository interfaces
│   └── memory.go          # In-memory implementations
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

## Related Documentation

- [PRD.md](../PRD.md) - Product requirements and operational targets
- [README.md](../README.md) - Overview and quick start
- [requirements/](../requirements/) - Detailed requirements by module
