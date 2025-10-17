-- Schema updates for API Library v2 features
-- Adds support for version tracking, enhanced pricing, and cost calculation

-- Add version column to apis table if not exists
ALTER TABLE apis ADD COLUMN IF NOT EXISTS version VARCHAR(50);

-- Create api_versions table for version history tracking
CREATE TABLE IF NOT EXISTS api_versions (
    id UUID PRIMARY KEY,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    change_summary TEXT,
    breaking_changes BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system',
    
    INDEX idx_api_versions_api_id (api_id),
    INDEX idx_api_versions_created_at (created_at)
);

-- Add updated_at column to pricing_tiers for refresh tracking
ALTER TABLE pricing_tiers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

-- Create cost_calculations table to store cost analysis history
CREATE TABLE IF NOT EXISTS cost_calculations (
    id UUID PRIMARY KEY,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    requests_per_month INTEGER NOT NULL,
    data_per_request_mb DECIMAL(10,3),
    recommended_tier VARCHAR(255),
    estimated_cost DECIMAL(12,2),
    cost_breakdown JSONB,
    savings_tip TEXT,
    alternatives JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_cost_calculations_api_id (api_id),
    INDEX idx_cost_calculations_created_at (created_at)
);

-- Create pricing_refresh_log table for tracking automatic refresh
CREATE TABLE IF NOT EXISTS pricing_refresh_log (
    id UUID PRIMARY KEY,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    pricing_url TEXT,
    refresh_status VARCHAR(50), -- success, failed, partial
    items_updated INTEGER DEFAULT 0,
    error_message TEXT,
    refreshed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_pricing_refresh_api_id (api_id),
    INDEX idx_pricing_refresh_date (refreshed_at)
);

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_apis_version ON apis(version);
CREATE INDEX IF NOT EXISTS idx_apis_last_refreshed ON apis(last_refreshed);
CREATE INDEX IF NOT EXISTS idx_pricing_tiers_updated ON pricing_tiers(updated_at);

-- Function to track API version changes
CREATE OR REPLACE FUNCTION track_api_version_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.version IS DISTINCT FROM NEW.version THEN
        INSERT INTO api_versions (
            id, 
            api_id, 
            version, 
            change_summary,
            breaking_changes,
            created_at
        ) VALUES (
            gen_random_uuid(),
            NEW.id,
            NEW.version,
            'Version updated from ' || COALESCE(OLD.version, 'null') || ' to ' || NEW.version,
            FALSE,
            NOW()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic version tracking
DROP TRIGGER IF EXISTS api_version_change_trigger ON apis;
CREATE TRIGGER api_version_change_trigger
    AFTER UPDATE ON apis
    FOR EACH ROW
    EXECUTE FUNCTION track_api_version_change();

-- Function to calculate estimated costs
CREATE OR REPLACE FUNCTION calculate_api_cost(
    p_api_id UUID,
    p_requests_per_month INTEGER,
    p_data_per_request_mb DECIMAL
)
RETURNS TABLE (
    tier_name VARCHAR(255),
    estimated_cost DECIMAL(12,2),
    base_cost DECIMAL(12,2),
    request_cost DECIMAL(12,2),
    data_cost DECIMAL(12,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        pt.name AS tier_name,
        (pt.monthly_cost + 
         GREATEST(0, p_requests_per_month - pt.free_tier_requests) * pt.price_per_request +
         p_requests_per_month * p_data_per_request_mb * pt.price_per_mb) AS estimated_cost,
        pt.monthly_cost AS base_cost,
        GREATEST(0, p_requests_per_month - pt.free_tier_requests) * pt.price_per_request AS request_cost,
        p_requests_per_month * p_data_per_request_mb * pt.price_per_mb AS data_cost
    FROM pricing_tiers pt
    WHERE pt.api_id = p_api_id
    ORDER BY estimated_cost ASC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Add sample data for testing (optional)
-- This can be commented out in production
INSERT INTO api_versions (id, api_id, version, change_summary, breaking_changes, created_at)
SELECT 
    gen_random_uuid(),
    id,
    '1.0.0',
    'Initial version',
    false,
    NOW() - INTERVAL '30 days'
FROM apis
WHERE NOT EXISTS (
    SELECT 1 FROM api_versions WHERE api_id = apis.id
)
LIMIT 3;