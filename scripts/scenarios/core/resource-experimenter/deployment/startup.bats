#!/usr/bin/env bats
# Tests for startup.sh

# Source test setup infrastructure
source "$(dirname "${BATS_TEST_FILENAME}")/../../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Load official mocks
    load "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/docker.sh"
    load "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh"
    
    # Reset mock state
    mock::docker::reset
    mock::http::reset
}

teardown() {
    vrooli_cleanup_test
}

@test "startup script sources var.sh correctly" {
    # Test that var.sh is sourced and variables are available
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Check that var_ variables are available
    [[ -n "$var_SCRIPTS_DIR" ]]
    [[ -n "$var_ROOT_DIR" ]]
}

@test "startup::log_info creates proper log entries" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Set up test log file
    export LOG_FILE="$VROOLI_TEST_TMPDIR/startup.log"
    
    # Run logging function
    run startup::log_info "Test message"
    
    # Verify log entry was created
    assert_success
    assert_output_contains "INFO: Test message"
    [[ -f "$LOG_FILE" ]]
}

@test "startup::is_service_running checks docker containers" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Set up mock container state
    mock::docker::set_container_state "vrooli-postgres" "running"
    mock::docker::set_container_state "test-container" "running"
    
    # Test service detection
    run startup::is_service_running "vrooli-postgres"
    assert_success
    
    run startup::is_service_running "nonexistent-service"
    assert_failure
}

@test "startup::wait_for_service waits for service health" {
    # Source the script functions
    source "$BATS_TEST_DIRNAME/startup.sh"
    
    # Set up test log file
    export LOG_FILE="$VROOLI_TEST_TMPDIR/startup.log"
    
    # Mock successful service health check
    mock::http::set_endpoint_response "http://localhost:8080/health" 200 "OK"
    
    # Run with short timeout to avoid long test times
    run startup::wait_for_service "test-service" "http://localhost:8080/health" 5 1
    
    # Should succeed quickly with mocked healthy service
    assert_success
    assert_output_contains "Waiting for test-service to be ready"
    assert_output_contains "test-service is ready"
}