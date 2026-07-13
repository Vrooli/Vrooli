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
    phases TEXT NOT NULL CHECK (json_valid(phases)),
    started_at TEXT NOT NULL,
    completed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_suite_executions_scenario
    ON suite_executions (scenario_name);

CREATE INDEX IF NOT EXISTS idx_suite_executions_completed_at
    ON suite_executions (completed_at DESC);
