CREATE TABLE IF NOT EXISTS inventory_events (
  workspace_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  item_id TEXT NOT NULL DEFAULT '',
  batch_id TEXT NOT NULL DEFAULT '',
  amount TEXT NOT NULL,
  unit TEXT NOT NULL,
  recipe_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_inventory_events_workspace_time ON inventory_events(workspace_id, created_at, event_id);
CREATE TABLE IF NOT EXISTS inventory_batches (
  workspace_id TEXT NOT NULL,
  batch_id TEXT NOT NULL,
  recipe_id TEXT NOT NULL DEFAULT '',
  recipe_revision INTEGER NOT NULL DEFAULT 0,
  yield_amount TEXT NOT NULL,
  available_amount TEXT NOT NULL,
  unit TEXT NOT NULL,
  PRIMARY KEY(workspace_id, batch_id)
);
CREATE TABLE IF NOT EXISTS inventory_receipt_proposals (
  workspace_id TEXT NOT NULL,
  proposal_id TEXT NOT NULL,
  source_id TEXT NOT NULL,
  transaction_id TEXT NOT NULL,
  line_key TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  item_id TEXT NOT NULL,
  amount TEXT NOT NULL,
  unit TEXT NOT NULL,
  price TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  event_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  applied_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(workspace_id, proposal_id),
  UNIQUE(workspace_id, source_id, transaction_id, line_key)
);
CREATE INDEX IF NOT EXISTS idx_inventory_receipt_proposals_workspace_status ON inventory_receipt_proposals(workspace_id, status, created_at);
