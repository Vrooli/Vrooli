-- Qualification evidence is read by product model promotion. Lab experiment
-- runs may write these receipts, but product model listing needs the table.
CREATE TABLE IF NOT EXISTS qualification_evidence (
  id TEXT PRIMARY KEY,
  engine_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  strategy TEXT NOT NULL,
  policy_profile TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL,
  fault_profile TEXT NOT NULL DEFAULT '',
  passed INTEGER NOT NULL,
  artifact_ref TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  machine_json TEXT NOT NULL DEFAULT '{}',
  observed_at TEXT NOT NULL,
  provider_tier TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_qualification_evidence_cell
  ON qualification_evidence(engine_id, model_id, strategy, policy_profile, observed_at DESC);
