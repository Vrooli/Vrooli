CREATE TABLE IF NOT EXISTS projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  revision INTEGER NOT NULL CHECK (revision > 0)
);
CREATE INDEX IF NOT EXISTS idx_projects_created_at_id ON projects ((substr(created_at,1,19) || '.' || CASE WHEN substr(created_at,20,1)='Z' THEN '000000000' ELSE substr(substr(created_at,21,instr(substr(created_at,21),'Z')-1) || '000000000',1,9) END) DESC, id DESC);

CREATE TABLE IF NOT EXISTS project_source_media (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  sha256 TEXT NOT NULL CHECK (length(sha256) = 64),
  size_bytes INTEGER NOT NULL CHECK (size_bytes > 0),
  media_type TEXT NOT NULL CHECK (media_type IN ('image/png', 'image/jpeg')),
  width INTEGER NOT NULL CHECK (width > 0),
  height INTEGER NOT NULL CHECK (height > 0),
  orientation TEXT NOT NULL,
  color_model TEXT NOT NULL,
  color_profile_status TEXT NOT NULL,
  original_filename TEXT NOT NULL,
  imported_at TEXT NOT NULL,
  import_method TEXT NOT NULL,
  decoder TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  UNIQUE (project_id, sha256)
);
CREATE INDEX IF NOT EXISTS idx_project_source_media_project_imported_id
  ON project_source_media(project_id, imported_at DESC, id DESC);
