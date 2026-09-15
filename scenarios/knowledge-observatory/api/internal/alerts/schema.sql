-- alerts domain schema (SQLite)
-- Quality alerts and notifications.
--
-- This domain owns these objects. Adding or removing the domain is a change
-- to this folder only; no central schema file declares them.

CREATE TABLE IF NOT EXISTS alerts (
    id TEXT PRIMARY KEY,
    level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'critical')),
    collection_name TEXT,
    metric_name TEXT,
    threshold_value REAL,
    actual_value REAL,
    message TEXT,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Partial index over unacknowledged alerts. Postgres wrote the predicate as
-- `WHERE NOT acknowledged`; SQLite stores booleans as integers and needs an
-- explicit comparison.
CREATE INDEX IF NOT EXISTS idx_alerts_level ON alerts(level) WHERE acknowledged = 0;
