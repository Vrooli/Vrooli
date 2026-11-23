-- Enabled-state overrides gain provenance (who, when, why, pinned, the seed flag
-- at write time) so a stale override cannot silently shadow a seed that moved on
-- (bug knw-1795312936997943545). The new tables are declared in the domain
-- schemas (model_override, adapter_override); this migration carries the old rows
-- across and drops the old tables, which a declarative schema cannot express.
--
-- A migration must not depend on the order schemas were applied, so it creates
-- the target tables itself (the shape frozen at this migration; the domain
-- schemas declare the same shape and the boot drift check compares them) and
-- creates each old table empty first, which makes the copy a no-op on a database
-- that never had it and the whole file safe to re-run after a crash. Copied rows
-- keep their effect: their author is unknown and their seed baseline is -1 until
-- the boot-time Reconcile adopts them.
CREATE TABLE IF NOT EXISTS model_override (
  id TEXT PRIMARY KEY, enabled INTEGER NOT NULL, set_by TEXT NOT NULL, set_at TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '', pinned INTEGER NOT NULL DEFAULT 0, seed_enabled INTEGER NOT NULL DEFAULT -1,
  superseded_at TEXT NOT NULL DEFAULT '', superseded_note TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS model_state (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL);
INSERT OR IGNORE INTO model_override (id, enabled, set_by, set_at, reason, pinned, seed_enabled)
  SELECT id, enabled, 'unknown (before provenance)', '', '', 0, -1 FROM model_state;
DROP TABLE model_state;

CREATE TABLE IF NOT EXISTS adapter_override (
  id TEXT PRIMARY KEY, enabled INTEGER NOT NULL, set_by TEXT NOT NULL, set_at TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL DEFAULT '', pinned INTEGER NOT NULL DEFAULT 0, seed_enabled INTEGER NOT NULL DEFAULT -1,
  superseded_at TEXT NOT NULL DEFAULT '', superseded_note TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS adapter_state (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL);
INSERT OR IGNORE INTO adapter_override (id, enabled, set_by, set_at, reason, pinned, seed_enabled)
  SELECT id, enabled, 'unknown (before provenance)', '', '', 0, -1 FROM adapter_state;
DROP TABLE adapter_state;
