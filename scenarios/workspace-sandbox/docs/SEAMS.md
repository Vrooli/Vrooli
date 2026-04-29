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
