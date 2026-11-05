CREATE TABLE IF NOT EXISTS provider_connections (
  id TEXT PRIMARY KEY,
  provider TEXT NOT NULL,
  display_name TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  status TEXT NOT NULL,
  health_message TEXT NOT NULL DEFAULT '',
  read_only INTEGER NOT NULL DEFAULT 1,
  calendar_count INTEGER NOT NULL DEFAULT 0,
  imported_event_count INTEGER NOT NULL DEFAULT 0,
  busy_minutes INTEGER NOT NULL DEFAULT 0,
  revision INTEGER NOT NULL DEFAULT 1,
  last_sync_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_provider_connections_status ON provider_connections(status);

CREATE TABLE IF NOT EXISTS imported_events (
  id TEXT PRIMARY KEY,
  connection_id TEXT NOT NULL,
  remote_event_id TEXT NOT NULL,
  local_date TEXT NOT NULL,
  start_minutes INTEGER NOT NULL,
  duration_minutes INTEGER NOT NULL,
  title TEXT NOT NULL,
  busy INTEGER NOT NULL DEFAULT 1,
  status TEXT NOT NULL DEFAULT 'active',
  updated_at TEXT NOT NULL,
  UNIQUE(connection_id, remote_event_id)
);
CREATE INDEX IF NOT EXISTS idx_imported_events_date ON imported_events(local_date, status, busy);
