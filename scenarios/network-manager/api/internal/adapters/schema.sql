CREATE TABLE IF NOT EXISTS adapter_capabilities (
    id TEXT PRIMARY KEY,
    adapter TEXT NOT NULL,
    action TEXT NOT NULL,
    supported INTEGER NOT NULL CHECK (supported IN (0, 1)),
    requires_admin INTEGER NOT NULL CHECK (requires_admin IN (0, 1)),
    rollback_supported INTEGER NOT NULL CHECK (rollback_supported IN (0, 1)),
    reason TEXT NOT NULL,
    observed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_adapter_capabilities_observed_at
ON adapter_capabilities(observed_at DESC);

CREATE INDEX IF NOT EXISTS idx_adapter_capabilities_action
ON adapter_capabilities(action, observed_at DESC);

CREATE TABLE IF NOT EXISTS adapter_platform_summaries (
    id TEXT PRIMARY KEY,
    os TEXT NOT NULL,
    arch TEXT NOT NULL,
    profile TEXT NOT NULL,
    notes_json TEXT NOT NULL,
    observed_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_adapter_platform_summaries_observed_at
ON adapter_platform_summaries(observed_at DESC);
