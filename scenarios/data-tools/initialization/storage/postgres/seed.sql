-- SCENARIO_NAME_PLACEHOLDER Seed Data
-- Initial data for application startup

-- Insert default configuration
INSERT INTO configuration (key, value, category, description, is_active) VALUES
    ('app.name', '"SCENARIO_NAME_PLACEHOLDER"', 'application', 'Application name', true),
    ('app.version', '"1.0.0"', 'application', 'Application version', true),
    ('app.environment', '"development"', 'application', 'Environment mode', true),
    ('api.rate_limit', '{"requests_per_minute": 60, "burst": 10}', 'api', 'API rate limiting configuration', true),
    ('api.timeout', '{"default": 30000, "long_running": 300000}', 'api', 'API timeout settings in milliseconds', true),
    ('features.authentication', '{"enabled": true, "provider": "internal"}', 'features', 'Authentication configuration', true),
    ('features.notifications', '{"enabled": true, "channels": ["email", "webhook"]}', 'features', 'Notification settings', true),
    ('resources.n8n', '{"enabled": true, "base_url": "http://localhost:5678"}', 'resources', 'n8n configuration', true),
    ('resources.ollama', '{"enabled": false, "base_url": "http://localhost:11434"}', 'resources', 'Ollama configuration', true)
ON CONFLICT (key) DO NOTHING;

-- Insert sample workflows (customize based on your scenario)
INSERT INTO workflows (workflow_id, name, description, platform, definition, is_active) VALUES
    ('main-workflow', 'Main Processing Workflow', 'Primary workflow for SCENARIO_NAME_PLACEHOLDER', 'n8n', 
     '{"nodes": [], "connections": {}, "settings": {}}', true),
    ('data-pipeline', 'Data Processing Pipeline', 'Handles data transformation and storage', 'n8n',
     '{"nodes": [], "connections": {}, "settings": {}}', true),
    ('ui-dashboard', 'Dashboard Application', 'Main user interface dashboard', 'n8n',
     '{"type": "app", "components": [], "layout": {}}', true)
ON CONFLICT (workflow_id) DO NOTHING;

-- Insert sample resources (customize based on your scenario)
INSERT INTO resources (name, description, type, status, config) VALUES
    ('Default Workspace', 'Primary workspace for SCENARIO_NAME_PLACEHOLDER', 'workspace', 'active',
     '{"settings": {"theme": "light", "language": "en"}}'),
    ('API Documentation', 'OpenAPI specification and documentation', 'documentation', 'active',
     '{"version": "3.0.0", "format": "openapi"}'),
    ('System Health Monitor', 'Monitors system health and performance', 'monitor', 'active',
     '{"interval": 60000, "endpoints": ["/health", "/metrics"]}')
ON CONFLICT ON CONSTRAINT unique_name_per_type DO NOTHING;

-- Insert initial events for audit trail
INSERT INTO events (event_type, event_name, payload, source, severity) VALUES
    ('system', 'application.initialized', 
     '{"message": "SCENARIO_NAME_PLACEHOLDER initialized successfully", "version": "1.0.0"}',
     'system', 'info'),
    ('configuration', 'config.loaded',
     '{"message": "Configuration loaded from database", "count": 10}',
     'system', 'info'),
    ('workflow', 'workflows.registered',
     '{"message": "Workflows registered", "count": 3}',
     'system', 'info');

-- Insert sample metrics
INSERT INTO metrics (metric_name, metric_type, value, unit, resource_type, dimensions) VALUES
    ('startup.time', 'gauge', 1500, 'milliseconds', 'application', '{"component": "api"}'),
    ('memory.usage', 'gauge', 256, 'megabytes', 'application', '{"component": "api"}'),
    ('database.connections', 'gauge', 5, 'count', 'database', '{"pool": "main"}');

-- Create default admin session (for testing - remove in production)
INSERT INTO sessions (session_token, user_id, data, expires_at) VALUES
    ('default-admin-token-CHANGE-IN-PRODUCTION', 'admin',
     '{"role": "admin", "permissions": ["read", "write", "execute"]}',
     NOW() + INTERVAL '30 days')
ON CONFLICT (session_token) DO NOTHING;

-- Summary message
DO $$
BEGIN
    RAISE NOTICE 'Seed data inserted successfully for SCENARIO_NAME_PLACEHOLDER';
    RAISE NOTICE 'Configuration entries: %', (SELECT COUNT(*) FROM configuration);
    RAISE NOTICE 'Workflows registered: %', (SELECT COUNT(*) FROM workflows);
    RAISE NOTICE 'Resources created: %', (SELECT COUNT(*) FROM resources);
END $$;
