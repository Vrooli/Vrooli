-- Web Console PostgreSQL Schema
-- Currently used for health checks only; session/shortcut/AI config data is
-- managed in-memory. This schema is the canonical reference for the data model
-- and will be activated when session persistence is implemented.

-- Enable extensions (idempotent)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Session expiration policy mode
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'policy_mode') THEN
        CREATE TYPE policy_mode AS ENUM ('never', 'preset', 'custom');
    END IF;
END$$;

-- Shortcut profile scope
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'shortcut_scope') THEN
        CREATE TYPE shortcut_scope AS ENUM ('service', 'workspace', 'parent');
    END IF;
END$$;

-- Terminal sessions (metadata only; PTY state is process-bound)
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    shell VARCHAR(255) NOT NULL DEFAULT '/bin/bash',
    cols SMALLINT NOT NULL DEFAULT 80,
    rows SMALLINT NOT NULL DEFAULT 24,
    policy_mode policy_mode NOT NULL DEFAULT 'never',
    policy_duration INTERVAL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sessions_created ON sessions(created_at DESC);

-- Shortcut profiles with scope hierarchy
CREATE TABLE IF NOT EXISTS shortcut_profiles (
    id VARCHAR(255) PRIMARY KEY,
    scope shortcut_scope NOT NULL DEFAULT 'service',
    name VARCHAR(255) NOT NULL,
    shortcuts JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_shortcut_profiles_scope ON shortcut_profiles(scope);

-- AI provider configuration
CREATE TABLE IF NOT EXISTS ai_provider_configs (
    name VARCHAR(100) PRIMARY KEY,
    enabled BOOLEAN NOT NULL DEFAULT true,
    priority SMALLINT NOT NULL DEFAULT 1,
    timeout_sec SMALLINT NOT NULL DEFAULT 30,
    max_retries SMALLINT NOT NULL DEFAULT 0
);

-- AI provider health tracking (ephemeral, reset on restart)
CREATE TABLE IF NOT EXISTS ai_provider_health (
    name VARCHAR(100) PRIMARY KEY REFERENCES ai_provider_configs(name) ON DELETE CASCADE,
    available BOOLEAN NOT NULL DEFAULT false,
    last_check TIMESTAMP WITH TIME ZONE,
    last_latency_ms INTEGER,
    error_count BIGINT NOT NULL DEFAULT 0,
    success_count BIGINT NOT NULL DEFAULT 0
);
