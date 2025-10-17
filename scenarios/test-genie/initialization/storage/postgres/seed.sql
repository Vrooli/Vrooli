-- Test Genie Seed Data
-- Inserts sample data and default configurations

-- Insert default test patterns
INSERT INTO test_patterns (pattern_name, pattern_type, template_code, description, variables) VALUES
    (
        'api_health_check',
        'unit',
        '#!/bin/bash
# Health check test for {{.scenario_name}}
API_PORT=${API_PORT:-{{.default_port}}}
response=$(curl -s -w "%{http_code}" http://localhost:${API_PORT}/health)
http_code=$(echo $response | tail -c 4)
if [ "$http_code" = "200" ]; then
    echo "✓ Health check passed"
    exit 0
else
    echo "✗ Health check failed (HTTP $http_code)"
    exit 1
fi',
        'Standard API health check test',
        '{"scenario_name": "string", "default_port": "number"}'
    ),
    (
        'database_connection',
        'unit',
        '#!/bin/bash
# Database connection test for {{.scenario_name}}
POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_DB=${POSTGRES_DB:-{{.database_name}}}

PGPASSWORD=${POSTGRES_PASSWORD} psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT 1;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Database connection successful"
    exit 0
else
    echo "✗ Database connection failed"
    exit 1
fi',
        'Database connectivity test',
        '{"scenario_name": "string", "database_name": "string"}'
    ),
    (
        'end_to_end_workflow',
        'integration',
        '#!/bin/bash
# End-to-end workflow test for {{.scenario_name}}
API_PORT=${API_PORT:-{{.default_port}}}
BASE_URL="http://localhost:${API_PORT}"

echo "Testing complete {{.scenario_name}} workflow..."

# Step 1: Health check
response=$(curl -s "$BASE_URL/health")
if [[ $response == *"healthy"* ]]; then
    echo "✓ Step 1: Health check passed"
else
    echo "✗ Step 1: Health check failed"
    exit 1
fi

# Step 2: Create test data
test_id=$(uuidgen)
response=$(curl -s -X POST "$BASE_URL/api/{{.api_endpoint}}" -H "Content-Type: application/json" -d "{\"id\":\"$test_id\",\"name\":\"workflow_test\"}")
if [[ $response == *"created"* ]] || [[ $response == *"success"* ]]; then
    echo "✓ Step 2: Test data creation passed"
else
    echo "✗ Step 2: Test data creation failed"
    exit 1
fi

# Step 3: Retrieve test data
response=$(curl -s "$BASE_URL/api/{{.api_endpoint}}/$test_id")
if [[ $response == *"workflow_test"* ]] || [[ $? -eq 0 ]]; then
    echo "✓ Step 3: Data retrieval passed"
else
    echo "✗ Step 3: Data retrieval failed"
    exit 1
fi

echo "✓ End-to-end workflow test completed successfully"',
        'Complete workflow integration test',
        '{"scenario_name": "string", "default_port": "number", "api_endpoint": "string"}'
    ),
    (
        'performance_baseline',
        'performance',
        '#!/bin/bash
# Performance baseline test for {{.scenario_name}}
API_PORT=${API_PORT:-{{.default_port}}}
BASE_URL="http://localhost:${API_PORT}"
CONCURRENT_REQUESTS={{.concurrent_requests}}
TOTAL_REQUESTS={{.total_requests}}
MAX_RESPONSE_TIME={{.max_response_time}}

echo "Running performance test with $CONCURRENT_REQUESTS concurrent connections, $TOTAL_REQUESTS total requests..."

if command -v ab >/dev/null 2>&1; then
    ab -c $CONCURRENT_REQUESTS -n $TOTAL_REQUESTS "$BASE_URL/{{.test_endpoint}}" > /tmp/perf_results.txt 2>&1
    
    avg_time=$(grep "Time per request" /tmp/perf_results.txt | head -1 | awk ''{print $4}'')
    failed_requests=$(grep "Failed requests" /tmp/perf_results.txt | awk ''{print $3}'')
    
    if (( $(echo "$avg_time < $MAX_RESPONSE_TIME" | bc -l) )) && [ "$failed_requests" = "0" ]; then
        echo "✓ Performance test passed (avg: ${avg_time}ms, failures: $failed_requests)"
        exit 0
    else
        echo "✗ Performance test failed (avg: ${avg_time}ms, failures: $failed_requests)"
        exit 1
    fi
else
    echo "Apache Bench not available, using basic curl test..."
    for i in $(seq 1 10); do
        response_time=$(curl -w "%{time_total}" -s -o /dev/null "$BASE_URL/{{.test_endpoint}}")
        echo "Request $i: ${response_time}s"
    done
    echo "✓ Basic performance test completed"
fi',
        'Performance baseline and load test',
        '{"scenario_name": "string", "default_port": "number", "concurrent_requests": "number", "total_requests": "number", "max_response_time": "number", "test_endpoint": "string"}'
    ),
    (
        'vault_phase_validation',
        'vault',
        '#!/bin/bash
# Vault {{.phase}} phase test for {{.scenario_name}}
echo "Testing vault {{.phase}} phase..."

case "{{.phase}}" in
    "setup")
        echo "Checking resource availability..."
        {{#each setup_checks}}
        {{this}}
        {{/each}}
        echo "✓ Setup phase validation completed"
        ;;
    "develop")
        echo "Checking development artifacts..."
        {{#each develop_checks}}
        {{this}}
        {{/each}}
        echo "✓ Develop phase validation completed"
        ;;
    "test")
        echo "Validating test execution..."
        {{#each test_checks}}
        {{this}}
        {{/each}}
        echo "✓ Test phase validation completed"
        ;;
    "deploy")
        echo "Checking deployment status..."
        {{#each deploy_checks}}
        {{this}}
        {{/each}}
        echo "✓ Deploy phase validation completed"
        ;;
    "monitor")
        echo "Validating monitoring setup..."
        {{#each monitor_checks}}
        {{this}}
        {{/each}}
        echo "✓ Monitor phase validation completed"
        ;;
esac

echo "Vault {{.phase}} phase test completed successfully"',
        'Vault phase validation test template',
        '{"scenario_name": "string", "phase": "string", "setup_checks": "array", "develop_checks": "array", "test_checks": "array", "deploy_checks": "array", "monitor_checks": "array"}'
    ),
    (
        'regression_baseline',
        'regression',
        '#!/bin/bash
# Regression test for {{.scenario_name}}
API_PORT=${API_PORT:-{{.default_port}}}
BASE_URL="http://localhost:${API_PORT}"

echo "Running regression tests for {{.scenario_name}}..."

# Test 1: Verify core API endpoints are still working
endpoints=({{#each endpoints}}"/{{this}}"{{#unless @last}} {{/unless}}{{/each}})
for endpoint in "${endpoints[@]}"; do
    response=$(curl -s -w "%{http_code}" "$BASE_URL$endpoint")
    http_code=$(echo $response | tail -c 4)
    if [ "$http_code" = "200" ] || [ "$http_code" = "404" ]; then
        echo "✓ Endpoint $endpoint: HTTP $http_code"
    else
        echo "✗ Endpoint $endpoint failed: HTTP $http_code"
        exit 1
    fi
done

# Test 2: Check database schema stability
echo "Checking database schema stability..."
# Add database schema validation here if needed

# Test 3: Verify performance hasn''t degraded significantly
echo "Checking performance baseline..."
response_time=$(curl -w "%{time_total}" -s -o /dev/null "$BASE_URL/health")
if (( $(echo "$response_time < {{.max_response_time}}" | bc -l) )); then
    echo "✓ Performance within acceptable range (${response_time}s)"
else
    echo "⚠ Performance may have degraded (${response_time}s)"
fi

echo "✓ Regression tests completed successfully"',
        'Regression testing to ensure no functionality loss',
        '{"scenario_name": "string", "default_port": "number", "endpoints": "array", "max_response_time": "number"}'
    );

-- Insert sample test suites for demonstration
INSERT INTO test_suites (scenario_name, suite_type, test_types, coverage_metrics) VALUES
    (
        'test-genie-demo',
        'comprehensive',
        ARRAY['unit', 'integration', 'performance'],
        '{"code_coverage": 95.2, "branch_coverage": 91.8, "function_coverage": 97.1}'
    ),
    (
        'document-manager-sample',
        'standard',
        ARRAY['unit', 'integration'],
        '{"code_coverage": 88.5, "branch_coverage": 85.2, "function_coverage": 92.3}'
    ),
    (
        'api-health-check-suite',
        'unit',
        ARRAY['unit'],
        '{"code_coverage": 100.0, "branch_coverage": 100.0, "function_coverage": 100.0}'
    );

-- Get the suite IDs for inserting test cases
-- Note: In a real implementation, you'd use the actual UUIDs returned from the INSERT

-- Insert sample test cases (using a subquery to get suite IDs)
INSERT INTO test_cases (suite_id, name, description, test_type, test_code, expected_result, execution_timeout, tags, priority)
SELECT 
    ts.id,
    'health_check_api_response',
    'Verify API health endpoint returns 200 status',
    'unit',
    '#!/bin/bash
API_PORT=${API_PORT:-8200}
response=$(curl -s -w "%{http_code}" http://localhost:${API_PORT}/health)
http_code=$(echo $response | tail -c 4)
if [ "$http_code" = "200" ]; then
    echo "✓ Health check passed"
    exit 0
else
    echo "✗ Health check failed (HTTP $http_code)"
    exit 1
fi',
    '{"status": "healthy"}',
    30,
    '["api", "health", "unit"]',
    'critical'
FROM test_suites ts WHERE ts.scenario_name = 'test-genie-demo'
LIMIT 1;

INSERT INTO test_cases (suite_id, name, description, test_type, test_code, expected_result, execution_timeout, tags, priority)
SELECT 
    ts.id,
    'database_connection_test',
    'Verify database connection is working',
    'unit',
    '#!/bin/bash
POSTGRES_HOST=${POSTGRES_HOST:-localhost}
POSTGRES_PORT=${POSTGRES_PORT:-5432}
POSTGRES_USER=${POSTGRES_USER:-postgres}
POSTGRES_DB=${POSTGRES_DB:-test_genie}

PGPASSWORD=${POSTGRES_PASSWORD} psql -h $POSTGRES_HOST -p $POSTGRES_PORT -U $POSTGRES_USER -d $POSTGRES_DB -c "SELECT 1;" > /dev/null 2>&1
if [ $? -eq 0 ]; then
    echo "✓ Database connection successful"
    exit 0
else
    echo "✗ Database connection failed"
    exit 1
fi',
    '{"connection": "success"}',
    30,
    '["database", "connection", "unit"]',
    'critical'
FROM test_suites ts WHERE ts.scenario_name = 'test-genie-demo'
LIMIT 1;

INSERT INTO test_cases (suite_id, name, description, test_type, test_code, expected_result, execution_timeout, tags, priority)
SELECT 
    ts.id,
    'end_to_end_test_generation',
    'Test complete test generation workflow',
    'integration',
    '#!/bin/bash
API_PORT=${API_PORT:-8200}
BASE_URL="http://localhost:${API_PORT}"

# Test test suite generation
response=$(curl -s -X POST "$BASE_URL/api/v1/test-suite/generate" \
    -H "Content-Type: application/json" \
    -d ''{"scenario_name":"integration-test","test_types":["unit"],"coverage_target":80}'')

if [[ $response == *"suite_id"* ]]; then
    echo "✓ Test generation successful"
    exit 0
else
    echo "✗ Test generation failed"
    echo "Response: $response"
    exit 1
fi',
    '{"suite_id": "uuid", "generated_tests": "number"}',
    120,
    '["integration", "workflow", "e2e"]',
    'high'
FROM test_suites ts WHERE ts.scenario_name = 'test-genie-demo'
LIMIT 1;

-- Insert sample coverage analysis
INSERT INTO coverage_analysis (scenario_name, analysis_type, overall_coverage, coverage_by_file, coverage_gaps, improvement_suggestions, priority_areas)
VALUES (
    'test-genie-demo',
    'comprehensive',
    92.5,
    '{"main.go": 95.2, "handlers.go": 91.8, "models.go": 88.3, "utils.go": 96.7}',
    '{"untested_functions": ["handleError", "validateConfig"], "untested_branches": ["error recovery", "timeout handling"], "untested_edge_cases": ["null input", "concurrent access"]}',
    '["Add error handling tests", "Include timeout scenario tests", "Test concurrent access patterns", "Add input validation edge cases"]',
    '["Error handling coverage", "Concurrent access testing", "Input validation"]'
);

-- Insert sample test vault
INSERT INTO test_vaults (scenario_name, vault_name, phases, phase_configurations, success_criteria)
VALUES (
    'comprehensive-demo',
    'full-lifecycle-vault',
    '["setup", "develop", "test", "deploy", "monitor"]',
    '{
        "setup": {"timeout": 300, "requirements": ["resource_availability", "database_connection"]},
        "develop": {"timeout": 600, "requirements": ["api_functionality", "business_logic"]},
        "test": {"timeout": 900, "requirements": ["unit_tests_pass", "integration_tests_pass"]},
        "deploy": {"timeout": 300, "requirements": ["deployment_successful", "health_check_pass"]},
        "monitor": {"timeout": 180, "requirements": ["monitoring_active", "alerts_configured"]}
    }',
    '{
        "all_phases_completed": true,
        "no_critical_failures": true,
        "coverage_threshold": 90,
        "performance_baseline_met": true
    }'
);

-- Insert sample system metrics
INSERT INTO system_metrics (metric_name, metric_value, metric_unit, tags) VALUES
    ('test_generation_time', 45.2, 'seconds', '{"scenario": "demo", "test_types": "unit,integration"}'),
    ('test_execution_time', 127.8, 'seconds', '{"suite": "comprehensive", "environment": "local"}'),
    ('coverage_analysis_time', 23.1, 'seconds', '{"depth": "comprehensive", "files_analyzed": 12}'),
    ('api_response_time', 156.3, 'milliseconds', '{"endpoint": "/test-suite/generate", "status": "success"}'),
    ('system_load', 0.65, 'ratio', '{"cpu_cores": 8, "memory_gb": 16}');

-- Update pattern usage counts (simulate some usage)
UPDATE test_patterns SET usage_count = 25, success_rate = 96.8 WHERE pattern_name = 'api_health_check';
UPDATE test_patterns SET usage_count = 18, success_rate = 94.2 WHERE pattern_name = 'database_connection';
UPDATE test_patterns SET usage_count = 12, success_rate = 91.7 WHERE pattern_name = 'end_to_end_workflow';
UPDATE test_patterns SET usage_count = 8, success_rate = 88.9 WHERE pattern_name = 'performance_baseline';
UPDATE test_patterns SET usage_count = 15, success_rate = 93.4 WHERE pattern_name = 'vault_phase_validation';
UPDATE test_patterns SET usage_count = 10, success_rate = 95.1 WHERE pattern_name = 'regression_baseline';

-- Create a sample completed execution
INSERT INTO test_executions (suite_id, execution_type, start_time, end_time, status, total_tests, passed_tests, failed_tests, skipped_tests, environment, performance_metrics)
SELECT 
    ts.id,
    'full',
    CURRENT_TIMESTAMP - INTERVAL '2 hours',
    CURRENT_TIMESTAMP - INTERVAL '1 hour 50 minutes',
    'completed',
    3,
    3,
    0,
    0,
    'local',
    '{"execution_time": 597.2, "resource_usage": {"cpu": "45%", "memory": "1.2GB"}, "error_count": 0}'
FROM test_suites ts 
WHERE ts.scenario_name = 'test-genie-demo'
LIMIT 1;

-- Insert corresponding test results for the execution
INSERT INTO test_results (execution_id, test_case_id, status, duration, started_at, completed_at)
SELECT 
    te.id,
    tc.id,
    'passed',
    CASE 
        WHEN tc.test_type = 'unit' THEN RANDOM() * 30 + 5
        WHEN tc.test_type = 'integration' THEN RANDOM() * 120 + 30
        ELSE RANDOM() * 60 + 15
    END,
    te.start_time + (RANDOM() * INTERVAL '10 minutes'),
    te.start_time + (RANDOM() * INTERVAL '10 minutes') + INTERVAL '30 seconds'
FROM test_executions te
JOIN test_suites ts ON te.suite_id = ts.id
JOIN test_cases tc ON tc.suite_id = ts.id
WHERE ts.scenario_name = 'test-genie-demo'
AND te.status = 'completed';

-- Refresh the materialized view
SELECT refresh_daily_metrics();

-- Final verification and summary
SELECT 
    'Sample data insertion completed successfully!' as status,
    (SELECT COUNT(*) FROM test_suites) as test_suites_created,
    (SELECT COUNT(*) FROM test_cases) as test_cases_created,
    (SELECT COUNT(*) FROM test_patterns) as patterns_available,
    (SELECT COUNT(*) FROM coverage_analysis) as coverage_analyses,
    (SELECT COUNT(*) FROM test_vaults) as vaults_created;

-- Show sample data summary
SELECT 'Test Suites Created:' as category, scenario_name, suite_type, array_length(test_types, 1) as test_type_count
FROM test_suites
UNION ALL
SELECT 'Test Patterns Available:', pattern_name, pattern_type, usage_count::text
FROM test_patterns
ORDER BY category, scenario_name;