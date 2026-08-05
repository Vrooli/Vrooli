CREATE TABLE IF NOT EXISTS summaries (
  id TEXT PRIMARY KEY,
  scope TEXT NOT NULL DEFAULT 'agent-memory',
  body TEXT NOT NULL,
  facet_id TEXT NOT NULL,
  vector_json TEXT NOT NULL DEFAULT '[]',
  vector_blob BLOB NOT NULL DEFAULT X'',
  depth INTEGER NOT NULL,
  generation INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS tree_edges (
  parent_id TEXT NOT NULL,
  child_id TEXT NOT NULL,
  child_kind TEXT NOT NULL CHECK(child_kind IN ('entry','summary')),
  PRIMARY KEY(parent_id, child_id, child_kind),
  FOREIGN KEY(parent_id) REFERENCES summaries(id)
);
CREATE INDEX IF NOT EXISTS idx_tree_edges_child ON tree_edges(child_id, child_kind);
