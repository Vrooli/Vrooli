#!/usr/bin/env bash
# Smoke tests for NSFW Detector resource
# Quick health validation - must complete in <30s

set -euo pipefail

# Test configuration
readonly TIMEOUT=5
readonly PORT="${NSFW_DETECTOR_PORT:-11451}"
readonly BASE_URL="http://localhost:${PORT}"

# Colors for output
readonly RED='\033[0;31m'
readonly GREEN='\033[0;32m'
readonly NC='\033[0m'

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test helper function
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing $test_name... "
    
    if eval "$test_command" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Main smoke tests
main() {
    echo "Running NSFW Detector smoke tests..."
    echo "===================================="
    
    # Test 1: Health endpoint responds
    run_test "health endpoint" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/health" || true
    
    # Test 2: Health returns valid JSON
    run_test "health JSON format" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/health | jq -e '.status' > /dev/null" || true
    
    # Test 3: Models endpoint exists
    run_test "models endpoint" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/models" || true
    
    # Test 4: Service reports healthy status
    run_test "service health status" \
        "timeout $TIMEOUT curl -sf ${BASE_URL}/health | jq -e '.status == \"healthy\"' > /dev/null" || true
    
    # Test 5: Port is correct - use ss instead of netstat for better compatibility
    run_test "port configuration" \
        "ss -tln | grep -q :${PORT} || lsof -i :${PORT} &>/dev/null" || true
    
    # Summary
    echo "===================================="
    echo "Smoke Test Results:"
    echo "  Passed: $TESTS_PASSED"
    echo "  Failed: $TESTS_FAILED"
    
    if [[ $TESTS_FAILED -eq 0 ]]; then
        echo -e "${GREEN}All smoke tests passed!${NC}"
        exit 0
    else
        echo -e "${RED}Some smoke tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"