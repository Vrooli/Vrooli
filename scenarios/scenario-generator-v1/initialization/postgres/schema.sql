-- Scenario Generator V1 - Simple Database Schema
-- PostgreSQL database for AI-powered scenario generation

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Scenarios table - tracks generated scenarios
CREATE TABLE IF NOT EXISTS scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    prompt TEXT NOT NULL,
    
    -- Generated content
    files_generated JSONB DEFAULT '{}',
    resources_used TEXT[] DEFAULT '{}',
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'requested' CHECK (status IN ('requested', 'generating', 'completed', 'failed')),
    generation_id VARCHAR(255),
    
    -- Metadata
    complexity VARCHAR(20) DEFAULT 'intermediate' CHECK (complexity IN ('simple', 'intermediate', 'advanced')),
    category VARCHAR(50) DEFAULT 'general',
    estimated_revenue INTEGER DEFAULT 15000,
    
    -- Claude Code integration
    claude_prompt TEXT,
    claude_response TEXT,
    generation_error TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_scenarios_status ON scenarios(status);
CREATE INDEX IF NOT EXISTS idx_scenarios_created_at ON scenarios(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scenarios_category ON scenarios(category);
CREATE INDEX IF NOT EXISTS idx_scenarios_complexity ON scenarios(complexity);
CREATE INDEX IF NOT EXISTS idx_scenarios_generation_id ON scenarios(generation_id);

-- Generation logs table - tracks AI interactions
CREATE TABLE IF NOT EXISTS generation_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    
    -- Log entry details
    step VARCHAR(100) NOT NULL,
    prompt TEXT NOT NULL,
    response TEXT,
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    
    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER
);

CREATE INDEX IF NOT EXISTS idx_generation_logs_scenario_id ON generation_logs(scenario_id);
CREATE INDEX IF NOT EXISTS idx_generation_logs_step ON generation_logs(step);
CREATE INDEX IF NOT EXISTS idx_generation_logs_started_at ON generation_logs(started_at DESC);

-- Scenario templates table - reusable patterns
CREATE TABLE IF NOT EXISTS scenario_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    
    -- Template content
    prompt_template TEXT NOT NULL,
    resources_suggested TEXT[] DEFAULT '{}',
    complexity_level VARCHAR(20) DEFAULT 'intermediate',
    estimated_revenue_min INTEGER DEFAULT 10000,
    estimated_revenue_max INTEGER DEFAULT 30000,
    
    -- Usage tracking
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2) DEFAULT 0,
    
    -- Template metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_scenario_templates_category ON scenario_templates(category);
CREATE INDEX IF NOT EXISTS idx_scenario_templates_complexity ON scenario_templates(complexity_level);
CREATE INDEX IF NOT EXISTS idx_scenario_templates_active ON scenario_templates(is_active) WHERE is_active = true;

-- Insert some default templates
INSERT INTO scenario_templates (name, description, category, prompt_template, resources_suggested, complexity_level, estimated_revenue_min, estimated_revenue_max) VALUES
('Simple SaaS Dashboard', 'A basic business dashboard for managing data and metrics', 'business-tool', 'Create a SaaS dashboard application that allows users to view and manage business metrics. Include user authentication, data visualization, and basic CRUD operations. Target audience: small business owners.', ARRAY['postgres', 'windmill'], 'simple', 10000, 20000),
('AI Document Processor', 'Intelligent document processing pipeline', 'ai-automation', 'Build an AI-powered document processing system that can extract data from PDFs, process it with AI, and provide structured output. Include file upload, processing status tracking, and results export.', ARRAY['postgres', 'unstructured-io', 'ollama'], 'intermediate', 20000, 35000),
('E-commerce Assistant', 'AI-powered e-commerce management tool', 'e-commerce', 'Create an AI assistant for e-commerce businesses that can manage inventory, process orders, handle customer inquiries, and generate product descriptions automatically.', ARRAY['postgres', 'ollama', 'n8n'], 'advanced', 30000, 50000),
('Content Generation Platform', 'Automated content creation system', 'content-marketing', 'Build a platform that generates blog posts, social media content, and marketing materials using AI. Include content templates, scheduling, and multi-platform publishing.', ARRAY['postgres', 'ollama', 'comfyui'], 'intermediate', 15000, 30000),
('Customer Support Bot', 'Intelligent customer service automation', 'customer-service', 'Create an AI-powered customer support system with ticket management, automated responses, knowledge base integration, and escalation workflows.', ARRAY['postgres', 'ollama', 'n8n'], 'intermediate', 18000, 32000);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_scenarios_updated_at BEFORE UPDATE ON scenarios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scenario_templates_updated_at BEFORE UPDATE ON scenario_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();