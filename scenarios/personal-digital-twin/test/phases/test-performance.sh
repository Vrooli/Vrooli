#!/bin/bash
set -euo pipefail

# Performance Testing Phase for personal-digital-twin
# This script runs performance tests to ensure API meets response time requirements

APP_ROOT="${APP_ROOT:-$(cd "${BASH_SOURCE[0]%/*}/../../../.." && pwd)}"
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/scenarios/testing/shell/phase-helpers.sh"

testing::phase::init --target-time "180s"

cd "$TESTING_PHASE_SCENARIO_DIR"

echo "⚡ Running performance tests for personal-digital-twin..."

# Ensure scenario is running
SCENARIO_NAME="personal-digital-twin"

if ! pgrep -f "personal-digital-twin-api" > /dev/null; then
    echo "⚠️  Scenario not running, attempting to start..."
    vrooli scenario start "$SCENARIO_NAME" || {
        echo "❌ Failed to start scenario for performance tests"
        exit 1
    }
    sleep 5
fi

API_PORT="${API_PORT:-8080}"
CHAT_PORT="${CHAT_PORT:-8081}"

# Wait for API to be ready
echo "⏳ Waiting for API to be ready..."
for i in {1..30}; do
    if curl -s "http://localhost:${API_PORT}/health" > /dev/null 2>&1; then
        echo "✅ API is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ API failed to become ready"
        exit 1
    fi
    sleep 1
done

# Create a test persona for performance tests
echo "📝 Setting up test data..."
PERSONA_RESPONSE=$(curl -s -X POST "http://localhost:${API_PORT}/api/persona/create" \
    -H "Content-Type: application/json" \
    -d '{"name": "Performance Test Persona", "description": "Persona for performance testing"}')

PERSONA_ID=$(echo "$PERSONA_RESPONSE" | jq -r '.id')
echo "✅ Test persona created: $PERSONA_ID"

# Performance Test 1: Health endpoint response time
echo ""
echo "🧪 Test 1: Health endpoint response time..."
HEALTH_TIME=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:${API_PORT}/health")
echo "  Response time: ${HEALTH_TIME}s"
if (( $(echo "$HEALTH_TIME < 0.1" | bc -l) )); then
    echo "  ✅ Health endpoint performance: PASS"
else
    echo "  ⚠️  Health endpoint performance: SLOW (expected < 0.1s)"
fi

# Performance Test 2: Persona retrieval response time
echo ""
echo "🧪 Test 2: Persona retrieval response time..."
GET_TIME=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:${API_PORT}/api/persona/${PERSONA_ID}")
echo "  Response time: ${GET_TIME}s"
if (( $(echo "$GET_TIME < 0.5" | bc -l) )); then
    echo "  ✅ Persona retrieval performance: PASS"
else
    echo "  ⚠️  Persona retrieval performance: SLOW (expected < 0.5s)"
fi

# Performance Test 3: List personas response time
echo ""
echo "🧪 Test 3: List personas response time..."
LIST_TIME=$(curl -s -w "%{time_total}" -o /dev/null "http://localhost:${API_PORT}/api/personas")
echo "  Response time: ${LIST_TIME}s"
if (( $(echo "$LIST_TIME < 1.0" | bc -l) )); then
    echo "  ✅ List personas performance: PASS"
else
    echo "  ⚠️  List personas performance: SLOW (expected < 1.0s)"
fi

# Performance Test 4: Chat response time
echo ""
echo "🧪 Test 4: Chat response time..."
CHAT_TIME=$(curl -s -w "%{time_total}" -o /dev/null \
    -X POST "http://localhost:${CHAT_PORT}/api/chat" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"message\": \"Performance test message\"}")
echo "  Response time: ${CHAT_TIME}s"
if (( $(echo "$CHAT_TIME < 2.0" | bc -l) )); then
    echo "  ✅ Chat response performance: PASS"
else
    echo "  ⚠️  Chat response performance: SLOW (expected < 2.0s)"
fi

# Performance Test 5: Concurrent request handling
echo ""
echo "🧪 Test 5: Concurrent request handling (10 simultaneous requests)..."
START_TIME=$(date +%s.%N)

for i in {1..10}; do
    curl -s "http://localhost:${API_PORT}/api/persona/${PERSONA_ID}" > /dev/null &
done

wait

END_TIME=$(date +%s.%N)
CONCURRENT_TIME=$(echo "$END_TIME - $START_TIME" | bc)
echo "  Total time for 10 concurrent requests: ${CONCURRENT_TIME}s"

if (( $(echo "$CONCURRENT_TIME < 5.0" | bc -l) )); then
    echo "  ✅ Concurrent request handling: PASS"
else
    echo "  ⚠️  Concurrent request handling: SLOW (expected < 5.0s)"
fi

# Performance Test 6: Search query response time
echo ""
echo "🧪 Test 6: Search query response time..."
SEARCH_TIME=$(curl -s -w "%{time_total}" -o /dev/null \
    -X POST "http://localhost:${API_PORT}/api/search" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"query\": \"performance test search\", \"limit\": 10}")
echo "  Response time: ${SEARCH_TIME}s"
if (( $(echo "$SEARCH_TIME < 1.0" | bc -l) )); then
    echo "  ✅ Search query performance: PASS"
else
    echo "  ⚠️  Search query performance: SLOW (expected < 1.0s)"
fi

# Performance Test 7: API token creation response time
echo ""
echo "🧪 Test 7: API token creation response time..."
TOKEN_TIME=$(curl -s -w "%{time_total}" -o /dev/null \
    -X POST "http://localhost:${API_PORT}/api/tokens/create" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"name\": \"perf-test-token\"}")
echo "  Response time: ${TOKEN_TIME}s"
if (( $(echo "$TOKEN_TIME < 0.5" | bc -l) )); then
    echo "  ✅ Token creation performance: PASS"
else
    echo "  ⚠️  Token creation performance: SLOW (expected < 0.5s)"
fi

# Performance Test 8: Data source connection response time
echo ""
echo "🧪 Test 8: Data source connection response time..."
DS_TIME=$(curl -s -w "%{time_total}" -o /dev/null \
    -X POST "http://localhost:${API_PORT}/api/datasource/connect" \
    -H "Content-Type: application/json" \
    -d "{\"persona_id\": \"${PERSONA_ID}\", \"source_type\": \"file\", \"source_config\": {}}")
echo "  Response time: ${DS_TIME}s"
if (( $(echo "$DS_TIME < 0.5" | bc -l) )); then
    echo "  ✅ Data source connection performance: PASS"
else
    echo "  ⚠️  Data source connection performance: SLOW (expected < 0.5s)"
fi

echo ""
echo "🎉 Performance testing completed!"

testing::phase::end_with_summary "Performance tests completed"
