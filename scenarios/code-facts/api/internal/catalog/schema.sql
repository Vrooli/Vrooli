PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS code_facts_catalog_migrations (
  version INTEGER PRIMARY KEY,
  applied_at_unix INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS code_facts_generations (
  id TEXT PRIMARY KEY,
  state TEXT NOT NULL CHECK (state IN ('shadow', 'active', 'retired', 'failed')),
  policy TEXT NOT NULL,
  source_digest TEXT NOT NULL DEFAULT '',
  descriptor_digest TEXT NOT NULL DEFAULT '',
  created_at_unix INTEGER NOT NULL,
  updated_at_unix INTEGER NOT NULL,
  failure TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX IF NOT EXISTS code_facts_one_active_generation
  ON code_facts_generations(state) WHERE state = 'active';

CREATE TABLE IF NOT EXISTS code_facts_source_files (
  generation_id TEXT NOT NULL REFERENCES code_facts_generations(id) ON DELETE CASCADE,
  id TEXT NOT NULL,
  path TEXT NOT NULL,
  language TEXT NOT NULL,
  role TEXT NOT NULL,
  scope TEXT NOT NULL,
  authority TEXT NOT NULL,
  owner TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  size_bytes INTEGER NOT NULL CHECK (size_bytes >= 0),
  mod_time_unix_nano INTEGER NOT NULL,
  searchable INTEGER NOT NULL CHECK (searchable IN (0, 1)),
  PRIMARY KEY (generation_id, id),
  UNIQUE (generation_id, path)
);

CREATE INDEX IF NOT EXISTS code_facts_source_files_role
  ON code_facts_source_files(generation_id, role, path);
CREATE INDEX IF NOT EXISTS code_facts_source_files_scope
  ON code_facts_source_files(generation_id, scope, path);

CREATE TABLE IF NOT EXISTS code_facts_documents (
  generation_id TEXT NOT NULL,
  id TEXT NOT NULL,
  source_file_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  start_line INTEGER NOT NULL DEFAULT 0,
  end_line INTEGER NOT NULL DEFAULT 0,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (generation_id, id),
  FOREIGN KEY (generation_id, source_file_id)
    REFERENCES code_facts_source_files(generation_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS code_facts_cards (
  generation_id TEXT NOT NULL,
  id TEXT NOT NULL,
  document_id TEXT NOT NULL,
  model TEXT NOT NULL,
  chunk_policy TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  eligible INTEGER NOT NULL CHECK (eligible IN (0, 1)),
  metadata_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (generation_id, id),
  FOREIGN KEY (generation_id, document_id)
    REFERENCES code_facts_documents(generation_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS code_facts_graph_facts (
  generation_id TEXT NOT NULL,
  id TEXT NOT NULL,
  source_file_id TEXT NOT NULL,
  subject_id TEXT NOT NULL,
  predicate TEXT NOT NULL,
  object_id TEXT NOT NULL,
  proof_status TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (generation_id, id),
  FOREIGN KEY (generation_id, source_file_id)
    REFERENCES code_facts_source_files(generation_id, id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS code_facts_index_jobs (
  id TEXT PRIMARY KEY,
  generation_id TEXT NOT NULL REFERENCES code_facts_generations(id) ON DELETE CASCADE,
  kind TEXT NOT NULL,
  state TEXT NOT NULL,
  cursor TEXT NOT NULL DEFAULT '',
  processed INTEGER NOT NULL DEFAULT 0,
  total INTEGER NOT NULL DEFAULT 0,
  created_at_unix INTEGER NOT NULL,
  updated_at_unix INTEGER NOT NULL,
  error TEXT NOT NULL DEFAULT ''
);
