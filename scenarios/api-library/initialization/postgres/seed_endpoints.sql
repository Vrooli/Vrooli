-- Seed endpoint data for API Library
-- This adds sample endpoints for APIs to demonstrate code generation capabilities

-- OpenAI endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema, response_schema) 
SELECT id, '/chat/completions', 'POST', 'Create a chat completion', 3500, 'minute', true,
    '{"type": "object", "properties": {"model": {"type": "string"}, "messages": {"type": "array"}, "temperature": {"type": "number"}}}',
    '{"type": "object", "properties": {"choices": {"type": "array"}, "usage": {"type": "object"}}}'
FROM apis WHERE name = 'OpenAI';

INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth)
SELECT id, '/embeddings', 'POST', 'Create embeddings for text', 3000, 'minute', true
FROM apis WHERE name = 'OpenAI';

INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth)
SELECT id, '/images/generations', 'POST', 'Generate images from text', 50, 'minute', true
FROM apis WHERE name = 'OpenAI';

-- Stripe endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema, response_schema)
SELECT id, '/v1/charges', 'POST', 'Create a charge', 100, 'second', true,
    '{"type": "object", "properties": {"amount": {"type": "integer"}, "currency": {"type": "string"}, "source": {"type": "string"}}}',
    '{"type": "object", "properties": {"id": {"type": "string"}, "amount": {"type": "integer"}, "status": {"type": "string"}}}'
FROM apis WHERE name = 'Stripe';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v1/customers', 'POST', 'Create a customer', true
FROM apis WHERE name = 'Stripe';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v1/customers/{id}', 'GET', 'Retrieve a customer', true
FROM apis WHERE name = 'Stripe';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v1/refunds', 'POST', 'Create a refund', true
FROM apis WHERE name = 'Stripe';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v1/subscriptions', 'POST', 'Create a subscription', true
FROM apis WHERE name = 'Stripe';

-- Twilio SMS endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema)
SELECT id, '/2010-04-01/Accounts/{AccountSid}/Messages.json', 'POST', 'Send an SMS message', 100, 'second', true,
    '{"type": "object", "properties": {"To": {"type": "string"}, "From": {"type": "string"}, "Body": {"type": "string"}}}'
FROM apis WHERE name = 'Twilio SMS';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/2010-04-01/Accounts/{AccountSid}/Messages/{MessageSid}.json', 'GET', 'Retrieve message details', true
FROM apis WHERE name = 'Twilio SMS';

-- SendGrid endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema)
SELECT id, '/mail/send', 'POST', 'Send an email', 100, 'second', true,
    '{"type": "object", "properties": {"personalizations": {"type": "array"}, "from": {"type": "object"}, "subject": {"type": "string"}, "content": {"type": "array"}}}'
FROM apis WHERE name = 'SendGrid';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/templates', 'POST', 'Create email template', true
FROM apis WHERE name = 'SendGrid';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/stats', 'GET', 'Get email statistics', true
FROM apis WHERE name = 'SendGrid';

-- Slack endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema)
SELECT id, '/chat.postMessage', 'POST', 'Post a message to a channel', 1, 'second', true,
    '{"type": "object", "properties": {"channel": {"type": "string"}, "text": {"type": "string"}, "blocks": {"type": "array"}}}'
FROM apis WHERE name = 'Slack';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/conversations.list', 'GET', 'List conversations', true
FROM apis WHERE name = 'Slack';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/users.list', 'GET', 'List users in workspace', true
FROM apis WHERE name = 'Slack';

-- AWS S3 endpoints
INSERT INTO endpoints (api_id, path, method, description, requires_auth, request_schema)
SELECT id, '/{bucket}/{key}', 'PUT', 'Upload an object', true,
    '{"type": "object", "properties": {"Body": {"type": "string"}, "ContentType": {"type": "string"}}}'
FROM apis WHERE name = 'AWS S3';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/{bucket}/{key}', 'GET', 'Download an object', true
FROM apis WHERE name = 'AWS S3';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/{bucket}/{key}', 'DELETE', 'Delete an object', true
FROM apis WHERE name = 'AWS S3';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/{bucket}/', 'GET', 'List objects in bucket', true
FROM apis WHERE name = 'AWS S3';

-- Anthropic Claude endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema, response_schema)
SELECT id, '/v1/messages', 'POST', 'Send a message to Claude', 1000, 'minute', true,
    '{"type": "object", "properties": {"model": {"type": "string"}, "messages": {"type": "array"}, "max_tokens": {"type": "integer"}}}',
    '{"type": "object", "properties": {"content": {"type": "array"}, "usage": {"type": "object"}}}'
FROM apis WHERE name = 'Anthropic Claude';

-- PayPal endpoints
INSERT INTO endpoints (api_id, path, method, description, requires_auth, request_schema)
SELECT id, '/v2/checkout/orders', 'POST', 'Create an order', true,
    '{"type": "object", "properties": {"intent": {"type": "string"}, "purchase_units": {"type": "array"}}}'
FROM apis WHERE name = 'PayPal';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v2/checkout/orders/{id}/capture', 'POST', 'Capture payment for an order', true
FROM apis WHERE name = 'PayPal';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v2/payments/refunds/{id}', 'GET', 'Show refund details', true
FROM apis WHERE name = 'PayPal';

-- Google Analytics endpoints
INSERT INTO endpoints (api_id, path, method, description, requires_auth, request_schema)
SELECT id, '/v4/reports:batchGet', 'POST', 'Get analytics reports', true,
    '{"type": "object", "properties": {"reportRequests": {"type": "array"}}}'
FROM apis WHERE name = 'Google Analytics';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v1/accounts', 'GET', 'List accounts', true
FROM apis WHERE name = 'Google Analytics';

-- Firebase endpoints
INSERT INTO endpoints (api_id, path, method, description, requires_auth, request_schema)
SELECT id, '/v1/projects/{project}/databases/(default)/documents', 'POST', 'Create a document', true,
    '{"type": "object", "properties": {"fields": {"type": "object"}}}'
FROM apis WHERE name = 'Firebase';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/v1/projects/{project}/databases/(default)/documents/{collection}', 'GET', 'List documents', true
FROM apis WHERE name = 'Firebase';

-- Hugging Face endpoints
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema)
SELECT id, '/models/{model_id}', 'POST', 'Run inference on a model', 1000, 'hour', true,
    '{"type": "object", "properties": {"inputs": {"type": "string"}, "parameters": {"type": "object"}}}'
FROM apis WHERE name = 'Hugging Face';

-- Mixpanel endpoints
INSERT INTO endpoints (api_id, path, method, description, requires_auth, request_schema)
SELECT id, '/track', 'POST', 'Track an event', true,
    '{"type": "object", "properties": {"event": {"type": "string"}, "properties": {"type": "object"}}}'
FROM apis WHERE name = 'Mixpanel';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/engage', 'POST', 'Update user profile', true
FROM apis WHERE name = 'Mixpanel';

INSERT INTO endpoints (api_id, path, method, description, requires_auth)
SELECT id, '/reports/funnel', 'GET', 'Get funnel report', true
FROM apis WHERE name = 'Mixpanel';