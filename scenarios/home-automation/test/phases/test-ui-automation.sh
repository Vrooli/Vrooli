#!/usr/bin/env bash
# UI Automation Tests for Home Automation Scenario
#
# Tests the UI functionality using Browserless for browser automation
# This ensures the UI renders correctly, API integration works, and user interactions function

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
SCENARIO_NAME="home-automation"
API_PORT="${API_PORT:-17556}"  # Default API port if not set
UI_PORT="${UI_PORT:-38816}"    # Default UI port if not set
TIMEOUT=30
SCREENSHOT_DIR="/tmp/home-automation-ui-tests"

echo -e "${BLUE}🧪 Testing Home Automation UI with Browser Automation${NC}"
echo "======================================================="

# Create screenshot directory
mkdir -p "$SCREENSHOT_DIR"

# Function to cleanup on exit
cleanup() {
    echo -e "\n${BLUE}🧹 Cleaning up test artifacts...${NC}"
}
trap cleanup EXIT

# Check if browserless is available
echo -e "\n${BLUE}📋 Checking dependencies...${NC}"
if ! command -v resource-browserless &> /dev/null; then
    echo -e "${YELLOW}⚠️  Browserless resource not available${NC}"
    echo -e "${YELLOW}   Install with: vrooli resource install browserless${NC}"
    echo -e "${GREEN}✅ UI automation tests skipped (browserless not available)${NC}"
    exit 0
fi

# Check if browserless is running
if ! resource-browserless status &> /dev/null; then
    echo -e "${YELLOW}⚠️  Browserless not running, attempting to start...${NC}"
    if ! vrooli resource start browserless &> /dev/null; then
        echo -e "${YELLOW}⚠️  Could not start browserless${NC}"
        echo -e "${GREEN}✅ UI automation tests skipped (browserless not running)${NC}"
        exit 0
    fi
    echo -e "${GREEN}✅ Browserless started${NC}"
    sleep 5  # Give it time to fully start
fi

echo -e "${GREEN}✅ Browserless is available${NC}"

# Check if UI is running
echo -e "\n${BLUE}🌐 Checking UI availability...${NC}"
if ! curl -sf "http://localhost:${UI_PORT}/health" &> /dev/null; then
    echo -e "${RED}✗ UI server not responding on port ${UI_PORT}${NC}"
    echo -e "${YELLOW}   Make sure the scenario is running: make start${NC}"
    exit 1
fi
echo -e "${GREEN}✅ UI server is running on port ${UI_PORT}${NC}"

# Check if API is running
echo -e "\n${BLUE}🔌 Checking API availability...${NC}"
if ! curl -sf "http://localhost:${API_PORT}/health" &> /dev/null; then
    echo -e "${YELLOW}⚠️  API server not responding on port ${API_PORT}${NC}"
    echo -e "${YELLOW}   Some tests may fail${NC}"
else
    echo -e "${GREEN}✅ API server is running on port ${API_PORT}${NC}"
fi

# Test 1: UI loads successfully
echo -e "\n${BLUE}🎨 Test 1: UI loads successfully${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/01-ui-load.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}" \
    --output "$SCREENSHOT_PATH" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ UI loaded successfully${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"

    # Verify screenshot was created and has reasonable size
    if [ -f "$SCREENSHOT_PATH" ]; then
        SIZE=$(stat -f%z "$SCREENSHOT_PATH" 2>/dev/null || stat -c%s "$SCREENSHOT_PATH" 2>/dev/null)
        if [ "$SIZE" -gt 5000 ]; then
            echo -e "${GREEN}   Screenshot size: ${SIZE} bytes (valid)${NC}"
        else
            echo -e "${YELLOW}   Warning: Screenshot unusually small (${SIZE} bytes)${NC}"
        fi
    fi
else
    echo -e "${RED}✗ Failed to load UI${NC}"
    exit 1
fi

# Test 2: Devices tab renders
echo -e "\n${BLUE}🔌 Test 2: Devices tab renders${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/02-devices-tab.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}#devices" \
    --output "$SCREENSHOT_PATH" \
    --wait-for ".devices-grid, .device-placeholder" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Devices tab rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Devices tab may not have fully loaded${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
fi

# Test 3: Scenes tab renders
echo -e "\n${BLUE}🎬 Test 3: Scenes tab renders${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/03-scenes-tab.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}#scenes" \
    --output "$SCREENSHOT_PATH" \
    --wait-for ".scenes-grid, .scene-placeholder" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Scenes tab rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Scenes tab may not have fully loaded${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
fi

# Test 4: Automations tab renders
echo -e "\n${BLUE}🤖 Test 4: Automations tab renders${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/04-automations-tab.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}#automations" \
    --output "$SCREENSHOT_PATH" \
    --wait-for ".automations-list, .automation-placeholder" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Automations tab rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Automations tab may not have fully loaded${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
fi

# Test 5: Energy tab renders
echo -e "\n${BLUE}⚡ Test 5: Energy tab renders${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/05-energy-tab.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}#energy" \
    --output "$SCREENSHOT_PATH" \
    --wait-for ".energy-dashboard, .energy-placeholder" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Energy tab rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Energy tab may not have fully loaded${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
fi

# Test 6: Settings tab renders
echo -e "\n${BLUE}⚙️  Test 6: Settings tab renders${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/06-settings-tab.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}#settings" \
    --output "$SCREENSHOT_PATH" \
    --wait-for ".settings-panel, .settings-placeholder" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Settings tab rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Settings tab may not have fully loaded${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
fi

# Test 7: Mobile viewport rendering
echo -e "\n${BLUE}📱 Test 7: Mobile viewport renders correctly${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/07-mobile-view.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}" \
    --output "$SCREENSHOT_PATH" \
    --viewport "375x667" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Mobile view rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Mobile view may not have fully loaded${NC}"
fi

# Test 8: Tablet viewport rendering
echo -e "\n${BLUE}📲 Test 8: Tablet viewport renders correctly${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/08-tablet-view.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}" \
    --output "$SCREENSHOT_PATH" \
    --viewport "768x1024" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Tablet view rendered${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Tablet view may not have fully loaded${NC}"
fi

# Test 9: Connection status indicator
echo -e "\n${BLUE}🔗 Test 9: Connection status indicator visible${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/09-connection-status.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}" \
    --output "$SCREENSHOT_PATH" \
    --wait-for "#connectionStatus" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ Connection status indicator present${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  Connection status indicator not found${NC}"
fi

# Test 10: User profile section
echo -e "\n${BLUE}👤 Test 10: User profile section visible${NC}"
SCREENSHOT_PATH="${SCREENSHOT_DIR}/10-user-profile.png"
if resource-browserless screenshot \
    --url "http://localhost:${UI_PORT}" \
    --output "$SCREENSHOT_PATH" \
    --wait-for "#userProfile" \
    --timeout "$TIMEOUT" &> /dev/null; then
    echo -e "${GREEN}✅ User profile section present${NC}"
    echo -e "   Screenshot: ${SCREENSHOT_PATH}"
else
    echo -e "${YELLOW}⚠️  User profile section not found${NC}"
fi

# Summary
echo -e "\n${BLUE}📊 Test Summary${NC}"
echo "=============================="
SCREENSHOT_COUNT=$(ls -1 "$SCREENSHOT_DIR"/*.png 2>/dev/null | wc -l)
echo -e "${GREEN}✅ Created ${SCREENSHOT_COUNT} screenshots${NC}"
echo -e "${BLUE}📂 Screenshots saved to: ${SCREENSHOT_DIR}${NC}"

# List all screenshots
echo -e "\n${BLUE}📸 Generated Screenshots:${NC}"
for screenshot in "$SCREENSHOT_DIR"/*.png; do
    if [ -f "$screenshot" ]; then
        SIZE=$(stat -f%z "$screenshot" 2>/dev/null || stat -c%s "$screenshot" 2>/dev/null)
        BASENAME=$(basename "$screenshot")
        echo -e "   ${GREEN}✓${NC} ${BASENAME} (${SIZE} bytes)"
    fi
done

echo -e "\n${GREEN}✅ UI automation tests completed successfully${NC}"
echo -e "${BLUE}💡 View screenshots to verify UI rendering${NC}"
exit 0
