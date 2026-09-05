-- Web Console SQLite Schema
-- Workspace layout (pane ordering, tab groups) is persisted here for
-- cross-device sync. Session PTY state remains process-bound.

-- Terminal sessions (metadata only; PTY state is process-bound).
-- status state machine: live -> awaiting_recovery -> {live | dismissed}.
-- Only Recover() and the recovery endpoints transition status; everywhere
-- else the row is implicitly 'live'. agent_type+agent_session_id are
-- populated from the codex rollout session_meta event or the claude Stop
-- hook payload, and are sufficient input to reattach an agent on recovery.
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    backend TEXT NOT NULL DEFAULT 'standard',
	 shell TEXT NOT NULL DEFAULT '',
    cols INTEGER NOT NULL DEFAULT 80,
    rows INTEGER NOT NULL DEFAULT 24,
    policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never', 'preset', 'custom')),
    policy_duration TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    detached INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'live'
        CHECK(status IN ('live','awaiting_recovery','dismissed')),
    agent_type TEXT NOT NULL DEFAULT 'none'
        CHECK(agent_type IN ('none','codex','claude','opencode','grok')),
    launch_command TEXT NOT NULL DEFAULT '',
    agent_session_id TEXT NOT NULL DEFAULT '',
    cwd TEXT NOT NULL DEFAULT '',
    last_rollout_path TEXT NOT NULL DEFAULT '',
    last_activity_at TEXT NOT NULL DEFAULT '',
    orphaned_at TEXT NOT NULL DEFAULT '',
    recovered_into TEXT NOT NULL DEFAULT '',
    archived_at TEXT NOT NULL DEFAULT '',
    -- Provenance: who opened this session. Rows predating this column are
    -- backfilled to 'ui' by the ALTER TABLE migration in api/main.go, because
    -- every historical session was opened from the web UI.
    origin TEXT NOT NULL DEFAULT 'ui',
    owner TEXT NOT NULL DEFAULT '',
    display_label TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);
-- Indexes on the recovery-hardening columns are created by the migrations
-- block in api/main.go AFTER the ALTER TABLE statements add the columns.
-- They live there so an existing DB whose `sessions` table predates this
-- migration can be brought up to date without schema.sql failing on
-- CREATE INDEX against a not-yet-added column.

-- Conversation tracking state. This is the semantic message history used by
-- the messages pane, unread counters, and TTS cursor tracking.
CREATE TABLE IF NOT EXISTS conversation_sessions (
    session_id TEXT PRIMARY KEY,
    last_sequence INTEGER NOT NULL DEFAULT 0,
    last_seen_sequence INTEGER NOT NULL DEFAULT 0,
    last_listened_sequence INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS conversation_events (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    source TEXT NOT NULL,
    role TEXT NOT NULL CHECK(role IN ('assistant', 'user')),
    text TEXT NOT NULL,
    speech_paragraphs TEXT NOT NULL DEFAULT '[]',
    original_speech_paragraphs TEXT,
    summarized INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    delivery_state TEXT NOT NULL,
    tts_state TEXT NOT NULL,
    consumption_state TEXT NOT NULL,
    FOREIGN KEY (session_id) REFERENCES conversation_sessions(session_id) ON DELETE CASCADE,
    UNIQUE(session_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_conversation_events_session_sequence
    ON conversation_events(session_id, sequence);

-- Generic per-source ingestion cursors for newer agent transcript adapters
-- (Grok updates.jsonl tailing, OpenCode HTTP reconciliation). The cursor is an
-- opaque, source-defined string: a byte offset for append-only JSONL, or a
-- JSON high-water mark for full-history reconciliation.
CREATE TABLE IF NOT EXISTS agent_transcript_checkpoints (
    source TEXT NOT NULL,
    source_key TEXT NOT NULL,
    web_console_session_id TEXT NOT NULL,
    cursor TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    PRIMARY KEY (source, source_key)
);

CREATE INDEX IF NOT EXISTS idx_agent_transcript_checkpoints_session
    ON agent_transcript_checkpoints(web_console_session_id);

-- Shortcut profiles with scope hierarchy
CREATE TABLE IF NOT EXISTS shortcut_profiles (
    id TEXT PRIMARY KEY,
    scope TEXT NOT NULL DEFAULT 'service' CHECK(scope IN ('service', 'workspace', 'parent')),
    name TEXT NOT NULL,
    shortcuts TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_shortcut_profiles_scope ON shortcut_profiles(scope);

-- Tab groups for organizing terminal panes
CREATE TABLE IF NOT EXISTS tab_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT 'Group',
    color TEXT NOT NULL DEFAULT '#3b82f6',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_collapsed INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_tab_groups_sort ON tab_groups(sort_order);

-- Workspace pane metadata (ordering, appearance, group membership)
-- session_id references sessions but without FK constraint since sessions
-- are managed in-memory by SessionManager.
CREATE TABLE IF NOT EXISTS workspace_panes (
    session_id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT 'terminal',
    header_color TEXT NOT NULL DEFAULT 'transparent',
    theme_id TEXT NOT NULL DEFAULT 'default',
    font_size INTEGER NOT NULL DEFAULT 14,
    sort_order INTEGER NOT NULL DEFAULT 0,
    group_id TEXT REFERENCES tab_groups(id) ON DELETE SET NULL,
    is_active INTEGER NOT NULL DEFAULT 0,
    supports_messages_view INTEGER NOT NULL DEFAULT 0,
    -- User-set "come back to this" flag. Independent of the conversation read
    -- cursor, which only moves forward and exists only for message-capable
    -- sessions. See applyColumnMigrations for the matching ADD COLUMN.
    manually_unread INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_workspace_panes_sort ON workspace_panes(sort_order);
CREATE INDEX IF NOT EXISTS idx_workspace_panes_group ON workspace_panes(group_id);

-- Roles: named positions inside a group.
--
-- session_id is NULL while the role is WAITING: the role holds a command and
-- no process. That is the whole point of the table — a group can express a
-- position that has not started yet, which is what makes auto-closing an
-- empty group safe (a group with a waiting role is not finished).
--
-- Roles are NOT a replacement for workspace_panes. A pane is the runtime
-- projection of a live session and is keyed BY session_id; a role is the
-- durable identity inside a group. They are joined by session_id, and a pane
-- whose session appears in no role is an ordinary hand-grouped session.
--
-- ON DELETE CASCADE differs deliberately from workspace_panes.group_id, which
-- is ON DELETE SET NULL. A pane survives its group because it owns a live
-- session; a role has no meaning outside its group, so it goes with it.
CREATE TABLE IF NOT EXISTS workspace_roles (
    id TEXT PRIMARY KEY,
    group_id TEXT NOT NULL REFERENCES tab_groups(id) ON DELETE CASCADE,
    label TEXT NOT NULL DEFAULT 'Role',
    command TEXT NOT NULL DEFAULT '',
    working_dir TEXT NOT NULL DEFAULT '',
    incoming_prompt TEXT NOT NULL DEFAULT '',
    backend TEXT NOT NULL DEFAULT '',
    target_id TEXT NOT NULL DEFAULT '',
    session_id TEXT,
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_workspace_roles_group ON workspace_roles(group_id, sort_order);

-- One session can back at most one role. The index is PARTIAL so any number
-- of roles may wait at once (NULL session_id), while a running session can
-- never be claimed by two roles.
CREATE UNIQUE INDEX IF NOT EXISTS idx_workspace_roles_session
    ON workspace_roles(session_id) WHERE session_id IS NOT NULL;

-- Group templates: a saved role list. `roles` is a JSON array, mirroring
-- shortcut_profiles.shortcuts, so this scenario has ONE storage idiom for a
-- configuration row that owns an ordered child list.
--
-- There is deliberately no privileged-row column. A seeded example is an ordinary
-- row written through the public upsert path and is deletable like any other.
CREATE TABLE IF NOT EXISTS group_templates (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '',
    roles TEXT NOT NULL DEFAULT '[]',
    use_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

-- Capture rules: decide when a handoff is SUGGESTED. A rule never sends
-- anything, which is why an operator-authored pattern is safe to ship.
CREATE TABLE IF NOT EXISTS handoff_rules (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    source TEXT NOT NULL DEFAULT 'file_path'
        CHECK(source IN ('file_path', 'message_text')),
    pattern TEXT NOT NULL DEFAULT '',
    surfaces TEXT NOT NULL DEFAULT '[]',
    sort_order INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_handoff_rules_sort ON handoff_rules(sort_order);

-- AI provider configuration
CREATE TABLE IF NOT EXISTS ai_provider_configs (
    name TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 1,
    timeout_sec INTEGER NOT NULL DEFAULT 30,
    max_retries INTEGER NOT NULL DEFAULT 0
);
