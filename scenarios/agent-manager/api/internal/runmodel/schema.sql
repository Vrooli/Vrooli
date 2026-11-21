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
    current_run_id TEXT,
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
    lifecycle_version INTEGER NOT NULL DEFAULT 0,
    owner_identity TEXT NOT NULL DEFAULT '',
    owner_epoch INTEGER NOT NULL DEFAULT 0,
    fresh_recovery_request_hash TEXT NOT NULL DEFAULT '',
    -- Attached operator sessions may intentionally have no task.
    task_id TEXT REFERENCES tasks(id) ON DELETE CASCADE,
    agent_profile_id TEXT,
    tag TEXT,
    label TEXT NOT NULL DEFAULT '',
    label_source TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL DEFAULT '[]',
    owner_subject TEXT NOT NULL DEFAULT '',
    owner_expires_at TEXT,
    owner_scopes TEXT NOT NULL DEFAULT '[]',
    requested_scopes TEXT NOT NULL DEFAULT '[]',
    work_references TEXT NOT NULL DEFAULT '[]',
    workload_kind TEXT NOT NULL DEFAULT 'adhoc',
    workload_key TEXT DEFAULT '',
    workload_instance TEXT DEFAULT '',
    billing_snapshot TEXT DEFAULT '{}',
    sandbox_id TEXT,
    run_mode TEXT DEFAULT 'sandboxed',
    execution_mode TEXT DEFAULT 'codec_pipe',
    harness_kind TEXT DEFAULT '',
    harness_session_id TEXT DEFAULT '',
    goal_delivery TEXT DEFAULT '',
    terminal_class TEXT DEFAULT '',
    stop_reason TEXT DEFAULT '',
    last_handoff TEXT DEFAULT '',
    web_console_session_id TEXT DEFAULT '',
    status TEXT DEFAULT 'pending',
    started_at TEXT,
    interactive_invocation_started_at TEXT,
    ended_at TEXT,
    cancel_requested_at TEXT,
    goal_id TEXT NOT NULL DEFAULT '',
    goal_status TEXT NOT NULL DEFAULT '',
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
    depth INTEGER NOT NULL DEFAULT 0,
    charter_path TEXT NOT NULL DEFAULT '',
    conversation_id TEXT DEFAULT '',
    identity_token_hash TEXT,
    identity_token_revoked_at TEXT,
    identity_renewal_anchor_hash TEXT NOT NULL DEFAULT '',
    identity_token_issued_at TEXT,
    custom_env TEXT,
    await_handle TEXT,
    effort_signal_cursors TEXT NOT NULL DEFAULT '{}',
    last_await_key TEXT DEFAULT '',
    last_await_result TEXT DEFAULT '',
    last_await_resolved_at TEXT,
    last_wake_seq INTEGER DEFAULT 0,
    same_key_park_streak INTEGER DEFAULT 0,
	requested_model TEXT DEFAULT '',
	actual_model TEXT DEFAULT '',
	canary_arm TEXT DEFAULT '',
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);

-- Accepted creation identity is not a TTL cache or a child of the run row.
-- Deleting/purging a run or task must never make its creation key reusable.
CREATE TABLE IF NOT EXISTS run_creation_receipts (
    idempotency_key TEXT PRIMARY KEY,
    run_id TEXT NOT NULL UNIQUE,
    task_id TEXT,
    owner_subject TEXT,
    created_at TEXT NOT NULL
);

-- Additive, idempotent adoption of already-persisted runs on schema application.
INSERT INTO run_creation_receipts (idempotency_key, run_id, task_id, owner_subject, created_at)
SELECT idempotency_key, id, task_id, owner_subject, COALESCE(created_at, datetime('now'))
FROM runs WHERE idempotency_key IS NOT NULL AND idempotency_key <> ''
ON CONFLICT(idempotency_key) DO NOTHING;
CREATE INDEX IF NOT EXISTS idx_runs_session_id ON runs(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_tag ON runs(tag);
CREATE INDEX IF NOT EXISTS idx_runs_source_investigation_run_id ON runs(source_investigation_run_id);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at);
CREATE INDEX IF NOT EXISTS idx_runs_parent_run_id ON runs(parent_run_id) WHERE parent_run_id IS NOT NULL;

-- Stats query indexes
CREATE INDEX IF NOT EXISTS idx_runs_created_status ON runs(created_at, status);
CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_import_provenance ON runs(import_source_harness, import_source_session_id) WHERE import_source_harness <> '' AND import_source_session_id <> '';
CREATE INDEX IF NOT EXISTS idx_runs_import_session ON runs(import_source_session_id) WHERE import_source_session_id <> '';
-- Authorization peer binding scans only nonterminal runs; keep that owner
-- lookup from traversing the retained terminal history.
CREATE INDEX IF NOT EXISTS idx_runs_status_created_at_id ON runs(status, created_at DESC, id DESC);
-- The reconciler resolves web-console sessions back to their runs.
CREATE INDEX IF NOT EXISTS idx_runs_web_console_session ON runs(web_console_session_id) WHERE web_console_session_id <> '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_runs_identity_renewal_anchor ON runs(identity_renewal_anchor_hash) WHERE identity_renewal_anchor_hash <> '';
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
-- Investigation Settings - Global investigation configuration (singleton table)
-- ============================================================================
CREATE TABLE IF NOT EXISTS investigation_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    default_depth TEXT NOT NULL DEFAULT 'standard',
    default_context TEXT NOT NULL DEFAULT '{}',
    investigation_tag_allowlist TEXT NOT NULL DEFAULT '[]',

    updated_at TEXT DEFAULT (datetime('now'))
);


-- durability_boundary is the stored analysis epoch. Durability grades nothing
-- before it, because run attribution was not captured before it. The row is
-- written once, on first read, and never silently recomputed: an epoch that
-- differed per process or per environment would make a deliberate scope limit
-- look like missing data.
CREATE TABLE IF NOT EXISTS durability_boundary (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    epoch TEXT NOT NULL,
    reason TEXT NOT NULL,
    established_at TEXT NOT NULL DEFAULT (datetime('now')),
    established_by TEXT NOT NULL DEFAULT ''
);


CREATE TRIGGER IF NOT EXISTS update_tasks_updated_at
    AFTER UPDATE ON tasks
    FOR EACH ROW
BEGIN
    UPDATE tasks SET updated_at = datetime('now') WHERE id = NEW.id;
END;

-- Run repositories own the committed nanosecond revision. The former trigger
-- replaced every explicit revision with second-precision wall time, making
-- HTTP and notification ordering ambiguous. Retire it on existing databases
-- as well as omitting it for fresh ones; rows and their history are preserved.
DROP TRIGGER IF EXISTS update_runs_updated_at;

-- Prospective terminal observations. No backfill; no cascade from mutable run
-- or task rows. Capture is disabled by the normal repository constructor.
CREATE TABLE IF NOT EXISTS run_terminal_receipt_source (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    source_instance TEXT NOT NULL,
    source_epoch TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS run_terminal_receipts (
    run_id TEXT NOT NULL,
    lifecycle_version INTEGER NOT NULL CHECK (lifecycle_version > 0),
    source_instance TEXT NOT NULL,
    source_epoch TEXT NOT NULL,
    observation_json TEXT NOT NULL,
    observation_sha256 TEXT NOT NULL,
    event_intent_json TEXT NOT NULL,
    event_intent_id TEXT NOT NULL UNIQUE,
    PRIMARY KEY (run_id, lifecycle_version)
);
CREATE TRIGGER IF NOT EXISTS run_terminal_receipts_no_update
BEFORE UPDATE ON run_terminal_receipts BEGIN
    SELECT RAISE(ABORT, 'terminal receipts are immutable');
END;
CREATE TRIGGER IF NOT EXISTS run_terminal_receipts_no_delete
BEFORE DELETE ON run_terminal_receipts BEGIN
    SELECT RAISE(ABORT, 'terminal receipts are immutable');
END;
CREATE TRIGGER IF NOT EXISTS run_terminal_receipt_source_no_update
BEFORE UPDATE ON run_terminal_receipt_source BEGIN
    SELECT RAISE(ABORT, 'terminal receipt source is immutable');
END;
CREATE TRIGGER IF NOT EXISTS run_terminal_receipt_source_no_delete
BEFORE DELETE ON run_terminal_receipt_source BEGIN
    SELECT RAISE(ABORT, 'terminal receipt source is immutable');
END;

-- Separate from immutable terminal observations: only durable Events acceptance
-- is retained here. No cascade from mutable runs or tasks, no implicit dispatch.
CREATE TABLE IF NOT EXISTS run_terminal_event_acknowledgments (
    event_intent_id TEXT PRIMARY KEY REFERENCES run_terminal_receipts(event_intent_id),
    acknowledgment_json TEXT NOT NULL
);
CREATE TRIGGER IF NOT EXISTS run_terminal_event_acknowledgments_no_update
BEFORE UPDATE ON run_terminal_event_acknowledgments BEGIN
    SELECT RAISE(ABORT, 'terminal event acknowledgments are immutable');
END;
CREATE TRIGGER IF NOT EXISTS run_terminal_event_acknowledgments_no_delete
BEFORE DELETE ON run_terminal_event_acknowledgments BEGIN
    SELECT RAISE(ABORT, 'terminal event acknowledgments are immutable');
END;

-- SQLite REPLACE can bypass DELETE triggers when recursive_triggers is off.
-- Guard conflicting INSERTs as well, preserving exact no-op acknowledgments.
CREATE TRIGGER IF NOT EXISTS run_terminal_receipts_no_replace
BEFORE INSERT ON run_terminal_receipts
WHEN EXISTS (SELECT 1 FROM run_terminal_receipts WHERE
    (run_id=NEW.run_id AND lifecycle_version=NEW.lifecycle_version)
    OR event_intent_id=NEW.event_intent_id)
BEGIN SELECT RAISE(ABORT, 'terminal receipts cannot be replaced'); END;
CREATE TRIGGER IF NOT EXISTS run_terminal_receipt_source_no_replace
BEFORE INSERT ON run_terminal_receipt_source
WHEN EXISTS (SELECT 1 FROM run_terminal_receipt_source WHERE singleton=NEW.singleton
    AND (source_instance!=NEW.source_instance OR source_epoch!=NEW.source_epoch))
BEGIN SELECT RAISE(ABORT, 'terminal receipt source cannot be replaced'); END;
CREATE TRIGGER IF NOT EXISTS run_terminal_event_acknowledgments_no_replace
BEFORE INSERT ON run_terminal_event_acknowledgments
WHEN EXISTS (SELECT 1 FROM run_terminal_event_acknowledgments
    WHERE event_intent_id=NEW.event_intent_id AND acknowledgment_json!=NEW.acknowledgment_json)
BEGIN SELECT RAISE(ABORT, 'terminal event acknowledgments cannot be replaced'); END;
