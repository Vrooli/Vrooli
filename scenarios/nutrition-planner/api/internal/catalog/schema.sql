CREATE TABLE IF NOT EXISTS catalog_concepts (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS catalog_revisions (
  id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  workspace_id TEXT NOT NULL,
  concept_id TEXT NOT NULL,
  name TEXT NOT NULL,
  product_name TEXT NOT NULL DEFAULT '',
  preparation TEXT NOT NULL DEFAULT '',
  serving_quantity TEXT NOT NULL,
  serving_unit TEXT NOT NULL,
  nutrients_json TEXT NOT NULL DEFAULT '[]',
  allergen_evidence_json TEXT NOT NULL DEFAULT '{}',
  source_type TEXT NOT NULL DEFAULT '',
  source_ref TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY(id, revision),
  FOREIGN KEY(concept_id) REFERENCES catalog_concepts(id)
);
CREATE INDEX IF NOT EXISTS idx_catalog_workspace ON catalog_revisions(workspace_id, created_at DESC);
