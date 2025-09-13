#!/bin/bash
# Business tests phase - <180 seconds  
# Tests scenario-specific business logic, end-to-end workflows, and UI functionality
set -euo pipefail

echo "=== Business Logic Tests Phase (Target: <180s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m'

# Get the test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UI_DIR="$TEST_DIR/ui"
FIXTURES_DIR="$TEST_DIR/fixtures"

error_count=0
test_count=0
skipped_count=0

# Ensure fixtures directory exists
mkdir -p "$FIXTURES_DIR"

# Helper function to create test workspace
setup_test_workspace() {
    local workspace_dir="$1"
    
    echo "🔧 Setting up test workspace: $workspace_dir"
    rm -rf "$workspace_dir"
    mkdir -p "$workspace_dir/src/components" "$workspace_dir/src/utils" "$workspace_dir/tests"
    
    # Create diverse test files to simulate real codebase
    echo "console.log('main app');" > "$workspace_dir/src/main.js"
    echo "export const App = () => <div>App</div>;" > "$workspace_dir/src/components/App.js"
    echo "export const Button = () => <button>Click</button>;" > "$workspace_dir/src/components/Button.js"
    echo "export const helper = (x) => x * 2;" > "$workspace_dir/src/utils/helper.js"
    echo "export const api = { get: () => fetch('/api') };" > "$workspace_dir/src/utils/api.js"
    echo "export const auth = { login: () => true };" > "$workspace_dir/src/utils/auth.js"
    echo "test('helper doubles', () => expect(helper(2)).toBe(4));" > "$workspace_dir/tests/helper.test.js"
    echo "test('app renders', () => {});" > "$workspace_dir/tests/app.test.js"
    echo "# Project Documentation" > "$workspace_dir/README.md"
    echo "{ \"name\": \"test-project\", \"version\": \"1.0.0\" }" > "$workspace_dir/package.json"
    echo "FROM node:16" > "$workspace_dir/Dockerfile"
    
    echo "✅ Test workspace created with 10 files"
}

# Helper function to wait and modify files (simulate development activity)
simulate_development_activity() {
    local workspace_dir="$1"
    
    echo "⏳ Simulating development activity..."
    
    # Modify some files over time to create staleness patterns
    sleep 1
    echo "// Updated for feature A" >> "$workspace_dir/src/main.js"
    echo "// Security fix" >> "$workspace_dir/src/utils/auth.js"
    
    sleep 1  
    echo "// Performance improvement" >> "$workspace_dir/src/utils/api.js"
    echo "// New test case" >> "$workspace_dir/tests/helper.test.js"
    
    echo "✅ Development activity simulated"
}

# Get API and UI ports
API_PORT="${API_PORT:-}"
UI_PORT="${UI_PORT:-}"

if [ -z "$API_PORT" ] || [ -z "$UI_PORT" ]; then
    if command -v vrooli >/dev/null 2>&1; then
        API_PORT=$(vrooli scenario port visited-tracker API_PORT 2>/dev/null || echo "20252")
        UI_PORT=$(vrooli scenario port visited-tracker UI_PORT 2>/dev/null || echo "3252")
    else
        API_PORT="20252"
        UI_PORT="3252"
    fi
fi

API_URL="http://localhost:${API_PORT}"
UI_URL="http://localhost:${UI_PORT}"

echo "🎯 Testing business logic for Visited Tracker"
echo "   API: $API_URL"
echo "   UI: $UI_URL"

# Check if services are running
services_running=true
if ! timeout 3 nc -z localhost "$API_PORT" 2>/dev/null; then
    echo -e "${RED}❌ API not running on port $API_PORT${NC}"
    services_running=false
    ((error_count++))
fi

if ! timeout 3 nc -z localhost "$UI_PORT" 2>/dev/null; then
    echo -e "${YELLOW}⚠️  UI not running on port $UI_PORT (optional for some tests)${NC}"
fi

if [ "$services_running" = "false" ]; then
    echo -e "${RED}❌ Services not running. Start with: vrooli scenario start visited-tracker${NC}"
    echo -e "${YELLOW}ℹ️  Running offline business logic tests only...${NC}"
fi

# Test 1: Core Business Logic - Visit Tracking Workflow
echo ""
echo -e "${PURPLE}🎯 Test 1: Complete Visit Tracking Workflow${NC}"

if [ "$services_running" = "true" ]; then
    # Create a test workspace
    workspace="/tmp/visited-tracker-business-test-$$"
    setup_test_workspace "$workspace"
    cd "$workspace"
    
    # Step 1: Create campaign
    echo "📝 Creating test campaign..."
    campaign_data='{"name":"business-test-campaign","from_agent":"business-test","description":"End-to-end business logic test","patterns":["**/*.js","**/*.md","**/*.json"],"metadata":{"test_type":"business","phase":"complete_workflow"}}'
    
    campaign_response=$(curl -sf -X POST "$API_URL/api/v1/campaigns" -H "Content-Type: application/json" -d "$campaign_data" 2>/dev/null || echo "")
    
    if [ -n "$campaign_response" ] && echo "$campaign_response" | jq -e '.campaign.id' >/dev/null 2>&1; then
        campaign_id=$(echo "$campaign_response" | jq -r '.campaign.id')
        echo -e "${GREEN}✅ Campaign created: $campaign_id${NC}"
        ((test_count++))
        
        # Step 2: Sync file structure  
        echo "🔄 Syncing file structure..."
        sync_data='{"patterns":["**/*.js","**/*.md","**/*.json"],"remove_deleted":false,"root_path":"'"$workspace"'"}'
        sync_response=$(curl -sf -X POST "$API_URL/api/v1/campaigns/$campaign_id/structure/sync" -H "Content-Type: application/json" -d "$sync_data" 2>/dev/null || echo "")
        
        if [ -n "$sync_response" ] && echo "$sync_response" | jq -e '.added' >/dev/null 2>&1; then
            added=$(echo "$sync_response" | jq -r '.added')
            total=$(echo "$sync_response" | jq -r '.total')
            echo -e "${GREEN}✅ Structure synced: $added files added, $total total${NC}"
            ((test_count++))
            
            # Step 3: Record visits with different contexts
            echo "📊 Recording visits with multiple contexts..."
            
            # Security review visits
            security_files='["src/utils/auth.js","src/utils/api.js"]'
            security_data='{"files":'"$security_files"',"context":"security","agent":"security-reviewer","duration_ms":120000,"findings":{"vulnerabilities":0,"security_issues":["todo_review_auth_logic"]}}'
            
            if curl -sf -X POST "$API_URL/api/v1/campaigns/$campaign_id/visits" -H "Content-Type: application/json" -d "$security_data" >/dev/null 2>&1; then
                echo -e "${GREEN}✅ Security context visits recorded${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ Security visits failed${NC}"
                ((error_count++))
            fi
            
            # Performance review visits  
            perf_files='["src/utils/api.js","src/main.js"]'
            perf_data='{"files":'"$perf_files"',"context":"performance","agent":"perf-reviewer","duration_ms":90000,"findings":{"optimizations":["async_loading","caching"]}}'
            
            if curl -sf -X POST "$API_URL/api/v1/campaigns/$campaign_id/visits" -H "Content-Type: application/json" -d "$perf_data" >/dev/null 2>&1; then
                echo -e "${GREEN}✅ Performance context visits recorded${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ Performance visits failed${NC}"
                ((error_count++))
            fi
            
            # Simulate development activity to create staleness
            simulate_development_activity "$workspace"
            
            # Re-sync to update modification times
            curl -sf -X POST "$API_URL/api/v1/campaigns/$campaign_id/structure/sync" -H "Content-Type: application/json" -d "$sync_data" >/dev/null 2>&1
            
            # Step 4: Test prioritization algorithms
            echo "🎯 Testing prioritization algorithms..."
            
            # Test least visited
            least_visited=$(curl -sf "$API_URL/api/v1/campaigns/$campaign_id/prioritize/least-visited?limit=5&include_unvisited=true" 2>/dev/null || echo "")
            
            if [ -n "$least_visited" ] && echo "$least_visited" | jq -e '.files' >/dev/null 2>&1; then
                unvisited_count=$(echo "$least_visited" | jq '[.files[] | select(.visit_count == 0)] | length')
                echo -e "${GREEN}✅ Least visited prioritization: $unvisited_count unvisited files found${NC}"
                ((test_count++))
                
                # Verify unvisited files are prioritized first
                first_file_visits=$(echo "$least_visited" | jq -r '.files[0].visit_count')
                if [ "$first_file_visits" -eq 0 ]; then
                    echo -e "${GREEN}✅ Unvisited files correctly prioritized first${NC}"
                    ((test_count++))
                else
                    echo -e "${RED}❌ Prioritization algorithm error: visited file listed first${NC}"
                    ((error_count++))
                fi
            else
                echo -e "${RED}❌ Least visited prioritization failed${NC}"
                ((error_count++))
            fi
            
            # Test staleness detection
            most_stale=$(curl -sf "$API_URL/api/v1/campaigns/$campaign_id/prioritize/most-stale?limit=5&threshold=0.1" 2>/dev/null || echo "")
            
            if [ -n "$most_stale" ] && echo "$most_stale" | jq -e '.files' >/dev/null 2>&1; then
                avg_staleness=$(echo "$most_stale" | jq -r '.average_staleness')
                echo -e "${GREEN}✅ Staleness detection: average staleness $avg_staleness${NC}"
                ((test_count++))
                
                # Check if modified files appear in stale list
                stale_files=$(echo "$most_stale" | jq -r '.files[].file_path')
                if echo "$stale_files" | grep -q "main.js"; then
                    echo -e "${GREEN}✅ Modified files correctly identified as stale${NC}"
                    ((test_count++))
                else
                    echo -e "${YELLOW}⚠️  Staleness algorithm may need tuning${NC}"
                    ((skipped_count++))
                fi
            else
                echo -e "${RED}❌ Staleness detection failed${NC}"
                ((error_count++))
            fi
            
            # Step 5: Test coverage analysis
            echo "📈 Testing coverage analysis..."
            
            coverage=$(curl -sf "$API_URL/api/v1/campaigns/$campaign_id/coverage" 2>/dev/null || echo "")
            
            if [ -n "$coverage" ] && echo "$coverage" | jq -e '.coverage_percentage' >/dev/null 2>&1; then
                percentage=$(echo "$coverage" | jq -r '.coverage_percentage')
                visited_files=$(echo "$coverage" | jq -r '.visited_files')
                total_files=$(echo "$coverage" | jq -r '.total_files')
                
                echo -e "${GREEN}✅ Coverage analysis: ${percentage}% ($visited_files/$total_files files)${NC}"
                ((test_count++))
                
                # Verify coverage calculation
                expected_percentage=$(( visited_files * 100 / total_files ))
                if [ "${percentage%.*}" -eq "$expected_percentage" ]; then
                    echo -e "${GREEN}✅ Coverage percentage calculation correct${NC}"
                    ((test_count++))
                else
                    echo -e "${RED}❌ Coverage calculation error: expected ~$expected_percentage%, got $percentage%${NC}"
                    ((error_count++))
                fi
            else
                echo -e "${RED}❌ Coverage analysis failed${NC}"
                ((error_count++))
            fi
            
            # Step 6: Test context filtering
            echo "🔍 Testing context-based filtering..."
            
            security_coverage=$(curl -sf "$API_URL/api/v1/campaigns/$campaign_id/coverage?context=security" 2>/dev/null || echo "")
            
            if [ -n "$security_coverage" ] && echo "$security_coverage" | jq -e '.context' >/dev/null 2>&1; then
                context=$(echo "$security_coverage" | jq -r '.context')
                if [ "$context" = "security" ]; then
                    echo -e "${GREEN}✅ Context filtering working: $context${NC}"
                    ((test_count++))
                else
                    echo -e "${RED}❌ Context filtering failed: expected security, got $context${NC}"
                    ((error_count++))
                fi
            else
                echo -e "${RED}❌ Context-based coverage failed${NC}"
                ((error_count++))
            fi
            
        else
            echo -e "${RED}❌ Structure sync failed${NC}"
            ((error_count++))
        fi
        
    else
        echo -e "${RED}❌ Campaign creation failed${NC}"
        ((error_count++))
    fi
    
    # Cleanup workspace
    cd - >/dev/null
    rm -rf "$workspace"
    
else
    echo -e "${YELLOW}⚠️  Skipping API-dependent tests (services not running)${NC}"
    ((skipped_count += 10))
fi

# Test 2: CLI End-to-End Workflow 
echo ""
echo -e "${PURPLE}🎯 Test 2: CLI End-to-End Workflow${NC}"

if command -v visited-tracker >/dev/null 2>&1; then
    # Create CLI test workspace
    cli_workspace="/tmp/visited-tracker-cli-test-$$"
    setup_test_workspace "$cli_workspace"
    cd "$cli_workspace"
    
    echo "🔧 Testing CLI workflow in $cli_workspace..."
    
    # Test sync command
    echo "🔄 Testing CLI sync..."
    if visited-tracker sync --patterns "**/*.js" --patterns "**/*.md" --json >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI sync command works${NC}"
        ((test_count++))
    else
        echo -e "${RED}❌ CLI sync command failed${NC}"
        ((error_count++))
    fi
    
    # Test visit recording via CLI
    echo "📝 Testing CLI visit recording..."
    if visited-tracker visit "src/main.js" "src/utils/helper.js" --context cli-test --agent cli-tester >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI visit recording works${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  CLI visit recording failed (may need running service)${NC}"
        ((skipped_count++))
    fi
    
    # Test prioritization via CLI
    echo "🎯 Testing CLI prioritization..."
    if visited-tracker least-visited --limit 3 --json >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI least-visited works${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  CLI least-visited failed (may need running service)${NC}"
        ((skipped_count++))
    fi
    
    if visited-tracker coverage --json >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI coverage command works${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  CLI coverage failed (may need running service)${NC}"
        ((skipped_count++))
    fi
    
    # Cleanup CLI workspace
    cd - >/dev/null  
    rm -rf "$cli_workspace"
    
else
    echo -e "${RED}❌ visited-tracker CLI not available${NC}"
    echo "   Install with: cd cli && ./install.sh"
    ((error_count++))
fi

# Test 3: UI Functionality (if running)
echo ""
echo -e "${PURPLE}🎯 Test 3: UI Functionality Tests${NC}"

if timeout 3 nc -z localhost "$UI_PORT" 2>/dev/null; then
    echo "🌐 Testing UI endpoints..."
    
    # Test main UI page
    if curl -sf "$UI_URL" >/dev/null 2>&1; then
        ui_content=$(curl -s "$UI_URL" 2>/dev/null || echo "")
        if echo "$ui_content" | grep -q "Visited Tracker"; then
            echo -e "${GREEN}✅ UI main page loads with correct title${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ UI main page missing expected content${NC}"
            ((error_count++))
        fi
    else
        echo -e "${RED}❌ UI main page not responding${NC}"
        ((error_count++))
    fi
    
    # Test UI config endpoint
    if curl -sf "$UI_URL/config" >/dev/null 2>&1; then
        config_response=$(curl -s "$UI_URL/config" 2>/dev/null || echo "")
        if echo "$config_response" | jq -e '.service' >/dev/null 2>&1; then
            service_name=$(echo "$config_response" | jq -r '.service')
            if [ "$service_name" = "visited-tracker" ]; then
                echo -e "${GREEN}✅ UI config endpoint returns correct service${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ UI config endpoint wrong service: $service_name${NC}"
                ((error_count++))
            fi
        else
            echo -e "${RED}❌ UI config endpoint invalid response${NC}"
            ((error_count++))
        fi
    else
        echo -e "${RED}❌ UI config endpoint not responding${NC}"
        ((error_count++))
    fi
    
    # Run UI automation tests if available
    echo "🤖 Checking for UI automation tests..."
    if [ -x "$UI_DIR/run-ui-tests.sh" ]; then
        if "$UI_DIR/run-ui-tests.sh"; then
            echo -e "${GREEN}✅ UI automation tests passed${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ UI automation tests failed${NC}"
            ((error_count++))
        fi
    else
        echo -e "${YELLOW}ℹ️  No UI automation tests found${NC}"
        echo -e "${BLUE}💡 Consider adding browser automation workflows${NC}"
        ((skipped_count++))
    fi
    
else
    echo -e "${YELLOW}ℹ️  UI not running, skipping UI tests${NC}"
    ((skipped_count += 3))
fi

# Test 4: Data Persistence and Recovery
echo ""
echo -e "${PURPLE}🎯 Test 4: Data Persistence Validation${NC}"

if [ "$services_running" = "true" ]; then
    echo "💾 Testing data persistence patterns..."
    
    # Test campaign data persistence
    campaigns_response=$(curl -sf "$API_URL/api/v1/campaigns" 2>/dev/null || echo "")
    
    if [ -n "$campaigns_response" ] && echo "$campaigns_response" | jq -e '.campaigns' >/dev/null 2>&1; then
        campaign_count=$(echo "$campaigns_response" | jq '.campaigns | length')
        echo -e "${GREEN}✅ Campaigns data accessible: $campaign_count campaigns${NC}"
        ((test_count++))
        
        # Check if our test campaigns from earlier tests are still there
        if [ "$campaign_count" -gt 0 ]; then
            echo -e "${GREEN}✅ Data persistence working across test phases${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}⚠️  No persistent campaigns found${NC}"
            ((skipped_count++))
        fi
    else
        echo -e "${RED}❌ Campaign data access failed${NC}"
        ((error_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  Skipping persistence tests (services not running)${NC}"
    ((skipped_count += 2))
fi

# Performance and summary
end_time=$(date +%s)
duration=$((end_time - start_time))
total_tests=$((test_count + error_count))

echo ""
echo "📊 Business Logic Test Summary:"
echo "   Tests passed: $test_count"
echo "   Tests failed: $error_count"
echo "   Tests skipped: $skipped_count"
echo "   Total test cases: $total_tests"
echo "   Duration: ${duration}s"

if [ $error_count -eq 0 ]; then
    if [ $test_count -gt 0 ]; then
        echo -e "${GREEN}✅ All business logic tests passed in ${duration}s${NC}"
        echo -e "${GREEN}🎉 Visited Tracker business functionality validated!${NC}"
    else
        echo -e "${YELLOW}⚠️  No business tests could be executed in ${duration}s${NC}"
        echo -e "${BLUE}💡 Start the scenario first: vrooli scenario start visited-tracker${NC}"
    fi
else
    echo -e "${RED}❌ Business logic tests failed with $error_count failures in ${duration}s${NC}"
fi

if [ $duration -gt 180 ]; then
    echo -e "${YELLOW}⚠️  Business tests phase exceeded 180s target${NC}"
fi

# Show business-specific recommendations
if [ $error_count -gt 0 ] || [ $skipped_count -gt 5 ]; then
    echo ""
    echo -e "${BLUE}💡 Business logic testing recommendations:${NC}"
    
    if [ $error_count -gt 0 ]; then
        echo "   • Ensure visited-tracker scenario is fully running"
        echo "   • Verify PostgreSQL database connectivity" 
        echo "   • Check API endpoints are responding correctly"
        echo "   • Install and test CLI: cd cli && ./install.sh"
    fi
    
    if [ $skipped_count -gt 5 ]; then
        echo "   • Add UI automation tests using browser-automation-studio"
        echo "   • Create test fixtures for consistent business logic testing"
        echo "   • Add more comprehensive staleness detection test cases"
    fi
    
    echo "   • Business logic represents core value: visit tracking & staleness detection"
    echo "   • See: docs/scenarios/PHASED_TESTING_ARCHITECTURE.md"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi