# Research Conclusion: AI Provider Routing for Audio Capabilities

## Status: Draft (Round 2)

## Research Question

How should the audio-tools scenario implement provider routing for its three AI capabilities (Whisper STT, Kokoro TTS, Ollama summarization), given the existing BAS three-tier provider chain pattern (BYOK → Vrooli/LPBS → Dev/Local)?

### Success Criteria
- Clear provider routing architecture that handles audio-specific modalities (not just text-in/text-out like BAS)
- Per-capability provider matrix documenting local requirements, external API options, and LPBS contracts
- Configuration schema that allows callers to specify routing preferences per-capability
- Reuse assessment with concrete recommendation

## Settled Decisions

### D1 (Round 1): Per-capability typed provider chains
Each capability (STT, TTS, summarize) gets its own Go interface and chain with capability-specific request/response types. No generics, no unified gateway.

### D2 (Round 1): LPBS for all three capabilities
All three capabilities should be routable through LPBS. This requires new LPBS gateway endpoints for STT (audio upload) and TTS (audio download). Summarization can reuse the existing `/api/v1/ai/chat` endpoint.

### D3 (Round 1): Replicate BAS pattern, don't extract shared code
Audio-tools implements its own provider chains following BAS patterns. BAS code stays unchanged. The interfaces differ too much (text vs. audio modalities) to share a generic package.

## Findings

### Finding 1: BAS Provider Chain — Exact Pattern to Replicate

The BAS chain (`scenarios/browser-automation-studio/api/services/ai/`) provides the blueprint:

**Provider Interface (4 methods per provider):**
- `Type() ProviderType` — identifies the tier (byok/vrooli/dev)
- `IsAvailable(ctx) bool` — cached health check
- `Execute(ctx, req) (result, error)` — the actual operation (this becomes Transcribe/Synthesize/Summarize for audio)
- `Model() string` — model identifier

**Chain Fallback Rules:**
1. Try BYOK if enabled AND request has BYOK key → on success, return (ChargedCredits=false)
2. Try Vrooli if enabled AND request has LPBS auth token → on success, return (ChargedCredits=true)
3. Try Dev/Local if enabled AND local resource available → on success, return (ChargedCredits=false)
4. All failed → return ErrAllProvidersUnavailable

**Critical error rule:** `ErrInsufficientCredits` stops fallback immediately. All other errors continue to next provider.

**Availability Caching (per-tier TTLs):**
- BYOK: 5-minute TTL, validates API key against provider endpoint
- Vrooli: 30-second TTL, health check against LPBS `/api/v1/ai/health`
- Dev/Local: One-time check (PATH lookup or localhost health probe)

**Factory Pattern:** Shared chain instance + per-request client that injects user credentials (BYOK key, LPBS token, user identity). Handler creates client via factory, passes to service layer.

**LPBS Usage Reporting:** Async goroutine with 3-attempt exponential backoff (500ms, 1s, 2s). Uses `OperationID` (UUID) for idempotency across retries. Report structure: `{UserIdentity, LimitKey: "ai_credits", Amount, AppBundleKey, OperationID, Metadata: {Operation, Model}}`.

**Hard-coded costs:** Operation costs are constants in code (security measure — no env var overrides).

### Finding 2: Per-Capability Provider Interfaces

Based on the BAS pattern adapted for audio modalities:

**STTProvider (Speech-to-Text):**
```go
type STTProvider interface {
    Type() ProviderType
    IsAvailable(ctx context.Context) bool
    Transcribe(ctx context.Context, audio []byte, opts STTOptions) (*STTResult, error)
}

type STTOptions struct {
    Language string // e.g., "en", "" for auto-detect
    Format   string // "wav", "mp3", "opus", "webm"
}

type STTResult struct {
    Text     string
    Language string // detected language
    Duration float64 // audio duration in seconds (for billing)
}
```

**TTSProvider (Text-to-Speech):**
```go
type TTSProvider interface {
    Type() ProviderType
    IsAvailable(ctx context.Context) bool
    Synthesize(ctx context.Context, req TTSRequest) (*TTSResult, error)
}

type TTSRequest struct {
    Text           string
    Voice          string // e.g., "af_heart"
    Speed          float64 // 0.5-4.0
    ResponseFormat string // "mp3", "wav", "opus"
}

type TTSResult struct {
    Audio       []byte
    ContentType string // e.g., "audio/mpeg"
    Duration    float64 // output duration in seconds (for billing)
}
```

**SummarizeProvider (Text Summarization):**
```go
type SummarizeProvider interface {
    Type() ProviderType
    IsAvailable(ctx context.Context) bool
    Summarize(ctx context.Context, text string, level string) (*SummarizeResult, error)
}

type SummarizeResult struct {
    Text         string
    PromptTokens int // for billing
    OutputTokens int
}
```

Each gets its own chain: `STTChain`, `TTSChain`, `SummarizeChain` — all following the BYOK → Vrooli → Local fallback order.

### Finding 3: Current Audio Stack Is Entirely Local

The web-console audio implementation has no routing layer:

| Capability | Endpoint | Local URL | Config |
|-----------|----------|-----------|--------|
| Whisper STT (batch) | `POST /api/v1/voice/transcribe` | `$WHISPER_URL` (localhost:8090) | Multipart audio → JSON {text} |
| Whisper STT (stream) | `WS /api/v1/voice/stream` | Same | WebSocket with partial/final/segment events |
| Kokoro TTS | `POST /api/v1/tts/synthesize` | `$KOKORO_URL` (localhost:8880) | JSON → raw audio bytes |
| Kokoro voices | `GET /api/v1/tts/voices` | Same | Voice list |
| TTS cache | `GET /api/v1/tts/cache/{eventId}` | Local storage | Cached audio retrieval |
| Ollama summarize | Internal (not exposed) | `$OLLAMA_URL` (localhost:11434) | POST /api/chat with level-specific prompts |

**Key details:**
- STT has hallucination filtering (`isWhisperHallucination()`)
- Summarizer has three levels (light/moderate/heavy) with level-specific system prompts
- Summarizer strips `<think>` tags from reasoning models (qwen3)
- TTS supports opportunistic caching by eventId
- All timeouts are 30s (STT, TTS) or configurable up to 300s (summarization)

### Finding 4: LPBS Gateway Extension Requirements

The LPBS AI gateway (`scenarios/landing-page-business-suite/api/`) currently only handles text chat:

**Existing endpoints:**
- `POST /api/v1/ai/chat` — non-streaming chat completion
- `POST /api/v1/ai/stream` — SSE streaming chat completion
- `GET /api/v1/ai/usage` — usage statistics
- `GET /api/v1/ai/models` — model list
- `GET /api/v1/ai/health` — health check

**New endpoints needed for audio:**
1. `POST /api/v1/ai/transcribe` — accepts multipart audio, proxies to cloud STT, charges credits
2. `POST /api/v1/ai/synthesize` — accepts text+voice JSON, proxies to cloud TTS, returns audio bytes
3. Summarization can reuse `/api/v1/ai/chat` with appropriate operation metadata

**Credit system reuse:**
- The LPBS reservation system (ReserveCredits → Execute → FinalizeReservation with 10-min auto-expiry) works for audio
- Key extension: audio pricing needs per-second or per-byte units rather than per-token
- Internal credit unit is 1/1,000,000 of a cent — flexible enough for audio pricing
- Idempotent reporting via OperationID works unchanged

**LPBS scope expansion is a separate execution item** — audio-tools can be built with local-only + BYOK first, with LPBS routing added once gateway endpoints exist.

### Finding 5: External API Options per Capability

**STT (Speech-to-Text) — BYOK tier:**
- OpenAI Whisper API ($0.006/min) — closest to local Whisper, same model family
- Deepgram ($0.0043/min Nova-2) — faster, cheaper, good accuracy
- AssemblyAI ($0.00025/sec ≈ $0.015/min) — feature-rich (diarization, sentiment)
- Google Cloud Speech-to-Text ($0.006/min for short, $0.012/min for long audio)

**TTS (Text-to-Speech) — BYOK tier:**
- OpenAI TTS ($15/1M chars HD, $0.015/1K chars) — simple API, good quality
- ElevenLabs ($0.18/1K chars on Creator plan) — best voice quality, expensive
- Google Cloud TTS ($4/1M chars standard, $16/1M WaveNet) — reliable, many languages

**Summarization — BYOK tier:**
- OpenRouter (already used by BAS) — multi-model access, $0.15-$15/1M tokens depending on model
- Direct OpenAI/Anthropic API — same models, no OpenRouter markup

### Finding 6: Operation Types and Cost Definitions

Following BAS's hard-coded cost pattern, audio-tools would define:

```go
const (
    OpAudioTranscribe  OperationType = "audio.stt_transcribe"
    OpAudioSynthesize  OperationType = "audio.tts_synthesize"
    OpAudioSummarize   OperationType = "audio.summarize"
)
```

LPBS usage metadata would include:
- STT: `{Operation: "audio.stt_transcribe", AudioDurationSec: N, Model: "whisper-large-v3"}`
- TTS: `{Operation: "audio.tts_synthesize", OutputDurationSec: N, Voice: "af_heart", Model: "kokoro"}`
- Summarize: `{Operation: "audio.summarize", PromptTokens: N, Model: "qwen3:1.7b"}`

## Per-Capability Provider Matrix

| | Local | BYOK (External API) | LPBS/Vrooli |
|---|---|---|---|
| **STT** | Whisper (localhost:8090). VRAM: 39MB (tiny) to 1.5GB (large). Free. | OpenAI Whisper API, Deepgram, AssemblyAI. User provides API key. | New LPBS `/ai/transcribe` endpoint needed. Proxies to cloud STT. Credits by duration. |
| **TTS** | Kokoro (localhost:8880). VRAM: ~82MB. Free. | OpenAI TTS, ElevenLabs, Google Cloud TTS. User provides API key. | New LPBS `/ai/synthesize` endpoint needed. Proxies to cloud TTS. Credits by output duration or char count. |
| **Summarize** | Ollama (localhost:11434). VRAM: varies by model (qwen3:1.7b ≈ 1.5GB). Free. | OpenRouter (same as BAS). User provides OpenRouter key. | Existing LPBS `/ai/chat` endpoint works. Credits by token count. |

## Limitations & Unknowns

- **LPBS gateway scope:** Adding audio endpoints is a significant cross-scenario effort requiring its own execution item. Audio-tools should be buildable with local + BYOK first.
- **Audio pricing model:** Per-second vs. flat-per-operation vs. tiered buckets — pending decision (round 2 d1).
- **WebSocket streaming routing:** Whether STT streaming should go through the provider chain or stay local-only — pending decision (round 2 d2).
- **Configuration schema:** How callers specify routing preferences — pending decision (round 2 d3).
- **BYOK provider heterogeneity:** Each external STT/TTS API has different auth, request format, and pricing. The BYOK tier may need provider-specific adapters (e.g., OpenAIWhisperBYOK, DeepgramBYOK) rather than one generic BYOK provider per capability.
- **Voice mapping across providers:** Kokoro voice IDs (e.g., "af_heart") won't match OpenAI or ElevenLabs voice IDs. Need a voice mapping layer or let callers specify provider-specific voice IDs.
- **Streaming TTS:** Some external TTS APIs support streaming audio output. Not addressed yet.

## Actions

1. **Create LPBS audio gateway execution item** — Define and implement `/ai/transcribe` and `/ai/synthesize` endpoints in LPBS. Depends on pricing model decision.
2. **Create audio-tools greenfield scenario** — Already exists as `execute/audio-tools-greenfield-scenario` in the initiative. Should implement the per-capability typed provider chains defined here.
3. **Define BYOK adapter interfaces** — For each external API option (OpenAI Whisper, Deepgram, ElevenLabs, OpenRouter), define the adapter that maps audio-tools provider interface to the external API contract.
4. **Resolve round 2 pending decisions** — Audio pricing model, WebSocket streaming routing, and configuration schema need user input before the architecture is fully specified.

## Confidence Assessment

| Area | Confidence | Notes |
|------|-----------|-------|
| BAS pattern analysis | High | Code thoroughly reviewed with exact signatures and flow |
| Per-capability interfaces | High | Typed interfaces defined based on actual API contracts |
| Current audio stack | High | Complete API contracts documented from source |
| LPBS integration path | Medium-High | Reservation system reusable; new endpoints well-scoped |
| External API options | Medium | Options identified with pricing; detailed adapter design pending |
| Audio pricing model | Low | Pending decision — multiple viable approaches |
| Configuration schema | Low | Pending decision — options defined but not selected |
