-- Relationships are metadata only. Drafts, revisions and attachments remain
-- owned by Artifacts; these soft references never consume draft slots.
CREATE TABLE IF NOT EXISTS content_relationships (
  id TEXT PRIMARY KEY,
  source_draft_id TEXT NOT NULL,
  source_revision_id TEXT NOT NULL,
  target_draft_id TEXT NOT NULL,
  target_revision_id TEXT NOT NULL,
  project_id TEXT NOT NULL,
  relation_kind TEXT NOT NULL CHECK (relation_kind IN ('related', 'family')),
  is_primary INTEGER NOT NULL DEFAULT 0 CHECK (is_primary IN (0, 1)),
  created_at TEXT NOT NULL,
  CHECK (source_draft_id <> target_draft_id),
  CHECK (relation_kind = 'family' OR is_primary = 0)
);
CREATE INDEX IF NOT EXISTS content_relationships_source_idx ON content_relationships(source_draft_id, created_at);
CREATE INDEX IF NOT EXISTS content_relationships_target_idx ON content_relationships(target_draft_id, created_at);
CREATE UNIQUE INDEX IF NOT EXISTS content_relationships_primary_family_idx
  ON content_relationships(source_draft_id) WHERE relation_kind = 'family' AND is_primary = 1;

CREATE TABLE IF NOT EXISTS saved_collections (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  UNIQUE(project_id, name)
);
CREATE TABLE IF NOT EXISTS saved_collection_items (
  collection_id TEXT NOT NULL REFERENCES saved_collections(id),
  draft_id TEXT NOT NULL,
  revision_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(collection_id, draft_id, revision_id)
);
CREATE INDEX IF NOT EXISTS saved_collection_items_draft_idx ON saved_collection_items(draft_id, revision_id);
