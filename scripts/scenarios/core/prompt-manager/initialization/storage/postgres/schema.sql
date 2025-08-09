-- Prompt Manager Database Schema

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'editor',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP,
    settings JSONB DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    parent_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
    client VARCHAR(255),
    project VARCHAR(255),
    status VARCHAR(50) DEFAULT 'active',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    description TEXT,
    variables JSONB DEFAULT '[]'::jsonb,
    version INTEGER DEFAULT 1,
    parent_version_id UUID REFERENCES prompts(id),
    created_by UUID REFERENCES users(id),
    status VARCHAR(50) DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS prompt_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    content TEXT NOT NULL,
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

CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]'::jsonb,
    category VARCHAR(100),
    is_public BOOLEAN DEFAULT false,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    execution_count INTEGER DEFAULT 0,
    success_rate FLOAT,
    avg_response_time FLOAT,
    avg_token_count INTEGER,
    cost_estimate FLOAT,
    performance_score FLOAT,
    last_used TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    provider VARCHAR(100) NOT NULL,
    model VARCHAR(100) NOT NULL,
    input_variables JSONB,
    response TEXT,
    response_time FLOAT,
    token_count INTEGER,
    cost FLOAT,
    rating INTEGER,
    notes TEXT,
    tested_by UUID REFERENCES users(id),
    tested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS ab_tests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    variant_a UUID REFERENCES prompts(id),
    variant_b UUID REFERENCES prompts(id),
    status VARCHAR(50) DEFAULT 'running',
    winner UUID REFERENCES prompts(id),
    confidence_level FLOAT,
    start_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_date TIMESTAMP,
    results JSONB
);

-- Indexes for performance
CREATE INDEX idx_campaigns_parent ON campaigns(parent_id);
CREATE INDEX idx_campaigns_owner ON campaigns(owner_id);
CREATE INDEX idx_prompts_campaign ON prompts(campaign_id);
CREATE INDEX idx_prompts_created_by ON prompts(created_by);
CREATE INDEX idx_prompt_versions_prompt ON prompt_versions(prompt_id);
CREATE INDEX idx_metrics_prompt ON metrics(prompt_id);
CREATE INDEX idx_test_results_prompt ON test_results(prompt_id);

-- Full text search
CREATE INDEX idx_prompts_content_search ON prompts USING gin(to_tsvector('english', content));
CREATE INDEX idx_prompts_title_search ON prompts USING gin(to_tsvector('english', title));

-- Default data
INSERT INTO tags (name, color, description) VALUES
    ('marketing', '#FF6B6B', 'Marketing and advertising prompts'),
    ('technical', '#4ECDC4', 'Technical documentation and code'),
    ('creative', '#95E77E', 'Creative writing and content'),
    ('analysis', '#FFE66D', 'Data analysis and research'),
    ('customer', '#A8E6CF', 'Customer communication')
ON CONFLICT (name) DO NOTHING;