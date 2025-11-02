#!/bin/bash
APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"


# Source scenario-specific testing helpers
SCENARIO_ROOT="$(cd "${BASH_SOURCE[0]%/*}/../.." && pwd)"
source "${SCENARIO_ROOT}/test/utils/testing-helpers.sh"
testing::phase::init --target-time "60s" --require-runtime

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::log "Running performance tests..."

# Get API port
API_PORT=$(vrooli scenario status data-backup-manager --json 2>/dev/null | jq -r '.scenario_data.allocated_ports.API_PORT // .raw_response.data.allocated_ports.API_PORT // "20010"')
testing::phase::log "Using API port: $API_PORT"

# Test health endpoint response time
testing::phase::log "Testing health endpoint response time..."

health_times=()
for i in {1..10}; do
    start_time=$(date +%s%3N)
    if curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1; then
        end_time=$(date +%s%3N)
        response_time=$((end_time - start_time))
        health_times+=($response_time)
    fi
done

if [ ${#health_times[@]} -gt 0 ]; then
    # Calculate average
    sum=0
    for time in "${health_times[@]}"; do
        sum=$((sum + time))
    done
    avg_time=$((sum / ${#health_times[@]}))

    testing::phase::log "Average health endpoint response time: ${avg_time}ms"

    if [ $avg_time -lt 200 ]; then
        testing::phase::success "Health endpoint meets performance target (<200ms)"
    else
        testing::phase::warn "Health endpoint slower than target: ${avg_time}ms (target: <200ms)"
    fi
fi

# Test backup status endpoint response time
testing::phase::log "Testing backup status endpoint response time..."

status_times=()
for i in {1..10}; do
    start_time=$(date +%s%3N)
    if curl -f -s "http://localhost:$API_PORT/api/v1/backup/status" >/dev/null 2>&1; then
        end_time=$(date +%s%3N)
        response_time=$((end_time - start_time))
        status_times+=($response_time)
    fi
done

if [ ${#status_times[@]} -gt 0 ]; then
    sum=0
    for time in "${status_times[@]}"; do
        sum=$((sum + time))
    done
    avg_time=$((sum / ${#status_times[@]}))

    testing::phase::log "Average status endpoint response time: ${avg_time}ms"

    if [ $avg_time -lt 500 ]; then
        testing::phase::success "Status endpoint meets performance target (<500ms)"
    else
        testing::phase::warn "Status endpoint slower than target: ${avg_time}ms (target: <500ms)"
    fi
fi

# Test concurrent requests
testing::phase::log "Testing concurrent request handling..."

concurrent_requests=10
concurrent_start=$(date +%s)

for i in $(seq 1 $concurrent_requests); do
    curl -f -s "http://localhost:$API_PORT/health" >/dev/null 2>&1 &
done
wait

concurrent_end=$(date +%s)
concurrent_duration=$((concurrent_end - concurrent_start))

testing::phase::log "Handled $concurrent_requests concurrent requests in ${concurrent_duration}s"

if [ $concurrent_duration -lt 3 ]; then
    testing::phase::success "Concurrent request handling acceptable"
else
    testing::phase::warn "Concurrent request handling slower than expected: ${concurrent_duration}s"
fi

# Run Go performance tests
testing::phase::log "Running Go performance tests..."

cd api
if go test -v -run "TestPerformance" -timeout 30s >/tmp/perf-test.log 2>&1; then
    testing::phase::success "Go performance tests passed"
else
    testing::phase::warn "Some Go performance tests failed (non-fatal)"
    cat /tmp/perf-test.log | grep -A 5 "FAIL:"
fi
cd ..

# Memory usage check
testing::phase::log "Checking memory usage..."

if command -v pgrep &>/dev/null; then
    api_pid=$(pgrep -f "data-backup-manager-api" | head -1)
    if [ -n "$api_pid" ]; then
        if command -v ps &>/dev/null; then
            mem_usage=$(ps -o rss= -p $api_pid 2>/dev/null || echo "0")
            mem_mb=$((mem_usage / 1024))
            testing::phase::log "Current memory usage: ${mem_mb}MB"

            if [ $mem_mb -lt 200 ]; then
                testing::phase::success "Memory usage within acceptable range"
            else
                testing::phase::warn "Memory usage high: ${mem_mb}MB"
            fi
        fi
    fi
fi

testing::phase::end_with_summary "Performance tests completed"
