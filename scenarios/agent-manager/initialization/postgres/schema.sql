-- Agent Manager Database Schema
-- This schema supports the core domain entities for agent orchestration.

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- AgentProfile - Defines HOW an agent runs
-- ============================================================================

CREATE TABLE IF NOT EXISTS agent_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,

    -- Runner configuration
    runner_type VARCHAR(50) NOT NULL CHECK (runner_type IN ('claude-code', 'codex', 'opencode')),
    model VARCHAR(100),
    max_turns INTEGER,
    timeout_ms BIGINT,

    -- Tool permissions (stored as JSON arrays)
    allowed_tools JSONB DEFAULT '[]'::jsonb,
    denied_tools JSONB DEFAULT '[]'::jsonb,

    -- Execution flags
    skip_permission_prompt BOOLEAN DEFAULT FALSE,

    -- Default policies
    requires_sandbox BOOLEAN DEFAULT TRUE,
    requires_approval BOOLEAN DEFAULT TRUE,

    -- Path restrictions (stored as JSON arrays)
    allowed_paths JSONB DEFAULT '[]'::jsonb,
    denied_paths JSONB DEFAULT '[]'::jsonb,

    -- Metadata
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_profiles_name ON agent_profiles(name);
CREATE INDEX idx_agent_profiles_runner_type ON agent_profiles(runner_type);

-- ============================================================================
-- Task - Defines WHAT needs to be done
-- ============================================================================

CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    description TEXT,

    -- Scope
    scope_path VARCHAR(1024) NOT NULL DEFAULT '',
    project_root VARCHAR(1024),

    -- Multi-phase support
    phase_prompt_ids JSONB DEFAULT '[]'::jsonb,

    -- Context attachments (stored as JSON array)
    context_attachments JSONB DEFAULT '[]'::jsonb,

    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'queued'
        CHECK (status IN ('queued', 'running', 'needs_review', 'approved', 'rejected', 'failed', 'cancelled')),

    -- Ownership
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at DESC);
CREATE INDEX idx_tasks_scope_path ON tasks(scope_path);

-- ============================================================================
-- Run - A concrete execution attempt
-- ============================================================================

CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent_profile_id UUID NOT NULL REFERENCES agent_profiles(id) ON DELETE RESTRICT,

    -- Sandbox integration
    sandbox_id UUID,  -- References workspace-sandbox (external)
    run_mode VARCHAR(50) NOT NULL DEFAULT 'sandboxed'
        CHECK (run_mode IN ('sandboxed', 'in_place')),

    -- Execution state
    status VARCHAR(50) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'starting', 'running', 'needs_review', 'complete', 'failed', 'cancelled')),
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,

    -- Results
    summary JSONB,
    error_msg TEXT,
    exit_code INTEGER,

    -- Approval workflow
    approval_state VARCHAR(50) NOT NULL DEFAULT 'none'
        CHECK (approval_state IN ('none', 'pending', 'partially_approved', 'approved', 'rejected')),
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,

    -- Artifacts
    diff_path VARCHAR(1024),
    log_path VARCHAR(1024),
    changed_files INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,

    -- Metadata
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_runs_task_id ON runs(task_id);
CREATE INDEX idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX idx_runs_status ON runs(status);
CREATE INDEX idx_runs_approval_state ON runs(approval_state);
CREATE INDEX idx_runs_created_at ON runs(created_at DESC);
CREATE INDEX idx_runs_sandbox_id ON runs(sandbox_id) WHERE sandbox_id IS NOT NULL;

-- ============================================================================
-- RunEvent - Append-only event stream
-- ============================================================================

CREATE TABLE IF NOT EXISTS run_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sequence BIGSERIAL,
    event_type VARCHAR(50) NOT NULL
        CHECK (event_type IN ('log', 'message', 'tool_call', 'tool_result', 'status', 'metric', 'artifact', 'error')),
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    data JSONB NOT NULL DEFAULT '{}'::jsonb
);

-- Unique constraint on run_id + sequence for ordering
CREATE UNIQUE INDEX idx_run_events_run_sequence ON run_events(run_id, sequence);
CREATE INDEX idx_run_events_run_id ON run_events(run_id);
CREATE INDEX idx_run_events_event_type ON run_events(event_type);
CREATE INDEX idx_run_events_timestamp ON run_events(timestamp);

-- ============================================================================
-- Policy - Rules governing execution
-- ============================================================================

CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    priority INTEGER NOT NULL DEFAULT 0,  -- Higher priority wins

    -- Scope matching
    scope_pattern VARCHAR(1024),  -- Glob pattern

    -- Rules stored as JSON
    rules JSONB NOT NULL DEFAULT '{}'::jsonb,

    -- Metadata
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE INDEX idx_policies_enabled ON policies(enabled) WHERE enabled = TRUE;
CREATE INDEX idx_policies_priority ON policies(priority DESC);
CREATE INDEX idx_policies_scope_pattern ON policies(scope_pattern);

-- ============================================================================
-- ScopeLock - Concurrency control
-- ============================================================================

CREATE TABLE IF NOT EXISTS scope_locks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    scope_path VARCHAR(1024) NOT NULL,
    project_root VARCHAR(1024) NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_scope_locks_run_id ON scope_locks(run_id);
CREATE INDEX idx_scope_locks_scope ON scope_locks(project_root, scope_path);
CREATE INDEX idx_scope_locks_expires_at ON scope_locks(expires_at);

-- Function to check for scope overlaps (ancestor/descendant relationship)
CREATE OR REPLACE FUNCTION check_scope_overlap(
    p_scope_path VARCHAR,
    p_project_root VARCHAR,
    p_exclude_run_id UUID DEFAULT NULL
) RETURNS TABLE (
    run_id UUID,
    scope_path VARCHAR,
    overlap_type VARCHAR
) AS $$
BEGIN
    RETURN QUERY
    SELECT
        sl.run_id,
        sl.scope_path,
        CASE
            WHEN sl.scope_path = p_scope_path THEN 'exact'
            WHEN sl.scope_path LIKE p_scope_path || '/%' THEN 'descendant'
            WHEN p_scope_path LIKE sl.scope_path || '/%' THEN 'ancestor'
            ELSE 'none'
        END AS overlap_type
    FROM scope_locks sl
    WHERE sl.project_root = p_project_root
      AND sl.expires_at > NOW()
      AND (p_exclude_run_id IS NULL OR sl.run_id != p_exclude_run_id)
      AND (
          sl.scope_path = p_scope_path
          OR sl.scope_path LIKE p_scope_path || '/%'
          OR p_scope_path LIKE sl.scope_path || '/%'
      );
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- Artifacts - Stored artifacts for runs
-- ============================================================================

CREATE TABLE IF NOT EXISTS artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL
        CHECK (type IN ('diff', 'log', 'validation', 'screenshot', 'summary', 'other')),
    name VARCHAR(255) NOT NULL,
    storage_path VARCHAR(1024) NOT NULL,
    content_size BIGINT DEFAULT 0,
    content_type VARCHAR(100),
    checksum VARCHAR(64),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_artifacts_run_id ON artifacts(run_id);
CREATE INDEX idx_artifacts_type ON artifacts(type);

-- ============================================================================
-- Triggers for updated_at
-- ============================================================================

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_agent_profiles_updated_at
    BEFORE UPDATE ON agent_profiles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at
    BEFORE UPDATE ON tasks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_runs_updated_at
    BEFORE UPDATE ON runs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_policies_updated_at
    BEFORE UPDATE ON policies
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- Default Policies
-- ============================================================================

-- Insert default sandbox-first policy
INSERT INTO policies (id, name, description, priority, rules, enabled)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'default-sandbox-required',
    'Default policy requiring sandbox for all runs',
    0,
    '{
        "requireSandbox": true,
        "allowInPlace": false,
        "requireApproval": true,
        "maxConcurrentRuns": 10,
        "maxConcurrentPerScope": 1
    }'::jsonb,
    TRUE
) ON CONFLICT (id) DO NOTHING;
