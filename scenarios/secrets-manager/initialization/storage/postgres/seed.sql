-- Secrets Manager Seed Data
-- Initial secret requirements for common Vrooli resources

-- PostgreSQL resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('postgres', 'POSTGRES_HOST', 'env_var', true, 'PostgreSQL server hostname', '^[a-zA-Z0-9.-]+$', 'https://hub.docker.com/_/postgres'),
('postgres', 'POSTGRES_PORT', 'env_var', false, 'PostgreSQL server port', '^[0-9]+$', 'https://hub.docker.com/_/postgres'),
('postgres', 'POSTGRES_DB', 'env_var', true, 'PostgreSQL database name', '^[a-zA-Z0-9_]+$', 'https://hub.docker.com/_/postgres'),
('postgres', 'POSTGRES_USER', 'env_var', true, 'PostgreSQL username', '^[a-zA-Z0-9_]+$', 'https://hub.docker.com/_/postgres'),
('postgres', 'POSTGRES_PASSWORD', 'password', true, 'PostgreSQL password', '.{8,}', 'https://hub.docker.com/_/postgres'),
('postgres', 'PGPASSWORD', 'password', false, 'PostgreSQL password for psql client', '.{8,}', 'https://www.postgresql.org/docs/current/libpq-envars.html')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Classification defaults for known infrastructure resources
UPDATE resource_secrets
SET classification = CASE
        WHEN resource_name IN ('postgres', 'vault', 'redis', 'minio') THEN 'infrastructure'
        WHEN resource_name IN ('n8n', 'ollama') THEN 'service'
        ELSE classification
    END,
    owner_team = CASE
        WHEN resource_name IN ('postgres', 'vault') THEN 'Platform Infra'
        WHEN resource_name IN ('n8n', 'ollama') THEN 'AI & Automation'
        ELSE owner_team
    END,
    owner_contact = CASE
        WHEN resource_name IN ('postgres', 'vault') THEN 'platform@vrooli.dev'
        WHEN resource_name IN ('n8n', 'ollama') THEN 'automation@vrooli.dev'
        ELSE owner_contact
    END
WHERE classification IS DISTINCT FROM 'infrastructure'
   OR owner_team IS NULL;

-- Deployment strategy scaffolding for Tier coverage
INSERT INTO secret_deployment_strategies (resource_secret_id, tier, handling_strategy, requires_user_input, prompt_label, prompt_description, generator_template, bundle_hints)
SELECT rs.id, 'tier-1-local', 'strip', false, NULL, NULL, NULL, '{"reason":"Managed by Tier 1 runtime"}'::jsonb
FROM resource_secrets rs
WHERE rs.resource_name = 'postgres' AND rs.secret_key IN ('POSTGRES_HOST', 'POSTGRES_PORT')
ON CONFLICT DO NOTHING;

INSERT INTO secret_deployment_strategies (resource_secret_id, tier, handling_strategy, requires_user_input, prompt_label, prompt_description, generator_template, bundle_hints)
SELECT rs.id, 'tier-2-desktop', 'generate', false, NULL, 'Generate embedded SQLite credentials during packaging', '{"type":"file","path":"config/db.json"}'::jsonb, '{"replace":"sqlite"}'::jsonb
FROM resource_secrets rs
WHERE rs.resource_name = 'postgres' AND rs.secret_key IN ('POSTGRES_USER', 'POSTGRES_PASSWORD')
ON CONFLICT DO NOTHING;

INSERT INTO secret_deployment_strategies (resource_secret_id, tier, handling_strategy, requires_user_input, prompt_label, prompt_description)
SELECT rs.id, 'tier-2-desktop', 'prompt', true, 'Desktop API host', 'Let the operator select a Tier 1 server or Cloudflare tunnel endpoint before pairing.'
FROM resource_secrets rs
WHERE rs.resource_name = 'vault' AND rs.secret_key = 'VAULT_ADDR'
ON CONFLICT DO NOTHING;

INSERT INTO secret_deployment_strategies (resource_secret_id, tier, handling_strategy, requires_user_input, prompt_label, prompt_description)
SELECT rs.id, 'tier-4-saas', 'delegate', false, 'Cloud Vault Token', 'Use provider-managed secret store (AWS Secrets Manager, DO App Platform env vars).'
FROM resource_secrets rs
WHERE rs.resource_name = 'vault' AND rs.secret_key = 'VAULT_TOKEN'
ON CONFLICT DO NOTHING;

-- Vault resource secrets  
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('vault', 'VAULT_ADDR', 'env_var', true, 'Vault server address', '^https?://[a-zA-Z0-9.-]+(:[0-9]+)?$', 'https://www.vaultproject.io/docs/commands#vault_addr'),
('vault', 'VAULT_TOKEN', 'token', false, 'Vault authentication token', '^[a-zA-Z0-9.-_]+$', 'https://www.vaultproject.io/docs/concepts/tokens'),
('vault', 'VAULT_NAMESPACE', 'env_var', false, 'Vault namespace (Enterprise)', '^[a-zA-Z0-9/_-]+$', 'https://www.vaultproject.io/docs/enterprise/namespaces'),
('vault', 'VAULT_SKIP_VERIFY', 'env_var', false, 'Skip TLS verification (dev only)', '^(true|false)$', 'https://www.vaultproject.io/docs/commands#vault_skip_verify'),
('vault', 'VAULT_DEV_ROOT_TOKEN_ID', 'token', false, 'Development mode root token', '^[a-zA-Z0-9.-_]+$', 'https://www.vaultproject.io/docs/concepts/dev-server')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Redis resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES  
('redis', 'REDIS_HOST', 'env_var', false, 'Redis server hostname', '^[a-zA-Z0-9.-]+$', 'https://hub.docker.com/_/redis'),
('redis', 'REDIS_PORT', 'env_var', false, 'Redis server port', '^[0-9]+$', 'https://hub.docker.com/_/redis'),  
('redis', 'REDIS_PASSWORD', 'password', false, 'Redis authentication password', '.+', 'https://redis.io/commands/auth'),
('redis', 'REDIS_DATABASE', 'env_var', false, 'Redis database number', '^[0-9]+$', 'https://redis.io/commands/select')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- N8N resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('n8n', 'N8N_ENCRYPTION_KEY', 'credential', true, 'N8N database encryption key', '.{32,}', 'https://docs.n8n.io/hosting/configuration/'),
('n8n', 'N8N_EMAIL', 'credential', false, 'N8N admin email for authentication', '^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$', 'https://docs.n8n.io/hosting/configuration/'),
('n8n', 'N8N_PASSWORD', 'password', false, 'N8N admin password', '.{8,}', 'https://docs.n8n.io/hosting/configuration/'),
('n8n', 'WEBHOOK_URL', 'env_var', false, 'N8N webhook base URL', '^https?://[a-zA-Z0-9.-]+(:[0-9]+)?', 'https://docs.n8n.io/hosting/configuration/'),
('n8n', 'N8N_JWT_SECRET', 'credential', false, 'JWT token secret for authentication', '.{32,}', 'https://docs.n8n.io/hosting/configuration/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Ollama resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('ollama', 'OLLAMA_HOST', 'env_var', false, 'Ollama server host', '^[a-zA-Z0-9.-]+$', 'https://ollama.ai/docs/'),
('ollama', 'OLLAMA_PORT', 'env_var', false, 'Ollama server port', '^[0-9]+$', 'https://ollama.ai/docs/'),
('ollama', 'OLLAMA_API_BASE_URL', 'env_var', false, 'Ollama API base URL', '^https?://[a-zA-Z0-9.-]+(:[0-9]+)?', 'https://ollama.ai/docs/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Minio resource secrets  
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('minio', 'MINIO_ROOT_USER', 'credential', true, 'MinIO root username', '^[a-zA-Z0-9_]+$', 'https://docs.min.io/docs/minio-docker-quickstart-guide.html'),
('minio', 'MINIO_ROOT_PASSWORD', 'password', true, 'MinIO root password', '.{8,}', 'https://docs.min.io/docs/minio-docker-quickstart-guide.html'),
('minio', 'MINIO_ACCESS_KEY', 'api_key', false, 'MinIO access key (legacy)', '^[a-zA-Z0-9]+$', 'https://docs.min.io/docs/minio-docker-quickstart-guide.html'),
('minio', 'MINIO_SECRET_KEY', 'api_key', false, 'MinIO secret key (legacy)', '.{8,}', 'https://docs.min.io/docs/minio-docker-quickstart-guide.html')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Qdrant resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('qdrant', 'QDRANT_HOST', 'env_var', false, 'Qdrant server hostname', '^[a-zA-Z0-9.-]+$', 'https://qdrant.tech/documentation/'),
('qdrant', 'QDRANT_PORT', 'env_var', false, 'Qdrant server port', '^[0-9]+$', 'https://qdrant.tech/documentation/'),
('qdrant', 'QDRANT_API_KEY', 'api_key', false, 'Qdrant API authentication key', '^[a-zA-Z0-9._-]+$', 'https://qdrant.tech/documentation/authentication/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Browserless resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('browserless', 'BROWSERLESS_TOKEN', 'token', false, 'Browserless authentication token', '^[a-zA-Z0-9_-]+$', 'https://docs.browserless.io/'),
('browserless', 'MAX_CONCURRENT_SESSIONS', 'env_var', false, 'Maximum concurrent browser sessions', '^[0-9]+$', 'https://docs.browserless.io/'),
('browserless', 'CONNECTION_TIMEOUT', 'env_var', false, 'Browser connection timeout in ms', '^[0-9]+$', 'https://docs.browserless.io/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Windmill resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('windmill', 'WM_TOKEN', 'token', false, 'Windmill API authentication token', '^[a-zA-Z0-9_.-]+$', 'https://www.windmill.dev/docs/'),
('windmill', 'DATABASE_URL', 'credential', false, 'Windmill database connection URL', '^postgresql://.*', 'https://www.windmill.dev/docs/'),
('windmill', 'WINDMILL_BASE_URL', 'env_var', false, 'Windmill instance base URL', '^https?://[a-zA-Z0-9.-]+(:[0-9]+)?', 'https://www.windmill.dev/docs/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Huginn resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('huginn', 'DATABASE_URL', 'credential', true, 'Huginn database connection URL', '^(mysql2|postgresql)://.*', 'https://github.com/huginn/huginn'),
('huginn', 'APP_SECRET_TOKEN', 'credential', true, 'Huginn application secret token', '.{64,}', 'https://github.com/huginn/huginn'),
('huginn', 'INVITATION_CODE', 'credential', false, 'Huginn user invitation code', '^[a-zA-Z0-9]+$', 'https://github.com/huginn/huginn'),
('huginn', 'SMTP_SERVER', 'env_var', false, 'SMTP server for email notifications', '^[a-zA-Z0-9.-]+$', 'https://github.com/huginn/huginn')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- SearXNG resource secrets  
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('searxng', 'SEARXNG_SECRET', 'credential', true, 'SearXNG secret key for session encryption', '.{32,}', 'https://searxng.readthedocs.io/'),
('searxng', 'SEARXNG_REDIS_URL', 'credential', false, 'Redis URL for SearXNG caching', '^redis://.*', 'https://searxng.readthedocs.io/'),
('searxng', 'SEARXNG_BASE_URL', 'env_var', false, 'SearXNG base URL', '^https?://[a-zA-Z0-9.-]+(:[0-9]+)?', 'https://searxng.readthedocs.io/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Judge0 resource secrets
INSERT INTO resource_secrets (resource_name, secret_key, secret_type, required, description, validation_pattern, documentation_url) VALUES
('judge0', 'JUDGE0_AUTH_TOKEN', 'token', false, 'Judge0 API authentication token', '^[a-zA-Z0-9_.-]+$', 'https://ce.judge0.com/'),
('judge0', 'REDIS_URL', 'credential', false, 'Redis connection URL for Judge0', '^redis://.*', 'https://ce.judge0.com/'),
('judge0', 'DATABASE_URL', 'credential', false, 'PostgreSQL connection URL for Judge0', '^postgresql://.*', 'https://ce.judge0.com/')
ON CONFLICT (resource_name, secret_key) DO NOTHING;

-- Insert initial scan record to track seeding
INSERT INTO secret_scans (scan_type, resources_scanned, secrets_discovered, scan_duration_ms, scan_status, scan_metadata) 
VALUES (
    'seed', 
    ARRAY['postgres', 'vault', 'redis', 'n8n', 'ollama', 'minio', 'qdrant', 'browserless', 'windmill', 'huginn', 'searxng', 'judge0'],
    (SELECT COUNT(*) FROM resource_secrets),
    0,
    'completed',
    jsonb_build_object('type', 'database_seed', 'seeded_at', CURRENT_TIMESTAMP)
);

-- Create a summary report for seeded data
DO $$
DECLARE 
    resource_count INTEGER;
    total_secrets INTEGER;
    required_secrets INTEGER;
BEGIN
    SELECT COUNT(DISTINCT resource_name) INTO resource_count FROM resource_secrets;
    SELECT COUNT(*) INTO total_secrets FROM resource_secrets;  
    SELECT COUNT(*) INTO required_secrets FROM resource_secrets WHERE required = true;
    
    RAISE NOTICE '🔐 Secrets Manager seed data loaded successfully:';
    RAISE NOTICE '   📦 Resources: % resources', resource_count;
    RAISE NOTICE '   🔑 Total secrets: % secrets', total_secrets;
    RAISE NOTICE '   ⚠️  Required secrets: % secrets', required_secrets;
    RAISE NOTICE '   ✅ Database schema ready for secret management';
END $$;
