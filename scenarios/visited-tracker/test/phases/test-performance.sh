#!/bin/bash
# Performance tests phase - <60 seconds
# Tests response times, concurrent requests, resource utilization, and performance baselines
set -euo pipefail

echo "=== Performance Tests Phase (Target: <60s) ==="
start_time=$(date +%s)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
PURPLE='\033[0;35m'
NC='\033[0m'

error_count=0
test_count=0
skipped_count=0
warning_count=0

# Performance thresholds (in milliseconds)
HEALTH_THRESHOLD=500
API_THRESHOLD=2000
BULK_THRESHOLD=5000
UI_THRESHOLD=3000

# Helper function to measure response time
measure_response_time() {
    local url="$1"
    local method="${2:-GET}"
    local data="${3:-}"
    local content_type="${4:-application/json}"
    
    local curl_args=("-w" "%{time_total}" "-s" "-o" "/dev/null")
    
    if [ "$method" = "POST" ]; then
        curl_args+=("-X" "POST" "-H" "Content-Type: $content_type")
        if [ -n "$data" ]; then
            curl_args+=("-d" "$data")
        fi
    fi
    
    curl_args+=("$url")
    
    # Return response time in milliseconds (curl returns seconds)
    local time_seconds
    time_seconds=$(curl "${curl_args[@]}" 2>/dev/null || echo "999")
    echo "$(echo "$time_seconds * 1000" | bc -l 2>/dev/null | cut -d. -f1)"
}

# Helper function to check if service is responsive
check_service_performance() {
    local url="$1"
    local threshold="$2"
    local test_name="$3"
    
    echo "⚡ Testing $test_name response time..."
    local response_time
    response_time=$(measure_response_time "$url")
    
    echo "   Response time: ${response_time}ms (threshold: ${threshold}ms)"
    
    if [ "$response_time" -lt "$threshold" ]; then
        echo -e "${GREEN}✅ $test_name performance: ${response_time}ms < ${threshold}ms${NC}"
        ((test_count++))
        return 0
    elif [ "$response_time" -lt $((threshold * 2)) ]; then
        echo -e "${YELLOW}⚠️  $test_name performance warning: ${response_time}ms (slow but acceptable)${NC}"
        ((warning_count++))
        ((test_count++))
        return 0
    else
        echo -e "${RED}❌ $test_name performance failure: ${response_time}ms > ${threshold}ms${NC}"
        ((error_count++))
        return 1
    fi
}

# Helper function for concurrent testing
run_concurrent_requests() {
    local url="$1"
    local concurrent_count="$2"
    local test_name="$3"
    
    echo "🚀 Testing $test_name with $concurrent_count concurrent requests..."
    
    local temp_dir="/tmp/visited-tracker-perf-$$"
    mkdir -p "$temp_dir"
    
    # Start concurrent requests in background
    for i in $(seq 1 "$concurrent_count"); do
        (
            response_time=$(measure_response_time "$url")
            echo "$response_time" > "$temp_dir/response_$i.txt"
        ) &
    done
    
    # Wait for all requests to complete
    wait
    
    # Analyze results
    local total_time=0
    local max_time=0
    local min_time=999999
    local successful_requests=0
    
    for i in $(seq 1 "$concurrent_count"); do
        if [ -f "$temp_dir/response_$i.txt" ]; then
            local time_ms
            time_ms=$(cat "$temp_dir/response_$i.txt")
            if [ "$time_ms" -lt 999000 ]; then  # Filter out timeout/error responses
                total_time=$((total_time + time_ms))
                if [ "$time_ms" -gt "$max_time" ]; then
                    max_time=$time_ms
                fi
                if [ "$time_ms" -lt "$min_time" ]; then
                    min_time=$time_ms
                fi
                ((successful_requests++))
            fi
        fi
    done
    
    # Cleanup
    rm -rf "$temp_dir"
    
    if [ "$successful_requests" -gt 0 ]; then
        local avg_time=$((total_time / successful_requests))
        echo "   Successful requests: $successful_requests/$concurrent_count"
        echo "   Average response time: ${avg_time}ms"
        echo "   Min/Max response time: ${min_time}ms / ${max_time}ms"
        
        # Performance evaluation
        if [ "$successful_requests" -eq "$concurrent_count" ] && [ "$avg_time" -lt $((API_THRESHOLD * 2)) ]; then
            echo -e "${GREEN}✅ $test_name concurrent performance: ${avg_time}ms average${NC}"
            ((test_count++))
        elif [ "$successful_requests" -ge $((concurrent_count * 80 / 100)) ]; then
            echo -e "${YELLOW}⚠️  $test_name concurrent performance degraded: ${avg_time}ms average${NC}"
            ((warning_count++))
            ((test_count++))
        else
            echo -e "${RED}❌ $test_name concurrent performance failure: only $successful_requests/$concurrent_count succeeded${NC}"
            ((error_count++))
        fi
    else
        echo -e "${RED}❌ $test_name concurrent test failed: no successful requests${NC}"
        ((error_count++))
    fi
}

# Get service ports
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

echo "⚡ Performance testing Visited Tracker"
echo "   API: $API_URL"
echo "   UI: $UI_URL"

# Check if required tools are available
required_tools=("curl" "bc")
for tool in "${required_tools[@]}"; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        echo -e "${RED}❌ Required tool missing: $tool${NC}"
        ((error_count++))
    fi
done

# Check if services are running
api_running=false
ui_running=false

if timeout 3 nc -z localhost "$API_PORT" 2>/dev/null; then
    api_running=true
    echo -e "${GREEN}✅ API service detected on port $API_PORT${NC}"
else
    echo -e "${RED}❌ API service not running on port $API_PORT${NC}"
    echo "   Start with: vrooli scenario start visited-tracker"
    ((error_count++))
fi

if timeout 3 nc -z localhost "$UI_PORT" 2>/dev/null; then
    ui_running=true
    echo -e "${GREEN}✅ UI service detected on port $UI_PORT${NC}"
else
    echo -e "${YELLOW}⚠️  UI service not running on port $UI_PORT${NC}"
    ((warning_count++))
fi

# Test 1: API Response Time Baselines
echo ""
echo -e "${PURPLE}🎯 Test 1: API Response Time Baselines${NC}"

if [ "$api_running" = "true" ]; then
    # Health endpoint performance
    check_service_performance "$API_URL/health" "$HEALTH_THRESHOLD" "Health endpoint"
    
    # API endpoints performance
    check_service_performance "$API_URL/api/v1/campaigns" "$API_THRESHOLD" "Campaigns list"
    
    # Test with sample data if possible
    echo "📝 Testing API endpoints with sample operations..."
    
    # Create a test campaign for performance testing
    campaign_data='{"name":"perf-test","from_agent":"performance-tester","patterns":["**/*.js"]}'
    campaign_response=$(curl -sf -X POST "$API_URL/api/v1/campaigns" -H "Content-Type: application/json" -d "$campaign_data" 2>/dev/null || echo "")
    
    if [ -n "$campaign_response" ] && echo "$campaign_response" | jq -e '.campaign.id' >/dev/null 2>&1; then
        campaign_id=$(echo "$campaign_response" | jq -r '.campaign.id')
        echo -e "${GREEN}✅ Performance test campaign created: $campaign_id${NC}"
        
        # Test visit recording performance
        visit_data='{"files":["test1.js","test2.js","test3.js"],"context":"perf-test"}'
        visit_time=$(measure_response_time "$API_URL/api/v1/campaigns/$campaign_id/visits" "POST" "$visit_data")
        
        echo "   Visit recording time: ${visit_time}ms (threshold: ${API_THRESHOLD}ms)"
        if [ "$visit_time" -lt "$API_THRESHOLD" ]; then
            echo -e "${GREEN}✅ Visit recording performance: ${visit_time}ms${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}⚠️  Visit recording slower than expected: ${visit_time}ms${NC}"
            ((warning_count++))
        fi
        
        # Test prioritization performance
        priority_time=$(measure_response_time "$API_URL/api/v1/campaigns/$campaign_id/prioritize/least-visited?limit=10")
        
        echo "   Prioritization time: ${priority_time}ms (threshold: ${API_THRESHOLD}ms)"
        if [ "$priority_time" -lt "$API_THRESHOLD" ]; then
            echo -e "${GREEN}✅ Prioritization performance: ${priority_time}ms${NC}"
            ((test_count++))
        else
            echo -e "${YELLOW}⚠️  Prioritization slower than expected: ${priority_time}ms${NC}"
            ((warning_count++))
        fi
        
    else
        echo -e "${YELLOW}⚠️  Could not create test campaign for performance testing${NC}"
        ((skipped_count++))
    fi
    
else
    echo -e "${YELLOW}ℹ️  Skipping API performance tests (service not running)${NC}"
    ((skipped_count += 4))
fi

# Test 2: Concurrent Request Handling
echo ""
echo -e "${PURPLE}🎯 Test 2: Concurrent Request Handling${NC}"

if [ "$api_running" = "true" ]; then
    # Test health endpoint under load
    run_concurrent_requests "$API_URL/health" 5 "Health endpoint concurrent"
    
    # Test campaigns endpoint under load
    run_concurrent_requests "$API_URL/api/v1/campaigns" 3 "Campaigns endpoint concurrent"
    
else
    echo -e "${YELLOW}ℹ️  Skipping concurrent tests (API not running)${NC}"
    ((skipped_count += 2))
fi

# Test 3: UI Performance
echo ""
echo -e "${PURPLE}🎯 Test 3: UI Performance${NC}"

if [ "$ui_running" = "true" ]; then
    # UI page load performance
    check_service_performance "$UI_URL" "$UI_THRESHOLD" "UI main page"
    
    # UI config endpoint performance
    check_service_performance "$UI_URL/config" "$HEALTH_THRESHOLD" "UI config endpoint"
    
    # Test static resource performance (if available)
    static_resources=("style.css" "script.js" "favicon.ico")
    for resource in "${static_resources[@]}"; do
        resource_url="$UI_URL/$resource"
        if curl -sf "$resource_url" >/dev/null 2>&1; then
            resource_time=$(measure_response_time "$resource_url")
            echo "   Static resource $resource: ${resource_time}ms"
            if [ "$resource_time" -lt 1000 ]; then
                echo -e "${GREEN}✅ Static resource performance: $resource${NC}"
                ((test_count++))
            else
                echo -e "${YELLOW}⚠️  Static resource slow: $resource (${resource_time}ms)${NC}"
                ((warning_count++))
            fi
        else
            echo -e "${YELLOW}ℹ️  Static resource not found: $resource${NC}"
        fi
    done
    
else
    echo -e "${YELLOW}ℹ️  Skipping UI performance tests (service not running)${NC}"
    ((skipped_count += 2))
fi

# Test 4: Memory and Resource Utilization (Basic Check)
echo ""
echo -e "${PURPLE}🎯 Test 4: Resource Utilization Check${NC}"

if [ "$api_running" = "true" ] || [ "$ui_running" = "true" ]; then
    echo "💾 Checking basic resource utilization..."
    
    # Check if process monitoring tools are available
    if command -v ps >/dev/null 2>&1; then
        # Find visited-tracker processes
        api_processes=$(ps aux | grep "[v]isited-tracker-api" | wc -l || echo "0")
        ui_processes=$(ps aux | grep "[s]erver\.js.*visited-tracker" | wc -l || echo "0")
        
        echo "   API processes running: $api_processes"
        echo "   UI processes running: $ui_processes"
        
        if [ "$api_processes" -gt 0 ] || [ "$ui_processes" -gt 0 ]; then
            echo -e "${GREEN}✅ Processes are running and detectable${NC}"
            ((test_count++))
            
            # Basic memory check if available
            if command -v free >/dev/null 2>&1; then
                available_memory=$(free -m | awk 'NR==2{printf "%.0f", $7}')
                if [ "$available_memory" -gt 100 ]; then
                    echo -e "${GREEN}✅ Sufficient memory available: ${available_memory}MB${NC}"
                    ((test_count++))
                else
                    echo -e "${YELLOW}⚠️  Low available memory: ${available_memory}MB${NC}"
                    ((warning_count++))
                fi
            else
                echo -e "${YELLOW}ℹ️  Memory monitoring not available${NC}"
                ((skipped_count++))
            fi
        else
            echo -e "${YELLOW}⚠️  Process detection may be inaccurate${NC}"
            ((skipped_count++))
        fi
        
    else
        echo -e "${YELLOW}ℹ️  Process monitoring tools not available${NC}"
        ((skipped_count += 2))
    fi
    
else
    echo -e "${YELLOW}ℹ️  Skipping resource utilization (no services running)${NC}"
    ((skipped_count += 2))
fi

# Test 5: Bulk Operations Performance
echo ""
echo -e "${PURPLE}🎯 Test 5: Bulk Operations Performance${NC}"

if [ "$api_running" = "true" ] && [ -n "${campaign_id:-}" ]; then
    echo "📦 Testing bulk operations performance..."
    
    # Test bulk visit recording
    bulk_files='["file1.js","file2.js","file3.js","file4.js","file5.js","file6.js","file7.js","file8.js","file9.js","file10.js"]'
    bulk_visit_data='{"files":'"$bulk_files"',"context":"bulk-perf-test","agent":"bulk-tester"}'
    
    bulk_time=$(measure_response_time "$API_URL/api/v1/campaigns/$campaign_id/visits" "POST" "$bulk_visit_data")
    
    echo "   Bulk visit recording (10 files): ${bulk_time}ms (threshold: ${BULK_THRESHOLD}ms)"
    
    if [ "$bulk_time" -lt "$BULK_THRESHOLD" ]; then
        echo -e "${GREEN}✅ Bulk operations performance: ${bulk_time}ms${NC}"
        ((test_count++))
    elif [ "$bulk_time" -lt $((BULK_THRESHOLD * 2)) ]; then
        echo -e "${YELLOW}⚠️  Bulk operations acceptable: ${bulk_time}ms${NC}"
        ((warning_count++))
        ((test_count++))
    else
        echo -e "${RED}❌ Bulk operations too slow: ${bulk_time}ms${NC}"
        ((error_count++))
    fi
    
else
    echo -e "${YELLOW}ℹ️  Skipping bulk operations test (API not running or no test campaign)${NC}"
    ((skipped_count++))
fi

# Performance summary and analysis
end_time=$(date +%s)
duration=$((end_time - start_time))
total_tests=$((test_count + error_count))

echo ""
echo "📊 Performance Test Summary:"
echo "   Tests passed: $test_count"
echo "   Tests failed: $error_count"
echo "   Tests skipped: $skipped_count"
echo "   Performance warnings: $warning_count"
echo "   Total test cases: $total_tests"
echo "   Duration: ${duration}s"

# Performance analysis
echo ""
echo "⚡ Performance Analysis:"
if [ $error_count -eq 0 ]; then
    if [ $warning_count -eq 0 ]; then
        echo -e "${GREEN}✅ Excellent performance: All tests passed within thresholds${NC}"
    elif [ $warning_count -le 2 ]; then
        echo -e "${GREEN}✅ Good performance: Minor warnings but acceptable${NC}"
    else
        echo -e "${YELLOW}⚠️  Performance concerns: Multiple warnings detected${NC}"
    fi
else
    echo -e "${RED}❌ Performance issues: $error_count tests failed thresholds${NC}"
fi

if [ $test_count -gt 0 ]; then
    echo -e "${GREEN}🎯 Performance testing completed in ${duration}s${NC}"
else
    echo -e "${YELLOW}⚠️  No performance tests could be executed in ${duration}s${NC}"
    echo -e "${BLUE}💡 Start the scenario first: vrooli scenario start visited-tracker${NC}"
fi

if [ $duration -gt 60 ]; then
    echo -e "${YELLOW}⚠️  Performance tests phase exceeded 60s target${NC}"
fi

# Performance recommendations
if [ $error_count -gt 0 ] || [ $warning_count -gt 2 ]; then
    echo ""
    echo -e "${BLUE}💡 Performance optimization recommendations:${NC}"
    
    if [ $error_count -gt 0 ]; then
        echo "   • API response times exceed thresholds - consider optimization"
        echo "   • Check database query performance and indexing"
        echo "   • Review concurrent request handling capacity"
    fi
    
    if [ $warning_count -gt 2 ]; then
        echo "   • Multiple performance warnings - monitor under load"
        echo "   • Consider adding response caching for read operations"
        echo "   • Review resource allocation and scaling strategies"
    fi
    
    echo "   • Monitor performance trends over time"
    echo "   • Set up performance alerts for production deployment"
    echo "   • See: docs/scenarios/PHASED_TESTING_ARCHITECTURE.md"
fi

# Exit with appropriate code
if [ $error_count -eq 0 ]; then
    exit 0
else
    exit 1
fi