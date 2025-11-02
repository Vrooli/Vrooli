#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"
cd "$TESTING_PHASE_SCENARIO_DIR"

API_PORT=""
API_PID=""
API_WAS_RUNNING=false
PERF_LOG="/tmp/test-data-generator-performance.log"

stop_api() {
    if [ "$API_WAS_RUNNING" = false ] && [ -n "$API_PID" ]; then
        testing::phase::log "INFO" "Stopping API server (PID: $API_PID)..."
        kill "$API_PID" 2>/dev/null || true
        wait "$API_PID" 2>/dev/null || true
        API_PID=""
    fi
}

measure_request() {
    local payload="$1"
    local start end duration response
    start=$(date +%s%N)
    response=$(curl -s -X POST "$API_URL/api/generate/${2}" -H "Content-Type: application/json" -d "$payload")
    end=$(date +%s%N)
    duration=$(( (end - start) / 1000000 ))
    echo "$duration" "$response"
}

main() {
    testing::phase::log "INFO" "Starting performance tests..."

    API_PORT=$(jq -r '.interfaces.api.port // .lifecycle.develop.steps[]? | select(.name=="start-api") | .env.API_PORT // empty' .vrooli/service.json 2>/dev/null)
    if [ -z "$API_PORT" ] || [ "$API_PORT" = "null" ]; then
        API_PORT=3001
    fi
    API_URL="http://localhost:${API_PORT}"

    if curl -fs "$API_URL/health" >/dev/null 2>&1; then
        testing::phase::log "INFO" "API is already running on port $API_PORT"
        API_WAS_RUNNING=true
    else
        testing::phase::log "INFO" "Starting API server on port $API_PORT..."
        API_WAS_RUNNING=false
        pushd api >/dev/null
        API_PORT=$API_PORT NODE_ENV=test nohup node server.js > "$PERF_LOG" 2>&1 &
        API_PID=$!
        popd >/dev/null
        if ! kill -0 "$API_PID" >/dev/null 2>&1; then
            testing::phase::log "ERROR" "Failed to start API server"
            API_PID=""
            return 1
        fi
        local wait_count=0
        local max_wait=30
        while ! curl -fs "$API_URL/health" >/dev/null 2>&1; do
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

    local start end
    # Small dataset
    testing::phase::log "INFO" "Test: Small dataset (10 records) generation time"
    start=$(date +%s%N)
    local small_response
    small_response=$(curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d '{"count": 10}')
    end=$(date +%s%N)
    SMALL_DURATION=$(( (end - start) / 1000000 ))
    if [ "$(echo "$small_response" | jq -r '.success')" != "true" ]; then
        testing::phase::log "ERROR" "Small dataset generation failed"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "Small dataset (10 records) generated in ${SMALL_DURATION}ms"
    if [ $SMALL_DURATION -gt 1000 ]; then
        testing::phase::log "WARN" "Small dataset took >1s (${SMALL_DURATION}ms)"
    else
        testing::phase::log "INFO" "✓ Small dataset performance acceptable"
    fi

    # Medium dataset
    testing::phase::log "INFO" "Test: Medium dataset (100 records) generation time"
    start=$(date +%s%N)
    local medium_response
    medium_response=$(curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d '{"count": 100}')
    end=$(date +%s%N)
    MEDIUM_DURATION=$(( (end - start) / 1000000 ))
    if [ "$(echo "$medium_response" | jq -r '.success')" != "true" ]; then
        testing::phase::log "ERROR" "Medium dataset generation failed"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "Medium dataset (100 records) generated in ${MEDIUM_DURATION}ms"
    if [ $MEDIUM_DURATION -gt 5000 ]; then
        testing::phase::log "WARN" "Medium dataset took >5s (${MEDIUM_DURATION}ms)"
    else
        testing::phase::log "INFO" "✓ Medium dataset performance acceptable"
    fi

    # Large dataset
    testing::phase::log "INFO" "Test: Large dataset (1000 records) generation time"
    start=$(date +%s%N)
    local large_response
    large_response=$(curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d '{"count": 1000}')
    end=$(date +%s%N)
    LARGE_DURATION=$(( (end - start) / 1000000 ))
    if [ "$(echo "$large_response" | jq -r '.success')" != "true" ]; then
        testing::phase::log "ERROR" "Large dataset generation failed"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "Large dataset (1000 records) generated in ${LARGE_DURATION}ms"
    if [ $LARGE_DURATION -gt 10000 ]; then
        testing::phase::log "WARN" "Large dataset took >10s (${LARGE_DURATION}ms)"
    else
        testing::phase::log "INFO" "✓ Large dataset performance acceptable"
    fi

    # Maximum dataset
    testing::phase::log "INFO" "Test: Maximum dataset (10000 records) generation time"
    start=$(date +%s%N)
    local max_response
    max_response=$(curl -s -X POST "$API_URL/api/generate/products" -H "Content-Type: application/json" -d '{"count": 10000}')
    end=$(date +%s%N)
    MAX_DURATION=$(( (end - start) / 1000000 ))
    if [ "$(echo "$max_response" | jq -r '.success')" != "true" ]; then
        testing::phase::log "ERROR" "Maximum dataset generation failed"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "Maximum dataset (10000 records) generated in ${MAX_DURATION}ms"
    if [ $MAX_DURATION -gt 30000 ]; then
        testing::phase::log "WARN" "Maximum dataset took >30s (${MAX_DURATION}ms)"
    else
        testing::phase::log "INFO" "✓ Maximum dataset performance acceptable"
    fi

    # Response time consistency
    testing::phase::log "INFO" "Test: Response time consistency (10 requests)"
    local durations=()
    local total=0
    local idx duration
    for idx in {1..10}; do
        start=$(date +%s%N)
        curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d '{"count": 50}' >/dev/null
        end=$(date +%s%N)
        duration=$(( (end - start) / 1000000 ))
        durations+=($duration)
        total=$((total + duration))
    done
    AVG_DURATION=$((total / 10))
    local sum_sq=0 diff
    for duration in "${durations[@]}"; do
        diff=$((duration - AVG_DURATION))
        sum_sq=$((sum_sq + diff * diff))
    done
    local variance=$((sum_sq / 10))
    STD_DEV=$(echo "scale=2; sqrt($variance)" | bc -l)
    testing::phase::log "INFO" "Average response time: ${AVG_DURATION}ms (σ=${STD_DEV}ms)"
    if (( $(echo "$STD_DEV > $AVG_DURATION * 0.5" | bc -l) )); then
        testing::phase::log "WARN" "High response time variance detected (σ > 50% of mean)"
    else
        testing::phase::log "INFO" "✓ Response times are consistent"
    fi

    # Concurrent load
    testing::phase::log "INFO" "Test: Concurrent load (10 simultaneous requests)"
    start=$(date +%s%N)
    local i
    for i in {1..10}; do
        curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d '{"count": 20}' > "/tmp/concurrent_perf_$i.json" &
    done
    wait
    end=$(date +%s%N)
    CONCURRENT_DURATION=$(( (end - start) / 1000000 ))
    local concurrent_failed=0
    for i in {1..10}; do
        if ! jq -e '.success == true' "/tmp/concurrent_perf_$i.json" >/dev/null 2>&1; then
            concurrent_failed=$((concurrent_failed + 1))
        fi
        rm -f "/tmp/concurrent_perf_$i.json"
    done
    if [ $concurrent_failed -gt 0 ]; then
        testing::phase::log "ERROR" "Concurrent load: $concurrent_failed out of 10 requests failed"
        stop_api
        return 1
    fi
    testing::phase::log "INFO" "10 concurrent requests (20 records each) completed in ${CONCURRENT_DURATION}ms"
    if [ $CONCURRENT_DURATION -gt 10000 ]; then
        testing::phase::log "WARN" "Concurrent load took >10s (${CONCURRENT_DURATION}ms)"
    else
        testing::phase::log "INFO" "✓ Concurrent load handling acceptable"
    fi

    # Memory usage
    testing::phase::log "INFO" "Test: Memory usage stability"
    if [ -n "$API_PID" ]; then
        local initial_mem final_mem mem_increase mem_increase_pct
        initial_mem=$(ps -o rss= -p "$API_PID" 2>/dev/null || echo "0")
        for i in {1..5}; do
            curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d '{"count": 1000}' >/dev/null
        done
        sleep 2
        final_mem=$(ps -o rss= -p "$API_PID" 2>/dev/null || echo "0")
        mem_increase=$((final_mem - initial_mem))
        mem_increase_pct=$(( (mem_increase * 100) / (initial_mem + 1) ))
        testing::phase::log "INFO" "Memory usage: ${initial_mem}KB -> ${final_mem}KB (${mem_increase_pct}% increase)"
        if [ $mem_increase_pct -gt 100 ]; then
            testing::phase::log "WARN" "Significant memory increase detected (>${mem_increase_pct}%)"
        else
            testing::phase::log "INFO" "✓ Memory usage is stable"
        fi
    else
        testing::phase::log "WARN" "Cannot check memory usage (API was already running)"
    fi

    # Data type performance
    testing::phase::log "INFO" "Test: Performance across different data types"
    local data_types=("users" "companies" "products")
    local type type_duration type_response type_success
    for type in "${data_types[@]}"; do
        start=$(date +%s%N)
        type_response=$(curl -s -X POST "$API_URL/api/generate/${type}" -H "Content-Type: application/json" -d '{"count": 100}')
        end=$(date +%s%N)
        type_duration=$(( (end - start) / 1000000 ))
        type_success=$(echo "$type_response" | jq -r '.success')
        if [ "$type_success" = "true" ]; then
            testing::phase::log "INFO" "Type '$type': ${type_duration}ms for 100 records"
        else
            testing::phase::log "WARN" "Type '$type': Failed to generate"
        fi
    done
    testing::phase::log "INFO" "✓ Performance tested across data types"

    # Format performance
    testing::phase::log "INFO" "Test: Format conversion performance impact"
    local formats=("json" "xml" "sql")
    local format format_response format_duration
    for format in "${formats[@]}"; do
        start=$(date +%s%N)
        format_response=$(curl -s -X POST "$API_URL/api/generate/users" -H "Content-Type: application/json" -d "{\"count\": 100, \"format\": \"${format}\"}")
        end=$(date +%s%N)
        format_duration=$(( (end - start) / 1000000 ))
        if [ "$(echo "$format_response" | jq -r '.success')" = "true" ]; then
            testing::phase::log "INFO" "Format '$format': ${format_duration}ms for 100 records"
        fi
    done
    testing::phase::log "INFO" "✓ Format conversion performance measured"

    testing::phase::log "INFO" "===== Performance Summary ====="
    testing::phase::log "INFO" "Small dataset (10):    ${SMALL_DURATION}ms"
    testing::phase::log "INFO" "Medium dataset (100):  ${MEDIUM_DURATION}ms"
    testing::phase::log "INFO" "Large dataset (1000):  ${LARGE_DURATION}ms"
    testing::phase::log "INFO" "Max dataset (10000):   ${MAX_DURATION}ms"
    testing::phase::log "INFO" "Average response:      ${AVG_DURATION}ms (σ=${STD_DEV}ms)"
    testing::phase::log "INFO" "Concurrent load (10):  ${CONCURRENT_DURATION}ms"
    testing::phase::log "INFO" "==============================="

    stop_api
    return 0
}

if main; then
    testing::phase::end_with_summary "Performance tests completed"
else
    stop_api
    testing::phase::end_with_summary "Performance tests failed"
    exit 1
fi
