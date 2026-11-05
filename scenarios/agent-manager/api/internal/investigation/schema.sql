CREATE TABLE IF NOT EXISTS investigations (
    investigation_id TEXT PRIMARY KEY,
    caller_authority TEXT NOT NULL DEFAULT '',
    request_key TEXT NOT NULL,
    request_digest TEXT NOT NULL,
    request_json TEXT NOT NULL,
    operation_status TEXT NOT NULL,
    result_json TEXT NOT NULL DEFAULT '',
    source_cut_json TEXT NOT NULL DEFAULT '',
    workflow_ref TEXT NOT NULL DEFAULT '',
    cancel_requested INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    completed_at TEXT,
    cancelled_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_investigations_caller_key ON investigations(caller_authority, request_key);
CREATE INDEX IF NOT EXISTS idx_investigations_status_updated ON investigations(operation_status, updated_at);
CREATE INDEX IF NOT EXISTS idx_investigations_created ON investigations(created_at, investigation_id);
