#!/usr/bin/env bash
################################################################################
# SimPy Smoke Tests - Quick health validation (<30s)
# 
# Tests basic service health and responsiveness
################################################################################

set -euo pipefail

# Get directories
PHASE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DIR="$(cd "$PHASE_DIR/.." && pwd)"
RESOURCE_DIR="$(cd "$TEST_DIR/.." && pwd)"
APP_ROOT="$(cd "$RESOURCE_DIR/../.." && pwd)"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

# Export configuration
simpy::export_config

# Test timeout
TIMEOUT_CMD="timeout 5"

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

# Main smoke test
main() {
    log::header "SimPy Smoke Tests"
    
    # Test 1: Check if SimPy is installed
    run_test "SimPy installation" "simpy::is_installed"
    
    # Test 2: Check Python availability
    run_test "Python3 availability" "command -v python3"
    
    # Test 3: Check SimPy module import
    run_test "SimPy module import" "python3 -c 'import simpy'"
    
    # Test 4: Start service if not running
    if ! simpy::is_running; then
        log::info "Starting SimPy service for testing..."
        if ! simpy::docker::start; then
            log::error "Failed to start SimPy service"
            exit 1
        fi
        # Wait for service to be ready
        sleep 3
    fi
    
    # Test 5: Health endpoint
    run_test "Health endpoint" "$TIMEOUT_CMD curl -sf http://localhost:${SIMPY_PORT}/health"
    
    # Test 6: Version endpoint
    run_test "Version endpoint" "$TIMEOUT_CMD curl -sf http://localhost:${SIMPY_PORT}/version"
    
    # Test 7: Examples endpoint
    run_test "Examples endpoint" "$TIMEOUT_CMD curl -sf http://localhost:${SIMPY_PORT}/examples"
    
    # Test 8: Service responds with correct structure
    run_test "Health response structure" "$TIMEOUT_CMD curl -sf http://localhost:${SIMPY_PORT}/health | grep -q 'status'"
    
    # Test 9: Port is correct
    run_test "Service port configuration" "[ '$SIMPY_PORT' = '9510' ]"
    
    # Test 10: Data directory exists
    run_test "Data directory exists" "[ -d '$SIMPY_DATA_DIR' ]"
    
    # Summary
    echo ""
    log::info "Test Summary:"
    log::success "  Passed: $TESTS_PASSED"
    if [[ $TESTS_FAILED -gt 0 ]]; then
        log::error "  Failed: $TESTS_FAILED"
        exit 1
    fi
    
    log::success "All smoke tests passed!"
}

main "$@"