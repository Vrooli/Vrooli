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

CREATE INDEX IF NOT EXISTS idx_reconcile_evidence_claim
  ON reconcile_evidence (scenario, page_id, state_id, claim_id, checked_at);
