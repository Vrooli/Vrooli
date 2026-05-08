-- Vrooli Autoheal SQLite Schema
-- Primary persistence backend for desktop/portable deployments.

CREATE TABLE IF NOT EXISTS health_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok', 'warning', 'critical')),
    message TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '{}',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_health_results_check_id_created
    ON health_results (check_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_health_results_created_at
    ON health_results (created_at DESC);

CREATE TABLE IF NOT EXISTS host_inventory_snapshots (
    id TEXT PRIMARY KEY,
    collected_at TEXT NOT NULL,
    platform TEXT NOT NULL,
    os TEXT NOT NULL,
    arch TEXT NOT NULL,
    boot_id TEXT,
    kernel_release TEXT,
    fingerprint TEXT NOT NULL,
    inventory_json TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_host_inventory_snapshots_collected
    ON host_inventory_snapshots (collected_at DESC);

CREATE INDEX IF NOT EXISTS idx_host_inventory_snapshots_fingerprint
    ON host_inventory_snapshots (fingerprint);

CREATE TABLE IF NOT EXISTS host_inventory_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_snapshot_id TEXT,
    to_snapshot_id TEXT,
    change_type TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    summary TEXT NOT NULL,
    details_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (from_snapshot_id) REFERENCES host_inventory_snapshots(id),
    FOREIGN KEY (to_snapshot_id) REFERENCES host_inventory_snapshots(id)
);

CREATE INDEX IF NOT EXISTS idx_host_inventory_changes_created
    ON host_inventory_changes (created_at DESC);

CREATE TABLE IF NOT EXISTS incidents (
    id TEXT PRIMARY KEY,
    fingerprint TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK (type IN ('host_integrity', 'unclean_boot', 'resource_failure', 'scenario_failure', 'autoheal_failure', 'manual')),
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    status TEXT NOT NULL CHECK (status IN ('open', 'acknowledged', 'resolved', 'ignored')),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    detected_at TEXT NOT NULL,
    last_seen_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    resolved_at TEXT,
    acknowledged_at TEXT,
    ignored_at TEXT,
    boot_id TEXT,
    previous_boot_id TEXT,
    source_check_ids_json TEXT NOT NULL DEFAULT '[]',
    source_result_ids_json TEXT NOT NULL DEFAULT '[]',
    evidence_json TEXT NOT NULL DEFAULT '{}',
    recommendations_json TEXT NOT NULL DEFAULT '[]',
    event_count INTEGER NOT NULL DEFAULT 1,
    observation_count INTEGER NOT NULL DEFAULT 0,
    operator_notes TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_incidents_status_updated
    ON incidents (status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_incidents_type_severity
    ON incidents (type, severity, updated_at DESC);

CREATE TABLE IF NOT EXISTS incident_observations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id TEXT NOT NULL,
    observed_at TEXT NOT NULL,
    source_check_id TEXT,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    status TEXT,
    message TEXT NOT NULL,
    evidence_json TEXT NOT NULL DEFAULT '{}',
    FOREIGN KEY (incident_id) REFERENCES incidents(id)
);

CREATE INDEX IF NOT EXISTS idx_incident_observations_incident
    ON incident_observations (incident_id, observed_at DESC);

CREATE TABLE IF NOT EXISTS incident_status_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    incident_id TEXT NOT NULL,
    from_status TEXT,
    to_status TEXT NOT NULL CHECK (to_status IN ('open', 'acknowledged', 'resolved', 'ignored')),
    note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (incident_id) REFERENCES incidents(id)
);

CREATE INDEX IF NOT EXISTS idx_incident_status_history_incident
    ON incident_status_history (incident_id, created_at DESC);

CREATE TABLE IF NOT EXISTS autoheal_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    target TEXT NOT NULL,
    success INTEGER NOT NULL DEFAULT 0,
    message TEXT,
    details TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_autoheal_actions_created_at
    ON autoheal_actions (created_at DESC);

CREATE TABLE IF NOT EXISTS action_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    action_id TEXT NOT NULL,
    success INTEGER NOT NULL DEFAULT 0,
    message TEXT NOT NULL,
    output TEXT,
    error TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_action_logs_check_id_created
    ON action_logs (check_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_action_logs_created_at
    ON action_logs (created_at DESC);

CREATE TABLE IF NOT EXISTS autoheal_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE IF NOT EXISTS heal_trackers (
    check_id TEXT PRIMARY KEY,
    last_attempt TEXT,
    last_success TEXT,
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    total_attempts INTEGER NOT NULL DEFAULT 0,
    total_successes INTEGER NOT NULL DEFAULT 0,
    cooldown_until TEXT,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE VIEW IF NOT EXISTS latest_health_results AS
SELECT hr.*
FROM health_results hr
JOIN (
    SELECT check_id, MAX(created_at) AS max_created_at
    FROM health_results
    GROUP BY check_id
) latest
ON hr.check_id = latest.check_id AND hr.created_at = latest.max_created_at;

CREATE VIEW IF NOT EXISTS health_summary AS
SELECT
    COUNT(*) AS total_checks,
    SUM(CASE WHEN status = 'ok' THEN 1 ELSE 0 END) AS ok_count,
    SUM(CASE WHEN status = 'warning' THEN 1 ELSE 0 END) AS warning_count,
    SUM(CASE WHEN status = 'critical' THEN 1 ELSE 0 END) AS critical_count,
    CASE
        WHEN SUM(CASE WHEN status = 'critical' THEN 1 ELSE 0 END) > 0 THEN 'critical'
        WHEN SUM(CASE WHEN status = 'warning' THEN 1 ELSE 0 END) > 0 THEN 'warning'
        ELSE 'ok'
    END AS overall_status
FROM latest_health_results;
