#!/usr/bin/env bash

# Smoke Tests for Device Sync Hub
# Quick tests to verify basic functionality
set -uo pipefail

# Test environment
export API_URL="${API_URL:-http://localhost:17564}"
export UI_URL="${UI_URL:-http://localhost:37181}"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}=== Device Sync Hub Smoke Tests ===${NC}"

# Quick health check
echo -n "API Health: "
if curl -sf "$API_URL/health" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    exit 1
fi

echo -n "UI Available: "
if curl -sf "$UI_URL/" >/dev/null 2>&1; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    exit 1
fi

echo -n "Database: "
DB_STATUS=$(curl -sf "$API_URL/health" 2>/dev/null | jq -r '.dependencies.database.connected' 2>/dev/null || echo "false")
if [[ "$DB_STATUS" == "true" ]]; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗${NC}"
    exit 1
fi

echo -e "${GREEN}All smoke tests passed!${NC}"