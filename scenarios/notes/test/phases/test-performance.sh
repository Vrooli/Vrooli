#!/usr/bin/env bash
set -euo pipefail

# Test: Performance Validation
# Tests response times, throughput, and resource efficiency

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Get ports from environment
API_PORT="${API_PORT:-}"

if [ -z "${API_PORT}" ]; then
    echo "❌ API_PORT not set and scenario not running"
    echo "ℹ️  Start the scenario first: make run"
    exit 1
fi

echo "⚡ Testing SmartNotes performance..."
echo "   API: http://localhost:${API_PORT}"

# Track failures
FAILURES=0
WARNINGS=0

# Performance thresholds (in milliseconds)
HEALTH_THRESHOLD=500
API_READ_THRESHOLD=500
API_WRITE_THRESHOLD=1000

measure_response_time() {
    local method=$1
    local endpoint=$2
    local desc=$3
    local threshold=$4
    local data=${5:-}

    local url="http://localhost:${API_PORT}${endpoint}"
    local start=$(date +%s%N)

    if [ -n "${data}" ]; then
        curl -sf -X "${method}" -H "Content-Type: application/json" -d "${data}" "${url}" > /dev/null 2>&1
    else
        curl -sf -X "${method}" "${url}" > /dev/null 2>&1
    fi

    local end=$(date +%s%N)
    local duration=$(( (end - start) / 1000000 )) # Convert to milliseconds

    if [ ${duration} -lt ${threshold} ]; then
        echo "  ✅ ${desc}: ${duration}ms (< ${threshold}ms)"
    elif [ ${duration} -lt $((threshold * 2)) ]; then
        echo "  ⚠️  ${desc}: ${duration}ms (target: ${threshold}ms)"
        ((WARNINGS++))
    else
        echo "  ❌ ${desc}: ${duration}ms (target: ${threshold}ms)"
        ((FAILURES++))
    fi
}

# Test 1: Health Check Response Time
echo "🏥 Testing health check performance..."
measure_response_time "GET" "/health" "Health check" ${HEALTH_THRESHOLD}

# Test 2: Read Operations
echo "📖 Testing read operation performance..."
measure_response_time "GET" "/api/notes" "List notes" ${API_READ_THRESHOLD}
measure_response_time "GET" "/api/folders" "List folders" ${API_READ_THRESHOLD}
measure_response_time "GET" "/api/tags" "List tags" ${API_READ_THRESHOLD}
measure_response_time "GET" "/api/templates" "List templates" ${API_READ_THRESHOLD}

# Test 3: Write Operations
echo "✍️  Testing write operation performance..."
NOTE_DATA='{"title":"Perf Test","content":"Testing write performance","content_type":"markdown"}'
measure_response_time "POST" "/api/notes" "Create note" ${API_WRITE_THRESHOLD} "${NOTE_DATA}"

# Test 4: Search Performance
echo "🔍 Testing search performance..."
SEARCH_DATA='{"query":"test","limit":10}'
measure_response_time "POST" "/api/search" "Text search" ${API_READ_THRESHOLD} "${SEARCH_DATA}"

# Test 5: Concurrent Requests (simple load test)
echo "🔄 Testing concurrent request handling..."
CONCURRENT_REQUESTS=10
START_TIME=$(date +%s)

for i in $(seq 1 ${CONCURRENT_REQUESTS}); do
    curl -sf "http://localhost:${API_PORT}/health" > /dev/null 2>&1 &
done

wait
END_TIME=$(date +%s)
TOTAL_TIME=$((END_TIME - START_TIME))

if [ ${TOTAL_TIME} -lt 2 ]; then
    echo "  ✅ Handled ${CONCURRENT_REQUESTS} concurrent requests in ${TOTAL_TIME}s"
elif [ ${TOTAL_TIME} -lt 5 ]; then
    echo "  ⚠️  Handled ${CONCURRENT_REQUESTS} concurrent requests in ${TOTAL_TIME}s (slower than expected)"
    ((WARNINGS++))
else
    echo "  ❌ Handled ${CONCURRENT_REQUESTS} concurrent requests in ${TOTAL_TIME}s (too slow)"
    ((FAILURES++))
fi

# Summary
echo ""
if [ ${FAILURES} -eq 0 ] && [ ${WARNINGS} -eq 0 ]; then
    echo "✅ Performance validation passed!"
    exit 0
elif [ ${FAILURES} -eq 0 ]; then
    echo "⚠️  Performance validation passed with ${WARNINGS} warning(s)"
    exit 0
else
    echo "❌ Performance validation failed with ${FAILURES} error(s) and ${WARNINGS} warning(s)"
    exit 1
fi
