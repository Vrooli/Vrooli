#!/usr/bin/env bash
# Performance monitoring script for Metareasoning API

set -e

cd "$(dirname "$0")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

API_URL=${API_URL:-"http://localhost:8093"}
AUTH_TOKEN=${AUTH_TOKEN:-"metareasoning_cli_default_2024"}

echo -e "${BLUE}=== Metareasoning API Performance Monitor ===${NC}"
echo "API URL: $API_URL"
echo "Timestamp: $(date)"
echo

# Function to make authenticated API calls and measure performance
api_call() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="$3"
    
    local start_time=$(date +%s%3N)
    
    if [[ "$method" == "POST" ]] && [[ -n "$data" ]]; then
        response=$(curl -s -w "\n%{http_code}\n%{time_total}" \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            -X "$method" \
            -d "$data" \
            "$API_URL$endpoint" 2>/dev/null)
    else
        response=$(curl -s -w "\n%{http_code}\n%{time_total}" \
            -H "Authorization: Bearer $AUTH_TOKEN" \
            -X "$method" \
            "$API_URL$endpoint" 2>/dev/null)
    fi
    
    local end_time=$(date +%s%3N)
    local total_time=$((end_time - start_time))
    
    # Parse response
    local body=$(echo "$response" | head -n -2)
    local status_code=$(echo "$response" | tail -n 2 | head -n 1)
    local curl_time=$(echo "$response" | tail -n 1)
    
    # Convert curl time to milliseconds
    local curl_time_ms=$(echo "$curl_time * 1000" | bc -l | cut -d. -f1)
    
    echo "  Status: $status_code | Time: ${curl_time_ms}ms | Total: ${total_time}ms"
    
    # Check for performance issues
    if [[ $curl_time_ms -gt 100 ]]; then
        echo -e "    ${YELLOW}⚠ Slow response: ${curl_time_ms}ms > 100ms${NC}"
    elif [[ $curl_time_ms -gt 50 ]]; then
        echo -e "    ${YELLOW}⚠ Moderate response time: ${curl_time_ms}ms${NC}"
    else
        echo -e "    ${GREEN}✓ Good response time: ${curl_time_ms}ms${NC}"
    fi
    
    # Check for X-Response-Time header if available
    local response_time_header=$(curl -s -I -H "Authorization: Bearer $AUTH_TOKEN" "$API_URL$endpoint" 2>/dev/null | grep -i "x-response-time" | cut -d: -f2 | tr -d ' \r')
    if [[ -n "$response_time_header" ]]; then
        echo "    Server reports: $response_time_header"
    fi
    
    # Check for cache headers
    local cache_header=$(curl -s -I -H "Authorization: Bearer $AUTH_TOKEN" "$API_URL$endpoint" 2>/dev/null | grep -i "x-cache" | cut -d: -f2 | tr -d ' \r')
    if [[ -n "$cache_header" ]]; then
        echo "    Cache: $cache_header"
    fi
    
    echo
}

# Check if API is running
echo "Checking API availability..."
if ! curl -s "$API_URL/health" > /dev/null; then
    echo -e "${RED}❌ API is not running at $API_URL${NC}"
    echo "Please start the API first with: go run . or make run"
    exit 1
fi

echo -e "${GREEN}✓ API is running${NC}"
echo

# Test performance of key endpoints
echo -e "${BLUE}Testing endpoint performance:${NC}"

echo "1. Health check (should be <10ms):"
api_call "/health" "GET"

echo "2. Models endpoint (should be <100ms):"
api_call "/models" "GET"

echo "3. Platforms endpoint (should be <50ms):"
api_call "/platforms" "GET"

echo "4. System stats (should be <200ms):"
api_call "/stats" "GET"

echo "5. List workflows (should be <100ms):"
api_call "/workflows?page=1&page_size=10" "GET"

echo "6. Search workflows (should be <150ms):"
api_call "/workflows/search?q=test" "GET"

# Test with data payload
echo "7. Generate workflow (AI operation, expected <5000ms):"
generate_data='{"prompt": "Create a simple webhook workflow", "platform": "n8n", "model": "llama2", "temperature": 0.7}'
api_call "/workflows/generate" "POST" "$generate_data"

echo "8. Import workflow (should be <200ms):"
import_data='{"platform": "n8n", "name": "Test Import", "data": {"nodes": [{"id": "1", "type": "webhook"}]}}'
api_call "/workflows/import" "POST" "$import_data"

# Test caching performance
echo -e "${BLUE}Testing caching performance:${NC}"
echo "Making repeated calls to test caching..."

for i in {1..3}; do
    echo "Call $i to /models:"
    api_call "/models" "GET"
done

# Load testing with concurrent requests
echo -e "${BLUE}Running concurrent request test:${NC}"
echo "Testing with 10 concurrent health check requests..."

start_time=$(date +%s%3N)
for i in {1..10}; do
    curl -s -H "Authorization: Bearer $AUTH_TOKEN" "$API_URL/health" > /dev/null &
done
wait
end_time=$(date +%s%3N)
concurrent_time=$((end_time - start_time))

echo "10 concurrent requests completed in ${concurrent_time}ms"
if [[ $concurrent_time -gt 1000 ]]; then
    echo -e "${RED}❌ Concurrent performance issue: ${concurrent_time}ms > 1000ms${NC}"
else
    echo -e "${GREEN}✓ Good concurrent performance: ${concurrent_time}ms${NC}"
fi

echo

# Memory and resource usage (if running locally)
if command -v ps > /dev/null; then
    echo -e "${BLUE}Resource usage:${NC}"
    api_pid=$(ps aux | grep "[g]o run ." | awk '{print $2}' | head -1)
    if [[ -n "$api_pid" ]]; then
        ps_output=$(ps -p "$api_pid" -o pid,pcpu,pmem,vsz,rss,time)
        echo "$ps_output"
    else
        echo "API process not found in process list"
    fi
    echo
fi

# Performance summary
echo -e "${BLUE}Performance Summary:${NC}"
echo -e "${GREEN}✓ Target: <100ms response time for most endpoints${NC}"
echo -e "${GREEN}✓ Health endpoint should be <10ms${NC}"
echo -e "${GREEN}✓ CRUD operations should be <100ms${NC}"
echo -e "${YELLOW}⚠ AI operations (generate) may take longer (up to 5s)${NC}"

# Recommendations
echo
echo -e "${BLUE}Optimization Recommendations:${NC}"
echo "1. Monitor slow endpoints (>100ms) regularly"
echo "2. Implement Redis caching for better performance"
echo "3. Add database query optimization"
echo "4. Use connection pooling for external services"
echo "5. Consider implementing request rate limiting"
echo "6. Add APM (Application Performance Monitoring)"

echo
echo -e "${BLUE}=== Performance monitoring completed ===${NC}"