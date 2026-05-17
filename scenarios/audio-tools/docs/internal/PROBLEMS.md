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
| _None yet._ |  |  |  |

## Cross-references

- [`PROGRESS.md`](PROGRESS.md) — lifecycle log (forward-looking)
- [`SEAMS.md`](SEAMS.md) — boundary registry (load-bearing for tests)
- [`TESTING.md`](TESTING.md) — test patterns
- [`../guides/troubleshooting.md`](../guides/troubleshooting.md) — generic-template issues
