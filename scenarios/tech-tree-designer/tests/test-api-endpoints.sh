#!/bin/bash

# Tech Tree Designer API Endpoint Tests
# Validates all strategic intelligence API endpoints

set -e

# Configuration
API_PORT=${API_PORT:-8080}
API_BASE="http://localhost:${API_PORT}/api/v1"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Test results
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
log_test() {
    echo -e "${BLUE}🧪 Testing: $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
    ((TESTS_PASSED++))
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
    ((TESTS_FAILED++))
}

check_api_running() {
    if ! curl -sf "$API_BASE/../health" > /dev/null 2>&1; then
        echo -e "${RED}❌ API is not running on port $API_PORT${NC}"
        echo -e "${YELLOW}Start with: vrooli scenario run tech-tree-designer${NC}"
        exit 1
    fi
}

test_endpoint() {
    local method="$1"
    local endpoint="$2"
    local expected_status="$3"
    local description="$4"
    local data="$5"
    
    log_test "$description"
    
    if [[ "$method" == "GET" ]]; then
        response=$(curl -s -w "%{http_code}" "$API_BASE$endpoint")
    else
        if [[ -n "$data" ]]; then
            response=$(curl -s -w "%{http_code}" -X "$method" -H "Content-Type: application/json" -d "$data" "$API_BASE$endpoint")
        else
            response=$(curl -s -w "%{http_code}" -X "$method" "$API_BASE$endpoint")
        fi
    fi
    
    # Extract status code (last 3 characters)
    status_code="${response: -3}"
    response_body="${response%???}"
    
    if [[ "$status_code" == "$expected_status" ]]; then
        log_success "Status $status_code - $description"
        
        # Additional validation for JSON responses
        if [[ "$status_code" == "200" && -n "$response_body" ]]; then
            if echo "$response_body" | jq . > /dev/null 2>&1; then
                log_success "Valid JSON response"
            else
                log_error "Invalid JSON response format"
            fi
        fi
    else
        log_error "Expected $expected_status, got $status_code - $description"
        if [[ -n "$response_body" ]]; then
            echo "Response: $response_body"
        fi
    fi
}

echo -e "${BLUE}🚀 Tech Tree Designer API Tests${NC}"
echo -e "${BLUE}===================================${NC}"

# Check if API is running
log_test "API availability"
check_api_running
log_success "API is running on port $API_PORT"

echo -e "\n${YELLOW}📊 Core API Endpoints${NC}"

# Health check
test_endpoint "GET" "/health" "200" "Health check endpoint"

# Tech tree endpoints  
test_endpoint "GET" "/tech-tree" "200" "Get main tech tree"
test_endpoint "GET" "/tech-tree/sectors" "200" "Get all sectors"

# Progress endpoints
test_endpoint "GET" "/progress/scenarios" "200" "Get scenario mappings"

# Strategic analysis endpoints
test_endpoint "GET" "/milestones" "200" "Get strategic milestones"
test_endpoint "GET" "/recommendations" "200" "Get basic recommendations"
test_endpoint "GET" "/dependencies" "200" "Get stage dependencies"
test_endpoint "GET" "/connections" "200" "Get cross-sector connections"

echo -e "\n${YELLOW}🧠 Strategic Analysis Endpoints${NC}"

# Test strategic analysis with sample data
test_endpoint "POST" "/tech-tree/analyze" "200" "Strategic path analysis" '{
    "current_resources": 5,
    "time_horizon": 12,
    "priority_sectors": ["software", "manufacturing"]
}'

# Test scenario status update
test_endpoint "PUT" "/progress/scenarios/test-scenario" "200" "Update scenario status" '{
    "completion_status": "completed",
    "notes": "API test completion"
}'

# Test adding scenario mapping
test_endpoint "POST" "/progress/scenarios" "200" "Add scenario mapping" '{
    "scenario_name": "api-test-scenario",
    "stage_id": "550e8400-e29b-41d4-a716-446655441001",
    "completion_status": "in_progress",
    "contribution_weight": 0.8,
    "priority": 2,
    "estimated_impact": 7.5
}'

echo -e "\n${YELLOW}📈 Performance Tests${NC}"

# Test response times
log_test "Response time for sectors endpoint"
start_time=$(date +%s%N)
curl -sf "$API_BASE/tech-tree/sectors" > /dev/null
end_time=$(date +%s%N)
response_time=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

if [[ $response_time -lt 2000 ]]; then
    log_success "Response time: ${response_time}ms (under 2s threshold)"
else
    log_error "Response time: ${response_time}ms (exceeds 2s threshold)"
fi

echo -e "\n${YELLOW}🔍 Data Validation Tests${NC}"

# Test that sectors endpoint returns valid data structure
log_test "Sectors data structure validation"
sectors_response=$(curl -s "$API_BASE/tech-tree/sectors")
if echo "$sectors_response" | jq '.sectors[] | select(.name and .progress_percentage != null)' > /dev/null 2>&1; then
    log_success "Sectors contain required fields (name, progress_percentage)"
else
    log_error "Sectors missing required fields"
fi

# Test milestones data structure  
log_test "Milestones data structure validation"
milestones_response=$(curl -s "$API_BASE/milestones")
if echo "$milestones_response" | jq '.milestones[] | select(.name and .completion_percentage != null)' > /dev/null 2>&1; then
    log_success "Milestones contain required fields"
else
    log_error "Milestones missing required fields or no data found"
fi

echo -e "\n${BLUE}📊 Test Results Summary${NC}"
echo -e "${BLUE}======================${NC}"
echo -e "Tests Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Tests Failed: ${RED}$TESTS_FAILED${NC}"

if [[ $TESTS_FAILED -eq 0 ]]; then
    echo -e "\n${GREEN}🎉 All API tests passed! Strategic Intelligence System is operational.${NC}"
    exit 0
else
    echo -e "\n${RED}⚠️  Some tests failed. Check the API implementation.${NC}"
    exit 1
fi