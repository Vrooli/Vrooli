#!/bin/bash
# API tests for funnel-builder
# Tests all API endpoints

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

test_count=0
pass_count=0
fail_count=0

# Discover API port from running scenario
# Try multiple methods to find the port
API_PORT=""

# Method 1: ss command (most reliable)
if [ -z "$API_PORT" ]; then
    API_PORT=$(ss -tlnp 2>/dev/null | grep "funnel-builder-" | grep -oP ':\K[0-9]+' | head -1)
fi

# Method 2: netstat fallback
if [ -z "$API_PORT" ]; then
    API_PORT=$(netstat -tlnp 2>/dev/null | grep "funnel-builder-" | awk '{print $4}' | cut -d':' -f2 | head -1)
fi

# Method 3: lsof fallback
if [ -z "$API_PORT" ]; then
    API_PORT=$(lsof -i -P -n 2>/dev/null | grep "funnel-builder-" | grep LISTEN | awk '{print $9}' | cut -d':' -f2 | head -1)
fi

if [ -z "$API_PORT" ]; then
    echo -e "${RED}Error: Could not discover API port. Is the scenario running?${NC}"
    echo "  Run: make run"
    exit 1
fi

API_BASE="http://localhost:${API_PORT}/api/v1"

run_test() {
    local test_name="$1"
    local test_command="$2"

    test_count=$((test_count + 1))
    echo -n "  Testing $test_name... "

    if eval "$test_command" &>/dev/null; then
        echo -e "${GREEN}✓${NC}"
        pass_count=$((pass_count + 1))
        return 0
    else
        echo -e "${RED}✗${NC}"
        fail_count=$((fail_count + 1))
        return 1
    fi
}

echo "Running API tests (API Port: $API_PORT)..."

# Health check
run_test "GET /health" "curl -sf 'http://localhost:${API_PORT}/health'"

# Funnel CRUD operations
run_test "GET /api/v1/funnels" "curl -sf '$API_BASE/funnels'"

# Get specific funnel (using demo funnel from seed data)
FUNNEL_ID=$(curl -sf "$API_BASE/funnels" | jq -r '.[0].id' 2>/dev/null)

if [ -n "$FUNNEL_ID" ] && [ "$FUNNEL_ID" != "null" ]; then
    run_test "GET /api/v1/funnels/:id" "curl -sf '$API_BASE/funnels/$FUNNEL_ID'"
fi

# Create new funnel
run_test "POST /api/v1/funnels" "curl -sf -X POST '$API_BASE/funnels' \
    -H 'Content-Type: application/json' \
    -d '{\"name\":\"Test Funnel\",\"description\":\"API test funnel\",\"steps\":[]}'"

# Summary
echo ""
echo "API Tests: $pass_count/$test_count passed"

if [ $fail_count -gt 0 ]; then
    exit 1
fi

exit 0
