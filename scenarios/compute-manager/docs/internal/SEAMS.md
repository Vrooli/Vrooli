# Seams — Compute Manager

> **Status: partially implemented.** Provider, metering, bridge, clock,
> storage, reconciliation and provisioning seams have executable fakes and
> focused tests. The remaining specified seams correspond to post-launch or
> provider-live behavior and are still design guidance.

A **seam** is a deliberate boundary where production code calls an
interface, not a concrete dependency. The fake substitutes through that
interface in tests; production wires the real implementation once at the
composition edge (`main.go` for cross-cutting dependencies, or the
domain's `handlers/<domain>/module.go` for domain-local dependencies).

This document is the authoritative **register** of seams in this
scenario: every boundary that exists, plus every boundary the design
commits to before the code lands. It is not a list of things that are
built. Each row carries its state, so "declared here" never reads as
"present in the tree." Add to it whenever you introduce a new interface
that production wires once and tests substitute, and change a row's
state when the code arrives rather than writing a second row. Remove
from it only when the seam is genuinely gone, not when "we don't fake it
yet."

The domain each seam belongs to is defined in
[`DOMAINS.md`](../concepts/DOMAINS.md). A seam that does not obviously
belong to one of the domains listed there is a sign the boundary is in
the wrong place, or that a domain is missing.

## Wire contracts live in proto, not seams

Before adding a new seam, ask: is this a *boundary* (a place where
production-vs-test substitution matters) or a *contract* (a payload
shape consumed by multiple processes)? Wire contracts belong in
`packages/proto/schemas/compute-manager/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/compute-manager/v1/shared/`
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

Every row also carries a state in its first line:

| State | Meaning |
|---|---|
| **specified** | The design commits to this boundary. No code exists. The interface path is where it will live. |
| **template** | The scaffold shipped it. It compiles and has a test, but this scenario has not yet used it for anything of its own. |

## Current seams

The five seams this scenario exists to have come first. Everything after
them is scaffold the template supplied, kept because it is genuinely
reusable and because the scenario seams are built on top of it.

### provider.Adapter

| | |
|---|---|
| **Seam** | Cloud provider control plane (**implemented**) |
| **Interface** | `api/internal/provider/adapter.go::Adapter`, four methods and no more: `Create(ctx, Spec) (Instance, error)`, `Describe(ctx, providerID) (Instance, error)`, `List(ctx) ([]Instance, error)`, `Destroy(ctx, providerID) error`. Each implementation also declares its provider's billing facts as data: rounding behaviour, minimum billable unit, whether a stopped instance bills, and whether inbound traffic counts against the transfer allowance. |
| **Production wiring** | A registry in `api/internal/provider` holds one adapter per configured provider, keyed by identifier. `main.go` registers the adapters the deployment is configured for and hands the registry to the domains that provision and reconcile. Callers select by identifier and no caller names a provider, which is what makes `OT-P1-004` a code-free change for a second provider. Hetzner Cloud is the first adapter, chosen on contract terms rather than API quality. |
| **Test fake** | `api/internal/provider/mocks::FakeAdapter`: an in-memory inventory with a per-method error knob, a settable create latency, and a lever for the specific failure the whole design is shaped around, a create that succeeds while its response is lost. |
| **Why it exists** | This is the most load-bearing seam in the scenario. It is the only boundary between the scenario and something that spends real money on a real account, and everything the scenario claims about ordering, refusal, expiry and reconciliation is claimed about calls through it. Two properties are enforced structurally rather than by review: the interface has exactly four methods, so a pause cannot be added by accident (`DECISIONS.md`, 2026-09-03, records the four-method decision, and a structural test asserts no fifth method exists), and no domain or handler package references a provider by name. The fake is also what buys the whole first wave of work: the reserve, provision and settle spine is built and measured against it before any API key exists, which `PERFORMANCE.md` describes as wave one of measurement. If this seam is skipped, every test in the scenario needs a cloud account and a budget. |

### Bridge onboarding client

| | |
|---|---|
| **Seam** | Enrollment delegation to `vrooli-bridge` (**implemented**) |
| **Interface** | `api/internal/enroll/bridge.go::Onboarder`: fetch the bridge onboarding public key, create the bridge Machine record with the instance address as a locator, and start onboarding. That is the whole surface. |
| **Production wiring** | `main.go` constructs a generated Connect client against the configured bridge API and injects it into the enroll domain's module. The key is served from the domain's cache rather than fetched on the provisioning path, because it has to be embedded in the create call and cannot be retrofitted afterwards. With a warm cache an unreachable bridge is a degrade: the instance is created, metered and expiring, and its enrollment is queued and visibly flagged. With a cold cache it is a refusal, because an instance created without the key in its first-boot configuration can never be enrolled by the unattended path. |
| **Test fake** | `api/internal/enroll/mocks::FakeOnboarder` records the machine record and locators it was asked to create, returns a fixed onboarding key, and can fail at each of the three steps independently. Cold cache and warm cache are separate levers, because they produce opposite outcomes and both need a test. The assertion that matters most is negative: no password, token or private key appears in any field it received or in any log line the path produced. |
| **Why it exists** | This scenario contains no SSH implementation and never will. First touch, bootstrap and node trust belong to bridge, and the instance trusts the bridge onboarding key from first boot, so no credential crosses any wire during enrollment. Keeping that delegation behind one narrow interface prevents the shortcut of passing an owner password through this scenario. Bridge now publishes the public key through its owner-gated RPC; the remaining gap is provider-live enrollment proof. Build against the fake and do not add a substitute SSH path. |

### Metering and reservation client

| | |
|---|---|
| **Seam** | Credit reservation and settlement against `landing-page-business-suite` (**implemented**) |
| **Interface** | `api/internal/meter/client.go::Reserver`: reserve credit for a request, re-reserve on a heartbeat before the upstream window closes, settle measured usage at teardown, and release a reservation when provisioning fails. |
| **Production wiring** | `main.go` constructs the business suite client from configured endpoint and credential references, and injects it into the meter domain. This scenario stores reservation identifiers and nothing else: no wallet, no plan, no invoice, no balance. |
| **Test fake** | `api/internal/meter/mocks::FakeReserver` returns a held reservation, a refusal that names a ceiling, or a transport error, each independently. Its counters are what the settle-and-release tests assert on, because the property under test is that exactly one terminal outcome is recorded per reservation. |
| **Why it exists** | **This is the one dependency in the scenario that fails closed.** Everything else degrades. When the business suite is unreachable no reservation can be obtained, so provisioning is refused and capacity becomes unavailable. That availability cost was accepted deliberately: refusing is recoverable, and a machine that boots unmetered is cost that grows hourly with no compensating action available afterwards. The seam also carries two upstream defects recorded in [`PROBLEMS.md`](PROBLEMS.md) that the fake must be able to reproduce: the reservation window is hard-coded to ten minutes upstream, which is shorter than an hour of compute and forces heartbeat re-reservation, and an out-of-credit refusal is indistinguishable from a server error in the reference client. Do not copy that client's error handling. See [`ERROR-HANDLING.md`](ERROR-HANDLING.md) for the branch order a refusal requires. |

### Credential resolver

| | |
|---|---|
| **Seam** | Resolve a provider or metering credential by reference at call time (**implemented**) |
| **Interface** | `api/internal/provider/credentials.go::CredentialResolver` (`Resolve(ctx, logicalReference) (value, error)`), modelled directly on Treasury's `instrument.CredentialResolver` at `scenarios/treasury/docs/internal/SEAMS.md:153`. |
| **Production wiring** | `main.go` constructs the repository-standard typed credential client over the Vrooli credential authority and injects the resolver into the provider registry and the meter client. An unavailable authority wires a fail-closed resolver, so a provider call cannot proceed with no credential rather than proceeding with an empty one. |
| **Test fake** | A recording resolver that returns an in-memory value and captures which logical reference was asked for. The assertions that matter are negative and belong in the same test: the resolved value appears in no database column, no response body, no log line, no error message and no command argument. |
| **Why it exists** | A provider token is the ability to create unlimited billable machines and to destroy the fleet, which makes its exfiltration the worst outcome available in this scenario. Resolving by reference at call time is what keeps it out of the four places it would otherwise end up: the environment, argv, the database, and a log line. The environment variable is the path of least resistance the template offers, so the seam has to exist before the first real credential is configured, not after. Treasury's version of this seam also re-resolves only after revalidating live authority; this scenario's equivalent is that a resolution happens per provider call rather than once at boot, so a revoked token stops working at the next call instead of at the next restart. |

### Clock

| | |
|---|---|
| **Seam** | Wall-clock time, and the schedule every cost-safety loop runs on (**implemented**) |
| **Interface** | `internal/clock/clock.go::Clock` (`Now() time.Time`), plus the tickers the expiry sweeper and the reconciler will schedule against. |
| **Production wiring** | `main.go` constructs `clock.System{}` and passes it via `server.Deps`. The expiry sweeper and the reconciler receive the same seam at composition time rather than reaching for `time.Now()` or `time.NewTicker` themselves. |
| **Test fake** | `internal/testutil/mocks::FakeClock` (`Now`, `Advance`, `SetNow`). Sweeper and reconciler tests advance it; they never sleep. |
| **Why it exists** | The template introduced this seam so request-duration log lines are deterministic, and that use still holds: `internal/middleware/logging_test.go::TestLoggingMiddleware_LogsDuration` advances the fake by 150 milliseconds and asserts a bit-for-bit duration string. In this scenario that is the least of it. Expiry, heartbeat re-reservation and reconciliation are all scheduled work whose correctness is defined in units of time, and every one of them costs money when it is late. An expiry test has to prove that an instance one second past its lifetime is destroyed and one second before it is not, which is a `FakeClock.Advance` away through this seam and an unrunnable test without it. The heartbeat tests have the same shape against a ten-minute upstream window. Treat any `time.Now()` or `time.NewTicker` in a domain package as a defect, not a shortcut. |

### Pinger (database reachability)

| | |
|---|---|
| **Seam** | Database reachability probe (**template**) |
| **Interface** | `internal/database/pinger.go::Pinger` (`PingContext(ctx) error`) |
| **Production wiring** | `main.go` opens `*sql.DB` via `database.Connect(...)` against `modernc.org/sqlite` (pure-Go, CGO-clean). `*sql.DB` satisfies `Pinger` directly — no wrapper. |
| **Test fake** | `internal/testutil/mocks::FakePinger` (`PingErr error`, atomic `Calls` counter). |
| **Why it exists** | The `/health` handler probes the database. Without the seam, every handler test would either open the on-disk SQLite file (slow at scale, parallel-test contention) or skip the database branch entirely (untested degradation path). With `FakePinger{PingErr: errors.New("connection refused")}`, the unhealthy branch is one line. See `handlers/health/handler_test.go`. |

### database.SchemaExecer (shared schema application)

| | |
|---|---|
| **Seam** | Shared api-core schema execution surface (**template**) |
| **Interface** | `api-core/database::SchemaExecer` (`ExecContext(ctx, query, args...)`) consumed by `database.EnsureSchemas`. |
| **Production wiring** | `main.go` and sqlite tests pass a real `*sql.DB`; each domain's sqlite helper composes scenario-specific providers (`localdb.SystemSchema`, then each `internal/<domain>.Schema`) before applying them. |
| **Test fake** | `api-core/databasetest::FakeExecer` is the canonical fake when a test needs to assert schema application order or injected execution failures without opening a real database. |
| **Why it exists** | Schema application is shared-package behavior, but each scenario owns its provider list. Keep scenario-specific schema composition local; use `databasetest.FakeExecer` only for tests of code that consumes the shared `SchemaExecer` interface directly. |

### Connect router (proto-typed transport)

| | |
|---|---|
| **Seam** | Generated Connect services mounted on the scenario's existing mux router (**template**) |
| **Interface** | `api-core/connectx::RegisterServices(router, mounts...)`, where each mount is `{Path, Handler}` returned by generated `New<Domain>Handler(...)` |
| **Production wiring** | `handlers/<domain>/module.go` constructs the domain service, passes it to `NewConnectHandler`, then mounts the generated handler with `connectx.RegisterServices`. The server's existing middleware still wraps the handler because Connect is standard `http.Handler`. |
| **Test fake** | `api-core/connectxtest::StartTestServer` is the canonical in-process server harness for handler tests. `connectxtest.NewLogger` is the canonical logger capture helper. Module tests can still mount the module on a mux router and issue real HTTP requests. No hand-written request JSON ribbon is needed in tests. |
| **Why it exists** | The proto service descriptor becomes the single wire contract for UI, CLI, and API. Handler path, method, request type, response type, and Connect error envelope all come from generated code instead of parallel route tables. |

### cliapp RunContext (CLI handler test context)

| | |
|---|---|
| **Seam** | Shared cli-core command handler context (**template**) |
| **Interface** | `cli-core/cliapp::RunContext` plus `ArgSchema` parser inputs. |
| **Production wiring** | The CLI dispatcher builds `RunContext` through cli-core's parser and injects the scenario app, stdout, stderr, and built-in `--json` state. |
| **Test fake** | `cli-core/cliapptest::NewTestRunContext` and `NewTestRunContextFromArgs` are the canonical constructors for tests that drive `RunCtx` handlers directly. |
| **Why it exists** | CLI domain tests should exercise handler behavior without duplicating parser setup or relying on `cliapp`'s inline test exports. The sibling test companion keeps future CLI tests aligned with shared-package test-helper ownership. |

### cli/manifest.json ↔ handlers bindings (CLI command surface)

| | |
|---|---|
| **Seam** | Declarative CLI command surface — single source of truth for groups, commands, args, governance, and proto-method bindings. (**template**) |
| **Interface** | `cli/manifest.json` validated against `.vrooli/schemas/cli-manifest.schema.json` (`cli-manifest/v1`); resolved via `repocontract.ScenarioCLIManifestPath`; consumed by `cliapp.LoadFromManifestPrimitives(raw, groupName, bindings)` where `bindings` is `map["<Service>.<Method>"]cliapp.PrimitiveHandler` (each built with `cliapp.ProtoList`/`ProtoMutation`/`ProtoOperational`). The observed primitive is reconciled against the command's `architecture.primitive` — a mismatch fails fast — so declared L4 maturity is verified, not self-certified. |
| **Production wiring** | `cli/manifest_embed.go` embeds the manifest bytes; `cli/app.go` passes them to `domains.SubcommandGroups(core, manifest)`; each domain's `Register(core, manifest)` calls `cliapp.LoadFromManifestPrimitives` with its group name and a bindings map keyed by `Service.Method`, each value a cli-core primitive (a `<name>Call` + `<name>Report` pair). A multipart-upload REST exception is appended outside the manifest path with a plain `RunCtx` (no primitive evidence — a documented exception) because cli-manifest/v1 only models `binding.kind=connect-rpc`. |
| **Test fake** | `cli-core/cliapp::RequireProtoServiceCoverage(t, manifest, fd, serviceName)` asserts every RPC on the bound proto service has either a binding or an entry in the manifest's `omitted` array — see each domain's `cli/domains/<domain>/<domain>_manifest_test.go`. `cliapp.ParseManifest` covers structural validation in isolation. |
| **Why it exists** | Without this seam, adding a new RPC to the proto compiles fine while the CLI silently lacks a corresponding command, and prompt-manager has no governance signal — every action falls back to `CertaintyOwnerOnly` and is rejected. The manifest crystallises both the command surface and the safety properties (effect, run_eligible, permissions, requires_confirmation) so the coverage test fails fast and prompt-manager can derive certainty automatically. |

### BlobStore (opaque bytes)

| | |
|---|---|
| **Seam** | Binary object storage for REST multipart edges (**template**) |
| **Interface** | `api-core/blobstore::BlobStore` (`Put`, `Get`, `Delete`) |
| **Production wiring** | A domain module that exposes multipart endpoints owns its blob store. A domain resolves filesystem-backed storage in its `handlers/<domain>/module.go::defaultBlobStore()`; tests inject `blobstore.NewMemoryBlobStore()` through `ModuleWithBlobStore(...)`. |
| **Test fake** | `api-core/blobstore.MemoryBlobStore` or a domain-local fake lets handler tests assert metadata and failure behavior without touching the filesystem. |
| **Why it exists** | Connect-RPC is the default for proto-typed payloads, but opaque bytes are not proto payloads. Keeping bytes behind `BlobStore` lets the handler stay transport-focused and lets future scenarios swap filesystem, S3, or another object store without changing domain services. |

### module.Module (domain composition)

| | |
|---|---|
| **Seam** | Domain-to-server composition; the contract every handler package returns from its `Module(...)` constructor. (**template**) |
| **Interface** | `internal/module/module.go::Module` (`Name string`, `Mount func(r *mux.Router)`, `Endpoints []EndpointDescriptor`). Data type, not behaviour — modules don't have methods. |
| **Production wiring** | `main.go` calls `healthH.Module(...)`, then each domain's `<domain>H.Module(...)`, and passes the slice to `server.New(deps, modules...)`. The server iterates `m.Mount(s.router)` after registering the logging middleware. |
| **Test fake** | A literal `module.Module{Name: "stub", Mount: func(r){...}}` in `internal/server/server_test.go` proves the iteration; per-domain `module_test.go` files (`handlers/health/module_test.go`, and one per domain) exercise the real constructors against in-memory fixtures. |
| **Why it exists** | Eliminates the central registry that would otherwise grow per-domain fields on `server.Deps` and per-domain wiring lines in `routes.go`. Adding a domain means creating files; deleting one means removing files. The endpoint descriptors travel with the module, so `.vrooli/endpoints.json` codegen has a single source per domain (no manual JSON editing). |

### Endpoints codegen (manifest source-of-truth)

| | |
|---|---|
| **Seam** | The `.vrooli/endpoints.json` API documentation manifest. (**template**) |
| **Interface** | `api/cmd/gen-endpoints/main.go` is a thin wrapper over the shared `github.com/vrooli/api-core/endpoints/gen.Generate`, which renders `internal/modules.AllEndpoints()` — the registry collecting each handler's static `Endpoints []module.EndpointDescriptor` slice — and cross-checks it against `cli/manifest.json` (the CLI-surface SSOT). Output is the canonical envelope at `.vrooli/endpoints.json`. |
| **Production wiring** | Run via `make endpoints`. CI runs `make endpoints && git diff --exit-code .vrooli/endpoints.json` so a stale manifest fails the build with an actionable diff. |
| **Test fake** | The shared `endpoints/gen` package owns the generator's unit tests (transport contract, API↔CLI mapping cross-check, JSON output stability). `internal/modules/registry_test.go` pins the registry shape (non-empty, stable order). The API↔CLI contract — every Connect endpoint is bound to a command or listed in `cli/manifest.json`'s `omitted[]` with a reason — is enforced at `make endpoints` and by the cli-health validation phase. |
| **Why it exists** | Hand-edited endpoints manifests drift from real handlers. The shared `modules` registry means runtime (`main.go`) and codegen (`gen-endpoints`) read endpoints + schema from one place — adding a domain is two registry lines, not separate edits in `main.go` and `gen-endpoints/main.go`. The CI drift check makes "I forgot to regenerate" a build failure, not a stale-doc bug. |

### database.SystemSchema (cross-cutting infrastructure)

| | |
|---|---|
| **Seam** | Cross-cutting database infrastructure (postgres extensions, custom types, cross-domain views) (**template**) |
| **Interface** | `internal/database/system.go::SystemSchema() string` (consumed via `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists `apidb.SchemaProviderFunc(localdb.SystemSchema)` first; `main.go` passes the slice into `apidb.EnsureSchemas`. |
| **Test fake** | None. The system file ships empty in the template and is verified empty by `internal/database/system_test.go::TestSystemSchema_IsEmpty` (a deliberate tripwire — adding a `CREATE TABLE` here forces a "yes, this is genuinely cross-cutting" decision). |
| **Why it exists** | Some bits don't belong to any one domain — postgres extensions, type definitions, reporting views. Putting them in a domain package would force fictional ownership. The system home is honest: cross-cutting goes here, single-domain bits go in `internal/<dom>/schema.sql`. |

### `<domain>`.Schema (per-domain schema)

| | |
|---|---|
| **Seam** | Per-domain SQL contribution (**template**) |
| **Interface** | `internal/<domain>/schema.go::Schema() string` (consumed via `handlers/<domain>/module.go::Schema` re-export, then `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` includes `apidb.SchemaProviderFunc(<domain>H.Schema)`; applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | A domain's `internal/<domain>/sqlite_test.go::newSchemaDB` uses `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` with the system + that domain's providers. Repository tests get a fresh table without touching the central registry. |
| **Why it exists** | Domain ownership of the schema. Adding a column lands in the same diff as the Go change. Deleting `internal/<domain>/` deletes the table definition with it, so removed domains do not leave tables created on boot. The `handlers/<domain>/module.go::Schema` re-export keeps the registry's import surface narrow — it imports handler packages, not their internal peers. |

### Doer (outbound HTTP)

| | |
|---|---|
| **Seam** | Outbound HTTP request boundary (**template**) |
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
| A fifth provider method | A stop, snapshot or reset is the obvious product affordance and a customer will ask for it. A stopped instance still bills at full rate on most surveyed providers, so the affordance costs full price for no value. | The adapter has four methods. Destroy is the only stop. | A structural test asserts no fifth method and no `stop`, `pause`, `suspend`, `halt` or `shutdown` function anywhere in the scenario, per `OT-P0-007`. |
| A provider named in a caller | The first adapter is the only adapter, so naming it reads as harmless. | Callers select by identifier through the registry. | A structural test asserts no domain or handler package references a provider by name, per `OT-P1-004`. |
| A domain reaching for `time.Now()` | Sweepers and heartbeats are easier to write against the wall clock, and the seam only pays off in tests. | Every scheduled loop takes the Clock seam. | An expiry or heartbeat test that has to sleep is the symptom. Treat it as a defect in the production code, not the test. |
| A credential read from the environment | The template offers an environment variable as the path of least resistance, and the first real token will arrive under time pressure. | Provider and metering credentials resolve by reference at call time through the credential resolver seam. | The negative assertions in the resolver's tests, plus the checks named in [`SECURITY.md`](SECURITY.md). |
| The reconciler acquiring a fix | Reporting feels incomplete when the correct action is obvious from the finding. | The reconciler reports and never resolves, and never writes money. Findings are drained by the domain that owns the write. | A sweep test asserts the sweep mutated no instance and settled no reservation. |

### Domain-scoped packages, not generic `services/`

When a seam belongs to a domain (intent, instance, provider, meter,
reconcile, expiry, enroll), it lives in
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
   // internal/intent/repository.go
   package intent

   type Repository interface {
       Create(ctx context.Context, in CreateInput) (Intent, error)
       GetByIdempotencyKey(ctx context.Context, key string) (Intent, error)
   }
   ```
   "No more, no less" is a harder rule here than it reads. The
   provider adapter is four methods because a fifth one would cost
   money, so treat every proposed method as a cost question first and
   an ergonomics question second.
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
| **Seam** | Single Connect-Web transport factory for proto-typed UI API calls (**template**) |
| **Module** | `ui/src/api/client.ts` exports `transport` from `@vrooli/api-base::createScenarioConnectTransport`, plus REST-only helpers (`ApiError`, `decodeApiError`, `makeApiError`, `uploadFile`) for multipart exceptions. |
| **Production wiring** | Every proto-typed domain client imports `transport` and constructs a generated client with `createClient(<Service>, transport)`. REST multipart helpers use `uploadFile()` and parse the metadata response with the generated proto descriptor. |
| **Test fake** | Component tests mock `api/<domain>` modules or typed client methods. REST helper tests stub `global.fetch` directly. Connect behavior is covered at the API boundary by the generated client and focused module tests. |
| **Why it exists** | Per-domain clients should not know URL suffix rules, fetch setup, or proto JSON parse details. Connect-Web centralizes those choices, while the REST helpers make the binary-upload exception explicit instead of becoming a second general transport pattern. |

### `api/<domain>` (per-domain client modules)

| | |
|---|---|
| **Seam** | UI ↔ API per-domain endpoints (**template**) |
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
| **Seam** | Active locale + translation lookup (**template**) |
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
- **The instance lifecycle state machine.** `requested`, `creating`,
  `running`, `draining`, `destroyed`, plus the `orphaned` and `unknown`
  outcomes reconciliation assigns, are domain policy. Production and
  tests call the same transition function. The side effects around it,
  the provider call and the reservation settle, are the seams.
- **A provider's billing facts.** Rounding, minimum billable unit,
  stopped-instance behaviour and inbound-traffic treatment are data an
  adapter declares, not behaviour anything substitutes. They are worth
  reading as facts and worth asserting on, and a fake declares its own
  set so a rounding test does not need a real provider.
- **A provider identifier.** Selecting an adapter by identifier is a
  registry lookup, not a substitution boundary. The seam is the adapter
  interface; the identifier is an argument.

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

## Cross-references

- The domains each seam belongs to: [`DOMAINS.md`](../concepts/DOMAINS.md).
- The order the seams are called in during a provisioning run, and the two orderings that cost money if reversed: [`FLOWS.md`](../concepts/FLOWS.md).
- Why the provider adapter has four methods, why the business suite fails closed, and why there is no SSH code: [`DECISIONS.md`](DECISIONS.md).
- The missing bridge onboarding-key endpoint that blocks the enrollment seam, and the two upstream metering defects the reservation seam has to work around: [`PROBLEMS.md`](PROBLEMS.md).
- What a credential may never touch, which is what the credential resolver seam exists to guarantee: [`SECURITY.md`](SECURITY.md).
- The error taxonomy these seams produce, including which provider failures retry: [`ERROR-HANDLING.md`](ERROR-HANDLING.md).
- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
