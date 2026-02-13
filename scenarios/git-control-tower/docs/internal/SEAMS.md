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

## UI Diff Minimap Seams

**Location**: `ui/src/components/DiffViewer.tsx`

The minimap feature is split into pure and imperative seams:

- Pure seam:
  - `buildMinimapMarkers(annotatedLines)` maps annotated diff lines to a bounded marker model.
  - `buildMinimapTextureRows(lines)` maps file content into lightweight structure-texture rows.
  - `scrollTopFromMinimapPointer(pointerOffsetY, railHeight, scrollHeight, clientHeight)` maps minimap pointer position to target scroll offset.
- Imperative seam:
  - `DiffViewer` owns DOM sync between scroll container and minimap viewport.
  - Pointer/keyboard minimap interaction routes through those pure mapping functions.

Guardrails:
- Minimap only appears in desktop `source`/`full_diff` modes and only for long files.
- Existing diff rendering and stage/unstage/discard flows remain unchanged.

## File Content Editing Seam

**Locations**:
- `api/file_content_service.go`
- `api/files_handler.go` (`PUT /api/v1/repo/files/content`)
- `ui/src/components/DiffViewer.tsx`

The editing flow is split across a strict backend seam and a UI seam:

- Backend seam:
  - `SaveFileContent` enforces path sanitization (`cleanFilePath`), text-only constraints, size limits, and optimistic concurrency via `expected_hash`.
  - Writes are atomic via `storage.WriteFileAtomic`.
  - Conflicts surface as `FileContentConflictError` (HTTP 409 with current hash).
- UI seam:
  - `DiffViewer` enables edit/save only in `source` and `full_diff` modes for text files.
  - Monaco (`@monaco-editor/react`) is the single editor surface.
  - Save conflicts are handled explicitly and shown to users with the latest hash.

Guardrails:
- Editing is disabled in history mode (`viewingCommit`).
- Binary/unsupported files remain read-only.
- Stage/unstage/discard semantics are unchanged; save does not auto-stage.

## Verification Checklist

When adding new behavior, verify:
- Git operations go through `GitRunner`.
- Repo-resolving handlers use `RepoOperation`.
- Repo registry updates go through `RepoService`/`RepoStore`.
- Tests can swap in `FakeGitRunner` or `SQLiteRepoStore` (memory DB).
