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

### `agent_transcript_checkpoints`
One checkpoint table for every transcript adapter. Primary key `(source,
source_key)`; columns `web_console_session_id`, `cursor`, `updated_at`. The
`cursor` is an opaque, source-defined string: for Codex and Grok it is a byte
offset (Grok advances only at turn-completion boundaries), while OpenCode uses
a JSON high-water mark (`{lastUserCreated,lastAssistantCompleted}`). The
startup migration copies legacy Codex rows with `source=codex_rollout` before
dropping the old table. Cleared per session on delete.

### `shortcut_profiles`
Saved shortcut bindings with scope hierarchy (`service` < `workspace` < `parent`). The `shortcuts` column is a JSON-encoded array. Effective bindings are computed by [CODE: api/shortcut_profiles.go] from this table; see [GLOSSARY — Shortcut Profile](../concepts/GLOSSARY.md#shortcut-profile).

The array's **order is meaningful**: it is the order the session launcher shows
agents in, and reordering the launcher grid rewrites this column.

Each entry carries `label`, `command`, an optional `description`, and an
optional `agent_id`. `agent_id` names one coding agent in the capability
catalogue (`claude`, `codex`, `opencode`, `grok`, `agy`) or is absent for a
plain operator command. It is stored rather than re-derived so that no consumer
has to pattern-match command text to work out which agent an entry launches — a
guess any operator-authored wrapper defeats. Rows written before the field
existed are backfilled on read by [CODE: api/shortcut_agent_identity.go], which
owns the single derivation.

### `tab_groups`
Workspace organization for terminal panes. Holds `name`, `color`, `sort_order`, `is_collapsed`. Referenced by `workspace_panes.group_id` with `ON DELETE SET NULL`.

### `workspace_panes`
Per-session pane appearance and ordering. `session_id` is the primary key (one pane row per persistent session). No FK to `sessions` — pane metadata can outlive a process-bound session row, since `workspace_panes` is the cross-device source of truth for layout. Columns: `name`, `header_color`, `theme_id`, `font_size`, `sort_order`, `group_id`, `is_active`, `supports_messages_view`.

### `workspace_roles`
A named position inside a group. `session_id` is **NULL while the role is
waiting**: the role holds a command and no process, so it costs no PTY. That
distinction is what lets the console tell a finished group (close it) from a
half-started one (keep it), which is the whole safety argument for closing an
empty group without asking.

Columns: `group_id` (FK to `tab_groups`, **`ON DELETE CASCADE`**), `label`,
`command`, `working_dir`, `incoming_prompt`, `backend`, `target_id`,
`session_id`, `sort_order`.

`ON DELETE CASCADE` differs deliberately from `workspace_panes.group_id`, which
is `ON DELETE SET NULL`. A pane survives its group because it owns a live
session; a role has no meaning outside its group, so it goes with the group.

Two indexes: `(group_id, sort_order)` for ordered reads, and a **partial**
unique index on `session_id WHERE session_id IS NOT NULL`. The partial form is
load-bearing in both directions — any number of roles may wait at once, while a
running session can never be claimed by two roles (which would deliver one
handoff twice).

**Role and pane are different things.** A role is the durable identity inside a
group; a pane is the runtime projection of a live session, keyed *by* session
id. They are joined by `session_id`:

| Row state | Renders as |
|---|---|
| Role with `session_id = NULL` | A dashed placeholder. No pane row exists. |
| Role with `session_id` set | Its pane, using the pane's colour and name. |
| Pane whose session is in no role | An ordinary ungrouped or hand-grouped session. |

**Roles are optional.** Dragging a session into a group, or assigning it from
the picker, creates no role. Every pre-roles grouping behaviour works unchanged
with this table empty.

Session recovery mints a new session id and re-keys the pane through
`ReassignPane`. `ReassignRoleSession` performs the matching move for roles, so a
recovered session keeps its role rather than leaving it pointing at an id that
no longer exists.

### `group_templates`
A saved list of role definitions. Creating a group from one creates the group
and its roles in a single action. `roles` is a **JSON array**, mirroring
`shortcut_profiles.shortcuts`, so this scenario has one storage idiom for a
configuration row that owns an ordered child list. The tradeoff is that a role
is not independently queryable; nothing needs that today.

Columns: `name`, `color`, `roles` (JSON), `use_count`.

Each role in the array carries a `start_mode` of `eager` or `waiting`, validated
server-side. Only an `eager` role starts a process when the group is created.

There is deliberately **no built-in marker column**. A shipped example is an
ordinary row written through the same `UpsertTemplate` call the UI uses, and is
deletable like any other. The seeder writes only into an empty table, so a
deleted example does not come back on the next boot.

### `handoff_rules`
A named pattern that decides when the console **offers** a handoff. A rule never
sends anything: a match produces a suggestion the operator can dismiss, and
pressing it opens the same composer a button opens. That is the safety property
that makes operator-authored patterns shippable.

Columns: `name`, `enabled`, `source` (`CHECK` constrained to `file_path` or
`message_text`), `pattern`, `surfaces` (JSON), `sort_order`.

Like templates, there is no built-in marker and the example is deletable.

### `ai_provider_configs`
Per-provider knobs consumed by the AI generation chain ([CODE: api/ai_generate.go]). Key: provider `name`. Columns: `enabled`, `priority`, `timeout_sec`, `max_retries`. Provider chain ordering and failover: [Architecture — AI Command Generation](../concepts/ARCHITECTURE.md#ai-command-generation).

## Migrations

Schema bootstrap runs `api/internal/<domain>/schema.sql` once at startup. Additive migrations live inline in [CODE: api/main.go] (search for the migrations block near where indexes on recovery columns are created). Recovery-hardening columns are added via `ALTER TABLE` before the corresponding indexes so an existing DB can be brought up to date without `schema.sql` failing.

`migrateSessionsAgentTypeConstraint` ([CODE: api/main.go]) relaxes the `sessions.agent_type` CHECK constraint to admit `opencode`/`grok`. SQLite cannot `ALTER` a CHECK in place, so it performs the canonical table-rebuild (rename → recreate with the new constraint → explicit-column copy → drop), guarded so it is a no-op on fresh DBs and idempotent on re-run.

## Filesystem-backed state (not in this DB)

| Concept | Location | Why not in SQLite |
|---|---|---|
| Codex rollouts | Codex-managed files referenced by `sessions.last_rollout_path` + `agent_transcript_checkpoints` | Codex owns the format; we only checkpoint offsets |
| Grok transcripts | grok-managed `updates.jsonl` under a per-session `GROK_HOME`, referenced by `sessions.last_rollout_path` + `agent_transcript_checkpoints` | grok owns the ACP format; we only checkpoint offsets |
| OpenCode sessions | OpenCode's global storage, accessed via the `opencode serve` HTTP API; only `agent_transcript_checkpoints` cursors persist | OpenCode owns the store; the HTTP API is the runtime contract (not SQLite) |
| Wake-word template binary | Voice config dir | Binary blob; managed via `/api/v1/voice/wakeword` |
| Speaker verification profiles | Speaker-verification resource | Owned by the resource; web-console references by id only |
