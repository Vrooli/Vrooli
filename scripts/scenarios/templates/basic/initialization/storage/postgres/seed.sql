-- Basic Scenario Template Seed Data
-- This file contains seed data for testing database connectivity and basic operations

-- Insert basic configuration values for testing
INSERT INTO basic_config (key, value, description) VALUES
    ('template.version', '"1.0.0"', 'Basic template version'),
    ('template.type', '"integration-test"', 'Template type identifier'),
    ('database.initialized', 'true', 'Database initialization status'),
    ('resources.postgres', '{"enabled": true, "version": "15"}', 'PostgreSQL resource configuration'),
    ('resources.redis', '{"enabled": true, "version": "7"}', 'Redis resource configuration'),
    ('testing.enabled', 'true', 'Testing framework enabled status')
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    description = EXCLUDED.description;

-- Insert sample test data
INSERT INTO test_data (name, description) VALUES
    ('Database Connection Test', 'Validates that PostgreSQL connection is working correctly'),
    ('Redis Integration Test', 'Tests Redis connectivity and basic operations'),
    ('Template Validation Test', 'Ensures template structure and configuration are valid'),
    ('Resource Health Check', 'Verifies all required resources are accessible')
ON CONFLICT (id) DO NOTHING;