CREATE TABLE IF NOT EXISTS music_catalog_tracks (
 sha256 TEXT PRIMARY KEY, title TEXT NOT NULL, source_path TEXT NOT NULL,
 take_id TEXT NOT NULL, receipt_path TEXT NOT NULL, tags_json TEXT NOT NULL,
 approval TEXT NOT NULL, feedback TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS music_catalog_uses (
 sha256 TEXT NOT NULL, project TEXT NOT NULL, revision TEXT NOT NULL,
 status TEXT NOT NULL CHECK (status IN ('draft','accepted')),
 PRIMARY KEY (sha256, project, revision),
 FOREIGN KEY (sha256) REFERENCES music_catalog_tracks(sha256)
);
CREATE INDEX IF NOT EXISTS music_catalog_uses_project ON music_catalog_uses(project, status);
