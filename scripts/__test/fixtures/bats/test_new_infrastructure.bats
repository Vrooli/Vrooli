#!/usr/bin/env bats
# Test for the new unified bats-fixtures infrastructure
# This test validates that the new mock registry and common setup work correctly

bats_require_minimum_version 1.5.0

# Setup for each test
setup() {
    # Load the new unified infrastructure
    # Source from relative path or use VROOLI_TEST_FIXTURES_DIR if set
    if [[ -n "${VROOLI_TEST_FIXTURES_DIR:-}" ]]; then
        source "${VROOLI_TEST_FIXTURES_DIR}/core/common_setup.bash"
    else
        # Find fixtures directory relative to this test file
        TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        source "${TEST_DIR}/core/common_setup.bash"
    fi
    
    # Use standard setup
    setup_standard_mocks
}

teardown() {
    # Clean up test environment
    cleanup_mocks
}

@test "new infrastructure: mock registry loads correctly" {
    # Test that mock registry is available
    assert_function_exists "mock::load"
    assert_function_exists "mock::setup_minimal"
    assert_function_exists "mock::setup_resource"
}

@test "new infrastructure: assertions library loads correctly" {
    # Test that enhanced assertions are available
    assert_function_exists "assert_output_contains"
    assert_function_exists "assert_resource_healthy"
    assert_function_exists "assert_docker_container_running"
    assert_function_exists "assert_json_valid"
}

@test "new infrastructure: system mocks work correctly" {
    # Test Docker mock
    run docker --version
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Docker version" ]]
    
    # Test HTTP mock
    run curl -s http://localhost:8080/health
    [ "$status" -eq 0 ]
    assert_json_valid "$output"
    
    # Test jq mock (run in a clean way to avoid setup logs)
    local jq_result=$(echo '{"status":"ok"}' | jq -r '.status')
    [ "$jq_result" = "ok" ]
}

@test "new infrastructure: resource-specific setup works" {
    # Test ollama resource setup
    setup_resource_test "ollama"
    
    # Verify environment is configured
    assert_env_set "OLLAMA_PORT"
    assert_env_set "OLLAMA_BASE_URL"
    assert_env_equals "OLLAMA_PORT" "11434"
}

@test "new infrastructure: performance optimizations work" {
    # Test that in-memory temp directory is used
    assert_env_set "BATS_TEST_TMPDIR"
    [[ "$BATS_TEST_TMPDIR" =~ /dev/shm ]] || [[ "$BATS_TEST_TMPDIR" =~ /tmp ]]
    
    # Test that test isolation is configured
    assert_env_set "TEST_NAMESPACE"
    assert_env_set "TEST_PORT_BASE"
}

@test "new infrastructure: mock tracking works" {
    # Test that command calls are tracked
    assert_env_set "MOCK_RESPONSES_DIR"
    assert_dir_exists "$MOCK_RESPONSES_DIR"
    
    # Execute some commands to test tracking
    docker ps
    curl -s http://test.example.com
    
    # Verify tracking files exist
    assert_file_exists "${MOCK_RESPONSES_DIR}/command_calls.log"
}

@test "new infrastructure: integration test setup works" {
    # Test multi-resource setup
    setup_integration_test "ollama" "whisper" "n8n"
    
    # Verify all resources are configured
    assert_env_set "OLLAMA_PORT"
    assert_env_set "WHISPER_PORT" 
    assert_env_set "N8N_PORT"
}

@test "new infrastructure: enhanced assertions work" {
    # Test new assertion functions
    
    # Test output assertions
    output="test output with specific content"
    assert_output_contains "specific content"
    
    # Test JSON assertions
    local test_json='{"status":"ok","version":"1.0.0"}'
    assert_json_valid "$test_json"
    assert_json_field_equals "$test_json" ".status" "ok"
    assert_json_field_exists "$test_json" ".version"
    
    # Test environment assertions
    export TEST_VAR="test_value"
    assert_env_set "TEST_VAR"
    assert_env_equals "TEST_VAR" "test_value"
}

@test "new infrastructure: backward compatibility works" {
    # Test that legacy functions still work
    assert_function_exists "setup_standard_mock_framework"
    assert_function_exists "cleanup_standard_mock_framework"
    
    # Test legacy setup
    setup_standard_mock_framework
    [ "$?" -eq 0 ]
}

@test "new infrastructure: error handling works" {
    # Test that invalid mock loading fails gracefully
    run mock::load "invalid_category" "invalid_name"
    [ "$status" -ne 0 ]
    
    # Test that missing assertions fail correctly
    run assert_env_set "NONEXISTENT_VAR"
    [ "$status" -ne 0 ]
}