-- ingest domain schema (SQLite)
-- Ingest auditing, replay history, and async ingest jobs.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS ingest_history (
    id TEXT PRIMARY KEY,
    record_id TEXT NOT NULL,
    namespace TEXT NOT NULL,
    collection_name TEXT NOT NULL,
    content_hash TEXT,
    visibility TEXT NOT NULL CHECK (visibility IN ('private', 'shared', 'global')),
    source TEXT,
    source_type TEXT,
    status TEXT NOT NULL CHECK (status IN ('success', 'failure')),
    error_message TEXT,
    took_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ingest_jobs (
    id TEXT PRIMARY KEY,
    request_json TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failure')),
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    total_chunks INTEGER DEFAULT 0,
    completed_chunks INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ingest_history_created ON ingest_history(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_history_namespace ON ingest_history(namespace);
CREATE INDEX IF NOT EXISTS idx_ingest_history_record_id ON ingest_history(record_id);

-- Provenance lookups filter by collection_name; Postgres had no such index and
-- relied on a sequential scan.
CREATE INDEX IF NOT EXISTS idx_ingest_history_collection ON ingest_history(collection_name);

CREATE INDEX IF NOT EXISTS idx_ingest_jobs_created ON ingest_jobs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ingest_jobs_status ON ingest_jobs(status);
