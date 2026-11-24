const e=`# Maturation Program — Audio Tools

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
| Debt register | \`PROBLEMS.md\` 2,631 lines, 22 unassigned entries |
| Duplication | \`dupl -t 80\` 28 clone groups; three chain packages + shared tiered/BYOK scaffolding ~5.4k LOC |
| Audio operations exposed | 8 exist in proto/handlers; **only \`transcode\` is in CLI and UI** |
| Engine declaration | \`sttengine/manifest.json\` declares 2 engines; bootstrap constructs 3; docs/UI name 3 |
| Provider spread | one provider id appears across ~20 non-test files |
| Coverage floors | \`bootstrap\` 754 src / 116 test; \`stt/pipeline\` floor 20%; \`experiment\`, \`eval\`, \`soak\`, \`validation\` have no floors |
| Hot spots | \`soak/driver.go\` 1,591; \`overlap_agree.go\` 912; \`experiment/report.go\` 884; \`runner.go\` 643; \`vad_segment.go\` 632 |
| Shared UI | \`useVoiceCore.ts\` 2,095; \`pcmVoiceStreamProvider.ts\` 918; \`turnJournal.ts\` 338 |
| Model catalog | no registry; free-form model IDs; gateway has no STT/TTS/duplex roles |

## Stance

Operator direction (verbatim):

> Be ambitious; orchestrate subagents to do significant portions in parallel.
> Prefer data-driven architecture: adding/upgrading a model or provider should
> be a declaration, not a code edit. Continue to drive down bugs, performance
> issues, race conditions, portability issues, debt, duplication, and
> unnecessary complexity.

> Also we seem to be using a lot of resources right now. Not sure how much of
> that comes from audio-tools, but it might and I want to make aure we use
> appropriate models. I think the defaults should be the much smaller ones in
> many cases, for tts, stt, voice cloning, etc. These smaller models typcially
> are sufficient and they free up our machine enough for other things that
> actually need it.

> Yeah I want stt using sherpa streaming zipformer, tts using kale nano when
> it's avilable. Sounds like we have a lot more work to do to support all of
> this properly. Please continue

> (2026-11-21) There is no deliberately chosen models. We're trying to make
> things more efficient here.

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
| Provider declaration | \`api/internal/providercatalog/catalog.seed.json\` + loader/validation + generated \`capabilities\` and TS catalogs | \`TestGeneratedCatalogIsNotStale\`, \`TestProviderEngineMappingIsBidirectional\`, \`TestLocalProviderIDsDeterministicAndResourceUnique\` |
| Cross-catalog integrity | provider↔model↔engine conformance | \`api/internal/catalogconformance\` (references, capability match, engine ownership, licence/default rules) |
| Engine drift | \`sttengine\` now declares \`sherpa-streaming\`; bootstrap and catalog agree | \`TestProviderEngineIDsMatchSTTEngineManifest\` |
| Operations | \`api/internal/operationcatalog\` declares 13 operations; all 8 audio transforms are reachable from CLI and the Diagnostics UI | \`TestAudioManifestCoversEveryOperation\`, generated \`audioOps.generated.ts\` |
| Models | \`api/internal/modelcatalog\` with 23 records + generated \`docs/reference/models.md\` | selection/default/licence tests + doc drift |
| Promotion receipts | Experiment qualification evidence owns corpus-quality receipts; persisted passing real-time browser product-path receipts with the explicit \`interactive-latency\` claim now own the latency facet. The claim requires 30+ attempts and passing latency assertions; replay timing alone is not sufficient. | \`api/internal/modelpromotion\` verifies the stored receipt identity and claim; memory, platform, and licence remain blocked without integrated owners |
| Model policy | \`whisper-large-v3-turbo\` and \`rumble-2-flash\` are candidate defaults; commercial/non-commercial/readiness rules are explicit | \`DefaultFor\` requires the \`default\` role; \`CommercialDefaultFor\` fails closed |
| Provider identity | public results report canonical provider IDs; engine IDs remain internal evidence | \`CanonicalProviderID\` + handler tests |
| STT reliability | VAD retry backoff, leading-silence trim, OverlapAgree committed-PCM compaction, Passthrough terminal \`Done\`, session snapshot ordering | focused \`-race\` regressions in \`internal/stt/{strategy,segmenter,session}\` |
| Audio correctness | AAC/ADTS, trim duration, fade-out anchor, merge crossfade label/temp cleanup, cancellation mapping, split validation, audio-only ffprobe | \`internal/audio\` + \`handlers/audio\` edge/integration tests |
| Concurrency/aliasing | Whisper dependencies snapshot behind a lock; stream, wake-word, and speaker settings use store-backed owners instead of duplicate pipeline state | \`TestServiceDependencySettersAreRaceFree\`, handler repository tests |
| Gates | measured coverage floors for bootstrap/lab packages; \`stt/pipeline\` floor is 55%; goleak ignores only the shared sql global | \`TestCoverageFloors\`, \`go vet\`, targeted \`-race\` |

At this point in the original tranche, the admin picker had not yet joined
\`ListModels\` to the engine rows. That gap is closed in the third tranche below.
Rumble/Kale runtime adapters, gateway audio roles, full-duplex, and the
device/evidence matrix remain phased work.

## Delivered in the second tranche (2026-11-19)

The registry stopped being documentation and became a runtime surface.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Model service | \`ModelRegistryService\` (\`ListModels\`, \`ResolveModel\`), mounted at \`api/handlers/models\` | Connect transport test + \`TestResolveModel_CommercialNeverReturnsNonCommercial\` |
| Host-aware defaults | \`Platform\` on list/resolve; \`is_default\` is host-relative | \`TestListModels_PlatformConstrainsDefaultFlag\`, \`TestResolveModel_PlatformConstraintCanMakeDefaultIneligible\` |
| CLI | \`audio-tools model list\` / \`model resolve\` with filters, JSON, and policy explanation | manifest↔handler coverage test; full CLI suite |
| UI | \`/models\` page: filters, readiness/licence/hardware facts, default badge, resolve panel | 29 component/service tests; full UI suite 134 files |
| Registry honesty | Summarize records mirror \`resources/ollama/model-policy.json\`; \`qwen3.5:9b\` is the runtime primary, \`llama3.2:3b\` remains the commercially certified/evidence-backed default | \`TestSummarizeRecordsMirrorResourceOllamaPolicy\` |
| Model-truth seam | Whisper env model ids must resolve in the registry and round-trip | \`TestWhisperModelIDsResolveInRegistry\` |
| Engine integrity | Provider engines, model engines, and \`sttengine/manifest.json\` are compared in both directions, including resource slugs | \`TestProviderEnginesMatchSTTEngineManifest\`, \`TestModelEngineIDsAreShippedBySTTEngineManifest\`, \`TestActiveLocalSTTModelsNameAnEngine\` |
| Streaming teardown | A decoder parked on a full frame buffer is unblocked by \`Close\` | \`TestProcessDecoderCloseUnblocksFullFrameBuffer\` |
| STT residuals | Rejected timeline ranges are observable (not a clean EOF), normalizer teardown joins its workers, strategies no longer mutate shared config, and cancel performs a bounded best-effort tail flush | focused \`-race\`/goleak regressions in \`internal/stt/{segmenter,strategy}\` |
| Chains | Shared BYOK/LPBS/stamping scaffolding consolidated into \`chains/tiered\`; streaming credit exhaustion is terminal instead of silently falling back; fallback reasons redact request credentials; availability probes cannot resurrect across \`Reconfigure\`; coverage restored to 81/86/94/83 | \`TestCoverageFloors\`, \`dupl -t 80\` = 0 clone groups, wrapped-sentinel mapping tests |
| Surface drift | \`.vrooli/endpoints.json\` regenerated for all eight audio operations and the model service | \`make endpoints\` + diff gate |
| UI debt | \`pnpm lint\` is clean (was 65 problems) | full UI suite + lint |

Still open after this tranche: Rumble/Kale adapters, gateway audio roles,
full-duplex, and the device/evidence matrix. The engine→model join is delivered
in the third tranche below.

## Delivered in the third tranche (2026-11-19)

This tranche made runtime truth measurable instead of asserted.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Whisper runtime | \`resources/whisper\` now pins \`ggml-large-v3-turbo.bin\` (checksum-verified) as the eager artifact; \`ggml-medium.bin\` is the \`medium-model\` on-demand rollback group. The capacity profile names the turbo digest | \`TestWhisperRuntimeArtifactMatchesRegistryDefault\` (registry default must equal the artifact the resource actually starts, and it must be eager) |
| Artifact truth | Model records name the resource artifact that supplies local weights; every \`available\` local model must name a declared artifact | \`TestAvailableLocalModelsNameShippedResourceArtifacts\`, loader rule \`invalid_artifact\` |
| Model receipt | The canonical \`quality_speech.wav\` scores WER 0.00 on turbo and medium; the 61-minute dictation fixture transcribes in 30.3 s (turbo, RTF 120.8x) versus 52.8 s (medium, 69.3x) with 0.9993 word-sequence agreement on one Linux/Vulkan host | Seed \`benchmarks\` + dated PROBLEMS entry; explicitly **not** a device-matrix or timestamp-stability claim |
| Audio responses | EBU R128 normalization is two-pass (linear) and \`measured_lufs\` is measured from the produced audio, never the requested target; \`duration_seconds\` is populated for transcode, trim, and merge | \`internal/audio/loudness_test.go\`, \`duration_test.go\`, handler tests |
| Lifecycle idempotency | Mutating provider actions dedupe on \`(Idempotency-Key, operation, target)\` with single-flight joiners, TTL, bounded eviction, no error caching, and cloned responses | \`handlers/provider_lifecycle/idempotency_test.go\` + handler tests (81.7% coverage) |
| Admin picker | Stream Config joins each STT engine to the registry's declared models (default badge, readiness/licence, runtime, size, languages, explicit no-model state) | \`StreamConfigPage.test.tsx\` (11 tests), full UI suite |
| Transport hardening | Transform and STT Connect handlers reject oversized messages before buffering them (256 MiB per transform request, 16 MiB per STT message) | \`TestModuleRejectsOversizedRequest\`, \`TestSTTModuleRejectsOversizedMessage\` |

Still open after this tranche: Rumble/Kale runtime adapters, timestamp-drift
scoring on long-form audio, a multi-device/multi-OS matrix, and full-duplex.
The gateway audio-role question is settled below.

Deliberate non-change: AI Gateway audio roles (\`stt.fast\`, \`tts.quality\`, …)
were **not** declared. The gateway's role catalog dispatches text-inference
candidates owned by resource providers (\`ollama\`, \`openrouter\`, \`pca\`, …); local
ASR/TTS resources are not in that provider family, so a role there would be
inert configuration that looks like capability. The audio role surface is
\`ModelRegistryService\` (\`ListModels\`/\`ResolveModel\`), which already applies
licence, readiness, language, and platform policy. Revisit the gateway only when
it owns an audio provider family and a dispatcher, not before.

## Delivered in the fourth tranche (2026-11-19)

This tranche closed user-visible correctness bugs and removed the last
hand-maintained provider/voice tables.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Transform correctness | An exit-zero ffmpeg that writes nothing is a failure; trim rejects a start past the measured end and clamps end; split reports the open-ended final chunk duration and rejects boundaries past the end; merge rejects a crossfade longer than the shortest source; transcode knobs and loudnorm targets are range-checked | \`internal/audio\` unit + real-ffmpeg integration tests |
| Streaming cancellation | STT strategy eligibility resolves the active engine under the session context, not \`context.Background()\` | \`TestSelectorHonorsCanceledContextDuringEngineResolution\` |
| Stream decode truth | A mid-stream decoder failure is reported as \`input_decode_failed\` and fails the replay ledger instead of looking like a clean EOF | \`TestNormalizeChunksReportsDecoderFailure\`, \`TestStreamErrorCode_UsesBackendState\` |
| Composition truth | Every endpoint declared in \`modules.AllEndpoints\` must be mounted by \`Compose\`; typed-nil optional repositories no longer panic during construction | \`TestCompose_MountsEveryDeclaredEndpoint\`, \`TestNewConnectHandlerToleratesTypedNilSpeakerConfig\` |
| Voice catalog | One canonical mapping per adapter (\`local:kokoro\`, \`byok:openai-tts\`, \`byok:elevenlabs\`), a real startup \`VerifyCatalog\`, override keys aligned to catalog adapter ids (legacy key honoured), settings rejects unknown voice/adapter pairs, and the Voices page is editable | \`TestTTSAdapterMappingsMatchProviderCatalog\`, settings tests, 10 Voices tests |
| Provider identity | Runtime aliases (\`kokoro-local\`, \`ollama-local\`) and BYOK probe facts live in the provider catalog; the hand switches are gone; active remote providers must declare a probe or a documented absence | providercatalog validation + \`TestBYOKProbeFactsMatchLegacyTuples\`, \`TestBYOKCredentialCheckersHaveCatalogProbes\` |
| BYOK integrity | Credential writes must name an active credential provider that declares the capability | \`TestSettings_BYOKRejectsProviderCapabilityMismatch\` |
| Credentials UX | The Configuration page picks credential providers from the generated catalog, filtered by capability, so a typo cannot create an unusable row | \`ConfigurationPage.test.tsx\` (7 tests) |
| Transport limits | TTS 8 MiB and summarize 4 MiB per message, joining transform 256 MiB and STT 16 MiB | module limit tests |
| Timeouts | Readiness probes use a 30 s doer instead of the 120 s generation doer; BYOK probes are bounded at 5 s; the soak receipt client has a timeout | \`TestReadinessDoerTimeout\`, BYOK deadline test |
| Health telemetry | Liveness checks are timed at the registry seam and \`latency_ms\` is a real measurement (was permanently 0); cache hits report the last check's duration | \`TestLatencyRecorder*\`, \`TestGetProviderHealth_ReportsMeasuredLatency\` |

The headline bug: a voice override saved from the Voices page used the catalog
adapter id \`local:kokoro\`, while the local provider looked up
\`local:kokoro-local\`. The override was stored, shown as configured, and silently
never applied. The keys now share one constant, the legacy key still applies,
and the settings edge rejects pairs that cannot match any adapter.

Still open after this tranche: Rumble/Kale runtime adapters, timestamp-drift
scoring, a multi-device/multi-OS matrix, and full-duplex.

## Delivered in the fifth tranche (2026-11-20)

This tranche made the machine footprint a product decision instead of an
accident, and put the lab boundary behind a measured gate.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Whisper default | \`ggml-small.bin\` (488 MB) is the eager artifact and the runtime default; turbo/medium/base/tiny are on-demand tiers. The capacity claim fell from 2 GiB to 1 GiB | \`TestWhisperRuntimeArtifactMatchesRegistryDefault\`, \`TestLocalDefaultArtifactsAreEager\`, \`vrooli resource validate whisper\` |
| Recommender | \`whisper recommend-model\` returns \`small\` on every usable host; \`--quality\` is the explicit accuracy opt-in; CPU no longer auto-picks medium | \`TestPick_Table\`, \`TestPickQuality_Table\` |
| Summarize default | resource-ollama \`summarize.default\` moved from \`qwen3.5:9b\` (~6 GB resident) to \`qwen3.5:4b\` (3.39 GB installed), which was already the role's qualification candidate; 9B remains a quality tier | \`TestSummarizeRecordsMirrorResourceOllamaPolicy\`, \`TestSummarizeDefaultMovedOffNineB\` |
| Footprint budget | Each capability has a declared default-size budget (STT 600 MB, TTS 400 MB, summarizer 4 GB, speaker 100 MB); a promoted default that exceeds it fails the build, and the aggregate depends on it | \`TestCapabilityDefaultsFitResourceBudget\`, \`TestCommercialDefaultsFitResourceBudget\`, \`TestPromotedDefaultFootprintFitsBudgets\` |
| Footprint visibility | \`docs/reference/models.md\` renders the per-capability default, size, budget, artifact, the promoted-default total, and the catalogue total; \`audio-tools model list\` adds SIZE and a \`Defaults: …\` summary; the Models page shows a Resource footprint panel | docs drift test, CLI/UI tests |
| Lab boundary | Experiment/eval/soak/validation packages have honest coverage floors, \`make lab-test\`, and an import gate that keeps them out of product serving paths | \`api/internal/lab/boundary_test.go\`, coverage policy |
| Health telemetry | Sub-millisecond liveness measurements are reported instead of truncating to 0; \`health show\` renders a LATENCY column | \`TestGetProviderHealth_ReportsSubMillisecondLatency\`, CLI health tests |
| Panel debt | The Models page no longer nests two "Registry" panels; the footprint panel is distinct | \`ModelsPage.test.tsx\` (7 tests) |

The headline change: the host was running \`ggml-medium\` in 1,864 MiB of VRAM
while the registry claimed turbo, and the shared summarize role was pinned to a
9B model. The shipped default is now \`small\` (about one third of turbo's
weights), the capacity reservation is halved, and summarize defaults to the 4B
tier. Bigger models remain one explicit command away.

Still open after this tranche: timestamp-drift scoring on long-form audio, a
multi-device/multi-OS matrix, and full-duplex.

## Delivered in the sixth tranche (2026-11-20)

This tranche made the low-footprint path real end to end.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Summarize policy at that tranche | resource-ollama \`summarize.default\` temporarily moved to \`qwen3:1.7b\` (Apache-2.0, 1.36 GB); a later role split restored generic \`summarize.default\` to \`qwen3.5:4b\` and retained \`qwen3:1.7b\` for \`summarize.audio_tools\` | Historical evidence: model-policy mirror test, \`TestSummarizeDefaultMovedOffNineB\` (updated), ai-gateway catalog regeneration; see current role split under Model policy |
| Sherpa streaming | Local source-built adapter qualified on this host: 2023 English Zipformer (310 MB) on port 8881, \`/healthz\` health contract, \`AUDIO_SHERPA_URL\` corrected from 8880 to 8881 | \`resources/sherpa-onnx/docs/LOCAL-ATTACH.md\`, live \`audio-tools stt transcribe-stream\` |
| Attach-only selection | A provider whose health probe passes is selectable even when the control plane reports the managed artifact unsupported/uninstalled; a dead attached provider is still rejected | \`TestLiveResolverAcceptsAttachedProvider\`, \`TestLiveResolverRejectsUnhealthyAttachedProvider\` |
| Footprint policy | Engine ranking is \`low_footprint\` → \`native_streaming\` → \`accelerated\`; Sherpa Zipformer wins when all engines are serviceable | \`TestShippedPolicyPrefersSherpaOverKyutai\`, \`stt engines\` |
| Kroko stream finalization | Native \`Finish()\` adds a bounded 1.2 s silent tail; JFK smoke preserves “your country” and event timestamps remain bounded by client PCM. Kroko stays on-demand pending broader receipts | opt-in \`TestNativeStreamingModelSmoke\` with Kroko model and \`SHERPA_ONNX_STREAMING_EXPECT_TEXT\` |
| Local TTS runtime selection | \`AUDIO_TTS_BACKEND=sherpa-onnx\` routes existing Kokoro synthesis through sherpa; \`kokoro\` stays default. Cheap request liveness uses sherpa model/encoder health; full readiness exercises speech | \`TestBuildSherpaTTSBackendUsesSherpaEndpointAndCacheBoundary\`, server health test |
| Kale Nano runtime candidate | Linux/amd64 CPU-only composed runtime and model/voice/frontend assets are SHA-pinned; \`compose-candidate --offline\` reproduces the composed tree digest. Audio Tools has an explicit \`kale-onnx\` route with WAV-to-requested-format conversion and backend-scoped cache keys. Remains planned/experimental pending a managed resource start, model-backed product-path receipt, and quality qualification | \`vrooli resource validate kale-tts\`; \`make -C resources/kale-tts test\`; \`TestBuildKaleTTSBackendUsesDedicatedRuntimeModelAndCacheNamespace\`; offline candidate receipt |
| Ollama capacity | Added the \`qwen3:1.7b\` acceleration claim rung (3 GiB) and its \`runtime_identity\` (observed digest); regenerated the gateway catalog | \`resource-ollama policy resolve\`, warm summarize 347 ms |
| Gateway CLI freshness | The gateway's resource-ollama subprocess rebuilt from source while the installed CLI was stale, exceeding its command timeout; after one explicit refresh the gateway path works with no override | live summarize receipt (347 ms warm, ~8.5 s cold) |

Still open: Kale Nano and Tiny weights are publicly accessible; Nano's pinned
Hugging Face revision is public, non-gated, and declares Apache-2.0 (metadata
checked 2026-11-21), while Tiny's publication state remains draft. Nano has a
pinned Linux/amd64 CPU runtime and product route, but lacks joined quality and
lifecycle qualification, so Kokoro remains the catalog default. Kroko
needs broad quality/latency/timestamp/license receipts;
sherpa lifecycle ownership still requires a signed release artifact;
long-form timestamp drift and the device matrix remain unqualified.

## Delivered in the seventh tranche (2026-11-20)

| Area | Delivered | Gate/evidence |
|---|---|---|
| Timestamp scoring | Evaluation can align timed reference/provider words using canonical lexical normalization and report matched-word coverage, signed boundary bias, and absolute p50/p95/max error. Two-bit traceback with rolling cost rows keeps the 10,000-word case bounded to about 28 MB | \`TestScoreTimestampDrift_*\`, randomized parity against \`AlignOperations\`, over-budget rejection test, \`BenchmarkScoreTimestampDrift_61MinuteWordCount\` |
| Long-form scorer performance | A synthetic 10,000-word run completed in 328 ms on the recorded Linux/amd64 host | \`docs/internal/TESTING.md\`; synthetic benchmark only |
| Kale managed candidate | At this tranche, only resource-level Nano evidence was available: a 3.05 s mono 24 kHz WAV in 770 ms cold and 59 ms warm. The later tenth tranche added a model-backed Audio Tools product-route receipt. Kale remains experimental and Kokoro remains the model-catalog default | managed lifecycle receipt + resource-backed \`/v1/audio/speech\`; tenth-tranche product-route evidence; naturalness/release qualification remains open |
| Hypothesis timing retention | Eval replay now retains provider-reported word intervals, bounded at 20,000 per clip, and serializes them as hypothesis data; it does not fabricate a reference or score | replay/report tests; native streaming providers can still omit intervals |
| Portability evidence integrity | Inventory availability no longer counts as inference evidence; the producer records D1 CPU, D1 accelerator, and D2 inference as not run, and the acceptance join rejects missing/unqualified cells | CLI/Python acceptance-board tests; no inference cell passed |

Still open: no human-reviewed timed long-form reference exists, so the scorer
cannot produce qualification evidence. Native streaming intervals are not yet
available from sherpa. Kale synthesis quality and the model release status are
unverified; the release candidate stays stopped and Kokoro remains the default.
The D1–D6 device/browser/OS matrix, sherpa release lifecycle, and a
provider-backed full-duplex integration also remain.

The current state and next outcomes are kept in
[MATURATION-STATE.md](MATURATION-STATE.md), which is edited in place; dated
evidence goes to [PROGRESS.md](PROGRESS.md) and [PROBLEMS.md](PROBLEMS.md). This
program remains the stable scope, stance, policy, and acceptance charter.

## Delivered in the eighth tranche (2026-11-20)

The product/lab boundary now holds at bootstrap, and offline fault evidence is
executable without being confused with product qualification.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Product/lab composition | The supervisor-owned \`default\` profile runs API + UI with product routes; the separate API-only \`lab\` profile injects \`VROOLI_VARIANT=lab\` and selects corpus, experiment, soak, and validation-provider routes. Caller \`AUDIO_ENABLE_LAB_API\` is ignored. | \`.vrooli/service.json\` profile declarations; bootstrap env test; composition route partition tests |
| Endpoint inventory | External Scenario Validation provider endpoints are included in \`modules.AllEndpoints\`; CLI omissions explain that Test Genie owns them | \`make endpoints\`; proto parity remains scoped to Audio Tools-owned proto files |
| Offline fault suite | Provider timeout, truncated decoder stream, and cancellation/worker cleanup execute through real server seams; diagnostic artifact explicitly has zero browser product-path rows and is non-promotable | \`api/internal/stt/segmenter/testdata/offline-fault-matrix.json\`, \`coverage/offline-fault-diagnostics-20261120.json\`, focused \`go test -race\` |
| Duplex lifecycle | Concurrent \`Handle.Close\` callers now wait for provider cleanup and receive the same close result | blocking close regression; focused \`go test -race ./internal/duplex\` |
| Coverage regression | New endpoint partition checks cover all \`internal/modules\` branches; stale CLI evidence regenerated by its owning test | module coverage 100%; full API race suite and CLI suite |
| Model/runtime attribution at this tranche | CLI resolved Sherpa Zipformer and Qwen3 1.7B; Ollama's resident 9B model is not the summarize default | [model footprint audit](MODEL-FOOTPRINT-AUDIT-2026-11-20.md) |
| Provider runtime identity closure | Canonical runtime identity now resolves declared provider IDs, engine IDs, and aliases from the provider catalog; conformance round-trips all declarations | \`go test ./internal/providercatalog ./internal/capabilities ./internal/catalogconformance\` |
| Kale runtime smoke | At this tranche, started the managed draft candidate, verified health and ten voices, and synthesized a 99,644-byte mono 24 kHz WAV in 0.74 s; the resource was stopped afterward. The later tenth tranche added an isolated model-backed product request | resource-level \`/v1/audio/speech\` smoke; later tenth-tranche product-route evidence; no quality qualification |

Still open: the running shared API process predates the product/lab source and
was not restarted; the offline fault suite leaves browser-only profiles
explicitly unsupported; no reviewed timed long-form references or full device
matrix exist; Kale still has no quality/release qualification; and GPT-Live
still needs the user-selected Platform target, confirmed local key destination,
authenticated product route, durable spend ledger, and production privacy
enforcement.

## Delivered in the ninth tranche (2026-11-20)

Timed-reference annotations can now enter and persist through the supported
corpus API and CLI. This creates the path for reviewed alignment evidence; it
does not create reviewed labels or qualify a model.

| Area | Delivered | Gate/evidence |
|---|---|---|
| Timed reference ingestion | Corpus proto, create/list/get API, SQLite persistence, and \`audio-tools corpus import --timed-reference-file\` carry word intervals and review provenance. Validation bounds ordering, clip duration, reviewer fields, timestamps, and payload size | \`make -C packages/proto verify-committed-gen\`; API race tests for corpus handler/storage; CLI corpus tests |
| Provider runtime identity | One provider-catalog resolver canonicalizes declared provider IDs, engine IDs, and aliases; conformance round-trips each identity | focused \`providercatalog\`, \`capabilities\`, and \`catalogconformance\` Go tests |
| Kale model-backed runtime smoke | Managed draft resource started, returned its voice catalog, and generated a valid short WAV; resource stopped cleanly | resource-level smoke only; product handler, naturalness, and qualification remain open |

Still open: human-reviewed timed references and a named model's timing score;
the Audio Tools product-path and quality receipt for Kale; full browser/device
fault qualification; provider-backed duplex with settled cost enforcement;
and GPT-Live's selected Platform target, secure credential destination, and
implementation. The live shared API remains untouched.

## Delivered in the tenth tranche (2026-11-20)

| Area | Delivered | Gate/evidence |
|---|---|---|
| Kale product route | An isolated bootstrap/Connect integration test exercises the actual product TTS route against a fake Kale runtime, including model and voice mapping, WAV output, provider metadata, and cache reuse | \`go test ./internal/bootstrap -run '^TestBuildKaleTTSProductConnectRouteAndCache$' -count=1\`; not real-model product qualification |
| Kale model-backed product route | The isolated Connect product route synthesized through the managed Kale Nano runtime, returned 98,424 bytes of WAV with 2,048 ms measured duration, and hit cache on repeat; 1.25x speed output retained valid finite WAV chunk sizes | \`go test ./internal/bootstrap -run '^TestBuildKaleTTSLiveProductRoute$' -count=1 -v\` with \`AUDIO_KALE_URL\`; route/runtime integration only, no naturalness or release qualification; resource stopped |
| Duplex spend primitive | Admission checks remote-processing permission, explicit consent, active provider/model catalog eligibility, and bounded process-local reservation before provider contact. The current planned GPT-Live declaration is intentionally ineligible. Unknown or un-settleable final usage retains the in-memory hold for the process lifetime, preventing reuse; this is conservative capacity protection, not durable billable-spend accounting or reconciliation | \`go test -race ./internal/duplex\`; production ledger and reconciliation remain open |
| Duplex composition fence | Route composition independently requires active catalog eligibility as well as every injected admission dependency; an injected predicate cannot promote the planned GPT-Live provider/model | Focused \`go test -race ./internal/server\` composition gates; route remains unmounted for the current catalog |
| GPT-Live catalog and WebSocket adapter | Catalog declares sourced integer micro-USD active/standby rates and one-second billing increments, without promoting duplex as a default. Adapter maps audio/transcript events, bounds reads/commentary, and prices confirmed final usage. \`StoreKeyResolver\` reads canonical credentials from encrypted BYOK storage. Barge-in uses the documented output fixture; user text input is unsupported. \`ProviderRegistry\` binds stable provider IDs and session kinds to adapters; routed coordination requires an explicit registered duplex provider and has no fallback | \`go test -race ./internal/duplex\`; authenticated product route, durable ledger, and privacy policy remain open |
| TTS automatic local selection | Source \`auto\` mode keeps the promoted-default gate for ordinary providers and honors the explicit \`conditional_auto\` declaration for planned providers with commercial candidate models. Kale Nano is tried before qualified Kokoro when available, with Kokoro as availability fallback; Kale remains planned/candidate and is not promoted. Explicit backend selection stays pinned, and response/cache identity follows the actual backend. \`audio-tools model resolve --capability tts\` reports the qualified catalog default and also explains this conditional product route | TTS/bootstrap and CLI model focused tests and isolated live Kale route; joined Kale quality and release qualification remain open |
| Active streaming default | Live CLI stream configuration resolves to \`sherpa-streaming\` with \`passthrough\`; summarize model resolution remains \`qwen3:1.7b\` | \`audio-tools stt stream-config --json\`; \`audio-tools model resolve --capability summarize --json\` |

Still open: the live shared API process has not been restarted onto the current
product/lab composition; Kale voice-quality/release qualification; reviewed
timed references and qualified timing scores; browser/device
fault coverage; authenticated duplex route and durable spend accounting; and
the user-selected Platform key target and confirmed secret destination.

## Continuation update (2026-11-20)

| Area | Update | Evidence / remaining gate |
|---|---|---|
| TTS failover robustness | Nil preferred/fallback services now fail or fall back cleanly without a panic; the healthy fallback remains usable if the preferred service is absent | \`go test -race -count=1 ./internal/ai/ttschain\` |
| Footprint policy reconciliation | Declarations at this update resolved batch STT to Whisper Small (488 MB), streaming to Sherpa Zipformer (310 MB), summarize to Qwen3 1.7B (1.36 GB), and speaker verification to Sherpa embeddings (~27 MB). Kokoro remains the TTS catalog default; \`auto\` excludes the unqualified Kale Nano candidate even when its runtime is healthy | model footprint audit and agent review; Kale promotion still needs joined lifecycle and held-out multi-voice quality evidence. Shared Ollama 9B residency is attributed to \`chat.default\`, not Audio Tools summarize |
| Verified identity context | Audio Tools' custom HTTP server now installs api-core authentication and provenance middleware, with the \`audio-tools\` scenario-authenticator audience enabled; tests prove a verified human reaches domain code | \`go test -race -count=1 ./internal/server\`; handlers still own authorization and existing routes were not globally gated |
| Duplex authorization boundary | Verified identity context is now available, but no duplex handler enforces a capability, source-bound consent, or account budget. Request booleans cannot authorize remote processing; process-local reservations are not durable spend authority | read-only auth survey against \`api-core/authn\` and AI Gateway consent precedent; do not expose external billing until revocable consent and durable account budget/settlement exist |
| Migration reliability | Added BYOK migration error/rollback cases; database coverage rose from 73.1% to 92.3%, clearing its 80% package floor | \`go test -coverprofile=/tmp/audio-tools-database.cover -count=1 ./internal/database\` |
| Kroko finalization regression | Re-ran the existing native 1.2 s silent-tail fix with the current Kroko model against two JFK inputs. Both outputs retained the expected “your country” ending and final timestamps stayed bounded by source PCM; 11 s/55 s clips processed in 362/844 ms | native-tagged server smoke; source transcript-tail regression only, not general accuracy/latency qualification or a default promotion |

Full current API validation: \`go test -race -count=1 -timeout=300s ./...\`
passed across all packages, including the coverage-floor gate.

The OpenAI Platform picker is awaiting organization/project selection. Key
creation and local secret write remain after the user confirms that target and
the ignored local env-file destination. No live OpenAI API request is enabled
yet.

## Continuation update (2026-11-21)

- **Timestamp scoring:** normalized identity streams at the admitted 20,000-word
  limit now use a linear-time path (6.18 ms / 4.57 MB on the recorded host).
  Packed traceback allows a full 20,000 × 20,000 alignment under a 128 MiB
  direction-buffer limit. Dissimilar text retains quadratic worst-case time;
  no human-reviewed long-form corpus or model qualification receipt exists.
- **Offline fault coverage:** deterministic HTTP 503 provider-busy and dropped
  provider-connection cases now terminate without transcript output. The
  current receipt is
  \`coverage/offline-fault-diagnostics-20261121.json\`; it remains
  \`promotable:false\`, with no browser/device rows. Delayed readiness,
  slow-consumer, acknowledgement, backend restart, and browser/device faults
  remain uncovered.
- **Browser capture decomposition:** provider selection and PCM warm-up moved
  behind \`voice/providerOrchestration.ts\`; 29 package files pass (230 passed,
  1 skipped) and the package build succeeds. This is incremental extraction;
  \`useVoiceCore\` remains a large orchestration hook.
- **Whisper host telemetry:** the generic Vrooli health command disagreed with
  the resource-specific status and a direct HTTP 200. The read-only finding was
  filed as \`knw-1795238304545903315\`; no shared resource was changed.
- **Summarize catalog mirror:** the Audio Tools catalog's summarize default
  mirrors the shared \`summarize.default\` policy primary. The
  catalog had incorrectly retained \`llama3.2:3b\` as another \`default\` role;
  that role is removed while the licensed model remains a fallback. The mirror
  test now requires the commercial default to match the policy primary, and
  generated model documentation was refreshed. \`make catalog-check\` and the
  full model catalog suite pass. This describes the policy at that point; the
  current app-scoped role split is documented under Model policy below.
- **Browser PCM retention:** full-turn PCM copy, size budgeting, overflow, and
  reset state now live in \`RetainedPcmBuffer\`; journal-based recovery remains
  in the provider. Focused behavior tests pass. The browser package suite passes
  (30 files, 233 passed, 1 skipped) and its build succeeds.
- **BYOK health composition:** the bootstrap no longer repeats the four
  probed provider/capability tuples. It constructs remote credential checkers
  from active catalog rows that declare a probe. A focused test verifies the
  produced IDs, capabilities, probe wiring, and that OpenRouter retains its
  separate environment-backed checker. Local resource checker constructors
  and aggregate health composition remain explicit in bootstrap.
- **Sherpa TTS route:** Audio Tools already has a Sherpa-backed local TTS route
  (\`AUDIO_TTS_BACKEND=sherpa-onnx\`) with voice mapping, backend-specific cache
  identity, and product event-cache replay tests. This does not change the
  runtime gate: the Sherpa endpoint is currently down and managed Linux
  lifecycle remains unsupported until a trusted signed adapter release exists.
- **Attached Sherpa serviceability:** resource declaration and schema now
  explicitly allow endpoint health probes when managed lifecycle is unsupported.
  The capability registry probes attached endpoints but leaves lifecycle
  support unsupported. The first post-change restart reported Sherpa STT
  available at 0.94 ms, with speaker verification unavailable because its model
  was not loaded. After the later package-aware restart, the attached endpoint
  stopped answering on port 8881 and current health reports Sherpa unavailable;
  streaming still resolves to Sherpa but requires a healthy attachment. Sherpa's
  service dependency uses SDA-applied \`startup_policy=ignore\`, preventing
  unsupported managed start attempts. A signed managed bundle or durable,
  authorized attachment remains the runtime gate.
- **TTS declarations:** auto-priority bootstrap tests now copy provider runtime
  records before changing test priority, preventing shared-catalog mutation
  from making results test-order-dependent. Unique request IDs isolate repeated
  fallback/cache runs. \`go test ./internal/bootstrap\` passes; the production
  fallback path is unchanged; Kale remains unpromoted pending release/license
  and held-out multi-voice quality evidence.
- **Sherpa footprint boundary:** the selected 2023 Zipformer and punctuation
  model now form a checksum-verified acquisition group. The resource defaults
  to streaming STT without loading Kokoro TTS; Kokoro is on demand, while the
  catalog's speaker-verification default stays eagerly available. Capability
  health probes distinguish STT from TTS readiness. Native model/WebSocket
  smokes pass, and an attached product-route fixture transcription completed
  in 202 ms. Resource/API source gates pass, but managed Linux lifecycle still
  awaits the trusted signed adapter release; the 4-second single-clip result
  is not model quality qualification.
- **Sherpa-backed TTS declaration:** the optional Sherpa endpoint, TTS health
  path, and \`sherpa-onnx:\` cache prefix now sit on the Kokoro provider's shared
  runtime declaration. Its \`fast\` model role resolves the distinct
  \`kokoro-82m-v1.0-sherpa-int8\` catalog record and Sherpa artifact; the model
  remains a candidate and is not the promoted default. Bootstrap resolves the
  URL, health probes, protocol model ID, and backend-scoped cache identity from
  declarations while retaining Kokoro provider and voice identity. A product
  Connect test verifies requested MP3 format, returned bytes/content type,
  response model identity, and event-cache replay. Provider catalog, model
  catalog, bootstrap and TTS tests plus \`make catalog-check\` pass; managed
  Sherpa serviceability remains unsupported on this Linux host.
- **Secure BYOK CLI input:** \`audio-tools settings byok-upsert\` supports
  \`duplex\` and now accepts the secret only through \`--key-stdin\`, capped at
  8 KiB. Output contains the redacted fingerprint only; missing input intent,
  read failures, over-limit input, and secret echo have focused coverage.
  The generated CLI primitive-evidence artifact was refreshed through its
  owning verifier; the full CLI race suite, vet, and portability builds pass.
- **Current resource attribution:** the 2026-11-21 08:21 UTC host sample found
  \`qwen3.5:9b\` resident in shared Ollama, not Audio Tools summarize; the
  selected \`qwen3:1.7b\` default was absent from residency. Whisper Small used
  719 MiB VRAM, while the pinned Sherpa stream route was unavailable because
  managed lifecycle is still unsupported. See the dated footprint audit for
  process-level readings and non-claims.
- **Test Genie unit gate:** \`20261121-084515-d5ee04c5\` passed (63 seconds); the previous failure led to a robust subsequent-session assertion in the backend-restart fault test and a strict tuple typing fix in the shared browser capture package. The package was rebuilt and the Audio Tools UI dependency refreshed. This unit phase still reports 61 low-coverage advisories.
- **W2 evidence read:** the setpoint program completes with \`partial\`: three
  diagnostics are readable, the evidence-list endpoint returns a remote error
  on the product-only API, and 0/13 acceptance rows resolve. The requirements
  validator now passes without business-contract findings after comprehensive
  run \`20261121-085049-9df8b854\`. Do not enable lab routes on the product API
  to hide this gap; a separately supervised lab identity is being established.

## Continuation update (2026-11-23)

- **Soak CLI reliability and coverage:** tests now exercise trusted Agent
  Manager/BAS/Test Genie receipt admission, exact provider-cell identity,
  Audio Tools API preflight, request flags/headers, both result formats, and
  protected evidence export. The owner package rose from 17.3% to 89.1%
  statement coverage. The tests exposed and fixed whitespace handling for
  positive integer flags; unused clock injection was removed from the soak
  registration boundary. Full CLI race tests and vet pass.
- **Close before terminal Done:** a real Connect bidi handler test cancels a
  client while the batch provider is in flight, verifies cancellation reaches
  the provider, and verifies the handler exits without waiting for a \`Done\`
  frame. It passes 25 race-enabled repetitions. This is handler-level
  diagnostic coverage; it does not create browser/product-path evidence.
- **Delayed provider readiness fault:** the manifest-driven server stream fault
  matrix now models a provider returning unavailable until a bounded readiness
  delay. It verifies the early stream emits no transcript and a subsequent
  stream completes after readiness. The matrix passes under \`-race\`, and the
  new case passes 10 repeated race-enabled runs. Receipt:
  \`coverage/offline-fault-diagnostics-20261123.json\`. Six other required fault
  profiles remain unsupported by this server-only matrix; it remains
  non-promotable without browser/product evidence.
- **Timestamp alignment scaling:** \`timestampMatches\` removes exact shared
  suffixes and only trims a shared prefix while each consumed word occurs once
  in each complete stream. This preserves canonical tie behavior for repeated
  words while avoiding dynamic-programming work over safe unchanged regions.
  On the recorded Linux/amd64 host, a synthetic 20,000-word case with one
  middle correction completes in 7.71 ms with 6.98 MB allocated; the former
  suffix-only path would align a 10,001 × 10,001 window (about 100 million
  cells). The first-token case completes in 7.33 ms with 5.23 MB. Race and
  randomized canonical-path tests pass. Fully changed/repeated-word inputs
  retain quadratic worst-case work, and no human-reviewed timing corpus or
  model qualification is claimed.
- **Turn-journal replay integrity:** startup replay now requires PCM record
  sequence/sample continuity, valid non-empty ranges, and acknowledgement
  cursors within the captured range. A malformed later record rolls back
  partial replay rather than leaving a partially recovered journal. Direct
  shared-package tests pass: 40 files, 281 tests passed, one skipped. This is
  deterministic journal evidence, not browser recovery qualification.
- **TTS start retry:** canceled/deadline-exceeded lazy backend startup no longer
  populates the 20-second cooldown. A deterministic canceled-leader/immediate
  retry test and the API race suite pass. Ordinary backend-start failures remain
  cooldown guarded; no live synthesis was used.
- **Duplex catalog gate:** server composition now enforces canonical GPT-Live
  provider/model eligibility in addition to the injected owner predicate and
  required consent, journal, account-credit, and budget dependencies. A
  concurrent eligibility-change regression passes under \`-race\`. The current
  catalog keeps GPT-Live planned, so the production route remains unmounted.
- **UI package snapshot:** source package tests pass, but ordinary Audio Tools
  UI tests resolve an older installed local package copy missing
  \`audioCueSession\`. \`vrooli package refresh audio-capture-browser
  --no-restart\` deferred consumer setup while Audio Tools is running. A
  temporary Vitest alias to the current shared package source passed all 136
  UI files / 1,296 tests; type-check and lint pass. The live consumer was not
  restarted, so the default UI test command still needs a package refresh.
- **Browser capture decomposition:** mic-lease recovery coordination moved
  from \`useVoiceCore.ts\` into \`voice/micLeaseRecovery.ts\`, with focused tests
  for in-flight microphone starts and synchronous reentrant release
  notifications. The package build and complete suite pass (40 files, 278
  passed, 1 skipped); microphone capture behavior is unchanged.

### W2 program endpoint routing dependency (2026-11-21)

- Historical blocker (resolved 2026-11-23): the \`audio-tools.setpoint-read\` binding initially had no governed per-instance lab target. The supervisor-owned \`lab\` profile now runs an API-only instance with its own port and isolated storage namespace; Program Runtime routes the declared binding to that instance. The product profile remains product-only. See the resolution and current evidence below.

**Resolution (2026-11-23):** The supervised lab profiles are now running separately: \`audio-tools@lab\` serves its API-only lab composition on port 15373, and \`program-runtime@lab\` serves the governed lab runner on port 16555. Running \`audio-tools.setpoint-read\` with \`program-runtime --instance lab\` succeeds through the lab-owned evidence binding; the product instance still omits that lab route. The current board joins 5 of 18 required rows from one simulated-policy cohort and keeps acceptance unknown. This resolves the evidence-routing dependency, not the product qualification gap.

- **Kale Nano product-path smoke (2026-11-21 09:19 UTC):** the current auto TTS route selected the managed \`kale-tts-1-nano\` model for the canonical neutral voice. The first product request took 4,992 ms and the warm request 42 ms; both produced finite 24 kHz mono PCM WAV outputs. The verified CPU resource is healthy at about 216 MiB RSS. This proves routing and warm serviceability only; held-out multi-voice naturalness and model qualification remain open.
- **Chain precedence consolidation:** \`tiered.Order(localFirst)\` now owns shared provider tier order for unary coordination, STT candidate selection/streaming, and TTS streaming. STT engine selection, streaming gates, STT \`RequireStreaming\`, and TTS terminal-credit behavior remain capability-specific. Focused tiered/STT/TTS/summarize tests pass.
- **Canonical GPT-Live credential setup:** the Configuration page no longer offers to save a duplex key into the legacy Audio Tools BYOK store; it displays the existing Vrooli credential-authority provisioning command instead. \`vrooli credentials status --identity vrooli/openai-gpt-live --field api-key\` reports the encrypted-file provider available and the field unconfigured. No Platform key-create action is exposed in this session, and the native setup chooser returned no confirmed choice, so no key was created. The lazy provider resolver reads this canonical identity and observes rotations, but credential provisioning still does not authorize or mount GPT-Live; consent, durable billing/recovery, and provider-usage evidence remain required.

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
lifecycle, settings, and usage (\`whisper-stt\`, \`sherpa-stt\`); an **engine id**
is the STT stream-selection identifier (\`whisper-local\`, \`kyutai\`,
\`sherpa-streaming\`). The catalog owns the mapping in both directions.

### B. Consolidate the runtime scaffolding

The three chain packages already share \`internal/ai/chains/tiered\`; finish the
consolidation where it still pays:

- one availability/health path, one fallback policy, one lifecycle path;
- generated/declared bootstrap wiring instead of hand-built checker maps;
- remove the legacy \`pipeline.Service\` shim responsibilities and raise its
  coverage floor;
- separate the lab/qualification runtime (\`experiment\`, \`eval\`, \`soak\`,
  \`validation\`) from the product runtime, at minimum behind a \`lab\` boundary.

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

- \`default\` is per capability × device × language × commercial-use;
- \`candidate\` / \`experimental\` / \`watch\` never become defaults by accident;
- a model cannot be promoted without a receipt (corpus quality, latency,
  memory, platform, license). The registry requires owner-referenced qualified
  receipts for all five dimensions before a new default role is eligible.
  Existing operational defaults without a joined receipt are explicitly
  grandfathered as unqualified and remain a documented limitation. The
  exception is a code-owned allowlist of the four current model IDs and their
  exact reason strings; seed data cannot authorize a fifth model.

The current gate resolves corpus-quality references against the experiment
service's persisted \`QualificationEvidence\` rows, requiring a passing owner
verdict and matching model, capability, engine, and strategy in the machine
receipt. The latency owner accepts only a persisted browser product-path row
whose machine receipt contains the real-time \`interactive-latency\` claim from
30 or more passing attempts; raw replay timing is not enough. Memory, platform,
and license promotion receipts still lack integrated owners. The adjacent
sources audited do not satisfy those contracts: host-portability rows currently
record an incomplete aggregate matrix as \`passed: false\` (and their bounded
cgroup sample is not a model-isolated memory qualification); resource manifests
declare platform support but do not qualify model inference across the target
matrix; model license fields are catalog declarations. Resource Ollama's
rights evaluator can produce model-use decisions, but no persisted
Audio-Tools promotion receipt owner currently joins those decisions to the
catalog's commercial-license claim. These facets remain hard blockers;
catalog-seed status strings and arbitrary references cannot satisfy them.
\`make catalog-check\` exercises this fail-closed verification contract; it does
not treat declaration strings as receipts. \`modelcatalog.DefaultForVerified\`
and \`DefaultForEngineVerified\` provide a generic selection seam, and the
opaque \`modelpromotion.Result.VerifiedFor\` callback is bound to the complete
model declaration, including evidence refs and provider/engine identity. The
product model handlers now use the seam with the experiment service's
corpus-quality and interactive-latency resolvers; verification is
request-scoped and database failures fail closed. Memory, platform, and
license remain hard blockers,
so no new model can be promoted. The four code-authorized legacy defaults
remain the only production defaults while those receipt owners are integrated.

## Phase plan and exit gates

### Phase 0 — Baseline and stop the bleeding

Fix the \`sherpa-streaming\` manifest drift. Freeze the provider/operation/model
inventory. Reconcile the requirements ledger. Establish the benchmark corpus and
the model-promotion evidence format. Add coverage floors for \`bootstrap\`,
\`experiment\`, \`eval\`, \`soak\`, \`validation\`; raise \`stt/pipeline\`.

**Exit:** bootstrap and the engine manifest agree; the ledger has no dead refs;
the baseline is reproducible.

### Phase 1 — Provider/model catalog + generated surfaces

Provider catalog, generated capabilities/mapping/lifecycle/UI/docs, drift and
conformance tests. Make the catalog load-bearing in this order: health →
lifecycle → bootstrap → BYOK/settings. Land Whisper \`large-v3-turbo\` as the
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

Kale Nano has a managed CPU candidate runtime and an explicit product API
route. \`auto\` conditionally tries the declared Kale Nano candidate before
qualified Kokoro when Kale is healthy, then falls back to Kokoro; the catalog
default remains Kokoro until Kale passes its held-out quality and release gates.
The live API's Kale route and recent product requests have been exercised.
Rumble now has a bounded vLLM adapter and shared local batch-provider seam,
plus an explicit unary model route on the existing Transcribe request via
\`X-Audio-Model-ID\`. Its provider/model/runtime identity is catalog-derived and
its optional endpoint is \`AUDIO_RUMBLE_VLLM_URL\`; requests fail closed unless
the endpoint serves the declared runtime model. Omitted selection retains the
Whisper default. Rumble stays out of streaming engine selection and default
promotion. There is still no managed resource, endpoint configured on this
host, runtime/quality receipt, or product quality qualification. Keep
Reverb non-commercial unless rights change, and Verba watch-only until it has a
release and license. Audio roles are not declared in AI Gateway while the
gateway has no audio dispatcher; the Audio Tools model registry remains the
load-bearing selection surface. Promote defaults only with receipts.

**Exit:** model selection is a registry + evidence decision, not a code edit.

### Phase 5 — Duplex + UX + hardening

The GPT-Live provider declaration, isolated WebSocket adapter, encrypted BYOK
resolver, and sourced pricing exist. The authenticated transport, source-bound
revocable consent checks, and durable LPBS account-credit reservation and
settlement path are implemented; the SQLite journal stores recovery metadata
and reconciles only the verified owner's operations, while LPBS remains the
balance authority and unknown usage stays pending. Server composition now
requires every admission dependency, a positive session budget, explicit
provider/model eligibility, and the canonical catalog's active eligibility.
The injected predicate is an additional constraint; it cannot promote a
planned provider/model pair. A concurrent-eligibility regression test proves
the planned GPT-Live route stays absent while the injected predicate changes
under \`-race\`. Production bootstrap now wires lazy credential-authority
resolution, Scenario Authenticator account-credit tokens, AI Gateway consent
verification, the LPBS base URL, and an explicit positive session-budget
setting. Admission revalidates the source-bound consent and routes reservation,
settlement, and recovery through LPBS while the local SQLite journal retains
recovery metadata. The provider/model remain planned/candidate, so the route
stays unmounted and no hosted request is made. There is also no browser SDK or
Audio Tools UI consumer for the route yet. The key is deliberately absent from
Audio Tools settings and remains unconfigured in the encrypted Vrooli
credential authority; the owner can provision it with the command documented
in \`docs/domains/settings/byok-and-voice-overrides.md\`. A key or successful
credential lookup alone does not qualify the provider or activate billing.
Consolidate the UI/integration surface and split the \`useVoiceCore\` god hook in
coordination with the shared package owner. Offline fault cases now cover
timeout, decoder truncation, and cancellation; run the real device/browser
matrix.

**Exit:** reliability claims are backed by dated receipts.

## Model policy

Model choice is data plus evidence. The registry declares facts; the selector
applies policy; the qualification harness promotes.

The current Audio Tools summarize default is the application-scoped
\`summarize.audio_tools\` role (\`qwen3:1.7b\`), declared by
\`api/internal/summarize/model_policy.go\` and resolved through the resource
policy. The generic \`summarize.default\` role stays on \`qwen3.5:4b\`, which also
serves \`chat.small\` and \`classify.routing\`; its shared runner avoids loading an
additional model for other consumers. This later role split supersedes the
sixth-tranche snapshot that temporarily moved the generic role to 1.7B. The
Audio Tools choice follows the operator's compact-model direction, with the
measured 14/22 fact-retention result on one 300-word report kept as a limitation
rather than a general quality claim.

| Candidate | License | Verdict |
|---|---|---|
| Whisper small (default), base/tiny (edge) | MIT | Shipped default and constrained-host tiers through the existing whisper.cpp adapter. The canonical clip passes exactly at every size; one host, not a dictation matrix. |
| Whisper large-v3-turbo | MIT | Explicit accuracy tier, on-demand. Use when accuracy outweighs ~1.3 GB more resident memory. |
| Reson8/Rumble-2-Flash | Apache-2.0 | Candidate local batch/multilingual STT tier. The existing unary Transcribe request can explicitly select its catalog-derived route with \`X-Audio-Model-ID\`; \`AUDIO_RUMBLE_VLLM_URL\` is optional and must serve the declared runtime model. Empty selection remains Whisper. No managed resource or configured host endpoint exists; runtime RAM/VRAM, latency, and product quality remain unmeasured. Batch, not native streaming; not promoted. |
| Summarize | Apache-2.0 (Qwen tiers) | Audio Tools resolves its app-scoped \`summarize.audio_tools\` role to Qwen3 1.7B, with Qwen3.5 4B as its quality fallback. Generic consumers keep \`summarize.default\` on Qwen3.5 4B, which shares its loaded runner with \`chat.small\` and \`classify.routing\`. Resolve both roles with \`resource-ollama policy resolve --role <role>\`. |
| Kale TTS 1 nano/tiny | Apache-2.0 model/code; CC BY 4.0 training set | Nano has a checksum-pinned Linux/amd64 CPU runtime and an exercised Audio Tools product route; an explicit conditional-auto policy tries it before Kokoro when available while its provider/model remain planned/candidate. It remains unqualified and unpromoted. Multi-voice quality, lifecycle reliability, and release qualification remain open; Kokoro stays the qualified catalog default. |
| Sopro V2 Turbo voice cloning | Apache-2.0 model/code | 120M CPU/browser-capable candidate, pinned in the provider/model catalogs at upstream snapshot \`c12d0dc7c4a33b3cb6ea3d11ceb70bc60c7c6312\` (952 MB full repository, including alternative weight formats). It requires \`sopro>=2.2.0\`; Audio Tools has no runtime or consented product path, and runtime dependency licenses, peak memory, quality, and latency remain unqualified. Candidate only; never default or selectable until owner gates pass. |
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
| \`TestEveryActiveProviderHasCapability\` | a provider with no health/capability meaning |
| \`TestEveryConstructedProviderHasManifest\` | the sherpa drift class |
| \`TestProviderEngineMappingIsBidirectional\` | engine/provider id confusion |
| \`TestProviderEnginesMatchSTTEngineManifest\` | a provider declaring an engine the runtime does not ship, or a shipped engine no provider owns |
| \`TestModelEngineIDsAreShippedBySTTEngineManifest\` | a model naming a nonexistent stream engine |
| \`TestActiveLocalSTTModelsNameAnEngine\` | weights with no selectable stream path |
| \`TestSummarizeRecordsMirrorResourceOllamaPolicy\` | registry default drifting from \`resources/ollama/model-policy.json\` |
| \`TestWhisperModelIDsResolveInRegistry\` | the Whisper model-truth seam producing an id the registry does not declare |
| \`TestProcessDecoderCloseUnblocksFullFrameBuffer\` | a stream decoder parked on a full frame buffer after teardown |
| \`TestDefaultsByCapabilityIsDeterministic\` + \`TestListModels_PlatformConstrainsDefaultFlag\` | a default that changes between calls or ignores the caller's host envelope |
| \`TestGeneratedCatalogUpToDate\` | hand-edited or stale generated surfaces |
| \`TestEveryOperationHasCLIAndJourney\` | the 7-of-8 server-only gap |
| \`TestGeneratedOperationSpecsUpToDate\` | UI forms drifting from the wire |
| \`TestModelDefaultsAreCommercialAndQualified\` | a non-commercial/experimental default |
| \`TestModelProviderReferencesResolve\` | a model pointing at a retired provider |
| \`TestDocsModelAndProviderMatrixUpToDate\` | documentation drift |
| \`make coverage-floors\` | thin packages (extend the existing policy) |

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
- [\`../../../image-tools/docs/internal/CAPABILITY-MANIFEST.md\`](../../../image-tools/docs/internal/CAPABILITY-MANIFEST.md) — the sibling substrate that motivated this program
`;export{e as default};
