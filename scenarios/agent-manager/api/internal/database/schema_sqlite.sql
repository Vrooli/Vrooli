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
    runner_type TEXT NOT NULL,
    model TEXT,
    model_preset TEXT,
    max_turns INTEGER,
    timeout_ms INTEGER,
    fallback_runner_types TEXT DEFAULT '[]',
    allowed_tools TEXT DEFAULT '[]',
    denied_tools TEXT DEFAULT '[]',
    skip_permission_prompt INTEGER DEFAULT 0,
    requires_sandbox INTEGER DEFAULT 1,
    requires_approval INTEGER DEFAULT 1,
    sandbox_config TEXT DEFAULT '{}',
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
    sandbox_config TEXT DEFAULT '{}',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_tag ON runs(tag);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);

-- Stats query indexes
CREATE INDEX IF NOT EXISTS idx_runs_created_status ON runs(created_at, status);

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

-- Stats query indexes for cost aggregation
CREATE INDEX IF NOT EXISTS idx_run_events_cost ON run_events(run_id, event_type) WHERE event_type = 'metric';
CREATE INDEX IF NOT EXISTS idx_run_events_tool_calls ON run_events(run_id, event_type) WHERE event_type = 'tool_call';
CREATE INDEX IF NOT EXISTS idx_run_events_errors ON run_events(run_id, event_type) WHERE event_type = 'error';

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
-- Model Pricing - Cached pricing data from providers
-- ============================================================================
CREATE TABLE IF NOT EXISTS model_pricing (
    id TEXT PRIMARY KEY,
    canonical_model_name TEXT NOT NULL,
    provider TEXT NOT NULL,

    -- Per-component pricing (USD per token)
    input_token_price REAL,
    output_token_price REAL,
    cache_read_price REAL,
    cache_creation_price REAL,
    web_search_price REAL,
    server_tool_use_price REAL,

    -- Per-component sources (manual_override, provider_api, historical_average, unknown)
    input_token_source TEXT DEFAULT 'unknown',
    output_token_source TEXT DEFAULT 'unknown',
    cache_read_source TEXT,
    cache_creation_source TEXT,
    web_search_source TEXT,
    server_tool_use_source TEXT,

    -- Metadata
    fetched_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    pricing_version TEXT,

    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),

    UNIQUE(canonical_model_name, provider)
);

CREATE INDEX IF NOT EXISTS idx_model_pricing_model ON model_pricing(canonical_model_name);
CREATE INDEX IF NOT EXISTS idx_model_pricing_provider ON model_pricing(provider);
CREATE INDEX IF NOT EXISTS idx_model_pricing_expires ON model_pricing(expires_at);

-- ============================================================================
-- Model Aliases - Maps runner model names to canonical names
-- ============================================================================
CREATE TABLE IF NOT EXISTS model_aliases (
    id TEXT PRIMARY KEY,
    runner_model TEXT NOT NULL,
    runner_type TEXT NOT NULL,
    canonical_model TEXT NOT NULL,
    provider TEXT NOT NULL,

    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),

    UNIQUE(runner_model, runner_type)
);

CREATE INDEX IF NOT EXISTS idx_model_aliases_runner ON model_aliases(runner_model, runner_type);
CREATE INDEX IF NOT EXISTS idx_model_aliases_canonical ON model_aliases(canonical_model);

-- ============================================================================
-- Manual Price Overrides - User-specified pricing overrides
-- ============================================================================
CREATE TABLE IF NOT EXISTS manual_price_overrides (
    id TEXT PRIMARY KEY,
    canonical_model_name TEXT NOT NULL,
    component TEXT NOT NULL,
    price_usd REAL NOT NULL,
    note TEXT,
    created_by TEXT,

    created_at TEXT DEFAULT (datetime('now')),
    expires_at TEXT,

    UNIQUE(canonical_model_name, component)
);

CREATE INDEX IF NOT EXISTS idx_manual_overrides_model ON manual_price_overrides(canonical_model_name);
CREATE INDEX IF NOT EXISTS idx_manual_overrides_expires ON manual_price_overrides(expires_at);

-- ============================================================================
-- Pricing Settings - Global pricing configuration (singleton table)
-- ============================================================================
CREATE TABLE IF NOT EXISTS pricing_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    historical_average_days INTEGER DEFAULT 7,
    provider_cache_ttl_seconds INTEGER DEFAULT 21600,

    updated_at TEXT DEFAULT (datetime('now'))
);

-- Insert default settings if not exists
INSERT OR IGNORE INTO pricing_settings (id, historical_average_days, provider_cache_ttl_seconds)
VALUES (1, 7, 21600);

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

CREATE TRIGGER IF NOT EXISTS update_model_pricing_updated_at
    AFTER UPDATE ON model_pricing
    FOR EACH ROW
BEGIN
    UPDATE model_pricing SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_model_aliases_updated_at
    AFTER UPDATE ON model_aliases
    FOR EACH ROW
BEGIN
    UPDATE model_aliases SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_pricing_settings_updated_at
    AFTER UPDATE ON pricing_settings
    FOR EACH ROW
BEGIN
    UPDATE pricing_settings SET updated_at = datetime('now') WHERE id = NEW.id;
END;
