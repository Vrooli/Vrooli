CREATE TABLE IF NOT EXISTS facet_definitions (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  label TEXT NOT NULL,
  classification_guidance TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS facet_policies (
  facet_id TEXT PRIMARY KEY REFERENCES facet_definitions(id),
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  retention_policy TEXT NOT NULL,
  compaction_eligible INTEGER NOT NULL DEFAULT 0,
  resident_budget INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS facet_examples (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  facet_id TEXT NOT NULL REFERENCES facet_definitions(id),
  entry_id TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(scope, facet_id, entry_id)
);

CREATE TABLE IF NOT EXISTS facet_assignments (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  facet_id TEXT NOT NULL REFERENCES facet_definitions(id),
  assigned_at TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pins (
  entry_id TEXT PRIMARY KEY,
  review_at TEXT,
  pinned_at TEXT NOT NULL,
  actor_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS pin_reviews (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  due_at TEXT NOT NULL,
  resolved_at TEXT
);

CREATE TABLE IF NOT EXISTS merge_proposals (
  id TEXT PRIMARY KEY,
  rationale TEXT NOT NULL,
  entry_ids_json TEXT NOT NULL DEFAULT '[]',
  resolved_at TEXT
);

CREATE TABLE IF NOT EXISTS marks (
  id TEXT PRIMARY KEY,
  entry_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  replacement_entry_id TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS classification_rules (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  priority INTEGER NOT NULL,
  facet_id TEXT NOT NULL REFERENCES facet_definitions(id),
  source_runtime TEXT NOT NULL DEFAULT '',
  kind TEXT NOT NULL DEFAULT '',
  source_path_glob TEXT NOT NULL DEFAULT '',
  body_pattern TEXT NOT NULL DEFAULT '',
  enabled INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_classification_rules_order ON classification_rules(scope, enabled, priority, id);

CREATE TABLE IF NOT EXISTS classification_rule_dry_runs (
  id TEXT PRIMARY KEY,
  rule_id TEXT NOT NULL REFERENCES classification_rules(id),
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  corpus_fingerprint TEXT NOT NULL,
  match_count INTEGER NOT NULL,
  samples_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_classification_rule_dry_runs_current ON classification_rule_dry_runs(scope, rule_id, created_at DESC);
