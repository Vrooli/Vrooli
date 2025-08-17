-- Video Downloader Database Schema
-- Manages download history, queue, and metadata

-- Downloads table for tracking all downloads
CREATE TABLE IF NOT EXISTS downloads (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    title TEXT,
    description TEXT,
    thumbnail_url TEXT,
    platform VARCHAR(50),
    duration INTEGER, -- in seconds
    format VARCHAR(20),
    quality VARCHAR(20),
    file_path TEXT,
    file_size BIGINT, -- in bytes
    status VARCHAR(20) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    user_id VARCHAR(100),
    session_id VARCHAR(100)
);

-- Download queue for managing batch downloads
CREATE TABLE IF NOT EXISTS download_queue (
    id SERIAL PRIMARY KEY,
    download_id INTEGER REFERENCES downloads(id),
    position INTEGER NOT NULL,
    priority INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    scheduled_at TIMESTAMP,
    UNIQUE(download_id)
);

-- Playlists table for managing playlist downloads
CREATE TABLE IF NOT EXISTS playlists (
    id SERIAL PRIMARY KEY,
    url TEXT NOT NULL,
    title TEXT,
    channel_name TEXT,
    total_videos INTEGER,
    downloaded_count INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Playlist items for tracking individual videos in playlists
CREATE TABLE IF NOT EXISTS playlist_items (
    id SERIAL PRIMARY KEY,
    playlist_id INTEGER REFERENCES playlists(id) ON DELETE CASCADE,
    download_id INTEGER REFERENCES downloads(id),
    position INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Format preferences for auto-selection
CREATE TABLE IF NOT EXISTS format_preferences (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50),
    preferred_quality VARCHAR(20),
    preferred_format VARCHAR(20),
    max_file_size BIGINT,
    audio_only BOOLEAN DEFAULT FALSE,
    user_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, user_id)
);

-- Analytics for tracking usage patterns
CREATE TABLE IF NOT EXISTS download_analytics (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    platform VARCHAR(50),
    total_downloads INTEGER DEFAULT 0,
    total_size BIGINT DEFAULT 0,
    avg_duration INTEGER DEFAULT 0,
    most_common_quality VARCHAR(20),
    user_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, platform, user_id)
);

-- Indexes for performance
CREATE INDEX idx_downloads_status ON downloads(status);
CREATE INDEX idx_downloads_created_at ON downloads(created_at DESC);
CREATE INDEX idx_downloads_user_id ON downloads(user_id);
CREATE INDEX idx_download_queue_position ON download_queue(position);
CREATE INDEX idx_download_queue_priority ON download_queue(priority DESC);
CREATE INDEX idx_playlists_status ON playlists(status);
CREATE INDEX idx_format_preferences_user ON format_preferences(user_id);