-- Scalable App Cookbook Database Schema
-- Stores architectural patterns, recipes, and implementations

-- Schema initialization
CREATE SCHEMA IF NOT EXISTS scalable_app_cookbook;

-- Ensure all unqualified objects land in the scenario schema first
SET search_path TO scalable_app_cookbook, public;

-- Enable extensions used by text search and indexing
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS btree_gin;

-- Patterns table - Core architectural patterns from the cookbook
CREATE TABLE IF NOT EXISTS patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    chapter VARCHAR(100) NOT NULL,
    section VARCHAR(100) NOT NULL,
    maturity_level VARCHAR(2) NOT NULL CHECK (maturity_level IN ('L0', 'L1', 'L2', 'L3', 'L4')),
    tags TEXT[] DEFAULT '{}',
    what_and_why TEXT NOT NULL,
    when_to_use TEXT NOT NULL,
    tradeoffs TEXT NOT NULL,
    reference_patterns TEXT[] DEFAULT '{}',
    failure_modes TEXT,
    cost_levers TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ensure reference_patterns exists on legacy tables
ALTER TABLE IF EXISTS patterns
    ADD COLUMN IF NOT EXISTS reference_patterns TEXT[] DEFAULT '{}'::text[];
ALTER TABLE IF EXISTS public.patterns
    ADD COLUMN IF NOT EXISTS reference_patterns TEXT[] DEFAULT '{}'::text[];
ALTER TABLE IF EXISTS patterns
    ADD COLUMN IF NOT EXISTS failure_modes TEXT;
ALTER TABLE IF EXISTS public.patterns
    ADD COLUMN IF NOT EXISTS failure_modes TEXT;
ALTER TABLE IF EXISTS patterns
    ADD COLUMN IF NOT EXISTS cost_levers TEXT;
ALTER TABLE IF EXISTS public.patterns
    ADD COLUMN IF NOT EXISTS cost_levers TEXT;
ALTER TABLE IF EXISTS patterns
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE IF EXISTS public.patterns
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP;

-- Recipes table - Step-by-step implementation guides
CREATE TABLE IF NOT EXISTS recipes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('greenfield', 'brownfield', 'migration')),
    prerequisites TEXT[] DEFAULT '{}',
    steps JSONB NOT NULL,
    config_snippets JSONB DEFAULT '{}',
    validation_checks TEXT[] DEFAULT '{}',
    artifacts TEXT[] DEFAULT '{}',
    metrics TEXT[] DEFAULT '{}',
    rollbacks TEXT[] DEFAULT '{}',
    prompts TEXT[] DEFAULT '{}',
    timeout_sec INTEGER DEFAULT 300,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Implementations table - Language-specific code examples
CREATE TABLE IF NOT EXISTS implementations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipe_id UUID NOT NULL REFERENCES recipes(id) ON DELETE CASCADE,
    language VARCHAR(20) NOT NULL CHECK (language IN ('go', 'javascript', 'typescript', 'python', 'java', 'rust', 'csharp')),
    code TEXT NOT NULL,
    file_path VARCHAR(500),
    description TEXT,
    dependencies TEXT[] DEFAULT '{}',
    test_code TEXT,
    benchmark_data JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- CI Gates table - Quality gates and policies for each pattern
CREATE TABLE IF NOT EXISTS ci_gates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    gate_type VARCHAR(50) NOT NULL,
    gate_name VARCHAR(100) NOT NULL,
    gate_config JSONB NOT NULL,
    enforcement_level VARCHAR(20) DEFAULT 'warning' CHECK (enforcement_level IN ('error', 'warning', 'info')),
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Observability table - SLIs, dashboards, alerts for patterns
CREATE TABLE IF NOT EXISTS observability (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    observability_type VARCHAR(30) NOT NULL,
    name VARCHAR(100) NOT NULL,
    config JSONB NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Usage Analytics - Track which patterns are most valuable
CREATE TABLE IF NOT EXISTS pattern_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id UUID NOT NULL REFERENCES patterns(id) ON DELETE CASCADE,
    user_agent VARCHAR(255),
    access_type VARCHAR(20) NOT NULL,
    success BOOLEAN NOT NULL,
    metadata JSONB,
    accessed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for fast searching and retrieval
CREATE INDEX IF NOT EXISTS idx_patterns_chapter ON patterns(chapter);
CREATE INDEX IF NOT EXISTS idx_patterns_section ON patterns(section);
CREATE INDEX IF NOT EXISTS idx_patterns_maturity ON patterns(maturity_level);
CREATE INDEX IF NOT EXISTS idx_patterns_tags ON patterns USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_patterns_search ON patterns USING GIN((title || ' ' || what_and_why || ' ' || when_to_use) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_recipes_pattern ON recipes(pattern_id);
CREATE INDEX IF NOT EXISTS idx_recipes_type ON recipes(type);
CREATE INDEX IF NOT EXISTS idx_implementations_recipe ON implementations(recipe_id);
CREATE INDEX IF NOT EXISTS idx_implementations_language ON implementations(language);

CREATE INDEX IF NOT EXISTS idx_ci_gates_pattern ON ci_gates(pattern_id);
CREATE INDEX IF NOT EXISTS idx_ci_gates_type ON ci_gates(gate_type);
CREATE INDEX IF NOT EXISTS idx_observability_pattern ON observability(pattern_id);
CREATE INDEX IF NOT EXISTS idx_usage_pattern ON pattern_usage(pattern_id);
CREATE INDEX IF NOT EXISTS idx_usage_accessed_at ON pattern_usage(accessed_at);

-- Create views for common queries
CREATE OR REPLACE VIEW pattern_summary AS
SELECT 
    p.id,
    p.title,
    p.chapter,
    p.section,
    p.maturity_level,
    p.tags,
    COUNT(r.id) AS recipe_count,
    COUNT(i.id) AS implementation_count,
    ARRAY_AGG(DISTINCT i.language) FILTER (WHERE i.language IS NOT NULL) AS languages,
    p.created_at,
    p.updated_at
FROM patterns p
LEFT JOIN recipes r ON p.id = r.pattern_id
LEFT JOIN implementations i ON r.id = i.recipe_id
GROUP BY p.id, p.title, p.chapter, p.section, p.maturity_level, p.tags, p.created_at, p.updated_at;

CREATE OR REPLACE VIEW popular_patterns AS
SELECT 
    p.id,
    p.title,
    p.chapter,
    p.maturity_level,
    COUNT(pu.id) AS access_count,
    COUNT(pu.id) FILTER (WHERE pu.success = true) AS success_count,
    MAX(pu.accessed_at) AS last_accessed
FROM patterns p
LEFT JOIN pattern_usage pu ON p.id = pu.pattern_id
WHERE pu.accessed_at > CURRENT_DATE - INTERVAL '30 days'
GROUP BY p.id, p.title, p.chapter, p.maturity_level
ORDER BY access_count DESC;

-- Deduplicate patterns before enforcing uniqueness on titles
WITH pattern_counts AS (
    SELECT 
        p.id,
        p.title,
        p.created_at,
        COUNT(r.id) AS recipe_count
    FROM patterns p
    LEFT JOIN recipes r ON r.pattern_id = p.id
    GROUP BY p.id, p.title, p.created_at
), ranked_patterns AS (
    SELECT 
        id,
        ROW_NUMBER() OVER (
            PARTITION BY title
            ORDER BY recipe_count DESC, created_at ASC, id ASC
        ) AS row_rank
    FROM pattern_counts
)
DELETE FROM patterns p
USING ranked_patterns rp
WHERE p.id = rp.id AND rp.row_rank > 1;

-- Ensure pattern titles stay unique to avoid duplicate search results
CREATE UNIQUE INDEX IF NOT EXISTS idx_patterns_title_unique ON patterns (title);

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
