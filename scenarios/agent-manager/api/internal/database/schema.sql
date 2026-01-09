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
    profile_key VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    runner_type VARCHAR(50) NOT NULL,
    model VARCHAR(100),
    model_preset VARCHAR(20),
    max_turns INTEGER,
    timeout_ms BIGINT,
    fallback_runner_types JSONB DEFAULT '[]',
    allowed_tools JSONB DEFAULT '[]',
    denied_tools JSONB DEFAULT '[]',
    skip_permission_prompt BOOLEAN DEFAULT FALSE,
    requires_sandbox BOOLEAN DEFAULT TRUE,
    requires_approval BOOLEAN DEFAULT TRUE,
    sandbox_config JSONB DEFAULT '{}'::jsonb,
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
    sandbox_config JSONB DEFAULT '{}'::jsonb,
    session_id VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_runs_task_id ON runs(task_id);
CREATE INDEX IF NOT EXISTS idx_runs_session_id ON runs(session_id) WHERE session_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_runs_agent_profile_id ON runs(agent_profile_id);
CREATE INDEX IF NOT EXISTS idx_runs_status ON runs(status);
CREATE INDEX IF NOT EXISTS idx_runs_tag ON runs(tag);
CREATE INDEX IF NOT EXISTS idx_runs_created_at ON runs(created_at DESC);

-- Stats query indexes
CREATE INDEX IF NOT EXISTS idx_runs_created_status ON runs(created_at DESC, status);
CREATE INDEX IF NOT EXISTS idx_runs_runner_type ON runs((resolved_config->>'runnerType'));
CREATE INDEX IF NOT EXISTS idx_runs_model ON runs((resolved_config->>'model'));

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

-- Stats query indexes for cost aggregation
CREATE INDEX IF NOT EXISTS idx_run_events_cost ON run_events(run_id, event_type) WHERE event_type = 'metric';
CREATE INDEX IF NOT EXISTS idx_run_events_tool_calls ON run_events(run_id, event_type) WHERE event_type = 'tool_call';
CREATE INDEX IF NOT EXISTS idx_run_events_errors ON run_events(run_id, event_type) WHERE event_type = 'error';

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
-- Model Pricing - Cached pricing data from providers
-- ============================================================================
CREATE TABLE IF NOT EXISTS model_pricing (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canonical_model_name VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,

    -- Per-component pricing (USD per token)
    input_token_price NUMERIC(20, 12),
    output_token_price NUMERIC(20, 12),
    cache_read_price NUMERIC(20, 12),
    cache_creation_price NUMERIC(20, 12),
    web_search_price NUMERIC(20, 12),
    server_tool_use_price NUMERIC(20, 12),

    -- Per-component sources (manual_override, provider_api, historical_average, unknown)
    input_token_source VARCHAR(50) DEFAULT 'unknown',
    output_token_source VARCHAR(50) DEFAULT 'unknown',
    cache_read_source VARCHAR(50),
    cache_creation_source VARCHAR(50),
    web_search_source VARCHAR(50),
    server_tool_use_source VARCHAR(50),

    -- Metadata
    fetched_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    pricing_version VARCHAR(100),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(canonical_model_name, provider)
);

CREATE INDEX IF NOT EXISTS idx_model_pricing_model ON model_pricing(canonical_model_name);
CREATE INDEX IF NOT EXISTS idx_model_pricing_provider ON model_pricing(provider);
CREATE INDEX IF NOT EXISTS idx_model_pricing_expires ON model_pricing(expires_at);

-- ============================================================================
-- Model Aliases - Maps runner model names to canonical names
-- ============================================================================
CREATE TABLE IF NOT EXISTS model_aliases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    runner_model VARCHAR(255) NOT NULL,
    runner_type VARCHAR(100) NOT NULL,
    canonical_model VARCHAR(255) NOT NULL,
    provider VARCHAR(100) NOT NULL,

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(runner_model, runner_type)
);

CREATE INDEX IF NOT EXISTS idx_model_aliases_runner ON model_aliases(runner_model, runner_type);
CREATE INDEX IF NOT EXISTS idx_model_aliases_canonical ON model_aliases(canonical_model);

-- ============================================================================
-- Manual Price Overrides - User-specified pricing overrides
-- ============================================================================
CREATE TABLE IF NOT EXISTS manual_price_overrides (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    canonical_model_name VARCHAR(255) NOT NULL,
    component VARCHAR(50) NOT NULL,
    price_usd NUMERIC(20, 12) NOT NULL,
    note TEXT,
    created_by VARCHAR(255),

    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,

    UNIQUE(canonical_model_name, component)
);

CREATE INDEX IF NOT EXISTS idx_manual_overrides_model ON manual_price_overrides(canonical_model_name);
CREATE INDEX IF NOT EXISTS idx_manual_overrides_expires ON manual_price_overrides(expires_at);

-- ============================================================================
-- Pricing Settings - Global pricing configuration (singleton table)
-- ============================================================================
CREATE TABLE IF NOT EXISTS pricing_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    historical_average_days INTEGER DEFAULT 7,
    provider_cache_ttl_seconds INTEGER DEFAULT 21600,

    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings if not exists
INSERT INTO pricing_settings (id, historical_average_days, provider_cache_ttl_seconds)
VALUES (1, 7, 21600)
ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- Investigation Settings - Configuration for investigation agents (singleton)
-- ============================================================================
CREATE TABLE IF NOT EXISTS investigation_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    -- Prompt template is plain text (no variables/templating)
    -- Dynamic content like depth, scenarios, runs are context attachments
    prompt_template TEXT NOT NULL,
    -- Default depth: "quick", "standard", "deep"
    default_depth VARCHAR(20) NOT NULL DEFAULT 'standard',
    -- Default context selections (what to include in investigations)
    default_context JSONB NOT NULL DEFAULT '{"runSummaries":true,"runEvents":true,"runDiffs":true,"scenarioDocs":true,"fullLogs":false}',

    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Insert default settings with the default investigation prompt
INSERT INTO investigation_settings (id, prompt_template, default_depth, default_context)
VALUES (
    1,
    '# Agent-Manager Investigation

You are an expert debugger. Your job is to ACTIVELY INVESTIGATE why the runs provided in context failed.

**Do NOT just analyze the provided data - EXPLORE THE CODEBASE to find root causes.**

## Required Investigation Steps
1. **Read the scenario''s CLAUDE.md or README.md** to understand the project structure
2. **Analyze the error messages** in the attached run events - find the actual failure
3. **Trace the error to source code** - use grep/read to find the failing code path
4. **Check related files** - look at imports, dependencies, callers, and configuration
5. **Identify the root cause** - distinguish symptoms from underlying issues

## Common Failure Patterns to Check
- **Log/Output Issues**: Large outputs breaking scanners, missing newlines, buffering problems
- **Path Issues**: Relative vs absolute paths, working directory assumptions
- **Timeout Issues**: Operations taking longer than expected
- **Tool Issues**: Missing tools, wrong tool usage, tool not trusted by agent
- **Prompt Issues**: Agent not understanding instructions, missing context
- **State Issues**: Stale data, cache invalidation, missing initialization

## How to Fetch Additional Run Data
If you need full details beyond the attachments, use the agent-manager CLI:
```bash
agent-manager run get <run-id>      # Full run details
agent-manager run events <run-id>  # All events with tool calls
agent-manager run diff <run-id>    # Code changes made
```

## Required Report Format
Provide your findings in this structure:

### 1. Executive Summary
One-paragraph summary of what went wrong and why.

### 2. Root Cause Analysis
- **Primary cause** with file:line references
- **Contributing factors**
- **Evidence** (run IDs, event sequences, code snippets)

### 3. Recommendations
- **Immediate fix** (copy-pasteable code if possible)
- **Preventive measures**
- **Monitoring suggestions**',
    'standard',
    '{"runSummaries":true,"runEvents":true,"runDiffs":true,"scenarioDocs":true,"fullLogs":false}'
)
ON CONFLICT (id) DO NOTHING;

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

    -- Model Pricing
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_model_pricing_updated_at') THEN
        CREATE TRIGGER update_model_pricing_updated_at
            BEFORE UPDATE ON model_pricing
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Model Aliases
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_model_aliases_updated_at') THEN
        CREATE TRIGGER update_model_aliases_updated_at
            BEFORE UPDATE ON model_aliases
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Pricing Settings
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_pricing_settings_updated_at') THEN
        CREATE TRIGGER update_pricing_settings_updated_at
            BEFORE UPDATE ON pricing_settings
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;

    -- Investigation Settings
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'update_investigation_settings_updated_at') THEN
        CREATE TRIGGER update_investigation_settings_updated_at
            BEFORE UPDATE ON investigation_settings
            FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    END IF;
END;
$$;
