#!/bin/bash
set -e

echo "=== Performance Tests ==="

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

# Test API response time
echo "Testing API response time..."
START_TIME=$(date +%s%N)
curl -sf http://localhost:${API_PORT}/health > /dev/null
END_TIME=$(date +%s%N)
ELAPSED=$((($END_TIME - $START_TIME) / 1000000)) # Convert to milliseconds

echo "Health endpoint response time: ${ELAPSED}ms"

if [ $ELAPSED -lt 500 ]; then
    echo "✅ Response time under 500ms threshold"
else
    echo "⚠️  Response time ${ELAPSED}ms exceeds 500ms threshold"
fi

echo "✅ Performance tests completed"
