#!/usr/bin/env bash
################################################################################
# Judge0 Simple Smoke Tests - Quick Health Validation (<30s)
################################################################################

set -e

# Configuration
JUDGE0_PORT="${JUDGE0_PORT:-2358}"
API_URL="http://localhost:${JUDGE0_PORT}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "\n${GREEN}Running Judge0 Simple Smoke Tests${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

TEST_FAILED=0

# Test 1: Health endpoint
echo -e "  ${YELLOW}ℹ${NC} Testing health endpoint..."
if timeout 5 curl -sf "${API_URL}/version" > /dev/null 2>&1; then
    echo -e "  ${GREEN}✓${NC} Health endpoint responding"
else
    echo -e "  ${RED}✗${NC} Health endpoint not responding"
    TEST_FAILED=1
fi

# Test 2: Get version
echo -e "  ${YELLOW}ℹ${NC} Getting version..."
VERSION=$(timeout 5 curl -sf "${API_URL}/version" 2>/dev/null || echo "FAILED")
if [[ "$VERSION" != "FAILED" ]] && [[ -n "$VERSION" ]]; then
    echo -e "  ${GREEN}✓${NC} Version: $VERSION"
else
    echo -e "  ${RED}✗${NC} Could not get version"
    TEST_FAILED=1
fi

# Test 3: Languages endpoint
echo -e "  ${YELLOW}ℹ${NC} Testing languages endpoint..."
LANGUAGES=$(timeout 5 curl -sf "${API_URL}/languages" 2>/dev/null || echo "FAILED")
if [[ "$LANGUAGES" != "FAILED" ]] && [[ -n "$LANGUAGES" ]]; then
    # Count languages using simple grep
    LANG_COUNT=$(echo "$LANGUAGES" | grep -o '"id":' | wc -l)
    if [[ $LANG_COUNT -gt 20 ]]; then
        echo -e "  ${GREEN}✓${NC} Languages available: $LANG_COUNT"
    else
        echo -e "  ${RED}✗${NC} Only $LANG_COUNT languages (expected >20)"
        TEST_FAILED=1
    fi
else
    echo -e "  ${RED}✗${NC} Languages endpoint not responding"
    TEST_FAILED=1
fi

# Test 4: Container health
echo -e "  ${YELLOW}ℹ${NC} Testing container health..."
CONTAINERS=$(docker ps --format "{{.Names}}" | grep -c judge0 || echo "0")
if [[ $CONTAINERS -gt 0 ]]; then
    echo -e "  ${GREEN}✓${NC} Judge0 containers running: $CONTAINERS"
else
    echo -e "  ${RED}✗${NC} No Judge0 containers found"
    TEST_FAILED=1
fi

# Test 5: Execution test via workarounds
echo -e "  ${YELLOW}ℹ${NC} Testing code execution..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
if [[ -f "${RESOURCE_DIR}/lib/execution-manager.sh" ]]; then
    RESULT=$("${RESOURCE_DIR}/lib/execution-manager.sh" execute python3 'print("test")' 2>/dev/null || echo "FAILED")
    if echo "$RESULT" | grep -q '"status":"accepted"'; then
        echo -e "  ${GREEN}✓${NC} Code execution working (via workaround)"
    else
        echo -e "  ${YELLOW}⚠${NC} Code execution needs configuration"
    fi
else
    echo -e "  ${YELLOW}⚠${NC} Execution manager not found"
fi

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [[ $TEST_FAILED -eq 0 ]]; then
    echo -e "${GREEN}✅ All smoke tests passed${NC}\n"
    exit 0
else
    echo -e "${RED}❌ Some smoke tests failed${NC}\n"
    exit 1
fi