-- Migration 003: Add deployment approvals for per-platform, per-commit release gating.
--
-- Each approval is tied to a (profile_id, git_commit_hash, platform) triple.
-- When a new commit is built, previous approvals for that profile+platform are
-- automatically marked "stale". Deployments are blocked until all required
-- platforms show "approved" for the same commit.

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

CREATE INDEX IF NOT EXISTS idx_approvals_profile_commit
    ON deployment_approvals (profile_id, git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_approvals_pending
    ON deployment_approvals (status) WHERE status = 'pending';

-- Track which platforms must be approved before a profile can be deployed.
-- Configurable per profile via PUT /api/v1/profiles/{id}/required-platforms.
CREATE TABLE IF NOT EXISTS profile_required_platforms (
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    platform   TEXT NOT NULL,
    PRIMARY KEY (profile_id, platform)
);
