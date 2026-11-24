# Progress — Audio Tools

This log keeps recent dated milestones. It does not certify current readiness.
Read [PROBLEMS.md](PROBLEMS.md) for tracked gaps and the owning Plan Manager
record for live execution status.

| 2026-11-24 | Codex | added (browser duplex session client contract) | Exported `GPTLiveSessionClient` in the shared browser package and added a server `type: ready` acknowledgment only after admission/provider startup succeeds. Capture waits for that acknowledgment; the client bounds socket-open/admission waits, inbound frames, outbound audio buffering, and budget integer precision, and handles transcript/audio/VAD/barge-in and deterministic cleanup. Browser duplex/media tests pass 20/20 and the package build passes. `GOWORK=off go test -race -count=1 ./internal/duplex ./internal/server ./internal/bootstrap` passes. No OpenAI key or provider call was used; no product UI route was added. Fake sockets do not qualify real browsers/devices, GPT-Live usage, consent authority, funded settlement, or provider quality, and the catalog remains gated. |

| 2026-11-24 | Codex | optimized (bounded uniform audio split pipeline) | Uniform splits of 2–32 chunks now share one FFmpeg decode/filter process with seekable per-chunk outputs; explicit-boundary and larger splits retain the established path. Added deterministic error/cancellation/temp-cleanup tests, real WAV sample-identity and exact-duration checks, MP3 duration parity, and a comparative benchmark. On Linux with FFmpeg 6.1.1, the six-chunk benchmark measured WAV 0.094 s vs 0.315 s (3.4× faster) and MP3 0.102 s vs 0.281 s (2.8× faster). Focused split tests, full `go test -race ./internal/audio`, Windows/amd64 and macOS/arm64 test-binary cross-compiles, and `git diff --check` passed. Cross-compilation does not qualify runtime FFmpeg behavior on those operating systems. |

| 2026-11-24 | Codex | measured (live shared-host resource attribution) | At 17:31 UTC, GPU memory was 5,367/16,376 MiB at 3% utilization. The compute list showed two shared Ollama llama-server processes at 2,982 and 608 MiB plus the shared reranker at 1,380 MiB; no Audio Tools process appeared in the GPU list. `resource-ollama models retention` showed Qwen3.5 4B (3,389,983,735 bytes) and Nomic embeddings (274,302,450 bytes) loaded; Audio Tools Qwen3 1.7B was not resident (its last recorded gateway request was 13:21 UTC). Product/lab Audio Tools API RSS was about 22.9/18.8 MiB, Kale 15.4 MiB, and Sherpa 7.6 MiB. No service or model was started, stopped, or unloaded. This point-in-time sample attributes the current GPU footprint to shared services, not an active Audio Tools inference. |

| 2026-11-24 | Codex | verified (bounded D1 CPU and D2 Sherpa host inference) | The lab-only `audio-tools validation run-host-portability` receipt `audio-tools:host-portability:20261124T142848.505902368Z:amd64` records digest-pinned, network-disabled D1 CPU Whisper Small inference passed in 5.323 s with measured 1,221,480,448-byte peak memory, and D2 two-CPU/four-GiB Sherpa Zipformer inference passed in 2.123 s with measured 440,201,216-byte peak memory. The lab `audio-tools.setpoint-read` run `prog_0c9ac9bb-62f8-4be4-8f57-ed85a8ea3131` confirms the board still resolves only 6/18 required rows; host portability remains failed because the D1 Kyutai accelerator route is not serviceable and no matching product-path fallback receipt exists. No D3-D6 or device claim is made. |

| 2026-11-24 | Codex | corrected (TTS model resolution route disclosure) | The Kokoro registry note now describes the actual policy: static qualified default, conditional Kale Nano first when healthy, Kokoro availability fallback. Added a model-catalog regression assertion, regenerated `docs/reference/models.md`, and verified model catalog, model handler, CLI model tests, and `make catalog-check`. This corrects operator-facing model resolution without promoting Kale or changing runtime behavior. |

| 2026-11-24 | Codex | strengthened (declaration and promotion-gate product coverage) | Measured provider catalog at 90.1%, operation catalog at 86.1%, model catalog at 87.0%, STT engine manifest at 81.3%, and model-promotion verification at 77.6%. Added race-enabled floors at 85%, 80%, 80%, 75%, and 75%, respectively. Model-promotion now has untagged, database-free product tests for fail-closed owner gaps, exact declaration binding, and owner-checked quality/latency/memory receipts; the remaining DB-backed end-to-end tests stay in the lab profile. The complete product `TestCoverageFloors` gate passed. |

| 2026-11-24 | Codex | measured (Audio Tools initial UI entry composition) | A production source-map build measured the entry at 646.59 kB / 195.20 kB gzip. React DOM, i18next, React Query, the shared spatial-navigation bridge and shell, and the Overview/API graph make up the required first-route path; feature pages and non-English locale catalogs are already lazy-loaded. Splitting this eager graph into another eager chunk would add a request without reducing bytes or work, and deferring the shell/Overview would delay the first usable route. No UI code change was justified. |

| 2026-11-24 | Codex | hardened (timestamp and host qualification evidence gates) | The long-form timestamp gate now checks maximum start/end boundary error as well as P95 and fitted drift, so an isolated two-second spike fails; focused and full `audio_lab` eval tests pass. This is scorer logic coverage only, not model timestamp qualification. The host-portability inventory producer now emits an explicit `product_path_capture` diagnostic naming the required owner-managed D1 browser cell and Agent Manager/BAS/Test Genie prerequisites; it does not claim the existing virtual-replay soak or host inference is a visible local product path. Host portability remains unqualified pending those receipts. The Python board's D1 CPU, D1 accelerator/fallback, and D2 contract now matches the producer cell shape, with pinned provenance and artifact-bound visible product receipts still required; all 38 board tests pass. `GOWORK=off go test -count=1 ./domains/validation`, `go vet ./domains/validation`, and scoped `git diff --check` pass. The current governed board remains unknown at 6/18 required rows. |

| 2026-11-24 | Codex | fixed (corpus review UI lint and accessible review controls) | Corpus clip review status/actions and caption-track labels now use the generated i18n registry; static and per-clip selectors are correctly split between literal and parameterized selector definitions. Added Arabic/Japanese locale entries, retained captions on both single-clip and grouped playback, and made the stale-review test compare the stable string key. `pnpm type-check`, full `pnpm lint`, and all 143 UI files / 1,361 tests pass; the four affected UI suites also pass 48/48. `git diff --check -- scenarios/audio-tools` is clean. |

| 2026-11-24 | Codex | measured (Sherpa first-text tuning probes) | On the same 11-second JFK fixture, 1/2/4 threads all emitted first text after 1,200 ms captured audio with the same final transcript and failed opening assertion; wall times were 1,223.1/1,218.8/1,216.5 ms. Int8 encoder/joiner reduced model-feed time (584 to 450 ms at two threads) but still emitted first text at 1,200 ms and failed the opening assertion. No runtime setting change is justified. Evidence and limitations are in [PERFORMANCE.md](PERFORMANCE.md) and [PROBLEMS.md](PROBLEMS.md). |

| 2026-11-24 | Codex | fixed (host portability join accepts only proven explicit fallback) | The validator now permits an unavailable D1 Kyutai accelerator only when a concrete reason, D1 host/browser provenance, and a passing Sherpa product-path fallback all share the same artifact identity; inventory-only fallback, missing/failed inference, wrong route/model, and mismatched provenance remain rejected. `GOWORK=off go test ./domains/validation` passes. The live producer still cannot source the required host assignment and product-path receipt, so the current host-portability row remains failed and overall acceptance remains unknown. |

| 2026-11-24 | Codex | fixed (split JSON payload and private CLI audio outputs) | `audio-tools audio split --json` no longer prints every audio chunk as base64 after writing those same chunks to disk; it emits the output directory and per-file path/timing/content-type/byte metadata while retaining the `proto_mutation` primitive. Transform output files are now owner-only (`0600`), including overwrites, and newly created split directories are owner-only (`0700`). CLI core and Audio Tools tests, `go vet` on both changed packages, and scoped `git diff --check` pass; a live split produced four metadata-only chunk rows with the expected file/directory modes. The full Audio Tools CLI suite passes; the broader cli-core suite still fails in unrelated, already-modified `cliutil` tests (`TestCodingBackupDecisionCompleteMatrix` and `TestEmbeddedFreshnessInputsMatchGoListDirectoryRules`). |

| 2026-11-24 | Codex | fixed (capture recovery cleanup and transform wire contract) | A microphone reacquisition that succeeded but could not construct an AudioWorklet/ScriptProcessor graph now releases its owned stream lease before reporting terminal `mic_source_lost`; a regression verifies tracks stop, provider state clears, and the lease registry has no active voice-stream lease. The focused recovery suite passes 44/44, the full shared capture package passes 318 tests with 1 skipped, and its TypeScript build passes. The operation-catalog Connect contract gate now covers all eight transforms, including repeated `MergeSource[]` / `SplitChunk[]` and `AudioMetadata` message ports; `GOWORK=off go test ./internal/catalogconformance ./internal/operationcatalog` and `make catalog-check` pass. Neither result qualifies the separate browser latency or model-quality gates. |

| 2026-11-24 | Codex | fixed (server resumes retained uncommitted PCM) | A handler resume now replays ordered ledger audio into the new recognition pipeline before it starts receiving new chunks. Matching client retries are checked by the ledger and discarded, avoiding double-fed intervals. The interrupted WebSocket regression confirms the recovered audio is complete even when the browser retries the same sequence; committed transcript backfill remains stable. `GOWORK=off go test -race ./handlers/stt -run 'TestBrowserTranscribeStreamWebSocket(ResumesInterruptedInput|BackfillsCommittedSegmentBeforeAudio)' -count=3` and the full `GOWORK=off go test ./handlers/stt` pass. Updated the replay invariant. The fix protects server-retained intervals across transport interruption; it does not close the separate browser p95 or lexical-quality gates. |

| 2026-11-24 | Codex | hardened (late AudioWorklet loads cannot collide with retries) | Shared PCM capture now assigns each AudioWorklet load attempt a unique processor registration name. If `addModule()` resolves after its 1 s fallback timeout, a later gesture's retry can register its own node without colliding with the uncancellable late load. The regression resolves the first load after timeout and verifies unique registrations and retry-node selection. All 316 package tests pass (1 skipped), package TypeScript build passes, and `git diff --check` is clean. This covers capture startup recovery; it does not improve the separate model-bound first-partial p95. |

| 2026-11-24 | Codex | optimized (Sherpa streaming packet conversion reuses bounded per-stream scratch) | `AcceptPCM` now converts PCM16LE into a reusable float buffer under the existing stream mutex, removing the per-packet allocation after capacity is reached while preserving sample values, packet boundaries, and sample cursors. Per-session scratch is capped at one second of samples; larger valid WebSocket packets use temporary storage that is released after synchronous native consumption. Tests cover signed conversion, reuse, zero allocations with adequate scratch, and oversized-buffer release. `GOWORK=off go test -race -count=10 .`, `make -C resources/sherpa-onnx test`, and `make -C resources/sherpa-onnx native-test ROOT=/tmp/sherpa-native-local.06tLL2/sherpa-onnx-v1.13.2-linux-x64-shared` pass. This does not measure product p95 or alter first-text model context. |

| 2026-11-24 | Codex | verified (native Sherpa smoke exercises PCM scratch path) | With the pinned v1.13.2 runtime, current 310 MB Zipformer/punctuation artifacts, paced 100 ms PCM packets, and the 11-second JFK fixture, `TestNativeStreamingModelSmoke` passes its required `your country.` final-suffix assertion. It loads in 1.33 s, first emits text after 1.2 s captured audio / 1.217 s wall time, and spends 571 ms total in model feed calls (RTF 0.99x). Output was `What's your country can do for you as, but you can do for your country.`, so the reference opening remains inaccurate. This is one native adapter fixture, not product/browser p95 or quality qualification; the 30-run browser p95 remains above 1,000 ms. |

| 2026-11-24 | Codex | reconciled (GPT-Live credential and route state) | Live credential status confirms `vrooli/openai-gpt-live` / `api-key` is unconfigured in the available encrypted-file provider; the write-capability doctor reports host-bound TPM2 storage writable. The existing hidden-prompt/stdin provision command and lazy rotation-aware resolver are the supported path, so no duplicate Audio Tools BYOK field or key was created. Updated maturation state/program text to distinguish wired consent/bootstrap/LPBS recovery from the still-gated candidate provider, absent browser consumer, and unavailable live qualification. Duplex, server-composition, and bootstrap packages pass under `GOWORK=off go test -race -count=1`. No OpenAI call was made; a key alone does not activate the route. |

| 2026-11-24 | Codex | fixed (shared capture recovers from a hung worklet load) | The shared browser capture package now bounds `AudioWorklet.addModule()` startup to 1 s, revokes its temporary module URL, clears the failed per-context load so a later attempt can retry, and falls back to ScriptProcessor instead of hanging capture. Audio Tools now delegates startup recovery to this shared owner instead of racing a second adapter-local timeout/fallback. Package regression exercises timeout, fallback, URL cleanup, and same-context retry. The full package suite passes (316 passed, 1 skipped); the Audio Tools UI suite passes (143 files, 1,359 tests); package TypeScript build and `git diff --check` pass. It does not change first-text model/context latency or establish native-device qualification. |

| 2026-11-24 | Codex | verified (consumer speech RPC misses coalesce) | Swarm Manager and Web Console now share per-input in-flight results for normalization and paragraph splitting, preventing concurrent cache misses from multiplying cross-scenario RPCs. Swarm Manager’s deterministic 12-caller regression confirms one RPC per operation under the race detector. `GOWORK=off go test -race -count=10 ./internal/audioports` passes in both consumer APIs. This reduces duplicate work under bursts; it does not change the >1s first-visible latency gate. |

| 2026-11-24 | Codex | verified (meter renders throttled without slowing silence detection) | Dictation Studio still samples microphone levels on every animation frame, but now updates React meter state at most every 50 ms (20 Hz). This removes unnecessary recorder-tree renders while preserving per-frame analysis and the no-audio detector. `DictationRecorder.test.tsx` passes 31/31; targeted ESLint and `git diff --check` pass. The first-visible p95 remains model/audio-context bound and this UI change is not claimed to meet that SLO. The isolated Moonshine Tiny Streaming browser smoke also failed to show an earlier interim; details and limits are recorded in [PROBLEMS.md](PROBLEMS.md). |

| 2026-11-24 | Codex | verified (Audio Tools consumer client refresh publication is race-safe) | Reviewed the shared Swarm Manager Audio Tools adapter: generated clients are now published/read as an atomic immutable `ClientSet`, and refresh/re-resolution is serialized. Its concurrent refresh/read regression and the dependent audio-port adapter suite pass under `GOWORK=off go test -race -count=10 ./integrations/audiotools ./internal/audioports`. This validates the shared-tree change under repeated race instrumentation; it is not a live provider or cross-scenario product receipt. |

| 2026-11-24 | Codex | fixed (consumer speech caches are isolated and bounded) | In both Swarm Manager and Web Console, mutation regressions reproduced corrupted cached paragraphs from provider/caller slice aliasing. Both `RemoteSpeechTextProcessor` implementations now clone provider data on insertion, return independent slices, and bound each normalization/split cache to 256 entries and 1 MiB of key/value payload, pruning expired entries and evicting oldest entries as needed; oversized values bypass caching. Capacity, byte-budget, aliasing, and oversized-input checks pass under `GOWORK=off go test -race -count=10 ./internal/audioports` in both API modules. |

| 2026-11-24 | Codex | measured (live host GPU and Audio Tools RSS attribution) | At 11:54 UTC the GPU had 5,375/16,376 MiB allocated at 8% utilization. Visible allocations were shared Ollama runners (608 and 2,984 MiB) and the shared reranker (1,380 MiB); 403 MiB remained unattributed, with no Audio Tools process in the GPU allocation list. Audio Tools processes measured about 219 MiB RSS for attached Sherpa, 30 MiB for product API, 19 MiB for lab API, 16 MiB for Kale Nano, and 12 MiB for the stopped Whisper activity-edge helper. This is a single host snapshot, not a peak or sustained-load profile; it shows the current GPU load belongs to shared services while Sherpa is the largest resident Audio Tools process observed. |

| 2026-11-24 | Codex | measured (current compact-model/resource attribution) | At 11:23 UTC, `nvidia-smi` reported 5,375/16,376 MiB used (8% GPU utilization). Compute rows attributed 608 MiB and 2,984 MiB to shared Ollama runners plus 1,380 MiB to the shared reranker; 403 MiB remained unattributed. No Audio Tools, Whisper, Sherpa, or Kale process appeared in GPU allocations. Audio Tools model resolution selected Whisper Small (488 MB) and Kokoro 82M (330 MB) without starting either inference model; the latter remains the qualified TTS default while Kale Nano is a smaller but unqualified candidate. API/lab/Sherpa/Kale/Whisper-control process RSS was 25/19/7/16/11 MiB, and idle Sherpa measured 0.00% CPU over five seconds. This is one idle-host snapshot, not inference peak or energy evidence. See [MODEL-FOOTPRINT-AUDIT-2026-11-24.md](MODEL-FOOTPRINT-AUDIT-2026-11-24.md). |

| 2026-11-24 | Codex | verified (stream recovery and session portability) | Resumed STT streams now replay durable committed transcript segments before retained PCM. A handler-level browser WebSocket regression seeds an acknowledged/compacted interval and a commit, then confirms `accepted` precedes the replayed stable segment without new PCM; cursors and durable state remain unchanged. Backfill now reads ordered committed metadata without copying retained PCM from the full ledger snapshot. Session IDs reject Windows-invalid filename forms (alternate streams, reserved device names, forbidden characters, control characters, and trailing dots) consistently on every host. `go test -race -count=1 ./handlers/stt ./internal/stt/session` passed; session tests also cross-compiled for Windows/amd64 and macOS/amd64. Existing browser dedupe coverage passed with the full UI suite (143 files, 1,358 tests). The measured JFK first-visible p95 remains 1,122.1 ms against the 1,000 ms target, so Phase 2 remains active. Plan evidence: note `3ccb5943-1ab8-42d6-9ca3-5ea51e7db99d`. |

| 2026-11-24 | Codex | measured (current compact-model/resource attribution) | At approximately 09:13 UTC, GPU use was 5,375/16,376 MiB at 1%; visible compute rows were shared Ollama runners at 608 and 2,984 MiB and the shared reranker at 1,380 MiB, leaving 403 MiB unattributed. Ollama `/api/ps` showed Qwen3.5 4B and Nomic embeddings, not Audio Tools' Qwen3 1.7B. Audio Tools resolves Whisper Small (488 MB) for batch STT, Sherpa Zipformer (310 MB) for streaming, and Qwen3 1.7B (1.36 GB) for summarize; Kale Nano is healthy and the `auto` TTS route tries it before Kokoro. Whisper was stopped/unavailable; the managed Sherpa resource remains unsupported on Linux even though the attached provider is available. This evidence does not attribute the shared GPU models to Audio Tools or justify smaller STT defaults: the 20M Zipformer emitted later first text, and Kroko's commercial terms remain unresolved. No model or service was started, stopped, or unloaded. The status checks caused the Kale and Sherpa CLI wrappers to install in `~/.vrooli/bin`; no repository files were changed by those checks. |

| 2026-11-23 | Codex | measured (30-attempt paced Sherpa browser cohort) | Headless Chrome fake-microphone runs through Dictation Studio completed 30/30 turns with visible revisions, final drain, sent/processed sequence equality, and captured/committed sample equality. First-partial p50/p95/max was 1,098.3/1,161.2/1,195.9 ms (p95 target 1,000 ms); post-onset audio available p95 was 1,117 ms and excess wall latency p95 was 55.5 ms. Capture-ready p95 was 20.6 ms, update-gap p95 352.1 ms, and stop-tail p95 165.8 ms. Synthetic fixture only; no lexical score or physical-device qualification. Receipt: [browser-paced-sherpa-30-20261123.json](../../coverage/browser-paced-sherpa-30-20261123.json). |
| 2026-11-24 | Codex | measured (native Sherpa model first-text comparison) | One 100 ms-paced run per model on the same 11-second JFK fixture: selected 310 MB Apache Zipformer first text at 1,200 ms captured / 1,217 ms wall; 20M Apache Zipformer at 2,100 / 2,107 ms despite lower load/feed-call cost; 68 MB Kroko at 1,500 / 1,534 ms and passed the checked reference opening. The 68 MB candidate's commercial license is unresolved. This is native-adapter diagnostic evidence only, not product p95, held-out quality, or memory qualification. Receipt: [native-sherpa-model-first-text-20261124.json](../../coverage/native-sherpa-model-first-text-20261124.json). |

| 2026-11-23 | Codex | measured (fresh first-partial context trace) | Rebuilt `@vrooli/audio-capture-browser`, refreshed the locked local file dependency in Audio Tools UI, and rebuilt the served `dist-live` bundle without restarting the scenario. The live UI served the new Dictation Studio asset. One headless Chrome fake-microphone product trace on local Sherpa Zipformer measured 1,096.7 ms speech-to-visible partial, with 1,094.9 ms of post-onset audio already captured and only 1.7 ms excess latency; capture readiness was 13.1 ms and stop-to-final-tail was 87.3 ms. Sent and processed sequence both reached 250, committed/captured samples both reached 200,363, and the capture callback maximum cost ratio was 0.042. Receipt: [browser-first-partial-context-20261123.json](../../coverage/browser-first-partial-context-20261123.json). This isolates the next latency work to early model output behavior; it does not establish p95, lexical accuracy, native microphone quality, or device/browser/OS qualification. |

| 2026-11-23 | Codex | measured (paced Sherpa browser route and stop telemetry) | Three real-time headless Chrome fake-microphone runs through Dictation Studio used the 11-second fixture `/tmp/opencode/jfk.wav` (SHA-256 `59dfb9a4acb36fe2a2affc14bacbee2920ff435cb13cc314a08c13f66ba7860e`) and the selected Sherpa Zipformer route. Capture/session readiness was 72–167 ms; speech-to-rendered interim was 1,094–1,109 ms against the adopted 1,000 ms p95 / 2,000 ms max bands; stop-to-captured wall time was 153–207 ms. All three final outputs retained the fixture’s ending, with 224–226 chunks sent and processed, no page errors, and processed readiness true. This is diagnostic synthetic-media evidence, not a physical-device or qualification receipt; the first-partial target remains out of band. The browser diagnostic also showed Dictation Studio omitted the stop-request marker; the recorder now calls it before provider stop, covered by its component test. At the time, the serving UI bundle predated that source fix; the fresh build and trace are recorded in the 2026-11-23 entry above. |
| 2026-11-23 | Codex | measured (native Zipformer vs Kroko smoke) | Built the current Sherpa server tests against the local native runtime and compared the cached 67 MB Kroko model with the selected 310 MB Zipformer model on the same 11-second JFK fixture. Both native and WebSocket tests preserved the expected final suffix; Kroko loaded in 2.108 s and decoded at 42.44x RTF, while Zipformer loaded in 3.038 s and decoded at 13.20x RTF. Both transcripts had substantial lexical errors, so this one-clip smoke does not qualify either model or justify changing the selected route. The managed server was not restarted; the product browser trace remains the source for interactive latency. |
| 2026-11-23 | Codex | verified (first-interim context telemetry) | The shared browser turn diagnostic now records the exact first-signal sample cursor, captured cursor at first visible interim, audio duration available since speech onset, and excess wall latency after that audio span. This distinguishes accumulated speech context from lag beyond captured audio without exposing transcript/audio or claiming server-only decode time. The browser capture package suite passes (307 passed, 1 skipped), package build and Audio Tools UI type-check pass. A new live product trace is still needed to collect these fields. |
| 2026-11-23 | Codex | verified (GPT-Live secure credential handoff) | The host-bound encrypted-file credential authority reports available and writable; the canonical `vrooli/openai-gpt-live` / `api-key` slot is declared and unconfigured. The Configuration page links to the OpenAI Platform API-key page and provides `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`; interactive entry suppresses echo and writes through the existing authority. Encrypted-store resolver, per-call rotation, and lazy-construction tests pass. The separate Keyring integration is not initialized and is not the credential path. No OpenAI key was minted, entered, or used, and the GPT-Live session route remains gated. |

| 2026-11-24 | Codex | verified (shared capture startup overlap) | Added a deferred-microphone regression proving microphone acquisition begins while persistent journal restore is pending, while capture setup remains fenced until restore resolves. The browser capture package test suite passes (306 passed, 1 skipped) and its TypeScript build passes. This establishes the startup overlap invariant in unit evidence; paced browser microphone/composer latency and losslessness acceptance remain open. |

| 2026-11-24 | Codex | done (product/lab schema ownership and acceptance refresh) | Bootstrap now applies `ProductSchemas` to fresh product databases and adds corpus/experiment schemas only for lab; qualification evidence remains product-owned because model promotion consumes it. Existing lab databases opened in product mode retain their lab tables, and product measure readers do not query absent lab tables. `GOWORK=off go test -race ./internal/bootstrap ./internal/modules ./internal/experiment ./internal/modelpromotion ./internal/server` passes. This narrows runtime initialization but the shared Go build still compiles lab handlers and endpoint metadata; the binary split remains open. The current lab acceptance join `audio-tools:acceptance-join:20261124T001858.065201342Z:a7a500a21ab7d883` validates six source receipts but is incomplete with twelve required rows missing; the D1-D6 matrix has 0/27 qualified cells. No acceptance or device qualification is claimed. |

| 2026-11-24 | Codex | unresolved (managed lab artifact selection) | Added per-variant API/UI output paths and a tagged `audio_lab` build profile. Source-level product and lab builds plus tagged handler tests passed, but the live managed lab process still returned 404 for `CorpusService/ListClips`. Its freshness record omitted `component_build_tags` and included the product-only `lab_composition_product.go`; the active lifecycle process therefore appears to be using pre-tag-aware build logic. A forced rebuild was refused while the product instance was running because the lifecycle still considered the outputs shared. Both managed instances are healthy, but the runtime binary split is not verified. GPT-Live's optional credential descriptor and lazy Vrooli credential-authority resolver are present; the operator can provide the key with `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`. The provider remains gated until production duplex and billing evidence pass. |

| 2026-11-24 | Codex | measured (current shared-host resource attribution) | At 2026-11-24 01:10 UTC, the GPU reported 7,868/16,376 MiB used at 16% utilization. Process rows attributed 606 MiB and 5,078 MiB to two Ollama runners, 1,380 MiB to the reranker, and 396 MiB to an Image Tools process; the 408 MiB difference is unaccounted. Ollama reports Qwen3.5 4B and nomic-embed-text loaded, while Audio Tools' app-scoped Qwen3 1.7B is not resident. Both Audio Tools API processes were about 17 MiB RSS. This snapshot does not attribute shared Ollama or reranker memory to Audio Tools, and no resource was unloaded. Current resolves remain Whisper Small (488 MB), Sherpa Zipformer for explicit English streaming (310 MB), Kokoro as the qualified TTS catalog default with conditional Kale-first auto routing, and Qwen3 1.7B for Audio Tools summarize (1.36 GB). |

| 2026-11-24 | Codex | measured (Kale Nano product auto route) | Fresh product synthesis selected `kale-tts` / `kale-tts-1-nano` and produced 24 kHz mono PCM16 WAV. The first 4.8-second output took 1.64 s; five subsequent distinct warm requests measured 93.7–142.0 ms (median 104.1 ms). This confirms conditional routing and output integrity on the current host, not speech quality, voice naturalness, lifecycle restartability, or promotion eligibility. Kokoro remains the qualified catalog default. |

| 2026-11-23 | Codex | measured (live compact-model route and expired discovery sample) | Fresh resolve calls select Sherpa Zipformer for `sherpa-streaming` (310 MB, Apache-2.0), Qwen3 1.7B for Audio Tools summarize (1.36 GB, Apache-2.0), and report Kokoro as qualified TTS catalog default while product auto prefers healthy Kale Nano. Kale is currently healthy and a new product request selected `kale-tts-1-nano` / `kenzie` with 525 ms provider latency. Direct Connect `ListEngines` took 114.8 ms initially, 4.0/3.7 ms on immediate repeats, and 80.1 ms after a 30+ second idle gap; this local API sample is not browser readiness or model cold-start qualification. Device use was 6,329/16,376 MiB. The separate compute-process rows attributed 5,080 and 606 MiB to shared Ollama runners, 550 MiB to an Image Tools probe, and 1,380 MiB to the shared reranker; their 7,616 MiB sum exceeds device use, so they are retained as independent attributions with the discrepancy unresolved. No Audio Tools model was resident, and no shared process was stopped or unloaded. |

| 2026-11-23 | Codex | measured (first-inference Sherpa resident memory) | A fresh 11-second file-fed product Connect stream completed through selected Sherpa and processed through sample 176,000. Sampling the attached server every 10 ms observed 314,808 KiB maximum RSS (301,148 KiB anonymous); post-run RSS was 314,904 KiB. The idle-before-stream reading had been only 13,376 KiB, explaining the earlier apparent mismatch as lazy model loading. This is a one-host first-inference sample, not long-session growth or device qualification. |

| 2026-11-23 | Codex | measured (compact local TTS route and host attribution) | A product `tts synthesize` request selected local Kale Nano; the first provider latency was 646.6 ms. Five unique warm requests measured 61.8–94.8 ms (median 68.9 ms); output was valid 24 kHz mono WAV with nonzero signal, and Kale RSS was about 90 MiB. GPU usage stayed at 7,465 MiB; current Ollama and reranker allocations accounted for about 7,066 MiB, while Audio Tools Qwen3 1.7B and Whisper were not resident. The shared Qwen3.5 4B is used by generic shared roles, so it was not unloaded or changed. This checks one voice route and a point-in-time host snapshot, not human naturalness or peak long-session resource use. |

| 2026-11-23 | Codex | verified (selected Sherpa product streaming path) | With `sherpa-streaming` selected, `audio-tools stt transcribe-stream` file-fed the 11-second JFK PCM fixture through the product Connect API in 3,200-byte chunks. The output contained 21 partials, two final segments, one completed terminal event, and cumulative durable acknowledgment through sequence 110 / sample 176,000; the final segment also ended at sample 176,000. The ending survived, while the opening and a phrase boundary were misrecognized. This checks one unpaced CLI/API route and interval tail, not microphone-paced UI latency, broad WER, or promotion eligibility. |

| 2026-11-23 | Codex | verified (Kroko stream finalization recheck) | Replayed the available 11.000-second JFK fixture (`/tmp/opencode/jfk.wav`, SHA-256 `59dfb9a4acb36fe2a2affc14bacbee2920ff435cb13cc314a08c13f66ba7860e`) through isolated native Sherpa WebSocket servers in 3,200-byte chunks. Both the default 1.2-second finish tail and zero-tail control produced the complete final phrase, with the last timestamp at 11,000 ms within source duration. The historical truncation did not reproduce; its fixture digest was not recorded, so this is not proven byte-identical to the earlier control. No default change or broad accuracy claim follows. |

| 2026-11-23 | Codex | measured (STT engine inventory liveness deduplication) | `ListEngines` resolved candidates with read-only provider health, then repeated the same health probe while projecting availability. It now reuses each result; a regression requires exactly one serviceability check per local provider and confirms no lifecycle-capable probe runs. Handler tests and race tests pass. After supported product restart, 8 localhost Connect calls reported p50 2.3 ms and max 17.7 ms (first sample included); the CLI command remained about 0.43–0.52 s after its initial 1.69 s invocation. These are small diagnostic samples, not browser click-to-ready qualification or a matched before/after SLO result. |

| 2026-11-23 | Codex | verified (manifest-bound corpus lineage and lab review access) | Added dry-run-first, all-or-none source-lineage backfill with manifest/audio/transcript digest and revision checks, plus grouped review that requires every source variant and records per-clip CAS-bound human review provenance atomically. Applied lineage metadata to all 1,140 lab clips: 360 held-out source groups have clean/10/20 dB variants and 60 dev groups are singletons; manifest and transcript digests match, and zero clips are marked reviewed. The lab profile now includes an isolated UI build and corpus API proxy: `/dictation-studio` and `/health` are healthy on port 22623 and `ListClips` succeeds; the product API corpus route remains 404. Endpoint inventory generation passes with review RPCs explicitly documented as UI-only. Human audition and review, quality acceptance, and device/browser qualification remain open. GPT-Live still has no key; the encrypted credential provider is writable and the proper entry point is `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`, after a key is created in the OpenAI Platform. No key or hosted request was created. |

| 2026-11-23 | Codex | verified (human corpus-reference review workflow) | The lab corpus UI now supports listening through an existing clip, correcting its transcript, and explicitly submitting reviewer/protocol metadata. The API timestamps the review, binds it to the SHA-256 of the saved transcript, rejects stale forms through a conditional update, and clears stale timed alignments when the transcript changes. Focused corpus API tests, five UI tests, UI TypeScript check, committed proto-generation verification, and focused diff check pass. This is a single-clip workflow; no imported rows were marked reviewed, no source-variant propagation was added, and no quality receipt or human review is claimed. The canonical GPT-Live credential store remains available and writable, while `vrooli/openai-gpt-live/api-key` remains unconfigured; the duplex route retains its separate eligibility and spend gates. |

| 2026-11-23 | Codex | measured (shared PCM callback cost) | The streaming resampler now avoids a joined-frame copy and temporary number/float arrays while preserving the previous PCM sample sequence across uneven frames at 16, 22.05, 44.1, 48, and 96 kHz. A bounded Node microbenchmark over 25,000 128-sample 48 kHz callbacks measured 50.57 ms before and 12.24 ms after (4.13x faster). AudioWorklet delivery also forwards its already-transferred frame instead of copying it again. Package suite: 40 files, 289 passed, 1 skipped; TypeScript build passes. This is capture-path CPU/allocation evidence, not browser-visible latency, model quality, or device qualification; those remain unmeasured.

| 2026-11-23 | Codex | verified (live small-model routing and resource attribution) | A live product TTS request with backend `auto` selected `kale-tts` / `kale-tts-1-nano` in 777 ms and returned a valid 3.325 s, mono 24 kHz PCM WAV; the managed CPU service used about 103 MiB RSS. Provider health showed Kale and Sherpa available while Kokoro, Whisper, and Kyutai were unavailable; live streaming config still selected Sherpa Zipformer. This confirms current low-footprint route selection, not voice-quality qualification or fallback behavior. The default STT registry resolves Whisper Small (488 MB) but its provider was unavailable in this snapshot. The Audio Tools summarize role resolves Qwen3 1.7B; it was not resident. Current GPU usage was 7,463/16,376 MiB, with 5,080 MiB on a shared Ollama runner, 606 MiB on another Ollama runner, and 1,380 MiB on the shared reranker; the loaded Qwen3.5 4B is shared across chat and summarize-quality roles. Audio Tools API/lab RSS was about 33/20 MiB and attached Sherpa about 7 MiB. No attribution of shared-model GPU use solely to Audio Tools is justified. |

| 2026-11-23 | Codex | verified (lab evidence route, speaker-policy receipt, and idle model reclamation) | The supported lab Program Runtime invocation reads lab-only evidence without product API changes. Added an explicit degraded-then-fail-closed speaker-policy probe using a temporary unregistered profile, so qualification no longer depends on or reads a user's enrolled voice profiles; its exact lab configuration is restored afterward. The synthetic TTS fixture passed batch Whisper Small and native Sherpa controls, producing `audio-tools:speaker-policy:20261123T124632.459170986Z:0c789111bdca8786`. A local simulated receipt `audio-tools:simulated-qualification:20261123T123846.440709170Z` passed five named rows. The v2 board now reads 29 receipts (12 valid, 17 retained failed, 0 invalid) and resolves 6/18 required rows; acceptance remains unknown. This one-time batch check started Whisper Small, adding about 740 MiB GPU memory; `audio-tools stt engine-impact` reported it safe to stop, so the managed optional resource was stopped afterward. GPU use returned to 7,463/16,376 MiB; Whisper is unavailable until the existing on-demand recovery starts it, while Sherpa remains the selected native streaming engine. No quality, browser latency, or full acceptance claim is made. |

| 2026-11-23 | Codex | verified (unique-prefix timestamp alignment scaling) | `timestampMatches` now trims a shared prefix only while each consumed token is unique in each full transcript, in addition to its safe exact-suffix trimming. The 20,000-word single-middle-correction benchmark now completes in 7.71 ms with 6.98 MB allocated on Linux/amd64; suffix-only alignment would process a 10,001 × 10,001 window. The 20,000-word first-token-correction case completes in 7.33 ms with 5.23 MB. Race tests and randomized parity against canonical minimum-edit alignment pass. This optimizes scorer computation only; repeated/dissimilar inputs retain quadratic worst-case work, and no human-reviewed timed corpus or model qualification is claimed. |

| 2026-11-23 | Codex | verified (recovery, TTS retry, and duplex route hardening) | Journal replay now rejects duplicate/out-of-order PCM records and acknowledgements beyond captured cursors, and rolls back partial in-memory replay on malformed history; 40 focused tests, package build, and the complete shared package suite pass (40 files, 281 passed, 1 skipped). A canceled or deadline-expired lazy TTS backend start no longer poisons the 20-second cooldown, so the next independent call can retry immediately. Duplex route composition now checks canonical GPT-Live catalog eligibility independently of the injected callback, preventing a permissive callback from mounting a planned provider. Full API race suite, API build/vet, CLI race suite, CLI build/vet, `make lab-test`, and `make catalog-check` pass. UI type-check/lint pass; default UI tests read a stale local package snapshot and fail to resolve `audioCueSession`; a temporary source alias ran all 136 files / 1,296 tests successfully. `vrooli package refresh audio-capture-browser --no-restart` deferred Audio Tools setup while its consumer was running. No live route, browser/device qualification, hosted call, or model promotion is claimed. |

| 2026-11-23 | Codex | verified (soak CLI qualification path and pre-Done cancellation) | Added end-to-end CLI owner tests for trusted receipt admission, exact engine/model identity, API preflight, request options/headers, JSON and human outcomes, and protected evidence export. The soak package moved from 17.3% to 89.1% statement coverage; those tests exposed and fixed whitespace around positive integer flags, and the unused clock injection was removed from the soak registration seam. Full CLI race suite and vet pass. A real Connect bidi handler test also verifies client cancellation reaches an in-flight batch provider before `Done` and the handler exits; it passes 25 race-enabled repetitions. These are source-level diagnostics, not browser/product qualification. |

| 2026-11-23 | Codex | verified (delayed readiness fault recovery) | Extended the manifest-driven offline stream fault matrix with a deterministic `delayed_ready` case: the early stream receives provider-unavailable responses and claims no transcript, then a later stream succeeds after the bounded readiness delay. The full matrix passes under `-race`; the new case also passes 10 repeated race-enabled runs. Refreshed [offline fault diagnostics](../../coverage/offline-fault-diagnostics-20261123.json). Six declared faults remain outside this server-only matrix, and product/browser evidence remains empty and non-promotable. |

| 2026-11-23 | Codex | verified (timestamp alignment scaling and browser capture decomposition) | Timestamp matching now consumes exact shared lexical suffixes before allocating traceback, while preserving the canonical path for repeated shared prefixes. This makes a 20,000-word case with one first-token correction complete in 7.28 ms with 5.23 MB allocated instead of computing a 400-million-cell alignment. Existing randomized canonical-path tests and the eval package race suite pass; no human-reviewed timed corpus or model timing qualification is claimed. The shared browser capture hook also moved mic-lease recovery coordination into a focused helper with in-flight-start and reentrant-release tests; package build and full suite pass (40 files, 278 passed, 1 skipped). |

| 2026-11-23 | Codex | measured (model-bound CPU memory owner and live receipts) | Added a dedicated `MODEL_MEMORY` qualification kind, strict schema-v1 receipt validation, canonical model/runtime/artifact identity joins, and a fail-closed memory-facet resolver. The model API now exposes runtime model and artifact identity from the same registry. The bounded host-portability producer records separate owner receipts rather than reusing its aggregate-failed device row. Live lab receipts: Whisper Small peak 1,382,625,280 bytes and Sherpa Zipformer peak 440,156,160 bytes, each under the canonical 2 GiB RAM budget; receipt IDs `e343294d-ca83-4d0b-af96-d9177f84a067` and `c6ed1ce7-1bee-435e-be67-bea726460581`. Both are Linux/amd64 CPU-only measurements and do not qualify GPU, other platforms, quality, latency, or the overall matrix; catalog defaults and legacy exceptions remain unchanged. Full API race suite, API build/vet, CLI suite, focused CLI race tests, `make catalog-check`, and `make -C packages/proto verify-committed-gen` pass. Only the isolated lab API was restarted; product stayed running. |

| 2026-11-23 | Codex | verified (TTS settings concurrency and remaining evidence audit) | TTS configuration reads and read-modify-write updates are now synchronized at the service boundary, preventing concurrent settings updates from racing or silently overwriting another field. A concurrent voice/speed update regression passes under `go test -race`; the full API `GOWORK=off go test -race -count=1 -timeout=300s ./...` and `go vet ./...` also pass. Current source already contains explicit Sherpa/Kokoro TTS routing, conditional Kale-first auto selection, and Kroko’s bounded decoder finish tail; no duplicate adapter work was added. Sherpa native model verification still lacks its local C runtime/fixture. The UI entry chunk remains 643.35 kB (194.33 kB gzip), while docs, diagnostics, and dictation routes are already lazy-loaded. Existing model-specific memory receipts are insufficient for promotion: host portability is aggregate-failed and D2 memory is nested under a different top-level model identity. No model defaults changed; quality, memory, platform, and license promotion gates remain fail-closed. |

| 2026-11-23 | Codex | verified (catalog conformance, provider health declarations, and resource attribution) | Model runtime declarations now enumerate supported capabilities, so the catalog rejects impossible runtime/capability pairs such as TTS on `whisper.cpp`. Provider health declares generic HTTP base-URL fields and readiness/liveness paths in the provider seed; validation rejects unsafe paths and arbitrary fields. Bootstrap reuses one HTTP checker strategy instead of three path-specific profiles. Product/lab route partition gates reject duplicate IDs, duplicate method/path pairs, and cross-surface aliases. Full API `GOWORK=off go test -race -count=1 -timeout=300s ./...`, `go vet ./...`, focused module/server/bootstrap tests, and `make catalog-check` pass. Two independent unsigned Sherpa Linux/amd64 stages verify `1.13.2-vrooli.6` with tree digest `96f3606d500423e7213e4f7494368ae0527d49fba21ac4074e1172720096f2d5`; publication provenance is still missing. A fresh read-only host sample confirms the selected Audio Tools defaults remain the smaller models; visible GPU memory is occupied by shared Ollama/reranker processes, and the selected Audio Tools Qwen 1.7B model is not resident. No shared services were stopped or restarted; GPU request attribution remains incomplete and default promotions still need quality receipts. |

| 2026-11-23 | Codex | verified (timestamp qualification gate and GPT-Live credential readiness) | The long-form timestamp-drift acceptance gate now rejects negative absolute-error P95 values, with a regression for both start and end boundaries; `GOWORK=off go test -race ./internal/eval` passes. This only hardens the scorer and does not qualify any model: the human-reviewed corpus, complete ledgers, and approved thresholds remain outstanding. For GPT-Live, the canonical encrypted credential provider reports available and writable, while `vrooli/openai-gpt-live/api-key` remains unconfigured. The connected Platform key-setup action returned no confirmed choice and no key-creation operation is available here. The product integration is prepared to resolve this canonical credential per session and observe rotations; provision through `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` after creating a key in the OpenAI Platform. No key or hosted request was created. |

| 2026-11-23 | Codex | verified (settlement hardening, UI dependency security, and current race gates) | Full-duplex coordinator now refuses authorization-only admission without a durable spend reservation; a regression proves the provider is not contacted. LPBS settlement explicitly documents the operator-configured URL boundary and rejects uint64 values that overflow signed LPBS amounts before conversion. Race tests pass. UI dependencies now use one `tailwind-merge` version plus Vite 6.4.4 and Vitest/@vitest/coverage-v8 4.1.12 through Scenario Dependency Analyzer; the duplicate tailwind copy removal reduced main JS from 661.24 kB / 200.35 kB gzip to 643.35 kB / 194.31 kB gzip. Vitest migration updates scoped filesystem access and uses only required fake timers. The capture package embeds source content in shipped source maps. API/CLI race suites and vet, UI type-check/lint and all 1,296 tests, production build, `make catalog-check`, and `make portability` pass. `scenario-dependency-analyzer deps approved validate audio-tools` has no findings. Security Health reports 0 blocking errors and no Vite/Vitest advisories, with 284 pre-existing warning/informational findings remaining (security maturity L1). The Vite 500 kB chunk warning remains; human-reviewed quality, browser/device, managed Sherpa publication, and funded duplex evidence remain external gates. |

| 2026-11-23 | Codex | verified (API hygiene finding reduction and generated endpoint repair) | The duplex same-origin guard now parses Origin input with `net/url`, avoiding construction of an HTTP request just to parse a header; malformed, user-info, and scheme-relative origins are rejected. `GOWORK=off go test -race -count=1 ./internal/duplex` passes. Measures now uses the canonical `vrooli.audio_tools.v1.measures` protobuf package, retains a compatibility alias for the old procedure URLs, and declares the shared registry's exact GET/POST routes at source. Protobuf outputs were regenerated by `make -C packages/proto generate SCENARIO=audio-tools`; `make -C packages/proto verify-committed-gen` and `make -C scenarios/audio-tools endpoints` pass. Fresh API Health and Test Genie run `20261123-060554-b07e0080` report 8 findings, down from 10; context-drop and unversioned-route findings are gone. API hygiene remains L1 with hardening debt 8. Three response-body findings remain ownership-analysis false positives backed by transfer comments and caller-close regression tests; five unstructured logging findings are in build/generator commands. Analyzer issue `knw-1795412263436891357` remains open; generator issue `knw-1795413589698612910` was closed as fixed. The analyzer investigator heartbeat remains unavailable because scenario-qa is globally paused by operator instruction. |

| 2026-11-23 | Codex | verified (full API and CLI Go suites) | After the Measures protobuf namespace change, `GOWORK=off go test -count=1 ./...` passed across the full Audio Tools API and CLI modules. The focused race suites for measures handlers, module registry, and server composition passed as well. |

| 2026-11-23 | Codex | verified (Kale managed product route and concurrency headroom) | A live product composition routed synthesis through managed Kale Nano and preserved provider/model/voice identity, WAV integrity, and two cache hits. The first 5-voice attempt reached HTTP 503 because Uvicorn's two-request ceiling was exhausted alongside liveness probes. The managed resource now allows eight simultaneous HTTP requests while retaining one worker, a serialized synthesis lock, and one CPU thread. After restart, the same product-path test passed all five canonical voices twice (`go test -count=2`). The documented 20-request/10-voice runtime-integrity sweep passed: median 77.09 ms, max 112.67 ms after warmup, no failed/empty/malformed results. `go test -race` for bootstrap/TTS and `vrooli resource validate kale-tts` pass. This verifies runtime and route integrity only; the resource/model remains unqualified for speech quality and default promotion. |

| 2026-11-23 | Codex | verified (current Sherpa streaming route smoke) | The healthy product API selected the operator-pinned `sherpa-streaming` engine and streamed the bundled 1.95-second synthetic speech fixture through the attached `sherpa-onnx/streaming-zipformer-en-2023-06-26` model. It emitted partials, durable processed-sample acknowledgements through sample 31,268, and the final five-word transcript. Reported completion was 5.35 seconds during heavy concurrent CPU/GPU host activity; the CLI file replay was unpaced and cannot qualify interactive latency. Current resource processes included the attached Sherpa server at about 138 MiB RSS; independent Ollama, Whisper, image, and other services also occupied host resources. |

| 2026-11-23 | Codex | verified (canonical Measures RPC and compatibility route) | After regenerating protobuf sources and restarting Audio Tools through the supported scenario lifecycle, build identity `sha256:7ef2e42620504a3b050f1c3e820704141dcc47b1f6c4853f6a767a1da8930f18` is healthy. Live POSTs to both `/vrooli.audio_tools.v1.measures.MeasuresService/CountCorpusClips` and the legacy `/audio_tools.v1.measures.MeasuresService/CountCorpusClips` returned HTTP 200. The regenerated CLI shared-measure command also returned the expected zero count for the current seven-day window. |

| 2026-11-23 | Codex | verified (compact voice-cloning candidate declaration) | Added Sopro V2 Turbo to the provider/model seeds at upstream snapshot `c12d0dc7c4a33b3cb6ea3d11ceb70bc60c7c6312` (120M parameters, Apache-2.0 model/code, 4 languages, 952 MB full repository snapshot). Generated provider projections and the models reference now include it. A cross-catalog gate keeps every provider marked `voice-cloning` planned without an adapter and every associated model candidate non-runnable/non-default; provider/model addition remains seed-driven. `make catalog-gen models-doc catalog-check` and race tests for modelcatalog, providercatalog, catalogconformance, and model handler pass. No Sopro artifact was downloaded and no inference/quality claim was made; runtime dependency licenses and consent/data handling remain open. |

| 2026-11-23 | Codex | verified (live compact-route recheck and duplex revocation race) | Against the healthy product API build `sha256:8d7487f298559a12e07f3a39396c6a3f651aeed742d0bc5426e7b9abbcc33755`, a 1.95 s speech fixture streamed Sherpa partials and final text in 486 ms; the 2023 Zipformer matched all five words, with a punctuation difference. Kale Nano produced a valid 24 kHz mono 2.425 s WAV in 720 ms. Qwen3:1.7b preserved the short test facts at 1.14 s warm after 15.77 s cold; it occupied about 2.1 GiB VRAM and its Ollama idle lease was due to expire within about three minutes. The separate Audio Tools API, Sherpa server, and Kale service used 24, 13, and 17 MiB RSS in the same snapshot; other resident Ollama models/reranker were shared consumers. Added a pre-provider consent recheck after LPBS dispatch authorization, with durable hold reconciliation on revocation; focused duplex and race tests pass. These local checks do not qualify naturalness, general STT quality, long-form quality, or browser/device performance. |

| 2026-11-23 | Codex | verified (visible transcript-update gap gate) | Dictation capture now measures intervals between non-empty transcript revisions after React commits each visible revision, recording only monotonic timing and revision count. The native realtime soak gate rejects missing, incomplete, or malformed timing evidence and enforces p95 ≤750 ms and max ≤2,000 ms. Browser diagnostic/recorder tests, UI type check, Go soak/conformance tests, and focused `git diff --check` pass. No browser/product qualification run was available; the gate is instrumentation, not a latency result. |

| 2026-11-23 | Codex | verified (Sherpa candidate provenance refresh) | Rebuilt the current Sherpa source twice as independent unsigned Linux/amd64 stages; both tree digests matched `20255bfe34a82514e6e05c90bd97d00c51737878da6b1ef71eb533686b1a10d5`, and both packaged archives matched SHA-256 `5b100adac89bb27c5e6e51824d90906a0eb4930e3a1a012cfe04148af113830c`. Updated the candidate declaration to `1.13.2-vrooli.5`; `make -C resources/sherpa-onnx check` and `vrooli resource validate sherpa-onnx` pass. `vrooli resource update-check sherpa-onnx --json` remains ineligible because there is no verified published artifact or running managed owner; no signature, publication, install, or server restart was performed. |

| 2026-11-23 | Codex | verified (stop-to-final-tail gate and timestamp coverage integrity) | Shared browser capture now records stop-request to real stream-final latency using a monotonic clock; fallback, failure, and incomplete turns remain unqualified. The native-stream soak requires every turn to have a final tail and enforces p95 ≤1,000 ms and max ≤2,000 ms. The timestamp-drift gate now rejects coverage percentages inconsistent with matched/reference word counts. Fresh Test Genie baseline run `20261123-043950-35f1c2e4` passes unit for the stop-tail changes; after the later timestamp-coverage integrity change, `make catalog-check && make lab-test` and the race suites for TTS, tiered chain, bootstrap, eval, and soak pass. The product soak remains unqualified because managed browser/session authorization is unavailable; no product latency claim follows from source tests. |

| 2026-11-23 | Codex | verified (lab acceptance read and route invocation) | A default Program Runtime invocation used product composition and returned `remote_error` for the lab-only experiment evidence binding. Re-running with `program-runtime --instance lab` succeeded: 20 receipts (8 valid, 12 retained failed, 0 invalid), 5/18 required rows resolved from one simulated-policy cohort, and overall acceptance remains unknown. An explicitly selected successful Sherpa VCTK 20 dB replay reported WER 0.0575 over 3,776 reference words; it is diagnostic replay, not current product acceptance. The setpoint-read contract and observability runbook now identify the required lab invocation and the current 20-row denominator. |

| 2026-11-23 | Codex | verified (compact local model route smoke) | Live lab resolution selected Sherpa Zipformer for streaming STT; one bundled file stream emitted partials, durable acknowledgements through captured sequence 15, and a final identifying `sherpa-onnx/streaming-zipformer-en-2023-06-26` (1,262 ms reported segment latency). This was unpaced CLI file replay, not browser latency or quality acceptance. A one-sentence Audio Tools TTS request routed to local `kale-tts` / `kale-tts-1-nano` in 527 ms; its temporary WAV was removed. Read-only current policy resolves `summarize.audio_tools` to installed Qwen3 1.7B with Qwen3.5 4B fallback; Qwen3 1.7B was not resident and no generation was invoked. `audio-tools diagnostics run --capability stt` failed on the separate batch Whisper chain, which is unavailable; do not use that result to infer Sherpa streaming readiness. No defaults changed and no hosted route was called. |

| 2026-11-23 | Codex | verified (GPT-Live credential provisioning readiness) | Rechecked `vrooli/openai-gpt-live/api-key`: still unconfigured. The encrypted credential store is initialized, host-bound TPM authority is active and unlocked, and unattended access is enabled. The existing `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` command reaches a `Credential value:` prompt without asking for the passphrase; the prompt was canceled before any value was entered. The OpenAI Developers setup-choice tool returned `not_selected`, and no key-creation operation is available in the connected tools. The identity and secure provisioning path are ready for a user-created Platform key; no secret or hosted request was created. |

| 2026-11-23 | Codex | verified (current GPU attribution) | `nvidia-smi` reported 15,242/16,376 MiB in use. Its process list attributed 13,452 MiB to three Ollama `llama-server` workers and 1,380 MiB to the separate reranker; Ollama `/api/ps` showed `nomic-embed-text`, `qwen3.5:4b`, and `gemma4:12b`, with no Audio Tools `qwen3:1.7b` resident. No Audio Tools process appeared in the GPU compute list. This is a point-in-time shared-host snapshot; no model or shared resource was unloaded. |

| 2026-11-23 | Codex | verified (late capture-graph cancellation) | Shared capture now fences async capture-factory results in both normal start and microphone reacquisition: a graph resolving after stop or a newer generation is stopped instead of being installed on an ended turn. New start/reacquire regressions cover the late result. The full package suite passes (36 files, 263 passed, 1 skipped) and the TypeScript build passes. This is package-level lifecycle evidence; browser-paced latency and product acceptance remain open. |

| 2026-11-23 | Codex | blocked (authenticated soak admission unavailable) | Test Genie run `20261123-082346-ccbab5b7` failed before the browser provider started with `SOAK_CONFIGURATION_MISSING`: its phase requires exactly one trusted Agent Manager identity token, BAS browser grant receipt, and Test Genie lease receipt. The persisted phase result is artifact `artifact_81eab706700791dd09da6e09a30f2902`; its finding report is `artifact_8cbd6f7782fe5afa7e0678c25ed3c8d6`. A fresh `agent-manager run identity --json` returned `identity_refresh_unavailable`; this session has no Agent Manager/BAS tool to create the required run context. No browser action or conformance artifact was produced. The fresh setpoint read `prog_105a861d-de3c-4cd8-be78-ec4e1ca98f04` remains `acceptance=unknown` (5/18 required rows resolved, 13 unavailable, 2 excluded). Business Health and requirements validation pass with a stale-evidence warning; Measures Health passes with no findings. |

| 2026-11-23 | Codex | verified (Portal voice regressions and model-footprint policy) | Portal's shared voice adapter now preserves segment sample cursors and provider/model identity, including provider tier; focused tests also cover rejection diagnostics, stream status/VAD/wakeword mapping, unsafe acknowledgement values, pending-permission disposal, and omission of an absent provider ID. Portal voice tests (10) and `pnpm exec tsc --noEmit` pass; focused `portalVoice.ts` branch coverage is 86.2% (not aggregate project coverage). Model policy remains unchanged: Whisper Small (488 MB) and Sherpa Zipformer (310 MB) are already comparatively small commercial-use defaults; Whisper Tiny (78 MB) lacks held-out qualification. The VCTK held-out corpus exists in the lab, but its references are still marked for human audit, so candidate comparisons cannot yet support promotion. No model default was changed. GPT-Live remains gated on provisioning `vrooli/openai-gpt-live/api-key`; this session has no Platform key-creation operation, and interactive credential-store passphrase entry is required. No key or hosted request was created. |

| 2026-11-23 | Codex | verified (current capture and performance suites) | The current shared `@vrooli/audio-capture-browser` tree passes its package suite (36 files, 261 passed, 1 skipped) and TypeScript build. Test Genie run `20261123-022714-4d279e03` passes unit (L2) and performance (L3); 60 advisory low-coverage locations remain and scenario completeness is still `functional_incomplete`. This does not establish paced microphone latency, native streaming, or product acceptance: the managed browser trace still lacks its required Agent Manager identity, browser grant, and Test Genie demand lease. GPT-Live credential setup remains bound to `vrooli/openai-gpt-live/api-key`; secure provisioning requires the interactive credential-store passphrase, and no key or API request was created. |

| 2026-11-23 | Codex | verified (duplex consent redirect protection) | Gateway consent verification now refuses redirects instead of allowing an HTTP client to replay the opaque receipt body to another endpoint. Static and service-discovery constructors use a copied client so shared caller settings are not mutated. Fake-server regressions confirm the redirect target is never contacted for either constructor; `GOWORK=off go test -race -count=1 ./internal/duplex` passes. GPT-Live remains unmounted and no hosted request was made. |

| 2026-11-23 | Codex | verified (bounded retained-audio expiry and timing gate) | Persistent browser journals now expire 24 hours after the first captured audio. Per-session activity metadata and a key index let restore inspect retention and remove only the owned rows without loading all journals' PCM. Expired or unageable rows and append records are deleted before restore; the provider clears the old recovery identity and starts fresh. A generic timestamp-drift gate binds reports to engine/model identity, reviewed provenance, duration, coverage, P95 boundaries, drift rate, and source count; its fixtures are synthetic logic tests only, and staged long-form references remain unreviewed. Resource attribution is refreshed in `MODEL-FOOTPRINT-AUDIT-2026-11-20.md`: Whisper Small/Kokoro/Qwen3 1.7B remain the declared batch/TTS/summarize defaults; current GPU residency is dominated by shared Ollama and reranker processes, while Audio Tools' summarize model was not resident. Test Genie run `20261123-021510-eaa295e4` passes unit and performance; 60 low-coverage warnings remain advisory. The managed paced browser run is still unqualified because this shell lacks the required Agent Manager identity, browser grant, and Test Genie demand lease. |

| 2026-11-22 | Codex | verified (compact summary escalation, runtime routes, and credential readiness) | The Connect summary path keeps the declared `summarize.audio_tools` compact role for short text and selects the declared `summarize.audio_tools.quality` role for inputs of 300+ words; focused handler/chain tests pass. Live TTS synthesis selected managed Kale Nano (5.10 s cold, 51 ms warm); the general model resolver intentionally continues to report qualified Kokoro as the catalog default while Kale stays candidate. Ollama shows `qwen3:1.7b` not loaded, `qwen3.5:4b` resident with 1,054 recent requests (864 attributed to AI Gateway), and `gemma4:12b` resident despite its nonresident role policy and last attributed request on Nov 21; no shared model was unloaded. GPT-Live credential authority reports encrypted-file provider available/writable and field unconfigured; the setup command and Platform key link are ready, but this session has no Platform key-creation operation and GPT-Live remains gated. No secret or hosted request was created. |

| 2026-11-22 | Codex | verified (adopted outcome inventory and setpoint bands) | Synced `docs/internal/TESTING.md` to all 20 current PRD targets and added explicit measurement routes for the five maturity outcomes; updated setpoint descriptions to show the adopted latency, recognition, settlement, and D1–D6 simulation targets while retaining unknown outcomes. Regression coverage cross-checks PRD, evidence inventory, and setpoint row mappings. Python setpoint tests (37), CLI validation tests, measure-health structure validation, and Test Genie `programs`, `skill-set`, and `business` run `20261122-225533-571e3027` pass. Governed lab read `prog_58567e34-3d58-4393-a5e4-26bd23125eb6` inspects 16 owner receipts (6 valid, 10 failed), resolves 0/18 required rows, and keeps acceptance unknown. A short paced browser trace remains unrun: the owner refused before execution without a verified Agent Manager identity, browser grant, and active Test Genie demand lease; no native or product qualification is claimed. |

| 2026-11-22 | Codex | verified (GPT-Live key setup UX) | Added a direct OpenAI Platform API-key link beside the existing encrypted-authority provisioning command in Audio Tools Configuration, with safe external-link attributes and localized copy. The UI still never accepts or displays the secret; key entry remains through `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`. Focused ConfigurationPage tests (8), UI type-check, string-generation check, and file lint pass. The key remains unconfigured and GPT-Live remains gated; no key or hosted request was created. |

| 2026-11-22 | Codex | verified (scenario phase suite after maturation updates) | Test Genie run `20261122-224135-1ea9447f` passed the `programs`, `skill-set`, and `business` phases. Overall scenario completeness remains `functional_incomplete` at 57/100, with 2/29 requirements and 2/20 operational targets passing and stale docs/proto/structure/unit evidence. This is phase validation, not product acceptance or qualification. |

| 2026-11-22 | Codex | verified (timestamp drift report persistence) | Added the provider-neutral `TimestampDrift` score, its unavailable skip reason, and timed-reference provenance to the persisted per-clip protobuf report. Mapping and protojson round-trip regression coverage confirms measured scores survive persistence while unreviewed references retain an explicit skip reason and no numeric score. Focused eval, experiment, and report tests pass. No model measurement was produced: the current VCTK corpus has no timed references, and the three staged AMI long-form packets contain forced-aligned draft boundaries with no independent human review. Whisper small/turbo and other engines therefore remain unmeasured for long-form timestamp drift until an attributable reviewer completes and imports the required reviewed recordings. This is report-path work only, not product acceptance or qualification. |

| 2026-11-22 | Codex | verified (metadata-only TTS CLI output) | Added opt-in `--json-metadata-only` for synthesis commands that also use `--out`; output-file audio is preserved while JSON retains provider/model/voice/timing metadata without embedding the base64 audio payload. The flag fails before a request when `--out` is absent, and default JSON keeps its existing full-proto shape. Focused and full CLI suites, CLI vet, catalog-check, and `git diff --check` pass. A live product call synthesized through Kale Nano and returned provider/model/voice/latency metadata with no `audio` field. This verifies the route and output behavior, not TTS quality qualification. The secure GPT-Live slot is available but unconfigured; this session has no Platform key-creation action, so the existing provisioning command remains the key-entry path. |

| 2026-11-22 | Codex | verified (current evidence-board and device inventory) | A fresh governed lab `audio-tools.setpoint-read` run (`prog_89df4acb-9668-48ac-a56e-c0ed9057b2da`) completed and retained the explicit `unknown` acceptance result: 5/18 required outcome rows resolve, 13 are unavailable, and 10 failed historical receipts remain visible beside 6 valid receipts. The current inventory-only device matrix lists 27 required-when-claimed cells; all are not-run, unverified, and unqualified. No physical device or product-path claim follows from the inventory. |

| 2026-11-22 | Codex | verified (Sherpa release archive repeatability) | Added offline release-stage regressions that build the same fixture tree twice with deliberately different source mtimes and require byte-identical archive, release manifest, and publication metadata; an unsigned stage also must leave the checkout's authoritative platform declaration and rendered support claims byte-for-byte unchanged. `python3 scripts/test_validate_sherpa_native_stage.py` passes all 10 tests, and `make -C resources/sherpa-onnx check` passes. This validates packaging only: all Sherpa OS targets remain unsupported until the release owner supplies signed publication and native target evidence. |

| 2026-11-22 | Codex | verified (VCTK 10 dB held-out quality cohort) | Completed local Sherpa streaming run `dd3cd80d-2e8c-474c-b656-2f384412c98f` for all 360 manifest-selected `noisy_10db` clips, explicitly labeled `vctk-heldout-noisy-10db`. WER was 11.76% overall (349 substitutions, 45 insertions, 50 deletions / 3,776 reference words). Joining per-clip scores through manifest clip/source IDs gives UK 14.37%, South African 14.17%, and US 6.79%; each is below the 20% noisy-cohort ceiling. This is quality-only evidence with no latency claim and does not address browser/device, recovery, long-duration, privacy, or promotion gates. The matching 20 dB result is recorded below. |

| 2026-11-22 | Codex | verified (VCTK 20 dB held-out quality cohort) | Completed local Sherpa streaming run `eed797ca-9a61-400e-98c1-9b275c8209ff` for all 360 manifest-selected `noisy_20db` clips, explicitly labeled `vctk-heldout-noisy-20db`. WER was 5.75% overall (165 substitutions, 34 insertions, 18 deletions / 3,776 reference words). Joining per-clip scores through manifest clip/source IDs gives UK 7.35%, South African 6.93%, and US 3.00%; each is below the 20% noisy-cohort ceiling. This is quality-only evidence with no latency claim and does not address browser/device, recovery, long-duration, privacy, or promotion gates. |

| 2026-11-22 | Codex | verified (operation-format parity and provider engine lookup) | The operation catalog and CLI now expose WebM, Opus, and PCM S16LE alongside the five existing formats; API codec mapping covers PCM in both directions, UI download names retain the PCM extension, and ffmpeg-backed checks transcode WebM/Opus/PCM plus decode PCM back to WAV. Experiment provider routing now resolves engine IDs through the provider catalog instead of a second engine list, with declared-engine and unknown-engine coverage. Focused tests pass for operationcatalog, audio handlers, CLI audio, bootstrap, and the AudioOps panel. A full API race suite, CLI suite, UI suite (135 files / 1,280 tests), catalog-check, lab-test, and Windows/amd64 plus macOS/arm64 portability builds pass. Initial UI-suite failures were traced to a stale pnpm snapshot of the shared audio-capture-browser package; rebuilding it and refreshing the scenario's file dependency restored module resolution. The CLI primitive-evidence artifact was refreshed by its declared verifier after the manifest updates. |

| 2026-11-22 | Codex | done (batch STT target added to D1-D6 inventory) | The declared D1-D6 matrix omitted Whisper Small, Audio Tools' batch STT default, while listing Sherpa streaming and conditional Kale TTS. Added Whisper as a required-when-claimed target and updated the focused inventory test and command description. The inventory now reports 27 target-profile cells, all not run/unverified/unqualified. Existing D1 Whisper and D2 Sherpa bounded host inference receipts remain smoke evidence only; this change adds no product-path or device qualification claim. |

| 2026-11-22 | Codex | verified (governed lab evidence route rechecked) | Re-ran the bounded evidence program against the explicit `program-runtime --instance lab` profile. The program completed with no binding errors, read 14 owner receipts, and resolved 5/18 required acceptance rows; 8 failed receipts remain visible, 13 required rows remain unavailable, and overall acceptance is still unknown. The default product Program Runtime profile cannot read the lab-only route and correctly reports a partial result. No simulated receipt was treated as native qualification. |

| 2026-11-22 | Codex | verified (engine-scoped model resolution and candidate audits) | Closed the STT model/engine visibility gap with an `engine_defaults` declaration projected through the model API/UI and `ResolveModel.engine_id` / CLI `--engine`: `model resolve --capability stt --engine sherpa-streaming --language en` resolves the operator-selected 310 MB Zipformer, while unscoped resolution remains multilingual Whisper Small. UI model details now display engine-scoped defaults, and the resolver offers an optional STT engine input. Engine-scoped defaults have unique declaration validation, an explicit evidence/legacy gate, commercial filtering, and fail closed without cross-engine substitution. Generated model docs and protobuf clients. Verified modelcatalog/handler/CLI tests and vet, modelcatalog/handler/CLI race tests, 32 model UI tests, UI type-check/lint, `make catalog-check`, `make verify-committed-gen`, and Windows/macOS portability builds. A primary-source voice-cloning audit found Sopro V2 Turbo promising at 120M/Apache-2.0 by its authors but without a verified full upstream artifact pin, Audio Tools runtime/quality/resource evidence, or consent controls; candidate remains non-runnable. A timing-source audit found Buckeye is noncommercial and FluencyBank boundaries are mostly forced-aligned; neither replaces the three human-reviewed AMI recordings. Exact owner review action is in `LONG-FORM-TIMING-CORPUS.md`. |
| 2026-11-22 | Codex | verified (cross-platform builds and credential setup) | Windows/amd64 and macOS/arm64 portability builds passed. Rechecked `vrooli credentials doctor --check-writes`: the host-bound encrypted-file authority is available and writable; `vrooli/openai-gpt-live/api-key` remains unconfigured. The connected OpenAI tooling has no Platform key-creation operation, so the existing secure flow remains `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` (secret read through stdin/secure prompt, never argv). Audio Tools resolves the canonical authority on each session open and sees rotations without restart. No key was created and no hosted call was made; GPT-Live remains unmounted until the qualification gates are satisfied. |
| 2026-11-22 | Codex | verified (live compact routes and terminal duplex cleanup) | Fresh product CLI calls selected Sherpa Zipformer streaming, qwen3:1.7b for a local summary, and Kale Nano for a 24 kHz mono WAV synthesis; the 1.7B model reported 14.1 s for the short cold request and retained all three input facts. A host snapshot showed 9,862/16,376 MiB GPU memory in use; Ollama separately reported qwen3:1.7b at 2.24 GB resident alongside qwen3.5:4b and embeddings, so this is shared-host attribution rather than Audio Tools-only usage. Fixed a duplex handler leak: the event writer now closes the WebSocket on terminal/event-context completion, unblocking the request goroutine's concurrent read. A regression keeps the client open after a synthetic upstream failure and proves it receives provider-error + closed events and the HTTP handler exits; the regression passed 20 race-enabled repetitions. Kroko finalization was rechecked with the WebSocket adapter: default-tail output retained the final suffix on both available JFK clips while zero-tail control truncated; Kroko remains on-demand and unqualified. Model runtime IDs and local-artifact ownership now share one typed declaration table; focused modelcatalog tests pass. |
| 2026-11-22 | Codex | verified (TTS format defaults and manifest-derived recovery allowlist) | Corrected the stored TTS `default_response_format` contract: the persisted preference now reaches unary/streaming synthesis and cache lookup when requests omit a format; explicit request formats still take precedence. Renamed the internal setting away from the overloaded `Backend` field and migrate legacy stored audio-format values while treating old runtime labels as MP3. STT backend recovery now receives its allowlist projected from the STT engine manifest at bootstrap, avoiding a duplicate resource inventory while preserving fail-closed rejection; the registry was not imported into the recovery package because that creates a dependency cycle. Verified with full API tests and vet, focused race tests for bootstrap/STT/TTS, catalog-check, and scoped `git diff --check`. The OpenAI GPT-Live credential slot remains unconfigured; the encrypted host-bound authority is available and writable, and secure provisioning remains ready at `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`. No authenticated Platform key-creation action is connected, and no key or hosted request was created. |
| 2026-11-22 | Codex | verified (duplex admission bootstrap wiring) | Production composition now constructs the LPBS account-credit client only when `AUDIO_LPBS_BASE_URL` and a positive `AUDIO_GPT_LIVE_MAX_SESSION_BUDGET_MICROS` are configured. The service-principal credential is read lazily from the canonical credential authority, while Scenario Authenticator and AI Gateway are resolved at use time through scenario discovery. The composition policy remains fail-closed because GPT-Live is not yet qualified, so the route is still absent and no hosted request or credential lookup was performed. Focused bootstrap, duplex, and server tests pass. The separate OpenAI key slot remains unconfigured; `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` is the secure operator path. |
| 2026-11-22 | Codex | verified (declaration-selected local TTS baseline and broad validation) | Removed a resource-name assumption from local TTS baseline composition: bootstrap now selects the active local provider whose declared default role resolves to an eligible catalog model. The targeted selector test, full API suite, full API race suite, API vet, catalog-check, Windows/amd64 and macOS/arm64 portability builds, and UI type-check/lint pass. `audio-tools validation device-matrix --json` still reports 0/18 qualified cells. A read-only evidence audit found the acquired AMI ES2002a timing labels are forced-aligned and explicitly not human-reviewed; the required three distinct reviewed long recordings remain unavailable. Kale Nano's short serviceability outputs are not retained as audio and have no naturalness review, so Kokoro remains the default. |
| 2026-11-22 | Codex | verified (LPBS endpoint precedence) | Fixed startup URL precedence for LPBS: an explicit `AUDIO_LPBS_BASE_URL` is now preserved, and service discovery is used only when the operator has not configured an endpoint. This prevents discovered addresses from silently replacing an intentional integration endpoint used by the LPBS client. Focused bootstrap test passes normally and under `-race`. |
| 2026-11-22 | Codex | source (Kale Nano human-review packet) | The current Audio Tools product API reports Kale Nano available while Kokoro is stopped. Without starting Kokoro or changing routing, generated four short canonical-voice WAV samples through the supported product CLI and saved them with a prompt, model/provider identity, route latency (261–315 ms), limitations, and SHA-256 checksums under `~/.local/share/vrooli/audio-tools/qualification/kale-nano-review-20261122/`. The files are valid 24 kHz mono PCM and checksums verify. This gives a reviewer actual audio for assessment; it is not human-reviewed quality evidence, a paired comparison, or a promotion receipt. |
| 2026-11-22 | Codex | verified (catalog-declared duplex consent) | Removed OpenAI provider-ID branches from duplex consent admission and active-session monitoring. Full-duplex provider declarations now carry a validated consent source and must require active revalidation; admission forwards that exact source after verifying a human consumer-token principal, explicit consent, and receipt. Coordinator fails closed before hold/provider contact if a declared active revalidator is missing. A non-OpenAI fixture exercises authorization and revocation. Focused provider-catalog/duplex/server tests and race tests, full API tests/vet, catalog-check, UI type-check/lint and 135-file/1,279-test suite pass. The live GPT-Live route remains closed because model qualification is incomplete. |

| 2026-11-22 | Codex | verified (legacy STT service façade cleanup) | Reduced `internal/stt/pipeline.Service` to its active Whisper batch transcription, availability, and recovery responsibilities. Removed uncalled stream/wake-word/speaker settings persistence wrappers, duplicated in-memory settings state, and the unused bootstrap `Deps.Voice` escape hatch. Composition now supplies only capability, URL, HTTP client, and audio engine; store-backed repositories remain the settings owners. Focused pipeline, sttchain, bootstrap, and segmenter suites passed both normally and under `-race`; full API `go test ./...` and `go vet ./...` passed. The host-encrypted GPT-Live key slot remains ready through `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`; no key was created and GPT-Live remains unmounted pending qualification and billing evidence. |
| 2026-11-22 | Codex | verified (live model footprint and host attribution) | Current selectors choose Whisper Small for batch STT, Sherpa Zipformer for streaming STT, Qwen3 1.7B for Audio Tools summaries, and Kale Nano first when the product TTS auto route is healthy; the catalog TTS default remains Kokoro 82M pending Kale quality qualification. The current Ollama `/api/ps` list contains Nomic embeddings, Qwen3.5 4B, and Gemma4 12B, but not Audio Tools' Qwen3 1.7B; `nvidia-smi` showed 15,238/16,376 MiB GPU use from Ollama workers and the separate reranker (1,380 MiB). This is a point-in-time host snapshot with shared Ollama ownership, not a scenario-level utilization trace. Audio Tools' declared weight defaults total about 2.2 GB; Kale Nano quality, voice cloning, and device qualification remain open. |

| 2026-11-22 | Codex | verified (credential/status path and provider-routing startup reconciliation) | Confirmed no connected OpenAI Platform key-creation tool is available. The canonical `vrooli/openai-gpt-live` identity uses the writable host-bound encrypted credential authority; `vrooli credentials status --identity vrooli/openai-gpt-live` reports it unconfigured, and `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` reads the value from a secure prompt/stdin. The resolver reads on every session open, supports rotation without restart, redacts failures, and does not qualify or mount GPT-Live by itself. Fixed startup application of saved provider-routing config under environment-level feature-flag upper bounds, with the same clamp for settings updates. TTS status now separates adapter registration, routing state, and local credential presence; present credentials are reported unverified without provider calls or secret exposure. Focused race tests pass across bootstrap, settings, TTS, and AI chains. `git diff --check` is clean apart from an existing CRLF warning on a WAV fixture. |
| 2026-11-22 | Codex | verified (native TTS service, Kroko tail control, live lab/product route fence) | Sherpa's existing local TTS adapter and Audio Tools cache/cancellation route tests pass; a temporary server loaded the cached 114 MB Kokoro int8 model and served `/v1/audio/speech` as a 24 kHz WAV (1.87 s, 90,010 bytes), then was stopped. `/healthz/tts` on the existing attached service still reports `tts_model_loaded:false`, so product-service synthesis is not live on that process. Native Kroko WebSocket tests pass with the default 1.2 s finish tail and fail the expected final suffix when the tail is explicitly disabled, confirming deterministic truncation coverage; Kroko remains experimental. The lab instance was restarted through its scoped lifecycle and its API-only evidence route returned 200 while product returned 404; a fresh supervisor status still reports the lab `UI_PORT` placeholder error, so full live supervisor-profile health remains open. |
| 2026-11-22 | Codex | verified (duplex consent revalidation) | GPT-Live remains unmounted and exact provider/model admission remains ineligible. Active session consent is now rechecked immediately after provider startup and every 30 seconds; revocation or consent-verifier failure closes the session through the existing settlement/uncertain-retention path. `go test ./internal/duplex` and the focused revocation test under `-race` pass. No API key or hosted request was used. |
| 2026-11-22 | Codex | verified (coverage gate and current source suites) | The fresh API suite exposed `internal/ai/chains` at 77.8% against its 80% floor because the new environment-upper-bound clamp lacked package-local coverage. Added focused clamp tests for disabled tiers and TTL preservation; package coverage is now 100% and the coverage-floor gate passes. Fresh `go test ./...` and `go vet ./...` pass for API; the full CLI suite and vet pass; UI type-check, lint, and 135-file/1,279-test suite pass; `make catalog-check`, `make lab-test`, and Windows/amd64 + macOS/arm64 `make portability` pass. |
| 2026-11-22 | Codex | verified (profile-aware supervisor health live) | Ran the guarded root install after supervisory verification passed, then `vrooli runtime supervisor install --user`. The supervisor adopted build identity `9db19196767db25a692f668a488e8d7ba0f000505a43cd0b9035ab97385995ca` (PID 1452770); `vrooli host safeguard runtime_supervisor --json` reports the installed process image current, unit active, enabled, and accepted. Fleet count remains 91 with zero unverified. The lab instance now reports healthy with no `UI_PORT` error; lab health and evidence are reachable, while product health stays HTTP 200 and the lab-only evidence route remains HTTP 404 on product. Full `go test ./internal/runtimesupervisor` passes. |

**Source correction (2026-11-21):** Hugging Face API metadata for the pinned
Kale Nano revision `b6ce1887268c3667994fc3b0b2c82887c069915c` reports public,
non-gated Apache-2.0 status, last modified 2026-11-19. Earlier dated entries
below reflect the model-card state observed at the time. Tiny remains draft.
This publication/license update does not qualify Nano's product quality or
lifecycle and does not promote it from candidate.

## Current milestone

| 2026-11-22 | Codex | verified (declaration-gated Rumble batch route) | Reused the existing bounded BatchBackend seam for the opt-in vLLM Rumble endpoint and made model records name the exact protocol adapter they use. Bootstrap now ignores undeclared vLLM protocols, preventing future models from being routed through Rumble's payload format by runtime-name coincidence. The adapter remains explicitly selected, candidate-only, and does not start/load weights. Focused modelcatalog/bootstrap/sttchain tests pass. Runtime footprint, live endpoint, latency, and quality remain unverified. |
| 2026-11-22 | Codex | verified (current Sherpa Linux bundle and unsigned stage) | Rebuilt the server bundle twice from the local Sherpa 1.13.2 runtime; both trees match digest `049564ce498d1bc512434a6019de887945732e9fabcf7f092b6f898ee3085093`, with all shared libraries resolved by the bundled RPATH. The former v3 pin (`4985ca8c…`) no longer matched current server source; the only changed staged file was the server executable. Advanced the candidate to `1.13.2-vrooli.4`, updated its tree pin, and `make release-stage TARGET=linux-amd64 … SIGN=0` now passes offline stage validation. The stage is unsigned and unpublished; the owner must provide publication URL and upstream provenance before signing or managed acquisition. |
| 2026-11-22 | Codex | verified (Connect streaming idle-timeout) | Connect STT streams now enforce a 30 s default idle interval, reset by incoming audio; timeout cancels the segmenter, persists terminal reason `input_idle_timeout`, and returns without joining the synchronous receive pump so transport teardown can release it. Quiet-timeout persistence and continuing audio past the interval pass under `go test -race ./handlers/stt -run 'TestTranscribeStream_(QuietInputTimesOutAndPersistsFailure|AudioActivityExtendsIdleWindow|DuplicateChunkIsIgnored)' -count=1`; the full `TestTranscribeStream_` subset also passes. |
| 2026-11-22 | Codex | verified (compact defaults and D1–D6 inventory) | Current CLI model resolution selects Whisper Small for batch English STT, Sherpa Zipformer is the selected native streaming engine, TTS's catalog default is Kokoro with Kale Nano reserved for its health-checked conditional product route, and Audio Tools summarize resolves to `qwen3:1.7b`. The live health snapshot reports Sherpa and Kale available; a short live product synthesis returned `provider_id=kale-tts`, `model_id=kale-tts-1-nano`, and a 681 ms service latency. This is a serviceability/route smoke, not a naturalness claim. `audio-tools validation device-matrix --json` enumerates 18 cells across D1–D6 for Sherpa streaming and conditional Kale Nano; zero are qualified, all are explicitly unrun/unverified. The CLI now builds and its module suite passed after primitive-evidence regeneration. This inventories required evidence; it is not a device/browser qualification. |
| 2026-11-22 | Codex | verified (routing fix, long stream, TTS cancellation, split guard, Kroko native WebSocket finalization) | The streaming CLI captured its API base before global flags, unlike cli-core's rebased unary calls; a fake-transport regression plus `go test -race ./domains/stt -count=1` pass. Source-built CLI streams now hit lab port 15373: the short synthetic fixture completed on Sherpa Zipformer, and the 21-minute AMI clip completed in 1m58.9s with 115 segments, all 1,242 sequences processed, and terminal `completed` (about 10.7x RTF, not quality/timestamp evidence). The product log has no newer stream request. A prior lab-intended replay strongly correlates with a product API request at 07:05:34; any possible ledger/spool remains untouched (details in PROBLEMS.md). Sherpa-backed TTS route/cache/model/voice wiring already existed; TTS now preserves caller cancellation/deadlines without invalidating health or caching partial response. `go test -race ./internal/tts ./internal/bootstrap` passes. Sherpa `/healthz/tts` is 503 (`tts_model_loaded:false`), so live TTS synthesis remains unverified. Split UI requires positive duration or explicit boundaries; 9 focused Vitest cases pass. The cached Kroko model and punctuation artifacts then passed native decoder and WebSocket smoke tests on `quality_speech.wav` (3.29 s total); transcript-bearing test output was kept out of the log view. This is one clean fixture and does not qualify broader quality, timing, license, or promotion; Kroko stays on-demand. |
| 2026-11-22 | Codex | verified (credential path and post-refactor gates) | Rechecked GPT-Live setup after the operator authorized secure provisioning: `vrooli credentials doctor --check-writes` reports the encrypted-file authority available and writable, while `vrooli credentials status --identity vrooli/openai-gpt-live --field api-key --json` reports the canonical identity unconfigured. The provision command takes the value through its terminal prompt/stdin; I canceled before entering a value. The lazy resolver reads on every session open and observes rotations. The exact key-source instruction is available from `vrooli credentials doctor --check-writes` (OpenAI Platform API keys); no OpenAI key-creation tool is connected here, no secret was entered, and no hosted call was made. The session route remains gated. Complexity cleanup split the 1,660-line soak driver into five files under 500 lines, moved report production into seven files under 240 lines, and moved overlap/VAD per-session state into dedicated files. Changed production functions in report, soak, STT strategy, and duplex now have no gocyclo scores above 15. Full API race tests/vet, `make lint`, `make type`, CLI tests/vet, catalog checks, Windows/macOS portability, and the focused Card UI test (11 passed) pass. The light tidiness scan returned API 500, so no fresh scan result is claimed. |
| 2026-11-22 | Codex | verified (live compact-model routes) | Against the running healthy product instance, `audio-tools model resolve` selected Whisper Small for batch STT and Qwen3 1.7B for the Audio Tools summarize capability; `audio-tools stt engines` showed Sherpa Zipformer active and selected for streaming. A current default-voice synthesis returned a valid 90,044-byte WAV through `local/kale-tts` in 1,848 ms. The general TTS registry still identifies Kokoro as its qualified default; live auto routing used Kale. This route check does not establish human quality or naturalness. |
| 2026-11-22 | Codex | verified (Kale Nano product route and declaration-backed speed handling) | A live product smoke synthesized valid 24 kHz mono WAV for all four configured canonical Kale voices; metadata-only evidence is in `coverage/kale-nano-product-voice-smoke-20261122.json`. Kale remains a conditional candidate because this was not a human quality review. TTS runtime declarations now require `speed_mode` (`native` or `postprocess`); synthesis behavior no longer branches on the Kale model ID. Full API unit/race suites, CLI tests/build/vet, UI type-check/lint and 1,283 tests, catalog drift checks, and Windows/macOS portability builds pass. Test Genie tidiness was skipped because host CPU admission had no capacity; debt is not cleared. |
| 2026-11-22 | Codex | verified (scenario gates clean; complexity debt remains open) | Refactored capability-checker composition, timed-reference validation, streaming STT normalization, experiment execution, and CLI STT handling; removed dead wrappers/simulation helpers; split model-catalog, audio-operation, and CLI STT tests; added `make type` for API/CLI compile checks and UI TypeScript. `make lint`, `make type`, full API `go vet ./...`, full API `go test -race ./...`, full CLI tests/vet, `make catalog-check`, and Windows/macOS `make portability` pass. Latest light scan reports 0 lint/type findings and both scenario gates pass; it flags two long files, the soak module and experiment runner (both remain above 500 lines by direct `wc`). `ops_edge_test.go` and CLI STT files have since been split into files below 500 lines. `OverlapAgree.Run` is down to gocyclo 34; `overlap_agree.go` is 476 lines with cohesive helpers split into adjacent files. `normalizeChunks` is gocyclo 5; CLI stream/config/speaker primary handlers are 10/6/7, with extracted helpers at or below 14. Sherpa's offline release validator checks POSIX executable mode for staged and archived entrypoints; release publication still needs owner provenance and a matching artifact. |
| 2026-11-22 | Codex | source (bounded duplex HTTP transport; production route remains closed) | Added an isolated WebSocket module requiring verified human identity and the exact LPBS bearer, applying coordinator consent/budget admission before provider contact, and bounding control/audio/message sizes. Regression coverage includes unauthenticated denial, admission denial without provider contact, oversized open envelopes, opaque/forwarded/wrong-scheme origin rejection, and continued reads after the initial 15-second envelope deadline. Focused duplex race tests and server tests pass. The module is intentionally unmounted: production gateway consent verification, LPBS reserve/settle plus recovery composition, and provider/model eligibility are still incomplete. |
| 2026-11-22 | Codex | verified (timestamp drift skip reason distinguishes valid zero-match output) | Eval reporting now labels a human-reviewed timed reference with zero lexical overlap as `timestamp_no_lexical_matches`, instead of incorrectly treating the timing evidence as malformed. Regression coverage passes with `GOWORK=off go test ./internal/eval -count=1`; `git diff --check` is clean. No reviewed timing references were created or used, so long-form model qualification remains open. The timing corpus guide also reconciles its previous empty-lab note with the current 1,140-clip VCTK inventory, which still has no timed references. |
| 2026-11-22 | Codex | verified (canonical Sherpa model ID joined live in lab) | After a first build safely aborted on changing shared React Component Library sources, the supported `audio-tools@lab` restart succeeded at identity `sha256:70358526431f8e167325563e24addc8f28e2f1afd7a4da9008ea5320cbbf4a23`. A one-clip local Sherpa experiment reported runtime model `sherpa-onnx/streaming-zipformer-en-2023-06-26`; its persisted quality receipt resolved to canonical `sherpa-zipformer-en-2023-06-26` / capability `stt` and no longer carries the undeclared-model reason. The receipt remains failed as expected (3 words, one clip/speaker, no 20/10 dB cohort). This validates the model-catalog-to-evidence join only, not corpus quality. |
| 2026-11-22 | Codex | verified (lab VCTK corpus built; quality remains diagnostic) | Created `~/.local/share/vrooli/audio-tools/qualification/voice-pilot-en-v1-manifest.json` as a metadata-only manifest for University of Edinburgh CSTR VCTK 0.92 (CC BY 4.0). The lab CorpusService now has 1,140 scripted records: 360 held-out clean prompts, paired 20/10 dB mixes for each, and 60 dev prompts from distinct speakers. Held-out: 6 speakers, 3,782 base-script words, 1,227 UK / 1,269 US / 1,286 South African English words; hashes are unique and measured augmentation SNRs match their labels. Fixed corpus CLI's `--scripted` boolean handling (`BoolFlag` instead of valued `Flag`) and replaced all rows through the supported lab corpus API; focused CLI test passes and a live re-import reports `CLIP_SOURCE_SCRIPTED`. Diagnostic experiment `b66a477d-062e-4028-8bbb-8704842cf222` on 60 clean prompts measured Sherpa Zipformer WER 6.18% (842 evaluated words). Its persisted quality receipt correctly failed due to 60 clips/848 corpus words, absent noisy cohorts, and the pre-fix build's undeclared runtime model ID. Added the Sherpa runtime ID to the model seed and made quality evidence resolve canonical model IDs by declared engine/runtime-ID; focused model-catalog/report tests pass, and generated model docs were refreshed. The source references still need human audit; no timestamp references exist. The first lifecycle restart attempt safely aborted before stopping the lab when shared React Component Library sources changed; the successful retry and live resolver smoke are recorded in the newer milestone above. |
| 2026-11-22 | Codex | blocked (selected BAS browser execution requires an authenticated account) | Workflow Health static validation passed for the registered five-cycle deterministic microphone-reuse case (5 cases, 3 flows; registry current). Routed execution acquired the isolation lease, but the BAS provider rejected workflow admission with `workflow execution requires an authenticated account` before opening a browser. BAS entitlement reports `anonymous` / `inactive`; no active owner session was available. The run recorded zero primary writes and zero isolation-root writes, and produced no browser evidence. Historical BAS session profiles are durability fixtures, so none were reused as credentials. Do not count this attempt as a case pass; rerun through Workflow Health after an owner-authorized BAS session is available. |
| 2026-11-22 | Codex | verified (current lab corpus inventory) | Used the supported `audio-tools --instance lab corpus list --limit 5 --json` route after confirming the earlier default CLI request had reached the product instance. The lab CorpusService responds successfully with an empty list (`{}`); no current corpus clips or timed references are available to score. The historical 180-clip count and synthetic 61-minute arithmetic fixture do not provide current human-reviewed timestamp evidence. No clips were imported and no labels were invented. |
| 2026-11-22 | Codex | verified (Kroko finalization path on the pinned native adapter) | With the pinned local Sherpa runtime and cached Kroko model, converted the bundled 24 kHz JFK WAV to the adapter's required 16 kHz PCM and ran both native decoder and WebSocket smoke tests. Each returned `Ask not what your country can do for you . Ask what you can do for your country.` with the exact `your country.` final suffix; final timestamps stayed within source audio. Warm inference was 56.5 ms for 3.846 s of audio (68.06x RTF); model load was 1.04 s. Added a WebSocket contract regression whose final-only sentinel suffix must survive the end frame. This establishes finalization behavior for one cached clean clip, not broader accuracy, license/quality qualification, or default promotion; Kroko remains on-demand. `make test-native` passed with the cached native runtime, punctuation model, and Kroko artifacts. |
| 2026-11-22 | Codex | verified (small-model product routes and lab-profile health) | Live CLI resolved batch STT to Whisper Small, streaming STT to the pinned Sherpa Zipformer, and the Audio Tools summarize role to Qwen3 1.7B. A local Sherpa stream completed the bundled 5.049-second clip; the short summary request served `qwen3:1.7b` locally in 21.839 s. The existing Kale Nano smoke remains the TTS route proof. Restarted only `audio-tools@lab` through `go run ./cmd/vrooli scenario restart ... --instance lab`; profile-aware health filtering removed the stale UI-port error, and the API-only lab instance now reports healthy at build identity `sha256:2ebc8b3ecc8db74445bf8444f291456f67d0238eff91c53d2c54c8982c9f9549`. Lab `/health` is ready; the product API remains healthy and returns 404 for `/experiment/list-evidence`, preserving the boundary. This validates routing and lifecycle only, not broad model quality. |
| 2026-11-22 | Codex | verified (local TTS and secure credential setup path) | A fresh CLI synthesis produced a 92,444-byte WAV through `kale-tts` model `kale-tts-1-nano` in 674 ms, confirming the current local small-model route works end to end. The resolver remains declaration-based; Sherpa's `/v1/audio/speech` is already implemented, while its direct health endpoint reports encoder and streaming models ready but no TTS model loaded. The OpenAI credential authority reports `vrooli/openai-gpt-live` / `api-key` unconfigured with the encrypted-file provider available. Secure provisioning remains available through the hidden-input `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` command. This session has no authenticated OpenAI key-creation operation, and no secret or hosted API request was made; credential presence alone does not mount the separately gated GPT-Live duplex route. |
| 2026-11-22 | Codex | verified (startup route split, iframe payload deferred, lab worker and spend safety tightened) | The UI now uses a 4.65 KB / 2.00 KB gzip loader that applies the storage shim before app imports; the app chunk is 719.10 KB / 221.26 KB gzip, with the 32.56 KB iframe bridge fetched only in embedded mode. Compared with the 854.84 KB / 254.68 KB gzip entry, standalone startup JS is down about 12% gzip; Vite still flags the app chunk above 500 KB. Shared iframe shortcut intent is now a small entry; package build and 106 tests pass. Full UI suite passes (135 files, 1,283 tests), plus type-check, focused lint, and route/boundary checks. Full API `go test ./...` and `go vet ./...`, scoped API race tests, and 34 host-matrix contract tests pass. The supported managed restart is healthy at build identity `sha256:f62c9a67d5708e48d9e2fba88db2cff1506d757f8475d2ba62b6dd1805f8378a`; API `/health` is ready and UI returns the new loader. Lab experiment workers now start only in lab variant; ambiguous non-durable duplex usage retains its budget hold; TTS cache speed keys distinguish close values and cache reads refresh LRU recency. The human timestamp corpus, native device rows, complete promotion facets, managed Sherpa release, and consented voice-cloning route remain open. |
| 2026-11-22 | Codex | verified (route-scoped audio client and encrypted GPT-Live credential path) | Audio Tools now creates its browser RPC client only for diagnostics, dictation studio, and stream configuration, keeping STT/TTS protobuf validation out of the initial page load. The built entry chunk fell from 854.84 KB / 254.68 KB gzip to 756.01 KB / 232.95 KB gzip; the validation descriptor is deferred to its own lazy chunk. TypeScript, targeted lint, and 21 affected UI tests pass. The supported managed restart is healthy with matching expected/served identity `sha256:df62317fcd2f5458399ee082680b20d978cd3d56dbeb01050e891e1b7eafed6f`; API `/health` and UI return HTTP 200. The initial bundle still exceeds Vite's 500 KB warning. GPT-Live remains optional and unconfigured; the encrypted credential authority and rotation-aware lookup are ready, but no Platform key-creation action is connected. Provision securely with `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` (hidden terminal prompt or stdin); no key or API call was created. Provisioning alone does not mount the gated duplex route. |
| 2026-11-21 | Codex | verified (current-source Audio Tools runtime healthy after lifecycle restart) | The latest supported lifecycle status reports a successful restart and matching expected/served build identity `sha256:a092a3de145adf57ca5631b9cc56b61c2ed8c297561235b88d648b07eab9e90f`. A fresh API `/health` read at 23:44 UTC is healthy/ready with the same identity; the UI port returns its app HTML. The restart occurred before this check and was not initiated here. This confirms the previous package consumer compile error no longer blocks this running build; repeated-mic/browser and other product qualification remain open. |
| 2026-11-22 | Codex | done (current-source model routes verified; managed Sherpa release still blocked) | The supported Audio Tools restart rebuilt the current source and returned healthy runtime identity `sha256:27fe0ee3ac83a27283df4bc5d5ece71afc27076d3a238458d16f7308891c209d`. Focused race tests for `internal/bootstrap`, `internal/ai/ttschain`, and `internal/tts`, plus fresh Sherpa server tests, pass. A live product TTS request selected local Kale Nano and synthesized 150,044 bytes of valid 24 kHz mono WAV in 927 ms. The current Sherpa Zipformer attachment was selected by the operator stream-engine pin; it transcribed that 3.125-second clip in 1,558 ms, with `route` recognized as `root`, so this is route smoke evidence only. A structured live summary confirmed `model_id=qwen3:1.7b`, 21,256 ms and six preserved facts; the follow-up Ollama snapshot reported 2.24 GB resident for 1.7B (above its 1.36 GB on-disk declaration). Model resolution returns Whisper Small for batch STT. The Sherpa resource status remains unsupported on Linux; preserved unsigned stages reproduce `e7f…a1a513`/`4494…5fdb0`, not the uncommitted `c8127175…` pin, and no signing or publication occurred. See [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-22 | Codex | done (privacy boundary, portability identity, and speaker-policy restore reliability tightened) | Metadata-only registry snapshots now omit replay audio and committed transcript payloads while exposing a committed-count scalar; the trusted internal soak driver uses an explicitly named conformance snapshot for transcript-based grading. D1 portability receipts now require the exact Kyutai accelerator model alongside D1 CPU Whisper and D2 Sherpa identities. Speaker-policy CLI tests cover success, early failure, leaked negative-control text, payload-free receipts, and restore failure; they exposed and fixed nil batch-response panic and evidence recorded before restoring the original policy. Race tests for `internal/stt/session` and `internal/soak`, CLI validation tests, and Test Genie unit run `20261122-010826-c409c4d7` pass. W1/W2 validation passes without stale evidence after supported requirements sync; 58 low-coverage advisories and broad product qualification gaps remain. GPT-Live's encrypted-file credential slot is available and unconfigured; secure owner provisioning instructions are already present, while no Platform key-creation action is exposed. |
| 2026-11-21 | Codex | done (shared browser package compile boundary recovered) | The previously reported `VoiceTransport.buildStreamUrl` compile mismatch is absent from current package and Audio Tools UI sources. `pnpm --dir packages/audio-capture-browser build` passes; its complete Vitest suite passes 244 tests with 1 skipped across 34 files; `pnpm --dir scenarios/audio-tools/ui type-check` passes, including generated strings. No service restart was performed, so this proves current-source compile/test integration, not that the already-running Audio Tools UI/API processes loaded the current build. The supported promotion path remains `vrooli scenario restart audio-tools` when a fresh live runtime proof is needed. |
| 2026-11-21 | Codex | done (cloning route assessed; integration gate identified) | Reviewed Qwen3-TTS 12Hz-0.6B-Base and OpenVoice V2 + MeloTTS from their primary sources. Qwen’s code/checkpoint declare Apache-2.0; the Qwen route is still not commercially or quality-qualified end to end, and has no Base CPU/footprint evidence. OpenVoice/MeloTTS declare MIT, but their pinned bundle, rights and product-resource costs remain unresolved. Audio Tools has planned declarations only, without a cloning runtime, provider adapter or consented reference-audio lifecycle. Next: review artifact and speaker-rights inventory, then build a pinned isolated candidate only after the authorized acquisition plan and consented held-out cohort exist. See the AT2 `CLONING-ROUTES.md` research artifact. |
| 2026-11-21 | Codex | done (stale microphone-probe race fixed; five-cycle qualification remains open) | A late unhealthy capability result from a stopped turn could dispose and overwrite a newer session. The hook regression reproduced two `dispose` calls while turn 2 was preparing; the per-start callback now rejects results whose captured generation is stale. `pnpm vitest run src/useVoiceCore.startup.test.ts src/longSession.test.ts` passes 34/34 in `packages/audio-capture-browser`. No real microphone five-cycle browser run was performed; the broader repeated-session finding remains open. See [PROBLEMS.md](PROBLEMS.md). |
| 2026-11-21 | Codex | done (duplex accounting substrate measured) | Added aggregate-only `CountDuplexAccountingOperations` over the metadata correlation journal with exact half-open creation-time bounds; wired through its API, proto, CLI, and command declaration. Focused API/duplex and CLI measure tests pass, proto generation freshness passes, and measures-health reports zero findings / maximum maturity. `make test-packages` completed red in six modules within the already dirty shared tree (`cli-core`, `credential-authority-go`, `maturity-go`, `proto` missing `flow-verifier/v1/errors`, `repo-contract-go`, `resources`); without a clean pre-change baseline, these failures are not attributed to this change. Details in [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-21 | Codex | done (low-resource routes rechecked; lab profile status corrected in source) | Sherpa Zipformer remains selected by the STT engine pin; its attached `/healthz` is `ok`, but managed Sherpa remains unsupported on Linux. Qwen3 1.7B remains the summarize policy/catalog default and is not resident. Fresh GPU attribution found 11,372 MiB in shared Ollama, reranker, and Ollaya processes; no Audio Tools GPU process. Product TTS auto selected managed Kale Nano (148,844-byte WAV output to `/dev/null`, 296 ms warm) while the qualified catalog default remains Kokoro. Fixed status/detail to apply the addressed instance profile; focused orchestrator and CLI tests pass. The live lab status still has a stale degraded health snapshot until its next lifecycle refresh. Tiny Voca's AT2 fit review found no declaration-only route; managed runtime and paired quality/resource measurement remain open. See [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-21 | Codex | done (GPT-Live secure credential path verified and docs aligned) | `vrooli credentials status --identity vrooli/openai-gpt-live --field api-key` reports the encrypted-file provider available and the identity unconfigured. Production composition prepares `OpenAILiveProvider` with the lazy canonical credential-authority resolver; resolver/server tests pass. The Configuration page directs owners to the canonical CLI and does not offer a duplicate GPT-Live BYOK write. Updated the settings architecture guide, which still described an older path. No Platform key-creation action was connected, so no key or API request was created. Provisioning does not mount the gated duplex route. See [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-21 | Codex | done (stream reliability, CLI JSON, low-footprint runtime and release evidence) | Fixed a buffered STT cancellation contract gap (Error then terminal Done, cancellation wins over queued audio); the full sttchain package and targeted race test pass. Fixed late Dictation Studio callbacks from disposed providers; 18 UI tests pass. STT engines, engine-impact, and formats `--json` now emit canonical proto JSON; source-built outputs parse against the live API. Updated Sherpa release metadata to `.2`/`c812…6e8c`; offline Linux stage and resource checks pass, but it remains unsigned/unpublished. Fresh product Kale Nano TTS emitted valid 24 kHz mono WAV in 4,188 ms. GPU attribution shows other shared services account for the current 11.4 GiB use; Qwen3 1.7B was not resident. Added simulated LPBS coverage for the $10/month Audio-only plan and reusable second app. [Timing-corpus research](LONG-FORM-TIMING-CORPUS.md) confirms AMI requires human boundary review; no clips downloaded. |
| 2026-11-21 | Codex | done (current-source product route and resource checks) | Managed Audio Tools restart is healthy and loaded the TTS changes. A fresh local synthesis returned a valid 127,244-byte WAV from `kale-tts` / `kale-tts-1-nano` / `kenzie` in 2,292 ms; `model resolve` labels Kokoro as the qualified catalog default and now explains the Kale-first conditional product route. Sherpa Zipformer still selects for streaming and the refreshed attached product server returns `The quick brown, fox jumps.`; repeated segment latency varied (195 ms before the API restart, 4,921 ms on the first stream after it), so no SLO claim is made. Two identical Qwen3 1.7B summary calls preserved all facts: 8,968 ms cold then 1,047 ms warm. The model used ~2,390 MiB GPU while loaded and was later unloaded by shared capacity policy; the latest host snapshot showed 9B `chat.default`, Nomic embeddings, reranker and Image Tools consuming GPU, with no Audio Tools inference model resident. |
| 2026-11-21 | Codex | done (current Qwen product route and credential path confirmed) | Managed `vrooli scenario restart ai-gateway` rebuilt and returned healthy; a fresh Audio Tools summary selected local Ollama `qwen3:1.7b` in 17,342 ms, retaining the key timing/fact-count results while omitting the fallback rationale. The secure OpenAI credential authority is available and GPT-Live's identity is unconfigured; the canonical secure-prompt command and lazy rotation-aware resolver are ready. No key-create action is connected, no key was created, and provisioning alone does not mount the gated duplex route. See [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-21 | Codex | done (Sherpa Zipformer punctuation fix deployed to the attached product route) | The first current-source rebuild exposed a stale attached server: it emitted `brown,fox` although the normalizer and unit test expected `brown, fox`. Rebuilt `resources/sherpa-onnx/server` into a local Linux/amd64 bundle against the pinned v1.13.2 runtime, restarted only the documented attach-only Sherpa process, and verified `/healthz` plus Audio Tools streaming STT. The same fixture now returns `The quick brown, fox jumps.` with Sherpa selected and 194.9 ms segment latency. The runtime stays attach-only and is not a signed/managed artifact. |
| 2026-11-21 | Codex | done (TTS model routing clarified; Sherpa and Kale adapters consolidated) | Explicit `sherpa-onnx` TTS is now a non-default, catalog-declared fast model with a backend-scoped cache namespace and canonical Kokoro voice mapping. Product tests cover requested MP3 bytes/content type, model identity, event-cache replay, and cancellation propagation. `AUDIO_TTS_BACKEND=auto` still tries serviceable Kale Nano first and falls back to Kokoro, while `model resolve` remains the qualified catalog default and now says so. Race tests for TTS/bootstrap/model/provider catalogs, CLI model tests, and `make catalog-check` pass. No live Sherpa TTS model receipt exists yet. |
| 2026-11-21 | Codex | diagnostic (pre-refresh state; superseded by the entries above) | At 22:08 UTC the attached Sherpa binary still emitted `brown,fox`, and TTS model resolution did not explain the Kale-first auto route. That snapshot had 8,706/16,376 MiB GPU use, mostly outside Audio Tools. Both mismatches are now addressed or retested above. |
| 2026-11-21 | Codex | done (Qwen3 summarize default restored; live route verified) | Restored `qwen3:1.7b` as `summarize.default` with `qwen3.5:4b` first fallback. The product summary initially failed at AI Gateway because its old binary used the legacy rights-context projection; current policy hash and model digest already matched the plan. Managed `vrooli scenario restart ai-gateway` rebuilt and served the current rights evaluator, and the same Audio Tools request then succeeded through local Ollama `qwen3:1.7b` in 20,985 ms. Policy/catalog focused tests and generated catalog checks pass. GPT-Live credential setup remains available at `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`; the encrypted-file provider is available and the key is unconfigured. No Platform key-create action is exposed, and production GPT-Live remains gated on route composition, identity/consent, spend recovery, and usage evidence. |
| 2026-11-21 | Codex | done (Gateway-owned GPT-Live consent; credential path ready) | AI Gateway now verifies owner/source-bound active consent receipts for `audio-tools-openai-gpt-live`; Audio Tools duplex admission checks online before reserve/dispatch. Focused consent and duplex tests pass. The OpenAI key field remains unconfigured in the available encrypted-file credential provider; no Platform key creation action is connected, and the existing owner provisioning command plus rotation-capable lazy resolver are ready. No key/API request or product route was used. |
| 2026-11-21 | Codex | done (streaming STT machine output corrected) | `stt transcribe-stream --json` now writes one compact proto JSON event per line in stream order; human output remains readable and stream Error events still return a non-zero result. Focused STT CLI tests and `make catalog-check` pass. The prior live reproducer showed `--json` being ignored and emitting plain event text; no server restart was needed. |
| 2026-11-21 | Codex | done (duplex accounting recovery substrate; still fail-closed) | Added a metadata-only SQLite journal of LPBS reservation correlation/phase and owner-scoped recovery using a domain-separated hash of the verified consumer subject. Recovery calls LPBS's authenticated owner endpoint; a two-user test proves user A cannot cause user B's operation to be queried. LPBS remains sole billing authority and unknown usage stays pending. Current full API race suite, full API vet, `make catalog-check`, and Windows/amd64 + macOS/arm64 portability builds pass. No GPT-Live route is mounted; source-bound revocable consent and full product auth remain. |
| 2026-11-21 | Codex | done (fresh small-model routing and host/resource attribution) | Live CLI selects Sherpa Zipformer for streaming and Qwen3 1.7B for summarize. Product TTS selected Kale Nano and returned a 2.975 s WAV in 3,009 ms first request / 822 ms repeat. Live Sherpa stream transcribed the dictation fixture in 1,994 ms but the attached server still emits `brown,fox`; the source punctuation fix is not deployed. GPU snapshot: 11,297 MiB used at 2%; the largest allocations are shared Gemma4 12B/Ollama and the reranker, not the Audio Tools summarize selection; Audio Tools API/lab use ~20/19 MiB RSS. New D1 host observation records ALC1220 endpoints but inaccessible capture nodes for the current account, no capture, and D1-D6 unqualified. Full details and non-claims are in `history/MATURATION-STATE-HISTORY-2026-11-21-to-22.md`, the footprint audit, and `coverage/host-portability-observation-20261121.json`. |
| 2026-11-21 | Codex | done (unit gate refreshed; secure credential handoff prepared) | Test Genie unit run `20261121-210232-67dc9817` passed L2/Complete in 92.4 s with 58 advisory low-coverage locations; `GOWORK=off go test ./...` passes in Test Genie CLI. Rechecked OpenAI credential authority: encrypted-file provider is available, GPT-Live field remains unconfigured. Source audit and focused resolver tests confirm lazy canonical credential lookup, per-open rotation, Bearer use, and redacted errors. The connected Platform key-create path did not return a confirmed choice and this session exposes no key-create action. Opened the existing hidden-input `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` prompt for owner entry. No secret or API request was supplied. GPT-Live remains unmounted until authenticated product access and privacy/cost policy are ready. |
| 2026-11-21 | Codex | done (batch STT extensibility + cancellation/evaluation guards; full API/CLI race gates pass) | Added a protocol-neutral `BatchBackend`/`BatchProvider` seam and bounded vLLM Rumble-2-Flash audio adapter, then moved Whisper's existing local batch provider through the shared wrapper so provider/model/tier/latency/language mapping has one owner. Rumble stays unwired pending a configured endpoint, managed resource, runtime footprint, and quality receipt. Timed-reference evaluation now rejects clip/transcript mismatch and out-of-bounds intervals; no human-reviewed corpus or model qualification is claimed. Segmenter cancellation preserves best-effort ordered Error/Done without blocking teardown and suppresses late partials. Sherpa punctuation spacing now repairs Latin word boundaries without changing numeric, CJK, abbreviation, or domain formatting; package/native tests pass, but the attached server binary was not restarted. Measures CLI RPCs honor terminal interrupt cancellation; cli-core caller deadlines remain unsupported. Full API and CLI `go test -race`, API/CLI `go vet`, catalog check, portability build, and Test Genie unit phase `20261121-201439-52e472a0` pass. Unit Health's stale 84% UI function floor was ratcheted to the measured 85.5% (committed 85%); 58 low-coverage advisories remain. |
| 2026-11-21 | Codex | done (setpoint fixture truthful for product/lab separation; broad Test Genie refreshed) | The live-inventory fixture now allows `partial` while it keeps acceptance unknown, matching the program contract when product composition omits lab-only evidence. Setpoint wall budget is 61,000 ms versus the historical rolling p95 of 60,006 ms; a fresh post-suite board run completed in 626 ms, with 5/18 target-aware simulated rows and `acceptance=unknown`. Full Test Genie run `20261121-180323-c0363bbb` recorded 22 passed, 7 failed, 1 not evaluated, 1 provider unavailable, and 1 skipped. The programs phase passed L2 and measures passed L3. Provider preparation took 21m20s and logged Audio Tools unreachable for 4m28s before service health recovered; that repeated lifecycle delay remains a reliability issue. Durable Test Genie status confirms the terminal result despite the attached waiter returning unexpected EOF. |
| 2026-11-21 | Codex | done (model facts de-duplicated; OpenRouter health uses encrypted BYOK; streaming egress cancellation fixed) | Provider catalog seeds no longer repeat model ID lists; model `provider_ids` are the single relationship owner, and strict seed decoding rejects unknown fields. OpenRouter health now uses the encrypted BYOK credential store and the catalog probe, removing the environment-key checker path. The deterministic fault matrix now exercises slow-consumer cancellation and catches an egress send that could block indefinitely; focused race tests pass. Full API race suite, `make catalog-check`, UI type-check/lint, Kale resource checks, and Sherpa package validation pass. Live paths selected Whisper Small, Sherpa Zipformer, Qwen3 1.7B, and Kale Nano; the attached Sherpa runtime still lacks signed managed lifecycle support, and Kale remains unqualified for quality promotion. |
| 2026-11-21 | Codex | done (persisted-domain measure coverage at L3) | Added owner-backed UTC-window count methods for BYOK credentials, corpus clips, experiments, experiment runs, speaker profiles, and wake-word templates. The BYOK measurement returns aggregate counts only. Replaced the temporary durable `byok_credentials_duplex` migration table with transaction-local staging. Six domain protos, MeasuresService RPCs, a measures-go registry and six CLI measure commands are live. `measures-health validate scenario audio-tools --probe --json` passes at L3 with 0 findings; each domain is covered at full tier and all six probes return conforming counts. Focused owner-store/repository/API/CLI tests passed. The latest full Test Genie run also passed the measures phase at L3. |
| 2026-11-21 | Codex | done (target-aware partial acceptance joined; broad suite findings retained) | Added a dedicated `ACCEPTANCE_COMPOSITE` evidence kind instead of mislabeling simulated results as interval accounting; simulated claims carry exact target ID/revision/method, artifact reference, execution lane, native distinction, and named checks. The acceptance join covers the same 18 required rows as the board and persists a partial composite with `acceptance_complete=false` and 13 missing rows. Hardened `setpoint-read` to reject legacy broad simulations and validate composite-wrapped simulated claims. Live lab simulated producer passed all 13 local checks; fresh board run `prog_74fa0118-ca6d-4993-872a-8894f7a69026` resolved 5/18 rows and kept overall acceptance unknown (`pending:required_owner_receipts`). All 32 Python setpoint tests, focused CLI validation tests, focused API experiment/qualification tests, and `make -C packages/proto verify-committed-gen` passed. Product API still returns 404 for the lab-only evidence route. |
| 2026-11-21 | Codex | done (Test Genie broad result captured) | Full server-owned run `20261121-164319-f007ff74` finished FAIL: 20/32 passed, 10 failed, 1 skipped in 349 seconds. The programs phase reported a live-inventory fixture mismatch during Audio Tools unavailability and a rolling p95 budget finding; the latest direct setpoint run completed successfully in 0.94 seconds with all three bindings readable, so the p95 finding is historical aggregation, not a measured current overrun. Passing phases included portability, structure, contracts, code graphs, performance, unit, business, measures, proto, provider conformance, and templates. Remaining maturity debt spans the Test Genie findings; no broad readiness claim is made. |
| 2026-11-21 | Codex | done (governed lab evidence binding restored; acceptance remains open) | Restarted only `program-runtime@lab` with `--variant-dependencies audio-tools`. Fresh `audio-tools.setpoint-read` run `prog_226780ad-ce00-4a9f-90f6-ce23b5ffbe19` succeeded through `audio-tools/experiment/list-evidence` with no binding errors. The apparent malformed item was a protojson omission of an empty repeated field, not a stored receipt; the board maps that typed default to zero receipts while still rejecting a present non-list field. After five PRD targets were added, the board and contract were extended to 20 exact target mappings (18 required, two explicit exclusions). Follow-up governed run `prog_9260077d-f0d9-4b86-8e76-6281dd3ac405` returns 20 rows, 18 required, 0 resolved, and `pending:required_owner_receipts`. Focused suite passes (31 tests). Product API still returns HTTP 404; product Program Runtime and Audio Tools remained running on original ports. The route-access and mapping gaps are closed; qualification remains unproven. Current details: [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-21 | Codex | done (bounded simulated policy receipt persisted in lab) | Verified `audio-tools@lab` health on API port 15373, ran `audio-tools --instance lab validation run-simulated --json`, and reran the governed lab setpoint as `prog_cd3541dd-03e9-48d5-b8c4-3d7851a89551`. All 13 simulation checks passed; its receipt is explicitly `execution_lane=simulated`, `simulated_as_native=false`, and `target_acceptance=false`. The scoreboard now shows 5/18 required rows with simulated coverage, 13 unavailable, and acceptance unknown. No product instance was targeted; this does not prove native/device, provider parity, corpus quality, latency, or overall acceptance. |
| 2026-11-21 | Codex | done (small-model runtime attribution and CLI JSON correctness) | Live product resolution confirms Whisper Small batch STT, Sherpa Zipformer stream config, and Qwen3 1.7B summarize. A short product synthesis selected Kale Nano (local, 1,281 ms) and wrote valid WAV. Kale readiness/liveness now resolves the provider-declared health path; its prior cached unavailable status was not reproducible against `/healthz` or synthesis. Host sample shows GPU residency of Whisper Small 719 MiB, shared Ollama `qwen3.5:9b` chat.default 7,702 MiB, and embedding 558 MiB; the 9B is not Audio Tools summarize. TTS `--json` previously prefixed binary audio, invalidating the proto JSON. It now emits valid Connect proto JSON; live output parses, TTS CLI race tests pass, provider/catalog drift gate passes, and bootstrap/capability package race tests pass. Full Kale quality/release and device qualification remain open. |
| 2026-11-21 | Codex | done (host resource attribution refreshed) | Read-only host sample: Whisper Small is active and uses 719 MiB VRAM; Ollama models use 8,308 MiB and shared reranker 1,380 MiB. Audio Tools API, attached Sherpa, and Kale processes are ~31 MiB, 8 MiB, and 19 MiB RSS. Resource record now reflects Whisper's real GPU share while confirming the smaller Small artifact is selected. No model was unloaded or service stopped. |

## Earlier history

| 2026-11-21 | Codex | done (timestamp drift evidence path + race fixes) | Eval replay now keeps provider word intervals with a 20,000-word cap and scores only against complete human-reviewed timing references; results include signed boundary drift rates per hour. Synthetic tests check offset vs progressive drift and 500 randomized alignment cases. Full API `GOWORK=off go test -race -count=1 -timeout=300s ./...` passes after fixing the experiment manager's eviction-before-`Wait` ordering and making OpenRouter transport tests independent of the real resource CLI. This adds a scoring path but does not qualify a model without reviewed recordings. |
| 2026-11-21 | Codex | done (Kroko finalization verified with model artifacts) | Native and WebSocket server tests built from current source passed against the repository dictation fixture and the Kroko bundle’s JFK sample. Both retained their final words, timestamps stayed within source duration, and punctuation rendered `brown, fox`; observed RTF was 69.76x / 67.93x on 1.955 s / 3.846 s clips. The attached server was not restarted, so current running-binary punctuation remains unverified; no general accuracy or promotion claim. |
| 2026-11-21 | Codex | done (API/CLI/UI validation after unary model selection) | Full API and CLI race suites, API/CLI vet, catalog check, Windows/amd64 + macOS/arm64 portability builds, UI type-check/lint, and 135-file / 1,282-test Vitest suite passed. Refreshed generated CLI primitive evidence after adding `stt transcribe --model`. Test Genie rerun could not get a run ID: concurrent `SQLITE_BUSY` on first attempt, then Test Genie CLI build failed on missing `sweepAbandonedDiskJobsTx` and `SQLiteStore.SweepAbandonedDiskJobs` in unrelated active code. |
| 2026-11-21 | Codex | done (Rumble unary product routing added) | Added explicit `X-Audio-Model-ID` selection to the existing unary Transcribe request. The route resolves provider/model/runtime identity from catalogs and checks the optional `AUDIO_RUMBLE_VLLM_URL` serves the declared model. Default remains Whisper; Rumble stays out of streaming selection and promotion. Fake-server routing/default/unavailable tests, full API race suite, `go vet ./...`, `make catalog-check`, and `make portability` pass; runtime quality remains unmeasured. |
| 2026-11-21 | Codex | done (Kale Nano selected by live auto product route) | Managed `kale-tts` was healthy; a fresh `audio-tools tts synthesize` request under `AUDIO_TTS_BACKEND=auto` returned a valid 24 kHz mono PCM WAV through `kale-tts` / `kale-tts-1-nano` / `kenzie` in 754 ms. This proves current route selection and serviceability, not held-out quality, multi-voice intelligibility, or release qualification. |
| 2026-11-21 | Codex | done (duplex model selection fails closed; runtime evidence refreshed) | GPT-Live now rejects an omitted model before credential lookup instead of silently selecting a provider-local model ID; `go test -race ./internal/duplex` passes. The attached Zipformer endpoint preserved the final “jumps” on the repository fixture; this is not Kroko-specific evidence, and fresh Kroko execution remains unavailable without its model artifact. Sherpa TTS route tests pass and remain separate from Kale qualification. |
| 2026-11-21 | Codex | done (unit rerun passed; measures CLI coverage expanded) | Test Genie unit run `20261121-202116-0d531f64` passed L2/Complete in 60 seconds (61 observations). The six measures RPC branches now have success/error/cancellation and JSON/human rendering coverage, plus time-window parsing and registration; package coverage is 96.1%. The overall scenario remains functionally incomplete against historical requirement and operational-target evidence, with 58 low-coverage warnings. The canonical OpenAI credential command was exercised only to its hidden prompt and canceled before secret entry; encrypted-file provider remains available and field unconfigured. |
| 2026-11-21 | Codex | done (credential path confirmed; Sherpa stage validates unsigned) | The OpenAI GPT-Live credential identity reports encrypted-file provider available and unconfigured; `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key` supports hidden terminal entry or stdin. No API key was created and production duplex remains gated. Downloaded official sherpa-onnx v1.13.2 Linux x64 shared runtime and verified SHA-256 `1ef6741535f7af4d69e394fd440a807108036d26ed4f542660191019da5c0daa`; native tagged server tests pass. Updated the stale linux-amd64 bundle digest to the current working-tree adapter (`e7e3f513383f15aa4f9e2050b752fdc94aa1ace16aee8306324fb97406a1a513`); fresh unsigned stage validation passes and archive checksum is `2c47c2e00eb6a044ef22c3e4cc6af1b70b9186a647d5e3d7785fec7e2e660a9c`. Signing/publication remains gated on release-owner URL/provenance. |
| 2026-11-21 | Codex | done (bounded session replay and duplex wire events) | Session fanout and shutdown now signal overflow with `ResourceExhausted` and a resume anchor. `Subscribe(from_event_id)` replays a contiguous 128-event / 1 MiB in-process tail, rejects unknown or expired cursors with `FailedPrecondition`, and invalidates history across oversized events. The session proto now carries TTS audio frames, provider errors, and overflow resume cursors; the mapper preserves them. Connect integration tests cover replay and cursor rejection. Full API `go test -race -count=1 ./...` passed; focused package vet and `make verify-committed-gen` passed. Replay does not survive API restart, so durable recovery acceptance remains open. |
| 2026-11-21 | Codex | done (Test Genie unit phase refreshed) | Server-owned run `20261121-142501-7355100b` passed the unit phase at L2/ceiling in 94 seconds; proto and shared browser/component packages were freshness-checked first. The run still records 59 advisories (58 low-coverage warnings, 1 committed UI coverage-floor info) and overall completeness `functional_incomplete` (2/29 requirements, 2/20 operational targets passing), with stale business/docs/proto/structure evidence. This is a green unit phase, not a full readiness verdict. |
| 2026-11-21 | Codex | done (GPT-Live credential setup path + UI lint debt) | The Configuration page now directs duplex credential setup to Vrooli's canonical encrypted credential authority and no longer accepts the key into the separate legacy BYOK store. The authority status reports provider available and the field unconfigured; the lazy resolver reads and observes rotations. No Platform key-create action was exposed in this session and the native setup chooser returned no confirmed choice, so no key or live call was created. Five copy-lint violations in `ModelsTable.test.tsx` were corrected by referring to stable model/engine IDs through named constants. Focused Configuration/ModelsTable tests (9 passed), full UI lint, and type-check pass. Provisioning does not enable the still-gated duplex route. |
| 2026-11-21 | Codex | done (capture monitor teardown race) | `startLevelMonitor` now captures a generation before awaiting the shared AudioContext; teardown invalidates it so a stale continuation cannot create audio nodes or restart RAF after capture ends. A deferred-context hook regression exercises the exact sequence. Browser capture package suite passes (34 files, 243 passed, 1 skipped) and build passes. Full UI lint is now clean after fixing the adjacent ModelsTable test copy-lint violations. |
| 2026-11-21 | Codex | done (UI result wrapper and production build) | `tryCall` no longer dynamically imports `api/client` to construct an error when it already imports `ApiError` from the same module; the redundant dynamic import was producing a Rollup chunk warning without splitting code. Result-wrapper tests, type-check, full lint, and production build pass. The unrelated >500 kB main-bundle warning remains; the entry chunk is ~854 kB / 254 kB gzip and route pages are already split. |

The complete pre-cleanup log, including intermediate results and unresolved
handoffs, is preserved byte-for-byte at the protected runtime-home location:

```text
<runtime-home>/plan-artifacts/docs-history-20260907/scenarios/audio-tools/docs/internal/PROGRESS.md
```

SHA-256: `5756d0535ed5652dfd404e391eace8dcdfc971d12d492ace26c22b61b0c2b331`. Restore or read it before relying on an
older completion claim; archival does not close any outstanding work. The
[project preservation record](../../../../docs/internal/PROGRESS.md#documentation-cleanup--2026-09-07)
records ownership and recovery.

## Progress Log

### Recent milestones

| Date | Author | Status | Notes |
|---|---|---|---|
| 2026-11-21 | Codex | done (D1 CPU + D2 Sherpa cells recorded; matrix incomplete) | The bounded host-portability producer now runs D1 Whisper Small CPU and D2 Sherpa int8 independently. D1 recognized the fixture in 2.246 s under a digest-pinned, no-network 8-CPU/8-GiB container; the same receipt's D2 run took 120.2 ms for 1.955 s (16.26x RTF), 1.765 s load, and 438,358,016-byte peak in its 2-CPU/4-GiB container. Receipt `audio-tools:host-portability:20261122T002105.273493236Z:amd64` preserves artifact hashes and reports `passed:false`; Kyutai accelerator inference remains unavailable. The earlier D2 relative-path defect attempt and serving API 404 remain visible in receipts; lab evidence belongs to the supported lab API. The setpoint remains 5/18 resolved, 13 unavailable, with host-portability claim out of band. Full API race and CLI suites, `make lab-test`, `make portability`, `make catalog-check`, API vet, and UI type-check/lint/coverage suites pass; the run also removed a redundant UI optional chain flagged by lint. Measures-health passes with zero findings. GPT-Live remains securely provisionable through `vrooli credentials provision --identity vrooli/openai-gpt-live --field api-key`; no key creation action is connected and the duplex route remains unmounted. Detailed gate/status: [MATURATION-STATE.md](MATURATION-STATE.md). |
| 2026-11-20 | Codex | done (GPT-Live read-loop cancellation) | The GPT-Live WebSocket reader now uses the session request context; cancellation closes the socket and exits the read loop without emitting a spurious disconnect. `gofmt -d` and scoped `git diff --check` are clean. The comprehensive Test Genie run `20261120-230712-5da63d84` completed with FAIL: unit/proto/API passed, while soak/workflow gates lacked verified grants/account, monetization and provider conformance remain incomplete, and security findings included the context issue fixed here plus separate SSRF findings. The run predates this source change and is not evidence of the fix. |
| 2026-11-20 | Codex | done (validation CLI injectable seams) | Replaced direct wall-clock use in the speaker-policy streaming probe with the handler's `Now` seam and wrapped the Docker host probe behind an injectable command runner. Unit Health's prior `MISSING_INJECTABLE_SEAM` came from the noncanonical lowercase clock seam; the handler now exposes the canonical function field. Full CLI race suite passes, and Test Genie unit run `20261120-225715-4d8dd9c5` passes at L2 Complete with no architecture blocker; advisory coverage findings remain. |
| 2026-11-20 | Codex | done (audio response metadata contract debt closed) | The historic audio-metadata issue was stale: transform handlers already probe output duration, Normalize measures EBU R128 output rather than echoing the target, and unavailable measurements return zero. Added explicit protobuf comments defining zero as unknown for output durations, split chunks, and measured LUFS; regenerated through the proto owner and `make -C packages/proto verify-committed-gen` now passes. Focused handler/audio race tests pass. |
| 2026-11-20 | Codex | done (cache fallback declaration cleanup + runtime and accounting gates sharpened) | Auto-mode TTS cache replay now derives backend keys from the configured local provider/failover chain instead of a separate hardcoded Kale/Kokoro list; a regression verifies a newly named fallback cache backend. Rumble remains candidate with no engine/artifact and cannot become default before a managed adapter; the footprint audit records the upstream 1.39 GB download size while leaving RAM/VRAM/runtime envelope unknown. Durable duplex accounting remains gated on verified account resolution and an authoritative limit source; the lossy usage analytics queue is not billing authority. Full API race suite, coverage floors, API vet, catalog check, modelcatalog tests, and scoped diff check pass. |
| 2026-11-20 | Codex | done (manual timing provenance + corpus readiness audit; no score qualification) | Human-reviewed timed references may now declare manual-only provenance without fabricated aligner metadata; aligner-only references still require both tool and model, and partial aligner provenance is rejected. Added API and CLI regression coverage. Corpus audit found num[sot]:180 registered clips and zero timed references, so no AMI labels were imported or promoted; AMI's published forced-alignment times still require human correction. Corrected the planned Rumble catalog declaration to batch. API full race suite and vet, CLI full race suite (after regenerating stale primitive evidence through its owning test), catalog generation/check, and scoped diff check pass. |
| 2026-11-20 | Codex | done (Kroko tail regression verified; identity context wired; TTS fallback hardened) | Native-tagged Kroko server smoke passed on two JFK WAVs with the existing bounded 1.2 s finish tail: both retain “your country” and source-bounded timestamps (11 s clip: 362 ms after 3.63 s model load; 55 s clip: 844 ms after 1.02 s warm load). No Kroko code change; this verifies tail preservation only, not model quality. Kale→Kokoro fallback now handles nil configured services without panic. Audio Tools' custom server now installs passive api-core authn/provenance middleware; verified-human propagation tests pass, but no duplex route or durable spend authority exists. Added BYOK migration rollback/error tests, raising database coverage from 73.1% to 92.3%. Full API race suite and package coverage floors pass: `go test -race -count=1 -timeout=300s ./...`. Current host resolves Whisper Small (488 MB), Zipformer streaming (310 MB), Qwen3 1.7B (1.36 GB), and 27 MB speaker embeddings; Kale remains unqualified and stopped. |
| 2026-11-20 | Codex | done (runtime Kale-to-Kokoro fallback; quality/release qualification open) | Default TTS auto mode now keeps Kale Nano preferred and Kokoro available for cached-liveness selection and transport/availability fallback; non-availability synthesis errors still surface. Runtime regression verifies Kale dropping after startup falls back with Kokoro voice/provider/model/cache identity and prefers Kale again after recovery. Focused bootstrap/TTS tests, focused race tests, and `make catalog-check` pass. No naturalness or release qualification is claimed; the model catalog still records those receipts as open. |
| 2026-11-20 | Codex | done (Kale model-backed product route; quality gate open) | Connected the isolated Audio Tools Connect product TTS route to the managed Kale Nano runtime. It returned 98,424 bytes / 2,048 ms of WAV; warm request was 175 ms and repeat calls hit cache. A 1.25x speed request retained valid finite RIFF/data sizes. Fixed the integration bug where Kale rejected an extra `speed` field; tempo conversion now happens locally. The resource was stopped after the live test. This proves model-backed product routing and cache behavior, not naturalness or release readiness; Kokoro remains default. See `MATURATION-STATE.md` and the tenth tranche in `MATURATION-PROGRAM.md`. |
| 2026-11-20 | Codex | done (offline fault diagnostics; non-promotable) | Implemented a data-driven offline matrix through the real server seams for provider timeout, truncated decoder stream, and cancellation/worker cleanup. Persisted `coverage/offline-fault-diagnostics-20261120.json` with `promotable:false`, zero product-path rows, and an explicit failed browser-product join; eleven browser/device profiles remain unsupported and visible. The focused `go test -race` command passed independently. Duplex `Handle.Close` now makes concurrent callers wait for cleanup and return the same error; its race regression passed. No browser/device qualification is claimed. |
| 2026-11-21 | Codex | done (offline provider recovery diagnostics; non-promotable) | Extended the server-only matrix with provider-busy, dropped-connection, and backend-restart recovery profiles. The restart case proves a transient 503 can be followed by a successful response through the same segmenter session. The receipt still records zero product-path rows and `promotable:false`; eight declared browser/device profiles remain unsupported. Focused segmenter race suite passed. No browser/device qualification is claimed. |
| 2026-11-20 | Codex | done (Kale product-route regression + duplex reservation primitive; provider work remains open) | Added an isolated bootstrap/Connect test proving the product TTS route selects Kale Nano, maps the canonical voice, returns WAV/provider metadata, and hits cache on repeat against a fake runtime (`go test ./internal/bootstrap -run '^TestBuildKaleTTSProductConnectRouteAndCache$' -count=1`). Added process-local duplex budget reservations with fail-closed admission, concurrent overspend prevention, exactly-once settlement and automatic release on close/failure/cancellation; nil/error reservation cases are covered. `go test -race ./internal/duplex` passed. Fresh live CLI resolves streaming STT to `sherpa-streaming` and summarize to `qwen3:1.7b`. No real Kale product audio quality, persistent spend ledger, pricing policy, or OpenAI Live integration is claimed. |
| 2026-11-20 | Codex | done (product/lab default boundary and qualification evidence repair) | **Product composition now excludes lab routes unless explicitly opted in.** Bootstrap uses product composition by default; `AUDIO_ENABLE_LAB_API=true` enables lab routes. External Test Genie validation endpoints are part of the endpoint inventory and are explicitly omitted from the product CLI. Added a partition test that brought `internal/modules` to 100% coverage after the API floor caught the gap. Registered the model-footprint audit and added a current state page. Fresh runtime attribution: Audio Tools API/UI were 19.8/38.3 MiB RSS; the idle Sherpa endpoint was 7.7 MiB; shared Ollama held 5.88 GB VRAM for `qwen3.5:9b` and 323 MB for embeddings, while live policy resolves summarize to `qwen3:1.7b`. Full API `-race`, API/CLI vet, CLI suite, catalog/endpoint checks, Windows+macOS builds, focused docs phase, and scoped diff check passed. Still open: the running API has not been restarted onto this source; human-reviewed timestamp labels, complete device/browser qualification, Kale TTS quality/release/product receipt, Rumble runtime, and provider-backed duplex remain unqualified. |
| 2026-11-20 | Codex | done (Kale resource-backed synthesis; product path and quality remain open) | Managed Kale Nano start/stop worked; startup wall time was 6.14 s. A 3.05 s mono 24 kHz WAV synthesized from one English utterance in 770 ms cold and 59 ms warm. Pinned graph SHA256 `4d5269974de4fba7a66f947608a9a5381b4e39c1a8a6073df6001ab7e7f76288`. The already-running Audio Tools API was not restarted; its process does not have the Kale backend configured, so this is resource-level output only. Resource returned to stopped. One voice/utterance does not assess naturalness, and upstream still marks weights DRAFT/NOT PUBLISHED. Kokoro remains the product default. |
| 2026-11-20 | Codex | done (Zipformer license provenance clarified) | Audited the live runtime declaration against the upstream Zengwei model card. The card declares Apache-2.0 for the model weights; LibriSpeech is training-data provenance, not a license statement for the weights. Removed the conflicting `CC BY 4.0` model-weight wording from the sherpa acquisition record and clarified the catalog notes. Verified with `make models-doc`, `make catalog-check`, and `vrooli resource validate sherpa-onnx`. |
| 2026-11-20 | Codex | done (local sherpa smoke comparison; not model-qualified) | Replayed the synthetic 1.955 s `quality_speech.wav` three times through native sherpa-onnx on CPU (2 threads), with the same runtime, punctuation model, and chunking. Both 2023 Zipformer and Kroko 2025 recognized the final word `jumps` every run. `AcceptPCM` inference RTF was 28.35–32.97x for 2023 and 57.83–80.82x for Kroko; load was 0.89–0.94 s and 0.89–1.05 s, respectively. One short synthetic clip is not an accuracy or product-latency qualification and does not justify changing the selected/default model. Kroko still needs full-corpus evaluation, licensing review, and managed runtime lifecycle support. |
| 2026-11-20 | Codex | done (timing observability + honest portability receipts; matrix still incomplete) | **Retained real hypothesis intervals and stopped inventory from masquerading as host qualification.** Unary local `TranscribeResponse` now includes provider word timestamps when egress passes and coordinates remain valid; eval replay carries bounded provider timings (20,000 words/clip) into reports without persisting audio or inventing a drift score. Native sherpa streaming still omits word intervals, and there is no human-reviewed long-form timed reference. The host portability producer now stores engine inventory separately and marks D1 CPU, D1 accelerator, and D2 inference `not_run`; the acceptance board and owner join reject a scalar pass without three successful inference cells. A live attach-only sherpa smoke on the synthetic five-word clip emitted partials/final but omitted “jumps” (20% WER); this is a route smoke only. Fixed native streaming segment latency reporting from hardcoded zero to elapsed fractional milliseconds, with regression coverage. Full API and CLI Go suites, targeted race suite, lab tests, catalog check, Python acceptance-board suite, proto generation/breaking/lint checks pass. Current host status: qwen3.5:9b remains resident under shared `chat.default`, not Audio Tools `summarize.default`; sherpa is attached externally, not managed. No model or device qualification is claimed. |
| 2026-11-20 | Codex | done (Kale isolated runtime + API route; not model-qualified) | **Added a checksum-pinned CPU-only Kale Nano managed runtime candidate and an explicit Audio Tools integration route.** `resources/kale-tts` pins the Linux/amd64 CPython 3.12.14 runtime, a 22-package hash-locked wheel closure, Nano int8 and required voice/tokenizer/English frontend assets; one worker and one CPU thread are enforced. The source-built resource tree digest `f70d4e25a86a3bb4ef9c269d99cc34164d267e7ffc4ab86a7c8b5d8708fe3061` was produced by `vrooli resource acquisition compose-candidate` and reproduced with `--offline`; `vrooli resource validate kale-tts` and the resource's 5 API contract tests pass. `/healthz` returns 503 when required assets are missing without loading ONNX. Audio Tools accepts explicit `AUDIO_TTS_BACKEND=kale-onnx`, requests Kale Nano/WAV, converts to the requested output format through the existing audio substrate, maps canonical voices, and namespaces persistent cache entries by backend. Fake-runtime integration and `go test ./...`, `go vet ./...`, focused TTS/backend `-race`, and catalog checks pass. Still not claimed: resource install/start, model-backed service receipt, provider-catalog promotion, quality qualification, or default change; at that time the upstream card showed DRAFT, NOT PUBLISHED; the 2026-11-21 source correction above now records Nano as public Apache-2.0. |
| 2026-11-20 | Codex | done (session lifecycle hardening + Kale draft audit) | **Fixed session close races and refreshed the Kale readiness facts.** Subscribe and Close now serialize on the observer lock; EmitEvent checks closed state under its fanout lock; the declared 16-observer cap is enforced; cancellable subscriptions release their watcher on explicit unsubscribe/close; stale action completion clears only its own in-flight ID. `go test -race ./internal/session` passed. At the time of this entry, Nano/Tiny pages showed draft status; current metadata (see correction above) now reports Nano public Apache-2.0, while Tiny remains draft; Nano's int8 checksum matched, and the upstream JS/ONNX path produced 3.82 s of 24 kHz speech in 648 ms including load on CPU/one thread (~238 MB process RSS). The provider remains planned because the existing sherpa Kokoro adapter cannot load this graph and no managed runtime or quality receipt exists. |
| 2026-11-20 | Codex | done (model catalog hardware truth) | **Removed unsupported Rumble default and host-capability claims.** The pinned Hugging Face repo is 1.39 GB including a 1.02 GB main safetensors file and a 357 MB CTC head. The candidate has no Audio Tools adapter/resource or latency receipt; its upstream path is GPU-oriented, so CPU support is no longer claimed, and required memory remains unknown. It is no longer assigned the default role before an engine exists. Catalog docs were regenerated and checked. |
| 2026-11-20 | Codex | done (runtime resource attribution) | **Verified the current light STT default through the product path and measured current host use.** Audio Tools API was idle at ~24 MiB RSS with no active model process; GPU use belonged to shared Ollama and reranker. A managed Whisper-small run transcribed the canonical five-word clip exactly in 75 ms with 716 MiB measured VRAM under its 1 GiB claim. Whisper was stopped afterward and its claim is now absent. Current word-level timestamp drift remains unscored: the 61-minute fixture has no timed reference, and product Transcribe omits word intervals. `vrooli capacity reconcile` also reports Ollama at 7.30 GiB observed vs 5.78 GiB claimed, with a resident qwen3.5:9b; attribution to audio-tools is unproven, so it was left untouched. |
| 2026-11-20 | Codex | done (local TTS route + Kroko finalization fix) | **Sherpa-hosted Kokoro is selectable without changing the default, and Kroko's final tokens are retained.** `AUDIO_TTS_BACKEND=sherpa-onnx` routes the existing `tts.Synthesizer` contract to `AUDIO_SHERPA_URL`; default `kokoro` remains unchanged, and service/RPC cache ownership is preserved. Sherpa `/healthz` now reports TTS model/encoder state and is used for cheap cached request liveness; explicit readiness still runs synthesis. Native Kroko smoke evidence isolated truncation to immediate `Finish()`: JFK ended at “for.” without tail padding and at “your country.” with 1.2 s. `Finish()` now feeds bounded silence without extending reported client timestamps. Verified: API full suite + full `-race`, API `go vet`, catalog check, sherpa server tests/vet, resource validation, and native Kroko model smoke with final-suffix and timestamp-bound assertions. Still open: broad Kroko quality/latency/license receipts; Kale weights; sherpa lifecycle publication; long-form timestamp drift and device matrix. |
| 2026-11-20 | Platform agent | done (low-footprint models + sherpa attach) | **Low-footprint defaults.** Summarize moved to `qwen3:1.7b` (1.36 GB, 8/8 facts short / 14/22 long; 4B remains quality escalation), with the ollama acceleration rung and ai-gateway catalog regenerated. Sherpa streaming is attached on port 8881 with the 310 MB 2023 English Zipformer; `/healthz`, port, and engine-ranking defects fixed; attach-only selection accepts a passing provider health probe and rejects a dead one. Engine policy is `native_streaming` → `low_footprint` → `accelerated`. The 68 MB Kroko model is on demand because the server path truncates it; the upstream file CLI is clean. The sherpa adapter also serves Kokoro int8 (183 MB), the intended host for Kale when it ships. Fleet: 12 GB abandoned Kyutai download removed. Verified: live streaming transcription, `stt engines`, API/CLI suites, `make catalog-check`, resource validation. |
| 2026-11-20 | Platform agent | done (footprint + lab boundary) | **Smaller defaults are now enforced, not aspirational.** Whisper ships `ggml-small` (488 MB) as the eager runtime default; turbo/medium/base/tiny are on-demand tiers; the recommender defaults to small with `--quality` as the explicit accuracy path; the capacity claim halved to 1 GiB. The shared summarize role moved from `qwen3.5:9b` (~5.9 GB) to `qwen3.5:4b` (3.39 GB installed), which was already its qualification candidate. Per-capability footprint budgets (STT 600 MB, TTS 400 MB, summarize 4 GB, speaker 100 MB) fail the build when a promoted default grows, individually and in aggregate; `TestLocalDefaultArtifactsAreEager` fails when a default names an on-demand artifact. Footprint is visible in `docs/reference/models.md`, the `model list` SIZE column + defaults summary, and a Models-page Resource footprint panel. One-host canonical-clip receipt: all Whisper tiers exact, warm 0.16/0.18/0.31/0.30 s for tiny/base/small/turbo. Lab/qualification packages now have measured coverage floors, `make lab-test`, and a parser-based import gate proving the serving handlers and chains cannot reach experiment/eval/soak/validation; bootstrap/server crossings are enumerated exceptions. Verified: API `go test ./...`, `go vet`, `TestCoverageFloors`, `make catalog-check`, `make lab-test`, `vrooli resource validate whisper`. |
| 2026-11-19 | Platform agent | done (correctness + identity + UX) | **Silent voice-override bug, transform edge cases, provider identity, and composition truth.** Local Kokoro overrides were saved as `local:kokoro` but read as `local:kokoro-local`, so they silently never applied; one constant now owns the key, legacy keys still resolve, `VerifyCatalog` runs at startup (it was test-only), and settings rejects unknown voice/adapter pairs. The Voices matrix is now editable with inline save/delete, error recovery, and 10 tests. Transforms: empty ffmpeg output is an error, merge rejects crossfades longer than the shortest source, trim/split reject windows past the measured end, split reports the final chunk duration, transcode knobs and loudnorm targets are range-checked. STT engine resolution inherits the session context; typed-nil optional repos no longer panic at construction; `Compose` must mount every declared endpoint. Provider identity aliases and BYOK probe facts moved into the provider catalog; BYOK credential writes must match an active provider capability; TTS/summarize requests are message-bounded; readiness probes use a 30 s doer and BYOK probes 5 s. A mid-stream decoder failure now surfaces as `input_decode_failed` and fails the replay ledger instead of looking like a clean EOF, and provider liveness latency is measured instead of hardcoded to zero. Verified: API `go test ./...` and full `-race`; CLI suite; UI type-check/lint/134 files 1273 tests; `make catalog-check`; `make portability`. |
| 2026-11-19 | Platform agent | done (runtime truth + first model receipt) | **Whisper runtime, audio-response truth, lifecycle idempotency, and the admin model join landed.** `resources/whisper` now ships checksum-pinned `ggml-large-v3-turbo.bin` as the eager artifact (medium is the on-demand rollback group) and the registry names the artifact behind every available local model; two new gates fail on registry/runtime artifact drift. Receipt on this host: canonical clip WER 0.00; 61-minute fixture 30.3 s turbo vs 52.8 s medium (RTF 120.8x vs 69.3x), 0.9993 word agreement. EBU R128 normalization is now two-pass/linear and reports a genuinely measured `measured_lufs`; transcode/trim/merge report output duration. Mutating provider actions dedupe on Idempotency-Key with single-flight, TTL, bounded eviction, no error caching, and cloned responses. Stream Config joins each STT engine to registry models (default/readiness/licence/runtime/size/languages, explicit no-model state). Transform and STT Connect handlers now reject oversized messages before buffering (256 MiB per transform, 16 MiB per STT message). Verified: API `go test ./...`, `go vet`, `-race` on stt/ai/tts and the touched handler packages; CLI suite; UI type-check/lint/134 files 1264 tests; `make catalog-check`; `vrooli resource validate whisper`. Not claimed: device matrix, timestamp-drift scoring, multi-language WER. |
| 2026-11-19 | Platform agent | done (registry surface, not qualification) | **Model registry became a runtime surface and the STT/lint backlog moved down.** Added `ModelRegistryService` (`ListModels`, `ResolveModel`) with host-`Platform`-aware defaults, a CLI `model` group, and a `/models` UI page (filters, readiness/licence/hardware facts, resolve explanation). Corrected the summarize records to mirror `resources/ollama/model-policy.json` (`qwen3.5:9b` runtime primary; `llama3.2:3b` remains the commercially certified, evidence-backed default) and gated that mirroring. Added engine integrity gates across provider seed, model seed, and `sttengine/manifest.json`, plus a whisperinfo↔registry round-trip gate. Fixed a streaming-decoder teardown leak (a reader parked on a full frame buffer now unblocks on `Close`). STT residuals: observable timeline-range failure, joined normalizer teardown, per-run strategy config, and a bounded cancel tail flush. Consolidated shared BYOK/LPBS chain scaffolding (`dupl -t 80` is now zero clone groups in `internal/ai`), made streaming credit exhaustion terminal, redacted request credentials from fallback reasons, fixed an availability-probe resurrection race, restored coverage floors (sttchain 81.2%, ttschain 86.0%, summarizechain 93.8%, handlers/stt 83.3%), and mapped wrapped chain sentinels with `errors.Is` instead of `==`. Regenerated `.vrooli/endpoints.json` for all eight audio ops and the model service. UI lint is now clean (was 65 problems). Verified: API `go test ./...` and `go vet ./...` pass; targeted `-race` pass; CLI suite pass; UI type-check/lint clean and 134 files / 1258 tests pass; generated artifacts byte-stable. No model/provider/device qualification is claimed. |
| 2026-11-19 | Platform agent | done (substrate, not qualification) | **Declaration substrate and first debt/reliability tranche landed.** Added `internal/providercatalog` (19 provider records, generated Go/TS catalogs, drift + provider↔model↔engine conformance), `internal/operationcatalog` (13 operations; all 8 audio transforms now reachable from CLI + Diagnostics UI), and `internal/modelcatalog` (23 model/voice records with licence/readiness/hardware, generated `docs/reference/models.md`, commercial-default fail-closed). Fixed the `sherpa-streaming` engine drift, derived the lifecycle allowlist and provider order from the catalog, and made public results report canonical provider IDs. Reliability: VAD retry backoff + leading-silence trim, OverlapAgree committed-PCM compaction, Passthrough terminal `Done`, session snapshot ordering, pipeline dependency/snapshot locks, and goleak baseline. Audio correctness: AAC→ADTS, trim duration, fade-out anchor, merge crossfade/temp cleanup, cancellation mapping, split validation, audio-only ffprobe, portability flags. Gates: full API/CLI/UI suites green (`go test ./...`, 131 UI files/1229 tests, `pnpm type-check`), `go vet` clean, measured coverage floors. No live model/provider/device qualification is claimed; see `MATURATION-PROGRAM.md` for the remaining phases. |
| 2026-08-18 | codex | docs/audit-resolution-refresh | Refreshed `plan-artifacts/docs-cleanup-20260907-final-txumdx73/docs/design/audio-reliability-audit-2026-08-09.html` so its resolution ledger and evidence delta distinguish the historical baseline from current state: the BAS/PipeWire device lane is automated and OS-path-scoped, current Audio Tools realtime evidence is named, Swarm Manager product-path credit remains pending, and native sherpa publication/target smoke remain explicit gaps. Removed stale active claims that all audio resources still require Docker or that the qualification instrument does not exist. |
| 2026-08-18 | codex | fix/native-release-reproducibility | Native sherpa release packaging now normalizes gzip metadata, detects tar's `--sort=ORDER` capability correctly, and has a sorted NUL-safe deterministic fallback for non-GNU tar (otherwise it fails closed). Identical target-native stages produce archive checksum `440f9d9738a2457761faa37545f1d28aa084db35d4a47bf7aa94595cedfe0032`; publication metadata includes `entry_path`. `bash -n`, the release-stage check, `make -C resources/sherpa-onnx check`, and two fresh signed release stages pass. Publication and target smoke gates remain intentionally open. |
| 2026-08-18 | codex | fix/swarm-search-registration | Repaired Swarm Manager's committed search descriptor by removing the obsolete `declared_at` field and adding required dense tuning for the initiative provider; the test now validates both intended providers and selects the records provider by id. The complete Swarm API package passes, and a governed restart is healthy with both providers updated in search-hub. |

## New entries

### 2026-09-09 — Portable voice documentation and contract revision

Regenerated the PRD through Business Health, preserving the original requirement
identities and evidence. Aligned requirements, architecture/domain mappings,
performance/testing and full-mandate completion, privacy/data/deployment,
commercial policy, experience priorities and canonical usage/improvement skills.
Four unsupported completion statuses returned to in-progress. No runtime voice
behavior was changed or qualified. The read-only board's v2 target inventory now
matches all 15 PRD targets, with every outcome still unknown. The hosted path, sensor joins,
native cohorts and pending acceptance decisions remain implementation/review work.

Append a dated milestone with its outcome, remaining constraint, and durable
owner reference. Store detailed command output with the producing owner.

### Whisper per-tier model-cache retention (2026-11-21)

- Split the former shared model storage entry into Medium and Turbo on-demand
  cache entries. Both preserve the existing `WHISPER_MODELS_DIR` container
  relocation. A new exact-artifact owner route validates declared artifact
  groups and paths, snapshots the whole owned tree, and removes only the
  selected model plus checksum after a reviewed preview remains unchanged.
- The distinct `model-cache-artifacts` command identity makes the legacy
  whole-entry route reject selective entries. The installed CLI refreshed its
  command list from source and previews now report only the selected model
  sizes: Medium 1,533,763,124 B and Turbo 1,624,555,275 B. Both are currently
  ineligible because use/model identity is unknown; no pin change or deletion
  occurred.
- Focused model-cache race tests, CLI/resource-handler tests, api-core storage
  tests, Whisper resource validation, and `git diff --check` pass. The resource
  manifest and resource-store state contracts are now v2 with their prior v1
  digests retained and v1 still readable, matching the new optional artifact
  declaration. `GOWORK=off go test -race -count=1 -timeout=900s
  ./internal/resources` passes in 352 seconds. The package exceeds its former
  300-second timeout; the isolated OpenRGB test passes in 6.8 seconds. A QA
  observation records the timeout; its heartbeat wake is held by the global
  scenario-qa pause. Wider manifest-check output remains noisy from pre-existing
  repository drift; no model-cache-artifacts drift was reported. The acquired
  models remain intact.

### 2026-11-22 — repeatable Sherpa stage and Kale product latency sample

- Rebuilt the Sherpa Linux/amd64 unsigned release stage twice from the pinned
  runtime and current source. Both builds produced tree digest
  `4985ca8c0384fee5457df3de9633430f8f5ea19029a533cad5333047e4ba89d3` and
  archive SHA `257ed38f47e717d243eec7b64a5cbc3da24e58ffce9b41120afd0fe9a02e015c`.
  Updated the resource pin to `1.13.2-vrooli.3`; resource validation,
  `validate-stage`, `make check`, and fresh unsigned staging pass. Signing and
  publication remain with the release owner.
- Follow-up isolated the earlier mismatch: the retained 10:12 stage is stale
  for the current source (`e7e3f513…06a1a513`), while the current tree and pin
  are `4985ca8c…e4ba89d3`. Two fresh native `release-stage
  TARGET=linux-amd64 SIGN=0` builds from the extracted runtime both passed
  `validate-stage` and produced the same archive SHA above. This is unsigned
  reproducibility evidence only; it does not qualify signing, publication, or
  managed install.
- Model-backed Kroko tests distinguish finalization behavior: the bounded
  1.2-second tail passes the expected final suffix, while tail=0 fails it on
  the same fixture. The WebSocket test also asserts timestamps stay within
  source duration. Kroko stays experimental pending broader quality and
  platform receipts.
- A healthy product API and Kale Nano service completed five identical
  `audio-tools tts synthesize` requests. Every response selected
  `kale-tts` / `kale-tts-1-nano` / `kenzie` and returned valid 24 kHz mono PCM
  WAV, 5.625 seconds long. Product latency was 1,462 ms first, then 171, 155,
  212 and 211 ms. Kale process RSS was 137,764 KiB in the contemporaneous
  process snapshot. A direct binary built once was used for the reported
  samples after the self-updating CLI wrapper emitted changing fingerprints;
  wrapper behavior is not yet explained. One prompt/voice and one host do not
  establish model quality, cohort latency, or default promotion.
- Added a focused summary-role fallback-description test to recover the
  enforced summarizer coverage floor, regenerated stale CLI primitive evidence
  with its owning test, and repaired the DiagnosticsPage service mock for the
  declared role query. The page suite passed 4/4 and the UI type-check passed.
  Audio Tools `make catalog-check` passed.
- LPBS plan creation now exposes app-scoped feature and limit declarations
  using the existing `app_entitlements` API shape. The focused UI suite passed
  17/17; backend declaration and entitlement tests passed. The local lifecycle
  test drives a simulated $10 offer, fake-provider upgrade invoice, payment
  failure/recovery, scheduled cancellation retaining access, and terminal
  deletion revoking it. Focused API tests passed; no Stripe call or production
  SKU was used. `pnpm type-check` still fails only at the Vite config because
  the shared component library and LPBS UI resolve incompatible Vite/Rollup
  types; a source-only check also hits the existing `billing.ts:215`
  `JsonValue` conversion type mismatch, not the new entitlement declaration.
  These static-check issues remain open. This simulation does not prove actual
  Stripe proration or settlement.
- Test Genie run `20261122-030155-bb18dcb3` passed the explicit unit phase at
  L2 with 58 advisory low-coverage findings. The advertised `quick` preset
  remains broken: the server rejected it before any phase executed. Bug
  `knw-1795315956599756235` is published, but its investigator wake was
  rejected because scenario-qa heartbeats are globally paused. Three unrelated
  repository trailing-blank-line findings remain in the full-tree
  `git diff --check`; touched audio-tools and Sherpa files pass that check.

### 2026-11-22 — Sherpa clean-fixture streaming result recorded in the model registry

- Ran `TestNativeStreamingModelSmoke` and `TestNativeStreamingWebSocketSmoke`
  with the pinned Zipformer 2023 runtime, punctuation model, and bundled
  `quality_speech.wav` fixture. Both returned `The quick brown, fox jumps.`;
  lexical WER is 0.00 against the five-word fixture reference. The warm native
  model test loaded in 3.04 s and inferred 1.955 s of audio in 147 ms (RTF
  13.29x). Exact full-text and final-suffix assertions were enabled for this
  run. The test asserted the full expected phrase was present and the expected
  final suffix was exact. This is one short clean fixture on one host; it does
  not qualify JFK, accented/noisy/long-form speech, native word timing, or
  supported product deployments.
- Added this result to the Sherpa Zipformer `benchmarks.notes` declaration and
  regenerated the model reference. `make models-doc` and `make catalog-check`
  pass. Existing JFK recognition counterexamples remain in the model note.
- A paired transcription through the product CLI did not yield a result in
  this agent run: the Agent Manager identity broker returned EOF, and the CLI
  warned that grant-backed operations would be refused. No transcript was
  written, and I did not route around the authorization boundary. The native
  server tests above do not imply current product API authorization or
  serviceability.

### 2026-11-22 — TTS cache speed-key collision fixed

- Cache-key speed encoding now uses a shortest round-trip float representation
  instead of rounding to four decimal places. A regression proves speeds
  `1.00001` and `1.00002` remain separate cache entries. The entire
  `internal/tts` package passes under the Go race detector.
- Cache reads now promote hits to most-recently-used order. The strengthened
  eviction regression reads the older entry, then proves the untouched entry
  is evicted when capacity is needed.
- Corrected the TTS cache invariant and proto comments: this is an event-keyed
  cache whose callers must keep event/version content snapshots immutable;
  `content_hash` remains correlation-only. The proto owner regenerated the
  audio-tools projection, and `make -C packages/proto verify-committed-gen`
  passes. This preserves current event replay behavior while removing a
  demonstrated wrong-speed cache hit.
- OpenAI GPT-Live credential status remains encrypted-file provider available
  and unconfigured. The Platform key-creation chooser returned no confirmed
  route in this session; owners can securely enter a key with `vrooli
  credentials provision --identity vrooli/openai-gpt-live --field api-key`.
  No key or live API request was created.
- Rechecked the no-pin STT path: the live engine resolver uses the declarative
  low-footprint ranking to select Sherpa when serviceable, and the source test
  `TestShippedPolicyPrefersSherpaOverKyutai` confirms that policy. The full
  `sttengine`, `stt`, and streaming-segmenter packages pass under `-race`.
  The current product instance still carries an operator engine pin, so this
  is policy/source evidence rather than a fresh live no-pin selection receipt.

### 2026-11-22 — Audio Tools summarize role clarified

- Confirmed the product's default role is `summarize.audio_tools` → Qwen3 1.7B,
  with Qwen3.5 4B as its quality fallback. The generic shared
  `summarize.default` remains Qwen3.5 4B because it shares a runner with
  `chat.small` and `classify.routing`; this preserves the operator's compact
  Audio Tools route without increasing model residency for other consumers.
- Updated the current-state scoreboard, maturation policy, and Ollama role
  reference to distinguish those contracts. The focused Audio Tools model
  catalog/summarize/chain suites and resource-ollama policy/policy-command
  suites pass.
- Fresh CLI resolver attempts could not rebuild their wrappers because the
  shared tree currently fails at `cliutil/agentlauncher.go:360` with an unused
  `environment` local. No live summary request was made; the earlier runtime
  receipts remain the latest product-path evidence.

### 2026-11-22 — Current product build and shared-host footprint rechecked

- The live Audio Tools API binary was older than the current model catalog and
  returned Qwen3.5 4B for summarize. Rebuilt with the scenario's `make build`
  target and restarted through `make restart`; the managed operation completed
  healthy with build identity `sha256:f511d3d7d7c884ba00f86ea9cfffc0c3045c730e9aabc1290365a0573c862b42`.
  Live `audio-tools model resolve` now returns Whisper Small (488 MB) for batch
  STT, Qwen3 1.7B (1.36 GB) for Audio Tools summarize, and `stt stream-config`
  selects Sherpa Zipformer. TTS model resolution names qualified Kokoro (330
  MB) while its conditional auto route selects Kale Nano when the healthy
  candidate is available.
- A short live product TTS request completed through `local/kale-tts` with
  `kale-tts-1-nano`: 6.3 s on the first request and 326 ms warm for the same
  prompt/voice, returning 73,244 WAV bytes. This checks route selection and
  warm latency only; it does not qualify voice quality or a broad latency
  cohort.
- A live Audio Tools `summarize text` request returned a local Ollama summary
  in 4.13 s through Qwen3 1.7B. The Ollama retention snapshot before this
  request showed the 1.7B model unloaded; the post-request snapshot showed it
  loaded. This is expected on-demand behavior under the resource's idle-unload
  policy, not evidence of permanent Audio Tools residency.
- Current `nvidia-smi` showed 7,662 MiB used of 16,376 MiB. Read-only capacity
  attribution measured 5.54 GiB across Ollama workers and 1.35 GiB for the
  reranker; Ollama's loaded Qwen3.5 4B is shared by `chat.small`,
  `classify.routing`, `rerank.llm_fallback`, `summarize.default`, and the
  explicit Audio Tools quality role. Qwen3 1.7B is declared for Audio Tools
  summarize but was not loaded in the Ollama retention snapshot. The Ollama
  caller ledger later showed 36 Qwen3 1.7B requests: 2 attributed to the
  `ai-gateway-start-api` resource-ollama process and 34 unattributed. Audio
  Tools reaches resource-ollama through that gateway path; direct Audio Tools
  identity is therefore not present in the model caller ledger.
- After the compact summary request, GPU use was 14,955 MiB of 16,376 MiB.
  Capacity reconciliation attributed 6.97 GiB to Ollama and 1.35 GiB to the
  reranker, while a separate `ollaya` process used 5.88 GiB without a unique
  capacity claim; the process inventory's owner attribution remains incomplete.
  The Ollaya process is outside Audio Tools. This host snapshot does not
  justify changing or stopping another resource, but it identifies a larger
  attribution gap than the Audio Tools model choice.
- Whisper is stopped and its 1 GiB claim is `cold_idle`; the owner-scoped,
  read-only `vrooli capacity residency open --owner whisper --json` now reveals
  its exact unresolved `ggml-small.bin` unload transition. No Whisper cache
  data was removed: both Medium and Turbo previews remain ineligible until
  their declared 30-day idle retention elapses. The stop transition has no
  verified placement event, and the current GPU inventory is incomplete, so
  no stop-only recovery or claim release was performed.
- The production model-promotion owner audit confirmed corpus-quality and
  latency have integrated resolvers. Memory, platform, and license still lack
  qualified receipt owners; adjacent host-portability, resource-support, and
  rights-decision data do not satisfy those model-bound promotion contracts.
  `go test ./internal/modelpromotion ./handlers/models` passed after adding a
  production-owner-set regression and documenting those evidence boundaries.

### 2026-11-22 — OpenRouter BYOK boundary and maintainability follow-up

- Added a resource-owned `resource-openrouter generate --byok-json-stdin`
  contract for one-shot BYOK calls. The typed request is size-bounded, rejects
  malformed and ambiguous role/model selections, and keeps the API key out of
  argv, environment, persistent state, and generic error output. Audio Tools
  now sends this request through stdin; `ai.invalid_provider_secret_env` no
  longer appears in the targeted AI conformance phase. The request still
  contains the key in transient stdin/memory by design; it is never logged.
- Validation: `make -C resources/openrouter test`; Audio Tools
  `go test -race ./handlers/experiment ./internal/eval ./internal/sttengine
  ./internal/byok -count=1`; `ai-gateway validation validate` with the explicit Audio Tools path passed. Test Genie run `20261122-075252-5f6a475b` passed `ai-conformance`;
  `tidiness` remains failed because complexity-over-threshold is 213 vs the
  baseline 148 (+65). The earlier `20261122-075125-ce96fd9a` tidiness scan
  showed long files improved from 12 to 10 after cohesive file extractions.
- Extracted experiment admission estimators, evaluation report explanations,
  and STT-engine capability projections into focused files without changing
  runtime behavior. Focused package tests and race checks passed. The long-file
  count is still one above its baseline, and complexity debt remains open.
- The broader `20261122-072139-31378112` run remains the latest full-suite
  evidence: 23 passed, 7 failed, 1 skipped. Experience-contract still reports
  that Experience Manager returns `assessment.phase="experience"` for the
  `experience-contract` request; its owner restart was in progress during the
  check. A direct post-restart reproduction confirms the same mismatch;
  published bug report `knw-1795334159222034566`. The investigator heartbeat
  could not be triggered because scenario-qa heartbeats are globally paused
  until infrastructure fixes are complete. Workflow admission still lacks an
  authenticated BAS account.
- Test Genie docs rerun `20261122-075521-5de8cdb3` passes after removing an
  absolute workspace path from this progress entry. It still reports 496
  documentation warnings and 30 infos.
- After updating the provider-defect report reference, the docs phase passed
  again as `20261122-075648-d734ffce` (497 warnings, 30 infos; no errors).
- Current-tree Test Genie unit run `20261122-075717-8ca29700` passed at L2 in
  112 seconds; 60 low-coverage advisories remain. This supersedes the earlier
  unit receipt for the current changes.

### 2026-11-22 — Sherpa model grouping and AMI inventory correction

- Corrected the Sherpa resource manifest so Spleeter source-separation weights
  are in their own on-demand group instead of being acquired with
  speaker-verification weights. Its URL, checksum, and license declaration are
  unchanged. `make check` in `resources/sherpa-onnx` and
  `vrooli resource validate sherpa-onnx` passed.
- Reconciled the timing-corpus guide with the external qualification cache. The
  cache contains AMI ES2002a Mix-Headset (1,272.64 seconds, mono 16 kHz) plus
  public manual annotations v1.6.2; `acquisition.json` records CC BY 4.0,
  source URLs, hashes, and `not_human_reviewed` status. These forced-aligned
  boundaries are not eligible as timing gold, and the source files remain
  outside Git and have not been imported into the lab corpus. The runbook now
  calls for two additional meetings before review can meet its three-stream
  tranche.
- Current product-path TTS check: `audio-tools health show --json` reported
  Kale available and Kokoro unavailable; `audio-tools tts synthesize` returned
  `provider_id=kale-tts`, `model_id=kale-tts-1-nano`, 90,044 WAV bytes, and
  1,393 ms latency for a short utterance. The route selected Kale automatically
  while healthy; this confirms routing and output validity, not voice quality.
- Reduced complexity in two more production seams: CLI `buildStartRequest`
  went from gocyclo 73 to 4 (largest extracted helper 14), and qualification
  `EvaluatePromotionMeasurements`/`Evaluate` now score 1 each with all rubric
  helpers at or below 15. Focused CLI experiment/STT tests and qualification
  tests passed; scoped diff checks are clean.
- CLI streaming now exits non-zero when the final event reports a failed
  terminal reason; `go test ./domains/stt` passed. A fresh replay against the
  lab API emitted 57 segments before failing at client chunk 560 (`write
  envelope: EOF`), after 37 seconds including CLI auto-build. The final event
  reported `input_failed`, received sequence 527, processed sequence -1. The
  updated CLI propagates this send failure without printing audio or transcript
  content. Connect's configured 16 MiB read cap is per framed message, not
  cumulative, and does not explain this cutoff. The payload-free API pump
  diagnostic still does not appear in either lab log stream, so the server-side
  receive failure stage remains unknown. A temporary local h2c harness using
  the current server timeout configuration held a Connect stream idle for 32
  seconds and then ended it successfully; it rules out a generic 30-second
  handler/transport idle deadline but not the AMI audio-processing path. The
  temporary harness was removed and did not touch either running API.
- Completed a transcript-free Sherpa Zipformer evaluation on all 360 held-out
  clean VCTK clips in the lab corpus (experiment
  `296c84f3-5204-4d30-b698-5cb7a6dd85f0`, `passthrough`, one repeat). The
  native passthrough quality lane is paced at real time even though the recipe
  lane is `deterministic`; it took about 27 minutes as predicted by the
  1,629-second dry-run. The result measured 4.8464% WER over 3,776 normalized
  reference words (137 substitutions, 17 deletions, 29 insertions), no clip
  errors, and passed stream safety with a maximum dropped span of one word.
  Latency was not measured and RTF is unavailable. The report has low
  confidence and remains non-promotable: the VCTK scripts still need human
  audit, only clean audio was evaluated, and interval accounting, bounded
  recovery, duration/fault profiles, and browser/device evidence are missing.
  Sherpa's attached service was about 347 MiB RSS. A transient CLI refresh was
  denied when host memory PSI reached 21.04% vs a 20% admission limit; pressure
  later cleared and the persisted experiment completed successfully.
- Added a profile-specific health refresh when a running scenario is reused,
  with focused lifecycle/profile tests passing. Live verification refreshed
  `audio-tools@lab` (API 15373); the first start rebuilt/restarted only lab
  because its build identity was stale. A second start returned
  `already_running=true`, but the subsequent `vrooli scenario status` still
  reported `degraded` with `ui_endpoint: unresolved placeholder UI_PORT`.
  Root cause: the runtime supervisor is a second snapshot writer and its
  periodic probe loaded the base manifest instead of the instance profile,
  overwriting the corrected lab snapshot. The supervisor probe now selects
  `instance.Variant`; `TestSupervisorHealthProbeUsesInstanceProfile`, the
  focused test, and the full `internal/runtimesupervisor` suite pass. Live
  status remains degraded because the active supervisor is running the
  previous executable. No lab-scoped reload exists; the supported install
  restarts the shared supervisor that currently manages 91 instances, so it
  was not invoked. Product (`audio-tools`, API 19630) remained healthy and
  was not restarted.
- GPT-Live provisioning is wired to the encrypted-file credential authority:
  `vrooli/openai-gpt-live` / `api-key` is declared as optional and the lazy
  in-process resolver reads it on each session open, so rotation takes effect
  without restart. The provider remains unconfigured, and session mounting is
  still gated by duplex and billing evidence. Key creation is not available
  in this session; the owner can create one at OpenAI Platform and enter it
  with `vrooli credentials provision --identity vrooli/openai-gpt-live --field
  api-key` (value via secure prompt/stdin, never argv or chat). No API call was
  made.

### 2026-11-22 — Voice-cloning footprint and license scan

- Rechecked primary model sources against the current Audio Tools inventory.
  Reverb is the smallest inventoried cloning candidate (297M parameters,
  about 307 MB int8, CPU-capable), but its weights are CC BY-NC 4.0 and are
  ineligible for a commercial default. Qwen3-TTS 0.6B Base supports reference
  cloning and declares Apache-2.0, but its pinned repository is 2.52 GB with a
  1.83 GB main model file; Audio Tools has no Base runtime, adapter, or measured
  cloning receipt. OpenVoice V2 declares MIT but requires a multi-model
  MeloTTS stack whose full pinned artifact/license inventory is unresolved; the
  upstream checkpoint URL returned 404 during the source check. No weights
  were downloaded or service started. Voice cloning remains open pending a
  bounded, consented, ephemeral-reference runtime and held-out quality/resource
  measurements.

### 2026-11-22 — STT boundary and TTS readiness declarations

- Retired the unused settings/wake-word/speaker persistence façade from the
  Whisper batch transcription service and renamed the remaining boundary to
  `WhisperTranscriber`. The active STT provider retains bounded concurrency,
  availability checks, and recovery behavior; the service no longer owns
  duplicate settings state.
- TTS capability, health, readiness, and aggregate registrations now iterate
  provider runtime declarations. A synthetic provider test verifies that a
  declared runtime registers without a provider-name branch. Shared Sherpa,
  Kokoro synthesis readiness, and Kale explicit/automatic selection remain
  covered by focused tests.
- Current-tree verification passed: `go test ./...`, `go vet ./...`,
  `make catalog-check`, and `make portability` (windows/amd64 and
  darwin/arm64). Focused STT race tests also passed before the final type rename;
  focused non-race packages passed after it. Model defaults remain unchanged
  pending qualification evidence.
- Credential provisioning remains ready through the declared encrypted
  authority: `vrooli credentials provision --identity vrooli/openai-gpt-live
  --field api-key`. No key was created or API call made in this session.

### 2026-11-22 — Duplex recovery retries and host evidence diagnostics

- Made duplex SQL journal creation idempotent for a retry with the same
  immutable owner/provider/model/budget identity. Retries preserve the current
  phase; attempts to rebind an operation are rejected. LPBS remains the
  authoritative durable balance/hold/settlement ledger, while Audio Tools
  persists bounded recovery metadata. Restart uncertainty, owner-scoped
  reconciliation, and single-winner settlement remain covered.
- D1 accelerator diagnostics now include the unavailable Kyutai runtime reason
  and any observed Sherpa fallback while keeping inference `not_run` and
  qualification false. The matrix remains 0/18: service health/inventory alone
  cannot satisfy browser/device product-path evidence.
- Ran the bounded host-portability receipt on lab, artifact
  `audio-tools:host-portability:20261122T145658.206802579Z:amd64`. D1's
  digest-pinned CPU Whisper Small container returned the exact bundled fixture
  transcript in 4.99 s (8 CPU / 8 GiB cap). D2's network-isolated Sherpa
  Zipformer int8 container passed its native smoke under 2 CPU / 4 GiB; load
  was 1.02 s, inference 112 ms for 1.955 s audio, and measured peak memory was
  441,339,904 bytes. Model, runtime, fixture, and test-binary hashes are in the
  lab receipt. Both are inference-smoke evidence only; D1 Kyutai is unavailable,
  Whisper's managed HTTP service is unavailable, browser/device execution was
  not run, and the overall receipt correctly remains `passed:false`.
- Re-verified the full current source tree: API build, vet, full suite, and full
  `go test -race -count=1 -timeout=300s ./...`; CLI build, vet, and full suite;
  UI type-check, lint, and 135-file/1,279-test suite; `make catalog-check`,
  `make lab-test`, and Windows/amd64 + Darwin/arm64 `make portability`. Resource
  validation passed for Sherpa ONNX, Whisper, and Kale TTS. Touched-source
  `git diff --check` is clean apart from existing WAV fixture CRLF warnings.
- A live point-in-time resource snapshot showed product/lab Audio Tools API
  processes at 24.1/24.5 MiB RSS. GPU use was 15,242/16,376 MiB at 4%; the
  compute processes were Ollama workers and the separate reranker. Ollama's
  resident list contained Nomic embeddings, Qwen3.5 4B, and Gemma4 12B, but not
  Audio Tools' application-scoped Qwen3 1.7B. Sherpa reported its streaming
  model loaded and TTS model unloaded; Kale Nano reported loaded/ready. This is
  a host snapshot, not complete process-level attribution or a model-quality
  receipt.
- Still open: the three-recording independently human-reviewed long-form
  timing tranche, real browser/device product-path cells, native signed and
  published Sherpa lifecycle artifact, Kale joined quality/release
  qualification, and provider-backed GPT-Live admission with production policy.

### 2026-11-22 — Fail-closed full-duplex product composition

- Added a production server-composition seam for the GPT-Live WebSocket route.
  It mounts only when the durable journal, LPBS account-credit client,
  source-bound consent verifier, positive session cap, and explicit provider/model
  model eligibility callback are present and admit `openai-gpt-live` /
  `gpt-live-1.1`. Tests prove absence by default, successful composition with
  explicit dependencies, and route absence when eligibility denies the pair.
- Production bootstrap continues to pass no duplex admission dependencies, so
  the route remains unmounted. Current GPT-Live provider/model readiness is
  still planned/candidate, and account-credit service-principal credential
  resolution plus consent endpoint configuration remain unwired. No OpenAI
  request or key provisioning occurred.
- Updated `MATURATION-STATE.md` and Phase 5 of `MATURATION-PROGRAM.md` to
  distinguish implemented transport / consent / LPBS recovery components from
  missing production composition. Full API `go test -count=1 ./...` and
  `go vet ./...` pass after the composition change; focused server/duplex race
  tests also pass.

### 2026-11-22 — Engine-scoped model resolver and live lab proof

- Added declarative STT defaults scoped by engine. The generic capability
  default remains multilingual Whisper Small; Sherpa Zipformer is explicitly
  selectable for `sherpa-streaming`, without engine substitution. The model
  record, API, CLI, UI, generated docs, and catalog validation expose the scope.
- Restarted only `audio-tools@lab` through the supported lifecycle after its
  deploy report identified the working-tree build inputs. The replacement is
  healthy on API port 15373. The live lab CLI resolves
  `sherpa-streaming` + English to `sherpa-zipformer-en-2023-06-26`, with the
  engine declaration visible; an unscoped STT resolve still selects
  `whisper-small`. Product was not restarted.
- Verification: focused API/modelcatalog tests and race tests, model-domain Go
  tests and vet, UI model tests/type-check/lint, catalog check, committed-gen
  verifier, portability builds, and scoped `git diff --check` passed. The
  default `audio-tools` CLI targets live; use `audio-tools --instance lab` to
  query the isolated lab build.
- GPT-Live remains unmounted because `vrooli/openai-gpt-live/api-key` is
  unconfigured and production admission prerequisites are absent. The lab
  startup confirms credential provisioning remains available through
  `vrooli credentials provision --identity vrooli/openai-gpt-live --field
  api-key`; Kyutai STT also remains an optional missing credential.
- Still open: independent human timing review for three long recordings,
  browser/device product-path evidence, managed signed Sherpa publication,
  Kale Nano quality review, cloning consent/runtime/quality qualification, and
  provider-backed GPT-Live admission.

### 2026-11-22 — Live compact-route check and AMI review packets

- Verified the live product stream configuration pins `sherpa-streaming`, and
  `stt engines --json` selects Sherpa because the operator pin is serviceable.
  The live product's optional engine-scoped `model resolve` request still
  returns the generic Whisper Small record because that product process
  predates this resolver update; the isolated lab accepts the scope and
  returns the declared Sherpa Zipformer record. The active stream route itself
  remains Sherpa. Product was not restarted.
  The read-only product deploy report shows 111 changed API source files and
  an unrecorded UI build manifest, so the current shared-tree candidate is not
  a narrow, isolated product replacement. The isolated lab API remains the
  verified source build; product stays on its previous healthy build pending
  a coherent release boundary.
  The live product TTS CLI synthesized a valid mono 24 kHz WAV through
  `kale-tts` / `kale-tts-1-nano` in 587 ms. This verifies the conditional
  product route, not naturalness or default promotion. `model resolve` still
  reports Kokoro as the quality-qualified catalog default; the product auto
  route prefers healthy Kale Nano.
- The Audio Tools summarize resolver returns Qwen3 1.7B. A current host snapshot
  showed 15,238 / 16,376 MiB GPU memory used; the Ollama residency endpoint
  listed Qwen3.5 4B (~4.27 GB VRAM), Gemma4 12B (~7.46 GB), and Nomic embeddings
  (~0.32 GB), but not Qwen3 1.7B. `nvidia-smi` attributed the visible GPU
  workers to three Ollama `llama-server` processes and a separate reranker
  (1,380 MiB); Audio Tools product/lab APIs used 29/30 MiB RSS. Attribution of
  shared Ollama residents to a caller is unavailable. No shared model was
  unloaded.
- Prepared three AMI review packets under
  `~/.local/share/vrooli/audio-tools/qualification/ami-en-v1/review-packets/`
  for ES2002a, IS1001a, and ES2008a. Each contains source audio, four speaker
  annotation XMLs, aligner-only draft transcript and timings, a blank ledger,
  attribution/instructions, and checksums. All three streams exceed ten
  minutes; packet checksums were independently verified. ES2008a has three
  untimed draft words to correct. No human-review status was asserted; a named
  reviewer and adjudicator must complete the full listens before import.
- A full-duplex HTTP disconnect regression was already present in the shared
  tree; the focused race test
  `TestWebSocketClosesReaderAfterTerminalProviderFailure` passed. Provider
  health checker composition now uses a catalog `health_profile` and declared
  readiness/liveness aggregate membership instead of provider-ID branches.
  Catalog validation rejects unknown profiles and incoherent aggregate flags;
  a test proves a synthetic provider can reuse an existing profile without an
  adapter edit. Whisper, Ollama, Sherpa, request-scoped OpenRouter, and ffmpeg
  checker behavior is preserved. `go test ./internal/providercatalog
  ./internal/bootstrap -count=1` and `go vet` for both packages pass.

### 2026-11-22 — Retired stale STT pipeline debt record

- The old `PROBLEMS.md` entry described an active `voice.Service` shim and a
  24% package coverage figure. Current source has no `voice.Service` callers;
  the Connect streaming path uses the strategy/chain/segmenter boundary, and
  `pipeline` owns local Whisper batch transcription/recovery as
  `WhisperTranscriber`.
- `go test ./internal/stt/pipeline -coverprofile=/tmp/audio-tools-stt-pipeline.cover`
  initially confirmed 71.2% statement coverage, then focused tests covered the
  environment toggle, logger injection, Whisper availability/recovery, and
  queued-cancellation paths. The current run passes at 72.9%; all five
  targeted methods are at 100%, and `go test -race ./internal/stt/pipeline`
  passes. `PROBLEMS.md` now lists the remaining speaker-client and configuration
  test gaps. No production code was deleted.
- A subsequent `make -C scenarios/audio-tools catalog-check` caught a source
  scan that required literal provider IDs in bootstrap. The gate now checks
  health-profile/TTS-runtime/no-checker declarations, while bootstrap tests
  exercise profile construction and aggregate membership. Re-running
  `make -C scenarios/audio-tools catalog-check` passed. The gate still rejects
  an active local provider with no declared health contract.

### 2026-11-22 — Refreshed Sopro cloning candidate facts

- Rechecked the current upstream sources. Sopro's repository says current model
  artifacts require `sopro>=2.2.0`; its model card reports 120M parameters,
  Apache-2.0 weights, four languages, CPU/browser paths, and reference cloning
  from 5–20 seconds. The current Hugging Face snapshot revision is `c12d0dc`
  and its 952 MB tree includes both a 486 MB Safetensors checkpoint and an
  alternative 318 MB GGUF plus encoders/vocoder. This is not an Audio Tools
  runtime pin or proof of a minimal deploy bundle.
- Sopro remains a non-default candidate. Audio Tools still lacks a pinned
  runtime/dependency license inventory, consented transient-reference flow,
  and local quality/latency/memory measurements. Upstream states it has no
  watermark and warns against impersonation, reinforcing the need for
  source-bound authorization in the product operation.

### 2026-11-22 — Bounded model routes, stream finalization, and resource attribution

- Revalidated the product choices against current declarations and live lab
  behavior: batch STT resolves to Whisper Small; the active stream engine is
  Sherpa Zipformer; `summarize.audio_tools` resolves to Qwen3 1.7B; and a short
  lab product request selected Kale Nano (`kale-tts-1-nano`, 629 ms). Kale is
  still a candidate: this smoke does not establish naturalness or promotion.
  Kokoro remains the qualified TTS catalog default, while `auto` tries healthy
  Kale first.
- The current GPU snapshot was 15,250/16,376 MiB used. `/api/ps` attributed
  model residency to Nomic embeddings (308 MiB), Qwen3.5 4B (4,074 MiB), and
  Gemma4 12B (7,114 MiB); `nvidia-smi` showed three Ollama runners plus a
  separate 1,380 MiB reranker. Audio Tools product/lab APIs were 22/26 MiB RSS;
  Sherpa and Kale were CPU processes, and no Audio Tools process appeared in
  the GPU process list. Whisper is stopped/unavailable, avoiding idle GPU
  residency. No shared model was unloaded. The 4B and 12B roles/callers are
  shared-host consumers, not Audio Tools' summarize role.
- Reproduced and fixed a real Sherpa Zipformer/Kroko end-of-input defect. A
  bounded silence tail now drains the online decoder before `InputFinished`,
  capped at five seconds and defaulting to 1.2 seconds. A synthetic decoder
  regression covers zero-tail suffix loss, positive-tail flush ordering, and
  the cap. With the Kroko JFK fixture, the native and WebSocket paths both
  return the complete final sentence at the default tail; setting the tail to
  zero reproduces `... As.` and fails the mandatory expected-suffix assertion.
  The fix does not add silence to client timestamp accounting. Apache Zipformer
  also dropped a final `s` with zero tail. The model-backed smoke remains
  opt-in because resource weights/audio fixtures are not repository assets.
- Tightened the timestamp scorer to enforce the existing 20,000-word-per-side
  cap before normalization and before the identical-stream fast path. Tests
  cover oversized input at both public and helper boundaries. Synthetic
  Linux/amd64 scorer measurements: 10k words 4.42 ms / 2.70 MB; 20k identity
  5.64 ms / 5.23 MB. These are evaluator overhead figures, not model latency or
  qualification evidence.
- Sherpa release-stage and packager validation now require absolute HTTPS
  publication authorities with valid host/port and no credentials, query, or
  fragment. Release lifecycle stays fail-closed pending owner provenance,
  signing authority, and publication.
- Verification: API race suite and API/CLI vet plus CLI tests passed in the
  prior checkpoint; current `go test -race ./internal/eval`, Sherpa `make
  check`, `vrooli resource validate sherpa-onnx`, generated catalog check,
  cgo-tagged Sherpa race tests, native model/WebSocket smoke, and scoped
  `git diff --check` passed. A separate no-tail native run failed as expected,
  confirming the regression is exercised. No commits were made.
- Qualification is still open: the live lab matrix reports 0/27 qualified
  cells, and `validation check-freshness` fails for stale/missing ATD-P0-001
  out-of-band TTS and dictation runs. No human-reviewed references or new
  acceptance claims were added. The hour-scale qualification was not started
  while the host was under substantial shared load.

### 2026-11-22 — Long qualification harness reliability fixes

- The streaming STT CLI client no longer inherits the ordinary 120-second
  `http.Client.Timeout`; the qualification context still bounds the run and
  user cancellation. Focused STT and validation race tests pass.
- Repeated qualification runs now use fresh cryptographically random durable
  session IDs. A retry exposed that the prior fixed ID collided with retained
  ledger sequence state and terminated early. A regression verifies IDs differ.
- Two lab retries uncovered distinct issues: fixed durable IDs caused ledger
  conflicts, and after 35 minutes a tone-only stream hit the server's 64 MiB
  retained-audio cap because it produced no committed speech. The harness now
  rotates and cleanly finalizes a session every 16 MiB. Focused race tests pass;
  a subsequent 60-minute lab run completed successfully. It recorded 360 TTS
  probes / 69,999,840 bytes and 35,994 STT chunks / 115,180,800 bytes across
  bounded stream sessions; `validation check-freshness --json` reports fresh.
  Evidence is `/tmp/audio-tools-expensive-validation-20261122.json`.
- GPT-Live credential state: encrypted-file provider is available, identity
  `vrooli/openai-gpt-live` / `api-key` is unconfigured. The supported secure
  terminal provisioning path is present; `vrooli credentials doctor
  --check-writes` confirms encrypted-store write/remove works. No OpenAI key
  value was available to enter, and no OpenAI API call was made.
- The sherpa-onnx resource-level TTS route was also smoke-tested using the
  existing local Kokoro int8 artifact on a temporary localhost port: health
  reported TTS loaded, the endpoint returned an `af_heart` WAV (24 kHz mono,
  2.09 s), and the temporary server was stopped. This confirms native asset
  serviceability only; managed lifecycle, product runtime selection, and voice
  quality qualification remain separate gates.
- The governed `audio-tools.setpoint-read` program still returns `partial`:
  three diagnostics are readable, but its product-bound experiment evidence
  binding returns `remote_error`, leaving 0/18 required owner receipts joined.
  The separate lab CLI can read 14 persisted evidence rows, including negative
  browser-product and corpus-quality results; this does not authorize pointing
  Program Runtime at an arbitrary lab URL or promote those rows.

| 2026-11-22 | Codex | implemented (metrics-only persistence view) | Added an explicit `EvalReport.AsMetricsOnly()` export schema containing aggregate/per-clip WER, latency, timestamp-drift metrics, route identity, and timing-review provenance. The schema has no transcript, clip-ID, arbitrary diagnostic, or word-timing fields; existing `EvalReport` serialization remains unchanged. A sentinel regression marshals a report containing reference/hypothesis text, normalized text, edit tokens, commit text, and safety-event text and confirms none is serialized while numeric metrics and human-review provenance survive. `go test ./internal/eval` passes. This is the safe export primitive, not yet wired to a recipe declaration or experiment blob-store write; current experiment persistence/report consumers retain their existing semantics. No STT model workload or private audio was run. |

| 2026-11-22 | Codex | verified (declaration-backed metrics-only experiment receipts) | Added `ExperimentRecipe.report_retention_mode` with legacy-compatible unspecified/full behavior and explicit metrics-only selection in Dictation Studio; CLI recipe JSON/file docs expose the same declaration. The experiment persistence branch keeps the `EvalReport` protobuf shape and labels transcript-free receipts with `retention_mode=metrics_only`; its field allowlist retains quality/latency/timestamp metrics, route identity, and timing-review provenance while removing transcript, clip-ID, label, free-form diagnostic, and safety/commit text. Per-run condition rows are similarly reduced. The scope is the report and condition artifacts: queued/running and failed experiment records retain the submitted recipe (including configured text) for execution/recovery, while successful metrics-only completion omits only the server-generated long-form reference from its completed recipe snapshot. A manager/repository/blob lifecycle regression checks submitted and completed recipe fields, the persisted report blob, per-run condition rows, and the no-report failure path. Default mode still retains the full historical report. Focused API (`handlers/eval`, eval, experiment, experiment/report, bootstrap), CLI experiment tests, Dictation Studio service/builder tests (12), UI type-check, selector check, and scoped ESLint pass; the metrics-only checkbox has an explicit accessible label association. `make -C packages/proto verify-committed-gen` passes. The generation session lacked a pre-run snapshot, so the broad generated-output and pnpm-lock status diffs remain unattributed and untouched. No model workload or private audio was used. |

### 2026-11-23 — Local qualification harness and residency reconciliation

- Fixed the bounded D2 host-smoke producer to pass the Sherpa stream's required
  final-suffix expectation in addition to expected text. A focused regression
  checks the producer arguments. The new receipt
  `audio-tools:host-portability:20261123T004023.567768581Z:amd64` records
  successful D1 CPU Whisper and D2 CPU Sherpa inference (2.671 s and 1.261 s
  model load + 100.627 ms inference for 1.955 s audio, respectively); its
  overall `passed:false` is preserved because D1 Kyutai and other required
  cells remain unavailable. The device matrix remains 0/27 qualified.
- Managed-service readiness now reconciles residency policy for already-loaded
  Ollama models after health and placement checks, before resuming the private
  activity edge. The edge skips reconciliation while a request is in flight to
  avoid a wake/start deadlock; it never loads an absent model. Focused race
  tests pass for the activity edge and runtime-profile reconciler, and the
  Ollama CLI test passes. `internal/resources` package tests remain blocked at
  compile time by unrelated undefined health-check symbols in the dirty shared
  tree; no files were reset or altered to attribute that failure.
- The consent verifier redirect replay issue is recorded below in PROBLEMS.md.
  No hosted inference, key creation, or live device qualification occurred in
  this continuation. Audio Tools was rebuilt and restarted through its Make
  lifecycle while checking an initial 404; a variant-scoped follow-up showed
  qualification evidence is mounted only in the lab instance by design.

### 2026-11-23 — Target-aware receipt identity repair

- The setpoint board had inspected persisted lab receipts but resolved 0/18
  required rows. Its strict resolver requires the top-level `device_profile` to
  equal `provider_cell.device_profile`; the simulated and composite receipt
  producers omitted the latter, and the Python test harness silently supplied
  it. Removed that test-only normalization and added the missing identity to
  simulated, composite, speaker-policy, and host-portability receipts.
- Re-ran `audio-tools --instance lab validation run-simulated --json`; the new
  transcript-free receipt is
  `audio-tools:simulated-qualification:20261123T010244.539762505Z`. The
  independent Program Runtime board now reads the owner endpoint cleanly and
  resolves exactly the five policy/adapter rows exercised by the simulation;
  remaining required rows stay pending and overall acceptance remains unknown.
- Regression coverage passes: 37 Python board tests no longer repair input
  receipts, and the focused Go validation suite checks producer identity for
  both simulated and composite envelopes. This validates simulation
  provenance only; it does not establish hosted billing, measured recognition,
  latency, or device qualification.

### 2026-11-22 — Dictation sample-aligned latency and backlog diagnostics

The shared browser stream recorder now uses monotonic timestamps for in-page
latency, backdates signal-threshold observation by the voiced sample offset
inside the capture block, leaves speech latency unknown when no signal was
observed, and measures committed backlog from captured versus committed sample
positions. Connect stream adapters carry the safe committed end-sample cursor
through Audio Tools, Web Console, and Swarm Manager. The live STT resolver runs
independent inventory probes concurrently and uses a bounded provider-liveness
check while one coalesced asynchronous refresh updates expired inventory.

`pnpm build` and the focused shared-browser suite pass; the suite includes the
accelerated 60-minute recovery harness. `GOWORK=off go test -race
./internal/sttengine -count=1` passes. Web Console's codec suite passes with the
end-sample assertion. Audio Tools and Swarm Manager still have isolated cached
`@vrooli/audio-capture-browser` artifacts: Audio Tools' runtime copy lacks
`safeSampleNumber`, and Swarm Manager's copy imports missing
`voice/rejectionController`. `vrooli package refresh audio-capture-browser`
built the shared package but deferred setup while consumers were running.
Audio Tools Test Genie run `20261122-233933-b6309d20` passed API and failed unit
(boundary allowlist plus eval coverage floor) and performance (stale consumer
build); package Test Genie run `20261122-233736-8d34723c` reported standing
`COVERAGE_CONFIG_MISSING` and `LONG_FILE` maturity findings. These are not
product-path qualification results. Browser paced cohorts remain unavailable
because managed browser authorization was denied before execution; no native
latency band is claimed as met. The GPT-Live encrypted credential path remains
available and write-tested, but the API key is still unconfigured and no hosted
call was made.

### 2026-11-23 — Replay cursor integrity and broad unit/performance validation

- `TurnJournal.acknowledgeProcessed` now rejects negative and future cursors
  before persisting an acknowledgement or compacting captured PCM. Focused
  tests prove an invalid cursor preserves the retained chunk and a valid
  inclusive acknowledgement still compacts it. The shared capture package
  passes 257 tests with one skipped, and its TypeScript build passes.
- Repaired Audio Tools test fixtures for the stream-final `endSample` argument
  and complete `VoiceStreamStart` contract. Added direct metrics-only report
  filtering coverage and an explicit bootstrap report-persistence import to the
  lab composition boundary. API eval handler coverage is now 90.9% against its
  87% floor; the CLI primitive-evidence artifact was regenerated from its
  canonical manifest.
- Test Genie run `20261123-011407-22ecc196` passes both `unit` and
  `performance`. It retains 60 advisory low-coverage warnings across the wider
  scenario; those remain separate follow-up work. The earlier run
  `20261123-010809-fa31be09` exposed the corrected fixture, retention coverage,
  boundary, and generated-evidence failures.
- Current GPU process inventory attributes the observed 15.2 GiB use to three
  shared Ollama processes and the reranker; no Whisper process appeared in the
  snapshot. Registry defaults remain Whisper small, Kokoro 82M, the 27M speaker
  embedding, and Audio Tools' 1.7B summarizer. This point-in-time inventory
  does not establish all Audio Tools resource use or change shared residency.
- Warm, expired-discovery, and cold-engine browser cohorts still lack matched
  product-path receipts; the plan phase remains active and no interactive
  latency SLO is claimed as qualified.

### 2026-11-23 — Exact-route retained recovery and feature-level Swarm gating

- Retained-audio recovery now requires an identified local tier and exact
  model ID, and only adapters declaring `exact-model` may submit it. BYOK,
  owned, unknown, or incomplete stream identities refuse the unary retry before
  a request leaves the browser; the journal remains available for recovery.
- Audio Tools, Web Console, Swarm Manager, and Portal carry the selected model
  through their same-origin unary hop. Web Console and Swarm Manager forward an
  explicit model to Audio Tools without resolving or attaching tier credentials;
  Audio Tools also skips default credential lookup for explicit model routes.
- Swarm Manager mic availability now uses `voice-input` and `voice-streaming`
  feature status. An unrelated `voice-output` degradation no longer disables
  dictation; unknown input status remains fail-closed.
- Focused verification: shared capture 32 tests, audio-tools adapter 55 tests,
  Web Console voice 16 tests, Swarm Manager voice/gate 29 tests, Portal voice 7
  tests; Audio Tools STT/chain, Web Console proxy, and Swarm proxy/discovery Go
  suites pass. Audio Tools `pnpm type-check` passes. Test Genie
  `20261123-014635-75b08857` passes unit and performance with 60 existing
  advisory low-coverage findings. Web Console/Swarm full UI type-checks remain
  red on unrelated Vite-version and application typing issues.
- Exact-route refusal preserves retained PCM and avoids unapproved remote work;
  the browser mic/composer cohorts are still unqualified because this shell has
  no managed Agent Manager identity, browser grant, or active Test Genie demand
  lease. No product latency band is claimed.

### 2026-11-23 — Rechecked local model routes and noisy Sherpa evidence

- The current lab-owned 10 dB VCTK experiment is
  `dd3cd80d-2e8c-474c-b656-2f384412c98f` (`sherpa-vctk-heldout-10db`), with
  360 clips and run `775031c1-25ad-46aa-8aec-69551bbc1d23`. Sherpa Zipformer
  scored 11.76% WER over 3,776 normalized reference words. The stream-safety
  gate passed (maximum dropped span 3 words against a threshold of 4; no
  retractions). Latency was not measured. The confidence is low and promotion
  remains unstable because interval accounting, duration/fault profiles,
  browser product-path, and device evidence are missing. All 1,140 listed VCTK
  clips still have zero reference-review records; this result is diagnostic,
  not a model qualification.
- `audio-tools model resolve` selected
  `sherpa-zipformer-en-2023-06-26` for the Sherpa streaming engine and
  `qwen3:1.7b` for Audio Tools summarization. The current TTS catalogue default
  remains Kokoro, while the product's healthy conditional route synthesized
  a short WAV through `kale-tts` / `kale-tts-1-nano` (653 ms on this host).
  This is a route check, not naturalness or multi-voice quality evidence.
- The canonical GPT-Live key address remains unconfigured. Current credential
  status reports the host-bound TPM encrypted-file provider available; its
  interactive provisioning path accepts the secret through a hidden prompt or
  stdin, never an argument. No OpenAI request was made. Provider/model
  qualification still prevents production duplex route composition.
- The owner `experiment wait <id> --json` and `experiment report <id> --json`
  commands both returned this persisted report in the current lab instance.
  Each JSON response is very large for a 360-clip cohort because it includes
  the complete recipe and per-clip/run details. Project aggregate fields with a
  client-side JSON filter when detailed rows are unnecessary. A redundant 20
  dB rerun was canceled after confirming the existing 10 dB receipt; it
  produced no report or run.
- The independent lab setpoint rerun `prog_53ac5a8c-260d-4655-b7ef-268dbc06cf84`
  still resolves 5/18 required rows, with 13 unavailable and overall acceptance
  unknown. The 10 dB experiment is diagnostic but does not join the required
  target-bound `corpus-quality` receipt because reference review and the other
  model promotion predicates are still absent. The 20 retained qualification
  receipts remain 8 valid, 12 failed, and 0 invalid; failures were preserved.

### 2026-11-23 — Revalidated shared capture and stream lifecycle races

- Re-ran the current shared `audio-capture-browser` package after the in-flight
  source changes: 36 test files passed, 266 tests passed, 1 skipped, and the
  TypeScript production build passed. Its accelerated 60-minute retention test
  is a simulated browser-package harness, not a physical microphone session.
- Targeted Go race suites pass for `internal/stt/segmenter`,
  `internal/stt/session`, and `internal/duplex`. These exercise decoder failure,
  session teardown/recovery, and duplex close/session concurrency; they do not
  qualify browser latency or a live GPT-Live provider.
- Test Genie run `20261123-040119-8162f2ed` passes the current `unit` and
  `performance` phases. The maturity report remains `functional_incomplete`
  (score 58), with stale business/docs/proto/structure evidence and only 2 of
  29 requirements passing. The unit phase reports 60 advisory `LOW_COVERAGE`
  findings; the performance phase reports none. This scoped green run is not
  whole-scenario acceptance or a device/product-path qualification.
- The new shared `internal/transcriptmetrics` package now has behavior tests
  and 97.8% statement coverage. Count-only `Align`/`WER` use two reusable rows
  instead of a full edit matrix; operation traceback is unchanged. Exhaustive
  parity checks compare binary token sequences through length 5 against the
  matrix recurrence. On this Linux host, one 800-token benchmark case measured
  1.43 ms, 40,960 B/op, and 2 allocations/op, versus the same implementation's
  recorded matrix baseline of 2.95 ms, 16.4 MB/op, and 801 allocations/op.
  This improves bounded alignment cost; it does not supply human-reviewed
  long-form timing references or change timestamp qualification.
- Follow-up Test Genie unit run `20261123-040714-31d32625` passes after the
  alignment change. Its unit report still has 60 advisory low-coverage
  findings overall, but no longer lists either `transcriptmetrics` source file;
  the broader maturity result remains `functional_incomplete`.

### 2026-11-23 — Removed one bounded TTS request copy and refreshed qualification status

- `SpeechSynthesizer` now passes its marshalled request bytes directly through
  `bytes.NewReader`, avoiding the extra `[]byte` to `string` copy before the
  HTTP request. The TTS input limit remains 8 MiB. The focused race suite
  `GOWORK=off go test -race -count=1 ./internal/tts` passes.
- `make catalog-check` passes. The live device matrix remains inventory-only
  with zero qualified cells; all 27 target/profile cells are unrun or
  unverified, so the device and browser qualification requirement remains open.
- API Health still reports L1 with eight warnings. Source review confirms the
  three response-body findings are false positives: one points at the
  idempotency store rather than an HTTP response, one transfers response-body
  ownership to a caller that closes it, and the TTS path closes bodies in its
  response conversion helpers. The five logging findings point to build and
  code-generation CLI output, not the API request path; no runtime changes were
  made to suppress these analyzer findings.

### 2026-11-23 — Rechecked selected local speech routes and credential write path

- The live Audio Tools product API selected Sherpa Zipformer for streaming STT
  and `qwen3:1.7b` for its application-scoped summarize route. The CLI streamed
  `api/internal/diagnostics/smokedata/quality_speech.wav` through
  `sherpa-onnx/streaming-zipformer-en-2023-06-26`, emitted visible partials,
  acknowledged captured samples through 31,268 with durable delivery, and
  finalized the five lexical words in 1,584 ms. The replay is unpaced and its
  punctuation/speaker-policy output is not interactive latency or quality
  qualification.
- A live product TTS request through the `auto` route selected
  `kale-tts` / `kale-tts-1-nano` and wrote a WAV response in 1,018 ms. This
  confirms conditional product routing while the model resolver's qualified
  catalog default remains Kokoro; it does not qualify Kale naturalness or
  multi-voice quality. Sherpa's attached server remains STT-only
  (`tts_model_loaded:false`), so Sherpa-hosted product TTS was not exercised.
- `vrooli credentials doctor --check-writes` reports the host-bound TPM
  encrypted-file credential store available and writable. The declared GPT-Live
  address remains unconfigured; no API key was created or used. Production
  duplex remains gated independently on provider qualification and
  authorization.
- The shared capture package's focused long-session/lifecycle suite passes
  269 tests with one skipped. The active dictation phase remains open: there is
  no current paced browser microphone/composer receipt covering readiness,
  rendered partial cadence, committed backlog, final tail, interruption, and
  resource pressure. Unit and unpaced CLI checks do not satisfy that acceptance.

### 2026-11-23 — STT engine routing now joins providers through declared adapters

- Bootstrap previously associated provider IDs with evaluation adapters, so a
  new local provider reusing an existing STT protocol still needed a provider
  switch edit. It now projects catalog-declared `stt_eval_adapter_id` values
  and engine IDs onto the registered protocol adapters.
- Added a synthetic provider/engine regression proving a new catalog
  declaration can reuse an existing adapter without a provider-specific code
  branch. This covers declaration-only additions within an implemented
  protocol; a genuinely new runtime protocol still needs its own adapter.
- Verification: `GOWORK=off go test -race -count=1 ./internal/bootstrap
  ./internal/providercatalog` and `make catalog-check` pass.

### 2026-11-23 — Test Genie now resolves the Audio Tools lab validation endpoint

- Added the generic `validation.endpointVariant` descriptor field and declared
  `lab` for Audio Tools soak. Test Genie now resolves `audio-tools@lab` for
  readiness, execution, and descriptor conformance while preserving
  `audio-tools` as the provider identity and lease key. The product API remains
  separate from the lab-only validation route.
- Focused Test Genie readiness, descriptor, provider-conformance, projection,
  and execution tests pass. The canonical soak run
  `20261123-064809-576a9c63` cleared readiness, started the lab provider, and
  reached the soak phase. It then failed closed because the required run-scoped
  identity, browser-grant, and Test Genie lease receipt set was incomplete.
  After deploying the diagnostic change, run
  `20261123-065337-dfed4974` confirms all three exact receipt types were absent
  on the direct CLI path. The provider diagnostic names missing receipt types
  without exposing receipt values; `GOWORK=off go test -race -count=1
  ./handlers/validation` passes.
- Credential authority status remains available/unconfigured for
  `vrooli/openai-gpt-live` / `api-key`. The native OpenAI setup chooser returned
  no confirmed key-creation choice; no key was created. The existing encrypted
  authority path is ready through `vrooli credentials provision --identity
  vrooli/openai-gpt-live --field api-key`, which uses a secure terminal prompt
  and does not require an Audio Tools restart after provisioning.
- Test Genie restarted healthy with the rebuilt API. Its standalone `make
  build` UI step still reports a missing
  `@vrooli/react-component-library/BaseStyles/1.3.0` dependency; lifecycle
  restart rebuilt the required runtime artifacts and succeeded. Browser grant
  and paced product-path qualification remain open.
- A focused TTS catalog audit found declaration-only provider additions remain
  supported when they reuse one of the configured Kokoro, Kale, or Sherpa
  endpoints; health path, cache namespace, speed mode, model role, and auto
  priority are declaration-driven. Arbitrary environment-variable endpoint
  names remain rejected by design. Adding a new service endpoint still needs
  typed environment/startup integration and its protocol adapter.

### 2026-11-23 — Dictation diagnostics now observe rendered transcript revisions

- `recordVisibleTranscriptRevision()` existed but had no product call site, so
  `visibleRevisionCount` and `visibleUpdateGapsMs` stayed empty in real browser
  diagnostics. The hook now records after React commits coalesced partial text
  or durable segment changes. First-partial latency is anchored to that same
  rendered revision rather than the earlier stream callback.
- Verification: the shared capture package passes 269 tests (one skipped),
  TypeScript build passes, and `git diff --check` is clean. This fixes telemetry
  wiring; paced browser measurements are still required to qualify the adopted
  latency bands.

### 2026-11-23 — Kale Nano candidate voice smoke on the live product route

- The already-running product API and managed Kale Nano resource synthesized
  the same spelling-and-numbers prompt through `audio-tools tts synthesize` for
  four distinct canonical voice mappings. All responses identified
  `kale-tts-1-nano`; the mapped voices were Kenzie, Simone, Earl, and Owen.
  Outputs were finite mono 24 kHz PCM WAV files, 7.875–9.275 s long. Request
  latency ranged from 130–1,046 ms, with the first request cold.
- Whisper Small then transcribed all four outputs with the same lexical words;
  only number formatting and punctuation varied. The first ASR request was
  cold at 9.80 s; the three warm requests took 95–105 ms. This is an automated
  intelligibility smoke only: the STT route substituted to Whisper Local, and
  it does not establish pronunciation quality, naturalness, voice diversity,
  robustness, or release qualification. Kokoro remains the qualified default.
- Preserved the four WAVs, exact prompt, product-route identities, latencies,
  durations, ASR limitation, and SHA-256 checksums in the review-only packet
  `~/.local/share/vrooli/audio-tools/qualification/kale-nano-review-20261123/`.
  Its manifest is valid and all four WAV checksums verify; human review remains
  pending.

### 2026-11-23 — Duplex interruption synchronization and Sherpa release audit

- Serialized OpenAI Live assistant-output delivery with the short Cancel
  interruption-state transition while leaving the provider-ack wait outside the
  lock. This closes a race without blocking the stream read loop. The focused
  `go test -race ./internal/duplex` suite passes, including a deterministic
  blocked-sink concurrency case.
- Sherpa's release packaging path is already reproducible and fail-closed:
  two retained v5 stages validate against the current manifest and share
  archive SHA-256 `5b100adac89bb27c5e6e51824d90906a0eb4930e3a1a012cfe04148af113830c`
  and tree digest `20255bfe34a82514e6e05c90bd97d00c51737878da6b1ef71eb533686b1a10d5`.
  Offline stage validation passed 10 cases and packaging passed 3. No signing or
  publication was performed: release-owner provenance and publication URL are
  absent, and configured trust-anchor status does not authorize signing.

### 2026-11-23 — D1/D2 bounded host inference and Test Genie build recheck

- A networkless lab host-portability run passed D1 Whisper Small CPU inference
  in 5.16 s and D2 Sherpa Zipformer int8 in 109 ms for 1.9553 s of audio
  (1.30 s model load, 454,516,736-byte peak). Its receipt is
  `audio-tools:host-portability:20261123T071432.779832374Z:amd64`; Kyutai remains
  unavailable. The governed read `prog_8c5ce335-df46-4dce-a46e-6cc818a4975c`
  still reports acceptance `unknown` and `failed:host_matrix_incomplete` (5/18
  required rows resolved). These are bounded inference receipts; they do not
  qualify any browser/product-device cell. The isolated Sherpa SDK archive was
  digest-verified and extracted under `/tmp`; no resource was started or
  published.
- Test Genie `make build` now passes against the authored/generated
  BaseStyles 1.3.0 package exports, with only the existing large-chunk
  advisory. No Test Genie source change was needed; the prior unresolved import
  report was not reproducible with the current generated package artifacts.

### 2026-11-23 — Bounded every local TTS response before cache or RPC buffering

- The direct WAV passthrough path transferred an unbounded backend body to the
  TTS service, where `io.ReadAll` could grow with any backend response. The
  shared service and conversion path now enforce the same 64 MiB audio limit;
  oversize audio is rejected before cache writes and marks the local response
  unavailable for the existing fallback policy.
- Added exact-limit and limit-plus-one tests, including proof that the reader
  stops at the first excess byte. Verification: race tests pass for
  `internal/tts`, `internal/bootstrap`, and `handlers/tts`.

### 2026-11-23 — Bounded speaker and streaming-health responses; fixed CLI tier label

- Speaker resource JSON responses are capped at 4 MiB and extracted audio at
  64 MiB; oversize resource output now fails instead of allocating without a
  bound. The TTS voice-catalog response now has a 1 MiB cap, sharing the same
  bounded-reader path as synthesis audio.
- Kyutai's resource health JSON now has a 64 KiB bound and rejects oversized
  metadata. `GOWORK=off go test -race -count=1 ./internal/ai/sttchain` passes,
  including the oversized-health regression.
- CLI provider-tier rendering now includes the `browser` tier rather than
  displaying it as `unknown`; the focused provider-label, settings, and
  provider tests pass. The consolidated race run `GOWORK=off go test -race
  -count=1 ./internal/tts ./internal/ai/sttchain ./internal/stt/pipeline
  ./internal/stt/ingress ./internal/ai/summarizechain ./internal/byok
  ./internal/bootstrap ./handlers/tts` passes.

### 2026-11-23 — Bound AI Gateway summary results

- The local summary adapter now rejects validated result JSON larger than 4 MiB
  before decoding and uses a string reader to avoid a duplicate full-response
  allocation. The focused race run `GOWORK=off go test -race -count=1
  ./internal/ai/summarizechain` passes, including oversized-result coverage.

### 2026-11-23 — Drive BYOK adapter registration from provider declarations

- Active BYOK catalog rows now declare `byok_adapter_id`, a protocol factory
  key separate from voice/TTS `adapter_id`. Registry construction walks those
  declarations and registers adapters under each catalog provider's canonical
  ID. A synthetic provider can reuse the OpenAI Whisper protocol without a
  provider-specific registry branch. A new protocol still needs one factory
  implementation; adding another provider on a declared protocol is data-only.
- Focused race tests pass for `internal/providercatalog`, `internal/byok`,
  `internal/bootstrap`, and `internal/catalogconformance`; `make -C ..
  catalog-check` passes. The target is owned by the Audio Tools scenario Makefile.
- Broad API regression sweep also passed: `GOWORK=off go test ./...` (all API
  packages, including duplex, STT, TTS, catalog, bootstrap, and handlers).

### 2026-11-23 — Protect actionable facts in compact summaries

- A live Audio Tools moderate summary through the selected `qwen3:1.7b` role
  took 18.2 s on its first host call and omitted the 42-person pilot size plus
  Priya's and Marcus's assigned responsibilities. The shared `qwen3.5:4b`
  quality role took 1.8 s warm on the same input but changed the March 29 review
  deadline to the nonexistent “March 39” and dropped other action details.
  These are single-input observations, not a comparative quality verdict.
- Replaced the moderate prompt's instruction to keep only one conclusion with
  compact preservation rules for people/responsibilities, dates, quantities,
  approvals, deadlines, risks, and contingencies; it explicitly forbids fact
  invention. A direct resource-role probe using that instruction preserved
  every listed fact in 89 output tokens, `done_reason=stop`, under the existing
  90-token short-input cap (0.675 s warm). A follow-up at a 120-token cap
  returned the same 89-token output with a complete stop in 0.82 s. The
  moderate short-input token floor is now 120, leaving margin without increasing
  observed output length. This exercises the model role and exact instruction,
  not a restarted product API. More caller cases and held-out review are still
  required.
- Updated prompt regression assertions. Race tests pass for `internal/summarize`,
  `internal/ai/summarizechain`, and `handlers/summarize`.

### 2026-11-23 — Sherpa TTS startup cancellation and Kroko tail audit

- The local TTS provider now returns context cancellation/deadline from lazy
  backend startup instead of swallowing it and falling through as unavailable.
  Focused non-race API tests pass for `internal/bootstrap`,
  `internal/ai/ttschain`, `internal/tts`, and `handlers/tts`. A live Sherpa
  endpoint product smoke was not run in this unit.
- Audited the in-flight Kroko native tail flush: injected silence is capped at
  1.2 s by default (5 s maximum), zero disables it, and tail samples do not
  advance client-audio sample counts used for timestamps. Flush runs only on
  explicit WebSocket end; disconnect/error cleanup closes the stream and
  releases admission. Ordinary Sherpa server tests pass. Native-tagged tests
  cannot compile here because the Sherpa C API headers are absent; no native
  Kroko model fixture was configured, so lexical-tail smoke remains unverified.
  Kroko remains opt-in and non-default.
| 2026-11-23 | Codex | measured (lab D1/D2 host-portability attempt) | Ran `audio-tools --instance lab validation run-host-portability --json`; evidence `2e1ba20c-4b10-4928-bb87-cd9b72d3b8dd`, artifact `audio-tools:host-portability:20261123T081508.854055977Z:amd64`. Digest-pinned, networkless Whisper Small CPU inference passed in 10.407 s on Linux/amd64 (8 CPU / 8 GiB container; fixture and model hashes recorded). Sherpa Zipformer int8 passed its native smoke in the 2-vCPU/4-GiB D2 container: 119.36 ms inference for 1.955 s audio (RTF 16.38x), 426,516,480-byte peak, model load 4.20 s. The selected product engine was available. Kyutai accelerator inference was unavailable and unrun; evidence is marked `passed=false`, so the host-portability acceptance claim remains incomplete. This is lab evidence and does not qualify browser capture, product-route latency, D3-D6, or broader quality. Product API was not changed. |
| 2026-11-23 | Codex | verified (current product Sherpa stream-route smoke) | `audio-tools --api-base http://127.0.0.1:19630 stt transcribe-stream --file scenarios/audio-tools/bas/fixtures/dictation-reference.wav --language en --json` used the running product API. Sherpa emitted partials, durable acknowledgements through captured sample 31,268, and final text `The quick brown, fox jumps.` with provider `sherpa-stt` and model `sherpa-onnx/streaming-zipformer-en-2023-06-26`; reported final latency was 3,053 ms for 1.955 s audio. This validates current CLI→product API→Sherpa routing and terminal delivery. The file feed is unpaced, so the duration is not user-perceived latency; a one-phrase fixture does not qualify quality, long-form stability, browser capture, or device behavior. Product build identity remained `sha256:b04c32340ac67d114da314a39fb41de2ab0797635ec19c6440cfc10e21594a19`. |
| 2026-11-23 | Codex | verified (live conditional Kale TTS route and resident footprint) | A current product synthesis through API 19630 for `voice.feminine.warm` returned `kale-tts` / `kale-tts-1-nano`, voice `kenzie`, in 482.2 ms. The output is a valid 24 kHz mono PCM16 WAV, 1.925 s. Kale's managed resource is healthy; its serving process used 83,652 KiB RSS at observation. The staged runtime bundle is 498 MiB, including Python runtime and wheels, so compact weights do not imply a compact install artifact. The generic `model resolve --capability tts` still reports Kokoro as its static catalog default; actual product auto-routing chooses Kale while healthy and retains fallback. This is one live route/footprint observation, not naturalness or held-out quality evidence; no default promotion is claimed. |
| 2026-11-23 | Codex | verified (Kale stripped runtime installed and ten-voice sweep) | Replaced the unstripped CPython archive with pinned upstream `install_only_stripped` (SHA-256 `5acfa3e9ba26b51ae161c83aff278da915b590d22373a424b2ba55b8afe91fcc`) and updated the managed tree digest to `cf38c6dc79ff578c40454b79c12b35356f6c4049de43b013b525d3fb95facb63`. Offline candidate composition reproduced the digest; `make -C resources/kale-tts test` passed 8 tests and `vrooli resource validate kale-tts` passed. The managed resource was stopped, reacquired, and started through the resource CLI; status is healthy with the new digest and no pending update. Installed bundle allocation is 257 MiB versus 498 MiB previously (about 241 MiB / 48% reduction). Post-update product synthesis selected `kale-tts-1-nano` and produced valid 24 kHz mono PCM16 WAVs: cold 1,073.5 ms / 2.55 s and a distinct warm request 113.8 ms / 3.05 s. The documented 20-request sweep covered all ten voices with no empty, near-silent, or clipped responses (median 75.2 ms, max 614.13 ms); receipt `/home/matthalloran8/.local/share/vrooli/audio-tools/qualification/kale-nano-stripped-runtime-20261123.json`, SHA-256 `67b012475d456861740d13426f35ba648088d6348b749db40d93fe319f9dddf2`. The one-thread process RSS reached about 225 MiB HWM against its 512 MiB RAM budget. This is runtime integrity evidence only; naturalness and held-out quality remain open, and Kokoro stays the static registry default. |

### 2026-11-23 — TTS startup cancellation does not poison retries

- Fixed a lifecycle cache edge in `ttschain.CLIBackendEnsurer`: a start canceled
  by its caller was recorded as a 20-second cooldown result, so a later
  independent request could be rejected without attempting startup. Canceled
  and deadline-exceeded attempts now remain retryable immediately; ordinary
  start failures retain the cooldown.
- Added a deterministic regression test for a canceled start leader followed by
  an immediate successful retry. `GOWORK=off go test -race ./internal/ai/ttschain
  ./internal/bootstrap` passes. This covers the lifecycle seam and bootstrap
  wiring only; no runtime synthesis or model quality was evaluated.

### 2026-11-23 — Shared browser capture cleanup and validation gates

- Fixed a teardown leak in `packages/audio-capture-browser`: if level-monitor
  filter-chain construction throws after creating the media-stream source, the
  source is now disconnected immediately. Added a regression assertion for the
  failure path. The focused test and package build pass; the package suite
  passes 282 tests with one existing skip, and its V8 coverage run passes the
  declared 65% floor (69.9% lines, 71.64% functions, 74.12% branches).
- Package lifecycle validation reports unit health and structure at L3. Its
  overall result remains unevaluated because dependency-analyzer, quality, and
  storage validation adapters do not support the package target (RPC
  `unimplemented`, unexpected EOF, and unsupported target kind respectively);
  these are runner limitations, not passing validations.
- A fresh direct V8 run passes 1,343 tests: branches 85.40%, functions 85.67%,
  lines 92.33%, and statements 89.94%. The committed 85% UI coverage floor is
  met. Added behavior tests for audio-operation request mapping,
  validation, results/errors, experiment-stream progress/fallback polling,
  model registry mapping, comparison and report fallbacks, recorder recovery,
  experiment-service defaults, corpus safety evidence, and suite-card states.
  The final Test Genie unit run `20261123-134448-f622a248` passed in 73 s at L2;
  it reports 70 advisory low-coverage findings but no blocking errors. The
  separate performance phase passed at L3 in run `20261123-134110-68e8b12a`.
  An earlier combined run exposed missing test-fixture fields and a direct
  Testing Library render; both were corrected before the passing unit run.
  Focused lint and `pnpm type-check` pass, with generated strings in sync.
  The current acceptance board resolves
  6/18 required rows and remains unknown; authenticated five-cycle browser
  execution still lacks the required Agent Manager identity, BAS grant, and
  Test Genie lease.

### 2026-11-23 — Current model defaults and host resource attribution

- Live resolvers still choose Whisper Small (488 MB, MIT) for batch STT and
  Sherpa Zipformer (310 MB, Apache-2.0) for English native streaming. TTS's
  catalog default remains Kokoro (330 MB, Apache-2.0); Kale Nano remains the
  conditional low-footprint route when healthy, with its earlier measured
  runtime/voice sweep retained above. Audio Tools summarization resolves to
  Qwen3 1.7B; no model-default change was warranted by this check.
- At 2026-11-23 13:55 UTC, `resource-ollama models retention` showed Qwen3 1.7B
  installed (1.36 GB) but not resident, while Qwen3.5 4B (3.39 GB) was
  resident for several shared roles including the Audio Tools quality fallback.
  The request ledger attributes 1,361/1,580 Qwen3.5 4B calls to AI Gateway,
  17 to Codex, and 122 as unattributed; it does not justify charging the whole
  resident model to Audio Tools. GPU process sampling showed two Ollama runners
  (606 and 5,080 MiB) plus a 1,380 MiB reranker; no Audio Tools process held GPU
  memory in that snapshot. Product API RSS was 28 MiB, lab API 22 MiB, Kale
  process 20 MiB, and the separate temporary Sherpa server 9 MiB. This is one
  host snapshot, not peak or sustained resource qualification.

### 2026-11-23 — Fresh product TTS smoke and GPU attribution

- A fresh product `tts synthesize` request selected `kale-tts` /
  `kale-tts-1-nano` through the default `auto` route, returning valid 24 kHz
  mono PCM16 WAV. The first distinct request took 644.9 ms for 2.25 s of audio;
  a second distinct request reported 202.9 ms. This reconfirms routing and
  service behavior only; it does not qualify pronunciation or naturalness.
  The Audio Tools scenario reported healthy after the product requests.
- The static TTS model resolver still reports Kokoro as the catalog default;
  the product `auto` route conditionally selects Kale while its managed
  provider is healthy. Audio Tools summarize resolves to `qwen3:1.7b`; the
  shared generic `summarize.default` remains `qwen3.5:4b`.
- At 2026-11-23 21:10 UTC, GPU sampling reported 16,376 MiB total, 7,465 MiB
  used, and 2% utilization. Compute processes were two Ollama runners (606 and
  5,078 MiB) and the shared reranker (1,380 MiB). Ollama `/api/ps` showed
  `qwen3.5:4b` and `nomic-embed-text`; the Audio Tools 1.7B summarize model was
  not resident. The Audio Tools API used 27 MiB RSS, Kale's managed server
  used 85 MiB after synthesis, and the attached Sherpa server used about 7 MiB
  in this idle sample. This snapshot does not attribute the unaccounted GPU
  memory or shared models to Audio Tools and is not a peak/sustained profile.

### 2026-11-24 — Kale product route and simulated device-route coverage

- A fresh live API product-route test selected Kale Nano through `auto`, then
  exercised five canonical voice mappings and two cache hits. WAV structure,
  bytes, content type, and route identity were captured in
  `coverage/kale-nano-product-live-20261124.json`. The runtime was warm; no
  human speech-quality, cold-start, sustained-load, peak-memory, or license
  qualification is claimed. Kale remains a conditional candidate and Kokoro
  remains the catalog default.
- Simulation now emits all 9 declared D1–D6 device/browser cells and 27
  local/BYOK/owned route-policy cases. Every case is labeled simulated and
  policy-only, with `physical_qualification=not_run`; iOS Safari/PWA and
  Android Chrome retain their physical-device prerequisites. This improves
  deterministic route coverage without counting it as browser or
  native-device qualification.
- Long-form timestamp scoring now has a 61-minute/10,000-word synthetic
  regression with repeated tokens and one omitted reference word. It verifies
  99.99% alignment coverage, drift slope detection, and that a localized edit
  does not offset later matches. The scorer tests pass; the synthetic result
  does not qualify model timing, and three-stream human corpus review remains
  open. See `LONG-FORM-TIMING-CORPUS.md`.
- Current owner checks passed: API `go test ./...`, API `go test -race ./...`,
  CLI `go test ./...`, UI 142 files / 1,352 tests, catalog conformance, and
  Windows/amd64 plus macOS/arm64 cross-builds. The CLI primitive-evidence
  verifier found a stale generated artifact after command-surface edits; it
  was regenerated through its owning test and the full CLI suite then passed.
- Non-transform operation conformance now checks every STT, TTS, and summarize
  catalog operation against its generated Connect service/method, streaming
  mode, primary wire port fields/types, and CLI manifest binding. This closes
  the prior gap where declarations only proved operation ID and CLI ownership.
  Focused catalog tests pass. Optional request fields and individual CLI flags
  remain outside the declared contract and are not checked by this gate.

### 2026-11-24 — Local summary and TTS route smoke

- A live Audio Tools summary request selected the scoped `qwen3:1.7b` model and
  preserved all 8 requested facts in a short report (89 output tokens). The
  first request took 10.51 s; a warm follow-up preserving 5 facts took 401 ms.
  This is a short content smoke, not a latency distribution or fidelity
  qualification. The shared `summarize.default` role remains `qwen3.5:4b`.
- A live product `tts synthesize` request through `auto` selected Kale Nano
  (`kale-tts-1-nano`, voice `kenzie`) and returned a valid mono 24 kHz PCM16
  WAV in 955 ms. This confirms current routing and output shape only; it does
  not qualify speech quality, cold start, concurrency, or sustained latency.
- A point-in-time host sample after these calls showed about 26 GiB available
  RAM and 7.8 GiB GPU memory in use. Ollama listed both shared `qwen3.5:4b`
  and scoped `qwen3:1.7b`; the latter had a roughly 10-minute keep-alive. The
  Audio Tools API, Sherpa, and Kale processes used approximately 26 MiB, 7 MiB,
  and 16 MiB RSS respectively. This is not a peak or sustained profile and
  does not attribute all shared GPU use to Audio Tools.

### 2026-11-24 — Admission refresh and reconnect retry bounds

- The STT live resolver now coalesces over-age inventory refresh in the
  background. While the snapshot is beyond its freshness bound, admission
  returns an explicit no-serviceable result instead of waiting on control-plane
  probes or silently selecting from unverified resource facts. The handler and
  segmenter preserve that fail-closed result. Focused resolver coverage checks
  prompt refusal, asynchronous refresh, and selection recovery after fresh
  facts arrive.
- Shared browser reconnect attempts no longer reset at socket readiness. The
  retry budget resets only after a newly processed cursor is durably compacted,
  so repeated ready-then-close connections exhaust the bounded retry path and
  enter retained-audio recovery. Regressions cover both the flapping connection
  and the progress-based reset.
- Verification passed: API race tests for `internal/sttengine`,
  `internal/stt/segmenter`, and `handlers/stt`; all 317 browser capture package
  tests (1 skipped); browser package TypeScript build; scoped `git diff --check`.
  Browser-paced latency and end-to-end host/device qualification remain open.
- Test Genie run `20261124-134105-357ccc09` passed Audio Tools `unit` and
  `performance` phases after repairing the observed test recorder race and UI
  test typing error. The run retains 67 low-coverage warnings as advisory
  findings. The earlier failed run `20261124-133645-c378639f` remains in history;
  its failure evidence was inspected and the blocking issues were corrected.
- A post-run host sample showed 24 GiB available RAM and 5,385 / 16,376 MiB
  GPU memory used at 1% utilization. Named GPU allocations were two Ollama
  runners (608 and 2,984 MiB) and the shared reranker (1,380 MiB); this leaves
  413 MiB unattributed and does not assign shared model use to Audio Tools.
  Audio Tools API/lab and Sherpa processes were about 29/22/8 MiB RSS. This is
  one idle snapshot, not a peak or sustained footprint measurement.

### 2026-11-24 — Rumble validation order and current resource attribution

- Rumble's explicit language-code validation now runs before it takes the
  single-flight slot or decodes/base64-encodes audio. Unsupported languages
  therefore fail cheaply and deterministically, even when the audio format is
  malformed. Regression coverage is in
  `api/internal/ai/sttchain/provider_rumble_vllm_test.go`; focused Rumble tests
  passed. No model/runtime qualification is implied.
- A 15:20 UTC host sample showed 11,428 / 16,376 MiB GPU memory used at 4%
  utilization. Named allocations were Ollama (608 + 2,982 MiB), reranker
  (1,380 MiB), and Ollaya (6,024 MiB), leaving 434 MiB unattributed. The Audio
  Tools API and lab API were about 29 and 21 MiB RSS; Kale about 16 MiB and the
  attached Sherpa process about 7 MiB. No Audio Tools process appeared in the
  GPU compute-process list. This is an idle point sample, not peak or sustained
  model memory; shared Ollama/Ollaya/reranker allocations are not attributed to
  Audio Tools and were not modified.
- Test Genie comprehensive run `20261124-152039-cfe18113` was started for the
  current scenario state; its terminal result is pending.

### 2026-11-24 — Full scenario gate and typecheck planner join

- Test Genie run `20261124-152039-cfe18113` terminated `FAIL`: 16 passed, 13
  failed, 1 skipped, 3 not evaluated (393 seconds). Unit reached L2 and
  performance reached L3. Remaining failures span shared structural/API
  analyzers, browser admission, provider conformance, and unrelated platform
  gates; the findings artifact is retained by Test Genie. The API health and
  AI conformance rules produced confirmed model/ownership-classification false
  positives, reported as `knw-1795534258046069139` and
  `knw-1795534656756877621`. The scenario-qa heartbeat remains manually paused,
  so both reports are published but awaiting the team's normal drain.
- The TypeScript planner error came from Quality Health joining solely by
  logical workspace ID: product `ui` and `lab-ui` are separate Code Facts
  surfaces sharing one physical UI root, while Unit Health correctly planned
  that root once. Quality Health now also joins plan coverage by canonical
  working directory. Regression coverage proves a single planned `ui` workspace
  covers both same-root surfaces and does not cover a different root.
- Validation: `GOWORK=off go test ./internal/audit ./internal/rules` passed in
  `scenarios/quality-health/api`; after restarting Quality Health through the
  lifecycle, its live `TYPECHECK_PLANNER_COVERAGE` audit passed with both `ui`
  and `lab-ui` covered and no planner finding. `GOWORK=off go mod tidy -diff`
  also passes in Audio Tools after removing two stale `httpsnoop v1.0.3`
  checksums. The current device-matrix resolver still reports 0 of 27 qualified
  cells; no physical-device claim is made.

### 2026-11-24 — Follow-up quality/dependency gates and browser storage fallback

- Added guarded localStorage reads/writes to locale initialization and locale
  changes. When browser storage is denied, the selected locale still applies to
  the current page. The regression mocks a `SecurityError` on storage write.
  The focused i18n suite passed 12/12; touched-file ESLint and TypeScript
  `tsc --noEmit` passed. Existing lazy-loaded locale chunks and current shared
  i18n edits were preserved.
- Test Genie single-phase `quality` run `20261124-153953-4cdf62b1` passed. Its
  remaining TS dangerous-pattern warnings cannot be safely cleaned from the
  current aggregate output: Quality Health scans tests and broad `as Type`
  patterns and emits no per-file locations. This was reported to the owning
  scenario-qa inbox as `knw-1795534942606442980`.
- Reconciled the Audio Tools API module with Scenario Dependency Analyzer; it
  removed two unused `replace` directives (`github.com/vrooli/cliresolve` and
  `.../scenario-to-desktop/runtime`). The analyzer `--check` is clean, and
  Test Genie dependencies run `20261124-154036-6cfa950d` passed. One informational
  service-dependency evidence gap and the unsupported `runtime` surface remain.
- Quality Health's same-root UI planner repair was validated after its managed
  restart: focused Go tests pass and its live audit covers both `ui` and
  `lab-ui` without a `TYPECHECK_PLANNER_COVERAGE` finding. Related detector
  findings are retained as `knw-1795534258046069139` (API ownership/CLI-output
  classification) and `knw-1795534656756877621` (model registry data versus
  hardcoded selection). All three bug reports are published; scenario-qa
  heartbeat triggering was refused because its manually-paused global state
  prohibits runs. No heartbeat resume or manual receipt was attempted.
- The live registry/resource state also now matches the resource-efficiency request:
  `whisper-small` is the eager default (488 MB artifact; ~716 MiB measured VRAM
  in the one-host product-path receipt), while Turbo is on-demand. Compact Kokoro
  remains the qualified TTS default; Kale Nano stays a conditional auto-route,
  not a promoted catalog default, because joined naturalness/quality evidence is
  incomplete. Focused model-footprint/default tests and `make catalog-check`
  passed. These receipts establish declared/default footprints and one measured
  Whisper route, not peak or multi-device resource qualification.
- GPT-Live setup check: `vrooli credentials status --identity
  vrooli/openai-gpt-live --field api-key --json` reports the canonical encrypted
  file provider available and the field unconfigured. The existing hidden-input
  provision command remains the owner-entry path; this workspace has no OpenAI
  Platform key-creation operation, so no secret or hosted request was created.
  The adapter endpoint and `session.start`/`session.started` handshake match the
  current official OpenAI Live WebSocket guide; production mounting remains
  gated on consent, spend, identity, model qualification, and browser/session
  acceptance evidence.

### 2026-11-24 — Managed lab/product route verification

- Restarted the default Audio Tools instance through `vrooli scenario restart`
  so it built current source and shared dependencies, then restarted
  `audio-tools@lab` through the lifecycle's variant route. Both reported healthy
  with matching expected/served build identities. API/UI health and UI-root
  probes returned HTTP 200 on both profiles.
- Runtime route proof: direct `CorpusService/ListClips` returned HTTP 404 on
  product port 19630 and HTTP 200 on lab port 15373. The installed CLI command
  `audio-tools --instance lab corpus list --limit 1 --json` also returned a lab
  corpus record. This confirms the product/lab route boundary against current
  processes, not only source-level composition tests.
- Current state is reflected in `docs/internal/MATURATION-STATE.md`. The
  lifecycle reported the optional Kyutai token and GPT-Live API key absent;
  startup remained healthy and neither credential is required for this proof.

### 2026-11-24 — Fresh full Test Genie evidence and cross-owner defects

- Full Test Genie run `20261124-160502-36ead68e` terminated `FAIL` after 462 s:
  19 phases passed, 11 failed, 2 not evaluated, and 1 skipped. Contracts passed
  cleanly; the failure is not the previous CLI binding or stale-evidence issue.
- The proto phase now exposes one shared Proto Health defect:
  `proto.rest_payload_invalid_conformance` rejects Audio Tools' canonical
  `websocket_binary` / `proto_binary` pair. Source and api-core descriptor
  documentation agree; the validator's allowed list does not. Published report
  `knw-1795536921941410052`.
- Docs, Experience Manager, Tidiness Manager, and Security Health phases fail
  on mismatched canonical phase presentations, with no Audio Tools findings.
  Published cross-provider contract report `knw-1795537067875124428`.
- Other remaining errors include known AI Conformance false positives for
  registry model IDs (`knw-1795534656756877621`), embedding-dimension matches in
  generated freshness snapshots, missing owner-issued Agent Manager/BAS/Test
  Genie soak receipts, missing BAS account authentication, missing monetization
  declarations/enforcement evidence, missing Test Genie provider execution
  metrics, and absent Stripe test outcome receipts. Their owner-specific
  artifacts remain the evidence source; no credentials, billing receipts, or
  authorization tokens were fabricated.
- Both published reports' heartbeat requests were refused because scenario-qa
  remains `paused-manual` by operator policy. Reports are retained for the
  normal reconciliation path; no resume or force request was made.

### 2026-11-24 — Monetization truth and usage-recorder shutdown race

- Corrected `.vrooli/monetization.json` to declare the implemented
  summarization credit route as `meter_only`, point its Class-A meter at the
  real LPBS summarization client, and omit hosted voice minutes while STT/TTS
  gateway delivery and durable billing evidence remain absent. The focused
  Test Genie `monetization-conformance` run `20261124-162346-759f944b` passed.
- Reconciled `docs/business/MONETIZATION.md`, `docs/internal/PROBLEMS.md`, and
  `docs/internal/MATURATION-STATE.md` with source: the summarization request
  client exists, STT/TTS clients are still unavailable, and the bounded local
  SQLite analytics queue is not a shared billing reporter or durable outbox.
- Fixed a shutdown race in `internal/usagereport.AsyncRecorder`: enqueue now
  shares a close gate, post-close calls are counted as drops, and repeated
  `Close` is safe. A concurrent enqueue/close regression verifies accepted rows
  drain before shutdown returns. `GOWORK=off go test -race
  ./internal/usagereport` passed.
- These checks establish static declaration conformance and local recorder
  race safety only. They do not qualify hosted billing, STT/TTS delivery, GPT-
  Live, or device/browser paths.

### 2026-11-24 — Keep cold STT inventory probes off stream admission

- `LiveResolver.Resolve` still synchronously observed a cold control-plane
  snapshot, so first stream admission could wait for the 10-second probe bound
  even though stale-cache refresh was already asynchronous. Added
  `ResolveForAdmission` for the segmenter and selector. It starts a coalesced
  background refresh and fails closed while inventory is unknown; fresh
  snapshots continue to receive bounded provider-liveness checks. The
  `ListEngines` endpoint already avoids control-plane calls, and Connect stream
  capacity accounting now uses the fast admission resolver too, so it cannot
  reintroduce a cold synchronous probe before the segmenter.
- A canceled request no longer starts a detached background refresh. Regression
  coverage proves cold admission returns under 100 ms, does not select without
  host facts, and admits after the background snapshot arrives.
- `GOWORK=off go test -race ./internal/sttengine ./internal/stt
  ./internal/stt/segmenter ./handlers/stt` passed. Browser-paced startup
  latency, first-request retry UX, and the cross-device matrix remain open.
- The focused Test Genie `performance` phase passed at L3 as run
  `20261124-163422-e58f5731`. This is a gate result, not a browser-paced
  measurement of first-request latency.
- After the Connect capacity-accounting call was switched to the fast resolver,
  the engine, selector, and segmenter race tests passed. One full-package
  invocation remained in a Connect bidi test waiting for client input; the
  subsequent unfiltered handler `-race -json` run passed in 3.5 seconds. The
  new capacity-accounting regression also passed focused under `-race`.

### 2026-11-24 — Bidi pipeline failure no longer waits for client half-close

- A real HTTP/2 Connect bidi regression reproduced a handler hang when engine
  admission failed immediately but the client kept the request side open while
  waiting for server events. The handler now joins input only when the pump has
  already observed end-of-input; otherwise it fails and persists the resumable
  ledger, sends the terminal acknowledgement, and returns so the transport can
  release the blocked `Receive`.
- Transport cancellation and websocket interruption still preserve resumable
  state, while receive/validation failures retain their original `input_failed`
  reason. The race suite exposed and closed a selection race between the
  pipeline and receive-pump results.
- `GOWORK=off go test -race ./handlers/stt -count=1` passed, as did the
  focused engine/selector/segmenter/handler race tests and the regression with
  the existing close-before-done, duplicate-start, and websocket-resume cases.
- The same Test Genie proto phase had reported a cross-owner false positive for
  Audio Tools' valid `websocket_binary` / `proto_binary` descriptor. Proto
  Health now accepts that explicit pairing and rejects mismatched binary
  transport declarations; its focused validator tests pass. This is source and
  validator evidence, not a rerun of the full Test Genie phase.
- Follow-up: the first fresh phase attempt reached an older running Proto
  Health process and reproduced that stale validator. After rebuilding
  `proto-health` with `vrooli scenario restart proto-health`,
  `proto-health validate scenario audio-tools --json` passed with transport
  contracts at L3, and Test Genie run `20261124-170042-feb88b04` passed its
  `proto` phase. The 60 remaining findings are warnings in existing domain,
  annotation, and endpoint-proof debt; no conformance error remains.
- Static checks are clean: `go vet ./...` in Audio Tools API and Proto Health
  API, plus `git diff --check` for both scenario trees.
