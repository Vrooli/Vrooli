-- Device Sync Hub Database Schema
-- Creates tables for sync items, device sessions, and related metadata

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Sync items table - stores metadata for all synchronized files and text
CREATE TABLE IF NOT EXISTS sync_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    filename VARCHAR(255) NOT NULL,
    mime_type VARCHAR(100) NOT NULL,
    file_size BIGINT NOT NULL,
    content_type VARCHAR(20) NOT NULL CHECK (content_type IN ('file', 'text', 'clipboard')),
    storage_path TEXT NOT NULL,
    thumbnail_path TEXT,
    metadata JSONB DEFAULT '{}',
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
CREATE INDEX IF NOT EXISTS idx_sync_items_user_id ON sync_items(user_id);
CREATE INDEX IF NOT EXISTS idx_sync_items_expires_at ON sync_items(expires_at);
CREATE INDEX IF NOT EXISTS idx_sync_items_content_type ON sync_items(content_type);
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
CREATE OR REPLACE FUNCTION get_user_sync_stats(user_uuid UUID)
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
        COALESCE(SUM(file_size), 0)::BIGINT as total_size,
        COUNT(*) FILTER (WHERE content_type = 'file')::INTEGER as files_count,
        COUNT(*) FILTER (WHERE content_type = 'text')::INTEGER as text_count,
        COUNT(*) FILTER (WHERE content_type = 'clipboard')::INTEGER as clipboard_count,
        COUNT(*) FILTER (WHERE expires_at <= NOW() + INTERVAL '1 hour')::INTEGER as expires_soon_count
    FROM sync_items 
    WHERE user_id = user_uuid AND expires_at > NOW();
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
        WHEN s.content_type = 'file' AND s.mime_type LIKE 'image/%' THEN 'image'
        WHEN s.content_type = 'file' AND s.mime_type LIKE 'video/%' THEN 'video'
        WHEN s.content_type = 'file' AND s.mime_type LIKE 'audio/%' THEN 'audio'
        WHEN s.content_type = 'file' AND s.mime_type LIKE 'text/%' THEN 'text_file'
        WHEN s.content_type = 'file' THEN 'other_file'
        ELSE s.content_type
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
COMMENT ON TABLE sync_items IS 'Stores metadata for all synchronized files and text content';
COMMENT ON TABLE device_sessions IS 'Tracks active WebSocket connections for real-time synchronization';

COMMENT ON COLUMN sync_items.content_type IS 'Type of content: file, text, or clipboard';
COMMENT ON COLUMN sync_items.storage_path IS 'Full filesystem path to the stored file';
COMMENT ON COLUMN sync_items.thumbnail_path IS 'Path to generated thumbnail (for images only)';
COMMENT ON COLUMN sync_items.metadata IS 'Additional metadata stored as JSON (original filename, upload info, etc.)';
COMMENT ON COLUMN sync_items.expires_at IS 'When this item should be automatically deleted';

COMMENT ON COLUMN device_sessions.device_info IS 'JSON containing device/browser information for identification';
COMMENT ON COLUMN device_sessions.websocket_id IS 'Unique identifier for the WebSocket connection';
COMMENT ON COLUMN device_sessions.last_seen IS 'Last activity timestamp for connection management';

-- Grant permissions (assuming standard Vrooli user setup)
-- Note: Adjust these based on your actual user setup
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO vrooli;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO vrooli;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO vrooli;