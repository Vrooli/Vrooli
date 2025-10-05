#!/bin/bash
# Performance test phase for scalable-app-cookbook
# Tests API response times and throughput

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "60s"

cd "$TESTING_PHASE_SCENARIO_DIR"

testing::phase::section "Performance Tests"

# Check if scenario is running
if ! vrooli scenario status scalable-app-cookbook | grep -q "running"; then
    testing::phase::error "Scenario must be running for performance tests"
    exit 1
fi

# Get API port from environment or default
API_PORT="${API_PORT:-3300}"
API_URL="http://localhost:${API_PORT}"

testing::phase::info "Testing API performance at ${API_URL}"

# Test response time for health endpoint
testing::phase::subsection "Response Time - Health Check"
RESPONSE_TIME=$(curl -o /dev/null -s -w "%{time_total}\n" "${API_URL}/health")
RESPONSE_TIME_MS=$(echo "$RESPONSE_TIME * 1000" | bc)
testing::phase::info "Health check response time: ${RESPONSE_TIME_MS}ms"

if [ "$(echo "$RESPONSE_TIME < 1.0" | bc)" -eq 1 ]; then
    testing::phase::success "Health check response < 1s"
else
    testing::phase::warning "Health check response time: ${RESPONSE_TIME}s"
fi

# Test response time for pattern search
testing::phase::subsection "Response Time - Pattern Search"
RESPONSE_TIME=$(curl -o /dev/null -s -w "%{time_total}\n" "${API_URL}/api/v1/patterns/search?limit=10")
RESPONSE_TIME_MS=$(echo "$RESPONSE_TIME * 1000" | bc)
testing::phase::info "Pattern search response time: ${RESPONSE_TIME_MS}ms"

if [ "$(echo "$RESPONSE_TIME < 2.0" | bc)" -eq 1 ]; then
    testing::phase::success "Pattern search response < 2s"
else
    testing::phase::warning "Pattern search response time: ${RESPONSE_TIME}s"
fi

# Test concurrent requests (basic load test)
testing::phase::subsection "Concurrent Requests"
CONCURRENT_REQUESTS=10
START_TIME=$(date +%s.%N)

for i in $(seq 1 $CONCURRENT_REQUESTS); do
    curl -s "${API_URL}/health" > /dev/null &
done
wait

END_TIME=$(date +%s.%N)
TOTAL_TIME=$(echo "$END_TIME - $START_TIME" | bc)
REQUESTS_PER_SEC=$(echo "$CONCURRENT_REQUESTS / $TOTAL_TIME" | bc -l)

testing::phase::info "Handled ${CONCURRENT_REQUESTS} concurrent requests in ${TOTAL_TIME}s"
testing::phase::info "Throughput: ${REQUESTS_PER_SEC} req/s"

if [ "$(echo "$TOTAL_TIME < 5.0" | bc)" -eq 1 ]; then
    testing::phase::success "Concurrent request handling acceptable"
else
    testing::phase::warning "Concurrent request handling slow: ${TOTAL_TIME}s"
fi

# Test database query performance
testing::phase::subsection "Database Query Performance"
RESPONSE_TIME=$(curl -o /dev/null -s -w "%{time_total}\n" "${API_URL}/api/v1/patterns/stats")
RESPONSE_TIME_MS=$(echo "$RESPONSE_TIME * 1000" | bc)
testing::phase::info "Stats query response time: ${RESPONSE_TIME_MS}ms"

if [ "$(echo "$RESPONSE_TIME < 3.0" | bc)" -eq 1 ]; then
    testing::phase::success "Database queries performing well"
else
    testing::phase::warning "Database queries slow: ${RESPONSE_TIME}s"
fi

testing::phase::end_with_summary "Performance tests completed"
