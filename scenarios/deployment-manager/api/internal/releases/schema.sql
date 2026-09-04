CREATE TABLE IF NOT EXISTS releases (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    deployment_id TEXT,
    profile_version INTEGER,
    git_commit_hash TEXT NOT NULL,
    artifact_digest TEXT,
    readiness_review_key TEXT,
    release_version TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'stable',
    status TEXT NOT NULL DEFAULT 'pending',
    release_notes TEXT,
    released_by TEXT,
    promoted_from_release_id TEXT,
    readiness_goal_ref TEXT,
    approved_at_commit TEXT,
    verification_evidence TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(profile_id, git_commit_hash, channel)
);
CREATE TABLE IF NOT EXISTS release_platforms (
    release_id TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    platform TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    approval_id TEXT,
    lpbs_artifact_id INTEGER,
    published_at TIMESTAMP,
    verified_at TIMESTAMP,
    error TEXT,
    PRIMARY KEY(release_id, platform)
);
CREATE INDEX IF NOT EXISTS idx_releases_profile_channel ON releases(profile_id, channel, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_releases_status ON releases(status);
CREATE INDEX IF NOT EXISTS idx_releases_commit ON releases(profile_id, git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_releases_deployment ON releases(deployment_id);
