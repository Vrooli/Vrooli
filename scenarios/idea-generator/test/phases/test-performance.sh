#!/bin/bash
# Performance Test Phase for idea-generator
# Validates response times and resource usage

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Ensure scenario is running
# Use command substitution to avoid pipefail issues
STATUS_OUTPUT=$(vrooli scenario status idea-generator 2>/dev/null)
if ! echo "$STATUS_OUTPUT" | grep -q "RUNNING"; then
    log::error "Scenario is not running. Start it with: make start"
    exit 1
fi

# Get API port
API_PORT=$(echo "$STATUS_OUTPUT" | grep "API_PORT:" | awk '{print $2}')

# Test API Health Response Time (< 500ms target)
log::subheader "API Health Response Time"
START_TIME=$(date +%s%N)
curl -sf "http://localhost:${API_PORT}/health" > /dev/null
END_TIME=$(date +%s%N)
RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))

if [ "$RESPONSE_TIME" -lt 500 ]; then
    log::success "API health response time: ${RESPONSE_TIME}ms (target: <500ms)"
else
    log::warning "API health response time: ${RESPONSE_TIME}ms (slower than target)"
fi

# Test Campaigns Endpoint Response Time
log::subheader "Campaigns Endpoint Response Time"
START_TIME=$(date +%s%N)
curl -sf "http://localhost:${API_PORT}/api/campaigns" > /dev/null
END_TIME=$(date +%s%N)
RESPONSE_TIME=$(( (END_TIME - START_TIME) / 1000000 ))

if [ "$RESPONSE_TIME" -lt 1000 ]; then
    log::success "Campaigns response time: ${RESPONSE_TIME}ms (target: <1000ms)"
else
    log::warning "Campaigns response time: ${RESPONSE_TIME}ms (slower than target)"
fi

# Test Memory Usage
log::subheader "Memory Usage"
API_PID=$(pgrep -f "idea-generator-api" | head -1)
if [ -n "$API_PID" ]; then
    MEMORY_KB=$(ps -o rss= -p "$API_PID")
    MEMORY_MB=$(( MEMORY_KB / 1024 ))
    log::info "API memory usage: ${MEMORY_MB}MB"

    if [ "$MEMORY_MB" -lt 500 ]; then
        log::success "Memory usage is reasonable"
    else
        log::warning "Memory usage is higher than expected"
    fi
else
    log::warning "Could not measure API memory usage"
fi

# Run Go performance tests if they exist
log::subheader "Go Performance Tests"
if [ -f "api/performance_test.go" ]; then
    cd api
    if go test -bench=. -benchtime=1s -run=^$ > /tmp/bench.out 2>&1; then
        log::success "Go performance benchmarks completed"
        grep "Benchmark" /tmp/bench.out | head -5
    else
        log::warning "Go performance benchmarks failed or not found"
    fi
    cd ..
else
    log::info "No Go performance tests found"
fi

testing::phase::end_with_summary "Performance tests completed"
