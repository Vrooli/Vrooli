CREATE TABLE IF NOT EXISTS publication_intents (
  id TEXT PRIMARY KEY,
  draft_id TEXT NOT NULL,
  destination TEXT NOT NULL,
  version INTEGER NOT NULL,
  source_revision_id TEXT NOT NULL DEFAULT '',
  body_sha256 TEXT NOT NULL,
  desired_local_date TEXT NOT NULL,
  time_zone TEXT NOT NULL,
  daypart TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL CHECK (status = 'desired'),
  created_at TEXT NOT NULL,
  UNIQUE(draft_id, destination)
);
CREATE TABLE IF NOT EXISTS publication_intent_revisions (
  id TEXT PRIMARY KEY,
  intent_id TEXT NOT NULL REFERENCES publication_intents(id),
  version INTEGER NOT NULL,
  source_revision_id TEXT NOT NULL DEFAULT '',
  body_sha256 TEXT NOT NULL,
  desired_local_date TEXT NOT NULL,
  time_zone TEXT NOT NULL,
  daypart TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(intent_id, version)
);
CREATE TABLE IF NOT EXISTS publication_intent_operations (
  idempotency_key TEXT PRIMARY KEY,
  request_fingerprint TEXT NOT NULL,
  intent_revision_id TEXT NOT NULL REFERENCES publication_intent_revisions(id),
  created_at TEXT NOT NULL
);
