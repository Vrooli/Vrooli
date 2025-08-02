#!/usr/bin/env bats
# Validation Test Suite for Assertion Functions
# This test validates that all assertion functions work correctly

bats_require_minimum_version 1.5.0

# Load BATS libraries
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../../helpers/bats-assert/load"

# Load the infrastructure we're testing
source "${BATS_TEST_DIRNAME}/../core/common_setup.bash"

setup() {
    setup_standard_mocks
    
    # Create test files for file assertions
    TEST_FILE="${TEST_TMPDIR}/test_file.txt"
    echo "test content" > "$TEST_FILE"
    chmod 644 "$TEST_FILE"
    
    TEST_DIR="${TEST_TMPDIR}/test_dir"
    mkdir -p "$TEST_DIR"
}

teardown() {
    cleanup_mocks
}

#######################################
# Output Assertions Tests
#######################################

@test "assert_output_contains works correctly" {
    output="Hello World Test"
    assert_output_contains "Hello"
    assert_output_contains "World"
    assert_output_contains "Test"
}

@test "assert_output_not_contains works correctly" {
    output="Hello World"
    assert_output_not_contains "Goodbye"
    assert_output_not_contains "Universe"
}

@test "assert_output_matches works correctly" {
    output="Hello World 123"
    assert_output_matches "Hello.*World"
    assert_output_matches "[0-9]+"
}

@test "assert_output_empty works correctly" {
    output=""
    assert_output_empty
}

@test "assert_output_line_count works correctly" {
    output="Line 1
Line 2
Line 3"
    assert_output_line_count 3
}

@test "assert_output_equals works correctly" {
    output="exact match"
    assert_output_equals "exact match"
}

#######################################
# File System Assertions Tests
#######################################

@test "assert_file_exists works correctly" {
    assert_file_exists "$TEST_FILE"
}

@test "assert_file_not_exists works correctly" {
    assert_file_not_exists "${TEST_TMPDIR}/nonexistent_file.txt"
}

@test "assert_dir_exists works correctly" {
    assert_dir_exists "$TEST_DIR"
}

@test "assert_dir_not_exists works correctly" {
    assert_dir_not_exists "${TEST_TMPDIR}/nonexistent_dir"
}

@test "assert_file_contains works correctly" {
    assert_file_contains "$TEST_FILE" "test content"
}

@test "assert_file_empty works correctly" {
    local empty_file="${TEST_TMPDIR}/empty_file.txt"
    touch "$empty_file"
    assert_file_empty "$empty_file"
}

@test "assert_file_permissions works correctly" {
    assert_file_permissions "$TEST_FILE" "644"
}

#######################################
# Environment Variable Assertions Tests
#######################################

@test "assert_env_set works correctly" {
    export TEST_ENV_VAR="test_value"
    assert_env_set "TEST_ENV_VAR"
}

@test "assert_env_equals works correctly" {
    export TEST_ENV_VAR="test_value"
    assert_env_equals "TEST_ENV_VAR" "test_value"
}

@test "assert_env_unset works correctly" {
    unset NONEXISTENT_VAR 2>/dev/null || true
    assert_env_unset "NONEXISTENT_VAR"
}

@test "assert_env_not_empty works correctly" {
    export TEST_ENV_VAR="not_empty"
    assert_env_not_empty "TEST_ENV_VAR"
}

#######################################
# Function and Command Assertions Tests
#######################################

@test "assert_command_exists works correctly" {
    assert_command_exists "echo"
    assert_command_exists "cat"
}

@test "assert_function_exists works correctly" {
    assert_function_exists "setup_standard_mocks"
    assert_function_exists "assert_output_contains"
}

#######################################
# Data Structure Assertions Tests
#######################################

@test "assert_arrays_equal works correctly" {
    local -a array1=("a" "b" "c")
    local -a array2=("a" "b" "c")
    assert_arrays_equal array1 array2
}

@test "assert_array_contains works correctly" {
    local -a test_array=("apple" "banana" "cherry")
    assert_array_contains test_array "banana"
    assert_array_contains test_array "apple"
}

#######################################
# JSON Assertions Tests
#######################################

@test "assert_json_valid works correctly" {
    local valid_json='{"status": "ok", "version": "1.0"}'
    assert_json_valid "$valid_json"
}

@test "assert_json_field_equals works correctly" {
    local json='{"status": "ok", "version": "1.0", "nested": {"key": "value"}}'
    assert_json_field_equals "$json" ".status" "ok"
    assert_json_field_equals "$json" ".version" "1.0"
    assert_json_field_equals "$json" ".nested.key" "value"
}

@test "assert_json_field_exists works correctly" {
    local json='{"status": "ok", "version": "1.0", "nested": {"key": "value"}}'
    assert_json_field_exists "$json" ".status"
    assert_json_field_exists "$json" ".nested"
    assert_json_field_exists "$json" ".nested.key"
}

#######################################
# Numeric Assertions Tests
#######################################

@test "assert_equals works correctly" {
    assert_equals "test" "test"
    assert_equals "123" "123"
}

@test "assert_not_empty works correctly" {
    assert_not_empty "not empty"
    assert_not_empty "123"
}

@test "assert_string_contains works correctly" {
    assert_string_contains "Hello World" "Hello"
    assert_string_contains "Hello World" "World"
}

@test "assert_greater_than works correctly" {
    assert_greater_than "10" "5"
    assert_greater_than "100" "99"
}

@test "assert_less_than works correctly" {
    assert_less_than "5" "10"
    assert_less_than "99" "100"
}

#######################################
# Error Handling Tests
#######################################

@test "assertions fail appropriately when conditions not met" {
    output="Hello World"
    
    # These should fail
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'; output='Hello World'; assert_output_contains 'Goodbye'"
    assert_failure
    
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'; assert_file_exists '/nonexistent/file'"
    assert_failure
    
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'; assert_env_set 'NONEXISTENT_VAR'"
    assert_failure
}

@test "JSON assertions handle invalid JSON gracefully" {
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'; assert_json_valid 'invalid json}'"
    assert_failure
    
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash'; assert_json_field_equals 'invalid json}' '.field' 'value'"
    assert_failure
}

#######################################
# Mock-Specific Assertions Tests
#######################################

@test "mock assertion functions are available" {
    assert_function_exists "assert_mock_used"
    assert_function_exists "assert_mock_response_used"
}

#######################################
# Network Assertions Tests (if available)
#######################################

@test "network assertion functions are available" {
    assert_function_exists "assert_port_available"
    assert_function_exists "assert_port_in_use"
    assert_function_exists "assert_http_endpoint_reachable"
    assert_function_exists "assert_http_status"
    assert_function_exists "assert_port_open"
}

#######################################
# Resource-Specific Assertions Tests
#######################################

@test "resource assertion functions are available" {
    assert_function_exists "assert_resource_healthy"
    assert_function_exists "assert_docker_container_running"
    assert_function_exists "assert_docker_container_healthy"
    assert_function_exists "assert_api_response_valid"
    assert_function_exists "assert_resource_chain_working"
}

@test "assertion count is as expected" {
    # Check that we have the expected number of assertion functions
    run bash -c "source '${BATS_TEST_DIRNAME}/../core/common_setup.bash' && compgen -A function | grep -c '^assert_'"
    assert_success
    
    # We should have at least 30 assertion functions
    local count="${output}"
    assert_greater_than "$count" "30"
}