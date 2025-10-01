-- External calendar integration tables
-- Stores OAuth tokens and sync configuration for Google Calendar and Outlook

-- External calendar connections
CREATE TABLE IF NOT EXISTS external_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('google', 'outlook')),
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    calendar_id VARCHAR(255),
    sync_enabled BOOLEAN DEFAULT true,
    sync_direction VARCHAR(20) DEFAULT 'bidirectional' CHECK (sync_direction IN ('bidirectional', 'import_only', 'export_only')),
    last_sync_time TIMESTAMPTZ,
    sync_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, provider)
);

-- OAuth state tokens for verification
CREATE TABLE IF NOT EXISTS oauth_states (
    state VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- External sync log for tracking sync history
CREATE TABLE IF NOT EXISTS external_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID REFERENCES external_calendars(id) ON DELETE CASCADE,
    sync_type VARCHAR(20) CHECK (sync_type IN ('manual', 'scheduled', 'webhook')),
    direction VARCHAR(20) CHECK (direction IN ('import', 'export', 'bidirectional')),
    events_created INTEGER DEFAULT 0,
    events_updated INTEGER DEFAULT 0,
    events_deleted INTEGER DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    error_details JSONB,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'success', 'partial', 'failed'))
);

-- External event mapping to track synced events
CREATE TABLE IF NOT EXISTS external_event_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    local_event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    external_metadata JSONB DEFAULT '{}',
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    sync_hash VARCHAR(64), -- To detect changes
    UNIQUE (external_id, provider)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_external_calendars_user_id ON external_calendars(user_id);
CREATE INDEX IF NOT EXISTS idx_external_calendars_sync ON external_calendars(sync_enabled, last_sync_time);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_sync_log_calendar ON external_sync_log(calendar_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_external_mappings_local ON external_event_mappings(local_event_id);
CREATE INDEX IF NOT EXISTS idx_external_mappings_external ON external_event_mappings(external_id, provider);

-- Cleanup expired OAuth states
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_states() RETURNS void AS $$
BEGIN
    DELETE FROM oauth_states WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Add unique constraint on events metadata for external IDs
-- This prevents duplicate imports
ALTER TABLE events ADD CONSTRAINT unique_external_id 
    UNIQUE ((metadata->>'external_id')) 
    WHERE metadata->>'external_id' IS NOT NULL;