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

-- Passive investigation projections are derived only from durable run events.
-- They contain no raw tool input/output and can be deterministically rebuilt.
CREATE TABLE IF NOT EXISTS investigation_invocation_facts (
    run_id TEXT NOT NULL,
    call_event_id TEXT NOT NULL,
    result_event_id TEXT,
    tool_call_id TEXT,
    tool_name TEXT NOT NULL,
    executable TEXT,
    command_path TEXT,
    ownership TEXT NOT NULL,
    catalog_snapshot TEXT,
    outcome TEXT NOT NULL,
    retry_of_call_event_id TEXT,
    help_recovery INTEGER NOT NULL DEFAULT 0,
    fingerprint TEXT NOT NULL,
    availability TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    PRIMARY KEY (run_id, call_event_id)
);
CREATE INDEX IF NOT EXISTS idx_investigation_invocation_facts_fingerprint ON investigation_invocation_facts(run_id, fingerprint);

-- Durable analytics read model. Unlike the investigation cache above, these
-- facts are written at terminal run completion and retained after run_events
-- are pruned. The source-event identifiers keep the model replayable while
-- time_basis makes pruned-history fallbacks explicit rather than guessed.
CREATE TABLE IF NOT EXISTS invocation_read_model_facts (
    run_id TEXT NOT NULL,
    call_event_id TEXT NOT NULL,
    result_event_id TEXT,
    tool_call_id TEXT,
    occurred_at TEXT NOT NULL,
    time_basis TEXT NOT NULL DEFAULT 'event',
    tool_name TEXT NOT NULL,
    executable TEXT,
    command_path TEXT,
    ownership TEXT NOT NULL,
    catalog_snapshot TEXT,
    outcome TEXT NOT NULL,
    retry_of_call_event_id TEXT,
    help_recovery INTEGER NOT NULL DEFAULT 0,
    fingerprint TEXT NOT NULL,
    availability TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    profile_id TEXT,
    runner_type TEXT NOT NULL DEFAULT 'unknown',
    model TEXT NOT NULL DEFAULT '',
    tag TEXT NOT NULL DEFAULT '',
    run_status TEXT NOT NULL,
    PRIMARY KEY (run_id, call_event_id)
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_occurred_at ON invocation_read_model_facts(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_ownership ON invocation_read_model_facts(ownership, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_outcome ON invocation_read_model_facts(outcome, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_executable ON invocation_read_model_facts(executable, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_fingerprint ON invocation_read_model_facts(fingerprint, occurred_at);

-- One durable terminal summary per run. This is intentionally separate from
-- invocation facts: throughput questions count runs, not tool calls, and must
-- retain terminal timing and cost after the source event log is pruned.
CREATE TABLE IF NOT EXISTS invocation_read_model_runs (
    run_id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    ended_at TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL,
    profile_id TEXT NOT NULL DEFAULT 'unknown',
    runner_type TEXT NOT NULL DEFAULT 'unknown',
    model TEXT NOT NULL DEFAULT 'unknown',
    tag TEXT NOT NULL DEFAULT 'unknown',
    total_cost_usd REAL NOT NULL DEFAULT 0,
    authoritative_cost_usd REAL NOT NULL DEFAULT 0,
    estimated_cost_usd REAL NOT NULL DEFAULT 0,
    unknown_cost_usd REAL NOT NULL DEFAULT 0,
    input_cost_usd REAL NOT NULL DEFAULT 0,
    output_cost_usd REAL NOT NULL DEFAULT 0,
    cache_read_cost_usd REAL NOT NULL DEFAULT 0,
    cache_creation_cost_usd REAL NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    cost_time_basis TEXT NOT NULL DEFAULT 'terminal_projection',
    projected_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_occurred_at ON invocation_read_model_runs(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_status ON invocation_read_model_runs(status, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_profile ON invocation_read_model_runs(profile_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_runner ON invocation_read_model_runs(runner_type, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_model ON invocation_read_model_runs(model, occurred_at);

-- Per-run efficiency signals use the same terminal lifecycle as the run
-- summary, while staying separate so this additive schema remains compatible
-- with databases created before the durable run projection existed.
CREATE TABLE IF NOT EXISTS invocation_read_model_run_signals (
    run_id TEXT PRIMARY KEY,
    read_calls INTEGER NOT NULL DEFAULT 0,
    files_read_more_than_once INTEGER NOT NULL DEFAULT 0
);

-- Error-code facts retain the analytical pattern after source run_events are
-- pruned. Messages and stack traces are intentionally excluded.
CREATE TABLE IF NOT EXISTS invocation_read_model_errors (
    run_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    time_basis TEXT NOT NULL,
    error_code TEXT NOT NULL,
    profile_id TEXT NOT NULL DEFAULT 'unknown',
    runner_type TEXT NOT NULL DEFAULT 'unknown',
    model TEXT NOT NULL DEFAULT 'unknown',
    tag TEXT NOT NULL DEFAULT 'unknown',
    PRIMARY KEY (run_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_errors_occurred_at ON invocation_read_model_errors(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_errors_code ON invocation_read_model_errors(error_code, occurred_at);

-- One watermark per source run. Fact replacement and watermark advancement
-- must share a transaction so incremental refresh never reports an event as
-- consumed without its corresponding durable facts.
CREATE TABLE IF NOT EXISTS invocation_read_model_watermarks (
    run_id TEXT PRIMARY KEY,
    last_event_id TEXT NOT NULL,
    last_event_at TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    projected_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE TABLE IF NOT EXISTS investigation_receipt_evidence (
    run_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    availability TEXT NOT NULL,
    PRIMARY KEY (run_id, event_id)
);
CREATE TRIGGER IF NOT EXISTS update_agent_profiles_updated_at
    AFTER UPDATE ON agent_profiles
    FOR EACH ROW
BEGIN
    UPDATE agent_profiles SET updated_at = datetime('now') WHERE id = NEW.id;
END;
