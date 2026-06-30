# Seams — Search Hub

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
`packages/proto/schemas/search-hub/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/search-hub/v1/health/`
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
| **Production wiring** | `main.go` and sqlite tests pass a real `*sql.DB`; the notes sqlite helper composes scenario-specific providers (`localdb.SystemSchema`, `notes.Schema`) before applying them. |
| **Test fake** | `api-core/databasetest::FakeExecer` is the canonical fake when a test needs to assert schema application order or injected execution failures without opening a real database. |
| **Why it exists** | Schema application is shared-package behavior, but each scenario owns its provider list. Keep scenario-specific schema composition local; use `databasetest.FakeExecer` only for tests of code that consumes the shared `SchemaExecer` interface directly. |

### notes.Repository (notes persistence)

| | |
|---|---|
| **Seam** | Notes persistence (CRUD) |
| **Interface** | `internal/notes/repository.go::Repository` (`Create`, `Get`, `List`) |
| **Production wiring** | `handlers/notes/module.go::Module(...)` constructs `notes.NewSQLiteRepository(db, clk)` and passes it into `notes.NewService(repo)`. `main.go` only sees the returned `module.Module`; note-specific dependencies do not appear on `server.Deps`. Wire shape lives in `packages/proto/schemas/search-hub/v1/notes/notes.proto`. |
| **Test fake** | `internal/notes/mocks::FakeRepository` (co-located with the domain — in-memory slice, per-method error knobs `CreateErr` / `GetErr` / `ListErr`, atomic call counters). Used by `internal/notes/service_test.go` to drive the service against a controllable persistence layer. |
| **Why it exists** | Repository owns the persistence contract — sqlite SQL today, anything else tomorrow. The handler depends on `notes.Service`, not directly on the repository, so a backend swap doesn't ripple through transport. The repository test in `internal/notes/sqlite_test.go` substitutes the real handle to pin SQL semantics (ordering, limit, RFC3339 round-trip). |

### notes.Service (notes application layer)

| | |
|---|---|
| **Seam** | Notes application surface (validation, defaults, cross-handler policy) |
| **Interface** | `internal/notes/service.go::Service` (`Create(CreateInput) → Note`, `Get(id) → Note`, `List(limit) → []Note`) |
| **Production wiring** | `handlers/notes/module.go::Module(db, clk, logger)` constructs `notes.NewSQLiteRepository(db, clk)` then `notes.NewService(repo)` then `NewConnectHandler(Deps{Service: svc, Logger: logger})` — fully internal to the notes module. `main.go` only sees the `module.Module` returned from that constructor; per-domain services don't appear on `server.Deps`. The handler imports `internal/notes` for both the interface and the typed sentinels (`ErrInvalidNote`, `ErrNoteNotFound`) it translates at the transport edge. |
| **Test fake** | `internal/notes/mocks::FakeService` (co-located with the domain — records `CreateInputs`, returns canned `CreateOut` / `GetByID` / `ListOut`, per-method error knobs). Used by `handlers/notes/connect_handler_test.go` to drive the handler without validation/repository plumbing in scope. |
| **Why it exists** | Validation (`title required` after whitespace trim) and default substitution (`defaultListLimit = 100` when caller passes 0) are business policy, not transport policy. Putting them in the service keeps the handler thin and makes the same rules reachable from any future surface (batch jobs, scheduled imports, additional RPCs) without copy-paste. Two-mock split (`FakeRepository` for service tests, `FakeService` for handler tests) means handler tests don't seed sqlite-shaped state to assert routing. |

### notes.AttachmentsRepository (attachment metadata persistence)

| | |
|---|---|
| **Seam** | Note attachment metadata persistence |
| **Interface** | `internal/notes/repository.go::AttachmentsRepository` (`CreateAttachment`, `ListAttachmentKeys`) |
| **Production wiring** | `handlers/notes/module.go::Module(...)` constructs `notes.NewSQLiteAttachmentsRepository(db, clk)` (declared in `internal/notes/sqlite.go`, methods in `attachments_sqlite.go`) and passes it into `notes.NewAttachmentsService(...)`. The opaque file bytes go to `BlobStore` (separate seam below); only the typed metadata row passes through this interface. |
| **Test fake** | `internal/notes/mocks::FakeAttachmentsRepository` (co-located with the domain — in-memory `Attachments` slice, per-method error knobs `CreateErr` / `ListErr`, atomic call counters, UploadedAt backfill mirroring the sqlite repository). Used by `internal/notes/attachments_service_test.go` to drive the attachments service against a controllable persistence layer. |
| **Why it exists** | Splitting attachment-metadata persistence from notes persistence keeps the per-method surface narrow (the notes repository never grows attachment-shaped methods) and lets the attachments service remain transport-agnostic. The repository test in `internal/notes/sqlite_test.go::TestSQLiteRepository_AttachmentMetadataRoundTrip` substitutes the real handle to pin SQL semantics; service tests use the fake. |

### notes.AttachmentsService (attachment application layer)

| | |
|---|---|
| **Seam** | Note attachment application surface (validation, parent-note lookup, repository delegation) |
| **Interface** | `internal/notes/attachments_service.go::AttachmentsService` (`Create(CreateAttachmentInput) → Attachment`) |
| **Production wiring** | `handlers/notes/module.go::Module(...)` constructs `notes.NewAttachmentsService(notesRepo, attachmentsRepo)` then passes it as `AttachmentsDeps.Service` into `NewAttachmentsHandler(...)`. The handler is the multipart REST exception (the only non-Connect transport in the notes domain); the service stays unaware of multipart and HTTP. |
| **Test fake** | `internal/notes/mocks::FakeAttachmentsService` (records `CreateInputs`, returns canned `CreateOut` or synthesises an Attachment from the input, gated on `CreateErr`). Available for any future handler test that wants to assert routing/multipart wiring without standing up the real notes-and-attachments service tree. |
| **Why it exists** | Attachment validation (note id + key required after trim, positive size, parent note must exist) is business policy; multipart parsing and BlobStore I/O are transport policy. Keeping them split means a future scenario that adds a non-multipart attachment surface (CLI direct upload, scheduled import, gRPC stream) reuses the same validation without copy-paste. Two-mock split (`FakeAttachmentsRepository` for service tests, `FakeAttachmentsService` for handler tests) mirrors the notes Repository/Service convention. |

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
| **Production wiring** | `cli/manifest_embed.go` embeds the manifest bytes; `cli/app.go` passes them to `domains.SubcommandGroups(core, manifest)`; each domain's `Register(core, manifest)` calls `cliapp.LoadFromManifest` with its group name and a bindings map keyed by `Service.Method`. The `notes attach` REST exception is appended outside the manifest path because cli-manifest/v1 only models `binding.kind=connect-rpc`. |
| **Test fake** | `cli-core/cliapp::RequireProtoServiceCoverage(t, manifest, fd, serviceName)` asserts every RPC on the bound proto service has either a binding or an entry in the manifest's `omitted` array — see `cli/domains/notes/notes_manifest_test.go`. `cliapp.ParseManifest` covers structural validation in isolation. |
| **Why it exists** | Without this seam, adding a new RPC to the proto compiles fine while the CLI silently lacks a corresponding command, and prompt-manager has no governance signal — every action falls back to `CertaintyOwnerOnly` and is rejected. The manifest crystallises both the command surface and the safety properties (effect, run_eligible, permissions, requires_confirmation) so the coverage test fails fast and prompt-manager can derive certainty automatically. |

### BlobStore (opaque bytes)

| | |
|---|---|
| **Seam** | Binary object storage for REST multipart edges |
| **Interface** | `api-core/blobstore::BlobStore` (`Put`, `Get`, `Delete`) |
| **Production wiring** | A domain module that exposes multipart endpoints owns its blob store. The notes reference resolves filesystem-backed storage in `handlers/notes/module.go::defaultBlobStore()`; tests inject `blobstore.NewMemoryBlobStore()` through `ModuleWithBlobStore(...)`. |
| **Test fake** | `api-core/blobstore.MemoryBlobStore` or a domain-local fake lets handler tests assert metadata and failure behavior without touching the filesystem. |
| **Why it exists** | Connect-RPC is the default for proto-typed payloads, but opaque bytes are not proto payloads. Keeping bytes behind `BlobStore` lets the handler stay transport-focused and lets future scenarios swap filesystem, S3, or another object store without changing domain services. |

### module.Module (domain composition)

| | |
|---|---|
| **Seam** | Domain-to-server composition; the contract every handler package returns from its `Module(...)` constructor. |
| **Interface** | `internal/module/module.go::Module` (`Name string`, `Mount func(r *mux.Router)`, `Endpoints []EndpointDescriptor`). Data type, not behaviour — modules don't have methods. |
| **Production wiring** | `main.go` calls `healthH.Module(...)`, `notesH.Module(...)`, ..., and passes the slice to `server.New(deps, modules...)`. The server iterates `m.Mount(s.router)` after registering the logging middleware. |
| **Test fake** | A literal `module.Module{Name: "stub", Mount: func(r){...}}` in `internal/server/server_test.go` proves the iteration; per-domain `module_test.go` files (`handlers/notes/module_test.go`, `handlers/health/module_test.go`) exercise the real constructors against in-memory fixtures. |
| **Why it exists** | Eliminates the central registry that would otherwise grow per-domain fields on `server.Deps` and per-domain wiring lines in `routes.go`. Adding a domain means creating files; deleting one means removing files. The endpoint descriptors travel with the module, so `.vrooli/endpoints.json` codegen has a single source per domain (no manual JSON editing). |

### Endpoints codegen (manifest source-of-truth)

| | |
|---|---|
| **Seam** | The `.vrooli/endpoints.json` API documentation manifest. |
| **Interface** | `api/cmd/gen-endpoints/main.go` reads `internal/modules.AllEndpoints()` — the shared registry that collects each handler's static `Endpoints []module.EndpointDescriptor` slice plus `cli_commands_seed.json`. Output is the canonical envelope at `.vrooli/endpoints.json`. |
| **Production wiring** | Run via `make endpoints`. CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json` so a stale manifest fails the build with an actionable diff. |
| **Test fake** | `api/cmd/gen-endpoints/main_test.go` exercises the codegen with hand-built fixtures and asserts the output is valid JSON with the canonical envelope. `internal/modules/registry_test.go` pins the registry shape (non-empty, stable order). The manifest coverage gate (every Connect endpoint is bound or explicitly omitted in `cli/manifest.json`) has its own unit test. |
| **Why it exists** | Hand-edited endpoints manifests drift from real handlers. The shared `modules` registry means runtime (`main.go`) and codegen (`gen-endpoints`) read endpoints + schema from one place — adding a domain is two registry lines, not separate edits in `main.go` and `gen-endpoints/main.go`. The CI drift check makes "I forgot to regenerate" a build failure, not a stale-doc bug. |

### database.SystemSchema (cross-cutting infrastructure)

| | |
|---|---|
| **Seam** | Cross-cutting database infrastructure (postgres extensions, custom types, cross-domain views) |
| **Interface** | `internal/database/system.go::SystemSchema() string` (consumed via `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists `apidb.SchemaProviderFunc(localdb.SystemSchema)` first; `main.go` passes the slice into `apidb.EnsureSchemas`. |
| **Test fake** | None. The system file ships empty in the template and is verified empty by `internal/database/system_test.go::TestSystemSchema_IsEmpty` (a deliberate tripwire — adding a `CREATE TABLE` here forces a "yes, this is genuinely cross-cutting" decision). |
| **Why it exists** | Some bits don't belong to any one domain — postgres extensions, type definitions, reporting views. Putting them in a domain package would force fictional ownership. The system home is honest: cross-cutting goes here, single-domain bits go in `internal/<dom>/schema.sql`. |

### notes.Schema (per-domain schema)

| | |
|---|---|
| **Seam** | Notes domain SQL contribution |
| **Interface** | `internal/notes/schema.go::Schema() string` (consumed via `handlers/notes/module.go::Schema` re-export, then `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` includes `apidb.SchemaProviderFunc(notesH.Schema)`; applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | `internal/notes/sqlite_test.go::newSchemaDB` uses `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` with the system + notes providers. Repository tests get a fresh table without touching the central registry. |
| **Why it exists** | Domain ownership of the schema. Adding a column lands in the same diff as the Go change. Deleting `internal/notes/` deletes the table definition with it, so removed domains do not leave tables created on boot. The `handlers/notes/module.go::Schema` re-export keeps the registry's import surface narrow — it imports handler packages, not their internal peers. |

### Doer (outbound HTTP)

| | |
|---|---|
| **Seam** | Outbound HTTP request boundary |
| **Interface** | `internal/httpc/doer.go::Doer` (`Do(*http.Request) (*http.Response, error)`) |
| **Production wiring** | Ships unwired in production by intent (no consumer until a real outbound call lands). `*http.Client` satisfies `Doer` directly via the compile-time assertion in `doer.go`; the first scenario to need an outbound call adds the field to `server.Deps` and wires `&http.Client{Timeout: …}` from `main.go`. |
| **Test fake** | `internal/testutil/mocks::FakeDoer` (canned `*http.Response` queue, recorded `*http.Request` log, atomic `Calls` counter). |
| **Why it exists** | Network calls in handler tests would be flaky and slow. Defining the seam *before* the first consumer means the first scenario to call outward doesn't reinvent ad-hoc mocking. Pattern proven in `scenarios/agent-manager/api/internal/promptmanager/client.go`. See `internal/httpc/doer_test.go` for the substitution reference. |

### ProviderClient.SearchCallOptions (query-time override forwarding)

| | |
|---|---|
| **Seam** | Per-call query-time overrides + control token on a provider Search |
| **Interface** | `internal/eval/runner.go::SearchCallOptions{Overrides *aisearch.SearchOverrides; ControlToken string}`, a parameter of `ProviderClient.Search`. |
| **Production wiring** | `handlers/eval/provider_client.go::httpProviderClient.Search` calls `applyOverrideHeaders`, which sets the shared `aisearch.OverridesHeader` / `aisearch.ControlTokenHeader` (`override_transport.go`) when `Overrides` is non-zero. The eval runner passes the zero value (baseline); the Phase 6 sweep fills it per arm (overrides from the factor enumeration, token from `registry.Store.Token`). |
| **Test fake** | `internal/eval/runner_test.go::fakeClient` (signature carries the param, ignores it for baseline runs); header forwarding is pinned in `handlers/eval/provider_client_test.go` via `mocks.FakeDoer`. |
| **Why it exists** | The override channel must travel without touching the provider-owned search body (a generic descriptor-rendered template). Headers carry the cross-cutting concern; the zero value keeps an ordinary eval run header-free so the public search path is untouched. The shared aisearch transport contract stops sender/receiver header drift. |

### registry.Store.Token (control-token lookup)

| | |
|---|---|
| **Seam** | Per-provider control-token read |
| **Interface** | `internal/registry/store.go::Store.Token(ctx, id) (string, error)` (mint/persist lives in `Store.Upsert(ctx, d, presentedToken)`). |
| **Production wiring** | `RegisterProvider` mints on first insert and echoes thereafter; the sweep (Phase 6) reads the token here to present it when issuing token-gated overrides to the provider. Persisted in the `providers.control_token` column. |
| **Test fake** | `handlers/registry/connect_handler_test.go::fakeStore` (token in/out + presented-token capture); SQLite behavior pinned in `internal/registry/store_test.go` (`TestStoreUpsertTokenOwnership`, `TestStoreToken`). |
| **Why it exists** | The control token is the shared secret gating override/reindex/config-write. search-hub is the authoritative minter+store; a provider re-acquires it from the register echo each boot (memory-only), so the token survives a provider restart without provider-side persistence. |

### ResultMapping.measure_field (measure carrier)

| | |
|---|---|
| **Seam** | Per-provider measure carrier on the generic result adapter |
| **Interface** | `registry.proto::ResultMapping.measure_field` (a JSON path) → `internal/providers/adapter.go::decodeMeasureHit`, populating `routing.proto::SearchHit.measure` (`MeasureHit{measure_id, scenario, params, answer, needs, effect, executed_query, confidence}`). |
| **Production wiring** | Only the single registered measures provider (`measures-health`, Phase 4) sets `measure_field` (to `"measure"`); the adapter then decodes the per-item measure object into `SearchHit.measure` for every hit. Every retrieval provider leaves `measure_field` unset, so `SearchHit.measure` stays nil — the carrier is opt-in via the descriptor, no provider-specific code in the adapter or router (the no-conditional-monolith invariant holds). |
| **Test fake** | `internal/providers/adapter_test.go` (`TestMapResultsMeasureCarrier_*`: executed / needs / write-unexecuted / nil-when-absent); end-to-end through the real router in `internal/routing/measure_carrier_test.go` (`referenceMeasureProvider` httptest server + `TestRouter_Carries*`). |
| **Why it exists** | An analytical ("how many / what's the rate / what's next") answer is a *structured* measure resolution, not a document snippet. Carrying it as a typed sub-message (rather than JSON-in-snippet) lets a consumer act on `answer`/`needs`/`effect` while the router stays thin: matching, param extraction, the auto-exec gate, and execution all happen *inside* the measures provider (`packages/measures-go` engine), and search-hub only carries the result. The measure object's keys are the fixed `MeasureHit` contract; the provider emits them, the adapter decodes them. |

### control.URLResolver (provider base-URL resolution for control calls)

| | |
|---|---|
| **Seam** | Call-time resolution of a provider's live base URL for the control plane |
| **Interface** | `internal/control/client.go::URLResolver.ResolveScenarioURL(ctx, scenarioID)` — the same shape the routing/eval domains use. Production: `control.DiscoveryResolver` (wraps `api-core/discovery`, no caching, so a restarted/re-ported provider is always reached). |
| **Production wiring** | The Phase-6 sweep constructs `control.NewClient(control.NewDiscoveryResolver())`; the scenario_id comes from the descriptor's `reindex_endpoint`/`config_endpoint`. |
| **Test fake** | `internal/control/client_test.go::fakeResolver` (canned URL + recorded scenario id). |
| **Why it exists** | Provider ports are lifecycle-allocated and dynamic, so the control client must resolve the live URL per call rather than hardcode it — identical to the public read path. |

### control.ServiceClientFactory (control Connect client)

| | |
|---|---|
| **Seam** | Construction of the generated `SearchControlServiceClient` for a resolved base URL |
| **Interface** | `internal/control/client.go::ServiceClientFactory func(baseURL) controlconnect.SearchControlServiceClient` (override via `control.WithClientFactory`). |
| **Production wiring** | Default builds `controlconnect.NewSearchControlServiceClient(&http.Client{...}, baseURL)` — a generated Connect client for all proto-owned calls (§12). The control token rides as a request FIELD (`control_token`), not a header. |
| **Test fake** | `internal/control/client_test.go::fakeControlClient` (queued errors + recorded requests) proves bounded retry: transient `Unavailable`/`DeadlineExceeded` are retried, permanent `PermissionDenied`/`InvalidArgument`/`NotFound` are returned on the first attempt. |
| **Why it exists** | The registry-side control client is what the sweep uses to drive a provider's reindex + config-write. The factory seam keeps the retry/resolve logic testable without an HTTP server, and isolates the one place the generated client is built. |

### sweep.Deps (the optimizer's seam bundle)

| | |
|---|---|
| **Seam** | The four consumer-declared interfaces the transport-free sweep core (`internal/sweep`) is built over — it imports no HTTP, Connect, or concrete store. |
| **Interface** | `internal/sweep/sweep.go`: `SuiteReader.GetSuite`, `ProviderReader.{Get,Token}`, `ArmRunner.Run(ctx, suite, tag, overrides, token, limit)`, `ConfigController.{WriteConfig,ReindexStatus}` — plus `clock.Clock`, a `Sleep func(time.Duration)`, and a `*rand.Rand` (seeded; deterministic in tests). |
| **Production wiring** | `handlers/eval/module.go`: `SuiteReader`/`ProviderReader` = the SQLite eval + registry stores; `ConfigController` = `internal/control.Client`; `ArmRunner` = `handlers/eval/module.go::armRunner` (adapts the pure `eval.Runner.RunWith` + `eval.Store.AppendRun` — one stored, tagged run per arm). |
| **Test fake** | `internal/sweep/sweep_test.go`: `fakeSuites`, `fakeProviders`, `fakeRunner` (produces a run from a per-tag outcome map), `fakeControl` (tracks live config, triggers reindex on an index-time change, returns a configurable poll state). They prove each overfit guard blocks promotion independently, write-back gating (`apply`), and the index-time coordinate-ascent (visits the right arms, polls to terminal, restores the incumbent). |
| **Why it exists** | The sweep is the system's optimization authority; its value is the decision logic (enumeration + the four guards + selection), which must be unit-testable without a network, a real reindex, or a live index. The seams keep that core pure (boundary-of-responsibility) and let the handler compose it with the real runner/control client. |

### eval.Runner.RunWith (per-arm override execution)

| | |
|---|---|
| **Seam** | Re-running a suite under explicit per-call `SearchCallOptions` (the arm's query-time overrides + control token) instead of the baseline path. |
| **Interface** | `internal/eval/runner.go::Runner.RunWith(ctx, suite, tag, limit, opts)`; `Run` delegates to it with the zero `opts` (exactly the baseline). |
| **Production wiring** | The sweep's `armRunner` adapter (above) calls `RunWith` with the arm's overrides, then persists. |
| **Test fake** | Covered by `internal/eval/runner_test.go` (override forwarding) and the sweep fakes. |
| **Why it exists** | The sweep must vary query-time factors per arm through the SAME execution + labeling path the baseline uses, so an arm's result is comparable to the incumbent's; `RunWith` is the one method that threads overrides without forking the runner. Case-level corpus scope is carried through `SearchCallOptions.Scope`, so scoped suites (for example doc path/scenario corpora) are not silently flattened during eval runs. |

### eval.Validator (corpus referential validation)

| | |
|---|---|
| **Seam** | Advisory re-probing of positive corpus labels through the registered provider endpoint. |
| **Interface** | `internal/eval/corpus_validate.go::Validator.ValidateCorpus(ctx, suite, deepK)`. |
| **Production wiring** | `handlers/eval/module.go` constructs `eval.NewValidator(registryStore, providerClient)`. The Connect handler exposes it as `EvalService.ValidateCorpus`, and the CLI wraps it as `search-hub evals validate`. |
| **Test fake** | `internal/eval/corpus_validate_test.go` injects fake provider descriptors and a fake provider client returning canned `SearchHit`s / errors. |
| **Why it exists** | Stale-label detection is heuristic under the no-list-all search contract, so the validator is deliberately advisory: deep query probe plus confirm probes classify LIVE/HARD/STALE/INCONCLUSIVE, while provider errors become inconclusive rows rather than failed builds. |

### providers.RenderBodyWithScope (scoped eval calls)

| | |
|---|---|
| **Seam** | Generic request-body placeholder rendering for provider calls that accept scoped queries. |
| **Interface** | `internal/providers/call.go::RenderBodyWithScope(tmpl, query, limit, type, scope)` supports `{{scope}}`, `{{scope_kind}}`, and `{{scope_value}}` in addition to the baseline `{{query}}`/`{{limit}}`/`{{type}}`. |
| **Production wiring** | The eval provider client uses `RenderBodyWithScope`; the router still uses `RenderBody` for unscoped free search. A provider opts in by declaring the placeholders in its `search.json` endpoint body template. |
| **Test fake** | `handlers/eval/provider_client_test.go::TestSearchRendersScopePlaceholders` records the outgoing request body through the fake HTTP doer. |
| **Why it exists** | Scoped test cases belong in the shared corpus contract, not in scenario-local harnesses. Rendering scope generically lets KO-style document suites preserve scenario/path filters without adding provider-specific code to search-hub. |

### corpusgen.Deps (the generator's seam bundle)

| | |
|---|---|
| **Seam** | The three consumer-declared interfaces the transport-free corpus-generation core (`internal/corpusgen`) is built over — it imports no HTTP, no Connect, no store. |
| **Interface** | `internal/corpusgen`: `Sampler.Sample(ctx, target)` (draw index items), `Inverter.{InvertPositive,InvertNegative}(ctx, item)` (the LLM query-inversion seam), `Deduper.IsDuplicate(candidate, seen)` (near-duplicate judgement). |
| **Production wiring** | `handlers/eval/corpusgen.go`: `Sampler` = `corpusSampler` (probes the provider's registered search endpoint via the shared `eval.ProviderClient` — the only enumeration the search contract affords); `Inverter` = `corpusgen.OllamaInverter` (the local gateway); `Deduper` = `corpusgen.JaccardDeduper` (lexical token-overlap — a documented stand-in until search-hub has its own embedder). |
| **Test fake** | `internal/corpusgen/*_test.go`: `fakeSampler` (canned items), `fakeInverter` (id→query map, empties model "failed" inversions), and the real `JaccardDeduper`. They prove the generated marker + anchoring, count cap, dedup vs corpus + self, negatives + floor, stable idempotent case ids, and strata reporting. |
| **Why it exists** | Corpus generation's value is the orchestration (sample → invert → dedup → mark), which must be unit-testable without an LLM, a live index, or a network. The seams keep that core pure (boundary-of-responsibility); the heavy live concern (probe quality) stays at the handler edge. |

### CorpusGenerator (the Generate-handler seam)

| | |
|---|---|
| **Seam** | One-call generation for the connect handler: given a suite + its provider descriptor + options, return de-duped proposals + sampling stats. Hides the per-request sampler construction from the handler. |
| **Interface** | `handlers/eval/corpusgen.go::CorpusGenerator.Generate(ctx, suite, desc, opts)`. |
| **Production wiring** | `handlers/eval/module.go`: `newLiveCorpusGenerator(client)` builds the per-request `corpusSampler` over the same `eval.ProviderClient` the runner/sweep use, plus the Ollama inverter + Jaccard deduper. |
| **Test fake** | `handlers/eval/generate_test.go::fakeGenerator` (canned `corpusgen.Result` / error). Proves preview-vs-apply (no persist on preview, append+upsert on apply), zero-proposal no-op, NotFound/Unregistered/Unimplemented translation, and adequacy surfacing. |
| **Why it exists** | The handler must stay transport-only; the generator owns the (per-suite, per-descriptor) sampler wiring. The seam lets the handler's orchestration (resolve → generate → merge → adequacy → persist) be tested without an LLM or live index. |

### routing.Reranker (unified federation ranking)

| | |
|---|---|
| **Seam** | Cross-provider rerank over the fused provider shortlist. |
| **Interface** | `internal/routing/reranker.go::Reranker` (`Rerank`, `Available`). |
| **Production wiring** | `handlers/routing/module.go` wires `routing.NewDefaultRerankerChain()`, an adapter over `packages/ai-go/search` with TEI cross-encoder primary and Ollama LLM fallback. |
| **Test fake** | `internal/routing/router_test.go::fakeReranker` drives router degradation, timeout, circuit-breaker, and half-open recovery. `internal/routing/reranker_chain_test.go::stubSharedReranker` drives adapter mapping and shared-chain preference/fallback. |
| **Why it exists** | Search Hub owns how heterogeneous provider hits are fused into its proto `SearchHit` shape, but TEI/Ollama client logic belongs in the shared AI search package. The seam keeps query failure semantics local: rerank failures degrade to by-provider grouping instead of failing the query. |

### ollama.Generate / Available (the one gateway to the local LLM)

| | |
|---|---|
| **Seam** | The single transport to the shared Ollama daemon for Search Hub-local LLM callers. Not substituted directly (callers seam at their own `generateFn` field); listed here because it is the one place the `resource-ollama gateway` shell + envelope-unwrap + think-strip lives, reused by the classifier and corpus inverter. |
| **Interface** | `internal/ollama`: `Generate(ctx, model, prompt, maxTokens)`, `Available(ctx)`, `UnwrapResponse`, `StripThink`, `ExtractJSONObject`. |
| **Production wiring** | `routing.NewOllamaClassifier` and `corpusgen.NewOllamaInverter` default their `generate` field to `ollama.Generate`. Production rerank uses the shared `ai-go/search` chain; the older local `OllamaReranker` remains covered by tests but is no longer the routing module's default. |
| **Test fake** | Each caller injects a deterministic `generateFn`; `internal/ollama/gateway_test.go` covers the envelope/think/JSON helpers directly. |
| **Why it exists** | Three LLM callers were each shelling the gateway + unwrapping its envelope. Extracting it removes the duplication (utils-unification) and gives one governed entry point to the throttled daemon. |

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

When a seam belongs to a domain (notes, tasks, users, …), it lives in
`internal/<domain>/`, NOT in `internal/database/` or
`internal/services/`. The notes package is the canonical example — copy
its layout:

```
internal/notes/
  types.go         # Note, CreateInput, ErrInvalidNote, ErrNoteNotFound
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
| **Seam** | UI ↔ API per-domain endpoints (canonical CRUD reference: `api/notes.ts`) |
| **Module** | `ui/src/api/notes.ts` exports `notesClient = createClient(NotesService, transport)` and `uploadAttachment(...)` for the multipart REST exception. |
| **Production wiring** | Feature components wire generated client methods through `useQuery` / `useMutation`, for example `notesClient.listNotes({})` and `notesClient.createNote({ title, body })`. Multipart flows call `uploadAttachment`, which uses `FormData` plus `uploadFile()` and returns generated metadata. |
| **Test fake** | Component tests use inline `vi.mock("./api/notes", async (importOriginal) => ...)` and replace `notesClient` methods or `uploadAttachment`. Factories build generated proto types, including `Timestamp` values. |
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

## Invariants — eval corpus is a file mirror (Tier-2, cross-cutting)

search-hub's eval store is a CACHE of each provider's `search.json` `tests` block,
never an independent authority. Each is tagged `// INVARIANT: <name>` at its site.

1. **`corpusMutationsGoThroughFile`.** A corpus mutation (`evals generate --apply`
   or `evals promote`)
   is written to the owning scenario's `search.json` via the token-gated
   `WriteCorpus` control RPC, then re-registered into the eval store — never
   upserted into the store directly. The file stays authoritative; no reverse
   drift is possible. A provider that declares no control endpoint can PREVIEW but
   not `--apply`. *Enforced:* `api/handlers/eval/generate.go` (`applyCorpus`);
   `generate_test.go` (`TestGenerateApplyWritesFileThenMirrorsStore`,
   `…NoControlPlaneIsPrecondition`).
2. **Tuning cache refresh is immediate, not reboot-gated.** After a sweep
   write-back (`WriteConfig`), the registry cache is re-upserted with the freshly
   written tuning so `ListProviders`/`Get` reflect the SSOT without waiting for the
   provider's next boot. *Enforced:* `api/internal/sweep/sweep.go`
   (`refreshTuningCache`); `sweep_test.go` (`TestSweep_WriteBack_RefreshesTuningCache`).
3. **Shipped eval seeds are a shrinking legacy.** A scenario that owns a
   `search.json` `tests` block self-registers its corpus at boot and ships no seed
   (knowledge-observatory graduated this way). The embedded seeds in
   `api/internal/eval/seeds/` persist ONLY for providers not yet self-registering
   (swarm-manager.records, ui-health.surfaces) and are deleted as each adopts.

## Cross-references

- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
