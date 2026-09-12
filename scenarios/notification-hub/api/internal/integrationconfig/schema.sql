CREATE TABLE IF NOT EXISTS event_integration_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    events_api_base TEXT NOT NULL DEFAULT '',
    webhook_url TEXT NOT NULL DEFAULT '',
    pattern TEXT NOT NULL DEFAULT '',
    templates_json TEXT NOT NULL DEFAULT '{}',
    sensitivity_by_severity_json TEXT NOT NULL DEFAULT '{}',
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
