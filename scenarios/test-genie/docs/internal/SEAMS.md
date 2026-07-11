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
| `DelegatedClient` | `orchestrator/phases/phase_validationprovider.go` (`DelegatedClient` type) | Default client calls `ScenarioValidationService` | Validation-provider tests inject fake clients through catalog specs | Keeps delegated phases on the same provider result, finding, summary, and pointer-writing path while preserving a narrow test seam. |
| `MergePresets` | `orchestrator/phases/presets.go` (`MergePresets`) | Orchestrator `loadPresets` supplies file/testing.json/default preset maps and the available-phase set | `TestMergePresetsPrecedenceAndFiltering` pins replacement, deletion, filtering, and default fill behavior | Keeps preset precedence beside the catalog-owned preset declarations so selection does not carry a second preset policy. |

## UI smoke (BAS workflow capture)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `WorkflowClient` | `browsercapture/browsercapture.go` (`WorkflowClient` interface) | `browsercapture.NewLiveClient` wrapping `execution.NewClientWithConfig` (Connect-RPC over BAS `apiconnect`) | `browsercapture.FakeWorkflowClient` (`browsercapture/fake.go`) returning a canned `execution.ParsedTimeline` | The BAS workflow-engine capture seam for UI-health runtime capture. Tests exercise the capture/verdict mapping without a live engine. |
| `FakeWorkflowClient` | `browsercapture/fake.go` | (test-only) | itself | Test wiring for `WorkflowClient`; records the executed workflow definition and returns a canned timeline / errors. |
| `CaptureClient` | `browsercapture/capture.go` (`CaptureClient` interface) | `browsercapture.NewLiveCaptureClient` (BAS `CaptureService` sharing the workflow client's connection via `execution.HTTPClient.CaptureServiceClient`) | `browsercapture.FakeCaptureClient` (`browsercapture/capture_fake.go`) returning canned `CaptureResponse`s | The BAS single-location capture seam for the all-pages visual capture (one `Capture` per discovered page). Reuses the ONE BAS connection — no second client. Plain multi-page screenshots only (no host-iframe handshake — that stays on `WorkflowClient`). |
| `FakeCaptureClient` | `browsercapture/capture_fake.go` | (test-only) | itself | Test wiring for `CaptureClient`; records every request and returns canned/per-URL responses, so the all-pages loop + cost guard run without a live engine. |
| `FileReader` (page discovery) | `pagediscovery/pagediscovery.go` (`FileReader` interface) | `pagediscovery.OSFileReader` (`os.ReadFile` of `.vrooli/lighthouse.json`) | `pagediscovery.FakeFileReader` (`pagediscovery/fake.go`) mapping path→bytes | The page-discovery filesystem seam. test-genie's single source of truth for the scenario page set (`lighthouse`/`fallback`/`explicit`), consumed by the smoke all-pages capture. Tests supply page configs without touching disk. |
| `FakeFileReader` | `pagediscovery/fake.go` | (test-only) | itself | Test wiring for the page-discovery `FileReader` seam. |
| timeline → evidence mapping | `playbooks/execution/evidence_adapter.go` (`ConsoleEntries`, `NetworkFailures`, `ToEvidence`) | pure functions over `execution.ParsedTimeline` → `evidence.Evidence` | unit-tested in `evidence_adapter_test.go` (canned timelines + golden happy fixture) | Compatibility adapter for historical BAS timeline artifacts. New workflow phase findings come from workflow-health; this package remains for seed/artifact compatibility and shared diagnostics. |
| Visual-health provider | ui-health `VisualHealthService` (`AnalyzeArtifacts`, `CompareArtifacts`, `ListRules`) | ui-health owns generic visual judgment: pixel blankness, DOM blankness, and neutral visual deltas. Test Genie supplies run artifact handles/bytes only. | ui-health `internal/visualhealth` tests cover synthetic screenshots/DOM; Test Genie `compare_visuals_test.go` injects a fake ui-health comparer to verify delegation. | **Observability:** ui-health is the single visual-health authority. Test Genie records and enumerates run artifacts; BAS captures browser artifacts; neither reimplements visual judgment. Thresholds are the `UI_HEALTH_VISUAL_*` levers (see ui-health configuration/reference). |
| Visual comparison RPC | `internal/app/runs/service.go` (`CompareRunVisuals`) delegating to ui-health | Per-page comparison of two runs' captures → neutral deltas (`identical`/`changed`/`added`/`removed` + changed fraction) returned by ui-health. | `compare_visuals_test.go` against `t.TempDir()` with a fake ui-health comparer. | **Observability:** this is a Test Genie run convenience endpoint only. git-control-tower can consume the deltas, but ui-health owns the comparison math and thresholds. |

## Workflow seed compatibility

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `isolationManagerFactory` | `phases/playbooks_seed_cycle.go` | `isolation.NewManager` | tests inject a `fakeIsolation` | Keeps legacy playbooks seed apply/cleanup testable while workflow execution is delegated to workflow-health. |

## Eligibility

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `eligibility.FetchRegisteredRules` | `eligibility/path_decision.go` | `GET /api/v1/rules` via `RequestJSON` | tests override to return a canned `map[string]struct{}` | Used by `AssertRulesObserved` to verify the routing rules are registered + enabled in the auditor. |
| `eligibility.Checker.Invalidate` | `eligibility/router.go` | per-scenario cache eviction | unit-tested in `path_decision_test.go` | Lets routing eligibility tests and handlers pick up code fixes in the same test-genie process. |

## Run history & diagnostics

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| Run index + terminal snapshot | `internal/shared/runs/index.go` (`Index`, `TerminalSnapshot`) | Compact `coverage/runs.index.json` plus one schema-versioned atomic terminal snapshot per run. Terminal wait, show, restart hydration, history, and downstream projections share that snapshot; legacy/corrupt records carry explicit degradation. | `index_test.go`, run-manager restart/finalization tests, and RunsService parity tests use `t.TempDir()` storage. | Keeps enumeration compact while preserving complete terminal phase status/duration and one canonical durable truth after in-memory retirement. |
| Retention GC | `internal/shared/runs/retention.go` (`GC`, `DefaultRetentionPolicy`) | Keeps last N + everything pinned; oldest-unpinned evicted first; runs in background on execute completion. | `retention_test.go` against `t.TempDir()`. | Bounds disk while never dropping a run an external consumer pinned (`len(Pins) > 0`). |
| Diagnostics → BAS plumbing | `internal/playbooks/config/config.go` (`DiagnosticsConfig`) → `internal/playbooks/execution/client.go` | `--diagnostics-preset {none,light,full}` sets `ExecuteWorkflowOptions` (`RequiresVideo/Har/Trace`) + `ArtifactCollectionConfig` toggles. | `config_test.go` (preset parsing/defaults). | Per-run diagnostics depth is immutable once a run completes (Decision 4); baselines just pin runs of differing depth. |
| RunsService RPC | `internal/app/runs/service.go` | Thin Connect-RPC wrapper over `internal/shared/runs.Index`; `scenariosRoot` resolves a slug → run index path. | `service_test.go` against `t.TempDir()`. | The HTTP surface the test-genie CLI and GCT baseline adapters consume. |
| Run evidence catalog + opaque serving | `internal/shared/artifacts/catalog.go`; `RunsService.ListRunArtifacts` / `GetRunArtifact`; REST `GET /scenarios/{name}/runs/{runId}/artifacts/{artifactId}` in `internal/app/httpserver/run_artifact_handler.go` | Atomically persists a digest-verified, run-salted catalog of open artifact kinds. Metadata is path-free; byte access resolves only a run-scoped opaque ID and enforces containment, symlink rejection, content type, no-sniff, and active-content sandbox headers. Read-only legacy projections derive their timestamp from immutable artifact metadata, so unchanged historical bytes produce a stable digest. `ListRunVideos` remains compatibility-only and is not the GCT boundary. | Catalog, RunsService, and HTTP tests cover runtime registration, legacy discovery, unknown kinds, traversal, cross-run IDs, symlinks, missing bytes, and active content. `TestCopiedRunEvidenceRehearsal` optionally copies a real scenario's run index/artifacts/logs into `t.TempDir()`, repeats every terminal projection, resolves every opaque artifact, and proves the copied byte tree is unchanged. | Lets any consumer select stable evidence kinds across all producer phases without learning run filesystem layout or introducing a phase registry. |

## Durable run lifecycle (server-owned runs)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| Run manager | `internal/runmanager/manager.go` (`Manager`, `ActiveRun`) | In-memory registry of in-flight runs keyed by runID, backed by the durable index, terminal snapshot, descriptor snapshot, and evidence catalog. Execution runs under a **server-lifetime** context, not the request, so a client disconnect never cancels the suite; startup hydrates terminal truth and resumes only lifecycle repair work. | `manager_test.go` against `t.TempDir()` covers disconnect, multi-follower fan-out, abort/failure, terminal restart hydration, legacy/corrupt degradation, and idempotent finalization. | Makes a run survive client cancellation and keeps terminal evidence available by one run ID after retirement or restart. |
| Event broadcaster | `internal/runmanager/broadcaster.go` | Fans `RunEvent`s to N subscribers and buffers recent history so a late/re-attaching follower replays phase progress from the start. | `manager_test.go` (two followers of one run both receive the full sequence). | Lets `FollowRun` be re-attachable and replayable instead of a one-shot live tap. |
| Run lifecycle RPC | `internal/app/runs/lifecycle.go` (`StartRun`/`FollowRun`/`WaitRun`/`AbortRun`/`GetRunStatus`) | Connect-RPC handlers over the run manager. Queued/running responses use live state; terminal `WaitRun` and `GetRun` use the same persisted snapshot projector, including the captured descriptor catalog and degraded metadata. | `lifecycle_test.go`, `service_test.go`, and CLI wait/show parity tests. | The owns-the-run control surface consumed by `vrooli scenario test` and `test-genie runs`; one blocking wait detaches safely and never determines execution outcome. |

Before any live historical-data migration, run the copy-first rehearsal from
`scenarios/test-genie/api`:

```text
TEST_GENIE_RUN_REHEARSAL_SOURCE=/path/to/scenarios/<scenario> \
  go test ./internal/app/runs -run TestCopiedRunEvidenceRehearsal -count=1 -v
```

The source is only copied. Production readers operate on the temporary copy,
and missing historical snapshots/catalogs remain explicit degraded/legacy
evidence rather than being backfilled or presented as canonical.

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
