-- Video Tools Database Schema
-- Core database structure for video processing and management

-- VideoAsset table: Main video file management
CREATE TABLE IF NOT EXISTS video_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    format VARCHAR(50) NOT NULL,
    duration_seconds DECIMAL(10,2),
    resolution_width INTEGER,
    resolution_height INTEGER,
    frame_rate DECIMAL(5,2),
    file_size_bytes BIGINT,
    minio_path VARCHAR(500),
    thumbnail_path VARCHAR(500),
    codec VARCHAR(100),
    bitrate_kbps INTEGER,
    has_audio BOOLEAN DEFAULT true,
    audio_channels INTEGER,
    metadata JSONB DEFAULT '{}',
    tags TEXT[],
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'pending',
    processing_error TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Indexes
    CONSTRAINT unique_minio_path UNIQUE(minio_path)
);

-- ProcessingJob table: Video processing tasks
CREATE TABLE IF NOT EXISTS processing_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID REFERENCES video_assets(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL CHECK (job_type IN ('edit', 'convert', 'analyze', 'enhance', 'stream', 'extract', 'compress')),
    parameters JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    progress_percentage INTEGER DEFAULT 0 CHECK (progress_percentage >= 0 AND progress_percentage <= 100),
    output_path VARCHAR(500),
    error_message TEXT,
    processing_time_ms BIGINT,
    created_by VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    
    -- Priority and resource management
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    resource_usage JSONB DEFAULT '{}'
);

-- VideoAnalytics table: AI-powered analysis results
CREATE TABLE IF NOT EXISTS video_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID REFERENCES video_assets(id) ON DELETE CASCADE,
    analysis_type VARCHAR(50) NOT NULL CHECK (analysis_type IN ('scene', 'object', 'speech', 'emotion', 'activity', 'quality')),
    confidence_score DECIMAL(5,4) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    start_time_seconds DECIMAL(10,2),
    end_time_seconds DECIMAL(10,2),
    bounding_box JSONB,
    detected_objects JSONB,
    transcript_text TEXT,
    speaker_id VARCHAR(100),
    emotion_data JSONB,
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- StreamingSession table: Live streaming management
CREATE TABLE IF NOT EXISTS streaming_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    input_source VARCHAR(50) CHECK (input_source IN ('file', 'camera', 'rtmp')),
    output_targets JSONB NOT NULL DEFAULT '[]',
    resolution VARCHAR(20),
    bitrate_kbps INTEGER,
    is_active BOOLEAN DEFAULT false,
    viewer_count INTEGER DEFAULT 0,
    total_bytes_streamed BIGINT DEFAULT 0,
    configuration JSONB DEFAULT '{}',
    stream_key VARCHAR(255) UNIQUE,
    
    -- Related video if file-based
    video_id UUID REFERENCES video_assets(id) ON DELETE SET NULL,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE
);

-- VideoEdit table: Edit operations history
CREATE TABLE IF NOT EXISTS video_edits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_video_id UUID REFERENCES video_assets(id) ON DELETE CASCADE,
    output_video_id UUID REFERENCES video_assets(id) ON DELETE SET NULL,
    operation_type VARCHAR(50) NOT NULL,
    operation_params JSONB NOT NULL,
    
    -- Tracking
    applied_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    applied_by VARCHAR(255)
);

-- Subtitle table: Subtitle and caption management
CREATE TABLE IF NOT EXISTS subtitles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID REFERENCES video_assets(id) ON DELETE CASCADE,
    language_code VARCHAR(10) NOT NULL,
    format VARCHAR(20) CHECK (format IN ('srt', 'vtt', 'ass', 'ssa')),
    content TEXT NOT NULL,
    is_auto_generated BOOLEAN DEFAULT false,
    burn_in BOOLEAN DEFAULT false,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- AudioTrack table: Audio track management
CREATE TABLE IF NOT EXISTS audio_tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID REFERENCES video_assets(id) ON DELETE CASCADE,
    track_number INTEGER NOT NULL,
    language_code VARCHAR(10),
    codec VARCHAR(50),
    bitrate_kbps INTEGER,
    channels INTEGER,
    sample_rate_hz INTEGER,
    duration_seconds DECIMAL(10,2),
    minio_path VARCHAR(500),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint
    CONSTRAINT unique_video_track UNIQUE(video_id, track_number)
);

-- Frame table: Extracted frames and thumbnails
CREATE TABLE IF NOT EXISTS frames (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    video_id UUID REFERENCES video_assets(id) ON DELETE CASCADE,
    timestamp_seconds DECIMAL(10,3) NOT NULL,
    frame_number INTEGER,
    minio_path VARCHAR(500) NOT NULL,
    width INTEGER,
    height INTEGER,
    format VARCHAR(20) DEFAULT 'jpg',
    is_thumbnail BOOLEAN DEFAULT false,
    
    -- Analysis data
    scene_change_score DECIMAL(5,4),
    blur_score DECIMAL(5,4),
    
    -- Timestamps
    extracted_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Unique constraint
    CONSTRAINT unique_video_timestamp UNIQUE(video_id, timestamp_seconds)
);

-- Create indexes for performance
CREATE INDEX idx_video_assets_status ON video_assets(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_video_assets_format ON video_assets(format) WHERE deleted_at IS NULL;
CREATE INDEX idx_video_assets_created ON video_assets(created_at DESC);
CREATE INDEX idx_video_assets_tags ON video_assets USING GIN(tags);

CREATE INDEX idx_processing_jobs_video ON processing_jobs(video_id);
CREATE INDEX idx_processing_jobs_status ON processing_jobs(status);
CREATE INDEX idx_processing_jobs_type ON processing_jobs(job_type);
CREATE INDEX idx_processing_jobs_created ON processing_jobs(created_at DESC);
CREATE INDEX idx_processing_jobs_priority ON processing_jobs(priority DESC) WHERE status = 'pending';

CREATE INDEX idx_video_analytics_video ON video_analytics(video_id);
CREATE INDEX idx_video_analytics_type ON video_analytics(analysis_type);
CREATE INDEX idx_video_analytics_confidence ON video_analytics(confidence_score DESC);

CREATE INDEX idx_streaming_sessions_active ON streaming_sessions(is_active);
CREATE INDEX idx_streaming_sessions_key ON streaming_sessions(stream_key) WHERE is_active = true;

CREATE INDEX idx_subtitles_video ON subtitles(video_id);
CREATE INDEX idx_subtitles_language ON subtitles(language_code);

CREATE INDEX idx_audio_tracks_video ON audio_tracks(video_id);

CREATE INDEX idx_frames_video ON frames(video_id);
CREATE INDEX idx_frames_timestamp ON frames(video_id, timestamp_seconds);
CREATE INDEX idx_frames_thumbnail ON frames(video_id) WHERE is_thumbnail = true;

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_video_assets_updated_at BEFORE UPDATE ON video_assets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_subtitles_updated_at BEFORE UPDATE ON subtitles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update video processing status
CREATE OR REPLACE FUNCTION update_video_status()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'completed' AND OLD.status != 'completed' THEN
        UPDATE video_assets SET
            status = 'ready'
        WHERE id = NEW.video_id;
    ELSIF NEW.status = 'failed' AND OLD.status != 'failed' THEN
        UPDATE video_assets SET
            status = 'error',
            processing_error = NEW.error_message
        WHERE id = NEW.video_id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_video_on_job_completion
    AFTER UPDATE OF status ON processing_jobs
    FOR EACH ROW
    WHEN (NEW.status IN ('completed', 'failed'))
    EXECUTE FUNCTION update_video_status();

-- Useful views for monitoring
CREATE OR REPLACE VIEW video_processing_queue AS
SELECT 
    pj.id,
    pj.job_type,
    pj.status,
    pj.progress_percentage,
    pj.priority,
    va.name as video_name,
    va.format,
    va.duration_seconds,
    pj.created_at,
    pj.started_at,
    EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - pj.started_at)) as processing_seconds
FROM processing_jobs pj
JOIN video_assets va ON pj.video_id = va.id
WHERE pj.status IN ('pending', 'processing')
ORDER BY pj.priority DESC, pj.created_at ASC;

CREATE OR REPLACE VIEW video_statistics AS
SELECT 
    COUNT(DISTINCT va.id) as total_videos,
    SUM(va.file_size_bytes) as total_storage_bytes,
    SUM(va.duration_seconds) as total_duration_seconds,
    COUNT(DISTINCT pj.id) as total_jobs,
    COUNT(DISTINCT pj.id) FILTER (WHERE pj.status = 'completed') as completed_jobs,
    COUNT(DISTINCT pj.id) FILTER (WHERE pj.status = 'failed') as failed_jobs,
    COUNT(DISTINCT ss.id) FILTER (WHERE ss.is_active = true) as active_streams
FROM video_assets va
LEFT JOIN processing_jobs pj ON va.id = pj.video_id
LEFT JOIN streaming_sessions ss ON va.id = ss.video_id;

CREATE OR REPLACE VIEW recent_analytics AS
SELECT 
    va.name as video_name,
    anal.analysis_type,
    anal.confidence_score,
    anal.start_time_seconds,
    anal.end_time_seconds,
    anal.transcript_text,
    anal.generated_at
FROM video_analytics anal
JOIN video_assets va ON anal.video_id = va.id
ORDER BY anal.generated_at DESC
LIMIT 100;