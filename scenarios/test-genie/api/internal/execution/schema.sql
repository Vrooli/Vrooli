-- Schema owned by the execution domain: persisted suite executions.
--
-- This table is the reliability ledger's source of truth. Every terminal run
-- outcome (success, failing phases, AND catastrophic abort/timeout/engine-error)
-- writes exactly one row here, so availability denominators are correct.
--
-- Idempotent (CREATE TABLE IF NOT EXISTS); safe to apply on every boot. The
-- terminal_outcome column classifies the run-level result; an existing
-- (pre-column) database is evolved by the guarded migration in
-- migrations.go (never a recreate — the accumulated history is non-disposable).

CREATE TABLE IF NOT EXISTS suite_executions (
    id TEXT PRIMARY KEY,
	-- Durable run artifact identity; required to reload immutable findings and descriptor snapshots.
    run_id TEXT,
    scenario_name TEXT NOT NULL,
    target_kind TEXT NOT NULL DEFAULT 'scenario',
    target_id TEXT NOT NULL DEFAULT '',
    preset_used TEXT,
    requested_preset TEXT,
    requested_phases TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(requested_phases)),
    requested_skip_phases TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(requested_skip_phases)),
    planned_phases TEXT NOT NULL DEFAULT '[]' CHECK (json_valid(planned_phases)),
	-- Comparability key for full-run timing history. Legacy rows intentionally
	-- remain NULL and therefore participate only in the degraded additive path.
	phase_set_digest TEXT,
	descriptor_snapshot_digest TEXT,
	configuration_fingerprint TEXT,
	host_os TEXT,
	host_arch TEXT,
	host_node TEXT,
	host_fact_digest TEXT,
	fail_fast INTEGER NOT NULL DEFAULT 0 CHECK (fail_fast IN (0, 1)),
    success INTEGER NOT NULL CHECK (success IN (0, 1)),
    -- terminal_outcome classifies the run-level result: passed | failed |
    -- errored | aborted | timeout. Nullable so the brownfield migration can
    -- add it to an existing table and backfill from success.
    terminal_outcome TEXT,
    -- requested_at is when the run was ASKED for; started_at is when it got a
    -- concurrency slot. The gap between them is queue latency, which is the
    -- part of "why was my suite slow" that execution timings cannot see.
    requested_at TEXT,
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_suite_executions_scenario
    ON suite_executions (scenario_name);

CREATE INDEX IF NOT EXISTS idx_suite_executions_completed_at
    ON suite_executions (completed_at DESC);

-- suite_execution_phases is the only queryable phase-history projection.
-- Rich findings, provider metrics, logs, and presentation payloads belong to
-- immutable run evidence.  Keeping the compact fields in rows, instead of a
-- JSON array on suite_executions, lets reliability and planning queries use
-- indexes without hydrating or parsing historical result documents.
CREATE TABLE IF NOT EXISTS suite_execution_phases (
    execution_id TEXT NOT NULL REFERENCES suite_executions(id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    phase_name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT '',
    duration_ms INTEGER NOT NULL DEFAULT 0 CHECK (duration_ms >= 0),
    predicted_duration_ms INTEGER CHECK (predicted_duration_ms IS NULL OR predicted_duration_ms >= 0),
    duration_seconds INTEGER NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    error_text TEXT NOT NULL DEFAULT '',
    classification TEXT NOT NULL DEFAULT '',
    classification_source TEXT NOT NULL DEFAULT '',
    remediation TEXT NOT NULL DEFAULT '',
    runnability_verdict TEXT NOT NULL DEFAULT '',
    runnability_reason TEXT NOT NULL DEFAULT '',
    finding_source TEXT NOT NULL DEFAULT '',
    metrics_present INTEGER NOT NULL DEFAULT 0 CHECK (metrics_present IN (0, 1)),
    wall_clock_ms INTEGER,
    cpu_user_ms INTEGER,
    cpu_sys_ms INTEGER,
    peak_rss_bytes INTEGER,
    cpu_reliability TEXT,
    memory_reliability TEXT,
    gpu_reliability TEXT,
    cache_hit INTEGER NOT NULL DEFAULT 0 CHECK (cache_hit IN (0, 1)),
    cache_source_run_id TEXT NOT NULL DEFAULT '',
    cache_audit INTEGER NOT NULL DEFAULT 0 CHECK (cache_audit IN (0, 1)),
    cache_audit_mismatch INTEGER NOT NULL DEFAULT 0 CHECK (cache_audit_mismatch IN (0, 1)),
    cache_no_saving INTEGER NOT NULL DEFAULT 0 CHECK (cache_no_saving IN (0, 1)),
    findings_blockers INTEGER NOT NULL DEFAULT 0,
    findings_errors INTEGER NOT NULL DEFAULT 0,
    findings_warnings INTEGER NOT NULL DEFAULT 0,
    findings_infos INTEGER NOT NULL DEFAULT 0,
    findings_total INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (execution_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_suite_execution_phases_execution
    ON suite_execution_phases (execution_id, ordinal);

CREATE INDEX IF NOT EXISTS idx_suite_execution_phases_name_duration
    ON suite_execution_phases (phase_name, duration_ms);

CREATE INDEX IF NOT EXISTS idx_suite_execution_phases_scenario_phase
    ON suite_execution_phases (phase_name, execution_id);

-- Preparation stages are the compact orchestration-timing projection. They
-- deliberately live beside, not inside, suite_execution_phases so historical
-- phase estimates and reliability remain semantically pure.
CREATE TABLE IF NOT EXISTS suite_execution_stages (
    execution_id TEXT NOT NULL REFERENCES suite_executions(id) ON DELETE CASCADE,
    ordinal INTEGER NOT NULL,
    stage_name TEXT NOT NULL,
    parent_stage TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    duration_milliseconds INTEGER NOT NULL DEFAULT 0 CHECK (duration_milliseconds >= 0),
    PRIMARY KEY (execution_id, ordinal)
);

CREATE INDEX IF NOT EXISTS idx_suite_execution_stages_name_duration
    ON suite_execution_stages (stage_name, duration_milliseconds);
