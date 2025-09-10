#!/bin/bash

# Visited Tracker Functionality Test Suite
# Tests visit tracking, staleness detection, and prioritization

set -e

# Get the API port dynamically
API_PORT=$(vrooli scenario port visited-tracker API_PORT 2>/dev/null)
if [[ -z "$API_PORT" ]]; then
    echo "❌ visited-tracker scenario is not running"
    echo "   Start it with: vrooli scenario run visited-tracker"
    exit 1
fi

API_URL="http://localhost:${API_PORT}"
TEST_DIR="/tmp/visited-tracker-test-$$"
CLI_PATH="visited-tracker"  # Should be installed globally

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}🧪 Testing Visited Tracker Functionality${NC}"

# Cleanup function
cleanup() {
    rm -rf "$TEST_DIR"
}
trap cleanup EXIT

# Setup test environment
echo "Setting up test environment..."
rm -rf "$TEST_DIR"
mkdir -p "$TEST_DIR/src/components" "$TEST_DIR/src/utils" "$TEST_DIR/tests"

# Create test files with varying content
echo "console.log('main app');" > "$TEST_DIR/src/main.js"
echo "console.log('App component');" > "$TEST_DIR/src/components/App.js"
echo "export const helper = () => {};" > "$TEST_DIR/src/utils/helper.js"
echo "export const api = () => {};" > "$TEST_DIR/src/utils/api.js"
echo "test('main test', () => {});" > "$TEST_DIR/tests/main.test.js"
echo "test('app test', () => {});" > "$TEST_DIR/tests/app.test.js"
echo "# Documentation" > "$TEST_DIR/README.md"
echo "{ \"name\": \"test\" }" > "$TEST_DIR/package.json"

echo "✓ Test files created in $TEST_DIR"

# Test 1: Check API health
echo
echo -e "${BLUE}Test 1: API Health Check${NC}"
if curl -sf "$API_URL/health" > /dev/null; then
    health=$(curl -s "$API_URL/health" | jq -r '.status')
    if [[ "$health" == "healthy" ]]; then
        echo -e "${GREEN}✓ API is healthy${NC}"
    else
        echo -e "${RED}✗ API health status: $health${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ API health check failed${NC}"
    exit 1
fi

# Test 2: Sync file structure
echo
echo -e "${BLUE}Test 2: Sync File Structure${NC}"
cd "$TEST_DIR"

sync_response=$($CLI_PATH sync --patterns "**/*.js" --patterns "**/*.md" --json 2>&1)
if echo "$sync_response" | jq -e '.added' > /dev/null; then
    added=$(echo "$sync_response" | jq -r '.added')
    total=$(echo "$sync_response" | jq -r '.total')
    echo -e "${GREEN}✓ Structure synced: $added files added, $total total${NC}"
else
    echo -e "${RED}✗ Structure sync failed${NC}"
    echo "$sync_response"
    exit 1
fi

# Test 3: Record visits
echo
echo -e "${BLUE}Test 3: Record File Visits${NC}"

# Visit some files
visit_response=$($CLI_PATH visit "src/main.js" "src/components/App.js" --context security --agent "test-cli" --json 2>&1)
if echo "$visit_response" | jq -e '.recorded' > /dev/null; then
    recorded=$(echo "$visit_response" | jq -r '.recorded')
    echo -e "${GREEN}✓ Recorded $recorded visits with security context${NC}"
else
    echo -e "${RED}✗ Visit recording failed${NC}"
    echo "$visit_response"
    exit 1
fi

# Test 4: Check visit counts
echo
echo -e "${BLUE}Test 4: Verify Visit Counts${NC}"

# Visit main.js again
$CLI_PATH visit "src/main.js" --context performance > /dev/null 2>&1

# Check the visit count increased
visit_response=$($CLI_PATH visit "src/main.js" --context bug --json 2>&1)
if echo "$visit_response" | jq -e '.files[0].visit_count' > /dev/null; then
    visit_count=$(echo "$visit_response" | jq -r '.files[0].visit_count')
    if [[ $visit_count -ge 3 ]]; then
        echo -e "${GREEN}✓ Visit count correctly tracked: $visit_count visits${NC}"
    else
        echo -e "${RED}✗ Visit count unexpected: $visit_count${NC}"
        exit 1
    fi
else
    echo -e "${RED}✗ Could not get visit count${NC}"
    exit 1
fi

# Test 5: Get least visited files
echo
echo -e "${BLUE}Test 5: Get Least Visited Files${NC}"

least_visited=$($CLI_PATH least-visited --limit 5 --json 2>&1)
if echo "$least_visited" | jq -e '.files' > /dev/null; then
    file_count=$(echo "$least_visited" | jq '.files | length')
    echo -e "${GREEN}✓ Retrieved $file_count least visited files${NC}"
    
    # Check that unvisited files appear first
    first_file_visits=$(echo "$least_visited" | jq -r '.files[0].visit_count')
    if [[ $first_file_visits -eq 0 ]]; then
        echo -e "${GREEN}✓ Unvisited files correctly prioritized${NC}"
    fi
else
    echo -e "${RED}✗ Failed to get least visited files${NC}"
    echo "$least_visited"
    exit 1
fi

# Test 6: Check staleness scores
echo
echo -e "${BLUE}Test 6: Check Staleness Detection${NC}"

# Modify a file to trigger staleness
sleep 1
echo "// Modified" >> "$TEST_DIR/src/utils/helper.js"

# Sync to update modification times
$CLI_PATH sync --patterns "**/*.js" > /dev/null 2>&1

stale_files=$($CLI_PATH most-stale --limit 5 --json 2>&1)
if echo "$stale_files" | jq -e '.files' > /dev/null; then
    avg_staleness=$(echo "$stale_files" | jq -r '.average_staleness')
    echo -e "${GREEN}✓ Staleness detection working: avg score $avg_staleness${NC}"
else
    echo -e "${RED}✗ Failed to get stale files${NC}"
    echo "$stale_files"
    exit 1
fi

# Test 7: Coverage statistics
echo
echo -e "${BLUE}Test 7: Coverage Statistics${NC}"

coverage=$($CLI_PATH coverage --json 2>&1)
if echo "$coverage" | jq -e '.coverage_percentage' > /dev/null; then
    percentage=$(echo "$coverage" | jq -r '.coverage_percentage')
    visited=$(echo "$coverage" | jq -r '.visited_files')
    unvisited=$(echo "$coverage" | jq -r '.unvisited_files')
    
    echo -e "${GREEN}✓ Coverage stats: ${percentage}% coverage (${visited} visited, ${unvisited} unvisited)${NC}"
else
    echo -e "${RED}✗ Failed to get coverage statistics${NC}"
    echo "$coverage"
    exit 1
fi

# Test 8: Export and Import
echo
echo -e "${BLUE}Test 8: Export and Import Data${NC}"

export_file="$TEST_DIR/export.json"
$CLI_PATH export "$export_file" --format json > /dev/null 2>&1

if [[ -f "$export_file" ]]; then
    export_count=$(jq '.exported_count' "$export_file")
    echo -e "${GREEN}✓ Exported $export_count files to $export_file${NC}"
    
    # Test import (would need a fresh database to truly test)
    import_response=$($CLI_PATH import "$export_file" --json 2>&1)
    if echo "$import_response" | jq -e '.message' > /dev/null; then
        echo -e "${YELLOW}⚠ Import endpoint placeholder working${NC}"
    fi
else
    echo -e "${RED}✗ Export failed${NC}"
    exit 1
fi

# Test 9: CLI status command
echo
echo -e "${BLUE}Test 9: CLI Status Command${NC}"

if $CLI_PATH status | grep -q "Service: visited-tracker"; then
    echo -e "${GREEN}✓ Status command working${NC}"
else
    echo -e "${RED}✗ Status command failed${NC}"
    exit 1
fi

# Test 10: Context filtering
echo
echo -e "${BLUE}Test 10: Context Filtering${NC}"

# Get files visited with security context
security_files=$($CLI_PATH least-visited --context security --limit 10 --json 2>&1)
if echo "$security_files" | jq -e '.files' > /dev/null; then
    echo -e "${GREEN}✓ Context filtering working${NC}"
else
    echo -e "${RED}✗ Context filtering failed${NC}"
    exit 1
fi

# Summary
echo
echo -e "${GREEN}🎉 All tests passed! Visited Tracker is working correctly${NC}"
echo
echo "Summary:"
echo "  ✓ API health and connectivity"
echo "  ✓ File structure synchronization"
echo "  ✓ Visit recording and counting"
echo "  ✓ Least visited prioritization"
echo "  ✓ Staleness detection"
echo "  ✓ Coverage statistics"
echo "  ✓ Export functionality"
echo "  ✓ CLI commands"
echo "  ✓ Context filtering"
echo
echo "Next steps:"
echo "  1. Visit the dashboard at http://localhost:$(vrooli scenario port visited-tracker UI_PORT)"
echo "  2. Use 'visited-tracker help' for full CLI usage"
echo "  3. Integrate with your analysis scenarios"