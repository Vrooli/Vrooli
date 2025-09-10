-- Visited Tracker Database Schema

-- Campaigns table: isolated tracking contexts for different agents/tasks
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    from_agent VARCHAR(255) NOT NULL DEFAULT 'manual', -- agent name or 'manual' for human-created
    description TEXT,
    patterns JSONB NOT NULL DEFAULT '[]', -- File patterns this campaign tracks
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    status VARCHAR(50) NOT NULL DEFAULT 'active', -- active/completed/archived
    metadata JSONB NOT NULL DEFAULT '{}',
    UNIQUE(name)
);

-- Tracked files table: stores file metadata and visit statistics per campaign
CREATE TABLE IF NOT EXISTS tracked_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    file_path VARCHAR(1000) NOT NULL, -- Relative path
    absolute_path VARCHAR(1000) NOT NULL, -- Full path
    visit_count INTEGER NOT NULL DEFAULT 0,
    first_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_visited TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    content_hash VARCHAR(64), -- SHA256 for change detection
    size_bytes BIGINT NOT NULL DEFAULT 0,
    staleness_score DECIMAL(10, 4) NOT NULL DEFAULT 0, -- Calculated: (mods × time) / (visits + 1)
    deleted BOOLEAN NOT NULL DEFAULT FALSE, -- Soft delete for removed files
    metadata JSONB NOT NULL DEFAULT '{}',
    UNIQUE(campaign_id, absolute_path)
);

-- Visits table: detailed visit history for each file
CREATE TABLE IF NOT EXISTS visits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    file_id UUID NOT NULL REFERENCES tracked_files(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    context VARCHAR(50), -- security/performance/bug/docs/general
    agent VARCHAR(255), -- which AI model/agent visited
    conversation_id VARCHAR(255), -- link visits within conversations
    duration_ms INTEGER, -- how long the visit took
    findings JSONB -- what was found/changed
);

-- Structure snapshots table: track file system changes over time
CREATE TABLE IF NOT EXISTS structure_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    total_files INTEGER NOT NULL,
    new_files JSONB NOT NULL DEFAULT '[]', -- Array of newly added file paths
    deleted_files JSONB NOT NULL DEFAULT '[]', -- Array of removed file paths
    moved_files JSONB NOT NULL DEFAULT '{}', -- Map of old_path -> new_path
    snapshot_data JSONB NOT NULL -- Full file structure for reference
);

-- Indexes for performance
-- Campaign indexes
CREATE INDEX IF NOT EXISTS idx_campaigns_name ON campaigns(name);
CREATE INDEX IF NOT EXISTS idx_campaigns_from_agent ON campaigns(from_agent);
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC);

-- File tracking indexes
CREATE INDEX IF NOT EXISTS idx_tracked_files_campaign_id ON tracked_files(campaign_id);
CREATE INDEX IF NOT EXISTS idx_tracked_files_path ON tracked_files(file_path);
CREATE INDEX IF NOT EXISTS idx_tracked_files_visit_count ON tracked_files(visit_count);
CREATE INDEX IF NOT EXISTS idx_tracked_files_staleness ON tracked_files(staleness_score DESC);
CREATE INDEX IF NOT EXISTS idx_tracked_files_last_visited ON tracked_files(last_visited);
CREATE INDEX IF NOT EXISTS idx_tracked_files_deleted ON tracked_files(deleted);
CREATE INDEX IF NOT EXISTS idx_tracked_files_path_pattern ON tracked_files(file_path varchar_pattern_ops);

-- Visit history indexes
CREATE INDEX IF NOT EXISTS idx_visits_file_id ON visits(file_id);
CREATE INDEX IF NOT EXISTS idx_visits_timestamp ON visits(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_visits_context ON visits(context);
CREATE INDEX IF NOT EXISTS idx_visits_agent ON visits(agent);
CREATE INDEX IF NOT EXISTS idx_visits_conversation ON visits(conversation_id);

-- Structure snapshot indexes
CREATE INDEX IF NOT EXISTS idx_structure_snapshots_timestamp ON structure_snapshots(timestamp DESC);

-- Function to update staleness score
CREATE OR REPLACE FUNCTION calculate_staleness_score(
    p_visit_count INTEGER,
    p_last_visited TIMESTAMP WITH TIME ZONE,
    p_last_modified TIMESTAMP WITH TIME ZONE
)
RETURNS DECIMAL(10, 4) AS $$
DECLARE
    time_since_visit INTERVAL;
    time_since_modified INTERVAL;
    days_since_visit DECIMAL;
    days_since_modified DECIMAL;
    modifications_estimate INTEGER;
    staleness DECIMAL(10, 4);
BEGIN
    -- Handle never visited files
    IF p_last_visited IS NULL THEN
        time_since_modified := NOW() - p_last_modified;
        days_since_modified := EXTRACT(EPOCH FROM time_since_modified) / 86400.0;
        -- High staleness for never-visited files
        RETURN LEAST(100.0, days_since_modified * 2.0);
    END IF;
    
    -- Calculate time intervals
    time_since_visit := NOW() - p_last_visited;
    time_since_modified := p_last_visited - p_last_modified;
    
    -- Convert to days
    days_since_visit := EXTRACT(EPOCH FROM time_since_visit) / 86400.0;
    days_since_modified := ABS(EXTRACT(EPOCH FROM time_since_modified) / 86400.0);
    
    -- Estimate modifications (simplified: assume 1 mod per week if modified after visit)
    IF p_last_modified > p_last_visited THEN
        modifications_estimate := GREATEST(1, FLOOR(days_since_modified / 7.0));
    ELSE
        modifications_estimate := 0;
    END IF;
    
    -- Calculate staleness: (modifications × days_since_visit) / (visit_count + 1)
    staleness := (modifications_estimate * days_since_visit) / (p_visit_count + 1.0);
    
    -- Cap at 100 for readability
    RETURN LEAST(100.0, staleness);
END;
$$ LANGUAGE plpgsql;

-- Trigger to update staleness score on visit or modification
CREATE OR REPLACE FUNCTION update_staleness_score()
RETURNS TRIGGER AS $$
BEGIN
    -- Update staleness score for the tracked file
    UPDATE tracked_files
    SET staleness_score = calculate_staleness_score(
        visit_count,
        last_visited,
        last_modified
    )
    WHERE id = COALESCE(NEW.file_id, NEW.id, OLD.id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

-- Trigger when a visit is recorded
CREATE TRIGGER update_staleness_on_visit
    AFTER INSERT ON visits
    FOR EACH ROW
    EXECUTE FUNCTION update_staleness_score();

-- Trigger when file metadata is updated
CREATE TRIGGER update_staleness_on_file_update
    AFTER UPDATE OF last_modified, visit_count ON tracked_files
    FOR EACH ROW
    EXECUTE FUNCTION update_staleness_score();

-- Function to record a visit and update file statistics
CREATE OR REPLACE FUNCTION record_visit(
    p_campaign_id UUID,
    p_file_path VARCHAR,
    p_absolute_path VARCHAR,
    p_context VARCHAR DEFAULT NULL,
    p_agent VARCHAR DEFAULT NULL,
    p_conversation_id VARCHAR DEFAULT NULL,
    p_duration_ms INTEGER DEFAULT NULL,
    p_findings JSONB DEFAULT NULL
)
RETURNS UUID AS $$
DECLARE
    v_file_id UUID;
    v_visit_id UUID;
BEGIN
    -- Get or create the tracked file for this campaign
    INSERT INTO tracked_files (campaign_id, file_path, absolute_path)
    VALUES (p_campaign_id, p_file_path, p_absolute_path)
    ON CONFLICT (campaign_id, absolute_path) DO NOTHING;
    
    -- Get the file ID
    SELECT id INTO v_file_id
    FROM tracked_files
    WHERE campaign_id = p_campaign_id AND absolute_path = p_absolute_path;
    
    -- Update visit count and last visited time
    UPDATE tracked_files
    SET 
        visit_count = visit_count + 1,
        last_visited = NOW()
    WHERE id = v_file_id;
    
    -- Record the visit
    INSERT INTO visits (file_id, context, agent, conversation_id, duration_ms, findings)
    VALUES (v_file_id, p_context, p_agent, p_conversation_id, p_duration_ms, p_findings)
    RETURNING id INTO v_visit_id;
    
    RETURN v_visit_id;
END;
$$ LANGUAGE plpgsql;

-- Function to get least visited files for a campaign
CREATE OR REPLACE FUNCTION get_least_visited_files(
    p_campaign_id UUID,
    p_limit INTEGER DEFAULT 10,
    p_context VARCHAR DEFAULT NULL,
    p_include_unvisited BOOLEAN DEFAULT TRUE
)
RETURNS TABLE (
    id UUID,
    file_path VARCHAR,
    absolute_path VARCHAR,
    visit_count INTEGER,
    last_visited TIMESTAMP WITH TIME ZONE,
    staleness_score DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        tf.id,
        tf.file_path,
        tf.absolute_path,
        tf.visit_count,
        tf.last_visited,
        tf.staleness_score
    FROM tracked_files tf
    WHERE 
        tf.campaign_id = p_campaign_id
        AND tf.deleted = FALSE
        AND (p_include_unvisited OR tf.visit_count > 0)
        AND (p_context IS NULL OR EXISTS (
            SELECT 1 FROM visits v 
            WHERE v.file_id = tf.id AND v.context = p_context
        ))
    ORDER BY tf.visit_count ASC, tf.staleness_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to get most stale files for a campaign
CREATE OR REPLACE FUNCTION get_most_stale_files(
    p_campaign_id UUID,
    p_limit INTEGER DEFAULT 10,
    p_threshold DECIMAL DEFAULT 0.0
)
RETURNS TABLE (
    id UUID,
    file_path VARCHAR,
    absolute_path VARCHAR,
    visit_count INTEGER,
    last_visited TIMESTAMP WITH TIME ZONE,
    last_modified TIMESTAMP WITH TIME ZONE,
    staleness_score DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        tf.id,
        tf.file_path,
        tf.absolute_path,
        tf.visit_count,
        tf.last_visited,
        tf.last_modified,
        tf.staleness_score
    FROM tracked_files tf
    WHERE 
        tf.campaign_id = p_campaign_id
        AND tf.deleted = FALSE
        AND tf.staleness_score >= p_threshold
    ORDER BY tf.staleness_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to get coverage statistics for a campaign
CREATE OR REPLACE FUNCTION get_coverage_stats(
    p_campaign_id UUID,
    p_patterns VARCHAR[] DEFAULT NULL
)
RETURNS TABLE (
    total_files INTEGER,
    visited_files INTEGER,
    unvisited_files INTEGER,
    coverage_percentage DECIMAL,
    average_visits DECIMAL,
    average_staleness DECIMAL
) AS $$
DECLARE
    pattern VARCHAR;
    pattern_clause TEXT := '';
BEGIN
    -- Build pattern matching clause
    IF p_patterns IS NOT NULL AND array_length(p_patterns, 1) > 0 THEN
        pattern_clause := ' AND (';
        FOREACH pattern IN ARRAY p_patterns
        LOOP
            IF pattern_clause != ' AND (' THEN
                pattern_clause := pattern_clause || ' OR ';
            END IF;
            pattern_clause := pattern_clause || 'file_path LIKE ' || quote_literal(pattern);
        END LOOP;
        pattern_clause := pattern_clause || ')';
    END IF;
    
    -- Calculate statistics
    RETURN QUERY EXECUTE format('
        SELECT 
            COUNT(*)::INTEGER as total_files,
            COUNT(CASE WHEN visit_count > 0 THEN 1 END)::INTEGER as visited_files,
            COUNT(CASE WHEN visit_count = 0 THEN 1 END)::INTEGER as unvisited_files,
            ROUND(COUNT(CASE WHEN visit_count > 0 THEN 1 END)::DECIMAL / NULLIF(COUNT(*), 0) * 100, 2) as coverage_percentage,
            ROUND(AVG(visit_count)::DECIMAL, 2) as average_visits,
            ROUND(AVG(staleness_score)::DECIMAL, 2) as average_staleness
        FROM tracked_files
        WHERE campaign_id = ' || quote_literal(p_campaign_id) || '
        AND deleted = FALSE %s
    ', pattern_clause);
END;
$$ LANGUAGE plpgsql;