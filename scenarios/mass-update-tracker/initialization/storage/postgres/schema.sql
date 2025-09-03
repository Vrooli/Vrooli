-- Mass Update Tracker Database Schema

-- Campaigns table: stores campaign metadata and configuration
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    file_patterns JSONB NOT NULL, -- Array of glob patterns
    working_directory VARCHAR(1000) NOT NULL,
    scenario_name VARCHAR(255),
    metadata JSONB NOT NULL DEFAULT '{}',
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'paused', 'completed', 'failed')),
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- File entries table: tracks individual file status within campaigns
CREATE TABLE IF NOT EXISTS file_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    file_path VARCHAR(1000) NOT NULL, -- Relative to working_directory
    absolute_path VARCHAR(1000) NOT NULL, -- Full file path
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'completed', 'failed', 'skipped')),
    error_message TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB NOT NULL DEFAULT '{}'
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_scenario ON campaigns(scenario_name);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_file_entries_campaign_id ON file_entries(campaign_id);
CREATE INDEX IF NOT EXISTS idx_file_entries_status ON file_entries(status);
CREATE INDEX IF NOT EXISTS idx_file_entries_campaign_status ON file_entries(campaign_id, status);
CREATE INDEX IF NOT EXISTS idx_file_entries_file_path ON file_entries(campaign_id, file_path);

-- Updated_at trigger for campaigns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_campaigns_updated_at
    BEFORE UPDATE ON campaigns
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Automatically update campaign status based on file completion
CREATE OR REPLACE FUNCTION update_campaign_status()
RETURNS TRIGGER AS $$
DECLARE
    total_files INTEGER;
    completed_files INTEGER;
    failed_files INTEGER;
    campaign_status VARCHAR(50);
BEGIN
    -- Count files in the campaign
    SELECT COUNT(*) INTO total_files
    FROM file_entries
    WHERE campaign_id = COALESCE(NEW.campaign_id, OLD.campaign_id);
    
    -- Count completed files
    SELECT COUNT(*) INTO completed_files
    FROM file_entries
    WHERE campaign_id = COALESCE(NEW.campaign_id, OLD.campaign_id)
    AND status IN ('completed', 'skipped');
    
    -- Count failed files
    SELECT COUNT(*) INTO failed_files
    FROM file_entries
    WHERE campaign_id = COALESCE(NEW.campaign_id, OLD.campaign_id)
    AND status = 'failed';
    
    -- Determine campaign status
    IF completed_files = total_files THEN
        campaign_status := 'completed';
    ELSIF failed_files > 0 AND (completed_files + failed_files) = total_files THEN
        campaign_status := 'failed';
    ELSE
        campaign_status := 'active';
    END IF;
    
    -- Update campaign status and completion time
    UPDATE campaigns
    SET 
        status = campaign_status,
        completed_at = CASE WHEN campaign_status = 'completed' THEN NOW() ELSE NULL END
    WHERE id = COALESCE(NEW.campaign_id, OLD.campaign_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_campaign_status_trigger
    AFTER INSERT OR UPDATE OR DELETE ON file_entries
    FOR EACH ROW
    EXECUTE FUNCTION update_campaign_status();