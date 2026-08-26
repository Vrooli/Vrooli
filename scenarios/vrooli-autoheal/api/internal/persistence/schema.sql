-- Vrooli Autoheal SQLite Schema
-- Primary persistence backend for desktop/portable deployments.

CREATE TABLE IF NOT EXISTS health_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    check_id TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok', 'warning', 'critical', 'not-applicable')),
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
    last_seen_at TEXT NOT NULL,
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

CREATE TABLE IF NOT EXISTS system_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fingerprint TEXT NOT NULL UNIQUE,
    occurred_at TEXT NOT NULL,
    ingested_at TEXT NOT NULL,
    source TEXT NOT NULL,
    platform TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL CHECK (severity IN ('info', 'warning', 'critical')),
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    boot_id TEXT,
    details_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_system_events_occurred
    ON system_events (occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_system_events_category_occurred
    ON system_events (category, occurred_at DESC);

CREATE INDEX IF NOT EXISTS idx_system_events_source_occurred
    ON system_events (source, occurred_at DESC);

CREATE TABLE IF NOT EXISTS system_event_sources (
    source TEXT PRIMARY KEY,
    platform TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('ok', 'unsupported', 'degraded', 'failed')),
    last_ingested_at TEXT NOT NULL,
    last_error TEXT NOT NULL DEFAULT '',
    capabilities_json TEXT NOT NULL DEFAULT '{}'
);

-- Incremental ingest progress for journal-backed sources. Keyed by a logical
-- source key (e.g. "journalctl/kernel"). `cursor` is the journald __CURSOR of
-- the last successfully-ingested entry on the current boot; it advances ONLY
-- after a successful ingest so a failed read never skips events. `boot_id`
-- pins the cursor to a boot so a reboot forces a fresh read rather than
-- replaying a stale cursor against a rotated journal.
CREATE TABLE IF NOT EXISTS journal_cursors (
    source_key TEXT PRIMARY KEY,
    cursor TEXT NOT NULL DEFAULT '',
    boot_id TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);

-- Per-boot "already scanned" markers so immutable historical boots are grepped
-- at most once instead of every ingest. `source_key` namespaces the marker to
-- a logical scan family so independent passes don't collide.
CREATE TABLE IF NOT EXISTS journal_scanned_boots (
    source_key TEXT NOT NULL,
    boot_id TEXT NOT NULL,
    scanned_at TEXT NOT NULL,
    PRIMARY KEY (source_key, boot_id)
);

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
    diagnosis TEXT NOT NULL DEFAULT '',
    confidence TEXT NOT NULL DEFAULT '',
    evidence_items_json TEXT NOT NULL DEFAULT '[]',
    corroboration_needed_json TEXT NOT NULL DEFAULT '[]',
    safe_actions_json TEXT NOT NULL DEFAULT '[]',
    operator_actions_json TEXT NOT NULL DEFAULT '[]',
    rollback_or_fallback_json TEXT NOT NULL DEFAULT '[]',
    post_checks_json TEXT NOT NULL DEFAULT '[]',
    remediation_candidates_json TEXT NOT NULL DEFAULT '[]',
    remediation_artifacts_json TEXT NOT NULL DEFAULT '[]',
    outcome_json TEXT,
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

-- Retention orders deletes by observed_at across the WHOLE table, which the
-- composite index above cannot serve: it is led by incident_id, so a query with
-- no incident_id predicate has to scan and sort. Every other budgeted table in
-- this schema carries a dedicated index on its time column for the same reason —
-- without one, each prune batch reads the entire table, which is how batched
-- deletion degrades to ~330 rows/sec on a large file.
CREATE INDEX IF NOT EXISTS idx_incident_observations_observed
    ON incident_observations (observed_at);

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

-- A notification answer is a one-time authorization for one incident and one
-- generated candidate. Keeping this separate from incident notes makes replay
-- detection durable and prevents a caller from turning an arbitrary ask id
-- into permission for a different remediation.
CREATE TABLE IF NOT EXISTS remediation_authorisations (
    ask_id TEXT PRIMARY KEY,
    incident_id TEXT NOT NULL,
    incident_fingerprint TEXT NOT NULL,
    remediation_id TEXT NOT NULL,
    approved_by TEXT NOT NULL DEFAULT '',
    approved_at TEXT NOT NULL,
    consumed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_remediation_authorisations_incident
    ON remediation_authorisations (incident_id, remediation_id);

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
    timed_out INTEGER NOT NULL DEFAULT 0,
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

CREATE TABLE IF NOT EXISTS check_shelves (
    check_id TEXT PRIMARY KEY,
    reason TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    set_by TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_check_shelves_expires_at
    ON check_shelves (expires_at);

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
