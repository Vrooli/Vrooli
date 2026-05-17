# Seams — Audio Tools

> Plan: `~/.vrooli/plans/audio-tools-greenfield-scenario-web-console-adoption.md`

## Layering (2026-05-17 post-extraction audit)

- `internal/ai/{stt,tts,summarize}chain` are the canonical provider
  orchestrators. Every `handlers/*` and `bootstrap` import goes through
  them. Primitives (`internal/{stt,tts,summarize}`) never import the
  chain layer — the dependency is one-way.
- `internal/text/normalizer` is shared by `handlers/tts` and
  `internal/summarize`. See `internal/text/normalizer/doc.go` for the
  consumer list; if a third domain appears, revisit
  [`DECISIONS.md`](DECISIONS.md) (2026-05-17 entry).

## Quick index (audio-tools build)

| Seam | Type | Producer | Consumer |
|---|---|---|---|
| `sttchain.Provider` / `ttschain.Provider` / `summarizechain.Provider` | Interface | `internal/ai/{stt,tts,summarize}chain/provider_*.go` | matching `handlers/{stt,tts,summarize}/` |
| `sttchain.BYOKAdapter` / `ttschain.BYOKAdapter` / `summarizechain.BYOKAdapter` | Interface | `internal/byok/*.go` | chain `BYOKProvider` registry dispatch |
| `sttchain.VrooliClient` / `ttschain.VrooliClient` / `summarizechain.VrooliClient` | Interface | `integrations/lpbs/clients/*_client.go` | chain `VrooliProvider` |
| `tts.VerifyCatalog` | Pure function | `internal/tts/voice_catalog.go` | `main.go` startup gate |
| `session.Session` pub/sub | Concrete | `internal/session/session.go` | `handlers/stt/stream_ws.go`, `handlers/session` |
| Browser-voice WS | Concrete | `handlers/stt/stream_ws.go` | `main.go` route mount |
| `audiotools.URLResolver` (consumer-side) | Interface | `scenarios/web-console/api/integrations/audiotools/discovery.go` | `audiotools.Client.refresh()` |
| `store.{ProviderConfig,BYOK,VoiceOverride,Usage,Wakeword,Speaker,STTStreamConfig,TTSConfig,Playback}Store` | Concrete (sql) | `internal/store/*.go` | every admin handler (settings/usage/stt/tts) |
| `byokstore.Store` / `byokstore.Encryptor` | Concrete | `internal/byokstore/{store,encryptor,fingerprint}.go` | `handlers/settings` and chain BYOK resolver |
| `usagereport.Recorder` | Interface | `internal/usagereport/recorder.go` (local SQLite write path for the UsageService dashboard) | `handlers/summarize` (other chain-adjacent handlers wired as they land) |
| `lpbs.RemoteReporter` | Concrete | `integrations/lpbs/remote_reporter.go` (remote LPBS hop; flag-off until the gateway lands) | wired into `main.go` once the LPBS gateway ships |
| `chains.Coordinator` | Concrete | `internal/ai/chains/chains.go` | `handlers/settings` UpdateProviderConfig — live chain Reconfigure |
| `chains/tiered.Coordinator[Req,Resp]` | Generic | `internal/ai/chains/tiered/tiered.go` | embedded into each of `sttchain.Chain`, `ttschain.Chain`, `summarizechain.Chain` (method promotion) — owns BYOK->Vrooli->Local routing, availability cache, Reconfigure, Probe, Eligible |
| `chains/tiered.ProviderSet[Req,Resp]` + `chains/tiered.NewChainFromSet` | Generic | `internal/ai/chains/tiered/set.go` | per-chain `NewChain` describes its domain via one declarative ProviderSet literal (tiers + Route + IsTerminal + AllFailed) and hands it to NewChainFromSet — replaces the duplicated Coordinator-wiring boilerplate that lived in each chain.go |
| `chains/tiered.Tier[Req,Resp]` | Struct (function fields) | `internal/ai/chains/tiered/tiered.go` | per-chain `sttTier`/`ttsTier`/`sumTier` generic helpers (pointer-shaped type param to dodge typed-nil-in-interface) wrap concrete provider methods |
| `sttchain.Chain.Probe` / `ttschain.Chain.Probe` / `summarizechain.Chain.Probe` | Inherited from embedded `*tiered.Coordinator` | `internal/ai/chains/tiered/tiered.go` | `handlers/tts` GetStatus + `cli/domains/settings` (`settings providers`) — returns `tiered.ProbeResult` |
| `stt.MultipartTranscribeHandler` / `audio.multipartTranscodeHandler` | Concrete | `handlers/{stt,audio}/` | UI multipart upload paths |
| `audio.Runner` + `audio.DefaultRunner` + `audio.SetFfmpegAvailableForTest` | Interface + var + test seam | `internal/audio/transcode.go` | `handlers/audio` unit tests substitute a fake Runner and seed ffmpeg presence so happy-path / error branches run without an ffmpeg binary on PATH |
| `stt.StreamWSHandler` | Concrete | `handlers/stt/stream_ws.go` | mounts `/api/v1/voice/stream` over `voice.Service.HandleStreamWS` |
| `stt.Segmenter` | Concrete | `internal/stt/segmenter/` | WS handler + Connect bidi handler (one impl, two transports) |
| `stt.StrategySelector` | Concrete | `internal/stt/selector.go` | `stt.Segmenter` at session start |
| `stt.StreamingStrategy` | Interface | `internal/stt/strategy/{vad_segment,overlap_agree,passthrough}.go` | `stt.StrategySelector` |
| `sttchain.ProviderTraits` | Struct (replaces `StreamingCapability() bool`) | `internal/ai/sttchain/interface.go` | `stt.StrategySelector` |
| `capabilities.ResourceController` | Interface | `internal/capabilities/lifecycle.go` (production impl: `CLIController` in `lifecycle_cli.go` shells out to `vrooli resource …`) | `handlers/provider_lifecycle/connect_handler.go` — single chokepoint for lifecycle shell-outs; tests substitute recording fakes |
| `capabilities.Registry.ResolveForce` | Concrete (additive) | `internal/capabilities/registry.go` | `handlers/health_status` (RefreshProviderHealth) and `handlers/provider_lifecycle` (post-mutation cache-bust) |

## Cross-scenario boundaries

| Direction | Mechanism |
|---|---|
| consumer → audio-tools (API) | Connect-RPC + multipart REST exceptions + browser-voice WS |
| consumer UI → audio-tools UI | `@audio-tools/embed` workspace package re-exported via consumer's `domains/audio/index.ts` |
| audio-tools → LPBS (flag-off) | HTTP POST per capability |
| audio-tools → BYOK third parties | HTTP POST per adapter (recorded fixtures via go-vcr) |

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
`packages/proto/schemas/audio-tools/v1/<domain>/<file>.proto` —
NOT in interfaces. The generated Go + TypeScript types are the
canonical types every test, handler, and UI component reads from.

The `health` proto in `packages/proto/schemas/audio-tools/v1/health/`
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

### envx.Reader (process environment)

| | |
|---|---|
| **Seam** | Process environment variable reads |
| **Interface** | `internal/envx/reader.go::Reader` (`Get(key) string`) |
| **Production wiring** | `main.go` / `internal/bootstrap` constructs `envx.OS{}` once and passes it to every domain that reads env vars (whisper URL, summarize model, BYOK key envelope key, etc.). |
| **Test fake** | `internal/testutil/mocks::FakeEnv` (map-backed, records reads, `Set(key,val)` mutator). |
| **Why it exists** | Domain code that calls `os.Getenv` directly forces tests to use `t.Setenv` — which mutates process-wide state and races under `t.Parallel`. The seam lets domain tests inject a per-call environment without touching the real process env. Bootstrap (the composition root) remains exempt from this rule. |

### logx.Logger (structured logging)

| | |
|---|---|
| **Seam** | Structured log emission from domain code |
| **Interface** | `internal/logx/logger.go::Logger` (`Printf(format, args...)`) |
| **Production wiring** | `main.go` / `internal/bootstrap` constructs `logx.Std{L: log.Default()}` once and threads it via `Deps.Logger` into every handler (`handlers/{audio,diagnostics,health_status,provider_lifecycle,session,settings,stt,summarize,tts,usage}/`) and the three `internal/` shims (`middleware`, `tts/service`, `usagereport/recorder`). `Deps.Logger` is a required field — no `nil` fallback. |
| **Test fake** | `internal/testutil/mocks::FakeLogger` (records every Printf call, exposes `Entries()` snapshot, `Reset()` between sub-tests). |
| **Why it exists** | Domain code calling `log.Printf` directly writes to a process-global default logger — tests can't capture or assert on those lines without redirecting stderr. The seam allows assertions like "the pipeline logged exactly one warning containing 'whisper unreachable'" without global state. |
| **Drift gate** | `rg -n 'log\.Default\(\)' scenarios/audio-tools/api/ -g '!*_test.go' -g '!internal/bootstrap/**' -g '!internal/logx/**' -g '!main.go'` must return empty. |

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

### sttchain.Provider (STT chain tier)

| | |
|---|---|
| **Seam** | One tier (Local / BYOK / Vrooli) of the STT provider chain |
| **Interface** | `internal/ai/sttchain/interface.go::Provider` (`Type`, `IsAvailable`, `Transcribe`, `Model`, `Traits`, `TranscribeStreaming`) |
| **Production wiring** | `main.go` builds the three concrete tiers (`NewLocalProvider`, `NewBYOKProvider`, `NewVrooliProvider`) and hands them to `sttchain.NewChain(Options{…})`. Per-request, `Chain.Execute` selects the first eligible tier in BYOK → Vrooli → Local order. |
| **Test fake** | `internal/ai/sttchain/mocks::FakeProvider` (configurable `Tier`, `Traits`, `Result` / `Err`, optional `TranscribeFn` / `StreamFn`, `Calls` counter). |
| **Why it exists** | Each tier has a different upstream (Whisper binary, vendor adapters, LPBS gateway) and a different failure mode. Putting the precedence + fallback + insufficient-credits short-circuit in `Chain` keeps that policy in one place; the per-tier struct only translates its own backend. Chain-orchestration tests in `chain_test.go` substitute `FakeProvider` to assert routing without spinning real backends. |

### sttchain.BYOKAdapter (per-vendor STT adapter)

| | |
|---|---|
| **Seam** | One vendor (openai-whisper, deepgram) implementing the BYOK STT contract |
| **Interface** | `internal/ai/sttchain/provider_byok.go::BYOKAdapter` (`ID`, `Transcribe`, `IsAvailable`, `Model`, `StreamingCapability`, `TranscribeStreaming`) |
| **Production wiring** | `internal/byok/registry.go::NewRegistries()` constructs the per-capability registry maps; `main.go` passes them into the chain's `NewBYOKProvider`. The chain dispatches per-request by `BYOKProvider` header (`envelope.HeaderProvider`). |
| **Test fake** | `internal/ai/sttchain/mocks::FakeBYOK` (configurable ID, availability, result, error, streaming flag + `StreamFn`). |
| **Why it exists** | Adding a vendor means adding a registry row and an adapter file — no chain-side changes. Tests on the chain don't need real vendor adapters; tests on the adapters use `httptest.NewServer` for wire format and a `FakeDoer` for payload assertions. |

### sttchain.VrooliClient (STT Vrooli/LPBS client)

| | |
|---|---|
| **Seam** | LPBS audio-gateway client for the STT Vrooli tier |
| **Interface** | `internal/ai/sttchain/provider_vrooli.go::VrooliClient` (`Transcribe`, `IsAvailable`, `Model`) |
| **Production wiring** | `integrations/lpbs/clients/stt_client.go` implements the interface against the LPBS HTTP gateway; `main.go` injects it through `NewVrooliProvider`. |
| **Test fake** | `internal/ai/sttchain/mocks::FakeVrooliClient` (configurable availability, result, error, optional `TranscribeFn`). |
| **Why it exists** | The chain MUST NOT import the lpbs package directly (cross-domain coupling) and MUST be able to test the `ErrInsufficientCredits` short-circuit without spinning LPBS. The interface gives both. |

### ttschain.Provider / BYOKAdapter / VrooliClient (TTS chain seams)

| | |
|---|---|
| **Seam** | TTS-side mirrors of the STT chain seams (same shape, different payload) |
| **Interface** | `internal/ai/ttschain/{interface,provider_byok,provider_vrooli}.go` |
| **Production wiring** | Mirrors STT — `main.go` builds `NewBYOKProvider`/`NewVrooliProvider`/`NewLocalProvider` and hands them to `ttschain.NewChain`. |
| **Test fake** | `internal/ai/ttschain/mocks::{FakeBYOK,FakeVrooliClient}` (plus the chain's own provider-level fakes via the interface). |
| **Why it exists** | TTS has identical tier-precedence + insufficient-credits + buffered-fallback semantics; sharing the shape across chains keeps both code paths reasoning-equivalent. |

### summarizechain.Provider / BYOKAdapter / VrooliClient (summarize chain seams)

| | |
|---|---|
| **Seam** | Summarize-side mirrors of the STT chain seams |
| **Interface** | `internal/ai/summarizechain/{interface,provider_byok,provider_vrooli}.go` |
| **Production wiring** | Mirrors STT — `main.go` constructs the three tiers from OpenRouter (BYOK), LPBS chat (Vrooli), and the local Ollama-backed `summarize.Summarizer` (Local). |
| **Test fake** | `internal/ai/summarizechain/mocks::{FakeBYOK,FakeVrooliClient}`. |
| **Why it exists** | Same as above — uniform shape across three chains keeps the test architecture portable and lets future tiers (Phase D streaming, additional vendors) follow the same drift gates. |

### audio.Runner (ffmpeg/ffprobe process boundary)

| | |
|---|---|
| **Seam** | The single-method `Run(ctx, name, stdin, args...) ([]byte, error)` surface every call to ffmpeg / ffprobe goes through. |
| **Interface** | `internal/audio/transcode.go::Runner` (production wired via `audio.DefaultRunner = execRunner{}`) |
| **Production wiring** | `runFfmpeg` / `runFfprobeJSON` in `internal/audio/` delegate to `DefaultRunner.Run(...)`. Tests swap `DefaultRunner` for the duration of the test, paired with `audio.SetFfmpegAvailableForTest(true, true)` to bypass the binary-presence cache. |
| **Test fake** | `internal/audio/mocks::FakeRunner` (records `Calls`, returns canned `Stdout` / `Err`, optional `Respond(name, args)` for argv-aware behaviour). |
| **Why it exists** | The binaries aren't available in CI runners or unit-test envs by default. Without the seam, every ops-level test (Transcode, Trim, Volume, Normalize, Split, Merge, Probe) would require ffmpeg on PATH — flaky at best, blocked at worst. The fake lets the same suite pin argv shape and per-format branches in isolation. |

### capabilities.Checker (runtime capability probe)

| | |
|---|---|
| **Seam** | One pluggable capability checker (audio, llm, scenario, …) registered with the cached registry. |
| **Interface** | `internal/capabilities/registry.go::Checker` (`Check(ctx) (Status, string)`) |
| **Production wiring** | `main.go` builds per-capability checkers (ffmpeg presence, ollama reachability, etc.) and registers them with `capabilities.NewRegistry`. Health and admin handlers query the registry. |
| **Test fake** | `internal/capabilities/mocks::FakeChecker` (configurable `Status` + `Message`, atomic call counter — proves caching short-circuits redundant probes). |
| **Why it exists** | Real checkers shell out, dial HTTP, or load model artifacts. Fakes let registry tests assert TTL caching and fan-out without those external dependencies. |

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
| **Interface** | `internal/modulekit/module.go::Module` (`Name string`, `Mount func(r *mux.Router)`, `Endpoints []EndpointDescriptor`). Data type, not behaviour — modules don't have methods. |
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
| **Why it exists** | Some bits don't belong to any one domain — postgres extensions, type definitions, reporting views. Putting them in a domain package would force fictional ownership. The system home is honest: cross-cutting goes here, single-domain bits go in `internal/<dom>/schema.sql`. |

### Per-domain schema (audio-tools)

| | |
|---|---|
| **Seam** | Domain-local SQL contribution to the boot-time schema fan-out |
| **Interface** | `internal/store/*.go::Schema() string` (each store package exports a schema string consumed via `api-core/database.SchemaProvider`) |
| **Production wiring** | `internal/modules/registry.go::AllSchemas()` lists the per-domain providers (byok, voice_overrides, usage, wakeword, speaker, stt_stream_config, tts_config, playback_events, provider_config); applied at boot via `apidb.EnsureSchemas`. |
| **Test fake** | `internal/store/*_sqlite_test.go` (one per domain) uses `db.NewSQLite(t)` + `apidb.EnsureSchemas(...)` with just the system + domain providers in scope. |
| **Why it exists** | Domain ownership of schema means adding a column lands in the same diff as the Go change, and removing a domain removes its tables. The registry's import surface stays narrow — it imports handler packages, not internal peers. |

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

## Streaming chain seams (audio-tools-web-console-restoration plan)

### `sttchain.Provider.TranscribeStreaming` / `StreamingCapability`
- **Owner:** `api/internal/ai/sttchain/interface.go`.
- **Production wires:** `LocalProvider`, `BYOKProvider`, `VrooliProvider`
  return `false`/`(nil,nil)` today; native streaming implementations
  land in plan Phases D (Local segmenter) and E (Deepgram WS, OpenAI
  Realtime). The chain's `Stream()` method calls these to negotiate a
  streaming-capable tier; if none accepts, it falls back to a buffered
  unary mode that emits a synthetic Segment + Done event pair with
  `DoneEvent.FellBackToUnary=true`.
- **Test substitutes:** in-process fake `Provider` defined per-test
  (see `chain_stream_test.go`); `goleak.VerifyNone(t)` guards every
  streaming test.

### `ttschain.Provider.SynthesizeStreaming` / `StreamingCapability`
- **Owner:** `api/internal/ai/ttschain/interface.go`.
- Mirrors the STT shape. The chain's `Stream()` falls back to a single
  `AudioFrame{IsFinal=true}` carrying the full unary audio when no tier
  declares streaming. Connect handler in `handlers/tts/connect_handler.go::SynthesizeStream`
  forwards frames as they arrive.

### `stt.Segmenter` (transport-free streaming orchestrator)
- **Owner:** `api/internal/stt/segmenter/segmenter.go`
  (see [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md)).
- **Production wires:** constructed once per streaming session by both
  the browser WS handler (`handlers/stt/stream_ws.go`) and the Connect
  bidi handler (`handlers/stt/transcribe_stream.go`). Owns the session lifecycle,
  the chunk-in/event-out channel pair, observer fanout to
  `session.Registry`, and the cancellation/barge-in fan-out into TTS.
- **Test substitutes:** in-process fakes feed a canned audio channel
  and assert the emitted event sequence; parity test runs the same
  WAV through both transports' Segmenter wiring and asserts equivalent
  event projections.

### `stt.StrategySelector` (decision boundary)
- **Owner:** `api/internal/stt/selector.go`.
- **Production wires:** called by the Segmenter at session start.
  Consumes the provider chain's negotiated tier, the operator's
  `StreamConfig` levers, and each provider's `ProviderTraits` to
  return a concrete `StreamingStrategy` bound to the chosen
  `sttchain.Provider`. The strategy × provider compatibility matrix
  lives here and only here.
- **Test substitutes:** table-driven tests assert that every
  (strategy, provider) cell from the matrix in
  [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md#strategy--provider-compatibility)
  either picks the documented strategy or returns the documented
  typed error.

### `stt.StreamingStrategy` (technique-axis interface)
- **Owner:** `api/internal/stt/strategy/`. Concrete
  implementations: `vad_segment.go`, `overlap_agree.go`,
  `passthrough.go`.
- **Production wires:** strategies are stateless orchestrators
  constructed per session by `StrategySelector`. Each consumes the
  audio-chunk channel and emits typed `sttchain.StreamEvent` values;
  the Segmenter translates them to wire events.
- **Test substitutes:** strategies are tested with fake `Provider`
  instances; matrix tests cover provider error injection, VAD
  boundary detection, LocalAgreement commit logic, and Passthrough
  vendor-event translation independently.

### `sttchain.ProviderTraits` (capability declaration)
- **Owner:** `api/internal/ai/sttchain/interface.go`. Today's
  `StreamingCapability() bool` becomes the `Stream` bit on a typed
  `ProviderTraits` struct; a new `Batch` bit is added (always `true`
  for existing providers; future native-streaming-only adapters may
  set it `false`).
- **Production wires:** read by `StrategySelector` to filter the
  strategy matrix. Each provider declares its traits once at
  construction time; no runtime probing.
- **Test substitutes:** trait variants are table-driven inputs to the
  selector tests above; no fake needed.

### Web-console audio-tools URL discovery
- **Boundary:** `handlers/discovery` in web-console serves
  `DiscoveryService.GetAudioToolsEndpoint`. The browser bootstrap reads
  this once and writes `window.__AUDIO_TOOLS_URL__` before React mounts;
  AudioToolsProvider then constructs the @audio-tools/embed client.
- **Why:** prevents client-side composition of scenario URLs (mirrors
  the `feedback_scenario_url_resolution` rule).

## Interface seam index (drift-gated)

The seam-registry test (`api/internal/testutil/seam_registry_test.go`)
walks every interface declaration tagged with a `// seam:` doc comment
and asserts it appears in this document. Each entry below is one of the
qualified names the test searches for.

- `clock.Clock` — wall-clock seam (see Clock section above).
- `envx.Reader` — process-environment seam (see envx.Reader section).
- `logx.Logger` — structured-logging seam (see logx.Logger section).
- `httpc.Doer` — outbound-HTTP seam (production: `*http.Client`).
- `database.Pinger` — database-reachability seam (production: `*sql.DB`).
- `audio.Runner` — ffmpeg/ffprobe process seam.
- `capabilities.Checker` — per-capability probe seam used by the
  capability registry to compute `/health` aggregates.
- `usagereport.Recorder` — usage-row recorder seam.
- `pipeline.HTTPDoer` — narrowed outbound-HTTP seam scoped to the stt
  pipeline (kept package-local to avoid a back-edge import on httpc;
  satisfied by anything implementing `httpc.Doer`).
- `strategy.Strategy` — streaming-strategy seam (alias of the
  `stt.StreamingStrategy` row at the top of this document).
- `strategy.BatchExecutor` — strategy→chain seam used to break the
  package import cycle.
- `sttchain.Provider` / `sttchain.BYOKAdapter` / `sttchain.VrooliClient`
  — STT chain seams.
- `ttschain.Provider` / `ttschain.BYOKAdapter` / `ttschain.VrooliClient`
  — TTS chain seams.
- `summarizechain.Provider` / `summarizechain.BYOKAdapter` /
  `summarizechain.VrooliClient` — summarize chain seams.
- `tts.HandlerService` — TTS application-layer seam consumed by the
  Connect handler.
- `tts.Synthesizer` — local-TTS engine seam (Kokoro HTTP synthesizer in
  production).
- `tts.VoiceLister` — voice-catalog seam.

## Cross-references

- Test fakes lifecycle and naming convention: [`docs/internal/TESTING.md`](TESTING.md).
- API contract manifest: `.vrooli/endpoints.json`.
- Documentation manifest (used by doc-rendering tooling): `docs/manifest.json`.
- Production-import quarantine for testutil: `api/internal/testutil/no_prod_import_test.go`.
- The unit-testing-architecture-steer skill (loaded via `prompt-manager skill read unit-testing-architecture-steer`) is the canonical source for "should this be a seam?" judgement calls.
