#!/bin/bash
# Phase 5: Performance tests - <120 seconds
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "120s"

cd "$TESTING_PHASE_SCENARIO_DIR"

# Define baseline performance thresholds
readonly HEALTH_ENDPOINT_MAX_MS=500
readonly API_ENDPOINT_MAX_MS=1000
readonly CONCURRENT_REQUESTS=10

testing::phase::section "API Response Time Tests"

# Test health endpoint response time
testing::phase::step "Health endpoint response time < ${HEALTH_ENDPOINT_MAX_MS}ms"
health_time=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/health" | awk '{print int($1*1000)}')
if [ "$health_time" -lt "$HEALTH_ENDPOINT_MAX_MS" ]; then
    testing::phase::pass "Health endpoint: ${health_time}ms (< ${HEALTH_ENDPOINT_MAX_MS}ms)"
else
    testing::phase::fail "Health endpoint too slow: ${health_time}ms (>= ${HEALTH_ENDPOINT_MAX_MS}ms)"
fi

# Test main API endpoints response time
testing::phase::step "API endpoints response time < ${API_ENDPOINT_MAX_MS}ms"
endpoints=("/api/applications" "/api/agents" "/api/queue")
all_passed=true
for endpoint in "${endpoints[@]}"; do
    endpoint_time=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}${endpoint}" | awk '{print int($1*1000)}')
    if [ "$endpoint_time" -lt "$API_ENDPOINT_MAX_MS" ]; then
        testing::phase::pass "  ${endpoint}: ${endpoint_time}ms"
    else
        testing::phase::fail "  ${endpoint} too slow: ${endpoint_time}ms (>= ${API_ENDPOINT_MAX_MS}ms)"
        all_passed=false
    fi
done
[ "$all_passed" = true ] || exit 1

testing::phase::section "Concurrent Request Handling"

# Test concurrent requests
testing::phase::step "Handle ${CONCURRENT_REQUESTS} concurrent requests"
start_time=$(date +%s%3N)
for i in $(seq 1 $CONCURRENT_REQUESTS); do
    curl -s "http://localhost:${API_PORT}/health" > /dev/null &
done
wait
end_time=$(date +%s%3N)
total_time=$((end_time - start_time))
avg_time=$((total_time / CONCURRENT_REQUESTS))

if [ "$avg_time" -lt "$HEALTH_ENDPOINT_MAX_MS" ]; then
    testing::phase::pass "Concurrent requests: ${total_time}ms total, ${avg_time}ms avg"
else
    testing::phase::fail "Concurrent requests too slow: ${avg_time}ms avg (>= ${HEALTH_ENDPOINT_MAX_MS}ms)"
fi

testing::phase::section "Database Connection Performance"

# Test database query performance
testing::phase::step "Database queries complete in reasonable time"
db_time=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/api/system/db-status" | awk '{print int($1*1000)}')
if [ "$db_time" -lt "$API_ENDPOINT_MAX_MS" ]; then
    testing::phase::pass "Database query: ${db_time}ms (< ${API_ENDPOINT_MAX_MS}ms)"
else
    testing::phase::fail "Database query too slow: ${db_time}ms (>= ${API_ENDPOINT_MAX_MS}ms)"
fi

testing::phase::section "Resource Integration Performance"

# Test vector database performance
testing::phase::step "Vector database queries complete in reasonable time"
vector_time=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/api/system/vector-status" | awk '{print int($1*1000)}')
if [ "$vector_time" -lt "$API_ENDPOINT_MAX_MS" ]; then
    testing::phase::pass "Vector DB query: ${vector_time}ms (< ${API_ENDPOINT_MAX_MS}ms)"
else
    testing::phase::fail "Vector DB query too slow: ${vector_time}ms (>= ${API_ENDPOINT_MAX_MS}ms)"
fi

# Test AI integration performance
testing::phase::step "AI integration checks complete in reasonable time"
ai_time=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/api/system/ai-status" | awk '{print int($1*1000)}')
if [ "$ai_time" -lt "$API_ENDPOINT_MAX_MS" ]; then
    testing::phase::pass "AI integration: ${ai_time}ms (< ${API_ENDPOINT_MAX_MS}ms)"
else
    testing::phase::fail "AI integration too slow: ${ai_time}ms (>= ${API_ENDPOINT_MAX_MS}ms)"
fi

testing::phase::end_with_summary "Performance tests completed"
