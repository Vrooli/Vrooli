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

-- Stats query indexes for cost aggregation
CREATE INDEX IF NOT EXISTS idx_run_events_cost ON run_events(run_id, event_type) WHERE event_type = 'metric';
CREATE INDEX IF NOT EXISTS idx_run_events_tool_calls ON run_events(run_id, event_type) WHERE event_type = 'tool_call';
CREATE INDEX IF NOT EXISTS idx_run_events_errors ON run_events(run_id, event_type) WHERE event_type = 'error';
