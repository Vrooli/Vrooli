-- Fleet dependency & vulnerability intelligence corpus. One row per
-- (scenario, ecosystem, name, version) tuple; vuln annotations and last_seen
-- are refreshed on each reconcile. Declarative schema (greenfield contract):
-- no migrations/ folder — this file is the single source of truth.

CREATE TABLE IF NOT EXISTS dependency_records (
    dep_key      TEXT PRIMARY KEY,            -- ecosystem|scenario|name|version
    scenario     TEXT NOT NULL,
    ecosystem    TEXT NOT NULL,               -- "go" | "npm"
    name         TEXT NOT NULL,
    version      TEXT NOT NULL,
    source_file  TEXT NOT NULL,
    vuln_ids     TEXT NOT NULL DEFAULT '',    -- comma-separated OSV/GHSA ids
    max_severity TEXT NOT NULL DEFAULT '',    -- "" | low | moderate | high | critical
    last_seen    TEXT NOT NULL                -- RFC3339
);

CREATE INDEX IF NOT EXISTS idx_dep_scenario ON dependency_records(scenario);
CREATE INDEX IF NOT EXISTS idx_dep_name     ON dependency_records(name);
CREATE INDEX IF NOT EXISTS idx_dep_eco      ON dependency_records(ecosystem);
CREATE INDEX IF NOT EXISTS idx_dep_vuln     ON dependency_records(max_severity);

CREATE TABLE IF NOT EXISTS vulnerability_records (
    vuln_key            TEXT PRIMARY KEY,         -- vulnerability_id|ecosystem|name|version|scenario|source_file|source
    vulnerability_id    TEXT NOT NULL,
    aliases             TEXT NOT NULL DEFAULT '', -- comma-separated aliases
    ecosystem           TEXT NOT NULL,
    name                TEXT NOT NULL,
    version             TEXT NOT NULL,
    affected_ranges     TEXT NOT NULL DEFAULT '[]', -- JSON array
    fixed_ranges        TEXT NOT NULL DEFAULT '[]', -- JSON array
    severity            TEXT NOT NULL DEFAULT '',
    normalized_severity TEXT NOT NULL DEFAULT '',
    advisory_url        TEXT NOT NULL DEFAULT '',
    summary             TEXT NOT NULL DEFAULT '',
    details             TEXT NOT NULL DEFAULT '',
    source              TEXT NOT NULL DEFAULT '',
    reachability        TEXT NOT NULL DEFAULT '',
    confidence          TEXT NOT NULL DEFAULT '',
    production          INTEGER NOT NULL DEFAULT 0,
    dev_only            INTEGER NOT NULL DEFAULT 0,
    first_seen          TEXT NOT NULL,
    last_seen           TEXT NOT NULL,
    scenario            TEXT NOT NULL,
    source_file         TEXT NOT NULL,
    remediation         TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_vuln_id       ON vulnerability_records(vulnerability_id);
CREATE INDEX IF NOT EXISTS idx_vuln_package  ON vulnerability_records(ecosystem, name);
CREATE INDEX IF NOT EXISTS idx_vuln_scenario ON vulnerability_records(scenario);
CREATE INDEX IF NOT EXISTS idx_vuln_conf     ON vulnerability_records(confidence);

CREATE TABLE IF NOT EXISTS dependency_reconcile_state (
    id                INTEGER PRIMARY KEY CHECK (id = 1),
    last_reconcile_at TEXT,
    last_outcome      TEXT
);
