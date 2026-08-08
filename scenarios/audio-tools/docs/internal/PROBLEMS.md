+# Problems — Audio Tools

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

**Real fix:** Rewire `handlers/stt/stream_ws.go` to upgrade the WS, marshal incoming binary frames into `sttchain.AudioChunk`, and forward `<-chan sttchain.StreamEvent` back over the wire shape expected by the shared browser capture client.

**Owner:** unassigned.

**Refs:** `internal/voice/stream_ws.go`, `handlers/stt/stream_ws.go`, plan Phase F.

### 2026-05-27 — ScriptProcessorNode → AudioWorklet migration (deferred)

**Symptom:** Two PCM capture sites use the deprecated `ScriptProcessorNode`: the wake-word `createPassiveCapturePipeline` (`hooks/voice/audioUtils.ts`) and the new streaming `createScriptProcessorPcmCapture` (`hooks/voice/pcmCapture.ts`). ScriptProcessor runs on the main thread and is deprecated in favor of `AudioWorkletNode`.

**Root cause:** ScriptProcessor is universally supported and kept deliberately for broad browser coverage; AudioWorklet requires loading a separate module script and more setup. Both sites carry `eslint-disable @typescript-eslint/no-deprecated` with this rationale.

**Real fix:** Introduce a single AudioWorklet processor that emits Float32 frames and back both capture sites with it. The `pcmCapture.PcmCaptureFactory` seam already isolates this — swap the production factory only; `VoiceStreamProvider` and its tests are unaffected.

**Owner:** unassigned.

**Refs:** `hooks/voice/pcmCapture.ts`, `hooks/voice/audioUtils.ts`.

### 2026-05-27 — Live ML-resource validation pending operator hardware (deferred)

**Symptom:** The two new ML resources (`resources/kyutai-stt/`, `resources/speaker-verification/`) are built, self-consistent, and contract-matched to their audio-tools clients, but have NOT been started end-to-end: that needs a GPU (Kyutai) + first-run multi-GB model downloads (Kyutai weights; SpeechBrain ECAPA, CPU-only).

**Status:** Deferred to the operator. Everything audio-tools-side is validated with fakes (Go build/test/lint, CLI, UI tsc/eslint/vitest, scenario restart). To validate live: build + start each resource (`vrooli resource start kyutai-stt` / `vrooli resource start speaker-verification`), confirm `/health` + `/ready` flip ready after the model download, then drive a real mic session (Kyutai partials/segments; speaker `filter` mode rejecting a second voice / background music). Kyutai enforces a single concurrent streaming session per model instance (asyncio lock) — revisit with a model pool if concurrency is needed.

**Owner:** unassigned (operator hardware).

### 2026-05-27 — Target-speaker extraction (IMPLEMENTED; empirical tuning pending)

**Symptom (original):** `SpeakerConfig.ExtractionEnabled` + `pipeline.ExtractTargetSpeaker` existed, but the `speaker-verification` resource returned HTTP 501 for `/v1/extract`; nothing isolated the enrolled speaker's audio.

**Resolution (2026-05-27):** Implemented as a pre-recognition **ingress** stage, not an egress method (egress can only drop text; isolation must substitute audio). `resources/speaker-verification` `/v1/extract` now runs SepFormer source separation + ECAPA target-selection and returns the isolated 16 kHz mono s16le PCM. The Go side adds `ingress.TargetExtractor`/`ingress.ExtractionEnhancer` (`internal/stt/ingress/extraction.go`), the handler-layer adapter `speakerExtraction` + `currentSpeakerExtraction` (built only when `extraction_enabled` + a bound profile), wired in `Segmenter.buildIngress` (config-gated like denoise — no manifest flag) and reachable via `audio-tools stt speaker-config --extraction-enabled`. The orphaned `pipeline.ExtractTargetSpeaker` (egress-shaped, zero callers) was deleted. Tested with fakes: `internal/stt/ingress/extraction_test.go`, `internal/stt/segmenter/ingress_build_test.go`, `handlers/stt/speaker_extraction_test.go`; resource non-model paths in `test/integration-test.sh`.

**Pending (environment-gated):** the separation-model spike — which SepFormer checkpoint + match threshold work best on real two-speaker audio, and CPU-vs-GPU latency — needs a GPU + model download + a live two-speaker A/B. Default OFF; degrades to passthrough if the resource/model is unavailable. See `resources/speaker-verification/docs/extraction.md`.

**Owner:** unassigned.

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


### 2026-05-17 — No cross-domain integration test

**Symptom:** Transcribe → summarize → synthesize is exercised only by per-handler unit tests; a regression in the cross-domain seam (chain wiring, normalizer ownership) would not fail a single test.

**Root cause:** Deferred from the 2026-05-17 post-extraction cleanup. The work needs deterministic stubs for all three external provider tiers in `internal/testutil/` and a new `api/tests/integration/` package.

**Workaround:** `vrooli scenario test audio-tools` exercises the per-handler happy paths against real chains with local providers when present; the cross-domain happy path is the diagnostics UI tab path.

**Real fix:** Add `api/tests/integration/cross_domain_test.go` per plan §G3 once the WS deferral above is also being worked.

**Owner:** unassigned.

**Refs:** plan §G3; `api/internal/testutil/`.


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

+### 2026-07-08 — Continuous-speech client-side auto-stop follow-up

**Symptom:**
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

### 2026-08-03 — Resolved: browser package lifecycle uses package-local test tool

**Symptom:** `vrooli package test audio-capture-browser` reached the package test
lifecycle but could not execute because `packages/audio-capture-browser` had no
local `vitest` binary, despite declaring Vitest as a development dependency.

**Root cause:** The package had no standalone pnpm boundary or package-local
dependency installation. The available Vitest binary from another workspace
package could run the tests, but that did not prove the package's normal
lifecycle environment.

**Workaround:** None remains. The borrowed executable was removed from the
validation path.

**Resolution:** Added the package-local pnpm workspace and lockfile, installed
Vitest, React, React DOM, jsdom, Testing Library, TypeScript, and Node types
through SDA, removed the absolute-path Vitest aliases, and reran the normal
control-plane lifecycle. The result is 7 test files and 94 tests passing, including
all 15 long-session recovery tests; the package-local
`node_modules/.bin/vitest` is the executable used by the run.

**Owner:** package governance / workspace maintenance.

**Refs:** `packages/audio-capture-browser/package.json`,
`internal/cli/packagecli/`, `internal/packagegov/`.

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues

## Work ladder

- Rung: W0 (goal/problem contract comparison)
- Evidence: Search found the requested reliability plan and related records, but no swarm-manager goal directly represents this audio-tools extraction work. The user-supplied plan is therefore the active contract; no unrelated goal was substituted.
- Constraint: Physical microphone confirmation remains an operator-only gate and is not claimed by this work record.
- Measured: 2026-08-03.
