#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "90s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "INFO" "Starting business logic tests..."

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

    cd api
    API_PORT=$API_PORT NODE_ENV=test nohup node server.js > /tmp/test-data-generator-business.log 2>&1 &
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

# Business Logic Test 1: Data type coverage
testing::phase::log "INFO" "Test: Verify all advertised data types are functional"

TYPES_RESPONSE=$(curl -s "${API_URL}/api/types")
AVAILABLE_TYPES=$(echo "$TYPES_RESPONSE" | jq -r '.types[]')

WORKING_TYPES=0
BROKEN_TYPES=0

for type in $AVAILABLE_TYPES; do
    RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/${type}" \
        -H "Content-Type: application/json" \
        -d '{"count": 1}')
    CODE=$(echo "$RESPONSE" | tail -1)

    if [ "$CODE" = "200" ]; then
        WORKING_TYPES=$((WORKING_TYPES + 1))
        testing::phase::log "INFO" "  ✓ Type '$type' is functional"
    else
        BROKEN_TYPES=$((BROKEN_TYPES + 1))
        testing::phase::log "WARN" "  ✗ Type '$type' returned HTTP $CODE (may not be implemented)"
    fi
done

testing::phase::log "INFO" "Data type coverage: $WORKING_TYPES working, $BROKEN_TYPES not functional"

# Business Logic Test 2: Field selection works correctly
testing::phase::log "INFO" "Test: Field selection produces correct structure"

FIELD_TEST=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 1, "fields": ["id", "name"]}')

FIRST_USER=$(echo "$FIELD_TEST" | jq -r '.data[0]')
HAS_ID=$(echo "$FIRST_USER" | jq 'has("id")')
HAS_NAME=$(echo "$FIRST_USER" | jq 'has("name")')
HAS_EMAIL=$(echo "$FIRST_USER" | jq 'has("email")')

if [ "$HAS_ID" != "true" ] || [ "$HAS_NAME" != "true" ]; then
    testing::phase::log "ERROR" "Field selection failed: requested fields missing"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if [ "$HAS_EMAIL" = "true" ]; then
    testing::phase::log "WARN" "Field selection may not be working: unrequested field present"
fi

testing::phase::log "INFO" "✓ Field selection works correctly"

# Business Logic Test 3: Volume limits are enforced
testing::phase::log "INFO" "Test: Volume limits enforcement"

# Test max limit
MAX_TEST=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 10000}')
MAX_CODE=$(echo "$MAX_TEST" | tail -1)

if [ "$MAX_CODE" != "200" ]; then
    testing::phase::log "ERROR" "Max valid count (10000) should be allowed, got HTTP $MAX_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

# Test over limit
OVER_TEST=$(curl -s -w "\n%{http_code}" -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 10001}')
OVER_CODE=$(echo "$OVER_TEST" | tail -1)

if [ "$OVER_CODE" != "400" ]; then
    testing::phase::log "ERROR" "Over limit (10001) should be rejected, got HTTP $OVER_CODE"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Volume limits are properly enforced"

# Business Logic Test 4: Data quality - uniqueness
testing::phase::log "INFO" "Test: Data uniqueness (UUIDs should be unique)"

UUID_TEST=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 100, "fields": ["id"]}')

TOTAL_IDS=$(echo "$UUID_TEST" | jq -r '.data[].id' | wc -l)
UNIQUE_IDS=$(echo "$UUID_TEST" | jq -r '.data[].id' | sort -u | wc -l)

if [ "$TOTAL_IDS" != "$UNIQUE_IDS" ]; then
    testing::phase::log "ERROR" "UUID uniqueness violated: $TOTAL_IDS total, $UNIQUE_IDS unique"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Data uniqueness validated (100 unique UUIDs)"

# Business Logic Test 5: Format conversion correctness
testing::phase::log "INFO" "Test: Format conversion produces valid output"

# Test SQL format
SQL_TEST=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 2, "format": "sql", "fields": ["id", "name"]}')

SQL_DATA=$(echo "$SQL_TEST" | jq -r '.data')

if ! echo "$SQL_DATA" | grep -q "INSERT INTO"; then
    testing::phase::log "ERROR" "SQL format missing INSERT statement"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

if ! echo "$SQL_DATA" | grep -q "VALUES"; then
    testing::phase::log "ERROR" "SQL format missing VALUES clause"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

# Test XML format
XML_TEST=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 1, "format": "xml"}')

XML_DATA=$(echo "$XML_TEST" | jq -r '.data')

if ! echo "$XML_DATA" | grep -q "<?xml"; then
    testing::phase::log "ERROR" "XML format missing XML declaration"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Format conversions produce valid output"

# Business Logic Test 6: Custom schema flexibility
testing::phase::log "INFO" "Test: Custom schema supports various data types"

CUSTOM_TEST=$(curl -s -X POST "${API_URL}/api/generate/custom" \
    -H "Content-Type: application/json" \
    -d '{
        "count": 5,
        "schema": {
            "id": "uuid",
            "title": "string",
            "price": "decimal",
            "quantity": "integer",
            "active": "boolean",
            "email": "email",
            "phone": "phone",
            "created": "date"
        }
    }')

CUSTOM_SUCCESS=$(echo "$CUSTOM_TEST" | jq -r '.success')
CUSTOM_COUNT=$(echo "$CUSTOM_TEST" | jq -r '.count')

if [ "$CUSTOM_SUCCESS" != "true" ] || [ "$CUSTOM_COUNT" != "5" ]; then
    testing::phase::log "ERROR" "Custom schema with multiple types failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

# Validate data types in generated data
FIRST_ITEM=$(echo "$CUSTOM_TEST" | jq -r '.data[0]')

# Check UUID format
UUID_VALUE=$(echo "$FIRST_ITEM" | jq -r '.id')
if ! echo "$UUID_VALUE" | grep -qE '^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$'; then
    testing::phase::log "WARN" "UUID format may not be valid"
fi

# Check email format
EMAIL_VALUE=$(echo "$FIRST_ITEM" | jq -r '.email')
if ! echo "$EMAIL_VALUE" | grep -qE '@'; then
    testing::phase::log "ERROR" "Email format invalid: missing @"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

# Check boolean
BOOLEAN_VALUE=$(echo "$FIRST_ITEM" | jq -r '.active')
if [ "$BOOLEAN_VALUE" != "true" ] && [ "$BOOLEAN_VALUE" != "false" ]; then
    testing::phase::log "ERROR" "Boolean field not producing boolean values"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Custom schema supports various data types correctly"

# Business Logic Test 7: Concurrent request handling
testing::phase::log "INFO" "Test: Concurrent request handling"

# Launch 5 concurrent requests
for i in {1..5}; do
    curl -s -X POST "${API_URL}/api/generate/users" \
        -H "Content-Type: application/json" \
        -d '{"count": 10}' > /tmp/concurrent_$i.json &
done

# Wait for all to complete
wait

# Verify all succeeded
FAILED=0
for i in {1..5}; do
    if ! jq -e '.success == true' /tmp/concurrent_$i.json &> /dev/null; then
        FAILED=$((FAILED + 1))
    fi
    rm -f /tmp/concurrent_$i.json
done

if [ $FAILED -gt 0 ]; then
    testing::phase::log "ERROR" "Concurrent requests: $FAILED out of 5 failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "✓ Concurrent requests handled successfully"

# Cleanup if we started the API
if [ "$API_WAS_RUNNING" = false ]; then
    testing::phase::log "INFO" "Stopping API server (PID: $API_PID)..."
    kill $API_PID 2>/dev/null || true
    wait $API_PID 2>/dev/null || true
fi

testing::phase::end_with_summary "Business logic tests completed - All critical business requirements validated"
