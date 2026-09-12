CREATE TABLE IF NOT EXISTS ledger_import_keys (
  import_key TEXT PRIMARY KEY,
  source_name TEXT NOT NULL,
  source_path TEXT NOT NULL,
  imported_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ledger_publish_records (
  id TEXT PRIMARY KEY,
  import_key TEXT UNIQUE,
  draft_id TEXT,
  series_id TEXT,
  channel TEXT NOT NULL DEFAULT '',
  audience TEXT NOT NULL DEFAULT '',
  published_url TEXT NOT NULL DEFAULT '',
  platform_post_id TEXT NOT NULL DEFAULT '',
  source_kind TEXT NOT NULL,
  published_at TEXT NOT NULL,
  payload_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ledger_series_links (
  record_id TEXT PRIMARY KEY REFERENCES ledger_publish_records(id),
  prior_record_id TEXT REFERENCES ledger_publish_records(id)
);

CREATE TABLE IF NOT EXISTS ledger_subject_mentions (
  id TEXT PRIMARY KEY,
  import_key TEXT UNIQUE,
  subject TEXT NOT NULL,
  subject_kind TEXT NOT NULL DEFAULT '',
  audience TEXT NOT NULL DEFAULT '',
  channel TEXT NOT NULL DEFAULT '',
  draft_ref TEXT,
  post_url TEXT,
  post_id TEXT,
  is_first_mention INTEGER NOT NULL DEFAULT 0,
  occurred_at TEXT NOT NULL,
  payload_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS ledger_subject_mentions_subject_audience_idx
  ON ledger_subject_mentions(subject, audience, occurred_at DESC);

CREATE TABLE IF NOT EXISTS ledger_narrated_items (
  id TEXT PRIMARY KEY,
  import_key TEXT UNIQUE,
  subject TEXT NOT NULL DEFAULT '',
  scenario TEXT NOT NULL DEFAULT '',
  occurred_at TEXT NOT NULL,
  payload_json TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS ledger_import_runs (
  id TEXT PRIMARY KEY,
  started_at TEXT NOT NULL,
  completed_at TEXT,
  status TEXT NOT NULL,
  source_count INTEGER NOT NULL,
  failure_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS ledger_metric_samples (
  sample_id TEXT PRIMARY KEY,
  release_id TEXT NOT NULL,
  draft_id TEXT NOT NULL,
  metric TEXT NOT NULL,
  value REAL NOT NULL,
  observed_at TEXT NOT NULL,
  received_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS ledger_metric_samples_draft_idx ON ledger_metric_samples(draft_id, observed_at);

CREATE TABLE IF NOT EXISTS ledger_remediations (
  id TEXT PRIMARY KEY,
  publish_record_id TEXT NOT NULL REFERENCES ledger_publish_records(id),
  kind TEXT NOT NULL,
  status TEXT NOT NULL,
  note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  resolved_at TEXT
);
CREATE INDEX IF NOT EXISTS ledger_remediations_record_idx ON ledger_remediations(publish_record_id, created_at DESC);
