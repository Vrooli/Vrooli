# TTS API Reference

Text-to-Speech endpoints for automatic voice delivery of AI assistant responses.

## Endpoints

### GET /api/v1/tts/config

Returns the current TTS configuration.

**Response** `200 OK`:
```json
{
  "autoEnabled": true,
  "backend": "auto",
  "kokoroVoice": "af_heart",
  "kokoroSpeed": 1.0
}
```

| Field | Type | Description |
|-------|------|-------------|
| `autoEnabled` | boolean | Whether auto-TTS delivery is active |
| `backend` | string | Preferred backend: `"auto"`, `"kokoro"`, or `"browser"` |
| `kokoroVoice` | string | Kokoro voice ID (e.g., `"af_heart"`, `"bf_emma"`) |
| `kokoroSpeed` | number | Playback speed, 0.5–4.0 |

---

### PUT /api/v1/tts/config

Applies a partial update to TTS config, persists to disk, and returns the updated config.

**Request body** — all fields optional (only provided fields are updated):
```json
{
  "autoEnabled": true,
  "backend": "auto",
  "kokoroVoice": "bf_emma",
  "kokoroSpeed": 1.2
}
```

**Response** `200 OK`: Full `TTSConfig` object (same shape as GET).

**Backend semantics**:
- `auto`: prefer Kokoro when available, otherwise use browser speech synthesis
- `kokoro`: strict Kokoro mode, no silent browser fallback during backend selection
- `browser`: strict browser speech synthesis

**Errors**:
| Code | Category | When |
|------|----------|------|
| `invalid_body` | validation | Request body is not valid JSON |

---

### GET /api/v1/tts/status

Returns runtime diagnostics for auto-TTS, including hook registration state and the most recent delivery result.

**Response** `200 OK`:
```json
{
  "config": {
    "autoEnabled": true,
    "backend": "auto",
    "kokoroVoice": "af_heart",
    "kokoroSpeed": 1.0
  },
  "hookRegistered": true,
  "hookCode": "hook_registered",
  "hookReason": "Claude Stop hook is registered",
  "hookSettingsPath": "/home/user/Vrooli/.claude/settings.json",
  "lastHookDelivery": {
    "delivered": true,
    "code": "tts_delivered",
    "reason": "TTS text was delivered to the terminal session",
    "source": "claude_hook",
    "sessionId": "sess-123",
    "usedTargetSession": true
  },
  "lastHookDeliveryAt": "2026-03-17T20:55:12Z",
  "lastTailerDelivery": {
    "delivered": false,
    "code": "tts_delivery_target_missing",
    "reason": "No active terminal session is available for TTS delivery",
    "source": "codex_tailer",
    "usedTargetSession": false
  },
  "lastTailerDeliveryAt": "2026-03-17T20:55:10Z",
  "kokoroCapability": "available",
  "kokoroCapabilityLabel": "resource is healthy"
}
```

This endpoint is intended for settings/diagnostics UI rather than long-term persistence.

`hookCode` is a stable machine-readable hook diagnostic:
- `hook_registered`
- `hook_missing_file`
- `hook_missing`
- `hook_stale`
- `hook_invalid_json`
- `hook_read_failed`

---

### POST /api/v1/tts/synthesize

Synthesizes speech from text via the Kokoro backend. Streams audio bytes back directly.

**Request body**:
```json
{
  "input": "Hello, world!",
  "voice": "af_heart",
  "response_format": "mp3",
  "speed": 1.0
}
```

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `input` | string | Yes | — | Text to synthesize (max 5000 chars) |
| `voice` | string | No | Config's `kokoroVoice` or `"af_heart"` | Kokoro voice ID |
| `response_format` | string | No | `"mp3"` | Audio format: `mp3`, `wav`, `opus`, `flac` |
| `speed` | number | No | `1.0` | Playback speed (clamped to 0–4.0) |

**Response** `200 OK`: Raw audio stream with `Content-Type` header matching the format (e.g., `audio/mpeg`).

**Errors**:
| Code | Category | When |
|------|----------|------|
| `tts_unavailable` | dependency | Kokoro is not running |
| `tts_input_required` | validation | Empty or missing `input` |
| `tts_input_too_long` | validation | Input exceeds 5000 characters |
| `tts_invalid_format` | validation | Unsupported `response_format` |
| `tts_synthesis_failed` | dependency | Kokoro returned an error |

**Example**:
```bash
curl -X POST http://localhost:4200/api/v1/tts/synthesize \
  -H 'Content-Type: application/json' \
  -d '{"input":"Hello world","voice":"af_heart","response_format":"mp3"}' \
  --output speech.mp3
```

---

### GET /api/v1/tts/voices

Returns available Kokoro TTS voices.

**Response** `200 OK`:
```json
[
  { "id": "af_heart", "name": "af_heart" },
  { "id": "bf_emma", "name": "bf_emma" }
]
```

**Errors**:
| Code | Category | When |
|------|----------|------|
| `tts_unavailable` | dependency | Kokoro is not running |
| `tts_voice_list_failed` | dependency | Kokoro returned an error |

---

### POST /api/v1/hooks/stop

Receives assistant response text from Claude Code's stop hook for TTS delivery.

**Authentication**: `X-Hook-Token` header with the server-generated hook token (stored in state and mirrored in the project-level Claude hook entry).

The canonical Claude project hook file is `.claude/settings.json` in the repository root. The `claude-code` resource owns reconciling this file; `web-console` declares the desired hook and delegates the write/heal operation to the resource seam.

**Request body**:
```json
{
  "hook_event_name": "Stop",
  "last_assistant_message": "The answer is 42.",
  "session_id": "optional-session-id"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `last_assistant_message` | string | Yes | Full text of the AI assistant's response |
| `session_id` | string | No | Target terminal session. When omitted, delivery falls back to the active pane. |
| `assistantResponse` | string | Legacy | Backward-compatible alias for `last_assistant_message` |
| `sessionId` | string | Legacy | Backward-compatible alias for `session_id` |

**Response** `200 OK`:
```json
{
  "status": "ok",
  "delivered": true,
  "delivery": {
    "delivered": true,
    "code": "tts_delivered",
    "reason": "TTS text was delivered to the terminal session",
    "source": "claude_hook",
    "sessionId": "sess-123",
    "usedTargetSession": true
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `delivered` | boolean | Whether text was sent to a TTS subscriber |
| `delivery` | object | Structured delivery result, including skip/failure reason |

**Errors**:
| Code | Category | When |
|------|----------|------|
| `unauthorized` | validation | Missing or invalid `X-Hook-Token` |
| `invalid_body` | validation | Request body is not valid JSON |

---

## WebSocket TTS Side-Channel

TTS text is delivered to clients via the existing terminal WebSocket at `/api/v1/sessions/{id}/ws`.

**Message format** (server → client):
```json
{
  "type": "tts",
  "data": "The answer is 42."
}
```

TTS messages are **not** written to the terminal — they are handled by the `onTTS` callback in `useTerminalSocket` and routed to the `useTextToSpeech` hook for audio playback.

### Delivery Pipeline

1. `web-console` asks the `claude-code` resource to reconcile the project-level `.claude/settings.json` Stop hook
2. Assistant response arrives via hook (`POST /hooks/stop`) or CodexTailer (rollout file polling)
3. `deliverTTS()` validates: auto-TTS enabled → target session (or active pane) exists → ANSI stripped → text found in output history → dedup check
4. `SendTTS()` fans out to all WebSocket subscribers on that session (non-blocking; drops if channel full)
5. Client receives `tts` message → splits into paragraphs → speaks via the runtime backend decision (`auto`, strict `kokoro`, or strict `browser`)

### Error Codes Reference

All error responses use the standard `ErrorResponse` shape:
```json
{
  "error": "Human-readable message",
  "code": "machine_readable_code",
  "category": "validation|resource_limit|dependency|internal",
  "recovery": "Suggested next step",
  "retry": false
}
```

[CODE: api/errors.go] — Error catalog with all codes, categories, and recovery hints.
[CODE: api/tts_deliver.go] — Delivery pipeline with dedup cache.
[CODE: api/tts_hook_handler.go] — Hook endpoint handler.
[CODE: api/tts_synthesize.go] — Synthesis endpoint and `TTSSynthesizer` interface.
[CODE: api/tts_voices.go] — Voice listing endpoint and `TTSVoiceLister` interface.
[CODE: api/tts_config.go] — Config endpoints and persistence.
