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

## Current Architecture (as of 2026-05-17)

> **One-shot auto-stop is server-VAD-led.** The browser's mic-button ring and
> the stop trigger consume the same source — `useServerVadStateStore` — via
> the pure `decideAutoStop` helper in
> `ui/src/audio-integration/hooks/voice/autoStopDecision.ts`. The client-side
> RMS VAD now only fires the stop when the server tick is stale (>250 ms) or
> absent. See [`SEAMS.md#auto-stop-decision`](../../internal/SEAMS.md#auto-stop-decision).
> Persistent-mode behaviour is unchanged.



Two transports exist; they run parallel, partially-overlapping
pipelines, and the streaming chain skeleton already exists in
`internal/ai/sttchain/` but is not yet wired by any transport.

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
│  internal/voice/Service      │      ┌──────────────────────────────┐
│  ─────────────────────────   │      │  sttchain.Chain.Execute      │
│  Hard-coded pipeline:        │      │  (BATCH — single audio blob) │
│    • VAD                     │      │                              │
│    • wake-word               │      │  Emits 1 Segment + 1 Done    │
│    • speaker verify          │      │  at StreamEnd. No live       │
│    • LOCAL WHISPER           │      │  partials, no wake-word      │
│    • event emission          │      │  events, no barge-in signal. │
│                              │      └──────────────────────────────┘
│  Local-tier-specific. No     │
│  tier negotiation. No        │      ┌──────────────────────────────┐
│  strategy abstraction.       │      │  sttchain.Chain.Stream       │
└──────────────────────────────┘      │  (capability-negotiated      │
                                      │   streaming entry point)     │
                                      │                              │
                                      │  EXISTS in code, but every   │
                                      │  Provider returns            │
                                      │  StreamingCapability=false   │
                                      │  today → buffered fallback.  │
                                      │  No transport calls it yet.  │
                                      └──────────────────────────────┘
```

What's already in place (groundwork from the prior
`audio-tools-web-console-restoration` plan):

- `sttchain.Provider` declares `StreamingCapability()` and
  `TranscribeStreaming(start, chunks) -> events` per tier.
- `sttchain.Chain.Stream(...)` negotiates a streaming-capable tier
  (BYOK → Vrooli → Local) and falls back to a buffered single-shot
  when none accept. See [`SEAMS.md`](../../internal/SEAMS.md#streaming-chain-seams-audio-tools-web-console-restoration-plan).

What is missing:

1. No `Provider` returns `StreamingCapability()=true` today, so every
   call lands in the buffered fallback.
2. The **strategy** axis does not exist as a first-class abstraction.
   `internal/voice/Service` hard-codes VAD + wake-word + speaker-verify
   + local-Whisper-batch as one fused pipeline. Adding a second
   strategy (overlap-and-agree) would require forking that file.
3. Neither transport calls `Chain.Stream`. The WS handler still calls
   `voice.Service.HandleStreamWS` directly; the Connect bidi handler
   buffers and calls `Chain.Execute`.
4. The two transports share neither code nor an event-sequence parity
   test, so they are guaranteed to drift.

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
│  Provider chain  (tier ordering: BYOK → Vrooli → Local)      │
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
| LocalWhisper (whisper.cpp / faster-whisper) | Local | yes | no | VADSegment, OverlapAgree |
| OpenAIWhisperAPI | BYOK | yes | no | VADSegment (overlap-and-agree forbidden — see below) |
| Deepgram | BYOK | yes | yes | Passthrough |
| AzureSpeechStreaming | BYOK | (not used) | yes | Passthrough |
| GoogleSpeechStreaming | BYOK | (not used) | yes | Passthrough |
| VrooliLPBS | Vrooli | yes | yes (planned) | Passthrough (when LPBS enables it); VADSegment until then |

### Strategy × Provider Compatibility

The selector enforces this table; pairs marked **forbidden** raise a
selector error rather than silently falling back.

| Strategy ↓ / Provider → | LocalWhisper | OpenAIWhisperAPI | Deepgram | Azure/Google | VrooliLPBS |
|---|---|---|---|---|---|
| VADSegment | ✅ explicit `preference=vad` (silence-bounded segments; one Segment per utterance) | ✅ only choice (API has no streaming) | ⛔ forbidden — Deepgram has native streaming, use Passthrough | ⛔ forbidden | ✅ until LPBS streaming lands |
| OverlapAgree | ✅ **default for Local Whisper** (growing-buffer LocalAgreement-N + VAD-anchored triggering; incremental Segment events mid-utterance; word-aligned cursor advance). See PROBLEMS.md "OverlapAgree commit gap" (RESOLVED 2026-05-28) for the rewrite history. | ⛔ forbidden — would burn money on each overlapping API call | ⛔ forbidden | ⛔ forbidden | ⛔ forbidden |
| Passthrough | ⛔ forbidden — provider can't stream | ⛔ forbidden — provider can't stream | ✅ only choice | ✅ only choice | ✅ when LPBS streaming flag flipped |

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
| VADSegmentStrategy | Silero-style VAD over the chunk stream; segment boundary detection; wake-word/speaker-verify stage composition | `Provider.Transcribe(audio)` once per VAD-bounded segment |
| OverlapAgreeStrategy | Sliding-window scheduling; LocalAgreement prefix commit; partial emission | `Provider.Transcribe(audio)` per overlapping window (Whisper-local only) |
| PassthroughStrategy | Thin wire translation; backpressure between client and vendor | `Provider.TranscribeStreaming(start, chunks)` once per session; forwards events |

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
