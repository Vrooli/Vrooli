#!/usr/bin/env bats
# CLI tests for visited-tracker campaign management
# Tests [REQ:VT-REQ-004] CLI Interface

setup() {
    # Get scenario directory
    SCENARIO_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")/../.." && pwd)"

    # Get API port from lifecycle system (environment variable is set by vrooli scenario start)
    # If not set, fall back to querying the vrooli CLI
    if [ -z "${API_PORT:-}" ]; then
        API_PORT=$(vrooli scenario port visited-tracker API_PORT 2>/dev/null || echo "17693")
    fi
    export API_PORT

    # Create temp directory for test data
    TEST_TEMP_DIR=$(mktemp -d)
    export TEST_TEMP_DIR
}

teardown() {
    # Clean up temp directory
    if [ -n "$TEST_TEMP_DIR" ] && [ -d "$TEST_TEMP_DIR" ]; then
        rm -rf "$TEST_TEMP_DIR"
    fi

    # Clean up any test campaigns created
    if [ -n "${TEST_CAMPAIGN_ID:-}" ]; then
        curl -s -X DELETE "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}" >/dev/null 2>&1 || true
    fi
}

# [REQ:VT-REQ-004] Test campaign creation via CLI/API
@test "create campaign via API" {
    local response
    response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-cli-campaign","from_agent":"bats-test","patterns":["**/*.go","**/*.ts"],"description":"Test campaign for CLI validation"}')

    # Verify response contains campaign ID
    TEST_CAMPAIGN_ID=$(echo "$response" | jq -r '.id // empty')
    [ -n "$TEST_CAMPAIGN_ID" ]

    # Verify campaign was created
    local get_response
    get_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}")
    local campaign_name
    campaign_name=$(echo "$get_response" | jq -r '.name // empty')
    [ "$campaign_name" = "test-cli-campaign" ]
}

# [REQ:VT-REQ-004] Test listing campaigns
@test "list campaigns via API" {
    # Create a test campaign first
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-list-campaign","from_agent":"bats-test","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id // empty')

    # List campaigns
    local list_response
    list_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns")

    # Verify response has campaigns array
    local campaign_count
    campaign_count=$(echo "$list_response" | jq '.campaigns | length')
    [ "$campaign_count" -ge 1 ]

    # Verify our campaign is in the list
    local found
    found=$(echo "$list_response" | jq -r --arg id "$TEST_CAMPAIGN_ID" '.campaigns[] | select(.id == $id) | .name')
    [ "$found" = "test-list-campaign" ]
}

# [REQ:VT-REQ-004] [REQ:VT-REQ-002] Test visit tracking
@test "record file visits via API" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-visit-campaign","from_agent":"bats-test","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id // empty')

    # Record visits
    local visit_response
    visit_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/visit" \
        -H "Content-Type: application/json" \
        -d '{"files":["api/main.go","api/types.go"],"context":"cli-test-visit","agent":"bats-test"}')

    # Verify visit was recorded (response should be 200)
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/visit" \
        -H "Content-Type: application/json" \
        -d '{"files":["api/main.go"],"context":"test","agent":"bats"}')

    [ "$status_code" = "200" ]
}

# [REQ:VT-REQ-004] [REQ:VT-REQ-009] Test prioritization queries
@test "query least visited files via API" {
    # Create campaign and record some visits
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-priority-campaign","from_agent":"bats-test","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id // empty')

    # Record visits to create differentiation
    curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/visit" \
        -H "Content-Type: application/json" \
        -d '{"files":["api/main.go"],"context":"test","agent":"bats"}' >/dev/null

    # Query least visited
    local priority_response
    priority_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/least-visited?limit=10")

    # Verify response structure
    local has_files
    has_files=$(echo "$priority_response" | jq 'has("files")')
    [ "$has_files" = "true" ]
}

# [REQ:VT-REQ-004] [REQ:VT-REQ-003] Test staleness prioritization
@test "query most stale files via API" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-stale-campaign","from_agent":"bats-test","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id // empty')

    # Query most stale
    local stale_response
    stale_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/most-stale?limit=10")

    # Verify response structure
    local has_files
    has_files=$(echo "$stale_response" | jq 'has("files")')
    [ "$has_files" = "true" ]
}

# [REQ:VT-REQ-004] Test campaign deletion
@test "delete campaign via API" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-delete-campaign","from_agent":"bats-test","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id // empty')

    # Delete campaign
    local delete_status
    delete_status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}")
    [ "$delete_status" = "200" ]

    # Verify campaign is gone
    local get_status
    get_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}")
    [ "$get_status" = "404" ]

    # Clear TEST_CAMPAIGN_ID to prevent double deletion in teardown
    unset TEST_CAMPAIGN_ID
}
