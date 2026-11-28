CREATE TABLE IF NOT EXISTS bridge_admin_dispatch_receipts (
 request_id TEXT PRIMARY KEY,
 external_request_id TEXT NOT NULL,
 caller TEXT NOT NULL,
 scenario TEXT NOT NULL,
 operation TEXT NOT NULL,
 node_id TEXT NOT NULL,
 target_tier TEXT NOT NULL CHECK (target_tier IN ('local','vps','test-device')),
 status TEXT NOT NULL CHECK (status IN ('requested','refused','uncertain','accepted','replayed')),
 run_id TEXT NOT NULL DEFAULT '',
 detail TEXT NOT NULL DEFAULT '',
 replayed BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_bridge_admin_receipts_caller_time ON bridge_admin_dispatch_receipts(caller, created_at DESC);
