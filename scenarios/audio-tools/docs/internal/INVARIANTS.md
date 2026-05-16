# INVARIANTS

Non-negotiable contracts. Changing any of these is an RFC-level decision, not a refactor.

## Provider precedence (sttchain / ttschain / summarizechain)

1. Order is **fixed**: BYOK → Vrooli/LPBS → Local. Not caller-configurable.
2. `ErrInsufficientCredits` from the Vrooli tier **short-circuits** the chain. It does NOT fall through to Local. Credit exhaustion is a billing failure, not an availability failure.
3. `ErrUnknownBYOKProvider` and `ErrMissingBYOKProvider` terminate the chain without fallback. Silent provider selection is forbidden — the caller must name the BYOK provider explicitly.
4. Per-tier availability is cached: BYOK 5 minutes, Vrooli 30 seconds, Local probed each call.
5. **Encoded in**: `api/internal/ai/{sttchain,ttschain,summarizechain}/chain.go`.

## Canonical voice catalog (TTS)

1. Canonical IDs are exactly: `voice.feminine.warm`, `voice.feminine.neutral`, `voice.masculine.warm`, `voice.masculine.neutral`, `voice.neutral.default`. Adding a canonical ID requires updating every registered TTS adapter.
2. Every registered TTS adapter MUST declare a mapping for every canonical ID. Missing entries fail startup with a structured error.
3. Override resolution order: per-request `voice_overrides["tier:provider-id"]` → adapter's declared mapping → adapter default + warning log.
4. **Encoded in**: `api/internal/tts/voice_catalog.go` (planned). `ttschain.Request.VoiceOverrides` carries per-call overrides.

## TTS cache

1. The audio-tools TTS cache is **content-addressable**. Key = SHA256(`voice|speed|format|text-content`).
2. Event-keyed invalidation lives in **consumer glue** (e.g., web-console conversation orchestration), not in audio-tools.
3. `GetCache` accepts either `content_hash` (direct hit) OR `event_id + voice + speed + version` (consumer-mapped lookup).
4. **Encoded in**: `tts.proto.GetCacheRequest`, `api/internal/tts/cache.go`.

## Voice-session contract

1. A session has at most one in-flight assistant action at a time. `MarkInflight` records the eventID; `ClearInflight` is called by the transport when the action completes naturally.
2. Barge-in (`BargeIn(reason)`) MUST: clear inflight, invoke the registered `CancelHook` (so the transport can stop streaming TTS), AND emit a `BargeInCancel` SessionEvent to every observer. If no inflight action is recorded, BargeIn is a no-op.
3. Multiple observers per session are first-class. Slow observers drop events rather than blocking the publisher.
4. Close emits a `SessionClosed` event then closes every observer channel. Subsequent `Subscribe` returns `ErrSessionClosed`.
5. **Encoded in**: `api/internal/session/session.go` + `session_test.go`.

## Credential handling

1. BYOK keys travel in per-request metadata headers (`X-Audio-BYOK-Key`, `X-Audio-BYOK-Provider`). They are never stored in audio-tools state with the bare key — only the redacted fingerprint (`sk-***abcd`) goes into the `BYOKCredentialSummary` returned by `SettingsService`.
2. The provider chain logs MUST NOT include the raw key in any form. The credential-redacting logger mirrors BAS's pattern.
3. CLI BYOK entry is via interactive secret prompt or environment variable, never as a command-line argument.
4. **Encoded in**: `api/internal/byok/` (planned), `cli/domains/config/byok.go` (planned).

## Transport classification

1. The four `RESTReason*` constants in `templates/scenarios/react-vite/api/internal/module/module.go` are the only escape hatches from Connect-RPC for proto-owned operations. For audio-tools:
   - `multipart_upload`: `POST /api/v1/voice/transcribe`, `POST /api/v1/audio/transcode`.
   - `ops_probe`: `GET /health`.
   - `webhook_receiver`: none in P0.
   - `third_party_shape`: none in P0.
2. The browser-voice WebSocket is **not** a REST exception; it is a `TransportReason: websocket_transport` endpoint. (This constant must be added to the template — see R-PROTO.)
3. No new endpoints in web-console for audio. Consumers reach audio-tools directly via the integrations adapter.

## Greenfield discipline

1. No compatibility shims inside audio-tools. No legacy aliases. No `// removed` comments. No `_unused` named vars carried for symmetry.
2. Migration adapters in **consumers** (web-console) are allowed only if they have a named removal trigger.
3. Pure-function utilities that cross the scenario boundary are still owned by audio-tools and called via Connect-RPC (`tts.NormalizeForSpeech`, `tts.SplitParagraphs`). No shared `packages/audio-text-utils` package — wrap-not-use.
