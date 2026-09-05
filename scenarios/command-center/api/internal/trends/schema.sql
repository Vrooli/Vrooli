CREATE TABLE IF NOT EXISTS command_center_metric_observations (
  metric_id TEXT NOT NULL,
  source TEXT NOT NULL,
  value REAL NOT NULL,
  observed_at TEXT NOT NULL,
  recorded_at TEXT NOT NULL,
  PRIMARY KEY (metric_id, source, observed_at)
);
CREATE INDEX IF NOT EXISTS idx_command_center_metric_observations_lookup
  ON command_center_metric_observations (metric_id, source, observed_at);
