# Seams — {{SCENARIO_DISPLAY_NAME}}

A **seam** is a deliberate boundary where production code calls an
interface, not a concrete dependency. The fake substitutes through that
interface in tests; production wires the real implementation in `main.go`.

This document is the authoritative list of seams in this scenario. Add
to it whenever you introduce a new interface that production wires once
and tests substitute. Remove from it only when the seam is genuinely
gone — not when "we don't fake it yet."

## Wire contracts live in proto, not seams

Before adding a new seam, ask: is this a *boundary* (a place where
production-vs-test substitution matters) or a *contract* (a payload
shape consumed by multiple processes)? Wire contracts belong in
`packages/proto/schemas/{{SCENARIO_ID}}/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/{{SCENARIO_ID}}/v1/health/`
is the worked example. The Go fixture (`api/internal/testutil/fixtures/health.go`)
re-exports the generated `Response` and provides functional-options
builders; the UI factory (`ui/src/test-utils/factories.ts`) builds
the same generated type via `create(ResponseSchema, ...)`. Drift
between the two is impossible because both consume one source of
truth.

If a piece of production code reaches for `time.Now()`, `*sql.DB`, or
the network without going through one of the entries below, that's a
new seam that hasn't been declared yet. Declaring it is the work — the
test ergonomics fall out for free.

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

### notes.Repository (notes persistence)

| | |
|---|---|
| **Seam** | Notes persistence (CRUD) |
| **Interface** | `internal/notes/repository.go::Repository` (`Create`, `Get`, `List`) |
| **Production wiring** | `main.go` constructs `notes.NewSQLiteRepository(db, clock.System{})`, then wraps it with `notes.NewService(repo)` (see next row) before passing the service via `server.Deps.NoteService`. Wire shape lives in `packages/proto/schemas/{{SCENARIO_ID}}/v1/notes/notes.proto`. |
| **Test fake** | `internal/notes/mocks::FakeRepository` (co-located with the domain — in-memory slice, per-method error knobs `CreateErr` / `GetErr` / `ListErr`, atomic call counters). Used by `internal/notes/service_test.go` to drive the service against a controllable persistence layer. |
| **Why it exists** | Repository owns the persistence contract — sqlite SQL today, anything else tomorrow. The handler depends on `notes.Service`, not directly on the repository, so a backend swap doesn't ripple through transport. The repository test in `internal/notes/sqlite_test.go` substitutes the real handle to pin SQL semantics (ordering, limit, RFC3339 round-trip). |

### notes.Service (notes application layer)

| | |
|---|---|
| **Seam** | Notes application surface (validation, defaults, cross-handler policy) |
| **Interface** | `internal/notes/service.go::Service` (`Create(CreateInput) → Note`, `Get(id) → Note`, `List(limit) → []Note`) |
| **Production wiring** | `handlers/notes/module.go::Module(db, clk, logger)` constructs `notes.NewSQLiteRepository(db, clk)` then `notes.NewService(repo)` then `NewHandler(Deps{Service: svc, Logger: logger})` — fully internal to the notes module. `main.go` only sees the `module.Module` returned from that constructor; per-domain services don't appear on `server.Deps`. The handler imports `internal/notes` for both the interface and the typed sentinels (`ErrInvalidNote`, `ErrNoteNotFound`) it translates at the transport edge. |
| **Test fake** | `internal/notes/mocks::FakeService` (co-located with the domain — records `CreateInputs`, returns canned `CreateOut` / `GetByID` / `ListOut`, per-method error knobs). Used by `handlers/notes/handler_test.go` to drive the handler without validation/repository plumbing in scope. |
| **Why it exists** | Validation (`title required` after whitespace trim) and default substitution (`defaultListLimit = 100` when caller passes 0) are business policy, not transport policy. Putting them in the service keeps the handler thin and makes the same rules reachable from any future surface (gRPC, batch jobs, scheduled imports) without copy-paste. Two-mock split (`FakeRepository` for service tests, `FakeService` for handler tests) means handler tests don't seed sqlite-shaped state to assert routing. |

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
| **Test fake** | `api/cmd/gen-endpoints/main_test.go` exercises the codegen with hand-built fixtures and asserts the output is valid JSON with the canonical envelope. `internal/modules/registry_test.go` pins the registry shape (non-empty, stable order). The cross-check (every `cli_mapping.command` in `endpoints[]` matches a `cli_commands[].name`) has its own unit test. |
| **Why it exists** | Hand-edited endpoints manifests drift from real handlers. The shared `modules` registry means runtime (`main.go`) and codegen (`gen-endpoints`) read endpoints + schema from one place — adding a domain is two registry lines, not separate edits in `main.go` and `gen-endpoints/main.go`. The CI drift check makes "I forgot to regenerate" a build failure, not a stale-doc bug. |

### database.SystemSchema (cross-cutting infrastructure)

| | |
|---|---|
| **Seam** | Cross-cutting database infrastructure (postgres extensions, custom types, cross-domain views) |
| **Interface** | `internal/database/system.go::SystemSchema() string` (consumed via `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists `apidb.SchemaProviderFunc(localdb.SystemSchema)` first; `main.go` passes the slice into `apidb.EnsureSchemas`. |
| **Test fake** | None. The system file ships empty in the template and is verified empty by `internal/database/system_test.go::TestSystemSchema_IsEmpty` (a deliberate tripwire — adding a `CREATE TABLE` here forces a "yes, this is genuinely cross-cutting" decision). |
| **Why it exists** | Some bits don't belong to any one domain — postgres extensions, type definitions, reporting views. Putting them in a domain package would force fictional ownership; a central `internal/store/schema.sql` was what Pass-3 deleted. The system home is honest: cross-cutting goes here, single-domain bits go in `internal/<dom>/schema.sql`. |

### notes.Schema (per-domain schema)

| | |
|---|---|
| **Seam** | Notes domain SQL contribution |
| **Interface** | `internal/notes/schema.go::Schema() string` (consumed via `handlers/notes/module.go::Schema` re-export, then `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` includes `apidb.SchemaProviderFunc(notesH.Schema)`; applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | `internal/notes/sqlite_test.go::newSchemaDB` uses `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` with the system + notes providers. Repository tests get a fresh table without touching the central registry. |
| **Why it exists** | Domain ownership of the schema. Adding a column lands in the same diff as the Go change. Deleting `internal/notes/` deletes the table definition with it (no orphan table created on boot — the Pass-3 smoking-gun bug). The `handlers/notes/module.go::Schema` re-export keeps the registry's import surface narrow — it imports handler packages, not their internal peers. |

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
`internal/<dom>/` takes the mocks (and the schema, and the tests) along
in one sweep — that's what makes REPLACING-NOTES.md work.

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

### `lib/api` (network boundary)

| | |
|---|---|
| **Seam** | UI ↔ API HTTP boundary |
| **Module** | `ui/src/lib/api.ts` (`fetchHealth` and any future endpoint wrappers) |
| **Production wiring** | `App.tsx` (and any component using React Query) imports `fetchHealth` directly. |
| **Test fake** | Inline `vi.mock("./lib/api", async (importOriginal) => …)` at the top of each test file. The factory closure invokes `makeHealthResponse()` from `@/test-utils` to produce typed proto-shaped responses. |
| **Why it exists** | Network calls in unit tests would be flaky and slow. Mocking at this single module boundary means every component that reads `/health` gets the same substitution. The wrapper is intentionally thin (decode-only) so `vi.mock` doesn't have to reproduce business logic. |

### `lib/notes` (network boundary, CRUD reference)

| | |
|---|---|
| **Seam** | UI ↔ API notes endpoints |
| **Module** | `ui/src/lib/notes.ts` (`listNotes`, `createNote`, `getNote`, `ApiError`) |
| **Production wiring** | `App.tsx` imports `listNotes` / `createNote` directly and wires them through `useQuery` / `useMutation`. |
| **Test fake** | Inline `vi.mock("./lib/notes", async (importOriginal) => …)`; the factory closure uses `makeNote()` / `makeListNotesResponse()` from `@/test-utils`. Unit tests in `lib/notes.test.ts` stub `global.fetch` directly via `vi.stubGlobal`. |
| **Why it exists** | The canonical CRUD wrapper pattern. Decodes proto-typed responses via `fromJson(<Schema>, ...)` and surfaces non-2xx as `ApiError` carrying the typed `ErrorEnvelope.code` — UI branches on `err.code === "not_found"` etc. without parsing strings. Mirror this shape when adding a second domain client (e.g., `lib/tasks.ts`). |

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
`Endpoints []module.EndpointDescriptor` slice (see the
"Endpoints codegen" seam above). To add or change an endpoint:

1. Update the handler.
2. Update the descriptor in `api/handlers/<dom>/endpoints.go` —
   path, method, summary, error codes (must match the `httpx.Code*`
   constants the handler emits), and a `cli_mapping` pointing at
   the real CLI subcommand registered in `cli/domains/`.
3. If the change touches a CLI command, update
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
