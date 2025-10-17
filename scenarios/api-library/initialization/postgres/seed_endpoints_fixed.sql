-- Fixed seed endpoint data for API Library
-- Using direct IDs from the database

-- OpenAI endpoints (ID: 861e4a10-5efc-4eb6-b793-140dd0093c43)
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema, response_schema) 
VALUES 
('861e4a10-5efc-4eb6-b793-140dd0093c43', '/embeddings', 'POST', 'Create embeddings for text', 3000, 'minute', true, NULL, NULL),
('861e4a10-5efc-4eb6-b793-140dd0093c43', '/images/generations', 'POST', 'Generate images from text', 50, 'minute', true, NULL, NULL),
('861e4a10-5efc-4eb6-b793-140dd0093c43', '/models', 'GET', 'List available models', 100, 'minute', true, NULL, NULL);

-- Stripe endpoints (ID: 53a6dc05-a562-4d40-bd6d-11c454917951)
INSERT INTO endpoints (api_id, path, method, description, rate_limit_requests, rate_limit_period, requires_auth, request_schema, response_schema)
VALUES
('53a6dc05-a562-4d40-bd6d-11c454917951', '/v1/charges', 'POST', 'Create a charge', 100, 'second', true,
    '{"type": "object", "properties": {"amount": {"type": "integer"}, "currency": {"type": "string"}, "source": {"type": "string"}}}',
    '{"type": "object", "properties": {"id": {"type": "string"}, "amount": {"type": "integer"}, "status": {"type": "string"}}}'),
('53a6dc05-a562-4d40-bd6d-11c454917951', '/v1/customers', 'POST', 'Create a customer', NULL, NULL, true, NULL, NULL),
('53a6dc05-a562-4d40-bd6d-11c454917951', '/v1/customers/{id}', 'GET', 'Retrieve a customer', NULL, NULL, true, NULL, NULL),
('53a6dc05-a562-4d40-bd6d-11c454917951', '/v1/refunds', 'POST', 'Create a refund', NULL, NULL, true, NULL, NULL),
('53a6dc05-a562-4d40-bd6d-11c454917951', '/v1/subscriptions', 'POST', 'Create a subscription', NULL, NULL, true, NULL, NULL),
('53a6dc05-a562-4d40-bd6d-11c454917951', '/v1/payment_intents', 'POST', 'Create a payment intent', 100, 'second', true, NULL, NULL);

-- AWS S3 endpoints (ID: 26bfc1a7-7469-48ea-8b72-899f98cdee2b)
INSERT INTO endpoints (api_id, path, method, description, requires_auth, request_schema)
VALUES
('26bfc1a7-7469-48ea-8b72-899f98cdee2b', '/{bucket}/{key}', 'PUT', 'Upload an object', true,
    '{"type": "object", "properties": {"Body": {"type": "string"}, "ContentType": {"type": "string"}}}'),
('26bfc1a7-7469-48ea-8b72-899f98cdee2b', '/{bucket}/{key}', 'GET', 'Download an object', true, NULL),
('26bfc1a7-7469-48ea-8b72-899f98cdee2b', '/{bucket}/{key}', 'DELETE', 'Delete an object', true, NULL),
('26bfc1a7-7469-48ea-8b72-899f98cdee2b', '/{bucket}/', 'GET', 'List objects in bucket', true, NULL);