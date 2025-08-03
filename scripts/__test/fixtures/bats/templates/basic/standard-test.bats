#!/usr/bin/env bats
# Template: Standard BATS Test
# Use this template for testing scripts that use basic system commands (Docker, HTTP, etc.)
#
# Copy this file and customize:
# 1. Update the test description
# 2. Add your specific test cases
# 3. Customize the setup/teardown if needed

bats_require_minimum_version 1.5.0

# Load the unified testing infrastructure
# The path resolver handles all path resolution automatically
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${TEST_DIR}/../../core/common_setup.bash"

setup() {
    # Load basic system mocks: docker, curl, jq, systemctl, etc.
    setup_standard_mocks
    
    # Add any custom setup here
    # export MY_TEST_VAR="value"
    # mkdir -p "$TEST_TMPDIR/my_test_data"
}

teardown() {
    # Always clean up test environment
    cleanup_mocks
    
    # Add any custom cleanup here
    # rm -rf "$TEST_TMPDIR/my_test_data"
}

# ================================
# EXAMPLE TESTS - Customize these
# ================================

@test "docker commands work correctly" {
    # Test Docker version check
    run docker --version
    assert_success
    assert_output_contains "Docker version"
    
    # Test container listing
    run docker ps
    assert_success
    assert_output_contains "CONTAINER"
}

@test "HTTP requests are handled" {
    # Test basic HTTP health check
    run curl -s http://localhost:8080/health
    assert_success
    assert_json_valid "$output"
    
    # Test specific JSON response
    local health_response
    health_response=$(curl -s http://localhost:8080/health)
    assert_json_field_equals "$health_response" ".status" "healthy"
}

@test "system commands are mocked" {
    # Test systemctl service status
    run systemctl status docker
    assert_success
    
    # Test jq JSON processing
    local json='{"version":"1.0.0","status":"healthy"}'
    local version
    version=$(echo "$json" | jq -r '.version')
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

# ================================
# ADD YOUR CUSTOM TESTS HERE
# ================================

# @test "my custom test" {
#     # Your test logic here
#     run your_command_here
#     assert_success
#     assert_output_contains "expected output"
# }