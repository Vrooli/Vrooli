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
- **Responsibility**: Interact with external systems (overlayfs, SQLite, patch command)
- **Does NOT own**: Business rules, API contracts

---

## Key Seams (Variation Points)

### 1. Driver Interface (Primary Seam)
**File**: `api/internal/driver/driver.go`

The `Driver` interface is the primary seam for OS-level isolation:

```go
type Driver interface {
    ID() DriverID
    Version() string
    IsAvailable(ctx context.Context) (bool, error)
    RequiresBwrap() IsolationMode
    Mount(ctx context.Context, s *types.Sandbox) (*MountPaths, error)
    Unmount(ctx context.Context, s *types.Sandbox) error
    GetChangedFiles(ctx context.Context, s *types.Sandbox) ([]*types.FileChange, error)
    Cleanup(ctx context.Context, s *types.Sandbox) error
    ListSandboxDirs(ctx context.Context) ([]uuid.UUID, error)
    CleanupOrphan(ctx context.Context, id uuid.UUID) error
    RemoveFromUpper(ctx context.Context, s *types.Sandbox, relPath string) error
}
```

`DriverID` is the canonical identifier across DB column (`driver_id`),
wire payloads, preference file, and every internal switch. Four values:
`overlayfs-userns`, `overlayfs-root`, `fuse-overlayfs`, `copy`.

`RequiresBwrap` declares the per-driver isolation mode (replaces the
prior central `exec.DriverModeFor` switch). `Mount` populates a
`MountPaths` struct that may include the per-sandbox HOME overlay
fields (`HomeMergedDir`, etc.) — both kernel overlayfs and
fuse-overlayfs set these up; CopyDriver leaves them empty.

**Implementations**:
- `OverlayfsDriver` (kernel; backs both UserNS and Root variants)
- `FuseOverlayfsDriver` (userspace daemon-per-mount fallback)
- `CopyDriver` (cross-platform fallback)

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

### 2c. Process Lifecycle (BwrapConfig + Tracker + Logger)
**Files**: `api/internal/driver/bwrap.go`, `api/internal/process/tracker.go`, `api/internal/process/logger.go`, `api/internal/handlers/process.go`

The `/processes` surface is the boundary between three responsibilities,
each owning one slice of the lifecycle:

| Concern | Owner | Mechanism |
|---|---|---|
| Bytes flowing in/out | `driver.BwrapConfig` | `StdoutWriter`, `StderrWriter`, `StdinReader` are wired by the handler before `Driver.StartProcess`. The driver wires them onto `cmd.Stdout/Stderr/Stdin` and never touches them again. |
| Per-process state | `process.Tracker` | After `StartProcess` returns, the handler calls `tracker.Track(...)` and (when stdin requested) `tracker.SetStdin(...)`. The tracker owns the stdin pipe writer for the life of the process. |
| Exit propagation | `BwrapConfig.OnExit` callback | The driver spawns a wait reaper that calls `cfg.OnExit(exitCode, signal, oomKilled)` once. The handler's closure forwards that into `tracker.RecordExit(...)` and `logger.CloseLogPair(...)`. |
| Log persistence + push notifications | `process.Logger` | `CreatePendingLogPair → FinalizePair` produces two `*logWriter` (one per stream). Each `logWriter.Write` fans out to subscribed channels so `StreamLog` / `/logs/stream` consumers see new bytes immediately. `CloseLogPair` writes the exit footer and unblocks subscribers via channel close. |

**Why this seam exists**: prior to 2026-04-28 the driver wrote both
streams into a single `LogWriter`, the tracker had no exit-channel
mechanism, and there was no stdin pipe. Each of the four concerns above
was tangled into the others, making "give me precise exit info" require
"give me a per-process callback" require "split the streams" require
"add a stdin pipe". Splitting them along the contract above lets each
piece evolve independently.

**Wire surface (canonical client = agent-manager `SandboxLauncher`):**
- `POST   /sandboxes/{id}/processes` — `withStdin: bool` opt-in for the stdin pipe.
- `POST   /sandboxes/{id}/processes/{pid}/stdin?close=true` — body is appended to `Tracker.WriteStdin`; `close=true` calls `Tracker.CloseStdin`.
- `GET    /sandboxes/{id}/processes/{pid}/logs?stream=stdout|stderr` — required `stream`; reads from `Logger.ReadLog`.
- `GET    /sandboxes/{id}/processes/{pid}/logs/stream?stream=stdout|stderr` — Server-Sent Events. Server emits `data:` frames as each chunk arrives, then a single `event: exit` carrying ExitInfo JSON, then `event: end`.
- `DELETE /sandboxes/{id}/processes/{pid}` — best-effort kill via `Tracker.KillProcess`.

**Testability**:
- `process/logger_test.go` covers stream separation, push subscriptions, EOF-on-close, pending-pair lifecycle.
- `process/tracker_test.go` covers `RecordExit` idempotency + notification, stdin pipe contract, `WaitForProcess` driven by exit channel.
- `handlers/process_*_test.go` cover the HTTP surface.

### 3. Diff Generation / Patch Application (UPDATED 2025-12-19)
**Files**: `api/internal/diff/diff.go`, `api/internal/diff/runner.go`, `api/internal/diff/gitops.go`

#### CommandRunner Interface (Critical Test Seam)

The `CommandRunner` interface is the primary seam for external command isolation:

```go
// api/internal/diff/runner.go
type CommandRunner interface {
    Run(ctx context.Context, dir string, stdin string, args ...string) CommandResult
}
```

**Why This Seam Exists**: The diff package shells out to `git`, `diff`, and `patch` commands. Without this seam, tests would:
1. Touch the real Vrooli git repository (DANGEROUS)
2. Create real git repos in temp dirs (slow, flaky)
3. Skip testing these code paths (inadequate coverage)

**Implementations**:
- `ExecCommandRunner` - Production implementation using `exec.CommandContext`
- `MockCommandRunner` - Test implementation with configurable responses
- `NoOpCommandRunner` - Always returns success (for suppressing commands)

**Usage in Production**:
```go
gen := diff.NewGenerator()  // Uses DefaultCommandRunner()
patcher := diff.NewPatcher()  // Uses DefaultCommandRunner()
```

**Usage in Tests**:
```go
mock := diff.NewMockCommandRunner()
mock.AddResponse("git rev-parse HEAD", diff.CommandResult{Stdout: "abc123\n"})
mock.AddResponse("diff -u", diff.CommandResult{Stdout: "diff output", ExitCode: 1})

gen := diff.NewGeneratorWithRunner(mock)
patcher := diff.NewPatcherWithRunner(mock)

// Verify specific commands were called
if err := mock.AssertCalled("git rev-parse"); err != nil {
    t.Error(err)
}
```

#### GitOps Interface (Git Operations Seam)

The `GitOperations` interface abstracts all git-related operations:

```go
// api/internal/diff/gitops.go
type GitOperations interface {
    IsGitRepo(ctx context.Context, dir string) bool
    GetCommitHash(ctx context.Context, repoDir string) (string, error)
    CheckRepoChanged(ctx context.Context, repoDir, baseHash string) (bool, string, error)
    GetChangedFilesSince(ctx context.Context, repoDir, baseCommit string) ([]string, error)
    GetUncommittedFiles(ctx context.Context, repoDir string) ([]GitFileStatus, error)
    CheckForConflicts(ctx context.Context, s *types.Sandbox, changes []*types.FileChange) (*ConflictCheckResult, error)
    ReconcilePendingWithGit(ctx context.Context, repoDir string, pendingPaths []string) (*ReconcileResult, error)
}
```

**Implementations**:
- `GitOps` - Production implementation using `CommandRunner`
- `MockGitOps` - Test implementation with configurable return values

**Usage in Tests**:
```go
mockGit := diff.NewMockGitOps()
mockGit.IsRepo = true
mockGit.CommitHash = "abc123"
mockGit.ConflictResult = &diff.ConflictCheckResult{HasChanged: false}

// Inject into service (future improvement)
// svc := sandbox.NewServiceWithGitOps(repo, drv, cfg, mockGit)
```

#### Generator and Patcher (Updated)

Both now use `CommandRunner` internally:

```go
type Generator struct {
    config GeneratorConfig
    runner CommandRunner  // Injected for test isolation
}

type Patcher struct {
    runner CommandRunner  // Injected for test isolation
}
```

**Testing Strategy**:
- Unit tests: Use `MockCommandRunner` to avoid real command execution
- Integration tests: Use real commands in isolated temp directories
- Handler tests: Mock the service entirely, bypassing diff operations

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

### ⚠️ CRITICAL: Test Isolation Guidelines

**The workspace-sandbox scenario handles git operations and filesystem changes. Tests MUST be properly isolated to avoid:**
1. Modifying the real Vrooli repository
2. Creating commits or branches in the real project
3. Corrupting production data
4. Leaving orphaned mounts or processes

### Unit Test Boundaries

| Package | Mock Dependencies | Isolation Strategy |
|---------|-------------------|-------------------|
| **handlers** | `ServiceAPI` (mock service) | No real service calls |
| **sandbox/service** | `Repository` interface, `Driver` interface, `GitOperations` via `WithGitOps()` | No real DB/FS/git |
| **repository** | None (real on-disk SQLite via `t.TempDir()`) | Embedded `modernc.org/sqlite`, schema applied per test |
| **driver** | N/A (tests real overlayfs in temp dirs) | Uses `t.TempDir()` |
| **diff** | `CommandRunner` interface, `GitOperations` interface | No real commands |
| **types** | None (pure functions) | No external dependencies |
| **policy** | None (pure logic) | No external dependencies |

### Mock Usage Examples

**Handler Tests** (mock service):
```go
func TestGetSandbox(t *testing.T) {
    mockService := &mocks.MockService{
        GetResult: &types.Sandbox{ID: uuid.New(), Status: types.StatusActive},
    }
    h := handlers.Handlers{Service: mockService}
    // ... test handler
}
```

**Repository Tests** (real SQLite per-test):
```go
func TestCreate(t *testing.T) {
    db := newTestDB(t) // opens modernc.org/sqlite at t.TempDir() and applies SchemaSQL
    repo := repository.NewSandboxRepository(db)
    if err := repo.Create(ctx, sandbox); err != nil {
        t.Fatalf("Create: %v", err)
    }
    got, _ := repo.Get(ctx, sandbox.ID)
    // assert round-trip
}
```

**Diff Tests** (mock command runner):
```go
func TestDiffModified(t *testing.T) {
    mockRunner := diff.NewMockCommandRunner()
    mockRunner.AddResponse("diff -u", diff.CommandResult{
        Stdout:   "--- a/file.txt\n+++ b/file.txt\n...",
        ExitCode: 1, // diff returns 1 when files differ
    })
    gen := diff.NewGeneratorWithRunner(mockRunner)
    // ... test diff generation
}
```

**Service Tests with Git Mocks** (FULLY IMPLEMENTED):
```go
func TestApproveWithConflictCheck(t *testing.T) {
    mockGit := diff.NewMockGitOps()
    mockGit.IsRepo = true
    mockGit.CommitHash = "abc123"
    mockGit.ConflictResult = &diff.ConflictCheckResult{
        HasChanged: false,
    }

    svc := sandbox.NewService(mockRepo, mockDriver, cfg,
        sandbox.WithGitOps(mockGit),  // Inject mock git operations
    )

    // All git operations are now mocked - no real git commands executed
    result, err := svc.Approve(ctx, &types.ApprovalRequest{...})

    // Verify git operations were called
    if !mockGit.WasCalled("CheckForConflicts") {
        t.Error("Expected CheckForConflicts to be called")
    }
}
```

### Integration Test Boundaries

| Test Type | Real Components | Mock Components | Location |
|-----------|-----------------|-----------------|----------|
| **API** | HTTP server | DB (optional), Driver | `handlers_test.go` |
| **Driver** | overlayfs, temp FS | None | `overlayfs_test.go` |
| **End-to-end** | All (SQLite, overlayfs) | None | Integration suite |

**IMPORTANT**: Integration tests that use real git operations MUST:
1. Create isolated temp directories with `t.TempDir()`
2. Initialize fresh git repos with `git init` in the temp dir
3. Never reference the real Vrooli project directory
4. Clean up all resources in `t.Cleanup()`

### Test File Naming Convention

```
*_test.go           # Unit tests with mocks
*_integration_test.go # Integration tests with real resources (use build tags)
```

### Build Tags for Test Isolation

For tests requiring root/special privileges:
```go
//go:build linux && integration

package driver_test
```

Run with: `go test -tags=integration ./...`

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
      sandbox_repo.go           # Repository interface + SQLite impl
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

The service now accepts functional options for injecting policies and git operations:

```go
svc := sandbox.NewService(repo, drv, cfg,
    sandbox.WithApprovalPolicy(approvalPolicy),
    sandbox.WithAttributionPolicy(attributionPolicy),
    sandbox.WithValidationPolicy(validationPolicy),
    sandbox.WithGitOps(gitOps),  // CRITICAL for test isolation
)
```

**Benefits**:
- Backwards compatible (options are optional)
- Easy to test with mock policies
- Clear injection point for behavior customization

### 7. Service GitOps Injection (IMPLEMENTED 2025-12-19)
**File**: `api/internal/sandbox/service.go`

The Service now has full test isolation for git operations via injected `GitOperations`:

```go
type Service struct {
    // ... other fields
    gitOps diff.GitOperations  // Injected for test isolation
}
```

**What Changed**:
- Added `gitOps` field to Service struct
- Added `WithGitOps(g diff.GitOperations)` option
- Updated all service methods to use `s.gitOps` instead of package-level functions:
  - `Create` uses `s.gitOps.GetCommitHash()`
  - `Approve` uses `s.gitOps.CheckForConflicts()`
  - `CheckConflicts` uses `s.gitOps.CheckForConflicts()`
  - `Rebase` uses `s.gitOps.GetCommitHash()` and `s.gitOps.GetChangedFilesSince()`
  - `CommitPending` uses `s.gitOps.ReconcilePendingWithGit()`
  - `GetCommitPreview` uses `s.gitOps.ReconcilePendingWithGit()`

**Usage in Tests**:
```go
mockGit := diff.NewMockGitOps()
mockGit.IsRepo = true
mockGit.CommitHash = "abc123"
mockGit.ConflictResult = &diff.ConflictCheckResult{HasChanged: false}

svc := sandbox.NewService(mockRepo, mockDriver, cfg,
    sandbox.WithGitOps(mockGit),
)

// Now all git operations are mocked - no real commands executed
```

**Test Safety Guarantees**:
- NO real git commands are executed in tests
- NO commits, branches, or modifications to the Vrooli repository
- Fully deterministic test behavior
- Fast test execution (no process spawning)

---

## Future Seam Candidates

### Medium Priority (Observability)

2. **Metrics Collector** - Observability seam for sandbox counts, sizes, latencies
3. **Structured Logger** - Logging interface for consistent structured logs

### Lower Priority (Feature Extensions)

4. **Safe-Git Wrapper** - Git command interception seam (currently convention-based)
5. **GC Policy** - Garbage collection strategy interface (TTL, size, idle)
6. **Scope Policy** - Path exclusion/inclusion rules interface

---

## Changelog

| Date | Change | Impact |
|------|--------|--------|
| 2025-12-19 | **Injected GitOperations into Service** | Full test isolation - service no longer calls package-level git functions |
| 2025-12-19 | Added `WithGitOps()` service option | Enables injecting MockGitOps in tests |
| 2025-12-19 | Updated mockService in handlers_test.go | Added missing methods: GetPendingChanges, GetFileProvenance, GetCommitPreview, CommitPending |
| 2025-12-19 | Added `CommandRunner` interface for external command isolation | Tests can now mock git/diff/patch commands |
| 2025-12-19 | Added `GitOperations` interface for git operation abstraction | Service layer git calls can be mocked |
| 2025-12-19 | Updated Generator and Patcher to use CommandRunner | Diff generation is now testable without real commands |
| 2025-12-16 | Added policy interfaces and config package | Configurable approval, attribution, validation |
| 2025-12-15 | Created SEAMS.md documenting architecture | Architectural decisions are now documented |

## Process Tracker — exit-event delivery contract (2026-04-28)

`process.Tracker.WaitForExit(ctx, sandboxID, pid) (*ExitInfo, error)` is
the single seam used by `StreamProcessLogs` to deterministically deliver
the SSE `event: exit` frame. Each tracked process owns an `exitCh`
channel that `RecordExit` closes exactly once when the wait reaper
returns; `WaitForExit` blocks on it (or `ctx.Done()`).

Wire ordering inside `handlers.StreamProcessLogs`:

1. Stream log content via `ProcessLogger.StreamLog`.
2. After the log writer closes, await exit info via
   `WaitForExit` with a bounded 5s timeout
   (`exitInfoWaitTimeout` in `internal/handlers/process.go`).
3. Emit `event: exit` with the JSON-encoded `ExitInfo` if the wait
   succeeded; otherwise emit `event: error` with a "exit info
   unavailable" message. Clients (agent-manager `SandboxLauncher`)
   treat the latter as failure, never success.
4. Emit `event: end` and close the SSE stream.

Pre-2026-04-28 the handler did a best-effort `GetExitInfo` lookup
between steps 1 and 2 and emitted `event: exit` only if it happened to
be populated. Fast-failing processes (bwrap chdir errors that exit in
~10ms) lost the race: the SSE stream closed before `spawnExitReaper`
called `RecordExit`, so no exit frame was ever sent. The
`agent-manager` client treated the missing frame as success and the
run completed with exit 0 despite never producing output.

Tests pinning the contract:

- `process.TestWaitForExit_BlocksUntilRecordExit` — channel semantics.
- `process.TestWaitForExit_ReturnsImmediatelyForAlreadyExited` — fast
  path for late SSE attachers.
- `process.TestWaitForExit_ContextDeadline` — bounded wait must respect
  ctx so a stuck reaper does not hang the SSE stream forever.

## Driver Layer (post-userns refactor)

The `internal/driver` package is split into three orthogonal capability
interfaces so each driver implementation declares exactly what it
supports:

- `MountDriver` — mount lifecycle (`Mount`/`Unmount`/`Cleanup`/
  `ListSandboxDirs`/`CleanupOrphan`/`Type`/`Version`/`IsAvailable`).
  Every driver implements it.
- `ChangeTracker` — change detection seam (`GetChangedFiles`,
  `RemoveFromUpper`). Every driver implements it.
- `MountVerifier` — health-check seam (`VerifyMountIntegrity`).
  `OverlayfsDriver` and `FuseOverlayfsDriver` implement it; `CopyDriver`
  does NOT (no real mount to verify).

`Driver = MountDriver + ChangeTracker` is the composite the service
layer holds. Callers that want to verify a mount but might be holding a
`CopyDriver` should call `driver.VerifyIfSupported(ctx, d, sb)`, which
short-circuits to `nil` when the driver is not a `MountVerifier`.

Process execution lives in the sub-package `driver/exec` behind a single
`IsolationMode` knob:

- `ModeNone` — no isolation, run in `s.MergedDir` directly. (`copy` driver)
- `ModeBwrapPreferred` — bwrap if installed, direct fallback otherwise.
  (`fuse-overlayfs` driver)
- `ModeBwrapRequired` — bwrap or hard error. (`overlayfs` driver — the
  mount lives inside the API's mount namespace, so a direct child can't
  see it.)

Each driver declares its required mode via `Driver.RequiresBwrap()`;
service code calls that and passes the result to `exec.Exec` /
`exec.StartProcess`. There is no central type-switch — adding a new
driver type means implementing the method, not editing a dispatcher.

### Isolation profile is the only knob (post-Phase B)

`exec.BwrapConfig` is constructed in three steps at every call site:

```go
cfg := exec.DefaultBwrapConfig()
exec.CaptureEnv().ApplyTo(&cfg)              // host env: HOME, MIRROR_PROJECT_ROOT
if err := exec.ApplyIsolationProfile(&cfg, profile); err != nil { … }
```

There is no preset fallback. A nil or unresolvable profile surfaces
`types.IsolationProfileNotFoundError` (HTTP 400). The legacy
`IsolationLevel` / `ApplyVrooliAwareConfig` / `GetVrooliEnvVars` knobs
are gone — every isolation behaviour is declared in the profile JSON.

### `BuildBwrapArgs` is a pure function (post-Phase C)

`exec.BuildBwrapArgs(s, cfg)` reads only its inputs — no `os.Getenv`,
no filesystem calls. Every host-derived input arrives via
`cfg.HostHome`, `cfg.MirrorProjectRoot`, or the bind maps populated by
`ApplyIsolationProfile`. The argv contract is pinned by
`args_golden_test.go`; changing it without updating the golden fails
the build.

### HOME overlay invariant (post-Phase A)

Both `OverlayfsDriver.Mount` and `FuseOverlayfsDriver.Mount` set up the
per-sandbox HOME overlay (lower=$HOME, upper=per-sandbox writable
layer). `Mount` populates `s.HomeMergedDir` for both drivers; the
`vrooli-aware` profile binds it at `$HOME` inside the namespace via
`BuildBwrapArgs`. The home overlay is best-effort: if mount fails (no
`$HOME`, kernel rejection), `HomeMergedDir` stays empty and the
profile's `HOME=$HOME` env entry degrades gracefully (no bind, no
overlay — the agent CLI runs without host-config visibility).

## Userns deployment contract (Phase 5)

The default driver is **kernel overlayfs in an unprivileged user
namespace** (`overlayfs-userns`). The deployment wrapper at
`.vrooli/service.json:start-api` runs the API binary inside `unshare -U
-m -r`, which is part of the contract:

```
"run": "cd api && exec unshare -U -m -r ./workspace-sandbox-api"
```

The boot self-check in `main.go::NewServer` verifies the wrapper is
active by reading `/proc/self/uid_map` (via `driver.InUserNamespace`).
If the selected driver is `overlayfs-userns` and the process is not
inside a user namespace, the API exits fatally — no silent fallback.

### Merged-dir visibility (cross-scenario invariant)

Because the API runs inside its own user/mount namespace, sandbox merged
directories are visible **only inside the API process's namespace**.
This is fine for every in-process and child-process consumer, but
operators who want to inspect a merged dir from a host shell must enter
the namespace:

```bash
sudo nsenter -t $(pidof workspace-sandbox-api) -U -m
```

Or use the file-CRUD endpoints (`/api/v1/sandboxes/{id}/files/*`)
which run in the API's namespace already.

The agent-manager `SandboxLauncher`
(`scenarios/agent-manager/api/internal/adapters/sandbox/sandbox_launcher.go`)
translates host merged paths to `/workspace` for in-namespace processes
and never touches the merged dir from the host directly. Any future
change to merged-path semantics must update that translator in lockstep.

## Driver hot-swap (Phase 4)

`driver.Slot` is a thread-safe holder for the active driver, backed by
`atomic.Pointer[Driver]`. It implements `Driver` (and `MountVerifier`,
which delegates via `VerifyIfSupported` so a slot wrapping `CopyDriver`
returns `nil` rather than panicking).

`driver.SwitchDriver(ctx, slot, cfg, optionID)` is the only path that
mutates the slot in production. Sequence: `NewDriverFor` →
`IsAvailable` → `Store` → `SaveDriverPreference`. In-flight ops
captured the prior driver and continue with it; only new ops see the
post-switch driver.

## Home overlay seam (2026-04-29)

Three independent contracts decide whether agent CLIs reachable under
`$HOME/.local/...` will work inside a sandbox. Each lives in exactly
one place; together they make the failure mode loud and structured
instead of silent (`env: ~/.local/bin/claude: No such file or directory`).

| Contract | Owner | Implementation |
|---|---|---|
| **Does this driver provide a home overlay?** | Driver | `MountDriver.Capabilities() DriverCapabilities` — pure (no I/O). `HomeOverlay`=true for both overlayfs variants and fuse-overlayfs; false for copy. [CODE: `internal/driver/driver.go::DriverCapabilities`] |
| **Does this profile need a home overlay?** | Profile | `IsolationProfile.HomeOverlayRequirement` ∈ {`not_needed`, `optional`, `required`}. `vrooli-aware`=`required`; `full`=`not_needed`. Optional profiles run with overlay if Present, fall back when Absent (decision carries `HOME_OVERLAY_FALLBACK`). [CODE: `internal/config/profiles.go`] |
| **Did this sandbox actually get one?** | Sandbox state | `Sandbox.HomeOverlayState` enum (`present | absent | not_requested | unsupported`), persisted in `sandboxes.home_overlay_state`. Set during `Mount`. [CODE: `internal/types/types.go::HomeOverlayState`] |

The driver records `HomeOverlayState=Absent` when the per-sandbox home
overlay fails to mount or fails post-mount verification (`isMountPoint`
+ stat + writable probe). The driver does NOT make the sandbox creation
fail — the sandbox is still useful for profiles that don't need the
overlay.

The handler (`process.go::applyIsolationProfile` and friends) refuses
exec with HTTP 409 + `HomeOverlayRequiredError` (code
`HOME_OVERLAY_REQUIRED`) when:

```
profile.HomeOverlayRequirement == "required" && sandbox.HomeOverlayState != HomeOverlayPresent
```

The agent-manager side mirrors this contract: `SandboxLauncher`
fetches the sandbox metadata, builds a `NamespaceLayout`, and refuses
`$HOME/.local/...` commands when the overlay is missing — surfacing
`SANDBOX_HOME_OVERLAY_UNAVAILABLE` on the run timeline before any
HTTP call to workspace-sandbox.

[CODE: `internal/driver/helpers.go::mountHomeOverlay`] •
[CODE: `internal/driver/{overlayfs,fuse_overlayfs}.go::Mount`] •
[CODE: `internal/handlers/process.go::applyIsolationProfile`]

## Home overlay storage seam (2026-04-29)

The home overlay's upper/work/merged dirs live **outside `$HOME`**.
Putting them inside `$HOME` (the lower layer) creates a self-referential
overlayfs mount whose behavior is undefined per kernel docs and
manifests as intermittent EBUSY/EINVAL or a "every sandbox sees every
other sandbox's upper layer" success path.

`config.ResolveHomeOverlayBaseDir` resolves the base dir at startup:

1. `WORKSPACE_SANDBOX_HOME_OVERLAY_BASE` env var (operator override).
2. `${XDG_RUNTIME_DIR}/workspace-sandbox`.
3. `/var/tmp/workspace-sandbox-$UID` (created mode 0700).

Validation rejects any path that resolves under `$HOME`. Config-load
fails fatally rather than producing a broken sandbox. The driver layer
threads `HomeOverlayBaseDir` from `Config` through to
`mountHomeOverlay`/`unmountHomeOverlay`/`cleanupSandboxDirAll` so the
project overlay (`baseDir/<id>/`) and home overlay
(`homeOverlayBaseDir/<id>/`) are released together.

[CODE: `internal/config/config.go::ResolveHomeOverlayBaseDir`] •
[CODE: `internal/driver/helpers.go::homeOverlayDir`]

## Daemon reaper seam (2026-04-29)

The orphan reconciler (`internal/sandbox/orphan_reconciler.go`) walks
directories, but a stale `fuse-overlayfs` daemon can outlive its
sandbox dir — we observed daemons running for three days after their
sandboxes were Delete()d. `internal/sandbox/daemon_reaper.go` adds a
process-level pass:

1. Walk `/proc/*/cmdline`. Match processes whose argv contains
   `fuse-overlayfs` AND `--upperdir=…/<uuid>/…`.
2. Extract the UUID. If the UUID is not in the live sandbox repo (or is
   marked `deleted`), and the daemon is older than the grace period
   (30 s, to avoid racing in-flight Mounts), reap it.
3. SIGTERM. Wait up to 5 s. If still alive, SIGKILL.
4. Log every kill with structured fields and emit a
   `sandbox.daemon-reaped` audit event.

Runs at startup AND on every `LifecycleReconciler` tick (15 min by
default). The implementation seam is `procEntry`/`realProcFS` so unit
tests substitute a synthetic `/proc` fixture.

[CODE: `internal/sandbox/daemon_reaper.go::ReconcileStaleDaemons`] •
[CODE: `internal/sandbox/lifecycle.go::runReconcilers`]

## Round 3 (2026-04-29): orchestration-layer refactor

### Driver collapse

`OverlayfsDriver` and `FuseOverlayfsDriver` were byte-isomorphic on 8 of
their methods. They are collapsed into a single `OverlayDriver` struct
parameterized by:

- `mountFunc` / `unmountFunc` — the kernel-syscall vs fuse-overlayfs
  subprocess split.
- `availabilityFunc` — userns vs root vs fuse-binary probe.
- `version`, `isolation` — per-flavor static data.

Three factories (`NewOverlayfsUserNSDriver`, `NewOverlayfsRootDriver`,
`NewFuseOverlayfsDriver`) and the auto-pick `NewOverlayfsDriver` keep
the wire surface identical: operators still see four `DriverOption`
entries on `/driver/options`.

[CODE: `internal/driver/overlay.go`]

### Reconciler abstraction

The Round-2 era hardcoded `LifecycleReconciler.runReconcilers` body
inlined every reconciler in dependency order. Round 3 extracts a
single `Reconciler` interface (`Name()` + `Run(ctx) ReconcileReport`)
and a `Runner` that drives a slice in registration order. Order is
data, not code.

The four periodic reconcilers (`lifecycle`, `heal`, `orphan`,
`daemon-reaper`) and the one startup-only (`manual-review-expiry`) all
implement the interface. Per-reconciler metrics (last-run-at,
items-processed, items-failed) fall out for free, surfaced via
`Runner.Metrics()` and the new `POST /admin/reconcilers/{name}`
endpoint that fires one on demand.

[CODE: `internal/sandbox/reconciler.go::Reconciler` /
`internal/sandbox/reconciler.go::Runner` /
`internal/handlers/reconcilers.go`]

### Home-overlay decision unification

The same comparison
(`profile.HomeOverlayRequirement` × `sb.HomeOverlayState` × driver
capability) previously lived inline in three places.
`internal/policy/home_overlay.go` now hosts a pure
`DecideHomeOverlay(caps, profile, sb) HomeOverlayDecision` function
consumed by the workspace-sandbox handler exec gate. Three-valued
profile requirement (`not_needed`/`optional`/`required`) lets
profiles express "uses $HOME if present, falls back when absent" via
the `HOME_OVERLAY_FALLBACK` decision code instead of a binary
allowed/refused. The
agent-manager mirror (`scenarios/agent-manager/.../sandbox_launcher.go`)
exposes `IsHomeOverlayPresent(state) bool`; both predicates are pinned
in lockstep by the parity test
`agent-manager/.../home_overlay_policy_test.go::TestIsHomeOverlayPresent_ParityWithWorkspaceSandbox`.

[CODE: `internal/policy/home_overlay.go::DecideHomeOverlay`] •
[CODE: `internal/policy/home_overlay.go::IsHomeOverlayPresent`]

### `internal/runtime/` package

Profile resolution + home-overlay enforcement + protected-mode git
allowlist + resource-limit defaulting moved out of the handler layer
into `internal/runtime/`. The HTTP handler (`process.go`) shrinks to
parse → call runtime → format response. A `process_loc_test.go`
meta-test pins `process.go` ≤ 600 LOC so a future change can't
re-balloon the file silently.

[CODE: `internal/runtime/profile.go::ProfileResolver` /
`internal/runtime/git_allowlist.go::EvaluateProtectedGitAllowlist`]

### Service carving

`internal/sandbox/service.go` (2,825 LOC, 60+ methods) was split into
8 files by responsibility — `service_create.go`, `service_lifecycle.go`,
`service_review.go`, `service_pending.go`, `service_acceptance.go`,
`service_paths.go`, `service_audit.go`, `service_rebase.go`. The
struct + ServiceAPI interface + ctor stay in `service.go` (now ≤ 350
LOC). Mechanical move only; no logic changes.

### Durable heal state

The auto-heal failure tracker was in-memory only — a permanently
broken sandbox would silently reset its loop-bomb counter on every
API restart and retry forever. Phase 6 adds a `heal_state` SQLite
table backing the in-memory cache (write-through). The Runner loads
the table at boot before the first reconciler tick; metrics
(`heal_state_active`, `heal_state_max_consecutive_failures`) make the
loop-bomb risk observable.

[CODE: `internal/repository/heal_state.go` /
`internal/sandbox/heal.go::healTracker`]

### Property tests for pathutil

`internal/sandbox/pathutil_property_test.go` adds 4 properties using
stdlib `testing/quick`: idempotence, traversal-rejection,
encoding-invariance (NFC/NFD), and symlink-containment.


### Round 4 Phase 1: Shared `internal/testutil/` package

Round 4 inverts the usual order: test infrastructure first, boundary
refactors driven by what tests want to inject. Phase 1 lands the
shared testutil package — every fake the test suite needs in one place.

Layout:

- `internal/testutil/mocks/` — interface fakes that DO NOT import the
  `sandbox` package. Hosts `FakeRepository` (full state, per-method
  error knobs), `FakeTxRepository`, `FakeDriver` (full Driver +
  MountVerifier), `FakeGitOps`, `FakePinger`, `FakeProfileStore`. Tests
  inside package `sandbox` import these directly.
- `internal/testutil/mocks/sandboxiface/` — fakes that DO import the
  `sandbox` package. Hosts `FakeReconciler`, `FakeProcFS`, `FakeService`
  (full ServiceAPI surface). External-test files import these. The
  subpackage split exists solely so package-`sandbox` tests can pull in
  `mocks/` without an import cycle.
- `internal/testutil/fixtures/` — domain-object factories: `NewSandbox`,
  `NewIsolationProfile`, `NewExitInfo`. Functional options keep
  defaults out of test bodies.
- `internal/testutil/db/` — `NewSQLite(t)` returns a connected DB with
  the production `repository.SchemaSQL` applied to a `t.TempDir()`
  file. Replaces the per-test `newTestDB` helpers.
- `internal/testutil/httpx/` — live-HTTP harness (`NewLiveServer`).
  The SSE parser originally seeded here moved into `internal/sse`
  in Phase 5 once the encoder needed to live next to it; this
  package is now exclusively about the production middleware +
  router boot path.
- `internal/testutil/assertx/` — domain assertions (`AssertStatus`,
  `AssertHomeOverlayState`, `AssertSSEFrameSequence`, `AssertAuditEvents`).

Guarantees pinned by tests:

- `internal/testutil/no_prod_import_test.go` walks every non-test `.go`
  file under `internal/` and asserts none imports
  `workspace-sandbox/internal/testutil/...`. Production code that
  accidentally references a fake fails CI loudly.
- Each fake has a smoke test verifying default state plus at least one
  error-injection knob.

Production-code changes Phase 1 made en route:

- Exported `sandbox.ProcFS` and `sandbox.ProcEntry` (was unexported
  inline interface) so `FakeProcFS` can satisfy them from outside the
  package.
- Renamed `sandbox.reconcileStaleDaemonsWithConfig` →
  `sandbox.ReconcileStaleDaemonsWithConfig` for the same reason.
- Fixed a pre-existing bug in `sandbox.daemonOwnerIsOrphan` surfaced
  by the production-faithful `FakeRepository.Get` contract: the real
  SQLite repo returns `(nil, nil)` for missing rows, but
  `daemonOwnerIsOrphan` only treated `*types.NotFoundError` as orphan,
  meaning daemon-reaper would skip orphan fuse-overlayfs processes
  in production. Now mirrors `orphan_reconciler.go::isOrphan`'s
  both-conventions accept.

[CODE: `internal/testutil/`]

## Clock Seam (Round 4 Phase 2)

`internal/clock/clock.go` is the canonical wall-clock seam. Every
production component that previously called `time.Now()`, `time.Since`,
`time.Sleep`, or `time.NewTicker` now consumes a `clock.Clock` instead.
DOD: zero `time.Now()` or `time.NewTicker` references in `internal/`
outside `internal/clock/` and `internal/testutil/`.

Interface:

```go
type Clock interface {
    Now() time.Time
    Since(time.Time) time.Duration
    Sleep(time.Duration)
    NewTicker(time.Duration) Ticker
}
type Ticker interface {
    C() <-chan time.Time
    Stop()
}
```

Production wires `clock.System{}` once in `main.go` and threads it
through every constructor:

- `repository.NewSandboxRepository(db, clk)` — every UTC timestamp the
  repo writes (CreatedAt, UpdatedAt, DeletedAt, EventTime, AppliedAt).
- `process.NewTracker(clk)` / `NewTrackerWithConfig(cfg, clk)` —
  StartedAt on Track, StoppedAt fallback in RecordExit, kill-grace
  Sleep loop.
- `process.NewLogger(cfg, clk)` — log header timestamp (per-stream)
  and exit-trailer wording in CloseLogPair.
- `gc.NewService(repo, drv, cfg, clk)` — StartedAt/CompletedAt on
  every GC pass, candidate-cutoff math.
- `sandbox.NewService(repo, drv, cfg, clk, opts...)` — Stop's
  StoppedAt, Start's LastUsedAt, Approve/finalizeApproval's now,
  Rebase/CheckConflicts timestamps, manual-review TTL evaluation,
  daemon-reaper start time, orphan reconciler duration.
- `sandbox.NewRunner(interval, periodic, startupOnly, clk)` — the
  reconciler dispatch ticker now flows through `clock.NewTicker`, and
  per-reconciler LastRunAt metrics flow through `clock.Now`.
- `driver.NewCopyDriver(cfg, clk)`, `NewOverlayfsUserNSDriver(cfg, clk)`,
  `NewOverlayfsRootDriver(cfg, clk)`, `NewFuseOverlayfsDriver(cfg, clk)`,
  `NewOverlayfsDriver(cfg, clk)` — FileChange.DetectedAt timestamps.
- `driver.SelectDriver(ctx, cfg, clk)`, `SelectDriverWithPreference`,
  `DriverInfo`, `NewDriverFor`, `SwitchDriver` — propagate the same
  clock to every driver they construct.
- `logging.New(service, WithClock(clk), opts...)` — entry-time
  timestamps in JSON log lines.
- `handlers.Handlers{Clock: clk, ...}` — admin/audit response
  `timestamp`, interactive-shell exit-message Sleep.

Test wiring: `testutil/mocks.FakeClock` advances only on `Advance` /
`SetNow`. `Sleep` is implemented as `Advance`, so production polling
loops (`for now < deadline { sleep(step) }`) terminate after one
iteration. Tickers returned from `FakeClock.NewTicker` fire on
`Advance`. Per-consumer deterministic tests live in
`internal/sandbox/clock_seam_test.go` (Service.Stop timestamp,
ReconcileLifecycle idle-TTL boundary, ReconcileManualReviewExpiry
boundary, Runner ticker through FakeClock) and
`internal/process/clock_seam_test.go` (Tracker.RecordExit StoppedAt
fallback, Logger header timestamp).

Production-code changes Phase 2 made en route:

- Removed the per-reconciler `now func() time.Time` parameter on
  `Service.ReconcileManualReviewExpiry` and the matching field on
  `ManualReviewExpiryReconciler`. Time now flows through
  `Service.clock`, eliminating a redundant injection point.
- `diff.GenerateDiff` no longer stamps `DiffResult.Generated`; the
  caller (Service, with its injected clock) does. Keeps the diff
  package clock-free without breaking the API contract — the field
  shape is unchanged, only the responsibility moved.
- `process.ExitInfo.StoppedAt` is now stamped exclusively by
  `Tracker.RecordExit` from the tracker's clock. Callers
  (`handlers/process_start.go`, `toolexecution/adapters.go`) leave it
  zero, removing a duplicated time source.

[CODE: `internal/clock/clock.go`, `internal/testutil/mocks/clock.go`]

## HTTP Middleware + Live-HTTP Test Harness (Round 4 Phase 3)

The 2026-04-28 SSE flusher bug shipped because handler tests used
`httptest.ResponseRecorder`, which natively implements `http.Flusher` and
`http.Hijacker`. The custom `responseWriter` wrapper in `main.go` was
missing those forwarders, so every SSE response 500'd in production.
Phase 3 closes the gap structurally.

### Production middleware lives in `internal/server`

`internal/server/middleware.go::Middleware{Logger, Clock,
CORSAllowedOrigins, UIPortEnv}.Apply(router)` is the single home for the
structured-logging + CORS middleware stack. The unexported
`responseWriter` wrapper (with explicit `Flush`/`Hijack` pass-throughs)
is private to this package; nothing else in the codebase implements an
HTTP middleware. Both `main.go` (production) and the test harness
construct a `Middleware` and call `Apply` on the same `*mux.Router` type,
so the production stack and the test stack are byte-for-byte identical.

### Live-HTTP test harness: `internal/testutil/httpx.NewLiveServer`

`testutil/httpx.NewLiveServer(t, h *handlers.Handlers, opts...)` boots
an `httptest.Server` wired with the production `server.Middleware`,
`handlers.RegisterRoutes`, and `gorillahandlers.RecoveryHandler`. The
returned `*httpx.LiveServer` carries a real `*http.Client`, helpful
`Do`/`DoJSON` wrappers, and a `LogBuffer` capturing every middleware
log line. Tests issue requests through real TCP — every status code,
header, body, and SSE frame flows through the same code path the
production binary runs.

Options:

- `WithClock(c)` — inject a `clock.Clock` (defaults to
  `clock.System{}` or the Handlers' Clock).
- `WithCORSAllowedOrigins([]string)` — seed the strict allowlist.
- `WithUIPortEnv(name)` — override the env var the CORS dev fallback
  reads (defaults to `WORKSPACE_SANDBOX_TESTUTIL_UI_PORT`, deliberately
  not the operator's `UI_PORT`, so tests can't be polluted by ambient
  state).
- `WithMetricsCollector(c)` — wire a metrics collector for endpoints
  that need it.
- `WithExtraRoutes(fn)` — register additional routes (e.g., a
  forced-500 probe to exercise the recovery handler) without touching
  the handlers package.

### Handler tests live in `package handlers_test`

The harness imports `internal/handlers`, so handler tests must live in
the external test package (`handlers_test`) to break the import cycle.
`internal/handlers/handlers_test.go` and
`internal/handlers/process_start_git_allowlist_test.go` migrated to
this layout in Phase 3; the in-package files (`process_loc_test.go`,
`process_git_allowlist_test.go`) remain `package handlers` because they
test pure helpers and do not import the harness.

### SSE regression tests: `internal/handlers/process_sse_test.go`

The five scenarios cover the contract that broke in 2026-04-28:

1. **Fast exit** — process exits before subscription; replayed disk
   content reaches the client and `event: exit` carries the structured
   `ExitInfo` JSON.
2. **Slow exit** — chunks are produced over the SSE connection
   lifetime; ordering invariant holds.
3. **Frame ordering invariant** — across no-data / single-chunk /
   many-chunk patterns, `event: end` is always preceded by
   `event: exit`.
4. **Multi-subscriber fanout** — two concurrent clients see the full
   sequence; one disconnect doesn't break the other.
5. **Client disconnect mid-stream** — context cancellation cleans up
   without leaking; a fresh subscriber after the disconnect still
   completes successfully.

Phase 3 tests originally asserted on raw-body content for streamed
data because the inlined `data: %s\n\n` encoder could not round-trip
multi-line chunks (every line after the first parsed as an unknown
field and was silently dropped). Phase 5 landed the
`internal/sse.HTTPWriter` seam, which encodes multi-line data
correctly; the Phase 3 raw-body checks have been replaced with
per-frame data assertions plus a dedicated multi-line regression
test (`TestStreamProcessLogs_MultiLineDataPreserved`). See
"SSE Writer Seam (Round 4 Phase 5)" below.

### Lint: `internal/handlers/handler_test_pattern_test.go`

A `go ast` walker over every `internal/handlers/*_test.go` fails any
use of `httptest.NewRecorder()` outside an empty allowlist. The
recorder's native Flusher/Hijacker satisfaction is the precise gap
that hid the 2026-04-28 bug, so Phase 3 makes the recorder
unavailable to handler tests by policy. Pure helper tests
(`process_git_allowlist_test.go`, `process_loc_test.go`) don't
construct a recorder, so the lint never fires for them.

### Production-code consolidation Phase 3 made en route

- `main.go::responseWriter` (and its `Flush`/`Hijack` pass-throughs)
  moved into `internal/server`. main.go no longer knows about the
  middleware-internal wrapper.
- `main.go::structuredLoggingMiddleware` and `corsMiddleware` deleted;
  replaced by `server.Middleware{}.Apply(s.router)`.
- The two remaining `time.Now()` / `time.Since(start)` calls in main.go
  (Phase 2 missed them — the DOD greps `internal/`, not the root)
  removed: middleware now reads duration through the injected clock.
- The deprecated `handlers.Health` method was deleted (truly dead — no
  route registered, no remaining callers); api-core/health is the
  canonical health handler in `main.go::setupRoutes`.
- `middleware_response_writer_test.go` at the api-package root was
  removed; the same regression is covered more thoroughly by
  `internal/server/middleware_test.go` against the new package's exact
  surface (Flusher / Hijacker / CORS allowlist / OPTIONS short-circuit
  / clock-driven duration measurement / Apply panics on missing deps).

[CODE: `internal/server/middleware.go`, `internal/server/middleware_test.go`,
`internal/testutil/httpx/server.go`, `internal/handlers/handlers_test.go`,
`internal/handlers/process_sse_test.go`,
`internal/handlers/handler_test_pattern_test.go`,
`internal/handlers/process_start_git_allowlist_test.go`]

## SSE Writer Seam (Round 4 Phase 5)

Round 4 Phase 5 (2026-04-29) extracted the SSE wire format into
`internal/sse`. Before this seam, `internal/handlers/process_logs.go`
inlined every frame with `fmt.Fprintf(w, "data: %s\n\n", chunk)` and
asserted `http.Flusher` per-handler. Two latent bugs piggybacked on
that inlining — and only the second showed up in production:

1. **Multi-line data encoding.** A chunk like `"alpha\nbeta\n"`
   produced wire output that strict SSE parsers (the agent-manager
   consumer included) could not reassemble. The first line was
   reported as a `data:` field; subsequent lines parsed as unknown
   fields and were silently dropped.
2. **`http.Flusher` pass-through.** Phase 3 already moved the
   middleware Flusher assertion into one place; Phase 5 does the same
   for the handler side. Every handler that streams now goes through
   `sse.NewHTTPWriter`, which is the only place that asserts Flusher.

### Layout

| File | Responsibility |
|---|---|
| `internal/sse/sse.go` | `Writer` interface + `HTTPWriter` production impl. Owns Flusher assertion, response headers, write-deadline reset, multi-line `data:` encoding, and the `event: end` close invariant. |
| `internal/sse/parser.go` | `Frame` + `ParseStream`. Inverse of `encodeFrame`; the round-trip property (`ParseStream(encodeFrame(e, d))[0] == {e, d}`) is unit-tested across 11 byte-shape cases including embedded newlines. |
| `internal/sse/sse_test.go` | `HTTPWriter` correctness against `httptest.ResponseRecorder` (legitimate use here): frame ordering, missing-Flusher rejection, header stamping, idempotent Close, write-after-Close = `ErrAlreadyClosed`, multi-flush forwarding, multi-line data round-trips. |
| `internal/sse/parser_test.go` | Parser correctness: data-only frames, named events, multi-line data, comments, trailing frame without blank line. |
| `internal/testutil/mocks/sse_writer.go` | `FakeSSEWriter` records every frame and enforces the same Close-once / no-write-after-Close contract as the production writer. |

### Contract

```go
type Writer interface {
    WriteData(data []byte) error
    WriteEvent(name string, data []byte) error
    Flush() error
    Close() error // emits event: end exactly once; idempotent
}
```

- Multi-line data is encoded per spec — every newline becomes a
  separate `data:` line; the parser reassembles to the original bytes.
- `Close` emits the trailing `event: end\ndata: stream closed\n\n`
  frame (preserving the pre-Phase-5 wire shape; agent-manager's
  consumer ignores the data field of an `end` frame).
- Idempotent Close means handlers can `defer sw.Close()` and also
  call it explicitly without producing duplicate `event: end` frames.
- Writes after Close return `ErrAlreadyClosed` so contract violations
  surface immediately rather than silently dropping bytes.
- Construction failure (`ErrFlusherUnsupported`) does not mutate the
  underlying `http.ResponseWriter` — the handler can still write a
  non-SSE error response on top.

### Handler simplification

`StreamProcessLogs` now uses the seam:

```go
sw, err := sse.NewHTTPWriter(w)
if err != nil { /* 500 */ }
defer sw.Close()
// ...
_ = sw.WriteData(chunk)
_ = sw.WriteEvent("exit", payload)
```

The Flusher assertion, header stamping, write-deadline reset, and
trailing `event: end` are all consolidated. The handler shrank to
204 LOC (≤ 500, pinned by `process_loc_test.go`).

### Production-code consolidation Phase 5 made en route

- The SSE parser previously living in `internal/testutil/httpx/sse.go`
  moved into `internal/sse/parser.go` (renamed `ParseSSEStream` →
  `ParseStream`, `SSEFrame` → `Frame`). The encoder and parser now
  live next to each other so the wire-format invariant is testable
  without crossing a package boundary. `testutil/httpx` now contains
  only the live-HTTP server harness.
- `internal/handlers/process_loc_test.go` extended to also bound
  `process_logs.go` at 500 LOC. The bound is a tripwire: if a future
  change re-balloons the file, the right move is to push more
  encoding logic into `internal/sse`, not to bump the bound.
- The Phase 3 SSE tests previously relied on raw-body substring
  checks for streamed data because the inlined encoder could not
  round-trip multi-line content. Those checks have been replaced
  with per-frame data assertions (`framesByEvent`, `joinFrameData`),
  and a new `TestStreamProcessLogs_MultiLineDataPreserved` regression
  test pins the multi-line invariant directly.

[CODE: `internal/sse/sse.go`, `internal/sse/parser.go`,
`internal/sse/sse_test.go`, `internal/sse/parser_test.go`,
`internal/testutil/mocks/sse_writer.go`,
`internal/testutil/mocks/sse_writer_test.go`,
`internal/testutil/assertx/sse.go`,
`internal/handlers/process_logs.go`,
`internal/handlers/process_sse_test.go`,
`internal/handlers/process_loc_test.go`]

## Audit Emitter Seam (Round 4 Phase 6)

Round 4 Phase 6 (2026-04-29) extracted audit-event emission into
`internal/audit`. Before this seam, three production sites
constructed `&types.AuditEvent{...}` literals inline and called
`Repository.LogAuditEvent` directly:

- `internal/sandbox/service_audit.go::logAuditEventWith` — Service
  layer with sandbox-state snapshot.
- `internal/sandbox/orphan_reconciler.go::logOrphanAuditEvent` —
  orphan-cleanup path (no Sandbox object).
- `internal/gc/gc.go` — GC-collected path.

Each site re-derived the event timestamp, the default ActorType, and
its own error policy. The new seam unifies those three concerns and
makes the Phase 6 DOD invariant
`grep -rn "&types.AuditEvent{" internal/ | grep -v _test.go | grep -v internal/audit/ | grep -v internal/testutil/`
return zero hits.

### Layout

| File | Responsibility |
|---|---|
| `internal/audit/audit.go` | `Event` (input shape) + `Emitter` interface + `RepoEmitter` production impl + `LogFunc` adapter. EventTime is stamped from the injected `clock.Clock`; ActorType defaults to `"system"`; ID is stamped via `uuid.New`. The `&types.AuditEvent{...}` literal lives only here. |
| `internal/audit/audit_test.go` | Stamping correctness: EventTime via clock, UTC normalization, ActorType default, explicit-Source preservation, empty-EventType rejection, log-error propagation, panics on nil log/clock. External test package (`audit_test`) so the test can import `testutil/mocks` for `FakeClock` without creating a cycle. |
| `internal/testutil/mocks/audit_emitter.go` | `FakeEmitter` records the same `*types.AuditEvent` shape `RepoEmitter` would have written. Existing tests that scan `FakeRepository.AuditEvents` continue to work because production paths (`RepoEmitter`) still hit the repo; tests that prefer to spy on the seam directly use `FakeEmitter.Events()`. |

### Contract

```go
type Event struct {
    EventType    string
    SandboxID    *uuid.UUID
    Actor        string
    ActorType    string
    Source       types.ApprovalSource
    Details      map[string]interface{}
    SandboxState map[string]interface{}
}

type Emitter interface {
    Emit(ctx context.Context, e Event) error
}
```

- `EventType` is required — `Emit` returns an error if empty so
  programming bugs surface instead of silently emitting blank events.
- `ActorType` defaults to `"system"` when omitted (matches the
  pre-seam convention used by reconcilers and GC).
- `EventTime` is stamped via the clock and converted to UTC. The
  repository's stamping path remains as a defensive backstop.
- Errors are propagated to the caller. Each call site picks its
  policy: Service logs and continues (`fmt.Printf("warning: ...")`),
  the orphan reconciler logs and continues (`log.Printf(...)`), GC
  surfaces the error in `result.Errors`. The seam doesn't make the
  policy choice for any of them.

### Wiring

Both `sandbox.NewService` and `gc.NewService` now require `audit.Emitter`
as a positional argument (Round 4 greenfield rule — no defaults). Production
wires `audit.NewRepoEmitter(repo.LogAuditEvent, clk)`; tests wire either
the same emitter (preserves `FakeRepository.AuditEvents` assertions) or
`mocks.NewFakeEmitter(clk)` (asserts on the seam directly).

### Production-code consolidation Phase 6 made en route

- `service_audit.go::logAuditEventWith` no longer hand-builds the
  `&types.AuditEvent{}` literal — calls `s.audit.Emit(ctx, audit.Event{...})`
  with the snapshot, preserving the user-vs-system ActorType
  defaulting that's specific to Service-level events.
- `orphan_reconciler.go` switched its nil-safety check from
  `s.repo == nil` to `s.audit == nil` (the seam is the relevant
  guard now; `s.repo` is still required by the constructor).
- `gc/gc.go::Run` calls `s.emitter.Emit(...)` instead of
  `s.repo.LogAuditEvent(...)`. The error path continues to populate
  `result.Errors`; that policy is unchanged.
- `sandbox.Service.audit` and `gc.Service.emitter` are required
  fields — `NewService` panics on nil to fail loudly during boot.

[CODE: `internal/audit/audit.go`, `internal/audit/audit_test.go`,
`internal/testutil/mocks/audit_emitter.go`,
`internal/testutil/mocks/audit_emitter_test.go`,
`internal/sandbox/service.go`, `internal/sandbox/service_audit.go`,
`internal/sandbox/orphan_reconciler.go`,
`internal/gc/gc.go`, `main.go`]

## Schema Version + Profile Snapshot Hardening (Round 4 Phase 9)

Round 4 Phase 9 (2026-04-29) closed two assumption gaps that were
fail-confused-rather-than-fail-fast at startup:

1. **Schema drift was silent.** The embedded SQLite schema is applied
   on every boot via idempotent `CREATE TABLE IF NOT EXISTS`. Two
   ad-hoc column migrations (`driver` → `driver_id`, add
   `home_overlay_state`) lived in `main.go` next to the schema apply.
   Future schema changes risked silent data loss because nothing
   compared the running binary's expectations to the persisted state.
2. **Profile registry was per-request.** `runtime.ProfileResolver`
   held a `config.ProfileStore` and called `Get` on every Resolve.
   File-system mutations to `profiles.json` (or a future external
   profile-source) could change the resolver's behavior mid-process
   in ways tests couldn't pin.

### Schema version

`internal/repository/schema.go` now exposes a single startup entry
point `EnsureSchema(ctx, db, clk)`:

```go
const ExpectedSchemaVersion = 1

func EnsureSchema(ctx context.Context, db *sql.DB, clk clock.Clock) error
```

Behavior:

1. Apply `schema.sql` (idempotent — every CREATE is `IF NOT EXISTS`).
2. Run forward-only legacy migrations (`migrateDriverColumn`,
   `migrateHomeOverlayStateColumn`) — both probe `pragma_table_info`
   before mutating, so they're no-ops on fresh DBs.
3. Read `MAX(version)` from `schema_version`. If empty, write
   `ExpectedSchemaVersion` stamped via the injected clock. If less
   than expected: refuse to start (forward-only migration missing).
   If greater: refuse to start (binary older than database).

The `schema_version` table is part of `schema.sql`:

```sql
CREATE TABLE IF NOT EXISTS schema_version (
    version    INTEGER NOT NULL PRIMARY KEY,
    applied_at TEXT NOT NULL
);
```

`main.go` calls `repository.EnsureSchema(ctx, db, clk)` once at boot;
the inline migration helpers were deleted (greenfield — single source
of truth lives next to the schema). `internal/testutil/db/sqlite.go`
also routes through `EnsureSchema`, so test DBs are byte-identical to
production DBs (including the `schema_version` row).

### Profile snapshot

`runtime.ProfileResolver` now holds an immutable
`Profiles map[string]config.IsolationProfile` instead of a Store
reference. `runtime.LoadProfiles(store)` is the snapshot factory used
at startup and after admin Save/Delete:

```go
type ProfileResolver struct {
    Profiles  map[string]config.IsolationProfile
    DefaultID string
    Caps      driver.DriverCapabilities
}

func LoadProfiles(store config.ProfileStore) (map[string]config.IsolationProfile, error)
```

`Resolve` reads from the snapshot and returns a copy so callers
cannot mutate it via the returned pointer. The resolver is detached
from the underlying store: deletion of a profile from `ProfileStore`
does not affect future `Resolve` calls on a previously-built
resolver. The snapshot lives on `Handlers.profileSnapshot`
(`atomic.Pointer[map[string]config.IsolationProfile]`); admin
Save/Delete handlers call `Handlers.RefreshProfileSnapshot()` after a
successful mutation so the runtime API stays consistent without
requiring a server restart.

External edits to the profiles JSON file (or a hypothetical future
SIGHUP-driven reload) are intentionally ignored after boot. The
contract decision (Round 4 plan §9) is "startup-only for greenfield";
SIGHUP reload is a follow-up if real operators ask for it.

### DOD invariants

- `grep -rn "schema_version" internal/repository/schema.go internal/repository/schema.sql` → present in both.
- `grep -rn "time.Now()" internal/repository/schema.go` → 0 hits (clock-stamped).
- `grep -rn "Store config.ProfileStore" internal/runtime/profile.go` → 0 hits (replaced by `Profiles map`).
- `internal/repository/schema_version_test.go::TestEnsureSchema_RefusesNewerVersionThanExpected` covers drift refusal.
- `internal/runtime/profile_test.go::TestResolveProfile_StartupCacheRejectsUnknown` covers snapshot detachment from Store.

### Files added/changed

- `internal/repository/schema.go` (now hosts `EnsureSchema`,
  `ExpectedSchemaVersion`, and the legacy column migrations).
- `internal/repository/schema.sql` (added `schema_version` table).
- `internal/repository/schema_version_test.go` (new — fresh init,
  idempotent re-init, refusal on older/newer drift, nil-deps,
  legacy-migration end-to-end).
- `internal/runtime/profile.go` (snapshot semantics; `LoadProfiles`).
- `internal/runtime/profile_test.go` (snapshot-detachment contract
  tests; per-call resolver tests in external test package).
- `internal/handlers/handlers.go` (`profileSnapshot` atomic pointer
  + `SetProfileSnapshot`/`RefreshProfileSnapshot`/`ProfileSnapshot`
  accessors).
- `internal/handlers/process.go` (`profileResolver()` reads from the
  snapshot instead of the Store).
- `internal/handlers/admin.go` (`SaveProfile`/`DeleteProfile` refresh
  the snapshot after Store mutation).
- `internal/testutil/db/sqlite.go` (routes through `EnsureSchema`
  instead of raw `repository.SchemaSQL`).
- `main.go` (calls `EnsureSchema` once; loads profile snapshot via
  `runtime.LoadProfiles`; installs it on `Handlers` via
  `SetProfileSnapshot`; legacy migration helpers deleted).

## Control Surface Hardening (Round 4 Phase 8)

Round 4 Phase 8 (2026-04-29) closed three operator-facing gaps that
were "quiet at boot, confusing at runtime":

1. **Validation was loose.** `Config.Validate()` checked a handful of
   fields and ignored mutual-exclusion or dependency constraints. An
   operator could set `AutoCleanupTerminal=false` *and* a non-zero
   `TerminalCleanupDelay` — the delay would be silently ignored.
2. **Resource-limit ordering was implicit.** `DefaultResourceLimits`
   could exceed `MaxResourceLimits` without `Validate` complaining;
   the runtime's clamp logic would silently override the operator's
   default.
3. **Env-var documentation didn't exist as a single reference.** Every
   knob's doc lived in a struct comment in `config.go`. Operators had
   to grep source to learn what they could tune. Drift between code
   and docs was undetectable.

### What Phase 8 added

- **Tightened `Validate()`**: covers ranges (port number, all positive
  durations), mutual exclusion (`AutoCleanupTerminal=false` blocks
  non-zero `TerminalCleanupDelay`), dependency rules (idle timeout
  ≤ default TTL, total size ≥ per-sandbox size, default resource
  limits ≤ max resource limits per resource, non-empty
  `DefaultIsolationProfile`, non-negative
  `AgentManagerSyncTimeout`), and explicit non-negative bounds for
  every duration knob. Seventeen new test cases in
  `config_test.go::TestValidate`.
- **`docs/reference/configuration.md`** now contains a complete env-var
  reference grouped by purpose (Server, Capacity, Lifecycle, Driver,
  Policy, Execution, Integration). Every knob has type, default,
  range, audience, and related-knob entries.
- **`config_test.go::TestExposedKnobs_DocumentationParity`** —
  meta-test that scans `config.go` for env var literals matching the
  canonical prefixes (`WORKSPACE_SANDBOX_*`, `API_PORT`,
  `SQLITE_PATH`, `SANDBOX_BASE_DIR`, `PROJECT_ROOT`) and asserts the
  same set appears in the doc. Drift either direction fails the test.

### What Phase 8 explicitly did NOT do

- **CORS knob deletion.** The plan flagged
  `WORKSPACE_SANDBOX_CORS_ORIGINS` as a possible vestigial removal.
  It is *not* vestigial — `internal/server/middleware.go::corsMiddleware`
  reads it and the live-HTTP harness has parity tests
  (`internal/server/middleware_test.go::TestCORS_*`). Kept and
  documented.
- **AgentManagerURL non-empty requirement.** The plan suggested
  `AgentManagerSyncEnabled=true` should require a URL. It does NOT —
  `Service.resolveAgentManagerURL` falls back to
  `discovery.ResolveScenarioURLDefault` when the URL is empty. The
  validation rule was dropped after empirically confirming the
  discovery contract. Documented in `configuration.md` and the
  `Validate()` comment.
- **AgentManagerSyncTimeout default change.** Already at `5s` in
  `Default()` — the plan's "set 30s default" suggestion was based on
  out-of-date code. Left at 5s; documented.

### DOD invariants

- `Config.Validate()` covers every range, mutual-exclusion, and
  dependency rule in §10 of the Round 4 plan.
- `docs/reference/configuration.md` enumerates every operator knob.
- `internal/config/config_test.go::TestExposedKnobs_DocumentationParity`
  passes — config.go and the doc cannot drift silently.
- `vrooli scenario restart workspace-sandbox` succeeds end-to-end with
  `Validate()` running on boot.

## Mounter & Process Starter Seams (Round 4 Phase 7)

### What Phase 7 introduced

Two canonical seams that consolidate every kernel-mount syscall and
every external-command invocation in workspace-sandbox into a single
home each. The driver, namespace, exec, diff, policy, and interactive
handler layers all route through them.

#### `internal/fsmount/mount.go::Mounter`

- `Mount(ctx, opts)` — mounts overlayfs at `opts.Merged` per
  `opts.Backend` (kernel-overlay or fuse-overlayfs). Required-arg
  validation rejects unset backends and missing layer paths.
- `Unmount(ctx, target, lazy)` — idempotent unmount. Tries
  `syscall.Unmount` first; falls back to `fusermount -u` then `umount
  -l` so a non-root caller without `CAP_SYS_ADMIN` still wins on the
  fuse path.
- `IsMountPoint(path)` — wraps the `mountpoint` binary so mount-state
  checks stay inside the seam.
- `ProbeKernelOverlayMount(ctx, m)` — free function used by the
  namespace package to test overlayfs availability without writing its
  own probe.

Production wiring: `fsmount.NewSystemMounter(starter)` from `main.go`,
threaded through `driver.Deps`. The legacy `mountFunc` / `unmountFunc`
closures and the package-level `isMountPoint` helper are gone.

#### `internal/process/starter.go::Starter`

- `Start(ctx, opts) (Handle, error)` — async spawn. Stdin/Stdout/Stderr
  wiring lives in `StartOpts`; `SysProcAttr` carries `Setpgid` for
  process-group reaping.
- `LookPath(name)` — resolves a binary; normalised to
  `process.ErrBinaryNotFound`.
- `Run(ctx, s, opts)` and `RunCombinedOutput(ctx, s, opts)` — sync
  helpers that capture output. Implemented in terms of `Start+Wait` so
  fakes only need to implement `Start`.
- `Handle.Wait` returns a `ProcessExit` (exit code, signal, OOM flag).
  `KillProcessGroup` is canonical here — every caller that used to
  reach for `syscall.Kill(-pgid, …)` now goes through the seam.

Production wiring: `process.NewOSExecStarter()` in `main.go`. Threaded
through every constructor that spawns: `driver.Deps.Starter`,
`sandbox.NewService(..., starter, …)`,
`toolexecution.ProcessExecutorConfig.Starter`,
`policy.NewHookValidationPolicy(starter, …)`,
`policy.NewHookTeardownPolicy(starter, …)`,
`diff.NewGitOps(starter)` / `NewGenerator(starter)` /
`NewPatcher(starter)`.

#### `internal/process/pty.go::PTYStart`

- The interactive WebSocket handler attaches a pseudo-terminal via
  `pty.StartWithSize`, which requires a `*os/exec.Cmd` directly (PTY
  allocation happens in the fork). PTY-attached spawning is therefore
  its own seam: `PTYStart(opts) (*PTYHandle, error)`. The
  `os/exec` dependency is confined to this single function so the
  Round 4 Phase 7 invariant (no production code outside the seams
  imports `os/exec` or `syscall.Mount`/`syscall.Unmount`) still holds
  for the interactive path.

### Failure modes now reachable from `go test`

| Scenario | Seam knob |
|---|---|
| Kernel `syscall.Mount` returns `EPERM` | `FakeMounter.SetMountErr(...)` |
| Mid-mount partial failure | `FakeMounter.SetMountErrFor(target, err)` |
| `fuse-overlayfs` daemon forks-and-dies | custom `Mounter` returning nil from `Mount` and false from `IsMountPoint` |
| `mountpoint -q` not on PATH | `FakeMounter.IsMountPoint` returns false |
| Binary not in PATH | `FakeStarter.SetLookPath` omits it; `LookPath` returns `ErrBinaryNotFound` |
| `fork()` failure | `FakeStarter.SetStartErr(...)` or per-command `StartErr` |
| Process exits non-zero with custom stderr | per-command `Stdout/Stderr/Exit` script |
| Hung process / context cancel | per-command `Hold: true` or `WaitDelay: ...` |
| OOM-killed process | `Exit: ProcessExit{ExitCode: -1, Signal: 9, OOMKilled: true}` |

### Test fakes

- `internal/testutil/mocks/fsmountmocks/FakeMounter` — records every
  `Mount`/`Unmount`/`IsMountPoint` call, lets tests inject per-target
  errors, pre-seed leftover mount points, and assert call ordering.
- `internal/testutil/mocks/procmocks/FakeStarter` + `FakeHandle` —
  longest-prefix-match command table, per-command behaviors (start
  error, wait error, scripted exit, scripted output, wait delay,
  hold-until-release), call recording with shape (path, args, dir,
  env). Default is paranoid: unmatched commands fail the test.

Both fakes live in subpackages under `internal/testutil/mocks/` to
break a would-be import cycle (process is imported by driver via
`Deps`, so testutil/mocks itself can't depend on process at the top
level without making process tests uncompilable).

### What Phase 7 explicitly did NOT do

- **Move the `process.Tracker` / `process.Logger` to Starter.** Those
  are higher-level constructs that consume the spawned-process exit
  stream; the seam is below them. They stay where they are, with
  Tracker still owning sandbox-PID bookkeeping.
- **Wrap `pty.StartWithSize` behind the abstract Starter contract.**
  PTY allocation requires a real `*os/exec.Cmd`; abstracting it
  through Starter would either lose typing or require parallel APIs.
  PTYStart is its own seam in `internal/process/pty.go`.
- **Migrate `syscall.Exec` in `namespace.EnterUserNamespace`.** That
  call replaces the running process (no fork), which is a different
  semantic from anything Starter exposes. Stays in namespace with a
  comment.

### DOD invariants

- `grep -rn "syscall\.Mount\|syscall\.Unmount" internal/ --include="*.go" | grep -v "_test.go" | grep -v "internal/fsmount/"` returns 0 hits.
- `grep -rn "exec\.Command\|exec\.CommandContext\|cmd\.Start()\|cmd\.Wait()" internal/ --include="*.go" | grep -v "_test.go" | grep -v "internal/process/" | grep -v "internal/fsmount/"` returns 0 hits.
- `go build ./...` clean. `go test ./...` passes for every package.
- `go vet ./...` clean. `gofumpt -l .` empty.
- Cross-scenario parity test in `agent-manager/internal/policy` still passes.

## Driver & Exec Failure-Mode Contract (Round 4 Phase 7+4)

Phase 7 introduced `Mounter` and `Starter` as required seams. Phase 4
exercises every failure mode they enable from `go test` so the failure
contract is permanently observable, not just a paper invariant.

### What Phase 4 added

- `internal/driver/contract_failure_test.go` — failure-mode contract
  test parameterized over every overlay flavor (`fuse-overlayfs`,
  `overlayfs-userns`, `overlayfs-root`) plus a `CopyDriver` block.
  Drives the `OverlayDriver` end-to-end against a `FakeMounter` so
  every failure path is reachable on hosts that can't actually mount
  overlayfs (macOS, sandboxed CI runners, dev machines without
  CAP_SYS_ADMIN).
- `internal/driver/exec/contract_test.go` — failure-mode contract
  test for the exec layer driven by `FakeStarter`. Pins the exit-code,
  signal, OOM, and timeout translations that handlers/process.go
  consumes.
- `FakeMounter.SetSilentMountFor(merged)` — new knob that models the
  "fuse-overlayfs forks-and-dies before signaling failure" /
  "kernel-mount returned 0 but no kernel mount appeared" stale-daemon
  scenarios. With it, `Mount(opts)` returns nil but the merged path
  is NOT registered as a mount point, forcing the driver's
  `verifyMounted` post-mount check to fire.

### Coverage matrix

#### `internal/driver/contract_failure_test.go`

| Scenario | Test | Per-flavor? |
|---|---|---|
| Project mount fails (`EPERM` from kernel/fuse) | `TestDriverFailure_ProjectMountErrorPropagates` | yes |
| Home overlay mount fails (project succeeds) | `TestDriverFailure_HomeOverlayMountFailsSoft` | yes |
| Silent mount failure caught by `verifyMounted` (project) | `TestDriverFailure_SilentMountCaughtByVerify` | yes |
| Silent mount failure on home overlay (soft) | `TestDriverFailure_HomeOverlaySilentVerifySoft` | yes |
| Unmount idempotent (second call is a no-op) | `TestDriverFailure_UnmountIdempotent` | yes |
| Unmount error propagates | `TestDriverFailure_UnmountErrorPropagates` | yes |
| Cleanup / CleanupOrphan idempotent | `TestDriverFailure_CleanupIdempotent` | yes |
| CleanupOrphan unmounts before rm -rf | `TestDriverFailure_CleanupOrphanWhenStillMounted` | yes |
| Partial-approval cycle (Add → Remove → empty) | `TestDriverFailure_PartialApprovalCycle` | yes |
| GetChangedFiles after Unmount still walks UpperDir | `TestDriverFailure_GetChangedFilesAfterUnmount` | yes |
| RemoveFromUpper rejects path traversal | `TestDriverFailure_RemoveFromUpperBlocksTraversal` | yes |
| MountVerifier returns error after Unmount | `TestDriverFailure_VerifyMountIntegrityAfterUnmount` | yes |
| Copy driver Cleanup / CleanupOrphan idempotent | `TestCopyDriverFailure_CleanupIdempotent` | n/a |
| Copy driver missing scope path → rollback | `TestCopyDriverFailure_MissingScopePath` | n/a |
| Copy driver reports `HomeOverlayUnsupported` | `TestCopyDriverFailure_UnsupportedHomeOverlayState` | n/a |

#### `internal/driver/exec/contract_test.go`

| Scenario | Test |
|---|---|
| Process exits 0 (fast path) | `TestExecContract_ExitZero` |
| Process exits non-zero (e.g. exit 7) | `TestExecContract_ExitNonZero` |
| `starter.Start` fails (fork EAGAIN) | `TestExecContract_StartError` |
| `Wait` returns an error alongside the exit state | `TestExecContract_WaitError` |
| Wall-clock timeout → `ExitCode=124` | `TestExecContract_WallClockTimeout` |
| `ModeBwrapRequired` without bwrap on PATH | `TestExecContract_BwrapRequired_NoBwrap` |
| `ModeBwrapPreferred` falls back to direct exec | `TestExecContract_BwrapPreferred_NoBwrap_FallsBackDirect` |
| `ModeBwrapPreferred` routes through bwrap when available | `TestExecContract_BwrapPreferred_HasBwrap` |
| ResourceLimits set but `prlimit` missing | `TestExecContract_BwrapRequired_ResourceLimitsRequirePrlimit` |
| `StartProcess.OnExit` fires once on fast exit | `TestExecContract_StartProcess_OnExitFastExit` |
| `StartProcess.OnExit` carries terminating signal | `TestExecContract_StartProcess_OnExitSignalKilled` |
| `StartProcess.OnExit` carries `OOMKilled=true` | `TestExecContract_StartProcess_OnExitOOMKilled` |
| `StartProcess` reaps even when `OnExit` is nil | `TestExecContract_StartProcess_OnExitNilStillReaped` |
| `StartProcess.Start` failure surfaces (no `OnExit` fire) | `TestExecContract_StartProcess_StartErrorSurfaces` |
| `StartProcess` stdout flushes through `cfg.StdoutWriter` | `TestExecContract_StartProcess_StdoutPiped` |
| `Exec` / `StartProcess` reject sandboxes with empty `MergedDir` | `TestExecContract_RejectsUnmountedSandbox` |
| `Exec` / `StartProcess` panic on nil starter (loud wiring) | `TestExecContract_NilStarterPanics` |

### Why this layering

The pre-Phase 7 driver layer constructed `*os/exec.Cmd` and called
`syscall.Mount` directly, so failure tests would have needed real
subprocesses, real privileges, and real overlayfs kernel support — the
test surface was "anyone running locally can run a quarter of them and
CI can run almost none." The Phase 7 seams pushed every syscall and
exec into `Mounter` and `Starter`. Phase 4 cashes that in: every
failure-mode listed above is now deterministic, runs in milliseconds,
and works on every host `go test` runs on.

The pre-existing `internal/driver/contract_test.go` (real-mount
parameterized lifecycle) stays — it's the load-bearing end-to-end
check that the seams' production impls actually satisfy the driver
contract on a live kernel. Phase 4 didn't replace it; it augmented it
with the deterministic failure-mode coverage that real-mount tests
can't provide on hosts without privileges.

### DOD invariants

- `internal/driver/contract_failure_test.go` and
  `internal/driver/exec/contract_test.go` exist and pass.
- Both files run against `FakeMounter` / `FakeStarter` from
  `internal/testutil/mocks/{fsmountmocks,procmocks}` — no real
  subprocesses, no privileged operations.
- `go test ./internal/driver/... -timeout 60s` passes deterministically
  on a developer host without `CAP_SYS_ADMIN` or a user namespace.
- The original real-mount `TestDriverContract` continues to pass
  whenever `IsAvailable` reports true (i.e. CI hosts with overlayfs).
- Cross-scenario parity test in
  `agent-manager/internal/adapters/sandbox/home_overlay_policy_test.go`
  continues to pass..

## Reliability Round 5 (2026-04-29) — Test-driven seams

Round 5 closes the residual reliability gaps that survived rounds 1–4.
None of these introduce new architecture; they push the existing
seams across the last untestable boundaries.

### Tri-state `HomeOverlayRequirement`

Profiles declare one of `not_needed` / `optional` / `required`. The
optional value carries the decision code `HOME_OVERLAY_FALLBACK` so
callers can record soft fallback rather than refusing the run; the
required value forces the existing HTTP 409 refusal when the sandbox
state is anything other than Present.

[CODE: `internal/types/types.go::HomeOverlayRequirement`] •
[CODE: `internal/config/profiles.go::IsolationProfile.HomeOverlayRequirement`] •
[CODE: `internal/policy/home_overlay.go::CodeHomeOverlayFallback`]

### `internal/driver/changedetect/` walker

Single shared walker plus two strategy implementations
(`OverlayStrategy`, `CopyStrategy`). Replaces the duplicate ~100-line
walks that previously lived in `helpers.go` and `copy.go`. The
contract test in `walker_contract_test.go` parameterises every
edge-case fixture across both strategies so a future drift-by-one bug
fails loudly.

[CODE: `internal/driver/changedetect/walker.go::Walk`] •
[CODE: `internal/driver/changedetect/overlay_strategy.go::OverlayStrategy`] •
[CODE: `internal/driver/changedetect/copy_strategy.go::CopyStrategy`]

### Deterministic daemon teardown on Delete

`Service.Delete` now owns the per-sandbox fuse-overlayfs daemon kill
via `killDaemonsForSandbox`. The background reaper
(`daemon_reaper.go`) stays as a safety net for API-crash paths only;
its kills are now labelled with cause `api_crash` (sandbox marked
deleted but daemon survived) or `orphan` (no row at all). The
`workspace_sandbox_daemon_reaped_total{cause=…}` Prometheus counter
exposes both labels for alerting.

I-MOUNT-1 — Delete returns ⇒ no fuse-overlayfs daemon remains for
that sandbox UUID. Pinned by `delete_daemon_lifecycle_test.go`.

[CODE: `internal/sandbox/delete_daemon.go::Service.killDaemonsForSandbox`] •
[CODE: `internal/metrics/metrics.go::Collector.IncDaemonReaped`]

### Invariant capture

Every behavioural invariant the runtime relies on is now listed in
`docs/internal/INVARIANTS.md` with a stable `I-XXX-N` ID. Each ID
appears as a `t.Run` subtest somewhere in the test tree;
`scripts/check-invariants.sh` is the CI-level scan that enforces this
correspondence so a missing test surfaces at PR time.

[CODE: `docs/internal/INVARIANTS.md`] •
[CODE: `scripts/check-invariants.sh`]
