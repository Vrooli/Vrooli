CREATE TABLE IF NOT EXISTS assisted_workflows (
  id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  asset_id TEXT NOT NULL DEFAULT '',
  source_scenario TEXT NOT NULL DEFAULT '',
  target_scenario TEXT NOT NULL DEFAULT '',
  source_path TEXT NOT NULL DEFAULT '',
  requested_version TEXT NOT NULL DEFAULT '',
  agent_manager_task_id TEXT NOT NULL DEFAULT '',
  agent_manager_run_id TEXT NOT NULL DEFAULT '',
  agent_manager_execution_id TEXT NOT NULL DEFAULT '',
  idempotency_key TEXT NOT NULL,
  status TEXT NOT NULL,
  last_event_sequence INTEGER NOT NULL DEFAULT 0,
  summary TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_assisted_workflows_active_idempotency
  ON assisted_workflows(idempotency_key) WHERE status IN ('queued', 'running', 'parked');
CREATE INDEX IF NOT EXISTS idx_assisted_workflows_asset_updated
  ON assisted_workflows(asset_id, updated_at DESC);
