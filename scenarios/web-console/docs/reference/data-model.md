# Data Model

The web-console persists workspace and conversation state in a single-file SQLite database. Active PTY/agent state stays in-memory inside `SessionManager`. This document is the index from concept to storage location.

## Storage layers

| Layer | Lifetime | Source of truth |
|---|---|---|
| In-memory `SessionManager` | Process | [CODE: api/session.go] `SessionManager` (around `type SessionManager struct`) |
| SQLite (workspace metadata, conversation, AI config) | Persistent (cross-device sync) | [CODE: initialization/sqlite/schema.sql] |
| On-disk rollouts / shell history | Persistent (filesystem) | Codex rollout files, claude-code session storage |

Storage posture and migration roadmap: [STORAGE_AUDIT](../internal/STORAGE_AUDIT.md).

## In-memory entities

### `Session`
[CODE: api/session.go] (`type Session struct`)

Per-PTY state owned by `SessionManager`. Holds the PTY file descriptor, output broadcast channels, expiration policy, agent metadata, and `exitCh`. Lifecycle and concurrency invariants: [Architecture — Session Lifecycle](../concepts/ARCHITECTURE.md#data-flow), [SEAMS](../internal/SEAMS.md#3-domain--session-lifecycle), [INVARIANTS](../internal/INVARIANTS.md).

A subset of session metadata is mirrored into the persistent `sessions` table so that orphaned PTYs can be recovered after a restart; see [Session Recovery](../guides/SESSION_RECOVERY.md).

## Persistent tables

[CODE: initialization/sqlite/schema.sql]

### `sessions`
Mirror of session metadata for recovery only. PTY state is **not** persisted. The `status` column enforces the recovery state machine: `live → awaiting_recovery → {live | dismissed}`. Transitions are owned by `Recover()` and the recovery endpoints. `agent_type` + `agent_session_id` are populated from the codex `session_meta` event or the claude `Stop` hook payload and are sufficient to reattach an agent on recovery.

Key columns: `id`, `backend`, `shell`, `cols`, `rows`, `policy_mode`, `policy_duration`, `agent_type` (`none|codex|claude|opencode|grok`), `launch_command`, `agent_session_id`, `cwd`, `last_rollout_path`, `last_activity_at`, `orphaned_at`, `recovered_into`, `detached`, `status`. For `opencode`/`grok`, `agent_session_id` is captured by the OpenCode watcher / Grok tailer respectively and is required for recovery.

### `conversation_sessions`
One row per session that has a semantic message history. Tracks `last_sequence` (highest event sequence emitted), `last_seen_sequence` (cursor for unread badge), and `last_listened_sequence` (cursor for TTS replay).

### `conversation_events`
Append-only event log scoped to a `conversation_sessions.session_id`. Each row is one assistant or user turn with `text`, `speech_paragraphs`, optional `original_speech_paragraphs` (pre-summarization), per-event `delivery_state`, `tts_state`, `consumption_state`, and a monotonic `sequence`. Uniqueness is `(session_id, sequence)`. Source-of-truth for the messages pane and TTS replay; see [Conversation Tracking guide](../guides/CONVERSATION_TRACKING.md).

### `codex_rollout_checkpoints`
Per-rollout-file byte offset checkpoints. Lets the server backfill messages written while the UI was closed and resume after restart without re-reading old lines. Key: rollout `path`.

### `agent_transcript_checkpoints`
Generic per-source ingestion cursors for the newer transcript adapters (Grok tailer, OpenCode watcher). Primary key `(source, source_key)`; columns `web_console_session_id`, `cursor`, `updated_at`. The `cursor` is an opaque, source-defined string: for Grok (`source=grok_tailer`, `source_key`=absolute `updates.jsonl` path) it is a byte offset advanced only at turn-completion boundaries; for OpenCode (`source=opencode_api`, `source_key`=opencode session id) it is a JSON high-water mark (`{lastUserCreated,lastAssistantCompleted}`) used to make full-history reconciliation idempotent. This table is deliberately separate from `codex_rollout_checkpoints` — rewriting Codex's proven byte-offset history is higher risk than an additive table. Cleared per session on delete.

### `shortcut_profiles`
Saved shortcut bindings with scope hierarchy (`service` < `workspace` < `parent`). The `shortcuts` column is a JSON-encoded array. Effective bindings are computed by [CODE: api/shortcut_profiles.go] from this table; see [GLOSSARY — Shortcut Profile](../concepts/GLOSSARY.md#shortcut-profile).

### `tab_groups`
Workspace organization for terminal panes. Holds `name`, `color`, `sort_order`, `is_collapsed`. Referenced by `workspace_panes.group_id` with `ON DELETE SET NULL`.

### `workspace_panes`
Per-session pane appearance and ordering. `session_id` is the primary key (one pane row per persistent session). No FK to `sessions` — pane metadata can outlive a process-bound session row, since `workspace_panes` is the cross-device source of truth for layout. Columns: `name`, `header_color`, `theme_id`, `font_size`, `sort_order`, `group_id`, `is_active`, `supports_messages_view`.

### `ai_provider_configs`
Per-provider knobs consumed by the AI generation chain ([CODE: api/ai_generate.go]). Key: provider `name`. Columns: `enabled`, `priority`, `timeout_sec`, `max_retries`. Provider chain ordering and failover: [Architecture — AI Command Generation](../concepts/ARCHITECTURE.md#ai-command-generation).

## Migrations

Schema bootstrap runs `initialization/sqlite/schema.sql` once at startup. Additive migrations live inline in [CODE: api/main.go] (search for the migrations block near where indexes on recovery columns are created). Recovery-hardening columns are added via `ALTER TABLE` before the corresponding indexes so an existing DB can be brought up to date without `schema.sql` failing.

`migrateSessionsAgentTypeConstraint` ([CODE: api/main.go]) relaxes the `sessions.agent_type` CHECK constraint to admit `opencode`/`grok`. SQLite cannot `ALTER` a CHECK in place, so it performs the canonical table-rebuild (rename → recreate with the new constraint → explicit-column copy → drop), guarded so it is a no-op on fresh DBs and idempotent on re-run.

## Filesystem-backed state (not in this DB)

| Concept | Location | Why not in SQLite |
|---|---|---|
| Codex rollouts | Codex-managed files referenced by `sessions.last_rollout_path` + `codex_rollout_checkpoints.path` | Codex owns the format; we only checkpoint offsets |
| Grok transcripts | grok-managed `updates.jsonl` under a per-session `GROK_HOME`, referenced by `sessions.last_rollout_path` + `agent_transcript_checkpoints` | grok owns the ACP format; we only checkpoint offsets |
| OpenCode sessions | OpenCode's global storage, accessed via the `opencode serve` HTTP API; only `agent_transcript_checkpoints` cursors persist | OpenCode owns the store; the HTTP API is the runtime contract (not SQLite) |
| Wake-word template binary | Voice config dir | Binary blob; managed via `/api/v1/voice/wakeword` |
| Speaker verification profiles | Speaker-verification resource | Owned by the resource; web-console references by id only |
