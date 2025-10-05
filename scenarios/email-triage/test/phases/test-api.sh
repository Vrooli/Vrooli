#!/bin/bash

# API Test Phase
# Tests Email Triage API endpoints

set -euo pipefail

echo "🔌 Testing Email Triage API endpoints..."

# Configuration - Get API port from environment, service.json, or use default
if [ -z "$API_PORT" ]; then
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    SERVICE_JSON="$(dirname "$(dirname "$SCRIPT_DIR")")/.vrooli/service.json"
    if [ -f "$SERVICE_JSON" ] && command -v jq >/dev/null 2>&1; then
        API_PORT=$(jq -r '.endpoints.api // "http://localhost:19528"' "$SERVICE_JSON" | sed 's/.*://')
    else
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

# Test authenticated endpoints (should return 401 without auth)
echo -e "\nTesting authentication requirement:"
test_api "GET" "/api/v1/accounts" 401 "Accounts list (no auth)" || ((FAILURES++))
test_api "GET" "/api/v1/rules" 401 "Rules list (no auth)" || ((FAILURES++))
test_api "GET" "/api/v1/emails/search?q=test" 401 "Email search (no auth)" || ((FAILURES++))

# Test with mock JWT token (should fail validation)
echo -e "\nTesting with invalid token:"
MOCK_TOKEN="Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
test_api "GET" "/api/v1/accounts" 401 "Accounts with invalid token" "" "Authorization: $MOCK_TOKEN" || ((FAILURES++))

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
    if [[ "$response" == "401" ]] || [[ "$response" == "200" ]]; then
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