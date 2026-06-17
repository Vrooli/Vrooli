-- Transfer domain — owned by internal/transfer/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (through modules.AllSchemas).
-- Times are RFC3339Nano strings matching the wire format and the time.Time
-- round-trip in sqlite.go. Forward-only: CREATE TABLE IF NOT EXISTS so re-runs
-- are no-ops; column additions land as `ALTER TABLE ... ADD COLUMN` (never
-- recreate — Vrooli storage rule).

-- An item is one transferred payload (file or text), keyed to the owner's trust
-- group. text_content holds the inline body for text items; blob_key locates the
-- bytes in api-core/blobstore for file items. target_device_id is '' for a
-- broadcast item, or a device id for a directed one. expires_at is '' for Pinned
-- items; otherwise the purge sweep removes the row (and its blob) once now passes it.
CREATE TABLE IF NOT EXISTS items (
  id               TEXT PRIMARY KEY,
  owner_id         TEXT NOT NULL,
  origin_device_id TEXT NOT NULL,
  kind             TEXT NOT NULL,
  name             TEXT NOT NULL DEFAULT '',
  mime             TEXT NOT NULL DEFAULT '',
  size_bytes       INTEGER NOT NULL DEFAULT 0,
  text_content     TEXT NOT NULL DEFAULT '',
  blob_key         TEXT NOT NULL DEFAULT '',
  thumb_key        TEXT NOT NULL DEFAULT '',
  retention        TEXT NOT NULL,
  target_device_id TEXT NOT NULL DEFAULT '',
  delivered        INTEGER NOT NULL DEFAULT 0,
  expires_at       TEXT NOT NULL DEFAULT '',
  created_at       TEXT NOT NULL
);

-- Primary list path: the owner's items newest-first.
CREATE INDEX IF NOT EXISTS idx_items_owner_created
  ON items(owner_id, created_at DESC);

-- Purge sweep scans items with a non-empty expires_at in time order.
CREATE INDEX IF NOT EXISTS idx_items_expires
  ON items(expires_at);
