# Seams — Vrooli Bridge

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
`packages/proto/schemas/vrooli-bridge/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/vrooli-bridge/v1/health/`
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
| **Interface** | `cli/manifest.json` validated against `.vrooli/schemas/cli-manifest.schema.json` (`cli-manifest/v1`); resolved via `repocontract.ScenarioCLIManifestPath`; consumed by `cliapp.LoadFromManifest(raw, groupName, bindings)` where `bindings` is `map["<Service>.<Method>"]func(RunContext) error`. |
| **Production wiring** | `cli/manifest_embed.go` embeds the manifest bytes; `cli/app.go` passes them to `domains.SubcommandGroups(core, manifest)`; each domain's `Register(core, manifest)` calls `cliapp.LoadFromManifest` with its group name and a bindings map keyed by `Service.Method`. A multipart-upload REST exception is appended outside the manifest path because cli-manifest/v1 only models `binding.kind=connect-rpc`. |
| **Test fake** | `cli-core/cliapp::RequireProtoServiceCoverage(t, manifest, fd, serviceName)` asserts every RPC on the bound proto service has either a binding or an entry in the manifest's `omitted` array — see each domain's `cli/domains/<domain>/<domain>_manifest_test.go`. `cliapp.ParseManifest` covers structural validation in isolation. |
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

### auth.Validator (owner identity)

| | |
|---|---|
| **Seam** | Owner bearer-token validation against scenario-authenticator (the "Owner → control plane" boundary) |
| **Interface** | `internal/auth/auth.go::Validator` (`Validate(ctx, bearerToken) (Identity, error)`) |
| **Production wiring** | `main.go` constructs `auth.NewClient(auth.Config{Resolver: discovery.NewResolver(...)})` and wraps the API handler with `auth.Middleware(client, logger)` (best-effort-inject); owner-gated handlers call `auth.RequireOwner(ctx)` (fail-closed). Verification is offline against the authenticator's published RS256 JWKS, fetched once and cached. |
| **Test fake** | `internal/auth.FakeValidator` (canned Identity/Err); handler tests inject identity directly via `auth.WithIdentity(ctx, Identity{...})`. |
| **Why it exists** | Bridge does not own user identity; it verifies the owner JWT locally so a momentarily-unavailable authenticator never breaks a live session. Trimmed from device-sync-hub's auth (no `RevokeSession` — bridge revokes node credentials, not auth sessions). The alg-lock (RS256 only) rejects "none"/HS* confusion. |

### auth.BreakGlassValidator / auth.BreakGlassAuditor (offline outage path)

| | |
|---|---|
| **Seam** | Explicit offline owner capability verification and accountability |
| **Interface** | `internal/auth/middleware.go::BreakGlassValidator` and `BreakGlassAuditor`; `internal/auth/auth.go::Client.ValidateBreakGlass` |
| **Production wiring** | `main.go` pins the public Ed25519 key supplied by the machine-linking path, accepts only the `BreakGlass` authorization scheme, and appends `audit.ActionBreakGlass` before injecting identity. A missing or failed audit sink refuses the request. |
| **Test fake** | `auth.BreakGlassAuditFunc` and `auth.FakeValidator`; pure credential and scope-ceiling tests live in `api-core/trustposture`. |
| **Why it exists** | Outage recovery must be a positive, signed, scoped capability. Keeping it structurally separate from `Validator.Validate` prevents an authenticator error from becoming an implicit allow path, while the audit seam makes emergency access observable and testable. |

## Decision Points

The major authorization decisions are now grouped in named seams rather than
scattered conditionals:

- `api-core/trustposture.DefaultsFor` chooses operational defaults from the
  declared posture; it never chooses verification behavior.
- `auth.Client.ensureKeys` chooses fresh JWKS, bounded cached-JWKS reuse, or
  refusal based on the posture grace window.
- `auth.MiddlewareWithAudit` chooses normal bearer validation or the explicit
  `BreakGlass` scheme from the authorization scheme name; no validation error
  selects break-glass.
- `api-core/trustposture.ScopeCeiling` chooses whether each emergency scope is
  within the account grant, refusing any widening request.

These decisions have table or boundary tests and are wired once at `main.go`.

### registry.Repository / registry.Service (node persistence + application surface)

| | |
|---|---|
| **Seam** | Durable node-record persistence (Repository) and validation/policy (Service) |
| **Interface** | `internal/registry/repository.go::Repository` (Create/Get/List/Update/Revoke/TouchLastSeen); `internal/registry/service.go::Service` (Register/List/Get/Update/Revoke) |
| **Production wiring** | `handlers/registry/module.go::Module` constructs `registry.NewSQLiteRepository(db, clk)` then `registry.NewService(repo)`, internal to the module. `main.go` also constructs a second SqliteRepository as the channel handler's last-seen recorder (same `nodes` table). |
| **Test fake** | `internal/registry/mocks::FakeRepository` (in-memory map, error knobs, atomic counters) for service tests; `internal/registry/mocks::FakeService` for handler tests; real sqlite in `internal/registry/sqlite_test.go`. |
| **Why it exists** | Domain ownership of the `nodes` table; the handler depends on `Service` so validation has a home and a backend swap doesn't ripple to transport. Two-mock split mirrors the device-sync-hub/notes convention. |

### registry handler Presence (live online/offline overlay)

| | |
|---|---|
| **Seam** | The read-path overlay that stamps live online/offline status onto stored node records |
| **Interface** | `handlers/registry/connect_handler.go::Presence` (`IsOnline(nodeID) bool`) |
| **Production wiring** | `main.go` passes the shared `*presence.Hub` into `registryH.Module`; the hub satisfies `IsOnline` directly. A nil Presence (or the `offlinePresence` default) reads every node offline. |
| **Test fake** | A literal `fakePresence{online: map[string]bool{...}}` in `handlers/registry/connect_handler_test.go`. |
| **Why it exists** | Presence is ephemeral; the registry persists only durable identity. Decoupling the overlay behind a narrow `IsOnline` seam lets the registry domain build and test without depending on the presence hub, and lets a revoked node always read REVOKED regardless of any lingering channel. |

### channel.LastSeenRecorder (heartbeat → last-seen persistence)

| | |
|---|---|
| **Seam** | Persisting a node's last-seen timestamp on heartbeat |
| **Interface** | `handlers/channel/heartbeat_handler.go::LastSeenRecorder` (`TouchLastSeen(ctx, nodeID, t) error`) |
| **Production wiring** | `main.go` passes the registry SqliteRepository (it satisfies `TouchLastSeen`) into `channelH.Module`. A persistence failure is logged and swallowed — the in-memory presence update is authoritative for liveness, so a DB hiccup never drops a heartbeat. |

### Interactive session transport

| Seam | Contract |
|---|---|
| **Wire** | `packages/proto/schemas/vrooli-bridge/v1/session/session.proto`; binary `Frame` messages over `/api/v1/channel/session`. |
| **Policy** | `api/internal/session.Manager` requires `vrooli-bridge:session`, owner re-authentication, sequence continuity, bounded receive window, idle timeout and hard lifetime. |
| **Security** | The ambient owner identity is insufficient. `X-Bridge-Owner-Reauth` is independently validated, WebSocket origins are same-origin checked, and denied opens are audited. |
| **Backend seam** | The WebSocket handler relays opaque bytes. PTY, agent and SSH backends are selected by the next session phase without changing this wire contract. |
| **Test fake** | A `fakeLastSeen` recorder in `handlers/channel/heartbeat_handler_test.go` (records ids; an injectable error proves the swallow path). |
| **Why it exists** | Keeps the channel handler decoupled from the registry's storage internals while still persisting "last seen 2h ago" across a control-plane restart. The presence hub itself stays pure in-memory. |

Note: `internal/presence.Hub` is a concrete shared component (constructed once in `main.go`, shared by the registry overlay and the channel handler), not a substitution seam — the *seam* is the narrow `Presence.IsOnline` interface the registry handler consumes. An optional Redis-backed Hub for scale-out is a declared future seam, not built while the fleet is single-instance. The Hub additionally exposes `Push(nodeID, payload)` (the control-plane → node frame push the SSE handler drains via `Conn.Out()`); like `IsOnline`, dispatch consumes it through the narrow `dispatch.JobPusher` seam, not the concrete Hub.

### runs.Repository / runs.Service (durable run persistence + lifecycle)

| | |
|---|---|
| **Seam** | Durable server-owned run persistence (Repository) and the block-once lifecycle + ingest coordination (Service) |
| **Interface** | `internal/runs/repository.go::Repository` (Create/Get/List/Update/AppendEvent/ListEvents); `internal/runs/service.go::Service` (Create/Get/List/AppendEvent/Wait/Abort/Subscribe) |
| **Production wiring** | `main.go` constructs `runs.NewService(runs.NewSQLiteRepository(db, clk), clk)` ONCE and shares the instance with `runsH.Module` (operator verbs + node-facing ReportRunEvent ingest) and `dispatchH.Module` (CreateRun), so the in-memory block-once waiter registry + live-event subscriber fan-out is one coherent instance across both call sites. |
| **Test fake** | `internal/runs/mocks::FakeRepository` (in-memory, error knobs) for service tests; `internal/runs/mocks::FakeService` for handler tests; real sqlite in `internal/runs/sqlite_test.go`. |
| **Why it exists** | Durability (survive disconnect, re-attach by id, block-once wait) lives in the service's coordinator; the repository is the durable source of truth a re-attaching client reads. Decoupling them lets the lifecycle be unit-tested against an in-memory repo and the persistence be integration-tested against real sqlite. |

### audit.Sink / audit.Reader (append-only accountability substrate)

| | |
|---|---|
| **Seam** | The write side (Sink.Append) and read side (Reader.List) of the append-only audit trail |
| **Interface** | `internal/audit/sink.go::Sink` (`Append(ctx, Record)`) and `::Reader` (`List(ctx, filter)`); the dispatch domain re-declares its own narrow `dispatch.AuditSink` so it imports neither audit nor proto |
| **Production wiring** | `main.go` constructs `audit.NewSQLiteStore(db, clk)` and shares it as the dispatch handler's write Sink and the audit handler's read Reader. A **workspace-sandbox-backed Sink** is the documented alternative behind the same Sink seam (the SECURITY.md accountability substrate), wired when that scenario is green without changing any caller — see PROBLEMS.md. |
| **Test fake** | `internal/audit/mocks::FakeSink` / `::FakeReader`; `internal/audit/sandbox_integration_test.go` proves a substrate is swappable behind the seam; real sqlite in `audit_test.go`. |
| **Why it exists** | Records are written only as a side effect of the operation they audit (dispatch/provision) — there is no proto RPC that writes audit, so there is no wire path to forge or mutate a record. The narrow Sink/Reader split makes the substrate (local SQLite today, workspace-sandbox later) a wiring choice, not a code change. |

### dispatch seams (NodeReader / Presence / RunController / AuditSink / JobPusher)

| | |
|---|---|
| **Seam** | The five outside-world dependencies of the allowlist gate, each declared in `internal/dispatch/seams.go` over dispatch-local DTOs |
| **Interface** | `dispatch.NodeReader` (node scopes/revocation), `dispatch.Presence` (online), `dispatch.RunController` (CreateRun/AbortRun), `dispatch.AuditSink` (Record), `dispatch.JobPusher` (PushJob) |
| **Production wiring** | `handlers/dispatch/adapter.go` is the single translation point binding these to the concrete registry service, presence hub, runs service, audit store, and the channel push (JobPush → ServerFrame → protojson → `Hub.Push`). The dispatch domain itself imports no sibling domain and no proto. |
| **Test fake** | `internal/dispatch/mocks` (one fake per seam, with error/delivery knobs) drives `service_test.go`; the real adapters are exercised end-to-end in `handlers/dispatch/connect_handler_test.go`. |
| **Why it exists** | The allowlist is the highest-stakes decision in the scenario; declaring every dependency as a narrow seam over proto-free DTOs keeps `Allow()` and the dispatch sequence pure and exhaustively table-testable, and keeps the proto/channel translation in one auditable place. |

### node-agent exec seams (EventReporter / CommandRunner)

| | |
|---|---|
| **Seam** | The node-agent runner's two effects: reporting RunEvents back, and executing the local CLI |
| **Interface** | `agent/internal/exec/exec.go::EventReporter` (`Report(ctx, *RunEvent)`) and `::CommandRunner` (`Run(ctx, argv, dir, onLog)`) |
| **Production wiring** | `agent/internal/channel/channel.go` wires `runEventReporter` (a signed `RunsService.ReportRunEvent` call) and the default `osCommandRunner` (`os/exec.CommandContext` over a pre-split argv — never `sh -c`). |
| **Test fake** | `agent/internal/exec/typedjob_test.go` substitutes a collecting reporter + a canned command runner; a real-`os/exec` smoke lives in `command_test.go`. |
| **Why it exists** | Lets the runner's lifecycle (STATUS→LOG→EXIT) and the no-shell-path proof (`BuildArgv` rejects shell metacharacters; the command seam only ever receives a `[]string`) be tested without a real `vrooli` binary or a live control plane. |

### provision seams (NodeReader / Presence / AuditSink / CommandPusher)

| | |
|---|---|
| **Seam** | The privileged provisioning tier's outside-world dependencies, each declared in `internal/provision/seams.go` over provision-local DTOs. Unlike dispatch, the provision domain ALSO owns its durable op tables (its own `Repository` seam), so it is run-lifecycle + orchestration in one domain. |
| **Interface** | `provision.NodeReader` (node revocation), `provision.Presence` (online), `provision.AuditSink` (Record), `provision.CommandPusher` (PushProvision → channel `ProvisionCommand`), plus `provision.Repository` (Create/Get/List/Update/AppendEvent/ListEvents/Upsert+GetNodeVersion). |
| **Production wiring** | `handlers/provision/adapter.go` binds the seams to the concrete registry service, presence hub, audit store, and the channel push (`ProvisionCommand` → ServerFrame → protojson → `Hub.Push`). `handlers/provision/module.go` constructs the service with the sqlite repository. The provision domain imports no sibling domain and no proto. |
| **Test fake** | `internal/provision/mocks` (in-memory `FakeRepository` + one fake per seam) drives `provision_test.go`; real sqlite in `sqlite_test.go`; the real adapters end-to-end in `handlers/provision/connect_handler_test.go`. |
| **Why it exists** | Privileged remote provisioning is the second-highest-stakes surface; the narrow proto-free seams keep the audited orchestration (resolve → validate → audit fail-closed → create durable op → push) and the block-once op lifecycle pure and exhaustively testable, with the proto/channel translation in one auditable place. |

### node-agent privsep seams (StepRunner / RevisionResolver / Reporter)

| | |
|---|---|
| **Seam** | The PRIVILEGED provisioning helper's three effects: running a step's argv, reading the current git revision, and reporting ProvisionEvents back. Declared in `agent/internal/privsep/privsep.go`. **Structurally separate from the runner's exec seams — the two packages never import each other** (proven by `privsep_test.go::TestPrivilegeSeparation_NoCrossImport`). |
| **Interface** | `privsep.StepRunner` (`Run(ctx, argv, dir, onLog)`), `privsep.RevisionResolver` (`Current(ctx, dir)`), `privsep.Reporter` (`Report(ctx, *ProvisionEvent)`). |
| **Production wiring** | The ordinary `agent/internal/channel/channel.go::runProvision` sends a protojson `ProvisionCommand` over the local IPC socket and forwards typed events from the separate `vrooli-bridge-provisioner` service. Only the helper constructs `osStepRunner` (`os/exec` over a typed argv — the privileged execution path) and `osRevisionResolver` (`git rev-parse HEAD`), then reports through the signed `ProvisionService.ReportProvisionEvent` call. Linux and Darwin validate peer UIDs at the socket boundary. |
| **Test fake** | `privsep_test.go` substitutes a recording step runner + scripted revision resolver + collecting reporter; covers idempotent re-provision, rollback-on-failed-setup, degraded-failure, and the no-shell typed `Steps()` plan. |
| **Why it exists** | Lets the provisioning sequence (fetch → checkout → setup → version/exit, with rollback) and the privilege-separation guarantee be tested without a real git/`vrooli setup` or a live control plane. |

### node-agent service-install adapters (Manager / unit renderers)

| | |
|---|---|
| **Seam** | The platform-native background-service install surface: one `Definition` rendered onto systemd / launchd / Windows SCM AND installed (write unit → enable → start) behind the same Manager abstraction. Renderers in `agent/internal/service/service.go`; real install layer in `agent/internal/service/service_install.go`. |
| **Interface** | `service.Manager` (`Kind()`, `Render(Definition)`, `Install/Status/Uninstall(ctx, Definition)`) selected by `service.NewManager()`/`ManagerForKind(kind)`; pure renderers `SystemdUnit` / `LaunchdPlist` (XML-escaped) / `WindowsServiceCreateArgs`. The native tool (systemctl/launchctl) is driven through an injected `commandRunner` seam (`execRunner` in prod, a fake in tests); the filesystem is real. |
| **Production wiring** | `agent/main.go`: `--print-service-unit` renders via `service.NewManager().Render(def)`; the `service install\|status\|uninstall` verbs build the SAME `Definition` (`serviceDefinition`) and call `Install/Status/Uninstall`. Ordinary Linux = systemd `--user` under `~/.config/systemd/user`; the privileged helper = machine-wide systemd under `/etc/systemd/system`. Ordinary macOS = LaunchAgent under `~/Library/LaunchAgents`; the privileged helper = LaunchDaemon under `/Library/LaunchDaemons`. Both paths converge idempotently through their native managers; Windows stays render-only (Install returns a render-only error). `NewManager` mirrors `platform.NativeServiceManager()`. The `Definition.User` field carries the OS principal, making the two trust tiers distinct principals at install time. |
| **Test fake** | `service_test.go` asserts kind selection + render invariants; `service_install_test.go` fakes the `commandRunner` and uses a temp unit dir to assert exact systemctl/launchctl argv, on-disk unit path+content, and the idempotent re-install / re-uninstall paths per OS. The real systemd install→status→kill-9→restart→uninstall lifecycle is exercised on the Linux dev host; the darwin path is argv-covered and awaits the phase-8 mac run. |
| **Why it exists** | One agent codebase installs itself natively on Linux/macOS/Windows (OT-P0-007) with no scattered GOOS checks and no Linux-only assumptions; the pure renderers + faked-exec install layer make both the unit content AND the install sequence testable without touching the host's real service manager. This is the capability the phase-4 bootstrap script drives. |

### compat + presence compatibility (protocol gating, OT-P1-001)

| | |
|---|---|
| **Seam** | The protocol-compatibility verdict the live layer stores per node, gating WORK dispatch. |
| **Interface** | `internal/compat::Evaluate(nodePV)`/`EvaluateAt` (pure, four-band) + `presence.Hub` `SetCompatibility`/`Compatibility`/`Dispatchable`; `dispatch.Presence` gained `Dispatchable(nodeID)`. |
| **Production wiring** | The SSE dial-out (`handlers/channel/sse_handler.go`) reads `?pv=`, calls `compat.Evaluate`, and stores it on the hub; the heartbeat returns the stored verdict; `dispatch.service` excludes non-dispatchable nodes with `ErrNodeNeedsUpdate` (provisioning exempt). |
| **Test fake** | `dispatch/mocks.FakePresence.Flagged`, `fleet/mocks.FakePresence.Flagged`; `compat_test.go`, `dialout_test.go`. |
| **Why it exists** | A version-drifted node holds presence but is FLAGGED and excluded from work rather than silently mis-driven — the mechanism that keeps a fleet coherent. |

### registry readiness facts (OT-P1-002)

| | |
|---|---|
| **Seam** | Five independent operator facts: registry record, heartbeat freshness, channel held, protocol compatibility, and dispatchable. |
| **Interface** | `presence.Hub.Readiness(nodeID)` → `ReadinessFacts`; `NodeRegistryService.GetNodeReadiness` exposes the translated facts to CLI/UI. |
| **Production wiring** | The registry handler overlays the live hub facts without replacing the legacy status enum; `nodes doctor <id>` walks the facts in order and returns a non-zero error at the first failed rung. Fleet rows render each fact with a text/shape marker, never colour alone. |
| **Test fake** | Existing registry presence fakes remain valid through the optional `ReadinessPresence` extension; Hub tests assert heartbeat staleness independently of channel and dispatchability. |
| **Why it exists** | “Online” is a transport observation, not a promise that work can be received. Keeping the facts separate makes half-open, stale, incompatible, revoked, and dispatchable states diagnosable and automatable. |

### fleet seams (NodeLister / Presence / Provisioner / Repository, OT-P1-001)

| | |
|---|---|
| **Seam** | Fleet-wide version roll: enumerate nodes, gate on presence+compat, delegate per-node provisioning, persist the rollout ledger. |
| **Interface** | `internal/fleet/seams.go` (`NodeLister`, `Presence`, `Provisioner`) + `repository.go::Repository`. |
| **Production wiring** | `handlers/fleet/adapter.go` binds `NodeLister`→registry.List, `Presence`→hub, `Provisioner`→the SHARED provision service (`provisionerAdapter`). `provision.Module` split into `NewService`+`Module` so one provision instance backs both the provision handler and the fleet roll. |
| **Test fake** | `internal/fleet/mocks` (FakeNodeLister/FakePresence/FakeProvisioner/FakeRepository); `rollout_test.go` (real sqlite), `service_test.go`. |
| **Why it exists** | Fleet NEVER reimplements provisioning — it fans out to the provision domain through the seam, keeping the privileged tier the single audited path. |

### queue scheduler seams (Pusher / Aborter + runs TerminalHook / Canceller, OT-P1-004)

| | |
|---|---|
| **Seam** | Per-node bounded-concurrency scheduling on the dispatch→push path, with run-terminal slot release and node-side cancel. |
| **Interface** | `internal/queue/seams.go` (`Pusher`, `Aborter`); `runs.Service` options `WithTerminalHook(TerminalHook)` + `WithCanceller(Canceller)`. |
| **Production wiring** | `main.go` builds `queue.NewScheduler(queueH.NewChannelPusher(hub), queueH.NewAborter(runsSvc), …)`; `runsSvc` is constructed with `WithCanceller(channelCanceller)` + `WithTerminalHook(scheduler.Complete)` (the hook closure captures the scheduler var, assigned just after, to break the construction cycle). Dispatch's `jobPusherAdapter` submits to the scheduler (satisfies dispatch's existing JobPusher seam unchanged). |
| **Test fake** | `internal/queue/queue_test.go` (fakePusher/fakeAborter), `scheduling_test.go`; `internal/runs/cancel_test.go` (fakeCanceller). |
| **Why it exists** | Each node runs ≤N jobs at once (default 1, test-genie discipline); a gate fan-out never thrashes a node, and AbortRun actually stops the node's process rather than leaving an ignored stale completion. |

### artifacts.DirectedDelivery (device-sync-hub byte transport, OT-P1-003)

| | |
|---|---|
| **Seam** | The "bridge orchestrates, device-sync-hub moves the bytes" boundary — bridge moves NO bytes and stores NO blob. |
| **Interface** | `internal/artifacts/seams.go::DirectedDelivery` (`Deliver(DeliveryRequest) DeliveryResult`) + `NodeReader`. |
| **Production wiring** | `handlers/artifacts/adapter.go::deviceSyncDelivery` produces a device-sync-hub delivery ref; the concrete device-sync-hub TransferService client is the documented drop-in behind this seam (mirroring audit's workspace-sandbox Sink — device-sync-hub carries an environmental authenticator blocker). |
| **Test fake** | `internal/artifacts/mocks.FakeDelivery`; `distribute_test.go`, `devicesync_integration_test.go` (real sqlite). |
| **Why it exists** | Bridge never reinvents file transport; the seam keeps the artifacts domain proto-free and lets the real device-sync-hub client drop in without touching the domain. |

### artifacts.ProducedArtifactRepository (authenticated run output)

| | |
|---|---|
| **Seam** | Bounded bytes produced by a typed node run and later retrieved by its owner. |
| **Interface** | `internal/artifacts/repository.go::ProducedArtifactRepository` (`PutProducedArtifact`, `GetProducedArtifact`) plus `RunReader` for node ownership validation. |
| **Production wiring** | `handlers/artifacts/module.go` wires the SQLite repository and the runs-service adapter; `connect_handler.go` verifies node signatures for upload and owner identity for retrieval. |
| **Test fake** | `internal/artifacts/mocks.FakeProducedRepository` and `FakeRunReader`; handler coverage exercises the signed upload → owner retrieval round trip. |
| **Why it exists** | A node-local screenshot path is not evidence. This seam makes the transfer explicit, bounded, owner-scoped, and replaceable without granting the node arbitrary filesystem access or making the control plane trust an untyped path. |

### node-agent discovery.Browser (mDNS LAN discovery, OT-P1-006)

| | |
|---|---|
| **Seam** | LAN auto-discovery of the control plane; manual URL stays the cross-network default and the fallback. |
| **Interface** | `agent/internal/discovery::Browser` (the mDNS querier) behind `Resolve` (manual URL wins; mDNS only when no URL given). |
| **Production wiring** | `agent/internal/discovery/mdns.go` — a dependency-free DNS-SD querier over UDP multicast; gated by the `--discover` flag. |
| **Test fake** | `agent/internal/discovery/mdns_test.go` (fake Browser), `fallback_test.go` (manual-URL-wins, Browser never invoked). |
| **Why it exists** | Pairing on a trusted LAN is "run the installer" without typing the control-plane URL; off-LAN bootstrap never depends on mDNS. |

### gate seams (NodeLister / Presence / Runner / Repository, OT-P1-002)

| | |
|---|---|
| **Seam** | Cross-OS deployment gate: select one eligible node per target OS, dispatch a validation run to each, recompute the live cross-OS verdict from the per-OS runs, persist the gate ledger. |
| **Interface** | `internal/gate/seams.go` (`NodeLister`, `Presence`, `Runner`) + `repository.go::Repository`. `Runner` is the dispatch+runs delegation seam (`Dispatch`→runID, `Verdict`/`Wait`→RunVerdict). |
| **Production wiring** | `handlers/gate/adapter.go` binds `NodeLister`→registry.List, `Presence`→hub, `Runner`→the SHARED dispatch service (`dispatchH.NewService`, allowlist+scopes+audit) + the runs service (durable lifecycle). `dispatch.Module` split into `NewService`+`Module` so one dispatch instance backs both the dispatch handler and the gate runner — every gate validation run flows through the same allowlist gate as any other job. |
| **Test fake** | `internal/gate/mocks` (FakeNodeLister/FakePresence/FakeRunner/FakeRepository); `aggregate_test.go` (pure aggregation), `service_test.go`; `handlers/gate/deployment_manager_test.go` drives the real gate→dispatch→runs path. |
| **Why it exists** | Gate NEVER reimplements dispatch or run management — it fans out to dispatch+runs through the seam, so the highest-stakes surface (remote execution) stays the single audited allowlist path even for cross-OS validation. |

### cross-OS gate consumer seam (deployment-manager owns the verdict, OT-P1-002)

| | |
|---|---|
| **Seam** | The boundary where bridge SUPPLIES the cross-OS validation capability and the *consumer* OWNS the promotion verdict. Bridge exposes `GateService` (RunGate/WaitGate); deployment-manager maps the aggregate gate verdict to its own production-readiness decision. |
| **Interface** | Consumer side: `scenarios/deployment-manager/api/crossosgate::Bridge` (`RunGate`/`WaitGate`) + `Gate.Evaluate(Request) Verdict`. Producer side: the generated `GateService` Connect contract. |
| **Production wiring** | deployment-manager's `crossosgate.NewHTTPBridge` speaks bridge's `GateService` over the Connect unary JSON protocol (no proto-module dependency on the consumer); route `POST /api/v1/cross-os-gate/evaluate` is additive + inert (503) until `VROOLI_BRIDGE_URL` is configured. |
| **Test fake** | `deployment-manager/api/crossosgate/crossosgate_test.go` (fake Bridge for the Evaluate mapping; httptest-backed `httpBridge` for the wire contract). |
| **Why it exists** | "Bridge supplies the capability, deployment-manager owns the verdict." The consumer never imports bridge internals; it speaks the wire contract, so the two scenarios evolve independently behind the proto contract. |

### exposed (not built) consumer seams — emulator / contribution-verification / remote-desktop

| | |
|---|---|
| **Seam** | Bridge exposes node identity/reach, durable dispatch, and isolated/ephemeral-node + typed-verdict capabilities that downstream scenarios consume; bridge does NOT build those integrations (out of scope, separate initiatives). |
| **Interface** | `registry.NodeRegistryService` (identity/reach), `dispatch.DispatchService` + `runs.RunsService` (durable allowlisted execution), `gate.GateService` (cross-OS typed verdict). Consumers: vrooli-emulator-remote-node-backend (identity/reach), contribution-verification (isolated node + reset + typed verdict, OT-P2 — see [`PROBLEMS.md`](PROBLEMS.md)), remote-desktop (identity/reach, BRG-P2-002). |
| **Production wiring** | None in bridge — these are the *deliverable seams*, consumed elsewhere. The contracts are the generated proto services above; bridge's obligation is to keep them stable + documented. |
| **Test fake** | Each consumer wires its own client/fake against the generated Connect contract (as deployment-manager does in `crossosgate`). |
| **Why it exists** | Hold the scope line: bridge ships reusable, drift-gated control-plane contracts; the emulator/triage/remote-desktop integrations are built by their own initiatives against these seams, not inside bridge. |

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

## Cross-references

- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
