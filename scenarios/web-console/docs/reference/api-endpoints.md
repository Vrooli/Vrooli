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

### Archive and recovery RPCs

These Connect-RPC methods are defined by `SessionsService` and `ConversationService`:

| Service method | Behavior |
|---|---|
| `SessionsService.Archive` | Sets `archived_at`, stops the session process, and preserves the row, pane metadata, and conversation. |
| `SessionsService.Unarchive` | Clears `archived_at` during the close undo window. It does not create a replacement session. |
| `SessionsService.ListArchived` | Returns collapsed archive lineages, restore state, storage context, and the `awaiting_recovery` marker. |
| `SessionsService.Reopen` | Reuses the recovery workflow for a deliberate archive. `X-Idempotency-Key` makes ambiguous retries replay-safe. |
| `SessionsService.GetArchiveRetention` | Returns retention policy plus measured transcript and agent-history bytes. |
| `SessionsService.PruneArchive` | Reports a dry run by default. `apply=true` executes only the returned eligible actions. |
| `ConversationService.SearchArchived` | Uses the FTS5 index to return message hits across archived lineages. |
| `ConversationService.GetRange` | Returns a bounded conversation range for the read-only archive reader and export. |

`SessionsService.Delete` remains the explicit permanent-delete operation. UI callers expose it only through a destructive confirmation.

**Create provenance, target, and launch fields.** `CreateRequest` (proto `web-console/v1/sessions`) carries provenance, target selection, working directory, and launch intent alongside the shell/backend fields:

- `origin` (`SessionOrigin`): who opened the session. The taxonomy is `SESSION_ORIGIN_UI` (human-opened browser tab), `SESSION_ORIGIN_PROGRAMMATIC` (agent/CLI caller), and `SESSION_ORIGIN_REMOTE`. `SESSION_ORIGIN_UNSPECIFIED` is **normalized to `SESSION_ORIGIN_PROGRAMMATIC`** on create — every first-party UI client sets `UI` explicitly, so an origin-less create can only have come from a programmatic caller. Persisted to `sessions.origin`.
- `owner`: free-form provenance tag (e.g. `agent-manager`). Persisted to `sessions.owner`.
- `display_label`: human-facing sidebar label. Persisted to `sessions.display_label`.
- `target_id`: target catalog ID. Empty or `local` creates on this Web Console host; a `bridge-node:<node-id>` ID routes creation through the server-owned Bridge adapter. Clients must discover IDs from `TargetCatalogService.List` rather than constructing Bridge URLs or sending credentials.
- `working_dir`: requested working directory on the selected target. The server validates and applies it at session creation; it is never used to select a target.
- `launch_command`: command staged in the new session. By default it is only staged; set `execute_launch_command=true` to have the server-side launch path paste it into the fresh session's stdin. Local sessions receive it immediately after create. Remote sessions receive it exactly once when the first terminal WebSocket attaches, because the Bridge PTY transport is established at attachment time. `execute_launch_command` is a create-time intent and is not persisted.
- `X-Idempotency-Key`: optional stable retry key for Create. Replaying the same request returns the original session, while reusing the key for different create inputs returns an explicit conflict instead of allocating a second session.

Remote sessions are intentionally honest about durability: they report `survives_restart=false` because the current Bridge session registry is process-local. Browser reconnects preserve the Bridge sequence while the Web Console process remains alive; restart recovery and durable remote leases require a future Bridge-owned session lease contract.

## Target catalog

`TargetCatalogService` is the single discovery and readiness surface for terminal locations. It returns a safe projection only: IDs, labels, platform, online/last-seen state, readiness facts, dispatchability, failure rung, recovery action, and restart durability. Owner tokens, re-authentication proofs, Bridge URLs, and device-auth material never appear in the response.

| Connect method | Behavior |
|---|---|
| `TargetCatalogService.List` | Returns the local Web Console host plus registered remote nodes and an explicit catalog state: `READY`, `CONFIGURED_EMPTY`, `UNCONFIGURED`, or `REGISTRY_ERROR`. |
| `TargetCatalogService.Get` | Returns one safe target projection by catalog ID. |
| `TargetCatalogService.Doctor` | Returns the target projection plus a concise summary for operational surfaces. |

Unavailable nodes remain visible so an operator can distinguish offline, protocol-update, unconfigured, and registry failures. Creation is rejected with a typed Connect error until the target is dispatchable.

Rows that predate these columns are backfilled to `origin='ui'` by an additive `ALTER TABLE` migration; see [data-model](./data-model.md#sessions).

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

User-facing contract for the messages feed: [Conversation Tracking guide](../guides/CONVERSATION_TRACKING.md).

## File Preview

[CODE: api/handlers/file_preview], [CODE: api/file_preview_handlers.go]

| Method | Path | Handler |
|---|---|---|
| POST | `FilePreviewService/Resolve` (Connect-RPC) | `file_preview.connectHandler.Resolve` |
| POST | `FilePreviewService/GetTextContent` (Connect-RPC) | `file_preview.connectHandler.GetTextContent` |
| GET/HEAD | `/api/v1/sessions/{id}/file-previews/{previewId}/blob` | `Server.handleFilePreviewBlob` (REST exception, `ops_probe`) |

The blob route serves bytes for an opaque, session-bound `preview_id` with HTTP Range support; it is consumed directly by native `<img>/<video>/<audio>/<iframe>`. See `docs/concepts/ARCHITECTURE.md#file-preview` and `docs/internal/SEAMS.md`.

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
