#!/bin/bash

# Token Economy Integration Tests

set -e

echo "Token Economy Integration Tests"
echo "================================"

# Configuration
API_URL="${TOKEN_ECONOMY_API_URL:-http://localhost:11080}"
CLI_PATH="../cli/token-economy"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_api() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"
    local expected_status="$5"
    
    echo -n "Testing: $name... "
    
    if [ "$method" = "GET" ]; then
        response=$(curl -s -w "\n%{http_code}" -X GET "${API_URL}${endpoint}")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "${API_URL}${endpoint}" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi
    
    status=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)
    
    if [ "$status" = "$expected_status" ]; then
        echo "✓ PASSED"
        ((TESTS_PASSED++))
    else
        echo "✗ FAILED (expected $expected_status, got $status)"
        echo "  Response: $body"
        ((TESTS_FAILED++))
    fi
}

test_cli() {
    local name="$1"
    local command="$2"
    local expected_exit="$3"
    
    echo -n "Testing CLI: $name... "
    
    if $CLI_PATH $command > /dev/null 2>&1; then
        exit_code=0
    else
        exit_code=$?
    fi
    
    if [ "$exit_code" = "$expected_exit" ]; then
        echo "✓ PASSED"
        ((TESTS_PASSED++))
    else
        echo "✗ FAILED (expected exit $expected_exit, got $exit_code)"
        ((TESTS_FAILED++))
    fi
}

# Start tests
echo ""
echo "1. API Health Checks"
echo "--------------------"
test_api "Health endpoint" "GET" "/health" "" "200"
test_api "API v1 health" "GET" "/api/v1/health" "" "200"

echo ""
echo "2. Token Operations"
echo "-------------------"
test_api "Create token" "POST" "/api/v1/tokens/create" \
    '{"symbol":"TEST","name":"Test Token","type":"fungible"}' "201"
test_api "List tokens" "GET" "/api/v1/tokens" "" "200"

echo ""
echo "3. Wallet Operations"
echo "--------------------"
test_api "Create user wallet" "POST" "/api/v1/wallets/create" \
    '{"type":"user","user_id":"test-user-1"}' "201"
test_api "Create scenario wallet" "POST" "/api/v1/wallets/create" \
    '{"type":"scenario","scenario_name":"test-scenario"}' "201"

echo ""
echo "4. CLI Operations"
echo "-----------------"
test_cli "Help command" "help" "0"
test_cli "Version command" "version" "0"
test_cli "Status command" "status" "0"

echo ""
echo "5. Transaction Flow"
echo "-------------------"
# Note: These would need actual wallet IDs from previous tests
echo "Skipping transaction tests (require dynamic wallet IDs)"

echo ""
echo "================================"
echo "Test Results"
echo "================================"
echo "Passed: $TESTS_PASSED"
echo "Failed: $TESTS_FAILED"
echo ""

if [ "$TESTS_FAILED" -eq 0 ]; then
    echo "✓ All tests passed!"
    exit 0
else
    echo "✗ Some tests failed"
    exit 1
fi