-- Web Console PostgreSQL Schema
-- Workspace layout (pane ordering, tab groups) is persisted here for
-- cross-device sync. Session PTY state remains process-bound.

-- Enable extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Session expiration policy mode
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'policy_mode') THEN
        CREATE TYPE policy_mode AS ENUM ('never', 'preset', 'custom');
    END IF;
END$$;

-- Shortcut profile scope
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'shortcut_scope') THEN
        CREATE TYPE shortcut_scope AS ENUM ('service', 'workspace', 'parent');
    END IF;
END$$;

-- Terminal sessions (metadata only; PTY state is process-bound)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shell VARCHAR(255) NOT NULL DEFAULT '/bin/bash',
    cols SMALLINT NOT NULL DEFAULT 80,
    rows SMALLINT NOT NULL DEFAULT 24,
    policy_mode policy_mode NOT NULL DEFAULT 'never',
    policy_duration INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);

-- Shortcut profiles with scope hierarchy
CREATE TABLE IF NOT EXISTS shortcut_profiles (
    id VARCHAR(255) PRIMARY KEY,
    scope shortcut_scope NOT NULL DEFAULT 'service',
    name VARCHAR(255) NOT NULL,
    shortcuts JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shortcut_profiles_scope ON shortcut_profiles(scope);

-- Tab groups for organizing terminal panes
CREATE TABLE IF NOT EXISTS tab_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL DEFAULT 'Group',
    color VARCHAR(32) NOT NULL DEFAULT '#3b82f6',
    sort_order SMALLINT NOT NULL DEFAULT 0,
    is_collapsed BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tab_groups_sort ON tab_groups(sort_order);

-- Workspace pane metadata (ordering, appearance, group membership)
-- session_id FK cascades: deleting a session auto-removes its pane metadata.
-- group_id FK sets null: deleting a group ungroups its panes.
CREATE TABLE IF NOT EXISTS workspace_panes (
    session_id UUID PRIMARY KEY REFERENCES sessions(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL DEFAULT 'terminal',
    header_color VARCHAR(32) NOT NULL DEFAULT 'transparent',
    theme_id VARCHAR(100) NOT NULL DEFAULT 'default',
    font_size SMALLINT NOT NULL DEFAULT 14,
    sort_order SMALLINT NOT NULL DEFAULT 0,
    group_id UUID REFERENCES tab_groups(id) ON DELETE SET NULL,
    is_active BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_workspace_panes_sort ON workspace_panes(sort_order);
CREATE INDEX IF NOT EXISTS idx_workspace_panes_group ON workspace_panes(group_id);

-- AI provider configuration
CREATE TABLE IF NOT EXISTS ai_provider_configs (
    name VARCHAR(100) PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority SMALLINT NOT NULL DEFAULT 1,
    timeout_sec SMALLINT NOT NULL DEFAULT 30,
    max_retries SMALLINT NOT NULL DEFAULT 0
);

-- AI provider health tracking (ephemeral, reset on restart)
CREATE TABLE IF NOT EXISTS ai_provider_health (
    name VARCHAR(100) PRIMARY KEY REFERENCES ai_provider_configs(name) ON DELETE CASCADE,
    available BOOLEAN NOT NULL DEFAULT false,
    last_check TIMESTAMP WITH TIME ZONE,
    last_latency_ms INTEGER,
    error_count BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0
);
