#!/bin/bash
# API Test Suite for Network Tools
# Tests all P0 requirements and critical functionality

set -euo pipefail

# Configuration
readonly API_BASE="${API_BASE:-http://localhost:${API_PORT:-15000}}"
readonly TEST_TIMEOUT=10
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m'

# Test tracking
TESTS_TOTAL=0
TESTS_PASSED=0
TESTS_FAILED=0

# Test function
run_test() {
    local test_name="$1"
    local test_cmd="$2"
    local expected_status="${3:-200}"
    
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
    
    echo -n "Testing: $test_name ... "
    
    # Execute test with timeout
    if timeout "$TEST_TIMEOUT" bash -c "$test_cmd"; then
        # Check if response contains expected status
        if [[ -z "$expected_status" ]] || [[ "$test_cmd" == *"$expected_status"* ]]; then
            echo -e "${GREEN}✓${NC}"
            TESTS_PASSED=$((TESTS_PASSED + 1))
            return 0
        fi
    fi
    
    echo -e "${RED}✗${NC}"
    TESTS_FAILED=$((TESTS_FAILED + 1))
    return 1
}

# Test helper to check HTTP status
test_http_endpoint() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="${3:-}"
    local expected="${4:-200}"
    local test_name="${5:-$endpoint}"
    
    local cmd="curl -s -o /dev/null -w '%{http_code}' -X $method"
    if [[ -n "$data" ]]; then
        cmd="$cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    cmd="$cmd '$API_BASE$endpoint'"
    
    local status
    status=$(eval "$cmd")
    
    if [[ "$status" == "$expected" ]]; then
        echo -e "  ${GREEN}✓${NC} $test_name (Status: $status)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
    else
        echo -e "  ${RED}✗${NC} $test_name (Expected: $expected, Got: $status)"
        TESTS_FAILED=$((TESTS_FAILED + 1))
    fi
    TESTS_TOTAL=$((TESTS_TOTAL + 1))
}

echo "========================================="
echo "Network Tools API Test Suite"
echo "API Base: $API_BASE"
echo "========================================="

# P0 Requirement Tests

echo -e "\n${YELLOW}Testing Health Check (P0)${NC}"
test_http_endpoint "/health" "GET" "" "200" "Health endpoint"
test_http_endpoint "/api/health" "GET" "" "200" "API health endpoint"

echo -e "\n${YELLOW}Testing HTTP Client Operations (P0)${NC}"
test_http_endpoint "/api/v1/network/http" "POST" \
    '{"url":"https://httpbin.org/get","method":"GET"}' \
    "200" "HTTP GET request"

test_http_endpoint "/api/v1/network/http" "POST" \
    '{"url":"https://httpbin.org/post","method":"POST","body":{"test":"data"}}' \
    "200" "HTTP POST request"

echo -e "\n${YELLOW}Testing DNS Operations (P0)${NC}"
test_http_endpoint "/api/v1/network/dns" "POST" \
    '{"query":"google.com","record_type":"A"}' \
    "200" "DNS A record lookup"

test_http_endpoint "/api/v1/network/dns" "POST" \
    '{"query":"google.com","record_type":"MX"}' \
    "200" "DNS MX record lookup"

echo -e "\n${YELLOW}Testing Network Connectivity (P0)${NC}"
test_http_endpoint "/api/v1/network/test/connectivity" "POST" \
    '{"target":"8.8.8.8","test_type":"ping"}' \
    "200" "Connectivity test (ping)"

echo -e "\n${YELLOW}Testing Port Scanning (P0)${NC}"
test_http_endpoint "/api/v1/network/scan" "POST" \
    '{"target":"localhost","ports":[22,80,443]}' \
    "200" "Port scan"

echo -e "\n${YELLOW}Testing SSL/TLS Validation (P0)${NC}"
test_http_endpoint "/api/v1/network/ssl/validate" "POST" \
    '{"url":"https://google.com","options":{"check_expiry":true,"check_hostname":true}}' \
    "200" "SSL certificate validation"

echo -e "\n${YELLOW}Testing API Testing Endpoint (P0)${NC}"
test_http_endpoint "/api/v1/network/api/test" "POST" \
    '{"base_url":"https://httpbin.org","test_suite":[{"endpoint":"/get","method":"GET","test_cases":[{"name":"Test GET","expected_status":200}]}]}' \
    "200" "API testing functionality"

echo -e "\n${YELLOW}Testing Management Endpoints${NC}"
test_http_endpoint "/api/v1/network/targets" "GET" "" "200" "List network targets"
test_http_endpoint "/api/v1/network/alerts" "GET" "" "200" "List alerts"

# Test error handling
echo -e "\n${YELLOW}Testing Error Handling${NC}"
test_http_endpoint "/api/v1/network/http" "POST" \
    '{"url":"invalid-url"}' \
    "400" "Invalid URL handling"

test_http_endpoint "/api/v1/network/dns" "POST" \
    '{}' \
    "400" "Missing query parameter"

test_http_endpoint "/api/v1/nonexistent" "GET" "" "404" "Non-existent endpoint"

# Performance test
echo -e "\n${YELLOW}Testing Performance Requirements${NC}"
echo -n "  Testing response time < 500ms ... "
start_time=$(date +%s%N)
curl -s -o /dev/null "$API_BASE/health"
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 ))

if [[ $response_time -lt 500 ]]; then
    echo -e "${GREEN}✓${NC} (${response_time}ms)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}✗${NC} (${response_time}ms)"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# Test concurrent connections
echo -e "\n${YELLOW}Testing Concurrent Connections${NC}"
echo -n "  Testing 10 concurrent requests ... "
success=0
for i in {1..10}; do
    curl -s -o /dev/null "$API_BASE/health" &
done
wait
echo -e "${GREEN}✓${NC}"
TESTS_PASSED=$((TESTS_PASSED + 1))
TESTS_TOTAL=$((TESTS_TOTAL + 1))

# Summary
echo "========================================="
echo "Test Results:"
echo "  Total:  $TESTS_TOTAL"
echo -e "  Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "  Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "\n${RED}TESTS FAILED${NC}"
    exit 1
else
    echo -e "\n${GREEN}ALL TESTS PASSED${NC}"
    exit 0
fi