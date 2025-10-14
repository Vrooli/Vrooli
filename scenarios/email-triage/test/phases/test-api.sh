#!/bin/bash

# API Test Phase
# Tests Email Triage API endpoints

set -euo pipefail

echo "🔌 Testing Email Triage API endpoints..."

# Configuration - Get API port from environment or use vrooli status
if [ -z "$API_PORT" ]; then
    # Try to get running port from vrooli status
    if command -v vrooli >/dev/null 2>&1; then
        SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
        SCENARIO_NAME=$(basename "$(dirname "$(dirname "$SCRIPT_DIR")")")
        API_PORT=$(vrooli scenario status "$SCENARIO_NAME" --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // empty' 2>/dev/null || echo "")
    fi

    # Fallback to default if still not found
    if [ -z "$API_PORT" ]; then
        API_PORT=19528
    fi
fi
API_URL="http://localhost:${API_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILURES=0

# Test helper function
test_api() {
    local method="$1"
    local endpoint="$2"
    local expected_status="$3"
    local description="$4"
    local data="${5:-}"
    local headers="${6:-}"
    
    echo -n "  $description... "
    
    local curl_cmd="curl -s -X $method -o /dev/null -w '%{http_code}' '${API_URL}${endpoint}'"
    
    if [[ -n "$headers" ]]; then
        curl_cmd="$curl_cmd -H '$headers'"
    fi
    
    if [[ -n "$data" ]]; then
        curl_cmd="$curl_cmd -d '$data'"
    fi
    
    response=$(eval "$curl_cmd" 2>/dev/null || echo "000")
    
    if [[ "$response" == "$expected_status" ]]; then
        echo -e "${GREEN}✓${NC}"
        return 0
    else
        echo -e "${RED}✗ (got $response, expected $expected_status)${NC}"
        return 1
    fi
}

# Test unauthenticated endpoints
echo "Testing public endpoints:"
test_api "GET" "/health" 200 "Health check" || ((FAILURES++))
test_api "GET" "/health/database" 200 "Database health" || ((FAILURES++))
test_api "GET" "/health/qdrant" 200 "Qdrant health" || ((FAILURES++))

# Test authenticated endpoints (should return 401 without auth, unless DEV_MODE is true)
echo -e "\nTesting authentication requirement:"
if [[ "${DEV_MODE:-}" == "true" ]]; then
    echo "  Note: DEV_MODE is enabled, auth checks bypassed"
    test_api "GET" "/api/v1/accounts" 200 "Accounts list (dev mode)" || ((FAILURES++))
    test_api "GET" "/api/v1/rules" 200 "Rules list (dev mode)" || ((FAILURES++))
    test_api "GET" "/api/v1/emails/search?q=test" 200 "Email search (dev mode)" || ((FAILURES++))
else
    test_api "GET" "/api/v1/accounts" 401 "Accounts list (no auth)" || ((FAILURES++))
    test_api "GET" "/api/v1/rules" 401 "Rules list (no auth)" || ((FAILURES++))
    test_api "GET" "/api/v1/emails/search?q=test" 401 "Email search (no auth)" || ((FAILURES++))
fi

# Test with mock JWT token (should fail validation unless in DEV_MODE)
if [[ "${DEV_MODE:-}" != "true" ]]; then
    echo -e "\nTesting with invalid token:"
    MOCK_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
    test_api "GET" "/api/v1/accounts" 401 "Accounts with invalid token" "" "Authorization: $MOCK_TOKEN" || ((FAILURES++))
fi

# Test API endpoints exist (even if auth fails)
echo -e "\nVerifying API endpoints exist:"
endpoints=(
    "/api/v1/accounts"
    "/api/v1/rules"
    "/api/v1/emails/search"
    "/api/v1/analytics/dashboard"
)

for endpoint in "${endpoints[@]}"; do
    response=$(curl -s -o /dev/null -w "%{http_code}" "${API_URL}${endpoint}" 2>/dev/null || echo "000")
    # 200=OK, 400=Bad Request (endpoint exists, missing params), 401=Unauthorized (endpoint exists, needs auth)
    if [[ "$response" == "200" ]] || [[ "$response" == "400" ]] || [[ "$response" == "401" ]]; then
        echo -e "  ${GREEN}✓${NC} $endpoint exists"
    else
        echo -e "  ${RED}✗${NC} $endpoint not found (status: $response)"
        ((FAILURES++))
    fi
done

# Test CORS headers
echo -e "\nTesting CORS configuration:"
response_headers=$(curl -s -I "${API_URL}/health" 2>/dev/null)
if echo "$response_headers" | grep -q "Access-Control-Allow-Origin"; then
    echo -e "  ${GREEN}✓${NC} CORS headers present"
else
    echo -e "  ${YELLOW}⚠${NC} CORS headers not found (may be OK for internal API)"
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo -e "\n${GREEN}✅ API tests passed${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILURES API test(s) failed${NC}"
    exit 1
fi