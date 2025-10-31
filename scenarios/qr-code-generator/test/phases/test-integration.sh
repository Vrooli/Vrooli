#!/bin/bash

set -e

echo "=== Test Integration Phase ==="

# Test CLI binary integration
echo "Testing CLI generation..."
qr-generator generate "integration test" --output /tmp/test-integration.png
if [ -f "/tmp/test-integration.png" ]; then
    rm /tmp/test-integration.png
    echo "✓ CLI integration test passed"
else
    echo "✗ CLI integration test failed"
    exit 1
fi

# Test API integration
echo "Testing API health..."
API_PORT="${API_PORT:-17315}"
HEALTH_RESPONSE=$(curl -sf "http://localhost:${API_PORT}/health" || echo "failed")
if echo "$HEALTH_RESPONSE" | grep -q "healthy"; then
    echo "✓ API health check passed"
else
    echo "✗ API health check failed"
    exit 1
fi

echo "✓ Integration tests passed"
