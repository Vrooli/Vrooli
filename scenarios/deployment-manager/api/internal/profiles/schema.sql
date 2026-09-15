CREATE TABLE IF NOT EXISTS profiles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    scenario TEXT NOT NULL,
    tiers TEXT NOT NULL DEFAULT '[]',
    swaps TEXT NOT NULL DEFAULT '{}',
    secrets TEXT NOT NULL DEFAULT '{}',
    settings TEXT NOT NULL DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT DEFAULT 'system',
    updated_by TEXT DEFAULT 'system'
);
CREATE TABLE IF NOT EXISTS profile_versions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    profile_id TEXT NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    name TEXT NOT NULL,
    scenario TEXT NOT NULL,
    tiers TEXT NOT NULL DEFAULT '[]',
    swaps TEXT NOT NULL DEFAULT '{}',
    secrets TEXT NOT NULL DEFAULT '{}',
    settings TEXT NOT NULL DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by TEXT DEFAULT 'system',
    change_description TEXT,
    UNIQUE(profile_id, version)
);
CREATE INDEX IF NOT EXISTS idx_profiles_scenario ON profiles(scenario);
CREATE INDEX IF NOT EXISTS idx_profiles_created_at ON profiles(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_profile_versions_profile_id ON profile_versions(profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_versions_version ON profile_versions(profile_id, version DESC);
CREATE TABLE IF NOT EXISTS profile_lpbs_release_config (
    profile_id TEXT PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    lpbs_domain TEXT NOT NULL DEFAULT '',
    lpbs_remote_profile TEXT NOT NULL DEFAULT '',
    lpbs_app_key TEXT NOT NULL DEFAULT '',
    default_channel TEXT NOT NULL DEFAULT 'stable',
    update_url TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
