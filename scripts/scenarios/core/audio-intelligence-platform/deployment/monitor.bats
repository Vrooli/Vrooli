#!/usr/bin/env bats
# Tests for audio-intelligence-platform monitor script
# Validates monitoring script functionality

bats_require_minimum_version 1.5.0

# Load test setup
# shellcheck disable=SC1091
load "../../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test
    
    # Override paths for testing to avoid conflicts
    export SCENARIO_ID="audio-intelligence-platform-test"
    export SCENARIO_NAME="Audio Intelligence Platform Test"
    
    # Load the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/monitor.sh"
}

teardown() {
    vrooli_cleanup_test
}

@test "monitor.sh loads without errors" {
    # Test that the script can be sourced without errors
    true
}

@test "monitor.sh has required variables" {
    [[ -n "${SCENARIO_ID:-}" ]]
    [[ -n "${SCENARIO_NAME:-}" ]]
    [[ -n "${MONITOR_LOG:-}" ]]
    [[ -n "${PID_FILE:-}" ]]
}

@test "monitor.sh monitoring functions are defined" {
    # Test that monitoring functions are defined
    declare -F monitor::log_monitor >/dev/null
    declare -F monitor::log_health >/dev/null
    declare -F monitor::log_alert >/dev/null
    declare -F monitor::log_performance >/dev/null
}

@test "monitor.sh monitoring functions work" {
    # Test that monitoring functions can be called
    run monitor::log_monitor "test message"
    assert_success
    
    run monitor::log_health "test health"
    assert_success
    
    run monitor::log_alert "test alert"
    assert_success
    
    run monitor::log_performance "test performance"
    assert_success
}

@test "monitor.sh log directories are created" {
    # Test that required directories exist or can be created
    [[ -d "$(dirname "$MONITOR_LOG")" ]]
    [[ -d "$(dirname "$PID_FILE")" ]]
}

# Assertion functions are provided by test fixtures