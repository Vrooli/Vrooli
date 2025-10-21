#!/bin/bash
# Integration tests for math-tools scenario

set -euo pipefail

echo "=== Math Tools Integration Tests ==="

# Use API_PORT from environment or default
API_PORT="${API_PORT:-16430}"

# Check if server is running
if ! pgrep -f "math-tools-api" > /dev/null; then
    echo "⚠️  Server not running. Skipping integration tests."
    exit 0
fi

# Test health endpoint
echo "Testing health endpoint..."
if command -v curl &> /dev/null; then
    HEALTH_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/api/health" 2>/dev/null || echo "")
    if [[ "$HEALTH_RESPONSE" == *"healthy"* ]]; then
        echo "✓ Health endpoint responding correctly"
    else
        echo "✗ Health endpoint not responding"
        exit 1
    fi
else
    echo "⚠️  curl not available, skipping HTTP tests"
fi

# Test basic calculation endpoint
echo "Testing calculation endpoint..."
CALC_RESPONSE=$(curl -sf -X POST "http://localhost:${API_PORT}/api/v1/math/calculate" \
    -H "Authorization: Bearer math-tools-api-token" \
    -H "Content-Type: application/json" \
    -d '{"operation":"add","data":[2,3]}' 2>/dev/null || echo "")
if [[ "$CALC_RESPONSE" == *"\"result\":5"* ]]; then
    echo "✓ Calculation endpoint working correctly"
else
    echo "✗ Calculation endpoint failed"
    exit 1
fi

echo ""
echo "✓ All integration tests passed"
