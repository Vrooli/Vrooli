-- Download domain schema.
--
-- Install experiences (download_apps) and their per-platform artifacts
-- (download_assets, FK to the app). Assets may require an active entitlement,
-- enforced by the authorizer. Forward-only declarative DDL.
CREATE TABLE IF NOT EXISTS download_apps (
    id SERIAL PRIMARY KEY,
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    tagline TEXT,
    description TEXT,
    install_overview TEXT,
    install_steps JSONB DEFAULT '[]'::jsonb,
    storefronts JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (bundle_key, app_key)
);
CREATE INDEX IF NOT EXISTS idx_download_apps_bundle ON download_apps(bundle_key);

CREATE TABLE IF NOT EXISTS download_assets (
    id SERIAL PRIMARY KEY,
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('windows','mac','linux')),
    artifact_url TEXT NOT NULL,
    release_version VARCHAR(50) NOT NULL,
    release_notes TEXT,
    checksum VARCHAR(255),
    requires_entitlement BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_download_app FOREIGN KEY (bundle_key, app_key)
        REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_download_assets_bundle_app_platform ON download_assets(bundle_key, app_key, platform);
