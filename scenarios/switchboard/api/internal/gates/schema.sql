CREATE TABLE IF NOT EXISTS switchboard_capability_gates (
 id TEXT PRIMARY KEY,
 thread_id TEXT NOT NULL,
 owner_id TEXT NOT NULL,
 scope TEXT NOT NULL,
 withheld TEXT NOT NULL,
 unblock TEXT NOT NULL,
 created_at TEXT NOT NULL,
 expires_at TEXT NOT NULL,
 status TEXT NOT NULL,
 grant_once INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS switchboard_capability_gates_thread ON switchboard_capability_gates(thread_id, created_at);
