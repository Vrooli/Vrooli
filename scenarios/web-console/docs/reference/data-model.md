# Data Model

The web-console persists workspace and conversation state in a single-file SQLite database. Active PTY/agent state stays in-memory inside `SessionManager`. This document is the index from concept to storage location.

## Storage layers

| Layer | Lifetime | Source of truth |
|---|---|---|
| In-memory `SessionManager` | Process | [CODE: api/session.go] `SessionManager` (around `type SessionManager struct`) |
| SQLite (workspace metadata, conversation, AI config) | Persistent (cross-device sync) | [CODE: api/internal/<domain>/schema.sql] |
| On-disk rollouts / shell history | Persistent (filesystem) | Codex rollout files, claude-code session storage |

Storage posture and migration roadmap: [STORAGE_AUDIT](../internal/STORAGE_AUDIT.md).

## In-memory entities

### `Session`
[CODE: api/session.go] (`type Session struct`)

Per-PTY state owned by `SessionManager`. Holds the PTY file descriptor, output broadcast channels, expiration policy, agent metadata, and `exitCh`. Lifecycle and concurrency invariants: [Architecture — Session Lifecycle](../concepts/ARCHITECTURE.md#data-flow), [SEAMS](../internal/SEAMS.md#3-domain--session-lifecycle), [INVARIANTS](../internal/INVARIANTS.md).

A subset of session metadata is mirrored into the persistent `sessions` table so that orphaned PTYs can be recovered after a restart; see [Session Recovery](../guides/SESSION_RECOVERY.md).

## Persistent tables

[CODE: api/internal/<domain>/schema.sql]

### `sessions`
Persistent session metadata for live-session continuity, deliberate archive, and crash recovery. PTY state is **not** persisted. The `status` column enforces the process-recovery state machine: `live → awaiting_recovery → {live | dismissed}`. The independent `archived_at` timestamp records the operator archive lifecycle without widening that status constraint. A non-empty `archived_at` means the operator closed the pane non-destructively; archive keeps the row, pane identity, and conversation. Reopen creates a replacement live session and records its ID in `recovered_into`. Archive listings follow `recovered_into` to the newest row and show one entry per lineage.

Key columns: `id`, `backend`, `shell`, `cols`, `rows`, `policy_mode`, `policy_duration`, `agent_type` (`none|codex|claude|opencode|grok`), `launch_command`, `agent_session_id`, `cwd`, `last_rollout_path`, `last_activity_at`, `orphaned_at`, `archived_at`, `recovered_into`, `detached`, `status`, `origin`, `owner`, `display_label`. For `opencode`/`grok`, `agent_session_id` is captured by the OpenCode watcher / Grok tailer respectively and is required for recovery.

Provenance columns: `origin` (`ui|programmatic|remote`) records who opened the session so the sidebar can separate human-opened tabs from agent- or remote-launched ones; `owner` is a free-form provenance tag (e.g. `agent-manager`); `display_label` is the human-facing sidebar label. All three are `TEXT NOT NULL` with defaults (`origin` defaults to `'ui'`, `owner`/`display_label` to `''`). The `ALTER TABLE sessions ADD COLUMN origin ... DEFAULT 'ui'` migration backfills every pre-existing row to `origin='ui'`, since all historical sessions were opened from the web UI. `idx_sessions_origin` indexes `origin`.

### `conversation_sessions`
One row per session that has a semantic message history. Tracks `last_sequence` (highest event sequence emitted), `last_seen_sequence` (cursor for unread badge), and `last_listened_sequence` (cursor for TTS replay).

### `conversation_events`
Append-only event log scoped to a `conversation_sessions.session_id`. Each row is one assistant or user turn with `text`, `speech_paragraphs`, optional `original_speech_paragraphs` (pre-summarization), per-event `delivery_state`, `tts_state`, `consumption_state`, and a monotonic `sequence`. Uniqueness is `(session_id, sequence)`. Source-of-truth for the messages pane and TTS replay; see [Conversation Tracking guide](../guides/CONVERSATION_TRACKING.md).

The `conversation_events_fts` FTS5 virtual table indexes `text` for cross-session archive search. Insert, update, and delete triggers keep it synchronized with `conversation_events`; startup reconciliation backfills missing index rows. Archive search joins through `sessions`, excludes live lineages, and returns message hits rather than scanning every transcript with `LIKE`.

## Archive retention

Archive retention is disabled by default: `WC_ARCHIVE_MESSAGELESS_AGE_DAYS=0`, `WC_ARCHIVE_AGENT_HOME_AGE_DAYS=0`, and `WC_ARCHIVE_MAX_BYTES=0` mean no age limit and no size ceiling. `PruneArchive` is a dry run unless the caller explicitly applies it. Candidate selection requires a non-empty `archived_at`; legacy dismissed rows and live rows are measured but are never prune candidates.

When configured, retention removes session-owned agent history before transcript data. This changes a `Reopenable` entry to `Read-only` while keeping its conversation searchable. A transcript row is eligible for deletion only when it has no messages and is older than `WC_ARCHIVE_MESSAGELESS_AGE_DAYS`.

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

Schema bootstrap runs `api/internal/<domain>/schema.sql` once at startup. Additive migrations live inline in [CODE: api/main.go] (search for the migrations block near where indexes on recovery columns are created). Recovery-hardening columns are added via `ALTER TABLE` before the corresponding indexes so an existing DB can be brought up to date without `schema.sql` failing.

`migrateSessionsAgentTypeConstraint` ([CODE: api/main.go]) relaxes the `sessions.agent_type` CHECK constraint to admit `opencode`/`grok`. SQLite cannot `ALTER` a CHECK in place, so it performs the canonical table-rebuild (rename → recreate with the new constraint → explicit-column copy → drop), guarded so it is a no-op on fresh DBs and idempotent on re-run.

## Filesystem-backed state (not in this DB)

| Concept | Location | Why not in SQLite |
|---|---|---|
| Codex rollouts | Codex-managed files referenced by `sessions.last_rollout_path` + `codex_rollout_checkpoints.path` | Codex owns the format; we only checkpoint offsets |
| Grok transcripts | grok-managed `updates.jsonl` under a per-session `GROK_HOME`, referenced by `sessions.last_rollout_path` + `agent_transcript_checkpoints` | grok owns the ACP format; we only checkpoint offsets |
| OpenCode sessions | OpenCode's global storage, accessed via the `opencode serve` HTTP API; only `agent_transcript_checkpoints` cursors persist | OpenCode owns the store; the HTTP API is the runtime contract (not SQLite) |
| Wake-word template binary | Voice config dir | Binary blob; managed via `/api/v1/voice/wakeword` |
| Speaker verification profiles | Speaker-verification resource | Owned by the resource; web-console references by id only |
