#!/usr/bin/env bash
################################################################################
# SimPy Unit Tests - Library function validation (<60s)
# 
# Tests individual functions and utilities
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASE_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities and libraries
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Export configuration
simpy::export_config

# Track test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "  Testing $test_name... "
    
    if eval "$test_command" &>/dev/null; then
        echo "✓"
        ((TESTS_PASSED++))
        return 0
    else
        echo "✗"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main unit test
main() {
    log::header "SimPy Unit Tests"
    
    # Test 1: Configuration export
    run_test "Configuration export" "simpy::export_config && [ -n '$SIMPY_PORT' ]"
    
    # Test 2: Directory initialization
    run_test "Directory initialization" "simpy::init && [ -d '$SIMPY_DATA_DIR' ]"
    
    # Test 3: Version detection
    run_test "Version detection" "[ '$(simpy::get_version)' != 'not_installed' ]"
    
    # Test 4: Port configuration
    run_test "Port configuration" "[ '$SIMPY_PORT' = '9510' ]"
    
    # Test 5: Host configuration
    run_test "Host configuration" "[ '$SIMPY_HOST' = 'localhost' ]"
    
    # Test 6: Python version check
    run_test "Python version requirement" "python3 --version | grep -E 'Python 3\.(8|9|10|11|12)'"
    
    # Test 7: SimPy module version
    run_test "SimPy module availability" "python3 -c 'import simpy; print(simpy.__version__)'"
    
    # Test 8: NumPy module availability
    run_test "NumPy module availability" "python3 -c 'import numpy'"
    
    # Test 9: Service script exists
    run_test "Service script exists" "[ -f '$SIMPY_DATA_DIR/simpy-service.py' ]"
    
    # Test 10: Log file creation
    run_test "Log file creation" "touch '$SIMPY_LOG_FILE' && [ -f '$SIMPY_LOG_FILE' ]"
    
    # Test 11: Results directory creation
    run_test "Results directory creation" "mkdir -p '$SIMPY_RESULTS_DIR' && [ -d '$SIMPY_RESULTS_DIR' ]"
    
    # Test 12: Examples directory exists
    run_test "Examples directory exists" "[ -d '$SIMPY_EXAMPLES_DIR' ]"
    
    # Test 13: List examples function
    run_test "List examples function" "simpy::list_examples"
    
    # Test 14: Data directory permissions
    run_test "Data directory writable" "touch '$SIMPY_DATA_DIR/test.tmp' && rm '$SIMPY_DATA_DIR/test.tmp'"
    
    # Test 15: Configuration values
    run_test "Default timeout configured" "[ '$SIMPY_DEFAULT_TIMEOUT' = '3600' ]"
    
    # Summary
    echo ""
    log::info "Test Summary:"
    log::success "  Passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log::error "  Failed: $TESTS_FAILED"
        exit 1
    fi
    
    log::success "All unit tests passed!"
}

main "$@"