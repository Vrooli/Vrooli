#!/bin/bash
# Smoke tests for api-library scenario

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "🧪 Running API Library Smoke Tests"

# Get ports
API_PORT=$(vrooli scenario port api-library api 2>/dev/null || echo "18904")
UI_PORT=$(vrooli scenario port api-library ui 2>/dev/null || echo "35100")

echo "API Port: ${API_PORT}"
echo "UI Port: ${UI_PORT}"

# Test 1: API is responding
echo -n "API Health Check... "
if curl -sf "http://localhost:${API_PORT}/health" &>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗ API not responding${NC}"
    exit 1
fi

# Test 2: Search endpoint works
echo -n "Search Endpoint... "
if curl -sf "http://localhost:${API_PORT}/api/v1/search?query=test" | jq -e '.results' &>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗ Search endpoint not working${NC}"
    exit 1
fi

# Test 3: CLI is installed
echo -n "CLI Installation... "
if command -v api-library &>/dev/null; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${YELLOW}⚠ CLI not in PATH${NC}"
fi

echo -e "\n${GREEN}✓ Smoke tests passed!${NC}"