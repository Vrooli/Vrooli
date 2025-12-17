# Workspace Sandbox - Seams and Boundaries

This document describes the architectural seams, responsibility boundaries, and key variation points in the workspace-sandbox scenario.

## Domain Mental Model

The workspace-sandbox system provides **copy-on-write workspaces** for running agents and tools against a project folder without risking modification of the canonical repository.

### Core Domain Concepts

| Concept | Description |
|---------|-------------|
| **Sandbox** | An isolated, path-scoped workspace with overlay filesystem |
| **Driver** | OS-level mechanism (overlayfs) for creating the isolated view |
| **Diff** | Representation of changes made in sandbox's upper layer |
| **Patch** | Application of approved changes back to canonical repo |
| **Scope** | Path boundary within project that sandbox covers |

### Primary Workflows

1. **Create -> Work -> Review -> Apply**
   - Create sandbox for scope path -> Mount overlay -> Agent works -> Generate diff -> Approve/reject -> Apply patch (if approved)

2. **Lifecycle Management**
   - Create -> Active -> Stop (preserve) -> Delete -or- Create -> Active -> Approve/Reject -> Delete

---

## Responsibility Zones

### Entry/Presentation Layer
- **Location**: `api/internal/handlers/handlers.go` (HTTP handlers), `cli/app.go` (CLI commands)
- **Responsibility**: Parse requests, validate input format, invoke service, format responses
- **Does NOT own**: Business logic, domain rules, persistence

### Server/Configuration Layer
- **Location**: `api/main.go`
- **Responsibility**: Server lifecycle, configuration loading, dependency wiring, route registration
- **Does NOT own**: Request handling, business logic

### Service/Orchestration Layer
- **Location**: `api/internal/sandbox/service.go`
- **Responsibility**: Coordinate operations across repository, driver, and diff generation
- **Does NOT own**: HTTP concerns, database queries, filesystem operations

### Domain Rules
- **Location**: `api/internal/sandbox/pathutil.go` (path validation), `api/internal/types/types.go` (domain types + CheckPathOverlap)
- **Responsibility**: Define sandbox states, ownership types, conflict detection rules
- **Does NOT own**: Persistence, external system calls

### Integration/Infrastructure Layer
- **Location**: `api/internal/driver/` (filesystem), `api/internal/repository/` (database), `api/internal/diff/` (diff tools)
- **Responsibility**: Interact with external systems (overlayfs, PostgreSQL, patch command)
- **Does NOT own**: Business rules, API contracts

---

## Key Seams (Variation Points)

### 1. Driver Interface (Primary Seam)
**File**: `api/internal/driver/driver.go`

The `Driver` interface is the primary seam for OS-level isolation:

```go
type Driver interface {
    Type() DriverType
    Version() string
    IsAvailable(ctx context.Context) (bool, error)
    Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error)
    Unmount(ctx context.Context, s *types.Sandbox) error
    GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error)
    Cleanup(ctx context.Context, s *types.Sandbox) error
}
```

**Implementations**:
- `OverlayfsDriver` - Linux overlayfs (primary)
- Future: fuse-overlayfs, fallback drivers for non-Linux

**Testing Strategy**: Mock driver for unit tests; real driver for integration tests.

**Status**: Properly injected into handlers via dependency injection.

### 2. Repository Interface (Database Seam)
**File**: `api/internal/repository/sandbox_repo.go`

The `Repository` interface enables testing with mock implementations:

```go
type Repository interface {
    Create(ctx context.Context, s *types.Sandbox) error
    Get(ctx context.Context, id uuid.UUID) (*types.Sandbox, error)
    Update(ctx context.Context, s *types.Sandbox) error
    Delete(ctx context.Context, id uuid.UUID) error
    List(ctx context.Context, filter *types.ListFilter) (*types.ListResult, error)
    CheckScopeOverlap(ctx context.Context, scopePath, projectRoot string, excludeID *uuid.UUID) ([]types.PathConflict, error)
    GetActiveSandboxes(ctx context.Context, projectRoot string) ([]*types.Sandbox, error)
    LogAuditEvent(ctx context.Context, event *types.AuditEvent) error
}
```

**Status**: Interface defined; service depends on interface, not concrete type.

### 3. Diff Generation / Patch Application
**File**: `api/internal/diff/diff.go`

Two distinct operations:
- `Generator` - Creates unified diffs from sandbox changes
- `Patcher` - Applies diffs to canonical repo

**Testing Strategy**: Use temp directories with known file states.

---

## Architectural Improvements Completed

### Issue 1: Type Re-export Anti-pattern [RESOLVED]
**Was**: `api/internal/sandbox/types.go` re-exported all types from `internal/types`

**Resolution**: Deleted the re-export file. All code now imports `internal/types` directly.

### Issue 2: Driver Instantiation Leaking Past Seam [RESOLVED]
**Was**: Health check and driver-info endpoints created new driver instances

**Resolution**: Handlers now receive injected driver via `handlers.Handlers` struct. All handlers use the same configured driver instance.

### Issue 3: Duplicated CheckPathOverlap [RESOLVED]
**Was**: Same function existed in both `sandbox/pathutil.go` and `repository/sandbox_repo.go`

**Resolution**: Moved `CheckPathOverlap` to `types/types.go` as a pure function on types. Both packages now use `types.CheckPathOverlap()`.

### Issue 4: main.go Responsibility Overload [RESOLVED]
**Was**: Single file handled configuration, database, server lifecycle, and all HTTP handlers

**Resolution**: Extracted HTTP handlers to `api/internal/handlers/handlers.go`. Main.go now focuses on server setup, configuration, and route wiring.

### Issue 5: Status State Machine Not Explicit [RESOLVED]
**Was**: State transition decisions were scattered across service methods as inline conditionals

**Resolution**: Created `types/status.go` with:
- Explicit state classification methods (`IsActive`, `IsTerminal`, `IsMounted`, `RequiresCleanup`)
- Named decision functions (`CanStop`, `CanApprove`, `CanReject`, `CanDelete`, `CanGenerateDiff`, `CanGetWorkspacePath`)
- State transition matrix documenting valid transitions
- `InvalidTransitionError` for clear error reporting

**Benefit**: State transitions are now testable, documented, and easy to find. The `ValidTransitions` map serves as the authoritative source for the state machine.

### Issue 6: Domain Errors Scattered [RESOLVED]
**Was**: `ScopeConflictError` and `NotFoundError` defined at the end of `service.go`, far from domain types

**Resolution**: Created `types/errors.go` with:
- `DomainError` interface that defines `HTTPStatus()` and `IsRetryable()`
- Co-located domain errors: `NotFoundError`, `ScopeConflictError`, `ValidationError`, `StateError`, `DriverError`
- Each error knows its appropriate HTTP status code

**Benefit**: Errors are now co-located with types, implement a common interface, and enable consistent error handling across the API.

### Issue 7: Repetitive HTTP Error Mapping [RESOLVED]
**Was**: Each handler had repetitive error-type-switch logic (`if _, ok := err.(*sandbox.NotFoundError)...`)

**Resolution**: Added `HandleDomainError()` method to handlers that:
- Checks for `DomainError` interface implementation
- Automatically maps domain errors to correct HTTP status codes
- Falls back to 500 for unknown errors

**Benefit**: Single line error handling in all handlers, consistent error responses, reduced code duplication.

### Issue 8: No Service Interface [RESOLVED]
**Was**: Handlers depended on concrete `*sandbox.Service` type, making testing harder

**Resolution**: Added `ServiceAPI` interface in `sandbox/service.go` documenting all service operations. Handlers now depend on the interface.

**Benefit**: Clear contract documentation, enables mock implementations for handler testing.

### Issue 9: Repetitive Response Handling [RESOLVED]
**Was**: Each handler had `w.Header().Set("Content-Type"...)` and `json.NewEncoder(w).Encode(...)` repetition

**Resolution**: Added response helper methods:
- `JSONSuccess(w, data)` - standard success response
- `JSONCreated(w, data)` - 201 response for created resources
- `JSONError(w, message, code)` - error response

**Benefit**: Consistent response format, reduced boilerplate, single place to modify response behavior.

---

## Testing Seams

### Unit Test Boundaries
- **Service tests**: Mock repository (using `Repository` interface) and driver (using `Driver` interface)
- **Handler tests**: Mock service
- **Path validation tests**: Pure functions in `types` package, no mocks needed
- **Diff tests**: Use temp directories with fixture files

### Integration Test Boundaries
- **API tests**: Real HTTP server, mock or real database
- **Driver tests**: Real overlayfs (Linux only, may need privileges)
- **End-to-end**: Full stack with real PostgreSQL and overlayfs

---

## Cross-Cutting Concerns

### Logging
**Current State**: Uses standard `log` package consistently.

**Future Improvement**: Consider structured logging with a consistent interface injected at startup.

### Error Handling
**Current State**: Domain errors defined in service (`ScopeConflictError`, `NotFoundError`), mapped to HTTP codes in handlers.

**Status**: Well-separated.

### Audit Trail
**Location**: `repository.LogAuditEvent()`

Called from service layer after successful operations. Clean separation.

---

## Package Structure

```
api/
  main.go                       # Server setup, configuration, route wiring
  internal/
    handlers/handlers.go        # HTTP request handlers with response helpers
    sandbox/
      service.go                # ServiceAPI interface + business logic orchestration
      pathutil.go               # Path validation utilities
    types/
      types.go                  # Domain types (Sandbox, FileChange, etc.)
      status.go                 # Status state machine + transition decisions
      errors.go                 # Domain errors with DomainError interface
    driver/
      driver.go                 # Driver interface
      overlayfs.go              # Linux overlayfs implementation
    repository/
      sandbox_repo.go           # Repository interface + PostgreSQL impl
    diff/diff.go                # Diff generation and patch application
```

---

## Decision Boundaries

The following decisions are now explicit and have dedicated locations:

### Status State Machine (types/status.go)
| Decision | Function | Description |
|----------|----------|-------------|
| Can stop sandbox? | `CanStop(status)` | Only active sandboxes can be stopped |
| Can approve? | `CanApprove(status)` | Only active/stopped sandboxes can be approved |
| Can reject? | `CanReject(status)` | Only active/stopped sandboxes can be rejected |
| Can delete? | `CanDelete(status)` | All sandboxes can be deleted (except already deleted) |
| Can generate diff? | `CanGenerateDiff(status)` | Active/stopped sandboxes have diff data |
| Can get workspace path? | `CanGetWorkspacePath(status)` | Only mounted sandboxes have workspace path |
| Valid transition? | `CanTransitionTo(from, to)` | Consults `ValidTransitions` matrix |

### Error Handling (types/errors.go)
| Error Type | HTTP Status | Use Case |
|------------|-------------|----------|
| `NotFoundError` | 404 | Resource not found |
| `ScopeConflictError` | 409 | Overlapping scope paths |
| `ValidationError` | 400 | Invalid input |
| `StateError` | 409 | Invalid state transition |
| `DriverError` | 500 | Filesystem driver failure |

### Approval Mode (service.go:Approve)
| Mode | Behavior |
|------|----------|
| `"all"` | Apply all changes |
| `"files"` | Apply only selected files |
| `"hunks"` | Apply only selected hunks (future) |

---

## New Extension Points (2025-12-16)

### 4. Unified Configuration (config package)
**File**: `api/internal/config/config.go`

The config package centralizes all tunable levers into coherent groups:

```go
type Config struct {
    Server    ServerConfig    // HTTP timeouts, port, CORS
    Limits    LimitsConfig    // Max sandboxes, sizes, list limits
    Lifecycle LifecycleConfig // TTL, GC interval, auto-cleanup
    Policy    PolicyConfig    // Approval mode, thresholds, attribution
    Driver    DriverConfig    // Base dir, fuse mode, project root
    Database  DatabaseConfig  // Connection settings
}
```

**Benefits**:
- Single source of truth for all configuration
- Environment variable loading with `WORKSPACE_SANDBOX_` prefix
- Validation with clear error messages
- Sane defaults that work out of the box

### 5. Policy Interfaces (policy package)
**Files**: `api/internal/policy/policy.go`, `approval.go`, `attribution.go`, `validation.go`

Three policy interfaces define the volatile behavior:

```go
// ApprovalPolicy decides whether changes can be auto-approved
type ApprovalPolicy interface {
    CanAutoApprove(ctx, sandbox, changes) (bool, string)
    ValidateApproval(ctx, sandbox, req) error
}

// AttributionPolicy controls commit authorship
type AttributionPolicy interface {
    GetCommitAuthor(ctx, sandbox, actor) string
    GetCommitMessage(ctx, sandbox, changes, userMessage) string
    GetCoAuthors(ctx, sandbox, actor) []string
}

// ValidationPolicy runs pre-commit hooks
type ValidationPolicy interface {
    ValidateBeforeApply(ctx, sandbox, changes) error
    GetValidationHooks() []ValidationHook
}
```

**Implementations provided**:
- `DefaultApprovalPolicy` - Config-driven approval with thresholds
- `RequireHumanApprovalPolicy` - Always requires human review
- `DefaultAttributionPolicy` - Config-driven author/message format
- `NoOpValidationPolicy` - No validation (fast path)
- `HookValidationPolicy` - Run configured hooks

### 6. Service Options Pattern
**File**: `api/internal/sandbox/service.go`

The service now accepts functional options for injecting policies:

```go
svc := sandbox.NewService(repo, drv, cfg,
    sandbox.WithApprovalPolicy(approvalPolicy),
    sandbox.WithAttributionPolicy(attributionPolicy),
    sandbox.WithValidationPolicy(validationPolicy),
)
```

**Benefits**:
- Backwards compatible (options are optional)
- Easy to test with mock policies
- Clear injection point for behavior customization

---

## Future Seam Candidates

1. **Metrics Collector** - Observability seam for sandbox counts, sizes, latencies
2. **Safe-Git Wrapper** - Git command interception seam
3. **GC Policy** - Garbage collection strategy interface (TTL, size, idle)
4. **Scope Policy** - Path exclusion/inclusion rules interface
