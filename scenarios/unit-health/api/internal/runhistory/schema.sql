-- Run-history schema for unit-health.
--
-- Persists each executed validation run's timing and status so the diagnostics
-- analyzer can compute runtime-growth (current vs a rolling baseline) and flake
-- (cross-run status variance) from real history instead of single-run
-- heuristics. Forward-only: CREATE TABLE IF NOT EXISTS, never recreate.
--
-- scenario + started_at are denormalized into the child tables so the history
-- queries are a single SELECT (no JOIN), which keeps them safe under the
-- MaxOpenConns:1 pool (no nested query inside an open rows loop).

CREATE TABLE IF NOT EXISTS unit_runs (
    run_id        TEXT PRIMARY KEY,
    scenario      TEXT NOT NULL,
    started_at    INTEGER NOT NULL,
    status        TEXT NOT NULL,
    maturity_rung INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_unit_runs_scenario ON unit_runs (scenario, started_at);

CREATE TABLE IF NOT EXISTS unit_run_commands (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id        TEXT NOT NULL,
    scenario      TEXT NOT NULL,
    started_at    INTEGER NOT NULL,
    workspace     TEXT NOT NULL,
    command       TEXT NOT NULL,
    duration_ms   INTEGER NOT NULL,
    status        TEXT NOT NULL,
    failure_class TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_unit_run_commands_lookup ON unit_run_commands (scenario, workspace, command, started_at);

CREATE TABLE IF NOT EXISTS unit_run_coverage (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    run_id    TEXT NOT NULL,
    scenario  TEXT NOT NULL,
    workspace TEXT NOT NULL,
    file      TEXT NOT NULL,
    percent   REAL NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_unit_run_coverage_lookup ON unit_run_coverage (scenario, workspace, file);
