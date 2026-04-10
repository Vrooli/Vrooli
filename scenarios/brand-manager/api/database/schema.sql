-- Brand Manager SQLite Schema
-- Idempotent: safe to run multiple times (IF NOT EXISTS patterns)
-- [REQ:BM-REQ-STORE-SCHEMA] Core tables for brands, versions, assignments, assets

CREATE TABLE IF NOT EXISTS brands (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    identity    TEXT NOT NULL DEFAULT '{}',  -- JSON: Identity facet
    colors      TEXT NOT NULL DEFAULT '{}',  -- JSON: Colors facet
    typography  TEXT NOT NULL DEFAULT '{}',  -- JSON: Typography facet
    voice       TEXT NOT NULL DEFAULT '{}',  -- JSON: Voice facet
    notes       TEXT NOT NULL DEFAULT '',
    version     INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS brand_versions (
    id         TEXT PRIMARY KEY,
    brand_id   TEXT NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    version    INTEGER NOT NULL,
    snapshot   TEXT NOT NULL,  -- JSON: full brand state at this version
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(brand_id, version)
);

CREATE TABLE IF NOT EXISTS assignments (
    id             TEXT PRIMARY KEY,
    brand_id       TEXT NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    scenario_name  TEXT NOT NULL,
    brand_version  INTEGER NOT NULL,
    elements       TEXT NOT NULL DEFAULT '[]',  -- JSON array of applied element names
    applied_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    UNIQUE(scenario_name)
);

CREATE TABLE IF NOT EXISTS assets (
    id        TEXT PRIMARY KEY,
    brand_id  TEXT NOT NULL REFERENCES brands(id) ON DELETE CASCADE,
    filename  TEXT NOT NULL,
    mime_type TEXT NOT NULL DEFAULT '',
    file_path TEXT NOT NULL,
    size      INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_brand_versions_brand_id ON brand_versions(brand_id);
CREATE INDEX IF NOT EXISTS idx_assignments_brand_id ON assignments(brand_id);
CREATE INDEX IF NOT EXISTS idx_assignments_scenario ON assignments(scenario_name);
CREATE INDEX IF NOT EXISTS idx_assets_brand_id ON assets(brand_id);
