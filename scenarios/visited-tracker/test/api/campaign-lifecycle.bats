#!/usr/bin/env bats
# API Integration tests for campaign lifecycle
# Tests [REQ:VT-REQ-001] [REQ:VT-REQ-002] [REQ:VT-REQ-003]

setup() {
    # Get API port from lifecycle system
    if [ -z "${API_PORT:-}" ]; then
        export API_PORT=$(vrooli scenario port visited-tracker API_PORT 2>/dev/null || echo "17693")
    fi

    # Track campaign ID for cleanup
    export TEST_CAMPAIGN_ID=""
}

teardown() {
    # Clean up test campaign if created
    if [ -n "$TEST_CAMPAIGN_ID" ]; then
        curl -s -X DELETE "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}" >/dev/null 2>&1 || true
    fi
}

# [REQ:VT-REQ-001] Campaign creation with patterns
@test "[API] Create campaign with glob patterns" {
    local response
    response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-integration-campaign","from_agent":"integration-test","patterns":["**/*.go","**/*.ts"],"description":"Integration test campaign"}')

    # Verify campaign ID returned
    TEST_CAMPAIGN_ID=$(echo "$response" | jq -r '.id // empty')
    [ -n "$TEST_CAMPAIGN_ID" ]

    # Verify campaign name
    local name
    name=$(echo "$response" | jq -r '.name')
    [ "$name" = "test-integration-campaign" ]

    # Verify patterns stored correctly
    local pattern_count
    pattern_count=$(echo "$response" | jq -r '.patterns | length')
    [ "$pattern_count" -eq 2 ]
}

# [REQ:VT-REQ-001] Campaign retrieval
@test "[API] Retrieve campaign by ID" {
    # Create campaign first
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-retrieve","patterns":["*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Retrieve campaign
    local get_response
    get_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}")

    # Verify retrieval
    local retrieved_name
    retrieved_name=$(echo "$get_response" | jq -r '.name')
    [ "$retrieved_name" = "test-retrieve" ]
}

# [REQ:VT-REQ-001] List all campaigns
@test "[API] List campaigns returns array" {
    # Create test campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-list","patterns":["*.js"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # List campaigns
    local list_response
    list_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns")

    # Verify array structure
    local campaign_count
    campaign_count=$(echo "$list_response" | jq '.campaigns | length')
    [ "$campaign_count" -ge 1 ]

    # Verify our campaign is in list
    local found
    found=$(echo "$list_response" | jq -r --arg id "$TEST_CAMPAIGN_ID" '.campaigns[] | select(.id == $id) | .name')
    [ "$found" = "test-list" ]
}

# [REQ:VT-REQ-002] Record file visit
@test "[API] Record visit increases visit count" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-visits","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Record first visit
    curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/visit" \
        -H "Content-Type: application/json" \
        -d '{"files":["api/main.go"],"context":"first-visit","agent":"integration-test"}' >/dev/null

    # Get campaign and verify visit recorded
    local get_response
    get_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}")

    local visit_count
    visit_count=$(echo "$get_response" | jq '.visits | length')
    [ "$visit_count" -ge 1 ]
}

# [REQ:VT-REQ-003] Staleness prioritization
@test "[API] Query least visited files" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-prioritize","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Record some visits to create differentiation
    curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/visit" \
        -H "Content-Type: application/json" \
        -d '{"files":["api/main.go"],"context":"test","agent":"integration"}' >/dev/null

    # Query least visited
    local priority_response
    priority_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/least-visited?limit=10")

    # Verify response structure
    local has_files
    has_files=$(echo "$priority_response" | jq 'has("files")')
    [ "$has_files" = "true" ]
}

# [REQ:VT-REQ-003] Most stale files prioritization
@test "[API] Query most stale files" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-stale","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Query most stale
    local stale_response
    stale_response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/most-stale?limit=10")

    # Verify response structure
    local has_files
    has_files=$(echo "$stale_response" | jq 'has("files")')
    [ "$has_files" = "true" ]
}

# [REQ:VT-REQ-001] Campaign deletion
@test "[API] Delete campaign removes it permanently" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-delete","patterns":["*.ts"]}')

    local campaign_id
    campaign_id=$(echo "$create_response" | jq -r '.id')

    # Delete campaign
    local delete_status
    delete_status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "http://localhost:${API_PORT}/api/v1/campaigns/${campaign_id}")
    [ "$delete_status" = "200" ]

    # Verify campaign is gone
    local get_status
    get_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/v1/campaigns/${campaign_id}")
    [ "$get_status" = "404" ]
}
