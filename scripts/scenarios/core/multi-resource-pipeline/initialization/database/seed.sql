-- Seed data template for scenario applications
-- This populates the database with initial data for testing and demonstration

-- Insert application metadata
INSERT INTO multi-resource-pipeline.app_metadata (
    app_name, version, configuration
) VALUES (
    'Multi-Resource Data Pipeline',
    '1.0.0',
    '{
        "complexity": "{{ complexity }}",
        "categories": {{ categories | tojson }},
        "required_resources": {{ resources.required | tojson }},
        "optional_resources": {{ resources.optional | tojson | default("[]") }},
        "business_model": {
            "value_proposition": "{{ business.value_proposition }}",
            "target_markets": {{ business.target_markets | tojson }},
            "revenue_potential": {{ business.revenue_potential | tojson }}
        }
    }'::jsonb
);

-- Insert default configuration values
INSERT INTO multi-resource-pipeline.app_config (config_key, config_value, config_type, description) VALUES 
    -- Feature flags
    ('feature_ai_processing', 'true'::jsonb, 'feature_flag', 'Enable AI processing capabilities'),
    ('feature_analytics_dashboard', '{{ "true" if "windmill" in resources.required else "false" }}'::jsonb, 'feature_flag', 'Enable analytics dashboard UI'),
    ('feature_workflow_automation', '{{ "true" if "n8n" in resources.required else "false" }}'::jsonb, 'feature_flag', 'Enable workflow automation'),
    ('feature_file_storage', '{{ "true" if "minio" in resources.required else "false" }}'::jsonb, 'feature_flag', 'Enable file storage capabilities'),
    ('feature_vector_search', '{{ "true" if "qdrant" in resources.required else "false" }}'::jsonb, 'feature_flag', 'Enable vector search functionality'),
    
    -- Application settings
    ('app_name', '"Multi-Resource Data Pipeline"'::jsonb, 'setting', 'Application display name'),
    ('app_description', '"{{ scenario.description }}"'::jsonb, 'setting', 'Application description'),
    ('max_concurrent_users', '{{ performance.throughput.concurrent_users | default(10) }}'::jsonb, 'setting', 'Maximum concurrent users'),
    ('session_timeout_minutes', '30'::jsonb, 'setting', 'User session timeout in minutes'),
    ('enable_usage_analytics', 'true'::jsonb, 'setting', 'Track usage analytics'),
    
    -- Resource configuration
    ('resource_timeout_seconds', '{{ testing.timeout_seconds | default(300) }}'::jsonb, 'resource_config', 'Default timeout for resource operations'),
    ('ai_model_temperature', '0.7'::jsonb, 'resource_config', 'AI model temperature for text generation'),
    ('max_file_size_mb', '10'::jsonb, 'resource_config', 'Maximum file upload size in MB'),
    ('cache_ttl_seconds', '3600'::jsonb, 'resource_config', 'Cache time-to-live in seconds'),
    
    -- UI customization
    ('ui_theme', '"default"'::jsonb, 'setting', 'UI theme selection'),
    ('ui_primary_color', '"#3b82f6"'::jsonb, 'setting', 'Primary UI color'),
    ('ui_brand_name', '"Multi-Resource Data Pipeline"'::jsonb, 'setting', 'Brand name displayed in UI'),
    ('ui_show_advanced_features', 'false'::jsonb, 'setting', 'Show advanced features in UI');

-- Insert sample user sessions for demonstration
INSERT INTO multi-resource-pipeline.user_sessions (
    session_id, user_identifier, context_data, preferences
) VALUES 
    ('demo-session-001', 'demo-user', 
     '{"source": "demo", "use_case": "{{ business.target_markets[0] if business.target_markets else "general" }}"}'::jsonb,
     '{"notifications": true, "theme": "light", "language": "en"}'::jsonb),
     
    ('demo-session-002', 'test-user',
     '{"source": "testing", "environment": "development"}'::jsonb,
     '{"notifications": false, "theme": "dark", "language": "en"}'::jsonb);

-- Insert sample activity log entries
INSERT INTO multi-resource-pipeline.activity_log (
    event_type, event_data, session_id, user_identifier, processing_time_ms, status
) VALUES 
    ('app_initialized', 
     '{"version": "1.0.0", "resources": {{ resources.required | tojson }}}'::jsonb,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     'demo-user', 150, 'completed'),
     
    ('configuration_loaded',
     '{"config_count": ' || (SELECT COUNT(*) FROM multi-resource-pipeline.app_config) || '}'::jsonb,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     'demo-user', 25, 'completed');

-- Insert sample resource metrics for demonstration
INSERT INTO multi-resource-pipeline.resource_metrics (
    resource_type, resource_name, operation, duration_ms, success, 
    tokens_used, estimated_cost_usd, session_id, request_metadata
) VALUES 
    {% if "ollama" in resources.required %}
    ('ai', 'ollama', 'text_generation', 1200, true, 150, 0.001,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     '{"model": "llama3.1:8b", "prompt_length": 50}'::jsonb),
    {% endif %}
    
    {% if "whisper" in resources.required %}
    ('ai', 'whisper', 'transcription', 800, true, null, 0.002,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     '{"audio_duration_seconds": 30, "language": "en"}'::jsonb),
    {% endif %}
    
    {% if "n8n" in resources.required %}
    ('automation', 'n8n', 'workflow_execution', 500, true, null, 0.0001,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     '{"workflow_id": "demo-workflow", "steps_executed": 5}'::jsonb),
    {% endif %}
    
    {% if "postgres" in resources.required %}
    ('storage', 'postgres', 'query_execution', 45, true, null, 0.00001,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     '{"query_type": "SELECT", "rows_returned": 10}'::jsonb);
    {% endif %}

-- Create sample data specific to scenario category
{% if "ai-assistance" in categories %}
-- AI Assistant specific sample data
INSERT INTO multi-resource-pipeline.activity_log (event_type, event_data, session_id, user_identifier, status) VALUES
    ('ai_query_processed', 
     '{"query": "Sample AI assistance request", "response_type": "text", "satisfaction": "high"}'::jsonb,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     'demo-user', 'completed');
{% endif %}

{% if "content-creation" in categories %}
-- Content Creation specific sample data
INSERT INTO multi-resource-pipeline.activity_log (event_type, event_data, session_id, user_identifier, status) VALUES
    ('content_generated',
     '{"content_type": "image", "generation_time_ms": 5000, "quality_score": 0.92}'::jsonb,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     'demo-user', 'completed');
{% endif %}

{% if "business-automation" in categories %}
-- Business Automation specific sample data
INSERT INTO multi-resource-pipeline.activity_log (event_type, event_data, session_id, user_identifier, status) VALUES
    ('workflow_automated',
     '{"process_name": "document_processing", "items_processed": 25, "time_saved_minutes": 120}'::jsonb,
     (SELECT id FROM multi-resource-pipeline.user_sessions WHERE session_id = 'demo-session-001'),
     'demo-user', 'completed');
{% endif %}

-- Verify the seeded data
SELECT 'Application metadata:' as info, COUNT(*) as count FROM multi-resource-pipeline.app_metadata
UNION ALL
SELECT 'Configuration entries:', COUNT(*) FROM multi-resource-pipeline.app_config
UNION ALL
SELECT 'Sample sessions:', COUNT(*) FROM multi-resource-pipeline.user_sessions
UNION ALL
SELECT 'Activity log entries:', COUNT(*) FROM multi-resource-pipeline.activity_log
UNION ALL
SELECT 'Resource metrics:', COUNT(*) FROM multi-resource-pipeline.resource_metrics;