-- SaaS Seed Data: Multi-tenant Users and Organizations
-- Description: Sample tenant and user data for SaaS automation

-- Create tenants/organizations table
CREATE TABLE IF NOT EXISTS tenants (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    domain VARCHAR(255),
    plan VARCHAR(50) DEFAULT 'starter',
    status VARCHAR(50) DEFAULT 'active',
    billing_email VARCHAR(255),
    max_users INTEGER DEFAULT 5,
    max_storage_gb INTEGER DEFAULT 10,
    features JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    trial_ends_at TIMESTAMP WITH TIME ZONE,
    subscription_ends_at TIMESTAMP WITH TIME ZONE
);

-- Create users table with tenant isolation
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    password_hash VARCHAR(255),
    role VARCHAR(50) DEFAULT 'user',
    status VARCHAR(50) DEFAULT 'active',
    email_verified BOOLEAN DEFAULT false,
    last_login_at TIMESTAMP WITH TIME ZONE,
    last_activity_at TIMESTAMP WITH TIME ZONE,
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, email)
);

-- Create subscriptions table
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    plan_id VARCHAR(100) NOT NULL,
    status VARCHAR(50) DEFAULT 'active',
    current_period_start TIMESTAMP WITH TIME ZONE,
    current_period_end TIMESTAMP WITH TIME ZONE,
    cancel_at_period_end BOOLEAN DEFAULT false,
    canceled_at TIMESTAMP WITH TIME ZONE,
    trial_start TIMESTAMP WITH TIME ZONE,
    trial_end TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create usage metrics table for billing
CREATE TABLE IF NOT EXISTS usage_metrics (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    metric_name VARCHAR(100) NOT NULL,
    metric_value INTEGER NOT NULL,
    metric_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(tenant_id, metric_name, metric_date)
);

-- Insert sample tenants
INSERT INTO tenants (name, slug, domain, plan, billing_email, max_users, max_storage_gb, trial_ends_at, subscription_ends_at, features) VALUES
('Acme Corporation', 'acme-corp', 'acme.com', 'enterprise', 'billing@acme.com', 100, 500, NULL, NOW() + INTERVAL '11 months', '{"api_access": true, "custom_branding": true, "priority_support": true}'),
('StartupXYZ', 'startup-xyz', 'startupxyz.com', 'professional', 'admin@startupxyz.com', 25, 100, NULL, NOW() + INTERVAL '10 months', '{"api_access": true, "custom_branding": false, "priority_support": false}'),
('FreelanceCo', 'freelance-co', NULL, 'starter', 'hello@freelanceco.com', 5, 10, NOW() + INTERVAL '14 days', NULL, '{"api_access": false, "custom_branding": false, "priority_support": false}'),
('TechSolutions Ltd', 'tech-solutions', 'techsolutions.co.uk', 'professional', 'accounts@techsolutions.co.uk', 25, 100, NULL, NOW() + INTERVAL '8 months', '{"api_access": true, "custom_branding": true, "priority_support": false}'),
('DevAgency', 'dev-agency', 'devagency.io', 'starter', 'contact@devagency.io', 5, 10, NULL, NOW() + INTERVAL '1 month', '{"api_access": false, "custom_branding": false, "priority_support": false}');

-- Insert sample users
INSERT INTO users (tenant_id, email, first_name, last_name, role, email_verified, last_login_at, preferences) VALUES
-- Acme Corporation users
(1, 'john.doe@acme.com', 'John', 'Doe', 'admin', true, NOW() - INTERVAL '1 hour', '{"theme": "dark", "notifications": {"email": true, "push": false}}'),
(1, 'jane.smith@acme.com', 'Jane', 'Smith', 'manager', true, NOW() - INTERVAL '3 hours', '{"theme": "light", "notifications": {"email": true, "push": true}}'),
(1, 'bob.wilson@acme.com', 'Bob', 'Wilson', 'user', true, NOW() - INTERVAL '1 day', '{"theme": "light", "notifications": {"email": false, "push": false}}'),

-- StartupXYZ users
(2, 'founder@startupxyz.com', 'Alice', 'Johnson', 'admin', true, NOW() - INTERVAL '30 minutes', '{"theme": "dark", "notifications": {"email": true, "push": true}}'),
(2, 'dev@startupxyz.com', 'Mike', 'Chen', 'user', true, NOW() - INTERVAL '2 hours', '{"theme": "light", "notifications": {"email": true, "push": false}}'),

-- FreelanceCo users (trial)
(3, 'sarah@freelanceco.com', 'Sarah', 'Brown', 'admin', true, NOW() - INTERVAL '4 hours', '{"theme": "light", "notifications": {"email": true, "push": false}}'),

-- TechSolutions users
(4, 'director@techsolutions.co.uk', 'David', 'Taylor', 'admin', true, NOW() - INTERVAL '6 hours', '{"theme": "dark", "notifications": {"email": true, "push": true}}'),
(4, 'project.manager@techsolutions.co.uk', 'Emma', 'Davis', 'manager', true, NOW() - INTERVAL '1 day', '{"theme": "light", "notifications": {"email": true, "push": false}}'),

-- DevAgency users
(5, 'owner@devagency.io', 'Carlos', 'Rodriguez', 'admin', true, NOW() - INTERVAL '2 days', '{"theme": "dark", "notifications": {"email": false, "push": false}}');

-- Insert sample subscriptions
INSERT INTO subscriptions (tenant_id, plan_id, status, current_period_start, current_period_end, trial_start, trial_end) VALUES
(1, 'enterprise_monthly', 'active', NOW() - INTERVAL '15 days', NOW() + INTERVAL '15 days', NULL, NULL),
(2, 'professional_monthly', 'active', NOW() - INTERVAL '10 days', NOW() + INTERVAL '20 days', NULL, NULL),
(3, 'starter_monthly', 'trialing', NULL, NULL, NOW() - INTERVAL '2 days', NOW() + INTERVAL '12 days'),
(4, 'professional_annual', 'active', NOW() - INTERVAL '60 days', NOW() + INTERVAL '305 days', NULL, NULL),
(5, 'starter_monthly', 'active', NOW() - INTERVAL '5 days', NOW() + INTERVAL '25 days', NULL, NULL);

-- Insert sample usage metrics
INSERT INTO usage_metrics (tenant_id, metric_name, metric_value, metric_date) VALUES
-- Current month usage
(1, 'api_calls', 15420, CURRENT_DATE),
(1, 'storage_used_gb', 245, CURRENT_DATE),
(1, 'active_users', 87, CURRENT_DATE),
(2, 'api_calls', 3200, CURRENT_DATE),
(2, 'storage_used_gb', 45, CURRENT_DATE),
(2, 'active_users', 18, CURRENT_DATE),
(3, 'api_calls', 150, CURRENT_DATE),
(3, 'storage_used_gb', 2, CURRENT_DATE),
(3, 'active_users', 1, CURRENT_DATE),
(4, 'api_calls', 8900, CURRENT_DATE),
(4, 'storage_used_gb', 89, CURRENT_DATE),
(4, 'active_users', 23, CURRENT_DATE),
(5, 'api_calls', 450, CURRENT_DATE),
(5, 'storage_used_gb', 8, CURRENT_DATE),
(5, 'active_users', 3, CURRENT_DATE),

-- Previous month usage for trend analysis
(1, 'api_calls', 14200, CURRENT_DATE - INTERVAL '1 month'),
(1, 'storage_used_gb', 230, CURRENT_DATE - INTERVAL '1 month'),
(1, 'active_users', 82, CURRENT_DATE - INTERVAL '1 month'),
(2, 'api_calls', 2800, CURRENT_DATE - INTERVAL '1 month'),
(2, 'storage_used_gb', 38, CURRENT_DATE - INTERVAL '1 month'),
(2, 'active_users', 15, CURRENT_DATE - INTERVAL '1 month');