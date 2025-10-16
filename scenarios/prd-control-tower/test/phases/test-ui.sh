#!/usr/bin/env bash
# UI automation tests using browserless
# Tests the PRD Control Tower user interface

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🔍 Testing PRD Control Tower UI...${NC}"

# Get UI port from environment or use default
UI_PORT="${UI_PORT:-36300}"
UI_URL="http://localhost:${UI_PORT}"

# Check if browserless is available
if ! command -v resource-browserless &> /dev/null; then
    echo -e "${RED}✗ browserless CLI not found${NC}"
    exit 1
fi

# Check if browserless is running
if ! resource-browserless status | grep -q "running"; then
    echo -e "${RED}✗ browserless is not running${NC}"
    echo "  Start it with: vrooli resource browserless manage start"
    exit 1
fi

echo "  Testing UI accessibility..."

# Test 1: Homepage loads
echo -n "    Testing homepage..."
SCREENSHOT_PATH="/tmp/prd-control-tower-home.png"
if resource-browserless screenshot --url "$UI_URL" --output "$SCREENSHOT_PATH" --width 1920 --height 1080 &> /dev/null; then
    if [ -f "$SCREENSHOT_PATH" ] && [ -s "$SCREENSHOT_PATH" ]; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${RED}✗ Screenshot empty${NC}"
        exit 1
    fi
else
    echo -e " ${RED}✗ Failed to capture${NC}"
    exit 1
fi

# Test 2: Catalog page loads
echo -n "    Testing catalog page..."
CATALOG_SCREENSHOT="/tmp/prd-control-tower-catalog.png"
if resource-browserless screenshot --url "$UI_URL/catalog" --output "$CATALOG_SCREENSHOT" --width 1920 --height 1080 &> /dev/null; then
    if [ -f "$CATALOG_SCREENSHOT" ] && [ -s "$CATALOG_SCREENSHOT" ]; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${RED}✗ Screenshot empty${NC}"
        exit 1
    fi
else
    echo -e " ${RED}✗ Failed to capture${NC}"
    exit 1
fi

# Test 3: Drafts page loads
echo -n "    Testing drafts page..."
DRAFTS_SCREENSHOT="/tmp/prd-control-tower-drafts.png"
if resource-browserless screenshot --url "$UI_URL/drafts" --output "$DRAFTS_SCREENSHOT" --width 1920 --height 1080 &> /dev/null; then
    if [ -f "$DRAFTS_SCREENSHOT" ] && [ -s "$DRAFTS_SCREENSHOT" ]; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${RED}✗ Screenshot empty${NC}"
        exit 1
    fi
else
    echo -e " ${RED}✗ Failed to capture${NC}"
    exit 1
fi

# Test 4: Verify screenshots are not empty (basic smoke test)
echo -n "    Verifying UI rendered content..."
# Check that screenshots are larger than 10KB (indicates actual content, not blank page)
MIN_SIZE=10240  # 10KB
HOME_SIZE=$(stat -c%s "$SCREENSHOT_PATH" 2>/dev/null || echo 0)
CATALOG_SIZE=$(stat -c%s "$CATALOG_SCREENSHOT" 2>/dev/null || echo 0)
DRAFTS_SIZE=$(stat -c%s "$DRAFTS_SCREENSHOT" 2>/dev/null || echo 0)

if [ "$HOME_SIZE" -gt "$MIN_SIZE" ] && [ "$CATALOG_SIZE" -gt "$MIN_SIZE" ] && [ "$DRAFTS_SIZE" -gt "$MIN_SIZE" ]; then
    echo -e " ${GREEN}✓${NC}"
else
    echo -e " ${RED}✗ Screenshots too small (possible blank pages)${NC}"
    echo "    Home: ${HOME_SIZE} bytes, Catalog: ${CATALOG_SIZE} bytes, Drafts: ${DRAFTS_SIZE} bytes"
    exit 1
fi

# Test 5: Settings page loads
echo -n "    Testing settings page..."
SETTINGS_SCREENSHOT="/tmp/prd-control-tower-settings.png"
if resource-browserless screenshot --url "$UI_URL/settings" --output "$SETTINGS_SCREENSHOT" --width 1920 --height 1080 &> /dev/null; then
    if [ -f "$SETTINGS_SCREENSHOT" ] && [ -s "$SETTINGS_SCREENSHOT" ]; then
        echo -e " ${GREEN}✓${NC}"
    else
        echo -e " ${RED}✗ Screenshot empty${NC}"
        exit 1
    fi
else
    echo -e " ${RED}✗ Failed to capture${NC}"
    exit 1
fi

echo -e "${GREEN}✅ UI automation tests passed${NC}"
echo -e "   Screenshots saved to /tmp/prd-control-tower-*.png"

exit 0
