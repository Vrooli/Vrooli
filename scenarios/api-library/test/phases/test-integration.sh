#!/bin/bash
# Integration tests for api-library scenario

set +e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test configuration
TIMEOUT=30
TEST_QUERY="payment processing"
API_PORT=$(vrooli scenario port api-library api 2>/dev/null || echo "15100")
UI_PORT=$(vrooli scenario port api-library ui 2>/dev/null || echo "35100")
API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

echo "🧪 Running API Library Integration Tests"
echo "API Port: ${API_PORT}"
echo "UI Port: ${UI_PORT}"
echo ""

# Counter for tests
PASSED=0
FAILED=0

# Helper function to run a test
run_test() {
    local test_name="$1"
    local test_command="$2"
    
    echo -n "Testing: $test_name... "
    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        ((PASSED++))
    else
        echo -e "${RED}✗${NC}"
        ((FAILED++))
    fi
}

# Test 1: API Health Check
run_test "API health endpoint" \
    "curl -sf ${API_URL}/health | jq -e '.status == \"healthy\"'"

# Test 2: Search API endpoint
run_test "Search API endpoint" \
    "curl -sf -X POST ${API_URL}/api/v1/search -H 'Content-Type: application/json' -d '{\"query\":\"${TEST_QUERY}\",\"limit\":5}' | jq -e '.results | length > 0'"

# Test 3: Semantic search capability
run_test "Semantic search returns relevant results" \
    "curl -sf -X POST ${API_URL}/api/v1/search -H 'Content-Type: application/json' -d '{\"query\":\"email\",\"limit\":3}' | jq -e '.results | length > 0'"

# Test 4: CLI search command
run_test "CLI search command" \
    "api-library search '${TEST_QUERY}' --json | jq -e '.results | length > 0'"

# Test 5: CLI show command (get first API ID from search)
API_ID=$(curl -sf -X POST "${API_URL}/api/v1/search" \
    -H "Content-Type: application/json" \
    -d "{\"query\":\"${TEST_QUERY}\",\"limit\":1}" | \
    jq -r '.results[0].id' 2>/dev/null || echo "")

if [ -n "$API_ID" ]; then
    run_test "CLI show command" \
        "api-library show '$API_ID' --json | jq -e '.api.id == \"$API_ID\"'"
else
    echo -e "Testing: CLI show command... ${YELLOW}SKIPPED (no API found)${NC}"
fi

# Test 6: Add note to API
if [ -n "$API_ID" ]; then
    run_test "Add note to API" \
        "curl -sf -X POST ${API_URL}/api/v1/apis/${API_ID}/notes -H 'Content-Type: application/json' -d '{\"content\":\"Test note from integration test\",\"type\":\"tip\"}' | jq -e '.id | length > 0'"
else
    echo -e "Testing: Add note to API... ${YELLOW}SKIPPED (no API found)${NC}"
fi

# Test 7: UI is accessible
run_test "UI is accessible" \
    "curl -sf ${UI_URL} | grep -q 'API Library'"

# Test 8: Search with filters
run_test "Search with configured filter" \
    "curl -sf -X POST ${API_URL}/api/v1/search -H 'Content-Type: application/json' -d '{\"query\":\"api\",\"limit\":10,\"filters\":{\"configured\":false}}' | jq -e '.results | all(.configured == false)'"

# Test 9: Request research endpoint
run_test "Request research endpoint" \
    "curl -sf -X POST ${API_URL}/api/v1/request-research -H 'Content-Type: application/json' -d '{\"capability\":\"video transcription\"}' | jq -e '.research_id | length > 0'"

# Test 10: Search response has correct structure
run_test "Search response has correct structure" \
    "curl -sf -X POST ${API_URL}/api/v1/search -H 'Content-Type: application/json' -d '{\"query\":\"test\",\"limit\":1}' | jq -e '.query and .method and .count >= 0 and .results'"

# Test 11: Relevance scoring works
run_test "Relevance scoring in search results" \
    "curl -sf -X POST ${API_URL}/api/v1/search -H 'Content-Type: application/json' -d '{\"query\":\"payment\",\"limit\":5}' | jq -e '.results | map(.relevance_score) | all(. > 0 and . <= 1)'"

# Test 12: CLI help command
run_test "CLI help command" \
    "api-library help | grep -q 'API Library CLI'"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Test Results:"
echo -e "  ${GREEN}Passed: ${PASSED}${NC}"
echo -e "  ${RED}Failed: ${FAILED}${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}✓ All integration tests passed!${NC}"
    exit 0
else
    echo -e "${RED}✗ Some tests failed${NC}"
    exit 1
fi