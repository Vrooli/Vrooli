# Seams — Data Backup Manager

A **seam** is a deliberate boundary where production code calls an
interface, not a concrete dependency. The fake substitutes through that
interface in tests; production wires the real implementation once at the
composition edge (`main.go` for cross-cutting dependencies, or the
domain's `handlers/<domain>/module.go` for domain-local dependencies).

This document is the authoritative list of seams in this scenario. Add
to it whenever you introduce a new interface that production wires once
and tests substitute. Remove from it only when the seam is genuinely
gone — not when "we don't fake it yet."

## Wire contracts live in proto, not seams

Before adding a new seam, ask: is this a *boundary* (a place where
production-vs-test substitution matters) or a *contract* (a payload
shape consumed by multiple processes)? Wire contracts belong in
`packages/proto/schemas/data-backup-manager/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/data-backup-manager/v1/health/`
is the worked example. The Go fixture (`api/internal/testutil/fixtures/health.go`)
re-exports the generated `Response` and provides functional-options
builders; the UI factory (`ui/src/test-utils/factories.ts`) builds
the same generated type via `create(ResponseSchema, ...)`. Drift
between the two is impossible because both consume one source of
truth.

For proto-typed API calls, the service block in the proto is also the
transport contract. Generated Connect-Go handlers and Connect-Web/Go
clients are the seam at the wire boundary. REST remains intentional
only where the payload is not proto-typed, such as multipart file
uploads; the response metadata still uses generated proto types.

If a piece of production code reaches for `time.Now()`, `*sql.DB`, or
the network without going through one of the entries below, that's a
new seam that hasn't been declared yet. Declaring it is the work — the
test ergonomics fall out for free.

## Workflow transitions are not seams

Temporal workflow logic is domain policy. A workflow file such as
`internal/<domain>/workflow.go` or
`ui/src/features/<domain>/<domain>Workflow.ts` defines allowed
state/event transitions and invariants; production and tests call the
same transition function directly.

The seams are the side-effect boundaries around that workflow:

- repositories that persist the state,
- BlobStore or filesystem clients that store opaque bytes,
- clocks, timers, schedulers, and retry drivers,
- outbound HTTP clients and other scenario clients,
- UI API modules that issue network requests.

Keep the rule split explicit:

```
handler/component/job
  -> call pure workflow transition
  -> perform side effects through seams
  -> persist/render the resulting state
```

If a test needs to substitute a dependency, add or reuse a seam. If a
test only needs to prove the state machine, call the workflow directly
and use matrix/trace helpers from the relevant testutil package.

## How to read this file

| Column | Meaning |
|---|---|
| **Seam** | Short name used to refer to the boundary in conversation. |
| **Interface** | Go file & symbol that defines the contract. |
| **Production wiring** | Where the real implementation is constructed. |
| **Test fake** | The fake under `internal/testutil/mocks/` (cross-domain) or `internal/<dom>/mocks/` (per-domain) that substitutes. |
| **Why it exists** | The class of bug it prevents or the test ergonomic it enables. |

## Current seams

### Clock

| | |
|---|---|
| **Seam** | Wall-clock time |
| **Interface** | `internal/clock/clock.go::Clock` (`Now() time.Time`) |
| **Production wiring** | `main.go` constructs `clock.System{}` and passes it via `server.Deps`. |
| **Test fake** | `internal/testutil/mocks::FakeClock` (`Now`, `Advance`, `SetNow`). |
| **Why it exists** | Middleware computes request-duration log lines from two `Now()` calls. With `time.Now()` direct, duration assertions are flaky on loaded CI and undefined on fast hardware. With `FakeClock.Advance(150 * time.Millisecond)` inside the inner handler, the duration string is bit-for-bit deterministic. See `internal/middleware/logging_test.go::TestLoggingMiddleware_LogsDuration`. |

### Pinger (database reachability)

| | |
|---|---|
| **Seam** | Database reachability probe |
| **Interface** | `internal/database/pinger.go::Pinger` (`PingContext(ctx) error`) |
| **Production wiring** | `main.go` opens `*sql.DB` via `database.Connect(...)` against `modernc.org/sqlite` (pure-Go, CGO-clean). `*sql.DB` satisfies `Pinger` directly — no wrapper. |
| **Test fake** | `internal/testutil/mocks::FakePinger` (`PingErr error`, atomic `Calls` counter). |
| **Why it exists** | The `/health` handler probes the database. Without the seam, every handler test would either open the on-disk SQLite file (slow at scale, parallel-test contention) or skip the database branch entirely (untested degradation path). With `FakePinger{PingErr: errors.New("connection refused")}`, the unhealthy branch is one line. See `handlers/health/handler_test.go`. |

### database.SchemaExecer (shared schema application)

| | |
|---|---|
| **Seam** | Shared api-core schema execution surface |
| **Interface** | `api-core/database::SchemaExecer` (`ExecContext(ctx, query, args...)`) consumed by `database.EnsureSchemas`. |
| **Production wiring** | `main.go` applies `modules.AllSchemas()` over a real `*sql.DB`; domain sqlite tests compose `localdb.SystemSchema` with the domain's own `Schema()` provider before exercising repositories. |
| **Test fake** | `api-core/databasetest::FakeExecer` is the canonical fake when a test needs to assert schema application order or injected execution failures without opening a real database. |
| **Why it exists** | Schema application is shared-package behavior, but each scenario owns its provider list. Keep scenario-specific schema composition local; use `databasetest.FakeExecer` only for tests of code that consumes the shared `SchemaExecer` interface directly. |

### Per-domain Repository + Service seams

Every product domain follows the same two-seam shape. The `targets` domain is
the canonical reference for catalog-style CRUD; `runs` and `restores` are the
references for orchestration seams.

| Domain | Repository (persistence seam) | Service (application seam) | Production wiring | Test fakes |
|---|---|---|---|---|
| targets | `internal/targets/repository.go::Repository` (Create/Update/GetByOwnerName/GetByID/List/DeleteByOwnerName) | `internal/targets/service.go::Service` (Register idempotent upsert, Deregister, Get, List) | `handlers/targets/module.go::Module(db, clk, logger)` builds repo→service→connect handler | `internal/targets/mocks::{FakeRepository, FakeService}` |
| destinations | `internal/destinations/repository.go::Repository` | `internal/destinations/service.go::Service` (Create→bundle prepare/RepoCreate/RepoStatus/bundle metadata, slug-safe name, separate-root rule, bundle-root vs repository-path split, usage-vs-cap, `WouldBlock`) | `handlers/destinations/module.go::Module(db, clk, KopiaEngine, protectedRoot, logger)` (constructs `FSBundleWriter` internally) | `internal/destinations/mocks::{FakeRepository, FakeService, FakeBundleWriter}` |
| plans | `internal/plans/repository.go::Repository` (membership tables) | `internal/plans/service.go::Service` (+ `SchedulablePlans` for the scheduler) | `handlers/plans/module.go::Module(db, clk, logger)` | `internal/plans/mocks::{FakeRepository, FakeService}` |
| runs | `internal/runs/repository.go::Repository` (runs + outcomes + last-success rollup) | `internal/runs/service.go::Service` (TriggerRun orchestration; deps via `deps.go` seams) | service built in `main.go` (needs cross-domain adapters), mounted via `handlers/runs/module.go::Module(svc, logger)` | `internal/runs/mocks::{FakeRepository, FakeService, FakePlanLookup, FakeTargetLookup, FakeDestinationLookup, FakeEventSink}` |
| restores | `internal/restores/repository.go::Repository` | `internal/restores/service.go::Service` (RestoreTarget, VerifyTarget gate) | service built in `main.go`, mounted via `handlers/restores/module.go::Module(svc, logger)` | `internal/restores/mocks::{FakeService, FakeTargetLookup, FakeDestinationLookup}` |

The cross-domain reader/effect seams the runs and restores orchestration depend
on (`PlanLookup`, `TargetLookup`, `DestinationLookup`, `EventSink`, restores'
`TargetLookup`/`DestinationLookup`) are declared in each consumer's `deps.go`
and satisfied by thin adapters in `api/adapters.go` (the composition root) over
the concrete sibling services — keeping the domains decoupled and unit-testable
against fakes.

### Destination readiness and preparation

| | |
|---|---|
| **Seam** | Read-only destination/device inspection |
| **Interface** | `internal/destinationreadiness/inspector.go::Inspector` (`Inspect(ctx, location)`) plus `VolumeScanner` for mounted-volume enumeration. |
| **Production wiring** | `handlers/destinations/module.go` constructs `destinationreadiness.NewReadOnlyInspector(sysmounts.New())` and passes it to `destinationreadiness.NewService(...)`. The inspector reads mount metadata and bounded top-level names only; it never writes probe files or mutates devices. |
| **Test fake** | `internal/destinationreadiness/service_test.go::fakeInspector` and `inspector_test.go::fakeVolumeScanner`. |
| **Why it exists** | Drive-readiness analysis must be testable without touching real removable media. The seam lets tests model FAT32, installer media, read-only mounts, low capacity, and identity drift without scanning or writing host devices. |

| | |
|---|---|
| **Seam** | Destination preparation executor |
| **Interface** | `internal/destinationreadiness/service.go::Preparer` (`Supported(action)`, `Execute(ctx, plan)`). |
| **Production wiring** | `handlers/destinations/module.go` wires `internal/destinationreadiness/preparer.go::LocalPreparer`. It supports only `create_subdir`: create a child directory, write a short-lived probe file, remove the probe, then return. `format`, `relabel`, and `clear_directory` remain explicitly unsupported. |
| **Test fake** | `internal/destinationreadiness/service_test.go::fakePreparer`, which records the requested plan and never touches devices. `preparer_test.go` uses `t.TempDir()` and injected filesystem functions to verify create-subdir behavior without mutating removable media. |
| **Why it exists** | Formatting, clearing, relabeling, and even creating backup directories are safety-sensitive. The domain proves that analysis and planning do not execute effects and that execution is unreachable until confirmation, data-loss acknowledgement, supported-platform, and identity guards pass. |

### Connect router (proto-typed transport)

| | |
|---|---|
| **Seam** | Generated Connect services mounted on the scenario's existing mux router |
| **Interface** | `api-core/connectx::RegisterServices(router, mounts...)`, where each mount is `{Path, Handler}` returned by generated `New<Domain>Handler(...)` |
| **Production wiring** | `handlers/<domain>/module.go` constructs the domain service, passes it to `NewConnectHandler`, then mounts the generated handler with `connectx.RegisterServices`. The server's existing middleware still wraps the handler because Connect is standard `http.Handler`. |
| **Test fake** | `api-core/connectxtest::StartTestServer` is the canonical in-process server harness for handler tests. `connectxtest.NewLogger` is the canonical logger capture helper. Module tests can still mount the module on a mux router and issue real HTTP requests. No hand-written request JSON ribbon is needed in tests. |
| **Why it exists** | The proto service descriptor becomes the single wire contract for UI, CLI, and API. Handler path, method, request type, response type, and Connect error envelope all come from generated code instead of parallel route tables. |

### cliapp RunContext (CLI handler test context)

| | |
|---|---|
| **Seam** | Shared cli-core command handler context |
| **Interface** | `cli-core/cliapp::RunContext` plus `ArgSchema` parser inputs. |
| **Production wiring** | The CLI dispatcher builds `RunContext` through cli-core's parser and injects the scenario app, stdout, stderr, and built-in `--json` state. |
| **Test fake** | `cli-core/cliapptest::NewTestRunContext` and `NewTestRunContextFromArgs` are the canonical constructors for tests that drive `RunCtx` handlers directly. |
| **Why it exists** | CLI domain tests should exercise handler behavior without duplicating parser setup or relying on `cliapp`'s inline test exports. The sibling test companion keeps future CLI tests aligned with shared-package test-helper ownership. |

### cli/manifest.json ↔ handlers bindings (CLI command surface)

| | |
|---|---|
| **Seam** | Declarative CLI command surface — single source of truth for groups, commands, args, governance, and proto-method bindings. |
| **Interface** | `cli/manifest.json` validated against `.vrooli/schemas/cli-manifest.schema.json` (`cli-manifest/v1`); resolved via `repocontract.ScenarioCLIManifestPath`; consumed by `cliapp.LoadFromManifest(raw, groupName, bindings)` where `bindings` is `map["<Service>.<Method>"]func(RunContext) error`. |
| **Production wiring** | `cli/manifest_embed.go` embeds the manifest bytes; `cli/app.go` passes them to `domains.SubcommandGroups(core, manifest)`; each domain's `Register(core, manifest)` calls `cliapp.LoadFromManifest` with its group name and a bindings map keyed by `Service.Method`. |
| **Test fake** | `cli-core/cliapp::RequireProtoServiceCoverage(t, manifest, fd, serviceName)` asserts every RPC on the bound proto service has either a binding or an entry in the manifest's `omitted` array — see the per-domain `*_manifest_test.go` files under `cli/domains/*/`. `cliapp.ParseManifest` covers structural validation in isolation. |
| **Why it exists** | Without this seam, adding a new RPC to the proto compiles fine while the CLI silently lacks a corresponding command, and prompt-manager has no governance signal — every action falls back to `CertaintyOwnerOnly` and is rejected. The manifest crystallises both the command surface and the safety properties (effect, run_eligible, permissions, requires_confirmation) so the coverage test fails fast and prompt-manager can derive certainty automatically. |

### BlobStore (opaque bytes)

| | |
|---|---|
| **Seam** | Binary object storage for REST multipart edges |
| **Interface** | `api-core/blobstore::BlobStore` (`Put`, `Get`, `Delete`) |
| **Production wiring** | A domain module that exposes multipart endpoints owns its blob store. Data Backup Manager currently has no multipart endpoint; restore and snapshot bytes move through the `KopiaEngine` seam instead. |
| **Test fake** | `api-core/blobstore.MemoryBlobStore` or a domain-local fake lets handler tests assert metadata and failure behavior without touching the filesystem. |
| **Why it exists** | Connect-RPC is the default for proto-typed payloads, but opaque bytes are not proto payloads. Keeping bytes behind `BlobStore` lets the handler stay transport-focused and lets future scenarios swap filesystem, S3, or another object store without changing domain services. |

### module.Module (domain composition)

| | |
|---|---|
| **Seam** | Domain-to-server composition; the contract every handler package returns from its `Module(...)` constructor. |
| **Interface** | `internal/module/module.go::Module` (`Name string`, `Mount func(r *mux.Router)`, `Endpoints []EndpointDescriptor`). Data type, not behaviour — modules don't have methods. |
| **Production wiring** | `main.go` calls `healthH.Module(...)`, `targetsH.Module(...)`, `destinationsH.Module(...)`, and the other product modules, then passes the slice to `server.New(deps, modules...)`. The server iterates `m.Mount(s.router)` after registering the logging middleware. |
| **Test fake** | A literal `module.Module{Name: "stub", Mount: func(r){...}}` in `internal/server/server_test.go` proves the iteration; per-domain `module_test.go` files exercise the real constructors against in-memory fixtures. |
| **Why it exists** | Eliminates the central registry that would otherwise grow per-domain fields on `server.Deps` and per-domain wiring lines in `routes.go`. Adding a domain means creating files; deleting one means removing files. The endpoint descriptors travel with the module, so `.vrooli/endpoints.json` codegen has a single source per domain (no manual JSON editing). |

### Endpoints codegen (manifest source-of-truth)

| | |
|---|---|
| **Seam** | The `.vrooli/endpoints.json` API documentation manifest. |
| **Interface** | `api/cmd/gen-endpoints/main.go` reads `internal/modules.AllEndpoints()` — the shared registry that collects each handler's static `Endpoints []module.EndpointDescriptor` slice plus `cli_commands_seed.json`. Output is the canonical envelope at `.vrooli/endpoints.json`. |
| **Production wiring** | Run via `make endpoints`. CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json` so a stale manifest fails the build with an actionable diff. |
| **Test fake** | `api/cmd/gen-endpoints/main_test.go` exercises the codegen with hand-built fixtures and asserts the output is valid JSON with the canonical envelope. `internal/modules/registry_test.go` pins the registry shape (non-empty, stable order). The cross-check (every `cli_mapping.command` in `endpoints[]` matches a `cli_commands[].name`) has its own unit test. |
| **Why it exists** | Hand-edited endpoints manifests drift from real handlers. The shared `modules` registry means runtime (`main.go`) and codegen (`gen-endpoints`) read endpoints + schema from one place — adding a domain is two registry lines, not separate edits in `main.go` and `gen-endpoints/main.go`. The CI drift check makes "I forgot to regenerate" a build failure, not a stale-doc bug. |

### database.SystemSchema (cross-cutting infrastructure)

| | |
|---|---|
| **Seam** | Cross-cutting database infrastructure (postgres extensions, custom types, cross-domain views) |
| **Interface** | `internal/database/system.go::SystemSchema() string` (consumed via `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists `apidb.SchemaProviderFunc(localdb.SystemSchema)` first; `main.go` passes the slice into `apidb.EnsureSchemas`. |
| **Test fake** | None. The system file ships empty in the template and is verified empty by `internal/database/system_test.go::TestSystemSchema_IsEmpty` (a deliberate tripwire — adding a `CREATE TABLE` here forces a "yes, this is genuinely cross-cutting" decision). |
| **Why it exists** | Some bits don't belong to any one domain — postgres extensions, type definitions, reporting views. Putting them in a domain package would force fictional ownership. The system home is honest: cross-cutting goes here, single-domain bits go in `internal/<dom>/schema.sql`. |

### Per-domain Schema (domain-owned SQL)

| | |
|---|---|
| **Seam** | Each domain's SQL contribution |
| **Interface** | `internal/<domain>/schema.go::Schema() string` (consumed via `handlers/<domain>/module.go::Schema` re-export, then `api-core/database.SchemaProvider`). Domains: targets, destinations, plans, runs, restores. |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists `apidb.SchemaProviderFunc(<domain>H.Schema)` for each domain (system → health → destinations → plans → restores → runs → targets); applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | Each domain's `sqlite_test.go` builds a fresh handle with `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` over the system + that domain's provider, so repository tests get a real table without touching the central registry. |
| **Why it exists** | Domain ownership of the schema. Adding a column lands in the same diff as the Go change. Deleting `internal/<domain>/` deletes the table definition with it, so removed domains do not leave tables created on boot. The `handlers/<domain>/module.go::Schema` re-export keeps the registry's import surface narrow. |

### Doer (outbound HTTP)

| | |
|---|---|
| **Seam** | Outbound HTTP request boundary |
| **Interface** | `internal/httpc/doer.go::Doer` (`Do(*http.Request) (*http.Response, error)`) |
| **Production wiring** | Ships unwired in production by intent (no consumer until a real outbound call lands). `internal/httpc/doer.go::NewDefaultClient` returns the timeout-backed production `Doer`; the first scenario to need an outbound call adds the field to `server.Deps` and wires that constructor from `main.go`. |
| **Test fake** | `internal/testutil/mocks::FakeDoer` (canned `*http.Response` queue, recorded `*http.Request` log, atomic `Calls` counter). |
| **Why it exists** | Network calls in handler tests would be flaky and slow. Defining the seam *before* the first consumer means the first scenario to call outward doesn't reinvent ad-hoc mocking. Pattern proven in `scenarios/agent-manager/api/internal/promptmanager/client.go`. See `internal/httpc/doer_test.go` for the substitution reference. |

### KopiaEngine (backup engine boundary)

| | |
|---|---|
| **Seam** | The single boundary to the wrapped backup engine (`resource-kopia`). |
| **Interface** | `internal/engine/kopia.go::KopiaEngine` (`RepoCreate`, `RepoStatus`, `RepoStats`, `SnapshotCreate(…, SnapshotMetadata)`/`List`/`Restore`/`Verify`, `BrowseSnapshot`, `PolicySet`). `SnapshotCreate` takes optional self-identifying metadata (description/override-source/tags) passed through to kopia; never carries a secret. |
| **Production wiring** | `engine.NewKopiaCLI()` shells out to `resource-kopia` through the `CommandRunner` seam; wired once and threaded into the destinations, runs, and restores modules. Lives in the substrate `internal/engine/` package (consumed by three domains), mirroring the `httpc.Doer` ambient-seam shape. |
| **Test fake** | `internal/testutil/mocks::FakeKopiaEngine` — per-method overridable func fields with minimally-working defaults (RepoCreate remembers the repo, SnapshotCreate returns deterministic ids, RepoStatus reports encryption on), plus a `Calls` log. A test programs `RepoStatsFn`/`SnapshotVerifyFn` etc. to drive cap-block and checksum-mismatch paths. |
| **Why it exists** | Backup/restore/verify must be substitutable so domain tests stay hermetic — they assert "the run snapshotted into the right repo and applied retention", not real kopia behavior. Encryption-on-by-default and "kopia owns the passphrase via vault" mean no secret ever crosses this interface; the production impl never puts a passphrase in argv. Real-engine behavior is covered by integration tests gated behind `KOPIA_INTEGRATION`. |

### BundleWriter (filesystem destination bundle)

| | |
|---|---|
| **Seam** | The boundary that turns a bare filesystem folder into a self-describing Vrooli backup bundle (README.txt, RECOVERY.txt, vrooli-backup-destination.json) and creates the nested `repositories/<slug>.kopia` repository directory. |
| **Interface** | `internal/destinations/bundle.go::BundleWriter` (`PrepareRepository`, `WriteMetadata`). |
| **Production wiring** | `&destinations.FSBundleWriter{}` constructed in `handlers/destinations/module.go::Module` and `main.go`; the destinations service calls it only for filesystem backends (S3 skips it). |
| **Test fake** | `internal/destinations/mocks::FakeBundleWriter` — records `PrepareRepository`/`WriteMetadata` calls so `CreateDestination` is exercised without touching the real filesystem, with `PrepareErr`/`MetadataErr` knobs. |
| **Why it exists** | A detached external drive must explain itself: the operator-facing bundle root carries human-readable README/RECOVERY plus a non-secret JSON manifest, while the vanilla kopia repository stays nested under `repositories/<slug>.kopia` (never wrapped or made DBM-proprietary). The seam keeps that filesystem materialization out of the service's decision logic and out of test paths. The writer is idempotent on identical content and fails closed on conflicting pre-existing files; it never writes a secret value (only a vault secret *reference*). |

### CommandRunner (process-exec boundary for resource-kopia)

| | |
|---|---|
| **Seam** | Process-exec boundary the production `KopiaCLI` shells through. |
| **Interface** | `internal/engine/kopia.go::CommandRunner` (`Run(ctx, args...) ([]byte, error)`). |
| **Production wiring** | `engine.ExecRunner{Binary: "resource-kopia"}` runs the wrapped CLI via `os/exec` and returns stdout. |
| **Test fake** | A recording fake (queued stdout + recorded argv) lets engine unit tests assert the exact `resource-kopia` argv — notably that secrets never appear as flags — and stub kopia's JSON without installing kopia. |
| **Why it exists** | Keeps argv construction and kopia-JSON parsing unit-testable without a kopia install, and makes the "no secret in argv" invariant a fast assertion rather than an integration-only property. |

### sources.Capturer (per-source-kind capture/restore)

| | |
|---|---|
| **Seam** | The per-source-kind boundary that turns a source into a snapshottable artifact and applies a restored artifact back. |
| **Interface** | `internal/sources/sources.go::Capturer` (`Kind`, `Capture(CaptureSpec) → Artifact`, `Restore(RestoreSpec)`). Resolved by kind through `sources.Registry`. |
| **Production wiring** | `sources.NewRegistry(fs, sqlite, postgres, redis, qdrant, object)` is built once (each impl in `internal/sources/<kind>.go`) and held by the runs and restores modules, which dispatch each target to the matching capturer. Filesystem/SQLite need no resource; postgres/redis/qdrant/object wrap their source resource CLIs (which self-source credentials, sidestepping the vault secret-read gap). |
| **Test fake** | `internal/sources/mocks::FakeCapturer` — claims a `SourceKind`, returns deterministic bytes or an injected failure via `CaptureFn`/`RestoreFn`, records calls. Runs/restores service tests build a Registry of fakes to drive fan-out and partial-failure without touching real sources. |
| **Why it exists** | The six kinds differ wildly (tar vs `VACUUM INTO` vs `pg_dump` vs prefix dump vs snapshot API vs object mirror) but the runs/restores orchestration is identical across them. The seam lets the workflow be tested once against fakes; per-kind round-trip correctness is proven by integration tests in `internal/sources/<kind>_test.go` (fs/sqlite always; the rest gated on their resources). |

### VolumeScanner (OS mount enumeration for discovery)

| | |
|---|---|
| **Seam** | The OS boundary for enumerating mounted volumes (the destination-suggestion source). |
| **Interface** | `internal/discovery/seams.go::VolumeScanner` (`Scan(ctx) → []Volume`). |
| **Production wiring** | `sysmounts.New()` (`internal/sysmounts/`) — the ONLY package allowed to import `gopsutil/v3/disk`. gopsutil supplies enumeration + free/total; a gopsutil-free per-OS classifier (`classify.go`) adds removable/network classification (Linux reads an injectable `/sys/block/<dev>/removable`; macOS keys off `/Volumes`; Windows stays conservative). |
| **Test fake** | `internal/discovery/mocks::FakeVolumeScanner` (fixed slice or error). The removable classifier itself is unit-tested against a fixture `/sys/block` tree in `internal/sysmounts/classify_test.go`. |
| **Why it exists** | Confines the one third-party dependency (gopsutil) and all OS-specific drive logic to one package so the discovery domain stays hermetic and the removable heuristic is testable without real hardware. |

### TargetSourceScanner (well-known runtime-state scan for discovery)

| | |
|---|---|
| **Seam** | Enumerates well-known sources worth protecting (the target-suggestion source). |
| **Interface** | `internal/discovery/seams.go::TargetSourceScanner` (`Scan(ctx) → []TargetCandidate`). |
| **Production wiring** | `discovery.NewWellKnownScanner()` reads a data-driven manifest scoped to the resolved runtime root (`APP_DATA_DIR`/`VROOLI_DATA`/`~/.vrooli`): `plans`, `state`, `config`, `secrets.json` (filesystem) and `runtime.db` (sqlite). Strictly read-only — stats paths and bounded-walks directory sizes; never reads file contents (e.g. `secrets.json` is suggested by path/size only). |
| **Test fake** | `internal/discovery/mocks::FakeTargetSourceScanner`; the production scanner is tested against a temp runtime root in `wellknown_test.go`. |
| **Why it exists** | Lets discovery suggest the right runtime state without the domain hard-coding paths, and keeps scenario-store scanning a future, additive `rootKind` (deferred). |

### discovery catalogs + ProtectedPaths + DismissalStore

| | |
|---|---|
| **Seam** | The live-catalog readers and dismissal writer the discovery service filters against. |
| **Interface** | `internal/discovery/seams.go`: `TargetCatalog`/`DestinationCatalog` (`ListAll`), `ProtectedPaths` (`ProtectedPaths(ctx) → []string`), `DismissalStore` (`IsDismissed`/`Dismiss`). |
| **Production wiring** | Composition-root adapters in `api/adapters.go` (`discoveryTargetCatalog`, `discoveryDestCatalog`, `discoveryProtectedPaths`) over the targets/destinations services; `DismissalStore` = `discovery.NewSQLiteDismissalStore`. `ProtectedPaths` = runtime root + every registered destination location + every registered target locator — deliberately wider than the destinations service's own `protectedRoot` (just `SCENARIO_DATA_DIR`), which is too narrow for destination filtering (Contract Decision D4). |
| **Test fake** | `internal/discovery/mocks` (`FakeTargetCatalog`, `FakeDestinationCatalog`, `FakeProtectedPaths`, `FakeDismissalStore`). |
| **Why it exists** | Keeps discovery decoupled from sibling domains (it reads their projections, never imports them) and keeps suggestions derived: only dismissals persist, so an accepted suggestion disappears once it exists in the catalog. |

## Adding a new seam

The right time to add a seam is the moment you find yourself reaching
past `*sql.DB`, `http.Get`, or `os.OpenFile` from a handler/service. The
process is mechanical.

## Architecture Alignment Notes

Use this section for durable boundary decisions discovered during
screaming-architecture work. Keep the full mental model in
[`ARCHITECTURE.md`](../concepts/ARCHITECTURE.md); this table records
why a boundary lives where it does and what still needs follow-up.

| Area | Drift | Decision | Follow-up |
|---|---|---|---|
| Domain-specific fakes | Earlier templates often put every fake under shared testutil. | One-domain fakes live under `api/internal/<domain>/mocks/`; cross-domain fakes stay under `api/internal/testutil/mocks/`. | Preserve this split when adding new domains. |
| Temporal workflow side effects | Async flows can bury transition rules inside handlers/components. | Pure workflow transitions are domain policy; side effects remain behind seams. | See [`FLOWS.md`](../concepts/FLOWS.md) for modeled flows. |
| Domain schemas | Central schema files make domain deletion and ownership harder. | Domain tables live beside domain code; `internal/database/system.sql` is only for genuinely cross-cutting DB infrastructure. | If a table is added to system schema, document why it is not domain-owned. |

### Domain-scoped packages, not generic `services/`

When a seam belongs to a domain (targets, destinations, plans, runs, restores, …), it lives in
`internal/<domain>/`, NOT in `internal/database/` or
`internal/services/`. The targets package is the canonical CRUD example:

```
internal/targets/
  types.go         # Target, RegisterInput, ErrInvalidTarget, ErrTargetNotFound
  repository.go    # Repository interface
  sqlite.go        # NewSQLiteRepository (production impl)
  sqlite_test.go   # Repository tests against real sqlite
  service.go       # Service interface + impl (validation, defaults)
  service_test.go  # Service tests against FakeRepository
  schema.sql       # Domain-owned table DDL (Pass-3 pattern)
  schema.go        # //go:embed schema.sql + Schema() string
  schema_test.go   # Embed-content tripwire
  mocks/           # Co-located test fakes (package mocks)
    repository.go
    service.go
    repository_test.go
    service_test.go
```

Mocks are co-located under `internal/<dom>/mocks/`, NOT under
`internal/testutil/mocks/`. `mocks/repository.go` defines
`FakeRepository`; `mocks/service.go` defines `FakeService`. Deleting
`internal/<dom>/` takes the mocks, schema, and tests along in one
sweep.

`internal/database/` retains only cross-cutting infrastructure
(`Pinger`, `SystemSchema` for the empty/cross-cutting SQL home) —
never domain-specific interfaces.

`internal/testutil/mocks/` retains only cross-domain fakes
(`FakeClock`, `FakePinger`, `FakeDoer`).

### Mechanical steps

1. **Define the interface in a domain package.** Methods are exactly
   what callers need — no more, no less. Example:
   ```go
   // internal/tasks/repository.go
   package tasks

   type Repository interface {
       Create(ctx context.Context, t Task) (Task, error)
       Get(ctx context.Context, id string) (Task, error)
   }
   ```
2. **Implement it in production with the concrete dependency wrapped
   in an unexported struct.** The struct holds `*sql.DB`; the methods
   translate domain calls to SQL. Production wires this in `main.go`;
   tests never see it.
3. **Add a service alongside the repository.** Even if `Service`
   currently does nothing more than pass through, define it now —
   handlers should depend on the service, not the repository, so
   future validation/policy has a home that doesn't require a handler
   refactor.
4. **Add fakes in `internal/<domain>/mocks/`** (co-located with the
   domain, `package mocks`) named `repository.go` and `service.go`.
   Each method takes a per-method error knob (`CreateErr error`) plus
   any state it needs to return. Counters use `atomic.Int64`, not plain
   `int`, so race-detector tests don't flap. Cross-domain fakes
   (`FakeClock`, `FakePinger`, `FakeDoer`) stay in
   `internal/testutil/mocks/`.
5. **Update this document.** A row in the table above with the same
   five columns. If you skip this step, the seam exists but isn't
   discoverable — future readers will reinvent it parallel.
6. **Add `var _` compile-time assertions** wherever the interface is
   defined: `var _ Repository = (*sqliteRepository)(nil)`. The
   assertion moves "this concrete type satisfies the interface" from a
   runtime surprise into a compile error.

## UI-side seams

The UI uses different mechanisms (Vitest's `vi.mock` hoisting), but
the goal is the same: production wires once, tests substitute.

### `api/client::transport` (Connect-Web transport)

| | |
|---|---|
| **Seam** | Single Connect-Web transport factory for proto-typed UI API calls |
| **Module** | `ui/src/api/client.ts` exports `transport` from `@vrooli/api-base::createScenarioConnectTransport`, plus REST-only helpers (`ApiError`, `decodeApiError`, `makeApiError`, `uploadFile`) for multipart exceptions. |
| **Production wiring** | Every proto-typed domain client imports `transport` and constructs a generated client with `createClient(<Service>, transport)`. REST multipart helpers use `uploadFile()` and parse the metadata response with the generated proto descriptor. |
| **Test fake** | Component tests mock `api/<domain>` modules or typed client methods. REST helper tests stub `global.fetch` directly. Connect behavior is covered at the API boundary by the generated client and focused module tests. |
| **Why it exists** | Per-domain clients should not know URL suffix rules, fetch setup, or proto JSON parse details. Connect-Web centralizes those choices, while the REST helpers make the binary-upload exception explicit instead of becoming a second general transport pattern. |

### `api/<domain>` (per-domain client modules)

| | |
|---|---|
| **Seam** | UI ↔ API per-domain endpoints (canonical CRUD reference: `api/targets.ts`) |
| **Module** | `ui/src/api/targets.ts` exports `targetsClient = createClient(TargetsService, transport)`. |
| **Production wiring** | Feature components wire generated client methods through `useQuery` / `useMutation`, for example `targetsClient.listTargets({})` and `targetsClient.registerTarget({ ... })`. |
| **Test fake** | Component tests use inline `vi.mock("./api/targets", async (importOriginal) => ...)` and replace typed client methods. Factories build generated proto types, including `Timestamp` values. |
| **Why it exists** | The canonical per-domain client pattern. Mirror this shape when adding a second domain client: export the generated Connect client, keep binary-upload helpers beside it when needed, and let components consume typed results rather than hand-written response interfaces. |

### ErrorBoundary (render-error catch)

| | |
|---|---|
| **Boundary** | App-level render-error catch |
| **Module** | `ui/src/components/ErrorBoundary.tsx` (class component with `getDerivedStateFromError` + `componentDidCatch`) |
| **Production wiring** | `ui/src/main.tsx` wraps `<App />` inside `<QueryClientProvider>`; the boundary catches any render-time exception thrown by `App` or its descendants and shows the localised default fallback. |
| **Test fake** | None — the boundary is the system-under-test. `ui/src/components/ErrorBoundary.test.tsx` drives it with a controlled-throw fixture (`Throw({ when })`). |
| **Why it exists** | Render-time exceptions silently nuke the page in raw React (white screen, no recovery). Every mature Vrooli scenario hand-rolls a class boundary; the template ships one as the canonical pattern. The `onError` prop is exposed for telemetry sinks (Sentry, etc.); the template wires nothing in production by intent — scenarios add their own sink as needed. |

### i18n singleton (locale state)

| | |
|---|---|
| **Seam** | Active locale + translation lookup |
| **Module** | `ui/src/i18n/index.ts` (`i18n`, `setLocale`, `getCurrentLocale`, `useTranslation`) |
| **Production wiring** | `main.tsx` imports the module, which initialises i18next as a process-wide singleton. Components consume via `useTranslation()`. |
| **Test fake** | `test-setup.ts` switches the singleton into `cimode` before every test (so `t('app.title')` returns the literal key `"app.title"`). Tests that need real-locale behaviour opt back in via `await setLocale("en")` in their own `beforeEach`. |
| **Why it exists** | Module-level singleton is intentional — i18next's React integration assumes one instance per renderer. The seam is the *configuration* (what locale is active), not the *interface*. cimode + the typed `strings.*` registry let tests assert on key paths instead of brittle copy. |

## What is NOT a seam

- **Pure-function helpers** (`internal/i18n/format.go`). They have no
  dependencies; tests call them directly. No interface required.
- **Standard-library types you don't control** (`time.Duration`, `url.URL`).
  The cost of a seam is overhead unless you'd otherwise be tempted to
  reach for global state.
- **Configuration structs** read once at startup. The seam is the
  *consumer* of the config, not the loader.
- **Generated proto types.** They're contracts, not seams. See "Wire
  contracts live in proto, not seams" above.

## API contract manifest

Not a seam (no production-vs-test substitution), but worth listing
here for discoverability: `.vrooli/endpoints.json` is the canonical
declaration of every public HTTP endpoint plus its CLI mirror. Doc
generators, Postman collection builders, and SDK-stub tooling read it.

**The file is generated** from each handler module's
Connect service metadata plus each handler module's
`Endpoints []module.EndpointDescriptor` slice (see the
"Endpoints codegen" seam above). To add or change a public operation:

1. Update the proto service and generated Connect handler for
   proto-typed operations, or update the REST handler for a documented
   exception such as multipart upload.
2. Keep the descriptor metadata in `api/handlers/<dom>/module.go`
   aligned with the operation. Connect paths use the generated service
   path; REST exceptions use the explicit REST path and error envelope.
3. If the operation has a CLI mirror, add or update the matching row in
   `api/cmd/gen-endpoints/cli_commands_seed.json`.
4. Run `make endpoints` to regenerate `.vrooli/endpoints.json`.
5. Commit both the descriptor edit AND the regenerated manifest.

The CI drift check (`make endpoints && git diff --exit-code
.vrooli/endpoints.json`) fails the build if step 4 was skipped, with
an actionable diff showing exactly which entries diverged.

## Cross-references

- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
