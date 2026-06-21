-- perf_samples persists per-scenario performance samples (build time, startup,
-- bundle size, LCP) so the trend is answerable. Writes are additive: producers
-- (benchmark, audit/analysis, startup) append rows; nothing is ever overwritten.
CREATE TABLE IF NOT EXISTS perf_samples (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  go_build_ms INTEGER NOT NULL DEFAULT 0,
  ui_build_ms INTEGER NOT NULL DEFAULT 0,
  bundle_bytes INTEGER NOT NULL DEFAULT 0,
  lcp_ms INTEGER NOT NULL DEFAULT 0,
  startup_ms INTEGER NOT NULL DEFAULT 0,
  slowest_component TEXT NOT NULL DEFAULT '',
  slowest_component_avg_ms REAL NOT NULL DEFAULT 0,
  slowest_component_max_ms REAL NOT NULL DEFAULT 0,
  note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_perf_samples_scenario_captured
  ON perf_samples (scenario, captured_at DESC);
