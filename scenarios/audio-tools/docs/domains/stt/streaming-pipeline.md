# STT Streaming Pipeline

This document is the canonical architecture reference for streaming
speech-to-text inside audio-tools. It explains how audio chunks flow
from a transport (browser WebSocket or non-browser Connect bidi) to a
typed event stream, and how the **technique** that produces those
events is kept independent of the **provider** that performs the
transcription.

Read this first when:

- adding a new STT provider (Local, BYOK, Vrooli/LPBS) and deciding
  whether it streams natively or relies on a server-side strategy,
- adding a new streaming **strategy** (VAD-segmented batch,
  overlap-and-agree, native passthrough),
- adding a new transport surface,
- debugging "why did this session emit only one segment at StreamEnd?"

For the unary `Transcribe` path, see
[`../../reference/api-endpoints.md`](../../reference/api-endpoints.md)
— this document covers only the streaming path.

## Latency expectation

The current Linux Whisper medium CPU configuration is intentionally declared
as CPU-only because no pinned Linux CUDA server is available at the selected
release. The same-corpus measurement is approximately 5.3 seconds for a
two-second clip; this is above the 2.5-second interactive target and is an
honest expectation, not a hidden performance claim. A future accelerator or
smaller-model change must repeat the quality smoke and latency measurement
before changing the declaration.

## Two Axes, One Pipeline

Streaming STT has two independent axes that the current code partly
fuses together. Keeping them separate is the load-bearing decision.

| Axis | Question it answers | Examples |
|---|---|---|
| Strategy | *How* are audio chunks turned into events? | VAD-bounded segments → batch; overlap-and-agree sliding windows; passthrough to a native streaming engine |
| Provider | *Who* actually performs the transcription, and which tier are they in? | Local Whisper (Local tier); OpenAI Whisper API (BYOK tier); Deepgram (BYOK tier, native streaming); Vrooli/LPBS (Vrooli tier) |

Some providers constrain which strategies they support. The matrix is
populated explicitly (see [§ Strategy × Provider Compatibility](#strategy--provider-compatibility))
rather than discovered at runtime.

The wire shape — `TranscribeStreamEvent` in
`packages/proto/schemas/audio-tools/v1/stt/stt.proto` — is the same
regardless of which (strategy, provider) pair is in use. Consumers
(UI, CLIs, other scenarios) never learn about strategies; the
abstraction terminates inside audio-tools.

## Current Architecture (as of 2026-08-18)

> **One-shot auto-stop is server-VAD-led.** The browser's mic-button ring and
> the stop trigger consume the same source — `useServerVadStateStore` — via
> the pure `decideAutoStop` helper in
> `ui/src/audio-integration/hooks/voice/autoStopDecision.ts`. The client-side
> RMS VAD now only fires the stop when the server tick is stale (>250 ms) or
> absent. See [`SEAMS.md#auto-stop-decision`](../../internal/SEAMS.md#auto-stop-decision).
> Persistent-mode behaviour is unchanged.



Two transports use the same transport-free segmenting boundary. The browser
WebSocket and Connect bidi surfaces translate their wire formats into the
shared `sttchain` event stream; they do not select providers or implement
their own retention policy.

```
┌─────────────────────────┐         ┌──────────────────────────────┐
│  /api/v1/voice/stream   │         │  STT.TranscribeStream        │
│  (WebSocket — browser)  │         │  (Connect bidi — non-browser)│
│  handlers/stt/          │         │  handlers/stt/               │
│    stream_ws.go         │         │    transcribe_stream.go      │
└──────────┬──────────────┘         └──────────────┬───────────────┘
           │                                       │
           │ delegates to                          │ accumulates all
           ▼                                       │ chunks, then calls
┌──────────────────────────────┐                   ▼
│  Segmenter + sttchain         │      ┌──────────────────────────────┐
│  ─────────────────────────   │      │  Batch strategies             │
│  Shared pipeline owns:       │      │  VADSegment / OverlapAgree / │
│    • strategy selection      │      │  BufferedFallback             │
│    • provider identity       │      │  call native whisper.cpp      │
│    • bounded retention       │      └──────────────────────────────┘
│    • speaker/policy gates    │
│    • durable event ordering  │      ┌──────────────────────────────┐
└──────────────────────────────┘      │  Native streaming strategies │
                                      │  Kyutai or sherpa-streaming  │
                                      │  emit partials, segments,    │
                                      │  processed acks, and done.    │
                                      └──────────────────────────────┘
```

The shipped implementation provides:

- `sttchain.Provider` declares provider traits and streaming behavior.
- `sttchain.Chain.Stream(...)` negotiates a streaming-capable tier
  and supports an explicit buffered strategy for `streaming_mode=off`.
  Production auto/streaming entry points fail closed when no native stream
  is available; they do not silently accumulate an unlimited whole-turn
  buffer. See [`SEAMS.md`](../../internal/SEAMS.md#streaming-chain-seams-audio-tools-web-console-restoration-plan).
- `StrategySelector` chooses a compatible strategy/provider cell from the
  manifest and operator preference.
- `whisper-local` uses the native whisper.cpp managed service for batch
  strategies; `kyutai` and `sherpa-streaming` use native passthrough
  streaming providers.
- Both transports preserve durable segments, acknowledgements, errors, and
  terminal events while allowing disposable partial/status snapshots to be
  coalesced.

Persistent voice mode is a strict native-streaming contract. If the capability
probe cannot confirm the durable streaming path, the shared voice core waits
for the probe when necessary and then refuses to start persistent capture with
a visible unavailable message. It does not silently change the request into a
one-shot or buffered transcription. Buffered recovery remains an explicit,
bounded error-recovery path for an already-started stream; one-shot mode must
be selected explicitly when a caller wants batch transcription.

The remaining work is qualification, not missing architecture: clean
15/60-minute real-time device evidence, same-corpus comparison reports, and
signed target-native resource publication are still required before promotion.

## Target Architecture

The destination separates strategy from provider, gives the "which
strategy do I use?" decision exactly one home, and shares one pipeline
across both transports.

```
TRANSPORT EDGE  (translate wire format ↔ Segmenter events; no logic)
┌────────────────────────┐    ┌────────────────────────┐
│  WS handler            │    │  Connect bidi handler  │
│  /api/v1/voice/stream  │    │  STT.TranscribeStream  │
└──────────┬─────────────┘    └────────────┬───────────┘
           │                               │
           │  audio chunks in              │
           │  TranscribeStreamEvents out   │
           ▼                               ▼
┌─────────────────────────────────────────────────────────┐
│  Segmenter   (transport-free orchestrator)               │
│  ─────────────────────────────────────────────────────  │
│  in:  audio chunks (channel)                             │
│  out: TranscribeStreamEvent stream                       │
│         { Partial, Segment, WakeWord, Rejection,         │
│           Error, Done }                                  │
│  owns: session lifecycle, observer fanout (session pub/  │
│        sub), barge-in signaling into TTS, cancellation.  │
│                                                          │
│  delegates "what produces events?" to a single           │
│  collaborator:                                           │
└──────────────────────────┬──────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│  StrategySelector  ← single explicit decision boundary   │
│  ─────────────────────────────────────────────────────  │
│  Inputs:                                                  │
│    • the negotiated provider tier (BYOK/Vrooli/Local)     │
│    • operator config (strategy_preference, VAD/overlap    │
│      tuning, streaming_mode)                              │
│    • Provider.StreamingCapability() + ProviderTraits      │
│  Output:                                                  │
│    • a concrete StreamingStrategy bound to the provider.  │
│                                                          │
│  Strategy/provider compatibility matrix lives here and    │
│  nowhere else.                                            │
└────┬──────────────────────────┬──────────────────────┬──┘
     │ picks one                │                      │
     ▼                          ▼                      ▼
┌────────────────────┐  ┌────────────────────┐ ┌──────────────────────┐
│ VADSegmentStrategy │  │ OverlapAgreeStrategy│ │ PassthroughStrategy  │
│ ─                  │  │ ─                   │ │ ─                    │
│ Silence-bounded    │  │ Sliding window +    │ │ Forward chunks to a  │
│ chunks → batch     │  │ LocalAgreement →    │ │ native-streaming     │
│ transcribe each    │  │ live Partials,      │ │ provider; translate  │
│ Emits: Segment +   │  │ stable commits as   │ │ vendor events.       │
│   wake_word        │  │ Segments.           │ │ Emits: Partial +     │
│ Needs: Batch       │  │ Emits: Partial +    │ │ Segment              │
│                    │  │ Segment             │ │ Needs: Stream        │
│                    │  │ Needs: Batch        │ │                      │
└────────┬───────────┘  └─────────┬───────────┘ └──────────┬───────────┘
         │ uses                   │ uses                    │ uses
         ▼                        ▼                         ▼
┌─────────────────────────────────────────────────────────────┐
│  Provider chain  (tier ordering: Local → BYOK → Vrooli)      │
│  ─────────────────────────────────────────────────────────  │
│  Each provider declares Capabilities{Batch, Stream} via      │
│  ProviderTraits and continues to implement Transcribe        │
│  (batch) and TranscribeStreaming (stream) per the existing   │
│  sttchain.Provider interface. Strategies that need Batch     │
│  only see providers with Batch=true; strategies that need    │
│  Stream only see providers with Stream=true.                 │
└─────────────────────────────────────────────────────────────┘

Same pipeline serves both transports.
Strategy and provider are independent axes.
```

The Segmenter, StrategySelector, and StreamingStrategy interfaces are
registered in [`SEAMS.md`](../../internal/SEAMS.md#streaming-chain-seams-audio-tools-web-console-restoration-plan).

### How the matrix narrows in practice

Each tier exposes one or more named providers. Each provider declares
which strategies it can serve. The selector enumerates compatible
pairs in tier precedence order and picks the first whose strategy
matches the operator's preference (or the configured `auto` default).

| Provider | Tier | Batch | Native Stream | Suitable Strategies |
|---|---|---|---|---|
| Whisper local managed service (whisper.cpp compatibility edge) | Local | yes | no | VADSegment, OverlapAgree, BufferedFallback |
| OpenAIWhisperAPI | BYOK | yes | no | VADSegment (overlap-and-agree forbidden — see below) |
| Deepgram | BYOK | yes | yes | Passthrough |
| AzureSpeechStreaming | BYOK | (not used) | yes | Passthrough |
| GoogleSpeechStreaming | BYOK | (not used) | yes | Passthrough |
| VrooliLPBS | Vrooli | yes | yes (planned) | Passthrough (when LPBS enables it); VADSegment until then |

### Strategy × Provider Compatibility

The selector enforces this table; pairs marked **forbidden** raise a
selector error rather than silently falling back.

| Strategy ↓ / Provider → | WhisperLocal | Kyutai | SherpaStreaming | OpenAIWhisperAPI | Deepgram | Azure/Google | VrooliLPBS |
|---|---|---|---|---|---|---|---|
| VADSegment | ✅ **default for native Whisper** (silence-bounded segments; one Segment per utterance; the most seamless batch strategy today) | ⛔ forbidden — native streaming | ⛔ forbidden — native streaming | ✅ only choice (API has no streaming) | ⛔ forbidden — native streaming, use Passthrough | ⛔ forbidden | ✅ until LPBS streaming lands |
| OverlapAgree | ✅ opt-in via explicit `preference=overlap` (growing-buffer LocalAgreement-N + VAD-anchored triggering; incremental Segment events mid-utterance; bounded prompt context; word-aligned cursor advance). No longer the auto default while its long-form quality is being qualified. | ⛔ forbidden — native streaming | ⛔ forbidden — native streaming | ⛔ forbidden — would burn money on each overlapping API call | ⛔ forbidden | ⛔ forbidden | ⛔ forbidden |
| Passthrough | ⛔ forbidden — provider is batch | ✅ native streaming | ✅ native streaming | ⛔ forbidden — provider is batch | ✅ only choice | ✅ only choice | ✅ when LPBS streaming flag flipped |

The "forbidden" cells are not theoretical; the selector returns a
typed error so misconfiguration shows up at startup or first call,
not as silent quality regressions.

## Decision Boundary: StrategySelector

The selector is the single named home for the question *"given this
session, how do we produce events?"*. It has exactly one entry point:

```go
type StrategySelector interface {
    // Select picks a (strategy, provider) pair for a streaming session.
    // Returns a typed error if the requested configuration is
    // incompatible (per the compatibility matrix above), the chain
    // has no eligible tier, or streaming_mode=off.
    Select(ctx context.Context, start StreamStart, cfg StreamConfig) (StreamingStrategy, Provider, error)
}
```

Inputs in `cfg` are the operator-tunable levers documented in
[`../../reference/configuration.md`](../../reference/configuration.md#streaming-stt-control-surface).
The selector NEVER reads provider configuration directly; it consumes
declared `ProviderTraits` (capability bits) from the chain. New
providers declare their traits in one place; the selector picks them
up automatically.

This collapses three currently-scattered decisions into one location:

| Today | Target |
|---|---|
| "VAD?" — implicit in `voice.Service.HandleStreamWS` | StrategySelector chose VADSegmentStrategy |
| "Batch then return?" — implicit in `transcribe_stream.go` accumulator | StrategySelector returned a strategy that consumes the chunk channel live |
| "Which tier?" — `Chain.Stream`'s loop | StrategySelector consumes the tier order, but pairs it with strategy choice |

## Strategy Interface

```go
// StreamingStrategy turns an audio-chunk channel into a TranscribeStreamEvent
// channel. It is transport-free and provider-agnostic at the type level;
// concrete strategies are constructed by the selector with a specific
// Provider already bound.
type StreamingStrategy interface {
    Run(ctx context.Context, in <-chan AudioChunk, out chan<- StreamEvent) error
}
```

Strategy implementations are pure orchestrators of audio framing,
windowing, and provider invocation. They do not own session state,
observer fanout, or transport details — those live in the Segmenter.

| Strategy | What it owns | What it calls into Provider for |
|---|---|---|
| VADSegmentStrategy | Energy VAD over the chunk stream; silence or bounded continuous-speech segment boundaries; wake-word/speaker-verify stage composition | `Provider.Transcribe(audio)` once per bounded segment |
| OverlapAgreeStrategy | Sliding-window scheduling; LocalAgreement prefix commit; partial emission | `Provider.Transcribe(audio)` per overlapping window (Whisper-local only) |
| PassthroughStrategy | Thin wire translation; backpressure between client and vendor | `Provider.TranscribeStreaming(start, chunks)` once per session; forwards events |

## Session lifecycle: idle-based, drain-then-close

### Protocol v2: replay-safe capture coverage

Browser and Connect clients use protocol version 2 for a streaming turn. A
turn has a random `session_id` and `resume_token`; each captured canonical PCM
chunk has a monotonically increasing `sequence`, absolute `start_sample` and
`end_sample` cursors, and a SHA-256 identity. Browser WebSocket frames encode
those values as `ATV2 | sequence | start_sample | end_sample | sha256 | PCM`.
The handler verifies the digest before the server session ledger accepts the
chunk. `done`, a socket close, and an empty final are **not** evidence that
audio was processed.

The server ledger persists received coverage and advances its processed cursor
from the strategy consumption cursor, not from text commits. This matters for
continuous speech: a recognizer can consume audio for many seconds while its
silence-based segmenter has not committed a segment. The cursor emits at most
one durable acknowledgement per roughly 100 ms of canonical audio, and the
WebSocket ledger compacts that coverage immediately into a bounded replay tail.
It returns a durable `processed_acknowledgement` status with both cursors before
terminal `final` and `done`.

The browser stores the same ordered batched chunks and digests in a bounded turn
journal before releasing a frame to the socket. Worklet quanta are combined
into approximately 100 ms ATV2 frames (at most 15 wire messages per second),
and IndexedDB uses append records with periodic snapshot compaction rather than
rewriting the entire retained turn for every frame. The server uses the same
append-log-plus-compaction pattern for receive, acknowledgement, commit, and
terminal mutations. Both sides surface reduced durability or quota failure
instead of discarding unacknowledged audio. Replay is at-least-once; the ledger
deduplicates the identical sequence/range/digest and rejects conflicts or gaps.

The browser's complete-turn retry copy is deliberately bounded and exists for
short-turn empty-result or speaker-policy retry UX; it is not the lifetime
storage for a long dictation. If that copy reaches its ceiling during a
healthy long session, streaming continues and a later backend failure recovers
from the journal's still-unacknowledged PCM tail. This preserves audio the
server has not durably consumed without making memory grow with session length.
If the bounded journal quota itself is exhausted while acknowledgements are
unavailable, capture fails visibly with a durability/quota reason rather than
silently dropping words. A committed terminal final also deletes the durable
journal records, preventing completed turns from accumulating in IndexedDB.

For supported same-tab reload recovery, the browser also retains only the
opaque session/resume identity in session storage. It restores that identity
with the IndexedDB journal, resumes the same server ledger, and derives the
next sample cursor from retained coverage. It deletes both identity and journal
only after terminal processed acknowledgement; ordinary telemetry never
contains the audio payload or transcript.

This is deliberately not an exactly-once *transport* promise: a reconnect may
deliver a retained chunk again, but it cannot silently replace it or advance
coverage without the server acknowledgement. Durable transcript segments have a
different, stronger rule: both transports commit a stable `segmentId` into the
same ledger before emitting it. On WebSocket reconnect, the server re-emits
those commits in canonical sample order and the browser delivers each identity
to host UI once. A future provider-recovery generation must replay retained
ranges under a new generation identity while preserving that same segment
idempotency rule.

A streaming session used to be bounded by a hard 5-minute *absolute*
WebSocket deadline (`context.WithTimeout(r.Context(), 5*time.Minute)`
in `handlers/stt/stream_ws.go`). That clock fired on *active* speakers
mid-utterance and closed the socket without draining the tail, so the
last words were silently lost. The lifecycle is now **inactivity-based
with a drain-then-close teardown**:

- **Idle deadline, reset per frame.** Each inbound audio frame arms a
  read deadline of `SessionIdleTimeoutMs` (see below). A session ends
  only after that many milliseconds with *no* audio — not on a fixed
  wall-clock cap. A user can dictate continuously for as long as they
  like.
- **Every close path drains, then closes.** Idle-timeout, an explicit
  client `done`, and request-context cancel all route through the same
  graceful-end path: the segmenter is signalled to end, the provider
  sends kyutai its end marker, and the reader awaits the flush/`done`
  **before** the socket closes. A bounded drain deadline prevents a
  wedged backend from hanging the close forever. Measured kyutai flush
  lag is small (RTF ≈ 0.13), so awaiting it is cheap.
- **Kyutai dials lazily.** The browser WebSocket may be opened before
  recording starts, but the kyutai provider does not dial `/v1/stream`
  until the first audio chunk arrives. Zero-audio pre-connects therefore
  never take or wait on the single-session model lock.
- **Backend death is durable.** If kyutai closes before a `done` frame,
  the provider emits a durable `error` event before terminal `done`.
  Kyutai admission reports durable `queued`, `ready`, `timed_out`, or
  `rejected` lifecycle status. Clients retain audio locally while queued;
  contention is never represented as a successful empty final.
- **Force-commit during continuous speech (kyutai).** Independently of
  teardown, the kyutai server force-commits a pending segment at the
  next word boundary once it has spanned `KYUTAI_STT_MAX_SEGMENT_FRAMES`
  frames, so a long unbroken utterance produces durable segments as the
  user speaks instead of stalling as one volatile partial until the end
  flush. This is a sibling knob to `KYUTAI_STT_SILENCE_COMMIT_FRAMES`
  (the pause-triggered commit) — see
  `resources/kyutai-stt` resource server in the workspace. The same bounded
  contract is implemented by the native `resources/sherpa-onnx` adapter.
- **Non-graceful closes are the drop metric.** A `nil` reader error is
  a graceful end (idle-timeout or client `done`) whose final flush was
  delivered; a non-graceful close is counted as a potential tail drop
  in the per-session delivery telemetry, so any teardown regression
  surfaces immediately rather than silently swallowing words.

**Knobs.** `SessionIdleTimeoutMs` lives on `stt_stream_config`
(`DefaultSessionIdleTimeoutMs = 30000`, i.e. 30 s; `<= 0` resolves to
the default at the handler). The kyutai commit cadence is set by
`KYUTAI_STT_MAX_SEGMENT_FRAMES` (default `48` ≈ 3.8 s at 12.5 Hz; `0`
disables force-commit → legacy pause-or-flush-only) and
`KYUTAI_STT_SILENCE_COMMIT_FRAMES` (default `16`). The whisper
`VADSegment` strategy keeps parity with the same drain-then-close seam. The
`BufferedFallback` strategy has a declared `10 MiB` whole-turn ceiling,
matching the unary transcription limit; it emits a typed refusal before
acknowledging audio beyond that bound, leaving the session ledger as the
replay owner rather than silently dropping the turn.

## Event-durability contract

**This is the single, authoritative statement of the streaming-audio delivery
rule. Every hop reads it; no other document restates it — they cross-reference
here.** It governs both directions of real-time audio (STT segment/partial
delivery *and* TTS paragraph playback) so the whole pipeline is backpressure-safe
without three ad-hoc buffers.

A streaming session emits two classes of event:

- **Disposable snapshots — `partial`, `vad-state`, and ordinary `status`.**
  Interim hypotheses and live progress snapshots may be **coalesced to the
  latest value or dropped** under consumer backpressure, and they **MUST NEVER
  back-pressure their producer**. Each stream has an independent latest-value
  slot at the browser WebSocket egress: a VAD tick cannot erase live text, and
  a status update cannot erase the ring state. Losing an intermediate snapshot
  is invisible because the next value supersedes it.
- **Durable — everything else** (`segment`, `segment-rejected`/speaker
  rejection, `acknowledgement`, `error`, `done`, and the ancillary `wake_word`).
  These are **ordered and lossless**: they are delivered in emission order and
  are never dropped, even under sustained backpressure. Because durables are
  low-rate, buffering them losslessly is cheap; a producer must never drop a
  durable to relieve pressure — it drops/coalesces partials instead.

**Why this closes the wedge.** The failure this pipeline previously suffered was
a fully-synchronous, tiny-buffer chain across two WebSocket hops: a slow browser
consumer back-pressured every hop until the kyutai decode loop stopped consuming
audio — total loss of everything spoken thereafter, not just a tail. The contract
removes the coupling at its root: partials cannot block a producer, and the relay
always drains the backend socket, so the decode loop is immune to consumer speed
while committed text remains lossless and ordered.

**One rule, one code encoding.** All three Go hops (the kyutai provider adapter,
the relay egress buffer, and the browser WS handler) consume the single predicate
`sttchain.StreamEvent.Durable()` / `IsDroppable()`
(`internal/ai/sttchain/interface.go`) rather than re-deriving the rule inline.
The kyutai resource (`resources/kyutai-stt`) applies the same semantics on its
send worker, and the native sherpa adapter follows the same contract. The
web-console client renders `partial`
disposably (coalesced) while treating `segment`/`final` durably.

**TTS twin.** A synthesized spoken reply is N ordered paragraphs; each paragraph
is a **durable ordered unit**. A single-paragraph fault is isolated (surfaced on
that unit) and MUST NOT truncate the paragraphs after it — the same durable-
ordered-delivery rule as an STT segment, applied to playback.

## Product-surface qualification

The browser qualification driver can exercise either Audio Tools Dictation
Studio (the default) or the Swarm Manager Plan surface with
`--surface swarm-manager`. The latter opens Quick Capture through the graph
action launcher and drives the shared `MessageComposer` microphone, interim
composer text, and terminal microphone state. Swarm Manager owns the composer;
Audio Tools owns the recorder, but both surfaces consume the same browser audio
capture and voice-core contract.

The Swarm Manager selectors are contract-owned in
`scenarios/swarm-manager/ui/src/consts/selectors.ts` and are exposed as
`captures-quick-input-mic`, `captures-quick-composer`, and the launcher action
selector. Qualification evidence records the selected surface so a passing
Audio Tools run is never presented as Swarm Manager evidence.

## Why decouple? — and what it costs

**Benefits.**

- One VAD implementation, used by every batch-only provider, instead
  of N copies inside each provider adapter.
- Operators get a real lever (`strategy_preference`) — quality vs.
  CPU vs. latency tradeoffs map to a config value, not a code fork.
- The compatibility matrix is enforced once, in the selector, not
  scattered as conditional branches across providers.
- Adding a provider does not invent a new pipeline; it adds an entry
  to the capability matrix and (if native-streaming) implements
  `TranscribeStreaming`.

**Costs.**

- Slightly more interface plumbing than a single hardcoded
  `voice.Service`. The payoff is that the Vrooli rule "don't add
  abstractions beyond what the task requires" is satisfied by the
  PRD: §P0-005 already commits to 5 BYOK starter adapters spanning
  all three techniques, so the matrix is real on day one.
- Strategies have to declare their CPU/latency profile so the selector
  can implement `auto` mode coherently. This is one extra metadata
  table, not a real cost.
- Native-streaming strategies bypass most of the strategy abstraction
  — `PassthroughStrategy` is mostly a translation shim. That is fine;
  it still gives every native-streaming provider one consistent home.

## Cross-References

- [`../../concepts/ARCHITECTURE.md`](../../concepts/ARCHITECTURE.md) — system-level shape
- [`../../internal/SEAMS.md`](../../internal/SEAMS.md#streaming-chain-seams-audio-tools-web-console-restoration-plan) — seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — durable decision record
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — current drift the target architecture closes
- [`../../reference/configuration.md`](../../reference/configuration.md#streaming-stt-control-surface) — operator-tunable levers
- `packages/proto/schemas/audio-tools/v1/stt/stt.proto` — wire shape (`TranscribeStreamEvent` oneof)
# Provider tiers and BYOK fallback

Audio-tools prefers an available local STT provider. When local providers are
unavailable, the chain selects a configured BYOK provider (including the
encrypted-store default when a caller supplies no explicit credential);
browser speech is the final client-side fallback. BYOK credentials are managed
through `audio-tools settings providers` and are never returned by health or
capability endpoints. A missing credential is reported as an actionable,
optional provider absence rather than as a failure of unrelated capabilities.
