-- Workflow Scheduler Seed Data
-- Initial presets and example schedules for demonstration

-- Insert common cron expression presets
INSERT INTO cron_presets (name, description, expression, category, is_system) VALUES
-- Time-based presets
('Every Minute', 'Runs every minute (use with caution)', '* * * * *', 'frequent', true),
('Every 5 Minutes', 'Runs every 5 minutes', '*/5 * * * *', 'frequent', true),
('Every 15 Minutes', 'Runs every 15 minutes', '*/15 * * * *', 'frequent', true),
('Every 30 Minutes', 'Runs every 30 minutes', '*/30 * * * *', 'frequent', true),
('Hourly', 'Runs at the start of every hour', '0 * * * *', 'hourly', true),
('Every 2 Hours', 'Runs every 2 hours', '0 */2 * * *', 'hourly', true),
('Every 4 Hours', 'Runs every 4 hours', '0 */4 * * *', 'hourly', true),
('Every 6 Hours', 'Runs 4 times a day', '0 */6 * * *', 'hourly', true),

-- Daily presets
('Daily at Midnight', 'Runs at midnight every day', '0 0 * * *', 'daily', true),
('Daily at 6 AM', 'Runs at 6 AM every day', '0 6 * * *', 'daily', true),
('Daily at 9 AM', 'Runs at 9 AM every day', '0 9 * * *', 'daily', true),
('Daily at Noon', 'Runs at noon every day', '0 12 * * *', 'daily', true),
('Daily at 6 PM', 'Runs at 6 PM every day', '0 18 * * *', 'daily', true),
('Twice Daily', 'Runs at 9 AM and 5 PM', '0 9,17 * * *', 'daily', true),
('Business Hours', 'Every hour during business hours', '0 9-17 * * 1-5', 'daily', true),

-- Weekly presets
('Weekly on Monday', 'Runs every Monday at 9 AM', '0 9 * * 1', 'weekly', true),
('Weekly on Friday', 'Runs every Friday at 5 PM', '0 17 * * 5', 'weekly', true),
('Weekdays Only', 'Runs Monday-Friday at 9 AM', '0 9 * * 1-5', 'weekly', true),
('Weekends Only', 'Runs Saturday-Sunday at 10 AM', '0 10 * * 0,6', 'weekly', true),
('MWF Schedule', 'Monday, Wednesday, Friday at 10 AM', '0 10 * * 1,3,5', 'weekly', true),

-- Monthly presets
('Monthly First Day', 'First day of month at midnight', '0 0 1 * *', 'monthly', true),
('Monthly Last Day', 'Last day of month at 11 PM', '0 23 28-31 * *', 'monthly', true),
('Monthly Mid-Month', 'Runs on the 15th at noon', '0 12 15 * *', 'monthly', true),
('Quarterly', 'First day of each quarter', '0 0 1 1,4,7,10 *', 'monthly', true),
('Bi-Monthly', 'Every two months on the 1st', '0 0 1 */2 *', 'monthly', true),

-- Special presets
('Maintenance Window', 'Sunday 2-4 AM maintenance', '0 2-4 * * 0', 'maintenance', true),
('Backup Schedule', 'Daily backup at 3 AM', '0 3 * * *', 'backup', true),
('Report Generation', 'Weekly reports on Monday morning', '0 8 * * 1', 'reporting', true),
('Health Check', 'Every 10 minutes health check', '*/10 * * * *', 'monitoring', true),
('Data Sync', 'Hourly data synchronization', '0 * * * *', 'sync', true);

-- Insert default notification channels
INSERT INTO notification_channels (name, channel_type, enabled, min_severity) VALUES
('Default Webhook', 'webhook', true, 'medium'),
('System UI Notifications', 'ui', true, 'low'),
('Critical Alerts', 'webhook', true, 'critical');

-- Insert example schedules (disabled by default for safety)
INSERT INTO schedules (
    name, 
    description, 
    cron_expression, 
    timezone,
    target_type,
    target_url,
    status,
    enabled,
    overlap_policy,
    tags,
    owner,
    priority
) VALUES
(
    'System Health Check',
    'Regular health check of all Vrooli services',
    '*/10 * * * *',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/health-check',
    'active',
    false,  -- Disabled by default
    'skip',
    ARRAY['monitoring', 'system', 'health'],
    'system',
    8
),
(
    'Daily Report Generator',
    'Generate daily analytics report',
    '0 9 * * *',
    'America/New_York',
    'webhook',
    'http://localhost:5678/webhook/daily-report',
    'active',
    false,  -- Disabled by default
    'skip',
    ARRAY['reporting', 'analytics', 'daily'],
    'analytics-team',
    6
),
(
    'Database Backup',
    'Nightly database backup to MinIO',
    '0 3 * * *',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/backup-database',
    'active',
    false,  -- Disabled by default
    'skip',
    ARRAY['backup', 'database', 'critical'],
    'ops-team',
    10
),
(
    'Cache Cleanup',
    'Clean expired cache entries',
    '0 */6 * * *',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/cache-cleanup',
    'active',
    false,  -- Disabled by default
    'allow',
    ARRAY['maintenance', 'cache', 'cleanup'],
    'system',
    4
),
(
    'Weekly Metrics Aggregation',
    'Aggregate weekly performance metrics',
    '0 0 * * 1',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/weekly-metrics',
    'active',
    false,  -- Disabled by default
    'queue',
    ARRAY['metrics', 'weekly', 'analytics'],
    'analytics-team',
    5
),
(
    'Document Sync',
    'Synchronize documentation across services',
    '0 */2 * * *',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/doc-sync',
    'active',
    false,  -- Disabled by default
    'skip',
    ARRAY['sync', 'documentation'],
    'docs-team',
    5
),
(
    'AI Model Update Check',
    'Check for AI model updates',
    '0 0 * * *',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/model-update-check',
    'active',
    false,  -- Disabled by default
    'skip',
    ARRAY['ai', 'models', 'updates'],
    'ml-team',
    7
),
(
    'Resource Usage Monitor',
    'Monitor resource usage and alert on thresholds',
    '*/5 * * * *',
    'UTC',
    'webhook',
    'http://localhost:5678/webhook/resource-monitor',
    'active',
    false,  -- Disabled by default
    'skip',
    ARRAY['monitoring', 'resources', 'alerts'],
    'ops-team',
    9
);

-- Insert sample execution history for demonstration
-- (Only for the System Health Check schedule)
WITH health_schedule AS (
    SELECT id FROM schedules WHERE name = 'System Health Check' LIMIT 1
)
INSERT INTO executions (
    schedule_id,
    scheduled_time,
    start_time,
    end_time,
    duration_ms,
    status,
    response_code,
    response_body
)
SELECT 
    hs.id,
    CURRENT_TIMESTAMP - (interval '10 minutes' * s),
    CURRENT_TIMESTAMP - (interval '10 minutes' * s) + interval '2 seconds',
    CURRENT_TIMESTAMP - (interval '10 minutes' * s) + interval '3 seconds',
    1000 + (random() * 2000)::int,
    CASE 
        WHEN s % 10 = 0 THEN 'failed'::execution_status
        ELSE 'success'::execution_status
    END,
    CASE 
        WHEN s % 10 = 0 THEN 500
        ELSE 200
    END,
    CASE 
        WHEN s % 10 = 0 THEN '{"error": "Service temporarily unavailable"}'
        ELSE '{"status": "healthy", "services": 12, "uptime": 99.9}'
    END
FROM health_schedule hs, generate_series(1, 20) s;

-- Update metrics for schedules with execution history
INSERT INTO schedule_metrics (schedule_id, total_executions, success_count, failure_count, success_rate, health_score)
SELECT 
    s.id,
    COUNT(e.id),
    COUNT(e.id) FILTER (WHERE e.status = 'success'),
    COUNT(e.id) FILTER (WHERE e.status = 'failed'),
    (COUNT(e.id) FILTER (WHERE e.status = 'success')::DECIMAL / NULLIF(COUNT(e.id), 0) * 100),
    CASE 
        WHEN (COUNT(e.id) FILTER (WHERE e.status = 'success')::DECIMAL / NULLIF(COUNT(e.id), 0)) > 0.95 THEN 100
        WHEN (COUNT(e.id) FILTER (WHERE e.status = 'success')::DECIMAL / NULLIF(COUNT(e.id), 0)) > 0.90 THEN 90
        WHEN (COUNT(e.id) FILTER (WHERE e.status = 'success')::DECIMAL / NULLIF(COUNT(e.id), 0)) > 0.80 THEN 80
        ELSE 70
    END
FROM schedules s
LEFT JOIN executions e ON s.id = e.schedule_id
GROUP BY s.id
ON CONFLICT (schedule_id) DO NOTHING;

-- Add helpful comments for new users
COMMENT ON SCHEMA public IS 'Workflow Scheduler - Enterprise-grade cron scheduling for Vrooli';

-- Print summary
DO $$
BEGIN
    RAISE NOTICE 'Workflow Scheduler database initialized successfully!';
    RAISE NOTICE '- Created % cron presets', (SELECT COUNT(*) FROM cron_presets);
    RAISE NOTICE '- Created % example schedules (disabled by default)', (SELECT COUNT(*) FROM schedules);
    RAISE NOTICE '- Created % notification channels', (SELECT COUNT(*) FROM notification_channels);
    RAISE NOTICE '- Generated sample execution history for demonstration';
    RAISE NOTICE '';
    RAISE NOTICE 'To enable example schedules, use the API or CLI:';
    RAISE NOTICE '  scheduler-cli enable "System Health Check"';
    RAISE NOTICE '  scheduler-cli enable "Daily Report Generator"';
END $$;