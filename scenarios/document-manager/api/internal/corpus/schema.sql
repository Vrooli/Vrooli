CREATE TABLE IF NOT EXISTS collections (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL UNIQUE,
  default_privacy_class INTEGER NOT NULL,
  federated INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS collection_documents (
  collection_id TEXT NOT NULL,
  document_hash TEXT NOT NULL,
  privacy_class INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(collection_id, document_hash),
  FOREIGN KEY(collection_id) REFERENCES collections(id)
);

CREATE TABLE IF NOT EXISTS collection_anchors (
  collection_id TEXT NOT NULL,
  anchor_uri TEXT NOT NULL,
  PRIMARY KEY(collection_id, anchor_uri),
  FOREIGN KEY(collection_id) REFERENCES collections(id)
);
