#!/bin/bash

# API Manager - Smoke Tests
# Quick validation that the service is running and basic functionality works

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}Running API Manager Smoke Tests${NC}"
echo "================================"

# Configuration
API_BASE_URL="${API_MANAGER_URL:-http://localhost:${API_PORT:-8080}}"
TIMEOUT=3

# Track test results
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test 1: Service is reachable
echo -n "1. Service is reachable... "
TESTS_RUN=$((TESTS_RUN + 1))
if curl -s --connect-timeout "$TIMEOUT" "$API_BASE_URL/health" > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Cannot reach API Manager at $API_BASE_URL${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    echo "  Ensure the service is running: vrooli scenario run api-manager"
    exit 1
fi

# Test 2: Health endpoint returns valid JSON
echo -n "2. Health endpoint returns valid JSON... "
TESTS_RUN=$((TESTS_RUN + 1))
if curl -s "$API_BASE_URL/health" | jq -e '.status' > /dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ Invalid health response${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 3: API v1 endpoints are accessible
echo -n "3. API v1 endpoints are accessible... "
TESTS_RUN=$((TESTS_RUN + 1))
response=$(curl -s -o /dev/null -w "%{http_code}" "$API_BASE_URL/api/v1/health" 2>/dev/null || echo "000")
if [[ "$response" == "200" || "$response" == "503" ]]; then
    echo -e "${GREEN}✓${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ API v1 not accessible (Status: $response)${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 4: Required dependencies check
echo -n "4. Checking dependencies... "
TESTS_RUN=$((TESTS_RUN + 1))
health_response=$(curl -s "$API_BASE_URL/health" 2>/dev/null || echo "{}")
if echo "$health_response" | jq -e '.dependencies.database' > /dev/null 2>&1; then
    db_status=$(echo "$health_response" | jq -r '.dependencies.database.status')
    if [[ "$db_status" == "healthy" ]]; then
        echo -e "${GREEN}✓ Database connected${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "${YELLOW}⚠ Database not healthy (status: $db_status)${NC}"
        TESTS_PASSED=$((TESTS_PASSED + 1))  # Warning, not failure
    fi
else
    echo -e "${RED}✗ Cannot check dependencies${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 5: CORS headers present
echo -n "5. CORS headers configured... "
TESTS_RUN=$((TESTS_RUN + 1))
headers=$(curl -sI "$API_BASE_URL/api/v1/health" 2>/dev/null || echo "")
if echo "$headers" | grep -qi "Access-Control-Allow-Origin"; then
    echo -e "${GREEN}✓${NC}"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗ CORS headers not found${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Summary
echo
echo "================================"
echo -e "${BLUE}Smoke Test Summary${NC}"
echo "================================"
echo "Tests Run: $TESTS_RUN"
echo "Tests Passed: $TESTS_PASSED"
echo "Tests Failed: $TESTS_FAILED"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✓ All smoke tests passed - service is operational${NC}"
    exit 0
else
    echo -e "${RED}✗ Some smoke tests failed - service may have issues${NC}"
    exit 1
fi