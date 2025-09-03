-- Notification Hub Seed Data
-- Initialize database with default profiles, templates, and sample data

-- =============================================================================
-- DEFAULT PROFILES
-- =============================================================================

-- Create a default "demo" profile for testing
INSERT INTO profiles (id, name, slug, api_key_hash, api_key_prefix, settings, plan, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'Demo Organization',
    'demo-org',
    crypt('demo_7f8e9d2c3b4a5e6f7890abcdef123456', gen_salt('bf')),
    'demo_',
    '{"timezone": "UTC", "default_from_email": "demo@vrooli.local", "default_from_name": "Demo Organization"}',
    'free',
    'active'
);

-- Create a default "system" profile for Vrooli internal notifications
INSERT INTO profiles (id, name, slug, api_key_hash, api_key_prefix, settings, plan, status)
VALUES (
    '00000000-0000-0000-0000-000000000002', 
    'Vrooli System',
    'vrooli-system',
    crypt('sys_1a2b3c4d5e6f7890abcdef123456789a', gen_salt('bf')),
    'sys_',
    '{"timezone": "UTC", "default_from_email": "system@vrooli.local", "default_from_name": "Vrooli"}',
    'unlimited',
    'active'
);

-- =============================================================================
-- DEFAULT PROFILE LIMITS
-- =============================================================================

-- Demo profile limits (free tier)
INSERT INTO profile_limits (profile_id, notification_type, hourly_limit, daily_limit, monthly_limit, hourly_cost_limit, daily_cost_limit, monthly_cost_limit)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'email', 100, 1000, 10000, 1000, 5000, 50000),
    ('00000000-0000-0000-0000-000000000001', 'sms', 10, 50, 500, 2000, 10000, 100000),
    ('00000000-0000-0000-0000-000000000001', 'push', 500, 5000, 50000, 500, 2000, 20000);

-- System profile limits (unlimited)
INSERT INTO profile_limits (profile_id, notification_type, hourly_limit, daily_limit, monthly_limit, hourly_cost_limit, daily_cost_limit, monthly_cost_limit)
VALUES 
    ('00000000-0000-0000-0000-000000000002', 'email', 10000, 100000, 1000000, 100000, 500000, 5000000),
    ('00000000-0000-0000-0000-000000000002', 'sms', 5000, 50000, 500000, 200000, 1000000, 10000000),
    ('00000000-0000-0000-0000-000000000002', 'push', 50000, 500000, 5000000, 50000, 200000, 2000000);

-- =============================================================================
-- DEFAULT PROVIDER CONFIGURATIONS
-- =============================================================================

-- Demo SMTP email provider (for testing)
INSERT INTO profile_providers (profile_id, channel, provider, config, priority, enabled)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'email',
    'smtp',
    '{"host": "smtp.mailtrap.io", "port": 587, "username": "demo", "password": "demo", "from_email": "demo@vrooli.local", "from_name": "Demo Organization"}',
    1,
    true
);

-- Demo SMS provider (mock/testing)
INSERT INTO profile_providers (profile_id, channel, provider, config, priority, enabled) 
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'sms',
    'mock',
    '{"from_number": "+15551234567", "mock_mode": true}',
    1,
    true
);

-- Demo push provider (mock/testing)
INSERT INTO profile_providers (profile_id, channel, provider, config, priority, enabled)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'push',
    'mock',
    '{"mock_mode": true}',
    1,
    true
);

-- System providers (for Vrooli internal use)
INSERT INTO profile_providers (profile_id, channel, provider, config, priority, enabled)
VALUES 
    ('00000000-0000-0000-0000-000000000002', 'email', 'smtp', '{"host": "localhost", "port": 1025, "from_email": "system@vrooli.local", "from_name": "Vrooli"}', 1, true),
    ('00000000-0000-0000-0000-000000000002', 'sms', 'mock', '{"from_number": "+15551234567", "mock_mode": true}', 1, true),
    ('00000000-0000-0000-0000-000000000002', 'push', 'mock', '{"mock_mode": true}', 1, true);

-- =============================================================================
-- DEFAULT CONTACTS
-- =============================================================================

-- Demo contacts for testing
INSERT INTO contacts (id, profile_id, external_id, identifier, first_name, last_name, timezone, preferences, status)
VALUES 
    ('10000000-0000-0000-0000-000000000001', '00000000-0000-0000-0000-000000000001', 'user_1', 'demo@example.com', 'Demo', 'User', 'UTC', '{"email": true, "sms": false, "push": true}', 'active'),
    ('10000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'user_2', 'test@example.com', 'Test', 'Account', 'America/New_York', '{"email": true, "sms": true, "push": false}', 'active');

-- Demo contact channels
INSERT INTO contact_channels (contact_id, type, value, verified, preferences, status)
VALUES 
    ('10000000-0000-0000-0000-000000000001', 'email', 'demo@example.com', true, '{}', 'active'),
    ('10000000-0000-0000-0000-000000000001', 'push', 'demo_push_token_123', false, '{}', 'active'),
    ('10000000-0000-0000-0000-000000000002', 'email', 'test@example.com', true, '{}', 'active'),
    ('10000000-0000-0000-0000-000000000002', 'sms', '+15551234567', false, '{}', 'active');

-- =============================================================================
-- DEFAULT TEMPLATES
-- =============================================================================

-- Welcome email template
INSERT INTO templates (id, profile_id, name, slug, channels, category, subject, content, variables, status)
VALUES (
    '20000000-0000-0000-0000-000000000001',
    '00000000-0000-0000-0000-000000000001',
    'Welcome Email',
    'welcome-email',
    '{"email"}',
    'onboarding',
    'Welcome to {{organization_name}}, {{first_name}}!',
    '{
        "email": {
            "html": "<h1>Welcome {{first_name}}!</h1><p>Thanks for joining {{organization_name}}. We''re excited to have you on board!</p><p>Get started by exploring our features and setting up your profile.</p><p>Best regards,<br>The {{organization_name}} Team</p>",
            "text": "Welcome {{first_name}}!\n\nThanks for joining {{organization_name}}. We''re excited to have you on board!\n\nGet started by exploring our features and setting up your profile.\n\nBest regards,\nThe {{organization_name}} Team"
        }
    }',
    '{"first_name": "User", "organization_name": "Demo Organization"}',
    'active'
);

-- Password reset template
INSERT INTO templates (id, profile_id, name, slug, channels, category, subject, content, variables, status)
VALUES (
    '20000000-0000-0000-0000-000000000002',
    '00000000-0000-0000-0000-000000000001', 
    'Password Reset',
    'password-reset',
    '{"email"}',
    'security',
    'Reset your password for {{organization_name}}',
    '{
        "email": {
            "html": "<h1>Password Reset Request</h1><p>Hi {{first_name}},</p><p>We received a request to reset your password for your {{organization_name}} account.</p><p><a href=\"{{reset_url}}\" style=\"background: #007cba; color: white; padding: 12px 24px; text-decoration: none; border-radius: 4px;\">Reset Password</a></p><p>This link will expire in {{expiry_hours}} hours.</p><p>If you didn''t request this, please ignore this email.</p>",
            "text": "Password Reset Request\n\nHi {{first_name}},\n\nWe received a request to reset your password for your {{organization_name}} account.\n\nReset your password: {{reset_url}}\n\nThis link will expire in {{expiry_hours}} hours.\n\nIf you didn''t request this, please ignore this email."
        }
    }',
    '{"first_name": "User", "organization_name": "Demo Organization", "reset_url": "https://example.com/reset", "expiry_hours": "24"}',
    'active'
);

-- Multi-channel notification template
INSERT INTO templates (id, profile_id, name, slug, channels, category, subject, content, variables, status)
VALUES (
    '20000000-0000-0000-0000-000000000003',
    '00000000-0000-0000-0000-000000000001',
    'Alert Notification',
    'alert-notification', 
    '{"email", "sms", "push"}',
    'alerts',
    'ALERT: {{alert_title}}',
    '{
        "email": {
            "html": "<h1 style=\"color: #d32f2f;\">⚠️ {{alert_title}}</h1><p>{{alert_message}}</p><p><strong>Severity:</strong> {{severity}}</p><p><strong>Time:</strong> {{timestamp}}</p><hr><p><small>This is an automated alert from {{organization_name}}.</small></p>",
            "text": "⚠️ ALERT: {{alert_title}}\n\n{{alert_message}}\n\nSeverity: {{severity}}\nTime: {{timestamp}}\n\nThis is an automated alert from {{organization_name}}."
        },
        "sms": {
            "text": "ALERT: {{alert_title}} - {{alert_message}} [{{severity}}] {{timestamp}}"
        },
        "push": {
            "title": "{{alert_title}}",
            "body": "{{alert_message}}",
            "data": {"severity": "{{severity}}", "timestamp": "{{timestamp}}"}
        }
    }',
    '{"alert_title": "System Alert", "alert_message": "Something important happened", "severity": "HIGH", "timestamp": "2024-01-01 12:00:00", "organization_name": "Demo Organization"}',
    'active'
);

-- System templates for Vrooli internal use
INSERT INTO templates (id, profile_id, name, slug, channels, category, subject, content, variables, status)
VALUES 
    (
        '20000000-0000-0000-0000-000000000004',
        '00000000-0000-0000-0000-000000000002',
        'Scenario Deployment Complete',
        'scenario-deployment-complete',
        '{"email", "push"}',
        'system',
        'Scenario {{scenario_name}} deployed successfully',
        '{
            "email": {
                "html": "<h1>🚀 Deployment Complete</h1><p>Your scenario <strong>{{scenario_name}}</strong> has been successfully deployed and is now running.</p><p><strong>URL:</strong> <a href=\"{{scenario_url}}\">{{scenario_url}}</a></p><p><strong>Deployment time:</strong> {{deployment_time}}</p>",
                "text": "🚀 Deployment Complete\n\nYour scenario {{scenario_name}} has been successfully deployed and is now running.\n\nURL: {{scenario_url}}\nDeployment time: {{deployment_time}}"
            },
            "push": {
                "title": "Deployment Complete",
                "body": "{{scenario_name}} is now running",
                "data": {"scenario": "{{scenario_name}}", "url": "{{scenario_url}}"}
            }
        }',
        '{"scenario_name": "Example Scenario", "scenario_url": "http://localhost:3000", "deployment_time": "2 minutes"}',
        'active'
    ),
    (
        '20000000-0000-0000-0000-000000000005', 
        '00000000-0000-0000-0000-000000000002',
        'System Maintenance Notice',
        'system-maintenance-notice',
        '{"email", "sms"}',
        'system',
        'Vrooli maintenance scheduled for {{maintenance_date}}',
        '{
            "email": {
                "html": "<h1>🔧 Scheduled Maintenance</h1><p>Vrooli will undergo scheduled maintenance on <strong>{{maintenance_date}}</strong> from {{start_time}} to {{end_time}} {{timezone}}.</p><p><strong>Expected downtime:</strong> {{duration}}</p><p><strong>Affected services:</strong> {{affected_services}}</p><p>We apologize for any inconvenience.</p>",
                "text": "🔧 Scheduled Maintenance\n\nVrooli will undergo scheduled maintenance on {{maintenance_date}} from {{start_time}} to {{end_time}} {{timezone}}.\n\nExpected downtime: {{duration}}\nAffected services: {{affected_services}}\n\nWe apologize for any inconvenience."
            },
            "sms": {
                "text": "Vrooli maintenance: {{maintenance_date}} {{start_time}}-{{end_time}} {{timezone}}. Duration: {{duration}}. Services: {{affected_services}}"
            }
        }',
        '{"maintenance_date": "2024-01-15", "start_time": "02:00", "end_time": "04:00", "timezone": "UTC", "duration": "2 hours", "affected_services": "Web UI, API"}',
        'active'
    );

-- Create default template versions
INSERT INTO template_versions (template_id, version, name, subject, content, variables, active)
SELECT 
    id,
    1,
    'Initial Version',
    subject,
    content,
    variables,
    true
FROM templates;

-- =============================================================================
-- SAMPLE ANALYTICS DATA
-- =============================================================================

-- Insert sample analytics for the past 7 days
INSERT INTO profile_analytics_daily (profile_id, date, notifications_sent, notifications_delivered, email_sent, email_delivered, email_opened, email_clicked)
SELECT 
    '00000000-0000-0000-0000-000000000001',
    CURRENT_DATE - INTERVAL '1 day' * i,
    FLOOR(RANDOM() * 100) + 20,
    FLOOR(RANDOM() * 95) + 18,
    FLOOR(RANDOM() * 80) + 15,
    FLOOR(RANDOM() * 75) + 13,
    FLOOR(RANDOM() * 30) + 5,
    FLOOR(RANDOM() * 10) + 1
FROM generate_series(0, 6) i;

-- =============================================================================
-- UTILITY FUNCTIONS AND VIEWS
-- =============================================================================

-- View for profile overview
CREATE VIEW profile_overview AS
SELECT 
    p.id,
    p.name,
    p.slug,
    p.api_key_prefix,
    p.plan,
    p.status,
    p.created_at,
    COUNT(DISTINCT c.id) as total_contacts,
    COUNT(DISTINCT t.id) as total_templates,
    COUNT(DISTINCT CASE WHEN n.created_at >= CURRENT_DATE - INTERVAL '30 days' THEN n.id END) as notifications_last_30_days,
    COUNT(DISTINCT CASE WHEN n.created_at >= CURRENT_DATE - INTERVAL '7 days' THEN n.id END) as notifications_last_7_days
FROM profiles p
LEFT JOIN contacts c ON p.id = c.profile_id AND c.status = 'active'
LEFT JOIN templates t ON p.id = t.profile_id AND t.status = 'active'  
LEFT JOIN notifications n ON p.id = n.profile_id
WHERE p.status = 'active'
GROUP BY p.id, p.name, p.slug, p.api_key_prefix, p.plan, p.status, p.created_at;

-- View for recent notifications
CREATE VIEW recent_notifications AS
SELECT 
    n.id,
    n.profile_id,
    p.name as profile_name,
    c.identifier as contact_identifier,
    c.first_name,
    c.last_name,
    n.subject,
    n.status,
    n.channels_requested,
    n.channels_attempted,
    n.created_at,
    n.sent_at,
    COUNT(nd.id) as delivery_attempts,
    COUNT(CASE WHEN nd.status = 'delivered' THEN 1 END) as successful_deliveries
FROM notifications n
JOIN profiles p ON n.profile_id = p.id
JOIN contacts c ON n.contact_id = c.id
LEFT JOIN notification_deliveries nd ON n.id = nd.notification_id
WHERE n.created_at >= CURRENT_DATE - INTERVAL '7 days'
GROUP BY n.id, n.profile_id, p.name, c.identifier, c.first_name, c.last_name, n.subject, n.status, n.channels_requested, n.channels_attempted, n.created_at, n.sent_at
ORDER BY n.created_at DESC;

-- View for delivery statistics
CREATE VIEW delivery_stats AS
SELECT 
    p.id as profile_id,
    p.name as profile_name,
    nd.channel,
    nd.provider,
    COUNT(*) as total_attempts,
    COUNT(CASE WHEN nd.status = 'delivered' THEN 1 END) as delivered,
    COUNT(CASE WHEN nd.status = 'failed' THEN 1 END) as failed,
    COUNT(CASE WHEN nd.status = 'bounced' THEN 1 END) as bounced,
    ROUND(COUNT(CASE WHEN nd.status = 'delivered' THEN 1 END) * 100.0 / COUNT(*), 2) as delivery_rate,
    SUM(COALESCE(nd.cost_cents, 0)) as total_cost_cents
FROM notification_deliveries nd
JOIN notifications n ON nd.notification_id = n.id
JOIN profiles p ON n.profile_id = p.id
WHERE nd.created_at >= CURRENT_DATE - INTERVAL '30 days'
GROUP BY p.id, p.name, nd.channel, nd.provider
ORDER BY p.name, nd.channel, nd.provider;

COMMENT ON TABLE profiles IS 'Multi-tenant profiles for notification isolation';
COMMENT ON TABLE contacts IS 'Notification recipients within each profile';
COMMENT ON TABLE templates IS 'Reusable notification templates per profile';
COMMENT ON TABLE notifications IS 'Individual notification instances';
COMMENT ON TABLE notification_deliveries IS 'Per-channel delivery tracking';
COMMENT ON VIEW profile_overview IS 'Summary statistics for each profile';
COMMENT ON VIEW recent_notifications IS 'Recent notifications with delivery status';
COMMENT ON VIEW delivery_stats IS 'Channel and provider performance metrics';