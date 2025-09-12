#!/usr/bin/env bash
# Mathlib Resource - Integration Tests
# End-to-end functionality tests (< 120s)

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

# Test full lifecycle
test_full_lifecycle() {
    # Clean state
    mathlib::stop > /dev/null 2>&1 || true
    
    # Install
    mathlib::install > /dev/null 2>&1 || return 1
    
    # Start
    mathlib::start --wait > /dev/null 2>&1 || return 1
    
    # Check health
    mathlib::test::health > /dev/null 2>&1 || return 1
    
    # Restart
    mathlib::restart --wait > /dev/null 2>&1 || return 1
    
    # Stop
    mathlib::stop > /dev/null 2>&1 || return 1
    
    return 0
}

# Test API endpoints
test_api_endpoints() {
    # Start service
    mathlib::start --wait > /dev/null 2>&1 || return 1
    
    # Test health endpoint
    local response=$(timeout 5 curl -sf "http://localhost:${MATHLIB_PORT}/health" 2>/dev/null)
    [[ -n "${response}" ]] || return 1
    
    # Validate JSON response
    echo "${response}" | jq -e '.status' > /dev/null 2>&1 || return 1
    
    # Stop service
    mathlib::stop > /dev/null 2>&1
    
    return 0
}

# Test resource info command
test_resource_info() {
    # Test text output
    mathlib::info > /dev/null 2>&1 || return 1
    
    # Test JSON output
    local json_output=$(mathlib::info --json)
    echo "${json_output}" | jq -e '.resource' > /dev/null 2>&1 || return 1
    
    return 0
}

# Test log functionality
test_logging() {
    # Start service to generate logs
    mathlib::start --wait > /dev/null 2>&1 || return 1
    
    # Check logs exist
    mathlib::logs --tail 10 > /dev/null 2>&1 || return 1
    
    # Stop service
    mathlib::stop > /dev/null 2>&1
    
    return 0
}

# Test contract compliance
test_contract_compliance() {
    # Required commands must exist
    "${TEST_DIR}/../cli.sh" help > /dev/null 2>&1 || return 1
    "${TEST_DIR}/../cli.sh" info > /dev/null 2>&1 || return 1
    "${TEST_DIR}/../cli.sh" manage install --help > /dev/null 2>&1 || return 1
    "${TEST_DIR}/../cli.sh" test smoke > /dev/null 2>&1 || return 1
    "${TEST_DIR}/../cli.sh" status > /dev/null 2>&1 || return 1
    
    # Required files must exist
    [[ -f "${TEST_DIR}/../config/runtime.json" ]] || return 1
    [[ -f "${TEST_DIR}/../config/schema.json" ]] || return 1
    [[ -f "${TEST_DIR}/../config/defaults.sh" ]] || return 1
    
    return 0
}

# Main integration test
main() {
    echo "Mathlib Integration Tests"
    echo "========================="
    
    # Clean initial state
    mathlib::stop > /dev/null 2>&1 || true
    
    # Run integration tests
    run_test "Full lifecycle" test_full_lifecycle
    run_test "API endpoints" test_api_endpoints
    run_test "Resource info" test_resource_info
    run_test "Logging" test_logging
    run_test "v2.0 contract compliance" test_contract_compliance
    run_test "Lifecycle operations" mathlib::test::lifecycle
    
    # Clean final state
    mathlib::stop > /dev/null 2>&1 || true
    
    # Summary
    echo ""
    echo "Integration Tests: ${TESTS_PASSED}/${TESTS_RUN} passed"
    
    if [[ ${TESTS_PASSED} -eq ${TESTS_RUN} ]]; then
        exit 0
    else
        exit 1
    fi
}

# Execute with timeout
timeout 120 bash -c "$(declare -f main); $(declare -f run_test); $(declare -f test_full_lifecycle); $(declare -f test_api_endpoints); $(declare -f test_resource_info); $(declare -f test_logging); $(declare -f test_contract_compliance); main" || {
    echo "Integration tests timed out (>120s)"
    exit 1
}