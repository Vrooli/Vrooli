# Maturation Program — Audio Tools

This document owns the durable program for making Audio Tools cheaper to
extend, easier to qualify, and harder to drift. It is deliberately concrete:
every workstream names its evidence, its target, and the gate that closes it.

It is not a release claim. Source presence and green unit tests do not qualify a
provider, device, model, or route. Use [TESTING.md](TESTING.md) and
[PROBLEMS.md](PROBLEMS.md) for the evidence contract and dated findings.

## Why this program exists

Image Tools proved that a mature scenario's dominant tax is **declaration
bookkeeping**: one operation declared in seven hand-synced places, with two
surfaces unguarded. Audio Tools has that tax **plus** duplicated runtime
scaffolding, an unfinished capability surface, a thin model catalog, and lab
tooling entangled with the product path. The fix is larger than a manifest, but
it is still phased: consolidate, declare, generate, gate, delete.

## Baseline (2026-11-19)

| Signal | Observed |
|---|---|
| Source / test LOC (Go + TS/TSX) | ~63k src / ~51k test |
| Requirements | 24 total: 3 complete, 6 in progress, 15 planned |
| Debt register | `PROBLEMS.md` 2,631 lines, 22 unassigned entries |
| Duplication | `dupl -t 80` 28 clone groups; three chain packages + shared tiered/BYOK scaffolding ~5.4k LOC |
| Audio operations exposed | 8 exist in proto/handlers; **only `transcode` is in CLI and UI** |
| Engine declaration | `sttengine/manifest.json` declares 2 engines; bootstrap constructs 3; docs/UI name 3 |
| Provider spread | one provider id appears across ~20 non-test files |
| Coverage floors | `bootstrap` 754 src / 116 test; `stt/pipeline` floor 20%; `experiment`, `eval`, `soak`, `validation` have no floors |
| Hot spots | `soak/driver.go` 1,591; `overlap_agree.go` 912; `experiment/report.go` 884; `runner.go` 643; `vad_segment.go` 632 |
| Shared UI | `useVoiceCore.ts` 2,095; `pcmVoiceStreamProvider.ts` 918; `turnJournal.ts` 338 |
| Model catalog | no registry; free-form model IDs; gateway has no STT/TTS/duplex roles |

## Stance

1. **No behavior change in a cleanup phase.** Cleanup and capability work do not
   share a commit.
2. **A guard replaces a habit.** If a rule matters, a test enforces it; if it
   cannot be enforced, it belongs in a doc, not a checklist.
3. **Every structural phase ships a capability.** A refactor that only moves
   files is not done until a user-visible operation becomes reachable or a
   default becomes measurable.
4. **Do not rewrite the streaming engine.** Consolidate around it; the
   egress/ingress gates and recovery tests are the value.

## Delivered in the first tranche (2026-11-19)

This is a progress record, not a readiness claim. It names the substrate that
now exists and the gates that enforce it.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Provider declaration | `api/internal/providercatalog/catalog.seed.json` + loader/validation + generated `capabilities` and TS catalogs | `TestGeneratedCatalogIsNotStale`, `TestProviderEngineMappingIsBidirectional`, `TestLocalProviderIDsDeterministicAndResourceUnique` |
| Cross-catalog integrity | provider↔model↔engine conformance | `api/internal/catalogconformance` (references, capability match, engine ownership, licence/default rules) |
| Engine drift | `sttengine` now declares `sherpa-streaming`; bootstrap and catalog agree | `TestProviderEngineIDsMatchSTTEngineManifest` |
| Operations | `api/internal/operationcatalog` declares 13 operations; all 8 audio transforms are reachable from CLI and the Diagnostics UI | `TestAudioManifestCoversEveryOperation`, generated `audioOps.generated.ts` |
| Models | `api/internal/modelcatalog` with 23 records + generated `docs/reference/models.md` | selection/default/licence tests + doc drift |
| Model policy | `whisper-large-v3-turbo` and `rumble-2-flash` are candidate defaults; commercial/non-commercial/readiness rules are explicit | `DefaultFor` requires the `default` role; `CommercialDefaultFor` fails closed |
| Provider identity | public results report canonical provider IDs; engine IDs remain internal evidence | `CanonicalProviderID` + handler tests |
| STT reliability | VAD retry backoff, leading-silence trim, OverlapAgree committed-PCM compaction, Passthrough terminal `Done`, session snapshot ordering | focused `-race` regressions in `internal/stt/{strategy,segmenter,session}` |
| Audio correctness | AAC/ADTS, trim duration, fade-out anchor, merge crossfade label/temp cleanup, cancellation mapping, split validation, audio-only ffprobe | `internal/audio` + `handlers/audio` edge/integration tests |
| Concurrency/aliasing | pipeline dependencies snapshot behind a lock; speaker/wake-word snapshots detach slices | `TestServiceDependencySettersAreRaceFree`, snapshot isolation tests |
| Gates | measured coverage floors for bootstrap/lab packages; `stt/pipeline` 20→55; goleak ignores only the shared sql global | `TestCoverageFloors`, `go vet`, targeted `-race` |

Still open: ListModels/engine-model join in the admin picker, Rumble/Kale
runtime adapters, gateway audio roles, full-duplex, and the device/evidence
matrix. Those are Phases 3–5 below.

## Delivered in the second tranche (2026-11-19)

The registry stopped being documentation and became a runtime surface.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Model service | `ModelRegistryService` (`ListModels`, `ResolveModel`), mounted at `api/handlers/models` | Connect transport test + `TestResolveModel_CommercialNeverReturnsNonCommercial` |
| Host-aware defaults | `Platform` on list/resolve; `is_default` is host-relative | `TestListModels_PlatformConstrainsDefaultFlag`, `TestResolveModel_PlatformConstraintCanMakeDefaultIneligible` |
| CLI | `audio-tools model list` / `model resolve` with filters, JSON, and policy explanation | manifest↔handler coverage test; full CLI suite |
| UI | `/models` page: filters, readiness/licence/hardware facts, default badge, resolve panel | 29 component/service tests; full UI suite 134 files |
| Registry honesty | Summarize records mirror `resources/ollama/model-policy.json`; `qwen3.5:9b` is the runtime primary, `llama3.2:3b` remains the commercially certified/evidence-backed default | `TestSummarizeRecordsMirrorResourceOllamaPolicy` |
| Model-truth seam | Whisper env model ids must resolve in the registry and round-trip | `TestWhisperModelIDsResolveInRegistry` |
| Engine integrity | Provider engines, model engines, and `sttengine/manifest.json` are compared in both directions, including resource slugs | `TestProviderEnginesMatchSTTEngineManifest`, `TestModelEngineIDsAreShippedBySTTEngineManifest`, `TestActiveLocalSTTModelsNameAnEngine` |
| Streaming teardown | A decoder parked on a full frame buffer is unblocked by `Close` | `TestProcessDecoderCloseUnblocksFullFrameBuffer` |
| STT residuals | Rejected timeline ranges are observable (not a clean EOF), normalizer teardown joins its workers, strategies no longer mutate shared config, and cancel performs a bounded best-effort tail flush | focused `-race`/goleak regressions in `internal/stt/{segmenter,strategy}` |
| Chains | Shared BYOK/LPBS/stamping scaffolding consolidated into `chains/tiered`; streaming credit exhaustion is terminal instead of silently falling back; fallback reasons redact request credentials; availability probes cannot resurrect across `Reconfigure`; coverage restored to 81/86/94/83 | `TestCoverageFloors`, `dupl -t 80` = 0 clone groups, wrapped-sentinel mapping tests |
| Surface drift | `.vrooli/endpoints.json` regenerated for all eight audio operations and the model service | `make endpoints` + diff gate |
| UI debt | `pnpm lint` is clean (was 65 problems) | full UI suite + lint |

Still open after this tranche: the admin-picker engine→model join (the data is
now available), Rumble/Kale adapters, gateway audio roles, full-duplex, and the
device/evidence matrix.

## Delivered in the third tranche (2026-11-19)

This tranche made runtime truth measurable instead of asserted.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Whisper runtime | `resources/whisper` now pins `ggml-large-v3-turbo.bin` (checksum-verified) as the eager artifact; `ggml-medium.bin` is the `medium-model` on-demand rollback group. The capacity profile names the turbo digest | `TestWhisperRuntimeArtifactMatchesRegistryDefault` (registry default must equal the artifact the resource actually starts, and it must be eager) |
| Artifact truth | Model records name the resource artifact that supplies local weights; every `available` local model must name a declared artifact | `TestAvailableLocalModelsNameShippedResourceArtifacts`, loader rule `invalid_artifact` |
| Model receipt | The canonical `quality_speech.wav` scores WER 0.00 on turbo and medium; the 61-minute dictation fixture transcribes in 30.3 s (turbo, RTF 120.8x) versus 52.8 s (medium, 69.3x) with 0.9993 word-sequence agreement on one Linux/Vulkan host | Seed `benchmarks` + dated PROBLEMS entry; explicitly **not** a device-matrix or timestamp-stability claim |
| Audio responses | EBU R128 normalization is two-pass (linear) and `measured_lufs` is measured from the produced audio, never the requested target; `duration_seconds` is populated for transcode, trim, and merge | `internal/audio/loudness_test.go`, `duration_test.go`, handler tests |
| Lifecycle idempotency | Mutating provider actions dedupe on `(Idempotency-Key, operation, target)` with single-flight joiners, TTL, bounded eviction, no error caching, and cloned responses | `handlers/provider_lifecycle/idempotency_test.go` + handler tests (81.7% coverage) |
| Admin picker | Stream Config joins each STT engine to the registry's declared models (default badge, readiness/licence, runtime, size, languages, explicit no-model state) | `StreamConfigPage.test.tsx` (11 tests), full UI suite |
| Transport hardening | Transform and STT Connect handlers reject oversized messages before buffering them (256 MiB per transform request, 16 MiB per STT message) | `TestModuleRejectsOversizedRequest`, `TestSTTModuleRejectsOversizedMessage` |

Still open after this tranche: Rumble/Kale runtime adapters, timestamp-drift
scoring on long-form audio, a multi-device/multi-OS matrix, and full-duplex.
The gateway audio-role question is settled below.

Deliberate non-change: AI Gateway audio roles (`stt.fast`, `tts.quality`, …)
were **not** declared. The gateway's role catalog dispatches text-inference
candidates owned by resource providers (`ollama`, `openrouter`, `pca`, …); local
ASR/TTS resources are not in that provider family, so a role there would be
inert configuration that looks like capability. The audio role surface is
`ModelRegistryService` (`ListModels`/`ResolveModel`), which already applies
licence, readiness, language, and platform policy. Revisit the gateway only when
it owns an audio provider family and a dispatcher, not before.

## Delivered in the fourth tranche (2026-11-19)

This tranche closed user-visible correctness bugs and removed the last
hand-maintained provider/voice tables.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Transform correctness | An exit-zero ffmpeg that writes nothing is a failure; trim rejects a start past the measured end and clamps end; split reports the open-ended final chunk duration and rejects boundaries past the end; merge rejects a crossfade longer than the shortest source; transcode knobs and loudnorm targets are range-checked | `internal/audio` unit + real-ffmpeg integration tests |
| Streaming cancellation | STT strategy eligibility resolves the active engine under the session context, not `context.Background()` | `TestSelectorHonorsCanceledContextDuringEngineResolution` |
| Stream decode truth | A mid-stream decoder failure is reported as `input_decode_failed` and fails the replay ledger instead of looking like a clean EOF | `TestNormalizeChunksReportsDecoderFailure`, `TestStreamErrorCode_UsesBackendState` |
| Composition truth | Every endpoint declared in `modules.AllEndpoints` must be mounted by `Compose`; typed-nil optional repositories no longer panic during construction | `TestCompose_MountsEveryDeclaredEndpoint`, `TestNewConnectHandlerToleratesTypedNilSpeakerConfig` |
| Voice catalog | One canonical mapping per adapter (`local:kokoro`, `byok:openai-tts`, `byok:elevenlabs`), a real startup `VerifyCatalog`, override keys aligned to catalog adapter ids (legacy key honoured), settings rejects unknown voice/adapter pairs, and the Voices page is editable | `TestTTSAdapterMappingsMatchProviderCatalog`, settings tests, 10 Voices tests |
| Provider identity | Runtime aliases (`kokoro-local`, `ollama-local`) and BYOK probe facts live in the provider catalog; the hand switches are gone; active remote providers must declare a probe or a documented absence | providercatalog validation + `TestBYOKProbeFactsMatchLegacyTuples`, `TestBYOKCredentialCheckersHaveCatalogProbes` |
| BYOK integrity | Credential writes must name an active credential provider that declares the capability | `TestSettings_BYOKRejectsProviderCapabilityMismatch` |
| Credentials UX | The Configuration page picks credential providers from the generated catalog, filtered by capability, so a typo cannot create an unusable row | `ConfigurationPage.test.tsx` (7 tests) |
| Transport limits | TTS 8 MiB and summarize 4 MiB per message, joining transform 256 MiB and STT 16 MiB | module limit tests |
| Timeouts | Readiness probes use a 30 s doer instead of the 120 s generation doer; BYOK probes are bounded at 5 s; the soak receipt client has a timeout | `TestReadinessDoerTimeout`, BYOK deadline test |
| Health telemetry | Liveness checks are timed at the registry seam and `latency_ms` is a real measurement (was permanently 0); cache hits report the last check's duration | `TestLatencyRecorder*`, `TestGetProviderHealth_ReportsMeasuredLatency` |

The headline bug: a voice override saved from the Voices page used the catalog
adapter id `local:kokoro`, while the local provider looked up
`local:kokoro-local`. The override was stored, shown as configured, and silently
never applied. The keys now share one constant, the legacy key still applies,
and the settings edge rejects pairs that cannot match any adapter.

Still open after this tranche: Rumble/Kale runtime adapters, timestamp-drift
scoring, a multi-device/multi-OS matrix, and full-duplex.

## Delivered in the fifth tranche (2026-11-20)

This tranche made the machine footprint a product decision instead of an
accident, and put the lab boundary behind a measured gate.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Whisper default | `ggml-small.bin` (488 MB) is the eager artifact and the runtime default; turbo/medium/base/tiny are on-demand tiers. The capacity claim fell from 2 GiB to 1 GiB | `TestWhisperRuntimeArtifactMatchesRegistryDefault`, `TestLocalDefaultArtifactsAreEager`, `vrooli resource validate whisper` |
| Recommender | `whisper recommend-model` returns `small` on every usable host; `--quality` is the explicit accuracy opt-in; CPU no longer auto-picks medium | `TestPick_Table`, `TestPickQuality_Table` |
| Summarize default | resource-ollama `summarize.default` moved from `qwen3.5:9b` (~6 GB resident) to `qwen3.5:4b` (~2.6 GB), which was already the role's qualification candidate; 9B remains a quality tier | `TestSummarizeRecordsMirrorResourceOllamaPolicy`, `TestSummarizeDefaultMovedOffNineB` |
| Footprint budget | Each capability has a declared default-size budget (STT 600 MB, TTS 400 MB, summarizer 3 GB, speaker 100 MB); a promoted default that exceeds it fails the build, and the aggregate depends on it | `TestCapabilityDefaultsFitResourceBudget`, `TestCommercialDefaultsFitResourceBudget`, `TestPromotedDefaultFootprintFitsBudgets` |
| Footprint visibility | `docs/reference/models.md` renders the per-capability default, size, budget, artifact, the promoted-default total, and the catalogue total; `audio-tools model list` adds SIZE and a `Defaults: …` summary; the Models page shows a Resource footprint panel | docs drift test, CLI/UI tests |
| Lab boundary | Experiment/eval/soak/validation packages have honest coverage floors, `make lab-test`, and an import gate that keeps them out of product serving paths | `api/internal/lab/boundary_test.go`, coverage policy |
| Health telemetry | Sub-millisecond liveness measurements are reported instead of truncating to 0; `health show` renders a LATENCY column | `TestGetProviderHealth_ReportsSubMillisecondLatency`, CLI health tests |
| Panel debt | The Models page no longer nests two "Registry" panels; the footprint panel is distinct | `ModelsPage.test.tsx` (7 tests) |

The headline change: the host was running `ggml-medium` in 1,864 MiB of VRAM
while the registry claimed turbo, and the shared summarize role was pinned to a
9B model. The shipped default is now `small` (about one third of turbo's
weights), the capacity reservation is halved, and summarize defaults to the 4B
tier. Bigger models remain one explicit command away.

Still open after this tranche: timestamp-drift scoring on long-form audio, a
multi-device/multi-OS matrix, and full-duplex.

## Workstreams

### A. One declaration layer, generated downward

One provider/model catalog and one operation manifest, both embedded seeds with
typed loaders, validated at boot. Generated projections replace hand tables.

- Providers: health/status, lifecycle actions, resource slugs, settings keys,
  BYOK/settings wiring, UI provider catalogs, docs matrices.
- Operations: CLI command coverage, diagnostics forms, UI catalogs, params.
- Drift tests compare generated output byte-for-byte; conformance tests prove
  every declaration has an executor and every executor has a declaration.

The provider/engine split is explicit: a **provider id** is stable for health,
lifecycle, settings, and usage (`whisper-stt`, `sherpa-stt`); an **engine id**
is the STT stream-selection identifier (`whisper-local`, `kyutai`,
`sherpa-streaming`). The catalog owns the mapping in both directions.

### B. Consolidate the runtime scaffolding

The three chain packages already share `internal/ai/chains/tiered`; finish the
consolidation where it still pays:

- one availability/health path, one fallback policy, one lifecycle path;
- generated/declared bootstrap wiring instead of hand-built checker maps;
- remove the legacy `pipeline.Service` shim responsibilities and raise its
  coverage floor;
- separate the lab/qualification runtime (`experiment`, `eval`, `soak`,
  `validation`) from the product runtime, at minimum behind a `lab` boundary.

### C. Finish the capability surface and make reliability a gate

- All 8 audio transforms end-to-end in CLI + UI.
- One streaming state machine (Connect bidi): partial/final/drain/cancel,
  reconnect, backpressure, bounded queues, speaker gating.
- Device/browser/OS qualification matrix with dated receipts.
- SLOs as gates: WER/CER, p50/p95 first-partial and final latency, RTF, drop
  rate, recovery success, memory, cold start.
- Use the existing corpus/eval/experiment harness as the evidence producer; do
  not create a parallel ledger.

### D. Build a real model/voice catalog and selection policy

Model facts (weights, runtime, size, languages, license, hardware, readiness)
live in a validated registry. Selection is a policy over evidence, not a
hardcoded winner:

- `default` is per capability × device × language × commercial-use;
- `candidate` / `experimental` / `watch` never become defaults by accident;
- a model cannot be promoted without a receipt (corpus quality, latency,
  memory, platform, license).

## Phase plan and exit gates

### Phase 0 — Baseline and stop the bleeding

Fix the `sherpa-streaming` manifest drift. Freeze the provider/operation/model
inventory. Reconcile the requirements ledger. Establish the benchmark corpus and
the model-promotion evidence format. Add coverage floors for `bootstrap`,
`experiment`, `eval`, `soak`, `validation`; raise `stt/pipeline`.

**Exit:** bootstrap and the engine manifest agree; the ledger has no dead refs;
the baseline is reproducible.

### Phase 1 — Provider/model catalog + generated surfaces

Provider catalog, generated capabilities/mapping/lifecycle/UI/docs, drift and
conformance tests. Make the catalog load-bearing in this order: health →
lifecycle → bootstrap → BYOK/settings. Land Whisper `large-v3-turbo` as the
first data-only model upgrade.

**Exit:** adding a provider of an existing kind is data + at most one adapter;
a constructed-but-undeclared provider fails the build.

### Phase 2 — Operation manifest + headless completeness

Operation catalog, generated forms and CLI coverage, all 8 audio transforms
reachable from CLI/API/UI. Add the gate "every declared operation has a
headless journey or a dated exception."

**Exit:** no operation is server-only by accident.

### Phase 3 — Chain/lab consolidation

One fallback/health/lifecycle path. Remove the legacy pipeline shim. Extract the
lab packages behind a boundary. Raise coverage floors.

**Exit:** product and lab runtimes are separable; duplication is measured down,
not merely reorganized.

### Phase 4 — Model adapters + qualification

Add Rumble (STT) and Kale (TTS) adapters. Evaluate Reverb/Verba. Extend AI
Gateway with audio roles (`stt.fast`, `stt.quality`, `tts.fast`, `tts.quality`,
`conversation.duplex`). Promote defaults only with receipts.

**Exit:** model selection is a registry + evidence decision, not a code edit.

### Phase 5 — Duplex + UX + hardening

Add the full-duplex provider kind (GPT-Live-style) behind policy/cost controls.
Consolidate the UI/integration surface and split the `useVoiceCore` god hook in
coordination with the shared package owner. Run the device/browser matrix and
fault-injection suite.

**Exit:** reliability claims are backed by dated receipts.

## Model policy

Model choice is data plus evidence. The registry declares facts; the selector
applies policy; the qualification harness promotes.

| Candidate | License | Verdict |
|---|---|---|
| Whisper small (default), base/tiny (edge) | MIT | Shipped default and constrained-host tiers through the existing whisper.cpp adapter. The canonical clip passes exactly at every size; one host, not a dictation matrix. |
| Whisper large-v3-turbo | MIT | Explicit accuracy tier, on-demand. Use when accuracy outweighs ~1.3 GB more resident memory. |
| Reson8/Rumble-2-Flash | Apache-2.0 | Benchmark for a future local batch/multilingual STT tier. Batch, not native streaming. |
| Qwen3.5 4B (summarize default) | Unverified upstream | Shared `summarize.default` role; already the role's qualification candidate. 9B stays a quality tier. |
| Kale TTS 1 nano/tiny | Apache-2.0 | Candidate fast English CPU/on-device TTS; experimental until out of private draft. Kokoro stays multilingual baseline. |
| software-mansion/Reverb-TTS | Weights CC BY-NC 4.0 | Not a commercial default. Experimental/non-commercial or negotiated license only. |
| Verba TTS Indic-Zero | No linked license/release | Watch only. Unique Indic/edge tier if it ships permissively. |
| OpenAI GPT-Live-1.1 | Proprietary | New hosted **duplex** capability, not an STT/TTS adapter. |

Do not make a batch model the "streaming" default. Do not default a
non-commercial model. Do not let the model registry become a second provider
list. Keep the promoted default inside the declared footprint budget; a larger
default is an explicit seed and budget change, not a side effect.

## Gates to add

| Gate | Catches |
|---|---|
| `TestEveryActiveProviderHasCapability` | a provider with no health/capability meaning |
| `TestEveryConstructedProviderHasManifest` | the sherpa drift class |
| `TestProviderEngineMappingIsBidirectional` | engine/provider id confusion |
| `TestProviderEnginesMatchSTTEngineManifest` | a provider declaring an engine the runtime does not ship, or a shipped engine no provider owns |
| `TestModelEngineIDsAreShippedBySTTEngineManifest` | a model naming a nonexistent stream engine |
| `TestActiveLocalSTTModelsNameAnEngine` | weights with no selectable stream path |
| `TestSummarizeRecordsMirrorResourceOllamaPolicy` | registry default drifting from `resources/ollama/model-policy.json` |
| `TestWhisperModelIDsResolveInRegistry` | the Whisper model-truth seam producing an id the registry does not declare |
| `TestProcessDecoderCloseUnblocksFullFrameBuffer` | a stream decoder parked on a full frame buffer after teardown |
| `TestDefaultsByCapabilityIsDeterministic` + `TestListModels_PlatformConstrainsDefaultFlag` | a default that changes between calls or ignores the caller's host envelope |
| `TestGeneratedCatalogUpToDate` | hand-edited or stale generated surfaces |
| `TestEveryOperationHasCLIAndJourney` | the 7-of-8 server-only gap |
| `TestGeneratedOperationSpecsUpToDate` | UI forms drifting from the wire |
| `TestModelDefaultsAreCommercialAndQualified` | a non-commercial/experimental default |
| `TestModelProviderReferencesResolve` | a model pointing at a retired provider |
| `TestDocsModelAndProviderMatrixUpToDate` | documentation drift |
| `make coverage-floors` | thin packages (extend the existing policy) |

## Non-goals

- A plugin/exec sandbox. New runtimes and semantics stay reviewed code.
- Merging media scenarios into one module. Interop is a contract.
- Data-izing streaming cadence. Strategy eligibility is data; the state machine
  and recovery behavior are code.
- Secrets, entitlement, or settlement in manifests.
- A big-bang rewrite. Each phase leaves the tree green.

## Cross-references

- [ARCHITECTURE.md](../concepts/ARCHITECTURE.md) — system shape and seams
- [DOMAINS.md](../concepts/DOMAINS.md) — capability ownership
- [TESTING.md](TESTING.md) — evidence contract
- [PROBLEMS.md](PROBLEMS.md) — incident-shaped findings
- [PROGRESS.md](PROGRESS.md) — dated milestones
- [`../../image-tools/docs/internal/CAPABILITY-MANIFEST.md`](../../image-tools/docs/internal/CAPABILITY-MANIFEST.md) — the sibling substrate that motivated this program
