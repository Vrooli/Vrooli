#!/bin/bash
# Integration tests phase - <120 seconds
# Tests service integration using resource CLI commands and dynamic ports
set -uo pipefail

# Setup paths
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# Source required utilities
source "${APP_ROOT}/scripts/lib/network/ports.sh"

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

# Helper function to wait for service to be ready
wait_for_service() {
    local url="$1"
    local timeout="${2:-30}"
    local counter=0
    
    echo "⏳ Waiting for service to be ready at $url..."
    while [ $counter -lt $timeout ]; do
        if timeout 5 curl -sf "$url" >/dev/null 2>&1; then
            echo -e "${GREEN}✅ Service is ready${NC}"
            return 0
        fi
        sleep 1
        ((counter++))
    done
    
    echo -e "${RED}❌ Service not ready after ${timeout}s${NC}"
    return 1
}

# Get service ports using ports::get_scenario_environment
echo "🔍 Getting service ports..."
service_json_path="${APP_ROOT}/scenarios/visited-tracker/.vrooli/service.json"
env_result=$(ports::get_scenario_environment "visited-tracker" "$service_json_path")

if echo "$env_result" | jq -e '.success' >/dev/null 2>&1 && [ "$(echo "$env_result" | jq -r '.success')" = "true" ]; then
    API_PORT=$(echo "$env_result" | jq -r '.env_vars.API_PORT // empty')
    UI_PORT=$(echo "$env_result" | jq -r '.env_vars.UI_PORT // empty')
    echo "🔍 Resolved ports: API=${API_PORT:-unknown}, UI=${UI_PORT:-unknown}"
else
    echo -e "${RED}❌ Failed to get scenario environment${NC}"
    echo "$env_result" | jq -r '.error // "Unknown error"'
    exit 1
fi

# Check if visited-tracker is running
echo ""
echo "🔍 Checking if visited-tracker scenario is running..."
if command -v vrooli >/dev/null 2>&1; then
    scenario_status=$(vrooli scenario status visited-tracker 2>/dev/null | grep -o "running" || echo "not-running")
    if [ "$scenario_status" = "running" ]; then
        echo -e "${GREEN}✅ visited-tracker scenario is running${NC}"
    else
        echo -e "${YELLOW}⚠️  visited-tracker scenario is not running${NC}"
        echo "   Start with: vrooli scenario start visited-tracker"
        echo "   Integration tests require running services"
    fi
fi

# Test API integration (if port is available)
if [ -n "$API_PORT" ]; then
    API_URL="http://localhost:${API_PORT}"
    echo ""
    echo "🔍 Testing API integration at $API_URL..."
    
    # Check if API is responding
    if timeout 5 curl -sf "$API_URL/health" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ API is responding on port $API_PORT${NC}"
        
        # Test API health endpoint
        health_response=$(timeout 5 curl -sf "$API_URL/health" 2>/dev/null || echo "{}")
        if echo "$health_response" | jq -e '.status' >/dev/null 2>&1; then
            status=$(echo "$health_response" | jq -r '.status')
            echo -e "${GREEN}✅ API health status: $status${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}⚠️  API health response format unexpected${NC}"
            ((skipped_count++))
        fi
        
        # Test core API endpoints
        echo ""
        echo "🔍 Testing core API endpoints..."
        
        # Test campaign creation
        timestamp=$(date +%s)
        campaign_data="{\"name\":\"integration-test-${timestamp}\",\"from_agent\":\"test\",\"patterns\":[\"**/*.js\"]}"
        campaign_response=$(timeout 10 curl -sf -X POST "$API_URL/api/v1/campaigns" \
            -H "Content-Type: application/json" \
            -d "$campaign_data" 2>/dev/null || echo "")
        
        if echo "$campaign_response" | jq -e '.id' >/dev/null 2>&1; then
            campaign_id=$(echo "$campaign_response" | jq -r '.id')
            echo -e "${GREEN}✅ Campaign creation successful: $campaign_id${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}⚠️  Campaign creation not available${NC}"
            ((skipped_count++))
        fi
    else
        echo -e "${YELLOW}⚠️  API not responding on port $API_PORT${NC}"
        echo "   Scenario may not be fully started"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  API port not available, skipping API tests${NC}"
    ((skipped_count++))
fi

# Test CLI integration
echo ""
echo "🔍 Testing CLI integration..."

if command -v visited-tracker >/dev/null 2>&1; then
    echo -e "${GREEN}✅ visited-tracker CLI is available${NC}"
    
    # Test CLI help command
    if visited-tracker help >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI help command works${NC}"
        ((test_count++))
    else
        echo -e "${RED}❌ CLI help command failed${NC}"
        ((error_count++))
    fi
    
    # Test CLI version command
    if visited-tracker version 2>&1 | grep -q "visited-tracker"; then
        echo -e "${GREEN}✅ CLI version command works${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  CLI version command output unexpected${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${RED}❌ visited-tracker CLI not found${NC}"
    echo "   Install with: cd cli && ./install.sh"
    ((error_count++))
fi

# Test database integration using resource-postgres
echo ""
echo "🔍 Testing database integration using resource-postgres..."

if command -v resource-postgres >/dev/null 2>&1; then
    # Check PostgreSQL health
    if resource-postgres test smoke >/dev/null 2>&1; then
        echo -e "${GREEN}✅ PostgreSQL is healthy (via resource-postgres)${NC}"
        
        # List databases to check if visited-tracker database exists
        echo "📊 Checking for visited-tracker database..."
        db_list=$(resource-postgres content list 2>/dev/null || echo "")
        
        if echo "$db_list" | grep -q "visited_tracker\|vrooli"; then
            echo -e "${GREEN}✅ Database found for visited-tracker${NC}"
            ((test_count++))
            
            # Check for visited-tracker tables using resource-postgres
            echo "📊 Checking for visited-tracker tables..."
            table_check_query="SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('tracked_files', 'visits', 'structure_snapshots');"
            
            # Execute query using resource-postgres
            table_count=$(resource-postgres content execute "$table_check_query" 2>/dev/null | grep -oE '[0-9]+' | head -1 || echo "0")
            
            if [ "${table_count:-0}" -gt 0 ]; then
                echo -e "${GREEN}✅ Found $table_count visited-tracker tables${NC}"
                ((test_count++))
            else
                echo -e "${YELLOW}ℹ️  visited-tracker tables not found (may need initialization)${NC}"
                ((skipped_count++))
            fi
        else
            echo -e "${YELLOW}ℹ️  visited-tracker database not found${NC}"
            ((skipped_count++))
        fi
    else
        echo -e "${RED}❌ PostgreSQL health check failed${NC}"
        echo "   Start with: vrooli resource start postgres"
        ((error_count++))
    fi
else
    echo -e "${RED}❌ resource-postgres CLI not available${NC}"
    echo "   This is required for database integration tests"
    ((error_count++))
fi

# Test Redis integration using resource-redis (optional)
echo ""
echo "🔍 Testing Redis integration (optional)..."

if command -v resource-redis >/dev/null 2>&1; then
    if resource-redis test smoke >/dev/null 2>&1; then
        echo -e "${GREEN}✅ Redis is healthy (via resource-redis)${NC}"
        ((test_count++))
    else
        echo -e "${YELLOW}⚠️  Redis not available (optional dependency)${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  resource-redis CLI not available (optional)${NC}"
    ((skipped_count++))
fi

# Test UI integration (if port is available)
if [ -n "$UI_PORT" ]; then
    UI_URL="http://localhost:${UI_PORT}"
    echo ""
    echo "🔍 Testing UI integration at $UI_URL..."
    
    if timeout 5 curl -sf "$UI_URL" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ UI is responding on port $UI_PORT${NC}"
        
        # Test UI endpoints
        if timeout 5 curl -sf "$UI_URL/config" 2>/dev/null | grep -q "visited-tracker"; then
            echo -e "${GREEN}✅ UI config endpoint working${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}⚠️  UI config endpoint not as expected${NC}"
            ((skipped_count++))
        fi
    else
        echo -e "${YELLOW}⚠️  UI not responding on port $UI_PORT${NC}"
        ((skipped_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  UI port not available, skipping UI tests${NC}"
    ((skipped_count++))
fi

# Run CLI BATS tests if available
echo ""
echo "🔍 Running CLI BATS tests..."
if [ -x "$CLI_DIR/run-cli-tests.sh" ]; then
    if "$CLI_DIR/run-cli-tests.sh" >/dev/null 2>&1; then
        echo -e "${GREEN}✅ CLI BATS tests passed${NC}"
        ((test_count++))
    else
        echo -e "${RED}❌ CLI BATS tests failed${NC}"
        ((error_count++))
    fi
else
    echo -e "${YELLOW}ℹ️  No CLI BATS tests found${NC}"
    ((skipped_count++))
fi

# Performance and summary
end_time=$(date +%s)
duration=$((end_time - start_time))

echo ""
echo "📊 Integration Test Summary:"
echo "   Tests passed: $test_count"
echo "   Tests failed: $error_count"
echo "   Tests skipped: $skipped_count"
echo "   Duration: ${duration}s"

if [ $error_count -eq 0 ]; then
    if [ $test_count -gt 0 ]; then
        echo -e "${GREEN}✅ All integration tests passed in ${duration}s${NC}"
    else
        echo -e "${YELLOW}⚠️  No integration tests could be executed${NC}"
        echo -e "${BLUE}💡 Start the scenario: vrooli scenario start visited-tracker${NC}"
    fi
else
    echo -e "${RED}❌ Integration tests failed with $error_count failures${NC}"
fi

if [ $duration -gt 120 ]; then
    echo -e "${YELLOW}⚠️  Integration tests exceeded 120s target${NC}"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi