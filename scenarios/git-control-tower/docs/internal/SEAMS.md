# Git Control Tower - Architecture Seams

This document describes intentional boundaries ("seams") in git-control-tower. Seams are points where behavior can be substituted without invasive changes, primarily for testing and side-effect isolation.

## Baseline lifecycle seams

`api/internal/baseline.Service` separates durable-run waiting from the short
terminal persistence boundary. `Executor` owns starting, waiting for, and
reading Test Genie runs; `RunsClient` owns owner-scoped pinning; `Storage` owns
atomic intent, manifest, and collection-member writes. Tests use controlled
`Executor` fakes to hold one `AwaitResult` while proving a terminal sibling can
commit. Handler logging is the observability seam: it emits collection
finalizer start, terminal commit, and failure with collection, scenario, and
run identifiers, without logging evidence payloads.

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
- `api/repo_files_connect.go` (`RepoService/SaveFileContent`)
- `ui/src/components/DiffViewer.tsx`

The editing flow is split across a strict backend seam and a UI seam:

- Backend seam:
  - `SaveFileContent` enforces path sanitization (`cleanFilePath`), text-only constraints, size limits, and optimistic concurrency via `expected_hash`.
  - Writes are atomic via `storage.WriteFileAtomic`.
- Conflicts surface as `FileContentConflictError` (typed Connect `Aborted` with the current hash metadata).
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
| Branch-list enrichment | `claimedBranchesFn` in `api/branch_handler.go` | Lazy `newWorktreeInspector().ClaimedBranches` | Test-time override (see `branch_worktree_test.go`) | Lets typed `BranchService.ListBranches` populate `checked_out_in_worktree` without touching git in tests. Errors are intentionally swallowed; empty string is the unclaimed sentinel. |
| CLI client factory | `clientFactory` in `cli/domains/worktree/handlers.go` | `cliapp.NewConnectHTTPClient` + `worktreeconnect.NewWorktreeServiceClient` | Test `fakeClient` (see `handlers_test.go`) | Substitutes the entire WorktreeService client so CLI command flag-plumbing tests need no network or git. |

Compile-time satisfaction: every production and fake impl carries a
`var _ worktree.Inspector = (*X)(nil)` / `var _ worktree.Mutator = ...`
assertion. Renaming a seam method fails the build everywhere it must.

Connect-RPC mount point: `api/connect_wiring.go::mountConnectHandlers`.
The WorktreeService and RepoService handlers register through
`api-core/connectx.RegisterServices`; repository file operations no longer
have parallel REST handlers. Repository settings/remediation operations
(grouping rules, gitignore health/move, and tracked-binary analysis/untracking)
also use RepoService; credentials, remote URL updates, and SSH key operations
use the same typed service, and their former REST handlers are retired.

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
| Connect server interceptor | `api/internal/policygate/interceptor.go::NewInterceptor` | Wraps unary Connect handlers; caller headers are attribution only, while verified principal plus exact intent are required for mutating procedures. | `interceptor_test.go` mounts a `UnimplementedWorktreeServiceHandler` shim and asserts fail-closed behavior. | Defense in depth for direct RPC callers and the canonical enforcement layer. |
| Connect client header stamp | `cli/internal/callerheader/interceptor.go::New` | Stamps attribution headers for diagnostics; it never carries human authority. | `interceptor_test.go` mounts a recording handler. | Preserves caller observability without making a header authoritative. |

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
- **Repair boundary (explicit mutation):** `BaselinesService.RepairBaseline` and `git-control-tower baseline repair` expose only deterministic lifecycle reconciliation. The command is a read-only plan by default; `--apply` is required before unpinning a retained tombstoned manifest, removing it, and appending lifecycle-audit evidence. Missing ready manifests and failed/capturing lifecycle records remain explicit recapture decisions rather than being fabricated by repair.

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
- **EvidenceService** (`handlers/evidence/`) is the sole GCT boundary over typed Test Genie run evidence. Its `RunsClient` seam wraps durable start, canonical run snapshots, captured descriptors, and typed artifact catalogs. Tests filter captured descriptor metadata; Screenshots and Workflows filter open artifact kinds across all producer phases. Binary bytes stream through `GET /repo/test-runs/{runId}/artifacts/{artifactId}?scenario=...`; GCT never accepts or exposes a relative artifact path. `phase_agnostic_guard_test.go` rejects a fixed Test Genie phase registry or producer-phase comparisons in this boundary.
- SQLite persistence tests use `api/internal/testutil/db`.
- UI tests use the shared setup and React Query/fetch/viewport helpers.

## Integration presentation seam

`SettingsTabIntegrations` maps repository-scoped HTTPS credentials and SSH
metadata into the shared `IntegrationCard` while retaining repository and
remote configuration in their dedicated settings surfaces. Credential actions
remain repository-scoped and use the existing hooks; unsupported connection
lifecycle operations are not invented by the UI. Runtime capability health is
rendered separately, so repository credential state cannot be mistaken for a
local dependency state. `SettingsModal.test.tsx` and `hooks-settings.test.tsx`
cover the shared composition and credential mutation paths.

## Outgoing-history safety and isolated recovery

| Seam | Production | Test substitute | Responsibility |
|---|---|---|---|
| `GitRunner.InspectPushSafety` | `ExecGitRunner` delegates to `internal/pushsafety.Inspect` | `FakeGitRunner.SafetyReport` | Push, publish, upstream, and Connect refusal before transfer; snapshot identity. |
| `pushsafety.Commands` | `safetyCommands` in `git_runner_push_safety.go` | Recording command fake | Git process boundary, bounded stdout/stderr, context budget, literal paths, credential environment, disabled replacement refs/hooks and inherited repository/index overrides. |
| `pushsafety.ArtifactStore` | `DiskStore` in `internal/pushsafety/prepare.go` | In-memory failure-injecting store | Exclusive operation reservation, atomic durable JSON records, idempotency and failure evidence outside the source checkout. |
| `GitRunner.PreparePushRecovery` | Isolated bare copy, original bundle and independent restore check, verified replacement trees, repaired bundle | Fake writer and command fault injection | Writes only owner-managed artifacts. Never checks out, resets, stashes, cleans, modifies the source index, or publishes. |
| `GitRunner.GetPushRecovery` | Repository-bound durable artifact read | Fake artifact | Read-only reattachment; an in-progress/interrupted record is not success. |
| UI RPC boundary | Generated RepoService client in `api-push-safety.ts` | Mocked service promises | Consent, unknown/stale/error states, repository changes, reattachment and separation of preparation from publication. |

`git_runner_push_safety_test.go` is a production-adapter contract test. Its real
Git processes run only in `t.TempDir()` repositories and local remotes. It verifies
four rewritten commits and preserved dirty source state. Domain and transport
tests use fakes; no test targets the live Vrooli repository or GitHub.

### Recovery limits and operation semantics

Inspection uses the live single push URL, exact HEAD/base IDs, all outgoing
objects, and NUL-delimited tree paths. GitHub.com has a 100 MiB hard limit and a
50 MiB warning threshold. Other host policy is unknown, not silently inferred.
A missing remote tip object requires a fetch. Multiple push URLs, failed reads,
source drift, or budgets exceeded produce incomplete evidence and prevent push.
The current budget is 100 outgoing commits, 60 seconds, and 64 MiB per command
output. This limit must be disclosed rather than truncating into a passing scan.

Preparation supports linear history descending from an existing live remote tip.
It removes every version of the exact blocked paths in those outgoing snapshots.
Other entries, including executable modes and symlinks, must match byte-for-byte
in Git tree metadata for each commit. Author, committer, timestamps, messages and
encoding survive; signatures are removed and disclosed. No ignore rule or
packaging edit is implicit. The operator must review those before application.

The fingerprint binds repository path, push URL, HEAD, remote base and evidence.
Human intent includes that fingerprint in its subject context. A fresh inspection
must match before artifact writes. Writes are bounded to ten minutes and continue
independently of browser cancellation after admission. An exclusive artifact
reservation prevents duplicate preparation. After a server crash, a `preparing`
record remains explicitly unresolved; inspect retained files before attempting
recovery. Failed records are retained and are not silently overwritten.

Artifacts live under the routed scenario data root at `push-recovery/<fingerprint>/`. Test-mode access uses `filerouting.PickRequired`: lease validation and root selection happen under one lock, so lease expiry cannot redirect an isolated write to production.
They contain committed history only, and may consume several copies of repository
history. No automatic cleanup deletes recovery evidence. `original.bundle` is
restored into an independent bare repository and checked before rewriting.
`repaired.bundle` contains only the candidate branch, so original backup refs are
not accidentally part of its publication. `recovery.json` records scope, mapping,
state and paths. A prepared artifact is never a claim that live uncommitted work
has been backed up or that the current remote still matches.

Use `git-control-tower repo push-safety --json` for inspection,
`repo prepare-push-recovery --fingerprint=<id>` for human-authorized preparation,
and `repo push-recovery-status --fingerprint=<id> --json` for read-only status.
The UI exposes the same boundaries through the Push action. It does not provide
an Apply action: activation, working-tree/index backup, writer coordination and
rollback must be designed and tested together before enabling that separate
mutation. A Git index lock alone cannot stop external agents writing live files.

### Recovery reattachment and integrity checks (2026-09-07)

- `ArtifactStore.Digest` is the streaming content-check seam. `DiskStore` uses
  SHA-256 with cancellation and regular-file checks; tests inject digest faults.
- Preparation restores **both** bundles into independent bare repositories,
  runs `fsck`, checks the candidate reference, and compares bundle digests before
  and after restore verification. The saved digests identify those checked bytes.
- `GetPushRecovery` accepts an empty fingerprint to discover the repository's
  most recently updated retained operation. An exact fingerprint selects an older
  operation. Discovery does not depend on current HEAD or remote authentication.
  Discovery is bounded to 1000 storage entries. Unreadable records produce an
  incomplete-discovery error, not an empty success. No files are deleted.
- Status reads rehash both bundles. Missing, changed, redirected, or unreadable
  bundles return `damaged`. Legacy records without digests and cancelled checks
  return `unverified`. `preparing` remains running-or-interrupted, never success.
  These are response projections; the original durable record is not rewritten.
- After bundle verification, the HTTP service reinspects the recorded push
  destination with repository credentials. Changed evidence returns `stale`;
  unavailable current evidence returns `unverified`. A matching snapshot is only
  current at that check. This does not authorize application or back up live work.
- The dialog always offers status discovery, including after an inspection error.
  It displays an operation ID for exact reattachment. A changed preview does not
  hide retained evidence, and old evidence does not authorize a new preparation.

`repo push-recovery-status --json` discovers the most recently updated operation;
`repo push-recovery-status --fingerprint=<operation-id> --json` reads an exact one.
The same semantics apply to `GetPushRecoveryRequest.fingerprint` without changing
its wire shape. No Apply, rollback, or publication mutation was added.

### Isolated HTTP and browser validation

`push_recovery_http_test.go`, called from the four-commit adapter fixture, runs
an ephemeral HTTP server with production handlers, single-use intent logic,
real Git preparation, an in-memory repository registry, and leased temporary file
storage. Authentication and remote inspection are controlled seams. It checks
refusal, replay, discovery, remote movement, unavailable evidence, and corruption.
The outer fixture compares source refs, index bytes, local deletions, symlinks,
staged/unstaged content, untracked content, and ignored content after the run.

For the optional real-browser exercise, create a new empty `/tmp` directory.
From `ui`, run `node scripts/build-push-recovery-fixture.mjs <directory>`.
From `api`, run `GCT_RECOVERY_BROWSER_FIXTURE=<directory> go test ./ -run
TestRecoveryAdapterPreservesLiveWorkspace -count=1 -v` (on one shell line).
The browser fixture imports the production dialog and RPC module. Only its
transport origin and authentication exchange are adapted to the test server.
BAS opens that ephemeral server. The fixture drives the real dialog through
consent and preparation, closes and reopens it after simulated remote movement,
and checks the retained stale operation. Capture waits for a success marker; the
HTTP test also requires the completion callback before it accepts browser evidence. Normal unit runs do not launch BAS. The fixture is not served by the
production application and cannot publish a branch.

### Proactive file-size indicators

The workspace shares a read-only `InspectPushSafety` report across desktop and
mobile staging, commit, history and sync controls. The index advisory fields are
excluded from recovery fingerprints. Index inspection reads `diff --cached --raw
-z` and batch object sizes, then compares the index diff again; it never reads
working-file sizes or writes the index. Unknown policy, command failures, missing
objects, merge conflicts and index movement do not produce a successful check.

Indicators are snapshot evidence, not permission to push. Observed status/history
changes immediately invalidate displayed claims, with a one-second debounce for
reinspection. A report expires after 60 seconds (expiry checked every five
seconds); background refreshes and failures suppress prior claims. Query keys
include repository identity and superseded requests are canceled. Undetected
external changes within that observation window remain possible; the push dialog
and server recheck before transfer. This is not a real-time writer lock.

Staged rows show warnings for staged Git objects, including the distinction from
working copies and LFS pointers. Local commits remain available under existing
human authorization. History identifies introducing commits, descendants in
verified linear history (including later deletions), and generic blockers for
merge/diverged history without claiming an ancestor relationship. History outside
the inspected set is not certified. Clicking a badge or blocked-push notice opens
read-only review. Preparing artifacts never clears the blocker or marks recovery
applied. Activation and rollback remain unavailable.

Regression seams: `internal/pushsafety/staged_test.go`,
`TestStagedSafetyReadsIndexAndPreservesWorkspace`, and
`ui/src/components/PushSafetyIndicators.test.tsx` cover staged-object semantics,
policy boundaries, incomplete snapshots, preservation, history attribution,
repository switching, stale/failing checks, and committability without mutation.

History mode keeps recovery review reachable through both navbar layouts.
Commit Files warnings mean a path is associated with an oversized outgoing blob;
they do not assert that the selected tree still contains that version. Paths
unchanged in the selected commit appear in a separate outgoing-blocker notice.
This preserves correct meaning for deletions and renames without upgrading the
report's separate path/commit sets into nonexistent pairwise evidence.
