CREATE TABLE IF NOT EXISTS signal (
  id TEXT PRIMARY KEY,
  source_kind TEXT NOT NULL,
  source_identity TEXT NOT NULL,
  source_url TEXT NOT NULL DEFAULT '',
  raw_payload_ref TEXT NOT NULL DEFAULT '',
  extracted_content TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL,
  needs_attention INTEGER NOT NULL DEFAULT 0,
  captured_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS signal_content_hash_unique ON signal(content_hash);
CREATE INDEX IF NOT EXISTS signal_captured_at_idx ON signal(captured_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS signal_media (
  signal_id TEXT NOT NULL REFERENCES signal(id),
  payload_ref TEXT NOT NULL,
  PRIMARY KEY (signal_id, payload_ref)
);

CREATE TABLE IF NOT EXISTS signal_annotations (
  id TEXT PRIMARY KEY,
  signal_id TEXT NOT NULL REFERENCES signal(id),
  kind TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS signal_annotations_signal_idx ON signal_annotations(signal_id, created_at);
