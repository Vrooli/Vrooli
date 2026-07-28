CREATE TABLE IF NOT EXISTS post_types (
  id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  paired_skill TEXT NOT NULL DEFAULT '',
  skill_exists INTEGER NOT NULL DEFAULT 0,
  doc_v1 INTEGER NOT NULL DEFAULT 0,
  responsibilities_declared INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS post_type_failure_modes (
  post_type_id TEXT NOT NULL REFERENCES post_types(id),
  failure_mode TEXT NOT NULL,
  PRIMARY KEY (post_type_id, failure_mode)
);
