-- Scalable App Cookbook Database Schema
-- Stores architectural patterns, recipes, and implementations

-- Schema for scalable_app_cookbook database
-- Note: Database should already exist

-- Enable full-text search extension
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- Patterns table - Core architectural patterns from the cookbook
CREATE TABLE scalable_app_cookbook.patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    chapter VARCHAR(100) NOT NULL, -- e.g., "Part A - Architectural Foundations"
    section VARCHAR(100) NOT NULL, -- e.g., "Architecture Styles & Boundaries"
    maturity_level VARCHAR(2) NOT NULL CHECK (maturity_level IN ('L0', 'L1', 'L2', 'L3', 'L4')),
    tags TEXT[] DEFAULT '{}',
    what_and_why TEXT NOT NULL,
    when_to_use TEXT NOT NULL,
    tradeoffs TEXT NOT NULL,
    reference_patterns TEXT[] DEFAULT '{}', -- Links to other patterns
    failure_modes TEXT,
    cost_levers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Recipes table - Step-by-step implementation guides
CREATE TABLE recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('greenfield', 'brownfield', 'migration')),
    prerequisites TEXT[] DEFAULT '{}',
    steps JSONB NOT NULL, -- Array of step objects with id, desc, cmds, code, etc.
    config_snippets JSONB DEFAULT '{}', -- Terraform, Helm, Docker configs
    validation_checks TEXT[] DEFAULT '{}',
    artifacts TEXT[] DEFAULT '{}', -- Expected outputs (ADRs, dashboards, etc.)
    metrics TEXT[] DEFAULT '{}', -- SLIs/SLOs to track
    rollbacks TEXT[] DEFAULT '{}', -- How to undo if needed
    prompts TEXT[] DEFAULT '{}', -- Agent-ready task prompts
    timeout_sec INTEGER DEFAULT 300,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Implementations table - Language-specific code examples
CREATE TABLE implementations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    language VARCHAR(20) NOT NULL CHECK (language IN ('go', 'javascript', 'typescript', 'python', 'java', 'rust', 'csharp')),
    code TEXT NOT NULL,
    file_path VARCHAR(500), -- Suggested file location
    description TEXT,
    dependencies TEXT[] DEFAULT '{}', -- Required packages/libraries
    test_code TEXT, -- Unit tests for the implementation
    benchmark_data JSONB, -- Performance metrics if available
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- CI Gates table - Quality gates and policies for each pattern
CREATE TABLE ci_gates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    gate_type VARCHAR(50) NOT NULL, -- lint, slo, security, a11y, performance
    gate_name VARCHAR(100) NOT NULL,
    gate_config JSONB NOT NULL, -- Configuration for the gate
    enforcement_level VARCHAR(20) DEFAULT 'warning' CHECK (enforcement_level IN ('error', 'warning', 'info')),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Observability table - SLIs, dashboards, alerts for patterns
CREATE TABLE observability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    observability_type VARCHAR(30) NOT NULL, -- sli, dashboard, alert, runbook
    name VARCHAR(100) NOT NULL,
    config JSONB NOT NULL, -- Configuration specific to the observability type
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Usage Analytics - Track which patterns are most valuable
CREATE TABLE pattern_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    user_agent VARCHAR(255), -- Which scenario/agent accessed this
    access_type VARCHAR(20) NOT NULL, -- search, retrieve, generate
    success BOOLEAN NOT NULL,
    metadata JSONB, -- Additional context about usage
    accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for fast searching and retrieval
CREATE INDEX idx_patterns_chapter ON patterns(chapter);
CREATE INDEX idx_patterns_section ON patterns(section);
CREATE INDEX idx_patterns_maturity ON patterns(maturity_level);
CREATE INDEX idx_patterns_tags ON patterns USING GIN(tags);
CREATE INDEX idx_patterns_search ON patterns USING GIN((title || ' ' || what_and_why || ' ' || when_to_use) gin_trgm_ops);

CREATE INDEX idx_recipes_pattern ON recipes(pattern_id);
CREATE INDEX idx_recipes_type ON recipes(type);
CREATE INDEX idx_implementations_recipe ON implementations(recipe_id);
CREATE INDEX idx_implementations_language ON implementations(language);

CREATE INDEX idx_ci_gates_pattern ON ci_gates(pattern_id);
CREATE INDEX idx_ci_gates_type ON ci_gates(gate_type);
CREATE INDEX idx_observability_pattern ON observability(pattern_id);
CREATE INDEX idx_usage_pattern ON pattern_usage(pattern_id);
CREATE INDEX idx_usage_accessed_at ON pattern_usage(accessed_at);

-- Create views for common queries
CREATE VIEW pattern_summary AS
SELECT 
    p.id,
    p.title,
    p.chapter,
    p.section,
    p.maturity_level,
    p.tags,
    COUNT(r.id) as recipe_count,
    COUNT(i.id) as implementation_count,
    ARRAY_AGG(DISTINCT i.language) FILTER (WHERE i.language IS NOT NULL) as languages,
    p.created_at,
    p.updated_at
FROM patterns p
LEFT JOIN recipes r ON p.id = r.pattern_id
LEFT JOIN implementations i ON r.id = i.recipe_id
GROUP BY p.id, p.title, p.chapter, p.section, p.maturity_level, p.tags, p.created_at, p.updated_at;

CREATE VIEW popular_patterns AS
SELECT 
    p.id,
    p.title,
    p.chapter,
    p.maturity_level,
    COUNT(pu.id) as access_count,
    COUNT(pu.id) FILTER (WHERE pu.success = true) as success_count,
    MAX(pu.accessed_at) as last_accessed
FROM patterns p
LEFT JOIN pattern_usage pu ON p.id = pu.pattern_id
WHERE pu.accessed_at > CURRENT_DATE - INTERVAL '30 days'
GROUP BY p.id, p.title, p.chapter, p.maturity_level
ORDER BY access_count DESC;

-- Trigger to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_patterns_updated_at BEFORE UPDATE ON patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recipes_updated_at BEFORE UPDATE ON recipes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_implementations_updated_at BEFORE UPDATE ON implementations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();