#!/usr/bin/env bats
# API Integration tests for campaign lifecycle
# Tests [REQ:VT-REQ-001] [REQ:VT-REQ-002] [REQ:VT-REQ-003]

setup() {
    # Get API port from lifecycle system or service.json
    # Note: Port is allocated via .vrooli/service.json (17694)
    if [ -z "${API_PORT:-}" ]; then
        export API_PORT="17694"
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
    response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-integration-campaign","from_agent":"integration-test","patterns":["**/*.go","**/*.ts"],"description":"Integration test campaign"}')

    # Extract HTTP status code and body
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')

    # Debug: print if not 200
    if [ "$http_code" != "200" ]; then
        echo "HTTP Status: $http_code" >&2
        echo "Response Body: $body" >&2
    fi

    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    # Verify campaign ID returned
    TEST_CAMPAIGN_ID=$(echo "$body" | jq -r '.id // empty')
    [ -n "$TEST_CAMPAIGN_ID" ]

    # Verify campaign name
    local name
    name=$(echo "$body" | jq -r '.name')
    [ "$name" = "test-integration-campaign" ]

    # Verify patterns stored correctly
    local pattern_count
    pattern_count=$(echo "$body" | jq -r '.patterns | length')
    [ "$pattern_count" -eq 2 ]
}

# [REQ:VT-REQ-001] Campaign retrieval
@test "[API] Retrieve campaign by ID" {
    # Create campaign first
    local create_response
    create_response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-retrieve","patterns":["*.go"]}')

    local http_code=$(echo "$create_response" | tail -n1)
    local body=$(echo "$create_response" | sed '$d')
    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    TEST_CAMPAIGN_ID=$(echo "$body" | jq -r '.id')

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
    create_response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-list","patterns":["*.js"]}')

    local http_code=$(echo "$create_response" | tail -n1)
    local body=$(echo "$create_response" | sed '$d')
    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    TEST_CAMPAIGN_ID=$(echo "$body" | jq -r '.id')

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
    create_response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-visits","patterns":["**/*.go"]}')

    local http_code=$(echo "$create_response" | tail -n1)
    local body=$(echo "$create_response" | sed '$d')
    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    TEST_CAMPAIGN_ID=$(echo "$body" | jq -r '.id')

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
    create_response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-prioritize","patterns":["**/*.go"]}')

    local http_code=$(echo "$create_response" | tail -n1)
    local body=$(echo "$create_response" | sed '$d')
    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    TEST_CAMPAIGN_ID=$(echo "$body" | jq -r '.id')

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
    create_response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-stale","patterns":["**/*.go"]}')

    local http_code=$(echo "$create_response" | tail -n1)
    local body=$(echo "$create_response" | sed '$d')
    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    TEST_CAMPAIGN_ID=$(echo "$body" | jq -r '.id')

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
    create_response=$(curl -s -w "\n%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-delete","patterns":["*.ts"]}')

    local http_code=$(echo "$create_response" | tail -n1)
    local body=$(echo "$create_response" | sed '$d')
    [ "$http_code" = "200" ] || [ "$http_code" = "201" ]

    local campaign_id
    campaign_id=$(echo "$body" | jq -r '.id')

    # Delete campaign
    local delete_status
    delete_status=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "http://localhost:${API_PORT}/api/v1/campaigns/${campaign_id}")
    [ "$delete_status" = "200" ]

    # Verify campaign is gone
    local get_status
    get_status=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/v1/campaigns/${campaign_id}")
    [ "$get_status" = "404" ]
}
