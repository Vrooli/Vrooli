CREATE TABLE IF NOT EXISTS device_control_retained_runs (
 run_id TEXT PRIMARY KEY,
 device_id TEXT NOT NULL,
 payload TEXT NOT NULL
);
