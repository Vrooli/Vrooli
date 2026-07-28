CREATE TABLE IF NOT EXISTS drafts (
  id TEXT PRIMARY KEY,
  campaign_id TEXT NOT NULL,
  post_type_id TEXT NOT NULL,
  lane TEXT NOT NULL DEFAULT '',
  sku TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS draft_events (
  id TEXT PRIMARY KEY,
  draft_id TEXT NOT NULL REFERENCES drafts(id),
  event TEXT NOT NULL,
  from_status TEXT NOT NULL,
  to_status TEXT NOT NULL,
  occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS draft_events_draft_id_idx ON draft_events(draft_id, occurred_at);

CREATE TABLE IF NOT EXISTS draft_slots (
  draft_id TEXT PRIMARY KEY REFERENCES drafts(id),
  campaign_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  format TEXT NOT NULL,
  released_at TEXT
);

CREATE TABLE IF NOT EXISTS draft_revisions (
  id TEXT PRIMARY KEY,
  draft_id TEXT NOT NULL REFERENCES drafts(id),
  body TEXT NOT NULL,
  actor_kind TEXT NOT NULL,
  capacity TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS draft_revisions_draft_id_idx ON draft_revisions(draft_id, created_at);

CREATE TABLE IF NOT EXISTS draft_approvals (
  draft_id TEXT PRIMARY KEY REFERENCES drafts(id),
  actor_kind TEXT NOT NULL,
  capacity TEXT NOT NULL,
  approved_at TEXT NOT NULL
);
