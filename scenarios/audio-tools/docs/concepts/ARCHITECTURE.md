# Architecture — Audio Tools

This document is the scenario's system map. It explains the invariant
shape inherited from the `react-vite` template, then points to the
specialized documents that own product domains, workflows, data,
integrations, deployment, operations, and business strategy.

Keep this file high-signal. Do not turn it into a warehouse for every
domain, endpoint, workflow, or decision. If a concern has a dedicated
document below, update that document and link it here.

## Purpose Of This Document

This document owns:

- the scenario's system shape,
- the role of each surface,
- how contracts and data flow between surfaces,
- the shared infrastructure boundary,
- extension rules for future code,
- architecture maturity and intentional deviations.

This document does not own:

- product capability inventory: [`DOMAINS.md`](DOMAINS.md),
- temporal and user/system workflows: [`FLOWS.md`](FLOWS.md),
- storage details and retention: [`DATA.md`](DATA.md),
- resource and scenario dependencies: [`INTEGRATIONS.md`](INTEGRATIONS.md),
- test seams and fakes: [`../internal/SEAMS.md`](../internal/SEAMS.md),
- test strategy: [`../internal/TESTING.md`](../internal/TESTING.md),
- deployment and operations: [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md),
- commercial strategy: [`../business/MONETIZATION.md`](../business/MONETIZATION.md).

## Scenario Shape

A scenario is one product expressed through three coordinated surfaces
and one canonical contract layer.

```
                       ┌─────────────────────────────┐
                       │  Generated proto types      │
                       │  packages/proto/schemas/    │
                       │   audio-tools/v1/...    │
                       └──────────────┬──────────────┘
                                      │ canonical wire shape
              ┌───────────────────────┼───────────────────────┐
              │                       │                       │
              ▼                       ▼                       ▼
        ┌──────────┐            ┌──────────┐            ┌──────────┐
        │   ui/    │ Connect-JSON│  api/   │ Connect-JSON│  cli/   │
        │ React    │ ◀────────▶ │   Go     │ ◀────────▶ │   Go     │
        │ + Vite   │            │ HTTP     │            │ cli-core │
        └──────────┘            └────┬─────┘            └──────────┘
                                     │
                                     ▼
                                ┌─────────┐
                                │ SQLite  │
                                │ (local) │
                                └─────────┘
```

| Surface | Role | Owns | Does Not Own |
|---|---|---|---|
| API (`api/`) | Scenario core | Business rules, persistence, integrations, transport edge | Browser state, CLI formatting |
| UI (`ui/`) | Browser presentation | Components, i18n, accessibility, browser interaction | Business rules, persistence policy |
| CLI (`cli/`) | Operator/agent wrapper | Argument parsing, output formatting, API invocation | Business rules, duplicated validation |
| Contracts (`packages/proto/schemas/audio-tools/`) | Wire shape | Proto messages/services and generated clients | Hand-written route/type mirrors |

The load-bearing principle: the API is the only surface that contains
business logic. UI and CLI translate user/operator intent into API
calls. Proto types flow from one source of truth so wire-shape drift
between surfaces is impossible.

## System Boundaries

### Development target: portable streaming voice

Audio Tools is the first pilot of
[contract-driven scenario development](../../../../docs/agent-system/SCENARIO_DEVELOPMENT.md).
This section states the target architecture, not qualification of the current runtime.
The dated observations in `docs/internal/PROBLEMS.md` remain the implementation evidence.

Keep these boundaries explicit while developing the STT vertical slice:

| Boundary | Target obligation |
| --- | --- |
| Shared capture and consumer integration | Capture lifecycle, portable host adapters, partial revisions, ordered final commits, and recoverable transport without app-private protocol copies. |
| Audio Tools routing | Resolve local, BYOK, and owned subscription/credits paths from explicit capability and policy; expose the selected path and unavailable reasons. |
| Engine/resource adapters | Describe actual streaming, platform, model, acceleration, and readiness capabilities. Replace engines without rewriting consumer integrations. |
| Monetization owner | Entitlement, wallet, session/usage linkage, and replay-safe settlement semantics. Keep billing policy out of consumer UI code. |
| Validation owners | Reproducible capture, route, quality, latency, duration, and device evidence with distinct real and simulated claims. |

Local compute and BYOK remain explicit modes. Hosted subscription value comes from
the owned service and benefits; this target does not authorize charging all local
or BYOK usage. Apply the existing shared monetization contracts rather than a
voice-specific ledger. STT is the first vertical slice, not a simultaneous rewrite
of every audio feature.

Runtime discovery must not impose unbounded readiness work on each voice session.
Measure cold and warm startup separately before selecting a cache/probe strategy.
Preserve freshness and unavailable reasons when optimizing discovery.

Numeric latency/quality floors, the supported-device commitment, real-provider
spend, and settlement/cancellation semantics require review before acceptance or
paid execution. They do not block authorized local measurement and repair.
The method and evidence matrix live in [Testing](../internal/TESTING.md#development-pilot-evidence-contract).

The published scope is PRD `portable-voice-v1`: route OT-P0-004, streaming
OT-P0-005, quality OT-P0-006, owned service OT-P0-007, device qualification
OT-P0-008, privacy OT-P0-009 and evidence OT-P0-010. Maintainability is
OT-P1-003. The v2 diagnostic board lists those targets but does not prove them.
The historical local-only proposal is not the full mandate.

### Replaceability and compatibility contract

Keep four independently changing decisions separate: processing route
(local/BYOK/owned), provider/model capability, streaming strategy, and capture
host. An installed model, a running process and a qualified interactive path
are different states. A resource registry entry is discovery metadata, not a
permission to promote a provider or a permanent choice of technology.

Adapters declare supported formats, native versus synthesized/batch streaming,
partial/final semantics, cancellation, concurrency, platform/artifact constraints,
acceleration needs and model identity. Select compatible combinations once per
session using explicit policy; expose why other combinations are unavailable.
Do not encode engine-name conditionals in consuming apps. Version incompatible
protocol changes and test mixed-version rejection or migration deliberately.

The shared browser package owns capture, journal and text-event semantics.
Audio Tools owns admission, interval accounting and provider orchestration.
Resources and the control plane own model installation and host lifecycle.
LPBS/shared delivery owns entitlement and settlement; a bounded asynchronous
usage-history recorder is not a durable customer billing ledger.

Refactor in tested slices: expose/inject the failing boundary, add its desired
behavior and negative controls, move duplicate logic into that owner, and prove
the shared package plus affected consumer paths. Keep native-streaming and batch
adapters in the same conformance suite without pretending their cadence is equal.
Do not replace the whole voice stack merely to shorten a large hook. Register
production/fake pairs in [SEAMS.md](../internal/SEAMS.md) as they actually land.

The scenario owns:

- source code under `api/`, `ui/`, and `cli/`,
- generated-scenario docs under `docs/`,
- scenario lifecycle metadata under `.vrooli/`,
- scenario-specific requirements under `requirements/`,
- scenario proto schemas relocated to
  `packages/proto/schemas/audio-tools/`.

The scenario does not own:

- shared package implementation under `packages/`,
- Vrooli resource implementation,
- scenario dependencies it calls,
- generated proto outputs under `packages/proto/gen/`.

Document dependency and resource decisions in
[`INTEGRATIONS.md`](INTEGRATIONS.md), not here.

## Contracts And Data Flow

Wire shapes do not live in TypeScript interfaces, Go structs, or
hand-written JSON schemas. They live in `.proto` files. For
proto-typed API calls, the `.proto` file also declares the service
block that generates Connect handlers and clients.

```
packages/proto/schemas/audio-tools/v1/<domain>/<file>.proto
       │
       ▼
       make generate
       │
       ├──▶ packages/proto/gen/go/audio-tools/v1/...              (api, cli)
       ├──▶ packages/proto/gen/go/audio-tools/v1/...connect       (Connect-Go)
       ├──▶ packages/proto/gen/typescript/audio-tools/v1/...   (ui)
       └──▶ packages/proto/gen/python/audio_tools/v1/...    (future tools)
```

Use Connect-RPC by default:

- UI to API for proto-typed payloads,
- CLI to API for proto-typed payloads,
- API to API / inter-scenario calls with Vrooli-owned protos.

### Standing health visibility (`health_status` domain)

`vrooli.audio_tools.v1.health_status.HealthStatusService` is the typed
contract for per-capability provider availability. It mirrors the
in-process `capabilities.Registry` rollup; the existing `/health` REST
endpoint stays as an ops-probe (now with a non-Critical `providers`
dependency so load balancers see degradation without losing readiness).
The UI `/status` page consumes `GetProviderHealth` through React Query
keyed on `["healthStatus","providers"]` with a polling interval driven
by the response's `cache_ttl_seconds`; the CLI exposes the same data
through `audio-tools health show` (default human table) and
`audio-tools health watch` (server-streaming `StreamProviderHealth`).
All cross-component reads go through Connect — never through the REST
probe.

### Provider lifecycle actions (`provider_lifecycle` domain)

`vrooli.audio_tools.v1.provider_lifecycle.ProviderLifecycleService` is
the operator-facing surface for starting, stopping, restarting, and
pulling models on the four local-tier providers audio-tools owns
(`whisper-stt`, `kokoro-tts`, `speaker-verification`, `ollama`). The native
`sherpa-onnx` resource backs the Kokoro-compatible TTS, streaming STT, speaker,
and source-separation capabilities while those logical capability names remain
stable for API and UI consumers. `whisper` remains the native batch STT
resource; these logical names do not imply separate Docker resources.
It
wraps `vrooli resource <verb> <slug>` through the
`capabilities.ResourceController` seam (production:
`capabilities.CLIController` resolved once at startup;
`CodeUnavailable` when the `vrooli` binary is not on PATH). Mutating
RPCs honor the canonical `X-Dry-Run: true` request header. After a
successful mutation, the handler kicks `Registry.ResolveForce` in a
background goroutine so the next `GetProviderHealth` reflects the new
process state without waiting for TTL. `PullModel` is allowed only on
`ollama`; other provider IDs return `CodeFailedPrecondition`. Log
streaming uses Connect server-streaming. The UI `/status` page renders
the action buttons advertised by `ListLocalProviders.supported_actions`
and opens a `LogsDrawer` for `View logs`. The CLI mirrors the surface
under `audio-tools provider {list,start,stop,restart,pull-model,logs}`.

### Silent-fallback observability (`x-audio-tools-fallback` header)

The existing chain fallback mechanism emits two signals when STT, TTS, or
Summarize serves a different eligible tier. This describes observability, not
consent enforcement. Hosted Vrooli delivery is not wired in the inspected
bootstrap; a hypothetical BYOK-to-owned fallback is not a working route.
The target requires prior policy authorization before any data-transfer or
paid fallback. A post-response toast cannot supply that authorization.

1. A structured log line `event=tier_fallback capability=<stt|tts|summarize>
   from_tier=<byok|vrooli|local> to_tier=<…> reason="<error class>"`
   from the chain layer. Wired at construction time via the chain
   `Options.Logx` field consumed by `fallbackLogger()` in each chain
   package.
2. A Connect response header `x-audio-tools-fallback:
   from=<tier>;to=<tier>;reason=<code>`. The chain layer fires its
   per-request hook through `tiered.WithOnFallback(ctx, …)`; the
   Connect handler closes over the `connect.Response` and sets the
   header. No proto contract change: this is metadata, not a typed
   field. The chain-fallback seam lives in
   `internal/ai/chains/tiered/{tiered.go,context.go}`.

UI consumption is in `ui/src/api/fallbackInterceptor.ts`. The Connect
interceptor reads the header, derives the capability from the response
service typeName, debounces at one toast per capability per 60 seconds,
and pushes through the minimal `Toaster` primitive in
`components/ui/toast.tsx`. Toast bodies deep-link to
`/status#<capability>`; `CapabilityRow` carries a matching DOM `id`
so the anchor scrolls to the affected row.

REST is allowed only for four enumerated reasons, defined as
`RESTReason` constants in `api/internal/module/module.go`:

| Reason | When it applies |
|---|---|
| `RESTReasonMultipartUpload` | Opaque file bytes via `multipart/form-data`. The notes attachments endpoint is the worked example. |
| `RESTReasonWebhookReceiver` | Endpoint shape is dictated by a third-party system (Stripe, GitHub, etc.) we do not own. |
| `RESTReasonThirdPartyShape` | Request or response is an externally-defined contract (OAuth callbacks, OpenAPI passthrough). |
| `RESTReasonOpsProbe` | Lifecycle systems, load balancers, and `curl` must reach the endpoint without a generated client (plain `GET /health`, static iframe-facing HTML wrappers). |

Mechanical enforcement: `cmd/gen-endpoints` rejects any
`EndpointDescriptor.Path` that is not a generated Connect procedure
constant (i.e. does not start with `/vrooli.`) unless the descriptor
carries a `RESTException` with one of the four reasons. A REST
endpoint without that tag fails `make endpoints`, which fails
`make test`, which fails CI. The fix is either to author a proto
service method (the preferred path) or to tag the exception
explicitly. There is no "internal endpoint, REST is fine" path —
that rationalization is exactly what the validation pass prevents.

Note: even for REST exceptions, the **payload shape** stays
proto-typed wherever possible. The notes attachments handler returns
the proto `UploadAttachmentResponse` message; only the request
transport is multipart. Drift between API/UI/CLI is eliminated as
long as the wire payload type is shared.

## Shared Infrastructure

Shared infrastructure is allowed only when the code is
business-vocabulary-free and used by unrelated domains or surfaces.

| Package/Folder | Purpose | Why Not Domain-Owned | Consumers |
|---|---|---|---|
| `api/internal/server/` | Compose modules and middleware into one HTTP server. | Server lifecycle is not a product capability. | API entrypoint and handler modules. |
| `api/internal/module/` | Shared module and endpoint descriptor types. | Domain modules return this common shape. | Handler packages, server, endpoint codegen. |
| `api/internal/modules/` | Thin registry for schemas and endpoints. | Boot/codegen need central lists; logic stays domain-owned. | `main.go`, `gen-endpoints`. |
| `api/internal/database/` | System schema and DB reachability seam. | Cross-cutting DB infrastructure, not one domain's data. | API boot, health. |
| `api/internal/clock/` | Deterministic time seam. | Time is cross-cutting and test-substitutable. | Middleware, repositories. |
| `api/internal/testutil/` | Cross-domain test harnesses and fakes. | Used by unrelated domains; domain fakes stay domain-local. | API tests. |
| `ui/src/test-utils/` | Cross-feature render helpers, a11y helpers, and model tests. | Used by unrelated UI features. | UI tests. |

If shared infrastructure starts using product vocabulary, move that
piece back into the owning domain or split a new domain first.

## Extension Rules

Add product behavior by adding or updating the owning domain, not by
growing generic buckets.

For a normal proto-backed domain:

1. Add proto messages and service methods under
   `packages/proto/schemas/audio-tools/v1/<domain>/`.
2. Add API domain code under `api/internal/<domain>/`.
3. Add transport code under `api/handlers/<domain>/`.
4. Register schemas/endpoints in `api/internal/modules/registry.go`
   and mount the module in `api/main.go`.
5. Add CLI commands under `cli/domains/<domain>/`.
6. Add UI API wrappers under `ui/src/api/<domain>.ts` and UI feature
   code under `ui/src/features/<domain>/`.
7. Update selectors, strings, endpoints, tests, and the docs contract
   in `docs/manifest.json`.

For detailed product ownership, update [`DOMAINS.md`](DOMAINS.md).
For persistence and retention, update [`DATA.md`](DATA.md). For
temporal behavior, update [`FLOWS.md`](FLOWS.md).

## Architecture Maturity

Source presence and declared document maturity do not establish product readiness.
Use the following map to find the owning implementation and its evidence:

| Area | Present structure | Qualification boundary |
|---|---|---|
| API | Domain handlers, provider chains, session/capture/recovery code, and injectable seams. | Local/BYOK routes need engine-specific evidence; the hosted clients remain unimplemented. See [Integrations](INTEGRATIONS.md). |
| UI | Shared audio integration and feature-owned capture/stream/recovery behavior. | Paced browser and consumer traces must prove visible partials and final drain. |
| CLI | Domain commands declared in `cli/manifest.json`. | Command tests and stream smoke results do not alone certify browser or device behavior. |
| Docs | `docs/manifest.json` registers document owners and structural checks. | Pilot targets remain partly undecided; a valid manifest is not exercised acceptance. |

Use [Testing](../internal/TESTING.md) for the pilot's evidence contract and
[Problems](../internal/PROBLEMS.md) for dated limitations. Neither the starter
architecture nor an old passing test substitutes for evidence on the current build.

## Testing Infrastructure

Domain tests are co-located with their code; shared fixtures and boundary doubles
live under `api/internal/testutil/` and the domain-owned fixture packages.
Persistence tests should use the real test substrate. Inject clocks, environment,
logging, and HTTP clients through the registered seams.

The seam registry tests reconcile `// seam:` tags with the `Interface seam index`
in [SEAMS.md](../internal/SEAMS.md); production-import checks protect the test-only
boundary. Run the relevant checks under [Testing](../internal/TESTING.md) and
retain the actual build and result. This section describes test architecture,
not an undated all-green maturity assessment.

## Streaming Pipelines (STT)

The unary `Transcribe` path follows the standard scenario shape above:
proto → handler → domain → provider chain → storage. The **streaming**
STT path is a layered orchestration that needs its own diagrams and
compatibility matrix; see
[`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md)
for the full architecture (current vs. target, strategy vs. provider
decoupling, capability table). The seams it introduces — `Segmenter`,
`StrategySelector`, `StreamingStrategy` — are registered in
[`../internal/SEAMS.md`](../internal/SEAMS.md#streaming-chain-seams-audio-tools-web-console-restoration-plan).

### Engine-capability manifest (`internal/sttengine`)

`internal/sttengine` is the single source of truth for which STT engines the
scenario can run and what each one is capable of. One checked-in JSON manifest
(`manifest.json`, validated against `schema.json` by a Go test) describes every
engine with **positive capability declarations** (`provides.nativeStreaming /
builtinVad / confidenceSignals / wordTimestamps`) plus the pluggable
`speakerIsolation` axis. Two consumers *derive* behavior from it — there is no
engine-id branch logic anywhere else:

- the **selector** derives the Local tier's eligible strategy whitelist from the
  active engine's `strategies[]` (`EligibleStrategies`); BYOK/Vrooli tiers keep
  their `ProviderTraits`.
- the **egress gate** derives its stage set from the engine's capabilities
  (`EgressStages` — the signal-domain stage only fires when the engine declares
  confidence signals; the audio-domain speaker stage fires when the manifest's
  active isolation method is set and the session wired an isolation).
- the **Segmenter** derives whether inbound chunks need PCM normalization for a
  Passthrough engine from `RequiresPCM` (the engine's `requires.pcm16kMono`), so
  a native-streaming engine like Kyutai or sherpa-onnx still receives canonical PCM while a
  Passthrough BYOK vendor that decodes for itself does not.

Three engines are declared today: **`whisper-local`** (native whisper.cpp,
batch; VAD-segment / overlap-agree / buffered strategies; confidence signals),
**`kyutai`** (native streaming; Passthrough only; no confidence signals), and
**`sherpa-streaming`** (native English-only Zipformer streaming with punctuation
and casing restoration; Passthrough only). Switching engines via the
admin picker consults `GetEngineSwitchImpact`, which scans every scenario's
`.vrooli/service.json` (`ScanResourceConsumers`, over an injected `fs.FS`) to
report who else uses the outgoing engine's resource — audio-tools never
auto-stops a shared resource, it surfaces the `vrooli resource stop <name>`
command for the operator.

Two layers, never conflated: the manifest holds **static capability facts**;
the **active selection + tunables** live in the persisted `StreamConfig`
(`engine_id`, egress toggles/thresholds) edited via the admin proto API. Adding
an engine is a manifest entry (plus a `resources/<name>/` folder + provider
adapter for `kind=local_resource`). The catalog is exposed to UI/CLI via
`ListEngines` — never hardcoded. `sttengine` imports only `internal/stt/egress`
(not `sttchain`, to avoid a cycle); strategy ids stay string-typed and the
selector converts.

### Post-recognition egress gate (`internal/stt/egress`)

The symmetric counterpart to the audioformat **ingress** point: a single
**egress** seam every transcribed segment passes through before the wire. The
`Segmenter` builds one `egress.Gate` per session (stage set from
`sttengine.EgressStages`) and runs each `SegmentEvent` through ordered,
capability-gated `Stage`s — strategies never call the gate. Stages assign an
outcome: `Emit`, `Drop` (suppressed + excluded from the rebuilt final
transcript), or `Reject` (suppressed text + a `StreamEventSpeakerRejection`).
Three signal domains run in order: **text** (`HallucinationStage` — Whisper
phrase filter), **signal** (`ConfidenceStage` — `no_speech_prob`/`avg_logprob`,
added only when the engine declares confidence signals), and **audio**
(`SpeakerStage` — speaker identity). This is where Whisper's silence
hallucinations are killed (alongside `vad_filter=true` at the source) and where
non-enrolled voices (e.g. background music) are rejected when speaker isolation
is enabled. The audio-domain stage is **engine-independent** (it operates on the
segment's canonical-PCM bytes, not the transcript) and pluggable: the manifest's
active `speakerIsolation.active` method selects the `egress.SpeakerIsolation`
implementation (`verification` today wraps `pipeline.EvaluateSpeaker` against the
native `sherpa-onnx` resource). A **profile is one identity holding N labeled
enrollment clips**; the resource trims each clip to its voiced span before
embedding (energy VAD) and verifies against the profile centroid + each clip
(hybrid score). The egress decision is **session-stateful** — a per-session
`pipeline.SessionSpeakerState` accumulates per-segment scores (EMA) and withholds
any rejection during a warm-up window (until `min_decision_seconds` of voiced
audio accrue), so the verdict stops swinging mid-utterance and a short first
utterance is never falsely dropped. Policy applicability depends on canonical
audio bound to recognition spans, not the engine name. Required policy must fail
closed if that evidence is unavailable; passthrough is not an authorized policy
bypass (OT-P0-003). Cross-engine proof remains in the requirements registry. The verification
adapter lives in the handler layer (not `pipeline`) to avoid the
`egress → sttchain → pipeline` import cycle. Seams: `egress.Stage`,
`egress.SpeakerIsolation`, `sttengine.Registry` (see
[`../internal/SEAMS.md`](../internal/SEAMS.md)).

The egress gate can only DROP a segment's text. To remove an interfering voice
from the **audio** — isolating the enrolled speaker before recognition — the
`Segmenter` runs a pre-recognition **ingress** pipeline (`internal/stt/ingress`,
the symmetric counterpart of egress): ordered `ingress.Enhancer`s wrap the
canonical-PCM stream before the VAD/strategy see it. Production enhancers are
`DenoiseEnhancer` (ffmpeg afftdn, gated on `denoise_enabled`) and
`ExtractionEnhancer` (target-speaker extraction via the `ingress.TargetExtractor`
seam — source separation + speaker-embedding target-selection in the `sherpa-onnx`
resource, gated on `extraction_enabled` + a bound profile). Both are config-gated
(no manifest flag) and engine-agnostic in principle; v1 wires them on the Whisper
PCM path. The extraction adapter also lives in the handler layer to avoid the
`ingress → sttchain → pipeline` cycle. Seam: `ingress.Enhancer`,
`ingress.TargetExtractor`.

### Audio-format substrate (`internal/audioformat`)

`internal/audioformat` is the single owner of audio-format handling — the
only package that knows ffmpeg argv or sniffs codec magic bytes. Everything
that ingests or emits audio routes through it:

- **STT streaming ingress:** the Segmenter normalizes inbound chunks to
  canonical PCM (16-bit LE, mono, 16 kHz) via one long-lived ffmpeg process
  per session (`StreamDecoder`); a declared `pcm_s16le` takes an
  ffmpeg-free fast-path. This is the single injection point both transports
  (WS + Connect bidi) inherit, so they cannot drift.
- **STT batch ingress:** `PrepareForWhisper` wraps canonical PCM in a WAV
  header (ffmpeg-free) and passes real containers straight through —
  Whisper's own bundled ffmpeg decodes them, so the batch path needs no
  local ffmpeg.
- **TTS egress:** `OutputFormat` is the single source of truth for the TTS
  format vocabulary + content types; the native sherpa-onnx adapter exposes
  the Kokoro voice catalogue and encodes the requested format itself
  (symmetric with Whisper decoding on ingress).

Zone/import rule: `audioformat` may import stdlib and the `internal/audio`
one-shot `Runner` seam only; it is imported by `internal/stt/*`,
`internal/ai/sttchain`, and `internal/tts`. It holds no per-session state —
every per-session decoder is created fresh (`NewStreamDecoder`) so
concurrent sessions never share a process. The streaming-decode seams
(`ProcessRunner`/`Process`/`StreamDecoder`) are registered in
[`../internal/SEAMS.md`](../internal/SEAMS.md). The real multi-session
ceiling is the Whisper resource's 5-concurrent cap, bounded by a semaphore
in `pipeline.Service` (queue with backpressure, never error).

## Intentional Deviations

Record deviations from the template or from Vrooli scenario standards
when they are deliberate and durable.

| Date | Deviation | Reason | Revisit Trigger |
|---|---|---|---|
| 2026-05-16 | None yet. | Generated from `react-vite`. | Update when the scenario intentionally diverges. |

## Documentation Architecture

Scenario docs follow the same ownership rule as code: one durable
question, one canonical home.

| Concern | Canonical Document |
|---|---|
| System map and extension rules | `docs/concepts/ARCHITECTURE.md` |
| Product capabilities and bounded contexts | `docs/concepts/DOMAINS.md` |
| Workflows and state transitions | `docs/concepts/FLOWS.md` |
| Data ownership, retention, and migrations | `docs/concepts/DATA.md` |
| Resources, scenarios, and external services | `docs/concepts/INTEGRATIONS.md` |
| Monetization and packaging | `docs/business/MONETIZATION.md` |
| Go-to-market strategy | `docs/business/GO-TO-MARKET.md` |
| Deployment tiers and readiness | `docs/operations/DEPLOYMENT.md` |
| Operator procedures | `docs/operations/RUNBOOK.md` |
| Telemetry, metrics, and alerts | `docs/operations/OBSERVABILITY.md` |
| Seams and test doubles | `docs/internal/SEAMS.md` |
| Testing strategy | `docs/internal/TESTING.md` |
| Known drift and deferred work | `docs/internal/PROBLEMS.md` |
| Change history | `docs/internal/PROGRESS.md` |

Every durable scenario document should be registered in
`docs/manifest.json`. Put deep domain-specific documentation under
`docs/domains/<domain>/` when `DOMAINS.md` would become noisy.

## Cross-References

- [`START-HERE.md`](../START-HERE.md) — first implementation workflow
- [`QUICKSTART.md`](../QUICKSTART.md) — clone-to-running flow
- [`DOMAINS.md`](DOMAINS.md) — bounded contexts and ownership
- [`FLOWS.md`](FLOWS.md) — workflow and state-transition map
- [`DATA.md`](DATA.md) — data ownership and storage
- [`INTEGRATIONS.md`](INTEGRATIONS.md) — dependency contracts
- [`../business/MONETIZATION.md`](../business/MONETIZATION.md) — commercial story
- [`../operations/DEPLOYMENT.md`](../operations/DEPLOYMENT.md) — deployment readiness
- [`../internal/SEAMS.md`](../internal/SEAMS.md) — seam registry
- [`../internal/TESTING.md`](../internal/TESTING.md) — test patterns
- [`../internal/ERROR-HANDLING.md`](../internal/ERROR-HANDLING.md) — error semantics
- [`../internal/PROBLEMS.md`](../internal/PROBLEMS.md) — known issues / tech debt
- [`../internal/PROGRESS.md`](../internal/PROGRESS.md) — lifecycle log
