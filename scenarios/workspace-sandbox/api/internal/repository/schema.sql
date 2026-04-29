-- Workspace Sandbox SQLite Schema
--
-- Single canonical schema for the embedded SQLite store. Idempotent so it
-- can be applied on every API startup.
--
-- Type mapping note:
--   UUID values are stored as canonical 36-character TEXT, generated in Go.
--   Timestamps are stored as RFC3339Nano UTC TEXT.
--   JSON objects (metadata, behavior, audit details, sandbox_state) are
--     stored as TEXT containing UTF-8 JSON.
--   String/int arrays (tags, reserved_paths, active_pids) are stored as TEXT
--     containing a JSON array; helpers in api/internal/repository/sqlite_codec.go
--     marshal/unmarshal them.
--   Booleans are stored as INTEGER (0/1).
--
-- Canonical enum values for run_outcome and provenance_state live in
-- packages/proto/schemas/workspace-sandbox/v1/domain/applied_change.proto.

CREATE TABLE IF NOT EXISTS sandboxes (
    id                  TEXT PRIMARY KEY,
    name                TEXT,
    scope_path          TEXT NOT NULL,
    reserved_path       TEXT,
    reserved_paths      TEXT NOT NULL DEFAULT '[]',
    no_lock             INTEGER NOT NULL DEFAULT 0,
    project_root        TEXT NOT NULL,
    owner               TEXT,
    owner_type          TEXT NOT NULL DEFAULT 'user',
    status              TEXT NOT NULL DEFAULT 'creating',
    error_message       TEXT,
    created_at          TEXT NOT NULL,
    last_used_at        TEXT NOT NULL,
    stopped_at          TEXT,
    approved_at         TEXT,
    deleted_at          TEXT,
    driver_id           TEXT NOT NULL DEFAULT 'overlayfs-userns',
    driver_version      TEXT NOT NULL DEFAULT '1.0',
    lower_dir           TEXT,
    upper_dir           TEXT,
    work_dir            TEXT,
    merged_dir          TEXT,
    size_bytes          INTEGER NOT NULL DEFAULT 0,
    file_count          INTEGER NOT NULL DEFAULT 0,
    active_pids         TEXT NOT NULL DEFAULT '[]',
    session_count       INTEGER NOT NULL DEFAULT 0,
    tags                TEXT NOT NULL DEFAULT '[]',
    metadata            TEXT NOT NULL DEFAULT '{}',
    behavior            TEXT NOT NULL DEFAULT '{}',
    idempotency_key     TEXT UNIQUE,
    version             INTEGER NOT NULL DEFAULT 1,
    updated_at          TEXT NOT NULL,
    base_commit_hash    TEXT,
    home_overlay_state  TEXT NOT NULL DEFAULT 'absent',
    CHECK (scope_path != '' AND substr(scope_path, 1, 1) = '/'),
    CHECK (project_root != '' AND substr(project_root, 1, 1) = '/'),
    CHECK (status IN ('creating', 'active', 'stopped', 'approved', 'rejected', 'deleted', 'error')),
    CHECK (home_overlay_state IN ('present', 'absent', 'not_requested', 'unsupported'))
);

CREATE TABLE IF NOT EXISTS sandbox_changes (
    id                  TEXT PRIMARY KEY,
    sandbox_id          TEXT NOT NULL REFERENCES sandboxes(id) ON DELETE CASCADE,
    file_path           TEXT NOT NULL,
    change_type         TEXT NOT NULL,
    file_size           INTEGER NOT NULL DEFAULT 0,
    file_mode           INTEGER NOT NULL DEFAULT 0,
    detected_at         TEXT NOT NULL,
    approval_status     TEXT NOT NULL DEFAULT 'pending',
    approved_at         TEXT,
    approved_by         TEXT,
    CHECK (change_type IN ('added', 'modified', 'deleted'))
);

CREATE TABLE IF NOT EXISTS sandbox_audit_log (
    id                  TEXT PRIMARY KEY,
    sandbox_id          TEXT REFERENCES sandboxes(id) ON DELETE SET NULL,
    event_type          TEXT NOT NULL,
    event_time          TEXT NOT NULL,
    actor               TEXT,
    actor_type          TEXT NOT NULL DEFAULT 'system',
    details             TEXT NOT NULL DEFAULT '{}',
    sandbox_state       TEXT
);

CREATE TABLE IF NOT EXISTS applied_changes (
    id                    TEXT PRIMARY KEY,
    sandbox_id            TEXT REFERENCES sandboxes(id) ON DELETE SET NULL,
    sandbox_owner         TEXT,
    sandbox_owner_type    TEXT,
    file_path             TEXT NOT NULL,
    project_root          TEXT NOT NULL,
    change_type           TEXT NOT NULL,
    file_size             INTEGER NOT NULL DEFAULT 0,
    applied_at            TEXT NOT NULL,
    committed_at          TEXT,
    commit_hash           TEXT,
    commit_message        TEXT,
    agent_manager_run_id  TEXT,
    run_outcome           TEXT,
    provenance_state      TEXT,
    conversation_id       TEXT,
    cost_usd              REAL,
    CHECK (change_type IN ('added', 'modified', 'deleted'))
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_sandboxes_status         ON sandboxes(status);
CREATE INDEX IF NOT EXISTS idx_sandboxes_owner          ON sandboxes(owner);
CREATE INDEX IF NOT EXISTS idx_sandboxes_scope_path     ON sandboxes(scope_path);
CREATE INDEX IF NOT EXISTS idx_sandboxes_reserved_path  ON sandboxes(reserved_path);
CREATE INDEX IF NOT EXISTS idx_sandboxes_project_root   ON sandboxes(project_root);
CREATE INDEX IF NOT EXISTS idx_sandboxes_created_at     ON sandboxes(created_at);
CREATE INDEX IF NOT EXISTS idx_sandboxes_last_used_at   ON sandboxes(last_used_at);
CREATE INDEX IF NOT EXISTS idx_sandboxes_active         ON sandboxes(status) WHERE status IN ('creating', 'active');
CREATE INDEX IF NOT EXISTS idx_sandboxes_idempotency_key ON sandboxes(idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_sandbox_changes_sandbox_id ON sandbox_changes(sandbox_id);
CREATE INDEX IF NOT EXISTS idx_sandbox_changes_approval   ON sandbox_changes(approval_status);

CREATE INDEX IF NOT EXISTS idx_sandbox_audit_log_sandbox_id ON sandbox_audit_log(sandbox_id);
CREATE INDEX IF NOT EXISTS idx_sandbox_audit_log_event_type ON sandbox_audit_log(event_type);
CREATE INDEX IF NOT EXISTS idx_sandbox_audit_log_event_time ON sandbox_audit_log(event_time);

CREATE INDEX IF NOT EXISTS idx_applied_changes_sandbox_id      ON applied_changes(sandbox_id);
CREATE INDEX IF NOT EXISTS idx_applied_changes_file_path       ON applied_changes(file_path);
CREATE INDEX IF NOT EXISTS idx_applied_changes_project_root    ON applied_changes(project_root);
CREATE INDEX IF NOT EXISTS idx_applied_changes_pending         ON applied_changes(committed_at) WHERE committed_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_applied_changes_run_id          ON applied_changes(agent_manager_run_id);
CREATE INDEX IF NOT EXISTS idx_applied_changes_conversation_id ON applied_changes(conversation_id) WHERE conversation_id IS NOT NULL;

-- last_used_at maintenance: the repository layer writes last_used_at
-- explicitly on every UPDATE that touches status/active_pids. We do not use
-- a trigger so the value format stays in lockstep with the RFC3339Nano
-- format Go produces; SQLite's strftime drops trailing zeros differently
-- and would yield mixed formats in the same column.

-- heal_state: per-sandbox automatic-mount-heal failure history. Survives
-- API restart so the loop-bomb (5 consecutive failures → mark Error) is
-- not silently reset every reboot. Cleared on successful heal or
-- sandbox delete. One row per sandbox; absence ≡ "no failures".
--
-- Round 3 (2026-04-29): introduced as the durable backing store for
-- heal_state in internal/sandbox/heal.go.
CREATE TABLE IF NOT EXISTS heal_state (
    sandbox_id            TEXT PRIMARY KEY,
    consecutive_failures  INTEGER NOT NULL DEFAULT 0,
    last_attempt          TEXT NOT NULL,
    last_error            TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (sandbox_id) REFERENCES sandboxes(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_heal_state_failures ON heal_state(consecutive_failures);
