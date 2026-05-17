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

**Owner:** unassigned.

**Refs:** `handlers/stt/stream_ws.go`, `internal/stt/strategy/webm.go`, `internal/stt/strategy/vad_segment.go`, `scenarios/audio-tools/embed/`.

### 2026-05-16 — WS endpoint tag still `ops_probe`

**Symptom:** `GET /api/v1/voice/stream` is tagged `RESTReason: ops_probe`; semantically it is a `TransportReason: websocket_transport`. Cosmetic only.

**Root cause:** The `websocket_transport` template constant in `internal/module` does not exist yet (tracked in R-PROTO upstream).

**Workaround:** Live with the mistag; documented in `api-endpoints.md` "Transport exceptions" with an explicit note.

**Real fix:** Retag when the template constant lands.

**Owner:** unassigned.

**Refs:** `docs/reference/api-endpoints.md` "Transport exceptions" table.

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
| ~~Coverage floors~~ | **Resolved** 2026-05-17. `scripts/check_coverage.sh` enforces per-package floors and is green: `internal/byok/envelope` 33.3 % → 100 %; `internal/protomap` 0 % → 91.7 %; every other package listed in plan §7 Phase 6 above its committed floor. The script is invoked from `make test` and fails CI when any package regresses below its floor. | Closed. | The script is the L4 evidence command; floors are committed in-band with the script and the plan. |

### 2026-05-17 — Browser-WS `/api/v1/voice/stream` lacks a transport test

**Symptom:** No automated test exercises the WS handshake, message echo, abrupt close, or context-cancel propagation paths in `api/handlers/stt/stream_ws.go`. Regressions surface only via the UI smoke run.

**Root cause:** Deferred from the 2026-05-17 post-extraction cleanup. A meaningful WS test requires a `httptest`/`gorilla/websocket` rig wired to the same session pub/sub fan-out the production transport uses; the fan-out seam exists (`internal/session/session.go`) but the handler-level fake is not yet hoisted into `handlers/stt/mocks/`.

**Workaround:** Manual UI smoke after restart confirms handshake + echo. Connect-RPC bidi `TranscribeStream` is covered end-to-end and shares the segmenter + strategy path, so most regressions surface there.

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
streaming-decoupling plan in [`PRD/`](../PRD/).

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

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
