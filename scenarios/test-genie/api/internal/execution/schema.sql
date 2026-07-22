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
    fail_fast INTEGER NOT NULL DEFAULT 0 CHECK (fail_fast IN (0, 1)),
    success INTEGER NOT NULL CHECK (success IN (0, 1)),
    -- terminal_outcome classifies the run-level result: passed | failed |
    -- errored | aborted | timeout. Nullable so the brownfield migration can
    -- add it to an existing table and backfill from success.
    terminal_outcome TEXT,
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
    duration_seconds INTEGER NOT NULL DEFAULT 0 CHECK (duration_seconds >= 0),
    error_text TEXT NOT NULL DEFAULT '',
    classification TEXT NOT NULL DEFAULT '',
    remediation TEXT NOT NULL DEFAULT '',
    runnability_verdict TEXT NOT NULL DEFAULT '',
    runnability_reason TEXT NOT NULL DEFAULT '',
    finding_source TEXT NOT NULL DEFAULT '',
    metrics_present INTEGER NOT NULL DEFAULT 0 CHECK (metrics_present IN (0, 1)),
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
    ON suite_execution_phases (phase_name, duration_seconds);

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
