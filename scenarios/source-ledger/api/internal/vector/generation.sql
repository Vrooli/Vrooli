-- A census is an observation of rows already owned by Source Ledger. Rows
-- whose historical encoder cannot be proven stay explicit sentinels; they
-- must not be used as a migration recipe or serving generation.
CREATE TABLE IF NOT EXISTS vector_generation_census (
  store_id TEXT NOT NULL,
  dataset_id TEXT NOT NULL,
  generation_id TEXT NOT NULL,
  recipe_id TEXT NOT NULL DEFAULT '',
  identity_state TEXT NOT NULL CHECK(identity_state IN ('identity_unknown','recipe_bound')),
  dimensions INTEGER NOT NULL CHECK(dimensions >= 0),
  vector_count INTEGER NOT NULL CHECK(vector_count >= 0),
  missing_count INTEGER NOT NULL CHECK(missing_count >= 0),
  invalid_count INTEGER NOT NULL CHECK(invalid_count >= 0),
  observed_at TEXT NOT NULL,
  PRIMARY KEY(store_id, dataset_id, generation_id, dimensions),
  CHECK((identity_state = 'identity_unknown' AND recipe_id = '' AND generation_id = 'unresolved') OR
        (identity_state = 'recipe_bound' AND recipe_id <> '' AND generation_id <> 'unresolved'))
);

-- Source changes are owner-emitted by triggers so re-embedding can catch up
-- after a pinned SQLite snapshot without using timestamps as cursors.
CREATE TABLE IF NOT EXISTS vector_generation_mutex (
  singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
  version INTEGER NOT NULL DEFAULT 0
);
INSERT OR IGNORE INTO vector_generation_mutex(singleton,version) VALUES(1,0);

-- In-place vector enrichment can complete without changing its source text.
-- Advance the recall snapshot version in the same transaction as that write.
CREATE TABLE IF NOT EXISTS vector_read_version (
  singleton INTEGER PRIMARY KEY CHECK(singleton = 1),
  version INTEGER NOT NULL DEFAULT 0
);
INSERT OR IGNORE INTO vector_read_version(singleton,version) VALUES(1,0);

CREATE TRIGGER IF NOT EXISTS vector_read_embedding_insert AFTER INSERT ON embeddings BEGIN
  UPDATE vector_read_version SET version=version+1 WHERE singleton=1;
END;
CREATE TRIGGER IF NOT EXISTS vector_read_embedding_update AFTER UPDATE ON embeddings BEGIN
  UPDATE vector_read_version SET version=version+1 WHERE singleton=1;
END;
CREATE TRIGGER IF NOT EXISTS vector_read_embedding_delete AFTER DELETE ON embeddings BEGIN
  UPDATE vector_read_version SET version=version+1 WHERE singleton=1;
END;
CREATE TRIGGER IF NOT EXISTS vector_read_summary_update AFTER UPDATE OF vector_blob,vector_json ON summaries BEGIN
  UPDATE vector_read_version SET version=version+1 WHERE singleton=1;
END;

CREATE TABLE IF NOT EXISTS vector_generation_changes (
  sequence INTEGER PRIMARY KEY AUTOINCREMENT,
  dataset_id TEXT NOT NULL CHECK(dataset_id IN ('journal.facets','forest.summaries')),
  source_id TEXT NOT NULL,
  operation TEXT NOT NULL CHECK(operation IN ('upsert','delete')),
  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vector_generation_changes_source
  ON vector_generation_changes(dataset_id,source_id,sequence);

DROP TRIGGER IF EXISTS vector_generation_facet_insert;
CREATE TRIGGER vector_generation_facet_insert AFTER INSERT ON facet_texts BEGIN
  UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1;
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('journal.facets',NEW.id,'upsert');
END;
DROP TRIGGER IF EXISTS vector_generation_facet_update;
CREATE TRIGGER vector_generation_facet_update AFTER UPDATE OF text,entry_id ON facet_texts BEGIN
  UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1;
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('journal.facets',OLD.id,'delete');
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('journal.facets',NEW.id,'upsert');
END;
DROP TRIGGER IF EXISTS vector_generation_facet_delete;
CREATE TRIGGER vector_generation_facet_delete AFTER DELETE ON facet_texts BEGIN
  UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1;
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('journal.facets',OLD.id,'delete');
END;
DROP TRIGGER IF EXISTS vector_generation_summary_insert;
CREATE TRIGGER vector_generation_summary_insert AFTER INSERT ON summaries BEGIN
  UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1;
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('forest.summaries',NEW.id,'upsert');
END;
DROP TRIGGER IF EXISTS vector_generation_summary_update;
CREATE TRIGGER vector_generation_summary_update AFTER UPDATE OF body,scope ON summaries BEGIN
  UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1;
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('forest.summaries',OLD.id,'delete');
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('forest.summaries',NEW.id,'upsert');
END;
DROP TRIGGER IF EXISTS vector_generation_summary_delete;
CREATE TRIGGER vector_generation_summary_delete AFTER DELETE ON summaries BEGIN
  UPDATE vector_generation_mutex SET version=version+1 WHERE singleton=1;
  INSERT INTO vector_generation_changes(dataset_id,source_id,operation)
  VALUES('forest.summaries',OLD.id,'delete');
END;

CREATE TABLE IF NOT EXISTS vector_generation_snapshots (
  snapshot_id TEXT PRIMARY KEY,
  watermark INTEGER NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS vector_generation_snapshot_items (
  item_index INTEGER PRIMARY KEY AUTOINCREMENT,
  snapshot_id TEXT NOT NULL REFERENCES vector_generation_snapshots(snapshot_id),
  dataset_id TEXT NOT NULL CHECK(dataset_id IN ('journal.facets','forest.summaries')),
  source_id TEXT NOT NULL,
  UNIQUE(snapshot_id,dataset_id,source_id)
);
CREATE INDEX IF NOT EXISTS idx_vector_generation_snapshot_items_page
  ON vector_generation_snapshot_items(snapshot_id,item_index);

CREATE TABLE IF NOT EXISTS vector_generations (
  generation_id TEXT PRIMARY KEY,
  recipe_id TEXT NOT NULL,
  recipe_canonical BLOB NOT NULL,
  dimensions INTEGER NOT NULL CHECK(dimensions > 0),
  state TEXT NOT NULL CHECK(state IN ('building','candidate','active','retained','rolled_back','deleted')),
  snapshot_id TEXT NOT NULL REFERENCES vector_generation_snapshots(snapshot_id),
  snapshot_watermark INTEGER NOT NULL,
  catchup_watermark INTEGER NOT NULL,
  snapshot_cursor INTEGER NOT NULL DEFAULT 0,
  predecessor_id TEXT NOT NULL DEFAULT '',
  rollback_expires_at TEXT NOT NULL DEFAULT '',
  protected INTEGER NOT NULL DEFAULT 0 CHECK(protected IN (0,1)),
  version INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_vector_generations_state_retention
  ON vector_generations(state,rollback_expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_vector_generations_single_active
  ON vector_generations(state) WHERE state='active';

CREATE TABLE IF NOT EXISTS vector_generation_active (
  singleton INTEGER PRIMARY KEY CHECK(singleton=1),
  generation_id TEXT NOT NULL DEFAULT '',
  version INTEGER NOT NULL DEFAULT 0
);
INSERT OR IGNORE INTO vector_generation_active(singleton,generation_id,version) VALUES(1,'',0);

CREATE TABLE IF NOT EXISTS vector_generation_points (
  generation_id TEXT NOT NULL REFERENCES vector_generations(generation_id),
  dataset_id TEXT NOT NULL CHECK(dataset_id IN ('journal.facets','forest.summaries')),
  source_id TEXT NOT NULL,
  scope TEXT NOT NULL DEFAULT '',
  vector_blob BLOB NOT NULL,
  source_sequence INTEGER NOT NULL,
  PRIMARY KEY(generation_id,dataset_id,source_id)
);
CREATE INDEX IF NOT EXISTS idx_vector_generation_points_source
  ON vector_generation_points(dataset_id,source_id,generation_id);

CREATE TABLE IF NOT EXISTS vector_generation_operations (
  operation_id TEXT PRIMARY KEY,
  operation TEXT NOT NULL CHECK(operation IN ('stage','catch_up','promote','rollback','collect')),
  generation_id TEXT NOT NULL,
  recipe_id TEXT NOT NULL DEFAULT '',
  request_sha256 TEXT NOT NULL,
  state TEXT NOT NULL CHECK(state IN ('planned','running','complete','failed')),
  receipt_id TEXT NOT NULL DEFAULT '',
  peak_bytes INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT ''
);


