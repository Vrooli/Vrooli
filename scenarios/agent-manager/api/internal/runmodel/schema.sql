-- ============================================================================
-- Tasks - Defines WHAT needs to be done
-- ============================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    scope_path TEXT NOT NULL,
    project_root TEXT,
    phase_prompt_ids TEXT DEFAULT '[]',
    context_attachments TEXT DEFAULT '[]',
    status TEXT DEFAULT 'queued',
    created_by TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at);

-- ============================================================================
-- Runs - Concrete execution attempts
-- ============================================================================
CREATE TABLE IF NOT EXISTS runs (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent_profile_id TEXT,
    tag TEXT,
    sandbox_id TEXT,
    run_mode TEXT DEFAULT 'sandboxed',
    execution_mode TEXT DEFAULT 'codec_pipe',
    web_console_session_id TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',
    started_at TEXT,
    ended_at TEXT,
    phase TEXT DEFAULT 'queued',
    last_checkpoint_id TEXT,
    last_heartbeat TEXT,
    progress_percent INTEGER DEFAULT 0,
    idempotency_key TEXT UNIQUE,
    summary TEXT,
    run_result TEXT,
    error_msg TEXT,
    exit_code INTEGER,
    approval_state TEXT DEFAULT 'none',
    approved_by TEXT,
    approved_at TEXT,
    finalization_status TEXT DEFAULT 'none',
    finalization_error TEXT DEFAULT '',
    finalized_at TEXT,
    resolved_config TEXT,
    diff_path TEXT,
    log_path TEXT,
    changed_files INTEGER DEFAULT 0,
    total_size_bytes INTEGER DEFAULT 0,
	commit_hash TEXT DEFAULT '',
    sandbox_config TEXT DEFAULT '{}',
    session_id TEXT,
    runner_pid INTEGER DEFAULT 0,
    runner_pgid INTEGER DEFAULT 0,
    transcript_path TEXT DEFAULT '',
    transcript_cursor INTEGER DEFAULT 0,
    transcript_last_seq INTEGER DEFAULT 0,
    import_source_harness TEXT DEFAULT '',
    import_source_session_id TEXT DEFAULT '',
    imported_at TEXT,
    source_run_ids TEXT DEFAULT '[]',
    source_investigation_run_id TEXT,
    parent_run_id TEXT,
    conversation_id TEXT DEFAULT '',
    identity_token_hash TEXT,
    identity_token_revoked_at TEXT,
    custom_env TEXT,
    await_handle TEXT,
    last_await_key TEXT DEFAULT '',
    last_await_result TEXT DEFAULT '',
    last_await_resolved_at TEXT,
    last_wake_seq INTEGER DEFAULT 0,
    same_key_park_streak INTEGER DEFAULT 0,
    requested_model TEXT DEFAULT '',
    actual_model TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_session_id ON runs(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_tag ON runs(tag);
CREATE INDEX IF NOT EXISTS idx_runs_source_investigation_run_id ON runs(source_investigation_run_id);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);

-- Stats query indexes
CREATE INDEX IF NOT EXISTS idx_runs_created_status ON runs(created_at, status);
-- ============================================================================
-- Run Checkpoints - For resumption after interruption
-- ============================================================================
CREATE TABLE IF NOT EXISTS run_checkpoints (
    run_id TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
    phase TEXT NOT NULL,
    step_within_phase INTEGER DEFAULT 0,
    sandbox_id TEXT,
    work_dir TEXT,
    lock_id TEXT,
    last_event_sequence INTEGER DEFAULT 0,
    last_heartbeat TEXT DEFAULT (datetime('now')),
    retry_count INTEGER DEFAULT 0,
    saved_at TEXT DEFAULT (datetime('now')),
    metadata TEXT DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_run_checkpoints_heartbeat ON run_checkpoints(last_heartbeat);

-- ============================================================================
-- Idempotency Records - For replay-safe operations
-- ============================================================================
CREATE TABLE IF NOT EXISTS idempotency_records (
    key TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    entity_id TEXT,
    entity_type TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL,
    response TEXT
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_records(expires_at);
CREATE INDEX IF NOT EXISTS idx_idempotency_status ON idempotency_records(status);
-- ============================================================================
-- Scope Locks - Concurrency control
-- ============================================================================
CREATE TABLE IF NOT EXISTS scope_locks (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    scope_path TEXT NOT NULL,
    project_root TEXT,
    acquired_at TEXT DEFAULT (datetime('now')),
    expires_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scope_locks_run_id ON scope_locks(run_id);
CREATE INDEX IF NOT EXISTS idx_scope_locks_scope ON scope_locks(scope_path, project_root);
CREATE INDEX IF NOT EXISTS idx_scope_locks_expires ON scope_locks(expires_at);
-- ============================================================================
-- Investigation Settings - Global investigation configuration (singleton table)
-- ============================================================================
CREATE TABLE IF NOT EXISTS investigation_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    default_depth TEXT NOT NULL DEFAULT 'standard',
    default_context TEXT NOT NULL DEFAULT '{}',
    investigation_tag_allowlist TEXT NOT NULL DEFAULT '[]',

    updated_at TEXT DEFAULT (datetime('now'))
);


CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
    AFTER UPDATE ON tasks
    FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_runs_updated_at
    AFTER UPDATE ON runs
    FOR EACH ROW
BEGIN
    UPDATE runs SET updated_at = datetime('now') WHERE id = NEW.id;
END;
