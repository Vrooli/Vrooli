CREATE TABLE IF NOT EXISTS event_integration_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    events_api_base TEXT NOT NULL DEFAULT '',
    webhook_url TEXT NOT NULL DEFAULT '',
    pattern TEXT NOT NULL DEFAULT '',
    templates_json TEXT NOT NULL DEFAULT '{}',
    sensitivity_by_severity_json TEXT NOT NULL DEFAULT '{}',
    decision_sensitivity TEXT NOT NULL DEFAULT 'private',
    decision_default_deadline_seconds INTEGER NOT NULL DEFAULT 86400,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
