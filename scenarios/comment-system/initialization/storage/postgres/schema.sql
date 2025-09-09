-- Comment System Database Schema
-- Version: 1.0.0
-- Description: Universal comment system for Vrooli scenarios

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create custom types
CREATE TYPE moderation_level AS ENUM ('none', 'manual', 'ai_assisted');
CREATE TYPE content_type AS ENUM ('plaintext', 'markdown');
CREATE TYPE comment_status AS ENUM ('active', 'deleted', 'moderated');

-- Scenario configurations table
-- Stores per-scenario comment system settings
CREATE TABLE scenario_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) UNIQUE NOT NULL,
    auth_required BOOLEAN DEFAULT true NOT NULL,
    allow_anonymous BOOLEAN DEFAULT false NOT NULL,
    allow_rich_media BOOLEAN DEFAULT false NOT NULL,
    moderation_level moderation_level DEFAULT 'manual' NOT NULL,
    theme_config JSONB DEFAULT '{}' NOT NULL,
    notification_settings JSONB DEFAULT '{"mentions": true, "replies": true, "new_comments": false}' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Constraints
    CONSTRAINT scenario_name_valid CHECK (
        scenario_name ~ '^[a-z0-9][a-z0-9-]*[a-z0-9]$' AND 
        LENGTH(scenario_name) BETWEEN 2 AND 50
    ),
    CONSTRAINT consistent_auth_settings CHECK (
        NOT (auth_required = true AND allow_anonymous = true)
    )
);

-- Comments table
-- Core comment data with threading support
CREATE TABLE comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) NOT NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE,
    author_id UUID, -- From session-authenticator, nullable for anonymous
    author_name VARCHAR(255), -- Cached display name
    content TEXT NOT NULL,
    content_type content_type DEFAULT 'markdown' NOT NULL,
    metadata JSONB DEFAULT '{}' NOT NULL, -- For attachments, mentions, etc.
    status comment_status DEFAULT 'active' NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    version INTEGER DEFAULT 1 NOT NULL,
    
    -- Computed columns for efficient queries
    thread_path TEXT, -- Materialized path for efficient tree queries
    depth INTEGER DEFAULT 0 NOT NULL,
    reply_count INTEGER DEFAULT 0 NOT NULL,
    
    -- Constraints
    CONSTRAINT content_not_empty CHECK (LENGTH(TRIM(content)) > 0),
    CONSTRAINT version_positive CHECK (version > 0),
    CONSTRAINT depth_reasonable CHECK (depth >= 0 AND depth <= 10),
    CONSTRAINT reply_count_non_negative CHECK (reply_count >= 0),
    CONSTRAINT no_self_parent CHECK (id != parent_id),
    
    -- Foreign key to scenario configs
    FOREIGN KEY (scenario_name) REFERENCES scenario_configs(scenario_name) 
        ON DELETE CASCADE ON UPDATE CASCADE
);

-- Comment history table
-- Tracks all edits to comments
CREATE TABLE comment_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    content_type content_type NOT NULL,
    metadata JSONB DEFAULT '{}' NOT NULL,
    edited_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    edited_by UUID, -- User who made the edit
    version INTEGER NOT NULL,
    
    -- Constraints
    CONSTRAINT content_not_empty CHECK (LENGTH(TRIM(content)) > 0),
    CONSTRAINT version_positive CHECK (version > 0),
    
    -- Unique constraint on comment_id + version
    UNIQUE(comment_id, version)
);

-- Comment mentions table
-- Tracks @mentions in comments for notifications
CREATE TABLE comment_mentions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    mentioned_user_id UUID NOT NULL,
    mentioned_username VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- Prevent duplicate mentions in same comment
    UNIQUE(comment_id, mentioned_user_id)
);

-- Comment reactions table (future enhancement)
-- Placeholder for likes/votes system
CREATE TABLE comment_reactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    comment_id UUID NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    reaction_type VARCHAR(50) NOT NULL DEFAULT 'like', -- like, dislike, heart, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP NOT NULL,
    
    -- One reaction per user per comment
    UNIQUE(comment_id, user_id, reaction_type)
);

-- Indexes for performance

-- Scenario configs
CREATE INDEX idx_scenario_configs_name ON scenario_configs(scenario_name);

-- Comments - core queries
CREATE INDEX idx_comments_scenario ON comments(scenario_name);
CREATE INDEX idx_comments_parent ON comments(parent_id);
CREATE INDEX idx_comments_author ON comments(author_id);
CREATE INDEX idx_comments_status ON comments(status);
CREATE INDEX idx_comments_created_at ON comments(created_at DESC);

-- Comments - threading and tree queries
CREATE INDEX idx_comments_thread_path ON comments(thread_path);
CREATE INDEX idx_comments_scenario_thread ON comments(scenario_name, thread_path);
CREATE INDEX idx_comments_scenario_created ON comments(scenario_name, created_at DESC);
CREATE INDEX idx_comments_scenario_status ON comments(scenario_name, status);

-- Comments - compound indexes for common queries
CREATE INDEX idx_comments_scenario_parent_created ON comments(scenario_name, parent_id, created_at DESC);
CREATE INDEX idx_comments_scenario_status_created ON comments(scenario_name, status, created_at DESC);

-- Comment history
CREATE INDEX idx_comment_history_comment_id ON comment_history(comment_id);
CREATE INDEX idx_comment_history_comment_version ON comment_history(comment_id, version);

-- Comment mentions
CREATE INDEX idx_comment_mentions_comment ON comment_mentions(comment_id);
CREATE INDEX idx_comment_mentions_user ON comment_mentions(mentioned_user_id);

-- Comment reactions
CREATE INDEX idx_comment_reactions_comment ON comment_reactions(comment_id);
CREATE INDEX idx_comment_reactions_user ON comment_reactions(user_id);

-- Functions and triggers

-- Function to update thread_path and depth
CREATE OR REPLACE FUNCTION update_comment_threading()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.parent_id IS NULL THEN
        -- Root comment
        NEW.thread_path := NEW.id::TEXT;
        NEW.depth := 0;
    ELSE
        -- Reply comment - build path and depth from parent
        SELECT 
            COALESCE(thread_path, id::TEXT) || '.' || NEW.id::TEXT,
            COALESCE(depth, 0) + 1
        INTO NEW.thread_path, NEW.depth
        FROM comments 
        WHERE id = NEW.parent_id;
        
        -- Increment parent's reply count
        UPDATE comments 
        SET reply_count = reply_count + 1 
        WHERE id = NEW.parent_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to maintain reply counts
CREATE OR REPLACE FUNCTION maintain_reply_counts()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        -- Decrement reply count when comment is deleted
        IF OLD.parent_id IS NOT NULL THEN
            UPDATE comments 
            SET reply_count = reply_count - 1 
            WHERE id = OLD.parent_id;
        END IF;
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to save comment history on updates
CREATE OR REPLACE FUNCTION save_comment_history()
RETURNS TRIGGER AS $$
BEGIN
    -- Only save history if content actually changed
    IF NEW.content != OLD.content OR NEW.metadata != OLD.metadata THEN
        -- Increment version
        NEW.version = OLD.version + 1;
        
        -- Save the old version to history
        INSERT INTO comment_history (
            comment_id, 
            content, 
            content_type,
            metadata,
            edited_at, 
            edited_by, 
            version
        ) VALUES (
            OLD.id, 
            OLD.content, 
            OLD.content_type,
            OLD.metadata,
            OLD.updated_at, 
            NEW.author_id, -- Assuming same user is editing
            OLD.version
        );
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to extract mentions from comment content
CREATE OR REPLACE FUNCTION extract_mentions()
RETURNS TRIGGER AS $$
DECLARE
    mention_pattern TEXT := '@([a-zA-Z0-9_-]+)';
    username TEXT;
    user_record RECORD;
BEGIN
    -- Clear existing mentions for this comment
    DELETE FROM comment_mentions WHERE comment_id = NEW.id;
    
    -- Extract @mentions from content (simplified - in practice would integrate with session-authenticator)
    -- This is a placeholder that would need to be enhanced with actual user lookup
    FOR username IN 
        SELECT DISTINCT regexp_replace(match[1], '^@', '') 
        FROM regexp_split_to_table(NEW.content, '\s+') AS match_table(match)
        WHERE match[1] ~ '^@[a-zA-Z0-9_-]+$'
    LOOP
        -- Insert mention record (would need user_id lookup in real implementation)
        INSERT INTO comment_mentions (comment_id, mentioned_user_id, mentioned_username)
        VALUES (NEW.id, uuid_generate_v4(), username) -- Placeholder UUID
        ON CONFLICT (comment_id, mentioned_user_id) DO NOTHING;
    END LOOP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers

-- Threading triggers
CREATE TRIGGER trigger_update_comment_threading
    BEFORE INSERT ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_comment_threading();

-- Reply count maintenance
CREATE TRIGGER trigger_maintain_reply_counts
    AFTER DELETE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION maintain_reply_counts();

-- Timestamp updates
CREATE TRIGGER trigger_comments_updated_at
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER trigger_scenario_configs_updated_at
    BEFORE UPDATE ON scenario_configs
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- History tracking
CREATE TRIGGER trigger_save_comment_history
    BEFORE UPDATE ON comments
    FOR EACH ROW
    EXECUTE FUNCTION save_comment_history();

-- Mention extraction
CREATE TRIGGER trigger_extract_mentions
    AFTER INSERT OR UPDATE ON comments
    FOR EACH ROW
    WHEN (NEW.content IS DISTINCT FROM OLD.content OR TG_OP = 'INSERT')
    EXECUTE FUNCTION extract_mentions();

-- Views for common queries

-- View for comment threads with metadata
CREATE VIEW comment_threads AS
SELECT 
    c.*,
    sc.auth_required,
    sc.allow_anonymous,
    sc.moderation_level,
    CASE 
        WHEN c.parent_id IS NULL THEN 0
        ELSE (
            SELECT COUNT(*) 
            FROM comments replies 
            WHERE replies.thread_path LIKE c.thread_path || '.%'
            AND replies.status = 'active'
        )
    END as total_replies
FROM comments c
JOIN scenario_configs sc ON c.scenario_name = sc.scenario_name
WHERE c.status = 'active';

-- View for comment statistics per scenario
CREATE VIEW scenario_comment_stats AS
SELECT 
    sc.scenario_name,
    sc.auth_required,
    sc.allow_anonymous,
    sc.moderation_level,
    COUNT(c.id) as total_comments,
    COUNT(CASE WHEN c.parent_id IS NULL THEN 1 END) as root_comments,
    COUNT(CASE WHEN c.parent_id IS NOT NULL THEN 1 END) as reply_comments,
    COUNT(CASE WHEN c.created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours' THEN 1 END) as comments_24h,
    COUNT(CASE WHEN c.created_at > CURRENT_TIMESTAMP - INTERVAL '7 days' THEN 1 END) as comments_7d,
    MAX(c.created_at) as last_comment_at
FROM scenario_configs sc
LEFT JOIN comments c ON sc.scenario_name = c.scenario_name AND c.status = 'active'
GROUP BY sc.scenario_name, sc.auth_required, sc.allow_anonymous, sc.moderation_level;

-- Insert default configuration for test scenario
INSERT INTO scenario_configs (
    scenario_name, 
    auth_required, 
    allow_anonymous, 
    allow_rich_media,
    moderation_level,
    theme_config,
    notification_settings
) VALUES (
    'test-scenario',
    false, -- Allow testing without auth
    true,  -- Allow anonymous for testing
    true,  -- Enable rich media for testing
    'none', -- No moderation for testing
    '{"theme": "default", "show_avatars": true}',
    '{"mentions": true, "replies": true, "new_comments": false}'
) ON CONFLICT (scenario_name) DO NOTHING;

-- Performance optimization: Analyze tables for query planner
ANALYZE scenario_configs;
ANALYZE comments;
ANALYZE comment_history;
ANALYZE comment_mentions;
ANALYZE comment_reactions;