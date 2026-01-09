-- Scenario Dependency Analyzer Seed Data
-- Initial configuration and sample data for the dependency analyzer

-- Insert initial configuration data
INSERT INTO scenario_metadata (
    scenario_name, 
    display_name, 
    description, 
    version,
    tags,
    is_active
) VALUES 
    ('scenario-dependency-analyzer', 
     'Scenario Dependency Analyzer', 
     'Meta-intelligence capability for analyzing, visualizing, and optimizing scenario and resource dependencies within Vrooli',
     '1.0.0',
     ARRAY['meta-intelligence', 'dependency-analysis', 'deployment-optimization'],
     true
    )
ON CONFLICT (scenario_name) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    version = EXCLUDED.version,
    tags = EXCLUDED.tags,
    updated_at = NOW();

-- Insert self-dependencies for the dependency analyzer scenario
INSERT INTO scenario_dependencies (
    scenario_name,
    dependency_type,
    dependency_name,
    required,
    purpose,
    access_method,
    configuration
) VALUES
    ('scenario-dependency-analyzer', 'resource', 'postgres', true, 
     'Store dependency metadata, analysis results, and optimization recommendations',
     'resource-postgres',
     '{"initialization": ["schema.sql", "seed.sql"]}'::jsonb),
    
    ('scenario-dependency-analyzer', 'resource', 'claude-code', true,
     'AI-powered analysis of scenario code, configurations, and proposed scenario descriptions',
     'resource-claude-code',
     '{"commands": ["analyze", "review"]}'::jsonb),

    ('scenario-dependency-analyzer', 'resource', 'ollama', true,
     'Generate embeddings for semantic similarity matching against Qdrant',
     'resource-ollama',
     '{"endpoints": ["${OLLAMA_URL}/api/embeddings"]}'::jsonb),

    ('scenario-dependency-analyzer', 'resource', 'qdrant', true,
     'Semantic similarity matching for proposed scenarios and dependency pattern recognition',
     'resource-qdrant',
     '{"collections": ["scenario_embeddings", "dependency_patterns"]}'::jsonb),
    
    ('scenario-dependency-analyzer', 'resource', 'redis', false,
     'Cache frequently accessed dependency data and analysis results for performance optimization',
     'resource-redis',
     '{"cache_ttl": 3600, "max_memory": "256mb"}'::jsonb)
ON CONFLICT DO NOTHING;

-- Insert initial analysis run record
INSERT INTO analysis_runs (
    run_type,
    status,
    results_summary,
    triggered_by,
    started_at,
    completed_at
) VALUES (
    'full_system',
    'completed',
    '{
        "scenarios_analyzed": 0,
        "dependencies_found": 4,
        "resources_identified": ["postgres", "claude-code", "qdrant", "redis"],
        "recommendations_generated": 0
    }'::jsonb,
    'initial_setup',
    NOW(),
    NOW()
);

-- Create some example optimization recommendation templates (for demonstration)
-- These will be replaced with real recommendations once analysis runs
INSERT INTO optimization_recommendations (
    scenario_name,
    recommendation_type,
    title,
    description,
    current_state,
    recommended_state,
    estimated_impact,
    confidence_score,
    priority,
    status
) VALUES
    ('scenario-dependency-analyzer',
     'shared_workflow_adoption',
     'Example: Adopt shared ollama workflow',
     'Example recommendation for using shared workflow instead of direct resource access',
     '{"direct_resource": "ollama", "access_pattern": "api_calls"}'::jsonb,
     '{"shared_workflow": "initialization/n8n/ollama.json", "access_pattern": "workflow_trigger"}'::jsonb,
     '{"reliability_improvement": "high", "maintenance_reduction": "medium", "resource_efficiency": "low"}'::jsonb,
     0.85,
     'medium',
     'pending'
    )
ON CONFLICT DO NOTHING;

-- Insert a sample dependency graph (empty but with structure)
INSERT INTO dependency_graphs (
    graph_type,
    title,
    description,
    nodes,
    edges,
    metadata
) VALUES
    ('combined',
     'Initial System Dependencies',
     'Initial dependency graph showing basic system structure',
     '[]'::jsonb,
     '[]'::jsonb,
     '{
        "total_nodes": 0,
        "total_edges": 0,
        "complexity_score": 0.0,
        "generated_by": "seed_data",
        "version": "1.0.0"
     }'::jsonb
    );
