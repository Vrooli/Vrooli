# Audio-Tools Greenfield Scenario — Implementation Plan

> **Status:** Seed plan. Only the Provider Routing Architecture is finalized (from `research/audio-provider-routing-contract`). Other sections are `<!-- TBD -->` and will be populated during workshop rounds for this execute item.

## 1. Purpose

Build a greenfield `audio-tools` scenario that serves as a shared audio microservice for any Vrooli scenario needing audio capabilities. The scenario replaces the ad-hoc audio stack currently embedded in `web-console` and becomes the canonical home for Whisper (STT), Kokoro (TTS), and Ollama-based summarization, with a three-tier provider chain (BYOK → Vrooli/LPBS → Local) inspired by BAS.

## 2. Required Reading

```bash
# Canonical authoring/plan skills
prompt-manager skill read cli-steer api-steer utils-unification seam-discovery-and-enforcement

# BAS provider chain — the pattern this scenario replicates per-capability
# (replicate, do not extract; see research D3)
ls scenarios/browser-automation-studio/api/services/ai/

# Research inputs for this execute item
swarm-manager backlog file-get --kind research --name audio-provider-routing-contract --path conclusion.md
swarm-manager backlog file-get --kind research --name audio-tools-shared-scenario-contract --path conclusion.md 2>/dev/null || true

# LPBS gateway (the target of the audio endpoints this scenario depends on)
ls scenarios/landing-page-business-suite/api/
```

## 3. Problem Statement

<!-- TBD via workshop — summarize current web-console audio stack pain points, the need for a shared scenario, and the integration seams required by swarm-manager/agent-manager. -->

## 4. Scope

**In scope:**
- Greenfield `scenarios/audio-tools/` scenario with its own API, CLI, and UI (per project convention — every scenario has a UI).
- Per-capability typed provider chains (STT, TTS, Summarize) following the settled architecture in Section 8.
- BYOK adapters for a starter set of external APIs (see `execute/audio-tools-byok-adapters`).
- HTTP batch flows route through the provider chain.

**Out of scope:**
- LPBS tier activation — depends on `execute/lpbs-audio-gateway-endpoints`. Ship audio-tools with local + BYOK tiers first.
- Streaming STT/TTS through the provider chain — WebSocket STT stays local-only (Round 2 D2).
- Audio format conversion at the adapter boundary — reject unsupported formats with a clear error.
- Modifying the existing web-console audio stack — that lives in `execute/web-console-adopt-audio-tools`.

## 5. Current Technical Context

<!-- TBD via workshop — capture file-level references for the existing web-console audio endpoints, Whisper/Kokoro/Ollama local URLs, and the BAS chain files to mirror. Starter list from research: -->

- `scenarios/browser-automation-studio/api/services/ai/` — BAS chain to mirror per-capability.
- `scenarios/browser-automation-studio/api/services/credits/` — credit reporting pattern (hard-coded costs, async retries, idempotent OperationID).
- Existing local-only audio endpoints to port: `POST /api/v1/voice/transcribe`, `WS /api/v1/voice/stream`, `POST /api/v1/tts/synthesize`, `GET /api/v1/tts/voices`, `GET /api/v1/tts/cache/{eventId}`.
- Local resources: Whisper (`$WHISPER_URL`, default `localhost:8090`), Kokoro (`$KOKORO_URL`, default `localhost:8880`), Ollama (`$OLLAMA_URL`, default `localhost:11434`).

## 6. Target End State

<!-- TBD via workshop — end-to-end narrative from a caller requesting transcription with BYOK credentials, through the chain resolving to the selected adapter, through LPBS metering on the Vrooli tier, and through local fallback. -->

Key invariants that must hold:
- Provider precedence is fixed at `BYOK → Vrooli/LPBS → Local`. Not caller-configurable.
- `ErrInsufficientCredits` from the Vrooli tier short-circuits (does not fall through to Local).
- `voice_overrides` map wins over canonical voice mapping; unmapped canonical voice falls back to the adapter's default and logs a structured warning.
- Usage reporting is idempotent per `OperationID` under retry.

## 7. Implementation Strategy (phased)

<!-- TBD via workshop. Rough phasing implied by the research: -->

- **Phase 1:** Scaffold scenario (api/cli/ui), define per-capability provider interfaces, ship local-only providers + chain skeleton with BYOK/LPBS tiers disabled by flag.
- **Phase 2:** Ship BYOK tier with adapter registry and `X-Audio-BYOK-Provider` header routing (depends on `execute/audio-tools-byok-adapters`).
- **Phase 3:** Activate Vrooli/LPBS tier once `execute/lpbs-audio-gateway-endpoints` lands — add LPBS provider implementations, wire usage reporting via `OperationID`.
- **Phase 4:** Web-console, swarm-manager, agent-manager adoption (separate execute items already in the initiative).

## 8. Contract Decisions — Provider Routing Architecture (FINAL)

This is the settled architecture from `research/audio-provider-routing-contract` (Rounds 1–3). All other execute items in this initiative build against these contracts.

### 8.1 Per-capability typed provider chains (Round 1 D1)

Each of the three capabilities has its own Go interface, its own chain, and its own set of provider implementations. **No generic unified chain. No shared generic package.** The scenario replicates the BAS chain pattern per-capability rather than extracting shared code (Round 1 D3).

```go
type STTProvider interface {
    Type() ProviderType
    IsAvailable(ctx context.Context) bool
    Transcribe(ctx context.Context, audio []byte, opts STTOptions) (*STTResult, error)
}

type STTOptions struct {
    Language string // e.g., "en"; "" = auto-detect
    Format   string // "wav" | "mp3" | "opus" | "webm"
}

type STTResult struct {
    Text     string
    Language string  // detected language
    Duration float64 // audio duration in seconds (for billing)
}

type TTSProvider interface {
    Type() ProviderType
    IsAvailable(ctx context.Context) bool
    Synthesize(ctx context.Context, req TTSRequest) (*TTSResult, error)
}

type TTSRequest struct {
    Text           string
    Voice          string            // canonical voice ID (see 8.5)
    VoiceOverrides map[string]string // optional: {"byok:elevenlabs": "Rachel", ...}
    Speed          float64           // 0.5–4.0
    ResponseFormat string            // "mp3" | "wav" | "opus"
}

type TTSResult struct {
    Audio       []byte
    ContentType string  // e.g., "audio/mpeg"
    Duration    float64 // output duration in seconds (for billing)
}

type SummarizeProvider interface {
    Type() ProviderType
    IsAvailable(ctx context.Context) bool
    Summarize(ctx context.Context, text string, level string) (*SummarizeResult, error)
}

type SummarizeResult struct {
    Text         string
    PromptTokens int
    OutputTokens int
}
```

Each chain (`STTChain`, `TTSChain`, `SummarizeChain`) owns an ordered list of providers keyed by tier and an availability cache with per-tier TTLs mirroring BAS:

- BYOK: 5-minute TTL, validates against the provider endpoint.
- Vrooli/LPBS: 30-second TTL, probes LPBS `/api/v1/ai/health`.
- Local: one-time check at process start (PATH lookup or localhost probe).

### 8.2 Provider precedence (Round 3 D3)

The chain always tries providers in the order **`BYOK → Vrooli/LPBS → Local`**, matching BAS. Precedence is **not caller-configurable** — there is no `X-Audio-Prefer` header and no per-capability override.

Fallback semantics:
1. Try BYOK if enabled AND request carries both `X-Audio-BYOK-Key` and `X-Audio-BYOK-Provider` AND the selected adapter passes availability. On success, return `ChargedCredits=false`.
2. Try Vrooli/LPBS if enabled AND request carries `X-Audio-LPBS-Token` (+ `X-Audio-User-Identity`) AND LPBS health check passes. On success, return `ChargedCredits=true`.
3. Try Local if enabled AND local resource reachable. On success, return `ChargedCredits=false`.
4. All failed → return `ErrAllProvidersUnavailable`.

**Critical error rule:** `ErrInsufficientCredits` from the Vrooli tier stops fallback immediately (does NOT cascade to Local). All other errors continue to the next provider in the chain. Callers who want to force local execution omit LPBS/BYOK credentials; callers who want BYOK pass the BYOK headers.

### 8.3 BYOK adapter registry (Round 3 D1)

One adapter per external API, each implementing the appropriate provider interface directly (no internal dispatcher). Adapters are registered in a per-capability registry keyed by a short provider id. The caller selects one via the `X-Audio-BYOK-Provider` header; `X-Audio-BYOK-Key` supplies the credential.

Starter registry (ships with `execute/audio-tools-byok-adapters`):

| Provider ID       | Capability  | Notes |
|-------------------|-------------|-------|
| `openai-whisper`  | STT         | POST multipart, `Authorization: Bearer` |
| `deepgram`        | STT         | POST raw audio, `Authorization: Token` |
| `openai-tts`      | TTS         | POST JSON, `Authorization: Bearer`, returns audio bytes |
| `elevenlabs`      | TTS         | POST JSON, `xi-api-key` header, returns audio stream |
| `openrouter`      | Summarize   | Reuses BAS's OpenRouter integration pattern |

Unknown provider id → `ErrUnknownBYOKProvider` (400-class, NO fallback to Vrooli/Local — caller error). Missing `X-Audio-BYOK-Provider` when `X-Audio-BYOK-Key` is present → `ErrMissingBYOKProvider` (400-class).

### 8.4 LPBS/Vrooli tier contract

Depends on `execute/lpbs-audio-gateway-endpoints`. Audio-tools ships with the LPBS tier wired but disabled by flag until that endpoint work lands. When enabled:

- STT routes to `POST <lpbs>/api/v1/ai/transcribe` (multipart upload; charged by `audio_duration_sec`).
- TTS routes to `POST <lpbs>/api/v1/ai/synthesize` (JSON body; returns audio bytes; charged by output `audio_duration_sec`).
- Summarize routes to existing `POST <lpbs>/api/v1/ai/chat` with `Operation: audio.summarize` metadata (charged by tokens).

Usage reporting mirrors BAS: async goroutine, 3-attempt exponential backoff (500ms, 1s, 2s), idempotent per `OperationID` (UUID). Report metadata includes `{UserIdentity, LimitKey: "ai_credits", Amount, AppBundleKey: "audio-tools", OperationID, Metadata: {Operation, Model}}`.

### 8.5 Canonical voices with per-provider overrides (Round 3 D2)

Audio-tools defines a canonical voice catalog. Each TTS adapter declares a mapping from canonical IDs to its concrete voice IDs. Callers may pass a `voice_overrides` map on the TTS request to pin exact voice IDs per adapter.

**Starter canonical voice set (extensible):**

- `voice.feminine.warm`
- `voice.feminine.neutral`
- `voice.masculine.warm`
- `voice.masculine.neutral`
- `voice.neutral.default`

**Override key shape:** `byok:<provider-id>`, `vrooli:<provider-id>`, or `local:<engine-id>`. Example: `{"byok:elevenlabs": "Rachel", "byok:openai-tts": "alloy", "local:kokoro": "af_heart"}`.

**Resolution order for each TTS request:**
1. If `voice_overrides["tier:provider-id"]` is set, use it.
2. Else, look up the canonical mapping declared by the selected adapter.
3. Else, use the adapter's default voice and emit a structured log warning naming the unmapped canonical ID.

Every registered adapter MUST declare a mapping entry for every canonical voice at registration time; violations fail startup.

### 8.6 Operation type constants (Finding 6)

```go
const (
    OpAudioTranscribe OperationType = "audio.stt_transcribe"
    OpAudioSynthesize OperationType = "audio.tts_synthesize"
    OpAudioSummarize  OperationType = "audio.summarize"
)
```

Costs are hard-coded constants in source (security measure — no env var overrides, matching BAS).

### 8.7 Configuration schema (Finding 7)

**Per-instance defaults (env vars at audio-tools startup):**

```
# Master tier enable flags
AUDIO_AI_ENABLE_BYOK=true
AUDIO_AI_ENABLE_VROOLI=false   # flip to true once execute/lpbs-audio-gateway-endpoints lands
AUDIO_AI_ENABLE_LOCAL=true

# Local endpoints
AUDIO_WHISPER_URL=http://localhost:8090
AUDIO_KOKORO_URL=http://localhost:8880
AUDIO_OLLAMA_URL=http://localhost:11434

# LPBS (Vrooli) endpoint
AUDIO_LPBS_BASE_URL=https://lpbs.example.com
AUDIO_LPBS_APP_BUNDLE_KEY=audio-tools

# Availability cache TTLs (match BAS defaults)
AUDIO_AVAIL_TTL_BYOK=5m
AUDIO_AVAIL_TTL_VROOLI=30s
```

**Per-request headers (HTTP):**

```
X-Audio-BYOK-Key:        <provider-specific API key>            # optional
X-Audio-BYOK-Provider:   openai-whisper|deepgram|openai-tts|... # required when BYOK key is set
X-Audio-LPBS-Token:      <LPBS auth token>                      # optional
X-Audio-User-Identity:   <user id>                              # required when LPBS token is set
```

### 8.8 BAS pattern replication details (Finding 1)

Mirror these BAS patterns exactly for each per-capability chain:
- **4-method provider interface:** `Type()`, `IsAvailable(ctx)`, `Execute(...)` (renamed to `Transcribe`/`Synthesize`/`Summarize`), `Model()`.
- **Factory pattern:** shared chain instance + per-request client that injects the caller's BYOK key, LPBS token, and user identity. Handler creates client via factory and passes to service layer.
- **Hard-coded costs:** operation costs are `const` values in code.
- **Async usage reporting with backoff:** 3 attempts at 500ms/1s/2s, idempotent by `OperationID`.
- **Credential redaction:** reuse BAS's credential-redacting logger so BYOK keys and LPBS tokens never appear in logs.
- **Circuit-breaker on repeated failure:** consider adding after 3 consecutive failures per-provider (Risk Matrix mitigation).

## 9. Testing Plan

<!-- TBD via workshop. Research specifies verification paths per finding (F1–F10). Minimum must cover: -->

- Unit tests per chain asserting fallback order (`BYOK → Vrooli → Local`) with fake providers.
- `ErrInsufficientCredits` short-circuit test (Vrooli credit error must NOT cascade to Local).
- Registry lookup test: `X-Audio-BYOK-Provider` routes to the correct adapter; unknown id → `ErrUnknownBYOKProvider` with no fallback.
- Voice resolution test: `voice_overrides["byok:elevenlabs"]` wins over canonical mapping; missing canonical mapping falls back to adapter default and logs a warning.
- Adapter-level contract tests per external API (recorded fixtures or sandbox keys).
- Integration test via `vrooli scenario test audio-tools`.

## 10. Rollout / Validation Checklist

<!-- TBD via workshop -->

- [ ] Scaffold scenario with `vrooli scenario ... react-vite` UI template.
- [ ] Local-only chain passes all tests with BYOK/LPBS disabled.
- [ ] BYOK tier passes adapter contract tests once `execute/audio-tools-byok-adapters` lands.
- [ ] LPBS tier activates and reports usage correctly once `execute/lpbs-audio-gateway-endpoints` lands.

## 11. Risks + Mitigations

Condensed from research Risk Matrix (see `research/audio-provider-routing-contract` for full table):

| Risk | Mitigation |
|------|-----------|
| LPBS audio endpoints slip | Ship audio-tools with local + BYOK only; activate Vrooli tier by flag once `execute/lpbs-audio-gateway-endpoints` lands. |
| BYOK adapter proliferation | Registry pattern keeps adapters isolated; start with 5-adapter set, add more on demand. |
| Canonical voice catalog drift | Single-file canonical catalog; require mapping entries per adapter at registration; `voice_overrides` escape hatch. |
| Audio format mismatch | Reject unsupported formats at adapter boundary with clear error; no silent conversion. |
| Hallucination filter drift (Whisper local vs cloud) | Keep hallucination filter at audio-tools boundary (post-provider), not inside any one provider. |
| Credential leaking in logs | Reuse BAS credential-redacting logger. |
| Fallback thrashing on flaky network | Respect availability cache TTLs; circuit-breaker after 3 consecutive failures per provider. |

## 12. Non-goals / Prohibited Patterns

- **No unified generic provider interface across capabilities.** STT/TTS/Summarize get their own typed interfaces (Round 1 D1).
- **No shared `provider-chain` package extracted from BAS.** Replicate the pattern; don't extract (Round 1 D3).
- **No caller-configurable precedence.** The order is fixed (Round 3 D3). Do not add `X-Audio-Prefer` or similar.
- **No internal BYOK dispatcher.** One adapter per external API, selected by `X-Audio-BYOK-Provider` (Round 3 D1).
- **No transparent audio format conversion at the adapter layer.** Reject unsupported formats (Limitations section of research).
- **No streaming STT/TTS through the provider chain.** WebSocket STT stays local-only (Round 2 D2).

## 13. Definition of Done

- [ ] `scenarios/audio-tools/` exists with api/cli/ui, `service.json`, and passing `vrooli scenario test audio-tools`.
- [ ] All three provider chains (STT, TTS, Summarize) implement the interfaces in §8.1 and the precedence rules in §8.2.
- [ ] Local providers work with Whisper/Kokoro/Ollama out of the box.
- [ ] BYOK tier selects adapters via `X-Audio-BYOK-Provider`; unknown id returns `ErrUnknownBYOKProvider` without fallback.
- [ ] Vrooli/LPBS tier is wired and disabled by default; enabling it with `AUDIO_AI_ENABLE_VROOLI=true` + valid LPBS endpoint passes integration tests once `execute/lpbs-audio-gateway-endpoints` lands.
- [ ] Canonical voice catalog defined; every registered TTS adapter declares a mapping for each canonical voice; `voice_overrides` tests pass.
- [ ] Usage reporting is async, idempotent by `OperationID`, and redacts credentials in logs.
- [ ] All research verification paths (F1–F10) have corresponding automated tests.
