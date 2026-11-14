CREATE TABLE IF NOT EXISTS music_pool_takes (
  id TEXT PRIMARY KEY, job_id TEXT NOT NULL, style_id TEXT NOT NULL, pool_state TEXT NOT NULL,
  blob_ref TEXT NOT NULL, reserved_by TEXT NOT NULL DEFAULT '', reserved_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, model_id TEXT NOT NULL, license_lane TEXT NOT NULL, applied_rung TEXT NOT NULL,
  seed INTEGER NOT NULL, caption_as_authored TEXT NOT NULL, caption_as_sent TEXT NOT NULL, times_offered INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS style_inventory (
	  style_id TEXT PRIMARY KEY, target INTEGER NOT NULL, threshold INTEGER NOT NULL, last_replenish_job TEXT NOT NULL DEFAULT ''
);