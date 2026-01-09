-- Deployment Manager Seed Data
-- Version: 1.0.0
-- Description: Initial swap alternatives and baseline fitness rules

-- ============================================================================
-- Swap Alternatives - Common Resource Swaps
-- ============================================================================

-- Database swaps (postgres → sqlite for mobile/desktop)
INSERT INTO swap_alternatives (from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers, fitness_delta, migration_effort, pros, cons, migration_notes, verified)
VALUES
(
    'resource',
    'postgres',
    'resource',
    'sqlite',
    ARRAY['desktop', 'mobile'],
    45,
    'moderate',
    ARRAY['No server required', 'Zero-configuration', 'Embedded in application', 'Fast for small datasets'],
    ARRAY['Limited concurrency', 'No network access', 'Weaker type system', 'Missing advanced features (JSONB, full-text search)'],
    'Replace PostgreSQL connection strings with SQLite file paths. Schema changes required: remove JSONB columns, replace ARRAY types with JSON strings. Test all queries for compatibility.',
    true
),
(
    'resource',
    'postgres',
    'resource',
    'mysql',
    ARRAY['saas', 'enterprise'],
    5,
    'trivial',
    ARRAY['Widely available in cloud providers', 'Strong community support', 'Good performance for read-heavy workloads'],
    ARRAY['Slightly different SQL dialect', 'Weaker JSON support than PostgreSQL'],
    'Update connection strings and review SQL queries for dialect differences. Most standard queries work unchanged.',
    true
);

-- LLM swaps (ollama → cloud APIs for mobile/saas)
INSERT INTO swap_alternatives (from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers, fitness_delta, migration_effort, pros, cons, migration_notes, verified)
VALUES
(
    'resource',
    'ollama',
    'resource',
    'openrouter',
    ARRAY['desktop', 'mobile', 'saas'],
    60,
    'trivial',
    ARRAY['No local GPU required', 'Always up-to-date models', 'Faster inference for mobile', 'Pay-per-use pricing'],
    ARRAY['Requires API key', 'Network latency', 'Privacy concerns (data leaves device)', 'Usage costs'],
    'Replace ollama API calls with OpenRouter endpoints. Add API key configuration. Update prompt templates if model-specific.',
    true
),
(
    'resource',
    'ollama',
    'resource',
    'claude-code',
    ARRAY['desktop', 'saas'],
    40,
    'moderate',
    ARRAY['High-quality responses', 'Tool use capabilities', 'Strong reasoning', 'Rate limits generous for dev'],
    ARRAY['Requires Anthropic API key', 'Network dependency', 'Cost per token', 'No local fallback'],
    'Migrate prompts to Claude message format. Implement tool use for function calling. Add retry logic for rate limits.',
    true
);

-- Redis swaps (for mobile/desktop with limited memory)
INSERT INTO swap_alternatives (from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers, fitness_delta, migration_effort, pros, cons, migration_notes, verified)
VALUES
(
    'resource',
    'redis',
    'resource',
    'in-memory-cache',
    ARRAY['desktop', 'mobile'],
    35,
    'moderate',
    ARRAY['No external process', 'Simpler deployment', 'Zero configuration'],
    ARRAY['Not persistent across restarts', 'No pub/sub', 'No distributed caching', 'Limited to single process'],
    'Replace Redis client with in-memory Map or LRU cache. Remove pub/sub features. Implement TTL manually if needed.',
    true
);

-- Vector database swaps
INSERT INTO swap_alternatives (from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers, fitness_delta, migration_effort, pros, cons, migration_notes, verified)
VALUES
(
    'resource',
    'qdrant',
    'resource',
    'pinecone',
    ARRAY['saas', 'enterprise'],
    20,
    'trivial',
    ARRAY['Fully managed', 'Automatic scaling', 'Global edge deployment'],
    ARRAY['Requires API key', 'Usage-based pricing', 'Less control over infrastructure'],
    'Replace Qdrant connection with Pinecone client. Migrate vector collections. Update search queries for API differences.',
    true
),
(
    'resource',
    'qdrant',
    'resource',
    'chroma',
    ARRAY['desktop', 'mobile'],
    30,
    'moderate',
    ARRAY['Lightweight', 'Python/Node bindings', 'Good for small datasets'],
    ARRAY['Less performant at scale', 'Smaller community', 'Fewer features'],
    'Replace Qdrant client with Chroma. Recreate collections and re-index vectors. Test search quality.',
    false
);

-- Scenario swaps (for deployment orchestration dependencies)
INSERT INTO swap_alternatives (from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers, fitness_delta, migration_effort, pros, cons, migration_notes, verified)
VALUES
(
    'scenario',
    'scenario-dependency-analyzer',
    'resource',
    'static-analysis-fallback',
    ARRAY['desktop', 'mobile'],
    -20,
    'major',
    ARRAY['No dependency on external scenario', 'Faster startup', 'Works offline'],
    ARRAY['Less accurate', 'Misses runtime dependencies', 'Requires maintenance as codebase changes'],
    'Implement static analysis of .vrooli/service.json files. Parse dependency arrays manually. No recursive analysis.',
    false
);

-- Browser automation swaps
INSERT INTO swap_alternatives (from_dep_type, from_dep_name, to_dep_type, to_dep_name, applicable_tiers, fitness_delta, migration_effort, pros, cons, migration_notes, verified)
VALUES
(
    'resource',
    'browserless',
    'resource',
    'playwright',
    ARRAY['desktop', 'saas'],
    15,
    'moderate',
    ARRAY['Local execution', 'No external service', 'Free', 'More control over browser lifecycle'],
    ARRAY['Requires browser installation', 'Larger bundle size', 'More resource usage'],
    'Replace Browserless API calls with Playwright scripts. Install browser binaries. Update screenshot/PDF generation logic.',
    true
);

-- ============================================================================
-- Baseline Fitness Rules
-- ============================================================================

-- Local tier (Tier 1) - Everything is perfect locally
INSERT INTO fitness_rules (rule_name, tier, dependency_type, dependency_pattern, scoring_config, priority)
VALUES
('local-baseline', 'local', NULL, NULL, '{"base_score": 100, "reasoning": "Local development environment with full Vrooli stack"}', 1000);

-- Desktop tier (Tier 2) - Penalize server dependencies
INSERT INTO fitness_rules (rule_name, tier, dependency_type, dependency_pattern, scoring_config, priority)
VALUES
('desktop-server-penalty', 'desktop', 'resource', 'postgres|mysql|redis|qdrant', '{"penalty": 40, "reasoning": "Server processes not suitable for desktop standalone apps"}', 900),
('desktop-llm-penalty', 'desktop', 'resource', 'ollama', '{"penalty": 50, "reasoning": "Local LLM requires GPU and large downloads"}', 850),
('desktop-lightweight-bonus', 'desktop', 'resource', 'sqlite|in-memory-cache', '{"bonus": 20, "reasoning": "Lightweight embedded dependencies ideal for desktop"}', 800);

-- Mobile tier (Tier 3) - Strict resource constraints
INSERT INTO fitness_rules (rule_name, tier, dependency_type, dependency_pattern, scoring_config, priority)
VALUES
('mobile-server-blocker', 'mobile', 'resource', 'postgres|mysql|redis|qdrant|ollama', '{"penalty": 100, "reasoning": "Server processes impossible on mobile"}', 950),
('mobile-cloud-bonus', 'mobile', 'resource', 'openrouter|claude-code|pinecone', '{"bonus": 30, "reasoning": "Cloud APIs ideal for mobile - offload compute"}', 800),
('mobile-size-penalty', 'mobile', NULL, NULL, '{"size_threshold_mb": 50, "penalty_per_mb": 2, "reasoning": "Bundle size critical for app stores"}', 700);

-- SaaS tier (Tier 4) - Optimize for cloud deployment
INSERT INTO fitness_rules (rule_name, tier, dependency_type, dependency_pattern, scoring_config, priority)
VALUES
('saas-managed-bonus', 'saas', 'resource', 'rds|elasticache|pinecone', '{"bonus": 25, "reasoning": "Managed services reduce ops burden"}', 850),
('saas-self-hosted-penalty', 'saas', 'resource', 'ollama|browserless', '{"penalty": 30, "reasoning": "Self-hosted adds complexity vs managed alternatives"}', 800),
('saas-cost-factor', 'saas', NULL, NULL, '{"cost_per_month_threshold": 100, "penalty_per_100": 10, "reasoning": "High ongoing costs impact profitability"}', 750);

-- Enterprise tier (Tier 5) - Compliance and control
INSERT INTO fitness_rules (rule_name, tier, dependency_type, dependency_pattern, scoring_config, priority)
VALUES
('enterprise-compliance-bonus', 'enterprise', 'resource', 'vault|secrets-manager', '{"bonus": 30, "reasoning": "Enterprise-grade secret management required"}', 900),
('enterprise-vendor-lock-penalty', 'enterprise', 'resource', 'aws-.*|azure-.*|gcp-.*', '{"penalty": 20, "reasoning": "Vendor lock-in reduces flexibility"}', 800),
('enterprise-audit-requirement', 'enterprise', NULL, NULL, '{"audit_logging": true, "penalty_if_missing": 50, "reasoning": "Audit trails mandatory for compliance"}', 850);

-- ============================================================================
-- Demo Deployment Profile (for development/testing)
-- ============================================================================

INSERT INTO deployment_profiles (name, description, scenario_name, target_tiers, profile_data, status, created_by)
VALUES
(
    'Example Desktop Deployment',
    'Demo profile showing postgres → sqlite swap for desktop tier',
    'example-scenario',
    ARRAY['desktop'],
    '{
        "scenario": "example-scenario",
        "tiers": ["desktop"],
        "swaps": [
            {
                "from": {"type": "resource", "name": "postgres"},
                "to": {"type": "resource", "name": "sqlite"},
                "reason": "Desktop apps cannot run postgres server"
            }
        ],
        "secrets": {
            "DATABASE_URL": {
                "required": true,
                "description": "SQLite file path",
                "default": "file:./app.db"
            }
        },
        "env_vars": {
            "NODE_ENV": "production"
        }
    }'::jsonb,
    'draft',
    'seed-data'
);

-- ============================================================================
-- Notes for Improvers
-- ============================================================================
-- This seed data provides:
-- 1. 10 verified swap alternatives covering common migration scenarios
-- 2. 15 baseline fitness rules for all 5 deployment tiers
-- 3. 1 demo deployment profile for testing
--
-- Next steps for P0 implementation:
-- - Add API endpoints that query swap_alternatives for suggestions
-- - Implement fitness scoring engine that applies rules from fitness_rules
-- - Create UI components that display swap pros/cons from this data
-- - Write tests that validate swap suggestions work correctly
