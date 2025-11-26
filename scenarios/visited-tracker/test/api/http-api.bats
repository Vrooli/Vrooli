#!/usr/bin/env bats
# HTTP API Integration tests
# Tests [REQ:VT-REQ-006] [REQ:VT-REQ-009]

setup() {
    # Get API port from lifecycle system or service.json
    # Note: Port is allocated via .vrooli/service.json (17694)
    if [ -z "${API_PORT:-}" ]; then
        export API_PORT="17694"
    fi

    export TEST_CAMPAIGN_ID=""
}

teardown() {
    if [ -n "$TEST_CAMPAIGN_ID" ]; then
        curl -s -X DELETE "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}" >/dev/null 2>&1 || true
    fi
}

# [REQ:VT-REQ-006] HTTP API - Health endpoint
@test "[HTTP] Health endpoint responds" {
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/health")
    [ "$status_code" = "200" ]
}

# [REQ:VT-REQ-006] HTTP API - CORS headers present
@test "[HTTP] CORS headers configured" {
    local response
    response=$(curl -s -i "http://localhost:${API_PORT}/api/v1/campaigns" -H "Origin: http://localhost:38440")

    # Should have CORS headers (exact header depends on implementation)
    echo "$response" | grep -i "access-control" >/dev/null
}

# [REQ:VT-REQ-006] HTTP API - Content-Type JSON
@test "[HTTP] API returns JSON content type" {
    # Create campaign to test
    local response
    response=$(curl -s -i -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-content-type","patterns":["*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$response" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)

    # Verify JSON content type
    echo "$response" | grep -i "content-type.*application/json" >/dev/null
}

# [REQ:VT-REQ-009] Prioritization - Least visited endpoint exists
@test "[HTTP] Least visited endpoint accessible" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-priority-endpoint","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Access prioritization endpoint
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/least-visited?limit=5")
    [ "$status_code" = "200" ]
}

# [REQ:VT-REQ-009] Prioritization - Most stale endpoint exists
@test "[HTTP] Most stale endpoint accessible" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-stale-endpoint","patterns":["**/*.ts"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Access staleness endpoint
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/most-stale?limit=5")
    [ "$status_code" = "200" ]
}

# [REQ:VT-REQ-009] Prioritization - Limit parameter works
@test "[HTTP] Prioritization respects limit parameter" {
    # Create campaign
    local create_response
    create_response=$(curl -s -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"name":"test-limit","patterns":["**/*.go"]}')

    TEST_CAMPAIGN_ID=$(echo "$create_response" | jq -r '.id')

    # Query with limit=3
    local response
    response=$(curl -s "http://localhost:${API_PORT}/api/v1/campaigns/${TEST_CAMPAIGN_ID}/prioritize/least-visited?limit=3")

    # Verify limit respected (files array length ≤ 3)
    local file_count
    file_count=$(echo "$response" | jq '.files | length')
    [ "$file_count" -le 3 ]
}

# [REQ:VT-REQ-006] HTTP API - Error handling for invalid requests
@test "[HTTP] API returns 400 for invalid campaign creation" {
    # Missing required field (name)
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "http://localhost:${API_PORT}/api/v1/campaigns" \
        -H "Content-Type: application/json" \
        -d '{"patterns":["*.go"]}')

    [ "$status_code" = "400" ]
}

# [REQ:VT-REQ-006] HTTP API - 404 for non-existent campaign
@test "[HTTP] API returns 404 for non-existent campaign" {
    local fake_id="00000000-0000-0000-0000-000000000000"
    local status_code
    status_code=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:${API_PORT}/api/v1/campaigns/${fake_id}")

    [ "$status_code" = "404" ]
}
