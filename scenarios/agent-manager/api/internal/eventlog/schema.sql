-- ============================================================================
-- Run Events - Append-only event stream
-- ============================================================================
CREATE TABLE IF NOT EXISTS run_events (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL,
    sequence INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    timestamp TEXT DEFAULT (datetime('now')),
    -- schema_version identifies the on-wire shape of `data` so the eventlog
    -- dispatch table can route old payloads to old payload types
    -- indefinitely. Defaults to 1; the eventlog package is the source of
    -- truth for which versions are registered for which event types.
    schema_version INTEGER NOT NULL DEFAULT 1,
    data TEXT NOT NULL,
    UNIQUE(run_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_run_events_run_id ON run_events(run_id);
CREATE INDEX IF NOT EXISTS idx_run_events_run_sequence ON run_events(run_id, sequence);
CREATE INDEX IF NOT EXISTS idx_run_events_type ON run_events(run_id, event_type);
-- Conversation projection walks use this exact stable cursor order. Keeping
-- the partial index with the authoritative event table prevents a backfill
-- from repeatedly sorting the full append-only stream.
CREATE INDEX IF NOT EXISTS idx_run_events_conversation_source
    ON run_events(timestamp, run_id, sequence)
    WHERE event_type IN ('message', 'tool_call', 'tool_result');

-- Stats query indexes for cost aggregation
CREATE INDEX IF NOT EXISTS idx_run_events_cost ON run_events(run_id, event_type) WHERE event_type = 'metric';
CREATE INDEX IF NOT EXISTS idx_run_events_tool_calls ON run_events(run_id, event_type) WHERE event_type = 'tool_call';
CREATE INDEX IF NOT EXISTS idx_run_events_errors ON run_events(run_id, event_type) WHERE event_type = 'error';

-- A generation changes atomically whenever raw events are pruned. Durable
-- consumers bind their opaque cursors to it and reconcile instead of silently
-- skipping history after retention.
CREATE TABLE IF NOT EXISTS event_retention_state (
    singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
    generation INTEGER NOT NULL,
    floor_rowid INTEGER NOT NULL,
    updated_at TEXT NOT NULL
);
INSERT OR IGNORE INTO event_retention_state (singleton, generation, floor_rowid, updated_at)
VALUES (1, 1, 0, datetime('now'));
