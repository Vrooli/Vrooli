-- metadata domain schema (SQLite)
-- Knowledge entry metadata cached from Qdrant, and external ID mappings for idempotency.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS knowledge_metadata (
    id TEXT PRIMARY KEY,
    vector_id TEXT UNIQUE NOT NULL,
    collection_name TEXT NOT NULL,
    content_hash TEXT,
    source_scenario TEXT,
    source_type TEXT,
    quality_score REAL,
    access_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS external_id_map (
    id TEXT PRIMARY KEY,
    namespace TEXT NOT NULL,
    external_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('record', 'document')),
    record_id TEXT,
    document_id TEXT,
    content_hash TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(namespace, external_id, kind)
);

CREATE INDEX IF NOT EXISTS idx_knowledge_metadata_collection ON knowledge_metadata(collection_name);
CREATE INDEX IF NOT EXISTS idx_knowledge_metadata_source ON knowledge_metadata(source_scenario);

CREATE INDEX IF NOT EXISTS idx_external_id_map_namespace ON external_id_map(namespace);
CREATE INDEX IF NOT EXISTS idx_external_id_map_external_id ON external_id_map(external_id);

CREATE TRIGGER IF NOT EXISTS trg_knowledge_metadata_updated_at
AFTER UPDATE ON knowledge_metadata
FOR EACH ROW BEGIN
    UPDATE knowledge_metadata SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS trg_external_id_map_updated_at
AFTER UPDATE ON external_id_map
FOR EACH ROW BEGIN
    UPDATE external_id_map SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
