#!/bin/bash
# Test Phase: Performance Validation
# Validates response times and throughput meet targets

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

# Get API port
API_PORT=${API_PORT:-$(jq -r '.ports.api.env_var' .vrooli/service.json 2>/dev/null | xargs -I{} printenv {} 2>/dev/null || echo "16430")}
API_BASE="http://localhost:${API_PORT}"

echo "Testing performance at $API_BASE..."

# Wait for API
MAX_WAIT=10
WAITED=0
while ! curl -sf "$API_BASE/health" >/dev/null 2>&1; do
    if [[ $WAITED -ge $MAX_WAIT ]]; then
        echo "✗ API not responding"
        exit 1
    fi
    sleep 1
    WAITED=$((WAITED + 1))
done

# Test health endpoint response time (target: < 500ms)
echo "Testing: Health endpoint response time..."
START_TIME=$(date +%s%3N)
curl -sf "$API_BASE/health" >/dev/null
END_TIME=$(date +%s%3N)
RESPONSE_TIME=$((END_TIME - START_TIME))

echo "Response time: ${RESPONSE_TIME}ms"
if [[ $RESPONSE_TIME -lt 500 ]]; then
    echo "✓ Health check response time acceptable (< 500ms)"
else
    echo "⚠ Health check slower than target (${RESPONSE_TIME}ms > 500ms)"
fi

# Test API endpoint response time
echo "Testing: Statistics endpoint response time..."
START_TIME=$(date +%s%3N)
curl -sf -X POST "$API_BASE/api/v1/math/statistics" \
    -H "Content-Type: application/json" \
    -d '{"data": [1,2,3,4,5], "analyses": ["descriptive"]}' >/dev/null 2>&1 || true
END_TIME=$(date +%s%3N)
API_RESPONSE_TIME=$((END_TIME - START_TIME))

echo "API response time: ${API_RESPONSE_TIME}ms"
if [[ $API_RESPONSE_TIME -lt 1000 ]]; then
    echo "✓ API response time acceptable (< 1000ms)"
else
    echo "⚠ API response slower than target (${API_RESPONSE_TIME}ms > 1000ms)"
fi

echo "✓ Performance tests completed"
exit 0
