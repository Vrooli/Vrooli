# Seams — Tunnel Manager

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
`packages/proto/schemas/tunnel-manager/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/tunnel-manager/v1/health/`
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

### Operator authorizer (privileged mutation gate)

| | |
|---|---|
| **Seam** | API/service-layer authorization for privileged mutation RPCs |
| **Interface** | `internal/authz/authz.go::Authorizer` (`Authorize(ctx, operation, headers) error`) |
| **Production wiring** | `handlers/{config,routes,exposure,recovery}/module.go` wires `authz.FromEnv()` into each Connect handler. Enforcement is disabled by default for lifecycle-managed local/operator use; `TUNNEL_MANAGER_AUTHZ_ENFORCED=1` requires `TUNNEL_MANAGER_OPERATOR_TOKEN` or fallback `API_TOKEN`. |
| **Test fake** | Handler tests inject `authz.StaticTokenAuthorizer{Enforced: true, Token: "secret"}` to prove missing/wrong tokens fail before the domain service is called and bearer/header tokens pass. |
| **Why it exists** | Privileged actions (config sync/mode switch, route mutation, exposure mutation/reconcile, manual recovery) must be enforced at the API boundary, not only in CLI or UI affordances. The seam keeps read RPCs usable while fail-closing internet-facing and restart-capable mutations before the scenario grows richer aud-scoped inter-scenario auth. |

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

### CredentialVerifier (live Cloudflare credential/scope probe)

| | |
|---|---|
| **Seam** | Live read-only Cloudflare credential + scope verification |
| **Interface** | `internal/config/types.go::CredentialVerifier` (`Verify(ctx, CFConfig, apexes) (CredentialVerification, error)`) |
| **Production wiring** | `internal/config.NewCFVerifier(doer)` over `httpc.Doer`, wired in `NewProductionService` and backing `ConfigService.VerifyCredentials` / `config credentials-status --verify`. Performs read-only calls (`/user/tokens/verify`, account/tunnel read, `GET /zones?name=<apex>`, DNS-records read) and maps each to `ok\|missing\|invalid\|insufficient_scope` with remediation. Never writes account state; never returns a secret value. |
| **Test fake** | `mocks.FakeDoer` (white-box `verifier_test.go` constructs `cfVerifier{doer:fake}`); each canned response asserts the verdict mapping. |
| **Why it exists** | Presence-only readiness reported `ready:true` for a token that authenticated but lacked `Zone:DNS:Edit`, then produced a dead URL. The probe makes "authenticated" vs "authorized for what TM needs" visible, and gates expose. |

### DNSClient (proxied CNAME automation)

| | |
|---|---|
| **Seam** | Cloudflare DNS-records boundary — the CNAMEs that make an exposed hostname publicly resolvable |
| **Interface** | `internal/config/types.go::DNSClient` (`EnsureRecord(ctx, hostname) (DNSResult, error)`, `RemoveRecord(ctx, hostname) (bool, error)`) |
| **Production wiring** | `internal/config.NewCFDNSClient(doer, cfg)` over `httpc.Doer` (nil when creds absent), wrapped by `resolvingDNSClient` in `NewProductionService` so the config API and exposure's reconcile share one credential path. `EnsureRecord` is additive/idempotent (`<sub>.<apex> CNAME <tunnel-id>.cfargotunnel.com`, proxied); it never clobbers an out-of-band record. Removal is gated by the `DNSLedger` (`dns_ownership` table) so TM only deletes records it created. Remote-mode only. |
| **Test fake** | `mocks.FakeDoer` (`dns_test.go` asserts request shapes/idempotency); `fakeDNS`/`fakeDNSLedger` (`dns_service_test.go`) for service-level ownership-guard tests. |
| **Why it exists** | Without DNS automation a freshly-exposed hostname returned NXDOMAIN (ingress live, no CNAME). The ledger mirror of the ingress-ownership pattern keeps prune/revoke from ever deleting a record TM did not create. |

### PortAssigner (ranged→fixed UI port via structure-health)

| | |
|---|---|
| **Seam** | Cross-scenario port assignment so a ranged scenario becomes exposable as a scenario route |
| **Interface** | `internal/exposure/service.go::PortAssigner` (`EnsureFixed(ctx, scenario) (assignedByTM bool, error)`, `Release(ctx, scenario) error`) + `PortOwnership` (`Record/Owned/Clear`) |
| **Production wiring** | `exposure.StructureHealthPortAssigner` resolves structure-health's API via `api-core/discovery` and drives its `ValidationService.AssignFixedPort`/`ReleaseFixedPort` RPCs; `exposure.NewSQLitePortOwnership` over the `tm_port_assignments` table. Wired via `WithPortAssigner` in `handlers/exposure/module.go`. Best-effort on expose (an already-fixed scenario still exposes if structure-health is down); revoke releases only TM-assigned ports (ownership-gated, never a hand-pinned port). |
| **Test fake** | `fakeAssigner`/`memOwnership` in `internal/exposure/portassign_test.go`. |
| **Why it exists** | The tunnel forwards to a concrete `localhost:<port>`, so a scenario needs a fixed UI port to be a scenario route; structure-health owns the port-band SSOT and the assign/release primitive, so TM calls it rather than re-implementing band logic. |

### CloudflaredUnitPresence (recovery self-gate)

| | |
|---|---|
| **Seam** | "Is there a cloudflared systemd unit to manage on this host?" |
| **Interface** | `internal/recovery/presence.go::UnitPresence` (`CloudflaredUnitPresent(ctx) bool`) — declared at the recovery consumer so the engine never imports a host/systemd package directly (same discipline as `HealthChecker`). |
| **Production wiring** | `recovery.NewSystemctlUnitPresence(cmdrunner.Default)`, wired in `handlers/recovery/module.go::NewProductionService`. Runs `systemctl list-unit-files --no-pager --no-legend cloudflared.service` and matches a non-empty line (catches units under both `/etc/systemd/system` and `/lib/systemd/system`). |
| **Test fake** | `fakePresence{present: bool}` / `togglePresence` in `internal/recovery/service_test.go`; `NewService` also accepts `nil` (treated as always-present) for tests exercising the restart/backoff paths that don't care about the gate. |
| **Why it exists** | Default-on recovery must stay dormant on a tunnel-less host — without the gate it would count `/ready` failures forever and flap a restart that can't help, eventually opening the circuit spuriously. Consulted at the **top of every `Evaluate()`** (not boot-time only) so a cloudflared installed after the scenario started is picked up on the next tick. The gate is unit **presence**, not live `/ready`, because readiness is false exactly when recovery is most needed. The `cloudflared_recovery_privileges` safeguard mirrors this same presence check at the host-provisioning layer. |

## Product Seams

> **Status: REALIZED (all seven domains built and green).** The seams
> below are the test-substitutable boundaries the seven product domains
> ([`../concepts/DOMAINS.md`](../concepts/DOMAINS.md)) need; they are now
> implemented and exercised by passing tests. The descriptions remain the
> authoritative intent; the as-built symbol names differ in a few places
> from the original planned names (the boundary and behaviour are
> identical). As-built map:
>
> | Planned name | As built |
> |---|---|
> | `internal/exec/runner.go::CommandRunner` (cross-cutting) | `internal/cmdrunner/runner.go::Runner` + `cmdrunner.Default`; fake `internal/testutil/mocks::FakeCmdRunner`. Wired domain-locally in each `handlers/<d>/module.go` (tunnel/config/recovery), not on `server.Deps`. |
> | `internal/config/cloudflare.go::CloudflareAPI` | `internal/config` `IngressClient` + `NewCFClient` over `httpc.Doer`; nil when CF creds absent → `ErrRemoteUnavailable`. `internal/config.NewProductionService` is the canonical production builder used by both `handlers/config` and `handlers/exposure` so ingress wiring cannot drift. |
> | `internal/tunnel/scrape.go::MetricsSource` | folded into `internal/tunnel` service over `httpc.Doer` + `cmdrunner.Runner` (Prometheus parse + `/ready`). |
> | `internal/probes/prober.go::Prober` | folded into `internal/probes` service over `httpc.Doer`. |
> | `internal/exposure/coreset.go::CoreSetProvider` (`CoreSet(ctx)([]string,error)`) | `exposure.CoreSetProvider func() []string`, wired to `coreset.CoreSeedScenarios`. |
> | `internal/exposure/lifecycle.go::LifecycleEnsurer` | `exposure.Runner` (`EnsureRunning(ctx, scenario)`) + `exposure.CLIRunner` over `cmdrunner` (`vrooli scenario start`). |
> | `internal/audit/servicejson.go::ServiceManifestReader` | `audit` service reads `service.json` directly with an injectable scenarios-root; `exposure.FilePortResolver` does the same for the UI port. |
> | Per-domain `Repository` + `<domain>.Schema` | built as specified for routes/exposure/config/tunnel/probes/recovery; `audit` owns no table. |
>
> Tunnel Manager actuates live infrastructure (cloudflared, the Cloudflare
> API, other scenarios' run state), so almost every product seam exists to
> keep those side effects out of the test path entirely — no test touches
> real cloudflared, the real Cloudflare API, or systemd. See
> [`DECISIONS.md`](DECISIONS.md) (auto-recovery LIVE) for why this
> discipline is load-bearing.

### Cloudflare ingress client (`config` domain)

| | |
|---|---|
| **Seam** | Cloudflare API v4 ingress management (remote-mode hostname → `localhost:<port>` config push, hot-reload) |
| **Interface** | `internal/config/types.go::IngressClient` with `ReadIngress(ctx)` and `PushIngress(ctx, []IngressRule)`. |
| **Production wiring** | `internal/config/production.go::NewProductionService(...)` wires a `CredentialStore` plus a resolving `IngressClient` that re-resolves Cloudflare credentials for each remote read/push before constructing `NewCFClient` over `internal/httpc.Doer`. Cloudflare credentials come only from the Vrooli credential authority; environment variables and legacy `CF_*` aliases are not accepted. Both `handlers/config/module.go` and `handlers/exposure/module.go` use this builder; exposure wraps the resulting config service with `ingressAdapter{cfg}.Reconcile`. |
| **Test fake** | Service tests use a small fake `IngressClient`; builder/adapter integration tests use `internal/testutil/mocks::FakeDoer` to assert Cloudflare GET/PUT request shape without network I/O. |
| **Why it exists** | Live ingress pushes against a real account would be destructive and non-deterministic. The fake lets `Sync`/`SwitchMode`/remote-mode reconciliation be asserted (correct rule set derived from the manifest, idempotent re-push, error classification on 5xx/auth-fail) without a Cloudflare account or network. OT-P0-002, OT-P1-002. |

### Exposure reconcile scheduler (`exposure` domain)

| | |
|---|---|
| **Seam** | Boot + periodic CORE reconcile and expired-lease reaping. |
| **Interface** | `internal/exposure/scheduler.go::Scheduler`, configured with the existing `exposure.Service` interface and an injectable tick channel (`SchedulerConfig.Ticks`) for tests. |
| **Production wiring** | `api/main.go` constructs one production exposure service via `handlers/exposure.NewProductionService`, mounts that same service in `ModuleWithService`, and starts `Scheduler.Run` in a cancellable goroutine. Cleanup cancels the scheduler and waits for it before closing SQLite. `TUNNEL_MANAGER_EXPOSURE_RECONCILE_INTERVAL` controls cadence; `TUNNEL_MANAGER_EXPOSURE_SCHEDULER_DISABLED` disables the loop while preserving manual reconcile. |
| **Test fake** | `internal/exposure/scheduler_test.go` uses a fake `Service` plus an injected tick channel to prove boot reconcile, periodic reconcile, retry-after-error, and context cancellation without sleeps or live Cloudflare calls. |
| **Why it exists** | CORE routes and expired leases are temporal guarantees, not only manual RPCs. Keeping the scheduler on the exposure service seam makes it idempotent, serial, cancellable, and independent of HTTP transport while reusing the same config/ingress wiring as operator-triggered reconcile. OT-P0-003, OT-P0-004. |

### Systemd / process-exec (`tunnel` + `recovery` domains)

| | |
|---|---|
| **Seam** | Out-of-process command execution against the cloudflared systemd unit (`systemctl status/restart cloudflared`) and any cloudflared invocation |
| **Interface** | `internal/cmdrunner.Runner`, a function seam `func(context.Context, string, ...string) (Result, error)` returning exit code, stdout, and stderr. |
| **Production wiring** | `handlers/recovery.NewProductionService` constructs the recovery engine over `cmdrunner.Default`. `api/main.go` constructs one recovery service and shares it between `RecoveryService` and the optional recovery scheduler. A real restart is issued only through `recovery.Service` after threshold/backoff/circuit checks. |
| **Test fake** | `internal/testutil/mocks::FakeCmdRunner` records invocations and injects command errors. |
| **Why it exists** | A test must NEVER restart real cloudflared — that is foundational infra for the whole host. The seam makes "on `/ready` failure → restart with backoff → circuit-break after N attempts" fully assertable against recorded fake invocations. Background recovery evaluation is opt-in with `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED`; manual recovery remains available. OT-P0-008, OT-P0-011. |

### Prometheus-scrape HTTP (`tunnel` domain)

| | |
|---|---|
| **Seam** | Read of cloudflared's Prometheus metrics endpoint (default `127.0.0.1:20241/metrics`) and the `/ready` endpoint |
| **Interface** | The `internal/tunnel` service reads `/ready` and Prometheus text-format metrics through `internal/httpc.Doer`, then parses them into domain metrics. |
| **Production wiring** | `handlers/tunnel/module.go::Module(...)` constructs the service over a timeout-bounded real HTTP client and the configured Prometheus endpoint from `tunnel_config`. |
| **Test fake** | `internal/testutil/mocks::FakeDoer` supplies canned Prometheus text-format bodies and `/ready` responses. |
| **Why it exists** | HA-connection / RTT / request-error parsing and degraded-mode detection (HA < 4, RTT spikes) must be deterministic. Canned scrape bodies pin the parse + threshold logic without a running cloudflared. OT-P0-008, OT-P1-003/006. |

### HTTP prober + probe scheduler (`probes` domain)

| | |
|---|---|
| **Seam** | Liveness probing of a route's local port (internal) and public URL (external) |
| **Interface** | `internal/httpc.Doer` for each outbound GET plus `internal/probes.Service.RunProbes` for scheduler-level execution. `internal/probes.Scheduler` accepts the service interface and an injectable tick channel. |
| **Production wiring** | `handlers/probes.NewProductionService` constructs the service over routes, SQLite, a timeout-bounded `*http.Client`, and the clock. `api/main.go` mounts that same service in `ModuleWithService` and starts `Scheduler.Run` in a cancellable goroutine unless `TUNNEL_MANAGER_PROBE_SCHEDULER_DISABLED` is set. |
| **Test fake** | Service tests use `internal/testutil/mocks::FakeDoer`; scheduler tests use a fake `probes.Service` plus injected ticks. |
| **Why it exists** | Failure classification currently derives from the *pattern* of internal-vs-external probe results: healthy, tunnel-down, scenario-down, or config-drift. DNS failure and Cloudflare outage require future resolver/edge signals and are not produced yet. The seams let probe execution, scheduler retry, and classification branches be tested without live routes. OT-P0-009/010, partial OT-P1-001. |

### Recovery evaluation scheduler (`recovery` domain)

| | |
|---|---|
| **Seam** | Boot + periodic recovery evaluation. |
| **Interface** | `internal/recovery/scheduler.go::Scheduler`, configured with the existing `recovery.Service` interface and an injectable tick channel (`SchedulerConfig.Ticks`) for tests. |
| **Production wiring** | `api/main.go` constructs one recovery service via `handlers/recovery.NewProductionService`, mounts that same service in `ModuleWithService`, and starts `Scheduler.Run` only when `TUNNEL_MANAGER_RECOVERY_SCHEDULER_ENABLED` is truthy. Cleanup cancels the scheduler and waits for it before closing SQLite. `TUNNEL_MANAGER_RECOVERY_EVALUATE_INTERVAL` controls cadence. |
| **Test fake** | `internal/recovery/scheduler_test.go` uses a fake `Service` plus an injected tick channel to prove boot evaluation, periodic evaluation, retry-after-error, action logging, and context cancellation without sleeps or systemd. |
| **Why it exists** | Recovery evaluation is temporal but potentially destructive. Keeping it behind the service seam makes the scheduler lifecycle-owned and testable while leaving all restart decisions to the engine's threshold/backoff/circuit policy. OT-P0-011. |

### Clock (leases TTL + backoff)

| | |
|---|---|
| **Seam** | Wall-clock time for lease expiry, the reaper, probe scheduling cadence, and recovery exponential backoff. **Reuses the existing `Clock` seam** (see "Clock" above) — no new interface. |
| **Production wiring** | The `exposure`, `probes`, and `recovery` modules receive the same `clock.Clock` already wired from `main.go` via `server.Deps`. |
| **Test fake** | `internal/testutil/mocks::FakeClock` (existing) — `Advance(d)` drives lease expiry, reaper eligibility, scheduler ticks, and backoff windows. |
| **Why it exists** | TTL ≈1-week leases, auto-reaping, probe cadence, and exponential-backoff timings are all time-dependent. With `FakeClock.Advance`, "lease expires after TTL", "reaper skips a CORE route", and "backoff doubles per attempt up to the cap" are exact, not sleep-and-hope. OT-P0-004, OT-P0-011. |

### CoreSet provider (`exposure` domain)

| | |
|---|---|
| **Seam** | The set of always-on CORE scenarios, sourced from `packages/api-core/coreset` |
| **Interface** | `internal/exposure/types.go::CoreSetProvider`, currently a function seam returning the configured CORE scenario ids. |
| **Production wiring** | `handlers/exposure.NewProductionService(...)` wires the provider to `api-core/coreset`. Reconciliation marks every member as a CORE route that never auto-expires. |
| **Test fake** | Service tests pass a canned provider function. |
| **Why it exists** | CORE-tier reconciliation logic ("every coreset member is exposed and never reaped") must be tested against a fixed, controllable membership without depending on the live coreset (which changes). Decouples exposure policy tests from coreset contents. OT-P0-003. See [`DECISIONS.md`](DECISIONS.md) (CORE tier = coreset). |

### Lifecycle ensure-running (`exposure` domain)

| | |
|---|---|
| **Seam** | Ensure-a-scenario-is-running delegation to the platform `internal/lifecycle` seam |
| **Interface** | `internal/exposure/types.go::Runner` with `EnsureRunning(ctx, scenario string) error`. |
| **Production wiring** | `handlers/exposure.NewProductionService(...)` wires `exposure.CLIRunner` over `cmdrunner.Default` and delegates to `vrooli scenario start`. `Expose(scenario, ttl)` calls `EnsureRunning` before requesting ingress. |
| **Test fake** | Service tests use a fake runner that records ensure calls and injects "scenario won't start" errors. |
| **Why it exists** | Tunnel Manager **delegates** lifecycle, never reimplements it ([`DECISIONS.md`](DECISIONS.md), PRD non-goal). The seam keeps that delegation a one-line call the exposure flow asserts on, while keeping real process management — slow, stateful, host-mutating — entirely out of the test path. OT-P0-006. |

### Filesystem service.json reader (`audit` domain)

| | |
|---|---|
| **Seam** | Read-only reads of other scenarios' `service.json` for port-compliance auditing |
| **Interface** | The audit service reads scenario `service.json` files from an injectable scenarios root. **Read-only by contract** — never writes another scenario's files. |
| **Production wiring** | `handlers/audit/module.go::Module(...)` constructs the service rooted at the workspace scenarios directory. |
| **Test fake** | Tests point the service at fixture directories containing controlled `service.json` shapes. |
| **Why it exists** | Audit findings (port matches manifest / missing fixed port / non-fixed ranged port) must be tested against controlled `service.json` shapes without populating a real scenarios tree. The read-only contract is also a security boundary (see [`SECURITY.md`](SECURITY.md)). OT-P0-007. |

### Per-domain SQLite store seams (`routes`/`exposure`/`config`/`tunnel`/`probes`/`recovery`)

| | |
|---|---|
| **Seam** | Persistence boundary per data-owning domain, following the established Repository/Service pattern (see the `<domain>.Schema` registry seams above) |
| **Interface** | One `Repository` interface per owning domain: `internal/routes/repository.go::Repository` (`routes` table), `internal/exposure/repository.go::Repository` (`leases` table), `internal/config/repository.go::Repository` (`tunnel_config`), `internal/tunnel/repository.go::Repository` (`metrics` time-series), `internal/probes/repository.go::Repository` (`probes` history), `internal/recovery/repository.go::Repository` (`recovery_events`). The `audit` domain owns no table (computed). |
| **Production wiring** | Each `handlers/<domain>/module.go::Module(...)` constructs `<domain>.NewSQLiteRepository(db, clk)` and passes it into `<domain>.NewService(repo)`. Each domain contributes its `schema.sql` via the existing `<domain>.Schema` registry seam. **SQLite only** ([`DECISIONS.md`](DECISIONS.md)) — no external DB. |
| **Test fake** | Per-domain co-located `internal/<domain>/mocks::FakeRepository` for service tests; real sqlite via `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` for repository tests (the canonical compose pattern in [`TESTING.md`](TESTING.md)). |
| **Why it exists** | Domain-owned persistence keeps the manifest, leases, metrics, probe history, and recovery log independently testable and deletable. The two-path test split (service-level fakes where useful, real sqlite for repository tests) keeps domain logic isolated without depending on a live scenario database. OT-P0-001/004, OT-P1-003/005. |

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
