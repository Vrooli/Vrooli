#!/bin/bash

# API Manager - Integration Tests
# Tests interaction between components and with external systems

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Running API Manager Integration Tests${NC}"
echo "====================================="

# Configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT:-8080}}"
TIMEOUT=5

# Track test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test API endpoint
test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local description="$3"
    local expected_status="${4:-200}"
    
    TESTS_RUN=$((TESTS_RUN + 1))
    
    echo -n "Testing $description... "
    
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -X "$method" \
        --connect-timeout "$TIMEOUT" \
        "$API_BASE_URL$endpoint" 2>/dev/null || echo "000")
    
    if [[ "$response" == "$expected_status" ]]; then
        echo -e "${GREEN}✓ (Status: $response)${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}✗ (Expected: $expected_status, Got: $response)${NC}"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Test database connectivity
echo -e "${YELLOW}Testing database connectivity...${NC}"
test_endpoint "GET" "/health" "Database health check" "200"

# Test scenario endpoints
echo -e "${YELLOW}Testing scenario endpoints...${NC}"
test_endpoint "GET" "/api/v1/scenarios" "List scenarios" "200"
test_endpoint "GET" "/api/v1/scenarios/nonexistent" "Get non-existent scenario" "404"
test_endpoint "POST" "/api/v1/system/discover" "Discover scenarios" "200"

# Test vulnerability endpoints
echo -e "${YELLOW}Testing vulnerability endpoints...${NC}"
test_endpoint "GET" "/api/v1/vulnerabilities" "List vulnerabilities" "200"
test_endpoint "GET" "/api/v1/scans/recent" "Get recent scans" "200"

# Test health monitoring endpoints
echo -e "${YELLOW}Testing health monitoring...${NC}"
test_endpoint "GET" "/api/v1/health/summary" "Health summary" "200"
test_endpoint "GET" "/api/v1/health/alerts" "Health alerts" "200"

# Test standards compliance endpoints
echo -e "${YELLOW}Testing standards compliance...${NC}"
test_endpoint "GET" "/api/v1/standards/violations" "List standards violations" "200"

# Test system endpoints
echo -e "${YELLOW}Testing system endpoints...${NC}"
test_endpoint "GET" "/api/v1/system/status" "System status" "200"
test_endpoint "GET" "/api/v1/system/validate-lifecycle" "Validate lifecycle" "200"

# Summary
echo
echo "====================================="
echo -e "${BLUE}Integration Test Summary${NC}"
echo "====================================="
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All integration tests passed${NC}"
    exit 0
else
    echo -e "${RED}✗ Some integration tests failed${NC}"
    exit 1
fi