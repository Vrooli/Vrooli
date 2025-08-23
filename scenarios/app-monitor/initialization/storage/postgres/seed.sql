-- App Monitor Database Seed Data

-- Insert sample app records for demonstration
INSERT INTO apps (name, scenario_name, path, status, port_mappings, environment, config)
VALUES 
('sample-web-app', 'web-application', '/scripts/scenarios/core/sample-web-app', 'stopped', 
 '{"web": 3000, "api": 8080}', '{"NODE_ENV": "production"}', 
 '{"container_name": "sample-web-app", "auto_restart": true}'),
('task-scheduler', 'automation', '/scripts/scenarios/core/task-scheduler', 'stopped',
 '{"api": 8090}', '{"LOG_LEVEL": "info"}',
 '{"container_name": "task-scheduler", "auto_restart": false}')
ON CONFLICT (name) DO NOTHING;

-- Insert sample app status records
INSERT INTO app_status (app_id, status, cpu_usage, memory_usage, disk_usage, timestamp)
SELECT 
    a.id,
    'healthy',
    random() * 50 + 10,  -- CPU usage between 10-60%
    random() * 30 + 20,  -- Memory usage between 20-50%
    random() * 20 + 5,   -- Disk usage between 5-25%
    NOW() - INTERVAL '1 hour'
FROM apps a
ON CONFLICT DO NOTHING;

-- Insert sample log entries
INSERT INTO app_logs (app_id, level, message, source, timestamp)
SELECT 
    a.id,
    'info',
    'Application started successfully',
    'system',
    NOW() - INTERVAL '2 hours'
FROM apps a
ON CONFLICT DO NOTHING;

INSERT INTO app_logs (app_id, level, message, source, timestamp)
SELECT 
    a.id,
    'info',
    'Health check passed',
    'monitor',
    NOW() - INTERVAL '30 minutes'
FROM apps a
ON CONFLICT DO NOTHING;

-- Create indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_apps_status ON apps(status);
CREATE INDEX IF NOT EXISTS idx_app_status_timestamp_desc ON app_status(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_app_logs_level ON app_logs(level);
EOF < /dev/null
