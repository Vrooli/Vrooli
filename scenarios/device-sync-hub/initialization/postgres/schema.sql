-- Device Sync Hub Database Schema
-- Creates tables for sync items, device sessions, and related metadata

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Devices table - stores registered devices per user
CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    platform VARCHAR(100) NOT NULL,
    capabilities TEXT,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Files table - stores file metadata for downloads
CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    path TEXT NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Sync items table - stores metadata for all synchronized files and text
CREATE TABLE IF NOT EXISTS sync_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('file', 'text', 'clipboard', 'notification')),
    content JSONB DEFAULT '{}',
    source_device VARCHAR(255),
    target_devices JSONB DEFAULT '[]',
    status VARCHAR(20) DEFAULT 'pending',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Device sessions table - tracks active WebSocket connections per user
CREATE TABLE IF NOT EXISTS device_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    device_info JSONB DEFAULT '{}',
    websocket_id VARCHAR(36) UNIQUE NOT NULL,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_devices_user_id ON devices(user_id);
CREATE INDEX IF NOT EXISTS idx_devices_last_seen ON devices(last_seen);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_expires_at ON files(expires_at);

CREATE INDEX IF NOT EXISTS idx_sync_items_user_id ON sync_items(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_items_expires_at ON sync_items(expires_at);
CREATE INDEX IF NOT EXISTS idx_sync_items_type ON sync_items(type);
CREATE INDEX IF NOT EXISTS idx_sync_items_status ON sync_items(status);
CREATE INDEX IF NOT EXISTS idx_sync_items_created_at ON sync_items(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_device_sessions_user_id ON device_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_device_sessions_websocket_id ON device_sessions(websocket_id);
CREATE INDEX IF NOT EXISTS idx_device_sessions_last_seen ON device_sessions(last_seen);

-- Trigger to update updated_at timestamp on sync_items
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_sync_items_updated_at 
    BEFORE UPDATE ON sync_items 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to clean up expired items (can be called by cleanup jobs)
CREATE OR REPLACE FUNCTION cleanup_expired_items()
RETURNS TABLE(deleted_count INTEGER, storage_paths TEXT[]) AS $$
DECLARE
    paths TEXT[];
    count INTEGER;
BEGIN
    -- Get storage paths before deletion for file cleanup
    SELECT array_agg(storage_path) INTO paths
    FROM sync_items 
    WHERE expires_at <= NOW();
    
    -- Delete expired items
    WITH deleted AS (
        DELETE FROM sync_items 
        WHERE expires_at <= NOW()
        RETURNING id
    )
    SELECT COUNT(*) INTO count FROM deleted;
    
    RETURN QUERY SELECT count, COALESCE(paths, ARRAY[]::TEXT[]);
END;
$$ LANGUAGE plpgsql;

-- Function to get user statistics
CREATE OR REPLACE FUNCTION get_user_sync_stats(user_id_param VARCHAR)
RETURNS TABLE(
    total_items INTEGER,
    total_size BIGINT,
    files_count INTEGER,
    text_count INTEGER,
    clipboard_count INTEGER,
    expires_soon_count INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER as total_items,
        COALESCE(SUM((content->>'file_size')::BIGINT), 0)::BIGINT as total_size,
        COUNT(*) FILTER (WHERE type = 'file')::INTEGER as files_count,
        COUNT(*) FILTER (WHERE type = 'text')::INTEGER as text_count,
        COUNT(*) FILTER (WHERE type = 'clipboard')::INTEGER as clipboard_count,
        COUNT(*) FILTER (WHERE expires_at <= NOW() + INTERVAL '1 hour')::INTEGER as expires_soon_count
    FROM sync_items 
    WHERE user_id = user_id_param AND expires_at > NOW();
END;
$$ LANGUAGE plpgsql;

-- Function to clean up old device sessions (older than 7 days)
CREATE OR REPLACE FUNCTION cleanup_old_device_sessions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    WITH deleted AS (
        DELETE FROM device_sessions 
        WHERE last_seen < NOW() - INTERVAL '7 days'
        RETURNING id
    )
    SELECT COUNT(*) INTO deleted_count FROM deleted;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- View for active sync items with additional computed fields
CREATE OR REPLACE VIEW active_sync_items AS
SELECT 
    s.*,
    CASE 
        WHEN s.expires_at <= NOW() + INTERVAL '1 hour' THEN 'expires_soon'
        WHEN s.expires_at <= NOW() + INTERVAL '6 hours' THEN 'expires_today'
        ELSE 'active'
    END as expiry_status,
    CASE 
        WHEN s.type = 'file' AND (s.content->>'mime_type') LIKE 'image/%' THEN 'image'
        WHEN s.type = 'file' AND (s.content->>'mime_type') LIKE 'video/%' THEN 'video'
        WHEN s.type = 'file' AND (s.content->>'mime_type') LIKE 'audio/%' THEN 'audio'
        WHEN s.type = 'file' AND (s.content->>'mime_type') LIKE 'text/%' THEN 'text_file'
        WHEN s.type = 'file' THEN 'other_file'
        ELSE s.type
    END as display_type
FROM sync_items s
WHERE s.expires_at > NOW();

-- View for device session summary
CREATE OR REPLACE VIEW device_session_summary AS
SELECT 
    user_id,
    COUNT(*) as active_sessions,
    MAX(last_seen) as most_recent_activity,
    array_agg(DISTINCT device_info->>'platform') FILTER (WHERE device_info->>'platform' IS NOT NULL) as platforms,
    array_agg(DISTINCT device_info->>'browser') FILTER (WHERE device_info->>'browser' IS NOT NULL) as browsers
FROM device_sessions
WHERE last_seen > NOW() - INTERVAL '1 day'
GROUP BY user_id;

-- Insert some helpful comments for documentation
COMMENT ON TABLE devices IS 'Stores registered devices for each user';
COMMENT ON TABLE files IS 'Stores metadata for uploaded files';
COMMENT ON TABLE sync_items IS 'Stores metadata for all synchronized content between devices';
COMMENT ON TABLE device_sessions IS 'Tracks active WebSocket connections for real-time synchronization';

COMMENT ON COLUMN sync_items.type IS 'Type of content: file, text, clipboard, or notification';
COMMENT ON COLUMN sync_items.content IS 'JSON content with type-specific data';
COMMENT ON COLUMN sync_items.source_device IS 'ID of the device that created this sync item';
COMMENT ON COLUMN sync_items.target_devices IS 'Array of target device IDs for this sync item';
COMMENT ON COLUMN sync_items.expires_at IS 'When this item should be automatically deleted';

COMMENT ON COLUMN device_sessions.device_info IS 'JSON containing device/browser information for identification';
COMMENT ON COLUMN device_sessions.websocket_id IS 'Unique identifier for the WebSocket connection';
COMMENT ON COLUMN device_sessions.last_seen IS 'Last activity timestamp for connection management';

-- Grant permissions (assuming standard Vrooli user setup)
-- Note: Adjust these based on your actual user setup
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO vrooli;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO vrooli;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO vrooli;