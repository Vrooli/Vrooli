-- ============================================================================
-- Policies - Rules governing execution
-- ============================================================================
CREATE TABLE IF NOT EXISTS policies (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    priority INTEGER DEFAULT 0,
    scope_pattern TEXT,
    rules TEXT NOT NULL DEFAULT '{}',
    enabled INTEGER DEFAULT 1,
    created_by TEXT,
    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_policies_enabled ON policies(enabled);
CREATE TRIGGER IF NOT EXISTS update_policies_updated_at
    AFTER UPDATE ON policies
    FOR EACH ROW
BEGIN
    UPDATE policies SET updated_at = datetime('now') WHERE id = NEW.id;
END;
