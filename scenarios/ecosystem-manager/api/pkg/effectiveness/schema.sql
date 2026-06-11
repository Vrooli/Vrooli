-- Skill/dimension effectiveness ledger — owned by pkg/effectiveness, applied via
-- database.EnsureSchemas at boot (see pkg/dbschema). SQLite dialect. Timestamps
-- are written as UTC time.Time values by the repository; the epoch default is a
-- schema-completeness placeholder (every insert supplies last_run_at).
CREATE TABLE IF NOT EXISTS skill_dimension_effectiveness (
    skill_id TEXT NOT NULL,
    dimension TEXT NOT NULL,
    closed_count INTEGER NOT NULL DEFAULT 0,
    introduced_count INTEGER NOT NULL DEFAULT 0,
    total_runs INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    last_run_at TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00',
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (skill_id, dimension)
);
