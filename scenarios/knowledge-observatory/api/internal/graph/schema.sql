-- graph domain schema (SQLite)
-- Knowledge relationships for graph visualization.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS knowledge_relationships (
    id TEXT PRIMARY KEY,
    source_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    relationship_type TEXT NOT NULL,
    weight REAL DEFAULT 0.5,
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, target_id, relationship_type)
);

CREATE INDEX IF NOT EXISTS idx_relationships_source ON knowledge_relationships(source_id);
CREATE INDEX IF NOT EXISTS idx_relationships_target ON knowledge_relationships(target_id);
