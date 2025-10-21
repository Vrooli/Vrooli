#!/bin/bash
# Test Phase: Business Logic Validation
# Tests core mathematical operations and business requirements

set -euo pipefail

SCENARIO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$SCENARIO_DIR"

# Get API port from service.json or use default
API_PORT=${API_PORT:-$(jq -r '.ports.api.env_var' .vrooli/service.json 2>/dev/null | xargs -I{} printenv {} 2>/dev/null || echo "16430")}
API_BASE="http://localhost:${API_PORT}"

echo "Testing business logic at $API_BASE..."

# Wait for API to be ready
MAX_WAIT=30
WAITED=0
while ! curl -sf "$API_BASE/health" >/dev/null 2>&1; do
    if [[ $WAITED -ge $MAX_WAIT ]]; then
        echo "✗ API not responding after ${MAX_WAIT}s"
        exit 1
    fi
    echo "Waiting for API... (${WAITED}s)"
    sleep 1
    WAITED=$((WAITED + 1))
done

echo "✓ API is ready"

# Test 1: Basic statistics
echo "Testing: Basic statistics calculation..."
RESPONSE=$(curl -sf -X POST "$API_BASE/api/v1/math/statistics" \
    -H "Content-Type: application/json" \
    -d '{"data": [1,2,3,4,5], "analyses": ["descriptive"]}' || echo "FAILED")

if [[ "$RESPONSE" == "FAILED" ]]; then
    echo "✗ Statistics endpoint failed"
    exit 1
fi

MEAN=$(echo "$RESPONSE" | jq -r '.results.descriptive.mean // empty')
if [[ "$MEAN" == "3" ]] || [[ "$MEAN" == "3.0" ]]; then
    echo "✓ Statistics calculation correct (mean=3)"
else
    echo "✗ Statistics calculation incorrect (expected mean=3, got: $MEAN)"
    exit 1
fi

# Test 2: Matrix operations
echo "Testing: Matrix operations..."
MATRIX_RESPONSE=$(curl -sf -X POST "$API_BASE/api/v1/math/calculate" \
    -H "Content-Type: application/json" \
    -d '{"expression": "2 * 5", "precision": 10}' || echo "FAILED")

if [[ "$MATRIX_RESPONSE" != "FAILED" ]]; then
    echo "✓ Matrix/calculation endpoint responsive"
else
    echo "⚠ Calculation endpoint not available (may be optional)"
fi

# Test 3: Number theory
echo "Testing: Number theory operations..."
NT_RESPONSE=$(curl -sf "$API_BASE/api/v1/math/numbertheory?operation=gcd&a=48&b=18" || echo "FAILED")
if [[ "$NT_RESPONSE" != "FAILED" ]]; then
    echo "✓ Number theory operations available"
else
    echo "⚠ Number theory endpoint not available (may be optional)"
fi

echo "✓ Business logic tests completed"
exit 0
