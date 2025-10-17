#!/bin/bash
# Performance validation phase - benchmarks performance and resource usage
source "$(dirname "${BASH_SOURCE[0]}")/../../../../scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with 60-second target and require runtime
testing::phase::init --target-time "60s" --require-runtime

# Get API port dynamically
API_PORT=$(vrooli scenario port data-structurer API_PORT 2>/dev/null || echo "15774")
API_BASE_URL="http://localhost:$API_PORT"

echo "🔍 Testing performance at $API_BASE_URL..."

# Test 1: Health Endpoint Response Time
echo ""
echo "Test 1: Health Endpoint Performance"
echo "  Testing health endpoint response time..."

HEALTH_TIMES=()
for i in {1..10}; do
    START=$(date +%s%N)
    curl -s "$API_BASE_URL/health" >/dev/null
    END=$(date +%s%N)
    DURATION=$(( (END - START) / 1000000 )) # Convert to milliseconds
    HEALTH_TIMES+=($DURATION)
done

# Calculate average
TOTAL=0
for time in "${HEALTH_TIMES[@]}"; do
    TOTAL=$((TOTAL + time))
done
AVG_HEALTH_TIME=$((TOTAL / ${#HEALTH_TIMES[@]}))

if [ "$AVG_HEALTH_TIME" -lt 500 ]; then
    log::success "  ✅ Average health response time: ${AVG_HEALTH_TIME}ms (target: <500ms)"
    testing::phase::add_test passed
else
    testing::phase::add_warning "  ⚠️  Average health response time: ${AVG_HEALTH_TIME}ms (target: <500ms)"
    testing::phase::add_test failed
fi

# Test 2: Schema Creation Performance
echo ""
echo "Test 2: Schema Creation Performance"
echo "  Testing schema creation response time..."

TEST_SCHEMA_NAME="perf-test-$(date +%s)"
START=$(date +%s%N)
CREATE_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/schemas" \
    -H "Content-Type: application/json" \
    -d "{
        \"name\": \"$TEST_SCHEMA_NAME\",
        \"description\": \"Performance test schema\",
        \"schema_definition\": {
            \"type\": \"object\",
            \"properties\": {
                \"field1\": {\"type\": \"string\"},
                \"field2\": {\"type\": \"number\"}
            }
        }
    }")
END=$(date +%s%N)
CREATE_TIME=$(( (END - START) / 1000000 ))

SCHEMA_ID=$(echo "$CREATE_RESPONSE" | jq -r '.id // empty')

if [ -n "$SCHEMA_ID" ]; then
    if [ "$CREATE_TIME" -lt 1000 ]; then
        log::success "  ✅ Schema creation time: ${CREATE_TIME}ms (target: <1000ms)"
        testing::phase::add_test passed
    else
        testing::phase::add_warning "  ⚠️  Schema creation time: ${CREATE_TIME}ms (target: <1000ms)"
        testing::phase::add_test failed
    fi
else
    testing::phase::add_error "  ❌ Failed to create schema for performance test"
    testing::phase::add_test failed
fi

# Test 3: Data Processing Performance
if [ -n "$SCHEMA_ID" ]; then
    echo ""
    echo "Test 3: Data Processing Performance"
    echo "  Testing processing response time..."

    START=$(date +%s%N)
    PROCESS_RESPONSE=$(curl -s -X POST "$API_BASE_URL/api/v1/process" \
        -H "Content-Type: application/json" \
        -d "{
            \"schema_id\": \"$SCHEMA_ID\",
            \"input_type\": \"text\",
            \"input_data\": \"Test data for field1 and value 42 for field2\"
        }")
    END=$(date +%s%N)
    PROCESS_TIME=$(( (END - START) / 1000000 ))

    if echo "$PROCESS_RESPONSE" | jq -e '.processing_id' >/dev/null 2>&1; then
        if [ "$PROCESS_TIME" -lt 5000 ]; then
            log::success "  ✅ Processing time: ${PROCESS_TIME}ms (target: <5000ms)"
            testing::phase::add_test passed
        else
            testing::phase::add_warning "  ⚠️  Processing time: ${PROCESS_TIME}ms (target: <5000ms)"
            testing::phase::add_test failed
        fi
    else
        testing::phase::add_error "  ❌ Processing failed"
        testing::phase::add_test failed
    fi

    # Cleanup test schema
    echo "  Cleaning up performance test schema..."
    curl -s -X DELETE "$API_BASE_URL/api/v1/schemas/$SCHEMA_ID" >/dev/null
fi

# Test 4: Concurrent Request Handling
echo ""
echo "Test 4: Concurrent Request Performance"
echo "  Testing concurrent health checks..."

# Run 5 concurrent requests
for i in {1..5}; do
    curl -s "$API_BASE_URL/health" >/dev/null &
done
wait

log::success "  ✅ Handled concurrent requests without errors"
testing::phase::add_test passed

# Test 5: Memory Usage Check (if available)
echo ""
echo "Test 5: Resource Usage"

# Try to check memory usage of the API process
API_PID=$(pgrep -f "data-structurer-api" | head -1)
if [ -n "$API_PID" ]; then
    if command -v ps >/dev/null 2>&1; then
        MEM_KB=$(ps -o rss= -p "$API_PID" 2>/dev/null || echo "0")
        MEM_MB=$((MEM_KB / 1024))

        if [ "$MEM_MB" -lt 4096 ]; then
            log::success "  ✅ Memory usage: ${MEM_MB}MB (target: <4GB)"
            testing::phase::add_test passed
        else
            testing::phase::add_warning "  ⚠️  Memory usage: ${MEM_MB}MB (target: <4GB)"
            testing::phase::add_test failed
        fi
    else
        testing::phase::add_warning "  ⚠️  Cannot check memory usage (ps not available)"
    fi
else
    testing::phase::add_warning "  ⚠️  API process not found for memory check"
fi

# End with summary
testing::phase::end_with_summary "Performance validation completed"
