-- Migration 005: Add LPBS release config and release-record tracking.
--
-- Introduces the deployment-manager-owned release lifecycle:
--   * profile_lpbs_release_config: 1:1 child of profiles holding LPBS
--     coordinates (domain, remote profile, app key, channel, update URL).
--   * releases: canonical record for a publish attempt, tied to a profile,
--     deployment, git commit, and channel. DM allocates release_id (UUID).
--   * release_platforms: per-platform publish/verify state for a release.

CREATE TABLE IF NOT EXISTS profile_lpbs_release_config (
    profile_id         TEXT PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    lpbs_domain        TEXT NOT NULL DEFAULT '',
    lpbs_remote_profile TEXT NOT NULL DEFAULT '',
    lpbs_app_key       TEXT NOT NULL DEFAULT '',
    default_channel    TEXT NOT NULL DEFAULT 'stable',
    update_url         TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS releases (
    id                        TEXT PRIMARY KEY,
    profile_id                TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    deployment_id             TEXT,
    profile_version           INTEGER,
    git_commit_hash           TEXT NOT NULL,
    release_version           TEXT NOT NULL,
    channel                   TEXT NOT NULL DEFAULT 'stable',
    status                    TEXT NOT NULL DEFAULT 'pending',
    release_notes             TEXT,
    released_by               TEXT,
    promoted_from_release_id  TEXT REFERENCES releases(id) ON DELETE SET NULL,
    verification_evidence     JSONB,
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at              TIMESTAMPTZ,
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (profile_id, git_commit_hash, channel)
);

CREATE TABLE IF NOT EXISTS release_platforms (
    release_id        TEXT NOT NULL REFERENCES releases(id) ON DELETE CASCADE,
    platform          TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'pending',
    approval_id       TEXT,
    lpbs_artifact_id  BIGINT,
    published_at      TIMESTAMPTZ,
    verified_at       TIMESTAMPTZ,
    error             TEXT,
    PRIMARY KEY (release_id, platform)
);

CREATE INDEX IF NOT EXISTS idx_releases_profile_channel
    ON releases (profile_id, channel, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_releases_status
    ON releases (status);
CREATE INDEX IF NOT EXISTS idx_releases_commit
    ON releases (profile_id, git_commit_hash);
CREATE INDEX IF NOT EXISTS idx_releases_deployment
    ON releases (deployment_id);
CREATE INDEX IF NOT EXISTS idx_release_platforms_status
    ON release_platforms (status);

-- Correlate legacy published_versions rows to the owning release when possible.
ALTER TABLE published_versions
    ADD COLUMN IF NOT EXISTS release_id TEXT REFERENCES releases(id) ON DELETE SET NULL;
