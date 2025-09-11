#!/bin/bash
# Calendar Scenario Integration Tests
# Tests the calendar API endpoints and functionality

set -e

# Configuration
# API_PORT must be set via environment - no default provided
if [ -z "$API_PORT" ]; then
    echo "Error: API_PORT environment variable is required"
    echo "Set it with: export API_PORT=<port_number>"
    exit 1
fi
API_BASE_URL="${CALENDAR_API_URL:-http://localhost:${API_PORT}}"
AUTH_TOKEN="${CALENDAR_AUTH_TOKEN:-test-token}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Helper functions
test_passed() {
    echo -e "${GREEN}✓${NC} $1"
    ((TESTS_PASSED++))
}

test_failed() {
    echo -e "${RED}✗${NC} $1: $2"
    ((TESTS_FAILED++))
}

api_request() {
    local method="$1"
    local endpoint="$2"
    local data="$3"
    
    curl -s -X "$method" \
        -H "Authorization: Bearer $AUTH_TOKEN" \
        -H "Content-Type: application/json" \
        ${data:+-d "$data"} \
        "$API_BASE_URL$endpoint"
}

# Start tests
echo -e "${BLUE}🧪 Running Calendar Integration Tests${NC}"
echo "API URL: $API_BASE_URL"
echo ""

# Test 1: Health Check
echo "Testing: Health Check"
response=$(curl -s "$API_BASE_URL/health")
if echo "$response" | grep -q '"status"'; then
    test_passed "Health check endpoint"
else
    test_failed "Health check endpoint" "No status in response"
fi

# Test 2: Create Event
echo "Testing: Create Event"

# Generate dynamic dates (tomorrow at 10:00 and 11:00 UTC)
tomorrow=$(date -u -d "tomorrow" +%Y-%m-%d)
start_time="${tomorrow}T10:00:00Z"
end_time="${tomorrow}T11:00:00Z"

create_data='{
    "title": "Test Meeting",
    "description": "Integration test meeting",
    "start_time": "'$start_time'",
    "end_time": "'$end_time'",
    "location": "Conference Room A",
    "event_type": "meeting",
    "reminders": [
        {"minutes_before": 15, "type": "email"}
    ]
}'

response=$(api_request POST "/api/v1/events" "$create_data")
if echo "$response" | grep -q '"success":true'; then
    event_id=$(echo "$response" | jq -r '.event.id' 2>/dev/null || echo "")
    test_passed "Create event endpoint"
else
    test_failed "Create event endpoint" "Failed to create event"
fi

# Test 3: List Events
echo "Testing: List Events"
response=$(api_request GET "/api/v1/events?days=7")
if echo "$response" | grep -q '"events"'; then
    test_passed "List events endpoint"
else
    test_failed "List events endpoint" "No events in response"
fi

# Test 4: Get Single Event
if [ -n "$event_id" ]; then
    echo "Testing: Get Single Event"
    response=$(api_request GET "/api/v1/events/$event_id")
    if echo "$response" | grep -q '"id"'; then
        test_passed "Get single event endpoint"
    else
        test_failed "Get single event endpoint" "Event not found"
    fi
fi

# Test 5: Update Event
if [ -n "$event_id" ]; then
    echo "Testing: Update Event"
    update_data='{
        "title": "Updated Test Meeting",
        "location": "Conference Room B"
    }'
    response=$(api_request PUT "/api/v1/events/$event_id" "$update_data")
    if echo "$response" | grep -q '"title":"Updated Test Meeting"'; then
        test_passed "Update event endpoint"
    else
        test_failed "Update event endpoint" "Failed to update event"
    fi
fi

# Test 6: Schedule Chat (NLP)
echo "Testing: Schedule Chat"
chat_data='{
    "message": "Schedule a meeting with John tomorrow at 3pm"
}'
response=$(api_request POST "/api/v1/schedule/chat" "$chat_data")
if echo "$response" | grep -q '"response"'; then
    test_passed "Schedule chat endpoint"
else
    test_failed "Schedule chat endpoint" "No response from chat"
fi

# Test 7: Schedule Optimization
echo "Testing: Schedule Optimization"
optimize_data='{
    "optimization_goal": "minimize_gaps",
    "start_date": "2024-12-20T00:00:00Z",
    "end_date": "2024-12-21T00:00:00Z",
    "constraints": {
        "business_hours_only": true
    }
}'
response=$(api_request POST "/api/v1/schedule/optimize" "$optimize_data")
if echo "$response" | grep -q '"optimization_goal"'; then
    test_passed "Schedule optimize endpoint"
else
    test_failed "Schedule optimize endpoint" "No optimization response"
fi

# Test 8: Delete Event
if [ -n "$event_id" ]; then
    echo "Testing: Delete Event"
    response=$(api_request DELETE "/api/v1/events/$event_id")
    if echo "$response" | grep -q '"success":true'; then
        test_passed "Delete event endpoint"
    else
        test_failed "Delete event endpoint" "Failed to delete event"
    fi
fi

# Test 9: Process Reminders
echo "Testing: Process Reminders"
response=$(api_request POST "/api/v1/reminders/process")
if echo "$response" | grep -q '"status":"success"'; then
    test_passed "Process reminders endpoint"
else
    test_failed "Process reminders endpoint" "Failed to process reminders"
fi

# Summary
echo ""
echo -e "${BLUE}📊 Test Summary${NC}"
echo -e "Passed: ${GREEN}$TESTS_PASSED${NC}"
echo -e "Failed: ${RED}$TESTS_FAILED${NC}"

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ Some tests failed${NC}"
    exit 1
fi