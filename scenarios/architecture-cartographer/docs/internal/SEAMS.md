# Seams — Architecture Cartographer

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
`packages/proto/schemas/architecture-cartographer/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/architecture-cartographer/v1/health/`
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

### Detector (conflict producer — pluggable, day-one)

| | |
|---|---|
| **Seam** | Pluggable conflict-detection plug-in |
| **Interface** | `api/internal/conflicts/detector.go::Detector` (`Name()`, `Description()`, `EmitsTypes()`, `Class()`, `Detect(ctx, DetectInput) ([]Conflict, error)`). |
| **Production wiring** | `main.go` registers detectors through `conflicts.NewRegistryWithProfiles(conflicts.DefaultSurfaceProfiles(), ...)`. Current production detectors include `cycle`, `layering`, `naming`, `glossary_drift`, `mislocated_file`, `convergence_drift`, `coupling_smell`, `surface_coherence`, `cross_scenario`, `domains_doc_parse_warning`, and `intent_alignment`; the registry stamps every emitted conflict with the detector's `finding_class` and rejects `unspecified`. Heuristic detector emissions are capped at `warn`. |
| **Test fake** | `internal/conflicts/mocks::FakeDetector` (canned `[]Conflict` return, recorded inputs). Used to drive the conflicts service against deterministic finding sets without standing up a full graph + manifest fixture. |
| **Why it exists** | Detectors are the open extension seam for the cartographer — new drift checks must be addable without changing the envelope or the resolution machinery. The registry is the single point that enumerates registered detectors; orchestration logic does not name-check detector types. See `SIGNAL_LADDER.md` for the analogous pattern on the scoring side. |

### ClaimProvider (intent extraction seam)

| | |
|---|---|
| **Seam** | Normalized PRD and requirement intent claims for detector consumption. |
| **Interface** | `api/internal/conflicts/claim_provider.go::ClaimProvider` returns `[]intent.CapabilityClaim` for a scenario. |
| **Production wiring** | `internal/app/modules.go` wires `conflicts.NewFileClaimProvider(domains.NewRepoScenarioLocator(repoRoot))`; the provider delegates PRD and requirements parsing to `packages/intent-go`, so cartographer does not re-parse those artifacts. |
| **Test fake** | `conflicts.StaticClaimProvider` returns canned claims and is used by `detectors/intentalignment` tests. |
| **Why it exists** | Intent alignment needs outcome and requirement claims, but detector code should stay independent of repository layout and extractor details. This seam keeps extraction in `intent-go` and detection in cartographer. |

### Intent Matcher (lexical / semantic strategy seam)

| | |
|---|---|
| **Seam** | Adjacent-rung intent matching strategy for the `intent_alignment` detector. |
| **Interface** | `api/internal/conflicts/detectors/intentalignment/matcher.go::Matcher` compares outcome, requirement, and domain claims and returns normalized `Match` values. |
| **Production wiring** | `intentalignment.New()` wires `LexicalMatcher`, which implements deterministic Tier 1 glossary-vocabulary matching and emits `intent.vocab_drift` through the detector. |
| **Test fake** | `intentalignment.NewWithMatcher(...)` accepts any matcher implementation; focused tests exercise the lexical matcher directly. |
| **Why it exists** | The deterministic spine and lexical matcher are production checks, while embedding and LLM strategies are explicit off-by-default seams. This keeps future semantic coverage work pluggable without changing the conflict envelope or re-parsing PRD, requirements, or DOMAINS. |

### FindingClass (deterministic vs heuristic gate)

| | |
|---|---|
| **Seam** | Finding classification shared by cartographer, test-genie, and campaign consumers. |
| **Interface** | `packages/proto/schemas/architecture-cartographer/v1/shared/shared.proto::FindingClass` and `packages/proto/schemas/architecture/v1/findings.proto::FindingClass`. Native `Conflict`, audit `ConflictSummary`, and shared `ArchitectureFinding` all carry the enum. |
| **Production wiring** | `conflicts.Registry.DetectAll` stamps detector-native class; `audit.Service.decideOutcome` gates only `deterministic` findings at `error` or `blocker`; test-genie reads the native `AuditRunResponse` from `ScenarioValidationService.native_detail` and applies the same class-aware gate. |
| **Test fake** | `internal/conflicts/mocks::FakeDetector` lets tests emit deterministic or heuristic classes. `test-genie/internal/orchestrator/phases/validationprovider` has fixtures proving heuristic native blockers remain advisory. |
| **Why it exists** | Severity alone cannot distinguish a deterministic broken boundary from an advisory placement/naming/coupling signal. The class seam keeps the report useful while preventing heuristic findings from hard-failing CI. `finding_class` is deliberately excluded from `csid:` and `afid:` stable identity. |

### SurfaceProfile (detector applicability)

| | |
|---|---|
| **Seam** | Per-surface detector selection for API/CLI/UI and language-specific graph evidence. |
| **Interface** | `api/internal/conflicts/profiles.go::SurfaceProfile` maps `(surface, language)` to detector names; `DefaultSurfaceProfiles()` is the production matrix. |
| **Production wiring** | The conflicts registry derives active surfaces from graph files/packages plus domain declarations, then runs only detectors selected by the matching profiles plus the universal floor (`cycle`, `glossary_drift`, `intent_alignment`, `naming`, `mislocated_file`). |
| **Test fake** | `api/internal/conflicts/profiles_test.go` uses tiny named detectors and synthetic graph/domain inputs to prove profile selection without invoking real detectors. |
| **Why it exists** | Not every detector is meaningful on every surface. Profiles keep applicability policy centralized, prevent UI-only or CLI-only scenarios from receiving irrelevant API findings, and make future surface-specific detectors addable without scattering surface checks through detector implementations. |

### SurfaceProvider (code-facts substrate)

| | |
|---|---|
| **Seam** | Scenario surface and parse-unit inventory used by domain derivation. |
| **Interface** | `api/internal/domains/surface_provider.go::SurfaceProvider` (`Inspect(ctx, scenarioDir) (SurfaceInventory, error)`). |
| **Production wiring** | `main.go` constructs `domains.NewCodeFactsSurfaceProvider(...)` and passes it to `domains.ExtractorsForWithSurfaceProvider(...)`. |
| **Test fake** | Domain tests pass a tiny fake provider with canned `SurfaceInventory`; offline production falls back through `LocalSurfaceProvider` and emits a `code_facts.unavailable` extraction warning. |
| **Why it exists** | Code-facts owns surface and parse-unit discovery. Cartographer owns domain reasoning. This seam keeps those responsibilities separate, lets tests substitute deterministic inventories, and makes code-facts outages loud instead of silently reintroducing heuristic authority. |

### Resolver (mechanical fixer — pluggable)

| | |
|---|---|
| **Seam** | Pluggable mechanical fix executor |
| **Interface** | `internal/conflicts/resolver.go::Resolver` (`Name() string` matching a `Conflict.Type`; `Apply(c Conflict, args []string) error`) |
| **Production wiring** | `internal/conflicts/registry.go::DefaultResolvers()` registers resolvers for conflict types that have a deterministic mechanical fix (initially: `mislocated_file`). Code-body refactors stay in agent-Edit territory — there is no resolver for cycle resolution in v1 because the design decision (interface inversion, type extraction, etc.) cannot be auto-made. |
| **Test fake** | `internal/conflicts/mocks::FakeResolver` (records `Apply` calls, returns canned error). Used to assert CLI `conflicts resolve` dispatches to the right resolver. |
| **Why it exists** | Some conflicts have mechanical resolutions (move a misplaced file → file move + import rewrites). Putting the mechanical action behind a resolver keeps `Apply` flows uniform; conflicts that require human judgment simply have no resolver registered and surface as "manual: edit code and run `arch-cart conflicts validate`." |

### Signal (chunk→domain scorer — pluggable, day-one)

| | |
|---|---|
| **Seam** | Pluggable scoring signal in the auto-placement ladder |
| **Interface** | `internal/signals/signal.go::Signal` (`Name() string`, `Score(chunk Chunk, domain Domain, ctx GraphContext) Score`) |
| **Production wiring** | `internal/signals/registry.go::DefaultRegistry()` registers the six day-one signals (`path-token`, `import-cluster`, `importer-voting`, `test-coupling`, `symbol-glossary`, `git-co-edit`). Aggregation logic in `internal/signals/aggregator.go` invokes them. All signals are pure functions over an immutable graph snapshot. `import-cluster` computes deterministic Louvain communities once per graph context and shares them through `signals.Caches`. |
| **Test fake** | `internal/signals/mocks::FakeSignal` (returns canned `Score`, records inputs). Used to test the aggregator and verdict-tier logic in isolation from real signals. |
| **Why it exists** | Auto-placement explainability requires that every verdict be decomposable into per-signal contributions. The seam enforces purity (no graph mutation, no side effects during scoring) and bounds output to `[0.0, 1.0]` so the aggregator math is well-defined. See [`../concepts/SIGNAL_LADDER.md`](../concepts/SIGNAL_LADDER.md). |

### Recipe (mechanical refactor — pluggable, P1)

| | |
|---|---|
| **Seam** | Pluggable mechanical refactor executor for known patterns |
| **Interface** | `internal/apply/recipe.go::Recipe` (`Name() string`, `Plan(args RecipeArgs) (Plan, error)`, `Apply(plan Plan) error`) |
| **Production wiring** | `internal/apply/recipes/registry.go::DefaultRegistry()` ships empty in v0.1. Recipes are added when ≥10 instances of a pattern justify the automation (OT-P1-002). First batch: `extract-shared-types`, `invert-dependency`, `split-file`. |
| **Test fake** | `internal/apply/mocks::FakeRecipe` (records `Plan` and `Apply` calls, returns canned outcomes). Used to assert recipe dispatch from CLI. |
| **Why it exists** | Recipes are mechanical executors for *known* patterns; the design decision (which recipe to apply where) stays with the agent. Behind the seam, each recipe enforces the build-green guardrail before reporting success. Adding a recipe means implementing the interface + registry entry — no orchestration change. |

### CodeGraphAdapter (cross-scenario graph extraction)

| | |
|---|---|
| **Seam** | Adapter to language-specific code-graph scenarios (`go-code-graph`, `typescript-code-graph`) |
| **Interface** | `internal/graph/adapter.go::CodeGraphAdapter` (`Extract(ctx, scenario string) (Graph, error)`, `Rewrite(ctx, ops []Operation) error`) |
| **Production wiring** | `internal/graph/gocodegraph/client.go` and `internal/graph/tscodegraph/client.go` construct Connect-RPC clients to the respective scenarios. URL resolution + retry policy follow the same pattern as ui-health's react-component-library client (`scenarios/ui-health/api/integrations/reactcomponentlibrary/client.go`). |
| **Test fake** | `internal/graph/mocks::FakeCodeGraphAdapter` (canned `Graph` and operation responses, recorded calls). Used by graph-service tests so cartographer integration tests don't require live language-graph scenarios. |
| **Why it exists** | Cartographer must never parse source code itself (architecture rule). The adapter is the single boundary between cartographer's normalized graph model and the wire format of each language-graph scenario. Adding a new language (Python, Rust) means writing a new adapter + registering it; the rest of the cartographer is unchanged. |

### BuildGuard (build-green baseline)

| | |
|---|---|
| **Seam** | Build-green baseline capture and diff |
| **Interface** | `internal/apply/buildguard.go::BuildGuard` (`Baseline(ctx, scenario string) (BuildStatus, error)`, `Current(ctx, scenario string) (BuildStatus, error)`, `Diff(baseline, current BuildStatus) Verdict`) |
| **Production wiring** | `internal/apply/buildguard/go.go` and `internal/apply/buildguard/ts.go` shell out to `go build ./...` and `tsc --noEmit` respectively, using the target scenario's toolchain. Cached by file-content hash to avoid re-running on unchanged trees. |
| **Test fake** | `internal/apply/mocks::FakeBuildGuard` (canned `BuildStatus` returns, recorded baseline captures, programmable diff). Used to test `--force --note` paths and per-domain apply refusal logic without touching real toolchains. |
| **Why it exists** | Every code-modifying operation must leave the build green or roll back. The seam centralizes baseline capture/diff so each recipe and each apply step uses the same guard, and tests can simulate broken-baseline states deterministically. |

### AnalyticsRecorder (event capture)

| | |
|---|---|
| **Seam** | Append-only event log for conflict detection, resolution, placement, override, build deltas |
| **Interface** | `internal/analytics/recorder.go::Recorder` (`Record(ctx, Event) error`, `Query(ctx, Filter) ([]Event, error)`) |
| **Production wiring** | `internal/analytics/sqlite.go::NewSQLiteRecorder(db)` persists to an append-only `events` table. Every domain that emits analytics calls through this seam — no direct DB writes. |
| **Test fake** | `internal/analytics/mocks::FakeRecorder` (in-memory event log, query helpers, atomic call counter). Used by every domain's tests to assert event emission without standing up SQLite. |
| **Why it exists** | Analytics is P0 (OT-P0-009) precisely because override tracking is the highest-value calibration signal. Centralizing capture through a recorder seam ensures the schema stays consistent and the minimum-N threshold for surfaced metrics is enforced in one place. |

### Doer (outbound HTTP)

| | |
|---|---|
| **Seam** | Outbound HTTP request boundary |
| **Interface** | `internal/httpc/doer.go::Doer` (`Do(*http.Request) (*http.Response, error)`) |
| **Production wiring** | Ships unwired in production by intent (no consumer until a real outbound call lands). `*http.Client` satisfies `Doer` directly via the compile-time assertion in `doer.go`; the first scenario to need an outbound call adds the field to `server.Deps` and wires `&http.Client{Timeout: …}` from `main.go`. |
| **Test fake** | `internal/testutil/mocks::FakeDoer` (canned `*http.Response` queue, recorded `*http.Request` log, atomic `Calls` counter). |
| **Why it exists** | Network calls in handler tests would be flaky and slow. Defining the seam *before* the first consumer means the first scenario to call outward doesn't reinvent ad-hoc mocking. Pattern proven in `scenarios/agent-manager/api/internal/promptmanager/client.go`. See `internal/httpc/doer_test.go` for the substitution reference. |

### DomainSourceExtractor (domain-derivation ladder rung)

| | |
|---|---|
| **Seam** | One rung of the domain-extraction ladder |
| **Interface** | `internal/domains/extractor.go::DomainSourceExtractor` (`Source() Source`, `Extract(ctx, scenarioDir) (Extraction, error)`) |
| **Production wiring** | `main.go` constructs the ladder through `domains.ExtractorsForWithSurfaceProvider(...)` — `DomainsDocExtractor` + surface-backed folder/CLI extractors, in trust order — and passes them to `domains.NewService(...)`. A future `APIManifestExtractor` registers ahead of the DOMAINS.md rung when api-health ships. |
| **Test fake** | `internal/domains/mocks::FakeExtractor` (programmable `Extraction`/`Err`, records `scenarioDir` calls). |
| **Why it exists** | The derived domain map replaces the deleted per-scenario architecture manifest. Each rung reads a different source (DOMAINS.md, code-facts-backed API folders, CLI groups); the seam lets ladder-resolution, convergence, and `domains draft` authoring be tested with synthetic declarations and lets new rungs register without touching resolution logic. See `internal/domains/ladder_test.go` and `service_test.go`. |

### ScenarioLocator (scenario-directory resolution)

| | |
|---|---|
| **Seam** | Resolve a scenario name to its on-disk root directory |
| **Interface** | `internal/domains/service.go::ScenarioLocator` (`Locate(scenario) (string, error)`) |
| **Production wiring** | `main.go` constructs `domains.NewRepoScenarioLocator(repoRoot)`, resolving `<repoRoot>/scenarios/<name>`. The same locator backs the suppression Provider and the apply marker-writer (it satisfies `suppressions.Locator` structurally). |
| **Test fake** | `internal/domains/mocks::FakeLocator` (fixed `Dir` / `Err`). |
| **Why it exists** | Domain derivation is filesystem-driven; the locator decouples "which scenario" from "where on disk", so service tests point at fixture directories without depending on repo layout. |

### SuppressionScanner (in-repo marker discovery)

| | |
|---|---|
| **Seam** | Discover `// arch:allow` suppression markers in a scenario's source tree |
| **Interface** | `internal/suppressions/scanner.go::Scanner` (`Scan(ctx, scenarioDir) ([]Marker, error)`); the higher-level `Provider` (`Active(ctx, scenario) ([]Marker, error)`) composes a `Locator` + `Scanner` + `Clock` and filters to valid, non-expired markers. |
| **Production wiring** | `main.go` constructs `suppressions.NewProvider(scenarioLocator, suppressions.NewFileScanner(), clk)` and passes it to the conflicts handler, which feeds active markers into `DetectConflicts`. |
| **Test fake** | `internal/suppressions/mocks::FakeScanner` / `FakeProvider` (canned markers). |
| **Why it exists** | Conflicts must be reported as suppressed-with-reason when an in-repo marker sanctions them. The seam lets conflict-suppression matching be tested with synthetic markers and lets the scan strategy change (filesystem today) without touching the conflicts service. |

### suppressions.Writer (safe marker write)

| | |
|---|---|
| **Seam** | Insert a suppression marker comment into a source file |
| **Interface** | `internal/suppressions/writer.go::Writer` (`WriteMarker(absPath, line, Marker) error`) |
| **Production wiring** | `main.go` wires `apply.WithSuppressionWriter(suppressions.NewFileWriter(), scenarioLocator)` so `ApplyService.WriteSuppression` can write a marker. This is the only mutating apply path in v0.1 — comment-only, idempotent per `(file, id)`; destructive file-moving execution stays deferred. |
| **Test fake** | `internal/suppressions/mocks::FakeWriter` (records `WriteCall`s instead of touching the filesystem). |
| **Why it exists** | "Resolve a finding as intentional" must write a durable marker without the apply test touching real files, and without coupling apply to a concrete filesystem writer. |

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
| No in-process source parsing | Cartographer could be tempted to parse Go/TS in-process for speed. | Strictly forbidden — all parsing goes through `go-code-graph` / `typescript-code-graph` via `CodeGraphAdapter`. Architecture rule, not just a preference. | If a future need surfaces, propose a new language-graph scenario; do not add a parser in `internal/graph/`. |
| Plug-in registries (Detector / Resolver / Signal / Recipe) | Hard-coded switch statements would bake conflict types into the orchestration layer. | Each plug-in type has a single registry; orchestration code dispatches by `Name()` and never enumerates types. | Adding a new conflict type or signal must not require changes to anything but the relevant registry and a new struct file. |
| Build-green guard as a seam, not a flag | Build verification could be ad-hoc per apply path. | `BuildGuard` is a first-class seam; every code-modifying operation routes through it. `--force --note` is the only bypass and is logged. | If a new apply path is added (e.g., recipe execution), it must consume `BuildGuard` — there is no other entry point. |

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

## Cross-references

- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
