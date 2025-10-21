#!/bin/bash
set -e

echo "=== Business Logic Tests ==="

# Navigate to scenario root
cd "$(dirname "$0")/../.."

# Always try to detect the correct port for THIS scenario first
if command -v vrooli &> /dev/null; then
    DETECTED_PORT=$(vrooli scenario status scenario-to-ios --json 2>/dev/null | grep -o '"API_PORT":[[:space:]]*[0-9]*' | head -1 | grep -o '[0-9]*' || echo "")
    # Only use detected port if it's not empty
    if [ -n "$DETECTED_PORT" ]; then
        API_PORT="$DETECTED_PORT"
    else
        API_PORT="${API_PORT:-18570}"
    fi
else
    API_PORT="${API_PORT:-18570}"
fi

# Test health check returns correct service name
echo "Testing service identification on port ${API_PORT}..."
RESPONSE=$(curl -sf http://localhost:${API_PORT}/health)
if echo "$RESPONSE" | grep -q "scenario-to-ios"; then
    echo "✅ Service correctly identifies as scenario-to-ios"
else
    echo "❌ Service identification failed"
    exit 1
fi

# Test health status
if echo "$RESPONSE" | grep -q '"status":"healthy"'; then
    echo "✅ Service reports healthy status"
else
    echo "❌ Service health status check failed"
    exit 1
fi

echo "✅ All business tests completed"
