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
CREATE INDEX IF NOT EXISTS idx_audit_log_operation ON git_audit_log(operation);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON git_audit_log(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_log_branch ON git_audit_log(branch);
CREATE INDEX IF NOT EXISTS idx_audit_log_op_created ON git_audit_log(operation, created_at DESC);

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

CREATE TABLE IF NOT EXISTS git_repo_precommit (
    repo_path TEXT PRIMARY KEY,
    enabled INTEGER NOT NULL DEFAULT 0,
    command TEXT NOT NULL DEFAULT '',
    working_directory TEXT NOT NULL DEFAULT '',
    timeout_seconds INTEGER NOT NULL DEFAULT 300,
    run_before_commit INTEGER NOT NULL DEFAULT 1,
    allow_override INTEGER NOT NULL DEFAULT 1,
    last_status TEXT,
    last_exit_code INTEGER,
    last_summary TEXT,
    last_stdout TEXT,
    last_stderr TEXT,
    last_duration_ms INTEGER,
    last_timestamp TEXT,
    hook_install_status TEXT,
    hook_install_reason TEXT,
    hook_existing_kind TEXT,
    hook_installed_at TEXT,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS git_commit_check_runs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_path TEXT NOT NULL,
    commit_hash TEXT NOT NULL,
    kind TEXT NOT NULL,
    status TEXT NOT NULL,
    command TEXT NOT NULL DEFAULT '',
    exit_code INTEGER NOT NULL DEFAULT 0,
    summary TEXT NOT NULL DEFAULT '',
    stdout TEXT,
    stderr TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    timestamp TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_commit_check_runs_repo_hash ON git_commit_check_runs(repo_path, commit_hash);
CREATE INDEX IF NOT EXISTS idx_commit_check_runs_repo_created ON git_commit_check_runs(repo_path, created_at DESC);
