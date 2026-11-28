-- startup_measurements persists startup-performance benchmarks per scenario so
-- the trend ("is this scenario starting faster or slower over time?") is
-- answerable. It is written ONLY by the StartupService.BenchmarkStartup RPC (an
-- explicit, operator-triggered measurement); validation/test phases never write
-- here. This is the migrated home of structure-health's former perf_measurements
-- table (axis ② of the three-axis performance model).
CREATE TABLE IF NOT EXISTS startup_measurements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  time_to_healthy_ms INTEGER NOT NULL DEFAULT 0,
  healthy INTEGER NOT NULL DEFAULT 0,
  surface_timings_json TEXT NOT NULL DEFAULT '[]',
  metrics_json TEXT NOT NULL DEFAULT '{}',
  note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_startup_measurements_scenario_captured
  ON startup_measurements (scenario, captured_at DESC);
