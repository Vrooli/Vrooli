CREATE TABLE IF NOT EXISTS forensic_cases (
  id TEXT PRIMARY KEY,
  document_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_forensic_cases_updated
  ON forensic_cases(updated_at DESC, id DESC);

CREATE TABLE IF NOT EXISTS forensic_packets (
  case_id TEXT NOT NULL,
  schema_version TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  packet_json BLOB NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (case_id, sha256),
  FOREIGN KEY (case_id) REFERENCES forensic_cases(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_forensic_packets_case_created
  ON forensic_packets(case_id, created_at DESC, sha256);

CREATE TABLE IF NOT EXISTS analyst_labels (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  subject TEXT NOT NULL,
  label TEXT NOT NULL,
  source_url TEXT NOT NULL DEFAULT '',
  source_citation TEXT NOT NULL DEFAULT '',
  grade TEXT NOT NULL,
  counter_explanation TEXT NOT NULL DEFAULT '',
  unresolved_gap TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  FOREIGN KEY (case_id) REFERENCES forensic_cases(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_analyst_labels_case_created
  ON analyst_labels(case_id, created_at, id);

CREATE TABLE IF NOT EXISTS forensic_case_revisions (
  id TEXT PRIMARY KEY,
  case_id TEXT NOT NULL,
  previous_updated_at TEXT NOT NULL,
  applied_at TEXT NOT NULL,
  additions_json TEXT NOT NULL,
  FOREIGN KEY (case_id) REFERENCES forensic_cases(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_forensic_case_revisions_case_applied
  ON forensic_case_revisions(case_id, applied_at, id);
