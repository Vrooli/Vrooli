-- Integration Snippets Schema Extension
-- Adds support for storing integration recipes and code snippets for APIs

-- Integration snippets/recipes table
CREATE TABLE IF NOT EXISTS integration_snippets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    
    -- Snippet metadata
    title VARCHAR(255) NOT NULL,
    description TEXT,
    language VARCHAR(50) NOT NULL CHECK (language IN ('javascript', 'typescript', 'python', 'go', 'java', 'ruby', 'php', 'csharp', 'rust', 'bash', 'curl', 'http')),
    framework VARCHAR(100), -- e.g., 'express', 'django', 'spring', 'rails'
    
    -- The actual code snippet
    code TEXT NOT NULL,
    
    -- Additional context
    dependencies JSONB, -- Required packages/libraries
    environment_variables TEXT[], -- Required env vars (names only, not values)
    prerequisites TEXT, -- Setup requirements or prerequisites
    
    -- Categorization
    snippet_type VARCHAR(50) NOT NULL CHECK (snippet_type IN ('authentication', 'basic_request', 'pagination', 'error_handling', 'webhook', 'batch_processing', 'rate_limiting', 'complete_integration')),
    tags TEXT[],
    
    -- Quality indicators
    tested BOOLEAN DEFAULT false,
    official BOOLEAN DEFAULT false, -- From official documentation
    community_verified BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,
    
    -- Voting system
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    
    -- Optional reference to specific endpoint
    endpoint_id UUID REFERENCES endpoints(id) ON DELETE CASCADE,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    source_url VARCHAR(500), -- Link to original source if applicable
    
    -- Version tracking
    version VARCHAR(50), -- API version this snippet works with
    last_verified TIMESTAMP -- When the snippet was last verified to work
);

-- Create indexes for performance
CREATE INDEX idx_snippets_api_id ON integration_snippets(api_id);
CREATE INDEX idx_snippets_language ON integration_snippets(language);
CREATE INDEX idx_snippets_type ON integration_snippets(snippet_type);
CREATE INDEX idx_snippets_tags ON integration_snippets USING GIN(tags);
CREATE INDEX idx_snippets_official ON integration_snippets(official);
CREATE INDEX idx_snippets_tested ON integration_snippets(tested);

-- Add trigger for updated_at
CREATE TRIGGER update_snippets_updated_at BEFORE UPDATE ON integration_snippets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- View for popular snippets
CREATE OR REPLACE VIEW popular_snippets AS
SELECT 
    s.id,
    s.title,
    s.description,
    s.language,
    s.snippet_type,
    a.name as api_name,
    a.provider,
    s.helpful_count,
    s.usage_count,
    s.official,
    s.tested
FROM integration_snippets s
JOIN apis a ON s.api_id = a.id
WHERE s.helpful_count > s.not_helpful_count
ORDER BY s.usage_count DESC, s.helpful_count DESC;

-- View for verified snippets
CREATE OR REPLACE VIEW verified_snippets AS
SELECT 
    s.*,
    a.name as api_name,
    a.provider
FROM integration_snippets s
JOIN apis a ON s.api_id = a.id
WHERE s.tested = true 
   OR s.official = true 
   OR s.community_verified = true;