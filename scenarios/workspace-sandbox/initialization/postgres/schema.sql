-- Workspace Sandbox PostgreSQL Schema
-- Provides persistent metadata storage for sandbox lifecycle management

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Sandbox status enum
CREATE TYPE sandbox_status AS ENUM (
    'creating',   -- Sandbox is being initialized
    'active',     -- Sandbox is mounted and ready
    'stopped',    -- Sandbox is unmounted but preserved
    'approved',   -- Changes have been approved and applied
    'rejected',   -- Changes have been rejected
    'deleted',    -- Sandbox has been deleted (soft delete)
    'error'       -- Sandbox is in an error state
);

-- Main sandboxes table
CREATE TABLE sandboxes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),

    -- Scope configuration
    scope_path TEXT NOT NULL,           -- Normalized absolute path within project
    project_root TEXT NOT NULL,         -- Project root directory

    -- Ownership and attribution
    owner TEXT,                         -- Agent/user/task identifier
    owner_type TEXT DEFAULT 'user',     -- 'agent', 'user', 'task', 'system'

    -- Status tracking
    status sandbox_status NOT NULL DEFAULT 'creating',
    error_message TEXT,                 -- Error details when status='error'

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    -- Driver configuration
    driver TEXT NOT NULL DEFAULT 'overlayfs',
    driver_version TEXT NOT NULL DEFAULT '1.0',

    -- Mount paths (populated after successful mount)
    lower_dir TEXT,      -- Read-only layer (canonical repo)
    upper_dir TEXT,      -- Writable layer (changes)
    work_dir TEXT,       -- Overlayfs work directory
    merged_dir TEXT,     -- Merged mount point

    -- Size accounting
    size_bytes BIGINT DEFAULT 0,
    file_count INT DEFAULT 0,

    -- Session tracking
    active_pids INTEGER[] DEFAULT '{}',
    session_count INT DEFAULT 0,

    -- Metadata
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',

    -- Idempotency support
    idempotency_key TEXT UNIQUE,        -- Optional client-provided key for idempotent creates

    -- Optimistic locking
    version BIGINT NOT NULL DEFAULT 1,  -- Version number for optimistic concurrency control
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Conflict detection (OT-P2-002)
    base_commit_hash TEXT,              -- Git commit hash at sandbox creation for conflict detection

    -- Constraints
    CONSTRAINT valid_scope_path CHECK (scope_path != '' AND scope_path ~ '^/'),
    CONSTRAINT valid_project_root CHECK (project_root != '' AND project_root ~ '^/')
);

-- Changed files tracking
CREATE TABLE sandbox_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sandbox_id UUID NOT NULL REFERENCES sandboxes(id) ON DELETE CASCADE,

    -- File information
    file_path TEXT NOT NULL,            -- Path relative to scope
    change_type TEXT NOT NULL,          -- 'added', 'modified', 'deleted'
    file_size BIGINT DEFAULT 0,
    file_mode INT DEFAULT 0,

    -- Timestamps
    detected_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Approval status
    approval_status TEXT DEFAULT 'pending', -- 'pending', 'approved', 'rejected'
    approved_at TIMESTAMPTZ,
    approved_by TEXT,

    CONSTRAINT valid_change_type CHECK (change_type IN ('added', 'modified', 'deleted'))
);

-- Audit log for sandbox operations
CREATE TABLE sandbox_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sandbox_id UUID REFERENCES sandboxes(id) ON DELETE SET NULL,

    -- Event information
    event_type TEXT NOT NULL,           -- 'created', 'mounted', 'stopped', 'approved', 'rejected', 'deleted', 'gc'
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Actor
    actor TEXT,
    actor_type TEXT DEFAULT 'system',

    -- Details
    details JSONB DEFAULT '{}',

    -- Snapshot of sandbox state at event time
    sandbox_state JSONB
);

-- Track files applied from sandboxes (provenance tracking)
-- Records which sandbox modified which files, enabling:
-- - Attribution of changes to specific sandboxes/agents
-- - Batch committing of pending changes
-- - Audit trail of file modifications
CREATE TABLE applied_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sandbox_id UUID REFERENCES sandboxes(id) ON DELETE SET NULL,

    -- Original sandbox ownership (preserved even if sandbox deleted)
    sandbox_owner TEXT,
    sandbox_owner_type TEXT,

    -- File information
    file_path TEXT NOT NULL,            -- Absolute path to the file
    project_root TEXT NOT NULL,         -- Project root directory
    change_type TEXT NOT NULL,          -- 'added', 'modified', 'deleted'
    file_size BIGINT DEFAULT 0,

    -- Timestamps
    applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Commit tracking (null until committed)
    committed_at TIMESTAMPTZ,
    commit_hash TEXT,
    commit_message TEXT,

    CONSTRAINT valid_applied_change_type CHECK (change_type IN ('added', 'modified', 'deleted'))
);

-- Indexes for common queries
CREATE INDEX idx_sandboxes_status ON sandboxes(status);
CREATE INDEX idx_sandboxes_owner ON sandboxes(owner);
CREATE INDEX idx_sandboxes_scope_path ON sandboxes(scope_path);
CREATE INDEX idx_sandboxes_project_root ON sandboxes(project_root);
CREATE INDEX idx_sandboxes_created_at ON sandboxes(created_at);
CREATE INDEX idx_sandboxes_last_used_at ON sandboxes(last_used_at);
CREATE INDEX idx_sandboxes_active ON sandboxes(status) WHERE status IN ('creating', 'active');
CREATE INDEX idx_sandboxes_idempotency_key ON sandboxes(idempotency_key) WHERE idempotency_key IS NOT NULL;

CREATE INDEX idx_sandbox_changes_sandbox_id ON sandbox_changes(sandbox_id);
CREATE INDEX idx_sandbox_changes_approval ON sandbox_changes(approval_status);

CREATE INDEX idx_sandbox_audit_log_sandbox_id ON sandbox_audit_log(sandbox_id);
CREATE INDEX idx_sandbox_audit_log_event_type ON sandbox_audit_log(event_type);
CREATE INDEX idx_sandbox_audit_log_event_time ON sandbox_audit_log(event_time);

CREATE INDEX idx_applied_changes_sandbox_id ON applied_changes(sandbox_id);
CREATE INDEX idx_applied_changes_file_path ON applied_changes(file_path);
CREATE INDEX idx_applied_changes_project_root ON applied_changes(project_root);
CREATE INDEX idx_applied_changes_pending ON applied_changes(committed_at) WHERE committed_at IS NULL;

-- Function to check for overlapping scope paths
CREATE OR REPLACE FUNCTION check_scope_overlap(
    new_scope TEXT,
    new_project TEXT,
    exclude_id UUID DEFAULT NULL
) RETURNS TABLE(id UUID, scope_path TEXT, status sandbox_status) AS $$
BEGIN
    RETURN QUERY
    SELECT s.id, s.scope_path, s.status
    FROM sandboxes s
    WHERE s.project_root = new_project
      AND s.status IN ('creating', 'active')
      AND (exclude_id IS NULL OR s.id != exclude_id)
      AND (
          -- new_scope is ancestor of existing scope
          s.scope_path LIKE new_scope || '/%'
          OR s.scope_path = new_scope
          -- existing scope is ancestor of new_scope
          OR new_scope LIKE s.scope_path || '/%'
      );
END;
$$ LANGUAGE plpgsql;

-- Trigger to update last_used_at on sandbox access
CREATE OR REPLACE FUNCTION update_sandbox_last_used()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_used_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER sandbox_last_used_trigger
    BEFORE UPDATE ON sandboxes
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status
          OR OLD.active_pids IS DISTINCT FROM NEW.active_pids)
    EXECUTE FUNCTION update_sandbox_last_used();

-- Function to get aggregate sandbox statistics
CREATE OR REPLACE FUNCTION get_sandbox_stats()
RETURNS TABLE(
    total_count BIGINT,
    active_count BIGINT,
    stopped_count BIGINT,
    error_count BIGINT,
    approved_count BIGINT,
    rejected_count BIGINT,
    deleted_count BIGINT,
    total_size_bytes BIGINT,
    avg_size_bytes NUMERIC
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        COUNT(*)::BIGINT AS total_count,
        COUNT(*) FILTER (WHERE status = 'active')::BIGINT AS active_count,
        COUNT(*) FILTER (WHERE status = 'stopped')::BIGINT AS stopped_count,
        COUNT(*) FILTER (WHERE status = 'error')::BIGINT AS error_count,
        COUNT(*) FILTER (WHERE status = 'approved')::BIGINT AS approved_count,
        COUNT(*) FILTER (WHERE status = 'rejected')::BIGINT AS rejected_count,
        COUNT(*) FILTER (WHERE status = 'deleted')::BIGINT AS deleted_count,
        COALESCE(SUM(s.size_bytes), 0)::BIGINT AS total_size_bytes,
        COALESCE(AVG(s.size_bytes), 0)::NUMERIC AS avg_size_bytes
    FROM sandboxes s;
END;
$$ LANGUAGE plpgsql;
