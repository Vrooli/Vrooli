-- Model storage lifecycle state — owned by internal/modelstorage/. Embedded by
-- store.go and applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry. Forward-only declarative.
--
-- model_keep records the operator's explicit "keep these weights" pins. A kept
-- model is never offloaded by the adaptive policy or a targeted offload; it can
-- still be removed explicitly with `image-tools models remove`.
CREATE TABLE IF NOT EXISTS model_keep (
  model_id   TEXT PRIMARY KEY,
  reason     TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);

-- model_offload_receipt is the durable record of every offload attempt made by
-- the policy (periodic, pre-acquisition or operator-requested): what was
-- removed or why removal failed, the evidence the decision used, and the
-- headroom it observed before and after.
CREATE TABLE IF NOT EXISTS model_offload_receipt (
  id                     TEXT PRIMARY KEY,
  model_id               TEXT NOT NULL,
  trigger                TEXT NOT NULL,
  reason                 TEXT NOT NULL,
  outcome                TEXT NOT NULL,
  error                  TEXT NOT NULL DEFAULT '',
  bytes                  INTEGER NOT NULL DEFAULT 0,
  checksum               TEXT NOT NULL DEFAULT '',
  last_use_at            TEXT NOT NULL DEFAULT '',
  last_use_basis         TEXT NOT NULL DEFAULT '',
  idle_seconds           INTEGER NOT NULL DEFAULT 0,
  idle_threshold_seconds INTEGER NOT NULL DEFAULT -1,
  mode                   TEXT NOT NULL DEFAULT '',
  free_before_bytes      INTEGER NOT NULL DEFAULT 0,
  free_after_bytes       INTEGER NOT NULL DEFAULT 0,
  created_at             TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS model_offload_receipt_created
  ON model_offload_receipt(created_at);
