#!/usr/bin/env bats
# Output Management Module Tests  
# Tests for output capture and variable management

bats_require_minimum_version 1.5.0

# Load test infrastructure (single entry point)
source "${BATS_TEST_DIRNAME}/../../../__test/fixtures/setup.bash"

setup() {
    vrooli_setup_unit_test  # Basic mocks and environment
    
    # Source the script under test
    # shellcheck disable=SC1091
    source "${BATS_TEST_DIRNAME}/output.sh"
}

teardown() {
    # Clean up output module
    output::cleanup
    
    vrooli_cleanup_test     # Clean up resources
}

@test "output::init - initializes output temporary directory" {
    run output::init
    assert_success
    
    # Should create OUTPUT_TEMP_DIR
    [[ -n "$OUTPUT_TEMP_DIR" ]]
    [[ -d "$OUTPUT_TEMP_DIR" ]]
}

@test "output::init - does not recreate existing temp dir" {
    output::init
    local first_dir="$OUTPUT_TEMP_DIR"
    
    output::init
    local second_dir="$OUTPUT_TEMP_DIR"
    
    [[ "$first_dir" == "$second_dir" ]]
}

@test "output::cleanup - removes temporary directory" {
    output::init
    local temp_dir="$OUTPUT_TEMP_DIR"
    
    # Directory should exist
    [[ -d "$temp_dir" ]]
    
    run output::cleanup
    assert_success
    
    # Directory should be removed
    [[ ! -d "$temp_dir" ]]
}

@test "output::set - stores step output" {
    run output::set "test_step" "test_output" "Hello World"
    assert_success
    
    # Verify output was stored
    local stored_output
    stored_output=$(output::get "test_step" "test_output")
    [[ "$stored_output" == "Hello World" ]]
}

@test "output::get - retrieves step output" {
    output::set "test_step" "message" "Test Message"
    
    local result
    result=$(output::get "test_step" "message")
    [[ "$result" == "Test Message" ]]
}

@test "output::get - returns empty for non-existent output" {
    local result
    result=$(output::get "nonexistent_step" "nonexistent_output")
    [[ -z "$result" ]]
}

@test "output::has - checks if output exists" {
    output::set "test_step" "result" "success"
    
    run output::has "test_step" "result"
    assert_success
    
    run output::has "test_step" "nonexistent"
    assert_failure
}

@test "output::list - lists all outputs for step" {
    output::set "test_step" "output1" "value1"
    output::set "test_step" "output2" "value2"
    output::set "other_step" "output3" "value3"
    
    local outputs
    outputs=$(output::list "test_step")
    
    # Should contain both outputs for test_step
    echo "$outputs" | grep -q "output1"
    echo "$outputs" | grep -q "output2"
    # Should not contain output from other_step
    ! echo "$outputs" | grep -q "output3"
}

@test "output::clear - clears outputs for step" {
    output::set "test_step" "output1" "value1"
    output::set "test_step" "output2" "value2"
    
    run output::clear "test_step"
    assert_success
    
    # Outputs should be gone
    run output::has "test_step" "output1"
    assert_failure
    
    run output::has "test_step" "output2"
    assert_failure
}

@test "output::export - exports output as environment variable" {
    output::set "test_step" "result" "exported_value"
    
    run output::export "test_step" "result" "EXPORTED_VAR"
    assert_success
    
    # Environment variable should be set
    [[ "$EXPORTED_VAR" == "exported_value" ]]
}