#!/usr/bin/env bash

# Integration Tests for Device Sync Hub
set -uo pipefail

# Test environment
export SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export API_URL="${API_URL:-http://localhost:17564}"
export UI_URL="${UI_URL:-http://localhost:37181}"
export AUTH_URL="${AUTH_URL:-http://localhost:15785}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== Device Sync Hub Integration Tests ===${NC}"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper function
test_endpoint() {
    local url="$1"
    local expected_status="$2"
    local description="$3"
    
    echo -n "Testing $description... "
    local status=$(curl -s -o /dev/null -w "%{http_code}" "$url" 2>/dev/null || echo "000")
    
    if [[ "$status" == "$expected_status" ]]; then
        echo -e "${GREEN}✓${NC} (HTTP $status)"
        ((TESTS_PASSED++))
        return 0
    else
        echo -e "${RED}✗${NC} (Expected $expected_status, got $status)"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Test API health
test_endpoint "$API_URL/health" "200" "API health endpoint"

# Test UI availability
test_endpoint "$UI_URL/" "200" "Web UI"

# Test API database connection
echo -n "Testing database connection... "
DB_STATUS=$(curl -s "$API_URL/health" 2>/dev/null | jq -r '.dependencies.database.connected' 2>/dev/null || echo "false")
if [[ "$DB_STATUS" == "true" ]]; then
    echo -e "${GREEN}✓${NC} Database connected"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Database not connected"
    ((TESTS_FAILED++))
fi

# Test API authentication service connection
echo -n "Testing auth service connection... "
AUTH_STATUS=$(curl -s "$API_URL/health" 2>/dev/null | jq -r '.dependencies.auth_service.connected' 2>/dev/null || echo "false")
if [[ "$AUTH_STATUS" == "true" ]]; then
    echo -e "${GREEN}✓${NC} Auth service connected"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Auth service not connected"
    ((TESTS_FAILED++))
fi

# Test WebSocket system
echo -n "Testing WebSocket system... "
WS_STATUS=$(curl -s "$API_URL/health" 2>/dev/null | jq -r '.dependencies.websocket_system.connected' 2>/dev/null || echo "false")
if [[ "$WS_STATUS" == "true" ]]; then
    echo -e "${GREEN}✓${NC} WebSocket system ready"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} WebSocket system not ready"
    ((TESTS_FAILED++))
fi

# Test storage system
echo -n "Testing storage system... "
STORAGE_STATUS=$(curl -s "$API_URL/health" 2>/dev/null | jq -r '.dependencies.storage_system.connected' 2>/dev/null || echo "false")
if [[ "$STORAGE_STATUS" == "true" ]]; then
    echo -e "${GREEN}✓${NC} Storage system ready"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Storage system not ready"
    ((TESTS_FAILED++))
fi

# Test CLI functionality
echo -n "Testing CLI status command... "
if API_URL="$API_URL" AUTH_URL="$AUTH_URL" device-sync-hub status --json 2>/dev/null | jq -e '.service' >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC} CLI status command works"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} CLI status command failed"
    ((TESTS_FAILED++))
fi

# Test protected endpoint (should return 401 without auth)
test_endpoint "$API_URL/api/v1/sync/items" "401" "Protected endpoint (no auth)"

# Results
echo ""
echo -e "${GREEN}=== Integration Test Results ===${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi