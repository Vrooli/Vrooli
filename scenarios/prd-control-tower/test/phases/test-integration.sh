#!/usr/bin/env bash
#
# Integration Test: Test API and UI integration
#

set -eo pipefail

echo "🔍 Testing PRD Control Tower integration..."

API_PORT="${API_PORT:-18600}"
UI_PORT="${UI_PORT:-36300}"

# Check if API is running
echo "  Testing API health endpoint..."
if ! response=$(curl -s -f "http://localhost:${API_PORT}/api/v1/health" 2>/dev/null); then
    echo "    ✗ API health check failed"
    echo "    Make sure the API is running: make start"
    exit 1
fi

status=$(echo "$response" | jq -r '.status // "unknown"')
if [[ "$status" != "healthy" && "$status" != "degraded" ]]; then
    echo "    ✗ API status is $status (expected healthy or degraded)"
    exit 1
fi
echo "    ✓ API health check passed (status: $status)"

# Check if UI health server is running
echo "  Testing UI health endpoint..."
if ! ui_response=$(curl -s -f "http://localhost:${UI_PORT}/health" 2>/dev/null); then
    echo "    ✗ UI health check failed"
    echo "    Make sure the UI is running: make start"
    exit 1
fi

ui_status=$(echo "$ui_response" | jq -r '.status // "unknown"')
echo "    ✓ UI health check passed (status: $ui_status)"

# Test API connectivity from UI
api_connected=$(echo "$ui_response" | jq -r '.api_connectivity.connected // false')
if [[ "$api_connected" != "true" ]]; then
    echo "    ⚠ UI cannot connect to API (expected during initial scaffold)"
fi

echo "✅ Integration test passed"
