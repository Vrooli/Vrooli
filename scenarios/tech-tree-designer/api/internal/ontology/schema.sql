CREATE TABLE IF NOT EXISTS capability (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    kind TEXT NOT NULL,
    parent_id TEXT NULL REFERENCES capability(id) ON DELETE SET NULL,
    sort_order INTEGER NOT NULL DEFAULT 0,
    importance REAL NOT NULL DEFAULT 1,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_capability_parent_id ON capability(parent_id);
CREATE INDEX IF NOT EXISTS idx_capability_kind ON capability(kind);

CREATE TABLE IF NOT EXISTS capability_edge (
    from_id TEXT NOT NULL,
    to_id TEXT NOT NULL,
    type TEXT NOT NULL,
    PRIMARY KEY (from_id, to_id, type),
    FOREIGN KEY (from_id) REFERENCES capability(id) ON DELETE CASCADE,
    FOREIGN KEY (to_id) REFERENCES capability(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS fulfillment (
    capability_id TEXT NOT NULL,
    scenario_slug TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    PRIMARY KEY (capability_id, scenario_slug),
    FOREIGN KEY (capability_id) REFERENCES capability(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fulfillment_scenario_slug ON fulfillment(scenario_slug);

CREATE TABLE IF NOT EXISTS coverage_exclusion (
    scenario_slug TEXT PRIMARY KEY,
    reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
