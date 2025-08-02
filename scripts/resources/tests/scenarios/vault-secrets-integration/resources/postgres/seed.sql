-- Seed data for Vault Secrets Integration testing
-- This creates sample audit records for testing the integration

-- Insert sample access logs for different scenarios
INSERT INTO vault_audit.secret_access_log (
    user_id, service_name, workflow_id, vault_path, success, duration_ms, 
    response_code, data_classification, compliance_tags
) VALUES 
    -- Successful API key retrieval
    ('test-user-001', 'n8n', 'workflow-123', 'secret/data/api/keys/service-a', 
     true, 45, 200, 'CONFIDENTIAL', ARRAY['api-access', 'production']),
    
    -- Successful database credential retrieval
    ('test-user-001', 'n8n', 'workflow-124', 'secret/data/database/postgres/prod', 
     true, 67, 200, 'RESTRICTED', ARRAY['database', 'production']),
    
    -- Failed access attempt (unauthorized)
    ('test-user-002', 'n8n', 'workflow-125', 'secret/data/api/keys/admin', 
     false, 23, 403, 'RESTRICTED', ARRAY['api-access', 'admin']),
    
    -- Successful third-party integration secret
    ('test-user-003', 'n8n', 'workflow-126', 'secret/data/integrations/stripe', 
     true, 89, 200, 'CONFIDENTIAL', ARRAY['payment', 'integration']),
    
    -- Performance test entries
    ('perf-test-user', 'n8n', 'perf-workflow-001', 'secret/data/test/performance', 
     true, 12, 200, 'PUBLIC', ARRAY['test', 'performance']);

-- Insert historical data for trend analysis
INSERT INTO vault_audit.secret_access_log (
    timestamp, user_id, service_name, workflow_id, vault_path, 
    success, duration_ms, response_code
)
SELECT 
    NOW() - INTERVAL '1 hour' * s.hour_offset,
    'analytics-user-' || (s.hour_offset % 3 + 1),
    'n8n',
    'analytics-workflow-' || (s.hour_offset % 5 + 1),
    'secret/data/analytics/key-' || (s.hour_offset % 10 + 1),
    s.hour_offset % 10 != 0,  -- 90% success rate
    20 + (s.hour_offset % 50), -- Variable duration
    CASE WHEN s.hour_offset % 10 != 0 THEN 200 ELSE 404 END
FROM generate_series(1, 168) AS s(hour_offset); -- Last 7 days of data

-- Insert sample rotation logs
INSERT INTO vault_audit.secret_rotation_log (
    secret_path, secret_type, old_version, new_version, 
    rotation_reason, initiated_by, status, completed_at
) VALUES 
    ('secret/data/api/keys/service-a', 'api_key', 1, 2, 
     'Scheduled rotation', 'rotation-service', 'completed', NOW() - INTERVAL '7 days'),
    
    ('secret/data/database/postgres/prod', 'database_password', 3, 4, 
     'Security policy', 'security-team', 'completed', NOW() - INTERVAL '3 days'),
    
    ('secret/data/integrations/stripe', 'api_secret', 2, 3, 
     'Vendor requirement', 'ops-team', 'completed', NOW() - INTERVAL '1 day'),
    
    ('secret/data/api/keys/service-b', 'api_key', 1, NULL, 
     'Scheduled rotation', 'rotation-service', 'pending', NULL);

-- Create test workflow configurations
CREATE TABLE IF NOT EXISTS vault_audit.test_workflows (
    workflow_id VARCHAR(255) PRIMARY KEY,
    workflow_name VARCHAR(255) NOT NULL,
    description TEXT,
    vault_paths TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

INSERT INTO vault_audit.test_workflows (workflow_id, workflow_name, description, vault_paths) VALUES
    ('workflow-123', 'API Integration Workflow', 'Handles external API calls with secured credentials', 
     ARRAY['secret/data/api/keys/service-a', 'secret/data/api/keys/service-b']),
    
    ('workflow-124', 'Database Sync Workflow', 'Synchronizes data between databases', 
     ARRAY['secret/data/database/postgres/prod', 'secret/data/database/postgres/staging']),
    
    ('workflow-126', 'Payment Processing Workflow', 'Processes payments through Stripe', 
     ARRAY['secret/data/integrations/stripe', 'secret/data/api/keys/payment-gateway']);

-- Verify the data
SELECT 'Access logs created:' as info, COUNT(*) as count FROM vault_audit.secret_access_log
UNION ALL
SELECT 'Rotation logs created:', COUNT(*) FROM vault_audit.secret_rotation_log
UNION ALL
SELECT 'Test workflows created:', COUNT(*) FROM vault_audit.test_workflows;

-- Sample query to show recent activity
SELECT 
    'Recent vault access activity:' as report,
    COUNT(*) as total_accesses,
    SUM(CASE WHEN success THEN 1 ELSE 0 END) as successful,
    SUM(CASE WHEN NOT success THEN 1 ELSE 0 END) as failed,
    AVG(duration_ms)::INTEGER as avg_duration_ms
FROM vault_audit.secret_access_log
WHERE timestamp > NOW() - INTERVAL '24 hours';