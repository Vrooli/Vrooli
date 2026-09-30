CREATE TABLE IF NOT EXISTS commitments (
  id TEXT PRIMARY KEY,
  result TEXT NOT NULL,
  definition_of_done TEXT NOT NULL DEFAULT '',
  promised_boundary TEXT NOT NULL,
  timezone TEXT NOT NULL,
  beneficiary TEXT NOT NULL DEFAULT '',
  assumptions TEXT NOT NULL DEFAULT '',
  scope_exclusions TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  risk TEXT NOT NULL,
  acknowledgment_status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  revision INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_commitments_state_boundary ON commitments(state, promised_boundary);
CREATE TABLE IF NOT EXISTS commitment_revisions (
  id TEXT PRIMARY KEY,
  commitment_id TEXT NOT NULL,
  promised_boundary TEXT NOT NULL,
  assumptions TEXT NOT NULL DEFAULT '',
  scope_exclusions TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  acknowledgment_status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  revision INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_commitment_revisions_commitment ON commitment_revisions(commitment_id, revision DESC);
