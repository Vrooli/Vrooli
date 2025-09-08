-- Video Downloader Database Schema - Enhanced Media Intelligence Platform
-- Manages download history, queue, transcripts, and metadata

-- Downloads table for tracking all downloads with enhanced audio and transcript support
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
    -- Enhanced audio format support
    audio_format VARCHAR(10), -- mp3, flac, aac, ogg, etc.
    audio_quality VARCHAR(10), -- 320k, 192k, 128k, etc.
    audio_path TEXT, -- Path to extracted audio file
    file_path TEXT,
    file_size BIGINT, -- in bytes
    audio_file_size BIGINT, -- Size of extracted audio file in bytes
    status VARCHAR(20) DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    -- Enhanced transcript support
    has_transcript BOOLEAN DEFAULT FALSE,
    transcript_status VARCHAR(20) DEFAULT NULL, -- pending, processing, completed, failed
    transcript_requested BOOLEAN DEFAULT FALSE,
    whisper_model VARCHAR(20), -- tiny, base, small, medium, large
    target_language VARCHAR(10), -- Language code for transcription
    error_message TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    transcript_started_at TIMESTAMP,
    transcript_completed_at TIMESTAMP,
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

-- Format preferences for auto-selection with enhanced audio support
CREATE TABLE IF NOT EXISTS format_preferences (
    id SERIAL PRIMARY KEY,
    platform VARCHAR(50),
    preferred_quality VARCHAR(20),
    preferred_format VARCHAR(20),
    -- Enhanced audio preferences
    preferred_audio_format VARCHAR(10), -- mp3, flac, aac, ogg
    preferred_audio_quality VARCHAR(10), -- 320k, 192k, 128k
    max_file_size BIGINT,
    audio_only BOOLEAN DEFAULT FALSE,
    -- Transcript preferences
    auto_transcript BOOLEAN DEFAULT FALSE,
    preferred_whisper_model VARCHAR(20) DEFAULT 'base',
    preferred_language VARCHAR(10),
    user_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(platform, user_id)
);

-- Transcripts table for storing complete transcript data
CREATE TABLE IF NOT EXISTS transcripts (
    id SERIAL PRIMARY KEY,
    download_id INTEGER REFERENCES downloads(id) ON DELETE CASCADE,
    language VARCHAR(10) NOT NULL, -- ISO language code (en, es, fr, etc.)
    detected_language VARCHAR(10), -- Actually detected language if different
    confidence_score FLOAT, -- Overall transcript confidence (0.0-1.0)
    model_used VARCHAR(20), -- Whisper model used (tiny, base, small, medium, large)
    full_text TEXT NOT NULL, -- Complete transcript text
    word_count INTEGER, -- Number of words in transcript
    processing_time_ms INTEGER, -- Time taken to generate transcript
    audio_duration_seconds FLOAT, -- Duration of processed audio
    whisper_version VARCHAR(20), -- Version of Whisper used
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(download_id) -- One transcript per download
);

-- Transcript segments for time-stamped transcript portions
CREATE TABLE IF NOT EXISTS transcript_segments (
    id SERIAL PRIMARY KEY,
    transcript_id INTEGER REFERENCES transcripts(id) ON DELETE CASCADE,
    start_time FLOAT NOT NULL, -- Start time in seconds
    end_time FLOAT NOT NULL, -- End time in seconds
    text TEXT NOT NULL, -- Segment text
    confidence FLOAT, -- Segment confidence score (0.0-1.0)
    speaker_id VARCHAR(20), -- Speaker identifier for future diarization
    word_timestamps JSONB, -- Individual word timing data
    sequence INTEGER NOT NULL, -- Order within transcript
    character_start INTEGER, -- Character offset in full transcript
    character_end INTEGER, -- Character end offset in full transcript
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CHECK (start_time >= 0),
    CHECK (end_time > start_time),
    CHECK (sequence > 0),
    UNIQUE(transcript_id, sequence)
);

-- Transcript exports for tracking exported files
CREATE TABLE IF NOT EXISTS transcript_exports (
    id SERIAL PRIMARY KEY,
    transcript_id INTEGER REFERENCES transcripts(id) ON DELETE CASCADE,
    format VARCHAR(10) NOT NULL, -- srt, vtt, txt, json
    file_path TEXT NOT NULL, -- Path to exported file
    file_size INTEGER, -- Size of exported file
    include_timestamps BOOLEAN DEFAULT TRUE,
    include_confidence BOOLEAN DEFAULT FALSE,
    export_settings JSONB, -- Format-specific settings
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP -- Optional expiration for temporary exports
);

-- Analytics for tracking usage patterns with enhanced transcript metrics
CREATE TABLE IF NOT EXISTS download_analytics (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL,
    platform VARCHAR(50),
    total_downloads INTEGER DEFAULT 0,
    total_size BIGINT DEFAULT 0,
    total_audio_size BIGINT DEFAULT 0, -- Total audio file sizes
    avg_duration INTEGER DEFAULT 0,
    most_common_quality VARCHAR(20),
    most_common_audio_format VARCHAR(10), -- Most used audio format
    -- Transcript analytics
    total_transcripts INTEGER DEFAULT 0,
    avg_transcript_confidence FLOAT,
    most_common_language VARCHAR(10),
    most_common_whisper_model VARCHAR(20),
    total_transcript_processing_time_ms BIGINT DEFAULT 0,
    user_id VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(date, platform, user_id)
);

-- Performance indexes for existing functionality
CREATE INDEX IF NOT EXISTS idx_downloads_status ON downloads(status);
CREATE INDEX IF NOT EXISTS idx_downloads_created_at ON downloads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_downloads_user_id ON downloads(user_id);
CREATE INDEX IF NOT EXISTS idx_download_queue_position ON download_queue(position);
CREATE INDEX IF NOT EXISTS idx_download_queue_priority ON download_queue(priority DESC);
CREATE INDEX IF NOT EXISTS idx_playlists_status ON playlists(status);
CREATE INDEX IF NOT EXISTS idx_format_preferences_user ON format_preferences(user_id);

-- Enhanced indexes for audio and transcript functionality
CREATE INDEX IF NOT EXISTS idx_downloads_has_transcript ON downloads(has_transcript);
CREATE INDEX IF NOT EXISTS idx_downloads_transcript_status ON downloads(transcript_status);
CREATE INDEX IF NOT EXISTS idx_downloads_audio_format ON downloads(audio_format);
CREATE INDEX IF NOT EXISTS idx_downloads_audio_quality ON downloads(audio_quality);
CREATE INDEX IF NOT EXISTS idx_downloads_whisper_model ON downloads(whisper_model);

-- Transcript table indexes
CREATE INDEX IF NOT EXISTS idx_transcripts_download_id ON transcripts(download_id);
CREATE INDEX IF NOT EXISTS idx_transcripts_language ON transcripts(language);
CREATE INDEX IF NOT EXISTS idx_transcripts_confidence ON transcripts(confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_transcripts_created_at ON transcripts(created_at DESC);

-- Transcript segments indexes for time-based queries
CREATE INDEX IF NOT EXISTS idx_transcript_segments_transcript_id ON transcript_segments(transcript_id);
CREATE INDEX IF NOT EXISTS idx_transcript_segments_time_range ON transcript_segments(start_time, end_time);
CREATE INDEX IF NOT EXISTS idx_transcript_segments_sequence ON transcript_segments(transcript_id, sequence);

-- Full-text search indexes for transcript content
CREATE INDEX IF NOT EXISTS idx_transcripts_full_text_search ON transcripts USING gin(to_tsvector('english', full_text));
CREATE INDEX IF NOT EXISTS idx_transcript_segments_text_search ON transcript_segments USING gin(to_tsvector('english', text));

-- Transcript exports indexes
CREATE INDEX IF NOT EXISTS idx_transcript_exports_transcript_format ON transcript_exports(transcript_id, format);
CREATE INDEX IF NOT EXISTS idx_transcript_exports_created_at ON transcript_exports(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_transcript_exports_expires_at ON transcript_exports(expires_at) WHERE expires_at IS NOT NULL;