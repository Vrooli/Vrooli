#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running performance tests..."

# Run Go performance tests
testing::phase::log "Running Go performance test suite..."

cd api

# Run performance tests (not in short mode, with benchmarks)
if go test -tags=testing -v -run "TestPerformance" -timeout 3m . 2>&1 | tee /tmp/performance-test.log; then
    testing::phase::success "Performance tests passed"

    # Extract and display performance metrics
    if grep -q "Created.*tasks in" /tmp/performance-test.log; then
        testing::phase::log "Performance metrics:"
        grep "Created\|Retrieved\|Updated\|Deleted\|Concurrent\|Mixed" /tmp/performance-test.log | grep -E "avg:|total ops" | while read -r line; do
            testing::phase::log "  $line"
        done
    fi
else
    testing::phase::error "Performance tests failed"
    cat /tmp/performance-test.log
    cd ..
    testing::phase::end_with_summary "Performance tests failed" 1
fi

# Run benchmarks
testing::phase::log "Running benchmarks..."

if go test -tags=testing -bench=. -benchmem -run=^$ . 2>&1 | tee /tmp/benchmark.log; then
    testing::phase::success "Benchmarks completed"

    # Display benchmark results
    if grep -q "Benchmark" /tmp/benchmark.log; then
        testing::phase::log "Benchmark results:"
        grep "Benchmark" /tmp/benchmark.log | while read -r line; do
            testing::phase::log "  $line"
        done
    fi
else
    testing::phase::warn "Benchmarks failed (non-fatal)"
fi

cd ..

# Test concurrent request handling
testing::phase::log "Testing concurrent request handling..."

API_PORT="${API_PORT:-8080}"
if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    testing::phase::log "Running concurrent health check test..."

    # Use GNU parallel if available, otherwise skip
    if command -v parallel &>/dev/null; then
        testing::phase::log "Sending 100 concurrent requests..."
        seq 100 | parallel -j 10 "curl -s -w '\n%{http_code}\n' http://localhost:$API_PORT/health" > /tmp/concurrent-test.log 2>&1

        # Count successful responses
        success_count=$(grep "^200$" /tmp/concurrent-test.log | wc -l)
        testing::phase::log "Successful responses: $success_count/100"

        if [ "$success_count" -ge 95 ]; then
            testing::phase::success "Concurrent request test passed (≥95% success rate)"
        else
            testing::phase::warn "Concurrent request test had lower success rate: $success_count%"
        fi
    else
        testing::phase::warn "GNU parallel not available, skipping concurrent request test"
    fi
else
    testing::phase::warn "API not running, skipping live concurrent tests"
fi

# Memory usage test (if running)
testing::phase::log "Checking memory usage..."

if pgrep -f "swarm-manager-api" >/dev/null; then
    pid=$(pgrep -f "swarm-manager-api" | head -1)
    if command -v ps &>/dev/null; then
        mem_usage=$(ps -p "$pid" -o rss= | awk '{print $1/1024}')
        testing::phase::log "Current memory usage: ${mem_usage}MB"

        # Warn if memory usage is very high (>500MB for this service)
        if (( $(echo "$mem_usage > 500" | bc -l) )); then
            testing::phase::warn "High memory usage detected: ${mem_usage}MB"
        else
            testing::phase::success "Memory usage within normal range"
        fi
    fi
else
    testing::phase::warn "swarm-manager-api not running, skipping memory test"
fi

# CPU usage test
testing::phase::log "Checking CPU usage..."

if pgrep -f "swarm-manager-api" >/dev/null; then
    pid=$(pgrep -f "swarm-manager-api" | head -1)
    if command -v ps &>/dev/null; then
        cpu_usage=$(ps -p "$pid" -o %cpu= 2>/dev/null || echo "0.0")
        testing::phase::log "Current CPU usage: ${cpu_usage}%"

        # This is just informational
        testing::phase::success "CPU usage checked"
    fi
else
    testing::phase::warn "swarm-manager-api not running, skipping CPU test"
fi

# Response time test
testing::phase::log "Testing response times..."

if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
    total_time=0
    iterations=10

    for i in $(seq 1 $iterations); do
        response_time=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:$API_PORT/health")
        total_time=$(echo "$total_time + $response_time" | bc)
    done

    avg_time=$(echo "scale=3; $total_time / $iterations" | bc)
    testing::phase::log "Average response time: ${avg_time}s"

    # Response should be under 100ms on average
    if (( $(echo "$avg_time < 0.1" | bc -l) )); then
        testing::phase::success "Response time within acceptable range"
    else
        testing::phase::warn "Response time slower than expected: ${avg_time}s"
    fi
else
    testing::phase::warn "API not running, skipping response time test"
fi

testing::phase::end_with_summary "Performance tests completed"
