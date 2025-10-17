-- Seed data for Scenario Authenticator
-- Creates initial admin user and test data

-- Create admin user (password: Admin123!)
-- Note: In production, this should be changed immediately
INSERT INTO users (
    email,
    username,
    password_hash,
    roles,
    metadata,
    email_verified,
    created_at
) VALUES (
    'admin@vrooli.local',
    'admin',
    -- bcrypt hash for 'Admin123!' (cost factor 10)
    '$2a$10$YKxo4cVSbKeP6pKnhWFXeOea8HBmW8W5GyY2PnCrI2vWXzqKJqWXe',
    '["admin", "user"]'::jsonb,
    '{"source": "seed", "description": "Initial admin user"}'::jsonb,
    true,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO NOTHING;

-- Create test user (password: Test123!)
INSERT INTO users (
    email,
    username,
    password_hash,
    roles,
    metadata,
    email_verified,
    created_at
) VALUES (
    'test@vrooli.local',
    'testuser',
    -- bcrypt hash for 'Test123!'
    '$2a$10$RHtcy3r8v9hPqGv7x3f1PuXmQdHVHcE0W7Kz8Y5P9vJQxR3kXxXXy',
    '["user"]'::jsonb,
    '{"source": "seed", "description": "Test user account"}'::jsonb,
    true,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO NOTHING;

-- Create demo user (password: Demo123!)
INSERT INTO users (
    email,
    username,
    password_hash,
    roles,
    metadata,
    email_verified,
    created_at
) VALUES (
    'demo@vrooli.local',
    'demo',
    -- bcrypt hash for 'Demo123!'
    '$2a$10$5kL8PYrPq4Hy.vT9YFqGYuWXzqKJqWXeYKxo4cVSbKeP6pKnhWFXe',
    '["guest"]'::jsonb,
    '{"source": "seed", "description": "Demo user with limited access"}'::jsonb,
    true,
    CURRENT_TIMESTAMP
) ON CONFLICT (email) DO NOTHING;

-- Create API key for admin user for testing
INSERT INTO api_keys (
    user_id,
    key_hash,
    name,
    permissions,
    rate_limit,
    created_at
)
SELECT 
    id,
    -- SHA256 hash of 'vak_test_admin_key_123456789'
    encode(sha256('vak_test_admin_key_123456789'::bytea), 'hex'),
    'Test Admin API Key',
    '["*"]'::jsonb,
    10000,
    CURRENT_TIMESTAMP
FROM users 
WHERE email = 'admin@vrooli.local'
ON CONFLICT (key_hash) DO NOTHING;

-- Create API key for test user
INSERT INTO api_keys (
    user_id,
    key_hash,
    name,
    permissions,
    rate_limit,
    expires_at,
    created_at
)
SELECT 
    id,
    -- SHA256 hash of 'vak_test_user_key_987654321'
    encode(sha256('vak_test_user_key_987654321'::bytea), 'hex'),
    'Test User API Key',
    '["read:own", "write:own"]'::jsonb,
    1000,
    CURRENT_TIMESTAMP + INTERVAL '365 days',
    CURRENT_TIMESTAMP
FROM users 
WHERE email = 'test@vrooli.local'
ON CONFLICT (key_hash) DO NOTHING;

-- Add sample audit logs
INSERT INTO audit_logs (
    user_id,
    action,
    resource_type,
    resource_id,
    ip_address,
    user_agent,
    metadata,
    success,
    created_at
)
SELECT 
    u.id,
    'user.login',
    'session',
    uuid_generate_v4()::text,
    '127.0.0.1'::inet,
    'Mozilla/5.0 (Test Browser)',
    '{"method": "password"}'::jsonb,
    true,
    CURRENT_TIMESTAMP - INTERVAL '1 hour'
FROM users u
WHERE u.email = 'admin@vrooli.local';

-- Create additional roles for testing
INSERT INTO roles (name, description, permissions) VALUES
    ('moderator', 'Moderator with content management access', '["read:all", "write:content", "delete:content"]'::jsonb),
    ('developer', 'Developer with API access', '["read:all", "write:api", "manage:keys"]'::jsonb),
    ('viewer', 'Read-only access', '["read:public", "read:own"]'::jsonb)
ON CONFLICT (name) DO NOTHING;

-- Assign roles to users
INSERT INTO user_roles (user_id, role_id, granted_by)
SELECT 
    u.id,
    r.id,
    u.id
FROM users u
CROSS JOIN roles r
WHERE u.email = 'admin@vrooli.local' 
  AND r.name = 'admin'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT 
    u.id,
    r.id
FROM users u
CROSS JOIN roles r
WHERE u.email = 'test@vrooli.local' 
  AND r.name = 'user'
ON CONFLICT DO NOTHING;

INSERT INTO user_roles (user_id, role_id)
SELECT 
    u.id,
    r.id
FROM users u
CROSS JOIN roles r
WHERE u.email = 'demo@vrooli.local' 
  AND r.name = 'guest'
ON CONFLICT DO NOTHING;

-- Log the seed completion
DO $$
DECLARE
    admin_id UUID;
BEGIN
    SELECT id INTO admin_id FROM users WHERE email = 'admin@vrooli.local';
    
    PERFORM log_auth_event(
        admin_id,
        'database.seeded',
        '127.0.0.1'::inet,
        'seed.sql',
        true,
        '{"users": 3, "api_keys": 2, "roles": 6}'::jsonb
    );
END $$;

-- Print summary
DO $$
DECLARE
    user_count INTEGER;
    key_count INTEGER;
    role_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users;
    SELECT COUNT(*) INTO key_count FROM api_keys;
    SELECT COUNT(*) INTO role_count FROM roles;
    
    RAISE NOTICE 'Seed completed: % users, % API keys, % roles', user_count, key_count, role_count;
END $$;