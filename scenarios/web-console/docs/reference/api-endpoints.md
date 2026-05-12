# API Reference

Every HTTP and WebSocket endpoint exposed by the web-console API. Routes are registered in [CODE: api/main.go] `setupRoutes` (around line 380). TTS and conversation-delivery endpoints are documented in depth in [TTS API Reference](./tts-api.md); only their routes are summarized here.

**Base URL**: `http://localhost:<api-port>` (port from `WEB_CONSOLE_API_PORT`, see [Configuration](./configuration.md)).

**Error envelope**: Structured JSON with `code`, `category`, `recovery`, and `retry` fields. See [Error Semantics](../internal/ERROR_SEMANTICS.md) and [CODE: api/errors.go].

## Health

| Method | Path | Handler |
|---|---|---|
| GET | `/health` | [CODE: api/main.go] `healthHandler` |
| GET | `/api/v1/health` | [CODE: api/main.go] `healthHandler` |

## Sessions

[CODE: api/session_handlers.go], [CODE: api/session_recovery_handler.go]

| Method | Path | Handler |
|---|---|---|
| POST | `/api/v1/sessions` | `handleCreateSession` |
| GET | `/api/v1/sessions` | `handleListSessions` |
| GET | `/api/v1/sessions/{id}` | `handleGetSession` |
| DELETE | `/api/v1/sessions/{id}` | `handleDeleteSession` |
| GET | `/api/v1/sessions/recoverable` | `handleListRecoverable` |
| DELETE | `/api/v1/sessions/recoverable/{id}` | `handleDismissRecoverable` |
| POST | `/api/v1/sessions/{id}/recover` | `handleRecoverSession` |
| GET | `/api/v1/sessions/{id}/policy` | `handleGetPolicy` |
| PUT | `/api/v1/sessions/{id}/policy` | `handleUpdatePolicy` |
| POST | `/api/v1/sessions/{id}/upload` | `handleUpload` |

Recovery contract and state machine: [Session Recovery guide](../guides/SESSION_RECOVERY.md).

## Terminal I/O (WebSocket)

| Method | Path | Handler |
|---|---|---|
| GET (Upgrade) | `/api/v1/sessions/{id}/ws` | [CODE: api/terminal_ws.go] `handleTerminalWS` |

Message protocol: see [Architecture — Terminal I/O](../concepts/ARCHITECTURE.md#terminal-io) and [Error Semantics — WebSocket](../internal/ERROR_SEMANTICS.md#websocket-error-protocol).

## Conversation & Files

[CODE: api/conversation_handlers.go], [CODE: api/conversation_summarize_handler.go], [CODE: api/file_reference_handlers.go]

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/sessions/{id}/conversation` | `handleGetConversationSession` |
| PUT | `/api/v1/sessions/{id}/conversation/cursor` | `handleUpdateConversationCursor` |
| POST | `/api/v1/sessions/{id}/conversation/{eventId}/summarize` | `handleSummarizeEvent` |
| POST | `/api/v1/sessions/{id}/files/resolve` | `handleResolveFileReference` |
| GET | `/api/v1/sessions/{id}/files/content` | `handleGetFileReferenceContent` |

User-facing contract for the messages feed: [Conversation Tracking guide](../guides/CONVERSATION_TRACKING.md).

## Workspace

[CODE: api/workspace_handlers.go]

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/workspace/layout` | `handleGetLayout` |
| PUT | `/api/v1/workspace/layout` | `handleSaveLayout` |
| PUT | `/api/v1/workspace/panes/{session_id}` | `handleUpdatePane` |
| DELETE | `/api/v1/workspace/panes/{session_id}` | `handleDeletePane` |
| POST | `/api/v1/workspace/groups` | `handleCreateGroup` |
| PUT | `/api/v1/workspace/groups/{id}` | `handleUpdateGroup` |
| DELETE | `/api/v1/workspace/groups/{id}` | `handleDeleteGroup` |

## Settings

[CODE: api/session_defaults_handler.go]

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/settings/session-defaults` | `handleGetSessionDefaults` |
| PUT | `/api/v1/settings/session-defaults` | `handleUpdateSessionDefaults` |

## AI

[CODE: api/ai_generate.go], [CODE: api/ai_config_handlers.go]

| Method | Path | Handler |
|---|---|---|
| POST | `/api/v1/ai/generate` | `handleAIGenerate` |
| POST | `/api/v1/ai/suggest` | `handleAISuggest` |
| GET | `/api/v1/ai/config` | `handleGetAIConfig` |
| PUT | `/api/v1/ai/config` | `handleUpdateAIConfig` |
| GET | `/api/v1/ai/health` | `handleGetAIHealth` |

Provider chain and retry behavior: [Architecture — AI Command Generation](../concepts/ARCHITECTURE.md#ai-command-generation).

## Shortcuts

[CODE: api/shortcut_handlers.go]

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/shortcuts` | `handleGetEffectiveShortcuts` |
| GET | `/api/v1/shortcuts/profiles` | `handleListShortcutProfiles` |
| PUT | `/api/v1/shortcuts/profiles` | `handleUpsertShortcutProfile` |
| DELETE | `/api/v1/shortcuts/profiles/{id}` | `handleDeleteShortcutProfile` |

## Voice

[CODE: api/voice_transcribe.go], [CODE: api/voice_stream_ws.go], [CODE: api/voice_config.go], [CODE: api/wakeword_handlers.go], [CODE: api/speaker_verification_handlers.go]

| Method | Path | Handler |
|---|---|---|
| POST | `/api/v1/voice/transcribe` | `handleVoiceTranscribe` |
| GET (Upgrade) | `/api/v1/voice/stream` | `handleVoiceStreamWS` |
| GET | `/api/v1/voice/config` | `handleGetVoiceConfig` |
| PUT | `/api/v1/voice/config` | `handleUpdateVoiceConfig` |
| GET | `/api/v1/voice/wakeword` | `handleGetWakeWordConfig` |
| PUT | `/api/v1/voice/wakeword` | `handleUpdateWakeWordTemplate` |
| DELETE | `/api/v1/voice/wakeword` | `handleDeleteWakeWordTemplate` |
| GET | `/api/v1/voice/speaker/config` | `handleGetSpeakerVerificationConfig` |
| PUT | `/api/v1/voice/speaker/config` | `handleUpdateSpeakerVerificationConfig` |
| GET | `/api/v1/voice/speaker/status` | `handleGetSpeakerVerificationStatus` |
| GET | `/api/v1/voice/speaker/profiles` | `handleGetSpeakerVerificationProfiles` |
| POST | `/api/v1/voice/speaker/enroll` | `handleEnrollSpeakerProfile` |
| DELETE | `/api/v1/voice/speaker/profile` | `handleClearSpeakerProfileBinding` |
| POST | `/api/v1/voice/speaker/profile/remove` | `handleRemoveSpeakerProfile` |
| POST | `/api/v1/voice/speaker/profile/delete` | `handleDeleteSpeakerProfile` |

Streaming pipeline and segment-final timing: [Temporal Flows — Voice](../internal/TEMPORAL-FLOWS.md#voice--persistent-mode--segment-finals).

## Capabilities

[CODE: api/capabilities_handler.go], [CODE: api/capabilities_checkers.go]

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/capabilities` | `handleCapabilities` |
| GET | `/api/v1/capabilities/liveness` | `handleCapabilitiesLiveness` |

## Hooks

[CODE: api/tts_hook_handler.go], [CODE: api/hook_prompt_submit_handler.go]

| Method | Path | Handler |
|---|---|---|
| POST | `/api/v1/hooks/stop` | `handleHookStop` |
| POST | `/api/v1/hooks/prompt-submit` | `handleHookPromptSubmit` |

## TTS

See [TTS API Reference](./tts-api.md) for full request/response schemas.

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/tts/config` | `handleGetTTSConfig` |
| PUT | `/api/v1/tts/config` | `handleUpdateTTSConfig` |
| GET | `/api/v1/tts/status` | `handleGetTTSStatus` |
| POST | `/api/v1/tts/events` | `handlePostTTSEvent` |
| GET | `/api/v1/tts/summarize/config` | `handleGetTTSSummarizeConfig` |
| PUT | `/api/v1/tts/summarize/config` | `handleUpdateTTSSummarizeConfig` |
| POST | `/api/v1/tts/synthesize` | `handleTTSSynthesize` |
| GET | `/api/v1/tts/cache/{eventId}` | `handleGetTTSCache` |
| GET | `/api/v1/tts/voices` | `handleTTSVoices` |

## Observability

[CODE: api/metrics.go], [CODE: api/events.go]

| Method | Path | Handler |
|---|---|---|
| GET | `/api/v1/metrics` | `handleMetrics` |
| GET | `/api/v1/events` | `handleEvents` |
