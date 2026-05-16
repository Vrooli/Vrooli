# Replay / Idempotency Invariants

This document records the idempotency and replay-safety status of all
state-mutating operations in the web-console scenario.

## State-Mutating Operations

### API Endpoints

| Operation | Endpoint | Idempotent? | Mechanism | Notes |
|-----------|----------|-------------|-----------|-------|
| Create session | `POST /sessions` | **Yes** (with key) | `X-Idempotency-Key` header → TTL cache (5 min) | Without key, each call creates a new session. With key, replays return cached response. |
| Delete session | `DELETE /sessions/{id}` | **Yes** | Returns 204 regardless | Events/metrics only fire on first (actual) deletion. Second call is a no-op 204. |
| List sessions | `GET /sessions` | N/A | Read-only | Safe to call any number of times. |
| Get session | `GET /sessions/{id}` | N/A | Read-only | Returns 404 if not found. |
| Update policy | `PUT /sessions/{id}/policy` | **Yes** | Value overwrite + change detection | SetPolicy is an overwrite. Events only emitted when mode/duration actually change. |
| Get policy | `GET /sessions/{id}/policy` | N/A | Read-only | |
| Upsert profile | `PUT /shortcuts/profiles` | **Yes** | ID-based upsert + content comparison | UpdatedAt only bumped when scope, name, or shortcuts actually differ. |
| Delete profile | `DELETE /shortcuts/profiles/{id}` | **Yes** | Returns 204 regardless | No-op if profile already deleted. |
| AI generate | `POST /ai/generate` | **No** | Calls external providers each time | Each call hits Ollama/OpenRouter, emits event, increments counter. Intentionally non-idempotent (LLM responses are non-deterministic). |
| Update AI config | `PUT /ai/config` | **Yes** | Value overwrite | Same config written twice produces same state. |

### Internal Operations

| Operation | Location | Idempotent? | Notes |
|-----------|----------|-------------|-------|
| Session broadcast | `session.go:broadcast()` | N/A | Output fan-out is append-only to subscriber channels. |
| Expiration sweep | `session_policy.go` | **Yes** | Re-running sweep on already-expired sessions is a no-op (session already deleted). |
| Offline buffer | `session.go:broadcast()` | Append-only | Buffer grows until cap. Not a mutation concern for replay. |

### UI Operations

| Operation | Location | Safe? | Mechanism |
|-----------|----------|-------|-----------|
| Launch session | `useSessionManager.ts:launchSession` | **Yes** | `createInFlight` ref prevents concurrent calls. |
| Remove pane | `useSessionManager.ts:removePane` | **Yes** | Removes from pane array (filter is idempotent). DELETE is idempotent server-side. |
| Refresh sessions | `SessionsPage.tsx:refresh` | **Yes** | Generation counter discards stale responses. |
| Save profile | `SettingsPage.tsx:handleSave` | **Yes** | Server upsert is idempotent. On error, refetches server state. |
| Delete profile | `SettingsPage.tsx:handleDelete` | **Yes** | Server delete is idempotent. On error, refetches server state. |
| TTS incoming event auto-play | `useTtsPlaybackController.ts:handleIncomingEvent` | **Yes** | Stable event id + current transport target prevent replay storms; `playbackIntent=paused/stopped` blocks automatic playback without mutating `lastListenedSequence`. |
| TTS stop/pause | `Workspace.tsx`, `TerminalPane.tsx`, `useTtsPlaybackController.ts` | **Yes** | User pause/stop persists local playback intent and stops/pauses provider state. It does not mark assistant events listened. |
| TTS natural completion | `useTtsPlaybackController.ts` | **Yes** | Listened cursor advances only when provider speaking transitions from playing to ended for the current event. Stale completion after stop/pause/new play is ignored by transport state/load id. |

## Idempotency Keys

| Key | Scope | TTL | Location |
|-----|-------|-----|----------|
| `X-Idempotency-Key` header | Session creation | 5 minutes | `session_handlers.go:idempotencyCache` |

The cache uses opportunistic eviction (triggered when size > 100 entries).

## Safe Retry Patterns

- **DELETE endpoints**: Always safe to retry. Return 204 regardless of whether the resource existed.
- **PUT endpoints (policy, config, profile)**: Safe to retry. Overwrites converge to the same state.
- **POST /sessions with idempotency key**: Safe to retry within TTL window (5 min). Returns cached response.
- **POST /sessions without key**: NOT safe to retry blindly. Creates a new session each time.
- **POST /ai/generate**: NOT safe to retry without user intent. Calls external APIs, emits events.
- **TTS incoming assistant event**: Safe to receive again; duplicate events are ignored by `useConversationStore.appendEvent`, and playback is controlled by persisted intent plus current target state.

## TTS Playback Intent Invariants

| Invariant | Enforcement | Location |
|-----------|-------------|----------|
| User pause blocks future automatic playback | Incoming assistant events check `playbackIntent` before speaking | `ui/src/domains/tts-playback/utils.ts`, `useTtsPlaybackController.ts` |
| Natural completion keeps continuous intent | Completion does not convert `continuous` to paused/stopped | `useTtsPlaybackController.ts` |
| Stop/pause never means listened | `TerminalPane.stopTts()` no longer advances `lastListenedSequence`; controller commits listened only on natural completion | `TerminalPane.tsx`, `useTtsPlaybackController.ts` |
| Playback controls are not dismissible | The TTS playback bar has no close/collapse state; visibility is derived from auto-TTS setting plus a valid active/replay target | `Workspace.tsx`, `AudioPlayerBar.tsx`, `store.ts` |
| Inactive pane messages do not speak | Incoming auto-play requires `activePaneId === sessionId` | `utils.ts` |
| Auto playback respects muted state | Provider `speakText` only force-unmutes manual playback; incoming auto playback preserves the current mute/start-muted setting | `TerminalPane.tsx`, `useTtsPlaybackController.ts` |

## Voice Input UI Invariants

| Invariant | Enforcement | Location |
|-----------|-------------|----------|
| Mic button presentation is voice-input only | TTS speaking state never changes the mic icon, title, color, or error visibility; playback state is shown by the TTS bar | `VoiceMicButton.tsx`, `AudioPlayerBar.tsx` |
| Starting voice input may stop active TTS, but only as an interaction side effect | `isTtsSpeaking`/`onTtsStop` are used before `onStart`; no presentation branch depends on TTS state | `VoiceMicButton.tsx`, `Workspace.tsx` |
| Audio level and VAD UI state share one sample source | `useVoiceInput` computes `audioLevel` and `voiceActivity` from the same audio-analysis tick after `vadTick` runs | `useVoiceInput.ts`, `voice/activity.ts` |
| Auto-stop countdown is VAD-derived | The circular ring is visible only for one-shot `watchingSilence` after the visual grace; persistent listening does not show the one-shot stop ring | `voice/activity.ts`, `VoiceMicButton.tsx` |
| Stop/cancel/error teardown clears voice activity | Teardown paths reset both `audioLevel` and `voiceActivity` to idle | `useVoiceInput.ts` |

## Unsafe Operations (Intentionally Non-Idempotent)

1. **AI generation** — Each call hits external LLM providers, which return non-deterministic results.
   Metrics and events intentionally track each API call, not unique prompts. Caching LLM responses
   would require prompt hashing and cache invalidation policy, which is out of scope.

2. **WebSocket stdin messages** — PTY input is inherently non-idempotent (typing "ls\n" twice
   runs the command twice). No sequence tracking is implemented because the WebSocket protocol
   provides ordered delivery within a single connection, and reconnection replays terminal
   output (via offline buffer) rather than replaying input.

## Speaker Verification Invariants

| Invariant | Enforcement | Location |
|-----------|-------------|----------|
| Accepted text must never come from an unverified segment when filter mode is enabled | `evaluateSpeakerVerification` gates both segment-final and final transcript emission; `!allowed` suppresses transcription entirely | `voice_stream_ws.go`, `voice_transcribe.go` |
| Speaker verification config requires `profileId` when enabled | `SpeakerVerificationConfig.Validate()` rejects `enabled=true` with empty `profileId` | `speaker_verification_config.go:47` |
| Threshold must be in [0, 1] | `Validate()` range check | `speaker_verification_config.go:34` |
| Mode must be one of: off, filter, advisory | `Validate()` enum check | `speaker_verification_config.go:37` |
| Reject behavior must be one of: drop, show-muted | `Validate()` enum check | `speaker_verification_config.go:42` |
| Fallback policy is explicit | `FallbackWithoutVerification` defaults to `false` — when the resource fails, transcripts are suppressed unless the user has opted in to fallback | `speaker_verification_config.go:29` |
| Config snapshot per session | Speaker verification config is read once at WebSocket session start; mid-session config changes take effect on the next recording | `voice_stream_ws.go:77` |

## Event Emission Guards

| Event | Guard | Location |
|-------|-------|----------|
| `session.deleted` | Only on actual deletion (not already-deleted) | `session_handlers.go:handleDeleteSession` |
| `session.policy_updated` | Only when mode or duration changes | `session_handlers.go:handleUpdatePolicy` |
| `session.created` | Always (each creation is a new session) | `session_handlers.go:handleCreateSession` |
| `session.connected/disconnected` | Always (each WS connect is a real event) | `terminal_ws.go` |
