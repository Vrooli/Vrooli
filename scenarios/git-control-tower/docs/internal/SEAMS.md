# Git Control Tower - Architecture Seams

This document describes intentional boundaries ("seams") in git-control-tower. Seams are points where behavior can be substituted without invasive changes, primarily for testing and side-effect isolation.

## Primary Seam: GitRunner Interface

**Location**: `api/git_runner.go`

`GitRunner` is the seam for all git operations. It isolates filesystem and process execution from domain logic.

Production implementation:
- `ExecGitRunner` shells out to the real git binary.

Test implementation:
- `FakeGitRunner` (in `api/git_runner_fake_test.go`) simulates repo state in memory and records calls.

Example test usage:
```go
fakeGit := NewFakeGitRunner()
fakeGit.Branch.Head = "main"
fakeGit.Staged["file.go"] = "+added line"
fakeGit.CloneError = fmt.Errorf("clone failed")

out, err := fakeGit.Diff(ctx, "/fake/repo", "file.go", true)
```

Seam guardrails:
- No `exec.Command("git", ...)` outside `ExecGitRunner`.
- New git operations must be added to `GitRunner` and implemented by both Exec and Fake runners.

## Repo Selection & Registry Seam

**Locations**:
- `api/repo_service.go`
- `api/repo_store.go`
- `api/http_handler.go`

`RepoService` owns repository registry and resolution rules:
- Resolves repositories by `X-Repo-Id` header or `repo_id` query param.
- Falls back to the active repo in the registry.
- Falls back to `GitRunner.ResolveRepoRoot()` when no registry entry exists.

`RepoStore` is the persistence seam for repo metadata and active state. The production implementation is `SQLiteRepoStore`.

`RepoOperation` is the HTTP boundary that resolves repo context for a request and centralizes repo-specific error handling. All repo-dependent handlers should use it.

## Service Layer Dependency Seams

Service functions accept explicit `Deps` structs to keep domain logic testable. Examples:
- `RepoStatusDeps` (git + repo dir)
- `DiffDeps`
- `BranchDeps`
- `FileDeps`
- `CredentialsDeps`

This pattern makes it straightforward to swap `GitRunner` or supply custom repo dirs in tests.

## Audit Logging Seam

**Location**: `api/audit_logger.go`

`AuditLogger` abstracts audit writes and queries. Production uses `SQLiteAuditLogger` and tests can use `FakeAuditLogger` (with call tracking). The server falls back to `NoOpAuditLogger` if the DB is unavailable.

## Database Health Seam

**Location**: `api/db_checker.go`

`DBChecker` abstracts database connectivity for health checks. `FakeDBChecker` supports testing scenarios without real DB access.

## Parser Seams (Pure Functions)

Parsers are intentionally pure, which makes them ideal unit-test targets without mocks:
- `ParsePorcelainV2Status`
- `ParseDiffOutput`
- Branch parsers in `branch_parser.go`

## Verification Checklist

When adding new behavior, verify:
- Git operations go through `GitRunner`.
- Repo-resolving handlers use `RepoOperation`.
- Repo registry updates go through `RepoService`/`RepoStore`.
- Tests can swap in `FakeGitRunner` or `SQLiteRepoStore` (memory DB).

