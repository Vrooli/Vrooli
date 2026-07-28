CREATE TABLE IF NOT EXISTS claims (
  id TEXT PRIMARY KEY,
  statement TEXT NOT NULL,
  kind TEXT NOT NULL,
  verification_status TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS claim_evidence (
  id TEXT PRIMARY KEY,
  claim_id TEXT NOT NULL REFERENCES claims(id),
  kind TEXT NOT NULL CHECK (kind IN ('citation','check')),
  reference TEXT NOT NULL DEFAULT '',
  command TEXT NOT NULL DEFAULT '',
  expected_result TEXT NOT NULL DEFAULT '',
  last_result TEXT NOT NULL DEFAULT '',
  last_run_at TEXT
);

CREATE TABLE IF NOT EXISTS claim_citations (
  draft_id TEXT NOT NULL,
  claim_id TEXT NOT NULL REFERENCES claims(id),
  span_start INTEGER NOT NULL CHECK (span_start >= 0),
  span_end INTEGER NOT NULL CHECK (span_end > span_start),
  PRIMARY KEY (draft_id, claim_id, span_start, span_end)
);
CREATE INDEX IF NOT EXISTS claim_citations_claim_idx ON claim_citations(claim_id);

CREATE TABLE IF NOT EXISTS claim_novelty_evidence (
  claim_id TEXT PRIMARY KEY REFERENCES claims(id),
  observed_at TEXT NOT NULL
);
