-- Metadata only. LPBS owns account balances and reservation state; this table
-- preserves Audio Tools' correlation and last-known phase across restarts.
CREATE TABLE IF NOT EXISTS duplex_accounting_operations (
  operation_id TEXT PRIMARY KEY,
  owner_key TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  model_id TEXT NOT NULL,
  budget_micros INTEGER NOT NULL,
  phase TEXT NOT NULL CHECK (phase IN ('reserve_pending','reserved','dispatch_pending','dispatched','settlement_pending','settled','uncertain','released','expired')),
  provider_receipt_id TEXT NOT NULL DEFAULT '',
  cost_micros INTEGER NOT NULL DEFAULT 0,
  active_seconds INTEGER NOT NULL DEFAULT 0,
  paused_seconds INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_duplex_accounting_recovery ON duplex_accounting_operations(owner_key, phase, updated_at);
