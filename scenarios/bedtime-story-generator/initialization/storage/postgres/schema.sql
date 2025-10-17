-- Bedtime Story Generator Database Schema
-- Stores generated stories and tracks reading sessions

-- Create enum for age groups
CREATE TYPE age_group AS ENUM ('3-5', '6-8', '9-12');

-- Create enum for story length
CREATE TYPE story_length AS ENUM ('short', 'medium', 'long');

-- Stories table
CREATE TABLE IF NOT EXISTS stories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    age_group age_group NOT NULL,
    theme VARCHAR(100),
    story_length story_length DEFAULT 'medium',
    reading_time_minutes INTEGER DEFAULT 10,
    character_names JSONB DEFAULT '[]'::jsonb,
    page_count INTEGER DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    times_read INTEGER DEFAULT 0,
    last_read TIMESTAMP WITH TIME ZONE,
    is_favorite BOOLEAN DEFAULT FALSE,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Reading sessions table
CREATE TABLE IF NOT EXISTS reading_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    story_id UUID NOT NULL REFERENCES stories(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    pages_read INTEGER DEFAULT 1,
    total_pages INTEGER DEFAULT 1,
    reading_progress_percent INTEGER DEFAULT 0,
    session_duration_seconds INTEGER,
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Story themes table (for categorization)
CREATE TABLE IF NOT EXISTS story_themes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    emoji VARCHAR(10),
    color VARCHAR(7), -- hex color
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User preferences table (for future personalization)
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    preferred_age_group age_group,
    preferred_themes TEXT[],
    preferred_length story_length DEFAULT 'medium',
    auto_nightlight_time TIME DEFAULT '21:00:00',
    reading_streak_days INTEGER DEFAULT 0,
    last_reading_date DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_stories_age_group ON stories(age_group);
CREATE INDEX idx_stories_theme ON stories(theme);
CREATE INDEX idx_stories_is_favorite ON stories(is_favorite);
CREATE INDEX idx_stories_created_at ON stories(created_at DESC);
CREATE INDEX idx_stories_times_read ON stories(times_read DESC);
CREATE INDEX idx_reading_sessions_story_id ON reading_sessions(story_id);
CREATE INDEX idx_reading_sessions_started_at ON reading_sessions(started_at DESC);

-- Function to update story read count and last read time
CREATE OR REPLACE FUNCTION update_story_read_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE stories 
    SET 
        times_read = times_read + 1,
        last_read = NEW.started_at,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.story_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update story stats when a reading session starts
CREATE TRIGGER trigger_update_story_stats
AFTER INSERT ON reading_sessions
FOR EACH ROW
EXECUTE FUNCTION update_story_read_stats();

-- Function to calculate session duration
CREATE OR REPLACE FUNCTION calculate_session_duration()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.completed_at IS NOT NULL AND NEW.started_at IS NOT NULL THEN
        NEW.session_duration_seconds = EXTRACT(EPOCH FROM (NEW.completed_at - NEW.started_at));
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to calculate session duration
CREATE TRIGGER trigger_calculate_duration
BEFORE UPDATE ON reading_sessions
FOR EACH ROW
WHEN (NEW.completed_at IS NOT NULL)
EXECUTE FUNCTION calculate_session_duration();

-- View for popular stories
CREATE OR REPLACE VIEW popular_stories AS
SELECT 
    s.*,
    COUNT(rs.id) as total_sessions,
    AVG(rs.reading_progress_percent) as avg_completion_rate
FROM stories s
LEFT JOIN reading_sessions rs ON s.id = rs.story_id
GROUP BY s.id
ORDER BY s.times_read DESC, s.is_favorite DESC;

-- View for recent reading activity
CREATE OR REPLACE VIEW recent_reading_activity AS
SELECT 
    rs.*,
    s.title as story_title,
    s.theme as story_theme,
    s.age_group as story_age_group
FROM reading_sessions rs
JOIN stories s ON rs.story_id = s.id
ORDER BY rs.started_at DESC
LIMIT 100;