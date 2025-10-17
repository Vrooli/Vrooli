#!/bin/bash
# UI tests for Make It Vegan

set -e

echo "Running UI tests for Make It Vegan..."

# Get the UI port
UI_PORT="${UI_PORT:-$(vrooli scenario info make-it-vegan --json 2>/dev/null | jq -r '.ui.port' || echo '35000')}"
UI_URL="http://localhost:${UI_PORT}"

# Wait for UI to be ready
echo "Waiting for UI at ${UI_URL}..."
for i in {1..30}; do
    if curl -sf "${UI_URL}" > /dev/null 2>&1; then
        echo "✓ UI is ready"
        break
    fi
    if [ $i -eq 30 ]; then
        echo "❌ UI failed to start"
        exit 1
    fi
    sleep 1
done

# Test UI loads
echo "✓ Testing UI loads..."
curl -sf "${UI_URL}" | grep -q "Make It Vegan" || { echo "❌ UI content check failed"; exit 1; }

# Test app.js exists
echo "✓ Testing app.js loads..."
curl -sf "${UI_URL}/app.js" > /dev/null || { echo "❌ app.js check failed"; exit 1; }

echo "✅ UI tests completed"
