CREATE TABLE IF NOT EXISTS music_pool_takes (
  id TEXT PRIMARY KEY, job_id TEXT NOT NULL, style_id TEXT NOT NULL, pool_state TEXT NOT NULL,
  blob_ref TEXT NOT NULL, reserved_by TEXT NOT NULL DEFAULT '', reserved_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL, model_id TEXT NOT NULL, license_lane TEXT NOT NULL, applied_rung TEXT NOT NULL,
  seed INTEGER NOT NULL, caption_as_authored TEXT NOT NULL, caption_as_sent TEXT NOT NULL, times_offered INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS style_inventory (
	  style_id TEXT PRIMARY KEY, target INTEGER NOT NULL, threshold INTEGER NOT NULL, last_replenish_job TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS music_candidate_refill_requests (
  request_id TEXT PRIMARY KEY,
  reservation_id TEXT NOT NULL UNIQUE,
  sha256 TEXT NOT NULL,
  purpose TEXT NOT NULL,
  project TEXT NOT NULL,
  revision TEXT NOT NULL,
  style_id TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('held','dispatched','cancelled')),
  job_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS music_candidate_refill_queue
  ON music_candidate_refill_requests(state,created_at);
