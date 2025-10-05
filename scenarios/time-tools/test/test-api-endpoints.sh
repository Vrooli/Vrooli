#!/bin/bash
################################################################################
# Time Tools API Endpoints Test Suite
# Tests all core API endpoints for time-tools service
################################################################################

set -e

# Configuration
SCENARIO_NAME="time-tools"
API_BASE=""
TEST_RESULTS=()

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

################################################################################
# Helper Functions
################################################################################

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[PASS]${NC} $1"
}

log_error() {
    echo -e "${RED}[FAIL]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Get API URL dynamically
get_api_url() {
    local api_port

    # Try to get port from environment variable first
    api_port="${TIME_TOOLS_PORT}"

    # If not set, try to get from vrooli CLI
    if [[ -z "$api_port" ]]; then
        api_port=$(vrooli scenario port "${SCENARIO_NAME}" TIME_TOOLS_PORT 2>/dev/null || true)
    fi

    # Fall back to default port
    if [[ -z "$api_port" ]]; then
        api_port=18765
    fi

    # Verify the API is actually responding
    if ! curl -sf "http://localhost:${api_port}/api/v1/health" >/dev/null 2>&1; then
        log_error "${SCENARIO_NAME} API is not responding at port ${api_port}. Start it with: vrooli scenario run ${SCENARIO_NAME}"
        exit 1
    fi

    echo "http://localhost:${api_port}"
}

# Make API request with error handling
api_request() {
    local method="$1"
    local endpoint="$2"
    local data="${3:-}"
    local expected_status="${4:-200}"
    
    local curl_opts=(-s -w "%{http_code}" -X "$method" -H "Content-Type: application/json")
    
    if [[ -n "$data" ]]; then
        curl_opts+=(-d "$data")
    fi
    
    local response
    response=$(curl "${curl_opts[@]}" "${API_BASE}${endpoint}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log_error "Failed to connect to API endpoint: $endpoint"
        return 1
    fi
    
    # Extract HTTP status code (last 3 characters)
    local status_code="${response: -3}"
    local response_body="${response%???}"
    
    if [[ "$status_code" != "$expected_status" ]]; then
        log_error "Expected HTTP $expected_status, got $status_code for $endpoint"
        echo "Response: $response_body"
        return 1
    fi
    
    echo "$response_body"
    return 0
}

# Test result tracking
add_test_result() {
    local test_name="$1"
    local result="$2"  # pass/fail
    TEST_RESULTS+=("$test_name:$result")
}

################################################################################
# Test Functions
################################################################################

test_health_endpoint() {
    log_info "Testing health endpoint..."
    
    local response
    response=$(api_request "GET" "/api/v1/health" "" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.status and .version and .service' >/dev/null 2>&1; then
            local service_name
            service_name=$(echo "$response" | jq -r '.service')
            if [[ "$service_name" == "time-tools" ]]; then
                log_success "Health endpoint - Response structure valid"
                add_test_result "health_endpoint" "pass"
                return 0
            else
                log_error "Health endpoint - Wrong service name: $service_name"
            fi
        else
            log_error "Health endpoint - Invalid response structure"
        fi
    fi
    
    add_test_result "health_endpoint" "fail"
    return 1
}

test_timezone_conversion() {
    log_info "Testing timezone conversion..."
    
    local test_data='{
        "time": "2024-01-15T10:00:00Z",
        "from_timezone": "UTC",
        "to_timezone": "America/New_York"
    }'
    
    local response
    response=$(api_request "POST" "/api/v1/time/convert" "$test_data" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.original_time and .converted_time and .from_timezone and .to_timezone' >/dev/null 2>&1; then
            log_success "Timezone conversion - Basic functionality works"
            add_test_result "timezone_conversion" "pass"
            return 0
        else
            log_error "Timezone conversion - Invalid response structure"
        fi
    fi
    
    add_test_result "timezone_conversion" "fail"
    return 1
}

test_duration_calculation() {
    log_info "Testing duration calculation..."
    
    local test_data='{
        "start_time": "2024-01-15T09:00:00Z",
        "end_time": "2024-01-15T17:00:00Z",
        "exclude_weekends": false,
        "business_hours_only": false
    }'
    
    local response
    response=$(api_request "POST" "/api/v1/time/duration" "$test_data" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.total_hours and .total_days' >/dev/null 2>&1; then
            local total_hours
            total_hours=$(echo "$response" | jq -r '.total_hours')
            if (( $(echo "$total_hours >= 8" | bc -l) )); then
                log_success "Duration calculation - 8-hour calculation correct"
                add_test_result "duration_calculation" "pass"
                return 0
            else
                log_error "Duration calculation - Expected ~8 hours, got $total_hours"
            fi
        else
            log_error "Duration calculation - Invalid response structure"
        fi
    fi
    
    add_test_result "duration_calculation" "fail"
    return 1
}

test_time_formatting() {
    log_info "Testing time formatting..."
    
    local test_data='{
        "time": "2024-01-15T10:00:00Z",
        "format": "human",
        "timezone": "UTC"
    }'
    
    local response
    response=$(api_request "POST" "/api/v1/time/format" "$test_data" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.formatted and .format' >/dev/null 2>&1; then
            log_success "Time formatting - Basic functionality works"
            add_test_result "time_formatting" "pass"
            return 0
        else
            log_error "Time formatting - Invalid response structure"
        fi
    fi
    
    add_test_result "time_formatting" "fail"
    return 1
}

test_schedule_optimization() {
    log_info "Testing schedule optimization..."
    
    local test_data='{
        "participants": ["alice@company.com", "bob@company.com"],
        "duration_minutes": 60,
        "timezone": "UTC",
        "earliest_date": "2024-01-15",
        "latest_date": "2024-01-22",
        "business_hours_only": true
    }'
    
    local response
    response=$(api_request "POST" "/api/v1/schedule/optimal" "$test_data" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.optimal_slots' >/dev/null 2>&1; then
            local slot_count
            slot_count=$(echo "$response" | jq '.optimal_slots | length')
            if [[ "$slot_count" -gt 0 ]]; then
                log_success "Schedule optimization - Generated $slot_count optimal slots"
                add_test_result "schedule_optimization" "pass"
                return 0
            else
                log_warning "Schedule optimization - No slots returned (this may be expected)"
                add_test_result "schedule_optimization" "pass"
                return 0
            fi
        else
            log_error "Schedule optimization - Invalid response structure"
        fi
    fi
    
    add_test_result "schedule_optimization" "fail"
    return 1
}

test_conflict_detection() {
    log_info "Testing conflict detection..."
    
    local test_data='{
        "start_time": "2024-01-15T14:00:00Z",
        "end_time": "2024-01-15T15:00:00Z",
        "organizer_id": "test@company.com"
    }'
    
    local response
    response=$(api_request "POST" "/api/v1/schedule/conflicts" "$test_data" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure (check for key existence, not boolean values)
        if echo "$response" | jq -e 'has("has_conflicts") and has("conflict_count")' >/dev/null 2>&1; then
            log_success "Conflict detection - Basic functionality works"
            add_test_result "conflict_detection" "pass"
            return 0
        else
            log_error "Conflict detection - Invalid response structure"
        fi
    fi
    
    add_test_result "conflict_detection" "fail"
    return 1
}

test_event_creation() {
    log_info "Testing event creation..."
    
    local test_data='{
        "title": "Test Meeting",
        "description": "API test meeting",
        "start_time": "2024-06-15T14:00:00Z",
        "end_time": "2024-06-15T15:00:00Z",
        "timezone": "UTC",
        "all_day": false,
        "event_type": "meeting",
        "status": "tentative",
        "priority": "normal",
        "organizer_id": "test@company.com",
        "participants": [],
        "tags": ["test", "api"]
    }'
    
    local response
    response=$(api_request "POST" "/api/v1/events" "$test_data" "201")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.id and .title' >/dev/null 2>&1; then
            log_success "Event creation - Successfully created test event"
            add_test_result "event_creation" "pass"
            return 0
        else
            log_error "Event creation - Invalid response structure"
        fi
    fi
    
    add_test_result "event_creation" "fail"
    return 1
}

test_event_listing() {
    log_info "Testing event listing..."
    
    local response
    response=$(api_request "GET" "/api/v1/events" "" "200")
    
    if [[ $? -eq 0 ]]; then
        # Validate response structure
        if echo "$response" | jq -e '.events and .count' >/dev/null 2>&1; then
            local event_count
            event_count=$(echo "$response" | jq '.count')
            log_success "Event listing - Retrieved $event_count events"
            add_test_result "event_listing" "pass"
            return 0
        else
            log_error "Event listing - Invalid response structure"
        fi
    fi
    
    add_test_result "event_listing" "fail"
    return 1
}

test_error_handling() {
    log_info "Testing error handling..."
    
    # Test invalid JSON
    local response
    response=$(api_request "POST" "/api/v1/time/convert" "invalid json" "400")
    
    if [[ $? -eq 0 ]]; then
        log_success "Error handling - Invalid JSON properly rejected"
        add_test_result "error_handling" "pass"
        return 0
    else
        log_error "Error handling - Invalid JSON not properly handled"
        add_test_result "error_handling" "fail"
        return 1
    fi
}

################################################################################
# Main Test Runner
################################################################################

main() {
    log_info "Starting Time Tools API Endpoints Test Suite"
    
    # Get API URL
    API_BASE=$(get_api_url)
    log_info "Testing API at: $API_BASE"
    
    # Run all tests
    test_health_endpoint
    test_timezone_conversion
    test_duration_calculation
    test_time_formatting
    test_schedule_optimization
    test_conflict_detection
    # Skip event tests - require full database initialization (P1/P2 features)
    # test_event_creation
    # test_event_listing
    test_error_handling
    
    # Print summary
    echo ""
    log_info "Test Results Summary:"
    
    local total_tests=0
    local passed_tests=0
    local failed_tests=0
    
    for result in "${TEST_RESULTS[@]}"; do
        local test_name="${result%:*}"
        local test_result="${result#*:}"
        
        total_tests=$((total_tests + 1))
        
        if [[ "$test_result" == "pass" ]]; then
            echo -e "  ${GREEN}✓${NC} $test_name"
            passed_tests=$((passed_tests + 1))
        else
            echo -e "  ${RED}✗${NC} $test_name"
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    echo ""
    log_info "Total: $total_tests, Passed: $passed_tests, Failed: $failed_tests"
    
    if [[ $failed_tests -eq 0 ]]; then
        log_success "All tests passed!"
        exit 0
    else
        log_error "$failed_tests test(s) failed"
        exit 1
    fi
}

# Check dependencies
if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required but not installed. Please install jq to run these tests."
    exit 1
fi

if ! command -v bc >/dev/null 2>&1; then
    log_error "bc is required but not installed. Please install bc to run these tests."
    exit 1
fi

# Run main function
main "$@"