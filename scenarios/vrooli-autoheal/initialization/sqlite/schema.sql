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
