#!/bin/bash
set -e
echo "=== Performance Tests ==="

# API_PORT is set by the lifecycle system when run via 'make test' or 'vrooli scenario test'
API_PORT=${API_PORT:-19796}

# Configuration
API_URL="${API_URL:-http://localhost:${API_PORT}}"
# Read token from environment, fallback to default for development/testing
AUTH_TOKEN="Bearer ${DATA_TOOLS_API_TOKEN:-data-tools-secret-token}"
PERFORMANCE_THRESHOLD_HEALTH=200
PERFORMANCE_THRESHOLD_PARSE=300
PERFORMANCE_THRESHOLD_TRANSFORM=300
PERFORMANCE_THRESHOLD_VALIDATE=300
PERFORMANCE_THRESHOLD_QUERY=500

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Helper function to measure response time
measure_response_time() {
    local endpoint="$1"
    local method="${2:-GET}"
    local data="${3:-}"

    if [ "$method" = "POST" ] && [ -n "$data" ]; then
        response_time=$(curl -o /dev/null -s -w '%{time_total}' -X POST \
            -H "Authorization: $AUTH_TOKEN" \
            -H "Content-Type: application/json" \
            -d "$data" \
            "${API_URL}${endpoint}" 2>/dev/null || echo "999")
    else
        response_time=$(curl -o /dev/null -s -w '%{time_total}' \
            -H "Authorization: $AUTH_TOKEN" \
            "${API_URL}${endpoint}" 2>/dev/null || echo "999")
    fi

    # Convert to milliseconds
    echo "$response_time * 1000" | bc
}

# Helper function to check threshold
check_threshold() {
    local name="$1"
    local response_time="$2"
    local threshold="$3"

    # Use bc for decimal comparison
    result=$(echo "$response_time < $threshold" | bc -l)

    if [ "$result" -eq 1 ]; then
        echo -e "${GREEN}✅ $name: ${response_time}ms (< ${threshold}ms target)${NC}"
        return 0
    else
        echo -e "${RED}❌ $name: ${response_time}ms (exceeds ${threshold}ms target)${NC}"
        return 1
    fi
}

echo "Testing API performance against defined thresholds..."
echo ""

# Test 1: Health endpoint performance
echo "Test 1: Health Endpoint Performance"
health_time=$(measure_response_time "/health")
check_threshold "Health endpoint" "$health_time" "$PERFORMANCE_THRESHOLD_HEALTH"
echo ""

# Test 2: Parse endpoint performance
echo "Test 2: Parse Endpoint Performance"
parse_data='{"data":"name,age\nJohn,30\nJane,25","format":"csv"}'
parse_time=$(measure_response_time "/api/v1/data/parse" "POST" "$parse_data")
check_threshold "Data parsing" "$parse_time" "$PERFORMANCE_THRESHOLD_PARSE"
echo ""

# Test 3: Transform endpoint performance
echo "Test 3: Transform Endpoint Performance"
transform_data='{"data":[{"age":30},{"age":20}],"transformations":[{"type":"filter","parameters":{"condition":"age > 25"}}]}'
transform_time=$(measure_response_time "/api/v1/data/transform" "POST" "$transform_data")
check_threshold "Data transformation" "$transform_time" "$PERFORMANCE_THRESHOLD_TRANSFORM"
echo ""

# Test 4: Validate endpoint performance
echo "Test 4: Validate Endpoint Performance"
validate_data='{"data":[{"name":"John"}],"schema":{"columns":[{"name":"name","type":"string"}]}}'
validate_time=$(measure_response_time "/api/v1/data/validate" "POST" "$validate_data")
check_threshold "Data validation" "$validate_time" "$PERFORMANCE_THRESHOLD_VALIDATE"
echo ""

# Test 5: Query endpoint performance
echo "Test 5: Query Endpoint Performance"
query_data='{"sql":"SELECT 1 as test"}'
query_time=$(measure_response_time "/api/v1/data/query" "POST" "$query_data")
check_threshold "SQL query" "$query_time" "$PERFORMANCE_THRESHOLD_QUERY"
echo ""

# Test 6: Concurrent requests (load test)
echo "Test 6: Concurrent Request Handling"
echo "Running 10 concurrent health checks..."
start_time=$(date +%s.%N)
for i in {1..10}; do
    curl -s -H "Authorization: $AUTH_TOKEN" "${API_URL}/health" > /dev/null &
done
wait
end_time=$(date +%s.%N)
concurrent_time=$(echo "($end_time - $start_time) * 1000" | bc)
echo -e "${GREEN}✅ Completed 10 concurrent requests in ${concurrent_time}ms${NC}"
echo ""

# Summary
echo "=== Performance Test Summary ==="
echo -e "Health endpoint:     ${health_time}ms (target: <${PERFORMANCE_THRESHOLD_HEALTH}ms)"
echo -e "Data parsing:        ${parse_time}ms (target: <${PERFORMANCE_THRESHOLD_PARSE}ms)"
echo -e "Data transformation: ${transform_time}ms (target: <${PERFORMANCE_THRESHOLD_TRANSFORM}ms)"
echo -e "Data validation:     ${validate_time}ms (target: <${PERFORMANCE_THRESHOLD_VALIDATE}ms)"
echo -e "SQL query:           ${query_time}ms (target: <${PERFORMANCE_THRESHOLD_QUERY}ms)"
echo -e "Concurrent requests: ${concurrent_time}ms for 10 parallel requests"
echo ""

# Overall pass/fail
health_pass=$(echo "$health_time < $PERFORMANCE_THRESHOLD_HEALTH" | bc)
parse_pass=$(echo "$parse_time < $PERFORMANCE_THRESHOLD_PARSE" | bc)
transform_pass=$(echo "$transform_time < $PERFORMANCE_THRESHOLD_TRANSFORM" | bc)
validate_pass=$(echo "$validate_time < $PERFORMANCE_THRESHOLD_VALIDATE" | bc)
query_pass=$(echo "$query_time < $PERFORMANCE_THRESHOLD_QUERY" | bc)

if [ "$health_pass" -eq 1 ] && [ "$parse_pass" -eq 1 ] && [ "$transform_pass" -eq 1 ] && [ "$validate_pass" -eq 1 ] && [ "$query_pass" -eq 1 ]; then
    echo -e "${GREEN}✅ All performance tests passed!${NC}"
    exit 0
else
    echo -e "${YELLOW}⚠️  Some performance tests exceeded thresholds${NC}"
    exit 1
fi
