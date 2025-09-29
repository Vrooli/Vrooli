#!/bin/bash
# Integration tests for network-tools scenario

set -euo pipefail

# Colors for output
readonly GREEN='\033[0;32m'
readonly RED='\033[0;31m'
readonly YELLOW='\033[1;33m'
readonly NC='\033[0m' # No Color

# Configuration
readonly API_BASE="http://localhost:${API_PORT:-15000}"
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
    test_start "Health endpoint"
    
    response=$(curl -sf "${API_BASE}/health" 2>/dev/null)
    if [ $? -eq 0 ]; then
        status=$(echo "$response" | jq -r '.status' 2>/dev/null)
        if [ "$status" = "healthy" ]; then
            test_pass
        else
            test_fail "Status not healthy: $status"
        fi
    else
        test_fail "Failed to connect to health endpoint"
    fi
}

# Test HTTP endpoint
test_http_request() {
    test_start "HTTP request endpoint"
    
    payload='{
        "url": "https://httpbin.org/get",
        "method": "GET"
    }'
    
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/network/http" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        status_code=$(echo "$response" | jq -r '.data.status_code' 2>/dev/null)
        if [ "$status_code" = "200" ]; then
            test_pass
        else
            test_fail "Unexpected status code: $status_code"
        fi
    else
        test_fail "HTTP request endpoint failed"
    fi
}

# Test DNS endpoint
test_dns_lookup() {
    test_start "DNS lookup endpoint"
    
    payload='{
        "query": "google.com",
        "record_type": "A"
    }'
    
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/network/dns" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        answers=$(echo "$response" | jq -r '.data.answers' 2>/dev/null)
        if [ "$answers" != "null" ] && [ "$answers" != "[]" ]; then
            test_pass
        else
            test_fail "No DNS answers returned"
        fi
    else
        test_fail "DNS lookup endpoint failed"
    fi
}

# Test connectivity endpoint
test_connectivity() {
    test_start "Connectivity test endpoint"
    
    payload='{
        "target": "8.8.8.8",
        "test_type": "ping",
        "options": {
            "count": 1
        }
    }'
    
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/network/test/connectivity" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        stats=$(echo "$response" | jq -r '.data.statistics' 2>/dev/null)
        if [ "$stats" != "null" ]; then
            test_pass
        else
            test_fail "No statistics returned"
        fi
    else
        test_fail "Connectivity test endpoint failed"
    fi
}

# Test port scan endpoint
test_port_scan() {
    test_start "Port scan endpoint"
    
    payload='{
        "target": "localhost",
        "scan_type": "port",
        "ports": [80, 443, 22]
    }'
    
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/network/scan" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        results=$(echo "$response" | jq -r '.data.results' 2>/dev/null)
        if [ "$results" != "null" ] && [ "$results" != "[]" ]; then
            test_pass
        else
            test_fail "No scan results returned"
        fi
    else
        test_fail "Port scan endpoint failed"
    fi
}

# Test SSL validation endpoint
test_ssl_validation() {
    test_start "SSL validation endpoint"
    
    payload='{
        "url": "https://www.google.com",
        "options": {
            "check_expiry": true,
            "check_hostname": true
        }
    }'
    
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/network/ssl/validate" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        valid=$(echo "$response" | jq -r '.data.valid' 2>/dev/null)
        if [ "$valid" = "true" ] || [ "$valid" = "false" ]; then
            test_pass
        else
            test_fail "Invalid SSL validation response"
        fi
    else
        test_fail "SSL validation endpoint failed"
    fi
}

# Test API testing endpoint
test_api_test() {
    test_start "API testing endpoint"
    
    payload='{
        "base_url": "https://httpbin.org",
        "test_suite": [{
            "endpoint": "/get",
            "method": "GET",
            "test_cases": [{
                "name": "Basic GET test",
                "expected_status": 200
            }]
        }]
    }'
    
    response=$(curl -sf -X POST \
        -H "Content-Type: application/json" \
        -d "$payload" \
        "${API_BASE}/api/v1/network/api/test" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        success_rate=$(echo "$response" | jq -r '.data.overall_success_rate' 2>/dev/null)
        if [ "$success_rate" != "null" ]; then
            test_pass
        else
            test_fail "No success rate returned"
        fi
    else
        test_fail "API test endpoint failed"
    fi
}

# Test rate limiting
test_rate_limiting() {
    test_start "Rate limiting"
    
    # Make 101 requests quickly (limit is 100 per minute)
    local rate_limited=false
    
    for i in {1..101}; do
        response=$(curl -sf -w "%{http_code}" -o /dev/null \
            "${API_BASE}/api/v1/network/dns" \
            -X POST \
            -H "Content-Type: application/json" \
            -d '{"query":"test.com","record_type":"A"}' 2>/dev/null)
        
        if [ "$response" = "429" ]; then
            rate_limited=true
            break
        fi
    done
    
    if [ "$rate_limited" = true ]; then
        test_pass
    else
        test_fail "Rate limiting not enforced"
    fi
}

# Test CORS headers
test_cors() {
    test_start "CORS headers"
    
    response=$(curl -sf -I -X OPTIONS \
        -H "Origin: http://localhost:35000" \
        "${API_BASE}/api/v1/network/http" 2>/dev/null)
    
    if [ $? -eq 0 ]; then
        if echo "$response" | grep -q "Access-Control-Allow-Origin"; then
            test_pass
        else
            test_fail "CORS headers not present"
        fi
    else
        test_fail "CORS preflight request failed"
    fi
}

# Test CLI commands
test_cli() {
    test_start "CLI commands"
    
    # Test CLI health command
    if network-tools health 2>/dev/null | grep -q "healthy"; then
        test_pass
    else
        test_fail "CLI health command failed"
    fi
}

# Main test execution
main() {
    echo -e "${YELLOW}=== Network Tools Integration Tests ===${NC}"
    
    # Wait for API to be ready
    if ! wait_for_api; then
        echo -e "${RED}API is not available, skipping tests${NC}"
        exit 1
    fi
    
    # Run tests
    test_health
    test_http_request
    test_dns_lookup
    test_connectivity
    test_port_scan
    test_ssl_validation
    test_api_test
    test_rate_limiting
    test_cors
    test_cli
    
    # Print summary
    echo -e "\n${YELLOW}=== Test Summary ===${NC}"
    echo -e "Total tests: ${TESTS_TOTAL}"
    echo -e "${GREEN}Passed: ${TESTS_PASSED}${NC}"
    echo -e "${RED}Failed: ${TESTS_FAILED}${NC}"
    
    if [ $TESTS_FAILED -eq 0 ]; then
        echo -e "\n${GREEN}All tests passed!${NC}"
        exit 0
    else
        echo -e "\n${RED}Some tests failed${NC}"
        exit 1
    fi
}

# Run tests
main "$@"