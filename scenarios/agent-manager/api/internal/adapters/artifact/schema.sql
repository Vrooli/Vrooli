CREATE TABLE IF NOT EXISTS artifacts (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    artifact_type TEXT NOT NULL,
    name TEXT NOT NULL,
    storage_path TEXT NOT NULL UNIQUE,
    content_size INTEGER NOT NULL,
    content_type TEXT NOT NULL DEFAULT '',
    checksum TEXT NOT NULL DEFAULT '',
    metadata BLOB NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_artifacts_run_created ON artifacts(run_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_artifacts_created ON artifacts(created_at);
