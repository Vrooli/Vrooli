-- Agent Manager SQLite Schema
-- SQLite variant for testing and lightweight deployments.

-- ============================================================================
-- Agent Profiles - Defines HOW an agent runs
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    runner_type TEXT NOT NULL,
    model TEXT,
    max_turns INTEGER,
    timeout_ms INTEGER,
    allowed_tools TEXT DEFAULT '[]',
    denied_tools TEXT DEFAULT '[]',
    skip_permission_prompt INTEGER DEFAULT 0,
    requires_sandbox INTEGER DEFAULT 1,
    requires_approval INTEGER DEFAULT 1,
    allowed_paths TEXT DEFAULT '[]',
    denied_paths TEXT DEFAULT '[]',
    created_by TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_name ON agent_profiles(name);

-- ============================================================================
-- Tasks - Defines WHAT needs to be done
-- ============================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    scope_path TEXT NOT NULL,
    project_root TEXT,
    phase_prompt_ids TEXT DEFAULT '[]',
    context_attachments TEXT DEFAULT '[]',
    status TEXT DEFAULT 'queued',
    created_by TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

-- ============================================================================
-- Runs - Concrete execution attempts
-- ============================================================================
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent_profile_id TEXT REFERENCES agent_profiles(id) ON DELETE SET NULL,
    tag TEXT,
    sandbox_id TEXT,
    run_mode TEXT DEFAULT 'sandboxed',
    status TEXT DEFAULT 'pending',
    started_at TEXT,
    ended_at TEXT,
    phase TEXT DEFAULT 'queued',
    last_checkpoint_id TEXT,
    last_heartbeat TEXT,
    progress_percent INTEGER DEFAULT 0,
    idempotency_key TEXT UNIQUE,
    summary TEXT,
    error_msg TEXT,
    exit_code INTEGER,
    approval_state TEXT DEFAULT 'none',
    approved_by TEXT,
    approved_at TEXT,
    resolved_config TEXT,
    diff_path TEXT,
    log_path TEXT,
    changed_files INTEGER DEFAULT 0,
    total_size_bytes INTEGER DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_tag ON runs(tag);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);

-- ============================================================================
-- Run Events - Append-only event stream
-- ============================================================================
CREATE TABLE IF NOT EXISTS run_events (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sequence INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    timestamp TEXT DEFAULT (datetime('now')),
    data TEXT NOT NULL,
    UNIQUE(run_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events(run_id);
CREATE INDEX IF NOT EXISTS idx_run_events_run_sequence ON run_events(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_run_events_type ON run_events(run_id, event_type);

-- ============================================================================
-- Run Checkpoints - For resumption after interruption
-- ============================================================================
CREATE TABLE IF NOT EXISTS run_checkpoints (
    run_id TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
    phase TEXT NOT NULL,
    step_within_phase INTEGER DEFAULT 0,
    sandbox_id TEXT,
    work_dir TEXT,
    lock_id TEXT,
    last_event_sequence INTEGER DEFAULT 0,
    last_heartbeat TEXT DEFAULT (datetime('now')),
    retry_count INTEGER DEFAULT 0,
    saved_at TEXT DEFAULT (datetime('now')),
    metadata TEXT DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_run_checkpoints_heartbeat ON run_checkpoints(last_heartbeat);

-- ============================================================================
-- Idempotency Records - For replay-safe operations
-- ============================================================================
CREATE TABLE IF NOT EXISTS idempotency_records (
    key TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    entity_id TEXT,
    entity_type TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL,
    response TEXT
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_records(expires_at);
CREATE INDEX IF NOT EXISTS idx_idempotency_status ON idempotency_records(status);

-- ============================================================================
-- Policies - Rules governing execution
-- ============================================================================
CREATE TABLE IF NOT EXISTS policies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    scope_pattern TEXT,
    rules TEXT NOT NULL DEFAULT '{}',
    enabled INTEGER DEFAULT 1,
    created_by TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_policies_enabled ON policies(enabled);
CREATE INDEX IF NOT EXISTS idx_policies_priority ON policies(priority);

-- ============================================================================
-- Scope Locks - Concurrency control
-- ============================================================================
CREATE TABLE IF NOT EXISTS scope_locks (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    scope_path TEXT NOT NULL,
    project_root TEXT,
    acquired_at TEXT DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scope_locks_run_id ON scope_locks(run_id);
CREATE INDEX IF NOT EXISTS idx_scope_locks_scope ON scope_locks(scope_path, project_root);
CREATE INDEX IF NOT EXISTS idx_scope_locks_expires ON scope_locks(expires_at);

-- ============================================================================
-- Triggers for automatic updated_at (SQLite style)
-- ============================================================================
CREATE TRIGGER IF NOT EXISTS update_agent_profiles_updated_at
    AFTER UPDATE ON agent_profiles
    FOR EACH ROW
BEGIN
    UPDATE agent_profiles SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
    AFTER UPDATE ON tasks
    FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_runs_updated_at
    AFTER UPDATE ON runs
    FOR EACH ROW
BEGIN
    UPDATE runs SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_policies_updated_at
    AFTER UPDATE ON policies
    FOR EACH ROW
BEGIN
    UPDATE policies SET updated_at = datetime('now') WHERE id = NEW.id;
END;
