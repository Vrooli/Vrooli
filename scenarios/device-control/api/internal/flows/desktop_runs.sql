CREATE TABLE IF NOT EXISTS device_control_desktop_flow_runs (
 actor TEXT NOT NULL,
 lease_id TEXT NOT NULL,
 run_id TEXT NOT NULL,
 payload TEXT NOT NULL,
 PRIMARY KEY(actor, lease_id, run_id)
);
CREATE TABLE IF NOT EXISTS device_control_desktop_saved_flows (
 actor TEXT NOT NULL,
 id TEXT NOT NULL,
 version INTEGER NOT NULL,
 source_lease_id TEXT NOT NULL,
 source_run_id TEXT NOT NULL,
 context_key TEXT NOT NULL,
 payload TEXT NOT NULL,
 PRIMARY KEY(actor, id, version),
 UNIQUE(actor, source_lease_id, source_run_id, context_key)
);
