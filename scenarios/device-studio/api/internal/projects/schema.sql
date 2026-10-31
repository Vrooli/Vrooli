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
  media_type TEXT NOT NULL CHECK (media_type IN ('image/png', 'image/jpeg', 'video/mp4', 'video/quicktime', 'video/webm', 'video/x-matroska')),
  kind TEXT NOT NULL CHECK (kind IN ('image', 'video')),
  readiness TEXT NOT NULL CHECK (readiness IN ('ready', 'pending', 'rejected', 'reconciliation_required')),
  width INTEGER CHECK (width IS NULL OR width > 0),
  height INTEGER CHECK (height IS NULL OR height > 0),
  orientation TEXT,
  color_model TEXT,
  color_profile_status TEXT,
  inspection_job_id TEXT NOT NULL DEFAULT '',
  inspection_attempt_id TEXT NOT NULL DEFAULT '',
  inspection_issue_code TEXT NOT NULL DEFAULT '',
  inspection_issue_message TEXT NOT NULL DEFAULT '',
  inspection_receipt_json TEXT NOT NULL DEFAULT '',
  inspection_receipt_key TEXT NOT NULL DEFAULT '',
  original_filename TEXT NOT NULL,
  imported_at TEXT NOT NULL,
  import_method TEXT NOT NULL,
  decoder TEXT NOT NULL,
  storage_key TEXT NOT NULL,
  CHECK ((width IS NULL AND height IS NULL) OR (width IS NOT NULL AND height IS NOT NULL AND width > 0 AND height > 0)),
  CHECK ((kind = 'image' AND media_type IN ('image/png', 'image/jpeg') AND readiness = 'ready' AND width IS NOT NULL AND height IS NOT NULL AND width > 0 AND height > 0)
      OR (kind = 'video' AND media_type IN ('video/mp4', 'video/quicktime', 'video/webm', 'video/x-matroska')
          AND ((readiness = 'pending' AND width IS NULL AND height IS NULL)
            OR (readiness = 'ready' AND width IS NOT NULL AND height IS NOT NULL AND width > 0 AND height > 0 AND inspection_receipt_json <> '')
            OR readiness IN ('rejected', 'reconciliation_required')))),
  UNIQUE (project_id, sha256)
);
CREATE INDEX IF NOT EXISTS idx_project_source_media_project_imported_id
  ON project_source_media(project_id, imported_at DESC, id DESC);
