-- Prompt Manager Database Schema
-- Personal prompt storage and management system

-- Simplified user system for personal use
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(100) UNIQUE NOT NULL DEFAULT 'default',
    email VARCHAR(255),
    name VARCHAR(255) DEFAULT 'Personal User',
    settings JSONB DEFAULT '{"theme": "dark", "defaultCampaign": null}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Campaign organization system (debugging, UX, coding, etc.)
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#6366f1',
    icon VARCHAR(50) DEFAULT 'folder',
    parent_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    sort_order INTEGER DEFAULT 0,
    is_favorite BOOLEAN DEFAULT false,
    prompt_count INTEGER DEFAULT 0,
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Core prompt storage with file-based content
CREATE TABLE IF NOT EXISTS prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    file_path VARCHAR(500) NOT NULL, -- Path to prompt file relative to initialization/prompts/
    content_cache TEXT, -- Optional cache of file content for performance
    description TEXT,
    variables JSONB DEFAULT '[]'::jsonb,
    
    -- Personal usage tracking
    usage_count INTEGER DEFAULT 0,
    last_used TIMESTAMP,
    is_favorite BOOLEAN DEFAULT false,
    is_archived BOOLEAN DEFAULT false,
    
    -- Quick access
    quick_access_key VARCHAR(50) UNIQUE,
    
    -- Version control
    version INTEGER DEFAULT 1,
    parent_version_id UUID REFERENCES prompts(id),
    
    -- Metadata
    word_count INTEGER,
    estimated_tokens INTEGER,
    effectiveness_rating INTEGER CHECK (effectiveness_rating >= 1 AND effectiveness_rating <= 5),
    notes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    file_path VARCHAR(500) NOT NULL, -- Path to versioned prompt file
    content_cache TEXT, -- Optional cache
    variables JSONB,
    change_summary TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(prompt_id, version_number)
);

CREATE TABLE IF NOT EXISTS tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) UNIQUE NOT NULL,
    color VARCHAR(7),
    description TEXT
);

CREATE TABLE IF NOT EXISTS prompt_tags (
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (prompt_id, tag_id)
);

-- Quick prompt templates for common patterns
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    file_path VARCHAR(500) NOT NULL, -- Path to template file
    content_cache TEXT, -- Optional cache
    variables JSONB DEFAULT '[]'::jsonb,
    category VARCHAR(100),
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Simple test history for prompt refinement
CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    model VARCHAR(100) NOT NULL DEFAULT 'ollama/llama3.2',
    input_variables JSONB,
    response TEXT,
    response_time FLOAT,
    token_count INTEGER,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    notes TEXT,
    tested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_campaigns_parent ON campaigns(parent_id);
CREATE INDEX idx_campaigns_sort_order ON campaigns(sort_order);
CREATE INDEX idx_campaigns_last_used ON campaigns(last_used DESC);
CREATE INDEX idx_prompts_campaign ON prompts(campaign_id);
CREATE INDEX idx_prompts_last_used ON prompts(last_used DESC);
CREATE INDEX idx_prompts_usage_count ON prompts(usage_count DESC);
CREATE INDEX idx_prompts_favorites ON prompts(is_favorite) WHERE is_favorite = true;
CREATE INDEX idx_prompts_quick_access ON prompts(quick_access_key) WHERE quick_access_key IS NOT NULL;
CREATE INDEX idx_prompt_versions_prompt ON prompt_versions(prompt_id);
CREATE INDEX idx_test_results_prompt ON test_results(prompt_id);

-- Full text search on cached content
CREATE INDEX idx_prompts_content_search ON prompts USING gin(to_tsvector('english', COALESCE(content_cache, '')));
CREATE INDEX idx_prompts_title_search ON prompts USING gin(to_tsvector('english', title));
CREATE INDEX idx_prompts_description_search ON prompts USING gin(to_tsvector('english', COALESCE(description, '')));

-- Function to update campaign prompt count
CREATE OR REPLACE FUNCTION update_campaign_prompt_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE campaigns SET prompt_count = prompt_count + 1, updated_at = CURRENT_TIMESTAMP 
        WHERE id = NEW.campaign_id;
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE campaigns SET prompt_count = prompt_count - 1, updated_at = CURRENT_TIMESTAMP 
        WHERE id = OLD.campaign_id;
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' AND NEW.campaign_id != OLD.campaign_id THEN
        UPDATE campaigns SET prompt_count = prompt_count - 1, updated_at = CURRENT_TIMESTAMP 
        WHERE id = OLD.campaign_id;
        UPDATE campaigns SET prompt_count = prompt_count + 1, updated_at = CURRENT_TIMESTAMP 
        WHERE id = NEW.campaign_id;
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to maintain campaign prompt counts
CREATE TRIGGER trigger_update_campaign_prompt_count
    AFTER INSERT OR DELETE OR UPDATE ON prompts
    FOR EACH ROW EXECUTE FUNCTION update_campaign_prompt_count();