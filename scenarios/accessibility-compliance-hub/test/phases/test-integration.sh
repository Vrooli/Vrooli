#!/bin/bash
# Integration tests for accessibility-compliance-hub scenario

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Configuration
readonly API_BASE="http://localhost:${API_PORT:-18000}"
readonly TEST_TIMEOUT=30

# Test counter
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_start() {
    local test_name="$1"
    echo -e "\n${YELLOW}Testing: ${test_name}...${NC}"
    ((TESTS_TOTAL++))
}

test_pass() {
    echo -e "${GREEN}✓ PASSED${NC}"
    ((TESTS_PASSED++))
}

test_fail() {
    local reason="${1:-Unknown reason}"
    echo -e "${RED}✗ FAILED: ${reason}${NC}"
    ((TESTS_FAILED++))
}

# Wait for API to be ready
wait_for_api() {
    local max_attempts=30
    local attempt=0

    echo -e "${YELLOW}Waiting for API to be ready...${NC}"

    while [ $attempt -lt $max_attempts ]; do
        if curl -sf "${API_BASE}/health" > /dev/null 2>&1; then
            echo -e "${GREEN}API is ready${NC}"
            return 0
        fi
        sleep 1
        ((attempt++))
    done

    echo -e "${RED}API failed to start after ${max_attempts} seconds${NC}"
    return 1
}

# Test health endpoint
test_health() {
    test_start "Health endpoint returns valid status"

    if response=$(curl -sf "${API_BASE}/health" 2>/dev/null); then
        status=$(echo "$response" | jq -r '.status' 2>/dev/null || echo "")
        if [ "$status" = "healthy" ]; then
            test_pass
        else
            test_fail "Status not healthy: $status"
        fi
    else
        test_fail "Failed to connect to health endpoint"
    fi
}

# Test scans endpoint (mock data)
test_scans_endpoint() {
    test_start "Scans endpoint returns mock data"

    if response=$(curl -sf "${API_BASE}/api/scans" 2>/dev/null); then
        count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
        if [ "$count" -gt 0 ]; then
            test_pass
        else
            test_fail "No scans returned"
        fi
    else
        test_fail "Scans endpoint failed"
    fi
}

# Test violations endpoint (mock data)
test_violations_endpoint() {
    test_start "Violations endpoint returns mock data"

    if response=$(curl -sf "${API_BASE}/api/violations" 2>/dev/null); then
        count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
        if [ "$count" -gt 0 ]; then
            test_pass
        else
            test_fail "No violations returned"
        fi
    else
        test_fail "Violations endpoint failed"
    fi
}

# Test reports endpoint (mock data)
test_reports_endpoint() {
    test_start "Reports endpoint returns mock data"

    if response=$(curl -sf "${API_BASE}/api/reports" 2>/dev/null); then
        count=$(echo "$response" | jq 'length' 2>/dev/null || echo "0")
        if [ "$count" -gt 0 ]; then
            test_pass
        else
            test_fail "No reports returned"
        fi
    else
        test_fail "Reports endpoint failed"
    fi
}

# Test content-type headers
test_content_types() {
    test_start "Endpoints return proper Content-Type headers"

    for endpoint in "/health" "/api/scans" "/api/violations" "/api/reports"; do
        content_type=$(curl -sf -I "${API_BASE}${endpoint}" 2>/dev/null | grep -i "content-type" | awk '{print $2}' | tr -d '\r')
        if [[ "$content_type" == application/json* ]]; then
            continue
        else
            test_fail "Endpoint $endpoint has wrong content-type: $content_type"
            return
        fi
    done
    test_pass
}

# Main test execution
main() {
    echo -e "${YELLOW}=== Accessibility Compliance Hub Integration Tests ===${NC}"
    echo -e "${YELLOW}Note: Testing prototype with mock data endpoints${NC}\n"

    # Wait for API
    if ! wait_for_api; then
        echo -e "${RED}Cannot run tests - API is not available${NC}"
        exit 1
    fi

    # Run tests
    test_health
    test_scans_endpoint
    test_violations_endpoint
    test_reports_endpoint
    test_content_types

    # Print summary
    echo -e "\n${YELLOW}=== Test Summary ===${NC}"
    echo -e "Total tests: ${TESTS_TOTAL}"
    echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"

    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All integration tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some integration tests failed!${NC}"
        exit 1
    fi
}

main "$@"
