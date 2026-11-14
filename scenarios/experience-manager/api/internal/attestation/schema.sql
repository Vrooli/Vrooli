CREATE TABLE IF NOT EXISTS manual_attestations (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  page_id TEXT NOT NULL,
  claim_id TEXT NOT NULL,
  author TEXT NOT NULL,
  rationale TEXT NOT NULL,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_manual_attestations_claim
  ON manual_attestations (scenario, page_id, claim_id, created_at);
