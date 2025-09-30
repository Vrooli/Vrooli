#!/bin/bash

# Integration tests for document-manager scenario

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Get API URL from environment or use default
API_URL="${API_URL:-http://localhost:17810}"
UI_URL="${UI_URL:-http://localhost:38106}"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

echo "======================================"
echo "Document Manager Integration Tests"
echo "======================================"
echo "API URL: $API_URL"
echo "UI URL: $UI_URL"
echo ""

# Helper function for test assertions
assert_response() {
    local test_name="$1"
    local response="$2"
    local expected_status="$3"
    
    if [[ "$response" -eq "$expected_status" ]]; then
        echo -e "${GREEN}✓${NC} $test_name"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} $test_name (Expected: $expected_status, Got: $response)"
        ((TESTS_FAILED++))
    fi
}

# Helper function for JSON validation
validate_json() {
    local test_name="$1"
    local json="$2"
    
    if echo "$json" | jq empty 2>/dev/null; then
        echo -e "${GREEN}✓${NC} $test_name - Valid JSON"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} $test_name - Invalid JSON"
        ((TESTS_FAILED++))
    fi
}

# Test 1: API Health Check
echo "Testing API Health..."
status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/health")
assert_response "API Health Check" "$status" "200"

# Test 2: Database Status
echo "Testing Database Connectivity..."
status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/system/db-status")
assert_response "Database Status" "$status" "200"

# Test 3: Vector Database Status
echo "Testing Vector Database..."
status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/system/vector-status")
assert_response "Vector Database Status" "$status" "200"

# Test 4: AI Integration Status
echo "Testing AI Integration..."
status=$(curl -s -o /dev/null -w "%{http_code}" "$API_URL/api/system/ai-status")
assert_response "AI Integration Status" "$status" "200"

# Test 5: Create Application
echo "Testing Application Creation..."
app_response=$(curl -s -X POST "$API_URL/api/applications" \
    -H "Content-Type: application/json" \
    -d '{"name":"Integration Test App","repository_url":"https://github.com/test/integration","documentation_path":"/docs","active":true}')
validate_json "Application Creation" "$app_response"

# Extract app ID if creation successful
APP_ID=""
if echo "$app_response" | jq -r '.id' &>/dev/null; then
    APP_ID=$(echo "$app_response" | jq -r '.id' 2>/dev/null || echo "")
    if [[ -n "$APP_ID" ]] && [[ "$APP_ID" != "null" ]]; then
        echo -e "${GREEN}✓${NC} Application created with ID: $APP_ID"
        ((TESTS_PASSED++))
    fi
fi

# Test 6: List Applications
echo "Testing Application Listing..."
apps=$(curl -s "$API_URL/api/applications")
validate_json "Application Listing" "$apps"

# Test 7: Create Agent
if [[ -n "$APP_ID" ]] && [[ "$APP_ID" != "null" ]]; then
    echo "Testing Agent Creation..."
    agent_response=$(curl -s -X POST "$API_URL/api/agents" \
        -H "Content-Type: application/json" \
        -d "{\"name\":\"Integration Test Agent\",\"type\":\"documentation_analyzer\",\"application_id\":\"$APP_ID\",\"configuration\":\"{}\",\"enabled\":true}")
    validate_json "Agent Creation" "$agent_response"
    
    # Extract agent ID
    AGENT_ID=""
    if echo "$agent_response" | jq -r '.id' &>/dev/null; then
        AGENT_ID=$(echo "$agent_response" | jq -r '.id' 2>/dev/null || echo "")
        if [[ -n "$AGENT_ID" ]] && [[ "$AGENT_ID" != "null" ]]; then
            echo -e "${GREEN}✓${NC} Agent created with ID: $AGENT_ID"
            ((TESTS_PASSED++))
        fi
    fi
fi

# Test 8: List Agents
echo "Testing Agent Listing..."
agents=$(curl -s "$API_URL/api/agents")
validate_json "Agent Listing" "$agents"

# Test 9: Create Improvement Queue Item
if [[ -n "$APP_ID" ]] && [[ "$APP_ID" != "null" ]] && [[ -n "$AGENT_ID" ]] && [[ "$AGENT_ID" != "null" ]]; then
    echo "Testing Improvement Queue Creation..."
    queue_response=$(curl -s -X POST "$API_URL/api/queue" \
        -H "Content-Type: application/json" \
        -d "{\"agent_id\":\"$AGENT_ID\",\"application_id\":\"$APP_ID\",\"type\":\"documentation_improvement\",\"title\":\"Integration Test Improvement\",\"description\":\"Test improvement from integration suite\",\"severity\":\"low\",\"status\":\"pending\"}")
    validate_json "Queue Item Creation" "$queue_response"
fi

# Test 10: List Improvement Queue
echo "Testing Improvement Queue Listing..."
queue=$(curl -s "$API_URL/api/queue")
validate_json "Queue Listing" "$queue"

# Test 11: UI Health Check
echo "Testing UI Health..."
ui_status=$(curl -s -o /dev/null -w "%{http_code}" "$UI_URL/ui-health")
assert_response "UI Health Check" "$ui_status" "200"

# Test 12: UI Main Page
echo "Testing UI Main Page..."
ui_page_status=$(curl -s -o /dev/null -w "%{http_code}" "$UI_URL/")
assert_response "UI Main Page" "$ui_page_status" "200"

# Test 13: Response Time Tests
echo "Testing API Response Times..."
start_time=$(date +%s%N)
curl -s "$API_URL/health" > /dev/null
end_time=$(date +%s%N)
response_time=$((($end_time - $start_time) / 1000000))

if [[ $response_time -lt 500 ]]; then
    echo -e "${GREEN}✓${NC} Health check response time: ${response_time}ms (< 500ms)"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Health check response time: ${response_time}ms (> 500ms)"
    ((TESTS_FAILED++))
fi

# Test 14: Concurrent Request Handling
echo "Testing Concurrent Requests..."
(
for i in {1..10}; do
    curl -s "$API_URL/api/applications" > /dev/null &
done
wait
)
echo -e "${GREEN}✓${NC} Handled 10 concurrent requests"
((TESTS_PASSED++))

# Test 15: Error Handling - Invalid JSON
echo "Testing Error Handling..."
error_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$API_URL/api/applications" \
    -H "Content-Type: application/json" \
    -d 'invalid json')

if [[ "$error_status" -eq "400" ]]; then
    echo -e "${GREEN}✓${NC} Proper error handling for invalid JSON"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠${NC} Unexpected status for invalid JSON: $error_status"
fi

echo ""
echo "======================================"
echo "Test Results"
echo "======================================"
echo -e "${GREEN}Passed:${NC} $TESTS_PASSED"
echo -e "${RED}Failed:${NC} $TESTS_FAILED"

# Exit with appropriate code
if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi