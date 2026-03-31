# Research Conclusion: AI Provider Routing for Audio Capabilities

## Status: Draft (Round 1)

## Research Question

How should the audio-tools scenario implement provider routing for its three AI capabilities (Whisper STT, Kokoro TTS, Ollama summarization), given the existing BAS three-tier provider chain pattern (BYOK → Vrooli/LPBS → Dev/Local)?

### Success Criteria
- Clear provider routing architecture that handles audio-specific modalities (not just text-in/text-out like BAS)
- Per-capability provider matrix documenting local requirements, external API options, and LPBS contracts
- Configuration schema that allows callers to specify routing preferences per-capability
- Reuse assessment with concrete recommendation

## Findings

### Finding 1: BAS Provider Chain is Text-Only by Design

The BAS provider chain (`scenarios/browser-automation-studio/api/services/ai/`) is built around a single modality: **text prompt → text response**. The core interface is:

```go
type AIProvider interface {
    ExecutePrompt(ctx context.Context, prompt string) (string, error)
    // ...
}
```

All three BAS providers (BYOK/OpenRouter, Vrooli/LPBS, Dev/local) route through OpenRouter-compatible APIs that accept text and return text. This is fundamentally different from audio capabilities:

- **Whisper (STT):** Audio bytes in → text out (multipart file upload, not JSON prompt)
- **Kokoro (TTS):** Text in → audio bytes out (binary response, not JSON text)
- **Ollama (summarization):** Text in → text out (closest to BAS pattern, but uses Ollama's `/api/chat` not OpenRouter)

**Implication:** The BAS `AIProvider` interface cannot be reused directly — audio-tools needs a different provider interface per capability, or a more generic abstraction.

### Finding 2: Audio Providers Have Heterogeneous APIs

Each audio capability has a fundamentally different API contract:

| Capability | Local API | Input | Output | Protocol |
|-----------|-----------|-------|--------|----------|
| **Whisper STT** | `localhost:8090/asr?output=json` | Multipart audio file | JSON with text | HTTP POST |
| **Kokoro TTS** | `localhost:8880/v1/audio/speech` | JSON (text, voice, format) | Audio bytes (mp3/wav/opus) | HTTP POST |
| **Ollama summarize** | `localhost:11434/api/chat` | JSON (model, messages) | JSON with text | HTTP POST |

External/cloud equivalents would be:
- **Whisper STT:** OpenAI Whisper API, Deepgram, AssemblyAI — all have different request formats
- **Kokoro TTS:** OpenAI TTS API, ElevenLabs, Google Cloud TTS — varying formats
- **Ollama summarize:** OpenRouter, OpenAI, Anthropic — most similar to BAS pattern

**Implication:** A single unified provider interface won't work. The routing system needs per-capability provider interfaces with capability-specific request/response types.

### Finding 3: BAS Patterns Worth Reusing (Contract, Not Code)

While the BAS `AIProvider` interface is text-specific, several architectural patterns are highly reusable:

1. **Chain fallback logic:** Try BYOK → Vrooli → Local, with configurable enable/disable per tier
2. **LPBS credit integration:** Atomic charging via LPBS gateway, idempotent usage reporting with `OperationID`
3. **Availability caching:** TTL-based health checks to avoid per-request latency
4. **Factory pattern:** Per-request client creation with user credentials (BYOK key, LPBS auth token)
5. **Configuration schema:** Per-provider enable flags + URLs via environment variables
6. **Error classification:** `ErrInsufficientCredits` stops fallback (don't fall back to free tier when credits exhausted)

These patterns can be replicated for audio-tools without extracting BAS code into a shared package.

### Finding 4: Current Audio Stack is Local-Only

The web-console currently connects directly to local resources:
- Whisper at `WHISPER_URL` (default `localhost:8090`)
- Kokoro at hardcoded port 8880
- Ollama at `localhost:11434`

There is **no provider routing, no BYOK support, no LPBS credit integration** for any audio capability. The audio-tools scenario would add this layer.

### Finding 5: LPBS Gateway Needs Audio Endpoints

The LPBS AI gateway currently only has `/api/v1/ai/health` and chat completion endpoints (text-based). For audio routing through LPBS, the gateway would need:
- STT endpoint accepting audio uploads
- TTS endpoint returning audio bytes
- Or: audio-tools routes through LPBS only for the summarization (text) capability, and handles STT/TTS locally or via direct external APIs

This is a significant scope question — LPBS gateway changes are outside the audio-tools scenario.

### Finding 6: Three Viable Architecture Approaches

**A) Per-capability provider chains (recommended):** Each capability (STT, TTS, summarize) gets its own typed provider interface and chain. Reuse the fallback/credit patterns from BAS but with capability-specific types.

**B) Generic provider with adapters:** One generic `Provider[Req, Resp]` interface with type parameters, plus adapters that convert capability-specific requests to the generic form.

**C) Unified audio gateway:** A single API gateway that accepts all audio requests and routes internally. More complex but cleaner external API.

## Limitations & Unknowns

- **LPBS gateway scope:** Whether LPBS will add audio endpoints is unknown and impacts the Vrooli tier for STT/TTS
- **External API costs:** Pricing for cloud STT/TTS APIs not yet researched (needed for credit metering)
- **VRAM requirements:** Local Whisper model sizes range from 39MB (tiny) to 1.5GB (large); Kokoro is 82M; these affect which models are practical for local tier
- **Streaming:** Whisper has WebSocket streaming support locally; unclear if external/LPBS tiers would support streaming

## Actions

<!-- TBD — pending decisions on architecture approach and LPBS scope -->

## Confidence Assessment

| Area | Confidence | Notes |
|------|-----------|-------|
| BAS pattern analysis | High | Code thoroughly reviewed |
| Audio capability mapping | High | Current implementations documented |
| Architecture options | Medium | Need user direction on approach |
| LPBS integration | Low | Gateway changes not yet scoped |
| External API options | Low | Not yet researched in detail |
