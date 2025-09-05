#!/bin/bash

# Basic functionality test for Visited Tracker
# Tests campaign creation, file tracking, and status updates

set -e

API_PORT="${API_PORT:-20251}"
API_URL="http://localhost:${API_PORT}"
TEST_DIR="/tmp/visited-tracker-test"
CLI_PATH="./cli/visited-tracker"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🧪 Testing Visited Tracker Basic Functionality${NC}"

# Setup test environment
echo "Setting up test environment..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/src" "$TEST_DIR/tests"

# Create test files
echo "console.log('test');" > "$TEST_DIR/src/main.js"
echo "console.log('helper');" > "$TEST_DIR/src/helper.js"
echo "test('example', () => {});" > "$TEST_DIR/tests/main.test.js"
echo "# Documentation" > "$TEST_DIR/README.md"

echo "✓ Test files created in $TEST_DIR"

# Test 1: Check API health
echo
echo "Test 1: API Health Check"
if curl -sf "$API_URL/health" > /dev/null; then
    echo -e "${GREEN}✓ API is healthy${NC}"
else
    echo -e "${RED}✗ API health check failed${NC}"
    exit 1
fi

# Test 2: Create campaign via CLI
echo
echo "Test 2: Create Campaign via CLI"
cd "$TEST_DIR"

CAMPAIGN_OUTPUT=$("$CLI_PATH" create "test-campaign" "src/**/*.js" "*.md" \
    --description "Test campaign for validation" \
    --scenario "visited-tracker-test" \
    --working-dir "$TEST_DIR" 2>&1)

if echo "$CAMPAIGN_OUTPUT" | grep -q "Campaign created successfully"; then
    echo -e "${GREEN}✓ Campaign created successfully${NC}"
    CAMPAIGN_ID=$(echo "$CAMPAIGN_OUTPUT" | grep "ID:" | cut -d' ' -f2)
    echo "Campaign ID: $CAMPAIGN_ID"
else
    echo -e "${RED}✗ Campaign creation failed${NC}"
    echo "$CAMPAIGN_OUTPUT"
    exit 1
fi

# Test 3: List campaigns
echo
echo "Test 3: List Campaigns"
if "$CLI_PATH" list | grep -q "test-campaign"; then
    echo -e "${GREEN}✓ Campaign appears in list${NC}"
else
    echo -e "${RED}✗ Campaign not found in list${NC}"
    exit 1
fi

# Test 4: List files in campaign
echo
echo "Test 4: List Files in Campaign"
FILES_OUTPUT=$("$CLI_PATH" files "$CAMPAIGN_ID")
if echo "$FILES_OUTPUT" | grep -q "src/main.js"; then
    echo -e "${GREEN}✓ Files detected correctly${NC}"
else
    echo -e "${RED}✗ Files not detected properly${NC}"
    echo "$FILES_OUTPUT"
    exit 1
fi

# Test 5: Update file status
echo
echo "Test 5: Update File Status"
if "$CLI_PATH" update-file "$CAMPAIGN_ID" "src/main.js" "in_progress"; then
    echo -e "${GREEN}✓ File status updated to in_progress${NC}"
else
    echo -e "${RED}✗ File status update failed${NC}"
    exit 1
fi

# Complete the file
if "$CLI_PATH" update-file "$CAMPAIGN_ID" "src/main.js" "completed"; then
    echo -e "${GREEN}✓ File status updated to completed${NC}"
else
    echo -e "${RED}✗ File status completion failed${NC}"
    exit 1
fi

# Test 6: Check campaign progress
echo
echo "Test 6: Check Campaign Progress"
PROGRESS_OUTPUT=$("$CLI_PATH" status --campaign-id "$CAMPAIGN_ID")
if echo "$PROGRESS_OUTPUT" | grep -q "1/"; then
    echo -e "${GREEN}✓ Progress tracking working${NC}"
else
    echo -e "${RED}✗ Progress tracking failed${NC}"
    echo "$PROGRESS_OUTPUT"
    exit 1
fi

# Test 7: API direct access
echo
echo "Test 7: Direct API Access"
API_RESPONSE=$(curl -s "$API_URL/api/v1/campaigns")
if echo "$API_RESPONSE" | grep -q "test-campaign"; then
    echo -e "${GREEN}✓ API direct access working${NC}"
else
    echo -e "${RED}✗ API direct access failed${NC}"
    echo "$API_RESPONSE"
    exit 1
fi

# Cleanup
echo
echo "Cleaning up test environment..."
rm -rf "$TEST_DIR"

echo -e "${GREEN}🎉 All tests passed! Visited Tracker is working correctly${NC}"
echo
echo "Next steps:"
echo "  1. Run 'vrooli scenario run visited-tracker' to start the full scenario"
echo "  2. Visit the dashboard at http://localhost:3251"
echo "  3. Use 'visited-tracker --help' for CLI usage"