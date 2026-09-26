CREATE TABLE IF NOT EXISTS retrieval_units (
  id TEXT PRIMARY KEY,
  collection_id TEXT NOT NULL,
  document_hash TEXT NOT NULL,
  privacy_class INTEGER NOT NULL,
  text TEXT NOT NULL,
  anchor_uri TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS retrieval_fts USING fts5(
  unit_id UNINDEXED,
  text
);

CREATE TABLE IF NOT EXISTS retrieval_vectors (
  unit_id TEXT PRIMARY KEY,
  dimension INTEGER NOT NULL,
  vector BLOB NOT NULL,
  FOREIGN KEY(unit_id) REFERENCES retrieval_units(id)
);
