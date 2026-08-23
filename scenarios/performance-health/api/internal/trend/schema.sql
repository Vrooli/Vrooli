-- perf_samples persists per-scenario performance samples (build time, startup,
-- bundle size, LCP) so the trend is answerable. Writes are additive: producers
-- (benchmark, audit/analysis, startup) append rows; nothing is ever overwritten.
CREATE TABLE IF NOT EXISTS perf_samples (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  flow TEXT NOT NULL DEFAULT '',
  captured_at TEXT NOT NULL,
  go_build_ms INTEGER NOT NULL DEFAULT 0,
  ui_build_ms INTEGER NOT NULL DEFAULT 0,
  bundle_bytes INTEGER NOT NULL DEFAULT 0,
  lcp_ms INTEGER NOT NULL DEFAULT 0,
  -- Cumulative layout shift is unitless and fractional, so REAL rather than the
  -- INTEGER millisecond columns around it.
  cls REAL NOT NULL DEFAULT 0,
  -- PerformanceNavigationTiming phases, milliseconds from navigation start.
  response_end_ms INTEGER NOT NULL DEFAULT 0,
  dom_interactive_ms INTEGER NOT NULL DEFAULT 0,
  dom_content_loaded_ms INTEGER NOT NULL DEFAULT 0,
  load_event_end_ms INTEGER NOT NULL DEFAULT 0,
  -- Navigation kind ("navigate", "reload", "back_forward", "prerender"); a
  -- reload is not comparable with a cold navigate.
  navigation_type TEXT NOT NULL DEFAULT '',
  startup_ms INTEGER NOT NULL DEFAULT 0,
  slowest_component TEXT NOT NULL DEFAULT '',
  slowest_component_avg_ms REAL NOT NULL DEFAULT 0,
  slowest_component_max_ms REAL NOT NULL DEFAULT 0,
  drawn_fps REAL NOT NULL DEFAULT 0,
  dropped_frame_rate REAL NOT NULL DEFAULT 0,
  long_task_total_ms INTEGER NOT NULL DEFAULT 0,
  long_task_max_ms REAL NOT NULL DEFAULT 0,
  raster_total_ms REAL NOT NULL DEFAULT 0,
  layout_total_ms REAL NOT NULL DEFAULT 0,
  paint_total_ms REAL NOT NULL DEFAULT 0,
  input_event_count INTEGER NOT NULL DEFAULT 0,
  note TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_perf_samples_scenario_captured
  ON perf_samples (scenario, captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_perf_samples_scenario_flow_captured
  ON perf_samples (scenario, flow, captured_at DESC);
