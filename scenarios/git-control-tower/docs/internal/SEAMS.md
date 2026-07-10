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

## Worktree Domain Seams

The worktree domain is the first proto+Connect-RPC domain in GCT. Two
narrow seams replace direct `git worktree ...` invocations everywhere a
caller needs worktree-shaped data or mutation:

| Seam | Declaration | Production Impl | Test Double | Why it exists |
|---|---|---|---|---|
| `worktree.Inspector` | `api/internal/worktree/inspector.go` | `gitInspector` in `api/internal/worktree/git_impl.go` | `mocks.FakeInspector` in `api/internal/worktree/mocks/inspector.go` | Read-side worktree state (list, identify path, claimed branches). Test doubles wire it everywhere — handlers, repo service, branch enrichment — so NO real git is invoked in tests. |
| `worktree.Mutator` | `api/internal/worktree/mutator.go` | `gitMutator` in `api/internal/worktree/git_impl.go` | `mocks.FakeMutator` in `api/internal/worktree/mocks/mutator.go` | Write-side worktree operations (add/remove/lock/unlock/move/prune). Service-layer validation refuses unsafe operations (e.g. remove main) before the Mutator is ever invoked. |
| Branch-list enrichment | `claimedBranchesFn` in `api/branch_handler.go` | Lazy `newWorktreeInspector().ClaimedBranches` | Test-time override (see `branch_worktree_test.go`) | Lets REST `/api/v1/repo/branches` populate `checked_out_in_worktree` without touching git in tests. Errors are intentionally swallowed; empty string is the unclaimed sentinel. |
| CLI client factory | `clientFactory` in `cli/domains/worktree/handlers.go` | `cliapp.NewConnectHTTPClient` + `worktreeconnect.NewWorktreeServiceClient` | Test `fakeClient` (see `handlers_test.go`) | Substitutes the entire WorktreeService client so CLI command flag-plumbing tests need no network or git. |

Compile-time satisfaction: every production and fake impl carries a
`var _ worktree.Inspector = (*X)(nil)` / `var _ worktree.Mutator = ...`
assertion. Renaming a seam method fails the build everywhere it must.

Connect-RPC mount point: `api/connect_wiring.go::mountConnectHandlers`.
The new WorktreeService and RepoService handlers register through
`api-core/connectx.RegisterServices`; existing flat-package REST handlers
are untouched.

Hard rule for this domain: **tests NEVER invoke real `git`**. The
production seam impls are only reachable at runtime. If you find
yourself reaching for a real-git integration test, add another
table-driven case against `FakeInspector` / `FakeMutator` instead.

## Agent-Access Policy Gate Seams

The agent-access policy gate (see `docs/concepts/ARCHITECTURE.md` →
"Policy Gate") introduces three narrow seams that are all individually
unit-testable without standing up a full server:

| Seam | Declaration | Production Impl | Test Double | Why it exists |
|---|---|---|---|---|
| Policy config loader | `api/internal/config/config.go::Load` | Reads `<scenarioDir>/.vrooli/config.json` `policy` block, merges over `DefaultConfig()`. | Table-driven `config_test.go` writes synthetic JSON into a `t.TempDir()` directory. | Lets an operator tune `agentAccess`/`callerDetection`/override-flag/message template without rebuilding. |
| Gate decision (pure) | `api/internal/policygate/policygate.go::Decide` | Pure function over `(CallerKind, CommandSpec, OverrideFlags, PolicyConfig)`. | Matrix tests in `policygate_test.go`. | Single source of truth for the allow/warn/deny/confirm matrix. Importable from both the API interceptor and the (future) CLI gate. |
| Connect server interceptor | `api/internal/policygate/interceptor.go::NewInterceptor` | Wraps unary Connect handlers; reads `X-Vrooli-Caller` + `X-Vrooli-Authorized`, falls back to `cliutil.DetectCallerKind()`, applies `Decide`. | `interceptor_test.go` mounts a `UnimplementedWorktreeServiceHandler` shim with the interceptor and asserts allow/deny/warn paths. | Defense in depth for direct curl callers and the canonical enforcement layer. |
| Connect client header stamp | `cli/internal/callerheader/interceptor.go::New` | Stamps every outbound RPC with `X-Vrooli-Caller` + (when env says so) `X-Vrooli-Authorized`. | `interceptor_test.go` mounts a recording handler. | Lets the CLI tell the server who the caller is without making the server guess. |

Audit log line shape (emitted by the server interceptor's
`StdAuditLogger`):

```
policygate event caller=<kind> procedure=<proc> effect=<write|destructive> policy=<agentAccess> decision=<allow|warn|deny> authorized=<bool>
```

Operator-facing config file:
`scenarios/git-control-tower/.vrooli/config.json` (top-level `policy`
key). See `api/internal/config/config.go` for schema.

## Baseline Seams

**Location**: `api/internal/baseline/` (declarations) + `api/baseline_clients.go` (production wiring) + `api/internal/git/state.go`.

The baseline subsystem anchors **one** comprehensive, durable Test Genie run and pins it once. It owns a run identity, not artifacts or phase groupings. Every external dependency is an injected interface so the orchestration `Service` is unit-testable with fakes — the `baseline` package never imports the flat `main` package (no import cycle), so all live-dependency wiring lives in `baseline_clients.go`. Test Genie owns phase identity, descriptors, comparison semantics, evidence catalogs, and artifact access; GCT consumes the complete `PhaseDiff[]` without a local registry.

| Seam | Declaration | Production Impl | Test Double | Why it exists |
|---|---|---|---|---|
| `Executor` | `internal/baseline/seams.go` | `baselineExecutor` (test-genie `RunsService` durable RPCs: `StartRun` with `preset=comprehensive` + `captureProfile=baseline`, then `WaitRun`+`GetRun`) in `baseline_clients.go` | `fakeExecutor` in `fakes_test.go` | Two-phase so a snapshot can return the run handle immediately and pin server-side on completion: `StartRun` returns `{runID, eta}` without blocking; `AwaitResult` blocks to terminal and reads the phase set. Capture and diff both reuse one run. |
| `RunsClient` | `internal/baseline/seams.go` | `baselineRunsClient` (test-genie `RunsService` Connect-RPC) | `fakeRuns` in `fakes_test.go` | Pin/unpin the shared run; `CompareRuns` returns every phase delta; `ListRunArtifacts` returns the path-free typed evidence catalog; `CompareRunVisuals` supplies advisory visual deltas. |
| `StalenessProbe` | `internal/baseline/seams.go` | `baselineStalenessProbe` (read-only `git rev-list`/`diff`) | injected fake in `service_test.go` | Commits/files-changed since the baseline sha; read-only (`feedback_no_git_mutations`). |
| `Reachability` | `internal/baseline/seams.go` | `baselineReachability` (short-timeout `GET /health` on the discovery-resolved test-genie URL, ~5s) in `baseline_clients.go` | `fakeReachability` in `fakes_test.go` | Fast, bounded liveness check probed BEFORE committing to the multi-minute comprehensive run. Unreachable → capture skips every surface (clear reason) / diff marks surfaces not-comparable, instead of blocking to the 15m/30m client deadlines (the reported silent-hang fix). Replaced the old `exec==nil||runs==nil` stub. |
| `CaptureGit` | `internal/baseline/service.go` (`Deps.CaptureGit`) | `git.Capture` (`internal/git/state.go`) | injected func in `service_test.go` | Reads sha/branch/dirty/detached; sandbox-aware; never mutates. |

Seam guardrails:
- The `baseline` package imports neither `main` nor `connectrpc.com/connect`; transport + live clients stay in `baseline_clients.go` / `handlers/baseline/`.
- Baseline manifests are pointers only — every surface references the one pinned test-genie run; they never copy artifacts.
- A baseline pins ONE run and unpins it ONCE on delete.
- `BaselineStorage` is branch-scoped and `flock`-protected (`storage.go`); writes are atomic temp-file renames.
- **Snapshot durability (return-fast):** `SnapshotForBaseline` STARTS the comprehensive run, records a durable snapshot intent, and returns immediately with `{run_id, estimated_total_seconds, eta_known}` (via `Service.StartCapture`). The pin + manifest write happen on a server-owned goroutine (`Service.FinalizeCapture` under `context.WithoutCancel(ctx)` + `snapshotTailCeiling`); if that attachment expires or GCT restarts, the intent remains pending and startup recovery / `baseline snapshot status` reattaches. Only terminal Test Genie failure marks it failed. The heavy run itself is durable in test-genie's `runmanager`; GCT keeps NO parallel job system.
- **Diff durability (return-fast + recoverable wait):** `StartDiff` resolves the current comprehensive run, records a durable diff intent, and returns the run id immediately. `FinalizeDiff` computes and caches the verdict on a server-owned context; `GetDiffResult` can recover the latest intent for a baseline when called with `latest=true`, so an interrupted `--wait` can reattach without guessing from test-genie run history.

### Observability Surface (baseline snapshot)
- **States/transitions:** `SnapshotForBaseline` logs a "started comprehensive run" line (`scenario`, `name`, `run`, `eta`) up front and, when finalization completes, a "pinned" line (`run`, `surfaces`, `skipped`) or a "finalize FAILED" line. The CLI prints an up-front banner — run id + ETA + the quiet `baseline snapshot status --run <run-id>` reattach command plus the human `test-genie runs follow <scenario> <run-id>` live-watch command — and returns immediately, so the snapshot never blocks or reads as a silent hang.
- **Skip reasons are first-class:** every fast-skip carries its cause into the manifest's `skipped` map (`comprehensive run failed: …`), surfaced by `show`/`diff` so a partial baseline can't masquerade as complete.
- **Signal stability:** the structured re-attach verb is `baseline snapshot status`; it reports `pending`, `ready`, `failed`, or `missing` and carries similar-name hints when the manifest is absent. The baseline becomes queryable via `baseline show`/`diff` once the run completes. The visuals surface verdict vocabulary gains the advisory `changed` tier (never gates; diff exit code unchanged).

### Observability Surface (baseline diff)
- **Start signal:** `baseline diff --wait` prints the run id once before blocking, preserving machine-readable JSON on stdout by writing that recovery notice to stderr when `--json` is used. It intentionally does not emit heartbeat/progress lines while waiting.
- **Recovery signal:** `baseline diff status --latest --scenario <s> --name <n>` resolves the newest durable diff intent for that baseline; `--run` remains the precise reattach path.
- **Server logs:** `GetDiffResult` logs one completion line per request with `scenario`, `name`, resolved `run`, `latest`, `wait`, `status`, `verdict`, `next_check`, and duration; errors include the same request identifiers and elapsed time.

## Verification Checklist

When adding new behavior, verify:
- Git operations go through `GitRunner`.
- Worktree operations go through `worktree.Inspector` / `worktree.Mutator` — never `exec.Command("git", "worktree", ...)` outside the production seam impls in `internal/worktree/git_impl.go`.
- Tests of worktree-aware code substitute `FakeInspector` / `FakeMutator` and never invoke real git.
- Precommit command execution goes through `CommandRunner`.
- Commit-check history goes through `CommitCheckRecorder` / `CommitCheckReader`.
- Workspace-sandbox operations go through `WorkspaceSandboxAPI`.
- Repo-resolving handlers use `RepoOperation`.
- Repo registry updates go through `RepoService`/`RepoStore`.
- Tests can swap in `FakeGitRunner`, `FakeWorkspaceSandboxAPI`, or `SQLiteRepoStore` (memory DB).
- Cross-scenario HTTP client tests use `api/internal/testutil/httpx`.
- Repository layout fixtures use `api/internal/testutil/fixtures` — `WriteRepoContract` copies the **live** `.vrooli/repo-contract.json` verbatim; never hand-type a contract literal (it drifts when the schema gains a required field).
- Baseline dependencies (`Executor`, `RunsClient`, `StalenessProbe`, `CaptureGit`) are injected via `baseline.Deps`; tests use the fakes in `internal/baseline/fakes_test.go`, never live clients.
- A baseline is ONE comprehensive run, pinned once. Diff exposes the complete `PhaseDiff[]` and typed base/current evidence catalogs. Visual comparison remains advisory; GCT does not copy baseline artifacts.
- The standalone GCT visual-capture REST feature (`visual_capture_*`, `/api/v1/repo/visual-captures`, periodic capture, review-panel screenshot dimensions) is a **separate** live capability with its own `VisualCaptureStorage`; it is NOT part of the baseline subsystem and was intentionally left in place.
- **WorkflowReplayService** (`handlers/workflowreplay/`) is a thin Connect proxy over typed Test Genie run evidence for the Workflows tab. Its `RunsClient` seam wraps `RunsService.{ListRuns,GetRun,ListRunArtifacts}` and selects `workflow.video` across all producer phases. Binary bytes stream through `GET /repo/workflow-runs/{runId}/video?artifact_id=<opaque-id>`; GCT never accepts or exposes a relative artifact path.
- SQLite persistence tests use `api/internal/testutil/db`.
- UI tests use the shared setup and React Query/fetch/viewport helpers.
