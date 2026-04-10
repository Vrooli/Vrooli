-- NOTE: This file documents the SQLite schema used by lifestyle-dashboard.
-- The actual schema is initialized in api/main.go for embedded SQLite.
-- This file is kept for reference and potential PostgreSQL migration.

-- Events table (P0-001): Common envelope with JSON payloads
-- Stores cross-domain events with optional causality markers
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,              -- UUID as text for SQLite compatibility
    timestamp TEXT NOT NULL,          -- ISO-8601 timestamp
    domain TEXT NOT NULL,             -- Source domain scenario (e.g., 'nootropics-tracker')
    event_type TEXT NOT NULL,         -- Event type within domain (e.g., 'supplement_taken')
    payload TEXT DEFAULT '{}',        -- JSON payload (TEXT for SQLite)
    is_intervention INTEGER DEFAULT 0, -- Causality marker: is this an intervention?
    hypothesis_id TEXT,               -- Link to experiment if intervention
    created_at TEXT NOT NULL          -- Record creation time
);

-- Indexes for efficient cross-domain queries (P0-003)
CREATE INDEX IF NOT EXISTS idx_events_domain_timestamp ON events(domain, timestamp);
CREATE INDEX IF NOT EXISTS idx_events_timestamp ON events(timestamp);
CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_hypothesis ON events(hypothesis_id) WHERE hypothesis_id IS NOT NULL;

-- Domains table (P0-002): Domain registration and discovery
-- Tracks registered domain scenarios with their capabilities and health status
CREATE TABLE IF NOT EXISTS domains (
    name TEXT PRIMARY KEY,            -- Domain scenario name (e.g., 'sleep-tracker')
    display_name TEXT NOT NULL,       -- Human-readable name
    description TEXT DEFAULT '',      -- Description of domain
    capabilities TEXT DEFAULT '[]',   -- JSON array of capabilities
    status TEXT DEFAULT 'active',     -- active, inactive, unhealthy
    health_url TEXT DEFAULT '',       -- URL for health checks
    last_health_at TEXT,              -- Last successful health check
    registered_at TEXT NOT NULL,      -- Registration timestamp
    updated_at TEXT NOT NULL          -- Last update timestamp
);

CREATE INDEX IF NOT EXISTS idx_domains_status ON domains(status);
