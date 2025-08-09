#!/usr/bin/env bats
# Validation Test Suite for Assertion Functions  
# Migrated from old validation/test-assertions.bats

bats_require_minimum_version 1.5.0

# Load new unified test infrastructure
load "${BATS_TEST_DIRNAME}/../../setup"

setup() {
    vrooli_setup_unit_test
    
    # Create test files for file assertions
    TEST_FILE="$VROOLI_TEST_TMPDIR/test_file.txt"
    echo "test content" > "$TEST_FILE"
    chmod 644 "$TEST_FILE"
    
    TEST_DIR="$VROOLI_TEST_TMPDIR/test_dir"
    mkdir -p "$TEST_DIR"
}

teardown() {
    vrooli_cleanup_test
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

#######################################
# File Assertions Tests
#######################################

@test "assert_file_exists works correctly" {
    assert_file_exists "$TEST_FILE"
}

@test "assert_file_not_exists works correctly" {
    assert_file_not_exists "$VROOLI_TEST_TMPDIR/nonexistent.txt"
}

@test "assert_file_contains works correctly" {
    assert_file_contains "$TEST_FILE" "test content"
}

@test "assert_file_not_contains works correctly" {
    assert_file_not_contains "$TEST_FILE" "wrong content"
}

@test "assert_dir_exists works correctly" {
    assert_dir_exists "$TEST_DIR"
}

@test "assert_dir_not_exists works correctly" {
    assert_dir_not_exists "$VROOLI_TEST_TMPDIR/nonexistent_dir"
}

#######################################
# Environment Assertions Tests
#######################################

@test "assert_env_set works correctly" {
    export TEST_VAR="value"
    assert_env_set "TEST_VAR"
}

@test "assert_env_not_set works correctly" {
    unset NONEXISTENT_VAR
    assert_env_not_set "NONEXISTENT_VAR"
}

@test "assert_env_equals works correctly" {
    export TEST_VAR="expected_value"
    assert_env_equals "TEST_VAR" "expected_value"
}

# Migrated: Comprehensive assertion validation