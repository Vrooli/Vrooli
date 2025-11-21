-- Deployment Manager Database Schema
-- Version: 1.0.0
-- Description: Core tables for deployment profiles, fitness rules, swap alternatives, and audit logs

-- Enable UUID extension for ID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Deployment Profiles
-- Stores complete deployment configuration including tier, scenario, swaps, and secrets
CREATE TABLE IF NOT EXISTS deployment_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    scenario_name VARCHAR(255) NOT NULL,
    target_tiers TEXT[] NOT NULL, -- Array of tier names (e.g., ['desktop', 'mobile'])
    profile_data JSONB NOT NULL, -- Complete profile configuration (swaps, env vars, etc.)
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, deploying, deployed, failed
    CONSTRAINT valid_status CHECK (status IN ('draft', 'active', 'deploying', 'deployed', 'failed', 'archived'))
);

CREATE INDEX idx_deployment_profiles_scenario ON deployment_profiles(scenario_name);
CREATE INDEX idx_deployment_profiles_status ON deployment_profiles(status);
CREATE INDEX idx_deployment_profiles_created_at ON deployment_profiles(created_at DESC);

-- Profile Version History
-- Immutable audit trail of all profile changes
CREATE TABLE IF NOT EXISTS profile_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES deployment_profiles(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    profile_snapshot JSONB NOT NULL, -- Full profile state at this version
    change_summary TEXT, -- Human-readable description of changes
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255),
    UNIQUE(profile_id, version)
);

CREATE INDEX idx_profile_versions_profile_id ON profile_versions(profile_id);
CREATE INDEX idx_profile_versions_created_at ON profile_versions(created_at DESC);

-- Fitness Rules
-- Pluggable rules for tier-specific fitness scoring
CREATE TABLE IF NOT EXISTS fitness_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_name VARCHAR(255) NOT NULL UNIQUE,
    tier VARCHAR(50) NOT NULL, -- Tier this rule applies to
    dependency_type VARCHAR(50), -- resource or scenario (null = applies to all)
    dependency_pattern VARCHAR(255), -- Regex pattern (null = applies to all)
    scoring_config JSONB NOT NULL, -- Rule configuration (weights, thresholds, etc.)
    enabled BOOLEAN DEFAULT true,
    priority INTEGER DEFAULT 100, -- Higher priority rules evaluated first
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT valid_tier CHECK (tier IN ('local', 'desktop', 'mobile', 'saas', 'enterprise'))
);

CREATE INDEX idx_fitness_rules_tier ON fitness_rules(tier);
CREATE INDEX idx_fitness_rules_enabled ON fitness_rules(enabled);
CREATE INDEX idx_fitness_rules_priority ON fitness_rules(priority DESC);

-- Swap Alternatives
-- Database of known dependency swaps with metadata
CREATE TABLE IF NOT EXISTS swap_alternatives (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    from_dep_type VARCHAR(50) NOT NULL, -- resource or scenario
    from_dep_name VARCHAR(255) NOT NULL,
    to_dep_type VARCHAR(50) NOT NULL,
    to_dep_name VARCHAR(255) NOT NULL,
    applicable_tiers TEXT[] NOT NULL, -- Which tiers this swap applies to
    fitness_delta INTEGER, -- Expected fitness improvement (+/-)
    migration_effort VARCHAR(50) NOT NULL, -- trivial, moderate, major
    pros TEXT[], -- Array of advantages
    cons TEXT[], -- Array of disadvantages
    migration_notes TEXT, -- Detailed migration instructions
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    verified BOOLEAN DEFAULT false, -- Has this swap been tested?
    verification_notes TEXT,
    CONSTRAINT valid_migration_effort CHECK (migration_effort IN ('trivial', 'moderate', 'major')),
    UNIQUE(from_dep_type, from_dep_name, to_dep_type, to_dep_name)
);

CREATE INDEX idx_swap_alternatives_from_dep ON swap_alternatives(from_dep_type, from_dep_name);
CREATE INDEX idx_swap_alternatives_applicable_tiers ON swap_alternatives USING GIN(applicable_tiers);
CREATE INDEX idx_swap_alternatives_verified ON swap_alternatives(verified);

-- Deployments
-- Tracks actual deployment instances and their status
CREATE TABLE IF NOT EXISTS deployments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES deployment_profiles(id) ON DELETE CASCADE,
    tier VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    deployment_config JSONB NOT NULL, -- Final resolved configuration sent to packager
    deployment_url TEXT, -- URL where deployment is accessible
    deployment_metadata JSONB, -- Tier-specific metadata (app store IDs, server IPs, etc.)
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    error_log TEXT, -- Captured error logs on failure
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    CONSTRAINT valid_status CHECK (status IN ('queued', 'deploying', 'deployed', 'failed', 'rolled_back')),
    CONSTRAINT valid_tier CHECK (tier IN ('local', 'desktop', 'mobile', 'saas', 'enterprise'))
);

CREATE INDEX idx_deployments_profile_id ON deployments(profile_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_tier ON deployments(tier);
CREATE INDEX idx_deployments_started_at ON deployments(started_at DESC);

-- Deployment Health Checks
-- Periodic health check results for deployed scenarios
CREATE TABLE IF NOT EXISTS deployment_health (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deployment_id UUID NOT NULL REFERENCES deployments(id) ON DELETE CASCADE,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    status VARCHAR(50) NOT NULL, -- up, degraded, down, unknown
    response_time_ms INTEGER,
    error_message TEXT,
    metadata JSONB, -- Additional health metrics
    CONSTRAINT valid_status CHECK (status IN ('up', 'degraded', 'down', 'unknown'))
);

CREATE INDEX idx_deployment_health_deployment_id ON deployment_health(deployment_id);
CREATE INDEX idx_deployment_health_checked_at ON deployment_health(checked_at DESC);
CREATE INDEX idx_deployment_health_status ON deployment_health(status);

-- Audit Logs
-- Immutable log of all deployment actions for compliance
CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type VARCHAR(50) NOT NULL, -- profile, deployment, swap, etc.
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL, -- create, update, deploy, rollback, delete
    actor VARCHAR(255), -- User or system that performed action
    changes JSONB, -- JSON diff of what changed
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX idx_audit_logs_entity ON audit_logs(entity_type, entity_id);
CREATE INDEX idx_audit_logs_timestamp ON audit_logs(timestamp DESC);
CREATE INDEX idx_audit_logs_actor ON audit_logs(actor);

-- Prevent deletion/modification of audit logs (append-only)
CREATE OR REPLACE FUNCTION prevent_audit_modification()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit logs are immutable and cannot be modified or deleted';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_immutable_update
    BEFORE UPDATE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

CREATE TRIGGER audit_logs_immutable_delete
    BEFORE DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_modification();

-- Auto-update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_deployment_profiles_updated_at BEFORE UPDATE ON deployment_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_fitness_rules_updated_at BEFORE UPDATE ON fitness_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_swap_alternatives_updated_at BEFORE UPDATE ON swap_alternatives
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
