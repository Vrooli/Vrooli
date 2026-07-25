# Problems — Audio Tools

Persistent register of known issues, tech debt, and deferred work
specific to **this** scenario. Future agents read this file to avoid
re-discovering the same constraint.

This file ships empty in newly generated scenarios. Append entries as
they appear.

## What belongs here

- **Known bugs** that are real but not yet worth fixing
- **Tech debt** — workarounds that need a real fix later
- **Deferred work** — features descoped from a phase, with the reason
- **Architecture drift** — code/docs/tests that no longer line up with
  the intended capability map or boundary model
- **Constraints discovered the hard way** that aren't visible from
  the code (e.g., "this resource needs warm-up before the first call;
  see commit X")

## What does NOT belong here

- **Generic template issues** — those go in
  [`../guides/troubleshooting.md`](../guides/troubleshooting.md)
- **Open feature requests** — track those in PRD operational targets
- **Code comments** — if the constraint is local to one file, a
  comment there is more discoverable
- **Test failures** — fix them, don't document them

## Entry template

Use this shape so entries are scannable. Append newest at the bottom.

```markdown
### YYYY-MM-DD — short title

**Symptom:** What goes wrong, observable from outside the system.

**Root cause:** What actually causes it (or "unknown" if not yet diagnosed).

**Workaround:** What to do today to keep moving.

**Real fix:** What needs to happen for this entry to be deleted.

**Owner:** Who should drive the fix (or "unassigned").

**Refs:** Code paths, related issues, prior commits.
```

## Entries
### 2026-06-29 — Eval auto-tune sweep/apply RPC deferred

**Symptom:** Dictation Studio experiment reports now recommend a winning
strategy, show corpus adequacy warnings, explain row trade-offs, and expose
per-clip diffs. They do not yet offer a backend-owned sweep/apply operation
that evaluates bounded candidate stream configs and explicitly applies the
recommended config from the experiment surface.

**Root cause:** The validated mutation path for streaming STT config is
`STTAdminService.UpdateStreamConfig`. The experiment lab intentionally owns
measurement and report semantics only; adding apply semantics requires
threading the existing STT admin writer or a shared config-apply seam rather
than creating a second JSON config writer.

**Workaround:** Use the report recommendation and row warnings to pick a
bounded lever change, then apply it through `audio-tools stt
stream-config-set` or the STT admin UI. Re-run `audio-tools experiment start
--realtime-repeats N` and compare reports to evaluate the changed config.

**Real fix:** Add a preview-first bounded sweep RPC that constructs named
candidate arms (`balanced`, `lowest_latency`, `lowest_cost`,
`highest_stability`), scores them with the same backend report semantics, and
calls the same validation/apply path as `STTAdminService.UpdateStreamConfig`
only when apply is explicit.

**Owner:** unassigned.

**Refs:** `packages/proto/schemas/audio-tools/v1/eval/eval.proto`,
`handlers/eval/`, `handlers/stt/stream_config.go`,
`docs/reference/eval-harness.md`.

### 2026-05-16 — Streaming providers declared but not implemented

**Symptom:** STT `TranscribeStream` and TTS `SynthesizeStream` RPCs work on the wire but emit zero partials, defeating the latency benefit of streaming. Every session falls back to the buffered unary path (`DoneEvent.FellBackToUnary=true`).

**Root cause:** Phases D and E of the audio-tools-web-console-restoration plan are deferred. `LocalProvider`, `BYOKProvider`, and `VrooliProvider` all declare `StreamingCapability()=false`; `chain.Stream` therefore takes the buffered fallback in every case.

**Workaround:** Functional: every consumer sees the final transcript / final audio. No functional gap, only latency.

**Real fix:** Plan Phase D: wrap `internal/voice` segmenter behind `LocalProvider.TranscribeStreaming`; emit live partials + segments. Plan Phase E: implement Deepgram WS and OpenAI Realtime STT adapters in `internal/byok`.

**Owner:** unassigned.

**Refs:** `internal/ai/sttchain/chain.go::Stream`, `internal/ai/ttschain/chain.go::Stream`, `~/.vrooli/plans/audio-tools-web-console-restoration-live-mic-diagnostics-chain-routed-streaming.md` Phases D + E.

### 2026-05-16 — WS handler not yet chain-routed

**Symptom:** `GET /api/v1/voice/stream` still calls `internal/voice/Service.HandleStreamWS` directly; chain.Stream is only reachable via the Connect bidi RPC.

**Root cause:** Plan Phase F's second half (WS handler rewire) deferred.

**Workaround:** Browser voice input still works via the legacy direct path; only BYOK / Vrooli streaming tiers are bypassed there.

**Real fix:** Rewire `handlers/stt/stream_ws.go` to upgrade the WS, marshal incoming binary frames into `sttchain.AudioChunk`, and forward `<-chan sttchain.StreamEvent` back over the wire shape `VoiceStreamProvider` (in `@audio-tools/embed`) expects.

**Owner:** unassigned.

**Refs:** `internal/voice/stream_ws.go`, `handlers/stt/stream_ws.go`, plan Phase F.

### 2026-05-16 — Strategy and provider axes are fused in the streaming path (RESOLVED)

**Resolution:** Closed by the streaming-STT decoupling plan landing 2026-05-16. `Segmenter` + `StrategySelector` are the single decision boundary; `ProviderTraits{Batch, Stream, Strategies}` replaces the old `StreamingCapability() bool`; `VADSegmentStrategy`, `OverlapAgree`, `Passthrough`, and `BufferedFallback` are the four strategies; the compatibility matrix is enforced in `internal/stt/selector.go` and tested in `selector_test.go`.

**Symptom (historical):** Adding a second streaming technique (e.g., overlap-and-agree for local Whisper) or a new BYOK adapter that needs a different technique (e.g., Deepgram passthrough vs. OpenAI Whisper API VAD-segment) currently requires either forking `voice.Service` or duplicating VAD logic inside each provider. Operators have no lever for the quality/latency tradeoff — the technique is implicit in the provider choice.

**Root cause:** `voice.Service.HandleStreamWS` hard-codes VAD + wake-word + speaker-verify + local-Whisper-batch as one fused pipeline; `sttchain.Provider` carries `StreamingCapability()`/`TranscribeStreaming` but no `ProviderTraits` capability struct, so the strategy decision has no inputs and no explicit home. The "which technique?" question is answered implicitly across multiple files.

**Workaround:** None — the current single-strategy local path works for browser voice. The cost is paid when the second strategy or second native-streaming provider lands and the code has to fork.

**Real fix:** Introduce `Segmenter`, `StrategySelector`, and `StreamingStrategy` per [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md). Replace `StreamingCapability() bool` with a `ProviderTraits` struct carrying `Batch` and `Stream` bits. Both transports call the same `Segmenter`. Compatibility matrix enforced once, in the selector.

**Owner:** unassigned.

**Refs:** [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md), `internal/voice/`, `internal/ai/sttchain/interface.go`, `handlers/stt/transcribe_stream.go`.

### 2026-05-16 — Two transports, no parity test (RESOLVED)

**Resolution:** Closed by the streaming-STT decoupling plan landing 2026-05-16. Both transports now go through `Segmenter` + `StrategySelector`; `handlers/stt/parity_connect_test.go::TestStreamingParity_ConnectBidi` drives the Connect bidi handler over an HTTP/2 httptest server and asserts event-sequence parity vs. the direct Segmenter path. WS-path parity remains pending until the browser sends raw PCM (see new entry below).

**Symptom (historical):** The browser WS path (`/api/v1/voice/stream`) and the Connect bidi path (`STTService.TranscribeStream`) reach STT through different code (`voice.Service.HandleStreamWS` vs. `Chain.Execute` accumulator). They will silently drift — a fix in one will not propagate to the other, and the proto contract claims they emit equivalent event streams.

**Root cause:** No transport-agnostic Segmenter exists yet; each transport owns its own audio-handling code. No test feeds the same audio through both paths and asserts equivalent event projections.

**Workaround:** Browser users see the WS path; non-browser users see the buffered Connect path. Behavior gap is currently masked by the fact that neither path emits live partials.

**Real fix:** Both transports become thin adapters over `Segmenter`. A `TestStreamingParity` test feeds a canned WAV through both wirings and asserts that the resulting event sequences (on a stable projection: text + ordering + final transcript) match.

**Owner:** unassigned.

**Refs:** `handlers/stt/stream_ws.go`, `handlers/stt/transcribe_stream.go`, [`../domains/stt/streaming-pipeline.md`](../domains/stt/streaming-pipeline.md).

### 2026-05-16 — Browser WebM partial-decoding regression after HandleStreamWS deletion

**Symptom:** Browsers using MediaRecorder send WebM/Opus-framed audio over `/api/v1/voice/stream`. The legacy `voice.Service.HandleStreamWS` extracted the WebM init segment and prepended it to each sub-stream slice so Whisper could decode mid-stream chunks; live partials and segment-final transcriptions worked against the WebM stream. The new `Segmenter` + strategy pipeline expects raw 16-bit PCM at 16 kHz on the chunks channel. Until the browser-side embed is updated to emit PCM (e.g. via WebAudio + AudioWorklet), only `BufferedFallback` produces a correct final transcript (Whisper decodes the complete WebM file at end-of-stream); `VADSegmentStrategy` and `OverlapAgree` will fail to decode mid-stream WebM slices and emit no useful partials.

**Root cause:** The streaming-STT decoupling plan made the strategy axis transport-agnostic — strategies see only the AudioChunk type, which is documented as raw PCM. WebM init handling was deleted along with `HandleStreamWS` because it is a browser-transport concern, not a strategy concern.

**Workaround:** Set `stt.streaming_mode=off` (or accept the auto-mode's BufferedFallback when no PCM-providing transport is wired) so the browser at least gets a correct final transcript via the buffered path.

**Real fix:** Update `@audio-tools/embed` (or the browser WS upgrade in `handlers/stt/stream_ws.go`) to transcode WebM/Opus → PCM before the bytes reach the strategy. Cleanest path: AudioWorklet in the embed emits PCM frames directly. Alternative: ffmpeg-wasm or a Go-side WebM demuxer at the WS boundary.

**Resolution (RESOLVED 2026-05-27):** Closed by the centralized audio-format substrate (`internal/audioformat`). The Segmenter now routes every PCM-consuming strategy's chunks through `audioformat`: a declared (or sniffed) codec is normalized to canonical PCM via **one long-lived ffmpeg process per session** (`audioformat.StreamDecoder`), so `VADSegmentStrategy`/`OverlapAgree` get clean PCM and emit live partials/segments against a WebM/Opus stream — no init-segment hack, server-side. `StreamStart` now carries `input_format`; the WS `format` query param + Connect `input_format` proto field declare the codec (declare-first, sniff-fallback). A `pcm_s16le` declaration takes an ffmpeg-free fast-path. When local ffmpeg is absent for a non-PCM stream the selector's capability gate downgrades to `BufferedFallback` (whole file → Whisper's own decoder), surfaced via `DoneEvent.FellBackToUnary`. The batch path and TTS egress route through the same substrate (`PrepareForWhisper` / `OutputFormat`).

**Embed PCM fast-path (RESOLVED 2026-05-27):** `VoiceStreamProvider` (`@audio-tools/embed`) no longer uses MediaRecorder/WebM. It now captures raw PCM through the `pcmCapture` seam (ScriptProcessor on the shared AudioContext — same proven pattern as `audioUtils.createPassiveCapturePipeline`; AudioWorklet migration tracked below), downsamples to canonical 16 kHz mono s16le (`hooks/voice/pcm.ts`), and declares `format=pcm_s16le` on the WS URL — so browser sessions hit the server's identity fast-path with zero server-side ffmpeg. The HTTP fallback + speaker-rejection retry wrap the captured PCM as a WAV blob. Pure conversion logic is unit-tested (`pcm.test.ts`); the capture seam is dependency-injected so `VoiceStreamProvider.tailDrop.test.ts` drives synthetic frames without a real AudioContext. **Not yet browser-validated** — the live AudioContext/ScriptProcessor wiring needs a real-browser smoke check (mic → live partials; confirm no ffmpeg process spawns for the session).

**Owner:** resolved.

### 2026-05-27 — ScriptProcessorNode → AudioWorklet migration (deferred)

**Symptom:** Two PCM capture sites use the deprecated `ScriptProcessorNode`: the wake-word `createPassiveCapturePipeline` (`hooks/voice/audioUtils.ts`) and the new streaming `createScriptProcessorPcmCapture` (`hooks/voice/pcmCapture.ts`). ScriptProcessor runs on the main thread and is deprecated in favor of `AudioWorkletNode`.

**Root cause:** ScriptProcessor is universally supported and kept deliberately for broad browser coverage; AudioWorklet requires loading a separate module script and more setup. Both sites carry `eslint-disable @typescript-eslint/no-deprecated` with this rationale.

**Real fix:** Introduce a single AudioWorklet processor that emits Float32 frames and back both capture sites with it. The `pcmCapture.PcmCaptureFactory` seam already isolates this — swap the production factory only; `VoiceStreamProvider` and its tests are unaffected.

**Owner:** unassigned.

**Refs:** `hooks/voice/pcmCapture.ts`, `hooks/voice/audioUtils.ts`.

### 2026-05-27 — Hallucination filter + confidence signals not wired (RESOLVED)

**Symptom:** Whisper narrated silence as "thank you for watching" / "please subscribe". The phrase filter `IsWhisperHallucination` existed but had ZERO callers; `TranscribeBytes` parsed only `{text}` from `/asr?output=json` (discarding `no_speech_prob`/`avg_logprob`) and never sent `vad_filter=true`, so the robust acoustic signals were thrown away and silence was never stripped before decode.

**Resolution (RESOLVED 2026-05-27):** Closed by the post-recognition **egress gate** (`internal/stt/egress/`) — the symmetric counterpart to the audioformat ingress point. The `Segmenter` builds one `egress.Gate` per session (stage set derived from the engine manifest via `sttengine.EgressStages`) and runs every `SegmentEvent` through it before the wire; strategies never call the gate. Three layers of defense: (1) `vad_filter=true` now reaches `/asr` (silence stripped at the source); (2) the signal-domain `ConfidenceStage` drops a segment when mean `no_speech_prob` > threshold AND mean `avg_logprob` < threshold (`TranscribeBytes` now parses `segments[]` into `pipeline.TranscriptionResult`, threaded via `sttchain.Result.Confidence` → `SegmentEvent.Confidence`); (3) the text-domain `HallucinationStage` wires `IsWhisperHallucination`. Dropped segments are excluded from the rebuilt `DoneEvent.FinalText`. Operator levers (`hallucination_filter_enabled`, `vad_filter_enabled`, `no_speech_threshold`, `logprob_threshold`) ship on proto `StreamConfig` + CLI; see `docs/reference/configuration.md#egress-gate`. Tested: `egress/gate_test.go`, `segmenter/egress_gate_test.go`, `pipeline/transcribe_ingress_test.go`.

**Parity update (2026-07-05):** The same stage construction is now wrapped by
`internal/stt/quality` and applied to unary Connect `Transcribe`, multipart
`/api/v1/voice/transcribe`, buffered final responses, and diagnostics STT
readiness previews. Unary responses surface `filtered`, `filter_reason`, and
`policy_details`; diagnostics keeps `diagnostic_scope=asr_readiness` while
reporting `transcript_filtered`, raw/filtered lengths, and suppressing filtered
hallucination previews. Handler tests cover VAD-filter propagation and filtered
metadata; diagnostics tests cover readiness pass with an empty filtered preview.

**Quality-smoke update (2026-07-06):** diagnostics gained a second STT layer
(`internal/diagnostics/quality_smoke.go`). Readiness (layer 1) still proves the
provider chain accepts audio; a new quality-smoke layer (layer 2) drives a
bundled silence fixture and a clean-speech fixture through the *same*
`internal/stt/quality` policy and grades them. A no-speech fixture that leaks a
surviving transcript **fails** the STT step (`error_code=quality_smoke_failed`)
even though readiness stays reachable — closing the gap where a green
diagnostics run could hide a hallucination-filter regression. Clean-speech WER
drift **warns** (never fails). Structured evidence rides the STT step details
(`quality_assessed`, `quality_status`, `quality_hallucination_detected`,
`quality_fixtures` JSON) and is rendered distinctly in CLI and UI. Corpus-grade
grading remains the Dictation Studio eval harness's job — see
`docs/reference/eval-harness.md#three-quality-surfaces-pick-by-cost-and-authority`.

**Owner:** resolved.

### 2026-05-27 — Speaker isolation ("only my voice") (RESOLVED)

**Symptom (historical):** Background music / other voices were transcribed; `SpeakerClient`/`EvaluateSpeaker` had zero callers and no embedding backend existed.

**Resolution (RESOLVED 2026-05-27):** Closed by Phase 4 of the pluggable-STT-engines plan. The audio-domain `egress.SpeakerStage` now runs the pluggable `egress.SpeakerIsolation` seam over each segment's canonical PCM; a non-enrolled voice under `filter` mode yields `Reject` → `StreamEventSpeakerRejection` (honoring `RejectBehavior` + `FallbackWithoutVerification`; `advisory` scores only; `off` omits the stage). The `verification` method (manifest `speakerIsolation.active`) wraps `pipeline.EvaluateSpeaker` against the new `resources/speaker-verification/` ECAPA service; the adapter lives in the handler layer to avoid the `egress→sttchain→pipeline` cycle. Enrollment is real: `EnrollSpeakerProfile` calls the resource `/v1/profiles` (the resource OWNS the 192-dim embedding; audio-tools stores only metadata + binding, `speaker_profiles.embedding` is now nullable), and delete purges the resource profile. Tested with fakes: `egress/speaker_stage_test.go`, `handlers/stt/speaker_isolation_test.go`, `sttengine/egress_stages_test.go`. **Live validation pending** — see the ML-resource entry below.

**Owner:** resolved.

### 2026-05-27 — Kyutai streaming engine (RESOLVED)

**Resolution (RESOLVED 2026-05-27):** Closed by Phase 3. `kyutai` is a second manifest engine (`internal/sttengine/manifest.json`, `kind=local_resource`, native-streaming, `passthrough`-only, no confidence signals). `resources/kyutai-stt/` wraps the Kyutai 1B model behind a stable JSON-over-WS contract (`/v1/stream`); `internal/ai/sttchain/provider_kyutai.go` speaks it. Both Whisper and Kyutai are Local-tier engines; the chain resolves the right provider per session from `StreamStart.EngineID` via `localEngines` (`StreamCandidates`). `Segmenter.requiresPCM` (manifest `requires.pcm16kMono`) ensures Passthrough→Kyutai still gets canonical PCM. Engine picker UI + `GetEngineSwitchImpact` informed-prompt (cross-scenario `ScanResourceConsumers`) shipped; `OverlapAgree` stays Whisper-eligible. **Live validation pending** — see below.

**Owner:** resolved.

### 2026-05-27 — Live ML-resource validation pending operator hardware (deferred)

**Symptom:** The two new ML resources (`resources/kyutai-stt/`, `resources/speaker-verification/`) are built, self-consistent, and contract-matched to their audio-tools clients, but have NOT been started end-to-end: that needs a GPU (Kyutai) + first-run multi-GB model downloads (Kyutai weights; SpeechBrain ECAPA, CPU-only).

**Status:** Deferred to the operator. Everything audio-tools-side is validated with fakes (Go build/test/lint, CLI, UI tsc/eslint/vitest, scenario restart). To validate live: build + start each resource (`vrooli resource start kyutai-stt` / `vrooli resource start speaker-verification`), confirm `/health` + `/ready` flip ready after the model download, then drive a real mic session (Kyutai partials/segments; speaker `filter` mode rejecting a second voice / background music). Kyutai enforces a single concurrent streaming session per model instance (asyncio lock) — revisit with a model pool if concurrency is needed.

**Owner:** unassigned (operator hardware).

### 2026-05-27 — Target-speaker extraction (IMPLEMENTED; empirical tuning pending)

**Symptom (original):** `SpeakerConfig.ExtractionEnabled` + `pipeline.ExtractTargetSpeaker` existed, but the `speaker-verification` resource returned HTTP 501 for `/v1/extract`; nothing isolated the enrolled speaker's audio.

**Resolution (2026-05-27):** Implemented as a pre-recognition **ingress** stage, not an egress method (egress can only drop text; isolation must substitute audio). `resources/speaker-verification` `/v1/extract` now runs SepFormer source separation + ECAPA target-selection and returns the isolated 16 kHz mono s16le PCM. The Go side adds `ingress.TargetExtractor`/`ingress.ExtractionEnhancer` (`internal/stt/ingress/extraction.go`), the handler-layer adapter `speakerExtraction` + `currentSpeakerExtraction` (built only when `extraction_enabled` + a bound profile), wired in `Segmenter.buildIngress` (config-gated like denoise — no manifest flag) and reachable via `audio-tools stt speaker-config --extraction-enabled`. The orphaned `pipeline.ExtractTargetSpeaker` (egress-shaped, zero callers) was deleted. Tested with fakes: `internal/stt/ingress/extraction_test.go`, `internal/stt/segmenter/ingress_build_test.go`, `handlers/stt/speaker_extraction_test.go`; resource non-model paths in `test/integration-test.sh`.

**Pending (environment-gated):** the separation-model spike — which SepFormer checkpoint + match threshold work best on real two-speaker audio, and CPU-vs-GPU latency — needs a GPU + model download + a live two-speaker A/B. Default OFF; degrades to passthrough if the resource/model is unavailable. See `resources/speaker-verification/docs/extraction.md`.

**Owner:** unassigned.

### 2026-05-27 — OverlapAgree commit gap (RESOLVED 2026-05-28)

**Original symptom:** `OverlapAgree` (LocalAgreement) rarely commits text. The sliding-window implementation advanced the cursor by `advanceBytes` (default WindowMs/2) but compared transcript prefixes across windows that covered *misaligned* audio spans, so the longest-agreed-prefix check rarely matched; the final tail flush emitted only `buf[cursor:]`, dropping earlier committed text.

**Resolution (2026-05-28):** Three-phase rewrite. The algorithm is now a growing-buffer LocalAgreement-N with VAD-anchored triggering, word-aligned cursor advance, and bounded agreement window.

  - **Phase A — correctness:**
    1. `longestAgreedPrefix` normalizes case + trailing punctuation (`pipeline.NormalizeToken`); previously Whisper's capitalization/punct jitter blocked agreement at position 0.
    2. `lastAdvanced` state + `appendAfterAdvance` merge function: after word-aligned cursor advance, the next hypothesis is over genuinely new audio (no overlap with committed expected); `mergeAgreed`'s divergence detector was rejecting that expected state and blocking all post-first-commit emissions.
    3. Tail flush is now unconditional via `appendAfterAdvance`: unsettled audio at channel close always reaches the user, never silently dropped.
  - **Phase B — bounded agreement:** `MaxAgreedTokens` (default 30) caps the per-iteration agreement walk so variance accumulation stays bounded on long uncommitted buffers.
  - **Phase C — VAD-anchored triggering:** `Trigger=vad` (default) replaces the stopwatch-based `nextTriggerAt` with frame RMS analysis (reusing the same logic as `VADSegmenter`). Settle attempts fire on silence boundaries — Whisper sees clean audio edges, agreement happens reliably. `MaxWindowMs` safety net unchanged. `Trigger=stopwatch` preserved for legacy/test use.

**Test coverage:**
  - `api/internal/stt/strategy/overlap_agree_behavior_test.go` —
    `TestOverlapAgree_NormalizesCaseAndPunctuationForAgreement`,
    `TestOverlapAgree_PostAdvanceCommitsContinue`,
    `TestOverlapAgree_TailFlushEmitsEvenOnDivergence`,
    `TestOverlapAgree_BoundedAgreementWindowSurvivesLongJitter`,
    `TestOverlapAgree_VAD_TriggersOnSilenceBoundary`,
    `TestOverlapAgree_VAD_NoTriggerWithoutSilence`,
    `TestOverlapAgree_VAD_MaxWindowForcedFallback`,
    `TestOverlapAgree_VAD_TailFlushOnChannelClose`.
  - `api/internal/stt/strategy/overlap_agree_internal_test.go` —
    `TestLongestAgreedPrefix_MaxTokensCap`,
    `TestLongestAgreedPrefix_CaseAndPunctuationNormalization`.

**Default flip:** selector `auto` now resolves to `OverlapAgree` for batch-only Local Whisper. `vad` remains available as an explicit preference for operators who want one-segment-per-utterance behaviour.

**Refs:** `api/internal/stt/strategy/overlap_agree.go`, `api/internal/stt/selector.go` auto branch.

**Owner:** resolved.

### 2026-05-27 — Whisper 5-concurrent ceiling (known capacity limit)

**Symptom:** The local Whisper resource accepts at most 5 concurrent `/asr` requests (`resources/whisper/docs/API.md`). With many simultaneous streaming sessions (each calling `Transcribe` per VAD segment) plus batch uploads, the sidecar is the real throughput wall — upstream of the audio-format layer.

**Mitigation (in place):** `pipeline.Service` bounds concurrent Whisper calls with a semaphore (`DefaultWhisperConcurrency = 5`); over-limit callers **block (queue with backpressure), never error**, and a cancelled session's ctx releases its place in line. The audio-format substrate must not mask this ceiling — one user looks fine, ten queue. Resizable via `SetWhisperConcurrency`.

**Status:** Known limit, bounded. Raising it requires scaling the Whisper resource (more workers/replicas), not a code change here.

**Owner:** resolved (bounded).

**Refs:** `handlers/stt/stream_ws.go`, `internal/stt/strategy/webm.go`, `internal/stt/strategy/vad_segment.go`, `scenarios/audio-tools/embed/`.

### 2026-05-16 — WS endpoint tag still `ops_probe`

**Symptom:** `GET /api/v1/voice/stream` is tagged `RESTReason: ops_probe`; semantically it is a `TransportReason: websocket_transport`. Cosmetic only.

**Root cause:** The `websocket_transport` template constant in `internal/module` does not exist yet (tracked in R-PROTO upstream).

**Workaround:** Live with the mistag; documented in `api-endpoints.md` "Transport exceptions" with an explicit note.

**Real fix:** Retag when the template constant lands.

**Owner:** unassigned.

**Refs:** `docs/reference/api-endpoints.md` "Transport exceptions" table.

### 2026-06-29 — Dictation Studio BAS cannot prove physical microphone capture

**Symptom:** The deterministic BAS flow can validate Dictation Studio's scripted
bench UI, but it cannot prove the browser is receiving real microphone audio or
that a physical recording can be saved and replayed through Eval.

**Root cause:** The shared BAS toolchain does not yet provide mic-permission
grants, deterministic media-stream fixtures, or fake audio-device routing for
the physical capture path.

**Workaround:** Run manual operator validation: select a built-in script, record
with a real mic, confirm the audio meter moves, stop into transcribing,
correct/save the clip, confirm it appears in Corpus, then run Eval.

**Real fix:** Add BAS media-stream fixtures/fake audio routing, then promote a
Dictation Studio real record -> save -> eval workflow from observer coverage to
requirement-bound playbook coverage.

**Owner:** unassigned.

**Refs:** `scenarios/audio-tools/bas/flows/audio-tools-dictation-studio-scripted-bench.json`,
`scenarios/audio-tools/docs/reference/eval-harness.md`.

### 2026-07-11 - BAS selector manifest qualification and streaming proxy blocker

Persisted BAS workflows now compile selectors from Audio Tools ui/src/consts/selectors.manifest.json through their project root. Execution 3c21f0f4-d75a-4e7a-9704-519aecdbd6d5 completed all browser controls with the WAV fake microphone, but it is not successful PCM-v2 evidence: Audio Tools runtime logs show the normal UI proxy forwarding /api/v1/voice/stream as a plain GET without WebSocket Upgrade, producing HTTP 400. Filed scenario-qa code-defect knw-1783819641843118934. ATD-P0-006 automation therefore remains planned; deterministic P0-005 faults are now authorized by the server-owned routed-isolation lease rather than a boot-only switch.

### 2026-07-11 - Test Genie cannot provision BAS fake-microphone fixtures (RESOLVED 2026-07-12)

**Symptom:** The standard Audio Tools workflow phase times out in deterministic-microphone-smoke after the repository workflow schema is valid.\n\n**Root cause:** Chromium fake capture was configured only by the Browser Automation Studio driver process environment (BAS_FAKE_MICROPHONE_FILE). Test Genie could run the playbook but had no per-workflow fixture provisioning path, so its normal BAS driver had no fake device.\n\n**Resolution:** BAS now supports a per-execution deterministic-media capability: a workflow declares `settings.fake_media.microphone_wav` (path resolved against the execution's `project_root` and required to stay within it), the API threads it to the Playwright driver, and the driver pools a dedicated Chromium instance per distinct WAV with the fake-capture launch flags plus context-level microphone permission. `deterministic-microphone-smoke.json` now carries `settings.fake_media.microphone_wav = fixtures/dictation-reference.wav` and passed on a standard (non-specially-booted) driver: BAS execution 3676aa0c-5fce-4159-bf93-17319d2e5d30 (2026-07-12) proved captured PCM, v2 send, done delivery, and processed acknowledgement. BAS_FAKE_MICROPHONE_FILE remains only as the default-browser fallback for dedicated qualification drivers.\n\n**Owner:** resolved.\n\n**Refs:** bas/cases/01-foundation/01-dictation/deterministic-microphone-smoke.json; bas/fixtures/dictation-reference.wav; browser-automation-studio/playwright-driver/src/session/browser-manager.ts; packages/proto/schemas/browser-automation-studio/v1/workflows/definition.proto (FakeMediaConfig).

### 2026-07-11 - P0 dictation requirements are in-progress pending live qualification

**Symptom:** The P0 registry previously showed  even where focused unit and deterministic-browser evidence had passed, while its test-only validations could eventually allow an over-broad completion status.\n\n**Root cause:** The original requirement entries had no separate validation for their real-resource and manual qualification gates.\n\n**Workaround:** The registry now marks verified test and BAS validation as implemented, leaves each P0 requirement , and records a planned manual gate for the outstanding full provider profile, live admission/recovery, cross-engine speaker policy, consumer adapters, and device evidence.\n\n**Real fix:** Complete and record those gates through the qualification harness and ; only then may the full sync promote the requirements.\n\n**Owner:** audio-tools provider-parity plan.\n\n**Refs:** requirements/01-must-ship/module.json; provider experiments e0a2596e-91e9-4e4f-8b44-b2bec7b79e3d, ddf20231-47c4-4da6-803e-1e5e33ae3fb0, 4e5505e6-3e61-405f-88ab-b9a83cc2538a.

### 2026-07-11 - Correction: P0 dictation qualification gates

**Correction:** The preceding entry's wording was damaged by shell interpolation. Treat this entry as authoritative.\n\n**State:** Each P0 dictation requirement is in_progress. Focused unit and deterministic-browser validations are implemented. A planned manual validation remains for each real-resource and device gate, so a future full sync cannot mark a P0 claim complete from unit evidence alone.\n\n**Completion path:** Run and record the full provider duration, interruption, recovery, fault, browser, consumer-adapter, cross-engine speaker-policy, and device qualification profiles. Then record manual evidence with the scenario requirements manual-log command and let the normal full-suite sync derive completion.\n\n**Refs:** requirements/01-must-ship/module.json; experiments e0a2596e-91e9-4e4f-8b44-b2bec7b79e3d, ddf20231-47c4-4da6-803e-1e5e33ae3fb0, 4e5505e6-3e61-405f-88ab-b9a83cc2538a.

### 2026-07-23 - Eval runner is now transport-free

Evaluation replay/session construction moved from handlers/eval into internal/eval with explicit corpus and speaker adapter ports. Bootstrap invokes the internal runner directly; remaining bootstrap handler imports are transport composition and speaker adapter construction pending experiment-engine relocation. Evidence: GOWORK=off go test ./... passes on 2026-07-23.

### 2026-07-23 - Bootstrap transport and experiment boundaries were separated

The experiment report workflow now lives in internal/experiment/report and HTTP module composition now lives in internal/transport. api/internal/bootstrap/bootstrap.go is 312 lines and has no handler imports; bootstrap's package files have no handler imports. Full API suite passed with GOWORK=off go test ./... on 2026-07-23. Remaining architecture findings still require a fresh evidence-manifest-producing architecture run.
## Architecture Drift

Use this section for deferred findings from `screaming-architecture-audit`.
Do not create a standalone architecture-audit report unless the work is
a migration handoff with a planned retirement path back into
`ARCHITECTURE.md`, `SEAMS.md`, or this file.

| Area | Drift | Maturity Impact | Real Fix |
|---|---|---|---|
| ~~L3 ambient leaks~~ | **Resolved** 2026-05-17. Every domain that touched `time.Now`, `os.Getenv`, `http.DefaultClient`, or `log.Printf` was migrated to the `clock.Clock`, `envx.Reader`, `logx.Logger`, or `httpc.Doer` seams (or a package-level seam variable where threading constructors would have ballooned the diff — `internal/store/clockx.go`, `internal/stt/pipeline.packageLogger`, `internal/httpx.packageLogger`, `internal/tts.configLogger`, `internal/summarize.configLogger`). Only `internal/clock`, `internal/envx`, `internal/logx`, and `internal/bootstrap` (composition root, plan §4 exempt) now contain the seam-bypassing primitives. | Closed. | L5 enforcement: the L3 grep in plan §9 over `internal/` excluding the exempted packages returns zero non-comment hits. |
| ~~Inline test fakes~~ | **Resolved** 2026-05-17. Eleven inline `fake*`/`stub*` test structs were hoisted into per-domain `mocks/` packages: `internal/audio/mocks/runner.go` (`FakeRunner` + `Call`), `internal/capabilities/mocks/checker.go` (`FakeChecker` with atomic counters), `internal/ai/sttchain/mocks/{provider,byok,vrooli}.go` (`FakeProvider`, `FakeBYOK`, `FakeVrooliClient`, `FakeBatchExecutor`), `internal/ai/ttschain/mocks/{byok,vrooli}.go`, and `internal/ai/summarizechain/mocks/{byok,vrooli}.go`. Every hoisted fake carries a `var _ <Iface> = (*Fake)(nil)` compile-time assertion. `grep -rn '^type \(fake\|mock\|stub\)\w\+ struct' --include='*_test.go' .` returns zero. | Closed. | — |
| ~~Sleep-based test waits~~ | **Resolved** 2026-05-17. `time.Sleep` removed from all six test sites: `integrations/lpbs/remote_reporter_test.go` → `require.Eventually`; `internal/usagereport/recorder_test.go` → `require.Eventually`; `internal/session/session_test.go` → `require.Eventually` over the per-observer counters; `handlers/session/handler_test.go` → buffered-channel reliance (no readiness wait needed); `internal/summarize/summarizer_test.go` → server blocks on a release channel + LIFO defer ordering; `main_e2e_test.go::waitForHealth` → `time.Ticker` poll loop. The acceptance `grep -rn 'time\.Sleep' --include='*_test.go' . | grep -v testutil/mocks/clock_test.go` returns zero. | Closed. | — |
| ~~Coverage floors~~ | **Resolved** 2026-05-17. `TestCoverageFloors` enforces the committed per-package floors under `-race`; its output names each floor breach. | Closed. | The Go test is the L4 evidence command; the floor policy is versioned in-band with the suite. |

### 2026-05-17 — Browser-WS `/api/v1/voice/stream` lacks a transport test

**Symptom:** No automated test exercises the WS handshake, message echo, abrupt close, or context-cancel propagation paths in `api/handlers/stt/stream_ws.go`. Regressions surface only via the UI smoke run.

**Root cause:** Deferred from the 2026-05-17 post-extraction cleanup. A meaningful WS test requires a `httptest`/`gorilla/websocket` rig wired to the same session pub/sub fan-out the production transport uses; the fan-out seam exists (`internal/session/session.go`) but the handler-level fake is not yet hoisted into `handlers/stt/mocks/`.

**Workaround:** Manual UI smoke after restart confirms handshake + echo. Connect-RPC bidi `TranscribeStream` is covered end-to-end and shares the segmenter + strategy path, so most regressions surface there.

### 2026-05-17 — Summarize default awaits newer-model local benchmark

**Symptom:** The summarize catalog includes newer research-backed candidates
(`gemma3:4b`, `gemma3n:e2b`, `phi4-mini:3.8b`), but the default remains
`llama3.2:3b`.

**Root cause:** The target machine has Ollama 0.11.7 and installed
`llama3.2:3b`, `llama3.2:1b`, `qwen2.5:3b`, `mistral:latest`, `qwen3:*`,
and `deepseek-r1:8b`, but not Gemma 3, Gemma 3n, or Phi-4 Mini. The system
must not pull large model artifacts without explicit operator approval.

**Workaround:** Use the web-console model picker. Missing recommended models
show `ollama pull <model>` commands for manual installation.

**Real fix:** After an operator installs candidates, benchmark them against
representative assistant-output fixtures. Change the default only if a
non-reasoning candidate beats `llama3.2:3b` on latency and preserves key facts
without prompt leakage or visible reasoning.

**Owner:** unassigned.

**Refs:** `api/internal/summarize/model_policy.go`,
`api/internal/summarize/model_policy_test.go`,
`packages/proto/schemas/audio-tools/v1/summarize/summarize.proto`.

**Real fix:** Hoist a `handlers/stt/mocks/ws.go` helper and add `stream_ws_test.go` with the four scenarios from plan §G2 (`audio-tools-post-extraction-cleanup-professionalization.md`).

**Owner:** unassigned.

**Refs:** plan §G2; `api/handlers/stt/stream_ws.go`.

**Resolved 2026-05-17:** `api/handlers/stt/stream_ws_test.go` now covers handshake + terminal Done, abrupt client close, and server-context cancel using a `newNoProviderDeps` rig (real Chain+Selector with no providers → BufferedFallback emits a terminal sequence). Per plan `audio-tools-test-quality-coverage-and-seam-hardening.md` §Phase 3.

### 2026-05-17 — No cross-domain integration test

**Symptom:** Transcribe → summarize → synthesize is exercised only by per-handler unit tests; a regression in the cross-domain seam (chain wiring, normalizer ownership) would not fail a single test.

**Root cause:** Deferred from the 2026-05-17 post-extraction cleanup. The work needs deterministic stubs for all three external provider tiers in `internal/testutil/` and a new `api/tests/integration/` package.

**Workaround:** `vrooli scenario test audio-tools` exercises the per-handler happy paths against real chains with local providers when present; the cross-domain happy path is the diagnostics UI tab path.

**Real fix:** Add `api/tests/integration/cross_domain_test.go` per plan §G3 once the WS deferral above is also being worked.

**Owner:** unassigned.

**Refs:** plan §G3; `api/internal/testutil/`.

**Resolved 2026-05-17:** `api/tests/integration/cross_domain_test.go` exercises the transcribe → summarize → synthesize sequence via per-chain fakes from `internal/diagnostics/mocks/` (hoisted in the same plan). Covers happy path, summarize-failure short-circuit, and empty-audio TTS branch. Per plan `audio-tools-test-quality-coverage-and-seam-hardening.md` §Phase 4.

### 2026-05-17 — Pre-existing standards drift not in cleanup scope

**Symptom:** After the 2026-05-17 cleanup, `scenario-auditor standards scan audio-tools` reports 6 violations (down from 29 at start of the cleanup):

- (critical) `Makefile` "required_layout" — the rule message ("Add the required resource at Makefile") is opaque; reference scenarios like `agent-inbox` pass with a similar Makefile shape, suggesting the rule may be probing for a specific resource file rather than a target.
- (high) `requirements/index.json` requirement `FOUND-001` `prd_ref` points to a PRD section that does not exist in `PRD.md`.
- (medium) `requirements/01-foundation/module.json` lacks operational-target linkage for `FOUND-001`.
- (medium) `ui/src/components/shell/SettingsDrawer.tsx` registers a `keydown` listener inline rather than via a dedicated `ui/src/hooks/` hook.
- (medium) `emitShortcutIntent` is not called from any keyboard hook in `ui/src/hooks/`.
- (medium) `api/internal/stt/segmenter/testaudio/` (fixtures-only package) has no `*_test.go`.

**Root cause:** All six predate the post-extraction cleanup plan. Fixing them requires substrate-level work (PRD-schema edits, a shared keyboard hook + `emitShortcutIntent` wiring, a standards-rule waiver for fixture-only packages, and reverse-engineering the `required_layout` resource check) that is out of scope per the plan §4 ("Cross-scenario refactors" and "No new product features").

**Workaround:** The cleanup pass reduced the violation count from 29 → 6. The remaining ones do not block `go test`, `pnpm test`, or runtime smoke; they only fail the `standards` phase of `vrooli scenario test audio-tools`.

**Real fix:** A separate cleanup plan should pick these up alongside the next round of standards-rule updates so the changes ride a single deploy.

**Owner:** unassigned.

**Refs:** plan §I; `scenario-auditor standards scan audio-tools` (last run 2026-05-17, job `standards-f069d6b4-…`).

### 2026-05-17 — `audio` domain scope is intentionally smaller than the proto surface

**Symptom:** Proto `audio-tools/v1/audio` declares Transcode + ops envelopes for trim/merge/split/fade/volume/normalize/metadata, but the only end-to-end (handler + CLI + UI) flow is Transcode.

**Root cause:** Post-extraction the `audio` domain scope was explicitly limited to "Transcode + ffprobe-backed primitives" so the cleanup did not balloon. The primitives exist in `api/internal/audio/` (trim/merge/split/fade/volume/normalize) and are exercised by `ops_test.go`, but no CLI verb or UI panel exposes them.

**Workaround:** Callers that need those ops can drive them via the multipart REST exception once handlers are added; for now use ffmpeg out-of-band.

**Real fix:** When a downstream scenario demands a second audio op end-to-end, lift the scope decision per [`DECISIONS.md`](DECISIONS.md) (2026-05-17 entry) and implement handler + CLI + UI for the new op set.

**Owner:** unassigned.

**Refs:** `docs/domains/audio/`; plan §D.

## Test Gaps

The "audio-tools-test-architecture-lift" plan (2026-05-17) closed the
0%-coverage chains (sttchain ≥85%, ttschain ≥85%, summarizechain ≥85%)
and the legacy audio-ops gap. The remaining shortfalls below are
intentional deferrals governed by separate plans.

### `internal/stt/pipeline/service.go` — legacy `voice.Service` shim

**Symptom:** Package coverage sits at ~24% with 28 functions still at
0.0%. Most are wrappers (`Transcribe`, `Diarize`, `SpeakerProfile`
shapes) that the new `internal/stt/segmenter` + chain pipeline supersedes.

**Root cause:** The WS handler (`handlers/stt/stream_ws.go`) and a few
admin endpoints still call into the legacy `voice.Service` until the
streaming-decoupling plan rewires them through `sttchain.Chain.Stream`
and the strategy selector.

**Workaround:** Do not grow new tests against the legacy shim. New
behaviour goes through `sttchain` + `segmenter`, which carry their own
coverage floors (85% / 80%).

**Real fix:** "Streaming pipeline migration" — owned by the PRD that
moves `stream_ws.go` to the strategy selector. Once the legacy paths
have no callers, the file is deleted.

**Owner:** unassigned.

**Refs:** [`SEAMS.md`](SEAMS.md) "sttchain.Provider" row; plan
[audio-tools-test-architecture-lift §4 Out of scope].

### `internal/byok/{deepgram,openai_whisper}.go` streaming methods

**Symptom:** `TranscribeStreaming` on the Deepgram + OpenAI Whisper
adapters is ~5% covered. The non-streaming `Transcribe` paths and
non-streaming adapters (openai_tts, elevenlabs, openrouter) sit ≥80%.

**Root cause:** Streaming requires a real WS upstream. Asserting against
a Deepgram-shaped WS handshake means standing up a fake vendor WS
server per test — substantial work for a single adapter branch.

**Workaround:** The streaming dispatch routing IS covered through the
`BYOKProvider.TranscribeStreaming` shape tests; only the adapter-side
WS wire format is uncovered.

**Real fix:** Add `vendor-ws` test fixtures alongside the streaming
plan that ships native streaming to production. Until then, regressions
are caught by the smoke-test phase of `vrooli scenario test`.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptom 1];
streaming-decoupling plan in [`PRD.md`](../../PRD.md).

**Resolved 2026-05-17 (Deepgram only):** The vendor-WS rig at
`internal/testutil/vendorws/` now drives three new
`internal/byok/deepgram_test.go` cases — happy-path (partial + final
frames), mid-stream 1011 close, and context-cancel cleanup — through
the injectable `DeepgramSTT.StreamEndpoint`. Package coverage:
`internal/byok` 87% (floor raised 70 → 80). OpenAI Whisper still
declares `StreamingCapability() bool { return false }` and
`TranscribeStreaming` returns `(nil, nil)`; the streaming-decoupling
plan's Phase E will land the real adapter and the matching
vendorws-driven tests in the same diff. Per plan
`audio-tools-test-follow-ups-new-handler-seams-logx-adoption-ui-cli-coverage-byok-streaming-rig.md`
Phase 6.

### Handler `time.Now()` and `log.Printf` bypasses (Phase 2 / Phase 6 deferred)

**Symptom:** Five handler files still call `time.Now()` directly
(`handlers/{summarize,session,tts,usage,stt}/...`) and
`handlers/stt/stream_ws.go` still uses `log.Printf`.

**Root cause:** The substitution wiring (Clock on Deps, Logger on Deps)
is straightforward but touches every module constructor + handler test;
the test-architecture-lift plan deferred this to a focused follow-up so
the chain-coverage work could land independently.

**Workaround:** Tests assert presence/monotonicity of timestamps via
the protobuf shape rather than exact bytes; WS error tests rely on the
status-code branch rather than captured log lines.

**Real fix:** Follow-up plan "handler clock + logger seam wiring" —
mechanical refactor across the listed files.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptoms 3, 4]
(Phase 2 + Phase 6 carry-over).

**Resolved 2026-05-17:** Every handler `Deps` now declares
`Logger logx.Logger` and (where time is read) `Clock clock.Clock` as
required fields — no `if x == nil { ... }` fallbacks. The two new
handlers (`handlers/health_status`, `handlers/provider_lifecycle`)
were wired through the seam from inception; the remaining ten handler
packages and three `internal/` shims (`middleware`, `tts/service`,
`usagereport/recorder`, plus `internal/server`) were converted in the
same pass. Drift gate: `rg -n 'log\.Default\(\)' scenarios/audio-tools/api/ -g '!*_test.go' -g '!internal/bootstrap/**' -g '!internal/logx/**' -g '!main.go'` returns empty;
`rg -n 'time\.Now\(\)' scenarios/audio-tools/api/handlers/ -g '!*_test.go'` returns empty. Tests substitute `mocks.FakeClock` +
`mocks.FakeLogger`; the two new handlers assert exact RFC3339
timestamps and at least one log line per error path. Per plan
`audio-tools-test-follow-ups-new-handler-seams-logx-adoption-ui-cli-coverage-byok-streaming-rig.md`
Phases 1–2.

### BYOK + LPBS client `*http.Client` direct construction (Phase 1 deferred)

**Symptom:** Eight files (`internal/byok/{openrouter,openai_tts,openai_whisper,deepgram,elevenlabs}.go` and
`integrations/lpbs/clients/{stt,tts,summarize}_client.go`) construct
their own `*http.Client` instead of accepting `httpc.Doer`.

**Root cause:** The `httpc.Doer` seam is declared and `mocks.FakeDoer`
ships ready, but threading the field through every constructor +
registry + `main.go` is invasive enough that the test-architecture-lift
plan split it out.

**Workaround:** Existing adapter tests use `httptest.NewServer` for
wire-format coverage; payload drift is caught indirectly via the
provider-level chain tests.

**Real fix:** Follow-up plan "BYOK adapter Doer wiring" — replaces
`HTTPClient *http.Client` with `Doer httpc.Doer` and adds FakeDoer
payload-shape tests alongside the existing httptest wire-format tests.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptom 2]
(Phase 1 carry-over).

**Resolved 2026-05-17:** The remaining `&http.Client{}` callsites
(`internal/tts/kokoro_{synthesize,voices}.go`,
`internal/summarize/summarizer.go`,
`internal/capabilities/checkers{,_audio,_llm}.go`,
`internal/stt/pipeline/{speaker_client,service}.go`) now accept
`httpc.Doer` as a required field/parameter; BYOK + LPBS clients were
migrated in the predecessor pass. Production wires
`httpc.DefaultDoer()` from `internal/bootstrap`. Per plan
`audio-tools-test-quality-coverage-and-seam-hardening.md` §Phase 1.

### Consumer-side `store.*Repository` seams (Phase 5 deferred)

**Symptom:** Admin handlers in `handlers/{settings,usage,tts,stt}/`
import `internal/store` concrete sqlite types directly; handler tests
must spin SQLite to drive success paths and cannot inject error
branches.

**Root cause:** Per-handler `Repository` interface introduction +
`var _` checks + fake substitution is a multi-package refactor.

**Workaround:** Existing sqlite-backed handler tests already cover the
happy paths; SQL semantics are pinned in
`internal/store/*_sqlite_test.go`.

**Real fix:** Follow-up plan "consumer-side repository seams" —
introduces narrow `Repository` interfaces in each handler package and
co-located `mocks/repository.go` fakes.

**Owner:** unassigned.

**Refs:** plan [audio-tools-test-architecture-lift §3 Symptom 7]
(Phase 5 carry-over).

**Resolved 2026-05-17:** Per-handler `Repository` interfaces with
`var _ Repository = (*store.X)(nil)` checks now live in
`handlers/{settings,usage,tts,stt}/repository.go`; co-located
`handlers/*/mocks/` fakes are wired in their existing handler tests.
Per plan `audio-tools-test-quality-coverage-and-seam-hardening.md`
§Phase 5.

### Kyutai/Passthrough segments are not speaker-gated

**Symptom:** Speaker verification only protects the Whisper VAD path. The
Kyutai streaming engine (and any Passthrough strategy) emits segments with no
`seg.Audio`, so the egress `SpeakerStage` returns them unchanged — a non-enrolled
voice transcribed via Kyutai is NOT rejected.

**Why unsolved:** Identity gating needs the segment PCM, which the streaming
passthrough path never carries. A real fix means either teeing the session PCM
to a side-channel verifier or having the engine surface per-segment audio.

**Workaround:** Use the Whisper engine when speaker isolation must be enforced;
the admin status surface and CLI `speaker-status` both print this caveat.

**Owner:** unassigned. **Refs:** plan
`speaker-verification-quality-overhaul-tiers-1-2` §Non-goals.

### Speaker threshold needs live calibration

**Symptom:** The canonical default `speaker.threshold` is `0.5` across all
layers, but the right cutoff depends on the ECAPA embedding distribution for the
actual enrolled voices + room. The plan ships VAD trim + multi-clip centroid +
session smoothing (which recover most of the lost separation) but the final
number is environment-specific.

**Resolution path:** After Phase 1 (VAD trim) is live, enroll 3–5 real clips and
run a genuine-vs-impostor mic test; set `speaker.threshold` to sit between the
two bands and record it in `docs/reference/configuration.md`. This cannot be
unit-tested (needs real voices + the model).

**Owner:** unassigned (operator calibration step).

### Resource integration harness has a broken source path

**Symptom:** `resources/speaker-verification/test/integration-test.sh` fails at
startup — `scripts/resources/common.sh` sources a missing
`scripts/lib/service/repository.sh`. This is pre-existing shared-harness drift,
not specific to this resource; the speaker contract itself is exercised by the
in-image `python -m unittest` suites and live curl checks.

**Owner:** unassigned (shared resource-test harness).

### 2026-07-08 — Continuous-speech streaming loses all text under load (RESOLVED)

**Symptom:** Dictating continuously into web-console with kyutai (default since
2026-07-06) worked for ~10 s, then live updates stopped and ALL further speech
was lost — even after stopping the mic; only a pause temporarily cleared it. Its
TTS twin: a fault in one paragraph of a multi-paragraph spoken reply truncated
every remaining paragraph.

**Root cause:** A fully-synchronous, tiny-buffer streaming chain across two
WebSocket hops with blocking writes at every stage. A browser consumer that
couldn't sustain the unthrottled partial firehose back-pressured every hop
(~32 slots of egress buffer) until the kyutai decode loop froze mid-emit and
STOPPED consuming audio — total loss, not a tail. Two aggravating bugs made it
unrecoverable on stop: the provider held `writeMu` across a blocking write (so
cancel's `sendEnd` deadlocked and drain-then-close never ran), and the client's
single-slot trailing-partial could only recover the last stale partial. A wedged
session then starved the next recording via kyutai's 1-stream model lock.

**Real fix (shipped):** One documented event-durability contract applied at
every hop (`docs/domains/stt/streaming-pipeline.md#event-durability-contract`,
predicate `sttchain.StreamEvent.Durable()`): partials are disposable
(coalesce-to-latest, droppable, never back-pressure their producer); segments /
rejections / errors / done are durable (ordered, lossless). Decode is decoupled
from send at each hop — kyutai `server.py` send worker + bounded queue + lock
reap; the provider's dedicated writer goroutine (no lock across a blocking
write); the browser WS handler's coalescing writer; and the client's
committed-length cursor (`uncommittedRemainder`) + rAF-coalesced partial render
+ bounded `allChunks`. The TTS twin was already isolated per-paragraph in the
client `speakSequence` (retry → browser-fallback → skip-with-notice, continues
in order); it is now cross-referenced to the same contract. Guarded permanently
by red-first oracles at every layer plus an always-on continuous-speech delivery
assertion wired to the drop counter.

**Owner:** shipped — `~/.vrooli/plans/streaming-stt-continuous-speech-backpressure-wedge-proper.md`.

**Live validation (DONE 2026-07-08):** verified end-to-end against the DEPLOYED
stack (rebuilt kyutai-stt image + restarted audio-tools relay; GPU is present —
RTX 4070 Ti SUPER — so this needs no special hardware beyond the running host). A
synthetic client streamed the real web-console wire format (webm/opus, 250 ms
timeslices, real-time paced) into `/api/v1/voice/stream` for 65 s of continuous
speech: (a) baseline fast-reader → 17 segments + done, full coverage; (b) the
wedge trigger — the browser consumer FROZE (stopped reading) for a full 20 s
mid-stream, far past the old ~10 s wedge point → on resume every segment spoken
during the freeze burst out losslessly in order, **17 segments total (identical
to baseline), zero loss, clean done**; (c) starvation — a stalled session
abandoned mid-stream did NOT block the next recording (fresh session transcribed
+ done immediately). Direct kyutai probe confirms the resource itself decouples
decode from send (37 partials / 3 segments / done through a slow consumer).
Remaining genuinely-optional check: a physical-mic dogfood (real getUserMedia),
which the automated webm/opus path already stands in for at the protocol level.

**Refs:** `resources/kyutai-stt/docker/server.py`,
`internal/ai/sttchain/provider_kyutai.go`, `handlers/stt/stream_ws.go`,
`scenarios/web-console/ui/src/audio-integration/hooks/{useVoiceCore.ts,voice/trailingPartial.ts,voice/VoiceStreamProvider.ts,tts/KokoroProvider.ts}`.

**Follow-up REVEALED by this fix (2026-07-08, web-console client, under triage):**
With the backpressure wedge closed, mid-recording text is no longer lost — but a
*different* symptom surfaced: during continuous speech the **microphone abruptly
stops** ~10 s in (the mic-state is now honestly synced to transcription, so a
premature turn-end is visible instead of masked by the frozen decode loop).
Root cause is CLIENT-side, not this relay: **kyutai/passthrough emits NO
`vad-state`** (grep `server.py`/`provider_kyutai.go` — none), so on the browser
the auto-stop SSOT (`decideAutoStop` §2) falls back to the **client RMS VAD as
the sole turn-ending authority**. Under whisper-VADSegment the server VAD was
authoritative and vetoed the client VAD's known "stops while I'm still talking"
false positives; under kyutai that veto is gone. A client-VAD silence verdict
then ends the whole turn whenever the analyser reads silence — a **muted mic
track** (sleep/wake, default-device change, another app seizing the mic; the
`ended` event does NOT fire for muting — see web-console `streamHealth.ts`), a
**suspended AudioContext**, or an adaptive `speechThreshold` risen above the
user's real speech RMS. Relay evidence: sessions close `graceful=true
tailFinalDelivered=true` in a regular ~10–12 s cadence (client-initiated stops,
not backend kills). The frequent kyutai "reaping wedged prior streaming session"
warnings are a *separate*, benign-but-noisy lock-release-latency issue (the relay
dials kyutai + takes `MODEL.lock` the instant the browser WS connects — including
the mount-time pre-connect — so overlapping/rapid sessions reap already-delivered
holders; not the cause of the mic stop). Web-console side owns the fix; see its
`PROBLEMS.md §8c`. First step done: full client instrumentation + honest
track-mute/ended/recorder-error handling + a guard that suppresses a client-VAD
auto-stop when the audio source is not delivering samples.

### 2026-07-12 - Kyutai deterministic long-form replay remains an unresolved flow-control blocker

**Symptom:** The persisted 15-minute provider profile completed the paced
Kyutai pass (11,337 frames, 154 segments, graceful end) but its preceding
deterministic quality pass closed after 4,330 frames and 57 segments with
`1011 keepalive ping timeout`. The incomplete quality transcript had WER 0.663
and 638 deleted reference words; the matching Whisper row had WER 0.042.

**Confirmed contributing defect:** Fast replay generates partials faster than
the Go `KyutaiProvider` downstream consumer can drain them. The adapter used to
synchronously write every partial into its 16-slot `events` channel, which could
stop `ReadMessage`. `provider_kyutai.go` now uses a nonblocking bounded send for
the explicitly droppable partial class; segments, session states, errors, and
done remain ordered durable delivery. The 60 MiB
`TestKyutaiProvider_PartialFloodDoesNotBlockWebSocketReader` oracle failed before
the change and passes after it.

**Current status (superseded by the 2026-07-12 addendum below):** The full
deterministic native-streaming evaluation still closed at the same 50-second
boundary after this repair (1,865 frames / 24 segments), so partial pressure
was not the sole cause. The server reported `disconnect`, not `reaped`, and the
paced pass proved the same model/audio path could complete. That isolated the
native transport's unbounded fast writer as the remaining suspect.

**Interim evidence correction:** A `realtime` evaluation cell now derives
WER/safety from its first paced repeat and no longer attaches an incompatible
unpaced deterministic failure to a row labeled `realtime`. An explicitly
deterministic Kyutai cell remains a required, currently failing qualification
until a real bounded transport-flow-control fix exists; this is not waived for
promotion.

**Validation:** Focused `go test ./internal/ai/sttchain ./handlers/stt` passes.
The paced 15-minute rerun is experiment `71a8ea70-2a4a-4a9e-9e5a-f46dbd2191a4`.

**Refs:** `api/internal/ai/sttchain/provider_kyutai.go`;
`api/internal/ai/sttchain/provider_kyutai_test.go`; experiments
`ddf20231-47c4-4da6-803e-1e5e33ae3fb0`,
`71a8ea70-2a4a-4a9e-9e5a-f46dbd2191a4`; record `rec-6d413e62862befd6`.

### 2026-07-12 — Deterministic experiment runtime is bounded; Kyutai decode throughput remains below budget

**Observed:** After bounded `processed_batches` credits were deployed, the
former 1011 disconnect no longer occurred, but a 15-minute deterministic run
(`02f07e9e-aff1-4714-b1b8-2b6b7d3583ca`) was still active after 40 minutes and
was cancelled as invalid evidence. A controlled 30-second, 100ms-chunk Kyutai
probe then exceeded its 2m30s runtime budget. The same audio with 1-second
chunks completed in about 30 seconds before an abnormal normal-close; adding a
cooperative `asyncio.sleep(0)` every four model frames removed that early close,
but the probe still exceeded the 2m30s budget. The resource is healthy and
loaded on the RTX 4070 Ti SUPER, so this is not a CPU fallback.

**Fixed:** `experiment.Manager` now receives the server-computed estimate and
enforces a runtime budget of estimate + 25% headroom (minimum two minutes;
30 minutes for unestimated jobs). Deadline expiry is a persisted **failed**
experiment with `experiment exceeded runtime budget …`, not a permanently
running worker. The evaluation harness and native passthrough forwarding now
honor context cancellation, so the deadline actually releases the worker.
Kyutai yields cooperatively during multi-frame decode so WebSocket sender and
control tasks are not starved by one large PCM batch.

**Remaining blocker:** Deterministic Kyutai quality evidence is still not
earned. The 100ms path is too slow, while the larger-chunk path needs a clean
`done` plus a non-error transcript before it can be used for quality evidence.
Do not promote Kyutai or launch the corrected 60-minute profile until a
deterministic 15-minute run completes inside its budget with valid transcript
and safety results.

**Validation:** `go test ./internal/eval ./internal/stt/strategy
./internal/experiment ./handlers/experiment` passes; Kyutai resource suites
pass (13 Python tests and `make test`). Live timeout experiment
`3b2c3483-3a9c-483a-bf6c-0667b838e744` failed at exactly 2m30s with the
persisted runtime-budget diagnostic. The scheduling probe
`cfd5bafd-83ca-4bb5-a06a-0c337d3f0cfa` ran until the same enforced deadline
without the prior 30-second close-before-`done`.

### 2026-07-12 addendum — resolved: compile guard restores bounded 100 ms-frame throughput

The prior entry records the state before the rebuilt resource was deployed.
The managed container now runs with `NO_TORCH_COMPILE=1` and
`KYUTAI_STT_TORCH_COMPILE=0`, so moshi cannot lazily start PyTorch Inductor
compilation on a user turn. The direct managed-resource probe sent 300 canonical
100 ms PCM frames (30 seconds total), received complete processed coverage, and
finished in **3.85 seconds** under its enforced 90-second wall-clock ceiling.
Persisted experiment `9dfbb669-216b-4151-9754-7ca34cd368c7` independently
succeeded for that deterministic 30-second path. The full real-time 60-minute
provider experiment `1733d1a1-4997-4227-af2a-6719a976f699` also completed both
Kyutai and Whisper over 3,602,659 ms of materialized PCM.

This resolves the decoder-throughput blocker and does **not** promote Kyutai:
the trust rubric still blocks both engines until independently persisted fault,
recovery, browser-product-path, and device evidence is complete. The runtime
budget remains enforced for every experiment, including any future
qualification retry.

### 2026-07-12 — Whisper unary batch is not a valid long-form promotion cell

**Symptom:** persisted full-real-time experiment
`1733d1a1-4997-4227-af2a-6719a976f699` reported a 24-word dropped span for
the Whisper row. Before 2026-07-12, promotion aggregation retained only WER
and duration, so that observed safety failure was omitted from the provider
verdict.

**Investigation:** the report's Whisper row is
`whisper-local/batch/realtime`, not a segmented streaming strategy. Its one
`batchSession` call buffered the entire 3,602,659 ms synthetic clip before
calling the unary Whisper provider. The report confirms a real loss signal:
the repeated product sentence occurs 25 times in reference and 24 times in
the hypothesis, and the hypothesis also ends with an unwanted trailing phrase.
The repeated corpus material can make global word alignment choose a longer
equivalent deletion path, but it does not explain away the missing occurrence
or make the unary result valid no-loss evidence.

**Resolution shipped:** promotion evidence is now keyed by engine, strategy,
and policy profile. Reports with no strategy identity are retained as legacy
history but excluded from promotion aggregation. Observed safety state is
preserved in `ReplayMeasurement`, so the batch cell explicitly fails the
dropped-span gate instead of contaminating or being hidden by another Whisper
strategy. The UI contract renders the cell identity.

**Remaining gate:** qualify a real-time long-form `whisper-local/vad_segment`
or `whisper-local/overlap_agree` provider cell (and the corresponding Kyutai
cell) through the full duration/fault/browser/device matrix. Do not use a
whole-clip unary batch baseline as stable-engine promotion evidence.

**Refs:** `api/handlers/eval/evalrun.go::batchSession`,
`api/internal/stt/trustfloor/rubric.go`,
`api/handlers/experiment/promotion_evidence.go`.

### 2026-07-12 — Dedicated qualification artifacts were previously not durable promotion input

**Symptom:** Deterministic BAS and focused recovery tests could prove parts of the
dictation contract, but promotion verdicts consumed only replay WER, duration,
and safety fields. Browser, recovery, fault, interval-accounting, and device
claims therefore had no persisted, inspectable connection to a provider cell.

**Resolution shipped:** `ExperimentService` now owns a qualification-evidence
ledger. Each record stores engine, exact model id, strategy, policy profile,
required artifact reference, pass/fail result, timestamp, and machine identity.
Trust-floor aggregation consumes only records from the exact
engine/model/strategy/policy cell on the same host/OS/architecture as the
report; it never credits another model, strategy, or machine. Failed records
remain visible as explicit promotion reasons. The operator surface is
`audio-tools experiment record-evidence` and `audio-tools experiment
list-evidence`.

**Remaining gate:** Existing browser and recovery results have not been
backfilled, because their stored artifacts do not assert one exact provider
strategy/policy cell. Run each real qualification profile with that identity and
record its durable artifact reference. Physical-device evidence remains absent.

**Refs:** `api/internal/experiment/`; `api/handlers/experiment/promotion_evidence.go`;
`cli/domains/experiment/`; `packages/proto/schemas/audio-tools/v1/experiment/experiment.proto`.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
