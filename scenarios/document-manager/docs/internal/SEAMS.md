# Seams — Document Manager

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
`packages/proto/schemas/document-manager/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/document-manager/v1/shared/`
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
| **Production wiring** | `cli/manifest_embed.go` embeds the manifest bytes; `cli/app.go` passes them to `domains.SubcommandGroups(core, manifest)`; each domain's `Register(core, manifest)` calls `cliapp.LoadFromManifestPrimitives` with its group name and a bindings map keyed by `Service.Method`, each value a cli-core primitive (a `<name>Call` + `<name>Report` pair). A multipart-upload REST exception is appended outside the manifest path with a plain `RunCtx` (no primitive evidence — a documented exception) because cli-manifest/v1 only models `binding.kind=connect-rpc`. |
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

### GatewayRequest constructor (residency choke point)

| | |
|---|---|
| **Seam** | The single construction site for every outbound AI Gateway request |
| **Interface** | `internal/gatewayreq/gatewayreq.go::Builder` (`For(ctx, doc DocumentClass, role string, opts …) (*sharedpb.GatewayRequest, error)`). It resolves `PrivacyClass` → `Profile`, attaches any declared `required_vram_bytes` footprint, and returns an error rather than a request when the caller asks for a profile weaker than the document's class. |
| **Production wiring** | Constructed once in `main.go` from the `sensitivity` domain's policy table and passed through `server.Deps`. `enrichment`, `sensitivity` and `derivation` (tier 3) receive the `Builder` — never the raw proto type. No domain imports `sharedpb.GatewayRequest` directly. |
| **Test fake** | `internal/testutil/mocks::FakeGatewayBuilder` records every `(class, role, profile)` triple it was asked for and can be primed to fail, so a test can assert *what was requested* rather than only what came back. Route-level behavior is faked separately at the gateway client. |
| **Why it exists** | **This seam is the product's central claim.** AI Gateway's fail-closed behavior is a property of the *profile*, not of the privacy class: `PROFILE_LOCAL_FIRST` is documented to fall back to a permitted remote provider, so a confidential document sent under `local-first` routes remote and the gateway is behaving *correctly*. The residency guarantee therefore lives entirely in the class→profile mapping, and a second construction site anywhere in the tree voids it silently — no test fails, no finding fires, the document just leaves. Enforced by an AST check (`internal/gatewayreq/exclusivity_test.go`) asserting exactly one construction site exists, in the same spirit as `storage-manager`'s routed-database check. Covers `DOC-P0-013` and `DOC-P0-026`. |

### Corpus authority boundary (no ledger content indexed here)

| | |
|---|---|
| **Seam** | The `retrieval` domain's index population path, and what it is structurally forbidden to reach |
| **Interface** | `internal/retrieval/indexer.go::Indexer` (`Index(ctx, DerivationVersion) error`) sources rows exclusively from `anchors` (units), `enrichment` (embeddings) and `corpus` (collection membership). It takes no ledger client, no `handoff` repository, and no federated result reader in its dependency set. |
| **Production wiring** | Composed in `main.go` from those three repositories only. The `handoff` domain publishes *outward* and is never a source for the indexer; the dependency edge runs one way by construction. |
| **Test fake** | Not a fake — an **AST check** (`internal/retrieval/authority_test.go`) asserting that no file under `internal/retrieval/` imports a ledger client, `internal/handoff`, or a search-hub result type. Same pattern and same spirit as the `gatewayreq` exclusivity check. |
| **Why it exists** | This is the boundary that has already drifted once, in both directions. An early decision forbade a `retrieval` domain outright — an over-correction — and the superseding decision narrowed the real obligation to: *never index ledger content here, and never expose recall over another scenario's corpus.* The failure mode is not "two vector stores exist"; it is the same content indexed twice with no defined authority, which produces answers that disagree with no way to say which is right. Sources and findings are separate content classes with separate lifecycles, and each store's authority over its own class is what keeps `search-hub` federation coherent rather than duplicative. Prose alone did not hold this boundary before; an AST check makes it survive an agent who has not read `DECISIONS.md`. |

### Anchor resolver (version-crossing resolution)

| | |
|---|---|
| **Seam** | Resolving an anchor to a source region, across derivation versions |
| **Interface** | `internal/anchors/resolver.go::Resolver` (`Resolve(ctx, Anchor, atVersion int) (SourceRegion, error)`) with one implementation per anchor kind behind a dispatch on `Anchor.Kind`. |
| **Production wiring** | `main.go` composes `GeometricResolver` (reads original bytes through the routed file-store seam) and `LogicalResolver` (reads `anchor_alignments` from SQLite). Neither reads a parse output from the artifact store. |
| **Test fake** | `internal/anchors/mocks::FakeResolver` plus a fixture corpus with a known v1→v2 reparse, so cross-version cases are table tests rather than integration runs. |
| **Why it exists** | `DOC-P0-009` promises a v1 anchor still resolves after v2, while parse outputs are `regenerable: true` and prunable. Those two facts are only compatible if resolution never depends on a prior version's parse output. Making resolution a seam with two typed implementations forces that: `geometric` resolves against the original bytes (`regenerable: false`), `logical` resolves through a stored alignment, and an unaligned `logical` anchor returns `ErrUnresolved` rather than a plausible wrong region. Without the seam, the natural implementation reads the old parse output and works perfectly until the first prune. |

## Write-spine seams (P2 — designed, not scaffolded)

The five seams below belong to the generation spine (`templates`,
`composition`, `render`). They are declared now so the boundary is
designed before anyone implements it, and **no code exists for them
yet** — the same treatment `handoff` received. See the write-spine rows
in [`DECISIONS.md`](DECISIONS.md).

### Spec authority boundary (nothing writes rendered bytes)

| | |
|---|---|
| **Seam** | The `composition` domain's mutation path, and what it is structurally forbidden to touch |
| **Interface** | `internal/composition/mutator.go::Mutator` (`Apply(ctx, SpecVersion, Mutation) (SpecVersion, error)`) writes only `spec_versions`, `blocks`, `source_bindings` and `overrides`. It takes no artifact-store writer, no render client, and no blob handle in its dependency set. |
| **Production wiring** | Composed in `main.go` from the composition repositories only. `render` reads specs and writes bytes; `composition` writes specs and never bytes. The dependency edge runs one way by construction. |
| **Test fake** | Not a fake — an **AST check** (`internal/composition/authority_test.go`) asserting no file under `internal/composition/` imports the artifact store, a blob writer, or `internal/render`. Same pattern as the `gatewayreq` exclusivity check and the corpus-authority boundary. |
| **Why it exists** | **The spec is the write spine's authority** and rendered bytes are a derivation of it. An edit path that could write bytes would let the artifact diverge from the record that explains it — and the moment it does, template switching, `refresh`, revert-as-undo and document diff all silently stop being correct while every test still passes. The natural implementation of "let the user nudge this heading" reaches for the output file; the seam is what makes that a compile-time impossibility rather than a code-review habit. Covers `DOC-P2-010` and `DOC-P2-022`. |

### Renderer registry and router

| | |
|---|---|
| **Seam** | Selecting a renderer chain for a (spec, template, target) triple |
| **Interface** | `internal/render/router.go::Router` (`Select(ctx, Spec, TemplateVersion, Target) (Chain, error)`) reads `registry.json` and dispatches on declared **fidelity**, never on a target name. Each `Renderer` is `Render(ctx, Chain, Spec) (Bytes, Alignment, []Unrepresentable, error)`. |
| **Production wiring** | `main.go` loads the embedded registry once and constructs the router; each renderer is a resource-CLI-backed implementation behind the `Renderer` interface. No renderer type is referenced outside its constructor. |
| **Test fake** | `internal/render/mocks::FakeRenderer` (canned bytes, canned alignment, primeable `Unrepresentable` list and per-call error knobs), plus a registry fixture. A test asserting the registry↔`render-matrix.md` agreement runs alongside, mirroring the format-matrix test. |
| **Why it exists** | Mirrors the handler registry exactly, for the reason the `anydoc` rows record: a good library is not the architecture. Returning `Alignment` and `[]Unrepresentable` from the same call is deliberate — both are byproducts the renderer alone knows, and a signature that made either optional would let a renderer succeed while silently dropping an element or losing the anchor mapping. Covers `DOC-P2-012`, `DOC-P2-016`, `DOC-P2-017`. |

### Source binding resolver (refresh is not render)

| | |
|---|---|
| **Seam** | Re-running a source binding to a snapshotted value |
| **Interface** | `internal/composition/sources.go::Resolver` (`Resolve(ctx, SourceBinding) (Resolution, error)`) with one implementation per binding kind — corpus anchor, `command-center` query, `content-desk` claim, `chart-generator` render id. |
| **Production wiring** | `main.go` composes the resolvers a deployment actually has; an unconfigured kind resolves to `ErrBindingUnresolvable` rather than being absent, so the failure is named. |
| **Test fake** | `internal/composition/mocks::FakeResolver` returning canned resolutions with controllable timestamps, so `refresh` semantics are table tests rather than cross-scenario integration runs. |
| **Why it exists** | A binding is a **re-runnable descriptor, not a captured value** — that is the entire reason `refresh` can exist as a verb distinct from `render`. But a live binding with no snapshot makes every past render unexplainable, so each resolution is recorded with its timestamp. The seam holds both halves together and keeps four upstream scenarios out of the render path. Covers `DOC-P2-011` and `DOC-P2-019`. |

### Brand token resolver (templates never hold literals)

| | |
|---|---|
| **Seam** | Resolving a template's presentation references to concrete values |
| **Interface** | `internal/templates/brand.go::TokenResolver` (`Resolve(ctx, TokenRef) (Value, error)`). Template validation rejects any presentation value that is not a `TokenRef`. |
| **Production wiring** | `main.go` wires a `brand-manager`-backed resolver. On resolution failure, template validation **fails loudly and never falls back to a literal**. |
| **Test fake** | `internal/templates/mocks::FakeTokenResolver` (canned token table, primeable failures) so template tests need no running `brand-manager`. |
| **Why it exists** | A template carrying a literal color or font drifts from brand permanently and silently, and the drift is only visible to someone who remembers what the brand used to be. Making the resolver a seam lets validation reject literals structurally, which is what turns a rebrand into a corpus-wide re-render rather than a corpus-wide template edit. The no-fallback rule matters as much as the seam: a silently un-branded render is worse than no render, because it ships. Covers `DOC-P2-013`. |

### Composer chat client boundary (no privileged path)

| | |
|---|---|
| **Seam** | What the in-UI agent chat is allowed to call |
| **Interface** | `ui/src/features/composer/agent/` may construct only generated Connect clients (`compositionClient`, `templatesClient`, `renderClient`). It has no bespoke endpoint, no direct repository access, and no server-side companion that bypasses the proto surface. |
| **Production wiring** | The chat panel composes the same public verbs the CLI binds. Server-side, an agent turn enters through the ordinary handlers and through `internal/gatewayreq` under the document's privacy class. |
| **Test fake** | Not a fake — an **AST check** (`ui/src/features/composer/agent/parity_test.ts`) asserting the agent surface imports only generated clients, plus a CLI-parity test asserting every verb the chat can invoke has a `cli/manifest.json` binding. |
| **Why it exists** | **This seam is the parity guarantee.** The requirement is that an agent spawned outside this scenario, loading the same skill, can do everything the chat panel can. Stated as prose that decays the first time someone adds a convenient shortcut; stated as a check it holds. Its corollary is the useful one: a chat interaction that cannot be expressed as a sequence of public verbs is a **missing verb, not a chat feature** — which is why there is no `chat` domain. Covers `DOC-P2-021`, `DOC-P2-023`, `DOC-P2-025`, `DOC-P2-027`. |

<!-- EXAMPLE-DOMAIN:notes START -->
### Example domain — `notes` (removed by `template-manager detemplate`)

The template ships a worked example domain as a copyable reference for
the abstract seams above. It is never product scope; `vrooli scenario
detemplate <scenario>` removes it. The seams below show the concrete
shape every abstract `<domain>` seam takes — repository, service,
attachment repository/service, per-domain schema, and the measures
registry — wired exactly as a real domain would be.

#### notes.Repository (notes persistence)

| | |
|---|---|
| **Seam** | Notes persistence (CRUD) |
| **Interface** | `internal/notes/repository.go::Repository` (`Create`, `Get`, `List`, `Count`) |
| **Production wiring** | `handlers/notes/module.go::Module(...)` constructs `notes.NewSQLiteRepository(db, clk)` and passes it into `notes.NewService(repo)`. `main.go` only sees the returned `module.Module`; note-specific dependencies do not appear on `server.Deps`. Wire shape lives in `packages/proto/schemas/document-manager/v1/notes/notes.proto`. |
| **Test fake** | `internal/notes/mocks::FakeRepository` (co-located with the domain — embeds `repokit.SliceRepo` for in-memory CRUD with per-method error knobs `CreateErr` / `GetErr` / `ListErr` + atomic call counters, plus a domain-specific `Count` knob `CountOut` / `CountErr` and a `CountWindows` recorder the generic substrate can't express). Used by `internal/notes/service_test.go` to drive the service against a controllable persistence layer. |
| **Why it exists** | Repository owns the persistence contract — sqlite SQL today, anything else tomorrow. `Count` is a real aggregate (`SELECT COUNT(*) … WHERE created_at >= ? AND < ?`) backing the `notes count` measure, kept on the repository so the measure answer is exact regardless of row volume. The handler depends on `notes.Service`, not directly on the repository, so a backend swap doesn't ripple through transport. The repository test in `internal/notes/sqlite_test.go` substitutes the real handle to pin SQL semantics (ordering, limit, RFC3339 round-trip, the count range). |

#### notes.Service (notes application layer)

| | |
|---|---|
| **Seam** | Notes application surface (validation, defaults, cross-handler policy) |
| **Interface** | `internal/notes/service.go::Service` (`Create(CreateInput) → Note`, `Get(id) → Note`, `List(limit) → []Note`, `CountInWindow(from, to) → int`) |
| **Production wiring** | `handlers/notes/module.go::Module(db, clk, logger)` constructs `notes.NewSQLiteRepository(db, clk)` then `notes.NewService(repo)` then `NewConnectHandler(Deps{Service: svc, Logger: logger})` — fully internal to the notes module. `main.go` only sees the `module.Module` returned from that constructor; per-domain services don't appear on `server.Deps`. The handler imports `internal/notes` for both the interface and the typed sentinels (`ErrInvalidNote`, `ErrNoteNotFound`) it translates at the transport edge. |
| **Test fake** | `internal/notes/mocks::FakeService` (co-located with the domain — records `CreateInputs`, returns canned `CreateOut` / `GetByID` / `ListOut`, per-method error knobs). Used by `handlers/notes/connect_handler_test.go` to drive the handler without validation/repository plumbing in scope. |
| **Why it exists** | Validation (`title required` after whitespace trim) and default substitution (`defaultListLimit = 100` when caller passes 0) are business policy, not transport policy. Putting them in the service keeps the handler thin and makes the same rules reachable from any future surface (batch jobs, scheduled imports, additional RPCs) without copy-paste. Two-mock split (`FakeRepository` for service tests, `FakeService` for handler tests) means handler tests don't seed sqlite-shaped state to assert routing. |

#### Measures serve registry (the `notes count` reference measure)

| | |
|---|---|
| **Seam** | The measures-go serve substrate mounted at `/measures` |
| **Interface** | `packages/measures-go::Registry` (`Register(decl, ComputeFunc)`, `Execute`, `Handler`) + the `ComputeFunc`, `Matcher`/`Executor`/`Completer` seams in `measures-go`. Per-param resolution is deterministic for the canonical `time_window` type (`measures.ResolveToken`, no LLM). |
| **Production wiring** | `handlers/notes/measures.go::MeasuresHandler(db, clk)` builds its own `notes.Service` over the shared db, registers the `notes.count` declaration (`notesCountDeclaration()`) with a compute func that resolves the window token and calls `Service.CountInWindow`, and returns `Registry.Handler()`. `main.go` mounts it once: `rootMux.Handle("/measures/", http.StripPrefix("/measures", notesMeasures))`. The same `Service.CountInWindow` backs the `CountNotes` Connect RPC, so the RPC and the measure can never report different numbers. |
| **Test fake** | `measures-go` test doubles (`mocks.FakeService` for the compute func + `clockmocks.FakeClock` for deterministic windows). `handlers/notes/measures_test.go` registers the declaration and executes it through the real `Registry` — the unit-level mirror of the `measures-health` behavioral probe (asserts the scalar value, the resolved `[from, to)`, and stamped provenance). |
| **Why it exists** | A **measure** is a named, typed, parameterized analytical query declared once so `search-hub` can match a natural-language question, fill params deterministically, and (for read-only, run-eligible measures) auto-answer. `notes.count` is the reference: a `time_window`-parameterized scalar at full tier. The manifest `measure` block (`cli/manifest.json`) + the bound proto request (`CountNotes`) are the static SSOT `cli-health` / `measures-health` validate; this registry is the runtime serve side `measures-health` harvests (`/measures/declarations`) and probes (`/measures/execute`). When you replace the notes example domain, delete this file and the one `main.go` mount line, then declare your own domain's measure. See `docs/concepts/MEASURES.md` (repo-level) and the `packages/measures-go` README. |

#### notes.AttachmentsRepository (attachment metadata persistence)

| | |
|---|---|
| **Seam** | Note attachment metadata persistence |
| **Interface** | `internal/notes/repository.go::AttachmentsRepository` (`CreateAttachment`, `ListAttachmentKeys`) |
| **Production wiring** | `handlers/notes/module.go::Module(...)` constructs `notes.NewSQLiteAttachmentsRepository(db, clk)` (declared in `internal/notes/sqlite.go`, methods in `attachments_sqlite.go`) and passes it into `notes.NewAttachmentsService(...)`. The opaque file bytes go to `BlobStore` (separate seam below); only the typed metadata row passes through this interface. |
| **Test fake** | `internal/notes/mocks::FakeAttachmentsRepository` (co-located with the domain — in-memory `Attachments` slice, per-method error knobs `CreateErr` / `ListErr`, atomic call counters, UploadedAt backfill mirroring the sqlite repository). Used by `internal/notes/attachments_service_test.go` to drive the attachments service against a controllable persistence layer. |
| **Why it exists** | Splitting attachment-metadata persistence from notes persistence keeps the per-method surface narrow (the notes repository never grows attachment-shaped methods) and lets the attachments service remain transport-agnostic. The repository test in `internal/notes/sqlite_test.go::TestSQLiteRepository_AttachmentMetadataRoundTrip` substitutes the real handle to pin SQL semantics; service tests use the fake. |

#### notes.AttachmentsService (attachment application layer)

| | |
|---|---|
| **Seam** | Note attachment application surface (validation, parent-note lookup, repository delegation) |
| **Interface** | `internal/notes/attachments_service.go::AttachmentsService` (`Create(CreateAttachmentInput) → Attachment`) |
| **Production wiring** | `handlers/notes/module.go::Module(...)` constructs `notes.NewAttachmentsService(notesRepo, attachmentsRepo)` then passes it as `AttachmentsDeps.Service` into `NewAttachmentsHandler(...)`. The handler is the multipart REST exception (the only non-Connect transport in the notes domain); the service stays unaware of multipart and HTTP. |
| **Test fake** | `internal/notes/mocks::FakeAttachmentsService` (records `CreateInputs`, returns canned `CreateOut` or synthesises an Attachment from the input, gated on `CreateErr`). Available for any future handler test that wants to assert routing/multipart wiring without standing up the real notes-and-attachments service tree. |
| **Why it exists** | Attachment validation (note id + key required after trim, positive size, parent note must exist) is business policy; multipart parsing and BlobStore I/O are transport policy. Keeping them split means a future scenario that adds a non-multipart attachment surface (CLI direct upload, scheduled import, gRPC stream) reuses the same validation without copy-paste. Two-mock split (`FakeAttachmentsRepository` for service tests, `FakeAttachmentsService` for handler tests) mirrors the notes Repository/Service convention. |

#### notes.Schema (per-domain schema)

| | |
|---|---|
| **Seam** | Notes domain SQL contribution |
| **Interface** | `internal/notes/schema.go::Schema() string` (consumed via `handlers/notes/module.go::Schema` re-export, then `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` includes `apidb.SchemaProviderFunc(notesH.Schema)`; applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | `internal/notes/sqlite_test.go::newSchemaDB` uses `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` with the system + notes providers. Repository tests get a fresh table without touching the central registry. |
| **Why it exists** | Domain ownership of the schema. Adding a column lands in the same diff as the Go change. Deleting `internal/notes/` deletes the table definition with it, so removed domains do not leave tables created on boot. The `handlers/notes/module.go::Schema` re-export keeps the registry's import surface narrow — it imports handler packages, not their internal peers. |

#### UI per-domain client (`ui/src/api/notes.ts`)

| | |
|---|---|
| **Seam** | UI ↔ API per-domain endpoints (canonical CRUD reference: `api/notes.ts`) |
| **Module** | `ui/src/api/notes.ts` exports `notesClient = createClient(NotesService, transport)` and `uploadAttachment(...)` for the multipart REST exception. |
| **Production wiring** | Feature components wire generated client methods through `useQuery` / `useMutation`, for example `notesClient.listNotes({})` and `notesClient.createNote({ title, body })`. Multipart flows call `uploadAttachment`, which uses `FormData` plus `uploadFile()` and returns generated metadata. |
| **Test fake** | Component tests use inline `vi.mock("./api/notes", async (importOriginal) => ...)` and replace `notesClient` methods or `uploadAttachment`. Factories build generated proto types, including `Timestamp` values. |
| **Why it exists** | The canonical per-domain client pattern. Mirror this shape when adding a second domain client: export the generated Connect client, keep binary-upload helpers beside it when needed, and let components consume typed results rather than hand-written response interfaces. |

#### Domain package layout (`internal/notes/`)

The notes package is the canonical layout example for the abstract
"Domain-scoped packages" section above — copy its shape:

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
<!-- EXAMPLE-DOMAIN:notes END -->

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
