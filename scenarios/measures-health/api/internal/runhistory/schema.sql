-- validation_runs: measures-health's own persisted, countable entity — one row
-- per measures-coverage validation it performs (a `validate scenario <s>`
-- invocation). This is the substrate behind the `validation_run` measures
-- (packages/proto/schemas/measures-health/v1/measures) — measures-health
-- dogfooding the capability it enforces.
--
-- Declarative greenfield schema. CREATE TABLE IF NOT EXISTS is migrate-safe:
-- adding a column later is an ALTER TABLE added below, never a DROP/recreate
-- (see memory feedback_sqlite_always_migrate_never_recreate).
--
-- ran_at is stored at second precision (RFC3339, no fractional seconds) so the
-- measures' lexical [from, to) range scan is correct — RFC3339Nano omits trailing
-- zeros and would make string comparison unsafe (".5Z" sorts before "Z").
CREATE TABLE IF NOT EXISTS validation_runs (
    id            TEXT PRIMARY KEY,
    scenario      TEXT NOT NULL,
    passed        INTEGER NOT NULL,
    error_count   INTEGER NOT NULL DEFAULT 0,
    warning_count INTEGER NOT NULL DEFAULT 0,
    ran_at        TEXT NOT NULL              -- RFC3339, second precision, UTC
);

CREATE INDEX IF NOT EXISTS idx_validation_runs_ran_at ON validation_runs(ran_at);
CREATE INDEX IF NOT EXISTS idx_validation_runs_scenario ON validation_runs(scenario);
