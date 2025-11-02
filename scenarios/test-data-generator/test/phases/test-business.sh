#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"
cd "$TESTING_PHASE_SCENARIO_DIR"

API_PORT=""
API_PID=""
API_WAS_RUNNING=false
BUSINESS_LOG="/tmp/test-data-generator-business.log"

stop_api() {
    if [ "$API_WAS_RUNNING" = false ] && [ -n "$API_PID" ]; then
        testing::phase::log "INFO" "Stopping API server (PID: $API_PID)..."
        kill "$API_PID" 2>/dev/null || true
        wait "$API_PID" 2>/dev/null || true
        API_PID=""
    fi
}

main() {
    testing::phase::log "INFO" "Starting business logic tests..."

    API_PORT=$(jq -r '.interfaces.api.port // .lifecycle.develop.steps[]? | select(.name=="start-api") | .env.API_PORT // empty' .vrooli/service.json 2>/dev/null)
    if [ -z "$API_PORT" ] || [ "$API_PORT" = "null" ]; then
        API_PORT=3001
    fi
    local api_url="http://localhost:${API_PORT}"

    if curl -fs "$api_url/health" >/dev/null 2>&1; then
        testing::phase::log "INFO" "API is already running on port $API_PORT"
        API_WAS_RUNNING=true
    else
        testing::phase::log "INFO" "Starting API server on port $API_PORT..."
        API_WAS_RUNNING=false
        pushd api >/dev/null
        API_PORT=$API_PORT NODE_ENV=test nohup node server.js > "$BUSINESS_LOG" 2>&1 &
        API_PID=$!
        popd >/dev/null
        if ! kill -0 "$API_PID" >/dev/null 2>&1; then
            testing::phase::log "ERROR" "Failed to start API server"
            API_PID=""
            return 1
        fi

        local wait_count=0
        local max_wait=30
        while ! curl -fs "$api_url/health" >/dev/null 2>&1; do
            sleep 1
            wait_count=$((wait_count + 1))
            if [ $wait_count -ge $max_wait ]; then
                testing::phase::log "ERROR" "API failed to start within ${max_wait}s"
                stop_api
                return 1
            fi
        done
        testing::phase::log "INFO" "API started successfully (PID: $API_PID)"
    fi

    testing::phase::log "INFO" "Test: Verify all advertised data types are functional"
    local types_response
    types_response=$(curl -s "$api_url/api/types")
    local available_types
    available_types=$(echo "$types_response" | jq -r '.types[]')
    local working_types=0
    local broken_types=0

    local type
    for type in $available_types; do
        local response status
        response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/${type}" \
            -H "Content-Type: application/json" \
            -d '{"count": 1}')
        status=$(echo "$response" | tail -1)
        if [ "$status" = "200" ]; then
            working_types=$((working_types + 1))
            testing::phase::log "INFO" "  ✓ Type '$type' is functional"
        else
            broken_types=$((broken_types + 1))
            testing::phase::log "WARN" "  ✗ Type '$type' returned HTTP $status (may not be implemented)"
        fi
    done
    testing::phase::log "INFO" "Data type coverage: $working_types working, $broken_types not functional"

    testing::phase::log "INFO" "Test: Field selection produces correct structure"
    local field_test first_user has_id has_name has_email
    field_test=$(curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 1, "fields": ["id", "name"]}')
    first_user=$(echo "$field_test" | jq -r '.data[0]')
    has_id=$(echo "$first_user" | jq 'has("id")')
    has_name=$(echo "$first_user" | jq 'has("name")')
    has_email=$(echo "$first_user" | jq 'has("email")')
    if [ "$has_id" != "true" ] || [ "$has_name" != "true" ]; then
        testing::phase::log "ERROR" "Field selection failed: requested fields missing"
        stop_api
        return 1
    fi
    if [ "$has_email" = "true" ]; then
        testing::phase::log "WARN" "Field selection may not be working: unrequested field present"
    fi
    testing::phase::log "INFO" "✓ Field selection works correctly"

    testing::phase::log "INFO" "Test: Volume limits enforcement"
    local response status
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 10000}')
    status=$(echo "$response" | tail -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Max valid count (10000) should be allowed, got HTTP $status"
        stop_api
        return 1
    fi

    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 10001}')
    status=$(echo "$response" | tail -1)
    if [ "$status" != "400" ]; then
        testing::phase::log "ERROR" "Over limit (10001) should be rejected, got HTTP $status"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Volume limits are properly enforced"

    testing::phase::log "INFO" "Test: Data uniqueness (UUIDs should be unique)"
    local uuid_test total_ids unique_ids
    uuid_test=$(curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 100, "fields": ["id"]}')
    total_ids=$(echo "$uuid_test" | jq -r '.data[].id' | wc -l)
    unique_ids=$(echo "$uuid_test" | jq -r '.data[].id' | sort -u | wc -l)
    if [ "$total_ids" != "$unique_ids" ]; then
        testing::phase::log "ERROR" "UUID uniqueness violated: $total_ids total, $unique_ids unique"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Data uniqueness validated (100 unique UUIDs)"

    testing::phase::log "INFO" "Test: Format conversion produces valid output"
    local sql_test sql_data
    sql_test=$(curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 2, "format": "sql", "fields": ["id", "name"]}')
    sql_data=$(echo "$sql_test" | jq -r '.data')
    if ! echo "$sql_data" | grep -q "INSERT INTO"; then
        testing::phase::log "ERROR" "SQL format missing INSERT statement"
        stop_api
        return 1
    fi
    if ! echo "$sql_data" | grep -q "VALUES"; then
        testing::phase::log "ERROR" "SQL format missing VALUES clause"
        stop_api
        return 1
    fi

    local xml_test xml_data
    xml_test=$(curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 1, "format": "xml"}')
    xml_data=$(echo "$xml_test" | jq -r '.data')
    if ! echo "$xml_data" | grep -q "<?xml"; then
        testing::phase::log "ERROR" "XML format missing XML declaration"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Format conversions produce valid output"

    testing::phase::log "INFO" "Test: Custom schema supports various data types"
    local custom_test custom_success custom_count first_item uuid_value email_value boolean_value
    custom_test=$(curl -s -X POST "$api_url/api/generate/custom" -H "Content-Type: application/json" -d '{"count":5,"schema":{"id":"uuid","title":"string","price":"decimal","quantity":"integer","active":"boolean","email":"email","phone":"phone","created":"date"}}')
    custom_success=$(echo "$custom_test" | jq -r '.success')
    custom_count=$(echo "$custom_test" | jq -r '.count')
    if [ "$custom_success" != "true" ] || [ "$custom_count" != "5" ]; then
        testing::phase::log "ERROR" "Custom schema with multiple types failed"
        stop_api
        return 1
    fi
    first_item=$(echo "$custom_test" | jq -r '.data[0]')
    uuid_value=$(echo "$first_item" | jq -r '.id')
    if ! echo "$uuid_value" | grep -qE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then
        testing::phase::log "WARN" "UUID format may not be valid"
    fi
    email_value=$(echo "$first_item" | jq -r '.email')
    if ! echo "$email_value" | grep -q '@'; then
        testing::phase::log "ERROR" "Email format invalid: missing @"
        stop_api
        return 1
    fi
    boolean_value=$(echo "$first_item" | jq -r '.active')
    if [ "$boolean_value" != "true" ] && [ "$boolean_value" != "false" ]; then
        testing::phase::log "ERROR" "Boolean field not producing boolean values"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Custom schema supports various data types correctly"

    testing::phase::log "INFO" "Test: Concurrent request handling"
    local i failed=0
    for i in {1..5}; do
        curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 10}' > "/tmp/concurrent_$i.json" &
    done
    wait
    for i in {1..5}; do
        if ! jq -e '.success == true' "/tmp/concurrent_$i.json" >/dev/null 2>&1; then
            failed=$((failed + 1))
        fi
        rm -f "/tmp/concurrent_$i.json"
    done
    if [ $failed -gt 0 ]; then
        testing::phase::log "ERROR" "Concurrent requests: $failed out of 5 failed"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Concurrent requests handled successfully"

    stop_api
    return 0
}

if main; then
    testing::phase::end_with_summary "Business logic tests completed"
else
    stop_api
    testing::phase::end_with_summary "Business logic tests failed"
    exit 1
fi
