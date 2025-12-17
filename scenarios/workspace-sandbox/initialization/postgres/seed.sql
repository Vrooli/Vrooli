-- workspace-sandbox PostgreSQL schema
-- This schema supports the sandbox lifecycle management system
-- Designed to be idempotent - safe to run multiple times

-- ============================================================================
-- EXTENSIONS (use public schema for extensions)
-- ============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp" SCHEMA public;

-- ============================================================================
-- CUSTOM TYPES (idempotent creation)
-- ============================================================================

-- Sandbox status enum matching types.Status in Go
DO $$ BEGIN
    CREATE TYPE sandbox_status AS ENUM (
        'creating',
        'active',
        'stopped',
        'approved',
        'rejected',
        'deleted',
        'error'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- Owner type enum matching types.OwnerType in Go
DO $$ BEGIN
    CREATE TYPE owner_type AS ENUM (
        'agent',
        'user',
        'task',
        'system'
    );
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================================
-- SANDBOXES TABLE
-- ============================================================================
-- Core table storing sandbox metadata and mount configuration
CREATE TABLE IF NOT EXISTS sandboxes (
    -- Identity
    id UUID PRIMARY KEY DEFAULT public.uuid_generate_v4(),

    -- Scope configuration
    scope_path TEXT NOT NULL,
    project_root TEXT NOT NULL,

    -- Ownership
    owner TEXT,
    owner_type owner_type NOT NULL DEFAULT 'user',

    -- Status tracking
    status sandbox_status NOT NULL DEFAULT 'creating',
    error_message TEXT,

    -- Timestamps
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    stopped_at TIMESTAMPTZ,
    approved_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ,

    -- Driver configuration
    driver TEXT NOT NULL DEFAULT 'overlayfs',
    driver_version TEXT NOT NULL DEFAULT '1.0.0',

    -- Mount paths (populated after mount)
    lower_dir TEXT,
    upper_dir TEXT,
    work_dir TEXT,
    merged_dir TEXT,

    -- Size accounting
    size_bytes BIGINT NOT NULL DEFAULT 0,
    file_count INTEGER NOT NULL DEFAULT 0,

    -- Session tracking
    active_pids INTEGER[] NOT NULL DEFAULT '{}',
    session_count INTEGER NOT NULL DEFAULT 0,

    -- Metadata
    tags TEXT[] NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',

    -- Idempotency & Concurrency Control (added for replay safety)
    idempotency_key TEXT UNIQUE,  -- Client-provided key for request deduplication
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),  -- Last modification time
    version BIGINT NOT NULL DEFAULT 1  -- Optimistic locking version counter
);

-- Index for fast status filtering (most common query pattern)
CREATE INDEX IF NOT EXISTS idx_sandboxes_status ON sandboxes(status);

-- Index for scope path lookups (needed for overlap checking)
CREATE INDEX IF NOT EXISTS idx_sandboxes_scope_path ON sandboxes(scope_path);

-- Index for project root lookups
CREATE INDEX IF NOT EXISTS idx_sandboxes_project_root ON sandboxes(project_root);

-- Index for owner filtering
CREATE INDEX IF NOT EXISTS idx_sandboxes_owner ON sandboxes(owner) WHERE owner IS NOT NULL;

-- Index for time-based queries (GC, listing)
CREATE INDEX IF NOT EXISTS idx_sandboxes_created_at ON sandboxes(created_at DESC);

-- Composite index for common filter combinations
CREATE INDEX IF NOT EXISTS idx_sandboxes_status_created
    ON sandboxes(status, created_at DESC);

-- Index for idempotency key lookups (UNIQUE constraint already provides this, but explicit for clarity)
CREATE INDEX IF NOT EXISTS idx_sandboxes_idempotency_key
    ON sandboxes(idempotency_key) WHERE idempotency_key IS NOT NULL;

-- ============================================================================
-- AUDIT LOG TABLE
-- ============================================================================
-- Immutable log of all sandbox operations for debugging and compliance
CREATE TABLE IF NOT EXISTS sandbox_audit_log (
    id UUID PRIMARY KEY DEFAULT public.uuid_generate_v4(),
    sandbox_id UUID REFERENCES sandboxes(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    actor TEXT,
    actor_type TEXT,
    details JSONB NOT NULL DEFAULT '{}',
    sandbox_state JSONB NOT NULL DEFAULT '{}'
);

-- Index for sandbox-specific audit trails
CREATE INDEX IF NOT EXISTS idx_audit_sandbox_id ON sandbox_audit_log(sandbox_id);

-- Index for time-based audit queries
CREATE INDEX IF NOT EXISTS idx_audit_event_time ON sandbox_audit_log(event_time DESC);

-- Index for event type filtering
CREATE INDEX IF NOT EXISTS idx_audit_event_type ON sandbox_audit_log(event_type);

-- ============================================================================
-- SCOPE OVERLAP CHECKING FUNCTION
-- ============================================================================
-- This function checks if a new scope path overlaps with any existing active sandbox
-- by checking ancestor/descendant relationships.
--
-- Overlap types:
--   - 'exact': Same path
--   - 'new_is_ancestor': New path is parent of existing
--   - 'existing_is_ancestor': Existing path is parent of new
--
-- Only checks against active sandboxes (creating, active status)
CREATE OR REPLACE FUNCTION check_scope_overlap(
    p_scope_path TEXT,
    p_project_root TEXT,
    p_exclude_id UUID DEFAULT NULL
)
RETURNS TABLE (
    id UUID,
    scope_path TEXT,
    status sandbox_status
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        s.id,
        s.scope_path,
        s.status
    FROM sandboxes s
    WHERE
        -- Only check active sandboxes
        s.status IN ('creating', 'active')
        -- Same project root
        AND s.project_root = p_project_root
        -- Exclude specified ID if provided
        AND (p_exclude_id IS NULL OR s.id != p_exclude_id)
        -- Check for path overlap (exact match or ancestor/descendant)
        AND (
            -- Exact match
            s.scope_path = p_scope_path
            -- New path is ancestor of existing (existing starts with new + /)
            OR s.scope_path LIKE (p_scope_path || '/%')
            -- Existing is ancestor of new (new starts with existing + /)
            OR p_scope_path LIKE (s.scope_path || '/%')
        );
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to update last_used_at timestamp
CREATE OR REPLACE FUNCTION update_last_used_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_used_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to auto-update last_used_at on any update
DROP TRIGGER IF EXISTS sandboxes_update_last_used ON sandboxes;
CREATE TRIGGER sandboxes_update_last_used
    BEFORE UPDATE ON sandboxes
    FOR EACH ROW
    EXECUTE FUNCTION update_last_used_at();

-- Function to get sandbox statistics (for dashboard/metrics)
CREATE OR REPLACE FUNCTION get_sandbox_stats()
RETURNS TABLE (
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
        COUNT(*)::BIGINT as total_count,
        COUNT(*) FILTER (WHERE s.status = 'active')::BIGINT as active_count,
        COUNT(*) FILTER (WHERE s.status = 'stopped')::BIGINT as stopped_count,
        COUNT(*) FILTER (WHERE s.status = 'error')::BIGINT as error_count,
        COUNT(*) FILTER (WHERE s.status = 'approved')::BIGINT as approved_count,
        COUNT(*) FILTER (WHERE s.status = 'rejected')::BIGINT as rejected_count,
        COUNT(*) FILTER (WHERE s.status = 'deleted')::BIGINT as deleted_count,
        COALESCE(SUM(s.size_bytes), 0)::BIGINT as total_size_bytes,
        COALESCE(AVG(s.size_bytes), 0)::NUMERIC as avg_size_bytes
    FROM sandboxes s
    WHERE s.status != 'deleted';
END;
$$ LANGUAGE plpgsql STABLE;

-- ============================================================================
-- COMMENTS FOR DOCUMENTATION
-- ============================================================================
COMMENT ON TABLE sandboxes IS 'Workspace sandbox instances with copy-on-write overlay configuration';
COMMENT ON COLUMN sandboxes.scope_path IS 'Relative or absolute path within project that this sandbox covers';
COMMENT ON COLUMN sandboxes.project_root IS 'Root directory of the project being sandboxed';
COMMENT ON COLUMN sandboxes.lower_dir IS 'Overlayfs lower directory (read-only canonical repo)';
COMMENT ON COLUMN sandboxes.upper_dir IS 'Overlayfs upper directory (writable changes)';
COMMENT ON COLUMN sandboxes.work_dir IS 'Overlayfs work directory (internal)';
COMMENT ON COLUMN sandboxes.merged_dir IS 'Overlayfs merged mount point (workspace)';
COMMENT ON COLUMN sandboxes.active_pids IS 'PIDs of processes running in this sandbox';

COMMENT ON TABLE sandbox_audit_log IS 'Immutable audit trail of all sandbox operations';
COMMENT ON COLUMN sandbox_audit_log.event_type IS 'Operation type: created, stopped, approved, rejected, deleted, error';
COMMENT ON COLUMN sandbox_audit_log.sandbox_state IS 'Snapshot of sandbox state at event time';

-- Idempotency & Concurrency Control documentation
COMMENT ON COLUMN sandboxes.idempotency_key IS 'Client-provided key for request deduplication; enables safe retries';
COMMENT ON COLUMN sandboxes.updated_at IS 'Last modification timestamp; updated on every change';
COMMENT ON COLUMN sandboxes.version IS 'Optimistic lock version; incremented on each update to detect concurrent modifications';

COMMENT ON FUNCTION check_scope_overlap IS 'Checks for path overlaps with active sandboxes to enforce mutual exclusion';
COMMENT ON FUNCTION get_sandbox_stats IS 'Returns aggregate statistics for dashboard and metrics';
