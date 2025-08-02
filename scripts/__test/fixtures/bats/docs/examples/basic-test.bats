#!/usr/bin/env bats
# Example: Basic system command testing with standard mocks
# Use this pattern for testing scripts that use Docker, HTTP, and system commands

bats_require_minimum_version 1.5.0

# Load the unified testing infrastructure  
source "/home/matthalloran8/Vrooli/scripts/__test/fixtures/bats/core/common_setup.bash"

setup() {
    # Load basic system mocks: docker, curl, jq, systemctl, etc.
    setup_standard_mocks
}

teardown() {
    # Always clean up test environment
    cleanup_mocks
}

@test "docker commands work correctly" {
    # Test Docker version check
    run docker --version
    assert_success
    assert_output_contains "Docker version"
    
    # Test container listing
    run docker ps
    assert_success
    assert_output_contains "CONTAINER ID"
}

@test "HTTP requests are handled" {
    # Test basic HTTP health check
    run curl -s http://localhost:8080/health
    assert_success
    assert_json_valid "$output"
    
    # Test specific JSON response
    local health_response=$(curl -s http://localhost:8080/health)
    assert_json_field_equals "$health_response" ".status" "ok"
}

@test "system commands are mocked" {
    # Test systemctl service status
    run systemctl status docker
    assert_success
    assert_output_contains "active (running)"
    
    # Test jq JSON processing
    local json='{"version":"1.0.0","status":"healthy"}'
    local version=$(echo "$json" | jq -r '.version')
    assert_equals "$version" "1.0.0"
}

@test "environment is properly configured" {
    # Test that basic environment is set up
    assert_env_set "TEST_TMPDIR"
    assert_env_set "MOCK_RESPONSES_DIR"
    
    # Test that temporary directory exists
    assert_dir_exists "$TEST_TMPDIR"
    assert_dir_exists "$MOCK_RESPONSES_DIR"
}

@test "file operations work" {
    # Test file creation in test directory
    local test_file="$TEST_TMPDIR/test_file.txt"
    echo "test content" > "$test_file"
    
    assert_file_exists "$test_file"
    assert_file_contains "$test_file" "test content"
}

@test "command tracking works" {
    # Execute some commands to test tracking
    docker ps > /dev/null
    curl -s http://example.com > /dev/null
    jq '.test' <<< '{"test":"value"}' > /dev/null
    
    # Verify commands were tracked
    assert_file_exists "${MOCK_RESPONSES_DIR}/command_calls.log"
    assert_file_contains "${MOCK_RESPONSES_DIR}/command_calls.log" "docker ps"
    assert_file_contains "${MOCK_RESPONSES_DIR}/command_calls.log" "curl"
    assert_file_contains "${MOCK_RESPONSES_DIR}/command_calls.log" "jq"
}