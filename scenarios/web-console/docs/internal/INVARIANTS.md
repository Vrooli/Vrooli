# Replay / Idempotency Invariants

## Conversation continuity invariants

Session process state and evidence-retention state are separate concerns. A
process exit, WebSocket disconnect, API restart, or workspace-pane removal
MUST NOT delete a session row, conversation event, transcript checkpoint, or
provider history. The only normal path allowed to physically remove those
artifacts is an explicit permanent-delete operation after its receipt has been
committed. Runtime cleanup therefore marks an exited standard session archived
and leaves persistent-session metadata available for recovery.

The provider-neutral lifecycle model is implemented in
`api/internal/continuity/lifecycle.go`. Illegal transitions are rejected by the
pure transition table, and `session_lifecycle_receipts` stores one immutable
result per operation key. Raw provider transcripts remain read-only source
evidence; reconciliation and indexing may only add or repair Web Console
projections.

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
| Expiration sweep | `session_policy.go` | **Yes** | Re-running sweep on an already archived/expired session is a no-op; production expiry is routed through the continuity archive handler and remains receipt-backed. |
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

## Messages Scroll Invariants

The Messages list (`ui/src/components/MessagesPane.tsx`) uses one scroll model: a follow boolean and the prepend anchor. Design record: `docs/internal/MESSAGES-VIEW-PROJECTION-UX.md`.

| Invariant | Enforcement | Location |
|-----------|-------------|----------|
| Only user intent writes follow | The scroll handler sets `followRef` from `remaining <= 200` only when no programmatic scroll is in flight; `scrollToBottom` sets it true; explicit jumps to a message set it false | `MessagesPane.tsx` (`setFollow`, scroll listener) |
| A user gesture ends a programmatic scroll | `useReleaseOnElementInteraction` clears `programmaticScrollRef` on wheel, touch, pointer, or key input, so the user's scroll events decide follow | `MessagesPane.tsx`, `hooks/useKeyboardListeners.ts` |
| New content moves the viewport only while follow is true | One layout effect keyed on events and `totalSize` scrolls to the end once per change while following; otherwise it only counts appended events for the new-messages pill | `MessagesPane.tsx` (follow rule effect) |
| Exactly two anchors exist | The virtualizer's resize compensation (rows above the viewport) and the prepend anchor (older page inserted). Browser scroll anchoring is off (`overflow-anchor: none`) so no shift is corrected twice | `hooks/useVirtualList.ts`, `MessagesPane.tsx` |
| Resize compensation lands in the same commit as the moved rows, and only for rows entirely above the viewport | `useVirtualList` applies the above-viewport delta in a layout effect on the size version, never in the measurement frame; the row straddling the top edge keeps its top fixed; rendered rows re-read their offsets on every size version | `hooks/useVirtualList.ts` |
| Refresh never refetches history it has not loaded | A refresh asks for "what is newer" only for a hydrated session, and a paged window's start is not a gap, so mount, focus, and reconnect refreshes cannot merge older history ahead of the viewport | `hooks/useConversationSession.ts`, `stores/useConversationStore.ts` |
| No retries, timers, or settle loops position the list | Enforced by review and by `messages-scroll-follow.test.tsx` (one write per change) | `ui/src/__tests__/messages-scroll-follow.test.tsx` |

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
| Auto-stop countdown uses the same authority as auto-stop | The circular ring is derived by `decideAutoStopRing`, which shares the server/client precedence with `decideAutoStop` including stale-but-latched server timeouts; persistent listening does not show the one-shot stop ring | `voice/autoStopDecision.ts`, `VoiceMicButton.tsx` |
| Stop/cancel/error teardown clears voice activity | Teardown paths reset both `audioLevel` and `voiceActivity` to idle | `useVoiceInput.ts` |
| Every browser mic stream has exactly one observable owner | All `getUserMedia` audio streams are acquired/registered through the mic ownership registry under a named `MicOwner`; releasing a lease stops all tracks exactly once (idempotent) | `audio-integration/hooks/voice/micOwnership.ts` |
| Passive wake-word listening is visible and user-cancellable | `passiveListeningActive` drives `isPassive`; the mic control shows the passive presentation while a passive stream is live and never reports ordinary idle/off; tapping it calls `exitPassiveMode()` | `useVoiceCore.ts`, `VoiceMicButton.tsx` |
| Passive wake-word listening is not a `voiceState` | The old `"passive"` workflow state is retired. `voiceState` stays `idle` while passive wake-word listening is live; `passiveListeningActive` is the UI-facing passive truth and the registry is the hardware truth | `types.ts`, `useVoiceCore.ts`, `passiveArmDecision.ts` |
| Mount and visibility alone never acquire the microphone | Persisted `lowLatencyVoice` or `wakeWordEnabled` flags do not call `getUserMedia` on app open or tab-visible transition. Low-latency prewarm and passive wake-word arming require explicit mic-control intent (`prepareRecording` / start) | `useVoiceCore.ts`, `VoiceMicButton.tsx`, `useVoiceInput.test.ts` |
| A hidden tab / backgrounded PWA holds no mic stream | Page-lifecycle cleanup is driven by the pure `decideMicLifecycle` policy: a **standalone/PWA releases ALL leases on `visibilitychange` hidden** (incl. active recording — iOS keeps the OS mic indicator on otherwise), a desktop tab releases non-active leases and the controller stops the active recording; ALL leases release on `pagehide`/`freeze`; passive listening never arms while hidden | `micLifecyclePolicy.ts#decideMicLifecycle`, `micOwnership.ts#installMicLifecycleCleanup`, `useVoiceCore.ts`, `passiveArmDecision.ts` |
| Hardware mic truth is the registry, not `voiceState` | An idle/off UI with a live lease that is NOT an expected passive/prewarm/settings owner is an invariant violation; `useVoiceCore` subscribes to the registry, derives `staleLiveMicLease`, and self-heals by releasing the orphan (logging a structured violation) through `controller.recoverStaleLeases` | `micLifecyclePolicy.ts#selectStaleLeases`, `voiceCaptureController.ts`, `useVoiceCore.ts` |
| Provider replacement / cleanup is single-authority and atomic | Every provider replace/dispose/shutdown goes through `VoiceCaptureController`; replacement disposes the previous provider (releasing its lease) BEFORE installing the next; error/fallback/cancel/unmount paths cannot leave the UI idle while a provider still owns a live mic track; direct `providerRef.current = …` assignment in error paths is prohibited | `voiceCaptureController.ts`, `useVoiceCore.ts` |
| A stale live mic surfaces a user recovery affordance | When `staleLiveMicLease` is true while the UI is idle, the mic button shows a "release microphone" affordance; tapping it calls `releaseMicrophone`/`onReleaseMic`, never `onStart` | `VoiceMicButton.tsx`, `useVoiceCore.ts` |
| `MediaStreamTrack.stop()` is the only mic release signal | Lifecycle/cleanup paths stop tracks via lease release; clearing React state or dropping a reference is never relied on to release the OS microphone | `micOwnership.ts` |
| Production mic acquisition goes through the registry | A structural UI test fails if production `src` code calls `navigator.mediaDevices.getUserMedia` outside `audio-integration/hooks/voice/micOwnership.ts` | `audio-boundary.test.ts`, `micOwnership.ts` |

## Unsafe Operations (Intentionally Non-Idempotent)

1. **AI generation** — Each call hits external LLM providers, which return non-deterministic results.
   Metrics and events intentionally track each API call, not unique prompts. Caching LLM responses
   would require prompt hashing and cache invalidation policy, which is out of scope.

2. **WebSocket stdin messages** — PTY input is inherently non-idempotent (typing "ls\n" twice
   runs the command twice). Reliable stdin uses cumulative UTF-8 byte offsets and a per-session
   accepted prefix. Reconnect replays only the unaccepted suffix; control frames are best-effort
   and never enter the reliable stream. An offset below the released prefix is unreconcilable and
   is surfaced without replay.

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
