-- Test data for Email Triage development
-- Only loaded in DEV_MODE

-- Insert dev user if not exists
INSERT INTO users (id, email_profiles, preferences, plan_type, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000001'::UUID,
    '[{"email": "dev@example.com", "name": "Dev User"}]'::jsonb,
    '{"theme": "dark", "notifications": true}'::jsonb,
    'pro',
    NOW(),
    NOW()
) ON CONFLICT (id) DO NOTHING;

-- Insert a test email account
INSERT INTO email_accounts (id, user_id, email_address, imap_settings, smtp_settings, sync_enabled)
VALUES (
    '00000000-0000-0000-0000-000000000002'::UUID,
    '00000000-0000-0000-0000-000000000001'::UUID,
    'dev@example.com',
    '{"server": "imap.example.com", "port": 993, "username": "dev@example.com", "password": "test", "use_tls": true}'::jsonb,
    '{"server": "smtp.example.com", "port": 587, "username": "dev@example.com", "password": "test", "use_tls": true}'::jsonb,
    true
) ON CONFLICT (user_id, email_address) DO NOTHING;

-- Insert some test processed emails with varying priorities
INSERT INTO processed_emails (id, account_id, message_id, subject, sender_email, recipient_emails, body_preview, full_body, priority_score, processed_at)
VALUES 
(
    '00000000-0000-0000-0000-000000000003'::UUID,
    '00000000-0000-0000-0000-000000000002'::UUID,
    'msg-001@example.com',
    'Urgent: Board Meeting Tomorrow',
    'ceo@company.com',
    ARRAY['dev@example.com'],
    'Please attend the board meeting tomorrow at 2 PM to discuss Q4 results...',
    'Please attend the board meeting tomorrow at 2 PM to discuss Q4 results. The meeting will be held in the main conference room. Please prepare your department reports.',
    0.95,
    NOW() - INTERVAL '2 hours'
),
(
    '00000000-0000-0000-0000-000000000004'::UUID,
    '00000000-0000-0000-0000-000000000002'::UUID,
    'msg-002@example.com',
    'Weekly Newsletter: Tech Updates',
    'newsletter@techblog.com',
    ARRAY['dev@example.com'],
    'This week in tech: AI breakthroughs, new frameworks...',
    'This week in tech: AI breakthroughs, new framework releases, industry updates. Click here to unsubscribe.',
    0.25,
    NOW() - INTERVAL '1 day'
),
(
    '00000000-0000-0000-0000-000000000005'::UUID,
    '00000000-0000-0000-0000-000000000002'::UUID,
    'msg-003@example.com',
    'Invoice #12345 Due Next Week',
    'billing@vendor.com',
    ARRAY['dev@example.com', 'accounting@company.com'],
    'Invoice #12345 for $5,000 is due next Friday...',
    'Invoice #12345 for $5,000 is due next Friday. Please process payment to avoid late fees. Thank you for your business.',
    0.75,
    NOW() - INTERVAL '3 days'
)
ON CONFLICT (account_id, message_id) DO NOTHING;

-- Insert a test triage rule
INSERT INTO triage_rules (id, user_id, name, description, conditions, actions, priority, enabled, created_by_ai, ai_confidence)
VALUES (
    '00000000-0000-0000-0000-000000000006'::UUID,
    '00000000-0000-0000-0000-000000000001'::UUID,
    'Archive Newsletters',
    'Automatically archive all newsletter emails',
    '[{"field": "sender", "operator": "contains", "value": "newsletter", "weight": 0.9}]'::jsonb,
    '[{"type": "archive", "parameters": {}}, {"type": "label", "parameters": {"label": "newsletter"}}]'::jsonb,
    100,
    true,
    false,
    0.85
) ON CONFLICT DO NOTHING;