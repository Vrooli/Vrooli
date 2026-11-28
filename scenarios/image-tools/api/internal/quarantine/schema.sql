-- Reviewer-refused outputs retained for bounded admin review. This is a
-- separate table from the feedback corpus because it carries blob lifecycle:
-- one row per quarantined file, purged with the blob by the safety janitor.
-- A refused blob lives under `quarantine/<job>/<uuid>.<ext>`, which the ordinary
-- `/api/v1/blobs/` route refuses to serve. Forward-only declarative.
CREATE TABLE IF NOT EXISTS safety_quarantine (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    storage_key   TEXT    NOT NULL UNIQUE,
    job_id        TEXT    NOT NULL,
    original_ref  TEXT    NOT NULL,
    reason_code   TEXT    NOT NULL DEFAULT '',
    safety_review TEXT    NOT NULL DEFAULT '',
    score         REAL    NOT NULL DEFAULT 0,
    evidence      TEXT    NOT NULL DEFAULT '',
    finding       TEXT    NOT NULL DEFAULT '',
    mime          TEXT    NOT NULL DEFAULT '',
    created_at    TEXT    NOT NULL,
    expires_at    TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_safety_quarantine_expires ON safety_quarantine (expires_at);
CREATE INDEX IF NOT EXISTS idx_safety_quarantine_job ON safety_quarantine (job_id);
