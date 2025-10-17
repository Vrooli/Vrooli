#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "INFO" "Starting integration tests..."

# Get port from service.json
API_PORT=$(jq -r '.lifecycle.api.port // 3001' .vrooli/service.json)
API_URL="http://localhost:${API_PORT}"

# Check if API is already running
if curl -f -s "${API_URL}/health" &> /dev/null; then
    testing::phase::log "INFO" "API is already running on port $API_PORT"
    API_WAS_RUNNING=true
else
    testing::phase::log "INFO" "Starting API server on port $API_PORT..."
    API_WAS_RUNNING=false

    # Start API in background
    cd api
    API_PORT=$API_PORT NODE_ENV=test nohup node server.js > /tmp/test-data-generator-integration.log 2>&1 &
    API_PID=$!
    cd ..

    # Wait for API to start
    MAX_WAIT=30
    WAIT_COUNT=0
    while ! curl -f -s "${API_URL}/health" &> /dev/null; do
        sleep 1
        WAIT_COUNT=$((WAIT_COUNT + 1))
        if [ $WAIT_COUNT -ge $MAX_WAIT ]; then
            testing::phase::log "ERROR" "API failed to start within ${MAX_WAIT}s"
            kill $API_PID 2>/dev/null || true
            exit 1
        fi
    done

    testing::phase::log "INFO" "API started successfully (PID: $API_PID)"
fi

# Run integration tests
testing::phase::log "INFO" "Running API integration tests..."

# Test 1: Health check
testing::phase::log "INFO" "Test: Health check endpoint"
HEALTH_RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/health")
HEALTH_CODE=$(echo "$HEALTH_RESPONSE" | tail -1)
HEALTH_BODY=$(echo "$HEALTH_RESPONSE" | head -n -1)

if [ "$HEALTH_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Health check failed: HTTP $HEALTH_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$HEALTH_BODY" | jq -e '.status == "healthy"' &> /dev/null; then
    testing::phase::log "ERROR" "Health check returned unexpected status"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Health check passed"

# Test 2: Get data types
testing::phase::log "INFO" "Test: Get available data types"
TYPES_RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/api/types")
TYPES_CODE=$(echo "$TYPES_RESPONSE" | tail -1)
TYPES_BODY=$(echo "$TYPES_RESPONSE" | head -n -1)

if [ "$TYPES_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Get types failed: HTTP $TYPES_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$TYPES_BODY" | jq -e '.types | length > 0' &> /dev/null; then
    testing::phase::log "ERROR" "No data types returned"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Get data types passed"

# Test 3: Generate users
testing::phase::log "INFO" "Test: Generate user data"
USERS_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 5, "format": "json", "fields": ["id", "name", "email"]}')
USERS_CODE=$(echo "$USERS_RESPONSE" | tail -1)
USERS_BODY=$(echo "$USERS_RESPONSE" | head -n -1)

if [ "$USERS_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Generate users failed: HTTP $USERS_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$USERS_BODY" | jq -e '.success == true and .count == 5' &> /dev/null; then
    testing::phase::log "ERROR" "User generation returned unexpected result"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Generate users passed"

# Test 4: Generate companies
testing::phase::log "INFO" "Test: Generate company data"
COMPANIES_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/companies" \
    -H "Content-Type: application/json" \
    -d '{"count": 3, "format": "json"}')
COMPANIES_CODE=$(echo "$COMPANIES_RESPONSE" | tail -1)
COMPANIES_BODY=$(echo "$COMPANIES_RESPONSE" | head -n -1)

if [ "$COMPANIES_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Generate companies failed: HTTP $COMPANIES_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$COMPANIES_BODY" | jq -e '.success == true and .count == 3' &> /dev/null; then
    testing::phase::log "ERROR" "Company generation returned unexpected result"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Generate companies passed"

# Test 5: Generate products
testing::phase::log "INFO" "Test: Generate product data"
PRODUCTS_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/products" \
    -H "Content-Type: application/json" \
    -d '{"count": 10, "format": "json", "fields": ["id", "name", "price"]}')
PRODUCTS_CODE=$(echo "$PRODUCTS_RESPONSE" | tail -1)
PRODUCTS_BODY=$(echo "$PRODUCTS_RESPONSE" | head -n -1)

if [ "$PRODUCTS_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Generate products failed: HTTP $PRODUCTS_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$PRODUCTS_BODY" | jq -e '.success == true and .count == 10' &> /dev/null; then
    testing::phase::log "ERROR" "Product generation returned unexpected result"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Generate products passed"

# Test 6: Custom schema generation
testing::phase::log "INFO" "Test: Generate custom schema data"
CUSTOM_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/custom" \
    -H "Content-Type: application/json" \
    -d '{"count": 5, "schema": {"id": "uuid", "name": "string", "active": "boolean"}}')
CUSTOM_CODE=$(echo "$CUSTOM_RESPONSE" | tail -1)
CUSTOM_BODY=$(echo "$CUSTOM_RESPONSE" | head -n -1)

if [ "$CUSTOM_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Generate custom data failed: HTTP $CUSTOM_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$CUSTOM_BODY" | jq -e '.success == true and .type == "custom"' &> /dev/null; then
    testing::phase::log "ERROR" "Custom schema generation returned unexpected result"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Generate custom data passed"

# Test 7: Seed consistency
testing::phase::log "INFO" "Test: Seed consistency"
SEED1=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 2, "seed": "12345", "fields": ["name", "email"]}')
SEED2=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 2, "seed": "12345", "fields": ["name", "email"]}')

HASH1=$(echo "$SEED1" | jq -c '.data' | md5sum | cut -d' ' -f1)
HASH2=$(echo "$SEED2" | jq -c '.data' | md5sum | cut -d' ' -f1)

if [ "$HASH1" != "$HASH2" ]; then
    testing::phase::log "ERROR" "Seed consistency failed: same seed produced different results"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Seed consistency passed"

# Test 8: Error handling - invalid count
testing::phase::log "INFO" "Test: Error handling (invalid count)"
ERROR_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": -1}')
ERROR_CODE=$(echo "$ERROR_RESPONSE" | tail -1)

if [ "$ERROR_CODE" != "400" ]; then
    testing::phase::log "ERROR" "Invalid count should return 400, got $ERROR_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Error handling passed"

# Test 9: 404 handling
testing::phase::log "INFO" "Test: 404 handling"
NOT_FOUND_RESPONSE=$(curl -s -w "\n%{http_code}" "${API_URL}/api/nonexistent")
NOT_FOUND_CODE=$(echo "$NOT_FOUND_RESPONSE" | tail -1)

if [ "$NOT_FOUND_CODE" != "404" ]; then
    testing::phase::log "ERROR" "Nonexistent endpoint should return 404, got $NOT_FOUND_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ 404 handling passed"

# Test 10: Format support
testing::phase::log "INFO" "Test: XML format support"
XML_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 1, "format": "xml"}')
XML_CODE=$(echo "$XML_RESPONSE" | tail -1)
XML_BODY=$(echo "$XML_RESPONSE" | head -n -1)

if [ "$XML_CODE" != "200" ]; then
    testing::phase::log "ERROR" "XML format failed: HTTP $XML_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$XML_BODY" | jq -e '.format == "xml"' &> /dev/null; then
    testing::phase::log "ERROR" "XML format not properly returned"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ XML format support passed"

# Cleanup if we started the API
if [ "$API_WAS_RUNNING" = false ]; then
    testing::phase::log "INFO" "Stopping API server (PID: $API_PID)..."
    kill $API_PID 2>/dev/null || true
    wait $API_PID 2>/dev/null || true
fi

testing::phase::end_with_summary "Integration tests completed - All tests passed"
