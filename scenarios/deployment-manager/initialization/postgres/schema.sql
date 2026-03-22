-- Deployment Manager database schema
-- Supports profile management with version history

-- Profiles table stores deployment profiles
CREATE TABLE IF NOT EXISTS profiles (
    id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    scenario VARCHAR(255) NOT NULL,
    tiers JSONB NOT NULL DEFAULT '[]',
    swaps JSONB NOT NULL DEFAULT '{}',
    secrets JSONB NOT NULL DEFAULT '{}',
    settings JSONB NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system',
    updated_by VARCHAR(255) DEFAULT 'system'
);

-- Profile versions table stores historical versions
CREATE TABLE IF NOT EXISTS profile_versions (
    id SERIAL PRIMARY KEY,
    profile_id VARCHAR(255) NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    scenario VARCHAR(255) NOT NULL,
    tiers JSONB NOT NULL DEFAULT '[]',
    swaps JSONB NOT NULL DEFAULT '{}',
    secrets JSONB NOT NULL DEFAULT '{}',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system',
    change_description TEXT,
    UNIQUE(profile_id, version)
);

-- Deployments table tracks deployment execution
CREATE TABLE IF NOT EXISTS deployments (
    id VARCHAR(255) PRIMARY KEY,
    profile_id VARCHAR(255) NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'queued',
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    artifacts JSONB DEFAULT '[]',
    message TEXT,
    logs TEXT,
    error TEXT
);

-- Deployment approvals for per-platform, per-commit release gating
CREATE TABLE IF NOT EXISTS deployment_approvals (
    id              TEXT PRIMARY KEY,
    profile_id      TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    git_commit_hash TEXT NOT NULL,
    platform        TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    approved_by     TEXT,
    approved_at     TIMESTAMPTZ,
    notes           TEXT,
    validation_id   TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, git_commit_hash, platform)
);

-- Required platforms per profile for release gating
CREATE TABLE IF NOT EXISTS profile_required_platforms (
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL,
    PRIMARY KEY (profile_id, platform)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_profiles_scenario ON profiles(scenario);
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_profile_versions_profile_id ON profile_versions(profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_versions_version ON profile_versions(profile_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_deployments_profile_id ON deployments(profile_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_deployments_started_at ON deployments(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_approvals_profile_commit ON deployment_approvals(profile_id, git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_approvals_pending ON deployment_approvals(status) WHERE status = 'pending';
