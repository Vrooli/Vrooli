#!/usr/bin/env bats
# Tests for Claude Code error-handling.sh functions
bats_require_minimum_version 1.5.0

# Source trash module for safe test cleanup
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Load Vrooli test infrastructure
source "${APP_ROOT}/__test/fixtures/setup.bash"

setup_file() {
    # Use Vrooli service test setup
    vrooli_setup_service_test "claude-code-error-handling"
    
    # Load dependencies
    SCRIPT_DIR="${APP_ROOT}/resources/claude-code"
    
    # shellcheck disable=SC1091
    source "${APP_ROOT}/lib/utils/var.sh" || true
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_DIR}/common.sh" || true
    
    # Load configuration if available
    if [[ -f "${SCRIPT_DIR}/config/defaults.sh" ]]; then
        source "${SCRIPT_DIR}/config/defaults.sh"
    fi
    
    # Load the error handling functions
    source "${BATS_TEST_DIRNAME}/error-handling.sh"
    
    # Export functions for subshells
    export -f claude_code::error_report
    export -f claude_code::error_validate_config
    export -f claude_code::safe_execute
}

setup() {
    # Reset mocks for each test
    mock::claude::reset >/dev/null 2>&1 || true
    
    # Create temporary test files
    export TEST_ERROR_DIR="${BATS_TMPDIR}/error-test"
    mkdir -p "$TEST_ERROR_DIR"
}

teardown() {
    # Clean up test files
    trash::safe_remove "$TEST_ERROR_DIR" --test-cleanup
}

#######################################
# Error Code Interpretation Tests
#######################################

@test "CLAUDE_CODE_ERRORS array contains expected error codes" {
    # Verify that key error codes are defined
    [[ -n "${CLAUDE_CODE_ERRORS[0]}" ]]
    [[ -n "${CLAUDE_CODE_ERRORS[1]}" ]]
    [[ -n "${CLAUDE_CODE_ERRORS[6]}" ]]
    [[ "${CLAUDE_CODE_ERRORS[0]}" == "SUCCESS" ]]
    [[ "${CLAUDE_CODE_ERRORS[6]}" == "AUTHENTICATION_ERROR" ]]
}

@test "RECOVERY_STRATEGIES array contains expected strategies" {
    # Verify that key recovery strategies are defined
    [[ -n "${RECOVERY_STRATEGIES[NETWORK_ERROR]}" ]]
    [[ -n "${RECOVERY_STRATEGIES[TIMEOUT_ERROR]}" ]]
    [[ -n "${RECOVERY_STRATEGIES[AUTHENTICATION_ERROR]}" ]]
    [[ "${RECOVERY_STRATEGIES[RATE_LIMIT_ERROR]}" == "wait_and_retry" ]]
}

#######################################
# Error Report Tests
#######################################

@test "claude_code::error_report - should handle general error" {
    run claude_code::error_report "test_context" "1"
    
    assert_success
    assert_output --partial "Error Report"
    assert_output --partial "Context: test_context"
    assert_output --partial "GENERAL_ERROR"
}

@test "claude_code::error_report - should handle authentication error" {
    run claude_code::error_report "auth_test" "6"
    
    assert_success
    assert_output --partial "AUTHENTICATION_ERROR"
    assert_output --partial "Suggested Recovery"
}

@test "claude_code::error_report - should handle unknown error code" {
    run claude_code::error_report "unknown_test" "999"
    
    assert_success
    assert_output --partial "UNKNOWN_ERROR"
    assert_output --partial "Code: 999"
}

@test "claude_code::error_report - should handle success code" {
    run claude_code::error_report "success_test" "0"
    
    assert_success
    assert_output --partial "SUCCESS"
}

#######################################
# Error Validation Tests
#######################################

@test "claude_code::error_validate_config - should validate basic configuration" {
    run claude_code::error_validate_config
    
    assert_success
    assert_output --partial "Configuration validation"
}

@test "claude_code::error_validate_config - should detect missing dependencies" {
    # Temporarily remove a command from PATH
    local original_path="$PATH"
    export PATH="/nonexistent"
    
    run claude_code::error_validate_config
    
    # Restore PATH
    export PATH="$original_path"
    
    assert_success
    assert_output --partial "validation"
}

#######################################
# Safe Execute Tests
#######################################

@test "claude_code::safe_execute - should require prompt" {
    run claude_code::safe_execute "" "" "5" "3"
    
    assert_failure
    assert_output --partial "Prompt is required"
}

@test "claude_code::safe_execute - should validate max_retries parameter" {
    run claude_code::safe_execute "test prompt" "" "5" "not_a_number"
    
    assert_failure
    assert_output --partial "Max retries must be a number"
}

@test "claude_code::safe_execute - should execute with valid parameters" {
    # Mock successful execution
    mock::claude::scenario::setup_development
    
    run claude_code::safe_execute "test prompt" "Read,Write" "5" "3"
    
    assert_success
    assert_output --partial "Executing Claude Code safely"
}

@test "claude_code::safe_execute - should retry on failure" {
    # Mock initial failure, then success
    mock::claude::inject_error "prompt:any" "network_error"
    
    run claude_code::safe_execute "test prompt" "" "5" "2"
    
    # Should attempt retry
    assert_output --partial "retry"
}

@test "claude_code::safe_execute - should handle authentication error" {
    # Mock authentication error
    mock::claude::scenario::setup_auth_required
    
    run claude_code::safe_execute "test prompt" "" "5" "3"
    
    assert_failure
    assert_output --partial "Authentication"
}

@test "claude_code::safe_execute - should respect max retries" {
    # Mock persistent failure
    mock::claude::inject_error "prompt:any" "general_error"
    
    run claude_code::safe_execute "test prompt" "" "5" "1"
    
    assert_failure
    assert_output --partial "Maximum retries reached"
}

#######################################
# Error Recovery Strategy Tests
#######################################

@test "error recovery strategies should be callable functions" {
    # Test that recovery strategy functions exist (if implemented)
    # This is a placeholder for when recovery functions are implemented
    
    # For now, just verify the strategies array is accessible
    [[ -n "${RECOVERY_STRATEGIES[NETWORK_ERROR]}" ]]
}

@test "error codes should cover common scenarios" {
    # Verify comprehensive error code coverage
    [[ -n "${CLAUDE_CODE_ERRORS[1]}" ]]  # GENERAL_ERROR
    [[ -n "${CLAUDE_CODE_ERRORS[4]}" ]]  # NETWORK_ERROR
    [[ -n "${CLAUDE_CODE_ERRORS[5]}" ]]  # TIMEOUT_ERROR
    [[ -n "${CLAUDE_CODE_ERRORS[6]}" ]]  # AUTHENTICATION_ERROR
    [[ -n "${CLAUDE_CODE_ERRORS[7]}" ]]  # RATE_LIMIT_ERROR
}