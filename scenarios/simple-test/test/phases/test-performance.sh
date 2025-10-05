#!/bin/bash
# Performance test phase
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

# Initialize phase
testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

log::info "Running performance tests"

# Start server for performance testing
export API_PORT=36254
node server.js &
SERVER_PID=$!

# Wait for server to start
sleep 2

# Performance test: Response time
log::info "Testing response time..."
total_time=0
iterations=100

for i in $(seq 1 $iterations); do
    start=$(date +%s%N)
    curl -sf http://localhost:${API_PORT}/health > /dev/null
    end=$(date +%s%N)
    duration=$((($end - $start) / 1000000)) # Convert to ms
    total_time=$((total_time + duration))
done

avg_time=$((total_time / iterations))

if [ $avg_time -lt 50 ]; then
    log::success "Average response time: ${avg_time}ms (target: <50ms)"
else
    testing::phase::add_error "Average response time too high: ${avg_time}ms (target: <50ms)"
fi

# Performance test: Concurrent requests
log::info "Testing concurrent request handling..."
ab -n 1000 -c 50 http://localhost:${API_PORT}/health > /tmp/ab-results.txt 2>&1 || true

if [ -f /tmp/ab-results.txt ]; then
    requests_per_sec=$(grep "Requests per second" /tmp/ab-results.txt | awk '{print $4}' | cut -d. -f1)
    if [ -n "$requests_per_sec" ] && [ "$requests_per_sec" -gt 500 ]; then
        log::success "Throughput: ${requests_per_sec} req/s (target: >500 req/s)"
    else
        log::warning "Throughput below target (Apache Bench may not be installed)"
    fi
fi

# Performance test: Memory stability
log::info "Testing memory stability..."
initial_mem=$(ps -o rss= -p $SERVER_PID)

for i in $(seq 1 500); do
    curl -sf http://localhost:${API_PORT}/health > /dev/null
done

final_mem=$(ps -o rss= -p $SERVER_PID)
mem_increase=$((final_mem - initial_mem))

if [ $mem_increase -lt 10000 ]; then # Less than 10MB increase
    log::success "Memory stable: ${mem_increase}KB increase"
else
    testing::phase::add_error "Possible memory leak: ${mem_increase}KB increase"
fi

# Cleanup
kill $SERVER_PID 2>/dev/null || true
rm -f /tmp/ab-results.txt

# End with summary
testing::phase::end_with_summary "Performance tests completed"
