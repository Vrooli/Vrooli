-- Agent Manager PostgreSQL Schema
-- This schema provides persistent storage for agent profiles, tasks, runs, and events.

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- Agent Profiles - Defines HOW an agent runs
-- ============================================================================
CREATE TABLE IF NOT EXISTS agent_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    runner_type VARCHAR(50) NOT NULL,
    model VARCHAR(100),
    max_turns INTEGER,
    timeout_ms BIGINT,
    allowed_tools JSONB DEFAULT '[]',
    denied_tools JSONB DEFAULT '[]',
    skip_permission_prompt BOOLEAN DEFAULT FALSE,
    requires_sandbox BOOLEAN DEFAULT TRUE,
    requires_approval BOOLEAN DEFAULT TRUE,
    allowed_paths JSONB DEFAULT '[]',
    denied_paths JSONB DEFAULT '[]',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_profiles_name ON agent_profiles(name);

-- ============================================================================
-- Tasks - Defines WHAT needs to be done
-- ============================================================================
CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(500) NOT NULL,
    description TEXT,
    scope_path VARCHAR(1000) NOT NULL,
    project_root VARCHAR(1000),
    phase_prompt_ids JSONB DEFAULT '[]',
    context_attachments JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'queued',
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_tasks_status ON tasks(status);
CREATE INDEX IF NOT EXISTS idx_tasks_created_at ON tasks(created_at DESC);

-- ============================================================================
-- Runs - Concrete execution attempts
-- ============================================================================
CREATE TABLE IF NOT EXISTS runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    agent_profile_id UUID REFERENCES agent_profiles(id) ON DELETE SET NULL,
    tag VARCHAR(255),
    sandbox_id UUID,
    run_mode VARCHAR(50) DEFAULT 'sandboxed',
    status VARCHAR(50) DEFAULT 'pending',
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    phase VARCHAR(50) DEFAULT 'queued',
    last_checkpoint_id UUID,
    last_heartbeat TIMESTAMPTZ,
    progress_percent INTEGER DEFAULT 0,
    idempotency_key VARCHAR(255) UNIQUE,
    summary JSONB,
    error_msg TEXT,
    exit_code INTEGER,
    approval_state VARCHAR(50) DEFAULT 'none',
    approved_by VARCHAR(255),
    approved_at TIMESTAMPTZ,
    resolved_config JSONB,
    diff_path VARCHAR(1000),
    log_path VARCHAR(1000),
    changed_files INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_tag ON runs(tag);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at DESC);

-- ============================================================================
-- Run Events - Append-only event stream
-- ============================================================================
CREATE TABLE IF NOT EXISTS run_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    sequence BIGINT NOT NULL,
    event_type VARCHAR(50) NOT NULL,
    timestamp TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    data JSONB NOT NULL,
    UNIQUE(run_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events(run_id);
CREATE INDEX IF NOT EXISTS idx_run_events_run_sequence ON run_events(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_run_events_type ON run_events(run_id, event_type);

-- ============================================================================
-- Run Checkpoints - For resumption after interruption
-- ============================================================================
CREATE TABLE IF NOT EXISTS run_checkpoints (
    run_id UUID PRIMARY KEY REFERENCES runs(id) ON DELETE CASCADE,
    phase VARCHAR(50) NOT NULL,
    step_within_phase INTEGER DEFAULT 0,
    sandbox_id UUID,
    work_dir VARCHAR(1000),
    lock_id UUID,
    last_event_sequence BIGINT DEFAULT 0,
    last_heartbeat TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    retry_count INTEGER DEFAULT 0,
    saved_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_run_checkpoints_heartbeat ON run_checkpoints(last_heartbeat);

-- ============================================================================
-- Idempotency Records - For replay-safe operations
-- ============================================================================
CREATE TABLE IF NOT EXISTS idempotency_records (
    key VARCHAR(500) PRIMARY KEY,
    status VARCHAR(50) NOT NULL,
    entity_id UUID,
    entity_type VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL,
    response JSONB
);

CREATE INDEX IF NOT EXISTS idx_idempotency_expires ON idempotency_records(expires_at);
CREATE INDEX IF NOT EXISTS idx_idempotency_status ON idempotency_records(status);

-- ============================================================================
-- Policies - Rules governing execution
-- ============================================================================
CREATE TABLE IF NOT EXISTS policies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    scope_pattern VARCHAR(500),
    rules JSONB NOT NULL DEFAULT '{}',
    enabled BOOLEAN DEFAULT TRUE,
    created_by VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_policies_enabled ON policies(enabled);
CREATE INDEX IF NOT EXISTS idx_policies_priority ON policies(priority DESC);

-- ============================================================================
-- Scope Locks - Concurrency control
-- ============================================================================
CREATE TABLE IF NOT EXISTS scope_locks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES runs(id) ON DELETE CASCADE,
    scope_path VARCHAR(1000) NOT NULL,
    project_root VARCHAR(1000),
    acquired_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_scope_locks_run_id ON scope_locks(run_id);
CREATE INDEX IF NOT EXISTS idx_scope_locks_scope ON scope_locks(scope_path, project_root);
CREATE INDEX IF NOT EXISTS idx_scope_locks_expires ON scope_locks(expires_at);

-- ============================================================================
-- Triggers for automatic updated_at
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DO $$
BEGIN
    -- Agent Profiles
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_agent_profiles_updated_at') THEN
        CREATE TRIGGER update_agent_profiles_updated_at
            BEFORE UPDATE ON agent_profiles
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Tasks
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_tasks_updated_at') THEN
        CREATE TRIGGER update_tasks_updated_at
            BEFORE UPDATE ON tasks
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Runs
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_runs_updated_at') THEN
        CREATE TRIGGER update_runs_updated_at
            BEFORE UPDATE ON runs
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Policies
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_policies_updated_at') THEN
        CREATE TRIGGER update_policies_updated_at
            BEFORE UPDATE ON policies
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END;
$$;
