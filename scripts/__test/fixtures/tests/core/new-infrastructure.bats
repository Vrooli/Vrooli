#!/usr/bin/env bats
# Test for New Vrooli Test Infrastructure
# Validates that the simplified infrastructure works correctly

# Load Vrooli test infrastructure
source "${VROOLI_TEST_ROOT:-/home/matthalloran8/Vrooli/scripts/__test}/fixtures/setup.bash"

setup() {
    # Test basic unit test setup
    vrooli_setup_unit_test
}

teardown() {
    # Test cleanup
    vrooli_cleanup_test
}

@test "new infrastructure: configuration system works" {
    # Test that configuration is loaded
    assert_command_exists "vrooli_config_get"
    
    # Test configuration values
    local timeout
    timeout=$(vrooli_config_get_timeout "basic")
    assert_greater_than "$timeout" 0
    
    # Test port configuration
    local ollama_port
    ollama_port=$(vrooli_config_get_port "ollama")
    assert_equals "$ollama_port" "11434"
}

@test "new infrastructure: test environment is configured" {
    # Test environment variables are set
    assert_env_set "VROOLI_TEST_TMPDIR"
    assert_env_set "TEST_NAMESPACE"
    assert_env_set "VROOLI_TEST_ROOT"
    
    # Test temp directory exists
    assert_dir_exists "$VROOLI_TEST_TMPDIR"
    
    # Test namespace is unique and contains the expected prefix
    [[ "$TEST_NAMESPACE" =~ ^vrooli_test_[0-9]+_[0-9]+$ ]]
}

@test "new infrastructure: assertions work correctly" {
    # Test basic assertions
    run echo "test output"
    assert_success
    assert_output_equals "test output"
    assert_output_contains "test"
    assert_output_not_contains "missing"
    
    # Test file assertions
    local test_file="$VROOLI_TEST_TMPDIR/test.txt"
    echo "file content" > "$test_file"
    assert_file_exists "$test_file"
    assert_file_contains "$test_file" "content"
    
    # Test numeric assertions
    assert_greater_than 10 5
    assert_less_than 5 10
}

@test "new infrastructure: system mocks work" {
    # Test Docker mock
    run docker --version
    assert_success
    assert_output_contains "Docker version"
    
    # Test systemctl mock
    run systemctl status test-service
    assert_success
    assert_output_contains "active"
    
    # Test HTTP mock
    run curl -s http://localhost:8080/health
    assert_success
    assert_json_valid "$output"
}

@test "new infrastructure: utilities work" {
    # Test utility functions
    local temp_dir
    temp_dir=$(vrooli_make_temp_dir)
    assert_dir_exists "$temp_dir"
    
    # Test string utilities
    local upper
    upper=$(vrooli_to_upper "test")
    assert_equals "$upper" "TEST"
    
    local lower
    lower=$(vrooli_to_lower "TEST")
    assert_equals "$lower" "test"
    
    # Test random string
    local random_str
    random_str=$(vrooli_random_string 8)
    assert_equals "${#random_str}" "8"
}

@test "new infrastructure: logging works" {
    # Test logging functions exist
    assert_function_exists "vrooli_log_info"
    assert_function_exists "vrooli_log_error"
    assert_function_exists "vrooli_log_debug"
    
    # Test logging (output goes to logs, not test output)
    vrooli_log_info "TEST" "This is a test log message"
    
    # If log file is configured, check it exists
    if [[ -n "${VROOLI_LOG_FILE:-}" ]]; then
        # Log file should exist after logging
        assert_file_exists "$VROOLI_LOG_FILE"
    fi
}

@test "new infrastructure: cleanup works properly" {
    # Create some test resources to clean up
    local test_file="$VROOLI_TEST_TMPDIR/cleanup-test.txt"
    echo "cleanup test" > "$test_file"
    assert_file_exists "$test_file"
    
    # Start a background process
    sleep 60 &
    local bg_pid=$!
    
    # Verify process is running using /proc filesystem (most reliable)
    if [[ ! -d "/proc/$bg_pid" ]]; then
        echo "ERROR: Background process not started correctly"
        return 1
    fi
    
    # Store the tmpdir path before cleanup
    local tmpdir_path="$VROOLI_TEST_TMPDIR"
    
    # Test cleanup function - specifically test process cleanup  
    _vrooli_cleanup_processes
    
    # Verify background process was killed
    sleep 1
    if [[ -d "/proc/$bg_pid" ]]; then
        # If process still exists, kill it manually and fail the test
        /bin/kill $bg_pid 2>/dev/null || true
        echo "Background process was not cleaned up properly"
        return 1
    fi
    
    # The temp file cleanup would normally be handled by teardown
    # For this test, we're just verifying process cleanup works
}

@test "new infrastructure: service setup works" {
    # Test service-specific setup
    vrooli_setup_service_test "ollama"
    
    # Check service environment is configured
    assert_env_set "VROOLI_OLLAMA_PORT"
    assert_env_set "VROOLI_OLLAMA_BASE_URL"
    assert_env_equals "VROOLI_OLLAMA_PORT" "11434"
    
    # Test service command (mocked)
    if command -v ollama >/dev/null 2>&1; then
        run ollama list
        assert_success
    fi
}

@test "new infrastructure: integration setup works" {
    # Test multi-service setup
    vrooli_setup_integration_test "ollama" "postgres"
    
    # Check both services are configured
    assert_env_set "VROOLI_OLLAMA_PORT"
    assert_env_set "VROOLI_POSTGRES_PORT"
    
    # Test that ports are different
    local ollama_port postgres_port
    ollama_port=$(vrooli_config_get_port "ollama")
    postgres_port=$(vrooli_config_get_port "postgres")
    
    if [[ "$ollama_port" == "$postgres_port" ]]; then
        echo "Services have same port: $ollama_port"
        return 1
    fi
}