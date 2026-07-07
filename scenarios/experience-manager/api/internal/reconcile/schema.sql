CREATE TABLE IF NOT EXISTS reconcile_evidence (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  page_id TEXT NOT NULL,
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
  ON reconcile_evidence (scenario, page_id, state_id, claim_id, checked_at);

CREATE INDEX IF NOT EXISTS idx_reconcile_evidence_viewports_viewport
  ON reconcile_evidence_viewports (viewport_id, viewport_width, viewport_height);
