#!/usr/bin/env bash
# Smoke tests for Device Sync Hub
# Fast sanity checks to verify basic functionality

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration from environment
API_PORT="${API_PORT:-17808}"
UI_PORT="${UI_PORT:-37197}"
API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

echo -e "${BLUE}🔥 Running Device Sync Hub Smoke Tests${NC}"
echo ""

TESTS_PASSED=0
TESTS_FAILED=0

test_api_health() {
    echo -n "Testing API health endpoint... "
    if curl -sf "${API_URL}/health" | jq -e '.readiness == true' >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_ui_accessible() {
    echo -n "Testing UI accessibility... "
    if curl -sf "${UI_URL}/" >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_auth_integration() {
    echo -n "Testing auth service integration... "
    local health
    health=$(curl -sf "${API_URL}/health" | jq -r '.dependencies.auth_service.connected')
    if [[ "$health" == "true" ]]; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_database_connection() {
    echo -n "Testing database connection... "
    local health
    health=$(curl -sf "${API_URL}/health" | jq -r '.dependencies.database.connected')
    if [[ "$health" == "true" ]]; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

test_cli_available() {
    echo -n "Testing CLI availability... "
    if command -v device-sync-hub >/dev/null 2>&1; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        ((TESTS_FAILED++))
        return 1
    fi
}

# Run tests
test_api_health || true
test_ui_accessible || true
test_auth_integration || true
test_database_connection || true
test_cli_available || true

# Results
echo ""
echo -e "${BLUE}Smoke Test Results:${NC}"
echo -e "  ${GREEN}Passed: $TESTS_PASSED${NC}"
if [[ $TESTS_FAILED -gt 0 ]]; then
    echo -e "  ${RED}Failed: $TESTS_FAILED${NC}"
    exit 1
else
    echo -e "  ${GREEN}All smoke tests passed!${NC}"
    exit 0
fi
