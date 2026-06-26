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

## Phase execution

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `StandardRunResult` | `orchestrator/phases/executor.go` (`StandardRunResult` interface) | Native runner result types: `shared.RunResult[T]` aliases plus `business.RunResult` | `RunNativePhase` unit tests use lightweight fake implementations | The native phase result contract. Structure, performance, integration, and business phases expose the same success/error/failure/remediation/observations/summary shape, so report assembly and phase-pointer writing live once in `RunNativePhase`. |
| `DelegatedClient` | `orchestrator/phases/phase_validationprovider.go` (`DelegatedClient` type) | Default client calls `ScenarioValidationService`; standards uses `standardsDelegatedClient` for scenario-auditor's standards API | Standards phase tests override the scenario-auditor base URL and execute through the catalog runner | Keeps every delegated phase on the same provider result, finding, summary, and pointer-writing path while allowing a provider-specific client only when the provider does not expose `ScenarioValidationService`. |
| `MergePresets` | `orchestrator/phases/presets.go` (`MergePresets`) | Orchestrator `loadPresets` supplies file/testing.json/default preset maps and the available-phase set | `TestMergePresetsPrecedenceAndFiltering` pins replacement, deletion, filtering, and default fill behavior | Keeps preset precedence beside the catalog-owned preset declarations so selection does not carry a second preset policy. |

## UI smoke (BAS workflow capture)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `WorkflowClient` | `browsercapture/browsercapture.go` (`WorkflowClient` interface) | `browsercapture.NewLiveClient` wrapping `execution.NewClientWithConfig` (Connect-RPC over BAS `apiconnect`) | `browsercapture.FakeWorkflowClient` (`browsercapture/fake.go`) returning a canned `execution.ParsedTimeline` | The BAS workflow-engine capture seam. Smoke runs its host-iframe + handshake-gate + screenshot workflow on the same BAS client playbooks uses — no second BAS client. Tests exercise the capture/verdict mapping without a live engine. |
| `FakeWorkflowClient` | `browsercapture/fake.go` | (test-only) | itself | Test wiring for `WorkflowClient`; records the executed workflow definition and returns a canned timeline / errors. |
| `CaptureClient` | `browsercapture/capture.go` (`CaptureClient` interface) | `browsercapture.NewLiveCaptureClient` (BAS `CaptureService` sharing the workflow client's connection via `execution.HTTPClient.CaptureServiceClient`) | `browsercapture.FakeCaptureClient` (`browsercapture/capture_fake.go`) returning canned `CaptureResponse`s | The BAS single-location capture seam for the all-pages visual capture (one `Capture` per discovered page). Reuses the ONE BAS connection — no second client. Plain multi-page screenshots only (no host-iframe handshake — that stays on `WorkflowClient`). |
| `FakeCaptureClient` | `browsercapture/capture_fake.go` | (test-only) | itself | Test wiring for `CaptureClient`; records every request and returns canned/per-URL responses, so the all-pages loop + cost guard run without a live engine. |
| `FileReader` (page discovery) | `pagediscovery/pagediscovery.go` (`FileReader` interface) | `pagediscovery.OSFileReader` (`os.ReadFile` of `.vrooli/lighthouse.json`) | `pagediscovery.FakeFileReader` (`pagediscovery/fake.go`) mapping path→bytes | The page-discovery filesystem seam. test-genie's single source of truth for the scenario page set (`lighthouse`/`fallback`/`explicit`), consumed by the smoke all-pages capture. Tests supply page configs without touching disk. |
| `FakeFileReader` | `pagediscovery/fake.go` | (test-only) | itself | Test wiring for the page-discovery `FileReader` seam. |
| timeline → evidence mapping | `playbooks/execution/evidence_adapter.go` (`ConsoleEntries`, `NetworkFailures`, `ToEvidence`) | pure functions over `execution.ParsedTimeline` → `evidence.Evidence` | unit-tested in `evidence_adapter_test.go` (canned timelines + golden happy fixture) | The single home for reading console/network/page-error/screenshot observations off a BAS timeline. Both browser phases consume it — smoke via `browsercapture/timeline.go`, playbooks via `playbooks/evidence_findings.go` — so the console/network extraction rules cannot drift between phases. The verdict rules themselves live once in `internal/evidence`. |
| Visual-health provider | ui-health `VisualHealthService` (`AnalyzeArtifacts`, `CompareArtifacts`, `ListRules`) | ui-health owns generic visual judgment: pixel blankness, DOM blankness, and neutral visual deltas. Test Genie supplies run artifact handles/bytes only. | ui-health `internal/visualhealth` tests cover synthetic screenshots/DOM; Test Genie `compare_visuals_test.go` injects a fake ui-health comparer to verify delegation. | **Observability:** ui-health is the single visual-health authority. Test Genie records and enumerates run artifacts; BAS captures browser artifacts; neither reimplements visual judgment. Thresholds are the `UI_HEALTH_VISUAL_*` levers (see ui-health configuration/reference). |
| Visual comparison RPC | `internal/app/runs/service.go` (`CompareRunVisuals`) delegating to ui-health | Per-page comparison of two runs' captures → neutral deltas (`identical`/`changed`/`added`/`removed` + changed fraction) returned by ui-health. | `compare_visuals_test.go` against `t.TempDir()` with a fake ui-health comparer. | **Observability:** this is a Test Genie run convenience endpoint only. git-control-tower can consume the deltas, but ui-health owns the comparison math and thresholds. |

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

## Durable run lifecycle (server-owned runs)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| Run manager | `internal/runmanager/manager.go` (`Manager`, `ActiveRun`) | In-memory registry of in-flight runs keyed by runID, each holding a cancel func + event broadcaster + live phase state, backed by `shared/runs.Index` for cross-restart durability. Execution runs under a **server-lifetime** context, not the request, so a client disconnect never cancels the suite. | `manager_test.go` against `t.TempDir()` (disconnect-does-not-cancel, multi-follower fan-out, abort→aborted, startup sweep idempotency). | The decouple keystone: makes a run survive client cancellation and re-attachable by one run-id (`StartRun`/`FollowRun`/`WaitRun`/`AbortRun`/`GetRunStatus`). |
| Event broadcaster | `internal/runmanager/broadcaster.go` | Fans `RunEvent`s to N subscribers and buffers recent history so a late/re-attaching follower replays phase progress from the start. | `manager_test.go` (two followers of one run both receive the full sequence). | Lets `FollowRun` be re-attachable and replayable instead of a one-shot live tap. |
| Run lifecycle RPC | `internal/app/runs/lifecycle.go` (`StartRun`/`FollowRun`/`WaitRun`/`AbortRun`/`GetRunStatus`) | Connect-RPC handlers over the run manager; `GetRunStatus` derives remaining-ETA from the plan preview and clamps `recommended_next_check_seconds`. | `lifecycle_test.go` against `t.TempDir()`. | The durable-run control surface the CLI default (`execute`) and the root `vrooli scenario test wait/status/follow/abort` proxy consume. |

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

- [`scenarios/storage-health/docs/concepts/test-isolation-contract.md`](../../../../scenarios/storage-health/docs/concepts/test-isolation-contract.md)
  — The canonical test-isolation contract: the routed path, the four-seam cookbook, and the fail-closed gate.
- [`packages/api-core/docs/internal/SEAMS.md`](../../../../packages/api-core/docs/internal/SEAMS.md)
  — Substrate-level seams (`RoutedDB`, `Clock`, `TestModeMiddleware`,
  `devrouting.Register`).
