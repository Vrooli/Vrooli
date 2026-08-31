# TTS Synthesis Pipeline

This document is the canonical architecture reference for text-to-speech
inside audio-tools. It explains how a `Synthesize` request is routed
through the three provider tiers (Local Kokoro → BYOK → Vrooli/LPBS)
and how the chain keeps a single wire shape across unary and streaming
calls.

Read this first when:

- adding a new BYOK TTS adapter (OpenAI TTS, ElevenLabs, future
  vendors),
- enabling native streaming on a tier that currently buffered-falls-back,
- debugging "why did the chain fall through from local to BYOK?" or
  "why did the chain fall through to Vrooli/LPBS?",
- changing voice-override resolution or the canonical voice catalog.

For the streaming STT counterpart, see
[`../stt/streaming-pipeline.md`](../stt/streaming-pipeline.md). The
chain skeleton here intentionally mirrors `summarizechain` so the two
domains evolve together; see
[`../summarize/chain.md`](../summarize/chain.md).

## Purpose

`ttschain.Chain` (`api/internal/ai/ttschain/chain.go:10`) is the single
home for the question *"which provider synthesizes this request?"*.
It owns:

- tier precedence (production: Local → BYOK → Vrooli/LPBS),
- availability caching with per-tier TTLs (BYOK 5 min, Vrooli 30 s),
- short-circuit error semantics (`ErrInsufficientCredits`,
  `ErrUnknownBYOKProvider`, `ErrMissingBYOKProvider` do not fall
  through),
- streaming negotiation via `Provider.StreamingCapability()`,
- a buffered fallback that wraps the unary path in a single
  `is_final=true` frame so consumers code against `Stream()` uniformly.

The chain is provider-agnostic at the type level. Each tier implements
the `Provider` interface (`api/internal/ai/ttschain/interface.go:52`)
and registers itself once at boot.

## Inputs

The `Synthesize` and `SynthesizeStream` Connect-RPC methods
(`api/handlers/tts/connect_handler.go:50`,
`api/handlers/tts/connect_handler.go:97`) translate
`ttsv1.SynthesizeRequest` into `ttschain.Request`
(`api/internal/ai/ttschain/interface.go:21`):

| Field | Source | Notes |
|---|---|---|
| `Text` | request body | Required. Caller should pre-normalize via `NormalizeForSpeech` for best quality. |
| `Voice` | request body | Canonical id (`voice.feminine.warm`, etc.). Adapter-specific names are resolved per tier. |
| `VoiceOverrides` | request body | Map keyed `tier:provider-id`; lets an operator pin a canonical voice to a vendor voice id. See [`../settings/byok-and-voice-overrides.md`](../settings/byok-and-voice-overrides.md). |
| `Speed` | request body | Multiplier; the native sherpa-onnx/Kokoro adapter clamps to `[0.5, 4.0]`. |
| `ResponseFormat` | request body | `mp3` \| `wav` \| `opus` \| `flac`. |
| `BYOKProvider`, `BYOKKey` | `X-BYOK-Provider` / decrypted from BYOK store via `envelope.FromConnectRequest` | Absence skips the BYOK tier. |
| `LPBSToken`, `UserIdentity` | Authorization / `X-User-Identity` via envelope | Absence skips the Vrooli tier. |
| `EventID`, `Version` | request body | Optional cache-key inputs (event-index round-trip in `internal/tts.Service`). |

The chain itself is transport-free; the Connect handler is a thin
translation shim.

## Outputs

Unary `Synthesize` returns `ttsv1.SynthesizeResponse` with audio bytes,
content-type, content-hash, and trace fields (`ProviderTier`,
`ProviderId`, `ModelId`, `VoiceUsed`, `LatencyMs`). The handler maps
`ttschain.Result` (`api/internal/ai/ttschain/interface.go:41`)
directly.

Streaming `SynthesizeStream` returns a server stream of
`ttsv1.AudioFrame`. Intermediate frames carry only `audio` +
`content_type` + `is_final=false`; the final frame populates the same
trace fields the unary response carries. `AudioFrame`
(`api/internal/ai/ttschain/interface.go:74`) is the chain-side
mirror; the handler at
`api/handlers/tts/connect_handler.go:72` copies into the wire type.

When no tier declares streaming, the buffered fallback
(`api/internal/ai/ttschain/chain.go:164`) calls `Execute` once and
emits a single `is_final=true` frame carrying the full audio. Consumer
code does not need to branch on streaming capability.

## Internal Chain

```
SynthesizeRequest (Connect)
        │
        ▼
envelope.FromConnectRequest          ← BYOK header parse, LPBS token, user
        │
        ▼
ttschain.Chain.Execute / Stream      ← tier ordering + availability gates
        │
        ├──► BYOKProvider.Synthesize          (api/internal/ai/ttschain/provider_byok.go:30)
        │       │
        │       ▼
        │   adapter[req.BYOKProvider]         (api/internal/byok/registry.go:32)
        │       e.g. NewOpenAITTS(), NewElevenLabsTTS()
        │
        ├──► VrooliProvider.Synthesize        (api/internal/ai/ttschain/provider_vrooli.go:29)
        │       │
        │       ▼
        │   LPBS chat with Operation: audio.synthesize
        │
        └──► LocalProvider.Synthesize         (api/internal/ai/ttschain/provider_local.go:32)
                │
                ▼
            tts.Service.Synthesize            (api/internal/tts/service.go:156)
                │
                ▼
            sherpa-onnx HTTP endpoint          (api/internal/tts/kokoro_synthesize.go)
```

Production execution order (`api/internal/ai/chains/tiered/tiered.go`):

1. **Local tier** is tried first iff `enableLocal && local.IsAvailable(ctx)`.
2. **BYOK tier** is tried iff `enableBYOK && req.BYOKKey != "" && availFor(BYOK)`.
   - `ErrUnknownBYOKProvider` / `ErrMissingBYOKProvider` short-circuit
     (configuration error — do not silently swap tiers).
   - Other errors record `lastErr` and fall through.
3. **Vrooli tier** is tried iff `enableVrooli && req.LPBSToken != "" && availFor(Vrooli)`.
   - `ErrInsufficientCredits` short-circuits (do not silently swap to
     Local — the user is owed the explicit price-gate error).
   - Other errors record `lastErr` and fall through.
4. If every eligible tier failed, returns `lastErr`; if no tier was
   eligible, returns `ErrAllProvidersFailed`.

The streaming path (`api/internal/ai/ttschain/chain.go:134`) filters
the same tier order by `StreamingCapability()` and falls back to the
buffered single-frame wrapper when nothing accepts. Today:

| Tier | `StreamingCapability()` | File |
|---|---|---|
| BYOK | Aggregated across registered adapters | `api/internal/ai/ttschain/provider_byok.go:55` |
| Vrooli | `false` (LPBS audio-gateway streaming is out of scope, PRD OT-P2-002) | `api/internal/ai/ttschain/provider_vrooli.go:60` |
| Local (sherpa-onnx/Kokoro) | `false` (incremental synthesis lands in Phase D) | `api/internal/ai/ttschain/provider_local.go:64` |

So every streaming call lands in the buffered fallback today unless a
BYOK adapter (e.g., a future ElevenLabs streaming adapter) flips its
capability bit to true.

### Voice resolution

Canonical voice ids (`voice.feminine.warm`, etc.) are resolved per
tier:

- Local sherpa-onnx/Kokoro uses `resolveLocalVoice`
  (`api/internal/ai/ttschain/provider_local.go:72`), consulting the
  voice-overrides map (`local:kokoro-local` key) then a hard-coded
  fallback table.
- BYOK adapters resolve via their own catalogues; overrides arrive
  through `req.VoiceOverrides` keyed `byok:openai-tts`,
  `byok:elevenlabs`, etc.
- Vrooli (LPBS) passes the canonical id through unchanged; LPBS owns
  the mapping on its side.

Overrides are persisted in `voice_overrides`
(`api/internal/store/voice_overrides.go:10`) and edited via
`SettingsService.SetVoiceOverride`
(`api/handlers/settings/voice_overrides.go:26`).

## Seams

The chain is shaped as one entry point with three pluggable providers
plus their adapters. See
[`../../internal/SEAMS.md`](../../internal/SEAMS.md) for the full
registry; the TTS-relevant entries are:

| Seam | Interface | Production | Test fake |
|---|---|---|---|
| TTS provider | `ttschain.Provider` (`api/internal/ai/ttschain/interface.go:52`) | `LocalProvider`, `BYOKProvider`, `VrooliProvider` | Stubs returning canned `Result` / `AudioFrame` per chain test (`api/internal/ai/ttschain/chain_test.go`) |
| BYOK adapter | `ttschain.BYOKAdapter` (`api/internal/ai/ttschain/provider_byok.go:8`) | `NewOpenAITTS()`, `NewElevenLabsTTS()` (`api/internal/byok/registry.go:32`) | Per-test fakes |
| Vrooli client | `ttschain.VrooliClient` (`api/internal/ai/ttschain/provider_vrooli.go:8`) | LPBS HTTP client | Per-test fake |
| sherpa-onnx/Kokoro backend | `tts.Deps.SynthesizeAudio` (`api/internal/tts/service.go:48`) | `internal/tts/kokoro_synthesize.go` | Function literal in service tests |
| Cache | `tts.Deps.GetCache` / `PutCache` (`api/internal/tts/service.go:49`) | `internal/tts.Cache` | In-process map in tests |

The chain's runtime reconfigure surface
(`api/internal/ai/ttschain/chain.go:95`) is invoked by the settings
domain via `chains.Coordinator` when an operator toggles tiers; see
[`../settings/byok-and-voice-overrides.md`](../settings/byok-and-voice-overrides.md).

## Failure Modes

| Cause | Symptom | Chain behavior | Wire mapping (`mapChainError`, `api/handlers/tts/connect_handler.go:160`) |
|---|---|---|---|
| BYOK key present but `BYOKProvider` blank | Misconfigured envelope | Short-circuits with `ErrMissingBYOKProvider` | `CodeInvalidArgument` |
| Unknown `BYOKProvider` value | Adapter not in `byok.NewRegistries()` | Short-circuits with `ErrUnknownBYOKProvider` | `CodeInvalidArgument` |
| BYOK adapter HTTP error (timeout, 5xx) | Vendor outage / network issue | Records `lastErr`, falls through to Vrooli then Local | `CodeInternal` if no tier succeeds |
| LPBS reports insufficient credits | Vrooli tier returns `ErrInsufficientCredits` | Short-circuits — does NOT fall through to Local | `CodeResourceExhausted` |
| LPBS transport error | Vrooli unreachable | Records `lastErr`, falls through to Local | `CodeInternal` if Local also fails |
| sherpa-onnx/Kokoro returns unsupported format | Local tier rejects | `tts.Service` returns `ErrInvalidArgument` | `CodeInternal` (wrapped) |
| All tiers disabled / unavailable | No eligible provider | Returns `ErrAllProvidersFailed` | `CodeUnavailable` |
| Streaming negotiation: nothing streams | All `StreamingCapability()=false` | Buffered fallback emits one final frame | Success — wire-shape preserved |
| Chain not wired (`Deps.Chain == nil`) | Bootstrap regression | Handler returns early | `CodeUnavailable` ("tts chain not configured") |

Availability caching (`api/internal/ai/ttschain/chain.go:188`) means a
freshly-broken BYOK key will keep getting tried until the TTL expires
(default 5 minutes). Operators toggling tiers via `SettingsService`
trigger `Reconfigure` which invalidates the cache immediately.

## Capacity Notes

The unary path is single-shot per request; concurrency is bounded by
the upstream provider's rate limits, not by the chain. Streaming
sessions hold one goroutine per active session per provider for the
duration of synthesis; for the buffered fallback that's the duration
of `Execute()` plus the one-frame send.

The local sherpa-onnx/Kokoro backend is the dominant bottleneck on a single
host: synthesis runs CPU-bound (or accelerator-backed) and is serialized by
the native resource's model runtime regardless of how many chain calls hit it
in parallel. BYOK and Vrooli scale with the upstream service; the chain
adds only the constant overhead of envelope parse and tier-loop
evaluation.

Cache hits avoid the entire chain: `tts.Service.Synthesize` consults
`Deps.GetCache` keyed on `(EventID, Voice, Speed, Version)`
(`api/internal/tts/service.go:27`) before dispatching, so re-synthesizing
the same `(event, version)` pair is an O(1) lookup.

Usage rows are written through `usagereport.Recorder` (the same
async pipeline summarize uses); see
[`../usage/reporting.md`](../usage/reporting.md). The current TTS
Connect handler does not yet thread `Usage` like the summarize handler
does — adding it follows the same shape.

## Cross-References

- [`../../internal/SEAMS.md`](../../internal/SEAMS.md) — full seam registry
- [`../../internal/DECISIONS.md`](../../internal/DECISIONS.md) — durable decisions (local-first speech ordering, credit short-circuit)
- [`../../internal/PROBLEMS.md`](../../internal/PROBLEMS.md) — current drift
- [`../../reference/configuration.md`](../../reference/configuration.md) — operator-tunable levers
- [`../summarize/chain.md`](../summarize/chain.md) — sibling chain with identical tier shape
- [`../settings/byok-and-voice-overrides.md`](../settings/byok-and-voice-overrides.md) — credential & voice-mapping storage
- `packages/proto/schemas/audio-tools/v1/tts/tts.proto` — wire shape
# Provider tiers and BYOK fallback

Audio-tools prefers Kokoro when it is available. A configured BYOK TTS
provider is the next service tier, followed by Vrooli/LPBS and browser speech
synthesis in the adopting UI. Configure BYOK credentials with `audio-tools
settings providers`; server-side requests use the configured encrypted-store
credential when no explicit request credential is supplied.
Health responses expose provider state and the selected tier, but never expose
credential values. Browser fallback must be announced to the user with its
reason.
