#!/usr/bin/env bats
# Shared Test Setup for Vrooli Resource Tests
# Provides common test utilities and assertions for BATS tests

# =============================================================================
# Test Environment Setup
# =============================================================================

# Source var.sh for standardized paths
_HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck disable=SC1091
source "${_HERE}/../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

# Set up VROOLI_TEST_ROOT for all tests  
if [[ -z "${VROOLI_TEST_ROOT:-}" ]]; then
    VROOLI_TEST_ROOT="${var_SCRIPTS_DIR}/resources/tests"
    export VROOLI_TEST_ROOT
fi

# Ensure we have BATS test utilities
if [[ -z "${BATS_TEST_TMPDIR:-}" ]]; then
    export BATS_TEST_TMPDIR="${TMPDIR:-/tmp}/bats_$$"
    mkdir -p "$BATS_TEST_TMPDIR"
fi

# =============================================================================
# Common Test Utilities
# =============================================================================

# Enhanced assertion for line content
assert_line() {
    local expected="$1"
    local found=false
    
    while IFS= read -r line; do
        if [[ "$line" == "$expected" ]]; then
            found=true
            break
        fi
    done <<< "$output"
    
    if [[ "$found" != "true" ]]; then
        echo "Expected line not found: $expected" >&2
        echo "Actual output:" >&2
        echo "$output" >&2
        return 1
    fi
}

# Negated line assertion
refute_line() {
    local unexpected="$1"
    
    while IFS= read -r line; do
        if [[ "$line" == "$unexpected" ]]; then
            echo "Unexpected line found: $unexpected" >&2
            echo "Full output:" >&2
            echo "$output" >&2
            return 1
        fi
    done <<< "$output"
}

# Assert output contains partial string
assert_output_contains() {
    local expected="$1"
    
    if [[ "$output" != *"$expected"* ]]; then
        echo "Expected substring not found: $expected" >&2
        echo "Actual output:" >&2
        echo "$output" >&2
        return 1
    fi
}

# Assert output does not contain partial string
refute_output_contains() {
    local unexpected="$1"
    
    if [[ "$output" == *"$unexpected"* ]]; then
        echo "Unexpected substring found: $unexpected" >&2
        echo "Full output:" >&2
        echo "$output" >&2
        return 1
    fi
}

# Assert file exists
assert_file_exists() {
    local file="$1"
    
    if [[ ! -f "$file" ]]; then
        echo "Expected file does not exist: $file" >&2
        return 1
    fi
}

# Assert file does not exist
refute_file_exists() {
    local file="$1"
    
    if [[ -f "$file" ]]; then
        echo "Unexpected file exists: $file" >&2
        return 1
    fi
}

# Assert directory exists
assert_dir_exists() {
    local dir="$1"
    
    if [[ ! -d "$dir" ]]; then
        echo "Expected directory does not exist: $dir" >&2
        return 1
    fi
}

# Assert directory does not exist
refute_dir_exists() {
    local dir="$1"
    
    if [[ -d "$dir" ]]; then
        echo "Unexpected directory exists: $dir" >&2
        return 1
    fi
}

# Assert file contains text
assert_file_contains() {
    local file="$1"
    local expected="$2"
    
    if [[ ! -f "$file" ]]; then
        echo "File does not exist: $file" >&2
        return 1
    fi
    
    if ! grep -q "$expected" "$file"; then
        echo "Expected text not found in file: $expected" >&2
        echo "File: $file" >&2
        echo "File contents:" >&2
        cat "$file" >&2
        return 1
    fi
}

# Assert variable is set
assert_variable_set() {
    local var_name="$1"
    local var_value="${!var_name:-}"
    
    if [[ -z "$var_value" ]]; then
        echo "Expected variable is not set: $var_name" >&2
        return 1
    fi
}

# Assert variable is not set
refute_variable_set() {
    local var_name="$1"
    local var_value="${!var_name:-}"
    
    if [[ -n "$var_value" ]]; then
        echo "Unexpected variable is set: $var_name=$var_value" >&2
        return 1
    fi
}

# Assert exit status
assert_status() {
    local expected_status="$1"
    
    if [[ "$status" -ne "$expected_status" ]]; then
        echo "Expected exit status $expected_status, got $status" >&2
        if [[ -n "${output:-}" ]]; then
            echo "Output:" >&2
            echo "$output" >&2
        fi
        return 1
    fi
}

# =============================================================================
# Test Data Helpers
# =============================================================================

# Create a temporary file with content
create_temp_file() {
    local content="$1"
    local filename="${2:-test_file}"
    local temp_file="$BATS_TEST_TMPDIR/$filename"
    
    echo "$content" > "$temp_file"
    echo "$temp_file"
}

# Create a temporary directory
create_temp_dir() {
    local dirname="${1:-test_dir}"
    local temp_dir="$BATS_TEST_TMPDIR/$dirname"
    
    mkdir -p "$temp_dir"
    echo "$temp_dir"
}

# Create a temporary script file
create_temp_script() {
    local content="$1"
    local filename="${2:-test_script.sh}"
    local temp_script="$BATS_TEST_TMPDIR/$filename"
    
    echo "#!/bin/bash" > "$temp_script"
    echo "$content" >> "$temp_script"
    chmod +x "$temp_script"
    echo "$temp_script"
}

# =============================================================================
# Official Mock System Integration
# =============================================================================

# Load official mocks - use the centralized mock system instead of ad-hoc mocks
load_official_mocks() {
    # Load system mocks for common commands
    if [[ -f "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/system.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/system.sh"
    fi
    
    # Load Docker mocks if needed
    if [[ -f "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/docker.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/docker.sh"
    fi
    
    # Load HTTP mocks if needed
    if [[ -f "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/http.sh"
    fi
}

# Initialize official mock logging
init_official_mock_logging() {
    if [[ -f "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh" ]]; then
        # shellcheck disable=SC1091
        source "${var_SCRIPTS_TEST_DIR}/fixtures/mocks/logs.sh"
        mock::init_logging
    fi
}

# =============================================================================
# Performance Helpers
# =============================================================================

# Time a command execution
time_command() {
    local start_time
    start_time=$(date +%s%N)
    
    "$@"
    local exit_code=$?
    
    local end_time
    end_time=$(date +%s%N)
    local duration=$(( (end_time - start_time) / 1000000 ))  # Convert to milliseconds
    
    export COMMAND_DURATION_MS="$duration"
    return $exit_code
}

# Assert command completes within time limit
assert_command_fast() {
    local max_duration_ms="$1"
    
    if [[ -z "${COMMAND_DURATION_MS:-}" ]]; then
        echo "No timing information available. Use time_command first." >&2
        return 1
    fi
    
    if [[ "$COMMAND_DURATION_MS" -gt "$max_duration_ms" ]]; then
        echo "Command took too long: ${COMMAND_DURATION_MS}ms (max: ${max_duration_ms}ms)" >&2
        return 1
    fi
}

# =============================================================================
# System State Helpers
# =============================================================================

# Save current environment
save_environment() {
    env > "$BATS_TEST_TMPDIR/saved_env"
}

# Restore saved environment
restore_environment() {
    if [[ -f "$BATS_TEST_TMPDIR/saved_env" ]]; then
        # This is a simplified restore - in practice, this is complex
        # For now, just restore PATH if it was saved
        if grep -q "^PATH=" "$BATS_TEST_TMPDIR/saved_env"; then
            export PATH=$(grep "^PATH=" "$BATS_TEST_TMPDIR/saved_env" | cut -d= -f2-)
        fi
    fi
}

# Save current working directory
save_cwd() {
    pwd > "$BATS_TEST_TMPDIR/saved_cwd"
}

# Restore saved working directory
restore_cwd() {
    if [[ -f "$BATS_TEST_TMPDIR/saved_cwd" ]]; then
        cd "$(cat "$BATS_TEST_TMPDIR/saved_cwd")"
    fi
}

# =============================================================================
# Cleanup Helpers
# =============================================================================

# Standard cleanup function
standard_cleanup() {
    # Clean up any background processes
    jobs -p | xargs -r kill 2>/dev/null || true
    
    # Clean up temporary files
    trash::safe_remove "$BATS_TEST_TMPDIR"/* --test-cleanup 2>/dev/null || true
    
    # Restore working directory
    restore_cwd 2>/dev/null || true
}

# =============================================================================
# Debug Helpers
# =============================================================================

# Print debug information
debug_output() {
    if [[ "${BATS_DEBUG:-}" == "1" ]]; then
        echo "=== DEBUG OUTPUT ===" >&2
        echo "Status: $status" >&2
        echo "Output:" >&2
        echo "$output" >&2
        echo "===================" >&2
    fi
}

# Print test environment information
debug_environment() {
    if [[ "${BATS_DEBUG:-}" == "1" ]]; then
        echo "=== DEBUG ENVIRONMENT ===" >&2
        echo "VROOLI_TEST_ROOT: $VROOLI_TEST_ROOT" >&2
        echo "BATS_TEST_TMPDIR: $BATS_TEST_TMPDIR" >&2
        echo "PWD: $(pwd)" >&2
        echo "=========================" >&2
    fi
}

# =============================================================================
# Integration Test Helpers
# =============================================================================

# Skip test if prerequisites not met
skip_if_not_available() {
    local command="$1"
    local message="${2:-$command not available}"
    
    if ! command -v "$command" &>/dev/null; then
        skip "$message"
    fi
}

# Skip test if file not found
skip_if_file_missing() {
    local file="$1"
    local message="${2:-Required file not found: $file}"
    
    if [[ ! -f "$file" ]]; then
        skip "$message"
    fi
}

# Skip test if directory not found
skip_if_dir_missing() {
    local dir="$1"
    local message="${2:-Required directory not found: $dir}"
    
    if [[ ! -d "$dir" ]]; then
        skip "$message"
    fi
}

# =============================================================================
# Validation Helpers
# =============================================================================

# Assert YAML is valid
assert_valid_yaml() {
    local yaml_file="$1"
    
    if ! command -v python3 &>/dev/null; then
        skip "python3 not available for YAML validation"
    fi
    
    if ! python3 -c "import yaml; yaml.safe_load(open('$yaml_file'))" 2>/dev/null; then
        echo "Invalid YAML file: $yaml_file" >&2
        return 1
    fi
}

# Assert JSON is valid
assert_valid_json() {
    local json_content="$1"
    
    if ! command -v python3 &>/dev/null; then
        skip "python3 not available for JSON validation"
    fi
    
    if ! echo "$json_content" | python3 -m json.tool >/dev/null 2>&1; then
        echo "Invalid JSON content:" >&2
        echo "$json_content" >&2
        return 1
    fi
}

# =============================================================================
# Export Functions
# =============================================================================

# Export all assertion functions for use in tests
export -f assert_line refute_line
export -f assert_output_contains refute_output_contains
export -f assert_file_exists refute_file_exists
export -f assert_dir_exists refute_dir_exists
export -f assert_file_contains
export -f assert_variable_set refute_variable_set
export -f assert_status
export -f create_temp_file create_temp_dir create_temp_script
export -f load_official_mocks init_official_mock_logging
export -f time_command assert_command_fast
export -f save_environment restore_environment
export -f save_cwd restore_cwd
export -f standard_cleanup
export -f debug_output debug_environment
export -f skip_if_not_available skip_if_file_missing skip_if_dir_missing
export -f assert_valid_yaml assert_valid_json