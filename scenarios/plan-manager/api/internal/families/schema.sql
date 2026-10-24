CREATE TABLE IF NOT EXISTS plan_families (
  family_id TEXT PRIMARY KEY,
  slug TEXT NOT NULL UNIQUE,
  revision INTEGER NOT NULL,
  document BLOB NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS plan_family_graph_revisions (
  family_id TEXT NOT NULL,
  graph_revision INTEGER NOT NULL,
  document BLOB NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY (family_id, graph_revision),
  FOREIGN KEY (family_id) REFERENCES plan_families(family_id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS plan_family_reviews (
  family_id TEXT NOT NULL,
  graph_revision INTEGER NOT NULL,
  document BLOB NOT NULL,
  reviewed_at TEXT NOT NULL,
  PRIMARY KEY (family_id, graph_revision),
  FOREIGN KEY (family_id, graph_revision) REFERENCES plan_family_graph_revisions(family_id, graph_revision) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_plan_families_updated ON plan_families(updated_at DESC, family_id);
