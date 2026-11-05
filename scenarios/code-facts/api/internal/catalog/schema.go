package catalog

import _ "embed"

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

// SearchSchema is migration 2. Search content lives outside the FTS virtual
// table so catalog metadata remains normalized and FTS can be rebuilt without
// rediscovering repository files.
func SearchSchema() string { return searchSchemaSQL }

func IndexControlSchema() string { return indexControlSchemaSQL }

func MetricsSchema() string { return metricsSchemaSQL }

// RevisionSchema records the repository commit through which a generation has
// been reconciled. Periodic audits can then inspect the Git delta instead of
// re-reading every governed file.
func RevisionSchema() string { return revisionSchemaSQL }

const searchSchemaSQL = `
CREATE TABLE IF NOT EXISTS code_facts_search_documents (
  rowid INTEGER PRIMARY KEY,
  generation_id TEXT NOT NULL,
  id TEXT NOT NULL,
  source_file_id TEXT NOT NULL,
  source_hash TEXT NOT NULL,
  path TEXT NOT NULL,
  language TEXT NOT NULL,
  role TEXT NOT NULL,
  scope TEXT NOT NULL,
  authority TEXT NOT NULL,
  kind TEXT NOT NULL,
  title TEXT NOT NULL,
  exact_text TEXT NOT NULL,
  split_text TEXT NOT NULL,
  path_text TEXT NOT NULL,
  body TEXT NOT NULL,
  contract_text TEXT NOT NULL DEFAULT '',
  aliases TEXT NOT NULL DEFAULT '',
  start_line INTEGER NOT NULL DEFAULT 0,
  end_line INTEGER NOT NULL DEFAULT 0,
  UNIQUE (generation_id, id),
  FOREIGN KEY (generation_id, source_file_id)
    REFERENCES code_facts_source_files(generation_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS code_facts_search_exact
  ON code_facts_search_documents(generation_id, exact_text);
CREATE INDEX IF NOT EXISTS code_facts_search_path
  ON code_facts_search_documents(generation_id, path);
CREATE INDEX IF NOT EXISTS code_facts_search_filters
  ON code_facts_search_documents(generation_id, scope, role, language);

CREATE VIRTUAL TABLE IF NOT EXISTS code_facts_search_fts USING fts5(
  title, exact_text, split_text, path_text, body, contract_text, aliases,
  content='code_facts_search_documents', content_rowid='rowid',
  tokenize='unicode61 remove_diacritics 2 tokenchars ''_$'''
);

CREATE TRIGGER IF NOT EXISTS code_facts_search_documents_ai AFTER INSERT ON code_facts_search_documents BEGIN
  INSERT INTO code_facts_search_fts(rowid, title, exact_text, split_text, path_text, body, contract_text, aliases)
  VALUES (new.rowid, new.title, new.exact_text, new.split_text, new.path_text, new.body, new.contract_text, new.aliases);
END;
CREATE TRIGGER IF NOT EXISTS code_facts_search_documents_ad AFTER DELETE ON code_facts_search_documents BEGIN
  INSERT INTO code_facts_search_fts(code_facts_search_fts, rowid, title, exact_text, split_text, path_text, body, contract_text, aliases)
  VALUES ('delete', old.rowid, old.title, old.exact_text, old.split_text, old.path_text, old.body, old.contract_text, old.aliases);
END;
CREATE TRIGGER IF NOT EXISTS code_facts_search_documents_au AFTER UPDATE ON code_facts_search_documents BEGIN
  INSERT INTO code_facts_search_fts(code_facts_search_fts, rowid, title, exact_text, split_text, path_text, body, contract_text, aliases)
  VALUES ('delete', old.rowid, old.title, old.exact_text, old.split_text, old.path_text, old.body, old.contract_text, old.aliases);
  INSERT INTO code_facts_search_fts(rowid, title, exact_text, split_text, path_text, body, contract_text, aliases)
  VALUES (new.rowid, new.title, new.exact_text, new.split_text, new.path_text, new.body, new.contract_text, new.aliases);
END;
`

const indexControlSchemaSQL = `
ALTER TABLE code_facts_index_jobs ADD COLUMN cancellation_requested INTEGER NOT NULL DEFAULT 0 CHECK (cancellation_requested IN (0, 1));

CREATE TABLE IF NOT EXISTS code_facts_promotions (
  id TEXT PRIMARY KEY,
  from_generation TEXT NOT NULL,
  to_generation TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('prepared','alias_promoted','committed','rolled_back','failed')),
  error TEXT NOT NULL DEFAULT '',
  created_at_unix INTEGER NOT NULL,
  updated_at_unix INTEGER NOT NULL
);
`

const metricsSchemaSQL = `
CREATE TABLE IF NOT EXISTS code_facts_generation_stats (
  generation_id TEXT PRIMARY KEY REFERENCES code_facts_generations(id) ON DELETE CASCADE,
  source_files INTEGER NOT NULL DEFAULT 0,
  search_documents INTEGER NOT NULL DEFAULT 0,
  semantic_cards INTEGER NOT NULL DEFAULT 0,
  graph_facts INTEGER NOT NULL DEFAULT 0
);
INSERT INTO code_facts_generation_stats(generation_id,source_files,search_documents,semantic_cards,graph_facts)
SELECT g.id,
  (SELECT COUNT(*) FROM code_facts_source_files f WHERE f.generation_id=g.id),
  (SELECT COUNT(*) FROM code_facts_search_documents d WHERE d.generation_id=g.id),
  (SELECT COUNT(*) FROM code_facts_cards c WHERE c.generation_id=g.id),
  (SELECT COUNT(*) FROM code_facts_graph_facts x WHERE x.generation_id=g.id)
FROM code_facts_generations g
WHERE true
ON CONFLICT(generation_id) DO UPDATE SET source_files=excluded.source_files,search_documents=excluded.search_documents,semantic_cards=excluded.semantic_cards,graph_facts=excluded.graph_facts;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_generation_ai AFTER INSERT ON code_facts_generations BEGIN
  INSERT OR IGNORE INTO code_facts_generation_stats(generation_id) VALUES(new.id);
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_files_ai AFTER INSERT ON code_facts_source_files BEGIN
  UPDATE code_facts_generation_stats SET source_files=source_files+1 WHERE generation_id=new.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_files_ad AFTER DELETE ON code_facts_source_files BEGIN
  UPDATE code_facts_generation_stats SET source_files=source_files-1 WHERE generation_id=old.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_documents_ai AFTER INSERT ON code_facts_search_documents BEGIN
  UPDATE code_facts_generation_stats SET search_documents=search_documents+1 WHERE generation_id=new.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_documents_ad AFTER DELETE ON code_facts_search_documents BEGIN
  UPDATE code_facts_generation_stats SET search_documents=search_documents-1 WHERE generation_id=old.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_cards_ai AFTER INSERT ON code_facts_cards BEGIN
  UPDATE code_facts_generation_stats SET semantic_cards=semantic_cards+1 WHERE generation_id=new.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_cards_ad AFTER DELETE ON code_facts_cards BEGIN
  UPDATE code_facts_generation_stats SET semantic_cards=semantic_cards-1 WHERE generation_id=old.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_graph_ai AFTER INSERT ON code_facts_graph_facts BEGIN
  UPDATE code_facts_generation_stats SET graph_facts=graph_facts+1 WHERE generation_id=new.generation_id;
END;
CREATE TRIGGER IF NOT EXISTS code_facts_generation_stats_graph_ad AFTER DELETE ON code_facts_graph_facts BEGIN
  UPDATE code_facts_generation_stats SET graph_facts=graph_facts-1 WHERE generation_id=old.generation_id;
END;
`

const revisionSchemaSQL = `
CREATE TABLE IF NOT EXISTS code_facts_generation_revisions (
  generation_id TEXT PRIMARY KEY REFERENCES code_facts_generations(id) ON DELETE CASCADE,
  git_revision TEXT NOT NULL,
  updated_at_unix INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS code_facts_generation_dirty_paths (
  generation_id TEXT NOT NULL REFERENCES code_facts_generations(id) ON DELETE CASCADE,
  path TEXT NOT NULL,
  PRIMARY KEY (generation_id, path)
);
`
