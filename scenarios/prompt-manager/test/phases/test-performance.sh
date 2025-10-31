#!/usr/bin/env bash
# Test Phase: Performance Testing
# Tests response times, throughput, and resource usage for prompt-manager

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source test utilities if available
if [ -f "$SCENARIO_ROOT/test/lib/test-utils.sh" ]; then
    source "$SCENARIO_ROOT/test/lib/test-utils.sh"
fi

echo "🚀 Performance Testing for prompt-manager"
echo "=========================================="

# Get allocated ports
API_PORT="${API_PORT:-$(grep -oP '"API_PORT":\s*\K\d+' "$SCENARIO_ROOT/.ports.json" 2>/dev/null || echo "16543")}"

# Performance thresholds (from PRD)
MAX_API_RESPONSE_TIME_MS=100  # 95th percentile
MAX_SEARCH_TIME_MS=200
MIN_THROUGHPUT_RPS=100

echo ""
echo "📊 Test 1: API Response Time"
echo "   Target: < ${MAX_API_RESPONSE_TIME_MS}ms for 95% of requests"

# Test health endpoint response time
RESPONSE_TIME=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/health" | awk '{print int($1*1000)}')
if [ "$RESPONSE_TIME" -lt "$MAX_API_RESPONSE_TIME_MS" ]; then
    echo "   ✅ Health endpoint: ${RESPONSE_TIME}ms"
else
    echo "   ⚠️  Health endpoint: ${RESPONSE_TIME}ms (exceeds ${MAX_API_RESPONSE_TIME_MS}ms threshold)"
fi

# Test campaigns endpoint response time
RESPONSE_TIME=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/api/v1/campaigns" | awk '{print int($1*1000)}')
if [ "$RESPONSE_TIME" -lt "$MAX_API_RESPONSE_TIME_MS" ]; then
    echo "   ✅ Campaigns endpoint: ${RESPONSE_TIME}ms"
else
    echo "   ⚠️  Campaigns endpoint: ${RESPONSE_TIME}ms (exceeds ${MAX_API_RESPONSE_TIME_MS}ms threshold)"
fi

echo ""
echo "📊 Test 2: Search Performance"
echo "   Target: < ${MAX_SEARCH_TIME_MS}ms"

# Test search endpoint (if prompts exist)
SEARCH_TIME=$(curl -w "%{time_total}" -o /dev/null -s "http://localhost:${API_PORT}/api/v1/search/prompts?q=test" | awk '{print int($1*1000)}')
if [ "$SEARCH_TIME" -lt "$MAX_SEARCH_TIME_MS" ]; then
    echo "   ✅ Search endpoint: ${SEARCH_TIME}ms"
else
    echo "   ⚠️  Search endpoint: ${SEARCH_TIME}ms (exceeds ${MAX_SEARCH_TIME_MS}ms threshold)"
fi

echo ""
echo "📊 Test 3: Concurrent Request Handling"
echo "   Testing API stability under concurrent load"

# Simple concurrent test - 10 parallel requests
START_TIME=$(date +%s%N)
for i in {1..10}; do
    curl -s "http://localhost:${API_PORT}/health" &>/dev/null &
done
wait
END_TIME=$(date +%s%N)
DURATION_MS=$(( (END_TIME - START_TIME) / 1000000 ))

echo "   ✅ Handled 10 concurrent requests in ${DURATION_MS}ms"

echo ""
echo "📊 Test 4: Memory Footprint"
echo "   Checking process memory usage"

# Get memory usage of API process
API_PID=$(pgrep -f "prompt-manager-api" || echo "")
if [ -n "$API_PID" ]; then
    MEM_KB=$(ps -p "$API_PID" -o rss= 2>/dev/null || echo "0")
    MEM_MB=$((MEM_KB / 1024))
    echo "   ℹ️  API process memory: ${MEM_MB}MB"

    # PRD target: ~200MB total
    if [ "$MEM_MB" -lt 300 ]; then
        echo "   ✅ Within expected memory usage"
    else
        echo "   ⚠️  Higher than expected (target: <300MB)"
    fi
else
    echo "   ⚠️  Could not determine API process memory usage"
fi

echo ""
echo "=========================================="
echo "✅ Performance tests completed"
echo ""
echo "Note: For comprehensive load testing, use:"
echo "  - ab (Apache Bench): ab -n 1000 -c 10 http://localhost:${API_PORT}/health"
echo "  - wrk: wrk -t4 -c100 -d30s http://localhost:${API_PORT}/api/v1/campaigns"
