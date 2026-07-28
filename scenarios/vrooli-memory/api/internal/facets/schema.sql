CREATE TABLE IF NOT EXISTS facet_definitions (
  id TEXT PRIMARY KEY,
  label TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS facet_policies (
  facet_id TEXT PRIMARY KEY REFERENCES facet_definitions(id),
  retention_policy TEXT NOT NULL,
  compaction_eligible INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS facet_assignments (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL REFERENCES entries(id),
  facet_id TEXT NOT NULL REFERENCES facet_definitions(id),
  assigned_at TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pins (
  entry_id TEXT PRIMARY KEY REFERENCES entries(id),
  review_at TEXT,
  pinned_at TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pin_reviews (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL REFERENCES entries(id),
  due_at TEXT NOT NULL,
  resolved_at TEXT
);

CREATE TABLE IF NOT EXISTS merge_proposals (
  id TEXT PRIMARY KEY,
  rationale TEXT NOT NULL,
  resolved_at TEXT
);

CREATE TABLE IF NOT EXISTS marks (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL REFERENCES entries(id),
  kind TEXT NOT NULL,
  replacement_entry_id TEXT,
  created_at TEXT NOT NULL
);
