#!/bin/bash
# CLI tests for funnel-builder
# Tests all CLI commands

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

# Discover API port and export it for CLI
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

export API_PORT

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

echo "Running CLI tests (API Port: $API_PORT)..."

# Check CLI is installed
run_test "CLI is installed" "command -v funnel-builder"

# Basic commands
run_test "funnel-builder help" "funnel-builder help"
run_test "funnel-builder version" "funnel-builder version"
run_test "funnel-builder status --json" "funnel-builder status --json"

# List funnels
run_test "funnel-builder list" "funnel-builder list"
run_test "funnel-builder list --json" "funnel-builder list --json"

# Get specific funnel
FUNNEL_ID=$(funnel-builder list --json 2>/dev/null | jq -r '.[0].id' 2>/dev/null)

if [ -n "$FUNNEL_ID" ] && [ "$FUNNEL_ID" != "null" ]; then
    run_test "funnel-builder get <id>" "funnel-builder get '$FUNNEL_ID'"
    run_test "funnel-builder get <id> --json" "funnel-builder get '$FUNNEL_ID' --json"
fi

# Summary
echo ""
echo "CLI Tests: $pass_count/$test_count passed"

if [ $fail_count -gt 0 ]; then
    exit 1
fi

exit 0
