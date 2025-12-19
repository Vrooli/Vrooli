# Agent Manager - Architectural Seams

This document describes the architectural seams (deliberate boundaries) in agent-manager that enable testing, extensibility, and safe evolution.

## Overview

Agent-manager uses a **screaming architecture** where the folder structure and naming clearly express the domain:

```
api/internal/
├── domain/          # Core domain entities (Task, Run, AgentProfile, Policy)
├── orchestration/   # Coordination layer (wires components together)
├── adapters/        # External integration seams
│   ├── runner/      # Agent runner implementations
│   ├── sandbox/     # workspace-sandbox integration
│   ├── event/       # Event streaming and storage
│   └── artifact/    # Diff and artifact collection
├── policy/          # Policy evaluation logic
├── repository/      # Persistence interfaces
├── handlers/        # HTTP handlers (thin presentation layer)
└── config/          # Configuration management
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
- `ClaudeCodeRunner` - Claude Code CLI integration
- `CodexRunner` - OpenAI Codex integration
- `OpenCodeRunner` - Open-source model integration
- `MockRunner` - For testing

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

**Why it's a seam:**
- Decouples domain logic from storage technology
- Enables testing with in-memory repositories
- Supports different databases (PostgreSQL, SQLite)
- Migration-safe: schema changes don't affect domain code

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

## File Structure Reference

```
api/internal/
├── domain/
│   ├── types.go         # Core entities (Task, Run, AgentProfile, etc.)
│   └── errors.go        # Domain error types
├── orchestration/
│   └── service.go       # Main orchestration service
├── adapters/
│   ├── runner/
│   │   └── interface.go # Runner interface and registry
│   ├── sandbox/
│   │   └── interface.go # Sandbox provider and lock manager
│   ├── event/
│   │   └── interface.go # Event store and collector
│   └── artifact/
│       └── interface.go # Artifact collector and validation
├── policy/
│   └── interface.go     # Policy evaluator interface
├── repository/
│   └── interface.go     # All repository interfaces
├── handlers/
│   └── (HTTP handlers)
└── config/
    └── (Configuration)
```

---

## Related Documentation

- [PRD.md](../PRD.md) - Product requirements and operational targets
- [README.md](../README.md) - Overview and quick start
- [requirements/](../requirements/) - Detailed requirements by module
