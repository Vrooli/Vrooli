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
    max_weighted_tokens INTEGER NOT NULL DEFAULT 0,
    allowed_tools TEXT DEFAULT '[]',
    denied_tools TEXT DEFAULT '[]',
    tool_restriction_policy TEXT NOT NULL DEFAULT 'enforced',
    skip_permission_prompt INTEGER DEFAULT 0,
    features TEXT DEFAULT '{}',
    extra_flags TEXT DEFAULT '{}',
    network_access TEXT NOT NULL DEFAULT 'localhost',
    sandbox_config TEXT DEFAULT '{}',
    spawn_policy TEXT DEFAULT '{}',
    allowed_paths TEXT DEFAULT '[]',
    denied_paths TEXT DEFAULT '[]',
    declared_scopes TEXT DEFAULT '[]',
    skill_pack TEXT DEFAULT '[]',
    skill_experiment_id TEXT DEFAULT '',
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
    program_id TEXT NOT NULL DEFAULT '',
    skill_id TEXT NOT NULL DEFAULT '',
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
	arg_tokens INTEGER NOT NULL DEFAULT 0,
	result_tokens INTEGER NOT NULL DEFAULT 0,
	output_event_id TEXT NOT NULL DEFAULT '',
	output_bytes INTEGER NOT NULL DEFAULT 0,
	token_basis TEXT NOT NULL DEFAULT 'unknown',
	residency_requests INTEGER NOT NULL DEFAULT 0,
	residency_segment INTEGER NOT NULL DEFAULT 0,
	incurred_input_tokens INTEGER NOT NULL DEFAULT 0,
	incurred_output_tokens INTEGER NOT NULL DEFAULT 0,
	incurred_cache_read_tokens INTEGER NOT NULL DEFAULT 0,
	incurred_cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
	tool_profiled INTEGER NOT NULL DEFAULT 0,
	tool_command TEXT NOT NULL DEFAULT '',
	tool_key TEXT NOT NULL DEFAULT '',
	tool_help INTEGER NOT NULL DEFAULT 0,
	tool_filtered INTEGER NOT NULL DEFAULT 0,
	tool_wrapped INTEGER NOT NULL DEFAULT 0,
	tool_truncated INTEGER NOT NULL DEFAULT 0,
	tool_shape_guess INTEGER NOT NULL DEFAULT 0,
	tool_json_paths TEXT NOT NULL DEFAULT '',
	tool_signature TEXT NOT NULL DEFAULT '',
	tool_error_line TEXT NOT NULL DEFAULT '',
	tool_notice TEXT NOT NULL DEFAULT '',
	tool_notice_line TEXT NOT NULL DEFAULT '',
	tool_bypass_owner TEXT NOT NULL DEFAULT '',
	tool_bypass_method TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (run_id, call_event_id, segment_index)
);

CREATE TABLE IF NOT EXISTS program_runtime_events (
    event_id TEXT PRIMARY KEY,
    event_type TEXT NOT NULL,
    source_scenario TEXT NOT NULL DEFAULT '',
    program_id TEXT NOT NULL DEFAULT '',
    session_id TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL DEFAULT '',
    payload_json TEXT NOT NULL,
    received_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_program_runtime_events_program_id ON program_runtime_events(program_id, received_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_occurred_at ON invocation_read_model_facts(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_ownership ON invocation_read_model_facts(ownership, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_outcome ON invocation_read_model_facts(outcome, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_executable ON invocation_read_model_facts(executable, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_fingerprint ON invocation_read_model_facts(fingerprint, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_tool_profiled ON invocation_read_model_facts(occurred_at) WHERE tool_profiled = 1;

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
    cost_source TEXT NOT NULL DEFAULT 'unknown',
    charge_reason TEXT NOT NULL DEFAULT '',
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
	preamble_injected_tokens INTEGER NOT NULL DEFAULT 0,
	preamble_fixed_tokens INTEGER NOT NULL DEFAULT 0,
	preamble_token_basis TEXT NOT NULL DEFAULT 'unknown',
	unattributed_tokens INTEGER NOT NULL DEFAULT 0,
	unattributed_reason TEXT NOT NULL DEFAULT '',
    cost_time_basis TEXT NOT NULL DEFAULT 'terminal_projection',
    time_basis TEXT NOT NULL DEFAULT 'ingestion',
    projected_at TEXT NOT NULL DEFAULT (datetime('now'))
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_occurred_at ON invocation_read_model_runs(occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_status ON invocation_read_model_runs(status, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_profile ON invocation_read_model_runs(profile_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_runner ON invocation_read_model_runs(runner_type, occurred_at);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_runs_model ON invocation_read_model_runs(model, occurred_at);

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

-- Redacted self-report spans share the invocation projection lifecycle, so they
-- remain queryable after raw run_events are pruned.
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
    self_report_classifier_version TEXT NOT NULL DEFAULT '',
    projection_complete INTEGER NOT NULL DEFAULT 0,
    projected_at TEXT NOT NULL DEFAULT (datetime('now'))
);
-- Durable "needs projection" marks. Every terminal-state writer marks its run;
-- the reconcile sweep also enqueues ended runs whose watermark is missing or
-- from an older classifier, then projects due rows idempotently. A mark is
-- cleared only for the generation that was projected, so a terminal write
-- during a projection is never lost; a failure defers the row with backoff.
CREATE TABLE IF NOT EXISTS invocation_read_model_pending (
    run_id TEXT PRIMARY KEY,
    generation INTEGER NOT NULL DEFAULT 1,
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_unix INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_invocation_read_model_pending_due ON invocation_read_model_pending(next_attempt_unix);
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
-- Interactive consent is independent of historical run admission and budgets.
CREATE TABLE IF NOT EXISTS authorization_grants (
    id TEXT PRIMARY KEY,
    root_pid INTEGER NOT NULL,
    state TEXT NOT NULL,
    body TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS authorization_grants_root ON authorization_grants(root_pid, created_at DESC);
-- Standing grants are operator-approved, durable authority for a verified
-- subject (a service principal, or — Phase 5 — a human session) to perform a
-- costly operation within bounds, without a human present for each call. Unlike
-- authorization_grants they are keyed by subject rather than a live process, and
-- carry no run/session binding beyond the optional session_id. Revoking is a
-- state flip; expiry is enforced both by expires_at here and by the grant's own
-- ExpiresAt at evaluation time.
CREATE TABLE IF NOT EXISTS standing_grants (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL,
    body TEXT NOT NULL,
    approved_by TEXT NOT NULL DEFAULT '',
    approved_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    revoked_at TEXT
);
CREATE INDEX IF NOT EXISTS standing_grants_subject ON standing_grants(subject, state);
-- Pending authorization requests back the "ask, don't block" path: when a
-- verified known caller lacks a covering grant for a costly op, agent-manager
-- raises an operator decision via notification-hub instead of a hard refuse.
-- The row makes a repeated raise idempotent (the id is a correlation hash of
-- subject+operation+target), caps outstanding asks per subject, and tells the
-- resolution what to mint on approval. It holds no authority itself.
CREATE TABLE IF NOT EXISTS pending_authorization_requests (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    state TEXT NOT NULL,
    body TEXT NOT NULL,
    raised_at TEXT NOT NULL,
    expires_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS pending_authorization_subject ON pending_authorization_requests(subject, state);
-- Transcript Indexer (USAGE-INTELLIGENCE T-1). One row per transcript file on
-- disk, keyed by file identity (device:inode) so Codex fork rollouts and
-- Claude sub-agent sidechains never collide with their parent session. The
-- indexer tails each file from `offset`; `owner = 'run'` rows are files an
-- Agent Manager run reads itself, registered here and never read for usage.
CREATE TABLE IF NOT EXISTS transcript_files (
    file_key TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    root_kind TEXT NOT NULL,
    root_path TEXT NOT NULL,
    harness TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    header_session_id TEXT NOT NULL DEFAULT '',
    parent_session_id TEXT NOT NULL DEFAULT '',
    parent_file_key TEXT NOT NULL DEFAULT '',
    sidechain INTEGER NOT NULL DEFAULT 0,
    web_console_session_id TEXT NOT NULL DEFAULT '',
    run_id TEXT NOT NULL DEFAULT '',
    owner TEXT NOT NULL DEFAULT 'indexer',
    status TEXT NOT NULL DEFAULT 'pending',
    error TEXT NOT NULL DEFAULT '',
    read_offset INTEGER NOT NULL DEFAULT 0,
    size INTEGER NOT NULL DEFAULT 0,
    mtime_unix_nano INTEGER NOT NULL DEFAULT 0,
    head_hash TEXT NOT NULL DEFAULT '',
    turns INTEGER NOT NULL DEFAULT 0,
    usage_reports INTEGER NOT NULL DEFAULT 0,
    active INTEGER NOT NULL DEFAULT 0,
    recall_bytes INTEGER NOT NULL DEFAULT 0,
    recall_evicted INTEGER NOT NULL DEFAULT 0,
    -- non-text work read after the newest usage report (F-W2-2)
    pending_facts_json TEXT NOT NULL DEFAULT '',
    -- version of the events the parser derived (runner.TranscriptVersioner);
    -- an older version's recall is re-imported from the file
    parser_version INTEGER NOT NULL DEFAULT 0,
    last_observed_event_at TEXT,
    first_seen_at TEXT NOT NULL,
    indexed_at TEXT
);
CREATE INDEX IF NOT EXISTS idx_transcript_files_path ON transcript_files(path);
CREATE INDEX IF NOT EXISTS idx_transcript_files_run ON transcript_files(run_id);
CREATE INDEX IF NOT EXISTS idx_transcript_files_web_console ON transcript_files(web_console_session_id);
CREATE INDEX IF NOT EXISTS idx_transcript_files_session ON transcript_files(session_id);
-- A legacy private transcript remains intact until an owner-controlled cleanup
-- operation can safely establish its identity. Never delete by this path alone.
CREATE TABLE IF NOT EXISTS transcript_legacy_cleanup (
    -- Preserve this obligation even if the run itself is later deleted.
    run_id TEXT PRIMARY KEY,
    file_key TEXT NOT NULL,
    private_path TEXT NOT NULL,
    source_path TEXT NOT NULL
);
-- Durable per-report usage summaries (SPEC D2). Written as a file is read and
-- never evicted with run events. Counters keep the provider's raw cumulative
-- value (Codex) and the report's own amount, so runsignal.AccountUsage derives
-- the same totals from these rows as from retained usage events. The
-- non-text columns count work the request consumed since the previous report
-- (SPEC AM-3): images in, images generated, web search/fetch, audio, tool kinds.
CREATE TABLE IF NOT EXISTS transcript_usage (
    file_key TEXT NOT NULL,
    line_offset INTEGER NOT NULL,
    ordinal INTEGER NOT NULL,
    run_id TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    session_id TEXT NOT NULL DEFAULT '',
    message_id TEXT NOT NULL DEFAULT '',
    parent_tool_use_id TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT '',
    turn_index INTEGER NOT NULL DEFAULT 0,
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    has_cumulative INTEGER NOT NULL DEFAULT 0,
    cum_input_tokens INTEGER NOT NULL DEFAULT 0,
    cum_output_tokens INTEGER NOT NULL DEFAULT 0,
    cum_cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cum_cache_creation_tokens INTEGER NOT NULL DEFAULT 0,
    reconciliation_authority INTEGER NOT NULL DEFAULT 0,
    images_in INTEGER NOT NULL DEFAULT 0,
    images_in_user INTEGER NOT NULL DEFAULT 0,
    images_in_high_detail INTEGER NOT NULL DEFAULT 0,
    image_bytes_in INTEGER NOT NULL DEFAULT 0,
    images_out INTEGER NOT NULL DEFAULT 0,
    web_searches INTEGER NOT NULL DEFAULT 0,
    web_fetches INTEGER NOT NULL DEFAULT 0,
    audio_in INTEGER NOT NULL DEFAULT 0,
    tool_calls INTEGER NOT NULL DEFAULT 0,
    tool_kinds_json TEXT NOT NULL DEFAULT '{}',
    credits_balance TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (file_key, line_offset, ordinal)
);
CREATE INDEX IF NOT EXISTS idx_transcript_usage_run ON transcript_usage(run_id);
CREATE INDEX IF NOT EXISTS idx_transcript_usage_observed ON transcript_usage(observed_at);

-- One row per named one-shot data migration, written in the migration's own
-- transaction so it never runs twice.
CREATE TABLE IF NOT EXISTS schema_migrations_done (
    name TEXT PRIMARY KEY,
    completed_at TEXT NOT NULL
);
