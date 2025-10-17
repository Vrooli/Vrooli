#!/usr/bin/env bash
# farmOS Unit Tests - Library function validation

set -euo pipefail

# Get resource directory
RESOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

# Source libraries
source "${RESOURCE_DIR}/config/defaults.sh"
source "${RESOURCE_DIR}/lib/core.sh"

echo "Running farmOS unit tests..."

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
test_function() {
    local test_name="$1"
    local expected_result="$2"
    shift 2
    
    echo -n "Testing ${test_name}... "
    
    # Run the test command with timeout
    if timeout 2 bash -c "$@" > /dev/null 2>&1; then
        ACTUAL_RESULT=0
    else
        ACTUAL_RESULT=$?
    fi
    
    if [[ $ACTUAL_RESULT -eq $expected_result ]]; then
        echo "✓ PASS"
        ((TESTS_PASSED++))
    else
        echo "✗ FAIL (expected $expected_result, got $ACTUAL_RESULT)"
        ((TESTS_FAILED++))
    fi
}

# Test 1: Help function
test_function "farmos::help" 0 "RESOURCE_DIR='${RESOURCE_DIR}' && source ${RESOURCE_DIR}/lib/core.sh && farmos::help"

# Test 2: Info function (text format)
test_function "farmos::info (text)" 0 "RESOURCE_DIR='${RESOURCE_DIR}' && source ${RESOURCE_DIR}/lib/core.sh && farmos::info"

# Test 3: Info function (JSON format)
test_function "farmos::info (json)" 0 "RESOURCE_DIR='${RESOURCE_DIR}' && source ${RESOURCE_DIR}/lib/core.sh && farmos::info --json"

# Test 4: Health check function (may fail if not running)
echo -n "Testing farmos::health_check... "
if farmos::health_check; then
    echo "✓ PASS (service running)"
    ((TESTS_PASSED++))
else
    echo "⊘ SKIP (service not running)"
fi

# Test 5: Status function
test_function "farmos::status" 0 "RESOURCE_DIR='${RESOURCE_DIR}' && source ${RESOURCE_DIR}/lib/core.sh && farmos::status"

# Test 6: Credentials function
test_function "farmos::credentials" 0 "RESOURCE_DIR='${RESOURCE_DIR}' && source ${RESOURCE_DIR}/lib/core.sh && farmos::credentials"

# Test 7: Configuration variables
echo -n "Testing configuration variables... "
if [[ -n "$FARMOS_PORT" ]] && [[ -n "$FARMOS_BASE_URL" ]] && [[ -n "$FARMOS_API_BASE" ]]; then
    echo "✓ PASS"
    ((TESTS_PASSED++))
else
    echo "✗ FAIL - Missing configuration variables"
    ((TESTS_FAILED++))
fi

# Test 8: Docker compose file exists
echo -n "Testing Docker compose file... "
if [[ -f "$FARMOS_COMPOSE_FILE" ]]; then
    echo "✓ PASS"
    ((TESTS_PASSED++))
else
    echo "✗ FAIL - Docker compose file not found"
    ((TESTS_FAILED++))
fi

# Test 9: Runtime configuration
echo -n "Testing runtime.json... "
if [[ -f "${RESOURCE_DIR}/config/runtime.json" ]]; then
    # Validate JSON structure
    if python3 -m json.tool "${RESOURCE_DIR}/config/runtime.json" > /dev/null 2>&1; then
        echo "✓ PASS"
        ((TESTS_PASSED++))
    else
        echo "✗ FAIL - Invalid JSON"
        ((TESTS_FAILED++))
    fi
else
    echo "✗ FAIL - File not found"
    ((TESTS_FAILED++))
fi

# Test 10: Schema validation
echo -n "Testing schema.json... "
if [[ -f "${RESOURCE_DIR}/config/schema.json" ]]; then
    # Validate JSON structure
    if python3 -m json.tool "${RESOURCE_DIR}/config/schema.json" > /dev/null 2>&1; then
        echo "✓ PASS"
        ((TESTS_PASSED++))
    else
        echo "✗ FAIL - Invalid JSON"
        ((TESTS_FAILED++))
    fi
else
    echo "✗ FAIL - File not found"
    ((TESTS_FAILED++))
fi

# Summary
echo ""
echo "Unit tests completed:"
echo "  Passed: $TESTS_PASSED"
echo "  Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    exit 0
else
    exit 1
fi