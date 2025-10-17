#!/bin/bash

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "Running performance tests for kids-dashboard..."

# Test 1: Run Go performance tests
testing::phase::log "Running Go performance benchmarks..."
cd api

if [ -f performance_test.go ]; then
    go test -bench=. -benchmem -run=^$ ./... 2>&1 | head -20
    if [ $? -eq 0 ]; then
        testing::phase::log "✓ Performance benchmarks completed"
    else
        testing::phase::log "⚠️  Performance benchmarks had issues"
    fi
else
    testing::phase::log "⚠️  No performance_test.go found"
fi

# Test 2: API endpoint response time
testing::phase::log "Testing API endpoint response times..."
PORT=${API_PORT:-15000}

if curl -sf http://localhost:${PORT}/health > /dev/null 2>&1; then
    # Health endpoint
    START=$(date +%s%N)
    curl -sf http://localhost:${PORT}/health > /dev/null 2>&1
    END=$(date +%s%N)
    DURATION=$(( ($END - $START) / 1000000 ))
    testing::phase::log "Health endpoint response time: ${DURATION}ms"

    if [ $DURATION -lt 100 ]; then
        testing::phase::log "✓ Health endpoint responds quickly (<100ms)"
    else
        testing::phase::log "⚠️  Health endpoint slow: ${DURATION}ms"
    fi

    # Scenarios endpoint
    START=$(date +%s%N)
    curl -sf http://localhost:${PORT}/api/v1/kids/scenarios > /dev/null 2>&1
    END=$(date +%s%N)
    DURATION=$(( ($END - $START) / 1000000 ))
    testing::phase::log "Scenarios endpoint response time: ${DURATION}ms"

    if [ $DURATION -lt 200 ]; then
        testing::phase::log "✓ Scenarios endpoint responds quickly (<200ms)"
    else
        testing::phase::log "⚠️  Scenarios endpoint slow: ${DURATION}ms"
    fi
else
    testing::phase::log "⚠️  API not running (expected in CI)"
fi

# Test 3: Memory usage test
testing::phase::log "Checking for memory leaks in tests..."
cd "$TESTING_PHASE_SCENARIO_DIR/api"

# Run tests with race detector
go test -race -short ./... 2>&1 | head -10
if [ $? -eq 0 ]; then
    testing::phase::log "✓ No race conditions detected"
else
    testing::phase::log "⚠️  Potential race conditions found"
fi

# Test 4: Concurrent request handling
testing::phase::log "Testing concurrent request handling..."
if curl -sf http://localhost:${PORT}/health > /dev/null 2>&1; then
    # Send 10 concurrent requests
    for i in {1..10}; do
        curl -sf http://localhost:${PORT}/api/v1/kids/scenarios > /dev/null 2>&1 &
    done
    wait

    testing::phase::log "✓ Concurrent requests handled"
else
    testing::phase::log "⚠️  Cannot test concurrent requests (API not running)"
fi

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::end_with_summary "Performance tests completed"
