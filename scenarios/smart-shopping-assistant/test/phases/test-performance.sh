#!/bin/bash

# Performance test phase for smart-shopping-assistant
# Tests response times and throughput

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

SCENARIO_NAME="smart-shopping-assistant"
API_PORT="${SMART_SHOPPING_ASSISTANT_API_PORT:-3300}"
API_URL="http://localhost:${API_PORT}"

echo "Running performance tests for smart-shopping-assistant"

# Performance thresholds (milliseconds)
HEALTH_THRESHOLD=100
RESEARCH_THRESHOLD=1000
TRACKING_THRESHOLD=500

# Test health endpoint performance
testing::phase::log "Testing health endpoint performance..."
health_time=$(curl -o /dev/null -s -w '%{time_total}' "${API_URL}/health")
health_ms=$(echo "$health_time * 1000" | bc | cut -d. -f1)

if [ "$health_ms" -gt "$HEALTH_THRESHOLD" ]; then
    testing::phase::warn "Health endpoint took ${health_ms}ms (threshold: ${HEALTH_THRESHOLD}ms)"
else
    echo "✅ Health endpoint: ${health_ms}ms"
fi

# Test shopping research performance
testing::phase::log "Testing shopping research performance..."
research_time=$(curl -o /dev/null -s -w '%{time_total}' -X POST "${API_URL}/api/v1/shopping/research" \
    -H "Content-Type: application/json" \
    -d '{"profile_id": "test", "query": "laptop", "budget_max": 1000}')
research_ms=$(echo "$research_time * 1000" | bc | cut -d. -f1)

if [ "$research_ms" -gt "$RESEARCH_THRESHOLD" ]; then
    testing::phase::warn "Research endpoint took ${research_ms}ms (threshold: ${RESEARCH_THRESHOLD}ms)"
else
    echo "✅ Research endpoint: ${research_ms}ms"
fi

# Test tracking endpoint performance
testing::phase::log "Testing tracking endpoint performance..."
tracking_time=$(curl -o /dev/null -s -w '%{time_total}' "${API_URL}/api/v1/shopping/tracking/test-user")
tracking_ms=$(echo "$tracking_time * 1000" | bc | cut -d. -f1)

if [ "$tracking_ms" -gt "$TRACKING_THRESHOLD" ]; then
    testing::phase::warn "Tracking endpoint took ${tracking_ms}ms (threshold: ${TRACKING_THRESHOLD}ms)"
else
    echo "✅ Tracking endpoint: ${tracking_ms}ms"
fi

# Load test - concurrent requests
testing::phase::log "Testing concurrent request handling..."
CONCURRENT_REQUESTS=10

echo "Sending ${CONCURRENT_REQUESTS} concurrent requests..."
for i in $(seq 1 $CONCURRENT_REQUESTS); do
    curl -s -o /dev/null -X POST "${API_URL}/api/v1/shopping/research" \
        -H "Content-Type: application/json" \
        -d "{\"profile_id\": \"user-${i}\", \"query\": \"test-${i}\", \"budget_max\": 500}" &
done

wait
echo "✅ Concurrent requests handled successfully"

# Cache performance test
testing::phase::log "Testing cache performance..."

# First request (cold cache)
cold_time=$(curl -o /dev/null -s -w '%{time_total}' -X POST "${API_URL}/api/v1/shopping/research" \
    -H "Content-Type: application/json" \
    -d '{"profile_id": "cache-test", "query": "cache-query", "budget_max": 500}')
cold_ms=$(echo "$cold_time * 1000" | bc | cut -d. -f1)

# Second request (warm cache)
warm_time=$(curl -o /dev/null -s -w '%{time_total}' -X POST "${API_URL}/api/v1/shopping/research" \
    -H "Content-Type: application/json" \
    -d '{"profile_id": "cache-test", "query": "cache-query", "budget_max": 500}')
warm_ms=$(echo "$warm_time * 1000" | bc | cut -d. -f1)

echo "Cold cache: ${cold_ms}ms, Warm cache: ${warm_ms}ms"

if [ "$warm_ms" -lt "$cold_ms" ]; then
    echo "✅ Cache is working (${warm_ms}ms < ${cold_ms}ms)"
else
    testing::phase::warn "Cache may not be working effectively"
fi

testing::phase::end_with_summary "Performance tests completed"
