CREATE TABLE IF NOT EXISTS design_capture_operations (
 id TEXT PRIMARY KEY,
 request_hash TEXT NOT NULL,
 request_json TEXT NOT NULL,
 state TEXT NOT NULL CHECK(state IN ('prepared','dispatching','dispatch_unknown','running','cancel_requested','cancelled','completed','failed')),
 producer_id TEXT NOT NULL DEFAULT '',
 artifacts_json TEXT NOT NULL DEFAULT '[]',
 detail TEXT NOT NULL DEFAULT '',
 version INTEGER NOT NULL DEFAULT 1 CHECK(version > 0)
);
