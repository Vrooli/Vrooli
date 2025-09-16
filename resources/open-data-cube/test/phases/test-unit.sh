#!/bin/bash

# Open Data Cube Unit Tests
# Tests for library functions and configurations (<60s)

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
source "${RESOURCE_DIR}/config/defaults.sh"

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Test function wrapper
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    
    echo -n "  ${test_name}... "
    
    if eval "$test_cmd" &>/dev/null; then
        echo "✓"
        ((TESTS_PASSED++))
        return 0
    else
        echo "✗"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Main unit test execution
main() {
    echo "Running Open Data Cube unit tests..."
    echo ""
    
    local start_time=$(date +%s)
    
    # Test 1: CLI script exists and is executable
    run_test "CLI script exists" "[[ -f '${RESOURCE_DIR}/cli.sh' ]]"
    run_test "CLI is executable" "[[ -x '${RESOURCE_DIR}/cli.sh' ]]"
    
    # Test 2: Core library exists
    run_test "Core library exists" "[[ -f '${RESOURCE_DIR}/lib/core.sh' ]]"
    
    # Test 3: Test library exists
    run_test "Test library exists" "[[ -f '${RESOURCE_DIR}/lib/test.sh' ]]"
    
    # Test 4: Configuration files exist
    run_test "defaults.sh exists" "[[ -f '${RESOURCE_DIR}/config/defaults.sh' ]]"
    run_test "runtime.json exists" "[[ -f '${RESOURCE_DIR}/config/runtime.json' ]]"
    run_test "schema.json exists" "[[ -f '${RESOURCE_DIR}/config/schema.json' ]]"
    
    # Test 5: Test infrastructure exists
    run_test "Test runner exists" "[[ -f '${RESOURCE_DIR}/test/run-tests.sh' ]]"
    run_test "Smoke tests exist" "[[ -f '${RESOURCE_DIR}/test/phases/test-smoke.sh' ]]"
    run_test "Integration tests exist" "[[ -f '${RESOURCE_DIR}/test/phases/test-integration.sh' ]]"
    
    # Test 6: Directory structure
    run_test "Docker directory exists" "[[ -d '${RESOURCE_DIR}/docker' ]]"
    run_test "Data directory exists" "[[ -d '${RESOURCE_DIR}/data' ]]"
    run_test "Examples directory exists" "[[ -d '${RESOURCE_DIR}/examples' ]]"
    run_test "Docs directory exists" "[[ -d '${RESOURCE_DIR}/docs' ]]"
    
    # Test 7: Validate JSON files
    run_test "runtime.json is valid JSON" "python3 -m json.tool '${RESOURCE_DIR}/config/runtime.json' > /dev/null"
    run_test "schema.json is valid JSON" "python3 -m json.tool '${RESOURCE_DIR}/config/schema.json' > /dev/null"
    
    # Test 8: Check runtime.json required fields
    run_test "runtime.json has name field" "grep -q '\"name\"' '${RESOURCE_DIR}/config/runtime.json'"
    run_test "runtime.json has dependencies" "grep -q '\"dependencies\"' '${RESOURCE_DIR}/config/runtime.json'"
    run_test "runtime.json has startup_order" "grep -q '\"startup_order\"' '${RESOURCE_DIR}/config/runtime.json'"
    
    # Test 9: Environment variables
    run_test "Resource name defined" "[[ -n '${ODC_RESOURCE_NAME}' ]]"
    run_test "Version defined" "[[ -n '${ODC_VERSION}' ]]"
    run_test "Container prefix defined" "[[ -n '${ODC_CONTAINER_PREFIX}' ]]"
    
    # Test 10: Port registry integration
    run_test "Port registry exists" "[[ -f '${VROOLI_ROOT:-${HOME}/Vrooli}/scripts/resources/port_registry.sh' ]]"
    
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    echo ""
    echo "----------------------------------------"
    echo "Unit Test Results:"
    echo "  Passed: ${TESTS_PASSED}"
    echo "  Failed: ${TESTS_FAILED}"
    echo "  Duration: ${duration}s"
    
    if [[ ${TESTS_FAILED} -eq 0 ]]; then
        echo "  Status: SUCCESS"
        echo "----------------------------------------"
        return 0
    else
        echo "  Status: FAILURE"
        echo "----------------------------------------"
        return 1
    fi
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi