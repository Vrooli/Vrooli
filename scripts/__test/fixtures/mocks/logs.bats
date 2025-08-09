#!/usr/bin/env bats
# Comprehensive tests for logs.sh mock utilities
# Tests both normal operation and edge cases

load "${BATS_TEST_DIRNAME}/../setup.bash"

# Test configuration - set BEFORE loading logs.sh
export MOCK_UTILS_VERBOSE="false"  # Keep tests quiet
export MOCK_UTILS_AUTO_INIT="false"  # Manual initialization for testing

# Load the logs.sh mock utilities
source "${BATS_TEST_DIRNAME}/logs.sh"

# Test helper to create temporary test directory
setup_test_directory() {
    export TEST_LOG_DIR="${BATS_TMPDIR}/mock-logs-test-$$"
    mkdir -p "$TEST_LOG_DIR"
}

# Test helper to cleanup test directory
cleanup_test_directory() {
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        rm -rf "$TEST_LOG_DIR"
    fi
    unset TEST_LOG_DIR
}

setup() {
    # Reset state before each test
    mock::reset_all 2>/dev/null || true
    
    # Force unset variables to ensure clean state
    unset MOCK_LOG_DIR
    unset MOCK_RESPONSES_DIR
    
    setup_test_directory
}

teardown() {
    # Clean up after each test
    mock::cleanup_logs reset 2>/dev/null || true
    cleanup_test_directory
}

#######################################
# Initialization Tests
#######################################

@test "mock::init_logging: creates log directory and files" {
    # Call directly (not with run) so environment variables persist
    mock::init_logging "$TEST_LOG_DIR"
    
    [ -d "$TEST_LOG_DIR" ]
    [ -f "$TEST_LOG_DIR/command_calls.log" ]
    [ -f "$TEST_LOG_DIR/http_calls.log" ]
    [ -f "$TEST_LOG_DIR/docker_calls.log" ]
    [ -f "$TEST_LOG_DIR/used_mocks.log" ]
    
    # Check that MOCK_LOG_DIR is set
    [ "$MOCK_LOG_DIR" = "$TEST_LOG_DIR" ]
    
    # Check backwards compatibility
    [ "$MOCK_RESPONSES_DIR" = "$TEST_LOG_DIR" ]
}

@test "mock::init_logging: creates log headers" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Check that log files have headers
    run head -n 1 "$TEST_LOG_DIR/command_calls.log"
    [[ "$output" =~ ^#.*Mock\ call\ log\ started ]]
    
    run head -n 1 "$TEST_LOG_DIR/http_calls.log"
    [[ "$output" =~ ^#.*HTTP\ mock\ calls\ started ]]
    
    run head -n 1 "$TEST_LOG_DIR/docker_calls.log"
    [[ "$output" =~ ^#.*Docker\ mock\ calls\ started ]]
    
    run head -n 1 "$TEST_LOG_DIR/used_mocks.log"
    [[ "$output" =~ ^#.*Mock\ state\ changes\ started ]]
}

@test "mock::init_logging: handles non-existent directory creation" {
    local nested_dir="$TEST_LOG_DIR/deep/nested/path"
    
    run mock::init_logging "$nested_dir"
    
    [ "$status" -eq 0 ]
    [ -d "$nested_dir" ]
    [ -f "$nested_dir/command_calls.log" ]
}

@test "mock::init_logging: returns early if already initialized" {
    # First initialization
    mock::init_logging "$TEST_LOG_DIR"
    original_dir="$MOCK_LOG_DIR"
    
    # Second initialization should return early (no custom dir provided)
    run mock::init_logging
    [ "$status" -eq 0 ]
    [ "$MOCK_LOG_DIR" = "$original_dir" ]
}

@test "mock::init_logging: fails gracefully with unwritable directory" {
    # Try to initialize in a location we can't write to (call directly so variables persist)
    mock::init_logging "/root/forbidden"
    
    # Should fallback to temp directory
    [[ "$MOCK_LOG_DIR" =~ ^/tmp/vrooli-mock-logs- ]]
}

#######################################
# Utility Function Tests
#######################################

@test "mock::is_initialized: returns correct status" {
    # Not initialized
    run mock::is_initialized
    [ "$status" -eq 1 ]
    
    # Initialize
    mock::init_logging "$TEST_LOG_DIR"
    
    # Now initialized
    run mock::is_initialized
    [ "$status" -eq 0 ]
}

@test "mock::get_log_dir: returns log directory path" {
    mock::init_logging "$TEST_LOG_DIR"
    
    run mock::get_log_dir
    [ "$status" -eq 0 ]
    [ "$output" = "$TEST_LOG_DIR" ]
}

@test "mock::get_log_dir: initializes if not already done" {
    # Should auto-initialize
    run mock::get_log_dir
    [ "$status" -eq 0 ]
    [ -n "$output" ]
    [[ "$output" =~ mock-logs ]]
}

@test "mock::reset_all: clears all state" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Verify state is set
    [ -n "$MOCK_LOG_DIR" ]
    [ -n "$MOCK_RESPONSES_DIR" ]
    
    # Reset (call directly so unset affects this shell)
    mock::reset_all
    
    # Verify state is cleared
    [ -z "${MOCK_LOG_DIR:-}" ]
    [ -z "${MOCK_RESPONSES_DIR:-}" ]
}

#######################################
# Logging Function Tests
#######################################

@test "mock::log_call: logs to appropriate files" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Test different system types
    run mock::log_call "http" "GET /api/health"
    [ "$status" -eq 0 ]
    
    run mock::log_call "docker" "ps -a"
    [ "$status" -eq 0 ]
    
    run mock::log_call "system" "systemctl status test"
    [ "$status" -eq 0 ]
    
    # Check logs were created in correct files
    grep "http: GET /api/health" "$TEST_LOG_DIR/http_calls.log"
    grep "docker: ps -a" "$TEST_LOG_DIR/docker_calls.log"
    grep "system: systemctl status test" "$TEST_LOG_DIR/command_calls.log"
}

@test "mock::log_call: includes timestamp in correct format" {
    mock::init_logging "$TEST_LOG_DIR"
    
    mock::log_call "test" "sample command"
    
    # Check timestamp format [YYYY-MM-DD HH:MM:SS]
    run grep "test: sample command" "$TEST_LOG_DIR/command_calls.log"
    [[ "$output" =~ ^\[[0-9]{4}-[0-9]{2}-[0-9]{2}\ [0-9]{2}:[0-9]{2}:[0-9]{2}\] ]]
}

@test "mock::log_call: validates required parameters" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Missing system name should fail
    run mock::log_call ""
    [ "$status" -eq 1 ]
    
    # Empty system name should fail
    export MOCK_LOG_CALL_STRICT="true"
    run mock::log_call "" "some command"
    [ "$status" -eq 1 ]
}

@test "mock::log_call: handles file write failures gracefully" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Make directory read-only to cause write failure
    chmod 444 "$TEST_LOG_DIR"
    
    # Should not fail by default (non-strict mode)
    run mock::log_call "test" "should fallback"
    [ "$status" -eq 0 ]
    
    # Should fail in strict mode
    export MOCK_LOG_CALL_STRICT="true"
    run mock::log_call "test" "should fail"
    [ "$status" -eq 1 ]
    
    # Restore permissions for cleanup
    chmod 755 "$TEST_LOG_DIR"
}

@test "mock::log_state: logs state changes" {
    mock::init_logging "$TEST_LOG_DIR"
    
    run mock::log_state "container" "nginx" "running"
    [ "$status" -eq 0 ]
    
    run mock::log_state "endpoint" "/health" "200"
    [ "$status" -eq 0 ]
    
    # Check logs were created
    grep "container:nginx:running" "$TEST_LOG_DIR/used_mocks.log"
    grep "endpoint:/health:200" "$TEST_LOG_DIR/used_mocks.log"
}

@test "mock::log_state: handles missing parameters" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Missing state_type should fail
    run mock::log_state ""
    [ "$status" -eq 1 ]
    
    # Missing identifier/state should work (empty strings)
    run mock::log_state "test" "" ""
    [ "$status" -eq 0 ]
    
    grep "test::" "$TEST_LOG_DIR/used_mocks.log"
}

@test "mock::log_and_verify: combines logging and verification" {
    mock::init_logging "$TEST_LOG_DIR"
    
    run mock::log_and_verify "http" "GET /api/test"
    [ "$status" -eq 0 ]
    
    # Check that logging worked
    grep "http: GET /api/test" "$TEST_LOG_DIR/http_calls.log"
}

@test "mock::log_and_verify: validates parameters" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Missing system name should fail
    run mock::log_and_verify ""
    [ "$status" -eq 1 ]
}

#######################################
# Cleanup Function Tests
#######################################

@test "mock::cleanup_logs: removes log files" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Create some log entries
    mock::log_call "test" "sample"
    mock::log_state "test" "id" "state"
    
    # Verify files exist and have content
    [ -f "$TEST_LOG_DIR/command_calls.log" ]
    [ -s "$TEST_LOG_DIR/command_calls.log" ]  # Non-empty
    
    # Clean up
    run mock::cleanup_logs
    [ "$status" -eq 0 ]
    
    # Files should be removed but directory should remain
    [ -d "$TEST_LOG_DIR" ]
    [ ! -f "$TEST_LOG_DIR/command_calls.log" ]
}

@test "mock::cleanup_logs: resets variables when requested" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Verify variables are set
    [ -n "$MOCK_LOG_DIR" ]
    [ -n "$MOCK_RESPONSES_DIR" ]
    
    # Clean up with reset (call directly so unset affects this shell)
    mock::cleanup_logs "reset"
    
    # Variables should be unset
    [ -z "${MOCK_LOG_DIR:-}" ]
    [ -z "${MOCK_RESPONSES_DIR:-}" ]
}

@test "mock::cleanup_logs: handles non-existent directory" {
    export MOCK_LOG_DIR="/does/not/exist"
    
    run mock::cleanup_logs
    [ "$status" -eq 0 ]  # Should not fail
}

#######################################
# Summary Function Tests
#######################################

@test "mock::print_log_summary: shows log statistics" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Add some log entries
    mock::log_call "http" "GET /test1"
    mock::log_call "http" "GET /test2"
    mock::log_call "docker" "ps"
    mock::log_state "container" "nginx" "running"
    
    run mock::print_log_summary
    [ "$status" -eq 0 ]
    
    # Check output contains expected information
    [[ "$output" =~ Mock\ Log\ Summary ]]
    [[ "$output" =~ Log\ Directory:.*$TEST_LOG_DIR ]]
    [[ "$output" =~ http_calls.log:\ 2\ entries ]]
    [[ "$output" =~ docker_calls.log:\ 1\ entries ]]
    [[ "$output" =~ used_mocks.log:\ 1\ entries ]]
}

@test "mock::print_log_summary: handles no logs" {
    # Don't initialize logging
    run mock::print_log_summary
    [ "$status" -eq 1 ]
    [[ "$output" =~ No\ mock\ logs\ available ]]
}

@test "mock::print_log_summary: handles empty log directory" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Remove all log files
    rm -f "$TEST_LOG_DIR"/*.log
    
    run mock::print_log_summary
    [ "$status" -eq 1 ]
    [[ "$output" =~ No\ log\ files\ found ]]
}

#######################################
# Integration Tests
#######################################

@test "integration: complete logging workflow" {
    # Initialize
    mock::init_logging "$TEST_LOG_DIR"
    
    # Log various types of calls
    mock::log_call "docker" "run nginx"
    mock::log_call "http" "GET /health" 
    mock::log_call "system" "ps aux"
    mock::log_state "container" "nginx" "running"
    mock::log_state "endpoint" "/health" "200"
    
    # Verify all logs were created
    [ -s "$TEST_LOG_DIR/docker_calls.log" ]
    [ -s "$TEST_LOG_DIR/http_calls.log" ]
    [ -s "$TEST_LOG_DIR/command_calls.log" ]
    [ -s "$TEST_LOG_DIR/used_mocks.log" ]
    
    # Check specific content
    grep "docker: run nginx" "$TEST_LOG_DIR/docker_calls.log"
    grep "http: GET /health" "$TEST_LOG_DIR/http_calls.log"
    grep "system: ps aux" "$TEST_LOG_DIR/command_calls.log"
    grep "container:nginx:running" "$TEST_LOG_DIR/used_mocks.log"
    grep "endpoint:/health:200" "$TEST_LOG_DIR/used_mocks.log"
    
    # Print summary
    run mock::print_log_summary
    [ "$status" -eq 0 ]
    [[ "$output" =~ docker_calls.log:\ 1\ entries ]]
    [[ "$output" =~ http_calls.log:\ 1\ entries ]]
    [[ "$output" =~ command_calls.log:\ 1\ entries ]]
    [[ "$output" =~ used_mocks.log:\ 2\ entries ]]
    
    # Clean up
    mock::cleanup_logs "reset"
    
    # Verify cleanup
    [ -z "${MOCK_LOG_DIR:-}" ]
    [ ! -f "$TEST_LOG_DIR/docker_calls.log" ]
}

#######################################
# Error Handling Tests
#######################################

@test "error handling: initialization with read-only parent directory" {
    # Create a directory we can write to initially
    local readonly_parent="$TEST_LOG_DIR/readonly"
    mkdir -p "$readonly_parent/child"
    
    # Make parent read-only AFTER creating child
    chmod 444 "$readonly_parent"
    
    # Should fail to create new subdirectory but fallback gracefully (call directly so variables persist)
    mock::init_logging "$readonly_parent/newchild"
    
    # Should succeed using fallback
    [[ "$MOCK_LOG_DIR" =~ ^/tmp/vrooli-mock-logs- ]]
    
    # Restore permissions for cleanup
    chmod 755 "$readonly_parent"
}

@test "error handling: concurrent access simulation" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Simulate concurrent logging (background processes)
    mock::log_call "test1" "concurrent call 1" &
    mock::log_call "test2" "concurrent call 2" &
    mock::log_call "test3" "concurrent call 3" &
    
    # Wait for all background jobs
    wait
    
    # All calls should have been logged
    grep "test1: concurrent call 1" "$TEST_LOG_DIR/command_calls.log"
    grep "test2: concurrent call 2" "$TEST_LOG_DIR/command_calls.log"
    grep "test3: concurrent call 3" "$TEST_LOG_DIR/command_calls.log"
}

#######################################
# Configuration Tests
#######################################

@test "configuration: verbose mode affects output" {
    # Test with verbose mode
    MOCK_UTILS_VERBOSE="true" mock::init_logging "$TEST_LOG_DIR" > "$TEST_LOG_DIR/verbose_output.txt" 2>&1
    
    # Should contain verbose messages
    grep "Mock logging initialized" "$TEST_LOG_DIR/verbose_output.txt"
    
    # Test with quiet mode
    MOCK_UTILS_VERBOSE="false" mock::init_logging "${TEST_LOG_DIR}_quiet" > "$TEST_LOG_DIR/quiet_output.txt" 2>&1
    
    # Should be empty or minimal
    [ ! -s "$TEST_LOG_DIR/quiet_output.txt" ] || [ $(wc -l < "$TEST_LOG_DIR/quiet_output.txt") -eq 0 ]
}

@test "configuration: strict mode affects error handling" {
    mock::init_logging "$TEST_LOG_DIR"
    
    # Make directory unwritable
    chmod 444 "$TEST_LOG_DIR"
    
    # Non-strict mode should succeed
    MOCK_LOG_CALL_STRICT="" run mock::log_call "test" "should succeed"
    [ "$status" -eq 0 ]
    
    # Strict mode should fail
    MOCK_LOG_CALL_STRICT="true" run mock::log_call "test" "should fail"
    [ "$status" -eq 1 ]
    
    # Restore permissions
    chmod 755 "$TEST_LOG_DIR"
}