CREATE TABLE IF NOT EXISTS evidence_receipts (
  receipt_id TEXT PRIMARY KEY,
  observation_id TEXT NOT NULL UNIQUE,
  producer_execution_id TEXT NOT NULL DEFAULT '',
  url TEXT NOT NULL,
  final_url TEXT NOT NULL DEFAULT '',
  redirect_urls TEXT NOT NULL DEFAULT '[]',
  retrieved_at TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  artifact_id TEXT NOT NULL,
  extraction_revision TEXT NOT NULL,
  retention TEXT NOT NULL DEFAULT '',
  failure_code TEXT NOT NULL DEFAULT '',
  etag TEXT NOT NULL DEFAULT '',
  last_modified TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS evidence_artifacts (
  artifact_id TEXT PRIMARY KEY,
  content_hash TEXT NOT NULL UNIQUE,
  content BLOB,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS evidence_passages (
  passage_id TEXT PRIMARY KEY,
  receipt_id TEXT NOT NULL,
  start_byte INTEGER NOT NULL,
  end_byte INTEGER NOT NULL,
  passage_hash TEXT NOT NULL,
  FOREIGN KEY (receipt_id) REFERENCES evidence_receipts(receipt_id)
);

CREATE INDEX IF NOT EXISTS idx_evidence_passages_receipt
  ON evidence_passages(receipt_id);

CREATE TABLE IF NOT EXISTS evidence_assessments (
  assessment_id TEXT PRIMARY KEY,
  claim_id TEXT NOT NULL,
  disposition TEXT NOT NULL,
  policy_revision TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '',
  evidence_json TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_evidence_assessments_claim
  ON evidence_assessments(claim_id);
