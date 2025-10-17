#!/bin/bash
# Test: Integration tests
# Tests the complete system integration

set -e

echo "  ✓ Checking if service is running..."

# Get the API port from running processes
API_PORT=$(vrooli scenario logs image-tools --step start-api --tail 5 2>/dev/null | grep -oP 'port \K\d+' | tail -1)

if [ -z "$API_PORT" ]; then
    echo "  ⚠️  Service not running, checking status..."
    vrooli scenario status image-tools | grep -q "RUNNING" || {
        echo "  ❌ Service is not running. Start with: make run"
        exit 1
    }
    # Try to get port again
    API_PORT=19364  # Fallback to known port
fi

echo "  ✓ Testing health endpoint on port $API_PORT..."
curl -sf "http://localhost:${API_PORT}/health" > /dev/null || {
    echo "  ❌ Health check failed"
    exit 1
}

echo "  ✓ Testing plugin registry..."
curl -sf "http://localhost:${API_PORT}/api/v1/plugins" | grep -q "jpeg" || {
    echo "  ❌ Plugin registry test failed"
    exit 1
}

echo "  ✓ Integration tests complete"