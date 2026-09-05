-- convergence_fitness + reference_health — owned by internal/convergence/. The
-- fitness-audit index: dated per-template four-lens counts (the convergence
-- trend = the compounding proof) and dated per-reference gold-star health
-- verdicts. Numbers are computed live from the template filesystem + soft
-- upstream reads; these tables retain the dated history so the trend is real.
-- Embedded by schema.go and applied via database.EnsureSchemas at boot through
-- the modules.AllSchemas registry. captured_at is RFC3339Nano (see sqlite.go).
-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS convergence_fitness (
  id                     INTEGER PRIMARY KEY AUTOINCREMENT,
  template               TEXT NOT NULL,
  per_replica_cost       INTEGER NOT NULL DEFAULT 0,
  drift_surfaces         INTEGER NOT NULL DEFAULT 0,
  comment_only_contracts INTEGER NOT NULL DEFAULT 0,
  coordinated_edits      INTEGER NOT NULL DEFAULT 0,
  tier                   INTEGER NOT NULL DEFAULT 0,
  captured_at            TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_convergence_fitness_template_at
  ON convergence_fitness(template, captured_at);

CREATE TABLE IF NOT EXISTS reference_health (
  id                  INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario            TEXT NOT NULL,
  stale_from_template INTEGER NOT NULL DEFAULT 0,
  last_template_sync  TEXT NOT NULL DEFAULT '',
  clean_on_all_tools  INTEGER NOT NULL DEFAULT 0,
  stability_days      INTEGER NOT NULL DEFAULT 0,
  breadth             INTEGER NOT NULL DEFAULT 0,
  eligibility         INTEGER NOT NULL DEFAULT 0,
  captured_at         TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reference_health_scenario_at
  ON reference_health(scenario, captured_at);
