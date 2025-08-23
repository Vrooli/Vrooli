-- SaaS Seed Data: Features and Billing
-- Description: Sample feature flags and billing data for SaaS automation

-- Create feature flags table
CREATE TABLE IF NOT EXISTS feature_flags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    enabled BOOLEAN DEFAULT false,
    rollout_percentage INTEGER DEFAULT 0,
    conditions JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create tenant feature overrides
CREATE TABLE IF NOT EXISTS tenant_features (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    feature_flag_id INTEGER REFERENCES feature_flags(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL,
    enabled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    enabled_by INTEGER REFERENCES users(id),
    UNIQUE(tenant_id, feature_flag_id)
);

-- Create billing events table
CREATE TABLE IF NOT EXISTS billing_events (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    amount DECIMAL(10,2),
    currency VARCHAR(3) DEFAULT 'USD',
    description TEXT,
    metadata JSONB DEFAULT '{}',
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create support tickets table
CREATE TABLE IF NOT EXISTS support_tickets (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    user_id INTEGER REFERENCES users(id),
    subject VARCHAR(255) NOT NULL,
    description TEXT,
    status VARCHAR(50) DEFAULT 'open',
    priority VARCHAR(20) DEFAULT 'medium',
    category VARCHAR(100),
    assigned_to VARCHAR(255),
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample feature flags
INSERT INTO feature_flags (name, description, enabled, rollout_percentage) VALUES
('api_v2', 'Access to API version 2 endpoints', true, 100),
('advanced_analytics', 'Advanced analytics dashboard', true, 75),
('custom_integrations', 'Custom third-party integrations', false, 25),
('white_label', 'White label branding options', true, 50),
('priority_support', 'Priority customer support', true, 100),
('bulk_operations', 'Bulk data operations', true, 90),
('advanced_exports', 'Advanced data export formats', false, 10),
('mobile_app_beta', 'Beta mobile application access', false, 5),
('ai_insights', 'AI-powered business insights', true, 30),
('sso_integration', 'Single Sign-On integration', true, 80);

-- Insert tenant-specific feature overrides
INSERT INTO tenant_features (tenant_id, feature_flag_id, enabled, enabled_by) VALUES
-- Acme Corporation (Enterprise) - gets all features
(1, 1, true, 1), -- api_v2
(1, 2, true, 1), -- advanced_analytics
(1, 3, true, 1), -- custom_integrations
(1, 4, true, 1), -- white_label
(1, 5, true, 1), -- priority_support
(1, 10, true, 1), -- sso_integration

-- StartupXYZ (Professional) - gets most features
(2, 1, true, 4), -- api_v2
(2, 2, true, 4), -- advanced_analytics
(2, 6, true, 4), -- bulk_operations
(2, 10, true, 4), -- sso_integration

-- TechSolutions (Professional) - similar to StartupXYZ
(4, 1, true, 7), -- api_v2
(4, 2, true, 7), -- advanced_analytics
(4, 4, true, 7), -- white_label
(4, 6, true, 7), -- bulk_operations

-- FreelanceCo (Trial) - limited features
(3, 1, false, NULL), -- api_v2 disabled
(3, 2, false, NULL); -- advanced_analytics disabled

-- Insert sample billing events
INSERT INTO billing_events (tenant_id, event_type, amount, description, metadata, processed_at) VALUES
-- Acme Corporation billing
(1, 'subscription_charge', 299.00, 'Enterprise monthly subscription', '{"plan": "enterprise_monthly", "period": "2025-07"}', NOW() - INTERVAL '15 days'),
(1, 'usage_charge', 45.50, 'Additional API calls over limit', '{"api_calls": 5000, "rate": 0.0091}', NOW() - INTERVAL '10 days'),

-- StartupXYZ billing
(2, 'subscription_charge', 99.00, 'Professional monthly subscription', '{"plan": "professional_monthly", "period": "2025-07"}', NOW() - INTERVAL '10 days'),

-- TechSolutions billing (annual)
(4, 'subscription_charge', 999.00, 'Professional annual subscription', '{"plan": "professional_annual", "period": "2024-2025"}', NOW() - INTERVAL '60 days'),

-- DevAgency billing
(5, 'subscription_charge', 29.00, 'Starter monthly subscription', '{"plan": "starter_monthly", "period": "2025-07"}', NOW() - INTERVAL '5 days'),

-- Failed payment example
(5, 'payment_failed', 29.00, 'Failed payment for starter subscription', '{"reason": "insufficient_funds", "retry_count": 1}', NULL);

-- Insert sample support tickets
INSERT INTO support_tickets (tenant_id, user_id, subject, description, status, priority, category, assigned_to) VALUES
(1, 1, 'API rate limit increase request', 'We need to increase our API rate limits for our growing user base', 'open', 'high', 'billing', 'support@saas-platform.com'),
(1, 2, 'Custom integration help needed', 'Need assistance setting up Salesforce integration', 'in_progress', 'medium', 'technical', 'tech@saas-platform.com'),
(2, 4, 'Billing discrepancy', 'Our last invoice seems to have incorrect usage charges', 'resolved', 'high', 'billing', 'billing@saas-platform.com'),
(3, 6, 'Trial extension request', 'Can we extend our trial period by another week?', 'open', 'low', 'sales', 'sales@saas-platform.com'),
(4, 7, 'Data export not working', 'Advanced export feature is not generating complete CSV files', 'open', 'medium', 'technical', 'tech@saas-platform.com'),
(5, 9, 'Account upgrade question', 'What features do we get if we upgrade to Professional?', 'resolved', 'low', 'sales', 'sales@saas-platform.com');

-- Update resolved tickets
UPDATE support_tickets SET 
    resolved_at = NOW() - INTERVAL '2 days',
    status = 'resolved'
WHERE id IN (3, 6);

-- Create webhook events table for automation
CREATE TABLE IF NOT EXISTS webhook_events (
    id SERIAL PRIMARY KEY,
    tenant_id INTEGER REFERENCES tenants(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    delivered BOOLEAN DEFAULT false,
    delivery_attempts INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP WITH TIME ZONE,
    webhook_url VARCHAR(500),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Insert sample webhook events
INSERT INTO webhook_events (tenant_id, event_type, payload, delivered, delivery_attempts, webhook_url) VALUES
(1, 'user.created', '{"user_id": 3, "email": "bob.wilson@acme.com", "tenant_id": 1}', true, 1, 'https://acme.com/webhooks/users'),
(2, 'subscription.updated', '{"tenant_id": 2, "plan_id": "professional_monthly", "status": "active"}', true, 1, 'https://startupxyz.com/webhooks/billing'),
(3, 'trial.expiring', '{"tenant_id": 3, "days_remaining": 14, "trial_end": "2025-08-14"}', false, 3, 'https://freelanceco.com/webhooks/billing'),
(4, 'usage.threshold_exceeded', '{"tenant_id": 4, "metric": "api_calls", "threshold": 5000, "current": 8900}', true, 1, 'https://techsolutions.co.uk/webhooks/usage');