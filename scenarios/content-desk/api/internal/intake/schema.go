package intake

const schema = `
CREATE TABLE IF NOT EXISTS article_intake_batches (
  id TEXT PRIMARY KEY,
  idempotency_key TEXT NOT NULL UNIQUE,
  source_key TEXT NOT NULL UNIQUE,
  request_fingerprint TEXT NOT NULL,
  source_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS article_intake_items (
  batch_id TEXT NOT NULL REFERENCES article_intake_batches(id),
  external_id TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('admissible','unresolved','rejected')),
  unresolved_fields_json TEXT NOT NULL,
  rejected_fields_json TEXT NOT NULL,
  reason TEXT NOT NULL,
  current_receipt_id TEXT NOT NULL,
  PRIMARY KEY(batch_id, external_id)
);
CREATE TABLE IF NOT EXISTS article_intake_receipts (
  id TEXT PRIMARY KEY,
  batch_id TEXT NOT NULL,
  external_id TEXT NOT NULL,
  attempt INTEGER NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('admissible','unresolved','rejected')),
  unresolved_fields_json TEXT NOT NULL,
  rejected_fields_json TEXT NOT NULL,
  reason TEXT NOT NULL,
  payload_json TEXT NOT NULL,
  payload_sha256 TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY(batch_id, external_id) REFERENCES article_intake_items(batch_id, external_id),
  UNIQUE(batch_id, external_id, attempt)
);
CREATE INDEX IF NOT EXISTS article_intake_receipts_item_idx ON article_intake_receipts(batch_id, external_id, attempt);
CREATE TABLE IF NOT EXISTS article_intake_promotions (
  draft_id TEXT PRIMARY KEY,
  batch_id TEXT NOT NULL,
  external_id TEXT NOT NULL,
  receipt_id TEXT NOT NULL,
  source_artifact_id TEXT NOT NULL,
  source_revision_id TEXT NOT NULL,
  variant_id TEXT NOT NULL,
  variant_name TEXT NOT NULL,
  source_payload_sha256 TEXT NOT NULL,
  config_sha256 TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(batch_id, external_id, variant_id),
  FOREIGN KEY(batch_id, external_id) REFERENCES article_intake_items(batch_id, external_id),
  FOREIGN KEY(receipt_id) REFERENCES article_intake_receipts(id)
);
`

func Schema() string { return schema }
