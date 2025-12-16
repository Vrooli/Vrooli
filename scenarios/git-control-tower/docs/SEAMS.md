# Git Control Tower - Architecture Seams

This document describes the intentional boundaries (seams) in the git-control-tower architecture. Seams are points where behavior can be substituted without invasive changes, primarily for testing and isolation.

## Overview

The git-control-tower API provides structured access to git operations. The primary architectural challenge is **isolating git side effects** so that:

1. Unit tests can run without touching the filesystem or executing real git commands
2. Integration tests can verify real git behavior in isolated temp directories
3. Domain logic (parsing, validation, business rules) can be tested independently

## Primary Seam: GitRunner Interface

**Location**: `api/git_runner.go`

The `GitRunner` interface is the primary seam for all git operations. It abstracts:

- `StatusPorcelainV2` - Get repository status
- `Diff` - Get file diffs
- `Stage` / `Unstage` - Staging area operations
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
| `HealthChecks.Run` | `HealthCheckDeps` | DB, GitRunner, RepoDir |

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

## Future Seam Considerations

### Database Operations

Currently `HealthChecks` accepts `*sql.DB` directly. Consider:
- Creating a `DBChecker` interface for testability
- Allowing mock database health checks

---

*Last updated: 2025-12-16*
