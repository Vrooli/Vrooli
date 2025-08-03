-- Scenario Generator V1 Database Schema
-- PostgreSQL database for tracking scenario generation history and metadata

-- Create database if not exists (run as superuser)
-- CREATE DATABASE scenario_generator;

-- Use the scenario_generator database
-- \c scenario_generator;

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Main table for tracking scenario generations
CREATE TABLE IF NOT EXISTS scenario_generations (
    -- Primary identification
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Input data
    customer_input TEXT NOT NULL,
    complexity VARCHAR(20) NOT NULL CHECK (complexity IN ('simple', 'intermediate', 'advanced')),
    category VARCHAR(50) NOT NULL,
    
    -- Generation metadata
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    scenario_id VARCHAR(100) UNIQUE NOT NULL,
    scenario_name VARCHAR(255) NOT NULL,
    
    -- Resource configuration
    resources_required JSONB NOT NULL DEFAULT '[]'::jsonb,
    resources_optional JSONB DEFAULT '[]'::jsonb,
    
    -- Status tracking
    status VARCHAR(50) NOT NULL DEFAULT 'generating' 
        CHECK (status IN ('generating', 'completed', 'failed', 'deployed', 'archived')),
    generation_time_ms INTEGER,
    error_message TEXT,
    
    -- Business metrics
    estimated_revenue JSONB DEFAULT '{"min": 0, "max": 0, "currency": "USD"}'::jsonb,
    actual_revenue DECIMAL(10, 2),
    deployment_count INTEGER DEFAULT 0,
    
    -- Storage information
    storage_path VARCHAR(500),
    storage_size_bytes BIGINT,
    
    -- Additional metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    
    -- Audit fields
    created_by VARCHAR(100) DEFAULT 'system',
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    version INTEGER DEFAULT 1
);

-- Create indexes for performance
CREATE INDEX idx_scenario_generations_status ON scenario_generations(status);
CREATE INDEX idx_scenario_generations_category ON scenario_generations(category);
CREATE INDEX idx_scenario_generations_generated_at ON scenario_generations(generated_at DESC);
CREATE INDEX idx_scenario_generations_scenario_id ON scenario_generations(scenario_id);
CREATE INDEX idx_scenario_generations_tags ON scenario_generations USING GIN(tags);
CREATE INDEX idx_scenario_generations_resources ON scenario_generations USING GIN(resources_required);

-- Table for tracking scenario deployments
CREATE TABLE IF NOT EXISTS scenario_deployments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_generation_id UUID NOT NULL REFERENCES scenario_generations(id) ON DELETE CASCADE,
    
    -- Deployment information
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    environment VARCHAR(50) NOT NULL CHECK (environment IN ('development', 'staging', 'production')),
    deployment_url VARCHAR(500),
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'deploying'
        CHECK (status IN ('deploying', 'active', 'stopped', 'failed')),
    
    -- Metrics
    uptime_hours DECIMAL(10, 2),
    request_count BIGINT DEFAULT 0,
    error_rate DECIMAL(5, 2),
    
    -- Configuration
    configuration JSONB DEFAULT '{}'::jsonb,
    resources_allocated JSONB DEFAULT '{}'::jsonb,
    
    -- Audit
    deployed_by VARCHAR(100),
    stopped_at TIMESTAMP WITH TIME ZONE,
    notes TEXT
);

CREATE INDEX idx_deployments_scenario_id ON scenario_deployments(scenario_generation_id);
CREATE INDEX idx_deployments_status ON scenario_deployments(status);
CREATE INDEX idx_deployments_deployed_at ON scenario_deployments(deployed_at DESC);

-- Table for tracking generation analytics
CREATE TABLE IF NOT EXISTS generation_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Time buckets
    date DATE NOT NULL,
    hour INTEGER CHECK (hour >= 0 AND hour < 24),
    
    -- Metrics
    total_generations INTEGER DEFAULT 0,
    successful_generations INTEGER DEFAULT 0,
    failed_generations INTEGER DEFAULT 0,
    avg_generation_time_ms INTEGER,
    
    -- Resource usage
    most_used_resources JSONB DEFAULT '[]'::jsonb,
    resource_combinations JSONB DEFAULT '{}'::jsonb,
    
    -- Categories
    category_breakdown JSONB DEFAULT '{}'::jsonb,
    complexity_breakdown JSONB DEFAULT '{}'::jsonb,
    
    -- Revenue
    estimated_revenue_total DECIMAL(12, 2) DEFAULT 0,
    
    -- Unique constraint
    UNIQUE(date, hour)
);

CREATE INDEX idx_analytics_date ON generation_analytics(date DESC);

-- Table for storing generation templates and patterns
CREATE TABLE IF NOT EXISTS generation_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Template information
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    
    -- Pattern data
    input_pattern TEXT,
    resource_pattern JSONB DEFAULT '[]'::jsonb,
    workflow_pattern JSONB DEFAULT '{}'::jsonb,
    ui_pattern JSONB DEFAULT '{}'::jsonb,
    
    -- Usage metrics
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5, 2),
    avg_revenue DECIMAL(10, 2),
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    tags TEXT[] DEFAULT '{}'
);

CREATE INDEX idx_templates_name ON generation_templates(name);
CREATE INDEX idx_templates_category ON generation_templates(category);
CREATE INDEX idx_templates_active ON generation_templates(is_active);

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    NEW.version = OLD.version + 1;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_scenario_generations_updated_at 
    BEFORE UPDATE ON scenario_generations 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_generation_templates_updated_at 
    BEFORE UPDATE ON generation_templates 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to update analytics on generation completion
CREATE OR REPLACE FUNCTION update_generation_analytics()
RETURNS TRIGGER AS $$
BEGIN
    -- Only process completed or failed generations
    IF NEW.status IN ('completed', 'failed') AND OLD.status = 'generating' THEN
        INSERT INTO generation_analytics (
            date,
            hour,
            total_generations,
            successful_generations,
            failed_generations,
            avg_generation_time_ms,
            estimated_revenue_total
        ) VALUES (
            DATE(NEW.generated_at),
            EXTRACT(HOUR FROM NEW.generated_at),
            1,
            CASE WHEN NEW.status = 'completed' THEN 1 ELSE 0 END,
            CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            NEW.generation_time_ms,
            CASE 
                WHEN NEW.status = 'completed' 
                THEN COALESCE((NEW.estimated_revenue->>'max')::DECIMAL, 0)
                ELSE 0
            END
        )
        ON CONFLICT (date, hour) DO UPDATE SET
            total_generations = generation_analytics.total_generations + 1,
            successful_generations = generation_analytics.successful_generations + 
                CASE WHEN NEW.status = 'completed' THEN 1 ELSE 0 END,
            failed_generations = generation_analytics.failed_generations + 
                CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            avg_generation_time_ms = (
                generation_analytics.avg_generation_time_ms * generation_analytics.total_generations + 
                NEW.generation_time_ms
            ) / (generation_analytics.total_generations + 1),
            estimated_revenue_total = generation_analytics.estimated_revenue_total + 
                CASE 
                    WHEN NEW.status = 'completed' 
                    THEN COALESCE((NEW.estimated_revenue->>'max')::DECIMAL, 0)
                    ELSE 0
                END;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for analytics update
CREATE TRIGGER update_analytics_on_generation 
    AFTER UPDATE ON scenario_generations 
    FOR EACH ROW 
    EXECUTE FUNCTION update_generation_analytics();

-- Insert some example templates
INSERT INTO generation_templates (name, description, category, resource_pattern, tags) VALUES
('Customer Support Bot', 'AI-powered customer service chatbot', 'customer-service', 
 '["ollama", "n8n", "postgres", "redis"]'::jsonb, 
 '{"chatbot", "support", "ai"}'),
('Document Processor', 'Automated document extraction and processing', 'data-processing',
 '["unstructured-io", "n8n", "minio", "postgres"]'::jsonb,
 '{"documents", "extraction", "automation"}'),
('Content Generator', 'AI content creation for social media', 'content-generation',
 '["ollama", "comfyui", "n8n", "minio"]'::jsonb,
 '{"content", "social-media", "ai", "images"}')
ON CONFLICT (name) DO NOTHING;

-- Create view for generation statistics
CREATE OR REPLACE VIEW generation_stats AS
SELECT 
    COUNT(*) as total_scenarios,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_scenarios,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_scenarios,
    COUNT(CASE WHEN status = 'deployed' THEN 1 END) as deployed_scenarios,
    AVG(generation_time_ms) as avg_generation_time_ms,
    SUM(COALESCE((estimated_revenue->>'max')::DECIMAL, 0)) as total_revenue_potential,
    array_agg(DISTINCT category) as categories_used,
    COUNT(DISTINCT DATE(generated_at)) as days_active
FROM scenario_generations;

-- Create view for resource usage statistics
CREATE OR REPLACE VIEW resource_usage_stats AS
SELECT 
    resource,
    COUNT(*) as usage_count,
    AVG(generation_time_ms) as avg_generation_time,
    AVG((estimated_revenue->>'max')::DECIMAL) as avg_revenue
FROM scenario_generations,
     jsonb_array_elements_text(resources_required) as resource
WHERE status = 'completed'
GROUP BY resource
ORDER BY usage_count DESC;

-- Grant permissions (adjust as needed)
-- GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO scenario_generator_user;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO scenario_generator_user;

-- Initial success message
DO $$
BEGIN
    RAISE NOTICE 'Scenario Generator V1 database schema created successfully';
    RAISE NOTICE 'Tables created: scenario_generations, scenario_deployments, generation_analytics, generation_templates';
    RAISE NOTICE 'Views created: generation_stats, resource_usage_stats';
END $$;