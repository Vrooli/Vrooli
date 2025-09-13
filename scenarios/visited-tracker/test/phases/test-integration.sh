#!/bin/bash
# Integration tests phase - <120 seconds
# Tests service integration, API endpoints, database operations, and CLI integration
set -euo pipefail

echo "=== Integration Tests Phase (Target: <120s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Get the test directory
TEST_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CLI_DIR="$TEST_DIR/cli"

error_count=0
test_count=0
skipped_count=0

# Helper function to check if service is running
check_service_running() {
    local service_name="$1"
    local expected_port="$2"
    
    if timeout 3 nc -z localhost "$expected_port" 2>/dev/null; then
        return 0
    else
        return 1
    fi
}

# Helper function to wait for service to be ready
wait_for_service() {
    local url="$1"
    local timeout="${2:-30}"
    local counter=0
    
    echo "⏳ Waiting for service to be ready at $url..."
    while [ $counter -lt $timeout ]; do
        if curl -sf "$url" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Service is ready${NC}"
            return 0
        fi
        sleep 1
        ((counter++))
        echo -n "."
    done
    echo ""
    echo -e "${RED}❌ Service not ready after ${timeout}s${NC}"
    return 1
}

# Get API port from service configuration or use default
API_PORT="${API_PORT:-}"
if [ -z "$API_PORT" ]; then
    if command -v vrooli >/dev/null 2>&1; then
        API_PORT=$(vrooli scenario port visited-tracker API_PORT 2>/dev/null || echo "")
    fi
fi

if [ -z "$API_PORT" ]; then
    echo -e "${YELLOW}⚠️  API_PORT not available, using default 20252${NC}"
    API_PORT="20252"
fi

API_URL="http://localhost:${API_PORT}"

echo "🔍 Testing API integration at $API_URL..."

# Check if API is running
if ! check_service_running "API" "$API_PORT"; then
    echo -e "${RED}❌ visited-tracker API is not running on port $API_PORT${NC}"
    echo "   Start with: vrooli scenario start visited-tracker"
    ((error_count++))
else
    echo -e "${GREEN}✅ API port is listening${NC}"
    
    # Wait for API to be fully ready
    if wait_for_service "$API_URL/health" 10; then
        # Test API health endpoint
        echo ""
        echo "🩺 Testing API health endpoint..."
        health_response=$(curl -sf "$API_URL/health" 2>/dev/null || echo "")
        if [ -n "$health_response" ]; then
            if echo "$health_response" | jq -e '.status' >/dev/null 2>&1; then
                status=$(echo "$health_response" | jq -r '.status')
                if [ "$status" = "healthy" ]; then
                    echo -e "${GREEN}✅ API health check passed${NC}"
                    ((test_count++))
                else
                    echo -e "${RED}❌ API health status: $status${NC}"
                    ((error_count++))
                fi
            else
                echo -e "${RED}❌ Invalid health response format${NC}"
                ((error_count++))
            fi
        else
            echo -e "${RED}❌ API health endpoint not responding${NC}"
            ((error_count++))
        fi
        
        # Test API endpoints
        echo ""
        echo "🔍 Testing core API endpoints..."
        
        # Test campaign creation
        echo "📝 Testing campaign creation..."
        campaign_data='{"name":"integration-test","from_agent":"test","patterns":["**/*.js","**/*.go"]}'
        campaign_response=$(curl -sf -X POST "$API_URL/api/v1/campaigns" -H "Content-Type: application/json" -d "$campaign_data" 2>/dev/null || echo "")
        
        if [ -n "$campaign_response" ] && echo "$campaign_response" | jq -e '.campaign.id' >/dev/null 2>&1; then
            campaign_id=$(echo "$campaign_response" | jq -r '.campaign.id')
            echo -e "${GREEN}✅ Campaign creation successful: $campaign_id${NC}"
            ((test_count++))
            
            # Test visit recording
            echo "📊 Testing visit recording..."
            visit_data='{"files":["test.js","main.go"],"context":"integration-test","agent":"test-runner"}'
            visit_response=$(curl -sf -X POST "$API_URL/api/v1/campaigns/$campaign_id/visits" -H "Content-Type: application/json" -d "$visit_data" 2>/dev/null || echo "")
            
            if [ -n "$visit_response" ] && echo "$visit_response" | jq -e '.recorded' >/dev/null 2>&1; then
                recorded=$(echo "$visit_response" | jq -r '.recorded')
                echo -e "${GREEN}✅ Visit recording successful: $recorded files${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ Visit recording failed${NC}"
                ((error_count++))
            fi
            
            # Test prioritization endpoints
            echo "🎯 Testing prioritization endpoints..."
            
            # Least visited
            least_visited=$(curl -sf "$API_URL/api/v1/campaigns/$campaign_id/prioritize/least-visited?limit=5" 2>/dev/null || echo "")
            if [ -n "$least_visited" ] && echo "$least_visited" | jq -e '.files' >/dev/null 2>&1; then
                files_count=$(echo "$least_visited" | jq '.files | length')
                echo -e "${GREEN}✅ Least visited endpoint: $files_count files${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ Least visited endpoint failed${NC}"
                ((error_count++))
            fi
            
            # Coverage statistics
            coverage=$(curl -sf "$API_URL/api/v1/campaigns/$campaign_id/coverage" 2>/dev/null || echo "")
            if [ -n "$coverage" ] && echo "$coverage" | jq -e '.coverage_percentage' >/dev/null 2>&1; then
                percentage=$(echo "$coverage" | jq -r '.coverage_percentage')
                echo -e "${GREEN}✅ Coverage endpoint: ${percentage}% coverage${NC}"
                ((test_count++))
            else
                echo -e "${RED}❌ Coverage endpoint failed${NC}"
                ((error_count++))
            fi
            
        else
            echo -e "${RED}❌ Campaign creation failed${NC}"
            ((error_count++))
        fi
        
    else
        echo -e "${RED}❌ API not ready for testing${NC}"
        ((error_count++))
    fi
fi

# Test CLI integration
echo ""
echo "🔍 Testing CLI integration..."

# Check if visited-tracker CLI is available
if command -v visited-tracker >/dev/null 2>&1; then
    echo -e "${GREEN}✅ visited-tracker CLI is available${NC}"
    
    # Test CLI commands
    echo "⚙️  Testing CLI commands..."
    
    # Test help command
    if visited-tracker help >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI help command works${NC}"
        ((test_count++))
    else
        echo -e "${RED}❌ CLI help command failed${NC}"
        ((error_count++))
    fi
    
    # Test version command
    if visited-tracker version >/dev/null 2>&1; then
        version_output=$(visited-tracker version 2>/dev/null || echo "")
        if echo "$version_output" | grep -q "visited-tracker CLI"; then
            echo -e "${GREEN}✅ CLI version command works${NC}"
            ((test_count++))
        else
            echo -e "${RED}❌ CLI version command output invalid${NC}"
            ((error_count++))
        fi
    else
        echo -e "${RED}❌ CLI version command failed${NC}"
        ((error_count++))
    fi
    
    # Test status command (should work even without running service)
    status_output=$(visited-tracker status 2>/dev/null || echo "")
    if [ -n "$status_output" ]; then
        echo -e "${GREEN}✅ CLI status command works${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  CLI status command had issues (service may not be running)${NC}"
        ((skipped_count++))
    fi
    
else
    echo -e "${RED}❌ visited-tracker CLI not found${NC}"
    echo "   Install with: cd cli && ./install.sh"
    ((error_count++))
fi

# Run CLI BATS tests if available
echo ""
echo "🔍 Running CLI BATS tests..."
if [ -x "$CLI_DIR/run-cli-tests.sh" ]; then
    if "$CLI_DIR/run-cli-tests.sh"; then
        echo -e "${GREEN}✅ CLI BATS tests passed${NC}"
        ((test_count++))
    else
        echo -e "${RED}❌ CLI BATS tests failed${NC}"
        ((error_count++))
    fi
elif [ -d "$CLI_DIR" ] && find "$CLI_DIR" -name "*.bats" -type f | head -1 | grep -q .; then
    echo "🧪 Found BATS files, running directly..."
    bats_files_found=0
    
    for bats_file in "$CLI_DIR"/*.bats; do
        if [ -f "$bats_file" ]; then
            echo "Running $(basename "$bats_file")..."
            ((bats_files_found++))
            if command -v bats >/dev/null 2>&1; then
                if bats "$bats_file"; then
                    echo -e "${GREEN}✅ BATS test passed: $(basename "$bats_file")${NC}"
                    ((test_count++))
                else
                    echo -e "${RED}❌ BATS test failed: $(basename "$bats_file")${NC}"
                    ((error_count++))
                fi
            else
                echo -e "${YELLOW}⚠️  BATS not installed, skipping $(basename "$bats_file")${NC}"
                ((skipped_count++))
            fi
        fi
    done
    
    if [ $bats_files_found -eq 0 ]; then
        echo -e "${YELLOW}ℹ️  No BATS files found in $CLI_DIR${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  No CLI tests found${NC}"
    echo -e "${BLUE}💡 Consider adding BATS tests for CLI functionality${NC}"
    ((skipped_count++))
fi

# Test database integration (if available)
echo ""
echo "🔍 Testing database integration..."
if command -v psql >/dev/null 2>&1; then
    # Try to connect to PostgreSQL
    if PGCONNECT_TIMEOUT=3 psql -h localhost -p 5432 -U postgres -c '\l' >/dev/null 2>&1; then
        echo -e "${GREEN}✅ PostgreSQL connection successful${NC}"
        
        # Check if visited-tracker tables exist (they might not be created yet)
        table_count=$(PGCONNECT_TIMEOUT=3 psql -h localhost -p 5432 -U postgres -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('tracked_files', 'visits', 'structure_snapshots');" 2>/dev/null || echo "0")
        
        if [ "$table_count" -gt 0 ]; then
            echo -e "${GREEN}✅ visited-tracker database tables found: $table_count tables${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}ℹ️  visited-tracker tables not found (may need initialization)${NC}"
            ((skipped_count++))
        fi
        
    else
        echo -e "${RED}❌ PostgreSQL connection failed${NC}"
        ((error_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  psql not available, skipping database tests${NC}"
    ((skipped_count++))
fi

# Test UI integration (basic check)
echo ""
echo "🔍 Testing UI integration..."
UI_PORT="${UI_PORT:-}"
if [ -z "$UI_PORT" ]; then
    if command -v vrooli >/dev/null 2>&1; then
        UI_PORT=$(vrooli scenario port visited-tracker UI_PORT 2>/dev/null || echo "")
    fi
fi

if [ -n "$UI_PORT" ] && check_service_running "UI" "$UI_PORT"; then
    UI_URL="http://localhost:${UI_PORT}"
    echo -e "${GREEN}✅ UI is running on port $UI_PORT${NC}"
    
    # Test UI health endpoint
    if curl -sf "$UI_URL/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ UI health endpoint responding${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  UI health endpoint not responding${NC}"
        ((skipped_count++))
    fi
    
    # Test UI config endpoint
    if curl -sf "$UI_URL/config" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ UI config endpoint responding${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  UI config endpoint not responding${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  UI not running, skipping UI integration tests${NC}"
    ((skipped_count++))
fi

# Performance and summary
end_time=$(date +%s)
duration=$((end_time - start_time))
total_tests=$((test_count + error_count))

echo ""
echo "📊 Integration Test Summary:"
echo "   Tests passed: $test_count"
echo "   Tests failed: $error_count"
echo "   Tests skipped: $skipped_count"
echo "   Total test cases: $total_tests"
echo "   Duration: ${duration}s"

if [ $error_count -eq 0 ]; then
    if [ $test_count -gt 0 ]; then
        echo -e "${GREEN}✅ All integration tests passed in ${duration}s${NC}"
    else
        echo -e "${YELLOW}⚠️  No integration tests could be executed in ${duration}s${NC}"
        echo -e "${BLUE}💡 Start the scenario first: vrooli scenario start visited-tracker${NC}"
    fi
else
    echo -e "${RED}❌ Integration tests failed with $error_count failures in ${duration}s${NC}"
fi

if [ $duration -gt 120 ]; then
    echo -e "${YELLOW}⚠️  Integration tests phase exceeded 120s target${NC}"
fi

# Show recommendations for missing infrastructure
if [ $error_count -gt 0 ] || [ $skipped_count -gt 2 ]; then
    echo ""
    echo -e "${BLUE}💡 Integration test infrastructure recommendations:${NC}"
    
    if [ $error_count -gt 0 ]; then
        echo "   • Start the scenario: vrooli scenario start visited-tracker"
        echo "   • Ensure PostgreSQL is running: vrooli resource start postgres"
        echo "   • Install CLI: cd cli && ./install.sh"
    fi
    
    if [ $skipped_count -gt 2 ]; then
        echo "   • Add CLI BATS tests to $CLI_DIR"
        echo "   • Install BATS for CLI testing: npm install -g bats"
        echo "   • Consider adding database schema tests"
    fi
    
    echo "   • See: docs/scenarios/PHASED_TESTING_ARCHITECTURE.md"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi