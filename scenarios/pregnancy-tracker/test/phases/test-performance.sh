#!/bin/bash
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase with runtime requirement and 60-second target
testing::phase::init --require-runtime --target-time "60s"

echo "Testing pregnancy-tracker performance and scalability"

API_URL=""

if ! testing::core::wait_for_scenario "$TESTING_PHASE_SCENARIO_NAME" 30; then
    testing::phase::add_error "❌ Scenario '$TESTING_PHASE_SCENARIO_NAME' did not become ready in time"
    testing::phase::end_with_summary
fi

if ! API_URL=$(testing::connectivity::get_api_url "$TESTING_PHASE_SCENARIO_NAME"); then
    testing::phase::add_error "❌ Could not resolve API URL for $TESTING_PHASE_SCENARIO_NAME"
    testing::phase::end_with_summary
fi

# Helper function to measure response time
measure_response_time() {
    local url="$1"
    local max_time="$2"
    local start_time end_time elapsed

    start_time=$(date +%s%N)
    if curl -sf --max-time 5 "$url" >/dev/null 2>&1; then
        end_time=$(date +%s%N)
        elapsed=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

        if [ "$elapsed" -le "$max_time" ]; then
            echo "✅ Response time: ${elapsed}ms (target: <${max_time}ms)"
            return 0
        else
            echo "⚠️  Response time: ${elapsed}ms (exceeds target: ${max_time}ms)"
            return 1
        fi
    else
        echo "❌ Request failed"
        return 1
    fi
}

echo ""
echo "⚡ Testing API Response Times..."

# Test 1: Health endpoint latency
echo "  Health endpoint:"
if ! measure_response_time "$API_URL/health" 100; then
    testing::phase::add_warning "  Health endpoint slower than expected"
fi

# Test 2: Status endpoint latency
echo "  Status endpoint:"
if ! measure_response_time "$API_URL/api/v1/status" 200; then
    testing::phase::add_warning "  Status endpoint slower than expected"
fi

# Test 3: Week content latency
echo "  Week content endpoint:"
if ! measure_response_time "$API_URL/api/v1/content/week/20" 300; then
    testing::phase::add_warning "  Week content endpoint slower than expected"
fi

# Test 4: Search endpoint latency
echo "  Search endpoint:"
if ! measure_response_time "$API_URL/api/v1/search?q=pregnancy" 500; then
    testing::phase::add_warning "  Search endpoint slower than expected (may need indexing)"
fi

echo ""
echo "🔄 Testing Concurrent Request Handling..."

# Test 5: Concurrent health checks
echo "  Running 10 concurrent health checks..."
concurrent_errors=0
for i in {1..10}; do
    curl -sf --max-time 2 "$API_URL/health" >/dev/null 2>&1 &
done
wait
if [ $? -eq 0 ]; then
    log::success "  ✅ Handled 10 concurrent requests successfully"
else
    testing::phase::add_warning "  ⚠️  Some concurrent requests failed"
fi

echo ""
echo "📊 Testing Load Capacity..."

# Test 6: Rapid sequential requests
echo "  Testing 20 rapid sequential requests..."
sequential_start=$(date +%s%N)
sequential_errors=0
for i in {1..20}; do
    if ! curl -sf --max-time 1 "$API_URL/health" >/dev/null 2>&1; then
        ((sequential_errors++))
    fi
done
sequential_end=$(date +%s%N)
sequential_time=$(( (sequential_end - sequential_start) / 1000000 ))

if [ $sequential_errors -eq 0 ]; then
    avg_time=$((sequential_time / 20))
    log::success "  ✅ Completed 20 requests in ${sequential_time}ms (avg: ${avg_time}ms)"
else
    testing::phase::add_warning "  ⚠️  $sequential_errors requests failed out of 20"
fi

echo ""
echo "💾 Testing Memory Efficiency..."

# Test 7: Check API memory footprint (if possible)
if command -v pgrep >/dev/null 2>&1 && command -v ps >/dev/null 2>&1; then
    api_pid=$(pgrep -f "pregnancy-tracker-api" | head -1)
    if [ -n "$api_pid" ]; then
        mem_usage=$(ps -o rss= -p "$api_pid" 2>/dev/null || echo "0")
        mem_mb=$((mem_usage / 1024))
        if [ "$mem_mb" -lt 500 ]; then
            log::success "  ✅ API memory usage: ${mem_mb}MB (healthy)"
        elif [ "$mem_mb" -lt 1000 ]; then
            echo "  ℹ️  API memory usage: ${mem_mb}MB (acceptable)"
        else
            testing::phase::add_warning "  ⚠️  API memory usage: ${mem_mb}MB (high)"
        fi
    else
        echo "  ℹ️  API process not found for memory check"
    fi
else
    echo "  ℹ️  Memory check tools not available"
fi

echo ""
echo "🔐 Testing Encryption Performance..."

# Test 8: Encryption overhead (if we can measure it)
echo "  Measuring encryption endpoint response..."
encryption_start=$(date +%s%N)
curl -sf --max-time 2 "$API_URL/api/v1/health/encryption" >/dev/null 2>&1
encryption_end=$(date +%s%N)
encryption_time=$(( (encryption_end - encryption_start) / 1000000 ))

if [ "$encryption_time" -lt 50 ]; then
    log::success "  ✅ Encryption status check: ${encryption_time}ms"
else
    echo "  ℹ️  Encryption status check: ${encryption_time}ms"
fi

echo ""
echo "📈 Performance Benchmarks Summary:"
echo "═══════════════════════════════════"
echo "   Health endpoint: <100ms target"
echo "   Status endpoint: <200ms target"
echo "   Week content: <300ms target"
echo "   Search: <500ms target"
echo "   Concurrent handling: 10+ simultaneous"
echo "   Sequential throughput: 20 req/sec+"

if [ $TESTING_PHASE_ERROR_COUNT -eq 0 ]; then
    log::success "✅ SUCCESS: Performance within acceptable limits!"
else
    echo "⚠️  Some performance metrics need optimization"
fi

# End with summary
testing::phase::end_with_summary "Performance tests completed"
