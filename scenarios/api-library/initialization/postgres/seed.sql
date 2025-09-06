-- Seed data for API Library
-- Initial set of commonly used APIs to bootstrap the system

-- Communication APIs
INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities) VALUES
('Twilio SMS', 'Twilio', 'Send and receive SMS messages globally', 'https://api.twilio.com', 'https://www.twilio.com/docs/sms', 'communication', 'basic', 
 ARRAY['sms', 'messaging', 'communication'], 
 ARRAY['send_sms', 'receive_sms', 'phone_numbers', 'messaging']),

('SendGrid', 'SendGrid', 'Email delivery and management platform', 'https://api.sendgrid.com/v3', 'https://docs.sendgrid.com', 'communication', 'api_key',
 ARRAY['email', 'transactional', 'marketing'],
 ARRAY['send_email', 'email_templates', 'email_analytics']),

('Slack', 'Slack', 'Team communication and collaboration', 'https://slack.com/api', 'https://api.slack.com', 'communication', 'oauth2',
 ARRAY['chat', 'collaboration', 'notifications'],
 ARRAY['send_message', 'create_channel', 'file_sharing']);

-- AI/ML APIs
INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities) VALUES
('OpenAI', 'OpenAI', 'Advanced AI models for text, code, and image generation', 'https://api.openai.com/v1', 'https://platform.openai.com/docs', 'ai', 'bearer',
 ARRAY['ai', 'gpt', 'language-model', 'generation'],
 ARRAY['text_generation', 'code_generation', 'embeddings', 'image_generation']),

('Anthropic Claude', 'Anthropic', 'Claude AI assistant API', 'https://api.anthropic.com', 'https://docs.anthropic.com', 'ai', 'api_key',
 ARRAY['ai', 'claude', 'language-model', 'assistant'],
 ARRAY['text_generation', 'conversation', 'analysis', 'code_generation']),

('Hugging Face', 'Hugging Face', 'Access thousands of ML models', 'https://api-inference.huggingface.co', 'https://huggingface.co/docs/api-inference', 'ai', 'bearer',
 ARRAY['ai', 'ml', 'models', 'inference'],
 ARRAY['text_generation', 'classification', 'translation', 'embeddings']);

-- Payment APIs
INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities) VALUES
('Stripe', 'Stripe', 'Payment processing and billing management', 'https://api.stripe.com', 'https://stripe.com/docs/api', 'payment', 'bearer',
 ARRAY['payment', 'billing', 'subscription'],
 ARRAY['process_payment', 'manage_subscriptions', 'invoicing', 'refunds']),

('PayPal', 'PayPal', 'Online payment processing', 'https://api.paypal.com', 'https://developer.paypal.com/docs/api', 'payment', 'oauth2',
 ARRAY['payment', 'checkout', 'invoicing'],
 ARRAY['process_payment', 'refunds', 'invoicing', 'recurring_payments']);

-- Data/Analytics APIs
INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities) VALUES
('Google Analytics', 'Google', 'Web and app analytics', 'https://analyticsreporting.googleapis.com', 'https://developers.google.com/analytics', 'analytics', 'oauth2',
 ARRAY['analytics', 'tracking', 'metrics'],
 ARRAY['page_views', 'user_tracking', 'conversion_tracking', 'reporting']),

('Mixpanel', 'Mixpanel', 'Product analytics platform', 'https://mixpanel.com/api', 'https://developer.mixpanel.com/docs', 'analytics', 'api_key',
 ARRAY['analytics', 'events', 'user-behavior'],
 ARRAY['event_tracking', 'user_profiles', 'funnels', 'retention_analysis']);

-- Storage/Database APIs
INSERT INTO apis (name, provider, description, base_url, documentation_url, category, auth_type, tags, capabilities) VALUES
('AWS S3', 'Amazon', 'Object storage service', 'https://s3.amazonaws.com', 'https://docs.aws.amazon.com/s3', 'storage', 'custom',
 ARRAY['storage', 'files', 'backup', 'cdn'],
 ARRAY['file_upload', 'file_download', 'file_management', 'versioning']),

('Firebase', 'Google', 'Backend-as-a-service platform', 'https://firebase.googleapis.com', 'https://firebase.google.com/docs', 'backend', 'bearer',
 ARRAY['database', 'auth', 'storage', 'hosting'],
 ARRAY['realtime_database', 'authentication', 'file_storage', 'cloud_functions']);

-- Add some pricing tiers
INSERT INTO pricing_tiers (api_id, name, price_per_request, free_tier_requests, monthly_cost) 
SELECT id, 'Free', 0, 100, 0 FROM apis WHERE name = 'SendGrid';

INSERT INTO pricing_tiers (api_id, name, price_per_request, free_tier_requests, monthly_cost) 
SELECT id, 'Pro', 0.001, 40000, 15 FROM apis WHERE name = 'SendGrid';

INSERT INTO pricing_tiers (api_id, name, price_per_request, monthly_cost) 
SELECT id, 'GPT-4', 0.03, 0 FROM apis WHERE name = 'OpenAI';

INSERT INTO pricing_tiers (api_id, name, price_per_request, monthly_cost) 
SELECT id, 'GPT-3.5', 0.002, 0 FROM apis WHERE name = 'OpenAI';

-- Add some helpful notes
INSERT INTO notes (api_id, content, type, created_by) 
SELECT id, 'Rate limits are strict on free tier. Consider caching responses for repeated queries.', 'tip', 'system' 
FROM apis WHERE name = 'OpenAI';

INSERT INTO notes (api_id, content, type, created_by) 
SELECT id, 'Webhook signature verification is mandatory for production use.', 'warning', 'system' 
FROM apis WHERE name = 'Stripe';

INSERT INTO notes (api_id, content, type, created_by) 
SELECT id, 'Phone number verification required before sending SMS in most countries.', 'gotcha', 'system' 
FROM apis WHERE name = 'Twilio SMS';

INSERT INTO notes (api_id, content, type, created_by) 
SELECT id, 'OAuth2 flow can be complex. Consider using their SDK for easier integration.', 'tip', 'system' 
FROM apis WHERE name = 'Slack';

-- Add some API relationships
INSERT INTO api_relationships (api_id, related_api_id, relationship_type, notes)
SELECT a1.id, a2.id, 'alternative', 'Both provide email delivery services'
FROM apis a1, apis a2 
WHERE a1.name = 'SendGrid' AND a2.provider = 'Mailgun'
AND a2.id IS NOT NULL;

-- Mark some APIs as configured (for demo purposes)
INSERT INTO api_credentials (api_id, is_configured, environment, configuration_notes)
SELECT id, true, 'development', 'Configured with test API key'
FROM apis WHERE name IN ('OpenAI', 'SendGrid');