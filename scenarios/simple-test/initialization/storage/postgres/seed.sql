-- Simple Test App Seed Data
-- Initial data for framework validation

-- Set search path
SET search_path TO simple_test, public;

-- Insert sample test data
INSERT INTO test_data (name, value) VALUES 
    ('framework-test', 'Simple framework validation test'),
    ('lifecycle-test', 'Testing lifecycle system functionality'),
    ('database-test', 'Verifying database connection and operations');

-- Insert initial test results
INSERT INTO test_results (test_name, status, result_data) VALUES 
    ('initialization', 'passed', '{"message": "Database initialized successfully", "timestamp": "' || NOW()::text || '"}'),
    ('schema-creation', 'passed', '{"message": "Schema created without errors", "tables_created": 2}'),
    ('seed-data', 'running', '{"message": "Loading seed data", "records": 3}');

-- Update the seed-data test as completed
UPDATE test_results 
SET status = 'passed', 
    completed_at = NOW(),
    result_data = '{"message": "Seed data loaded successfully", "records": 3}'
WHERE test_name = 'seed-data';