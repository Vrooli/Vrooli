#!/usr/bin/env bats
# Condition Evaluation Module Tests
# Tests for condition evaluation and parsing

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/condition.sh"
}

teardown() {
    vrooli_cleanup_test     # Clean up resources
}

@test "condition::evaluate - empty condition returns true" {
    run condition::evaluate ""
    assert_success
}

@test "condition::evaluate - true string returns success" {
    run condition::evaluate "true"
    assert_success
    
    run condition::evaluate "TRUE"
    assert_success
    
    run condition::evaluate "yes"
    assert_success
    
    run condition::evaluate "YES"
    assert_success
    
    run condition::evaluate "1"
    assert_success
}

@test "condition::evaluate - false string returns failure" {
    run condition::evaluate "false"
    assert_failure
    
    run condition::evaluate "FALSE"
    assert_failure
    
    run condition::evaluate "no"
    assert_failure
    
    run condition::evaluate "NO"
    assert_failure
    
    run condition::evaluate "0"
    assert_failure
}

@test "condition::evaluate_string_equals - basic equality" {
    run condition::evaluate_string_equals "test == test"
    assert_success
    
    run condition::evaluate_string_equals "foo == bar"
    assert_failure
}

@test "condition::evaluate_string_not_equals - basic inequality" {
    run condition::evaluate_string_not_equals "foo != bar"
    assert_success
    
    run condition::evaluate_string_not_equals "test != test"
    assert_failure
}

@test "condition::evaluate_exists - file existence check" {
    # Create a temporary file
    local test_file="${BATS_TEST_TMPDIR}/test_file"
    touch "$test_file"
    
    local json="{\"type\": \"exists\", \"path\": \"$test_file\"}"
    run condition::evaluate_json "$json"
    assert_success
    
    # Test non-existent file
    local json2="{\"type\": \"exists\", \"path\": \"/nonexistent/file\"}"
    run condition::evaluate_json "$json2"
    assert_failure
}

@test "condition::evaluate_equals - json equality check" {
    local json='{"type": "equals", "left": "test", "right": "test"}'
    run condition::evaluate_json "$json"
    assert_success
    
    local json2='{"type": "equals", "left": "foo", "right": "bar"}'
    run condition::evaluate_json "$json2"
    assert_failure
}

@test "condition::evaluate_command - command execution check" {
    local json='{"type": "command", "command": "true"}'
    run condition::evaluate_json "$json"
    assert_success
    
    local json2='{"type": "command", "command": "false"}'
    run condition::evaluate_json "$json2"
    assert_failure
}

@test "condition::should_skip_step - ONLY_STEP environment variable" {
    export ONLY_STEP="target_step"
    
    run condition::should_skip_step "target_step"
    assert_failure  # Should not skip
    
    run condition::should_skip_step "other_step"
    assert_success  # Should skip
    
    unset ONLY_STEP
}

@test "condition::should_skip_step - SKIP_STEPS_LIST environment variable" {
    export SKIP_STEPS_LIST="step1:step2:step3"
    
    run condition::should_skip_step "step2"
    assert_success  # Should skip
    
    run condition::should_skip_step "step4"
    assert_failure  # Should not skip
    
    unset SKIP_STEPS_LIST
}