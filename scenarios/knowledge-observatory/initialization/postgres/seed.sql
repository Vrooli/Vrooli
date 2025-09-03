-- Knowledge Observatory Seed Data
-- Initial configuration and sample metrics

-- Insert initial collection statistics
INSERT INTO knowledge_observatory.collection_stats (collection_name, total_entries, total_searches, avg_search_score)
VALUES 
    ('vrooli_knowledge', 0, 0, 0.0),
    ('scenario_memory', 0, 0, 0.0),
    ('agent_decisions', 0, 0, 0.0),
    ('workflow_patterns', 0, 0, 0.0),
    ('prompt_templates', 0, 0, 0.0)
ON CONFLICT (collection_name) DO NOTHING;

-- Insert initial quality metrics
INSERT INTO knowledge_observatory.quality_metrics (
    collection_name, 
    coherence_score, 
    freshness_score, 
    redundancy_score, 
    coverage_score,
    total_entries
)
VALUES 
    ('vrooli_knowledge', 0.85, 0.90, 0.88, 0.75, 0),
    ('scenario_memory', 0.82, 0.85, 0.90, 0.70, 0),
    ('agent_decisions', 0.88, 0.75, 0.85, 0.80, 0),
    ('workflow_patterns', 0.90, 0.88, 0.92, 0.85, 0),
    ('prompt_templates', 0.87, 0.92, 0.89, 0.78, 0);

-- Insert sample alerts for monitoring
INSERT INTO knowledge_observatory.alerts (level, collection_name, metric_name, threshold_value, actual_value, message)
VALUES 
    ('info', 'vrooli_knowledge', 'coherence', 0.80, 0.85, 'Coherence score is healthy'),
    ('warning', 'scenario_memory', 'freshness', 0.85, 0.75, 'Freshness score below recommended threshold'),
    ('info', 'agent_decisions', 'redundancy', 0.80, 0.85, 'Redundancy within acceptable range');

-- Insert default user preferences
INSERT INTO knowledge_observatory.user_preferences (
    user_id,
    default_collection,
    saved_queries,
    dashboard_layout,
    alert_preferences
)
VALUES (
    'default',
    'vrooli_knowledge',
    '["scenario workflow", "agent decision", "resource management", "error handling"]'::jsonb,
    '{
        "panels": ["health", "search", "graph", "metrics"],
        "refresh_interval": 10000,
        "theme": "dark"
    }'::jsonb,
    '{
        "email_notifications": false,
        "critical_only": false,
        "collections": ["all"]
    }'::jsonb
)
ON CONFLICT (user_id) DO NOTHING;

-- Insert sample knowledge relationships
INSERT INTO knowledge_observatory.knowledge_relationships (source_id, target_id, relationship_type, weight)
VALUES 
    ('scenario-generator', 'workflow-patterns', 'uses', 0.95),
    ('agent-dashboard', 'agent-decisions', 'monitors', 0.88),
    ('prompt-manager', 'prompt-templates', 'manages', 0.92),
    ('research-assistant', 'vrooli_knowledge', 'contributes', 0.85),
    ('scenario-memory', 'workflow-patterns', 'stores', 0.80)
ON CONFLICT DO NOTHING;

-- Update collection stats with realistic values
UPDATE knowledge_observatory.collection_stats 
SET 
    most_searched_terms = '[
        {"term": "scenario", "count": 145},
        {"term": "workflow", "count": 98},
        {"term": "agent", "count": 87},
        {"term": "resource", "count": 76},
        {"term": "knowledge", "count": 65}
    ]'::jsonb,
    growth_rate = 12.5
WHERE collection_name = 'vrooli_knowledge';

UPDATE knowledge_observatory.collection_stats 
SET 
    most_searched_terms = '[
        {"term": "decision", "count": 234},
        {"term": "reasoning", "count": 189},
        {"term": "pattern", "count": 156},
        {"term": "strategy", "count": 123},
        {"term": "analysis", "count": 98}
    ]'::jsonb,
    growth_rate = 8.3
WHERE collection_name = 'agent_decisions';

-- Create a materialized view for quick dashboard loading
CREATE MATERIALIZED VIEW IF NOT EXISTS knowledge_observatory.health_summary AS
SELECT 
    COUNT(DISTINCT collection_name) as total_collections,
    SUM(total_entries) as total_entries,
    AVG(avg_quality) as overall_health,
    MAX(measured_at) as last_update
FROM knowledge_observatory.dashboard_metrics;

-- Refresh the materialized view
REFRESH MATERIALIZED VIEW knowledge_observatory.health_summary;

-- Log initial setup
INSERT INTO knowledge_observatory.search_history (query, collection, result_count, avg_score, response_time_ms)
VALUES 
    ('System initialization', NULL, 0, 0, 0);