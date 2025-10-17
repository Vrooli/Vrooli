#!/bin/bash
# ====================================================================
# Network Tools Scenario Integration Test
# ====================================================================
#
# This test validates the network operations and testing capabilities
# including HTTP requests, DNS operations, SSL validation, and API testing.
#
# ====================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
# Try to detect the actual running port
if ps aux | grep -q "[n]etwork-tools-api"; then
    ACTUAL_PORT=$(lsof -i -P -n | grep "network-t.*LISTEN" | head -1 | awk '{print $9}' | cut -d: -f2)
    API_PORT="${API_PORT:-${ACTUAL_PORT:-17177}}"
else
    API_PORT="${API_PORT:-17177}"
fi
API_BASE="http://localhost:${API_PORT}"
TEST_RESULTS=()
FAILED_TESTS=()

# Helper function to test endpoint
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="${4:-}"
    local expected_status="${5:-200}"
    
    echo -e "${BLUE}Testing: ${name}...${NC}"
    
    if [[ -n "$data" ]]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "${API_BASE}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data" 2>/dev/null || echo "CURL_ERROR")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "${API_BASE}${endpoint}" 2>/dev/null || echo "CURL_ERROR")
    fi
    
    if [[ "$response" == "CURL_ERROR" ]]; then
        echo -e "${RED}  ✗ Failed to connect to API${NC}"
        FAILED_TESTS+=("$name")
        return 1
    fi
    
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [[ "$status_code" == "$expected_status" ]]; then
        echo -e "${GREEN}  ✓ ${name} passed (HTTP ${status_code})${NC}"
        TEST_RESULTS+=("PASS: $name")
        return 0
    else
        echo -e "${RED}  ✗ ${name} failed (expected ${expected_status}, got ${status_code})${NC}"
        FAILED_TESTS+=("$name")
        return 1
    fi
}

echo "======================================================================"
echo "Network Tools Integration Test"
echo "API Endpoint: ${API_BASE}"
echo "======================================================================"
echo ""

# Test health endpoint
test_endpoint "Health Check" "GET" "/health" "" "200"

# Test HTTP client endpoint
test_endpoint "HTTP Client - GET" "POST" "/api/v1/network/http" \
    '{"url":"https://httpbin.org/get","method":"GET"}' "200"

# Test DNS lookup
test_endpoint "DNS Lookup" "POST" "/api/v1/network/dns" \
    '{"query":"google.com","record_type":"A"}' "200"

# Test SSL validation
test_endpoint "SSL Validation" "POST" "/api/v1/network/ssl/validate" \
    '{"url":"https://google.com"}' "200"

# Test connectivity
test_endpoint "Connectivity Test" "POST" "/api/v1/network/test/connectivity" \
    '{"target":"8.8.8.8","test_type":"ping","options":{"count":1}}' "200"

# Test port scanning (simple test)
test_endpoint "Port Scan" "POST" "/api/v1/network/scan" \
    '{"target":"google.com","scan_type":"port","ports":[80,443]}' "200"

# Test API testing endpoint
test_endpoint "API Test" "POST" "/api/v1/network/api/test" \
    '{"base_url":"https://httpbin.org","test_suite":[{"endpoint":"/get","method":"GET","test_cases":[{"name":"Basic GET","input":{},"expected_status":200}]}]}' "200"

echo ""
echo "======================================================================"
echo "Test Summary"
echo "======================================================================"

total_tests=$((${#TEST_RESULTS[@]} + ${#FAILED_TESTS[@]}))
passed_tests=${#TEST_RESULTS[@]}
failed_tests=${#FAILED_TESTS[@]}

echo "Total Tests: ${total_tests}"
echo -e "${GREEN}Passed: ${passed_tests}${NC}"
echo -e "${RED}Failed: ${failed_tests}${NC}"

if [[ ${#FAILED_TESTS[@]} -gt 0 ]]; then
    echo ""
    echo "Failed Tests:"
    for test in "${FAILED_TESTS[@]}"; do
        echo "  - $test"
    done
    exit 1
fi

echo -e "${GREEN}All tests passed!${NC}"
exit 0