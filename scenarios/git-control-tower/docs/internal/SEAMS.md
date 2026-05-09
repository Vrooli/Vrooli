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
- `CommitDeps` (git + optional precommit/check recorders)
- `RepoHistoryDeps` (git + optional commit-check reader)

This pattern makes it straightforward to swap `GitRunner` or supply custom repo dirs in tests.

## Precommit Command and Commit Check Seams

**Locations**:
- `api/precommit_service.go`
- `api/commit_check_store.go`
- `api/commit_service.go`
- `api/repo_status_service_helpers.go`

`CommandRunner` is the seam for arbitrary configured precommit commands. Production uses `ShellCommandRunner`; tests should provide a fake runner and must not execute shell commands just to simulate pass/fail/timeout behavior.

`CommitCheckRecorder` and `CommitCheckReader` isolate commit-scoped check persistence from commit creation and history rendering. Production uses `CommitCheckStore` backed by SQLite. Tests can use `api/internal/testutil/db` plus `ensureRepoSchema`, or a small fake reader/recorder when persistence is not under test.

Guardrails:
- Treat configured precommit commands as opaque repo-agnostic user data.
- Do not infer historical check status from repo-level `git_repo_precommit.last_*` fields.
- Do not run real git or real precommit commands in unit tests; use `FakeGitRunner` and fake `CommandRunner`.
- Only persist commit-scoped check runs after a git commit succeeds and a commit hash exists.

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

## Workspace-Sandbox Communication Seam

**Location**: `api/workspace_sandbox_api.go`

`WorkspaceSandboxAPI` is the seam for all workspace-sandbox operations. It isolates cross-scenario HTTP communication from handler and commit logic.

Production implementation:
- `WorkspaceSandboxClient` (in `api/workspace_sandbox_client.go`) makes HTTP requests to workspace-sandbox, resolved via `discovery.ResolveScenarioURLDefault`.

Test implementation:
- `FakeWorkspaceSandboxAPI` (in `api/workspace_sandbox_fake_test.go`) records calls and returns configurable responses.

Methods:
- `GetCommitPreview` / `GetCommitPreviewForPaths` — read pending approved changes
- `MarkCommitted` — notify WS that files were committed externally (called async after successful commit)
- `GetProvenanceByRun` — fetch pending changes grouped by agent-manager run ID

Seam guardrails:
- No direct HTTP calls to workspace-sandbox outside `WorkspaceSandboxClient`.
- The `Server.sandbox` field is typed as `WorkspaceSandboxAPI` (interface), not the concrete client.
- `MarkCommitted` is fire-and-forget; failures must not block the commit response.

## Cross-Scenario HTTP Client Test Seam

**Locations**:
- `api/*_client.go`
- `api/internal/testutil/httpx/`

Cross-scenario clients are the only API layer that should know the concrete HTTP shape of neighboring scenarios such as agent-manager, test-genie, scenario-auditor, browser-automation-studio, tidiness-manager, and workspace-sandbox.

Test helpers in `internal/testutil/httpx` provide:
- `NewServer` for route-scoped `httptest.Server` setup with automatic cleanup.
- `NewHandlerServer` for mux/custom handler tests with the same cleanup behavior.
- `AssertMethod` for consistent method checks.
- `DecodeJSON` for typed request-body decoding.
- `WriteJSON` for status/content-type/encoding consistency.
- `TestClient` for timeout-safe client construction.

Guardrails:
- Client and integration tests should use `httpx` helpers instead of repeating `httptest.NewServer`, `json.NewEncoder`, and ad hoc timeout setup.
- Production code must not import `git-control-tower/internal/testutil/...`; this is enforced by `api/internal/testutil/no_prod_import_test.go`.
- Handler tests can keep package-local fixtures when they need access to unexported server internals, but shared cross-scenario HTTP behavior belongs in `httpx`.

Package-local fakes such as `FakeGitRunner`, `FakeAuditLogger`, `FakeDBChecker`, `FakeFileIO`, and `FakeWorkspaceSandboxAPI` intentionally remain in root `_test.go` files while the API package is `package main`. They model root-package interfaces and, in some cases, unexported behavior. Generic helpers that do not need root-package access belong in `internal/testutil`; `api/testutil_test.go` should remain a thin compatibility wrapper only.

## API Test Fixture and Persistence Seam

**Locations**:
- `api/internal/testutil/fixtures/`
- `api/internal/testutil/db/`
- `api/scenario_envelope_test.go`
- `api/visual_capture_service_test.go`
- `api/audit_logger_test.go`
- `api/repo_store_test.go`

Tests that need a Vrooli repository layout should use `fixtures.WriteRepoContract` and `fixtures.WriteScenarioServiceJSON`. Tests that need SQLite should use `db.OpenSQLiteMemory` or `db.OpenSQLiteFile`, then run the package-local schema initializer that owns the production schema under test.

Guardrails:
- Do not recreate repo-contract or scenario `service.json` fixtures inline in package tests.
- Do not open SQLite handles directly in new tests unless the test is specifically about driver-level behavior.
- The `db` helpers own temp paths and cleanup; package tests remain responsible for invoking the schema setup they are validating.

## UI Test Harness Seam

**Locations**:
- `ui/src/test-setup.ts`
- `ui/src/test-utils/`
- `ui/vite.config.ts`

Vitest loads one shared setup file for all React tests. It owns:
- central `@testing-library/jest-dom` registration.
- deterministic `@vrooli/api-base` behavior.
- global fetch/storage cleanup after each test.
- shared React Query render helpers through `renderWithQueryClient` and `renderHookWithQueryClient`.
- fetch and viewport helpers for UI surfaces that vary by API response or breakpoint.

Guardrails:
- New React Query component/hook tests should use `renderWithQueryClient` or `renderHookWithQueryClient` instead of creating one-off providers.
- Tests that need API responses should use `mockFetchJson`, `jsonResponse`, or `textResponse` unless they are specifically asserting raw fetch wiring.
- Mobile/desktop tests should use the viewport helpers instead of mutating globals inline.

## Verification Checklist

When adding new behavior, verify:
- Git operations go through `GitRunner`.
- Precommit command execution goes through `CommandRunner`.
- Commit-check history goes through `CommitCheckRecorder` / `CommitCheckReader`.
- Workspace-sandbox operations go through `WorkspaceSandboxAPI`.
- Repo-resolving handlers use `RepoOperation`.
- Repo registry updates go through `RepoService`/`RepoStore`.
- Tests can swap in `FakeGitRunner`, `FakeWorkspaceSandboxAPI`, or `SQLiteRepoStore` (memory DB).
- Cross-scenario HTTP client tests use `api/internal/testutil/httpx`.
- Repository layout fixtures use `api/internal/testutil/fixtures`.
- SQLite persistence tests use `api/internal/testutil/db`.
- UI tests use the shared setup and React Query/fetch/viewport helpers.
