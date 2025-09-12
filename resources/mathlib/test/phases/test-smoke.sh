#!/usr/bin/env bash
# Mathlib Resource - Smoke Tests
# Quick validation that service is responsive (< 30s)

set -euo pipefail

# Source test library
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

# Main smoke test
main() {
    echo "Mathlib Smoke Tests"
    echo "==================="
    
    # Ensure service is started for tests
    echo "Starting service for tests..."
    mathlib::stop > /dev/null 2>&1 || true
    if ! mathlib::start --wait > /dev/null 2>&1; then
        echo "Failed to start service"
        exit 1
    fi
    
    # Run smoke tests
    run_test "Health endpoint" mathlib::test::health
    run_test "Configuration valid" mathlib::test::config
    run_test "CLI help works" "${TEST_DIR}/../cli.sh" help
    run_test "CLI info works" "${TEST_DIR}/../cli.sh" info
    run_test "CLI status works" "${TEST_DIR}/../cli.sh" status
    
    # Stop service
    echo "Stopping service..."
    mathlib::stop > /dev/null 2>&1 || true
    
    # Summary
    echo ""
    echo "Smoke Tests: ${TESTS_PASSED}/${TESTS_RUN} passed"
    
    if [[ ${TESTS_PASSED} -eq ${TESTS_RUN} ]]; then
        exit 0
    else
        exit 1
    fi
}

# Execute with timeout
timeout 30 bash -c "$(declare -f main); main" || {
    echo "Smoke tests timed out (>30s)"
    exit 1
}