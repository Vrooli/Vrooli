-- Delivery catalog, artifacts, and storage configuration.
CREATE TABLE IF NOT EXISTS download_apps (id SERIAL PRIMARY KEY, bundle_key VARCHAR(100) NOT NULL, app_key VARCHAR(100) NOT NULL, name VARCHAR(255) NOT NULL, tagline TEXT, description TEXT, icon_url TEXT, screenshot_url TEXT, install_overview TEXT, install_steps JSONB DEFAULT '[]'::jsonb, storefronts JSONB DEFAULT '[]'::jsonb, metadata JSONB DEFAULT '{}'::jsonb, display_order INTEGER DEFAULT 0, update_api_key TEXT, update_policy JSONB NOT NULL DEFAULT '{"check_interval_hours":4,"update_mode":"optional","allow_downgrade":false}'::jsonb, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW(), UNIQUE (bundle_key, app_key));
CREATE INDEX IF NOT EXISTS idx_download_apps_bundle ON download_apps(bundle_key);
CREATE TABLE IF NOT EXISTS download_artifacts (id SERIAL PRIMARY KEY, bundle_key VARCHAR(100) NOT NULL, app_key VARCHAR(100), provider VARCHAR(50) NOT NULL DEFAULT 's3', bucket TEXT NOT NULL, object_key TEXT NOT NULL, etag TEXT, size_bytes BIGINT, sha256 TEXT, sha512 TEXT, content_type TEXT, original_filename TEXT, platform VARCHAR(50), release_version VARCHAR(50), release_id TEXT, git_commit_hash TEXT, metadata JSONB DEFAULT '{}'::jsonb, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW(), UNIQUE (bundle_key, bucket, object_key));
CREATE INDEX IF NOT EXISTS idx_download_artifacts_bundle ON download_artifacts(bundle_key); CREATE INDEX IF NOT EXISTS idx_download_artifacts_app_key ON download_artifacts(app_key); CREATE INDEX IF NOT EXISTS idx_download_artifacts_platform ON download_artifacts(platform); CREATE INDEX IF NOT EXISTS idx_download_artifacts_release_version ON download_artifacts(release_version); CREATE INDEX IF NOT EXISTS idx_download_artifacts_release_id ON download_artifacts(release_id);
CREATE TABLE IF NOT EXISTS download_assets (id SERIAL PRIMARY KEY, bundle_key VARCHAR(100) NOT NULL, app_key VARCHAR(100) NOT NULL, platform VARCHAR(50) NOT NULL CHECK (platform IN ('windows','mac','linux')), variant_key VARCHAR(50) NOT NULL DEFAULT 'default', artifact_url TEXT, artifact_source VARCHAR(20) NOT NULL DEFAULT 'direct', artifact_id INTEGER REFERENCES download_artifacts(id) ON DELETE SET NULL, release_version VARCHAR(50) NOT NULL, release_notes TEXT, checksum VARCHAR(255), requires_entitlement BOOLEAN DEFAULT FALSE, metadata JSONB DEFAULT '{}'::jsonb, display_order INTEGER DEFAULT 0, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW(), CONSTRAINT fk_download_app FOREIGN KEY (bundle_key, app_key) REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE);
CREATE UNIQUE INDEX IF NOT EXISTS idx_download_assets_bundle_app_platform_variant ON download_assets(bundle_key, app_key, platform, variant_key); CREATE INDEX IF NOT EXISTS idx_download_assets_artifact_id ON download_assets(artifact_id);
CREATE TABLE IF NOT EXISTS download_storage_settings (id SERIAL PRIMARY KEY, bundle_key VARCHAR(100) UNIQUE NOT NULL, provider VARCHAR(50) NOT NULL DEFAULT 's3', bucket TEXT, region TEXT, endpoint TEXT, force_path_style BOOLEAN DEFAULT FALSE, default_prefix TEXT, signed_url_ttl_seconds INTEGER DEFAULT 900, public_base_url TEXT, created_at TIMESTAMP DEFAULT NOW(), updated_at TIMESTAMP DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_download_storage_settings_bundle ON download_storage_settings(bundle_key);

-- Channel heads are the only mutable visibility pointer. Every promoted
-- artifact set is retained as an immutable revision so stale publishers can be
-- rejected by predecessor revision and operators can reconcile the history.
CREATE TABLE IF NOT EXISTS download_channel_revisions (
    id BIGSERIAL PRIMARY KEY,
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    variant_key VARCHAR(50) NOT NULL,
    revision BIGINT NOT NULL,
    predecessor_revision BIGINT NOT NULL,
    artifact_set JSONB NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (bundle_key, app_key, variant_key, revision),
    CONSTRAINT fk_channel_revision_app FOREIGN KEY (bundle_key, app_key)
        REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_download_channel_revisions_lookup
    ON download_channel_revisions(bundle_key, app_key, variant_key, revision DESC);

CREATE TABLE IF NOT EXISTS download_channel_heads (
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    variant_key VARCHAR(50) NOT NULL,
    current_revision BIGINT NOT NULL,
    artifact_set JSONB NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bundle_key, app_key, variant_key),
    CONSTRAINT fk_channel_head_app FOREIGN KEY (bundle_key, app_key)
        REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE
);

-- A halt is a durable offer gate. It does not mutate installed clients or
-- delete immutable revisions; it only prevents future feed and download use.
CREATE TABLE IF NOT EXISTS download_channel_halts (
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    variant_key VARCHAR(50) NOT NULL,
    revision BIGINT NOT NULL,
    halted BOOLEAN NOT NULL DEFAULT TRUE,
    reason TEXT NOT NULL,
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (bundle_key, app_key, variant_key),
    CONSTRAINT fk_channel_halt_app FOREIGN KEY (bundle_key, app_key)
        REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE
);
