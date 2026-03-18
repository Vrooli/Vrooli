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

Returns runtime diagnostics for auto-TTS, including hook registration state, backend routing, terminal acknowledgments, and browser playback state.

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
  "lastHookRouting": {
    "routed": true,
    "code": "tts_candidate_routed",
    "reason": "TTS candidate was routed to the mapped terminal session",
    "source": "claude_hook",
    "sessionId": "sess-123",
    "eventId": "abc123"
  },
  "lastHookRoutingAt": "2026-03-17T20:55:12Z",
  "lastHookAck": {
    "eventId": "abc123",
    "source": "claude_hook",
    "sessionId": "sess-123",
    "stage": "playback_succeeded",
    "backend": "browser"
  },
  "lastHookAckAt": "2026-03-17T20:55:13Z",
  "lastTailerRouting": {
    "routed": false,
    "code": "tts_target_missing",
    "reason": "No terminal session was available for TTS routing",
    "source": "codex_tailer",
    "sessionId": ""
  },
  "lastTailerRoutingAt": "2026-03-17T20:55:10Z",
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
  "session_id": "claude-internal-session-id"
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `last_assistant_message` | string | Yes | Full text of the AI assistant's response |
| `session_id` | string | No | Claude's internal session identifier from the hook input. Present for diagnostics, but web-console now routes primarily via `web_console_session_id`. |
| `web_console_session_id` | string | No | Explicit owning web-console terminal session ID injected by the Stop hook command from `WC_WEB_CONSOLE_SESSION_ID`. This is the primary Claude routing field. |
| `assistantResponse` | string | Legacy | Backward-compatible alias for `last_assistant_message` |

**Response** `200 OK`:
```json
{
  "status": "ok",
  "routed": true,
  "routing": {
    "routed": true,
    "code": "tts_candidate_routed",
    "reason": "TTS candidate was routed to the mapped terminal session",
    "source": "claude_hook",
    "sessionId": "sess-123",
    "eventId": "abc123"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `routed` | boolean | Whether a TTS candidate was routed to a terminal subscriber |
| `routing` | object | Structured backend routing result, including skip/failure reason |

**Errors**:
| Code | Category | When |
|------|----------|------|
| `unauthorized` | validation | Missing or invalid `X-Hook-Token` |
| `invalid_body` | validation | Request body is not valid JSON |

---

## WebSocket TTS Side-Channel

TTS candidates are delivered to clients via the existing terminal WebSocket at `/api/v1/sessions/{id}/ws`.

**Message format** (server → client):
```json
{
  "type": "tts_candidate",
  "eventId": "abc123",
  "source": "claude_hook",
  "data": "The answer is 42."
}
```

Candidates are **not** written to the terminal. The client correlates them against the rendered xterm buffer, speaks them if they match, and reports the outcome via:

```json
{
  "type": "tts_ack",
  "eventId": "abc123",
  "source": "claude_hook",
  "stage": "playback_succeeded",
  "backend": "browser"
}
```

### Delivery Pipeline

1. `web-console` asks the `claude-code` resource to reconcile the project-level `.claude/settings.json` Stop hook
2. Assistant response arrives via hook (`POST /hooks/stop`) or CodexTailer (rollout file polling)
3. `routeTTSCandidate()` validates: auto-TTS enabled → explicit terminal mapping exists → ANSI stripped → dedup check
4. `SendTTS()` fans the candidate out to WebSocket subscribers on that session (non-blocking; drops if channel full)
5. Client receives `tts_candidate` → correlates against the rendered terminal buffer → speaks via the runtime backend decision (`auto`, strict `kokoro`, or strict `browser`)
6. Client emits `tts_ack` stages so `/api/v1/tts/status` can distinguish routing success from browser-side rejection/playback failure

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
[CODE: api/tts_router.go] — Backend routing pipeline with dedup cache.
[CODE: api/tts_hook_handler.go] — Hook endpoint handler.
[CODE: api/tts_synthesize.go] — Synthesis endpoint and `TTSSynthesizer` interface.
[CODE: api/tts_voices.go] — Voice listing endpoint and `TTSVoiceLister` interface.
[CODE: api/tts_config.go] — Config endpoints and persistence.
