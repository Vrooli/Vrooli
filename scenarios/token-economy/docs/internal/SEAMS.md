# Seams — Token Economy

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
`packages/proto/schemas/token-economy/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/token-economy/v1/shared/`
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
| **Production wiring** | `main.go` and sqlite tests pass a real `*sql.DB`; each domain's sqlite helper composes scenario-specific providers (`localdb.SystemSchema`, then each `internal/<domain>.Schema`) before applying them. |
| **Test fake** | `api-core/databasetest::FakeExecer` is the canonical fake when a test needs to assert schema application order or injected execution failures without opening a real database. |
| **Why it exists** | Schema application is shared-package behavior, but each scenario owns its provider list. Keep scenario-specific schema composition local; use `databasetest.FakeExecer` only for tests of code that consumes the shared `SchemaExecer` interface directly. |

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
| **Interface** | `cli/manifest.json` validated against `.vrooli/schemas/cli-manifest.schema.json` (`cli-manifest/v1`); resolved via `repocontract.ScenarioCLIManifestPath`; consumed by `cliapp.LoadFromManifestPrimitives(raw, groupName, bindings)` where `bindings` is `map["<Service>.<Method>"]cliapp.PrimitiveHandler` (each built with `cliapp.ProtoList`/`ProtoMutation`/`ProtoOperational`). The observed primitive is reconciled against the command's `architecture.primitive` — a mismatch fails fast — so declared L4 maturity is verified, not self-certified. |
| **Production wiring** | `cli/manifest_embed.go` embeds the manifest bytes; `cli/app.go` passes them to `domains.SubcommandGroups(core, manifest)`; every one of the seven domain packages calls descriptor-driven `cliapp.ProtoPrimitiveBindings` and selects only its owned `Service.Method` handlers before `cliapp.LoadFromManifestPrimitives`. Mixed-audience groups combine selected generated handlers without sharing clients: holder view/browse/redeem remain `HolderService`, administration remains `MinterService`, and adapter entry remains `EarningService`. |
| **Test fake** | `cli/manifest_test.go` applies `cliapp.RequireProtoServiceCoverage` to all three generated services and asserts every read uses the JSON-capable `proto_list` primitive. Each `cli/domains/<domain>/register_test.go` builds its real group with a nil transport core (construction executes no RPC) and verifies every command carries primitive evidence. `cli/domains/domains_test.go` additionally proves the `mints` group contains no holder-service binding. |
| **Why it exists** | Without this seam, adding a new RPC to the proto compiles fine while the CLI silently lacks a corresponding command, and prompt-manager has no governance signal — every action falls back to `CertaintyOwnerOnly` and is rejected. The manifest crystallises both the command surface and the safety properties (effect, run_eligible, permissions, requires_confirmation) so the coverage test fails fast and prompt-manager can derive certainty automatically. |

### BlobStore (opaque bytes)

| | |
|---|---|
| **Seam** | Binary object storage for REST multipart edges |
| **Interface** | `api-core/blobstore::BlobStore` (`Put`, `Get`, `Delete`) |
| **Production wiring** | A domain module that exposes multipart endpoints owns its blob store. A domain resolves filesystem-backed storage in its `handlers/<domain>/module.go::defaultBlobStore()`; tests inject `blobstore.NewMemoryBlobStore()` through `ModuleWithBlobStore(...)`. |
| **Test fake** | `api-core/blobstore.MemoryBlobStore` or a domain-local fake lets handler tests assert metadata and failure behavior without touching the filesystem. |
| **Why it exists** | Connect-RPC is the default for proto-typed payloads, but opaque bytes are not proto payloads. Keeping bytes behind `BlobStore` lets the handler stay transport-focused and lets future scenarios swap filesystem, S3, or another object store without changing domain services. |

### module.Module (domain composition)

| | |
|---|---|
| **Seam** | Domain-to-server composition; the contract every handler package returns from its `Module(...)` constructor. |
| **Interface** | `internal/module/module.go::Module` (`Name string`, `Mount func(r *mux.Router)`, `Endpoints []EndpointDescriptor`). Data type, not behaviour — modules don't have methods. |
| **Production wiring** | `main.go` calls `healthH.Module(...)`, then each domain's `<domain>H.Module(...)`, and passes the slice to `server.New(deps, modules...)`. The server iterates `m.Mount(s.router)` after registering the logging middleware. |
| **Test fake** | A literal `module.Module{Name: "stub", Mount: func(r){...}}` in `internal/server/server_test.go` proves the iteration; per-domain `module_test.go` files (`handlers/health/module_test.go`, and one per domain) exercise the real constructors against in-memory fixtures. |
| **Why it exists** | Eliminates the central registry that would otherwise grow per-domain fields on `server.Deps` and per-domain wiring lines in `routes.go`. Adding a domain means creating files; deleting one means removing files. The endpoint descriptors travel with the module, so `.vrooli/endpoints.json` codegen has a single source per domain (no manual JSON editing). |

### Endpoints codegen (manifest source-of-truth)

| | |
|---|---|
| **Seam** | The `.vrooli/endpoints.json` API documentation manifest. |
| **Interface** | `api/cmd/gen-endpoints/main.go` is a thin wrapper over the shared `github.com/vrooli/api-core/endpoints/gen.Generate`, which renders `internal/modules.AllEndpoints()` — the registry collecting each handler's static `Endpoints []module.EndpointDescriptor` slice — and cross-checks it against `cli/manifest.json` (the CLI-surface SSOT). Output is the canonical envelope at `.vrooli/endpoints.json`. |
| **Production wiring** | Run via `make endpoints`. CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json` so a stale manifest fails the build with an actionable diff. |
| **Test fake** | The shared `endpoints/gen` package owns the generator's unit tests (transport contract, API↔CLI mapping cross-check, JSON output stability). `internal/modules/registry_test.go` pins the registry shape (non-empty, stable order). The API↔CLI contract — every Connect endpoint is bound to a command or listed in `cli/manifest.json`'s `omitted[]` with a reason — is enforced at `make endpoints` and by the cli-health validation phase. |
| **Why it exists** | Hand-edited endpoints manifests drift from real handlers. The shared `modules` registry means runtime (`main.go`) and codegen (`gen-endpoints`) read endpoints + schema from one place — adding a domain is two registry lines, not separate edits in `main.go` and `gen-endpoints/main.go`. The CI drift check makes "I forgot to regenerate" a build failure, not a stale-doc bug. |

### database.SystemSchema (cross-cutting infrastructure)

| | |
|---|---|
| **Seam** | Cross-cutting database infrastructure (postgres extensions, custom types, cross-domain views) |
| **Interface** | `internal/database/system.go::SystemSchema() string` (consumed via `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists `apidb.SchemaProviderFunc(localdb.SystemSchema)` first; `main.go` passes the slice into `apidb.EnsureSchemas`. |
| **Test fake** | None. The system file ships empty in the template and is verified empty by `internal/database/system_test.go::TestSystemSchema_IsEmpty` (a deliberate tripwire — adding a `CREATE TABLE` here forces a "yes, this is genuinely cross-cutting" decision). |
| **Why it exists** | Some bits don't belong to any one domain — postgres extensions, type definitions, reporting views. Putting them in a domain package would force fictional ownership. The system home is honest: cross-cutting goes here, single-domain bits go in `internal/<dom>/schema.sql`. |

### `<domain>`.Schema (per-domain schema)

| | |
|---|---|
| **Seam** | Per-domain SQL contribution |
| **Interface** | `internal/<domain>/schema.go::Schema() string` (consumed via `handlers/<domain>/module.go::Schema` re-export, then `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` includes `apidb.SchemaProviderFunc(<domain>H.Schema)`; applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | A domain's `internal/<domain>/sqlite_test.go::newSchemaDB` uses `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` with the system + that domain's providers. Repository tests get a fresh table without touching the central registry. |
| **Why it exists** | Domain ownership of the schema. Adding a column lands in the same diff as the Go change. Deleting `internal/<domain>/` deletes the table definition with it, so removed domains do not leave tables created on boot. The `handlers/<domain>/module.go::Schema` re-export keeps the registry's import surface narrow — it imports handler packages, not their internal peers. |

### Doer (outbound HTTP)

| | |
|---|---|
| **Seam** | Outbound HTTP request boundary |
| **Interface** | `internal/httpc/doer.go::Doer` (`Do(*http.Request) (*http.Response, error)`) |
| **Production wiring** | Ships unwired in production by intent (no consumer until a real outbound call lands). `*http.Client` satisfies `Doer` directly via the compile-time assertion in `doer.go`; the first scenario to need an outbound call adds the field to `server.Deps` and wires `&http.Client{Timeout: …}` from `main.go`. |
| **Test fake** | `internal/testutil/mocks::FakeDoer` (canned `*http.Response` queue, recorded `*http.Request` log, atomic `Calls` counter). |
| **Why it exists** | Network calls in handler tests would be flaky and slow. Defining the seam *before* the first consumer means the first scenario to call outward doesn't reinvent ad-hoc mocking. Pattern proven in `scenarios/agent-manager/api/internal/promptmanager/client.go`. See `internal/httpc/doer_test.go` for the substitution reference. |

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

When a seam belongs to a domain (tasks, users, …), it lives in
`internal/<domain>/`, NOT in `internal/database/` or
`internal/services/`. Copy this layout for each domain package:

```
internal/<domain>/
  types.go         # Entity, CreateInput, ErrInvalid<Entity>, Err<Entity>NotFound
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
| **Seam** | UI ↔ API per-domain endpoints |
| **Module** | `ui/src/api/<domain>.ts` exports `<domain>Client = createClient(<Service>, transport)` plus any multipart-upload helper for the REST exception. |
| **Production wiring** | Feature components wire generated client methods through `useQuery` / `useMutation`, for example `<domain>Client.list<Entity>s({})` and `<domain>Client.create<Entity>({ ... })`. Multipart flows call the upload helper, which uses `FormData` plus `uploadFile()` and returns generated metadata. |
| **Test fake** | Component tests use inline `vi.mock("./api/<domain>", async (importOriginal) => ...)` and replace client methods or the upload helper. Factories build generated proto types, including `Timestamp` values. |
| **Why it exists** | The per-domain client pattern. Mirror this shape for each domain client: export the generated Connect client, keep binary-upload helpers beside it when needed, and let components consume typed results rather than hand-written response interfaces. |

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
3. If the operation has a CLI mirror, bind it (or list it in `omitted[]`)
   in `cli/manifest.json`.
4. Run `make endpoints` to regenerate `.vrooli/endpoints.json`.
5. Commit both the descriptor edit AND the regenerated manifest.

The CI drift check (`make endpoints && git diff --exit-code
.vrooli/endpoints.json`) fails the build if step 4 was skipped, with
an actionable diff showing exactly which entries diverged.

## Planned scenario seams

These do not exist yet — no product code has been written. They are recorded
here during initialization because each one guards a P0 property, and deciding
them after handlers are written is how a scenario ends up with untestable
correctness claims.

### JournalAppender

| | |
|---|---|
| **Seam** | The only path that writes a journal event |
| **Interface** | `internal/journal/appender.go::Appender` (append-only; **no** update or delete method, deliberately) |
| **Production wiring** | Constructed once in API composition over the routed storage handle; injected into `grants` and `redemption` services. |
| **Test fake** | `internal/journal/mocks::FakeAppender` recording appended events in order. |
| **Why it exists** | Two reasons, both structural. First, it makes `TKE-P0-010` mechanically checkable: a test can assert the interface has no mutating method, so "append-only" is enforced by the type rather than by reviewer discipline. Second, it is the choke point where `TKE-P0-011` provenance is stamped — one construction site means an event cannot be written without an actor. |

### BalanceProjector

| | |
|---|---|
| **Seam** | Balance derivation from events |
| **Interface** | `internal/journal/projection.go::Projector` (`BalanceAt`, `Rebuild`) |
| **Production wiring** | Cache-backed implementation constructed in API composition. |
| **Test fake** | `internal/journal/mocks::FakeProjector`, plus a real replay-only implementation used to assert cache agreement. |
| **Why it exists** | `TKE-P0-004` requires that a cached balance always equals a full replay. That equality is only testable if replay-only and cache-backed implementations satisfy one interface and can be compared directly. Without the seam, the equality test would have to reach into cache internals and would rot the first time the cache changed shape. |

### RuleEvaluator

| | |
|---|---|
| **Seam** | Server-side grant-rule evaluation |
| **Interface** | `internal/grants/rules.go::Evaluator` (`Evaluate(ctx, grant, request) (Decision, error)`) |
| **Production wiring** | Constructed from the closed condition vocabulary; injected into `redemption`. |
| **Test fake** | `internal/grants/mocks::FakeEvaluator` returning scripted decisions. |
| **Why it exists** | `TKE-P0-003` says the decision never leaves the server, and refusals must name the rule that refused. Isolating evaluation lets `redemption` be tested against every decision shape without constructing real grants, and lets rule evaluation be tested exhaustively without a transport. A `Decision` carrying a typed reason is what makes the holder-facing refusal copy possible. |

### HolderScope

| | |
|---|---|
| **Seam** | The isolation boundary |
| **Interface** | `internal/holders/scope.go::Scope` — a required parameter on every holder-facing repository read |
| **Production wiring** | Derived from verified authentication claims at the handler edge; never constructed from request data. |
| **Test fake** | Constructed directly in tests for a given holder id. |
| **Why it exists** | `TKE-P0-006` requires isolation at the repository layer, not only the handler, so a future handler written without the check still cannot leak. Making the scope a **required parameter** rather than an ambient filter means omitting it is a compile error rather than a silent full-table read. This is the single most important seam in the scenario. |

### IdempotencyStore

| | |
|---|---|
| **Seam** | Caller-supplied idempotency keys |
| **Interface** | `internal/redemption/idempotency.go::Store` (`Claim(ctx, key) (priorResult, claimed, error)`) |
| **Production wiring** | Row-locked SQLite implementation sharing the settlement transaction. |
| **Test fake** | In-memory fake, plus a failure-injecting variant. |
| **Why it exists** | `TKE-P0-009` must be tested against **induced failure between debit and event write**, not merely happy-path retry. That test is impossible without a seam that can be told to fail at a chosen point. The failure it guards against — a debit without its event — has no repair verb in an append-only store, which is exactly why it is tested rather than assumed. |

### EarningAdapterAuth

| | |
|---|---|
| **Seam** | Adapter authentication and dedup |
| **Interface** | `internal/earning/adapter.go::AdapterAuthenticator` |
| **Production wiring** | Constructed in API composition; operator entry satisfies the same interface as any programmatic adapter. |
| **Test fake** | `internal/earning/mocks::FakeAdapterAuthenticator`. |
| **Why it exists** | `TKE-P0-007` requires that no satisfier is privileged. A shared seam makes "operator entry and programmatic adapters traverse identical code" an assertion a test can make, rather than a claim a reviewer has to verify by reading two call paths. |

### NotificationRelay

| | |
|---|---|
| **Seam** | Out-of-band approval relay |
| **Interface** | `internal/redemption/relay.go::Relay` (`Notify(ctx, request) error`) |
| **Production wiring** | `notification-hub` client when the scenario resolves; a no-op implementation otherwise. |
| **Test fake** | Recording fake, plus an always-failing variant. |
| **Why it exists** | `TKE-P0-013` requires that approval works with `notification-hub` unavailable. The no-op implementation is the production wiring for the common case — absence is the baseline, not degraded mode — and the always-failing fake proves a relay error never blocks a decision. |

### ProvenanceResolver

| | |
|---|---|
| **Seam** | Actor identity verification |
| **Interface** | `internal/journal/provenance.go::ProvenanceResolver` (`Resolve(ctx) Actor`) |
| **Production wiring** | `main.go` wraps the API with `api-core/provenance.Middleware(provenance.CLIUtilVerifier{})`; `journal.ContextProvenanceResolver` reads that shared result at the single transactional append point. Verified agent subject/profile and run id win; otherwise the authenticated product subject is retained as the operator identity. |
| **Test fake** | Tests place controlled shared provenance results in context and cover each of `verified`, `unavailable`, `invalid`, and `absent`. |
| **Why it exists** | `TKE-P0-011` requires all four ecosystem statuses to be stored correctly, including verifier outage and malformed claims. The context seam keeps those cases deterministic while ensuring production uses `cliutil.IdentityEnv.VerifyIdentity` through the shared verifier instead of inventing a second attribution model. |

## Cross-references

- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
