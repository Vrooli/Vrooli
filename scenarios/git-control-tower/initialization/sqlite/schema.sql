-- [REQ:GCT-OT-P0-007] SQLite audit logging
-- Creates the git_audit_log table for tracking mutating operations

CREATE TABLE IF NOT EXISTS git_audit_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    operation TEXT NOT NULL,
    repo_dir TEXT NOT NULL,
    branch TEXT,
    paths TEXT,
    commit_hash TEXT,
    commit_message TEXT,
    success INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    metadata TEXT
);

-- Index for querying by operation type
CREATE INDEX IF NOT EXISTS idx_audit_log_operation ON git_audit_log(operation);

-- Index for querying by timestamp (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON git_audit_log(created_at DESC);

-- Index for querying by branch
CREATE INDEX IF NOT EXISTS idx_audit_log_branch ON git_audit_log(branch);

-- Composite index for common filter combinations
CREATE INDEX IF NOT EXISTS idx_audit_log_op_created ON git_audit_log(operation, created_at DESC);

-- Repository registry for repo switching
CREATE TABLE IF NOT EXISTS git_repos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    remote_url TEXT,
    added_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_opened_at TEXT,
    is_favorite INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_git_repos_last_opened ON git_repos(last_opened_at DESC);
CREATE INDEX IF NOT EXISTS idx_git_repos_added_at ON git_repos(added_at DESC);

CREATE TABLE IF NOT EXISTS git_repo_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL
);
