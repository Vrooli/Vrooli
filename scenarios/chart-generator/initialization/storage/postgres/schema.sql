-- Chart Generator PostgreSQL Schema
-- Creates tables for storing chart templates, styles, and generation history

-- Chart Styles Table
-- Stores custom styling configurations that can be reused across charts
CREATE TABLE IF NOT EXISTS chart_styles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    category VARCHAR(50) NOT NULL DEFAULT 'professional',
    description TEXT,
    color_palette JSONB NOT NULL, -- Array of hex colors
    typography JSONB, -- Font settings: family, sizes, weights
    spacing JSONB, -- Margin, padding, grid spacing settings
    animations JSONB, -- Animation preferences and timings
    is_public BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT valid_category CHECK (category IN ('professional', 'minimal', 'vibrant', 'dark', 'corporate', 'creative', 'technical', 'branded')),
    CONSTRAINT valid_color_palette CHECK (jsonb_array_length(color_palette) >= 1)
);

-- Chart Templates Table  
-- Stores reusable chart configurations and default settings
CREATE TABLE IF NOT EXISTS chart_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(200) NOT NULL,
    chart_type VARCHAR(50) NOT NULL,
    description TEXT,
    default_config JSONB NOT NULL, -- Chart configuration options
    style_id UUID REFERENCES chart_styles(id) ON DELETE SET NULL,
    preview_data JSONB, -- Sample data for preview generation
    is_public BOOLEAN DEFAULT true,
    created_by VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    version VARCHAR(20) DEFAULT '1.0.0',
    
    -- Constraints
    CONSTRAINT valid_chart_type CHECK (chart_type IN ('bar', 'line', 'pie', 'scatter', 'area', 'gantt', 'heatmap', 'treemap', 'candlestick', 'radar'))
);

-- Chart Instances Table
-- Stores actual generated charts with their data and metadata
CREATE TABLE IF NOT EXISTS chart_instances (
    id VARCHAR(100) PRIMARY KEY, -- External chart ID from generation request
    template_id UUID REFERENCES chart_templates(id) ON DELETE SET NULL,
    chart_type VARCHAR(50) NOT NULL,
    data_source JSONB NOT NULL, -- The actual data used to generate the chart
    config_overrides JSONB, -- Any configuration overrides from the template
    generated_files JSONB, -- Array of generated file information {format, filename, filepath, size_bytes}
    generation_metadata JSONB NOT NULL, -- Processing stats, timing, data points count, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE, -- When temporary files should be cleaned up
    
    -- Constraints
    CONSTRAINT valid_chart_type_instance CHECK (chart_type IN ('bar', 'line', 'pie', 'scatter', 'area', 'gantt', 'heatmap', 'treemap', 'candlestick', 'radar'))
);

-- Chart Generation History Table
-- Tracks usage patterns and performance metrics
CREATE TABLE IF NOT EXISTS chart_generation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chart_instance_id VARCHAR(100) REFERENCES chart_instances(id) ON DELETE CASCADE,
    chart_type VARCHAR(50) NOT NULL,
    data_points INTEGER NOT NULL,
    style_used VARCHAR(100),
    formats_requested TEXT[],
    generation_time_ms INTEGER,
    success BOOLEAN NOT NULL DEFAULT true,
    error_message TEXT,
    client_info JSONB, -- User agent, IP, session info if available
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexing for analytics
    INDEX idx_history_chart_type (chart_type),
    INDEX idx_history_created_at (created_at),
    INDEX idx_history_success (success)
);

-- Style Usage Tracking
-- Helps identify popular styles for recommendations
CREATE TABLE IF NOT EXISTS style_usage_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    style_id UUID REFERENCES chart_styles(id) ON DELETE CASCADE,
    style_name VARCHAR(100) NOT NULL,
    usage_count INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    avg_generation_time_ms NUMERIC(10,2),
    success_rate NUMERIC(5,4) DEFAULT 1.0, -- Percentage as decimal (0.95 = 95%)
    
    UNIQUE(style_id)
);

-- Indexes for Performance
CREATE INDEX IF NOT EXISTS idx_chart_styles_category ON chart_styles(category);
CREATE INDEX IF NOT EXISTS idx_chart_styles_public ON chart_styles(is_public) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_chart_styles_default ON chart_styles(is_default) WHERE is_default = true;

CREATE INDEX IF NOT EXISTS idx_chart_templates_type ON chart_templates(chart_type);
CREATE INDEX IF NOT EXISTS idx_chart_templates_public ON chart_templates(is_public) WHERE is_public = true;
CREATE INDEX IF NOT EXISTS idx_chart_templates_style ON chart_templates(style_id);

CREATE INDEX IF NOT EXISTS idx_chart_instances_type ON chart_instances(chart_type);
CREATE INDEX IF NOT EXISTS idx_chart_instances_created ON chart_instances(created_at);
CREATE INDEX IF NOT EXISTS idx_chart_instances_expires ON chart_instances(expires_at);

-- Views for Common Queries

-- Popular chart types view
CREATE OR REPLACE VIEW popular_chart_types AS
SELECT 
    chart_type,
    COUNT(*) as usage_count,
    AVG(generation_time_ms) as avg_generation_time_ms,
    AVG(data_points) as avg_data_points,
    SUM(CASE WHEN success THEN 1 ELSE 0 END)::FLOAT / COUNT(*) as success_rate
FROM chart_generation_history 
WHERE created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY chart_type
ORDER BY usage_count DESC;

-- Style performance metrics view
CREATE OR REPLACE VIEW style_performance AS
SELECT 
    cs.id,
    cs.name,
    cs.category,
    COALESCE(sus.usage_count, 0) as usage_count,
    sus.avg_generation_time_ms,
    sus.success_rate,
    sus.last_used_at,
    cs.created_at as style_created_at
FROM chart_styles cs
LEFT JOIN style_usage_stats sus ON cs.id = sus.style_id
ORDER BY COALESCE(sus.usage_count, 0) DESC, cs.created_at DESC;

-- Active chart instances (not expired)
CREATE OR REPLACE VIEW active_chart_instances AS
SELECT 
    ci.*,
    ct.name as template_name,
    cs.name as style_name
FROM chart_instances ci
LEFT JOIN chart_templates ct ON ci.template_id = ct.id
LEFT JOIN chart_styles cs ON ct.style_id = cs.id
WHERE ci.expires_at IS NULL OR ci.expires_at > CURRENT_TIMESTAMP;

-- Functions for Maintenance

-- Function to clean up expired chart instances
CREATE OR REPLACE FUNCTION cleanup_expired_charts()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM chart_instances 
    WHERE expires_at IS NOT NULL AND expires_at < CURRENT_TIMESTAMP;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Also clean up corresponding generation history older than 90 days
    DELETE FROM chart_generation_history 
    WHERE created_at < CURRENT_DATE - INTERVAL '90 days';
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to update style usage statistics
CREATE OR REPLACE FUNCTION update_style_usage_stats()
RETURNS VOID AS $$
BEGIN
    INSERT INTO style_usage_stats (style_id, style_name, usage_count, last_used_at, avg_generation_time_ms, success_rate)
    SELECT 
        cs.id,
        cs.name,
        COUNT(cgh.id) as usage_count,
        MAX(cgh.created_at) as last_used_at,
        AVG(cgh.generation_time_ms) as avg_generation_time_ms,
        AVG(CASE WHEN cgh.success THEN 1.0 ELSE 0.0 END) as success_rate
    FROM chart_styles cs
    LEFT JOIN chart_templates ct ON cs.id = ct.style_id
    LEFT JOIN chart_instances ci ON ct.id = ci.template_id
    LEFT JOIN chart_generation_history cgh ON ci.id = cgh.chart_instance_id
    WHERE cgh.created_at >= CURRENT_DATE - INTERVAL '30 days' OR cgh.id IS NULL
    GROUP BY cs.id, cs.name
    ON CONFLICT (style_id) 
    DO UPDATE SET
        usage_count = EXCLUDED.usage_count,
        last_used_at = EXCLUDED.last_used_at,
        avg_generation_time_ms = EXCLUDED.avg_generation_time_ms,
        success_rate = EXCLUDED.success_rate;
END;
$$ LANGUAGE plpgsql;

-- Triggers for automatic maintenance

-- Update chart template timestamp on changes
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_chart_styles_updated_at 
    BEFORE UPDATE ON chart_styles
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chart_templates_updated_at 
    BEFORE UPDATE ON chart_templates
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Auto-create generation history entry when chart instance is created
CREATE OR REPLACE FUNCTION create_generation_history_entry()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO chart_generation_history (
        chart_instance_id,
        chart_type,
        data_points,
        style_used,
        generation_time_ms,
        success,
        created_at
    ) VALUES (
        NEW.id,
        NEW.chart_type,
        COALESCE(jsonb_array_length(NEW.data_source), 0),
        COALESCE(NEW.generation_metadata->>'style_applied', 'unknown'),
        COALESCE((NEW.generation_metadata->>'generation_time_ms')::INTEGER, 0),
        true,
        NEW.created_at
    );
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER create_history_on_chart_creation
    AFTER INSERT ON chart_instances
    FOR EACH ROW 
    EXECUTE FUNCTION create_generation_history_entry();

-- Comments for documentation
COMMENT ON TABLE chart_styles IS 'Stores reusable chart styling configurations with color palettes, typography, and animation settings';
COMMENT ON TABLE chart_templates IS 'Stores chart templates with default configurations that can be used across multiple chart generations';
COMMENT ON TABLE chart_instances IS 'Stores actual generated charts with their data, configuration, and file information';
COMMENT ON TABLE chart_generation_history IS 'Tracks chart generation usage patterns and performance metrics for analytics';
COMMENT ON TABLE style_usage_stats IS 'Aggregated statistics on style usage for recommendations and performance monitoring';

COMMENT ON FUNCTION cleanup_expired_charts() IS 'Removes expired chart instances and old generation history entries';
COMMENT ON FUNCTION update_style_usage_stats() IS 'Updates aggregated style usage statistics based on recent generation history';

-- Grant appropriate permissions (assuming standard application roles)
-- Note: In production, these should be more restrictive based on actual application roles
GRANT SELECT, INSERT, UPDATE ON chart_styles TO PUBLIC;
GRANT SELECT, INSERT, UPDATE ON chart_templates TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON chart_instances TO PUBLIC;
GRANT SELECT, INSERT ON chart_generation_history TO PUBLIC;
GRANT SELECT, INSERT, UPDATE ON style_usage_stats TO PUBLIC;

GRANT SELECT ON popular_chart_types TO PUBLIC;
GRANT SELECT ON style_performance TO PUBLIC;
GRANT SELECT ON active_chart_instances TO PUBLIC;

-- Complete schema setup
SELECT 'Chart Generator PostgreSQL schema initialized successfully' AS status;