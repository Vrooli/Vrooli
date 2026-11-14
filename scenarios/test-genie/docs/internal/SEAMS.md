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
| Evidence retention lifecycle | `internal/shared/runs/retention.go` (`RetentionService`, `DetailStore`) | Post-SQLite-persistence collector applies count/age/size policy to complete run artifacts, logs, and compact execution rows as one tombstoned operation. Active owner-scoped `PinLease`s protect a run until expiry. | `retention_test.go` against `t.TempDir()` plus execution repository deletion tests. | Bounds durable evidence without stale index-pin bypasses; an interrupted filesystem/database deletion converges on the next collection. |
| Compact execution timing history | `internal/execution/phase_history.go`; `internal/execution/stage_history.go` | Transactional `suite_executions` header + `suite_execution_phases` rows for validation phases and sibling `suite_execution_stages` rows for preparation spans. Rich result data remains immutable run evidence. | `execution/repository_test.go` and aggregation tests against temporary SQLite. | Gives self-health and planning indexable bounded phase input while retaining provider-readiness and orchestration timing without misclassifying those spans as phases. |
| Post-listener advisory work | `internal/app/runtime/background.go` (`BackgroundCoordinator`) | The HTTP server invokes one once-only coordinator after binding its listener; self-health, the opt-in fleet scheduler, and Agent Manager initialization share its cancellation context. | `background_test.go`, health handler held-pool test, and self-health timeout observer test. | Keeps lifecycle serving independent of optional analytics and ensures shutdown cancels every advisory job. |
| Diagnostics → BAS plumbing | `internal/orchestrator/testconfig/config/config.go` (`DiagnosticsConfig`) → `internal/basprobe/execution/client.go` | `--diagnostics-preset {none,light,full}` sets `ExecuteWorkflowOptions` (`RequiresVideo/Har/Trace`) + `ArtifactCollectionConfig` toggles. | `config_test.go` (preset parsing/defaults). | Per-run diagnostics depth is immutable once a run completes (Decision 4); baselines just pin runs of differing depth. |
| RunsService RPC | `internal/app/runs/service.go` | Thin Connect-RPC wrapper over `internal/shared/runs.Index`; `scenariosRoot` resolves a slug → run index path. | `service_test.go` against `t.TempDir()`. | The HTTP surface the test-genie CLI and GCT baseline adapters consume. |
| Run evidence catalog + opaque serving | `internal/shared/artifacts/catalog.go`; `RunsService.ListRunArtifacts` / `GetRunArtifact`; REST `GET /scenarios/{name}/runs/{runId}/artifacts/{artifactId}` in `internal/app/httpserver/run_artifact_handler.go` | Atomically persists a digest-verified, run-salted catalog of open artifact kinds. Metadata is path-free; byte access resolves only a run-scoped opaque ID and enforces containment, symlink rejection, content type, no-sniff, and active-content sandbox headers. Read-only legacy projections derive their timestamp from immutable artifact metadata, so unchanged historical bytes produce a stable digest. `ListRunVideos` remains compatibility-only and is not the GCT boundary. | Catalog, RunsService, and HTTP tests cover runtime registration, legacy discovery, unknown kinds, traversal, cross-run IDs, symlinks, missing bytes, and active content. `TestCopiedRunEvidenceRehearsal` optionally copies a real scenario's run index/artifacts/logs into `t.TempDir()`, repeats every terminal projection, resolves every opaque artifact, and proves the copied byte tree is unchanged. | Lets any consumer select stable evidence kinds across all producer phases without learning run filesystem layout or introducing a phase registry. |

## Durable run lifecycle (server-owned runs)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| Run manager | `internal/runmanager/manager.go` (`Manager`, `ActiveRun`) | In-memory registry of in-flight runs keyed by runID, backed by the durable index, terminal snapshot, descriptor snapshot, and evidence catalog. Execution runs under a **server-lifetime** context, not the request, so a client disconnect never cancels the suite; startup hydrates terminal truth and resumes only lifecycle repair work. Before the first test phase, orchestrator progress events expose `preparing:planning`, `preparing:provider_readiness`, and `preparing:target_runtime`; those remain lifecycle `preparing` with a current stage and heartbeat, rather than appearing idle. A nonterminal index record without a live owner is terminalized as aborted on startup, `status`, `wait`, or `abort`; callers never receive an unprogressable running state. A preflight executor error is persisted as the terminal result's `failureReason` and returned by terminal wait status, even when no phase could start. | `manager_test.go` against `t.TempDir()` covers disconnect, multi-follower fan-out, abort/failure, terminal restart hydration, orphan recovery through status/wait/abort, preparation-stage visibility, preflight-error persistence, legacy/corrupt degradation, and idempotent finalization. | Makes a run survive client cancellation, distinguishes active preflight from an apparently stuck run, and keeps terminal evidence available by one run ID after retirement or restart. |
| Event broadcaster | `internal/runmanager/broadcaster.go` | Fans compact `RunEvent`s to N subscribers and retains only a byte-and-event-bounded diagnostic tail. It never retains terminal result pointers; durable evidence owns terminal detail. | `broadcaster_test.go`, `manager_test.go`. | Lets `FollowRun` reattach safely without letting followers create unbounded run memory. |
| Execution evidence writer | `internal/executionevidence` (`Writer`, `Manifest`) | Writes size-limited immutable artifacts, digest-backed references, and the versioned run manifest. | `writer_test.go`; orchestrator findings-artifact test. | Establishes one canonical owner for detailed findings and rich evidence; projections retain only counts and references. |
| Run lifecycle RPC | `internal/app/runs/lifecycle.go` (`StartRun`/`FollowRun`/`WaitRun`/`AbortRun`/`GetRunStatus`) | Connect-RPC handlers over the run manager. Queued/running responses use live state; terminal `WaitRun` and `GetRun` use the same persisted snapshot projector, including the captured descriptor catalog and degraded metadata. | `lifecycle_test.go`, `service_test.go`, and CLI wait/show parity tests. | The owns-the-run control surface consumed by `vrooli scenario test` and `test-genie runs`; one blocking wait detaches safely and never determines execution outcome. |

For agent-initiated runs, the manager carries only the middleware-verified agent
credential pair in memory into the server-owned context. Downstream owners
reverify forwarded credentials when called, so expiry or revocation still
applies to queued and running work. Program Runtime's private agent bridge
currently carries the baseline identity, not supplemental operation proofs.
In-flight coalescing also
requires the same verified credential pair; absent or invalid credentials do
not coalesce with a verified caller. Credentials are not stored in run evidence,
and they are not recovered after a server restart. This does not provide a
human operator credential or authorize operations on its own.

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
| Lifecycle health connection | `internal/app/runtime/bootstrap.go` (`openHealthDatabase`) → `internal/app/httpserver/health_handlers.go` | Dedicated one-connection SQLite handle running bounded `PRAGMA schema_version`; normal work remains on the routed runtime pool | HTTP handler tests use the primary fallback; runtime tests open a temporary independent SQLite handle | Prevents advisory or normal runtime-pool contention from making lifecycle health queue behind Test Genie’s one runtime connection. |

## Background coordination

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| Serving-gated advisory startup | `runtime.Bootstrapped.StartBackground` → `httpserver.Dependencies.StartBackground` | HTTP server invokes it only after binding the listener, with a cancellable process context | App wiring tests provide a nil/no-op callback; sweeper tests use temporary repositories | Keeps self-health analytics outside dependency construction and prevents startup starvation of lifecycle health. |
| Per-sweep deadline | `selfhealthsnapshots.SweeperConfig.RunTimeout` | `RunLoop` creates one deadline context for each advisory build/persist attempt | `TestSweeperRunLoopBoundsAdvisoryBuild` blocks a fake builder until deadline | A stuck aggregation must be logged/deferred, never hold an advisory task indefinitely. |

## DB detection (playbooks)

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `Manifest` | `internal/orchestrator/phases/dbdetect/manifest.go` | reads the scenario's `.vrooli/service.json` | tests inject a canned manifest | Boundary between dbdetect and the on-disk service.json so driver detection is testable without fixtures on disk. |
| `Filesystem` | `internal/orchestrator/phases/dbdetect/fs.go` | read-only file inspection of the scenario tree | tests inject an in-memory FS | Boundary between dbdetect's file scanning and the real filesystem. |
| `Collector` | `internal/orchestrator/phases/dbdetect/types.go` | concrete scanning of scenario inputs | tests inject canned collected inputs | Boundary between raw scanning and the driver-ranking decision. |

## Playbooks claims

| Seam | Declaration | Production impl | Test double | Why |
|---|---|---|---|---|
| `Repository` | `internal/playbooksclaims/repository.go` | `NewSqliteRepository` | tests use an in-memory/`t.TempDir()` repo | Persists the per-scenario playbooks claim that guards against concurrent runs. |
| `Clock` | `internal/playbooksclaims/service.go` | wall clock | tests inject a fixed clock | Drives claim TTL/heartbeat expiry deterministically in tests. |

## Related docs

- [`docs/agent-system/routed-test-db.md`](../../../../docs/agent-system/routed-test-db.md)
  — The canonical two-engine routed-storage contract and fail-closed gate.
- [`packages/api-core/docs/internal/SEAMS.md`](../../../../packages/api-core/docs/internal/SEAMS.md)
  — Substrate-level seams (`RoutedDB`, `Clock`, `TestModeMiddleware`,
  `devrouting.Register`).


## Suite replacement and lifecycle work

Test Genie supplies the shared API Core lifecycle provider with its manager-owned
Snapshot, Quiesce and Resume hooks. Quiesce closes new suite and preview admission
without canceling admitted preparation, queued work or running suites. Snapshot
covers admission tickets before registration, queued/running run identities,
previews and terminal finalization until each run's done channel closes. Queue
promotion and preview release preserve accounting. Inventory is bounded and
context-limited; missing, inconsistent or unavailable owner state refuses drain
instead of asserting zero. Resume reopens new admission under the retained fence.

The normal control-plane restart stages first, then owns the linked Prepare and
Drain generation before stop. A drain deadline leaves the existing executor
contexts intact. Shutdown cancellation is reached only after complete natural
quiescence; it is not used to manufacture a drained result. Durable seed sessions
remain a separate replacement/recovery authority, not nondurable suite work.

A serving image without these hooks proves HTTP ingress only. Its health route
does not inventory detached jobs, and the closed generic fence blocks ordinary
run/admission reads. Installing the source fix does not retroactively provide a
first-upgrade proof: a supported complete old-owner inventory under the linked
closed fence must be established before claiming that bootstrap is safe.

## Seed lifecycle and routed ownership

`SeedService` is the generated inter-scenario contract in
`packages/proto/schemas/test-genie/v1/seeds/seeds.proto`. Workflow Health consumes
it; legacy seed CLI routes use the same reservation and cleanup registry.
`api/internal/app/httpserver/seed_sessions.go` reserves before effects, coalesces
the same retained scenario/request/checkout, and refuses competing owners without
exposing their token. Retired outcomes remain for one hour, with a 1024-record
capacity bound and at most four active preparations/sessions. Active or failed
cleanup ownership is never evicted to free capacity.
The shared apply boundary validates scenario/request lengths and the raw and
canonical checkout path (4096 bytes, no NUL or line breaks) before reservation.
New admissions have a 7 MiB encoded registry budget; the 8 MiB read/write ceiling
leaves headroom for active cleanup and pruning. A byte-capacity rejection rolls
back its effect-free checkpoint and returns ResourceExhausted without poisoning
known cleanup authority. Private error excerpts are bounded to 4 KiB.

The private `state/seed-sessions` domain contains the registry, per-run cleanup
checkpoints, exact allocation descriptors and native execution gates. Files use
mode 0600 under a 0700 owner directory. One process-lifetime platform lock
serializes replacements; HTTP drain alone does not release it. Missing or corrupt
ownership state fails seed admission closed. Registry and checkpoint readers require exactly one complete JSON document;
trailing malformed content or a second document refuses authority without rewriting
it. Registry pruning is committed before retired checkpoint deletion and remains
retryable after interruption. A retirement whose registry publication failed
keeps its exact slot and cleanup handle; the next successful publication releases
only that matching slot, and pending retirement cannot be pruned.
A missing registry is a fresh owner only when its private namespace contains no
retained state; a checkpoint, backing or gate entry causes refusal. Preparation
publishes the session, consumer deadline and cleanup capability together under
the same lock used by journal snapshots.
The entire journal must validate before admission, observation, cleanup or
maintenance may mutate it. Failed initial loads retain their exact bytes across
replacement; the background loop cannot replace them with a guessed empty map.
A later transient write failure on a validated owner preserves known cleanup
authority and its safe retry path. A successful registry publication clears only
an error tagged as a registry-publication failure. It cannot clear checkpoint or
unrelated authority refusal.

Before checkpoint creation, the same registry publishes one bounded
`pending_reservation` with the exact run, request, token and checkout identity in
`checkpoint-only-v1` state. It counts toward active/retained and encoded byte
bounds; capacity refusal precedes any checkpoint effect. Only an acknowledged
transfer into the normal session row authorizes setup. Maintenance or replacement
can retire this pending plan only when its private checkpoint is absent or byte
identical to the canonical effect-free reservation. Nonempty, foreign, public or
aliased files remain untouched and keep admission refused. Removal requires a
directory-sync acknowledgement; a failed removal, sync acknowledgement or
retirement publication retains the same pending plan for retry. Successful exact
retirement publishes a retained failed/pruning outcome before reopening admission,
so the original error and same-key no-replay behavior survive replacement.
Unsupported pending state or duplicate ownership fails initial loading closed;
older strict readers also refuse a pending extension they do not understand.
The complete document's semantic identity validates before any checkpoint recovery:
one canonical scenario/raw-request binding regardless checkout, unique token/run
ownership, and one active scenario including pending. The same expiration
predicate governs Apply, loading and pruning: a retired outcome expires strictly
after one hour. Expired retired history may coexist with a new request owner while
pruning lags; a pending reservation never expires by creation age. Distinct raw
request bytes and legitimate retained legacy history remain independent. Empty
legacy request identities remain unbound, as Apply generates a new request ID
for empty input before lookup.

A successful Apply starts a 60-second consumer lease. GetSeed renews it while
its exact token is active; Workflow Health observes it every 10 seconds. The
five-second owner sweep revokes expired consumers and retries cleanup. A lost
Apply response can reattach with the same identity during this lease. An expired
consumer cannot revive its session. A replacement reconstructs cleanup only,
marks interrupted validation failed and retires the original exact backing; it
never reruns a seed or invents healthy execution proof. Unreachable targets and
unverified native exit retain their cleanup handle and capacity until exact
retirement can be proved.

`phases.ApplyPlaybooksSeed` owns provisioning, paired installation, seed
execution and retirement. The current suite workflow phase delegates to durable
Workflow Health runs, whose seed coordinator calls SeedService; it does not call
this helper directly. SeedService always supplies the private checkpoint. Direct
helper calls without that context retain normal live cleanup but do not provide
process-loss recovery and are not qualified as durable owners.
The script `seeds.timeout_ms` budget (30 seconds by default) retains its script
meaning. Total preparation, including detection, allocation, migrations, routing
and scripts, has an owner deadline of two minutes plus that script budget; an
earlier caller deadline wins. SQLite migrations pass that context to every
statement. Canceled preparation cannot proceed to routing or native dispatch,
and exact cleanup runs under its separate bounded cancellation-independent budget.
Ownership begins with the first partial allocation,
including migration and install-preflight failures. Cleanup errors and original
routing counters stay visible after a later successful transport. Before backing
retirement, the owner proves seed descendants quiescent, obtains exact target
clear acknowledgment and persists its counters. SQLite recovery requires the
private root and exact owner marker. SQLite prepares a directory in the locked
owner's private `backing/.staging/<runID>` namespace, fsyncs its marker, then
publishes it with Linux no-replace directory rename and syncs the parent.
Replacement may retire only the exact planned staging entry, containing either
no file or its bounded partial owner marker; foreign content and aliases refuse
cleanup. Final unmarked or mismatched roots are always retained. Standalone
callers get a dedicated private temporary owner namespace and retire it after
normal cleanup. Other platforms refuse this publication capability.
PostgreSQL fallback requires the database's
exact owner comment; an uncertain CREATE without that marker remains retained.
Private Docker allocations bind an already-installed immutable image ID, UUID
name and owner label, with loopback-only ports and AutoRemove disabled. They
have no process-session reaper. Image provisioning belongs to the resource owner;
this allocator never pulls images. Shared Redis DB-index/FLUSHDB fallback is
unsupported because it cannot prove private ownership.

`seeds.FileManager` retains a copy of the current API worker image and a bounded
0600 child-input file inside its private operation root. Native argv carries
only input paths and hashes, and its environment contains no child credentials.
The early worker validates its image/input, exact native root and one consumed
claim before using `exec.Cmd.Env` for the seed. A bounded private log contains
child output; native/public diagnostics remain generic. Worker images and secret
inputs retire as soon as exact native exclusion is proved, even if subsequent
target clear is unavailable. Historical outcomes retain compact metadata rather
than API image copies. The shared `platform-go` owner imposes wall, memory and
task bounds and proves
recursive native quiescence before cleanup. Its private revocable pre-exec gate
is persisted with the consumed operation identity. Cleanup durably revokes the
gate before interpreting complete unit/cgroup absence; a visible pending unit
still requires exact stop. Gate preparation and revocation serialize across
processes. A denied condition is failed execution, not successful seeding.
Repository-backed preset/catalog defaults resolve during ordinary orchestration,
not package initialization. The full API entry can therefore reach worker
validation with only its minimal native bootstrap environment, including in
`-trimpath` builds; it needs no repository-root environment workaround. The
API main regression exercises this full entry with no API/storage credentials
and invalid private input before any API construction.
Native ownership is qualified on Linux; unsupported backends refuse execution.
Seed values come only from a freshly produced, size-bounded state file after a
supported seed entrypoint succeeds. Both raw and re-encoded seed state must fit
1 MiB before publication into the session; JSON escaping must not inflate a
loadable input into an unloadable cleanup checkpoint. Checkpoint reads and
writes share a 4 MiB ceiling, checked before replacing authority. Repeated
checkpoint failures retain the first sticky error instead of accumulating
duplicate retry errors. Explicit legacy debug retention keeps backing
by request and retains its private descriptor; the generated browser path always
uses normal automatic retirement.

Tests include actual killed owner replacement both after successful seeding and
while a native descendant runs, accelerated consumer expiry with real paired
storage, partial-allocation cleanup retry, foreign-owner refusal,
killed SQLite preparation at unmarked-stage, marker-synced and publication
cuts, default standalone allocation and marker-write failure, actual held-pipe
and redirected-output descendants, and native delayed-admission/pre-exec gate
races. These fixtures do not establish natural served expiry or simultaneous
target-process restart/counter recovery; retain the separate operated evidence
and the target restart protocol in `docs/agent-system/routed-test-db.md`.
