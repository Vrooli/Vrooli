#!/bin/bash
# UI automation test runner for visited-tracker
# Executes browser automation workflows using browser-automation-studio
set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo "🌐 Running UI automation tests for Visited Tracker..."

# Get test directory paths
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKFLOWS_DIR="$TEST_DIR/workflows"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$TEST_DIR/../../.." && pwd)}"

# Check for browser automation studio
BROWSER_AUTOMATION_CLI="$VROOLI_ROOT/scenarios/browser-automation-studio/cli/browser-automation-studio"

if [ ! -x "$BROWSER_AUTOMATION_CLI" ]; then
    echo -e "${YELLOW}⚠️  Browser automation studio not available at:${NC}"
    echo "   $BROWSER_AUTOMATION_CLI"
    echo ""
    echo -e "${BLUE}💡 To enable UI testing:${NC}"
    echo "   1. Set up browser-automation-studio scenario"
    echo "   2. Ensure CLI is installed and executable"
    echo "   3. Create UI workflows in $WORKFLOWS_DIR"
    echo ""
    echo -e "${YELLOW}ℹ️  Skipping UI automation tests${NC}"
    exit 0
fi

# Get UI port for visited-tracker
UI_PORT="${UI_PORT:-}"
if [ -z "$UI_PORT" ]; then
    if command -v vrooli >/dev/null 2>&1; then
        UI_PORT=$(vrooli scenario port visited-tracker UI_PORT 2>/dev/null || echo "3252")
    else
        UI_PORT="3252"
    fi
fi

UI_URL="http://localhost:${UI_PORT}"

echo "🎯 UI automation target: $UI_URL"

# Check if UI is running
if ! timeout 3 nc -z localhost "$UI_PORT" 2>/dev/null; then
    echo -e "${RED}❌ Visited Tracker UI is not running on port $UI_PORT${NC}"
    echo "   Start with: vrooli scenario start visited-tracker"
    exit 1
fi

# Ensure workflows directory exists
mkdir -p "$WORKFLOWS_DIR"

# Create basic workflows if they don't exist
if [ ! -f "$WORKFLOWS_DIR/smoke-test.json" ]; then
    echo "📝 Creating basic smoke test workflow..."
    
    cat > "$WORKFLOWS_DIR/smoke-test.json" << EOF
{
  "name": "Visited Tracker UI Smoke Test",
  "description": "Basic smoke test for visited-tracker UI functionality",
  "version": "1.0.0",
  "timeout": 30000,
  "steps": [
    {
      "action": "navigate",
      "url": "$UI_URL",
      "description": "Navigate to main page"
    },
    {
      "action": "wait",
      "selector": "title",
      "timeout": 5000,
      "description": "Wait for page to load"
    },
    {
      "action": "assert_title_contains",
      "text": "Visited Tracker",
      "description": "Verify page title contains 'Visited Tracker'"
    },
    {
      "action": "assert_element_exists",
      "selector": "body",
      "description": "Verify page body exists"
    },
    {
      "action": "navigate",
      "url": "$UI_URL/config",
      "description": "Navigate to config endpoint"
    },
    {
      "action": "assert_text_contains",
      "selector": "body",
      "text": "visited-tracker",
      "description": "Verify config contains service name"
    }
  ]
}
EOF
    echo -e "${GREEN}✅ Created smoke test workflow${NC}"
fi

# Create user journey workflow
if [ ! -f "$WORKFLOWS_DIR/user-journey.json" ]; then
    echo "📝 Creating user journey workflow..."
    
    cat > "$WORKFLOWS_DIR/user-journey.json" << EOF
{
  "name": "Visited Tracker User Journey",
  "description": "End-to-end user journey through visited-tracker UI",
  "version": "1.0.0",
  "timeout": 60000,
  "steps": [
    {
      "action": "navigate",
      "url": "$UI_URL",
      "description": "Start at main page"
    },
    {
      "action": "wait",
      "timeout": 3000,
      "description": "Wait for initial load"
    },
    {
      "action": "assert_title_contains",
      "text": "Visited Tracker",
      "description": "Verify we're on the right page"
    },
    {
      "action": "screenshot",
      "filename": "ui-main-page.png",
      "description": "Capture main page"
    },
    {
      "action": "navigate",
      "url": "$UI_URL/campaign.html",
      "description": "Navigate to campaign page (if exists)"
    },
    {
      "action": "screenshot", 
      "filename": "ui-campaign-page.png",
      "description": "Capture campaign page"
    }
  ]
}
EOF
    echo -e "${GREEN}✅ Created user journey workflow${NC}"
fi

# Find and execute all workflow files
workflow_files=()
if [ -d "$WORKFLOWS_DIR" ]; then
    while IFS= read -r -d '' file; do
        workflow_files+=("$file")
    done < <(find "$WORKFLOWS_DIR" -name "*.json" -type f -print0 2>/dev/null)
fi

if [ ${#workflow_files[@]} -eq 0 ]; then
    echo -e "${YELLOW}⚠️  No workflow files found in $WORKFLOWS_DIR${NC}"
    echo -e "${BLUE}💡 Create workflow JSON files in the workflows directory${NC}"
    exit 1
fi

echo ""
echo "🤖 Running ${#workflow_files[@]} UI automation workflows..."

# Execute workflows
workflow_count=0
failed_count=0

for workflow in "${workflow_files[@]}"; do
    workflow_name=$(basename "$workflow")
    echo ""
    echo -e "${CYAN}🚀 Executing workflow: $workflow_name${NC}"
    
    if "$BROWSER_AUTOMATION_CLI" execute "$workflow"; then
        echo -e "${GREEN}✅ Workflow passed: $workflow_name${NC}"
        ((workflow_count++))
    else
        echo -e "${RED}❌ Workflow failed: $workflow_name${NC}"
        ((failed_count++))
        ((workflow_count++))
    fi
done

# Summary
echo ""
echo "📊 UI Automation Test Summary:"
echo "   Workflows executed: $workflow_count"
echo "   Workflows passed: $((workflow_count - failed_count))"
echo "   Workflows failed: $failed_count"

# Check for screenshots and artifacts
if [ -d "screenshots" ]; then
    screenshot_count=$(find screenshots -name "*.png" -type f | wc -l)
    if [ "$screenshot_count" -gt 0 ]; then
        echo "   Screenshots captured: $screenshot_count (in screenshots/)"
    fi
fi

if [ $failed_count -eq 0 ]; then
    echo ""
    echo -e "${GREEN}✅ All $workflow_count UI automation workflows passed${NC}"
    echo -e "${GREEN}🎉 UI functionality validated successfully!${NC}"
    
    if [ $workflow_count -gt 0 ]; then
        echo ""
        echo -e "${BLUE}📸 UI Test Artifacts:${NC}"
        echo "   • Workflows: $WORKFLOWS_DIR"
        echo "   • Screenshots: screenshots/ (if generated)"
        echo "   • Browser logs: Available via browser-automation-studio"
    fi
    
    exit 0
else
    echo ""
    echo -e "${RED}❌ $failed_count of $workflow_count UI workflows failed${NC}"
    
    echo ""
    echo -e "${BLUE}💡 UI testing troubleshooting:${NC}"
    echo "   • Verify visited-tracker UI is fully loaded"
    echo "   • Check browser-automation-studio is working"
    echo "   • Review workflow definitions for accuracy"
    echo "   • Examine screenshots for visual verification"
    echo "   • Ensure UI elements have stable selectors"
    
    exit 1
fi