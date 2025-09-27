-- Fix data consistency between Qdrant and PostgreSQL
-- These IDs are what Qdrant currently has and what the tests expect

-- First, delete any existing entries with these names to avoid conflicts
DELETE FROM apis WHERE name IN ('OpenAI API', 'Stripe API', 'SendGrid API');

-- Insert APIs with specific IDs that match what's in Qdrant
INSERT INTO apis (
    id, name, provider, description, base_url, documentation_url, 
    category, status, auth_type, tags, capabilities, version
) VALUES 
(
    '8beb3bcf-1a33-4e85-8267-154b9f59b76a'::uuid,
    'OpenAI API',
    'OpenAI',
    'Access to GPT models for text generation, embeddings, and more',
    'https://api.openai.com',
    'https://platform.openai.com/docs',
    'AI/ML',
    'active',
    'bearer',
    ARRAY['ai', 'ml', 'nlp', 'gpt'],
    ARRAY['text-generation', 'embeddings', 'chat'],
    '1.0.0'
),
(
    '8883b6cf-f5e8-4545-89f3-00c9f7c8f636'::uuid,
    'Stripe API',
    'Stripe',
    'Payment processing and financial services',
    'https://api.stripe.com',
    'https://stripe.com/docs',
    'Payment',
    'active',
    'bearer',
    ARRAY['payment', 'finance', 'billing'],
    ARRAY['payment-processing', 'subscriptions', 'invoicing'],
    '2023.10.16'
),
(
    '63356933-88d6-4b71-bde8-9ea6958a6aae'::uuid,
    'SendGrid API',
    'SendGrid',
    'Email sending and delivery service',
    'https://api.sendgrid.com',
    'https://docs.sendgrid.com',
    'Communication',
    'active',
    'api_key',
    ARRAY['email', 'messaging', 'transactional'],
    ARRAY['email-sending', 'email-templates', 'analytics'],
    'v3'
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    provider = EXCLUDED.provider,
    description = EXCLUDED.description,
    base_url = EXCLUDED.base_url,
    documentation_url = EXCLUDED.documentation_url,
    category = EXCLUDED.category,
    status = EXCLUDED.status,
    auth_type = EXCLUDED.auth_type,
    tags = EXCLUDED.tags,
    capabilities = EXCLUDED.capabilities,
    version = EXCLUDED.version,
    updated_at = CURRENT_TIMESTAMP;

-- Add some sample notes for testing
INSERT INTO notes (api_id, content, type, created_by) VALUES
(
    '8883b6cf-f5e8-4545-89f3-00c9f7c8f636'::uuid,
    'Remember to use webhook endpoints for production environments',
    'tip',
    'system'
),
(
    '8beb3bcf-1a33-4e85-8267-154b9f59b76a'::uuid,
    'Rate limits apply: 3 RPM for free tier, 3500 RPM for paid tier',
    'warning',
    'system'
)
ON CONFLICT DO NOTHING;

-- Add sample pricing tiers
INSERT INTO pricing_tiers (api_id, name, price_per_request, monthly_cost, free_tier_requests) VALUES
(
    '8883b6cf-f5e8-4545-89f3-00c9f7c8f636'::uuid,
    'Standard',
    0.029,
    0,
    0
),
(
    '8beb3bcf-1a33-4e85-8267-154b9f59b76a'::uuid,
    'GPT-4',
    0.03,
    0,
    0
)
ON CONFLICT DO NOTHING;