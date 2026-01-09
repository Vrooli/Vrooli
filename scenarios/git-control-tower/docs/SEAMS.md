# Git Control Tower - Architecture Seams

This document describes the intentional boundaries (seams) in the git-control-tower architecture. Seams are points where behavior can be substituted without invasive changes, primarily for testing and isolation.

## Overview

The git-control-tower API provides structured access to git operations. The primary architectural challenge is **isolating side effects** so that:

1. Unit tests can run without touching the filesystem, executing real git commands, or requiring a database
2. Integration tests can verify real git behavior in isolated temp directories
3. Domain logic (parsing, validation, business rules) can be tested independently

## Primary Seam: GitRunner Interface

**Location**: `api/git_runner.go`

The `GitRunner` interface is the primary seam for all git operations. It abstracts:

- `StatusPorcelainV2` - Get repository status
- `Diff` - Get file diffs
- `Stage` / `Unstage` - Staging area operations
- `Commit` - Create commits with message
- `RevParse` - Repository validation
- `LookPath` - Git binary availability check
- `ResolveRepoRoot` - Determine repository root directory (environment or git detection)

### Production Implementation: ExecGitRunner

The `ExecGitRunner` struct implements `GitRunner` by shelling out to the real git binary. It is used when the API runs in production.

```go
srv := &Server{
    git: &ExecGitRunner{GitPath: "git"},
}
```

### Test Implementation: FakeGitRunner

**Location**: `api/git_runner_fake_test.go`

The `FakeGitRunner` implements `GitRunner` without touching the filesystem:

```go
fakeGit := NewFakeGitRunner().
    WithBranch("main", "origin/main", 2, 1).
    WithRepoRoot("/fake/repo").
    AddStagedFile("file.go").
    AddUnstagedFile("config.yaml").
    AddUntrackedFile("notes.txt")

repoDir := fakeGit.ResolveRepoRoot(ctx) // Returns "/fake/repo"
status, err := GetRepoStatus(ctx, RepoStatusDeps{
    Git:     fakeGit,
    RepoDir: repoDir,
})
```

Features:
- Simulates repository state in memory
- Error injection via `StatusError`, `DiffError`, `StageError`, etc.
- Call tracking via `AssertCalled()`, `CallCount()`
- Builder methods for fluent test setup (`WithBranch`, `WithRepoRoot`, `AddStagedFile`, etc.)

## Service Layer Boundaries

Each service function accepts a `*Deps` struct containing its dependencies:

| Service | Deps Struct | Dependencies |
|---------|-------------|--------------|
| `GetRepoStatus` | `RepoStatusDeps` | GitRunner, RepoDir |
| `GetDiff` | `DiffDeps` | GitRunner, RepoDir |
| `StageFiles` / `UnstageFiles` | `StagingDeps` | GitRunner, RepoDir |
| `CreateCommit` | `CommitDeps` | GitRunner, RepoDir |
| `HealthChecks.Run` | `HealthCheckDeps` | DBChecker, GitRunner, RepoDir |

This pattern enables:
- Explicit dependency injection
- Easy substitution in tests
- Clear documentation of what each service needs

## Parser Layer (Pure Functions)

Parsers are pure functions with no side effects:

- `ParsePorcelainV2Status(output []byte)` - Parse git status output
- `ParseDiffOutput(raw string)` - Parse diff output

These can be tested directly with static input strings, no mocks needed.

## Test Organization

### Unit Tests (FakeGitRunner)

Fast tests that exercise domain logic without real git:

```go
func TestGetRepoStatus_WithFakeGit(t *testing.T) {
    fakeGit := NewFakeGitRunner().AddStagedFile("file.go")
    // Test logic without filesystem or git binary
}
```

### Integration Tests (Real Git)

Tests that verify ExecGitRunner works correctly with the git binary:

```go
func TestStageFiles_WithRealRepo(t *testing.T) {
    if _, err := exec.LookPath("git"); err != nil {
        t.Skip("git not available in PATH")
    }
    repoDir := SetupTestRepo(t) // Creates temp git repo
    // Test with real git commands
}
```

### Parser Tests (Pure Functions)

Tests that verify parsing logic with static inputs:

```go
func TestParsePorcelainV2Status_BranchAndFiles(t *testing.T) {
    out := []byte("# branch.head main\n...")
    parsed, err := ParsePorcelainV2Status(out)
    // Verify parsed result
}
```

## Shared Test Utilities

**Location**: `api/testutil_test.go`

Consolidated helpers for integration tests:

- `RunGitCommand(t, dir, args...)` - Execute git with proper env
- `SetupTestRepo(t)` - Create temp repo
- `WriteTestFile(t, path, content)` - Create test files
- `AssertContains(t, slice, expected)` - Assertion helper

## Seam Verification Checklist

When modifying the API, verify:

- [ ] New git operations go through `GitRunner` interface
- [ ] No direct `exec.Command("git", ...)` outside of `ExecGitRunner`
- [ ] New service functions accept deps struct with `GitRunner`
- [ ] Unit tests use `FakeGitRunner` for service logic
- [ ] Integration tests skip gracefully when git unavailable
- [ ] Parser functions remain pure (no side effects)

## Secondary Seam: DBChecker Interface

**Location**: `api/db_checker.go`

The `DBChecker` interface abstracts database health checks:

- `Ping` - Check if database is reachable
- `IsConfigured` - Check if database handle exists

### Production Implementation: SQLDBChecker

The `SQLDBChecker` struct wraps a `*sql.DB` and delegates to its `PingContext` method:

```go
checks := NewHealthChecks(HealthCheckDeps{
    DB:      NewSQLDBChecker(db),
    Git:     git,
    RepoDir: repoDir,
})
```

### Test Implementation: FakeDBChecker

**Location**: `api/db_checker_fake_test.go`

The `FakeDBChecker` simulates database connectivity:

```go
fakeDB := NewFakeDBChecker()                    // Connected by default
fakeDB := NewFakeDBChecker().WithUnconfigured() // No database
fakeDB := NewFakeDBChecker().WithDisconnected() // Unreachable
fakeDB := NewFakeDBChecker().WithPingError(err) // Specific error
```

Features:
- Simulates database configuration state
- Error injection via `PingError`
- Call tracking via `PingCalls`

## HTTP Response Seam

**Location**: `api/http_response.go`, `api/http_handler.go`

HTTP responses are centralized through the `HTTPResponse` helper:

```go
resp := NewResponse(w)
resp.OK(data)           // 200 OK
resp.BadRequest(msg)    // 400 Bad Request
resp.InternalError(msg) // 500 Internal Server Error
```

Request handling utilities in `http_handler.go` provide:
- `ParseJSONBody` - Consistent JSON parsing with error handling
- `ValidateStagingRequest` - Staging operation validation

These reduce cognitive load by ensuring consistent response patterns across all handlers.

## Tertiary Seam: AuditLogger Interface

**Location**: `api/audit_logger.go`

The `AuditLogger` interface abstracts audit logging operations:

- `Log` - Record an audit entry for mutating operations
- `Query` - Retrieve audit entries matching a request
- `IsConfigured` - Check if audit logging is available

### Production Implementation: PostgresAuditLogger

The `PostgresAuditLogger` writes audit entries to the `git_audit_log` PostgreSQL table:

```go
srv := &Server{
    audit: NewPostgresAuditLogger(db),
}
```

### Test Implementation: FakeAuditLogger

**Location**: `api/audit_logger_fake_test.go`

The `FakeAuditLogger` simulates audit logging in memory:

```go
fakeAudit := NewFakeAuditLogger()
// ... perform operations ...
if fakeAudit.HasOperation(AuditOpCommit) {
    // Verify commit was logged
}
if fakeAudit.CountOperation(AuditOpStage) != 2 {
    // Verify stage count
}
```

Features:
- In-memory entry storage
- Operation filtering via Query
- Test helpers: `HasOperation()`, `CountOperation()`, `LastEntry()`
- Configurable state via `WithUnconfigured()`

### Graceful Degradation: NoOpAuditLogger

When database is unavailable, `NoOpAuditLogger` silently succeeds without storing:

```go
logger := &NoOpAuditLogger{}
logger.Log(ctx, entry) // No-op, returns nil
```

---

*Last updated: 2025-12-16*
