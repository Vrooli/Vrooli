-- Steering queue state — owned by pkg/steering, applied via
-- database.EnsureSchemas at boot (see pkg/dbschema). SQLite dialect. Timestamps
-- are RFC3339 strings (QueueState.CreatedAt/UpdatedAt are string fields).
CREATE TABLE IF NOT EXISTS steering_queue_state (
    task_id TEXT PRIMARY KEY,
    current_index INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL DEFAULT ''
);
