#!/usr/bin/env bash
# Mathlib Resource - Unit Tests
# Test library functions in isolation (< 60s)

set -euo pipefail

# Source libraries
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source "${TEST_DIR}/../lib/test.sh"
source "${TEST_DIR}/../lib/core.sh"

# Test counter
TESTS_RUN=0
TESTS_PASSED=0

# Run a test
run_test() {
    local test_name="$1"
    shift
    
    TESTS_RUN=$((TESTS_RUN + 1))
    echo -n "  ${test_name}... "
    
    if "$@" > /dev/null 2>&1; then
        echo "PASS"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo "FAIL"
    fi
}

# Test configuration loading
test_config_loading() {
    # Check defaults are loaded
    [[ -n "${MATHLIB_PORT}" ]] || return 1
    [[ -n "${MATHLIB_WORK_DIR}" ]] || return 1
    [[ -n "${MATHLIB_CACHE_DIR}" ]] || return 1
    return 0
}

# Test directory creation
test_directory_creation() {
    # Test that install creates directories
    local test_dir="/tmp/mathlib_test_$$"
    MATHLIB_INSTALL_DIR="${test_dir}/install"
    MATHLIB_WORK_DIR="${test_dir}/work"
    MATHLIB_CACHE_DIR="${test_dir}/cache"
    
    mathlib::install > /dev/null 2>&1
    
    local result=0
    [[ -d "${MATHLIB_INSTALL_DIR}" ]] || result=1
    [[ -d "${MATHLIB_WORK_DIR}" ]] || result=1
    [[ -d "${MATHLIB_CACHE_DIR}" ]] || result=1
    
    # Cleanup
    rm -rf "${test_dir}"
    
    return ${result}
}

# Test PID file management
test_pid_management() {
    local test_pid_file="/tmp/mathlib_test_pid_$$"
    MATHLIB_PID_FILE="${test_pid_file}"
    
    # Write test PID
    echo "12345" > "${MATHLIB_PID_FILE}"
    
    # Check if detected as not running (invalid PID)
    if mathlib::is_running; then
        rm -f "${test_pid_file}"
        return 1
    fi
    
    # Cleanup
    rm -f "${test_pid_file}"
    return 0
}

# Test JSON parsing
test_json_parsing() {
    # Test runtime.json is valid JSON
    local runtime_file="${TEST_DIR}/../config/runtime.json"
    jq -e . "${runtime_file}" > /dev/null 2>&1 || return 1
    
    # Test specific fields exist
    jq -e '.startup_order' "${runtime_file}" > /dev/null 2>&1 || return 1
    jq -e '.dependencies' "${runtime_file}" > /dev/null 2>&1 || return 1
    
    return 0
}

# Main unit test
main() {
    echo "Mathlib Unit Tests"
    echo "=================="
    
    # Run unit tests
    run_test "Configuration loading" test_config_loading
    run_test "Directory creation" test_directory_creation
    run_test "PID management" test_pid_management
    run_test "JSON parsing" test_json_parsing
    run_test "CLI parsing" mathlib::test::cli
    
    # Summary
    echo ""
    echo "Unit Tests: ${TESTS_PASSED}/${TESTS_RUN} passed"
    
    if [[ ${TESTS_PASSED} -eq ${TESTS_RUN} ]]; then
        exit 0
    else
        exit 1
    fi
}

# Execute with timeout
timeout 60 bash -c "$(declare -f main); $(declare -f run_test); $(declare -f test_config_loading); $(declare -f test_directory_creation); $(declare -f test_pid_management); $(declare -f test_json_parsing); main" || {
    echo "Unit tests timed out (>60s)"
    exit 1
}