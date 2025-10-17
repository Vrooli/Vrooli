#!/bin/bash
# Don't use set -e as it can cause issues with curl commands
# set -e

echo "=== Integration Tests ==="

# Get the API and UI ports
API_PORT="${API_PORT:-19079}"
UI_PORT="${UI_PORT:-38809}"
BASE_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function to test API endpoint
test_api_endpoint() {
    local method="$1"
    local endpoint="$2"
    local expected_status="$3"
    local test_name="$4"
    
    echo -n "Testing: $test_name... "
    
    # Add timeout to prevent hanging
    response=$(timeout 5 curl -s -X "$method" -w "\n%{http_code}" "${BASE_URL}${endpoint}" 2>/dev/null | tail -1)
    
    if [ "$response" = "$expected_status" ]; then
        echo "✅ PASSED"
        ((TESTS_PASSED++))
    else
        echo "❌ FAILED (expected $expected_status, got $response)"
        ((TESTS_FAILED++))
    fi
}

# Test UI endpoint
test_ui_endpoint() {
    local endpoint="$1"
    local expected_status="$2"
    local test_name="$3"
    
    echo -n "Testing: $test_name... "
    
    # Add timeout to prevent hanging
    response=$(timeout 5 curl -s -w "%{http_code}" -o /dev/null "${UI_URL}${endpoint}" 2>/dev/null)
    
    if [ "$response" = "$expected_status" ]; then
        echo "✅ PASSED"
        ((TESTS_PASSED++))
    else
        echo "❌ FAILED (expected $expected_status, got $response)"
        ((TESTS_FAILED++))
    fi
}

# Wait for services to be ready
echo "Waiting for services to be ready..."
for i in {1..10}; do
    if curl -sf "${BASE_URL}/health" &>/dev/null && curl -sf "${UI_URL}/health" &>/dev/null; then
        echo "Services are ready!"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "⚠️ Services may not be fully ready, continuing anyway..."
    fi
    sleep 1
done

echo ""
echo "Running API integration tests..."
echo "================================"

# Test health endpoint
test_api_endpoint "GET" "/health" "200" "Health endpoint"

# Trigger a scan first to ensure agents are loaded
echo -n "Triggering initial agent scan... "
scan_response=$(curl -s -X POST "${BASE_URL}/api/v1/agents/scan" 2>/dev/null || echo '{"success":false}')
if echo "$scan_response" | grep -q '"success".*true'; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
    # Wait for scan to complete
    sleep 3
elif echo "$scan_response" | grep -q "already in progress"; then
    echo "⚠️ SKIPPED (scan already in progress)"
    # Wait for existing scan to complete
    sleep 3
else
    echo "❌ FAILED (response: $scan_response)"
    ((TESTS_FAILED++))
fi

# Test agent list endpoint
test_api_endpoint "GET" "/api/v1/agents" "200" "Get agents list"

# Test scan endpoint (may return 409 if scan is in progress)
echo -n "Testing: Trigger agent scan... "
response=$(timeout 5 curl -s -X POST -w "\n%{http_code}" "${BASE_URL}/api/v1/agents/scan" 2>/dev/null | tail -1)
if [ "$response" = "200" ] || [ "$response" = "409" ]; then
    echo "✅ PASSED (status: $response)"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED (expected 200 or 409, got $response)"
    ((TESTS_FAILED++))
fi

# Test capabilities endpoint
test_api_endpoint "GET" "/api/v1/capabilities" "200" "Get capabilities"

# Test search endpoint with parameter
echo -n "Testing: Search agents by capability... "
response=$(curl -s -w "%{http_code}" -o /dev/null "${BASE_URL}/api/v1/agents/search?capability=text-generation" 2>/dev/null)
if [ "$response" = "200" ]; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED (expected 200, got $response)"
    ((TESTS_FAILED++))
fi

# Test search endpoint without parameter (should fail)
echo -n "Testing: Search without capability (should fail)... "
response=$(curl -s -w "%{http_code}" -o /dev/null "${BASE_URL}/api/v1/agents/search" 2>/dev/null)
if [ "$response" = "400" ]; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED (expected 400, got $response)"
    ((TESTS_FAILED++))
fi

# Test CORS headers
echo -n "Testing: CORS headers on OPTIONS request... "
cors_header=$(curl -s -X OPTIONS -I "${BASE_URL}/api/v1/agents" 2>/dev/null | grep -i "access-control-allow-origin" | cut -d' ' -f2 | tr -d '\r')
if [ "$cors_header" = "*" ]; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED (expected *, got $cors_header)"
    ((TESTS_FAILED++))
fi

echo ""
echo "Running UI integration tests..."
echo "================================"

# Test UI endpoints
test_ui_endpoint "/" "200" "UI main page"
test_ui_endpoint "/health" "200" "UI health endpoint"
test_ui_endpoint "/index.html" "200" "UI index.html"
test_ui_endpoint "/script.js" "200" "UI JavaScript"
test_ui_endpoint "/styles.css" "200" "UI CSS"
test_ui_endpoint "/radar.js" "200" "UI radar component"
test_ui_endpoint "/logs.js" "200" "UI logs component"

echo ""
echo "Running CLI integration tests..."
echo "================================"

# Test CLI commands
echo -n "Testing: CLI help command... "
if agent-dashboard --help &>/dev/null; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED"
    ((TESTS_FAILED++))
fi

echo -n "Testing: CLI version command... "
if agent-dashboard --version &>/dev/null; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED"
    ((TESTS_FAILED++))
fi

echo -n "Testing: CLI list command... "
if agent-dashboard list &>/dev/null; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED"
    ((TESTS_FAILED++))
fi

echo -n "Testing: CLI health command... "
if agent-dashboard health &>/dev/null; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED"
    ((TESTS_FAILED++))
fi

echo -n "Testing: CLI scan command... "
if agent-dashboard scan &>/dev/null; then
    echo "✅ PASSED"
    ((TESTS_PASSED++))
else
    echo "❌ FAILED"
    ((TESTS_FAILED++))
fi

echo ""
echo "================================"
echo "Integration Test Results"
echo "================================"
echo "✅ Passed: $TESTS_PASSED"
echo "❌ Failed: $TESTS_FAILED"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo "🎉 All integration tests passed!"
    exit 0
else
    echo "⚠️ Some integration tests failed"
    exit 1
fi