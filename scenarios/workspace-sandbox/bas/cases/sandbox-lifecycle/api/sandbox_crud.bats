#!/usr/bin/env bats
# [REQ:REQ-P0-001] Fast Sandbox Creation
# [REQ:REQ-P0-002] Stable Sandbox Identifier
# [REQ:REQ-P0-008] Sandbox Lifecycle API
# Phase: e2e
# Validates sandbox CRUD operations via API

setup() {
    API_PORT="${API_PORT:-$(vrooli scenario port workspace-sandbox API_PORT 2>/dev/null || echo 15427)}"
    API_BASE="http://127.0.0.1:${API_PORT}/api/v1"
    TEST_SCOPE="/tmp/workspace-sandbox-test-$$"
    mkdir -p "$TEST_SCOPE"
}

teardown() {
    # Clean up test sandbox if created
    if [ -n "$SANDBOX_ID" ]; then
        curl -s -X DELETE "${API_BASE}/sandboxes/${SANDBOX_ID}" >/dev/null 2>&1 || true
    fi
    rm -rf "$TEST_SCOPE" 2>/dev/null || true
}

# [REQ:REQ-P0-001] System creates sandbox within 2 seconds
@test "sandbox creation completes within 2 seconds" {
    skip "Requires overlayfs root permissions"
    start_time=$(date +%s%N)
    response=$(curl -s -X POST "${API_BASE}/sandboxes" \
        -H "Content-Type: application/json" \
        -d "{\"scopePath\": \"${TEST_SCOPE}\", \"projectRoot\": \"/tmp\"}")
    end_time=$(date +%s%N)
    duration=$(( (end_time - start_time) / 1000000000 ))
    [ "$duration" -le 2 ]
    SANDBOX_ID=$(echo "$response" | jq -r '.id // empty')
}

# [REQ:REQ-P0-002] Sandbox has stable UUID identifier
@test "sandbox returns valid UUID on creation" {
    skip "Requires overlayfs root permissions"
    response=$(curl -s -X POST "${API_BASE}/sandboxes" \
        -H "Content-Type: application/json" \
        -d "{\"scopePath\": \"${TEST_SCOPE}\", \"projectRoot\": \"/tmp\"}")
    SANDBOX_ID=$(echo "$response" | jq -r '.id // empty')
    # Validate UUID format
    [[ "$SANDBOX_ID" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]]
}

# [REQ:REQ-P0-008] List sandboxes returns valid response
@test "list sandboxes returns valid response" {
    response=$(curl -s "${API_BASE}/sandboxes")
    # sandboxes can be null or array, totalCount should be >= 0
    echo "$response" | jq -e 'has("sandboxes")' >/dev/null
    echo "$response" | jq -e '.totalCount >= 0' >/dev/null
}

# [REQ:REQ-P0-008] List sandboxes supports status filter
@test "list sandboxes supports status filter" {
    response=$(curl -s "${API_BASE}/sandboxes?status=active")
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/sandboxes?status=active")
    [ "$http_code" = "200" ]
}

# [REQ:REQ-P0-008] Get sandbox returns 404 for missing ID
@test "get sandbox returns 404 for unknown ID" {
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000")
    [ "$http_code" = "404" ]
}

# [REQ:REQ-P0-008] Get sandbox returns error hint for 404
@test "get sandbox 404 response includes hint" {
    response=$(curl -s "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000")
    echo "$response" | jq -e '.hint' >/dev/null
}

# [REQ:REQ-P0-008] Delete sandbox is idempotent
@test "delete sandbox returns success for unknown ID" {
    # Should not error - idempotent delete
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "${API_BASE}/sandboxes/00000000-0000-0000-0000-000000000000")
    # Should return 204 No Content or 404 Not Found (both acceptable for idempotent delete)
    [ "$http_code" = "204" ] || [ "$http_code" = "404" ]
}

# [REQ:REQ-P0-001] Create sandbox validates scope path
@test "create sandbox requires scope path" {
    response=$(curl -s -X POST "${API_BASE}/sandboxes" \
        -H "Content-Type: application/json" \
        -d "{}")
    http_code=$(curl -s -o /dev/null -w "%{http_code}" -X POST "${API_BASE}/sandboxes" \
        -H "Content-Type: application/json" \
        -d "{}")
    [ "$http_code" = "400" ]
}

# [REQ:REQ-P0-008] Stats endpoint returns aggregate data
@test "stats endpoint returns valid counts" {
    response=$(curl -s "${API_BASE}/stats")
    echo "$response" | jq -e '.stats.totalCount >= 0' >/dev/null
    echo "$response" | jq -e '.stats.activeCount >= 0' >/dev/null
}

# [REQ:REQ-P0-008] Driver info endpoint works
@test "driver info endpoint returns status" {
    response=$(curl -s "${API_BASE}/driver/info")
    http_code=$(curl -s -o /dev/null -w "%{http_code}" "${API_BASE}/driver/info")
    [ "$http_code" = "200" ]
}
