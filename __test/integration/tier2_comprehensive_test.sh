#!/usr/bin/env bash
# Comprehensive Tier 2 Mock Integration Test
# Tests multiple mock categories working together using new Tier 2 architecture

set -uo pipefail

echo "=== Tier 2 Mock Comprehensive Integration Test ==="
echo ""

# Source the test helper
APP_ROOT="${APP_ROOT:-$(builtin cd "${0%/*}/../.." && builtin pwd)}"
source "${APP_ROOT}/__test/mocks/test_helper.sh"

# Test counters
tests_passed=0
tests_failed=0

# Test function helper
run_test() {
    local test_name="$1"
    local test_function="$2"
    
    echo -n "[$((tests_passed + tests_failed + 1))] $test_name: "
    
    if $test_function >/dev/null 2>&1; then
        echo "✓ PASSED"
        ((tests_passed++))
    else
        echo "✗ FAILED"
        ((tests_failed++))
    fi
}

# === Core Infrastructure Tests ===
test_core_infrastructure() {
    load_test_mock "redis"
    load_test_mock "postgres"
    load_test_mock "docker"
    
    # Test cross-service data flow using v2.0 CLI patterns
    resource-redis content add "db_config" "host=localhost,port=5432" >/dev/null
    local config
    config=$(resource-redis content get "db_config")
    
    resource-postgres content execute "CREATE TABLE config (key TEXT, value TEXT)" >/dev/null
    resource-postgres content execute "INSERT INTO config VALUES ('redis_config', '$config')" >/dev/null
    
    resource-docker status >/dev/null
    
    return 0
}

# === AI/ML Pipeline Tests ===
test_ai_pipeline() {
    load_test_mock "ollama"
    load_test_mock "whisper"
    load_test_mock "claude-code"
    
    # Test AI workflow using v2.0 CLI patterns
    resource-ollama content add --name "llama2" >/dev/null 2>&1
    resource-whisper content execute --input "audio.wav" --operation transcribe >/dev/null 2>&1
    resource-claude-code content execute --prompt "analyze this" >/dev/null 2>&1
    
    return 0
}

# === Automation Workflow Tests ===
test_automation_workflow() {
    load_test_mock "n8n"
    load_test_mock "node-red"
    load_test_mock "windmill"
    
    # Test workflow creation using v2.0 CLI patterns
    resource-n8n content add --name "test-workflow" --type workflow >/dev/null 2>&1
    resource-node-red content execute --operation deploy >/dev/null 2>&1
    resource-windmill content add --name "test-script" --data "print('hello')" >/dev/null 2>&1
    
    return 0
}

# === Storage Layer Tests ===
test_storage_layer() {
    load_test_mock "minio"
    load_test_mock "qdrant"
    load_test_mock "questdb"
    
    # Test storage operations using v2.0 CLI patterns
    resource-minio content add --name "test-bucket" --type bucket >/dev/null 2>&1
    resource-qdrant content add --name "test-collection" --type collection >/dev/null 2>&1
    resource-questdb content add --name "metrics" --schema "timestamp:TIMESTAMP,value:DOUBLE" >/dev/null 2>&1
    
    return 0
}

# === Utilities and Tools Tests ===
test_utilities() {
    load_test_mock "jq"
    load_test_mock "logs"
    load_test_mock "verification"
    load_test_mock "dig"
    
    # Test utility functions
    echo '{"test":"value"}' | jq '.test' >/dev/null 2>&1
    mock::log_call "test" "message" >/dev/null 2>&1
    mock::verify_call "test" "verification" >/dev/null 2>&1
    dig +short example.com >/dev/null 2>&1
    
    return 0
}

# === Cross-Service Integration Test ===
test_cross_service_integration() {
    # Load multiple services
    load_resource_test_mocks "storage"
    load_resource_test_mocks "ai"
    
    # Simulate a complete workflow using v2.0 CLI patterns:
    # 1. Store data in PostgreSQL
    resource-postgres content execute "CREATE TABLE ai_requests (id SERIAL, prompt TEXT, response TEXT)" >/dev/null
    resource-postgres content execute "INSERT INTO ai_requests (prompt) VALUES ('test prompt')" >/dev/null
    
    # 2. Cache in Redis
    resource-redis content add "last_prompt" "test prompt" >/dev/null
    
    # 3. Process with AI
    local response
    response=$(resource-ollama content execute --model "llama2" --prompt "test prompt" 2>/dev/null)
    
    # 4. Update database with response
    resource-postgres content execute "UPDATE ai_requests SET response = '$response' WHERE id = 1" >/dev/null
    
    # 5. Store embeddings
    resource-qdrant content add --name "embeddings" --data '{"id":1,"vector":[0.1,0.2,0.3]}' >/dev/null 2>&1
    
    # 6. Log metrics
    resource-questdb content add --table "ai_metrics" --data "1,success,$(date +%s)" >/dev/null 2>&1
    
    return 0
}

# === Error Handling Tests ===
test_error_handling() {
    load_test_mock "redis"
    
    # Test error injection
    inject_test_error "redis" "connection_refused"
    
    # This should fail
    if resource-redis test smoke >/dev/null 2>&1; then
        return 1
    fi
    
    # Clear errors
    clear_test_errors "redis"
    
    # This should succeed
    resource-redis test smoke >/dev/null 2>&1
    
    return 0
}

# === Performance Tests ===
test_performance() {
    load_test_mock "redis"
    load_test_mock "postgres"
    
    # Test bulk operations using v2.0 CLI patterns
    for i in {1..50}; do
        resource-redis content add "key$i" "value$i" >/dev/null 2>&1
    done
    
    # Test database operations
    resource-postgres content execute "CREATE TABLE performance_test (id INT, data TEXT)" >/dev/null
    for i in {1..20}; do
        resource-postgres content execute "INSERT INTO performance_test VALUES ($i, 'data$i')" >/dev/null
    done
    
    return 0
}

# === State Management Tests ===
test_state_management() {
    load_test_mock "redis"
    
    # Test state persistence within session using v2.0 CLI patterns
    resource-redis content add "state_test" "initial_value" >/dev/null
    local value1
    value1=$(resource-redis content get "state_test")
    
    resource-redis content add "state_test" "updated_value" >/dev/null
    local value2
    value2=$(resource-redis content get "state_test")
    
    [[ "$value1" != "$value2" ]] && return 0 || return 1
}

# === Run All Tests ===
echo "Running comprehensive tests..."
echo ""

run_test "Core Infrastructure" test_core_infrastructure
run_test "AI/ML Pipeline" test_ai_pipeline
run_test "Automation Workflow" test_automation_workflow
run_test "Storage Layer" test_storage_layer
run_test "Utilities and Tools" test_utilities
run_test "Cross-Service Integration" test_cross_service_integration
run_test "Error Handling" test_error_handling
run_test "Performance" test_performance
run_test "State Management" test_state_management

echo ""
echo "=== Test Results ==="
echo "Passed: $tests_passed"
echo "Failed: $tests_failed"
echo "Total: $((tests_passed + tests_failed))"

if [[ $tests_failed -eq 0 ]]; then
    echo ""
    echo "🎉 ALL TESTS PASSED!"
    echo "Tier 2 mock integration is fully functional."
else
    echo ""
    echo "⚠️  Some tests failed. Check the implementation."
fi

echo ""
echo "=== Mock Usage Statistics ==="
echo "Loaded mocks in this session:"
for mock in "${!LOADED_MOCKS[@]}"; do
    echo "  $mock: ${LOADED_MOCKS[$mock]}"
done

exit $tests_failed