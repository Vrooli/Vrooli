#!/usr/bin/env bats
# Step Executor Module Tests
# Tests for step execution, retry logic, and statistics

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source dependencies
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/condition.sh"
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/output.sh" || true  # Skip if output.sh doesn't exist
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/executor.sh"
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

# Helper function for testing
test_function() {
    echo "test function called with: $*"
    return 0
}

failing_function() {
    echo "failing function called"
    return 1
}

@test "executor::execute_function - executes function without arguments" {
    local step_json='{"name": "test_step", "function": "test_function"}'
    
    run executor::execute_function "$step_json"
    assert_success
    assert_output_contains "test function called"
}

@test "executor::execute_function - executes function with arguments" {
    local step_json='{"name": "test_step", "function": "test_function", "args": ["arg1", "arg2"]}'
    
    run executor::execute_function "$step_json"
    assert_success
    assert_output_contains "arg1"
    assert_output_contains "arg2"
}

@test "executor::execute_function - fails for non-existent function" {
    local step_json='{"name": "test_step", "function": "nonexistent_function"}'
    
    run executor::execute_function "$step_json"
    assert_failure
}

@test "executor::handle_step_failure - handles ignore strategy" {
    local step_json='{"name": "test_step", "on_error": "ignore"}'
    local exit_code=1
    
    run executor::handle_step_failure "$step_json" "$exit_code"
    assert_success  # Should return 0 for ignore strategy
}

@test "executor::handle_step_failure - handles stop strategy" {
    local step_json='{"name": "test_step", "on_error": "stop"}'
    local exit_code=5
    
    run executor::handle_step_failure "$step_json" "$exit_code"
    [[ $status -eq 5 ]]  # Should return the original exit code
}

@test "executor::set_step_stat - stores step statistics" {
    executor::set_step_stat "test_step" "duration" "123"
    executor::set_step_stat "test_step" "exit_code" "0"
    
    local duration
    duration=$(executor::get_step_stat "test_step" "duration")
    [[ "$duration" == "123" ]]
    
    local exit_code
    exit_code=$(executor::get_step_stat "test_step" "exit_code")
    [[ "$exit_code" == "0" ]]
}

@test "executor::get_step_stat - retrieves step statistics" {
    executor::set_step_stat "test_step" "attempts" "3"
    
    local attempts
    attempts=$(executor::get_step_stat "test_step" "attempts")
    [[ "$attempts" == "3" ]]
}

@test "executor::get_step_stat - returns empty for non-existent stat" {
    local result
    result=$(executor::get_step_stat "nonexistent_step" "nonexistent_stat")
    [[ -z "$result" ]]
}

@test "executor::get_step_stat - returns all stats for step" {
    executor::set_step_stat "test_step" "duration" "100"
    executor::set_step_stat "test_step" "exit_code" "0"
    executor::set_step_stat "test_step" "attempts" "1"
    
    local all_stats
    all_stats=$(executor::get_step_stat "test_step")
    
    # Should contain all three stats
    echo "$all_stats" | grep -q "duration: 100"
    echo "$all_stats" | grep -q "exit_code: 0"
    echo "$all_stats" | grep -q "attempts: 1"
}

@test "executor::should_retry_step - basic retry logic" {
    # Step with retry configuration
    local step_json='{"name": "test_step", "retry": {"max_attempts": 3, "delay": 1}}'
    
    # First failure should allow retry
    executor::set_step_stat "test_step" "attempts" "1"
    run executor::should_retry_step "$step_json"
    assert_success
    
    # After max attempts, should not retry
    executor::set_step_stat "test_step" "attempts" "3"
    run executor::should_retry_step "$step_json"
    assert_failure
}

@test "executor::should_retry_step - no retry config means no retry" {
    local step_json='{"name": "test_step"}'
    
    run executor::should_retry_step "$step_json"
    assert_failure
}