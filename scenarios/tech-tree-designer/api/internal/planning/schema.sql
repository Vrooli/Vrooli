CREATE TABLE IF NOT EXISTS planned_scenario (
    slug TEXT PRIMARY KEY,
    display_name TEXT NOT NULL DEFAULT '',
    sector TEXT NOT NULL DEFAULT '',
    tier TEXT NOT NULL DEFAULT '',
    target_stability TEXT NOT NULL DEFAULT 'experimental',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS planned_proto_file (
    planned_slug TEXT NOT NULL,
    path TEXT NOT NULL,
    text TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (planned_slug, path),
    FOREIGN KEY (planned_slug) REFERENCES planned_scenario(slug) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS planned_scenario_sector_idx ON planned_scenario(sector);
CREATE INDEX IF NOT EXISTS planned_scenario_tier_idx ON planned_scenario(tier);
