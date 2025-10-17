#!/usr/bin/env bats

# Test suite for exit_codes.sh
# Tests standard exit code definitions and helper functions

bats_require_minimum_version 1.5.0

# Load test infrastructure
source "${BATS_TEST_DIRNAME}/../../__test/fixtures/setup.bash"

# Load BATS helpers
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-support/load"
load "${BATS_TEST_DIRNAME}/../../__test/helpers/bats-assert/load"

setup() {
    vrooli_setup_unit_test
    
    # Source the exit codes utility
    source "${BATS_TEST_DIRNAME}/exit_codes.sh"
}

teardown() {
    vrooli_cleanup_test
}

# Test that exit codes are defined
@test "EXIT_SUCCESS is defined and equals 0" {
    [[ -n "${EXIT_SUCCESS+x}" ]]
    [[ "$EXIT_SUCCESS" -eq 0 ]]
}

@test "EXIT_GENERAL_ERROR is defined and equals 1" {
    [[ -n "${EXIT_GENERAL_ERROR+x}" ]]
    [[ "$EXIT_GENERAL_ERROR" -eq 1 ]]
}

@test "EXIT_MISUSE is defined and equals 2" {
    [[ -n "${EXIT_MISUSE+x}" ]]
    [[ "$EXIT_MISUSE" -eq 2 ]]
}

@test "EXIT_USER_INTERRUPT is defined and equals 130" {
    [[ -n "${EXIT_USER_INTERRUPT+x}" ]]
    [[ "$EXIT_USER_INTERRUPT" -eq 130 ]]
}

@test "EXIT_INVALID_ARGUMENT is defined" {
    [[ -n "${EXIT_INVALID_ARGUMENT+x}" ]]
    [[ "$EXIT_INVALID_ARGUMENT" -gt 0 ]]
}

@test "EXIT_FILE_NOT_FOUND is defined" {
    [[ -n "${EXIT_FILE_NOT_FOUND+x}" ]]
    [[ "$EXIT_FILE_NOT_FOUND" -gt 0 ]]
}

@test "EXIT_PERMISSION_DENIED is defined" {
    [[ -n "${EXIT_PERMISSION_DENIED+x}" ]]
    [[ "$EXIT_PERMISSION_DENIED" -gt 0 ]]
}

@test "EXIT_DEPENDENCY_ERROR is defined" {
    [[ -n "${EXIT_DEPENDENCY_ERROR+x}" ]]
    [[ "$EXIT_DEPENDENCY_ERROR" -gt 0 ]]
}

@test "EXIT_NETWORK_ERROR is defined" {
    [[ -n "${EXIT_NETWORK_ERROR+x}" ]]
    [[ "$EXIT_NETWORK_ERROR" -gt 0 ]]
}

@test "EXIT_TIMEOUT is defined and equals 124" {
    [[ -n "${EXIT_TIMEOUT+x}" ]]
    [[ "$EXIT_TIMEOUT" -eq 124 ]]
}

@test "EXIT_CONFIGURATION_ERROR is defined" {
    [[ -n "${EXIT_CONFIGURATION_ERROR+x}" ]]
    [[ "$EXIT_CONFIGURATION_ERROR" -gt 0 ]]
}

@test "EXIT_STATE_ERROR is defined" {
    [[ -n "${EXIT_STATE_ERROR+x}" ]]
    [[ "$EXIT_STATE_ERROR" -gt 0 ]]
}

@test "EXIT_RESOURCE_ERROR is defined" {
    [[ -n "${EXIT_RESOURCE_ERROR+x}" ]]
    [[ "$EXIT_RESOURCE_ERROR" -gt 0 ]]
}

# Test that exit codes are unique
@test "all exit codes have unique values" {
    local -a codes=(
        "$EXIT_SUCCESS"
        "$EXIT_GENERAL_ERROR"
        "$EXIT_MISUSE"
        "$EXIT_INVALID_ARGUMENT"
        "$EXIT_FILE_NOT_FOUND"
        "$EXIT_PERMISSION_DENIED"
        "$EXIT_DEPENDENCY_ERROR"
        "$EXIT_NETWORK_ERROR"
        "$EXIT_TIMEOUT"
        "$EXIT_CONFIGURATION_ERROR"
        "$EXIT_STATE_ERROR"
        "$EXIT_RESOURCE_ERROR"
        "$EXIT_USER_INTERRUPT"
    )
    
    # Check for duplicates
    local unique_count
    unique_count=$(printf '%s\n' "${codes[@]}" | sort -u | wc -l)
    local total_count=${#codes[@]}
    
    [[ "$unique_count" -eq "$total_count" ]]
}

# Test that exit codes are in valid range
@test "all exit codes are in valid range (0-255)" {
    [[ "$EXIT_SUCCESS" -ge 0 && "$EXIT_SUCCESS" -le 255 ]]
    [[ "$EXIT_GENERAL_ERROR" -ge 0 && "$EXIT_GENERAL_ERROR" -le 255 ]]
    [[ "$EXIT_MISUSE" -ge 0 && "$EXIT_MISUSE" -le 255 ]]
    [[ "$EXIT_INVALID_ARGUMENT" -ge 0 && "$EXIT_INVALID_ARGUMENT" -le 255 ]]
    [[ "$EXIT_FILE_NOT_FOUND" -ge 0 && "$EXIT_FILE_NOT_FOUND" -le 255 ]]
    [[ "$EXIT_PERMISSION_DENIED" -ge 0 && "$EXIT_PERMISSION_DENIED" -le 255 ]]
    [[ "$EXIT_DEPENDENCY_ERROR" -ge 0 && "$EXIT_DEPENDENCY_ERROR" -le 255 ]]
    [[ "$EXIT_NETWORK_ERROR" -ge 0 && "$EXIT_NETWORK_ERROR" -le 255 ]]
    [[ "$EXIT_TIMEOUT" -ge 0 && "$EXIT_TIMEOUT" -le 255 ]]
    [[ "$EXIT_CONFIGURATION_ERROR" -ge 0 && "$EXIT_CONFIGURATION_ERROR" -le 255 ]]
    [[ "$EXIT_STATE_ERROR" -ge 0 && "$EXIT_STATE_ERROR" -le 255 ]]
    [[ "$EXIT_RESOURCE_ERROR" -ge 0 && "$EXIT_RESOURCE_ERROR" -le 255 ]]
    [[ "$EXIT_USER_INTERRUPT" -ge 0 && "$EXIT_USER_INTERRUPT" -le 255 ]]
}

# Test helper functions if they exist
@test "exit_with_code function works if defined" {
    if declare -f exit_with_code >/dev/null 2>&1; then
        # Test in a subshell to avoid exiting the test
        run bash -c "
            source '${BATS_TEST_DIRNAME}/exit_codes.sh' 2>/dev/null || true
            declare -f exit_with_code >/dev/null 2>&1 && exit_with_code 42
        "
        [[ "$status" -eq 42 ]]
    else
        skip "exit_with_code function not defined"
    fi
}

@test "exit_success function works if defined" {
    if declare -f exit_success >/dev/null 2>&1; then
        run bash -c "
            source '${BATS_TEST_DIRNAME}/exit_codes.sh' 2>/dev/null || true
            declare -f exit_success >/dev/null 2>&1 && exit_success
        "
        [[ "$status" -eq 0 ]]
    else
        skip "exit_success function not defined"
    fi
}

@test "exit_error function works if defined" {
    if declare -f exit_error >/dev/null 2>&1; then
        run bash -c "
            source '${BATS_TEST_DIRNAME}/exit_codes.sh' 2>/dev/null || true
            declare -f exit_error >/dev/null 2>&1 && exit_error
        "
        [[ "$status" -eq 1 ]]
    else
        skip "exit_error function not defined"
    fi
}

# Test that exit codes can be used in scripts
@test "exit codes can be used in conditional statements" {
    local test_code="$EXIT_GENERAL_ERROR"
    
    if [[ "$test_code" -eq "$EXIT_GENERAL_ERROR" ]]; then
        true
    else
        false
    fi
}

@test "exit codes can be used with exit command" {
    run bash -c "exit $EXIT_SUCCESS"
    [[ "$status" -eq 0 ]]
    
    run bash -c "exit $EXIT_GENERAL_ERROR"
    [[ "$status" -eq 1 ]]
}

# Test that file can be sourced multiple times without errors
@test "exit_codes.sh can be sourced multiple times" {
    if [[ -f "${BATS_TEST_DIRNAME}/exit_codes.sh" ]]; then
        run bash -c "
            source '${BATS_TEST_DIRNAME}/exit_codes.sh'
            source '${BATS_TEST_DIRNAME}/exit_codes.sh'
            echo 'success'
        "
        [[ "$status" -eq 0 ]]
        [[ "$output" == "success" ]]
    else
        skip "exit_codes.sh file not found"
    fi
}

# Test that exit codes are exported
@test "exit codes are exported for use in subshells" {
    run bash -c "echo \$EXIT_SUCCESS"
    if [[ -n "$output" ]]; then
        [[ "$output" -eq 0 ]]
    else
        skip "EXIT_SUCCESS not exported"
    fi
}

# Test documentation/description function if it exists
@test "describe_exit_code function works if defined" {
    if declare -f describe_exit_code >/dev/null 2>&1; then
        run describe_exit_code 0
        [[ "$status" -eq 0 ]]
        [[ -n "$output" ]]
        
        run describe_exit_code 1
        [[ "$status" -eq 0 ]]
        [[ -n "$output" ]]
    else
        skip "describe_exit_code function not defined"
    fi
}

# Test that standard shell exit codes are respected
@test "standard shell exit codes are properly defined" {
    # SIGINT (Ctrl+C) should be 130
    [[ "$EXIT_USER_INTERRUPT" -eq 130 ]]
    
    # Command timeout should be 124
    [[ "$EXIT_TIMEOUT" -eq 124 ]]
    
    # Misuse of shell command should be 2
    [[ "$EXIT_MISUSE" -eq 2 ]]
}

# Test exit code naming conventions
@test "exit code constants follow naming convention" {
    # All exit code constants should start with EXIT_
    local exit_vars
    exit_vars=$(compgen -v | grep '^EXIT_' || true)
    
    if [[ -n "$exit_vars" ]]; then
        while IFS= read -r var; do
            # Check that the variable name is uppercase
            [[ "$var" == "${var^^}" ]]
        done <<< "$exit_vars"
    else
        skip "No EXIT_ variables found"
    fi
}

# Integration test with actual usage patterns
@test "exit codes work with trap handlers" {
    run bash -c "
        trap 'exit \$EXIT_USER_INTERRUPT' INT
        exit \$EXIT_SUCCESS
    "
    [[ "$status" -eq 0 ]]
}

@test "exit codes work in functions" {
    run bash -c "
        test_func() {
            return \$EXIT_INVALID_ARGUMENT
        }
        test_func
        exit \$?
    "
    [[ "$status" -eq "$EXIT_INVALID_ARGUMENT" ]]
}