# TTS and Conversation API Reference

Auto-TTS is now driven by first-class per-session conversation events. This document covers:

1. TTS configuration and synthesis endpoints
2. Conversation history/cursor endpoints used by messages view, unread counts, and replay
3. WebSocket conversation-event delivery and acknowledgment

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
    "appended": true,
    "code": "conversation_event_appended",
    "reason": "Conversation event was appended to the mapped terminal session",
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
    "appended": false,
    "code": "conversation_target_missing",
    "reason": "No terminal session was available for conversation delivery",
    "source": "codex_tailer",
    "sessionId": ""
  },
  "lastTailerRoutingAt": "2026-03-17T20:55:10Z",
  "kokoroCapability": "available",
  "kokoroCapabilityLabel": "resource is healthy"
}
```

This endpoint is intended for settings/diagnostics UI rather than long-term persistence. It reports the latest conversation-ingestion and playback snapshots, not a durable event log.

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
    "appended": true,
    "code": "conversation_event_appended",
    "reason": "Conversation event was appended to the mapped terminal session",
    "source": "claude_hook",
    "sessionId": "sess-123",
    "eventId": "abc123"
  }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `routed` | boolean | Whether the assistant response was appended as a conversation event |
| `routing` | object | Structured append result, including skip/failure reason |

**Errors**:
| Code | Category | When |
|------|----------|------|
| `unauthorized` | validation | Missing or invalid `X-Hook-Token` |
| `invalid_body` | validation | Request body is not valid JSON |

---

### POST /api/v1/hooks/notification

Receives Claude Code `Notification` hook events. `web-console hooks` registers these Claude hooks against the running API: `web-console-tts` (Stop, which also drives auto-TTS), `web-console-prompt` (UserPromptSubmit), and `web-console-activity` (Notification). A notification whose message says Claude needs permission or approval, or is waiting for input, marks the session waiting in its activity ([CODE: api/hook_notification_handler.go]); it never starts playback.

## Conversation Endpoints

### GET /api/v1/sessions/{id}/conversation

Returns the semantic conversation history for one terminal session.

**Response** `200 OK`:
```json
{
  "sessionId": "sess-123",
  "events": [
    {
      "id": "evt-1",
      "sessionId": "sess-123",
      "source": "claude_hook",
      "role": "assistant",
      "text": "The answer is 42.",
      "createdAt": "2026-03-17T21:14:00Z",
      "sequence": 1,
      "deliveryState": "seen",
      "ttsState": "played",
      "consumptionState": "listened"
    }
  ],
  "cursor": {
    "lastSeenSequence": 1,
    "lastListenedSequence": 1
  }
}
```

### PUT /api/v1/sessions/{id}/conversation/cursor

Updates the optimistic conversation cursors for one session.

**Request body**:
```json
{
  "lastSeenSequence": 3,
  "lastListenedSequence": 2
}
```

**Response** `200 OK`:
```json
{
  "lastSeenSequence": 3,
  "lastListenedSequence": 2
}
```

Cursor semantics:
- `lastSeenSequence`: highest event the UI has surfaced to the user
- `lastListenedSequence`: highest assistant event whose TTS playback completed successfully

## WebSocket Conversation Side-Channel

Conversation events are delivered to clients via the existing terminal WebSocket at `/api/v1/sessions/{id}/ws`.

**Message format** (server → client):
```json
{
  "type": "conversation_event",
  "id": "abc123",
  "source": "claude_hook",
  "role": "assistant",
  "data": "The answer is 42.",
  "createdAt": "2026-03-17T21:14:00Z",
  "sequence": 4
}
```

Conversation events are semantic side-channel messages. They are not written to the terminal. The client stores them, can render them in messages mode, and may trigger TTS based on session/view state. The client reports progress via:

```json
{
  "type": "conversation_event_ack",
  "eventId": "abc123",
  "source": "claude_hook",
  "stage": "playback_succeeded",
  "backend": "browser"
}
```

### Delivery Pipeline

1. Claude Stop hook or Codex rollout tailer extracts assistant text.
2. The API appends a `ConversationEvent` into the per-session in-memory conversation store.
3. The API fans the event out to WebSocket subscribers for that session.
4. The UI stores the event in its conversation store.
5. If the pane is active and auto-TTS is enabled, the browser synthesizes and plays the event text.
6. The browser acknowledges stages such as `received`, `seen`, `playback_started`, `playback_succeeded`, and `playback_failed`.
7. The API records those state transitions for diagnostics and cursor advancement.
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

## Playback in the UI

### Playback transport

Each terminal pane publishes its TTS provider's state — `currentTime`,
`duration`, `playbackRate`, `volume`, `isMuted`, `isPaused`, and
`capabilities` — to the transport store in
`ui/src/domains/tts-playback/transport.ts` whenever the provider reports a
change. Kokoro reports position from the audio element's `timeupdate` event
(about 4 Hz) and settings from pause, rate, volume, and mute calls. Browser
speech synthesis reports no position, so its duration stays unknown and the
scrub is disabled.

Nothing in the UI polls. `useTtsPlaybackController().subscribeTransport(cb)`
delivers the active pane's transport on each change and nothing on a timer.
Workspace subscribes only to the pause flag (`usePlaybackPaused`), so position
updates re-render the pill alone. Lock-screen seek controls read the position
when pressed.

### The playback pill

`ui/src/components/tts/PlaybackPill.tsx` is the one playback surface. It floats
over the pane, above the toolbar, and takes no layout height. It shows while
playback is loading, playing, or paused (so a paused message can be resumed,
with auto-TTS on or off) and hides when playback ends or is dismissed.

- Collapsed: play/pause, an equalizer while audio plays, "speaker · time", the
  elapsed and total time, close, and a progress hairline.
- Expanded (tap the pill): a header with the queue count, settings, and close;
  a scrub; previous, play/pause, and next; chips for speed, the summarize mode,
  the voice, and Jump to message. Settings opens the volume, mute, and speed
  controls in a dialog.
- Gestures (`usePillGestures.ts`): a tap under 8 px toggles; a drag of 48 px or
  more down dismisses; on the collapsed pill a sideways drag of 64 px or more
  plays the next (left) or previous (right) message, and a drag along the
  progress hairline seeks. A drag keeps the axis it started on.
- Close and swipe-down stop playback; there is no background playback. The
  toolbar's restore button (`tts-restore`) shows while a queue exists and the
  pill is hidden, and a message starting to speak brings the pill back.

