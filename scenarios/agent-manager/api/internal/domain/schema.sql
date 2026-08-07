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

CREATE TABLE IF NOT EXISTS investigation_cross_scenario_calls (
    run_id TEXT NOT NULL,
    receipt_event_id TEXT NOT NULL,
	occurred_at TEXT,
    target_scenario TEXT NOT NULL,
    operation TEXT NOT NULL,
    outcome TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    duration_ms INTEGER NOT NULL,
    verified INTEGER NOT NULL,
    projection TEXT NOT NULL DEFAULT '{}',
    ledger_availability TEXT NOT NULL,
    PRIMARY KEY (run_id, receipt_event_id)
);
CREATE INDEX IF NOT EXISTS idx_investigation_cross_scenario_calls_run ON investigation_cross_scenario_calls(run_id);
CREATE TABLE IF NOT EXISTS investigation_cross_scenario_call_projections (
	receipt_event_id TEXT NOT NULL,
	key TEXT NOT NULL,
	value_json TEXT NOT NULL,
	PRIMARY KEY (receipt_event_id, key)
);

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
	wrapper TEXT NOT NULL DEFAULT '',
    capability TEXT NOT NULL DEFAULT 'other',
    intent_class TEXT NOT NULL DEFAULT '',
    executable TEXT,
    command_path TEXT,
    ownership TEXT NOT NULL,
    ownership_reason TEXT NOT NULL DEFAULT '',
    segment_index INTEGER NOT NULL DEFAULT 0,
    segment_count INTEGER NOT NULL DEFAULT 1,
    catalog_snapshot TEXT,
    outcome TEXT NOT NULL,
    pairing_basis TEXT NOT NULL DEFAULT 'unpaired',
    failure_signature TEXT NOT NULL DEFAULT '',
    signature_truncated INTEGER NOT NULL DEFAULT 0,
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
    semantics_kind TEXT NOT NULL DEFAULT '',
    semantics_verdict TEXT NOT NULL DEFAULT '',
	exit_code INTEGER,
	duration_ms INTEGER,
    PRIMARY KEY (run_id, call_event_id, segment_index)
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_occurred_at ON invocation_read_model_facts(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_ownership ON invocation_read_model_facts(ownership, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_outcome ON invocation_read_model_facts(outcome, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_executable ON invocation_read_model_facts(executable, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_fingerprint ON invocation_read_model_facts(fingerprint, occurred_at);

CREATE TABLE IF NOT EXISTS invocation_cohort_definitions (
    name TEXT PRIMARY KEY,
    filter_json TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    created_at TEXT NOT NULL,
    change_binding TEXT NOT NULL DEFAULT ''
);

-- One durable terminal summary per run. This is intentionally separate from
-- invocation facts: throughput questions count runs, not tool calls, and must
-- retain terminal timing and cost after the source event log is pruned.
CREATE TABLE IF NOT EXISTS invocation_read_model_runs (
    run_id TEXT PRIMARY KEY,
    goal_id TEXT NOT NULL DEFAULT '',
    goal_status TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL DEFAULT '[]',
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
    workload_kind TEXT NOT NULL DEFAULT 'adhoc',
    workload_key TEXT NOT NULL DEFAULT '',
    workload_instance TEXT NOT NULL DEFAULT '',
    total_cost_usd REAL NOT NULL DEFAULT 0,
    input_cost_usd REAL NOT NULL DEFAULT 0,
    output_cost_usd REAL NOT NULL DEFAULT 0,
    cache_read_cost_usd REAL NOT NULL DEFAULT 0,
    cache_creation_cost_usd REAL NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    turns INTEGER NOT NULL DEFAULT 0,
    tool_calls INTEGER NOT NULL DEFAULT 0,
    total_charge_micro_usd INTEGER NOT NULL DEFAULT 0,
    metered_charge_micro_usd INTEGER NOT NULL DEFAULT 0,
    unpriced_token_count INTEGER NOT NULL DEFAULT 0,
    cost_time_basis TEXT NOT NULL DEFAULT 'terminal_projection',
    time_basis TEXT NOT NULL DEFAULT 'ingestion',
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

-- Time attribution is retained independently of raw events so completed runs
-- remain explainable after the event retention sweep.
CREATE TABLE IF NOT EXISTS invocation_read_model_time_accounting (
    run_id TEXT PRIMARY KEY,
    model_generating_ms INTEGER NOT NULL DEFAULT 0,
    tool_executing_ms INTEGER NOT NULL DEFAULT 0,
    idle_waiting_ms INTEGER NOT NULL DEFAULT 0,
    awaiting_human_ms INTEGER NOT NULL DEFAULT 0,
    unattributable_ms INTEGER NOT NULL DEFAULT 0,
    model_tokens INTEGER NOT NULL DEFAULT 0,
    tool_tokens INTEGER NOT NULL DEFAULT 0,
    idle_tokens INTEGER NOT NULL DEFAULT 0,
    human_tokens INTEGER NOT NULL DEFAULT 0,
    unattributable_tokens INTEGER NOT NULL DEFAULT 0,
    unattributable_reason TEXT NOT NULL DEFAULT ''
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

-- Episodes and redacted self-report spans share the invocation projection
-- lifecycle, so they remain queryable after raw run_events are pruned.
CREATE TABLE IF NOT EXISTS invocation_read_model_episodes (
    run_id TEXT NOT NULL,
    episode_id TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    time_basis TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    pattern TEXT NOT NULL,
    cause_scope TEXT NOT NULL,
    severity TEXT NOT NULL,
    honesty_flags_json TEXT NOT NULL,
    start_event_id TEXT NOT NULL,
    end_event_id TEXT NOT NULL,
    evidence_event_ids_json TEXT NOT NULL,
    turns INTEGER NOT NULL,
    cycle_count INTEGER NOT NULL DEFAULT 0,
    repeated_element TEXT NOT NULL DEFAULT '',
    tokens INTEGER NOT NULL,
    wall_clock_ms INTEGER NOT NULL,
    suspected_owner_scenario TEXT NOT NULL DEFAULT '',
    suspected_owner_command TEXT NOT NULL DEFAULT '',
    owner_confidence TEXT NOT NULL,
    failed_joined_calls INTEGER NOT NULL,
    fingerprint TEXT NOT NULL,
    profile_id TEXT NOT NULL DEFAULT 'unknown',
    runner_type TEXT NOT NULL DEFAULT 'unknown',
    model TEXT NOT NULL DEFAULT 'unknown',
    tag TEXT NOT NULL DEFAULT 'unknown',
    run_status TEXT NOT NULL DEFAULT 'unknown',
    PRIMARY KEY (run_id, episode_id)
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_episodes_occurred_at ON invocation_read_model_episodes(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_episodes_pattern ON invocation_read_model_episodes(pattern, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_episodes_dimensions ON invocation_read_model_episodes(profile_id, runner_type, model, tag, occurred_at);

CREATE TABLE IF NOT EXISTS invocation_read_model_self_report_spans (
    run_id TEXT NOT NULL,
    event_id TEXT NOT NULL,
    rule_id TEXT NOT NULL,
    start_offset INTEGER NOT NULL,
    end_offset INTEGER NOT NULL,
    occurred_at TEXT NOT NULL,
    time_basis TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    cause_scope TEXT NOT NULL,
    text TEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'assistant',
    operator_correction INTEGER NOT NULL DEFAULT 0,
    span_capped INTEGER NOT NULL DEFAULT 0,
    profile_id TEXT NOT NULL DEFAULT 'unknown',
    runner_type TEXT NOT NULL DEFAULT 'unknown',
    model TEXT NOT NULL DEFAULT 'unknown',
    tag TEXT NOT NULL DEFAULT 'unknown',
    run_status TEXT NOT NULL DEFAULT 'unknown',
    PRIMARY KEY (run_id, event_id, rule_id, start_offset)
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_self_report_spans_occurred_at ON invocation_read_model_self_report_spans(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_self_report_spans_rule ON invocation_read_model_self_report_spans(rule_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_self_report_spans_dimensions ON invocation_read_model_self_report_spans(profile_id, runner_type, model, tag, occurred_at);

-- One watermark per source run. Fact replacement and watermark advancement
-- must share a transaction so incremental refresh never reports an event as
-- consumed without its corresponding durable facts.
CREATE TABLE IF NOT EXISTS invocation_read_model_watermarks (
    run_id TEXT PRIMARY KEY,
    last_event_id TEXT NOT NULL,
    last_event_at TEXT NOT NULL,
    classifier_version TEXT NOT NULL,
    episode_classifier_version TEXT NOT NULL DEFAULT '',
    self_report_classifier_version TEXT NOT NULL DEFAULT '',
    projection_complete INTEGER NOT NULL DEFAULT 0,
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
