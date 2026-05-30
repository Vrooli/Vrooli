# test-genie seam registry

Seams declared in `scenarios/test-genie/api/internal/`. Drift is gated by
`// seam:` tags; if you add a new tag, add a row here. The reconciliation is
enforced automatically by `internal/seamregistry/seam_registry_test.go`, which
walks every `// seam:` tag in the codebase and fails the build if its name is
not documented below.

## Runnability gate

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `Resolver` | `orchestrator/runnability/runnability.go` (`Resolver` interface) | `runnability.StandardResolver` (`orchestrator/runnability/resolver.go`) | `mocks.FakeResolver` (`orchestrator/runnability/mocks/resolver.go`) | The single per-phase RUN/RUN_DEGRADED/SKIP decision: `PhaseCapabilities × RunContext → Verdict`. Absorbs the old `runtimeNeeds` switch, the `EnsureRunning` self-clobber, the playbooks routed-vs-fallback `PathDecision`, and the requirements skip vocabulary. Pure; exhaustively unit-tested. |

## Playbooks phase

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `routingChecker` | `phases/phase_playbooks.go` (`eligibilityChecker` interface) | `*eligibility.Checker` | tests reassign the package var to a stub satisfying the interface | Routes the routed-vs-fallback decision; tests inject canned `Eligibility`. |
| `resolveScenarioRoutingClient` | `phases/phase_playbooks.go` | `routing_v1connect.NewRoutingServiceClient` against the scenario's discovered base URL | tests override to return a stub `RoutingServiceClient` | Connect-RPC client for the scenario's `RoutingService`. |
| `probeRoutingServiceEnabled` | `phases/phase_playbooks.go` | HTTP GET against the install procedure URL; 404 → `errRoutingServiceDisabled` | tests override to return `nil` or `errRoutingServiceDisabled` directly | Detects production-mode targets (route unmounted → fallback). |
| `extractTestDSN` | `phases/phase_playbooks.go` | strict per-driver DSN selection | unit-tested via `extractTestDSN_test.go` | Refuses to guess when `primaryDriver` is empty or the env doesn't carry the matching DSN. |
| `isolationManagerFactory` | `phases/phase_playbooks.go` | `isolation.NewManager` | tests inject a `fakeIsolation` | Stub isolation without requiring Docker. |

## Eligibility

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `eligibility.ResolveBaseURL` | `eligibility/auditor.go` | `discovery.ResolveScenarioURLDefault(ctx, "scenario-auditor")` | tests override the package var | scenario-auditor discovery indirection. |
| `eligibility.HTTPClient` | `eligibility/auditor.go` | `&http.Client{Timeout: 10s}` | tests can override | HTTP client used by auditor calls. |
| `eligibility.FetchRegisteredRules` | `eligibility/path_decision.go` | `GET /api/v1/rules` via `RequestJSON` | tests override to return a canned `map[string]struct{}` | Used by `AssertRulesObserved` to verify the routing rules are registered + enabled in the auditor. |
| `eligibility.Checker.Invalidate` | `eligibility/router.go` | per-scenario cache eviction | unit-tested in `path_decision_test.go` | Called from the playbooks defer so a successive run in the same test-genie process picks up code fixes. |

## Run history & diagnostics

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| Run index | `internal/shared/runs/index.go` (`Index`) | Append-only `coverage/runs.index.json`, `flock`-protected, atomic temp-file rename; addressed per-scenario by `NewIndex(scenarioDir)`. | Exercised directly against a `t.TempDir()` in `index_test.go` (incl. 50-goroutine concurrent-append torture) — concrete store, not faked. | Stops the old overwrite-per-run model; gives every run a stable runID key so GCT baselines can pin/compare (Decision 3). |
| Retention GC | `internal/shared/runs/retention.go` (`GC`, `DefaultRetentionPolicy`) | Keeps last N + everything pinned; oldest-unpinned evicted first; runs in background on execute completion. | `retention_test.go` against `t.TempDir()`. | Bounds disk while never dropping a run an external consumer pinned (`len(Pins) > 0`). |
| Diagnostics → BAS plumbing | `internal/playbooks/config/config.go` (`DiagnosticsConfig`) → `internal/playbooks/execution/client.go` | `--diagnostics-preset {none,light,full}` sets `ExecuteWorkflowOptions` (`RequiresVideo/Har/Trace`) + `ArtifactCollectionConfig` toggles. | `config_test.go` (preset parsing/defaults). | Per-run diagnostics depth is immutable once a run completes (Decision 4); baselines just pin runs of differing depth. |
| RunsService RPC | `internal/app/runs/service.go` | Thin Connect-RPC wrapper over `internal/shared/runs.Index`; `scenariosRoot` resolves a slug → run index path. | `service_test.go` against `t.TempDir()`. | The HTTP surface the test-genie CLI and GCT baseline adapters consume. |
| Run artifact enumeration/serving | `internal/shared/artifacts/run_artifacts.go` (`ListRunVideos`, `ResolveRunArtifact`); `RunsService.ListRunVideos` (Connect) + REST `GET /scenarios/{name}/runs/{runId}/artifact?path=` (`httpserver/run_artifact_handler.go`) | `ListRunVideos` walks `automation/<wf>/video/*.webm`; `ResolveRunArtifact` traversal-guards a run-relative path to `RunDir`. Structured listing is Connect; binary bytes stream over REST (range via `http.ServeFile`). | `run_artifacts_test.go` (traversal + enumeration) against `t.TempDir()`. | Lets GCT's WorkflowReplayService list + proxy playbooks-run videos without reaching into test-genie's filesystem. |

## Database handle

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `Executor` | `internal/dbexec/dbexec.go` | `*database.RoutedDB` (wired in `app/runtime/bootstrap.go`) | `*sql.DB` fixture from `internal/testsqlite` | The narrow DB-execution seam every repository and the schema applier capture instead of a raw `*sql.DB`. Lets production route per-request to an installed test pool (the routed test-DB path) while tests keep plain `*sql.DB`; keeps test-genie clear of the `routed_database_handle_capture` rule and thus routed-eligible. |

## DB detection (playbooks)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `Manifest` | `internal/playbooks/dbdetect/manifest.go` | reads the scenario's `.vrooli/service.json` | tests inject a canned manifest | Boundary between dbdetect and the on-disk service.json so driver detection is testable without fixtures on disk. |
| `Filesystem` | `internal/playbooks/dbdetect/fs.go` | read-only file inspection of the scenario tree | tests inject an in-memory FS | Boundary between dbdetect's file scanning and the real filesystem. |
| `Collector` | `internal/playbooks/dbdetect/types.go` | concrete scanning of scenario inputs | tests inject canned collected inputs | Boundary between raw scanning and the driver-ranking decision. |

## Playbooks claims

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `Repository` | `internal/playbooksclaims/repository.go` | `NewSqliteRepository` | tests use an in-memory/`t.TempDir()` repo | Persists the per-scenario playbooks claim that guards against concurrent runs. |
| `Clock` | `internal/playbooksclaims/service.go` | wall clock | tests inject a fixed clock | Drives claim TTL/heartbeat expiry deterministically in tests. |

## Related docs

- [`docs/agent-system/routed-test-db.md`](../../../../docs/agent-system/routed-test-db.md)
  — End-to-end routed-vs-fallback path documentation.
- [`packages/api-core/docs/internal/SEAMS.md`](../../../../packages/api-core/docs/internal/SEAMS.md)
  — Substrate-level seams (`RoutedDB`, `Clock`, `TestModeMiddleware`,
  `devrouting.Register`).
