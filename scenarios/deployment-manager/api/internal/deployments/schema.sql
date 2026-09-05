CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'queued',
    started_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    artifacts TEXT DEFAULT '[]',
    message TEXT,
    logs TEXT,
    error TEXT
);
CREATE TABLE IF NOT EXISTS deployment_approvals (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    git_commit_hash TEXT NOT NULL,
    platform TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    approved_by TEXT,
    approved_at TIMESTAMP,
    notes TEXT,
    validation_id TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(profile_id, git_commit_hash, platform)
);
CREATE TABLE IF NOT EXISTS profile_required_platforms (
    profile_id TEXT NOT NULL,
    platform TEXT NOT NULL,
    PRIMARY KEY(profile_id, platform)
);
CREATE TABLE IF NOT EXISTS profile_required_targets (
    profile_id TEXT NOT NULL,
    ramp TEXT NOT NULL,
    platform TEXT NOT NULL,
    os TEXT NOT NULL,
    device_kind INTEGER NOT NULL,
    PRIMARY KEY(profile_id, ramp, platform, os, device_kind)
);
CREATE TABLE IF NOT EXISTS published_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    version TEXT NOT NULL,
    git_commit_hash TEXT,
    artifact_id INTEGER,
    deployment_id TEXT,
    release_id TEXT,
    published_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_deployments_profile_id ON deployments(profile_id);
CREATE INDEX IF NOT EXISTS idx_deployments_status ON deployments(status);
CREATE INDEX IF NOT EXISTS idx_approvals_profile_commit ON deployment_approvals(profile_id, git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_published_versions_profile_platform ON published_versions(profile_id, platform, published_at DESC);
