-- Database schema for Test Genie scenario.
-- Captures suite generation requests and execution metadata.

CREATE TABLE IF NOT EXISTS suite_requests (
    id UUID PRIMARY KEY,
    scenario_name TEXT NOT NULL,
    requested_types TEXT[] NOT NULL,
    coverage_target INTEGER NOT NULL CHECK (coverage_target BETWEEN 1 AND 100),
    priority TEXT NOT NULL CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    status TEXT NOT NULL CHECK (status IN ('queued', 'delegated', 'running', 'completed', 'failed')),
    notes TEXT,
    delegation_issue_id TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_suite_requests_scenario
    ON suite_requests (scenario_name);

CREATE INDEX IF NOT EXISTS idx_suite_requests_status
    ON suite_requests (status);

CREATE TABLE IF NOT EXISTS suite_executions (
    id UUID PRIMARY KEY,
    suite_request_id UUID REFERENCES suite_requests(id) ON DELETE SET NULL,
    scenario_name TEXT NOT NULL,
    preset_used TEXT,
    success BOOLEAN NOT NULL,
    phases JSONB NOT NULL,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_suite_executions_scenario
    ON suite_executions (scenario_name);

-- Agent spawning and tracking tables.
-- Enables persistence across server restarts and historical analysis.

CREATE TABLE IF NOT EXISTS spawned_agents (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT UNIQUE,  -- Client-provided key for deduplication
    session_id TEXT,
    scenario TEXT NOT NULL,
    scope TEXT[] NOT NULL DEFAULT '{}',
    phases TEXT[] NOT NULL DEFAULT '{}',
    model TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'timeout', 'stopped')),
    prompt_hash TEXT NOT NULL,
    prompt_index INTEGER NOT NULL,
    prompt_text TEXT NOT NULL,
    output TEXT,
    error TEXT,
    started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Process tracking for orphan detection
    pid INTEGER,  -- OS process ID when running
    hostname TEXT  -- Host where the process runs (for multi-host deployments)
);

-- Migration: Add pid and hostname columns if they don't exist (for existing databases)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='spawned_agents' AND column_name='pid') THEN
        ALTER TABLE spawned_agents ADD COLUMN pid INTEGER;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='spawned_agents' AND column_name='hostname') THEN
        ALTER TABLE spawned_agents ADD COLUMN hostname TEXT;
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_spawned_agents_scenario
    ON spawned_agents (scenario);

CREATE INDEX IF NOT EXISTS idx_spawned_agents_status
    ON spawned_agents (status);

CREATE INDEX IF NOT EXISTS idx_spawned_agents_started_at
    ON spawned_agents (started_at DESC);

-- Tracks path locks for conflict detection.
-- Locks auto-expire but can be renewed via heartbeat.

CREATE TABLE IF NOT EXISTS agent_scope_locks (
    id SERIAL PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES spawned_agents(id) ON DELETE CASCADE,
    scenario TEXT NOT NULL,
    path TEXT NOT NULL,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    renewed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_scope_locks_scenario
    ON agent_scope_locks (scenario);

CREATE INDEX IF NOT EXISTS idx_agent_scope_locks_expires
    ON agent_scope_locks (expires_at);

CREATE INDEX IF NOT EXISTS idx_agent_scope_locks_agent_id
    ON agent_scope_locks (agent_id);

-- Spawn intents table for idempotent spawn requests.
-- Prevents duplicate spawns from race conditions or retries.

CREATE TABLE IF NOT EXISTS spawn_intents (
    key TEXT PRIMARY KEY,
    scenario TEXT NOT NULL,
    scope TEXT[] NOT NULL DEFAULT '{}',
    agent_id TEXT REFERENCES spawned_agents(id) ON DELETE SET NULL,
    status TEXT NOT NULL CHECK (status IN ('pending', 'completed', 'failed')),
    result_json TEXT,  -- Cached response for replaying to duplicate requests
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_spawn_intents_expires
    ON spawn_intents (expires_at);

CREATE INDEX IF NOT EXISTS idx_spawn_intents_scenario
    ON spawn_intents (scenario);

-- Agent file operations audit log.
-- Tracks all file changes made by agents for accountability and rollback capability.

CREATE TABLE IF NOT EXISTS agent_file_operations (
    id SERIAL PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES spawned_agents(id) ON DELETE CASCADE,
    scenario TEXT NOT NULL,
    operation TEXT NOT NULL CHECK (operation IN ('create', 'modify', 'delete')),
    file_path TEXT NOT NULL,
    content_hash TEXT,  -- SHA256 hash of content for deduplication
    content_before TEXT,  -- Previous content (for modify/delete)
    content_after TEXT,  -- New content (for create/modify)
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_agent_file_operations_agent_id
    ON agent_file_operations (agent_id);

CREATE INDEX IF NOT EXISTS idx_agent_file_operations_scenario
    ON agent_file_operations (scenario);

CREATE INDEX IF NOT EXISTS idx_agent_file_operations_recorded_at
    ON agent_file_operations (recorded_at DESC);

-- Server-side spawn session tracking.
-- Tracks spawn activity by user/IP to prevent duplicate spawns across browser tabs/sessions.
-- This replaces the browser-only sessionStorage tracking for stronger guarantees.

CREATE TABLE IF NOT EXISTS spawn_sessions (
    id SERIAL PRIMARY KEY,
    -- User identifier: IP address, API key, or user ID (depending on auth)
    user_identifier TEXT NOT NULL,
    scenario TEXT NOT NULL,
    scope TEXT[] NOT NULL DEFAULT '{}',
    phases TEXT[] NOT NULL DEFAULT '{}',
    agent_ids TEXT[] NOT NULL DEFAULT '{}',
    status TEXT NOT NULL CHECK (status IN ('active', 'completed', 'failed', 'cleared')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- Auto-expire sessions after 30 minutes of inactivity
    expires_at TIMESTAMPTZ NOT NULL,
    -- Last activity timestamp for session renewal
    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_spawn_sessions_user
    ON spawn_sessions (user_identifier);

CREATE INDEX IF NOT EXISTS idx_spawn_sessions_scenario
    ON spawn_sessions (scenario);

CREATE INDEX IF NOT EXISTS idx_spawn_sessions_expires
    ON spawn_sessions (expires_at);

CREATE INDEX IF NOT EXISTS idx_spawn_sessions_status
    ON spawn_sessions (status);
