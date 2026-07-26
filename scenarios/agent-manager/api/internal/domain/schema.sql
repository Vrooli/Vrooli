-- Agent Manager SQLite Schema
-- SQLite variant for testing and lightweight deployments.

-- ============================================================================
-- Agent Profiles - Defines HOW an agent runs
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    profile_key TEXT NOT NULL UNIQUE,
    description TEXT,
    role_ref TEXT NOT NULL,
    max_turns INTEGER,
    timeout_ms INTEGER,
    effort TEXT NOT NULL DEFAULT '',
    allowed_tools TEXT DEFAULT '[]',
    denied_tools TEXT DEFAULT '[]',
    tool_restriction_policy TEXT NOT NULL DEFAULT 'enforced',
    skip_permission_prompt INTEGER DEFAULT 0,
    features TEXT DEFAULT '{}',
    extra_flags TEXT DEFAULT '{}',
    network_access TEXT NOT NULL DEFAULT 'localhost',
    sandbox_config TEXT DEFAULT '{}',
    allowed_paths TEXT DEFAULT '[]',
    denied_paths TEXT DEFAULT '[]',
    created_by TEXT,
    owner_scenario TEXT DEFAULT '',
    source_path TEXT DEFAULT '',
    source_hash TEXT DEFAULT '',
    last_applied_hash TEXT DEFAULT '',
    source_updated_at TEXT,
    local_override INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_name ON agent_profiles(name);
CREATE TRIGGER IF NOT EXISTS update_agent_profiles_updated_at
    AFTER UPDATE ON agent_profiles
    FOR EACH ROW
BEGIN
    UPDATE agent_profiles SET updated_at = datetime('now') WHERE id = NEW.id;
END;
