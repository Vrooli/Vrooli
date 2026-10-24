CREATE TABLE IF NOT EXISTS drafts (
  id TEXT PRIMARY KEY,
  campaign_id TEXT NOT NULL,
  post_type_id TEXT NOT NULL,
  lane TEXT NOT NULL DEFAULT '',
  sku TEXT NOT NULL DEFAULT '',
  scenario_name TEXT NOT NULL DEFAULT '',
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

-- Agent Manager dispatches are immutable editorial provenance. They never
-- confer approval authority and contain no agent transcript or credentials.
CREATE TABLE IF NOT EXISTS draft_agent_commissions (
  id TEXT PRIMARY KEY,
  draft_id TEXT NOT NULL REFERENCES drafts(id),
  action TEXT NOT NULL CHECK (action IN ('draft', 'evidence-hunt', 'review')),
  task_id TEXT NOT NULL,
  run_id TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS draft_agent_commissions_draft_idx ON draft_agent_commissions(draft_id, created_at);
CREATE TABLE IF NOT EXISTS draft_agent_adoptions (
  commission_id TEXT PRIMARY KEY REFERENCES draft_agent_commissions(id),
  draft_id TEXT NOT NULL REFERENCES drafts(id),
  run_id TEXT NOT NULL,
  adopted_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS draft_approvals (
  draft_id TEXT PRIMARY KEY REFERENCES drafts(id),
  actor_kind TEXT NOT NULL,
  capacity TEXT NOT NULL,
  approved_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS draft_release_targets (
  draft_id TEXT PRIMARY KEY REFERENCES drafts(id),
  identity_id TEXT NOT NULL,
  lane TEXT NOT NULL,
  eligibility TEXT NOT NULL,
  checked_at TEXT NOT NULL
);

-- Content Desk stores only Asset Studio metadata references. Asset bytes and
-- provenance remain owned by Asset Studio.
CREATE TABLE IF NOT EXISTS draft_attachments (
  id TEXT PRIMARY KEY,
  draft_id TEXT NOT NULL REFERENCES drafts(id),
  asset_id TEXT NOT NULL,
  role TEXT NOT NULL,
  aspect_ratio TEXT NOT NULL,
  alt_text TEXT NOT NULL,
  position INTEGER NOT NULL CHECK (position >= 0),
  created_at TEXT NOT NULL,
  UNIQUE(draft_id, position)
);
CREATE INDEX IF NOT EXISTS draft_attachments_draft_idx ON draft_attachments(draft_id, position);
