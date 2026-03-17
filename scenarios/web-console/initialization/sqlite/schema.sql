-- Web Console SQLite Schema
-- Workspace layout (pane ordering, tab groups) is persisted here for
-- cross-device sync. Session PTY state remains process-bound.

-- Terminal sessions (metadata only; PTY state is process-bound)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    shell TEXT NOT NULL DEFAULT '/bin/bash',
    cols INTEGER NOT NULL DEFAULT 80,
    rows INTEGER NOT NULL DEFAULT 24,
    policy_mode TEXT NOT NULL DEFAULT 'never' CHECK(policy_mode IN ('never', 'preset', 'custom')),
    policy_duration TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);

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
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_workspace_panes_sort ON workspace_panes(sort_order);
CREATE INDEX IF NOT EXISTS idx_workspace_panes_group ON workspace_panes(group_id);

-- AI provider configuration
CREATE TABLE IF NOT EXISTS ai_provider_configs (
    name TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 1,
    priority INTEGER NOT NULL DEFAULT 1,
    timeout_sec INTEGER NOT NULL DEFAULT 30,
    max_retries INTEGER NOT NULL DEFAULT 0
);
