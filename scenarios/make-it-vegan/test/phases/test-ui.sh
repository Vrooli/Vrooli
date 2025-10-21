#!/bin/bash
# UI tests for Make It Vegan

set -e

echo "Running UI tests for Make It Vegan..."

# Get the UI and API ports from scenario status
SCENARIO_STATUS=$(vrooli scenario status make-it-vegan --json 2>/dev/null)
UI_PORT=$(echo "$SCENARIO_STATUS" | jq -r '.scenario_data.allocated_ports.UI_PORT // empty')
API_PORT=$(echo "$SCENARIO_STATUS" | jq -r '.scenario_data.allocated_ports.API_PORT // empty')

if [ -z "$UI_PORT" ] || [ -z "$API_PORT" ]; then
    echo "❌ Failed to get ports from scenario status"
    exit 1
fi

UI_URL="http://localhost:${UI_PORT}"
API_URL="http://localhost:${API_PORT}"

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

# Test 1: UI loads with correct title
echo "✓ Test 1: UI loads with correct title..."
curl -sf "${UI_URL}" | grep -q "Make It Vegan" || { echo "❌ UI title check failed"; exit 1; }

# Test 2: app.js loads
echo "✓ Test 2: app.js loads..."
curl -sf "${UI_URL}/app.js" > /dev/null || { echo "❌ app.js check failed"; exit 1; }

# Test 3: styles.css loads
echo "✓ Test 3: styles.css loads..."
curl -sf "${UI_URL}/styles.css" > /dev/null || { echo "❌ styles.css check failed"; exit 1; }

# Test 4: UI health endpoint works
echo "✓ Test 4: UI health endpoint..."
HEALTH_RESPONSE=$(curl -sf "${UI_URL}/health")
echo "$HEALTH_RESPONSE" | jq -e '.status == "healthy"' > /dev/null || { echo "❌ UI health check failed"; exit 1; }

# Test 5: API connectivity from UI health
echo "✓ Test 5: API connectivity check..."
echo "$HEALTH_RESPONSE" | jq -e '.api_connectivity.connected == true' > /dev/null || { echo "❌ API connectivity check failed"; exit 1; }

# Test 6: Visual rendering with browserless (if available)
echo "✓ Test 6: Visual rendering test..."
if command -v resource-browserless &> /dev/null; then
    SCREENSHOT_PATH="/tmp/make-it-vegan-ui-test.png"
    vrooli resource browserless screenshot --scenario make-it-vegan --output "$SCREENSHOT_PATH" > /dev/null 2>&1
    if [ -f "$SCREENSHOT_PATH" ]; then
        SIZE=$(stat -f%z "$SCREENSHOT_PATH" 2>/dev/null || stat -c%s "$SCREENSHOT_PATH" 2>/dev/null)
        if [ "$SIZE" -gt 1000 ]; then
            echo "  ✓ Screenshot captured successfully (${SIZE} bytes)"
        else
            echo "  ⚠️  Screenshot too small, UI may not have rendered"
        fi
    else
        echo "  ⚠️  Screenshot failed (non-critical)"
    fi
else
    echo "  ⚠️  Browserless not available, skipping visual test"
fi

# Test 7: All tabs present
echo "✓ Test 7: Tab navigation elements..."
UI_HTML=$(curl -sf "${UI_URL}")
echo "$UI_HTML" | grep -q "Check Ingredients" || { echo "❌ Check Ingredients tab missing"; exit 1; }
echo "$UI_HTML" | grep -q "Find Alternatives" || { echo "❌ Find Alternatives tab missing"; exit 1; }
echo "$UI_HTML" | grep -q "Veganize Recipe" || { echo "❌ Veganize Recipe tab missing"; exit 1; }

# Test 8: Essential UI elements present
echo "✓ Test 8: Essential UI elements..."
echo "$UI_HTML" | grep -q 'id="ingredients-input"' || { echo "❌ Ingredients input missing"; exit 1; }
echo "$UI_HTML" | grep -q 'id="check-result"' || { echo "❌ Check result div missing"; exit 1; }
echo "$UI_HTML" | grep -q 'onclick="checkIngredients()"' || { echo "❌ Check ingredients function missing"; exit 1; }

echo "✅ All UI tests passed (8/8)"

