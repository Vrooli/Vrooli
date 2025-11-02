#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"
cd "$TESTING_PHASE_SCENARIO_DIR"

API_PORT=""
API_PID=""
API_WAS_RUNNING=false
LOG_FILE="/tmp/test-data-generator-integration.log"

stop_api() {
    if [ "$API_WAS_RUNNING" = false ] && [ -n "$API_PID" ]; then
        testing::phase::log "INFO" "Stopping API server (PID: $API_PID)..."
        kill "$API_PID" 2>/dev/null || true
        wait "$API_PID" 2>/dev/null || true
        API_PID=""
    fi
}

run_health_check() {
    local response
    response=$(curl -s -w "\n%{http_code}" "$1/health")
    local status
    status=$(echo "$response" | tail -1)
    local body
    body=$(echo "$response" | head -n -1)

    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Health check failed: HTTP $status"
        return 1
    fi

    if ! echo "$body" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "Health check returned unexpected status"
        return 1
    fi
    return 0
}

main() {
    testing::phase::log "INFO" "Starting integration tests..."

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
        API_PORT=$API_PORT NODE_ENV=test nohup node server.js > "$LOG_FILE" 2>&1 &
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

    testing::phase::log "INFO" "Running API integration tests..."

    if ! run_health_check "$api_url"; then
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Health check passed"

    local response status body

    testing::phase::log "INFO" "Test: Get available data types"
    response=$(curl -s -w "\n%{http_code}" "$api_url/api/types")
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Get types failed: HTTP $status"
        stop_api
        return 1
    fi
    if ! echo "$body" | jq -e '.types | length > 0' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "No data types returned"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Get data types passed"

    testing::phase::log "INFO" "Test: Generate user data"
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/users" \
        -H "Content-Type: application/json" \
        -d '{"count": 5, "format": "json", "fields": ["id", "name", "email"]}')
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Generate users failed: HTTP $status"
        stop_api
        return 1
    fi
    if ! echo "$body" | jq -e '.success == true and .count == 5' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "User generation returned unexpected result"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Generate users passed"

    testing::phase::log "INFO" "Test: Generate company data"
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/companies" \
        -H "Content-Type: application/json" \
        -d '{"count": 3, "format": "json"}')
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Generate companies failed: HTTP $status"
        stop_api
        return 1
    fi
    if ! echo "$body" | jq -e '.success == true and .count == 3' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "Company generation returned unexpected result"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Generate companies passed"

    testing::phase::log "INFO" "Test: Generate product data"
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/products" \
        -H "Content-Type: application/json" \
        -d '{"count": 10, "format": "json", "fields": ["id", "name", "price"]}')
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Generate products failed: HTTP $status"
        stop_api
        return 1
    fi
    if ! echo "$body" | jq -e '.success == true and .count == 10' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "Product generation returned unexpected result"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Generate products passed"

    testing::phase::log "INFO" "Test: Generate custom schema data"
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/custom" \
        -H "Content-Type: application/json" \
        -d '{"count": 5, "schema": {"id": "uuid", "name": "string", "active": "boolean"}}')
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "Generate custom data failed: HTTP $status"
        stop_api
        return 1
    fi
    if ! echo "$body" | jq -e '.success == true and .type == "custom"' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "Custom schema generation returned unexpected result"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Generate custom data passed"

    testing::phase::log "INFO" "Test: Seed consistency"
    local seed1 seed2 hash1 hash2
    seed1=$(curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 2, "seed": "12345", "fields": ["name", "email"]}')
    seed2=$(curl -s -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": 2, "seed": "12345", "fields": ["name", "email"]}')
    hash1=$(echo "$seed1" | jq -c '.data' | md5sum | cut -d' ' -f1)
    hash2=$(echo "$seed2" | jq -c '.data' | md5sum | cut -d' ' -f1)
    if [ "$hash1" != "$hash2" ]; then
        testing::phase::log "ERROR" "Seed consistency failed: same seed produced different results"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Seed consistency passed"

    testing::phase::log "INFO" "Test: Error handling (invalid count)"
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/users" -H "Content-Type: application/json" -d '{"count": -1}')
    status=$(echo "$response" | tail -1)
    if [ "$status" != "400" ]; then
        testing::phase::log "ERROR" "Invalid count should return 400, got $status"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ Error handling passed"

    testing::phase::log "INFO" "Test: 404 handling"
    response=$(curl -s -w "\n%{http_code}" "$api_url/api/nonexistent")
    status=$(echo "$response" | tail -1)
    if [ "$status" != "404" ]; then
        testing::phase::log "ERROR" "Nonexistent endpoint should return 404, got $status"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ 404 handling passed"

    testing::phase::log "INFO" "Test: XML format support"
    response=$(curl -s -w "\n%{http_code}" -X POST "$api_url/api/generate/users" \
        -H "Content-Type: application/json" \
        -d '{"count": 1, "format": "xml"}')
    status=$(echo "$response" | tail -1)
    body=$(echo "$response" | head -n -1)
    if [ "$status" != "200" ]; then
        testing::phase::log "ERROR" "XML format failed: HTTP $status"
        stop_api
        return 1
    fi
    if ! echo "$body" | jq -e '.format == "xml"' >/dev/null 2>&1; then
        testing::phase::log "ERROR" "XML format not properly returned"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "✓ XML format support passed"

    stop_api
    return 0
}

if main; then
    testing::phase::end_with_summary "Integration tests completed"
else
    stop_api
    testing::phase::end_with_summary "Integration tests failed"
    exit 1
fi
