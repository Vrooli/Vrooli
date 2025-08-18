#!/usr/bin/env bats
# Comprehensive tests for jq mock system
# Tests the jq.sh mock implementation for correctness and integration

bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
MOCK_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${MOCK_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Test setup - load dependencies
setup() {
    # Set up test environment
    export MOCK_UTILS_VERBOSE=false
    export MOCK_VERIFICATION_ENABLED=true
    export JQ_MOCK_MODE="normal"
    
    # Load the mock utilities first (required by jq mock)
    source "${BATS_TEST_DIRNAME}/logs.sh"
    
    # Load verification system if available
    if [[ -f "${BATS_TEST_DIRNAME}/verification.bash" ]]; then
        source "${BATS_TEST_DIRNAME}/verification.sh"
    fi
    
    # Load the jq mock
    source "${BATS_TEST_DIRNAME}/jq.sh"
    
    # Initialize clean state for each test
    mock::jq::reset
    
    # Create test log directory
    TEST_LOG_DIR=$(mktemp -d)
    mock::init_logging "$TEST_LOG_DIR"
}

# Wrapper for run command that reloads jq state afterward
run_jq_command() {
    run "$@"
    # Reload state from file after jq commands that might modify state
    if [[ -n "${JQ_MOCK_STATE_FILE}" && -f "$JQ_MOCK_STATE_FILE" ]]; then
        eval "$(cat "$JQ_MOCK_STATE_FILE")" 2>/dev/null || true
    fi
}

# Test cleanup
teardown() {
    # Clean up test logs
    if [[ -n "${TEST_LOG_DIR:-}" && -d "$TEST_LOG_DIR" ]]; then
        trash::safe_remove "$TEST_LOG_DIR" --test-cleanup
    fi
    
    # Clean up environment
    unset JQ_MOCK_MODE
    unset TEST_LOG_DIR
    
    # Clean up mock state files
    if [[ -n "${JQ_MOCK_STATE_FILE:-}" && -f "$JQ_MOCK_STATE_FILE" ]]; then
        trash::safe_remove "$JQ_MOCK_STATE_FILE" --test-cleanup
    fi
}

# ===================================
# Basic Functionality Tests
# ===================================

@test "jq mock loads without errors" {
    # Test should pass if we get here - setup() would have failed otherwise
    [[ "${JQ_MOCKS_LOADED}" == "true" ]]
}

@test "jq --version returns expected version" {
    run_jq_command jq --version
    [ "$status" -eq 0 ]
    [ "$output" = "jq-1.6" ]
}

@test "jq --help returns help text" {
    run_jq_command jq --help
    [ "$status" -eq 0 ]
    [[ "$output" =~ "jq - JSON processor" ]]
    [[ "$output" =~ "Usage: jq" ]]
    [[ "$output" =~ "--compact-output" ]]
    [[ "$output" =~ "--raw-output" ]]
}

# ===================================
# Basic Filter Tests
# ===================================

@test "jq identity filter returns input unchanged" {
    mock::jq::set_data "stdin" '{"test":"value"}'
    run_jq_command jq '.'
    [ "$status" -eq 0 ]
    [ "$output" = '{"test":"value"}' ]
}

@test "jq extracts simple field" {
    mock::jq::set_data "stdin" '{"name":"Alice","age":30}'
    run_jq_command jq '.name'
    [ "$status" -eq 0 ]
    [ "$output" = '"Alice"' ]
}

@test "jq extracts numeric field" {
    mock::jq::set_data "stdin" '{"name":"Alice","age":30}'
    run_jq_command jq '.age'
    [ "$status" -eq 0 ]
    [ "$output" = '30' ]
}

@test "jq extracts boolean field" {
    mock::jq::set_data "stdin" '{"active":true}'
    run_jq_command jq '.active'
    [ "$status" -eq 0 ]
    [ "$output" = 'true' ]
}

@test "jq returns null for non-existent field" {
    mock::jq::set_data "stdin" '{"name":"Alice"}'
    run_jq_command jq '.nonexistent'
    [ "$status" -eq 0 ]
    [ "$output" = 'null' ]
}

# ===================================
# Array Processing Tests
# ===================================

@test "jq processes array elements" {
    mock::jq::set_data "stdin" '[{"id":1},{"id":2},{"id":3}]'
    run_jq_command jq '.[]'
    [ "$status" -eq 0 ]
    [[ "$output" =~ '{"id":1}' ]]
    [[ "$output" =~ '{"id":2}' ]]
    [[ "$output" =~ '{"id":3}' ]]
}

@test "jq extracts field from array elements" {
    run_jq_command jq '.users[].name'
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"Alice"' ]]
    [[ "$output" =~ '"Bob"' ]]
    [[ "$output" =~ '"Charlie"' ]]
}

@test "jq gets array length" {
    mock::jq::set_data "stdin" '[1,2,3]'
    run_jq_command jq 'length'
    [ "$status" -eq 0 ]
    [ "$output" = '3' ]
}

# ===================================
# Built-in Function Tests
# ===================================

@test "jq keys function returns object keys" {
    mock::jq::set_data "stdin" '{"b":2,"a":1,"c":3}'
    run_jq_command jq 'keys'
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"active"' ]]
    [[ "$output" =~ '"items"' ]]
    [[ "$output" =~ '"name"' ]]
    [[ "$output" =~ '"value"' ]]
}

@test "jq type function returns data type" {
    mock::jq::set_data "stdin" '{"test":"value"}'
    run_jq_command jq 'type'
    [ "$status" -eq 0 ]
    [ "$output" = '"object"' ]
}

@test "jq type function detects array" {
    mock::jq::set_data "stdin" '[1,2,3]'
    run_jq_command jq 'type'
    [ "$status" -eq 0 ]
    [ "$output" = '"array"' ]
}

@test "jq type function detects string" {
    mock::jq::set_data "stdin" '"hello world"'
    run_jq_command jq 'type'
    [ "$status" -eq 0 ]
    [ "$output" = '"string"' ]
}

@test "jq type function detects number" {
    mock::jq::set_data "stdin" '42.5'
    run_jq_command jq 'type'
    [ "$status" -eq 0 ]
    [ "$output" = '"number"' ]
}

@test "jq type function detects boolean" {
    mock::jq::set_data "stdin" 'true'
    run_jq_command jq 'type'
    [ "$status" -eq 0 ]
    [ "$output" = '"boolean"' ]
}

@test "jq type function detects null" {
    mock::jq::set_data "stdin" 'null'
    run_jq_command jq 'type'
    [ "$status" -eq 0 ]
    [ "$output" = '"null"' ]
}

# ===================================
# Output Format Tests
# ===================================

@test "jq --raw-output flag removes quotes from strings" {
    mock::jq::set_data "stdin" '{"name":"Alice"}'
    run_jq_command jq -r '.name'
    [ "$status" -eq 0 ]
    [ "$output" = 'Alice' ]
}

@test "jq --compact-output flag works" {
    mock::jq::set_data "stdin" '{"test": "value"}'
    run_jq_command jq -c '.'
    [ "$status" -eq 0 ]
    # Should remove extra whitespace
    [[ ! "$output" =~ "  " ]]
}

@test "jq --tab flag works" {
    mock::jq::set_data "stdin" '{"test":"value"}'
    run_jq_command jq --tab '.'
    [ "$status" -eq 0 ]
    [[ "$output" =~ $'\t' ]]
}

# ===================================
# File Input Tests
# ===================================

@test "jq reads from file input" {
    mock::jq::set_data "test.json" '{"file":"data"}'
    run_jq_command jq '.file' test.json
    [ "$status" -eq 0 ]
    [ "$output" = '"data"' ]
}

@test "jq handles multiple files" {
    mock::jq::set_data "file1.json" '{"id":1}'
    mock::jq::set_data "file2.json" '{"id":2}'
    run_jq_command jq '.id' file1.json
    [ "$status" -eq 0 ]
    [ "$output" = '1' ]
}

@test "jq --null-input uses null as input" {
    run_jq_command jq -n '.'
    [ "$status" -eq 0 ]
    [ "$output" = 'null' ]
}

# ===================================
# Advanced Filter Tests
# ===================================

@test "jq select filter works" {
    run_jq_command jq 'select(.active == true)'
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"Alice"' ]]
    [[ "$output" =~ '"Charlie"' ]]
    [[ ! "$output" =~ '"Bob"' ]]
}

@test "jq map function works" {
    run_jq_command jq 'map(.name)'
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"Alice"' ]]
    [[ "$output" =~ '"Bob"' ]]
    [[ "$output" =~ '"Charlie"' ]]
}

@test "jq sort_by function works" {
    run_jq_command jq 'sort_by(.age)'
    [ "$status" -eq 0 ]
    [[ "$output" =~ '"Bob"' ]]  # Should be first (age 25)
}

@test "jq group_by function works" {
    run_jq_command jq 'group_by(.active)'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "[[" ]]  # Should return nested arrays
}

@test "jq min function works" {
    run_jq_command jq 'min'
    [ "$status" -eq 0 ]
    [ "$output" = '25' ]
}

@test "jq max function works" {
    run_jq_command jq 'max'
    [ "$status" -eq 0 ]
    [ "$output" = '35' ]
}

@test "jq add function works" {
    run_jq_command jq 'add'
    [ "$status" -eq 0 ]
    [ "$output" = '90' ]
}

@test "jq empty filter produces no output" {
    run_jq_command jq 'empty'
    [ "$status" -eq 0 ]
    [ "$output" = '' ]
}

# ===================================
# Error Handling Tests
# ===================================

@test "jq mock mode error returns exit code 1" {
    export JQ_MOCK_MODE="error"
    run_jq_command jq '.'
    [ "$status" -eq 1 ]
    [[ "$output" =~ "error" ]]
}

@test "jq mock mode invalid_json returns exit code 4" {
    export JQ_MOCK_MODE="invalid_json"
    run_jq_command jq '.'
    [ "$status" -eq 4 ]
    [[ "$output" =~ "parse error" ]]
}

@test "jq mock mode filter_error returns exit code 5" {
    export JQ_MOCK_MODE="filter_error"
    run_jq_command jq '.'
    [ "$status" -eq 5 ]
    [[ "$output" =~ "compile error" ]]
}

@test "jq injected parse error works" {
    mock::jq::inject_error "." "parse_error"
    run_jq_command jq '.'
    [ "$status" -eq 4 ]
    [[ "$output" =~ "parse error" ]]
}

@test "jq injected compile error works" {
    mock::jq::inject_error ".invalid" "compile_error"
    run_jq_command jq '.invalid'
    [ "$status" -eq 5 ]
    [[ "$output" =~ "compile error" ]]
}

@test "jq injected type error works" {
    mock::jq::inject_error ".string" "type_error"
    run_jq_command jq '.string'
    [ "$status" -eq 5 ]
    [[ "$output" =~ "type error" ]]
}

@test "jq file not found error works" {
    mock::jq::inject_error "missing.json" "file_not_found"
    run_jq_command jq '.' missing.json
    [ "$status" -eq 2 ]
    [[ "$output" =~ "No such file or directory" ]]
}

@test "jq permission denied error works" {
    mock::jq::inject_error "restricted.json" "permission_denied"
    run_jq_command jq '.' restricted.json
    [ "$status" -eq 2 ]
    [[ "$output" =~ "Permission denied" ]]
}

# ===================================
# Mock State Management Tests
# ===================================

@test "jq mock reset clears all state" {
    mock::jq::set_data "test" '{"test":true}'
    mock::jq::set_response ".custom" "custom_response"
    mock::jq::inject_error ".error" "test_error"
    
    mock::jq::reset
    
    # Verify state is cleared
    [[ -z "${MOCK_JQ_DATA[test]}" ]]
    [[ -z "${MOCK_JQ_RESPONSES[.custom]}" ]]
    [[ -z "${MOCK_JQ_ERRORS[.error]}" ]]
    [[ "$JQ_MOCK_CALL_COUNTER" -eq 0 ]]
}

@test "jq mock set_data stores data correctly" {
    mock::jq::set_data "custom.json" '{"custom":"data"}'
    [[ "${MOCK_JQ_DATA[custom.json]}" == '{"custom":"data"}' ]]
}

@test "jq mock set_response stores response correctly" {
    mock::jq::set_response ".custom_filter" '{"response":"data"}'
    [[ "${MOCK_JQ_RESPONSES[.custom_filter]}" == '{"response":"data"}' ]]
}

@test "jq custom response is used when set" {
    mock::jq::set_response ".custom" "CUSTOM_RESPONSE"
    mock::jq::set_data "stdin" '{"any":"data"}'
    run_jq_command jq '.custom'
    [ "$status" -eq 0 ]
    [ "$output" = "CUSTOM_RESPONSE" ]
}

# ===================================
# Call Tracking Tests
# ===================================

@test "jq call counter increments correctly" {
    run_jq_command jq '.'
    mock::jq::assert::call_count 1
    
    run_jq_command jq '.name'
    mock::jq::assert::call_count 2
}

@test "jq call history is recorded" {
    run_jq_command jq -r '.name'
    run_jq_command jq '.age'
    
    local history
    history=$(mock::jq::get::call_history)
    [[ "$history" =~ "-r .name" ]]
    [[ "$history" =~ ".age" ]]
}

@test "jq last call is tracked correctly" {
    run_jq_command jq '.first'
    run_jq_command jq '.second'
    
    local last_call
    last_call=$(mock::jq::get::last_call)
    [[ "$last_call" == ".second" ]]
}

@test "jq assert filter called works" {
    run_jq_command jq '.name'
    mock::jq::assert::filter_called '.name'
}

@test "jq assert filter called fails when not called" {
    run ! mock::jq::assert::filter_called '.never_called'
}

# ===================================
# Scenario Helper Tests
# ===================================

@test "jq scenario setup_user_data works" {
    mock::jq::scenario::setup_user_data
    
    [[ -n "${MOCK_JQ_DATA[stdin]}" ]]
    [[ -n "${MOCK_JQ_DATA[users.json]}" ]]
    [[ "${MOCK_JQ_DATA[stdin]}" =~ "users" ]]
    [[ "${MOCK_JQ_DATA[stdin]}" =~ "Alice" ]]
}

@test "jq scenario setup_simple_object works" {
    mock::jq::scenario::setup_simple_object
    
    [[ -n "${MOCK_JQ_DATA[stdin]}" ]]
    [[ -n "${MOCK_JQ_DATA[simple.json]}" ]]
    [[ "${MOCK_JQ_DATA[stdin]}" =~ "test" ]]
    [[ "${MOCK_JQ_DATA[stdin]}" =~ "42" ]]
}

@test "jq scenario setup_array_data works" {
    mock::jq::scenario::setup_array_data
    
    [[ -n "${MOCK_JQ_DATA[stdin]}" ]]
    [[ -n "${MOCK_JQ_DATA[array.json]}" ]]
    [[ "${MOCK_JQ_DATA[stdin]}" =~ "first" ]]
    [[ "${MOCK_JQ_DATA[stdin]}" =~ "second" ]]
}

# ===================================
# Assertion Helper Tests
# ===================================

@test "jq assert output_equals works with correct output" {
    mock::jq::set_data "stdin" '{"test":"value"}'
    mock::jq::assert::output_equals '"value"' '.test'
}

@test "jq assert output_equals fails with incorrect output" {
    mock::jq::set_data "stdin" '{"test":"value"}'
    run ! mock::jq::assert::output_equals '"wrong"' '.test'
}

# ===================================
# Integration Tests
# ===================================

@test "jq mock integrates with logging system" {
    run_jq_command jq '.test'
    
    # Check that logs were created
    [[ -f "$TEST_LOG_DIR/command_calls.log" ]]
    
    # Check log content
    local log_content
    log_content=$(cat "$TEST_LOG_DIR/command_calls.log")
    [[ "$log_content" =~ "jq: .test" ]]
}

@test "jq mock state persists across subshells" {
    mock::jq::set_data "persistent.json" '{"persistent":"data"}'
    
    # Run in subshell
    (
        source "${BATS_TEST_DIRNAME}/jq.sh"
        run_jq_command jq '.persistent' persistent.json
        [ "$status" -eq 0 ]
        [ "$output" = '"data"' ]
    )
}

@test "jq complex filter combination works" {
    mock::jq::scenario::setup_user_data
    
    # Test chained operations
    run_jq_command jq '.users[] | select(.active) | .name'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "Alice" ]]
    [[ "$output" =~ "Charlie" ]]
    [[ ! "$output" =~ "Bob" ]]
}

@test "jq handles multiple output flags" {
    mock::jq::set_data "stdin" '{"name":"test value"}'
    run_jq_command jq -r -c '.name'
    [ "$status" -eq 0 ]
    [ "$output" = 'test value' ]
}

# ===================================
# Edge Case Tests
# ===================================

@test "jq handles empty input gracefully" {
    mock::jq::set_data "stdin" ''
    run_jq_command jq '.'
    [ "$status" -eq 0 ]
}

@test "jq handles null input correctly" {
    mock::jq::set_data "stdin" 'null'
    run_jq_command jq '.'
    [ "$status" -eq 0 ]
    [ "$output" = 'null' ]
}

@test "jq handles complex nested structures" {
    local complex_json='{"level1":{"level2":{"level3":{"value":"deep"}}}}'
    mock::jq::set_data "stdin" "$complex_json"
    run_jq_command jq '.'
    [ "$status" -eq 0 ]
    [[ "$output" =~ "deep" ]]
}

@test "jq debug dump shows current state" {
    mock::jq::set_data "test" '{"debug":"data"}'
    mock::jq::set_response ".debug" "debug_response"
    mock::jq::inject_error ".error" "debug_error"
    run_jq_command jq '.test'
    
    local debug_output
    debug_output=$(mock::jq::debug::dump_state)
    [[ "$debug_output" =~ "jq Mock State Dump" ]]
    [[ "$debug_output" =~ "Call Counter: 1" ]]
    [[ "$debug_output" =~ "debug_data" ]]
    [[ "$debug_output" =~ "debug_response" ]]
    [[ "$debug_output" =~ "debug_error" ]]
}

# ===================================
# Performance and Stress Tests
# ===================================

@test "jq mock handles multiple rapid calls" {
    for i in {1..10}; do
        run_jq_command jq ".test$i"
    done
    
    mock::jq::assert::call_count 10
}

@test "jq mock maintains state consistency under load" {
    # Set up multiple data sources
    for i in {1..5}; do
        mock::jq::set_data "file$i.json" "{\"id\":$i}"
    done
    
    # Access them all
    for i in {1..5}; do
        run_jq_command jq '.id' "file$i.json"
        [ "$status" -eq 0 ]
        [ "$output" = "$i" ]
    done
}

# ===================================
# Final Validation Tests
# ===================================

@test "jq mock cleanup works correctly" {
    # Set up some state
    mock::jq::set_data "cleanup_test" '{"test":"data"}'
    run_jq_command jq '.test'
    
    # Verify state exists
    [[ -n "${MOCK_JQ_DATA[cleanup_test]}" ]]
    [[ "$JQ_MOCK_CALL_COUNTER" -gt 0 ]]
    
    # Reset and verify cleanup
    mock::jq::reset
    [[ -z "${MOCK_JQ_DATA[cleanup_test]}" ]]
    [[ "$JQ_MOCK_CALL_COUNTER" -eq 0 ]]
}

@test "jq mock exports all required functions" {
    # Test that all critical functions are exported and callable
    declare -F jq >/dev/null
    declare -F mock::jq::reset >/dev/null
    declare -F mock::jq::set_data >/dev/null
    declare -F mock::jq::set_response >/dev/null
    declare -F mock::jq::inject_error >/dev/null
    declare -F mock::jq::assert::output_equals >/dev/null
    declare -F mock::jq::debug::dump_state >/dev/null
}