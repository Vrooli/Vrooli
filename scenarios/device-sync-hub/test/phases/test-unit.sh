#!/usr/bin/env bash

# Unit Tests for Device Sync Hub
set -uo pipefail

# Test environment
export SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
export TEST_DIR="$SCENARIO_DIR/test"
export API_PORT="${API_PORT:-17564}"
export UI_PORT="${UI_PORT:-37181}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}=== Device Sync Hub Unit Tests ===${NC}"

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Test API binary exists
echo "Testing API binary..."
if [[ -f "$SCENARIO_DIR/api/device-sync-hub-api" ]]; then
    echo -e "${GREEN}✓${NC} API binary exists"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} API binary not found"
    ((TESTS_FAILED++))
fi

# Test CLI exists and is executable
echo "Testing CLI..."
if command -v device-sync-hub &>/dev/null; then
    echo -e "${GREEN}✓${NC} CLI is installed"
    ((TESTS_PASSED++))
else
    echo -e "${YELLOW}⚠${NC} CLI not installed globally (checking local)"
    if [[ -x "$SCENARIO_DIR/cli/device-sync-hub" ]]; then
        echo -e "${GREEN}✓${NC} CLI exists locally"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}✗${NC} CLI not found"
        ((TESTS_FAILED++))
    fi
fi

# Test UI files exist
echo "Testing UI files..."
if [[ -f "$SCENARIO_DIR/ui/index.html" ]] && [[ -f "$SCENARIO_DIR/ui/app.js" ]]; then
    echo -e "${GREEN}✓${NC} UI files present"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} UI files missing"
    ((TESTS_FAILED++))
fi

# Test database schema exists
echo "Testing database schema..."
if [[ -f "$SCENARIO_DIR/initialization/postgres/schema.sql" ]]; then
    echo -e "${GREEN}✓${NC} Database schema exists"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Database schema missing"
    ((TESTS_FAILED++))
fi

# Test configuration files
echo "Testing configuration..."
if [[ -f "$SCENARIO_DIR/.vrooli/service.json" ]]; then
    echo -e "${GREEN}✓${NC} Service configuration exists"
    ((TESTS_PASSED++))
else
    echo -e "${RED}✗${NC} Service configuration missing"
    ((TESTS_FAILED++))
fi

# Results
echo ""
echo -e "${GREEN}=== Unit Test Results ===${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -gt 0 ]]; then
    exit 1
fi