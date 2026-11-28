CREATE TABLE IF NOT EXISTS placement_plans (
    id TEXT PRIMARY KEY,
    entry TEXT NOT NULL,
    source TEXT NOT NULL,
    destination TEXT NOT NULL,
    created_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'preview'
);
CREATE TABLE IF NOT EXISTS placement_audit (
    id TEXT PRIMARY KEY,
    plan_id TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    event TEXT NOT NULL,
    source TEXT NOT NULL,
    destination TEXT NOT NULL,
    bytes INTEGER NOT NULL DEFAULT 0,
    verified INTEGER NOT NULL DEFAULT 0,
    source_preserved INTEGER NOT NULL DEFAULT 1,
    message TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_placement_audit_time ON placement_audit (occurred_at DESC);
