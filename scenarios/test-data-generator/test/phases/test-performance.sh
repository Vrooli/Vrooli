#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "INFO" "Starting performance tests..."

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
    API_PORT=$API_PORT NODE_ENV=test nohup node server.js > /tmp/test-data-generator-performance.log 2>&1 &
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

# Performance Test 1: Small dataset generation speed
testing::phase::log "INFO" "Test: Small dataset (10 records) generation time"

START_TIME=$(date +%s%N)
SMALL_RESPONSE=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 10}')
END_TIME=$(date +%s%N)

SMALL_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))  # Convert to milliseconds
SMALL_SUCCESS=$(echo "$SMALL_RESPONSE" | jq -r '.success')

if [ "$SMALL_SUCCESS" != "true" ]; then
    testing::phase::log "ERROR" "Small dataset generation failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "Small dataset (10 records) generated in ${SMALL_DURATION}ms"

if [ $SMALL_DURATION -gt 1000 ]; then
    testing::phase::log "WARN" "Small dataset took >1s (${SMALL_DURATION}ms)"
else
    testing::phase::log "INFO" "✓ Small dataset performance acceptable"
fi

# Performance Test 2: Medium dataset generation speed
testing::phase::log "INFO" "Test: Medium dataset (100 records) generation time"

START_TIME=$(date +%s%N)
MEDIUM_RESPONSE=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 100}')
END_TIME=$(date +%s%N)

MEDIUM_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
MEDIUM_SUCCESS=$(echo "$MEDIUM_RESPONSE" | jq -r '.success')

if [ "$MEDIUM_SUCCESS" != "true" ]; then
    testing::phase::log "ERROR" "Medium dataset generation failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "Medium dataset (100 records) generated in ${MEDIUM_DURATION}ms"

if [ $MEDIUM_DURATION -gt 5000 ]; then
    testing::phase::log "WARN" "Medium dataset took >5s (${MEDIUM_DURATION}ms)"
else
    testing::phase::log "INFO" "✓ Medium dataset performance acceptable"
fi

# Performance Test 3: Large dataset generation speed
testing::phase::log "INFO" "Test: Large dataset (1000 records) generation time"

START_TIME=$(date +%s%N)
LARGE_RESPONSE=$(curl -s -X POST "${API_URL}/api/generate/users" \
    -H "Content-Type: application/json" \
    -d '{"count": 1000}')
END_TIME=$(date +%s%N)

LARGE_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
LARGE_SUCCESS=$(echo "$LARGE_RESPONSE" | jq -r '.success')

if [ "$LARGE_SUCCESS" != "true" ]; then
    testing::phase::log "ERROR" "Large dataset generation failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "Large dataset (1000 records) generated in ${LARGE_DURATION}ms"

if [ $LARGE_DURATION -gt 10000 ]; then
    testing::phase::log "WARN" "Large dataset took >10s (${LARGE_DURATION}ms)"
else
    testing::phase::log "INFO" "✓ Large dataset performance acceptable"
fi

# Performance Test 4: Maximum allowed dataset
testing::phase::log "INFO" "Test: Maximum dataset (10000 records) generation time"

START_TIME=$(date +%s%N)
MAX_RESPONSE=$(curl -s -X POST "${API_URL}/api/generate/products" \
    -H "Content-Type: application/json" \
    -d '{"count": 10000}')
END_TIME=$(date +%s%N)

MAX_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
MAX_SUCCESS=$(echo "$MAX_RESPONSE" | jq -r '.success')

if [ "$MAX_SUCCESS" != "true" ]; then
    testing::phase::log "ERROR" "Maximum dataset generation failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "Maximum dataset (10000 records) generated in ${MAX_DURATION}ms"

if [ $MAX_DURATION -gt 30000 ]; then
    testing::phase::log "WARN" "Maximum dataset took >30s (${MAX_DURATION}ms)"
else
    testing::phase::log "INFO" "✓ Maximum dataset performance acceptable"
fi

# Performance Test 5: Response time consistency
testing::phase::log "INFO" "Test: Response time consistency (10 requests)"

DURATIONS=()
for i in {1..10}; do
    START_TIME=$(date +%s%N)
    curl -s -X POST "${API_URL}/api/generate/users" \
        -H "Content-Type: application/json" \
        -d '{"count": 50}' > /dev/null
    END_TIME=$(date +%s%N)

    DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
    DURATIONS+=($DURATION)
done

# Calculate average
TOTAL=0
for dur in "${DURATIONS[@]}"; do
    TOTAL=$((TOTAL + dur))
done
AVG_DURATION=$((TOTAL / 10))

# Calculate standard deviation (simplified)
SUM_SQUARED_DIFF=0
for dur in "${DURATIONS[@]}"; do
    DIFF=$((dur - AVG_DURATION))
    SUM_SQUARED_DIFF=$((SUM_SQUARED_DIFF + DIFF * DIFF))
done
VARIANCE=$((SUM_SQUARED_DIFF / 10))
STD_DEV=$(echo "scale=2; sqrt($VARIANCE)" | bc -l)

testing::phase::log "INFO" "Average response time: ${AVG_DURATION}ms (σ=${STD_DEV}ms)"

if (( $(echo "$STD_DEV > $AVG_DURATION * 0.5" | bc -l) )); then
    testing::phase::log "WARN" "High response time variance detected (σ > 50% of mean)"
else
    testing::phase::log "INFO" "✓ Response times are consistent"
fi

# Performance Test 6: Concurrent load handling
testing::phase::log "INFO" "Test: Concurrent load (10 simultaneous requests)"

START_TIME=$(date +%s%N)

for i in {1..10}; do
    curl -s -X POST "${API_URL}/api/generate/users" \
        -H "Content-Type: application/json" \
        -d '{"count": 20}' > /tmp/concurrent_perf_$i.json &
done

wait

END_TIME=$(date +%s%N)
CONCURRENT_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))

# Verify all succeeded
CONCURRENT_FAILED=0
for i in {1..10}; do
    if ! jq -e '.success == true' /tmp/concurrent_perf_$i.json &> /dev/null; then
        CONCURRENT_FAILED=$((CONCURRENT_FAILED + 1))
    fi
    rm -f /tmp/concurrent_perf_$i.json
done

if [ $CONCURRENT_FAILED -gt 0 ]; then
    testing::phase::log "ERROR" "Concurrent load: $CONCURRENT_FAILED out of 10 requests failed"
    [ "$API_WAS_RUNNING" = false ] && kill $API_PID 2>/dev/null || true
    exit 1
fi

testing::phase::log "INFO" "10 concurrent requests (20 records each) completed in ${CONCURRENT_DURATION}ms"

if [ $CONCURRENT_DURATION -gt 10000 ]; then
    testing::phase::log "WARN" "Concurrent load took >10s (${CONCURRENT_DURATION}ms)"
else
    testing::phase::log "INFO" "✓ Concurrent load handling acceptable"
fi

# Performance Test 7: Memory efficiency check
testing::phase::log "INFO" "Test: Memory usage stability"

if [ -n "$API_PID" ]; then
    # Get initial memory
    INITIAL_MEM=$(ps -o rss= -p $API_PID 2>/dev/null || echo "0")

    # Generate several large datasets
    for i in {1..5}; do
        curl -s -X POST "${API_URL}/api/generate/users" \
            -H "Content-Type: application/json" \
            -d '{"count": 1000}' > /dev/null
    done

    sleep 2  # Allow GC to run

    # Get final memory
    FINAL_MEM=$(ps -o rss= -p $API_PID 2>/dev/null || echo "0")

    MEM_INCREASE=$((FINAL_MEM - INITIAL_MEM))
    MEM_INCREASE_PCT=$(( (MEM_INCREASE * 100) / (INITIAL_MEM + 1) ))

    testing::phase::log "INFO" "Memory usage: ${INITIAL_MEM}KB -> ${FINAL_MEM}KB (${MEM_INCREASE_PCT}% increase)"

    if [ $MEM_INCREASE_PCT -gt 100 ]; then
        testing::phase::log "WARN" "Significant memory increase detected (>${MEM_INCREASE_PCT}%)"
    else
        testing::phase::log "INFO" "✓ Memory usage is stable"
    fi
else
    testing::phase::log "WARN" "Cannot check memory usage (API was already running)"
fi

# Performance Test 8: Different data types performance
testing::phase::log "INFO" "Test: Performance across different data types"

TYPES=("users" "companies" "products")
for type in "${TYPES[@]}"; do
    START_TIME=$(date +%s%N)
    TYPE_RESPONSE=$(curl -s -X POST "${API_URL}/api/generate/${type}" \
        -H "Content-Type: application/json" \
        -d '{"count": 100}')
    END_TIME=$(date +%s%N)

    TYPE_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
    TYPE_SUCCESS=$(echo "$TYPE_RESPONSE" | jq -r '.success')

    if [ "$TYPE_SUCCESS" = "true" ]; then
        testing::phase::log "INFO" "Type '$type': ${TYPE_DURATION}ms for 100 records"
    else
        testing::phase::log "WARN" "Type '$type': Failed to generate"
    fi
done

testing::phase::log "INFO" "✓ Performance tested across data types"

# Performance Test 9: Format conversion overhead
testing::phase::log "INFO" "Test: Format conversion performance impact"

FORMATS=("json" "xml" "sql")
for format in "${FORMATS[@]}"; do
    START_TIME=$(date +%s%N)
    FORMAT_RESPONSE=$(curl -s -X POST "${API_URL}/api/generate/users" \
        -H "Content-Type: application/json" \
        -d "{\"count\": 100, \"format\": \"${format}\"}")
    END_TIME=$(date +%s%N)

    FORMAT_DURATION=$(( (END_TIME - START_TIME) / 1000000 ))
    FORMAT_SUCCESS=$(echo "$FORMAT_RESPONSE" | jq -r '.success')

    if [ "$FORMAT_SUCCESS" = "true" ]; then
        testing::phase::log "INFO" "Format '$format': ${FORMAT_DURATION}ms for 100 records"
    fi
done

testing::phase::log "INFO" "✓ Format conversion performance measured"

# Performance Summary
testing::phase::log "INFO" "===== Performance Summary ====="
testing::phase::log "INFO" "Small dataset (10):    ${SMALL_DURATION}ms"
testing::phase::log "INFO" "Medium dataset (100):  ${MEDIUM_DURATION}ms"
testing::phase::log "INFO" "Large dataset (1000):  ${LARGE_DURATION}ms"
testing::phase::log "INFO" "Max dataset (10000):   ${MAX_DURATION}ms"
testing::phase::log "INFO" "Average response:      ${AVG_DURATION}ms (σ=${STD_DEV}ms)"
testing::phase::log "INFO" "Concurrent load (10):  ${CONCURRENT_DURATION}ms"
testing::phase::log "INFO" "==============================="

# Cleanup if we started the API
if [ "$API_WAS_RUNNING" = false ]; then
    testing::phase::log "INFO" "Stopping API server (PID: $API_PID)..."
    kill $API_PID 2>/dev/null || true
    wait $API_PID 2>/dev/null || true
fi

testing::phase::end_with_summary "Performance tests completed - All performance metrics collected"
