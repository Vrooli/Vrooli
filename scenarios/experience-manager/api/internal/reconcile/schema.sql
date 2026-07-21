CREATE TABLE IF NOT EXISTS reconcile_evidence (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  document_kind TEXT NOT NULL DEFAULT 'page',
  page_id TEXT NOT NULL,
  component_id TEXT NOT NULL DEFAULT '',
  component_title TEXT NOT NULL DEFAULT '',
  example_name TEXT NOT NULL DEFAULT '',
  route TEXT NOT NULL,
  state_id TEXT NOT NULL,
  claim_id TEXT NOT NULL,
  claim_type TEXT NOT NULL,
  verdict TEXT NOT NULL,
  capture_ref TEXT NOT NULL,
  ax_node_json TEXT NOT NULL,
  message TEXT NOT NULL,
  checked_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reconcile_evidence_viewports (
  evidence_id TEXT PRIMARY KEY,
  viewport_id TEXT NOT NULL,
  viewport_width INTEGER NOT NULL,
  viewport_height INTEGER NOT NULL,
  FOREIGN KEY (evidence_id) REFERENCES reconcile_evidence(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_reconcile_evidence_claim
  ON reconcile_evidence (scenario, page_id, component_id, state_id, claim_id, checked_at);

CREATE INDEX IF NOT EXISTS idx_reconcile_evidence_viewports_viewport
  ON reconcile_evidence_viewports (viewport_id, viewport_width, viewport_height);

CREATE TABLE IF NOT EXISTS reconcile_capture_timings (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  document_kind TEXT NOT NULL,
  page_id TEXT NOT NULL,
  component_id TEXT NOT NULL DEFAULT '',
  route TEXT NOT NULL,
  state_id TEXT NOT NULL,
  viewport_id TEXT NOT NULL,
  viewport_width INTEGER NOT NULL,
  viewport_height INTEGER NOT NULL,
  total_milliseconds INTEGER NOT NULL,
  navigation_milliseconds INTEGER NOT NULL,
  readiness_wait_milliseconds INTEGER NOT NULL,
  strategy TEXT NOT NULL,
  outcome TEXT NOT NULL,
  captured_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_reconcile_capture_timings_target
  ON reconcile_capture_timings (scenario, page_id, component_id, state_id, viewport_id, captured_at DESC);
