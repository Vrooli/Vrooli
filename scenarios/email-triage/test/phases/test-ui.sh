#!/bin/bash

# UI Test Phase
# Tests Email Triage UI functionality

set -euo pipefail

echo "🖥️  Testing Email Triage UI..."

# Configuration - Get UI port from environment, service.json, or use default
if [ -z "$UI_PORT" ]; then
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    SERVICE_JSON="$(dirname "$(dirname "$SCRIPT_DIR")")/.vrooli/service.json"
    if [ -f "$SERVICE_JSON" ] && command -v jq >/dev/null 2>&1; then
        UI_PORT=$(jq -r '.endpoints.ui // "http://localhost:3201"' "$SERVICE_JSON" | sed 's/.*://')
    else
        UI_PORT=3201
    fi
fi
UI_URL="http://localhost:${UI_PORT}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

FAILURES=0

# Test UI server is running
echo "Testing UI server:"
echo -n "  Checking UI availability... "
if response=$(curl -s -o /dev/null -w "%{http_code}" "$UI_URL" 2>/dev/null); then
    if [[ "$response" == "200" ]]; then
        echo -e "${GREEN}✓ UI is accessible${NC}"
    else
        echo -e "${RED}✗ UI returned status $response${NC}"
        ((FAILURES++))
    fi
else
    echo -e "${RED}✗ UI not accessible${NC}"
    ((FAILURES++))
fi

# Check main HTML file
echo -n "  Checking index.html... "
if curl -s "$UI_URL" | grep -q "Email Triage"; then
    echo -e "${GREEN}✓${NC}"
else
    echo -e "${RED}✗ Missing or invalid index.html${NC}"
    ((FAILURES++))
fi

# Check required UI files
echo -e "\nChecking UI assets:"
ui_files=(
    "/dashboard.js"
    "/styles.css"
)

for file in "${ui_files[@]}"; do
    echo -n "  Checking $file... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "${UI_URL}${file}" 2>/dev/null || echo "000")
    if [[ "$response" == "200" ]]; then
        echo -e "${GREEN}✓${NC}"
    else
        echo -e "${RED}✗ (status: $response)${NC}"
        ((FAILURES++))
    fi
done

# Take a screenshot if browserless is available
echo -e "\nTaking UI screenshot:"
if command -v vrooli >/dev/null 2>&1; then
    echo -n "  Capturing UI screenshot... "
    if vrooli resource browserless screenshot \
        --url "$UI_URL" \
        --output /tmp/email-triage-ui.png \
        >/dev/null 2>&1; then
        echo -e "${GREEN}✓ Screenshot saved to /tmp/email-triage-ui.png${NC}"
    else
        echo -e "${YELLOW}⚠ Screenshot failed (browserless may not be running)${NC}"
    fi
else
    echo -e "${YELLOW}⚠ Vrooli CLI not available for screenshot${NC}"
fi

# Check UI responsiveness
echo -e "\nChecking UI performance:"
echo -n "  Testing page load time... "
load_time=$(curl -o /dev/null -s -w '%{time_total}' "$UI_URL" 2>/dev/null || echo "999")
load_time_ms=$(echo "$load_time * 1000" | bc | cut -d. -f1)

if [[ $load_time_ms -lt 1500 ]]; then
    echo -e "${GREEN}✓ ${load_time_ms}ms (target: <1500ms)${NC}"
else
    echo -e "${YELLOW}⚠ ${load_time_ms}ms (target: <1500ms)${NC}"
fi

# Summary
if [[ $FAILURES -eq 0 ]]; then
    echo -e "\n${GREEN}✅ UI tests passed${NC}"
    exit 0
else
    echo -e "\n${RED}❌ $FAILURES UI test(s) failed${NC}"
    exit 1
fi