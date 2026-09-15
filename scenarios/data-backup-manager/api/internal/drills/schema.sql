CREATE TABLE IF NOT EXISTS recovery_drills (
  id              TEXT PRIMARY KEY,
  plan_id         TEXT NOT NULL,
  target_id       TEXT NOT NULL,
  destination_id  TEXT NOT NULL,
  snapshot_id     TEXT NOT NULL DEFAULT '',
  restore_id      TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL,
  scheduled       INTEGER NOT NULL DEFAULT 0,
  idempotency_key TEXT NOT NULL DEFAULT '',
  error           TEXT NOT NULL DEFAULT '',
  next_action     TEXT NOT NULL DEFAULT '',
  requested_at    TEXT NOT NULL,
  started_at      TEXT NOT NULL DEFAULT '',
  finished_at     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_recovery_drills_unit
  ON recovery_drills(plan_id, target_id, destination_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_recovery_drills_list
  ON recovery_drills(plan_id, target_id, requested_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_recovery_drills_idempotency
  ON recovery_drills(idempotency_key) WHERE idempotency_key <> '';
