CREATE TABLE IF NOT EXISTS local_work_native_owner (
 singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
 scope_sha256 TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS local_work_native_receipts (
 event_id TEXT PRIMARY KEY,
 event_type TEXT NOT NULL,
 source_scenario TEXT NOT NULL,
 body_sha256 TEXT NOT NULL,
 received_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS local_work_native_selection (
 singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
 scope_sha256 TEXT NOT NULL,
 request_key TEXT NOT NULL,
 admission_deadline TEXT NOT NULL,
 receipts_json TEXT NOT NULL
);

-- Terminal collector reconciliation is not cloud delivery. Receipts and these
-- tombstones are never pruned; the receipt capacity bounds retained metadata.
CREATE TABLE IF NOT EXISTS local_work_native_completed (
 request_key TEXT PRIMARY KEY,
 scope_sha256 TEXT NOT NULL,
 admission_deadline TEXT NOT NULL,
 receipts_json TEXT NOT NULL,
 program_identity_json TEXT NOT NULL,
 program_status TEXT NOT NULL,
 stdout_sha256 TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS local_work_native_completed_events (
 event_id TEXT PRIMARY KEY REFERENCES local_work_native_receipts(event_id),
 request_key TEXT NOT NULL REFERENCES local_work_native_completed(request_key)
);

CREATE TABLE IF NOT EXISTS local_work_native_bindings (
 request_key TEXT PRIMARY KEY,
 events_sha256 TEXT NOT NULL,
 collector_sha256 TEXT NOT NULL
);
