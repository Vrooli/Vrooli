#!/bin/bash
# Test: Integration tests
# Tests the complete system integration

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCENARIO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

echo "  ✓ Checking if service is running..."

# Get API port - use environment variable first, then check the service status
if [ -n "$API_PORT" ]; then
  : # Use provided API_PORT
elif [ -f "${SCENARIO_DIR}/.vrooli/runtime/ports.json" ]; then
  API_PORT=$(jq -r '.api // empty' "${SCENARIO_DIR}/.vrooli/runtime/ports.json" 2>/dev/null || echo "")
fi

# If still not found, check vrooli scenario status output
if [ -z "$API_PORT" ]; then
  API_PORT=$(vrooli scenario status image-tools 2>/dev/null | grep -oP 'http://localhost:\K[0-9]+' | head -1 || echo "")
fi

# Verify service is running
if [ -z "$API_PORT" ] || ! pgrep -f "image-tools-api" > /dev/null; then
    echo "  ❌ Service is not running. Start with: make start"
    exit 1
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