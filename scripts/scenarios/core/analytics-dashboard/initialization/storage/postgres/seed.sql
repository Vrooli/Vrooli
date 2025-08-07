-- Resource Monitoring Platform Seed Data
-- Initial configuration and resource action definitions

SET search_path TO resource_monitoring;

-- ============================================
-- Default Resource Actions for Each Type
-- ============================================

-- AI Resource Actions
INSERT INTO resource_actions (resource_type, action_name, action_command, description, requires_confirmation, is_dangerous, icon, button_color) VALUES
    ('ai', 'restart', 'systemctl restart {{resource_name}}', 'Restart AI service', true, false, 'refresh', 'warning'),
    ('ai', 'stop', 'systemctl stop {{resource_name}}', 'Stop AI service', true, true, 'stop', 'danger'),
    ('ai', 'start', 'systemctl start {{resource_name}}', 'Start AI service', false, false, 'play', 'success'),
    ('ai', 'reload-models', 'curl -X POST {{base_url}}/api/reload', 'Reload AI models', true, false, 'download', 'info'),
    ('ai', 'view-logs', 'journalctl -u {{resource_name}} -n 100', 'View recent logs', false, false, 'file-text', 'primary');

-- Storage Resource Actions
INSERT INTO resource_actions (resource_type, action_name, action_command, description, requires_confirmation, is_dangerous, icon, button_color) VALUES
    ('storage', 'restart', 'docker restart {{resource_name}}', 'Restart storage service', true, false, 'refresh', 'warning'),
    ('storage', 'backup', '/scripts/backup-{{resource_type}}.sh {{resource_name}}', 'Create backup', true, false, 'save', 'success'),
    ('storage', 'check-health', 'curl -s {{health_check_endpoint}}', 'Check health status', false, false, 'heart', 'info'),
    ('storage', 'view-metrics', 'curl -s {{base_url}}/metrics', 'View metrics', false, false, 'bar-chart', 'primary'),
    ('storage', 'clear-cache', 'redis-cli -p {{port}} FLUSHDB', 'Clear cache (Redis only)', true, true, 'trash', 'danger');

-- Automation Resource Actions
INSERT INTO resource_actions (resource_type, action_name, action_command, description, requires_confirmation, is_dangerous, icon, button_color) VALUES
    ('automation', 'restart', 'docker restart {{resource_name}}', 'Restart automation service', true, false, 'refresh', 'warning'),
    ('automation', 'pause-workflows', 'curl -X POST {{base_url}}/api/workflows/pause', 'Pause all workflows', true, false, 'pause', 'warning'),
    ('automation', 'resume-workflows', 'curl -X POST {{base_url}}/api/workflows/resume', 'Resume workflows', false, false, 'play', 'success'),
    ('automation', 'export-config', 'curl -s {{base_url}}/api/export > /tmp/{{resource_name}}-export.json', 'Export configuration', false, false, 'download', 'info'),
    ('automation', 'view-active', 'curl -s {{base_url}}/api/workflows?status=active', 'View active workflows', false, false, 'activity', 'primary');

-- Agent Resource Actions
INSERT INTO resource_actions (resource_type, action_name, action_command, description, requires_confirmation, is_dangerous, icon, button_color) VALUES
    ('agent', 'restart', 'docker restart {{resource_name}}', 'Restart agent service', true, false, 'refresh', 'warning'),
    ('agent', 'reset-session', 'curl -X POST {{base_url}}/api/session/reset', 'Reset agent session', true, false, 'rotate-cw', 'warning'),
    ('agent', 'view-tasks', 'curl -s {{base_url}}/api/tasks', 'View current tasks', false, false, 'list', 'primary'),
    ('agent', 'clear-queue', 'curl -X DELETE {{base_url}}/api/queue', 'Clear task queue', true, true, 'trash-2', 'danger'),
    ('agent', 'view-logs', 'docker logs {{resource_name}} --tail 100', 'View recent logs', false, false, 'file-text', 'primary');

-- Execution Resource Actions
INSERT INTO resource_actions (resource_type, action_name, action_command, description, requires_confirmation, is_dangerous, icon, button_color) VALUES
    ('execution', 'restart', 'docker restart {{resource_name}}', 'Restart execution service', true, false, 'refresh', 'warning'),
    ('execution', 'clear-sandbox', 'rm -rf /tmp/{{resource_name}}-sandbox/*', 'Clear sandbox files', true, true, 'trash', 'danger'),
    ('execution', 'view-queue', 'curl -s {{base_url}}/api/submissions?status=pending', 'View execution queue', false, false, 'clock', 'primary'),
    ('execution', 'kill-running', 'curl -X POST {{base_url}}/api/submissions/kill-all', 'Kill all running executions', true, true, 'x-circle', 'danger'),
    ('execution', 'view-stats', 'curl -s {{base_url}}/api/statistics', 'View execution statistics', false, false, 'bar-chart-2', 'info');

-- Search Resource Actions
INSERT INTO resource_actions (resource_type, action_name, action_command, description, requires_confirmation, is_dangerous, icon, button_color) VALUES
    ('search', 'restart', 'docker restart {{resource_name}}', 'Restart search service', true, false, 'refresh', 'warning'),
    ('search', 'update-engines', 'curl -X POST {{base_url}}/api/engines/update', 'Update search engines', true, false, 'download-cloud', 'info'),
    ('search', 'clear-cache', 'curl -X DELETE {{base_url}}/api/cache', 'Clear search cache', true, false, 'trash', 'warning'),
    ('search', 'view-stats', 'curl -s {{base_url}}/api/stats', 'View search statistics', false, false, 'pie-chart', 'primary'),
    ('search', 'test-search', 'curl -s "{{base_url}}/search?q=test"', 'Test search functionality', false, false, 'search', 'success');

-- ============================================
-- Default Alert Rules
-- ============================================

-- Critical service down alerts
INSERT INTO alert_rules (rule_name, description, resource_type, metric_type, threshold_value, threshold_operator, threshold_duration_seconds, severity, notification_channels, cooldown_minutes) VALUES
    ('critical-service-down', 'Alert when critical service is down', NULL, 'availability', 0, 'eq', 60, 'critical', '["sms", "email"]', 5),
    ('service-degraded', 'Alert when service response time is high', NULL, 'response_time', 5000, 'gt', 120, 'warning', '["email"]', 15),
    ('storage-high-usage', 'Alert when storage usage is high', 'storage', 'disk_usage', 90, 'gt', 300, 'warning', '["email"]', 30),
    ('automation-workflow-failed', 'Alert when automation workflow fails', 'automation', 'error_rate', 50, 'gt', 60, 'warning', '["email", "slack"]', 20),
    ('ai-model-unavailable', 'Alert when AI model is unavailable', 'ai', 'availability', 0, 'eq', 30, 'critical', '["sms", "email"]', 10);

-- ============================================
-- Default Notification Channels
-- ============================================

INSERT INTO notification_channels (channel_name, channel_type, vault_path, config, is_enabled, is_default, priority) VALUES
    ('primary-sms', 'sms', 'secret/data/monitoring/twilio', '{"provider": "twilio", "to": "+1234567890"}', true, true, 100),
    ('primary-email', 'email', 'secret/data/monitoring/smtp', '{"to": "ops@vrooli.com", "from": "monitoring@vrooli.com"}', true, true, 90),
    ('slack-alerts', 'slack', 'secret/data/monitoring/slack', '{"channel": "#alerts", "username": "Resource Monitor"}', false, false, 80),
    ('webhook-backup', 'webhook', NULL, '{"url": "https://hooks.vrooli.com/monitoring", "method": "POST"}', false, false, 70);

-- ============================================
-- Sample Resources (Will be replaced by auto-discovery)
-- ============================================

-- These are examples that will be overwritten by auto-discovery
INSERT INTO resources (resource_name, resource_type, display_name, description, port, base_url, health_check_endpoint, is_critical) VALUES
    ('ollama', 'ai', 'Ollama LLM Service', 'Local LLM inference engine', 11434, 'http://localhost:11434', '/api/tags', true),
    ('postgres', 'storage', 'PostgreSQL Database', 'Primary relational database', 5433, NULL, NULL, true),
    ('redis', 'storage', 'Redis Cache', 'In-memory data store and message broker', 6380, 'http://localhost:6380', '/ping', true),
    ('questdb', 'storage', 'QuestDB Time-Series', 'High-performance time-series database', 9009, 'http://localhost:9009', '/status', true),
    ('n8n', 'automation', 'n8n Workflow Engine', 'Workflow automation platform', 5678, 'http://localhost:5678', '/healthz', true),
    ('node-red', 'automation', 'Node-RED', 'Flow-based development tool', 1880, 'http://localhost:1880', '/settings', true),
    ('windmill', 'automation', 'Windmill', 'Developer platform for scripts and workflows', 5681, 'http://localhost:5681', '/api/version', false),
    ('vault', 'storage', 'HashiCorp Vault', 'Secrets management system', 8200, 'http://localhost:8200', '/v1/sys/health', true)
ON CONFLICT (resource_name) DO NOTHING;

-- ============================================
-- Default Metric Configurations
-- ============================================

-- Add metric configurations for critical resources
INSERT INTO metric_configurations (resource_id, metric_name, metric_type, metric_unit, collection_interval_seconds, aggregation_method, retention_days)
SELECT 
    r.id,
    m.metric_name,
    m.metric_type,
    m.metric_unit,
    m.collection_interval,
    m.aggregation_method,
    m.retention_days
FROM resources r
CROSS JOIN (VALUES
    ('availability', 'gauge', 'boolean', 30, 'last', 30),
    ('response_time', 'gauge', 'ms', 30, 'avg', 30),
    ('error_count', 'counter', 'count', 60, 'sum', 7),
    ('request_count', 'counter', 'count', 60, 'sum', 7)
) AS m(metric_name, metric_type, metric_unit, collection_interval, aggregation_method, retention_days)
WHERE r.is_critical = true;

-- ============================================
-- System Configuration Updates
-- ============================================

-- Update system configuration with monitoring-specific settings
INSERT INTO system_config (key, value, description) VALUES
    ('port_discovery', '{"ranges": ["1000-2000", "3000-4000", "5000-6000", "8000-9000", "9000-10000", "11000-12000"], "timeout_ms": 100}', 'Port ranges for auto-discovery'),
    -- Note: known_services will be populated by startup script from resource-registry.json
    ('health_endpoints', '{"default": ["/health", "/healthz", "/api/health", "/status", "/api/status", "/ping", "/api/ping"], "custom": {"ollama": "/api/tags", "vault": "/v1/sys/health", "windmill": "/api/version", "node-red": "/settings", "questdb": "/status", "minio": "/minio/health/live", "qdrant": "/health", "judge0": "/system_info", "comfyui": "/system_stats", "huginn": "/users/sign_in", "searxng": "/healthz", "whisper": "/health", "unstructured-io": "/general/v0/general", "browserless": "/health", "agent-s2": "/health", "claude-code": "/health"}}', 'Health check endpoints to try'),
    ('alert_templates', '{"sms": "🚨 {{severity}}: {{resource_name}} - {{message}}", "email": "<h2>Alert: {{resource_name}}</h2><p>Severity: {{severity}}</p><p>{{message}}</p><p>Time: {{timestamp}}</p>"}', 'Message templates for notifications')
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_at = NOW();

-- ============================================
-- Dashboard Configuration Defaults
-- ============================================

-- Create default dashboard configuration
INSERT INTO dashboard_configs (user_id, layout, widgets, resource_filters, severity_filters) VALUES
    ('default', 
     '{"grid": {"cols": 12, "rowHeight": 60, "margin": [10, 10]}}',
     '[
        {"id": "status-grid", "type": "resource-status", "position": {"x": 0, "y": 0, "w": 12, "h": 4}},
        {"id": "alerts-timeline", "type": "alert-timeline", "position": {"x": 0, "y": 4, "w": 8, "h": 3}},
        {"id": "metrics-chart", "type": "metrics-chart", "position": {"x": 8, "y": 4, "w": 4, "h": 3}},
        {"id": "resource-actions", "type": "action-panel", "position": {"x": 0, "y": 7, "w": 6, "h": 2}},
        {"id": "system-logs", "type": "log-viewer", "position": {"x": 6, "y": 7, "w": 6, "h": 2}}
     ]',
     '{"show_critical": true, "show_enabled": true, "types": ["ai", "storage", "automation"]}',
     '["warning", "critical"]')
ON CONFLICT (user_id) DO NOTHING;

-- ============================================
-- Create Helper Functions
-- ============================================

-- Function to get resource status summary
CREATE OR REPLACE FUNCTION get_resource_health_summary()
RETURNS TABLE(
    total_resources INTEGER,
    healthy_count INTEGER,
    degraded_count INTEGER,
    down_count INTEGER,
    critical_down INTEGER
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER AS total_resources,
        COUNT(*) FILTER (WHERE current_status = 'healthy')::INTEGER AS healthy_count,
        COUNT(*) FILTER (WHERE current_status = 'degraded')::INTEGER AS degraded_count,
        COUNT(*) FILTER (WHERE current_status = 'down')::INTEGER AS down_count,
        COUNT(*) FILTER (WHERE current_status = 'down' AND is_critical = true)::INTEGER AS critical_down
    FROM resources
    WHERE is_enabled = true;
END;
$$ LANGUAGE plpgsql;

-- Function to format alert message
CREATE OR REPLACE FUNCTION format_alert_message(
    p_template TEXT,
    p_resource_name TEXT,
    p_severity TEXT,
    p_message TEXT
) RETURNS TEXT AS $$
BEGIN
    RETURN REPLACE(
        REPLACE(
            REPLACE(
                REPLACE(p_template, '{{resource_name}}', p_resource_name),
                '{{severity}}', p_severity
            ),
            '{{message}}', p_message
        ),
        '{{timestamp}}', TO_CHAR(NOW(), 'YYYY-MM-DD HH24:MI:SS')
    );
END;
$$ LANGUAGE plpgsql;