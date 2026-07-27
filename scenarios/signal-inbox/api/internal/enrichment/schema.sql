CREATE TABLE IF NOT EXISTS signal_enrichment (
  id TEXT PRIMARY KEY,
  signal_id TEXT NOT NULL REFERENCES signal(id),
  extracted_content TEXT NOT NULL DEFAULT '',
  content_units INTEGER NOT NULL,
  needs_attention INTEGER NOT NULL DEFAULT 0,
  attention_reason TEXT NOT NULL DEFAULT '',
  attempted_at TEXT NOT NULL,
  CHECK (content_units >= 0),
  CHECK ((content_units = 0 AND extracted_content = '') OR content_units > 0)
);
CREATE INDEX IF NOT EXISTS signal_enrichment_signal_attempt_idx
  ON signal_enrichment(signal_id, attempted_at DESC, id DESC);
